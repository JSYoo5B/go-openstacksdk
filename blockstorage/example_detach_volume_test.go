package blockstorage_test

import (
	"context"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func detachDefaultDetachVolume(ctx context.Context, conn *sdk.Connection, serverID, volumeID string) (*blockstorage.DetachVolumeResult, error) {
	return conn.DetachVolume(ctx, blockstorage.DetachVolumeRequest{
		Server: resource.ID(serverID), Volume: resource.ID(volumeID),
	})
}

func detachNovaOnlyDetachVolume(ctx context.Context, nova *gophercloud.ServiceClient, serverID, volumeID string) (*blockstorage.DetachVolumeResult, error) {
	return blockstorage.DetachVolume(ctx, nova, nil,
		blockstorage.DetachVolumeRequest{
			Server: resource.ID(serverID), Volume: resource.ID(volumeID),
		}, blockstorage.WithDetachVolumeWait(false))
}

func detachWithDeadlineDetachVolume(ctx context.Context, conn *sdk.Connection, input blockstorage.DetachVolumeRequest, progress func(int) error) (*blockstorage.DetachVolumeResult, error) {
	return conn.DetachVolume(ctx, input,
		blockstorage.WithDetachVolumeWaitOptions(
			resource.WithTimeout(5*time.Minute), resource.WithPollInterval(time.Second),
			resource.WithProgressCallback(progress),
		))
}

func detachWithClientsDetachVolume(ctx context.Context, nova, cinder *gophercloud.ServiceClient) (*blockstorage.DetachVolumeResult, error) {
	return blockstorage.DetachVolume(ctx, nova, cinder,
		blockstorage.DetachVolumeRequest{
			Server: resource.Name("worker"), Volume: resource.Name("data"),
		})
}
