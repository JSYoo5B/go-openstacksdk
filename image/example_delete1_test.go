package image_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func deleteWholeImageDelete1(ctx context.Context, client *gophercloud.ServiceClient, id string) (*image.DeleteImageResult, error) {
	return image.New(client).DeleteImage(ctx, resource.ID(id))
}
