package snapshots_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
	"github.com/gophercloud/gophercloud/v2"
)

func replaceSnapshotMetadataSnapshots(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]any, error) {
	return snapshots.New(client).UpdateMetadata(ctx, id, snapshots.UpdateMetadataOpts{
		Metadata: map[string]any{"owner": "team-a", "purpose": "backup"},
	})
}
