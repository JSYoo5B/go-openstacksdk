package image

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// AddImageLocation submits a literal URL and optional hash pair to Glance's
// asynchronous location task. It returns only an actual 202 acknowledgement;
// the server owns storage, permissions and eventual content verification.
func (s *Service) AddImageLocation(ctx context.Context, ref resource.Ref, locationURL string, options ...AddImageLocationOption) (*AddImageLocationResult, error) {
	options = append([]AddImageLocationOption(nil), options...)
	var body json.RawMessage
	prepared, err := s.prepareImageMutation(ctx, ref, nil, []ImageMutationOption{func(config *ImageMutationOpts) error {
		if locationURL == "" || !utf8.ValidString(locationURL) {
			return uploadInvalid("location URL must be nonempty valid UTF-8")
		}
		policy, err := parseAddImageLocationOptions(options)
		if err != nil {
			return err
		}
		validation := make(map[string]string)
		if pair := policy.ValidationData; pair != nil {
			validation["os_hash_algo"] = pair.OSHashAlgo
			validation["os_hash_value"] = pair.OSHashValue
		}
		body, err = json.Marshal(map[string]any{"url": locationURL, "validation_data": validation})
		config.Headers = policy.Headers
		return err
	}})
	if err != nil {
		return nil, wrapImageMutationError(ctx, "AddImageLocation", err)
	}
	endpoint := prepared.base + "images/" + url.PathEscape(prepared.id) + "/locations"
	response, err := rest.DoJSON(ctx, prepared.client, http.MethodPost, endpoint, body, nil, http.StatusAccepted)
	return addImageLocationResult(prepared.id, locationURL, response), wrapImageMutationError(ctx, "AddImageLocation", err)
}

// GetImageLocations fetches one complete bare array without pagination. An ID
// needs no lookup; an explicit Name resolves exactly before the fixed GET.
// Any response-body or decoding failure returns nil with raw response evidence.
func (s *Service) GetImageLocations(ctx context.Context, ref resource.Ref, options ...GetImageLocationsOption) (*ImageLocationsResult, error) {
	options = append([]GetImageLocationsOption(nil), options...)
	prepared, err := s.prepareImageMutation(ctx, ref, nil, []ImageMutationOption{func(config *ImageMutationOpts) error {
		policy, err := parseGetImageLocationsOptions(options)
		config.Headers = policy.Headers
		return err
	}})
	if err != nil {
		return nil, wrapImageMutationError(ctx, "GetImageLocations", err)
	}
	endpoint := prepared.base + "images/" + url.PathEscape(prepared.id) + "/locations"
	response, err := rest.DoJSON(ctx, prepared.client, http.MethodGet, endpoint, nil, nil, http.StatusOK)
	if err != nil {
		return nil, wrapImageMutationError(ctx, "GetImageLocations", err)
	}
	locations, err := decodeImageLocations(response.Body)
	if err == nil {
		if err = ctx.Err(); err != nil && !errors.Is(err, context.Cause(ctx)) {
			err = errors.Join(err, context.Cause(ctx))
		}
	}
	if err != nil {
		return nil, wrapImageMutationError(ctx, "GetImageLocations", response.Fail(err))
	}
	return &ImageLocationsResult{ImageID: prepared.id, Locations: locations, Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}, nil
}
