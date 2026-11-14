package imageimport_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/images"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

func importKnownFromWebImageimport2(ctx context.Context, client *gophercloud.ServiceClient, image *images.Image, uri string) (*imageimport.ImportResult, error) {
	return imageimport.New(client).ImportKnownImage(ctx, image,
		request.Custom(func(c *imageimport.ImportOpts) error { c.Method = "web-download"; return nil }),
		request.Custom(func(c *imageimport.ImportOpts) error { c.URI = uri; return nil }),
		imageimport.WithImportStores("fast", "reliable"),
		request.Custom(func(c *imageimport.ImportOpts) error { v := false; c.AllStoresMustSucceed = &v; return nil }),
	)
}
