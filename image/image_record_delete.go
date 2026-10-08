package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ImageRecordDeleteRequest selects a literal ID or an SDK-produced record.
// The record's private raw current ID is authoritative; edits to its public
// descriptor view, Wire or fetch receipt do not select another route.
type ImageRecordDeleteRequest struct {
	ID     string
	Record *ImageRecord
}

// ImageRecordDeleteResult separates a whole-image descriptor result from the
// actual opaque DELETE acknowledgement. Store deletion returns only the latter.
// Accepted response handling failures retain an acknowledgement and no Record.
type ImageRecordDeleteResult struct {
	Record          *ImageRecord
	Acknowledgement *DeleteImageResult
}

// DeleteImageRecord deletes a fixed image or one selected store copy without
// discovery. Whole-image success returns an independent current-location view
// while preserving raw dirty/original state and earlier fetch receipts. DELETE
// bytes are opaque and never overlay image attributes or clean its body state.
// Default ignore-missing suppresses only a clean final native HTTP 404.
func (s *Service) DeleteImageRecord(ctx context.Context, input ImageRecordDeleteRequest, options ...ImageRecordDeleteOption) (*ImageRecordDeleteResult, error) {
	const operation = "DeleteImageRecord"
	fail := func(err error) (*ImageRecordDeleteResult, error) {
		return nil, wrapImageMutationError(ctx, operation, err)
	}
	owned := slices.Clone(options)
	p, err := s.captureImageRecord(ctx)
	if err != nil {
		return fail(err)
	}
	// Capture all caller-owned image channels before location/options callbacks.
	seed, identity, err := imageRecordDeleteSeed(input)
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return fail(err)
	}
	var policy ImageRecordDeleteOpts
	var storeID string
	var storeMode bool
	err = p.prepare(func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		var err error
		policy, err = prepareImageRecordDeleteOptions(opctx, check, owned)
		if err != nil {
			return nil, err
		}
		storeID, storeMode, err = imageRecordDeleteStoreIdentity(policy)
		if err != nil {
			return nil, err
		}
		if !storeMode {
			// Source's whole-image _update materializes all descriptor values.
			// A store helper needs only IDs, so unrelated conversions are skipped.
			seed.ImportMethods = make([]string, 0)
			var passive resource.Metadata
			if seed.Resource != nil {
				passive = seed.Resource.Metadata
			}
			view, err := projectImageRecord(seed.bodyState.current, p.location,
				resource.Metadata{Header: seed.Header.Clone(), StatusCode: seed.StatusCode})
			if err != nil {
				return nil, err
			}
			// These already-owned passive fields do not enter Body or the route.
			view.CreatedAt, view.UpdatedAt, view.Links = passive.CreatedAt, passive.UpdatedAt, passive.Links
			seed.Resource = view
		}
		return policy.Headers, check(opctx)
	})
	if err != nil {
		return fail(err)
	}
	endpoint := imageRecordEndpoint(p, identity)
	if storeMode {
		endpoint = p.base + "stores/" + url.PathEscape(storeID) + "/" + url.PathEscape(identity)
	}
	response, err := rest.DoJSONGuardedRejections(p.ctx, p.client, p.check, http.MethodDelete, endpoint, nil, nil,
		rest.RejectionPolicy{Codes: imageRecordTagRejectionCodes(), PreserveCleanRetry: true}, imageRecordCodes()...)
	if guardErr := p.check(p.ctx); guardErr != nil {
		if response != nil {
			guardErr = response.Fail(guardErr)
		}
		err = errors.Join(err, guardErr)
	}
	if response == nil {
		if native, clean := err.(gophercloud.ErrUnexpectedResponseCode); clean && native.Actual == http.StatusNotFound && (policy.IgnoreMissing == nil || *policy.IgnoreMissing) {
			return nil, nil
		}
		return fail(err)
	}
	ack := &DeleteImageResult{ImageID: identity, StoreID: storeID,
		Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	result := &ImageRecordDeleteResult{Acknowledgement: ack}
	if err != nil {
		return result, wrapImageMutationError(ctx, operation, err)
	}
	if !storeMode {
		// has_body=False translates response headers but does not clean Body.
		// Retain the original fetch evidence; the new receipt belongs to Ack.
		seed.ImportMethods = imageRecordHTTPImportMethods(response.Header)
		result.Record = seed
	}
	return result, nil
}

func imageRecordDeleteSeed(input ImageRecordDeleteRequest) (*ImageRecord, string, error) {
	if input.ID != "" && input.Record != nil {
		return nil, "", uploadInvalid("select image ID or Record, not both")
	}
	seed := cloneImageRecord(input.Record)
	if seed == nil {
		if err := validateImageRecordIdentity(input.ID); err != nil {
			return nil, "", err
		}
		rawID, _ := json.Marshal(input.ID)
		seed = &ImageRecord{bodyState: pendingImageRecordBodyState(map[string]json.RawMessage{"id": rawID})}
	} else if seed.bodyState == nil {
		return nil, "", uploadInvalid("image Record must retain SDK-produced raw body state")
	}
	identity, err := imageRecordDeleteLiteral(seed.bodyState.current["id"], "image Record")
	if err != nil {
		return nil, "", err
	}
	return seed, identity, nil
}

func imageRecordDeleteStoreIdentity(policy ImageRecordDeleteOpts) (string, bool, error) {
	if policy.StoreID != nil && policy.StoreRecord != nil {
		return "", false, uploadInvalid("select store ID or StoreRecord, not both")
	}
	if policy.StoreID != nil {
		if err := validateImageRecordIdentity(*policy.StoreID); err != nil {
			return "", false, err
		}
		return *policy.StoreID, true, nil
	}
	if policy.StoreRecord == nil {
		return "", false, nil
	}
	if policy.StoreRecord.Resource == nil {
		return "", false, uploadInvalid("store Record Resource is required")
	}
	identity, err := imageRecordDeleteLiteral(policy.StoreRecord.Resource.Body["id"], "store Record")
	return identity, true, err
}

func imageRecordDeleteLiteral(raw json.RawMessage, label string) (string, error) {
	identity, err := decodeImageRecordString(raw, label+" identity")
	if err != nil {
		return "", err
	}
	if err := validateImageRecordIdentity(identity); err != nil {
		return "", err
	}
	return identity, nil
}
