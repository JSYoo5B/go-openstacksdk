package imagedata_test

import (
	"context"
	"io"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

func stageKnownImageImagedata2(ctx context.Context, client *gophercloud.ServiceClient, image *images.Image, data io.Reader, remainingBytes int64) (*imagedata.StageImageResult, error) {
	return imagedata.New(client).StageKnownImage(ctx, image, data,
		request.Custom(func(c *imagedata.StageOpts) error { v := remainingBytes; c.Size = &v; return nil }),
		request.Header("X-Trace", "stage-image"),
	)
}
