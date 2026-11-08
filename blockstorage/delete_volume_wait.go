package blockstorage

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/pollschedule"
	"github.com/JSYoo5B/go-openstacksdk/internal/progress"
)

func (p *preparedDeleteVolume) wait(ctx context.Context, result *DeleteVolumeResult) error {
	waitCtx := ctx
	if p.options.timeout > 0 {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, p.options.timeout)
		defer cancel()
	}
	pacer := pollschedule.NewPacer(p.options.strategy.Delay)
	for {
		// The timeout starts after lookup and mutation. Its first observation
		// is an immediate fresh GET of the already fixed canonical volume ID.
		response, err := p.exchange(waitCtx, deleteVolumeObservation)
		if response != nil {
			result.LastAccepted = deleteVolumeObservationProof(response)
			if response.StatusCode == http.StatusNotFound {
				result.Absent = deleteVolumeProof(response)
			}
		}
		if err != nil {
			// A rejected current request keeps its native evidence rather than
			// receiving an earlier accepted response's status/body wrapper.
			return attachContextError(waitCtx, err)
		}
		if response.StatusCode == http.StatusNotFound {
			result.Deleted = true
			return nil
		}
		volume, err := decodeCreatedVolume(response)
		if err != nil {
			return attachContextError(waitCtx, err)
		}
		result.LastAccepted.Volume = volume
		if err := validateDeleteVolumeIdentity(volume, p.volumeID); err != nil {
			return response.Fail(attachContextError(waitCtx, err))
		}
		if err := p.guard(waitCtx); err != nil {
			return response.Fail(err)
		}
		if volume.Status != nil && strings.EqualFold(*volume.Status, "deleted") {
			// An actual deleted-status observation independently owns Ready;
			// physical 404 completion never fabricates a volume model.
			ready, err := decodeCreatedVolume(response)
			if err != nil {
				return attachContextError(waitCtx, err)
			}
			if err := p.guard(waitCtx); err != nil {
				return response.Fail(err)
			}
			result.Ready = ready
			result.Deleted = true
			return nil
		}
		// Deletion has no failure-state predicate: error/error_deleting and
		// nullable status remain observations until deleted or absent.
		if callback := p.options.settings.ProgressCallback; callback != nil {
			if err := p.guard(waitCtx); err != nil {
				return response.Fail(err)
			}
			callbackErr := callback(0)
			if err := errors.Join(callbackErr, p.guard(waitCtx)); err != nil {
				return response.Fail(attachContextError(waitCtx, err))
			}
		}
		if ch := p.options.settings.Progress; ch != nil {
			if err := p.guard(waitCtx); err != nil {
				return response.Fail(err)
			}
			sendErr := progress.Send(waitCtx, ch, 0)
			if err := errors.Join(sendErr, p.guard(waitCtx)); err != nil {
				return response.Fail(attachContextError(waitCtx, err))
			}
		}
		if err := p.guard(waitCtx); err != nil {
			return response.Fail(err)
		}
		timer := time.NewTimer(pacer.Next())
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
