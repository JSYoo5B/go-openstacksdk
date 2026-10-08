package blockstorage

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudlocation"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type preparedVolumeTypes struct {
	reader        *preparedGetVolumes
	options       VolumeTypeSearchOpts
	location      resource.CloudLocation
	locationError error
	memberFailure error
}

func captureVolumeTypesRead(ctx context.Context, cinder *gophercloud.ServiceClient, options []VolumeTypeReadOption) (*preparedVolumeTypes, error) {
	reader, err := captureCinderReader(ctx, cinder, "types")
	if err != nil {
		return nil, err
	}
	policy, err := applyVolumeTypeReadOptions(options, func() error { return reader.guard(ctx) })
	if err != nil {
		return nil, err
	}
	p := &preparedVolumeTypes{reader: reader, options: VolumeTypeSearchOpts{Location: policy.Location}}
	p.captureLocation()
	return p, reader.guard(ctx)
}

func captureVolumeTypesSearch(ctx context.Context, cinder *gophercloud.ServiceClient, options []VolumeTypeSearchOption) (*preparedVolumeTypes, error) {
	reader, err := captureCinderReader(ctx, cinder, "types")
	if err != nil {
		return nil, err
	}
	p := &preparedVolumeTypes{reader: reader}
	p.options, err = applyVolumeTypeSearchOptions(options, func() error { return reader.guard(ctx) })
	if err != nil {
		return nil, err
	}
	p.captureLocation()
	return p, reader.guard(ctx)
}

func (p *preparedVolumeTypes) captureLocation() {
	if p.options.Location != nil {
		p.location = p.options.Location.Clone()
	} else {
		p.location.Project.ID, p.locationError = cloudlocation.ProjectID(p.reader.cinder.provider)
	}
}

func (p *preparedVolumeTypes) currentLocation() (json.RawMessage, error) {
	if p.locationError != nil {
		return nil, p.locationError
	}
	// Type has no project/zone descriptors. Preserve the complete owned override;
	// wire project_id/availability_zone never become location evidence.
	return p.location.ForResource(nil, p.location.Zone)
}

func wrapVolumeTypeError(ctx context.Context, operation string, err error) error {
	if err == nil {
		return nil
	}
	return request.Wrap(operation, "volume type", attachContextError(ctx, err))
}
