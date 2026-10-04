package blockstorage

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// AttachVolumeRequest explicitly selects a server and volume by ID or name.
// Names resolve once before the workflow constructs its fixed request URLs.
type AttachVolumeRequest struct {
	Server resource.Ref
	Volume resource.Ref
}

// AttachVolume checks a fresh available Cinder volume, creates its Nova
// attachment, and by default waits for a fresh Cinder in-use observation. It
// retains accepted phase responses on failure and never rolls back creation.
// The original providers supply live authentication while request settings and
// the concrete policy are captured before options or name lookup run.
func AttachVolume(ctx context.Context, computeClient, blockStorageClient *gophercloud.ServiceClient, input AttachVolumeRequest, options ...AttachVolumeOption) (*AttachVolumeResult, error) {
	p, err := captureAttach(ctx, computeClient, blockStorageClient, input, options)
	if err != nil {
		return nil, wrapAttachError(ctx, err)
	}
	result := &AttachVolumeResult{ServerID: p.serverID, VolumeID: p.volumeID}
	response, err := p.exchange(ctx, http.MethodGet, p.volumeURL, nil)
	result.Checked = attachObservationProof(response)
	if err != nil {
		return result, wrapAttachError(ctx, err)
	}
	checked, err := decodeAttachObservation(response)
	if err != nil {
		return result, wrapAttachError(ctx, err)
	}
	result.Checked.Volume = checked
	if err := p.validateChecked(checked); err != nil {
		return result, wrapAttachError(ctx, response.Fail(err))
	}
	if err := p.guard(ctx); err != nil {
		return result, wrapAttachError(ctx, response.Fail(err))
	}
	attachment := map[string]any{"volumeId": p.volumeID}
	if p.options.policy.Device != "" {
		attachment["device"] = p.options.policy.Device
	}
	response, err = p.exchange(ctx, http.MethodPost, p.attachmentURL, map[string]any{"volumeAttachment": attachment})
	result.Created = attachCreationProof(response)
	if err != nil {
		return result, wrapAttachError(ctx, err)
	}
	created, err := decodeCreatedAttachment(response)
	if err != nil {
		return result, wrapAttachError(ctx, err)
	}
	if err := p.validateCreated(created); err != nil {
		return result, wrapAttachError(ctx, response.Fail(err))
	}
	if err := p.guard(ctx); err != nil {
		return result, wrapAttachError(ctx, response.Fail(err))
	}
	result.Created.Attachment = created
	if !p.options.wait {
		return result, nil
	}
	// Start the SDK timeout only after creation has been decoded and its
	// canonical identities and original service sources have been checked.
	err = p.wait(ctx, result)
	return result, wrapAttachError(ctx, err)
}

func wrapAttachError(ctx context.Context, err error) error {
	return request.Wrap("AttachVolume", "volume", attachContextError(ctx, err))
}

func (p *preparedAttach) validateObservation(volume *AttachVolumeObservation) error {
	if volume == nil || volume.ID == nil || *volume.ID != p.volumeID {
		return attachInvalid("Cinder volume response must identify the selected volume %q", p.volumeID)
	}
	if volume.Status == nil {
		return attachInvalid("Cinder volume response must contain a nonnull status")
	}
	return nil
}

func (p *preparedAttach) validateChecked(volume *AttachVolumeObservation) error {
	if err := p.validateObservation(volume); err != nil {
		return err
	}
	if *volume.Status != "available" {
		return attachInvalid("selected volume %q must be available before attachment, observed %q", p.volumeID, *volume.Status)
	}
	if _, exists := attachmentNonNull(volume.Body, "attachments"); !exists || volume.Attachments == nil {
		return attachInvalid("Cinder precondition response must contain a nonnull attachments array")
	}
	for _, attachment := range volume.Attachments {
		if attachment == nil {
			return attachInvalid("Cinder attachments must contain nonnull records")
		}
		if attachment.ServerID != nil && *attachment.ServerID == p.serverID {
			return attachInvalid("volume %q is already associated with server %q", p.volumeID, p.serverID)
		}
	}
	return nil
}

func (p *preparedAttach) validateCreated(attachment *VolumeAttachmentInfo) error {
	if attachment == nil || attachment.VolumeID == nil || *attachment.VolumeID != p.volumeID {
		return attachInvalid("Nova attachment response must identify the selected volume %q", p.volumeID)
	}
	if attachment.ServerID != nil && *attachment.ServerID != p.serverID {
		return attachInvalid("Nova attachment response identifies a different server")
	}
	return nil
}

func (p *preparedAttach) wait(ctx context.Context, result *AttachVolumeResult) error {
	waitCtx := ctx
	if p.options.timeout > 0 {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, p.options.timeout)
		defer cancel()
	}
	for {
		// The first observation is immediate and fresh; the precondition
		// response cannot establish post-creation readiness.
		response, err := p.exchange(waitCtx, http.MethodGet, p.volumeURL, nil)
		if response != nil {
			result.LastAccepted = attachObservationProof(response)
		}
		if err != nil {
			// A rejected request keeps its own native status/body/header error.
			// Do not make an earlier accepted response its HTTP evidence.
			return attachContextError(waitCtx, err)
		}
		volume, err := decodeAttachObservation(response)
		if err != nil {
			return attachContextError(waitCtx, err)
		}
		result.LastAccepted.Volume = volume
		if err := p.validateObservation(volume); err != nil {
			return response.Fail(attachContextError(waitCtx, err))
		}
		if err := p.guard(waitCtx); err != nil {
			return response.Fail(err)
		}
		status := *volume.Status
		if strings.EqualFold(status, "in-use") {
			// Decode independently so Ready and LastAccepted never share
			// nullable pointers, raw fields, attachment records or headers.
			ready, err := decodeAttachObservation(response)
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
