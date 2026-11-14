package imagedata_test

import (
	"context"
	"io"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func stageExistingImageImagedata1(ctx context.Context, client *gophercloud.ServiceClient, id string, data io.Reader) (*imagedata.StageImageResult, error) {
	return imagedata.New(client).StageImage(ctx, resource.ID(id), data)
}
