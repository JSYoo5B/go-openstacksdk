package openstack

import (
	"context"
	"errors"
	"fmt"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderaction"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudlimits"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"

	"github.com/gophercloud/gophercloud/v2"
)

// ListVolumes reads the full detailed list through cached Cinder v3.
func (c *Connection) ListVolumes(ctx context.Context, options ...blockstorage.VolumeReadOption) (*blockstorage.ListVolumesResult, error) {
	client, policy, err := c.prepareVolumeRead(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "ListVolumes", err)
	}
	return blockstorage.ListVolumes(ctx, client, blockstorage.WithVolumeReadOptions(policy))
}

// GetVolumeByID retains all rejected member statuses; it does not search names.
func (c *Connection) GetVolumeByID(ctx context.Context, input blockstorage.GetVolumeByIDRequest, options ...blockstorage.VolumeReadOption) (*blockstorage.GetVolumeResult, error) {
	client, policy, err := c.prepareVolumeRead(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "GetVolumeByID", err)
	}
	return blockstorage.GetVolumeByID(ctx, client, input, blockstorage.WithVolumeReadOptions(policy))
}

// VolumeExists returns false only after a completed ordinary lookup is absent.
func (c *Connection) VolumeExists(ctx context.Context, input blockstorage.VolumeExistsRequest, options ...blockstorage.VolumeReadOption) (*blockstorage.VolumeExistsResult, error) {
	client, policy, err := c.prepareVolumeRead(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "VolumeExists", err)
	}
	return blockstorage.VolumeExists(ctx, client, input, blockstorage.WithVolumeReadOptions(policy))
}

func (c *Connection) prepareVolumeRead(ctx context.Context, options []blockstorage.VolumeReadOption) (*gophercloud.ServiceClient, blockstorage.VolumeReadOpts, error) {
	if ctx == nil {
		return nil, blockstorage.VolumeReadOpts{}, fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return nil, blockstorage.VolumeReadOpts{}, err
	}
	if c == nil {
		return nil, blockstorage.VolumeReadOpts{}, fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption)
	}
	policy, err := blockstorage.PrepareVolumeReadOptions(ctx, options...)
	if err != nil {
		return nil, policy, err
	}
	if policy.Location == nil {
		location, err := c.CurrentLocation()
		if err != nil {
			return nil, blockstorage.VolumeReadOpts{}, err
		}
		policy.Location = &location
	}
	service, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, blockstorage.VolumeReadOpts{}, err
	}
	return service.RawClient(), policy, nil
}

// SearchVolumes selects cached Cinder v3 after applying originals once. The
// connection supplies its own recorded cloud location unless overridden.
func (c *Connection) SearchVolumes(ctx context.Context, input blockstorage.SearchVolumesRequest, options ...blockstorage.VolumeSearchOption) (*blockstorage.SearchVolumesResult, error) {
	client, policy, err := c.prepareVolumeSearch(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "SearchVolumes", err)
	}
	return blockstorage.SearchVolumes(ctx, client, input, blockstorage.WithVolumeSearchOptions(policy))
}

func (c *Connection) GetVolume(ctx context.Context, input blockstorage.GetVolumeRequest, options ...blockstorage.VolumeSearchOption) (*blockstorage.GetVolumeResult, error) {
	client, policy, err := c.prepareVolumeSearch(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "GetVolume", err)
	}
	return blockstorage.GetVolume(ctx, client, input, blockstorage.WithVolumeSearchOptions(policy))
}

func (c *Connection) prepareVolumeSearch(ctx context.Context, options []blockstorage.VolumeSearchOption) (*gophercloud.ServiceClient, blockstorage.VolumeSearchOpts, error) {
	if ctx == nil {
		return nil, blockstorage.VolumeSearchOpts{}, fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return nil, blockstorage.VolumeSearchOpts{}, err
	}
	if c == nil {
		return nil, blockstorage.VolumeSearchOpts{}, fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption)
	}
	policy, err := blockstorage.PrepareVolumeSearchOptions(ctx, options...)
	if err != nil {
		return nil, policy, err
	}
	if policy.Location == nil {
		location, err := c.CurrentLocation()
		if err != nil {
			return nil, blockstorage.VolumeSearchOpts{}, err
		}
		policy.Location = &location
	}
	service, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, blockstorage.VolumeSearchOpts{}, err
	}
	return service.RawClient(), policy, nil
}

func wrapConnectionVolumeSearch(ctx context.Context, operation string, err error) error {
	if ctx != nil && ctx.Err() != nil {
		for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
			if cause != nil && !errors.Is(err, cause) {
				err = errors.Join(err, cause)
			}
		}
	}
	return request.Wrap(operation, "volume", err)
}

// GetVolumes materializes Cinder's detail list before local server association
// selection. It prepares originals once and selects only cached Cinder v3.
func (c *Connection) GetVolumes(ctx context.Context, input blockstorage.GetVolumesRequest, options ...blockstorage.GetVolumesOption) (*blockstorage.GetVolumesResult, error) {
	wrap := func(err error) error {
		if ctx != nil && ctx.Err() != nil {
			for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
				if cause != nil && !errors.Is(err, cause) {
					err = errors.Join(err, cause)
				}
			}
		}
		return request.Wrap("GetVolumes", "volume", err)
	}
	if ctx == nil {
		return nil, wrap(fmt.Errorf("%w: context is required", resource.ErrInvalidOption))
	}
	if err := ctx.Err(); err != nil {
		return nil, wrap(err)
	}
	if c == nil {
		return nil, wrap(fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption))
	}
	policy, err := blockstorage.PrepareGetVolumesOptions(ctx, options...)
	if err != nil {
		return nil, wrap(err)
	}
	cinder, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	return blockstorage.GetVolumes(ctx, cinder.RawClient(), input, blockstorage.WithGetVolumesOptions(policy))
}

// GetVolumeID uses cached Cinder v3 to resolve an exact name or ID and preserves
// the response's untyped ID. Original read options run once before selection.
func (c *Connection) GetVolumeID(ctx context.Context, input blockstorage.GetVolumeIDRequest, options ...blockstorage.VolumeReadOption) (*blockstorage.GetVolumeIDResult, error) {
	client, policy, err := c.prepareVolumeRead(ctx, options)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "GetVolumeID", err)
	}
	return blockstorage.GetVolumeID(ctx, client, input, blockstorage.WithVolumeReadOptions(policy))
}

// GetVolumeLimits owns defaults and cached service selection. Original options
// run once; a requested project's Identity lookup completes before selecting
// Cinder. No Identity getter runs for an omitted project.
func (c *Connection) GetVolumeLimits(ctx context.Context, input blockstorage.GetVolumeLimitsRequest, options ...blockstorage.GetVolumeLimitsOption) (*blockstorage.GetVolumeLimitsResult, error) {
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, wrapConnectionVolumeLimits(ctx, err)
	}
	if err := cloudlimits.ValidateInput(ctx, input); err != nil {
		return nil, wrapConnectionVolumeLimits(ctx, err)
	}
	policy, err := blockstorage.PrepareGetVolumeLimitsOptions(ctx, options...)
	if err != nil {
		return nil, wrapConnectionVolumeLimits(ctx, err)
	}
	if policy.Location == nil {
		location, err := c.CurrentLocation()
		if err != nil {
			return nil, wrapConnectionVolumeLimits(ctx, err)
		}
		policy.Location = &location
	}
	cinder := func(ctx context.Context) (*gophercloud.ServiceClient, error) {
		service, err := c.BlockStorageV3(ctx)
		if err != nil {
			return nil, err
		}
		if service == nil {
			return nil, fmt.Errorf("%w: Cinder service is required", resource.ErrInvalidOption)
		}
		return service.RawClient(), nil
	}
	identity := func(ctx context.Context) (*gophercloud.ServiceClient, error) {
		service, err := c.IdentityV3(ctx)
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

func wrapConnectionVolumeLimits(ctx context.Context, err error) error {
	return request.Wrap("GetVolumeLimits", "volume limits", cloudread.ContextError(ctx, err))
}

// CreateVolume prepares options once, then shares cached Cinder v3 and selects
// Glance v2 only for an image Name. Explicit image IDs need no Glance access.
// Original callbacks precede service selection; configuration must be prepared
// before concurrent use. Creation, waiting and bootable acknowledgement retain
// their separate accepted results on later failures.
func (c *Connection) CreateVolume(ctx context.Context, input blockstorage.CreateVolumeRequest, options ...blockstorage.CreateVolumeOption) (*blockstorage.CreateVolumeResult, error) {
	wrap := func(err error) error {
		if ctx != nil && ctx.Err() != nil {
			for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
				if cause != nil && !errors.Is(err, cause) {
					err = errors.Join(err, cause)
				}
			}
		}
		return request.Wrap("CreateVolume", "volume", err)
	}
	if ctx == nil {
		return nil, wrap(fmt.Errorf("%w: context is required", resource.ErrInvalidOption))
	}
	if err := ctx.Err(); err != nil {
		return nil, wrap(err)
	}
	if c == nil {
		return nil, wrap(fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption))
	}
	if input.Image != nil {
		reference := *input.Image
		input.Image = &reference
		ref := reference
		if err := ref.Validate(); err != nil {
			return nil, wrap(err)
		}
		if !utf8.ValidString(ref.String()) {
			return nil, wrap(fmt.Errorf("%w: reference must be valid UTF-8", resource.ErrInvalidOption))
		}
		for _, char := range ref.String() {
			if unicode.IsControl(char) || !ref.IsName() && unicode.IsSpace(char) {
				return nil, wrap(fmt.Errorf("%w: invalid reference character", resource.ErrInvalidOption))
			}
		}
	}
	policy, err := blockstorage.PrepareCreateVolumeOptions(ctx, options...)
	if err != nil {
		return nil, wrap(err)
	}
	cinder, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	var glanceClient *gophercloud.ServiceClient
	if input.Image != nil && input.Image.IsName() {
		glance, err := c.ImageV2(ctx)
		if err != nil {
			return nil, wrap(err)
		}
		glanceClient = glance.RawClient()
	}
	return blockstorage.CreateVolume(ctx, cinder.RawClient(), glanceClient, input, blockstorage.WithCreateVolumeOptions(policy))
}

// DeleteVolume prepares original callbacks once before selecting cached Cinder
// v3. Even Wait(false) performs initial lookup. Configuration must be ready
// before concurrent use; accepted phases remain available after later errors.
func (c *Connection) DeleteVolume(ctx context.Context, input blockstorage.DeleteVolumeRequest, options ...blockstorage.DeleteVolumeOption) (*blockstorage.DeleteVolumeResult, error) {
	wrap := func(err error) error {
		if ctx != nil && ctx.Err() != nil {
			for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
				if cause != nil && !errors.Is(err, cause) {
					err = errors.Join(err, cause)
				}
			}
		}
		return request.Wrap("DeleteVolume", "volume", err)
	}
	if ctx == nil {
		return nil, wrap(fmt.Errorf("%w: context is required", resource.ErrInvalidOption))
	}
	if err := ctx.Err(); err != nil {
		return nil, wrap(err)
	}
	if c == nil {
		return nil, wrap(fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption))
	}
	ref := input.Volume
	if err := ref.Validate(); err != nil {
		return nil, wrap(err)
	}
	if !utf8.ValidString(ref.String()) {
		return nil, wrap(fmt.Errorf("%w: reference must be valid UTF-8", resource.ErrInvalidOption))
	}
	for _, char := range ref.String() {
		if unicode.IsControl(char) || !ref.IsName() && unicode.IsSpace(char) {
			return nil, wrap(fmt.Errorf("%w: invalid reference character", resource.ErrInvalidOption))
		}
	}
	policy, err := blockstorage.PrepareDeleteVolumeOptions(ctx, options...)
	if err != nil {
		return nil, wrap(err)
	}
	cinder, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	return blockstorage.DeleteVolume(ctx, cinder.RawClient(), input, blockstorage.WithDeleteVolumeOptions(policy))
}

// UpdateVolume resolves one volume through cached Cinder v3 and applies only
// changed known fields. Original options run once before selecting the service.
func (c *Connection) UpdateVolume(ctx context.Context, input blockstorage.UpdateVolumeRequest, options ...blockstorage.UpdateVolumeOption) (*blockstorage.UpdateVolumeResult, error) {
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "UpdateVolume", err)
	}
	policy, err := blockstorage.PrepareUpdateVolumeOptions(ctx, options...)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "UpdateVolume", err)
	}
	client, location, err := c.volumeMutationClient(ctx, policy.Location)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "UpdateVolume", err)
	}
	policy.Location = location
	return blockstorage.UpdateVolume(ctx, client, input, blockstorage.WithUpdateVolumeOptions(policy))
}

// SetVolumeBootable defaults to true unless explicitly disabled. Applied owns
// the opaque accepted acknowledgement, without inferring server completion.
func (c *Connection) SetVolumeBootable(ctx context.Context, input blockstorage.SetVolumeBootableRequest, options ...blockstorage.SetVolumeBootableOption) (*blockstorage.SetVolumeBootableResult, error) {
	if err := c.volumeMutationPreflight(ctx); err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "SetVolumeBootable", err)
	}
	policy, err := blockstorage.PrepareSetVolumeBootableOptions(ctx, options...)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "SetVolumeBootable", err)
	}
	client, location, err := c.volumeMutationClient(ctx, policy.Location)
	if err != nil {
		return nil, wrapConnectionVolumeSearch(ctx, "SetVolumeBootable", err)
	}
	policy.Location = location
	return blockstorage.SetVolumeBootable(ctx, client, input, blockstorage.WithSetVolumeBootableOptions(policy))
}

func (c *Connection) volumeMutationPreflight(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("%w: context is required", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if c == nil {
		return fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption)
	}
	return nil
}

func (c *Connection) volumeMutationClient(ctx context.Context, override *resource.CloudLocation) (*gophercloud.ServiceClient, *resource.CloudLocation, error) {
	var location resource.CloudLocation
	if override != nil {
		location = override.Clone()
	} else {
		var err error
		location, err = c.CurrentLocation()
		if err != nil {
			return nil, nil, err
		}
	}
	service, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, nil, err
	}
	return service.RawClient(), &location, nil
}

// AttachVolume shares this Connection's cached Nova v2 and Cinder v3 clients.
// Client initialization, including microversion discovery, may precede option
// validation. The workflow validates its complete policy before name lookup,
// the fresh Cinder guard, attachment creation or polling.
func (c *Connection) AttachVolume(ctx context.Context, input blockstorage.AttachVolumeRequest, options ...blockstorage.AttachVolumeOption) (*blockstorage.AttachVolumeResult, error) {
	wrap := func(err error) error {
		if ctx != nil && ctx.Err() != nil {
			for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
				if cause != nil && !errors.Is(err, cause) {
					err = errors.Join(err, cause)
				}
			}
		}
		return request.Wrap("AttachVolume", "volume", err)
	}
	if ctx == nil {
		return nil, wrap(fmt.Errorf("%w: context is required", resource.ErrInvalidOption))
	}
	if err := ctx.Err(); err != nil {
		return nil, wrap(err)
	}
	if c == nil {
		return nil, wrap(fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption))
	}
	for _, ref := range []resource.Ref{input.Server, input.Volume} {
		if err := ref.Validate(); err != nil {
			return nil, wrap(err)
		}
		if !utf8.ValidString(ref.String()) {
			return nil, wrap(fmt.Errorf("%w: reference must be valid UTF-8", resource.ErrInvalidOption))
		}
		for _, char := range ref.String() {
			if unicode.IsControl(char) || !ref.IsName() && unicode.IsSpace(char) {
				return nil, wrap(fmt.Errorf("%w: invalid reference character", resource.ErrInvalidOption))
			}
		}
	}
	nova, err := c.ComputeV2(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	cinder, err := c.BlockStorageV3(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	return blockstorage.AttachVolume(ctx, nova.RawClient(), cinder.RawClient(), input, options...)
}

// DetachVolume shares cached service clients. Options are prepared once before
// service selection. Cinder is selected only for waiting or a volume name;
// explicit IDs with Wait(false) require only Nova. Service discovery may then
// run before lookup or removal. Configuration must be ready before concurrent use.
func (c *Connection) DetachVolume(ctx context.Context, input blockstorage.DetachVolumeRequest, options ...blockstorage.DetachVolumeOption) (*blockstorage.DetachVolumeResult, error) {
	wrap := func(err error) error {
		if ctx != nil && ctx.Err() != nil {
			for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
				if cause != nil && !errors.Is(err, cause) {
					err = errors.Join(err, cause)
				}
			}
		}
		return request.Wrap("DetachVolume", "volume", err)
	}
	if ctx == nil {
		return nil, wrap(fmt.Errorf("%w: context is required", resource.ErrInvalidOption))
	}
	if err := ctx.Err(); err != nil {
		return nil, wrap(err)
	}
	if c == nil {
		return nil, wrap(fmt.Errorf("%w: Connection is required", resource.ErrInvalidOption))
	}
	for _, ref := range []resource.Ref{input.Server, input.Volume} {
		if err := ref.Validate(); err != nil {
			return nil, wrap(err)
		}
		if !utf8.ValidString(ref.String()) {
			return nil, wrap(fmt.Errorf("%w: reference must be valid UTF-8", resource.ErrInvalidOption))
		}
		for _, char := range ref.String() {
			if unicode.IsControl(char) || !ref.IsName() && unicode.IsSpace(char) {
				return nil, wrap(fmt.Errorf("%w: invalid reference character", resource.ErrInvalidOption))
			}
		}
	}
	policy, err := blockstorage.PrepareDetachVolumeOptions(ctx, options...)
	if err != nil {
		return nil, wrap(err)
	}
	nova, err := c.ComputeV2(ctx)
	if err != nil {
		return nil, wrap(err)
	}
	var cinderClient *gophercloud.ServiceClient
	if *policy.Wait || input.Volume.IsName() {
		cinder, err := c.BlockStorageV3(ctx)
		if err != nil {
			return nil, wrap(err)
		}
		cinderClient = cinder.RawClient()
	}
	return blockstorage.DetachVolume(ctx, nova.RawClient(), cinderClient, input, blockstorage.WithDetachVolumeOptions(policy))
}

// AttachCinderVolume prepares originals once and uses cached Cinder directly.
func (c *Connection) AttachCinderVolume(ctx context.Context, input blockstorage.VolumeActionRequest, mountpoint string, options ...blockstorage.CinderVolumeAttachOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "AttachCinderVolume"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareCinderVolumeAttachOptions(ctx, options...)
	if err == nil {
		err = cinderaction.ValidateMountpoint(mountpoint)
	}
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.DirectAttach(ctx, client, input.VolumeID, mountpoint, cinderaction.WithDirectAttachOptions(policy))
}

// DetachCinderVolume performs one direct action without a Nova detach workflow.
func (c *Connection) DetachCinderVolume(ctx context.Context, input blockstorage.VolumeActionRequest, attachmentID string, options ...blockstorage.CinderVolumeDetachOption) (*blockstorage.VolumeActionResult, error) {
	const operation = "DetachCinderVolume"
	if err := c.volumeFlagPreflight(ctx); err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	policy, err := blockstorage.PrepareCinderVolumeDetachOptions(ctx, options...)
	if err == nil {
		err = cinderaction.ValidateAttachmentID(attachmentID)
	}
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	client, err := c.volumeFlagClient(ctx, input.VolumeID)
	if err != nil {
		return nil, cinderaction.WrapOperation(ctx, operation, err)
	}
	return cinderaction.DirectDetach(ctx, client, input.VolumeID, attachmentID, cinderaction.WithDirectDetachOptions(policy))
}
