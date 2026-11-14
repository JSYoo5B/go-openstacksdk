package volumes_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func mergeVolumeMetadataVolumes(ctx context.Context, client *gophercloud.ServiceClient, id string) (map[string]string, error) {
	scope, err := volumes.New(client).MetadataIn(ctx, resource.ID(id))
	if err != nil {
		return nil, err
	}
	result, err := scope.Merge(ctx, map[string]string{"owner": "team-a"})
	if err != nil {
		return nil, err
	}
	return result.Metadata, nil // full actual server response, including retained keys
}
