package image_test

import (
	"context"
	"io"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata"
	"github.com/gophercloud/gophercloud/v2"
)

func createAndWaitCreateImport1(ctx context.Context, client *gophercloud.ServiceClient, data io.Reader, size int64) (*image.CreateImportResult, error) {
	return image.New(client).CreateAndImport(ctx,
		image.CreateAndImportRequest{Name: "worker", Data: data},
		image.WithCreateImportStage(imagedata.StageOpts{Size: &size}),
		image.WithCreateImportWait(),
	)
}
