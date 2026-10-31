package openstack

import (
	"context"
	"errors"
	"iter"
	"slices"
	"sync"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// SearchContainers shares the cached Swift service and supplies configured
// Connection location before identifier/mapping/expression selection. Explicit
// per-call typed location overrides remain owned; a reset DTO uses the Connection
// default without requiring callers to construct a resource or builder.
func (c *Connection) SearchContainers(ctx context.Context, options ...objectstorage.ContainerSearchOption) (*objectstorage.ContainerSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchContainers", "containers", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.ObjectStorageV1(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), func(cfg *objectstorage.ContainerSearchOpts) error {
		if cfg.Location == nil {
			value := location.Clone()
			cfg.Location = &value
		}
		return cloudread.Context(ctx)
	})
	result, err := service.Cloud.SearchContainers(ctx, owned...)
	return result, wrap(err)
}

// SearchObjects shares the cached Swift service, collects its container listing
// and supplies the configured Connection location before cloud filtering.
// Explicit per-call location overrides remain owned; a reset DTO returns to the
// Connection default without asking users to build individual resource objects.
func (c *Connection) SearchObjects(ctx context.Context, container string, options ...objectstorage.ObjectSearchOption) (*objectstorage.ObjectSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchObjects", "objects", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.ObjectStorageV1(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	location, err := c.CurrentLocation()
	if err != nil {
		return nil, wrap(err)
	}
	owned := append(slices.Clone(options), func(cfg *objectstorage.ObjectSearchOpts) error {
		if cfg.Location == nil {
			value := location.Clone()
			cfg.Location = &value
		}
		return cloudread.Context(ctx)
	})
	result, err := service.Cloud.SearchObjects(ctx, container, owned...)
	return result, wrap(err)
}

// GetObjectText supplies the cached Swift facade for eager owned text or output handling.
func (c *Connection) GetObjectText(ctx context.Context, container, object string, options ...objectstorage.ObjectGetOption) (*objectstorage.ObjectGetResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetObjectText", "objects", cloudread.ContextError(ctx, err))
	}
	if err := c.recordListContext(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := c.ObjectStorageV1(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	value, err := service.Cloud.GetObjectText(ctx, container, object, slices.Clone(options)...)
	return value, wrap(err)
}

// StreamObjectChunks supplies the cached Swift facade when the SDK-owned
// iterator is first consumed. Close and Err preserve cleanup after early break.
func (c *Connection) StreamObjectChunks(ctx context.Context, container, object string, options ...objectstorage.ObjectStreamOption) objectstorage.ObjectChunkStream {
	return &connectionObjectChunkStream{connection: c, ctx: ctx, container: container, object: object, options: slices.Clone(options)}
}

type connectionObjectChunkStream struct {
	mu                sync.Mutex
	connection        *Connection
	ctx               context.Context
	container, object string
	options           []objectstorage.ObjectStreamOption
	child             objectstorage.ObjectChunkStream
	started, closed   bool
	err               error
}

func (s *connectionObjectChunkStream) wrap(err error) error {
	if err == nil {
		return nil
	}
	return request.Wrap("StreamObjectChunks", "objects", cloudread.ContextError(s.ctx, err))
}

func (s *connectionObjectChunkStream) setError(err error) {
	if err != nil {
		s.mu.Lock()
		s.err = errors.Join(s.err, err)
		s.mu.Unlock()
	}
}

func (s *connectionObjectChunkStream) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

func (s *connectionObjectChunkStream) Chunks() iter.Seq2[[]byte, error] {
	return func(yield func([]byte, error) bool) {
		s.mu.Lock()
		if s.started || s.closed {
			s.mu.Unlock()
			return
		}
		s.started = true
		s.mu.Unlock()
		defer s.Close()
		if err := s.connection.recordListContext(s.ctx); err != nil {
			s.setError(err)
			yield(nil, s.wrap(err))
			return
		}
		service, err := s.connection.ObjectStorageV1(s.ctx)
		if err != nil {
			s.setError(err)
			yield(nil, s.wrap(err))
			return
		}
		if s.isClosed() {
			return
		}
		child := service.Cloud.StreamObjectChunks(s.ctx, s.container, s.object, s.options...)
		s.mu.Lock()
		s.child = child
		closed := s.closed
		s.mu.Unlock()
		if closed {
			_ = child.Close()
			return
		}
		for data, err := range child.Chunks() {
			if s.isClosed() {
				return
			}
			if !yield(data, s.wrap(err)) || err != nil {
				return
			}
		}
	}
}

func (s *connectionObjectChunkStream) Close() error {
	s.mu.Lock()
	s.closed = true
	child := s.child
	s.mu.Unlock()
	if child != nil {
		_ = child.Close()
	}
	return s.Err()
}

func (s *connectionObjectChunkStream) Err() error {
	s.mu.Lock()
	err, child := s.err, s.child
	s.mu.Unlock()
	if child != nil {
		err = errors.Join(err, child.Err())
	}
	return s.wrap(err)
}

func (s *connectionObjectChunkStream) Receipt() *objects.CloudStreamReceipt {
	s.mu.Lock()
	child := s.child
	s.mu.Unlock()
	if child == nil {
		return nil
	}
	value := child.Receipt()
	if value == nil {
		return nil
	}
	copy := *value
	copy.Header = value.Header.Clone()
	return &copy
}
