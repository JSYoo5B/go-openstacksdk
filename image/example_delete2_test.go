package image_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func deleteStoreCopyDelete2(ctx context.Context, client *gophercloud.ServiceClient, id, storeID string) (*image.DeleteImageResult, error) {
	ignoreMissing := false
	return image.New(client).DeleteImage(ctx, resource.ID(id),
		image.WithDeleteImageOpts(image.DeleteImageOpts{
			StoreID:       &storeID,
			IgnoreMissing: &ignoreMissing,
		}),
	)
}
