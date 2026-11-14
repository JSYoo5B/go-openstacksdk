package blockstorage_test

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func ConfiguredVolumeSnapshots(ctx context.Context, cloudName string) (*sdk.Connection, error) {
	return sdk.Connect(ctx, sdk.WithCloud(cloudName))
}

func AdoptVolumeSnapshots(provider *gophercloud.ProviderClient) (*sdk.Connection, error) {
	return sdk.FromProvider(provider, sdk.WithRegion("RegionOne"))
}

func InspectPagesVolumeSnapshots(pages []*blockstorage.VolumeSnapshotsPage) {
	for _, page := range pages {
		fmt.Println("admitted snapshot page:", page.StatusCode, "bytes:", len(page.Body))
	}
}

func InspectVolumeSnapshots(result *blockstorage.GetVolumeSnapshotResult) {
	if result == nil {
		return
	}
	InspectPagesVolumeSnapshots(result.Pages)
	if result.Observed != nil {
		fmt.Println("member response:", result.Observed.StatusCode, len(result.Observed.Body))
	}
	fmt.Println("requested ID:", result.RequestedID, "seeded logical ID:", result.SeededID)
	if result.Snapshot != nil {
		owned := result.Snapshot.Clone()
		fmt.Println("actual response ID JSON:", string(owned.Body["id"]))
		var wire map[string]json.RawMessage
		if err := owned.Decode(&wire); err != nil {
			fmt.Println("optional wire projection:", err)
		} else {
			fmt.Println("wire fields:", len(wire))
		}
	}
}

func ViaConnectionVolumeSnapshots(ctx context.Context, conn *sdk.Connection) error {
	listed, err := conn.ListVolumeSnapshots(ctx,
		blockstorage.WithVolumeSnapshotListFilters(json.RawMessage(
			`{"status":"available","volume_id":"volume-id","description":"nightly"}`)))
	if listed != nil {
		InspectPagesVolumeSnapshots(listed.Pages)
	}
	if err != nil {
		return err
	}
	fmt.Println("complete normalized list:", string(listed.Value), "raw rows:", len(listed.Snapshots))

	searched, err := conn.SearchVolumeSnapshots(ctx,
		blockstorage.SearchVolumeSnapshotsRequest{NameOrID: "nightly*"},
		blockstorage.WithVolumeSnapshotSearchFilters(json.RawMessage(`{"status":"available"}`)))
	if searched != nil {
		InspectPagesVolumeSnapshots(searched.Pages)
	}
	if err != nil {
		return err
	}
	fmt.Println("selected normalized snapshots:", string(searched.Value))

	selected, err := conn.GetVolumeSnapshot(ctx,
		blockstorage.GetVolumeSnapshotRequest{NameOrID: "nightly"})
	InspectVolumeSnapshots(selected) // Physical proof remains useful when err is nonnil.
	if err != nil {
		return err
	}
	fmt.Println("selected normalized JSON:", string(selected.Value))

	byID, err := conn.GetVolumeSnapshotByID(ctx,
		blockstorage.GetVolumeSnapshotByIDRequest{ID: "snapshot-id"})
	InspectVolumeSnapshots(byID)
	return err
}

func ProjectNamesVolumeSnapshots(ctx context.Context, conn *sdk.Connection) error {
	result, err := conn.SearchVolumeSnapshots(ctx,
		blockstorage.SearchVolumeSnapshotsRequest{},
		blockstorage.WithVolumeSnapshotSearchExpression("[].name"))
	if result != nil {
		InspectPagesVolumeSnapshots(result.Pages)
	}
	if err != nil {
		return err
	}
	fmt.Println("arbitrary expression JSON:", string(result.Value))
	return nil
}

func FirstPageVolumeSnapshots(ctx context.Context, conn *sdk.Connection) error {
	detailed, paginated, max := false, false, 10
	result, err := conn.ListVolumeSnapshots(ctx,
		blockstorage.WithVolumeSnapshotListOptions(blockstorage.VolumeSnapshotListOpts{
			Detailed: &detailed, Paginated: &paginated, MaxItems: &max,
			Headers: map[string]string{"X-Request-ID": "snapshot-read-example"},
		}))
	if result != nil {
		InspectPagesVolumeSnapshots(result.Pages)
	}
	return err
}

func DirectVolumeSnapshots(ctx context.Context, cinder *gophercloud.ServiceClient) error {
	location := resource.CloudLocation{}
	listed, err := blockstorage.ListVolumeSnapshots(ctx, cinder,
		request.Custom(func(c *blockstorage.VolumeSnapshotListOpts) error {
			loc := location.Clone()
			c.Location = &loc
			return nil
		}))
	if listed != nil {
		InspectPagesVolumeSnapshots(listed.Pages)
	}
	if err != nil {
		return err
	}

	searched, err := blockstorage.SearchVolumeSnapshots(ctx, cinder,
		blockstorage.SearchVolumeSnapshotsRequest{NameOrID: "nightly*"},
		func(c *blockstorage.VolumeSnapshotSearchOpts) error {
			loc := location.Clone()
			c.Location = &loc
			return nil
		})
	if searched != nil {
		InspectPagesVolumeSnapshots(searched.Pages)
	}
	if err != nil {
		return err
	}

	// Explicit {} filters choose full Search, rather than ordinary GET-first find.
	selected, err := blockstorage.GetVolumeSnapshot(ctx, cinder,
		blockstorage.GetVolumeSnapshotRequest{NameOrID: "nightly"},
		blockstorage.WithVolumeSnapshotSearchFilters(json.RawMessage(`{}`)))
	InspectVolumeSnapshots(selected)
	if err != nil {
		return err
	}

	byID, err := blockstorage.GetVolumeSnapshotByID(ctx, cinder,
		blockstorage.GetVolumeSnapshotByIDRequest{ID: "snapshot-id"},
		blockstorage.WithVolumeSnapshotReadOptions(blockstorage.VolumeSnapshotReadOpts{Location: &location}))
	InspectVolumeSnapshots(byID)
	return err
}
