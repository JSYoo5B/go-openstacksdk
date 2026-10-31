// Package cloudobjectstorage implements the cloud-layer Swift helpers of
// Connection (openstack/cloud/_object_store.py search_containers,
// search_objects, get_object and the chunked object stream) over the
// objectstorage facade. Connection supplies its guard, lazy Swift service and
// location through Dependencies, so this package never imports the root.
package cloudobjectstorage

import (
	"context"
	"errors"
	"iter"
	"slices"
	"sync"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
	objectstorageapi "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Dependencies are the Connection facilities the Swift cloud helpers use.
type Dependencies struct {
	// Guard checks the context and that the Connection is usable before any
	// service is resolved.
	Guard func(context.Context) error
	// ObjectStorage is the lazy Swift service.
	ObjectStorage func(context.Context) (*objectstorageapi.Service, error)
	// Location is the current cloud, region and project scope.
	Location func() (resource.CloudLocation, error)
}

// SearchContainers shares the cached Swift service and supplies configured
// Connection location before identifier/mapping/expression selection. Explicit
// per-call typed location overrides remain owned; a reset DTO uses the Connection
// default without requiring callers to construct a resource or builder.
func SearchContainers(ctx context.Context, deps Dependencies, options ...objectstorage.ContainerSearchOption) (*objectstorage.ContainerSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchContainers", "containers", cloudread.ContextError(ctx, err))
	}
	if err := deps.Guard(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := deps.ObjectStorage(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	location, err := deps.Location()
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
func SearchObjects(ctx context.Context, deps Dependencies, container string, options ...objectstorage.ObjectSearchOption) (*objectstorage.ObjectSearchResult, error) {
	wrap := func(err error) error {
		return request.Wrap("SearchObjects", "objects", cloudread.ContextError(ctx, err))
	}
	if err := deps.Guard(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := deps.ObjectStorage(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	location, err := deps.Location()
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
func GetObjectText(ctx context.Context, deps Dependencies, container, object string, options ...objectstorage.ObjectGetOption) (*objectstorage.ObjectGetResult, error) {
	wrap := func(err error) error {
		return request.Wrap("GetObjectText", "objects", cloudread.ContextError(ctx, err))
	}
	if err := deps.Guard(ctx); err != nil {
		return nil, wrap(err)
	}
	service, err := deps.ObjectStorage(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	value, err := service.Cloud.GetObjectText(ctx, container, object, slices.Clone(options)...)
	return value, wrap(err)
}

// StreamObjectChunks supplies the cached Swift facade when the SDK-owned
// iterator is first consumed. Close and Err preserve cleanup after early break.
// Receipt returns a copy whose Header does not alias the child receipt.
func StreamObjectChunks(ctx context.Context, deps Dependencies, container, object string, options ...objectstorage.ObjectStreamOption) objectstorage.ObjectChunkStream {
	return &chunkStream{deps: deps, ctx: ctx, container: container, object: object, options: slices.Clone(options)}
}

type chunkStream struct {
	mu                sync.Mutex
	deps              Dependencies
	ctx               context.Context
	container, object string
	options           []objectstorage.ObjectStreamOption
	child             objectstorage.ObjectChunkStream
	started, closed   bool
	err               error
}

func (s *chunkStream) wrap(err error) error {
	if err == nil {
		return nil
	}
	return request.Wrap("StreamObjectChunks", "objects", cloudread.ContextError(s.ctx, err))
}

func (s *chunkStream) setError(err error) {
	if err != nil {
		s.mu.Lock()
		s.err = errors.Join(s.err, err)
		s.mu.Unlock()
	}
}

func (s *chunkStream) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

func (s *chunkStream) Chunks() iter.Seq2[[]byte, error] {
	return func(yield func([]byte, error) bool) {
		s.mu.Lock()
		if s.started || s.closed {
			s.mu.Unlock()
			return
		}
		s.started = true
		s.mu.Unlock()
		defer s.Close()
		if err := s.deps.Guard(s.ctx); err != nil {
			s.setError(err)
			yield(nil, s.wrap(err))
			return
		}
		service, err := s.deps.ObjectStorage(s.ctx)
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

func (s *chunkStream) Close() error {
	s.mu.Lock()
	s.closed = true
	child := s.child
	s.mu.Unlock()
	if child != nil {
		_ = child.Close()
	}
	return s.Err()
}

func (s *chunkStream) Err() error {
	s.mu.Lock()
	err, child := s.err, s.child
	s.mu.Unlock()
	if child != nil {
		err = errors.Join(err, child.Err())
	}
	return s.wrap(err)
}

func (s *chunkStream) Receipt() *objects.CloudStreamReceipt {
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
