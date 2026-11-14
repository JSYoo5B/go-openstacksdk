package instances_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk"
)

func rootEnabledRootAccess(ctx context.Context, conn *openstack.Connection, instanceID string) (bool, error) {
	service, err := conn.DatabaseV1(ctx)
	if err != nil {
		return false, err
	}
	return service.Instances.IsRootEnabled(ctx, instanceID)
}
