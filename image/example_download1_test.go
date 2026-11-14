package image_test

import (
	"context"
	"io"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func downloadToDownload1(ctx context.Context, client *gophercloud.ServiceClient, id string, output io.Writer) (*image.DownloadImageResult, error) {
	return image.New(client).DownloadTo(ctx, resource.ID(id), output)
}
