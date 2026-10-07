package blockstorage

import (
	"context"
	"net/http"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// DetachVolumeRequest explicitly selects a server and volume by ID or name.
// Volume is the Cinder volume identity, not an attachment identity.
type DetachVolumeRequest struct {
	Server resource.Ref
	Volume resource.Ref
}

// DetachVolume sends a Nova attachment DELETE and by default waits for a fresh
// Cinder available observation. A missing attachment is an error. It preserves
// accepted responses on later failures and performs no compensating mutation.
// A Cinder client may be nil when waiting is disabled and Volume is an ID;
// every provided client is captured and validated before options run.
func DetachVolume(ctx context.Context, computeClient, blockStorageClient *gophercloud.ServiceClient, input DetachVolumeRequest, options ...DetachVolumeOption) (*DetachVolumeResult, error) {
	p, err := captureDetach(ctx, computeClient, blockStorageClient, input, options)
	if err != nil {
		return nil, wrapDetachError(ctx, err)
	}
	result := &DetachVolumeResult{ServerID: p.serverID, VolumeID: p.volumeID}
	response, err := p.exchange(ctx, http.MethodDelete)
	result.Deleted = detachDeletionProof(response)
	if err != nil {
		return result, wrapDetachError(ctx, err)
	}
	// The accepted DELETE body is opaque; acknowledgement does not require
	// a JSON envelope or a decoded attachment model.
	if !p.options.wait {
		return result, nil
	}
	err = waitVolumeAttachment(ctx, volumeAttachmentWait{
		volumeID: p.volumeID, target: "available",
		timeout: p.options.timeout, interval: p.options.interval,
		failures: p.options.failures, progress: p.options.policy.WaitPolicy.ProgressCallback,
		guard: p.guard,
		exchange: func(waitCtx context.Context) (*rest.Response, error) {
			return p.exchange(waitCtx, http.MethodGet)
		},
		lastAccepted: &result.LastAccepted, ready: &result.Ready,
	})
	return result, wrapDetachError(ctx, err)
}

func wrapDetachError(ctx context.Context, err error) error {
	return request.Wrap("DetachVolume", "volume", attachContextError(ctx, err))
}
