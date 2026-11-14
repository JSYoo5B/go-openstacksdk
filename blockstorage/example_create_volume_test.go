package blockstorage_test

import (
	"context"
	"encoding/json"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func createDefaultCreateVolume(ctx context.Context, conn *sdk.Connection) (*blockstorage.CreateVolumeResult, error) {
	return conn.CreateVolume(ctx, blockstorage.CreateVolumeRequest{Size: 10},
		func(c *blockstorage.CreateVolumeOpts) error { v := "data"; c.Attributes.Name = &v; return nil },
		blockstorage.WithCreateVolumeMetadata(map[string]string{"owner": "worker"}))
}

func createWithoutWaitingCreateVolume(ctx context.Context, cinder *gophercloud.ServiceClient) (*blockstorage.CreateVolumeResult, error) {
	return blockstorage.CreateVolume(ctx, cinder, nil,
		blockstorage.CreateVolumeRequest{Size: 10},
		blockstorage.WithCreateVolumeWait(false))
}

func createBootableFromImageCreateVolume(ctx context.Context, conn *sdk.Connection) (*blockstorage.CreateVolumeResult, error) {
	image := resource.Name("ubuntu")
	return conn.CreateVolume(ctx,
		blockstorage.CreateVolumeRequest{Size: 20, Image: &image},
		func(c *blockstorage.CreateVolumeOpts) error { v := true; c.Bootable = &v; return nil })
}

func createWithPolicyCreateVolume(ctx context.Context, conn *sdk.Connection, progress func(int) error) (*blockstorage.CreateVolumeResult, error) {
	return conn.CreateVolume(ctx, blockstorage.CreateVolumeRequest{Size: 30},
		func(c *blockstorage.CreateVolumeOpts) error {
			v := "snapshot-id"
			c.Attributes.SnapshotID = &v
			return nil
		},
		blockstorage.WithCreateVolumeFields(map[string]json.RawMessage{
			"volume_type_id": json.RawMessage(`"type-id"`),
			"is_multiattach": json.RawMessage(`false`),
		}),
		blockstorage.WithCreateVolumeSchedulerHints(map[string]json.RawMessage{
			"same_host": json.RawMessage(`["source-volume-id"]`),
		}),
		blockstorage.WithCreateVolumeWaitOptions(
			resource.WithTimeout(5*time.Minute), resource.WithPollInterval(time.Second),
			resource.WithProgressCallback(progress),
		))
}
