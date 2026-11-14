package serviceinfo_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func discoverStoresServiceinfo1(ctx context.Context, conn *openstack.Connection) ([]*serviceinfo.Store, error) {
	service, err := conn.ImageV2(ctx)
	if err != nil {
		return nil, err
	}
	return service.ServiceInfo.AllStores(ctx,
		request.Custom(func(c *request.Config[serviceinfo.ListStoresOpts]) error { c.Options.Details = true; return nil }),
		request.MaxItems(20),
	)
}
