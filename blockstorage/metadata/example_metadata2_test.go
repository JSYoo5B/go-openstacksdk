package metadata_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func deleteSnapshotMetadataKeysMetadata2(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
	scope, err := snapshots.New(client).MetadataIn(ctx, resource.ID(id))
	if err != nil {
		return err
	}
	result, err := scope.DeleteKeys(ctx, []string{"temporary", "obsolete"})
	if result != nil {
		for _, deleted := range result.Deleted {
			_ = deleted.Key // requested key with its actual response evidence
			_ = deleted.StatusCode
		}
	}
	return err // successful earlier deletes remain recorded on failure
}
