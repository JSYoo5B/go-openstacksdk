package openstack

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudvolume"

	"github.com/gophercloud/gophercloud/v2"
)

// cloudVolumeDependencies fills the volume workflow dependencies; a nil
// Connection yields the zero value.
func (c *Connection) cloudVolumeDependencies() cloudvolume.Dependencies {
	if c == nil {
		return cloudvolume.Dependencies{}
	}
	return cloudvolume.Dependencies{
		Location:       c.CurrentLocation,
		Provider:       func() *gophercloud.ProviderClient { return c.provider },
		BlockStorageV3: c.BlockStorageV3,
		ComputeV2:      c.ComputeV2,
		ImageV2:        c.ImageV2,
		IdentityV3:     c.IdentityV3,
		Image:          c.Image,
		ImageFormat:    func() json.RawMessage { return c.options.imageCreatePolicy.ImageFormat },
	}
}

// ListVolumes reads the full detailed list through cached Cinder v3.
func (c *Connection) ListVolumes(ctx context.Context, options ...blockstorage.VolumeReadOption) (*blockstorage.ListVolumesResult, error) {
	return cloudvolume.ListVolumes(ctx, c.cloudVolumeDependencies(), options...)
}

// GetVolumeByID retains all rejected member statuses; it does not search names.
func (c *Connection) GetVolumeByID(ctx context.Context, input blockstorage.GetVolumeByIDRequest, options ...blockstorage.VolumeReadOption) (*blockstorage.GetVolumeResult, error) {
	return cloudvolume.GetVolumeByID(ctx, c.cloudVolumeDependencies(), input, options...)
}

// VolumeExists returns false only after a completed ordinary lookup is absent.
func (c *Connection) VolumeExists(ctx context.Context, input blockstorage.VolumeExistsRequest, options ...blockstorage.VolumeReadOption) (*blockstorage.VolumeExistsResult, error) {
	return cloudvolume.VolumeExists(ctx, c.cloudVolumeDependencies(), input, options...)
}

// SearchVolumes selects cached Cinder v3 after applying originals once. The
// connection supplies its own recorded cloud location unless overridden.
func (c *Connection) SearchVolumes(ctx context.Context, input blockstorage.SearchVolumesRequest, options ...blockstorage.VolumeSearchOption) (*blockstorage.SearchVolumesResult, error) {
	return cloudvolume.SearchVolumes(ctx, c.cloudVolumeDependencies(), input, options...)
}

func (c *Connection) GetVolume(ctx context.Context, input blockstorage.GetVolumeRequest, options ...blockstorage.VolumeSearchOption) (*blockstorage.GetVolumeResult, error) {
	return cloudvolume.GetVolume(ctx, c.cloudVolumeDependencies(), input, options...)
}

// GetVolumes materializes Cinder's detail list before local server association
// selection. It prepares originals once and selects only cached Cinder v3.
func (c *Connection) GetVolumes(ctx context.Context, input blockstorage.GetVolumesRequest, options ...blockstorage.GetVolumesOption) (*blockstorage.GetVolumesResult, error) {
	return cloudvolume.GetVolumes(ctx, c.cloudVolumeDependencies(), input, options...)
}

// GetVolumeID uses cached Cinder v3 to resolve an exact name or ID and preserves
// the response's untyped ID. Original read options run once before selection.
func (c *Connection) GetVolumeID(ctx context.Context, input blockstorage.GetVolumeIDRequest, options ...blockstorage.VolumeReadOption) (*blockstorage.GetVolumeIDResult, error) {
	return cloudvolume.GetVolumeID(ctx, c.cloudVolumeDependencies(), input, options...)
}

// GetVolumeLimits owns defaults and cached service selection. Original options
// run once; a requested project's Identity lookup completes before selecting
// Cinder. No Identity getter runs for an omitted project.
func (c *Connection) GetVolumeLimits(ctx context.Context, input blockstorage.GetVolumeLimitsRequest, options ...blockstorage.GetVolumeLimitsOption) (*blockstorage.GetVolumeLimitsResult, error) {
	return cloudvolume.GetVolumeLimits(ctx, c.cloudVolumeDependencies(), input, options...)
}

// CreateVolume prepares options once, then shares cached Cinder v3 and selects
// Glance v2 only for an image Name. Explicit image IDs need no Glance access.
// Original callbacks precede service selection; configuration must be prepared
// before concurrent use. Creation, waiting and bootable acknowledgement retain
// their separate accepted results on later failures.
func (c *Connection) CreateVolume(ctx context.Context, input blockstorage.CreateVolumeRequest, options ...blockstorage.CreateVolumeOption) (*blockstorage.CreateVolumeResult, error) {
	return cloudvolume.CreateVolume(ctx, c.cloudVolumeDependencies(), input, options...)
}

// DeleteVolume prepares original callbacks once before selecting cached Cinder
// v3. Even Wait(false) performs initial lookup. Configuration must be ready
// before concurrent use; accepted phases remain available after later errors.
func (c *Connection) DeleteVolume(ctx context.Context, input blockstorage.DeleteVolumeRequest, options ...blockstorage.DeleteVolumeOption) (*blockstorage.DeleteVolumeResult, error) {
	return cloudvolume.DeleteVolume(ctx, c.cloudVolumeDependencies(), input, options...)
}

// UpdateVolume resolves one volume through cached Cinder v3 and applies only
// changed known fields. Original options run once before selecting the service.
func (c *Connection) UpdateVolume(ctx context.Context, input blockstorage.UpdateVolumeRequest, options ...blockstorage.UpdateVolumeOption) (*blockstorage.UpdateVolumeResult, error) {
	return cloudvolume.UpdateVolume(ctx, c.cloudVolumeDependencies(), input, options...)
}

// SetVolumeBootable defaults to true unless explicitly disabled. Applied owns
// the opaque accepted acknowledgement, without inferring server completion.
func (c *Connection) SetVolumeBootable(ctx context.Context, input blockstorage.SetVolumeBootableRequest, options ...blockstorage.SetVolumeBootableOption) (*blockstorage.SetVolumeBootableResult, error) {
	return cloudvolume.SetVolumeBootable(ctx, c.cloudVolumeDependencies(), input, options...)
}

// AttachVolume shares this Connection's cached Nova v2 and Cinder v3 clients.
// Client initialization, including microversion discovery, may precede option
// validation. The workflow validates its complete policy before name lookup,
// the fresh Cinder guard, attachment creation or polling.
func (c *Connection) AttachVolume(ctx context.Context, input blockstorage.AttachVolumeRequest, options ...blockstorage.AttachVolumeOption) (*blockstorage.AttachVolumeResult, error) {
	return cloudvolume.AttachVolume(ctx, c.cloudVolumeDependencies(), input, options...)
}

// DetachVolume shares cached service clients. Options are prepared once before
// service selection. Cinder is selected only for waiting or a volume name;
// explicit IDs with Wait(false) require only Nova. Service discovery may then
// run before lookup or removal. Configuration must be ready before concurrent use.
func (c *Connection) DetachVolume(ctx context.Context, input blockstorage.DetachVolumeRequest, options ...blockstorage.DetachVolumeOption) (*blockstorage.DetachVolumeResult, error) {
	return cloudvolume.DetachVolume(ctx, c.cloudVolumeDependencies(), input, options...)
}

// AttachCinderVolume prepares originals once and uses cached Cinder directly.
func (c *Connection) AttachCinderVolume(ctx context.Context, input blockstorage.VolumeActionRequest, mountpoint string, options ...blockstorage.CinderVolumeAttachOption) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.AttachCinderVolume(ctx, c.cloudVolumeDependencies(), input, mountpoint, options...)
}

// DetachCinderVolume performs one direct action without a Nova detach workflow.
func (c *Connection) DetachCinderVolume(ctx context.Context, input blockstorage.VolumeActionRequest, attachmentID string, options ...blockstorage.CinderVolumeDetachOption) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.DetachCinderVolume(ctx, c.cloudVolumeDependencies(), input, attachmentID, options...)
}
