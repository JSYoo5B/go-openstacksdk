package image_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport"
	"github.com/gophercloud/gophercloud/v2"
)

func createFromWebCreateImport2(ctx context.Context, client *gophercloud.ServiceClient, uri string) (*image.CreateImportResult, error) {
	requireAll := false
	return image.New(client).CreateAndImport(ctx,
		image.CreateAndImportRequest{Name: "worker-from-web"},
		image.WithCreateImportImport(imageimport.ImportOpts{
			Method: "web-download", URI: uri,
			Stores: []string{"fast", "reliable"}, AllStoresMustSucceed: &requireAll,
		}),
	)
}
