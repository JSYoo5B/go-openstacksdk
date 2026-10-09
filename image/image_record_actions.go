package image

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// ImageRecordActionRequest selects a literal ID or an SDK-produced record's
// private current ID. Public view, Wire and receipt edits do not retarget it.
type ImageRecordActionRequest struct {
	ID     string
	Record *ImageRecord
}

// ImageRecordActionResult separates the prepared local descriptor view from
// an actual opaque action receipt. The Record never asserts a new image status.
// Accepted response handling failures return only acknowledgement plus error.
type ImageRecordActionResult struct {
	Record          *ImageRecord
	Acknowledgement *ImageActionResult
}

// DeactivateImageRecord posts one fixed action without GET, discovery or wait.
// It prepares a current-location descriptor view and preserves raw pending
// state and earlier fetch receipts. Response bytes/headers do not update it.
func (s *Service) DeactivateImageRecord(ctx context.Context, input ImageRecordActionRequest, options ...ImageRecordActionOption) (*ImageRecordActionResult, error) {
	return s.mutateImageRecordAction(ctx, input, slices.Clone(options), "deactivate", "DeactivateImageRecord")
}

// ReactivateImageRecord preserves the same owned action contract. Server
// permission, existence and state transition checks remain server-owned.
func (s *Service) ReactivateImageRecord(ctx context.Context, input ImageRecordActionRequest, options ...ImageRecordActionOption) (*ImageRecordActionResult, error) {
	return s.mutateImageRecordAction(ctx, input, slices.Clone(options), "reactivate", "ReactivateImageRecord")
}

func (s *Service) mutateImageRecordAction(ctx context.Context, input ImageRecordActionRequest, options []ImageRecordActionOption, action, operation string) (*ImageRecordActionResult, error) {
	fail := func(err error) (*ImageRecordActionResult, error) {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	p, err := s.captureImageRecord(ctx)
	if err != nil {
		return fail(err)
	}
	seed, identity, err := imageRecordMutationSeed(input.ID, input.Record)
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return fail(err)
	}
	if err := p.prepare(func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		headers, err := prepareImageRecordActionHeaders(opctx, check, options)
		if err != nil {
			return nil, err
		}
		err = projectImageRecordMutationSeed(seed, p.location)
		return headers, errors.Join(err, check(opctx))
	}); err != nil {
		return fail(err)
	}
	// Pinned Python _action does not raise for response status by itself and
	// Proxy.request defaults raise_exc=False. This Go profile deliberately uses
	// the SDK's explicit native rejection policy for 400..599 and accepts 200..399.
	response, err := rest.DoJSONGuardedRejections(p.ctx, p.client, p.check, http.MethodPost,
		imageRecordEndpoint(p, identity)+"/actions/"+action, nil, nil,
		rest.RejectionPolicy{Codes: imageRecordTagRejectionCodes(), PreserveCleanRetry: true}, imageRecordCodes()...)
	if guardErr := p.check(p.ctx); guardErr != nil {
		if response != nil {
			guardErr = response.Fail(guardErr)
		}
		err = errors.Join(err, guardErr)
	}
	if response == nil {
		return fail(err)
	}
	result := &ImageRecordActionResult{Acknowledgement: &ImageActionResult{
		ImageID: identity, Action: action, Body: bytes.Clone(response.Body),
		Header: response.Header.Clone(), StatusCode: response.StatusCode,
	}}
	if err != nil {
		return result, wrapImageMutationError(ctx, operation, err)
	}
	// Image._action never invokes Resource._translate_response. Retain the
	// prepared import-method reset and prior fetch/status/current/dirty state;
	// even response import-method headers belong only to the acknowledgement.
	result.Record = seed
	return result, nil
}
