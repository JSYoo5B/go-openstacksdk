package blockstorage

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Only the owning attach/detach workflows construct this fixed-ID loop. The
// source exchange and guard retain their own exact method, route and HTTP codes.
type volumeAttachmentWait struct {
	volumeID, target  string
	timeout, interval time.Duration
	failures          []string
	progress          func(int) error
	guard             func(context.Context) error
	exchange          func(context.Context) (*rest.Response, error)
	lastAccepted      **AttachVolumeObservationResponse
	ready             **AttachVolumeObservation
}

func waitVolumeAttachment(ctx context.Context, policy volumeAttachmentWait) error {
	waitCtx := ctx
	if policy.timeout > 0 {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, policy.timeout)
		defer cancel()
	}
	for {
		// Start with an immediate fresh observation after the owning mutation;
		// no supplied model or earlier response can establish readiness.
		response, err := policy.exchange(waitCtx)
		if response != nil {
			*policy.lastAccepted = attachObservationProof(response)
		}
		if err != nil {
			// Preserve a rejected request's native HTTP evidence rather than
			// making an earlier accepted response its status/body wrapper.
			return attachContextError(waitCtx, err)
		}
		volume, err := decodeAttachObservation(response)
		if err != nil {
			return attachContextError(waitCtx, err)
		}
		(*policy.lastAccepted).Volume = volume
		if volume == nil || volume.ID == nil || *volume.ID != policy.volumeID {
			return response.Fail(attachContextError(waitCtx, attachInvalid("Cinder volume response must identify the selected volume %q", policy.volumeID)))
		}
		if volume.Status == nil {
			return response.Fail(attachContextError(waitCtx, attachInvalid("Cinder volume response must contain a nonnull status")))
		}
		if err := policy.guard(waitCtx); err != nil {
			return response.Fail(err)
		}
		status := *volume.Status
		if strings.EqualFold(status, policy.target) {
			// Ready independently owns nullable fields, nested records, raw
			// JSON and headers, rather than sharing LastAccepted.Volume.
			ready, err := decodeAttachObservation(response)
			if err != nil {
				return attachContextError(waitCtx, err)
			}
			if err := policy.guard(waitCtx); err != nil {
				return response.Fail(err)
			}
			*policy.ready = ready
			return nil
		}
		for _, failure := range policy.failures {
			if strings.EqualFold(status, failure) {
				return response.Fail(&resource.FailedStateError{Resource: "volume", ID: policy.volumeID, Status: status})
			}
		}
		if progress := policy.progress; progress != nil {
			if err := policy.guard(waitCtx); err != nil {
				return response.Fail(err)
			}
			callbackErr := progress(0)
			if err := errors.Join(callbackErr, policy.guard(waitCtx)); err != nil {
				return response.Fail(attachContextError(waitCtx, err))
			}
		}
		if err := policy.guard(waitCtx); err != nil {
			return response.Fail(err)
		}
		timer := time.NewTimer(policy.interval)
		select {
		case <-waitCtx.Done():
			timer.Stop()
			return response.Fail(attachContextError(waitCtx, waitCtx.Err()))
		case <-timer.C:
		}
		if err := policy.guard(waitCtx); err != nil {
			return response.Fail(err)
		}
	}
}
