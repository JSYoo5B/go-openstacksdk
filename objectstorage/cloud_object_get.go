package objectstorage

import (
	"context"
	"io"

	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
)

// ObjectGetResult returns Source's lowercase tuple headers/optional text while
// retaining actual receipt and accepted destination-byte evidence separately.
type ObjectGetResult = objects.CloudGetResult
type ObjectGetOpts = objects.CloudGetOpts
type ObjectGetOption = objects.CloudGetOption

func WithObjectGetOpts(value ObjectGetOpts) ObjectGetOption { return objects.WithCloudGetOpts(value) }
func WithObjectGetOutfile(value string) ObjectGetOption     { return objects.WithCloudGetOutfile(value) }
func WithObjectGetWriter(value io.Writer) ObjectGetOption   { return objects.WithCloudGetWriter(value) }
func WithoutObjectGetContent() ObjectGetOption              { return objects.WithoutCloudGetContent() }
func WithoutObjectGetOutfile() ObjectGetOption              { return objects.WithoutCloudGetOutfile() }

// GetObjectText eagerly gets the closed cloud string-name branch. Path output
// is SDK-owned; Writer output remains caller-owned and may implement Flush.
// The default returns decoded text; unresolved statistical charset detection
// remains an explicit unsupported error with actual received-body evidence.
func (c *Cloud) GetObjectText(ctx context.Context, container, object string, options ...ObjectGetOption) (*ObjectGetResult, error) {
	if c == nil {
		return (*objects.API)(nil).GetCloudObjectText(ctx, container, object, options...)
	}
	return objects.New(c.client).GetCloudObjectText(ctx, container, object, options...)
}
