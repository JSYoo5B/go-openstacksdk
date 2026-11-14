package serviceinfo_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
)

func discoverUsage(ctx context.Context, conn *openstack.Connection) (*serviceinfo.UsageInfo, error) {
	service, err := conn.ImageV2(ctx)
	if err != nil {
		return nil, err
	}
	return service.ServiceInfo.GetUsageInfo(ctx)
}
