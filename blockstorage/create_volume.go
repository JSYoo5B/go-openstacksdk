package blockstorage

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// CreateVolumeRequest supplies the required size and an optional explicit image
// selector. An image name resolves once; an image ID requires no Glance client
// or lookup. Size is sent unchanged for the Cinder server to validate.
type CreateVolumeRequest struct {
	Size  int
	Image *resource.Ref
}

// CreateVolume creates a Cinder volume and by default waits for a fresh
// available observation. A nonnil Bootable option forces waiting; only true
// sends a bootable action after readiness. Accepted responses remain available
// on later failures, and the workflow performs no compensating mutation.
// Every provided service source is captured before external options run.
func CreateVolume(ctx context.Context, blockStorageClient, imageClient *gophercloud.ServiceClient, input CreateVolumeRequest, options ...CreateVolumeOption) (*CreateVolumeResult, error) {
	p, err := captureCreateVolume(ctx, blockStorageClient, imageClient, input, options)
	if err != nil {
		return nil, wrapCreateVolumeError(ctx, err)
	}
	result := &CreateVolumeResult{}
	response, err := p.exchange(ctx, createVolumeCreation)
	result.Created = createVolumeProof(response)
	if err != nil {
		return result, wrapCreateVolumeError(ctx, err)
	}
	created, err := decodeCreatedVolume(response)
	if err != nil {
		return result, wrapCreateVolumeError(ctx, err)
	}
	// The server's canonical identity is checked before it can establish any
	// fixed polling/action route, including when waiting is disabled.
	if err := p.bindVolume(ctx, created); err != nil {
		return result, wrapCreateVolumeError(ctx, response.Fail(attachContextError(ctx, err)))
	}
	result.Created.Volume = created
	result.VolumeID = p.volumeID
	// The pinned cloud helper rejects this literal initial status even when
	// wait=false. Fresh waiting observations use case-insensitive states.
	if *created.Status == "error" {
		return result, wrapCreateVolumeError(ctx, response.Fail(attachContextError(ctx, &resource.FailedStateError{Resource: "volume", ID: p.volumeID, Status: *created.Status})))
	}
	if !p.options.wait {
		return result, nil
	}
	// Only the fresh status phase consumes the SDK wait timeout. Creation,
	// decoding, canonical binding and any later action use the parent context.
	if err := p.wait(ctx, result); err != nil {
		return result, wrapCreateVolumeError(ctx, err)
	}
	if p.options.policy.Bootable != nil && *p.options.policy.Bootable {
		response, err := p.exchange(ctx, createVolumeBootableAction)
		result.BootableSet = createVolumeActionProof(response)
		if err != nil {
			return result, wrapCreateVolumeError(ctx, err)
		}
		// This acknowledgement is opaque. It does not change the observed
		// Ready model's bootable field or require another GET.
	}
	return result, nil
}

func wrapCreateVolumeError(ctx context.Context, err error) error {
	return request.Wrap("CreateVolume", "volume", attachContextError(ctx, err))
}
