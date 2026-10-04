package gophercloudsdk

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/cloudread"
	"gophercloudsdk/internal/cloudsnapshot"
	"gophercloudsdk/request"
)

// ListVolumeBackups prepares owned policy before selecting cached Cinder v3.
func (c *Connection) ListVolumeBackups(ctx context.Context, options ...blockstorage.VolumeBackupListOption) (*blockstorage.ListVolumeBackupsResult, error) {
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, wrapConnectionBackup(ctx, "ListVolumeBackups", err)
	}
	policy, err := blockstorage.PrepareVolumeBackupListOptions(ctx, options...)
	if err == nil {
		err = cloudsnapshot.ValidateBackupListOptions(policy)
	}
	if err != nil {
		return nil, wrapConnectionBackup(ctx, "ListVolumeBackups", err)
	}
	client, err := c.cinderCloudReadClient(ctx, &policy.Location)
	if err != nil {
		return nil, wrapConnectionBackup(ctx, "ListVolumeBackups", err)
	}
	return blockstorage.ListVolumeBackups(ctx, client, blockstorage.WithVolumeBackupListOptions(policy))
}

func (c *Connection) SearchVolumeBackups(ctx context.Context, input blockstorage.SearchVolumeBackupsRequest, options ...blockstorage.VolumeBackupSearchOption) (*blockstorage.SearchVolumeBackupsResult, error) {
	client, policy, err := c.prepareBackupSearch(ctx, options)
	if err != nil {
		return nil, wrapConnectionBackup(ctx, "SearchVolumeBackups", err)
	}
	return blockstorage.SearchVolumeBackups(ctx, client, input, blockstorage.WithVolumeBackupSearchOptions(policy))
}

func (c *Connection) GetVolumeBackup(ctx context.Context, input blockstorage.GetVolumeBackupRequest, options ...blockstorage.VolumeBackupSearchOption) (*blockstorage.GetVolumeBackupResult, error) {
	client, policy, err := c.prepareBackupSearch(ctx, options)
	if err != nil {
		return nil, wrapConnectionBackup(ctx, "GetVolumeBackup", err)
	}
	return blockstorage.GetVolumeBackup(ctx, client, input, blockstorage.WithVolumeBackupSearchOptions(policy))
}

func (c *Connection) prepareBackupSearch(ctx context.Context, options []blockstorage.VolumeBackupSearchOption) (*gophercloud.ServiceClient, blockstorage.VolumeBackupSearchOpts, error) {
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, blockstorage.VolumeBackupSearchOpts{}, err
	}
	policy, err := blockstorage.PrepareVolumeBackupSearchOptions(ctx, options...)
	if err != nil {
		return nil, blockstorage.VolumeBackupSearchOpts{}, err
	}
	client, err := c.cinderCloudReadClient(ctx, &policy.Location)
	return client, policy, err
}

func wrapConnectionBackup(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "volume backup", cloudread.ContextError(ctx, err))
}
