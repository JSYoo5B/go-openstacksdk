package objectstorage

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
)

// ObjectChunkStream is the library-owned closeable lazy Swift chunk iterator.
// Use Chunks to consume it and Err after an early break to check cleanup.
type ObjectChunkStream = objects.CloudObjectStream
type ObjectStreamReceipt = objects.CloudStreamReceipt
type ObjectStreamOpts = objects.CloudStreamOpts
type ObjectStreamOption = objects.CloudStreamOption

func WithObjectStreamOpts(value ObjectStreamOpts) ObjectStreamOption {
	return objects.WithCloudStreamOpts(value)
}
func WithObjectStreamChunkSize(value int) ObjectStreamOption {
	return objects.WithCloudStreamChunkSize(value)
}
func WithObjectStreamQueryString(value string) ObjectStreamOption {
	return objects.WithCloudStreamQueryString(value)
}

// StreamObjectChunks creates a lazy stream_object(container, obj) adapter. Its
// fixed query_string option is ignored by Source. A nil cloud fails during the
// first iteration with no request; the stream always owns response cleanup.
func (c *Cloud) StreamObjectChunks(ctx context.Context, container, object string, options ...ObjectStreamOption) ObjectChunkStream {
	if c == nil {
		return (*objects.API)(nil).StreamCloudObject(ctx, container, object, options...)
	}
	return objects.New(c.client).StreamCloudObject(ctx, container, object, options...)
}
