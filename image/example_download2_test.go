package image_test

import (
	"bytes"
	"context"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func downloadByNameDownload2(ctx context.Context, client *gophercloud.ServiceClient, name string) ([]byte, *image.DownloadImageResult, error) {
	var output bytes.Buffer
	chunk := 64 * 1024
	result, err := image.New(client).DownloadTo(ctx, resource.Name(name), &output,
		image.WithDownloadImageOpts(image.DownloadImageOpts{
			ChunkSize:        &chunk,
			StorePreferences: []string{"fast", "archive", "fast"},
		}),
	)
	return output.Bytes(), result, err
}
