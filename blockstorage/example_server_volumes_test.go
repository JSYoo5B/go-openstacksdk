package blockstorage_test

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func getForServerServerVolumes(ctx context.Context, conn *sdk.Connection, serverID string) error {
	result, err := conn.GetVolumes(ctx, blockstorage.GetVolumesRequest{ServerID: serverID})
	return inspectResultServerVolumes(result, err)
}

func getWithClientServerVolumes(ctx context.Context, cinder *gophercloud.ServiceClient, serverID string) (*blockstorage.GetVolumesResult, error) {
	return blockstorage.GetVolumes(ctx, cinder,
		blockstorage.GetVolumesRequest{ServerID: serverID})
}

func getForRawServerServerVolumes(ctx context.Context, conn *sdk.Connection) (*blockstorage.GetVolumesResult, error) {
	// An explicit null ID can match a null attachment server_id.
	server := map[string]json.RawMessage{"id": json.RawMessage(`null`)}
	return conn.GetVolumes(ctx, blockstorage.GetVolumesRequest{},
		blockstorage.WithGetVolumesServerFields(server))
}

func prepareServerFieldsServerVolumes(ctx context.Context, server map[string]json.RawMessage) (blockstorage.GetVolumesOpts, error) {
	// Runs options locally once and owns their maps/raw bytes; no HTTP.
	return blockstorage.PrepareGetVolumesOptions(ctx,
		blockstorage.WithGetVolumesServerFields(server))
}

func getWithPreparedOptionsServerVolumes(ctx context.Context, conn *sdk.Connection, prepared blockstorage.GetVolumesOpts) (*blockstorage.GetVolumesResult, error) {
	return conn.GetVolumes(ctx, blockstorage.GetVolumesRequest{},
		blockstorage.WithGetVolumesOptions(prepared))
}

func inspectResultServerVolumes(result *blockstorage.GetVolumesResult, callErr error) error {
	if callErr != nil {
		if result != nil {
			// Volumes is nil on every workflow error. Pages is actual response evidence.
			for i, page := range result.Pages {
				fmt.Printf("accepted page %d: status=%d request=%q bytes=%d\n",
					i, page.StatusCode, page.Header.Get("X-Openstack-Request-Id"), len(page.Body))
			}
		}
		return callErr
	}
	for i, volume := range result.Volumes {
		fmt.Printf("match %d: raw id=%s raw status=%s\n", i,
			volume.Body["id"], volume.Body["status"])

		var projected blockstorage.VolumeInfo
		if err := volume.Decode(&projected); err != nil {
			// Selection succeeded; this optional stricter typed projection failed.
			fmt.Printf("typed projection %d: %v\n", i, err)
		} else {
			fmt.Printf("typed projection %d: status=%d request=%q\n", i,
				projected.StatusCode, projected.Header.Get("X-Openstack-Request-Id"))
		}

		privateCopy := volume.Clone()
		privateCopy.Body["name"] = json.RawMessage(`"private-copy"`)
		fmt.Printf("copy %d name=%s; original name=%s\n", i,
			privateCopy.Body["name"], volume.Body["name"])
	}
	return nil
}

func independentMatchesServerVolumes(result *blockstorage.GetVolumesResult) []*resource.RawResource {
	copies := make([]*resource.RawResource, len(result.Volumes))
	for i, volume := range result.Volumes {
		copies[i] = volume.Clone()
	}
	return copies
}
