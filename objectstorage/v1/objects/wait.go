package objects

import (
	"context"
	"net/http"
	"strings"

	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// WaitForDelete polls fresh HEAD until a clean physical 404. It never issues
// DELETE and does not infer deletion from a status header or an acknowledgement.
func (a *API) WaitForDelete(ctx context.Context, container, object string, options ...ObjectWaitOption) (*ObjectWaitResult, error) {
	result, err := a.waitForObject(ctx, container, object, "", true, options)
	return result, request.Wrap("WaitForDelete", "objects", err)
}

// WaitForStatus polls fresh HEAD and compares one selected string. Swift has
// no native status field: select a supported attribute or explicit header.
func (a *API) WaitForStatus(ctx context.Context, container, object, status string, options ...ObjectWaitOption) (*ObjectWaitResult, error) {
	result, err := a.waitForObject(ctx, container, object, status, false, options)
	return result, request.Wrap("WaitForStatus", "objects", err)
}

func (a *API) waitForObject(ctx context.Context, container, object, status string, deletion bool, options []ObjectWaitOption) (*ObjectWaitResult, error) {
	w, err := a.prepareObjectWait(ctx, container, object, status, deletion, options)
	if err != nil {
		return nil, err
	}
	if w.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, w.timeout)
		defer cancel()
	}
	result := &ObjectWaitResult{}
	for {
		if err := w.request.guard(ctx); err != nil {
			return result, objectWaitLastError(result, err)
		}
		out := w.request.exchange(ctx, http.MethodHead, w.request.metadata.target, "object-wait", nil, w.options.Headers, result.Polls+1, http.StatusOK, http.StatusNoContent, http.StatusNotFound)
		result.Polls++
		// Cancellation or a source guard may stop an exchange before HTTP.
		// Keep the previous actual observation in that case.
		if len(out.phase.Attempts) != 0 {
			result.Last, result.Status = out.phase, nil
		}
		if err = joinMetadataErrors(out.err, w.request.guard(ctx)); err != nil {
			return result, objectWaitLastError(result, err)
		}
		response := result.Last.Acknowledgement
		if response == nil {
			return result, metadataInvalid("object waiter received no accepted response")
		}
		if response.StatusCode == http.StatusNotFound {
			if deletion {
				result.Complete, result.Deleted = true, true
				return result, nil
			}
			return result, objectCreateResponseError(response, &resource.NotFoundError{Resource: "objects", Reference: container + "/" + object})
		}
		var selected *string
		if !deletion {
			selected, err = w.selected(response.Header)
			if err = joinMetadataErrors(err, w.request.guard(ctx)); err != nil {
				return result, objectCreateResponseError(response, err)
			}
			if strings.EqualFold(*selected, status) {
				result.Status, result.Complete = selected, true
				return result, nil
			}
			for _, failure := range w.options.FailureStates {
				if strings.EqualFold(*selected, failure) {
					result.Status = selected
					return result, objectCreateResponseError(response, &resource.FailedStateError{Resource: "objects", ID: container + "/" + object, Status: *selected})
				}
			}
		}
		if callback := w.options.ProgressCallback; callback != nil {
			if err := w.request.guard(ctx); err != nil {
				return result, objectCreateResponseError(response, err)
			}
			callbackErr := callback(0)
			if err = joinMetadataErrors(callbackErr, w.request.guard(ctx)); err != nil {
				return result, objectCreateResponseError(response, err)
			}
		}
		if err := w.request.guard(ctx); err != nil {
			return result, objectCreateResponseError(response, err)
		}
		result.Status = selected
		if err := w.pause(ctx); err != nil {
			return result, objectCreateResponseError(response, err)
		}
	}
}
func objectWaitLastError(result *ObjectWaitResult, cause error) error {
	if result != nil && result.Last != nil {
		return objectCreateResponseError(result.Last.Acknowledgement, cause)
	}
	return cause
}
