package blockstorage_test

import (
	"context"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func attachDefaultAttachVolume(ctx context.Context, conn *sdk.Connection, serverID, volumeID string) (*blockstorage.AttachVolumeResult, error) {
	return conn.AttachVolume(ctx, blockstorage.AttachVolumeRequest{
		Server: resource.ID(serverID), Volume: resource.ID(volumeID),
	})
}

func attachWithoutWaitingAttachVolume(ctx context.Context, conn *sdk.Connection, serverID, volumeID string) (*blockstorage.AttachVolumeResult, error) {
	return conn.AttachVolume(ctx, blockstorage.AttachVolumeRequest{
		Server: resource.ID(serverID), Volume: resource.ID(volumeID),
	}, func(c *blockstorage.AttachVolumeOpts) error { c.Device = "/dev/vdb"; return nil },
		blockstorage.WithAttachVolumeWait(false))
}

func attachWithDeadlineAttachVolume(ctx context.Context, conn *sdk.Connection) (*blockstorage.AttachVolumeResult, error) {
	return conn.AttachVolume(ctx, blockstorage.AttachVolumeRequest{
		Server: resource.Name("worker"), Volume: resource.Name("data"),
	}, blockstorage.WithAttachVolumeWaitOptions(
		resource.WithTimeout(5*time.Minute), resource.WithPollInterval(time.Second),
	))
}

func attachWithClientsAttachVolume(ctx context.Context, nova, cinder *gophercloud.ServiceClient, input blockstorage.AttachVolumeRequest) (*blockstorage.AttachVolumeResult, error) {
	return blockstorage.AttachVolume(ctx, nova, cinder, input)
}
