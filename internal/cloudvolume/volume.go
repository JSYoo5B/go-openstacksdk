package cloudvolume

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudlimits"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// ListVolumes reads the full detailed list through cached Cinder v3.
func ListVolumes(ctx context.Context, deps Dependencies, options ...blockstorage.VolumeReadOption) (*blockstorage.ListVolumesResult, error) {
	client, policy, err := prepareVolumeRead(ctx, deps, options)
	if err != nil {
		return nil, wrapVolume(ctx, "ListVolumes", err)
	}
	return blockstorage.ListVolumes(ctx, client, blockstorage.WithVolumeReadOptions(policy))
}

// GetVolumeByID retains all rejected member statuses; it does not search names.
func GetVolumeByID(ctx context.Context, deps Dependencies, input blockstorage.GetVolumeByIDRequest, options ...blockstorage.VolumeReadOption) (*blockstorage.GetVolumeResult, error) {
	client, policy, err := prepareVolumeRead(ctx, deps, options)
	if err != nil {
		return nil, wrapVolume(ctx, "GetVolumeByID", err)
	}
	return blockstorage.GetVolumeByID(ctx, client, input, blockstorage.WithVolumeReadOptions(policy))
}

// VolumeExists returns false only after a completed ordinary lookup is absent.
func VolumeExists(ctx context.Context, deps Dependencies, input blockstorage.VolumeExistsRequest, options ...blockstorage.VolumeReadOption) (*blockstorage.VolumeExistsResult, error) {
	client, policy, err := prepareVolumeRead(ctx, deps, options)
	if err != nil {
		return nil, wrapVolume(ctx, "VolumeExists", err)
	}
	return blockstorage.VolumeExists(ctx, client, input, blockstorage.WithVolumeReadOptions(policy))
}

// GetVolumeID resolves an exact name or ID and preserves the untyped ID.
func GetVolumeID(ctx context.Context, deps Dependencies, input blockstorage.GetVolumeIDRequest, options ...blockstorage.VolumeReadOption) (*blockstorage.GetVolumeIDResult, error) {
	client, policy, err := prepareVolumeRead(ctx, deps, options)
	if err != nil {
		return nil, wrapVolume(ctx, "GetVolumeID", err)
	}
	return blockstorage.GetVolumeID(ctx, client, input, blockstorage.WithVolumeReadOptions(policy))
}

func prepareVolumeRead(ctx context.Context, deps Dependencies, options []blockstorage.VolumeReadOption) (*gophercloud.ServiceClient, blockstorage.VolumeReadOpts, error) {
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, blockstorage.VolumeReadOpts{}, err
	}
	policy, err := blockstorage.PrepareVolumeReadOptions(ctx, options...)
	if err != nil {
		return nil, policy, err
	}
	if policy.Location == nil {
		location, err := deps.Location()
		if err != nil {
			return nil, blockstorage.VolumeReadOpts{}, err
		}
		policy.Location = &location
	}
	service, err := deps.BlockStorageV3(ctx)
	if err != nil {
		return nil, blockstorage.VolumeReadOpts{}, err
	}
	return service.RawClient(), policy, nil
}

// SearchVolumes selects cached Cinder v3 after applying originals once.
func SearchVolumes(ctx context.Context, deps Dependencies, input blockstorage.SearchVolumesRequest, options ...blockstorage.VolumeSearchOption) (*blockstorage.SearchVolumesResult, error) {
	client, policy, err := prepareVolumeSearch(ctx, deps, options)
	if err != nil {
		return nil, wrapVolume(ctx, "SearchVolumes", err)
	}
	return blockstorage.SearchVolumes(ctx, client, input, blockstorage.WithVolumeSearchOptions(policy))
}

// GetVolume selects cached Cinder v3 after applying originals once.
func GetVolume(ctx context.Context, deps Dependencies, input blockstorage.GetVolumeRequest, options ...blockstorage.VolumeSearchOption) (*blockstorage.GetVolumeResult, error) {
	client, policy, err := prepareVolumeSearch(ctx, deps, options)
	if err != nil {
		return nil, wrapVolume(ctx, "GetVolume", err)
	}
	return blockstorage.GetVolume(ctx, client, input, blockstorage.WithVolumeSearchOptions(policy))
}

func prepareVolumeSearch(ctx context.Context, deps Dependencies, options []blockstorage.VolumeSearchOption) (*gophercloud.ServiceClient, blockstorage.VolumeSearchOpts, error) {
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, blockstorage.VolumeSearchOpts{}, err
	}
	policy, err := blockstorage.PrepareVolumeSearchOptions(ctx, options...)
	if err != nil {
		return nil, policy, err
	}
	if policy.Location == nil {
		location, err := deps.Location()
		if err != nil {
			return nil, blockstorage.VolumeSearchOpts{}, err
		}
		policy.Location = &location
	}
	service, err := deps.BlockStorageV3(ctx)
	if err != nil {
		return nil, blockstorage.VolumeSearchOpts{}, err
	}
	return service.RawClient(), policy, nil
}

// GetVolumes materializes Cinder's detail list before local server association
// selection. It does not consume the current location.
func GetVolumes(ctx context.Context, deps Dependencies, input blockstorage.GetVolumesRequest, options ...blockstorage.GetVolumesOption) (*blockstorage.GetVolumesResult, error) {
	const operation = "GetVolumes"
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, wrapVolume(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareGetVolumesOptions(ctx, options...)
	if err != nil {
		return nil, wrapVolume(ctx, operation, err)
	}
	cinder, err := deps.BlockStorageV3(ctx)
	if err != nil {
		return nil, wrapVolume(ctx, operation, err)
	}
	return blockstorage.GetVolumes(ctx, cinder.RawClient(), input, blockstorage.WithGetVolumesOptions(policy))
}

// GetVolumeLimits runs originals once; a requested project's Identity lookup
// completes before selecting Cinder. No Identity getter runs for an omitted
// project.
func GetVolumeLimits(ctx context.Context, deps Dependencies, input blockstorage.GetVolumeLimitsRequest, options ...blockstorage.GetVolumeLimitsOption) (*blockstorage.GetVolumeLimitsResult, error) {
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, wrapVolumeLimits(ctx, err)
	}
	if err := cloudlimits.ValidateInput(ctx, input); err != nil {
		return nil, wrapVolumeLimits(ctx, err)
	}
	policy, err := blockstorage.PrepareGetVolumeLimitsOptions(ctx, options...)
	if err != nil {
		return nil, wrapVolumeLimits(ctx, err)
	}
	if policy.Location == nil {
		location, err := deps.Location()
		if err != nil {
			return nil, wrapVolumeLimits(ctx, err)
		}
		policy.Location = &location
	}
	cinder := func(ctx context.Context) (*gophercloud.ServiceClient, error) {
		service, err := deps.BlockStorageV3(ctx)
		if err != nil {
			return nil, err
		}
		if service == nil {
			return nil, fmt.Errorf("%w: Cinder service is required", resource.ErrInvalidOption)
		}
		return service.RawClient(), nil
	}
	identity := func(ctx context.Context) (*gophercloud.ServiceClient, error) {
		service, err := deps.IdentityV3(ctx)
		if err != nil {
			return nil, err
		}
		if service == nil {
			return nil, fmt.Errorf("%w: Identity service is required", resource.ErrInvalidOption)
		}
		return service.RawClient(), nil
	}
	return cloudlimits.ReadFrom(ctx, input, policy, cinder, identity)
}

func wrapVolumeLimits(ctx context.Context, err error) error {
	return request.Wrap("GetVolumeLimits", "volume limits", cloudread.ContextError(ctx, err))
}

// CreateVolume prepares options once, then shares cached Cinder v3 and selects
// Glance v2 only for an image Name.
func CreateVolume(ctx context.Context, deps Dependencies, input blockstorage.CreateVolumeRequest, options ...blockstorage.CreateVolumeOption) (*blockstorage.CreateVolumeResult, error) {
	const operation = "CreateVolume"
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, wrapVolume(ctx, operation, err)
	}
	if input.Image != nil {
		reference := *input.Image
		input.Image = &reference
		if err := validateRef(reference); err != nil {
			return nil, wrapVolume(ctx, operation, err)
		}
	}
	policy, err := blockstorage.PrepareCreateVolumeOptions(ctx, options...)
	if err != nil {
		return nil, wrapVolume(ctx, operation, err)
	}
	cinder, err := deps.BlockStorageV3(ctx)
	if err != nil {
		return nil, wrapVolume(ctx, operation, err)
	}
	var glanceClient *gophercloud.ServiceClient
	if input.Image != nil && input.Image.IsName() {
		glance, err := deps.ImageV2(ctx)
		if err != nil {
			return nil, wrapVolume(ctx, operation, err)
		}
		glanceClient = glance.RawClient()
	}
	return blockstorage.CreateVolume(ctx, cinder.RawClient(), glanceClient, input, blockstorage.WithCreateVolumeOptions(policy))
}

// DeleteVolume prepares original callbacks once before selecting cached Cinder
// v3. Even Wait(false) performs initial lookup.
func DeleteVolume(ctx context.Context, deps Dependencies, input blockstorage.DeleteVolumeRequest, options ...blockstorage.DeleteVolumeOption) (*blockstorage.DeleteVolumeResult, error) {
	const operation = "DeleteVolume"
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, wrapVolume(ctx, operation, err)
	}
	if err := validateRef(input.Volume); err != nil {
		return nil, wrapVolume(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareDeleteVolumeOptions(ctx, options...)
	if err != nil {
		return nil, wrapVolume(ctx, operation, err)
	}
	cinder, err := deps.BlockStorageV3(ctx)
	if err != nil {
		return nil, wrapVolume(ctx, operation, err)
	}
	return blockstorage.DeleteVolume(ctx, cinder.RawClient(), input, blockstorage.WithDeleteVolumeOptions(policy))
}

// UpdateVolume resolves one volume through cached Cinder v3 and applies only
// changed known fields.
func UpdateVolume(ctx context.Context, deps Dependencies, input blockstorage.UpdateVolumeRequest, options ...blockstorage.UpdateVolumeOption) (*blockstorage.UpdateVolumeResult, error) {
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, wrapVolume(ctx, "UpdateVolume", err)
	}
	policy, err := blockstorage.PrepareUpdateVolumeOptions(ctx, options...)
	if err != nil {
		return nil, wrapVolume(ctx, "UpdateVolume", err)
	}
	client, location, err := ownedLocationClient(ctx, deps, policy.Location)
	if err != nil {
		return nil, wrapVolume(ctx, "UpdateVolume", err)
	}
	policy.Location = location
	return blockstorage.UpdateVolume(ctx, client, input, blockstorage.WithUpdateVolumeOptions(policy))
}

// SetVolumeBootable defaults to true unless explicitly disabled.
func SetVolumeBootable(ctx context.Context, deps Dependencies, input blockstorage.SetVolumeBootableRequest, options ...blockstorage.SetVolumeBootableOption) (*blockstorage.SetVolumeBootableResult, error) {
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, wrapVolume(ctx, "SetVolumeBootable", err)
	}
	policy, err := blockstorage.PrepareSetVolumeBootableOptions(ctx, options...)
	if err != nil {
		return nil, wrapVolume(ctx, "SetVolumeBootable", err)
	}
	client, location, err := ownedLocationClient(ctx, deps, policy.Location)
	if err != nil {
		return nil, wrapVolume(ctx, "SetVolumeBootable", err)
	}
	policy.Location = location
	return blockstorage.SetVolumeBootable(ctx, client, input, blockstorage.WithSetVolumeBootableOptions(policy))
}

// AttachVolume shares cached Nova v2 and Cinder v3 clients. Client
// initialization may precede option validation by the workflow.
func AttachVolume(ctx context.Context, deps Dependencies, input blockstorage.AttachVolumeRequest, options ...blockstorage.AttachVolumeOption) (*blockstorage.AttachVolumeResult, error) {
	const operation = "AttachVolume"
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, wrapVolume(ctx, operation, err)
	}
	for _, ref := range []resource.Ref{input.Server, input.Volume} {
		if err := validateRef(ref); err != nil {
			return nil, wrapVolume(ctx, operation, err)
		}
	}
	nova, err := deps.ComputeV2(ctx)
	if err != nil {
		return nil, wrapVolume(ctx, operation, err)
	}
	cinder, err := deps.BlockStorageV3(ctx)
	if err != nil {
		return nil, wrapVolume(ctx, operation, err)
	}
	return blockstorage.AttachVolume(ctx, nova.RawClient(), cinder.RawClient(), input, options...)
}

// DetachVolume prepares options once before service selection. Cinder is
// selected only for waiting or a volume name.
func DetachVolume(ctx context.Context, deps Dependencies, input blockstorage.DetachVolumeRequest, options ...blockstorage.DetachVolumeOption) (*blockstorage.DetachVolumeResult, error) {
	const operation = "DetachVolume"
	if err := mutationPreflight(ctx, deps); err != nil {
		return nil, wrapVolume(ctx, operation, err)
	}
	for _, ref := range []resource.Ref{input.Server, input.Volume} {
		if err := validateRef(ref); err != nil {
			return nil, wrapVolume(ctx, operation, err)
		}
	}
	policy, err := blockstorage.PrepareDetachVolumeOptions(ctx, options...)
	if err != nil {
		return nil, wrapVolume(ctx, operation, err)
	}
	nova, err := deps.ComputeV2(ctx)
	if err != nil {
		return nil, wrapVolume(ctx, operation, err)
	}
	var cinderClient *gophercloud.ServiceClient
	if *policy.Wait || input.Volume.IsName() {
		cinder, err := deps.BlockStorageV3(ctx)
		if err != nil {
			return nil, wrapVolume(ctx, operation, err)
		}
		cinderClient = cinder.RawClient()
	}
	return blockstorage.DetachVolume(ctx, nova.RawClient(), cinderClient, input, blockstorage.WithDetachVolumeOptions(policy))
}

// AttachCinderVolume prepares originals once and uses cached Cinder directly.
func AttachCinderVolume(ctx context.Context, deps Dependencies, input blockstorage.VolumeActionRequest, mountpoint string, options ...blockstorage.CinderVolumeAttachOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "AttachCinderVolume"
	if err := flagPreflight(ctx, deps); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareCinderVolumeAttachOptions(ctx, options...)
	if err == nil {
		err = cinderaction.ValidateMountpoint(mountpoint)
	}
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := flagClient(ctx, deps, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.DirectAttach(ctx, client, input.VolumeID, mountpoint, cinderaction.WithDirectAttachOptions(policy))
}

// DetachCinderVolume performs one direct action without a Nova detach workflow.
func DetachCinderVolume(ctx context.Context, deps Dependencies, input blockstorage.VolumeActionRequest, attachmentID string, options ...blockstorage.CinderVolumeDetachOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "DetachCinderVolume"
	if err := flagPreflight(ctx, deps); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareCinderVolumeDetachOptions(ctx, options...)
	if err == nil {
		err = cinderaction.ValidateAttachmentID(attachmentID)
	}
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := flagClient(ctx, deps, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.DirectDetach(ctx, client, input.VolumeID, attachmentID, cinderaction.WithDirectDetachOptions(policy))
}
