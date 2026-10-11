package snapshots

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	gophercloud "github.com/gophercloud/gophercloud/v2"
	upstream "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/snapshots"
)

// waitForStatus polls like native WaitForStatus, but a Get response without
// the snapshot envelope is reported as an error instead of a nil dereference.
func waitForStatus(ctx context.Context, client *gophercloud.ServiceClient, id string, status string) error {
	return gophercloud.WaitFor(ctx, func(ctx context.Context) (bool, error) {
		current, err := upstream.Get(ctx, client, id).Extract()
		if err != nil {
			return false, err
		}
		if current == nil {
			return false, fmt.Errorf("%w: snapshot %q response has no snapshot", resource.ErrUnsupported, id)
		}
		return current.Status == status, nil
	})
}
