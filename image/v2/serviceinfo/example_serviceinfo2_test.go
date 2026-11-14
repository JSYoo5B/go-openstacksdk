package serviceinfo_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
	"github.com/gophercloud/gophercloud/v2"
)

func discoverImportMethodsServiceinfo2(ctx context.Context, client *gophercloud.ServiceClient) (*serviceinfo.ImportInfo, error) {
	return image.New(client).API.ServiceInfo.GetImportInfo(ctx)
}
