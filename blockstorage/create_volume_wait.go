package blockstorage

import (
	"context"
	"errors"
	"strings"
	"time"

	"gophercloudsdk/resource"
)

func (p *preparedCreateVolume) wait(ctx context.Context, result *CreateVolumeResult) error {
	waitCtx := ctx
	if p.options.timeout > 0 {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, p.options.timeout)
		defer cancel()
	}
	for {
		// Even an available creation response requires this fresh immediate
		// GET to establish readiness for the returned canonical identity.
		response, err := p.exchange(waitCtx, createVolumeObservation)
		if response != nil {
			result.LastAccepted = createVolumeProof(response)
		}
		if err != nil {
			// A rejected request retains its own native HTTP evidence, without
			// acquiring an earlier accepted response's body or status wrapper.
			return attachContextError(waitCtx, err)
		}
		volume, err := decodeCreatedVolume(response)
		if err != nil {
			return attachContextError(waitCtx, err)
		}
		result.LastAccepted.Volume = volume
		if err := validateCreateVolumeIdentity(volume, p.volumeID); err != nil {
			return response.Fail(attachContextError(waitCtx, err))
		}
		if err := p.guard(waitCtx); err != nil {
			return response.Fail(err)
		}
		status := *volume.Status
		if strings.EqualFold(status, "available") {
			// Ready independently owns all nullable fields, nested records,
			// raw JSON and headers rather than sharing LastAccepted.Volume.
			ready, err := decodeCreatedVolume(response)
			if err != nil {
				return attachContextError(waitCtx, err)
			}
			if err := p.guard(waitCtx); err != nil {
				return response.Fail(err)
			}
			result.Ready = ready
			return nil
		}
		for _, failure := range p.options.failures {
			if strings.EqualFold(status, failure) {
				return response.Fail(&resource.FailedStateError{Resource: "volume", ID: p.volumeID, Status: status})
			}
		}
		if progress := p.options.policy.WaitPolicy.ProgressCallback; progress != nil {
			if err := p.guard(waitCtx); err != nil {
				return response.Fail(err)
			}
			callbackErr := progress(0)
			if err := errors.Join(callbackErr, p.guard(waitCtx)); err != nil {
				return response.Fail(attachContextError(waitCtx, err))
			}
		}
		if err := p.guard(waitCtx); err != nil {
			return response.Fail(err)
		}
		timer := time.NewTimer(p.options.interval)
		select {
		case <-waitCtx.Done():
			timer.Stop()
			return response.Fail(attachContextError(waitCtx, waitCtx.Err()))
		case <-timer.C:
		}
		if err := p.guard(waitCtx); err != nil {
			return response.Fail(err)
		}
	}
}
