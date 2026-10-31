package openstack

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudobjectstorage"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage"
)

// objectStorageDependencies supplies the Connection guard, lazy Swift service
// and location to the cloud-layer Swift helpers.
func (c *Connection) objectStorageDependencies() cloudobjectstorage.Dependencies {
	return cloudobjectstorage.Dependencies{
		Guard:         c.recordListContext,
		ObjectStorage: c.ObjectStorageV1,
		Location:      c.CurrentLocation,
	}
}

// SearchContainers shares the cached Swift service and supplies configured
// Connection location before identifier/mapping/expression selection. Explicit
// per-call typed location overrides remain owned; a reset DTO uses the Connection
// default without requiring callers to construct a resource or builder.
func (c *Connection) SearchContainers(ctx context.Context, options ...objectstorage.ContainerSearchOption) (*objectstorage.ContainerSearchResult, error) {
	return cloudobjectstorage.SearchContainers(ctx, c.objectStorageDependencies(), options...)
}

// SearchObjects shares the cached Swift service, collects its container listing
// and supplies the configured Connection location before cloud filtering.
// Explicit per-call location overrides remain owned; a reset DTO returns to the
// Connection default without asking users to build individual resource objects.
func (c *Connection) SearchObjects(ctx context.Context, container string, options ...objectstorage.ObjectSearchOption) (*objectstorage.ObjectSearchResult, error) {
	return cloudobjectstorage.SearchObjects(ctx, c.objectStorageDependencies(), container, options...)
}

// GetObjectText supplies the cached Swift facade for eager owned text or output handling.
func (c *Connection) GetObjectText(ctx context.Context, container, object string, options ...objectstorage.ObjectGetOption) (*objectstorage.ObjectGetResult, error) {
	return cloudobjectstorage.GetObjectText(ctx, c.objectStorageDependencies(), container, object, options...)
}

// StreamObjectChunks supplies the cached Swift facade when the SDK-owned
// iterator is first consumed. Close and Err preserve cleanup after early break.
func (c *Connection) StreamObjectChunks(ctx context.Context, container, object string, options ...objectstorage.ObjectStreamOption) objectstorage.ObjectChunkStream {
	return cloudobjectstorage.StreamObjectChunks(ctx, c.objectStorageDependencies(), container, object, options...)
}
