package blockstorage

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
)

func captureVolumeRead(ctx context.Context, cinder *gophercloud.ServiceClient, options []VolumeReadOption) (*preparedVolumeSearch, error) {
	reader, err := captureGetVolumes(ctx, cinder, GetVolumesRequest{}, nil)
	if err != nil {
		return nil, err
	}
	policy, err := applyVolumeReadOptions(options, func() error { return reader.guard(ctx) })
	if err != nil {
		return nil, err
	}
	p := &preparedVolumeSearch{reader: reader, options: VolumeSearchOpts{Location: policy.Location}}
	return p, reader.guard(ctx)
}
