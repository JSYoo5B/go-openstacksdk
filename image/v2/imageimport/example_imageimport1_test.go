package imageimport_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func importStagedImageImageimport1(ctx context.Context, client *gophercloud.ServiceClient, id string) (*imageimport.ImportResult, error) {
	return imageimport.New(client).ImportImage(ctx, resource.ID(id))
}
