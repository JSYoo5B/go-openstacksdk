package openstack

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
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

// SetVolumeBootableStatus uses only cached Cinder and a required bool.
func (c *Connection) SetVolumeBootableStatus(ctx context.Context, input blockstorage.VolumeActionRequest, bootable bool) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.SetVolumeBootableStatus(ctx, c.cloudVolumeDependencies(), input, bootable)
}

// SetVolumeReadonly prepares originals once before selecting cached Cinder.
func (c *Connection) SetVolumeReadonly(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeReadonlyOption) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.SetVolumeReadonly(ctx, c.cloudVolumeDependencies(), input, options...)
}

// ReserveVolume selects cached Cinder only; no CurrentLocation is consumed.
func (c *Connection) ReserveVolume(ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.StateAction(ctx, c.cloudVolumeDependencies(), input, cinderaction.Reserve)
}

func (c *Connection) UnreserveVolume(ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.StateAction(ctx, c.cloudVolumeDependencies(), input, cinderaction.Unreserve)
}

func (c *Connection) BeginVolumeDetaching(ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.StateAction(ctx, c.cloudVolumeDependencies(), input, cinderaction.BeginDetaching)
}

func (c *Connection) AbortVolumeDetaching(ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.StateAction(ctx, c.cloudVolumeDependencies(), input, cinderaction.AbortDetaching)
}

func (c *Connection) ExtendVolume(ctx context.Context, input blockstorage.VolumeActionRequest, newSize int) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.ExtendVolume(ctx, c.cloudVolumeDependencies(), input, newSize)
}

// RetypeVolume prepares originals once before selecting cached Cinder.
func (c *Connection) RetypeVolume(ctx context.Context, input blockstorage.VolumeActionRequest, newType string, options ...blockstorage.VolumeRetypeOption) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.RetypeVolume(ctx, c.cloudVolumeDependencies(), input, newType, options...)
}

// CompleteVolumeExtend acknowledges one action without polling the volume.
func (c *Connection) CompleteVolumeExtend(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeExtendCompletionOption) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.CompleteVolumeExtend(ctx, c.cloudVolumeDependencies(), input, options...)
}

func (c *Connection) ResetVolumeStatus(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeStatusResetOption) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.ResetVolumeStatus(ctx, c.cloudVolumeDependencies(), input, options...)
}

func (c *Connection) MigrateVolume(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeMigrationOption) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.MigrateVolume(ctx, c.cloudVolumeDependencies(), input, options...)
}

func (c *Connection) CompleteVolumeMigration(ctx context.Context, input blockstorage.VolumeActionRequest, newVolume string, options ...blockstorage.VolumeMigrationCompletionOption) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.CompleteVolumeMigration(ctx, c.cloudVolumeDependencies(), input, newVolume, options...)
}

// RevertVolumeToSnapshot selects cached Cinder after local input admission.
func (c *Connection) RevertVolumeToSnapshot(ctx context.Context, input blockstorage.VolumeActionRequest, snapshotID string) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.RevertVolumeToSnapshot(ctx, c.cloudVolumeDependencies(), input, snapshotID)
}

// UnmanageVolume uses cached Cinder and the SDK-owned null action policy.
func (c *Connection) UnmanageVolume(ctx context.Context, input blockstorage.VolumeActionRequest) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.UnmanageVolume(ctx, c.cloudVolumeDependencies(), input)
}

// ListVolumeBackups prepares owned policy before selecting cached Cinder v3.
func (c *Connection) ListVolumeBackups(ctx context.Context, options ...blockstorage.VolumeBackupListOption) (*blockstorage.ListVolumeBackupsResult, error) {
	return cloudvolume.ListVolumeBackups(ctx, c.cloudVolumeDependencies(), options...)
}

func (c *Connection) SearchVolumeBackups(ctx context.Context, input blockstorage.SearchVolumeBackupsRequest, options ...blockstorage.VolumeBackupSearchOption) (*blockstorage.SearchVolumeBackupsResult, error) {
	return cloudvolume.SearchVolumeBackups(ctx, c.cloudVolumeDependencies(), input, options...)
}

func (c *Connection) GetVolumeBackup(ctx context.Context, input blockstorage.GetVolumeBackupRequest, options ...blockstorage.VolumeBackupSearchOption) (*blockstorage.GetVolumeBackupResult, error) {
	return cloudvolume.GetVolumeBackup(ctx, c.cloudVolumeDependencies(), input, options...)
}

// CreateVolumeBackup prepares original options and validates literal input
// before selecting cached Cinder v3. Location and defaults are SDK-owned.
func (c *Connection) CreateVolumeBackup(ctx context.Context, input blockstorage.CreateVolumeBackupRequest, options ...blockstorage.CreateVolumeBackupOption) (*blockstorage.CreateVolumeBackupResult, error) {
	return cloudvolume.CreateVolumeBackup(ctx, c.cloudVolumeDependencies(), input, options...)
}

// DeleteVolumeBackup skips service/location selection for an empty input.
// Original callbacks still run once; a nil Connection is sufficient for this
// service-free no-op. Nonempty input resolves and deletes through cached v3.
func (c *Connection) DeleteVolumeBackup(ctx context.Context, input blockstorage.DeleteVolumeBackupRequest, options ...blockstorage.DeleteVolumeBackupOption) (*blockstorage.DeleteVolumeBackupResult, error) {
	return cloudvolume.DeleteVolumeBackup(ctx, c.cloudVolumeDependencies(), input, options...)
}

// RestoreVolumeBackup owns original options before location/Cinder selection.
func (c *Connection) RestoreVolumeBackup(ctx context.Context, input blockstorage.RestoreVolumeBackupRequest, options ...blockstorage.RestoreVolumeBackupOption) (*blockstorage.RestoreVolumeBackupResult, error) {
	return cloudvolume.RestoreVolumeBackup(ctx, c.cloudVolumeDependencies(), input, options...)
}

// ResetVolumeBackupStatus selects cached Cinder only and forwards opaque status.
func (c *Connection) ResetVolumeBackupStatus(ctx context.Context, input blockstorage.ResetVolumeBackupStatusRequest) (*blockstorage.ResetVolumeBackupStatusResult, error) {
	return cloudvolume.ResetVolumeBackupStatus(ctx, c.cloudVolumeDependencies(), input)
}

// ExportVolumeBackup validates one literal ID before selecting cached Cinder v3.
// Export returns opaque HTTP evidence and consumes no configured resource scope.
func (c *Connection) ExportVolumeBackup(ctx context.Context, input blockstorage.ExportVolumeBackupRequest) (*blockstorage.ExportVolumeBackupResult, error) {
	return cloudvolume.ExportVolumeBackup(ctx, c.cloudVolumeDependencies(), input)
}

// ExportVolumeBackupRecord selects only cached Cinder v3 after local preflight.
func (c *Connection) ExportVolumeBackupRecord(ctx context.Context, input blockstorage.ExportVolumeBackupRecordRequest) (*blockstorage.ExportVolumeBackupRecordResult, error) {
	return cloudvolume.ExportVolumeBackupRecord(ctx, c.cloudVolumeDependencies(), input)
}

// ImportVolumeBackup uses cached Cinder without consuming CurrentLocation.
func (c *Connection) ImportVolumeBackup(ctx context.Context, input blockstorage.ImportVolumeBackupRequest) (*blockstorage.ImportVolumeBackupResult, error) {
	return cloudvolume.ImportVolumeBackup(ctx, c.cloudVolumeDependencies(), input)
}

// ListVolumeSnapshots owns original options and location before selecting the
// connection's cached Cinder v3 service. SDK callers need no custom builder.
func (c *Connection) ListVolumeSnapshots(ctx context.Context, options ...blockstorage.VolumeSnapshotListOption) (*blockstorage.ListVolumeSnapshotsResult, error) {
	return cloudvolume.ListVolumeSnapshots(ctx, c.cloudVolumeDependencies(), options...)
}

func (c *Connection) SearchVolumeSnapshots(ctx context.Context, input blockstorage.SearchVolumeSnapshotsRequest, options ...blockstorage.VolumeSnapshotSearchOption) (*blockstorage.SearchVolumeSnapshotsResult, error) {
	return cloudvolume.SearchVolumeSnapshots(ctx, c.cloudVolumeDependencies(), input, options...)
}

func (c *Connection) GetVolumeSnapshot(ctx context.Context, input blockstorage.GetVolumeSnapshotRequest, options ...blockstorage.VolumeSnapshotSearchOption) (*blockstorage.GetVolumeSnapshotResult, error) {
	return cloudvolume.GetVolumeSnapshot(ctx, c.cloudVolumeDependencies(), input, options...)
}

func (c *Connection) GetVolumeSnapshotByID(ctx context.Context, input blockstorage.GetVolumeSnapshotByIDRequest, options ...blockstorage.VolumeSnapshotReadOption) (*blockstorage.GetVolumeSnapshotResult, error) {
	return cloudvolume.GetVolumeSnapshotByID(ctx, c.cloudVolumeDependencies(), input, options...)
}

// CreateVolumeSnapshot prepares original options and validates literal input
// before selecting cached Cinder v3. Location and defaults are SDK-owned.
func (c *Connection) CreateVolumeSnapshot(ctx context.Context, input blockstorage.CreateVolumeSnapshotRequest, options ...blockstorage.CreateVolumeSnapshotOption) (*blockstorage.CreateVolumeSnapshotResult, error) {
	return cloudvolume.CreateVolumeSnapshot(ctx, c.cloudVolumeDependencies(), input, options...)
}

// DeleteVolumeSnapshot skips service/location selection for an empty input.
// Original callbacks still run once; a nil Connection is sufficient for this
// service-free no-op. Nonempty input resolves and deletes through cached v3.
func (c *Connection) DeleteVolumeSnapshot(ctx context.Context, input blockstorage.DeleteVolumeSnapshotRequest, options ...blockstorage.DeleteVolumeSnapshotOption) (*blockstorage.DeleteVolumeSnapshotResult, error) {
	return cloudvolume.DeleteVolumeSnapshot(ctx, c.cloudVolumeDependencies(), input, options...)
}

// ListVolumeTypes applies originals once before selecting cached Cinder v3 and
// supplies the connection's recorded location unless explicitly overridden.
func (c *Connection) ListVolumeTypes(ctx context.Context, options ...blockstorage.VolumeTypeReadOption) (*blockstorage.ListVolumeTypesResult, error) {
	return cloudvolume.ListVolumeTypes(ctx, c.cloudVolumeDependencies(), options...)
}

func (c *Connection) SearchVolumeTypes(ctx context.Context, input blockstorage.SearchVolumeTypesRequest, options ...blockstorage.VolumeTypeSearchOption) (*blockstorage.SearchVolumeTypesResult, error) {
	return cloudvolume.SearchVolumeTypes(ctx, c.cloudVolumeDependencies(), input, options...)
}

func (c *Connection) GetVolumeType(ctx context.Context, input blockstorage.GetVolumeTypeRequest, options ...blockstorage.VolumeTypeSearchOption) (*blockstorage.GetVolumeTypeResult, error) {
	return cloudvolume.GetVolumeType(ctx, c.cloudVolumeDependencies(), input, options...)
}

// GetVolumeTypeAccess resolves an exact Type through cached Cinder v3, then
// reads its raw access value. Originals run once before selecting the service.
func (c *Connection) GetVolumeTypeAccess(ctx context.Context, input blockstorage.GetVolumeTypeAccessRequest, options ...blockstorage.VolumeTypeReadOption) (*blockstorage.GetVolumeTypeAccessResult, error) {
	return cloudvolume.GetVolumeTypeAccess(ctx, c.cloudVolumeDependencies(), input, options...)
}

// AddVolumeTypeAccess forwards the literal ProjectID after Type resolution. The
// returned Applied proof acknowledges the action; it does not infer a change.
func (c *Connection) AddVolumeTypeAccess(ctx context.Context, input blockstorage.VolumeTypeAccessRequest, options ...blockstorage.VolumeTypeReadOption) (*blockstorage.VolumeTypeAccessActionResult, error) {
	return cloudvolume.AddVolumeTypeAccess(ctx, c.cloudVolumeDependencies(), input, options...)
}

// RemoveVolumeTypeAccess forwards the literal ProjectID through cached Cinder.
// Originals and the owned location follow the same policy as ordinary Type reads.
func (c *Connection) RemoveVolumeTypeAccess(ctx context.Context, input blockstorage.VolumeTypeAccessRequest, options ...blockstorage.VolumeTypeReadOption) (*blockstorage.VolumeTypeAccessActionResult, error) {
	return cloudvolume.RemoveVolumeTypeAccess(ctx, c.cloudVolumeDependencies(), input, options...)
}

// VolumeImageCreateRequest mirrors Cinder v3 Proxy create_image. Empty formats
// are Python-falsey: DiskFormat uses the cloud image_format setting and
// ContainerFormat uses "bare". AllowDuplicates is sent as the upload force flag.
// The pinned Proxy also accepts wait and timeout but never reads them.
type VolumeImageCreateRequest struct {
	Name            string
	VolumeID        string
	AllowDuplicates bool
	ContainerFormat string
	DiskFormat      string
}

// VolumeImageCreateResult keeps the Cinder upload evidence separately from the
// synchronized ImageRecord built from its image_id without a Glance request.
type VolumeImageCreateResult struct {
	Upload *blockstorage.VolumeImageUploadResult
	Image  *image.ImageRecord
}

// CreateVolumeImageRecord implements Cinder v3 Proxy create_image: one volume
// upload-to-image action, then Image.existing(id=image_id) with the current
// Connection location. It does not wait for the image or fetch it.
func (c *Connection) CreateVolumeImageRecord(ctx context.Context, input VolumeImageCreateRequest) (*VolumeImageCreateResult, error) {
	result, err := cloudvolume.CreateVolumeImageRecord(ctx, c.cloudVolumeDependencies(), cloudvolume.ImageCreateRequest(input))
	return (*VolumeImageCreateResult)(result), err
}

// SetVolumeImageMetadata prepares caller options once before selecting Cinder.
func (c *Connection) SetVolumeImageMetadata(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeImageMetadataOption) (*blockstorage.VolumeActionResult, error) {
	return cloudvolume.SetVolumeImageMetadata(ctx, c.cloudVolumeDependencies(), input, options...)
}

// DeleteVolumeImageMetadata uses the cached Cinder service and retains partial
// action acknowledgements. Omitted keys select the current image-metadata map.
func (c *Connection) DeleteVolumeImageMetadata(ctx context.Context, input blockstorage.VolumeActionRequest, options ...blockstorage.VolumeImageMetadataDeleteOption) (*blockstorage.VolumeImageMetadataDeleteResult, error) {
	return cloudvolume.DeleteVolumeImageMetadata(ctx, c.cloudVolumeDependencies(), input, options...)
}

// UploadVolumeToImage prepares originals once before selecting cached Cinder.
// It sends a volume action rather than uploading bytes through a Glance client.
func (c *Connection) UploadVolumeToImage(ctx context.Context, input blockstorage.VolumeActionRequest, imageName string, options ...blockstorage.VolumeImageUploadOption) (*blockstorage.VolumeImageUploadResult, error) {
	return cloudvolume.UploadVolumeToImage(ctx, c.cloudVolumeDependencies(), input, imageName, options...)
}
