package image

import (
	"maps"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GetSchemaOpts supplies ordinary headers for fixed schema discovery requests.
// Authentication, representation and request routing remain SDK-owned.
type GetSchemaOpts struct {
	Headers map[string]string
}

// GetSchemaOption is one option of the schema reads: WithGetSchemaOpts,
// WithGetSchemaHeaders (merges), request.Header (adds one ordinary header;
// later applications win by name) or request.Custom over *GetSchemaOpts.
// Other shared calls, including request.Headers, fail with
// resource.ErrInvalidOption before HTTP.
type GetSchemaOption = request.GetCall

var getSchemaCalls = calls.Spec[GetSchemaOpts]{Header: calls.MergeHeader[GetSchemaOpts](WithGetSchemaHeaders)}

// WithGetSchemaOpts snapshots and replaces the complete typed header policy.
func WithGetSchemaOpts(value GetSchemaOpts) GetSchemaOption {
	snapshot := copyGetSchemaOpts(value)
	return request.Custom(func(config *GetSchemaOpts) error { *config = copyGetSchemaOpts(snapshot); return nil })
}

// WithGetSchemaHeaders snapshots and merges ordinary headers. Conflicting
// aliases within the supplied map are rejected.
func WithGetSchemaHeaders(values map[string]string) GetSchemaOption {
	return request.Custom(getSchemaHeaderOption(mergeImageMutationHeaders(values)))
}

func getSchemaHeaderOption(apply func(*ImageMutationOpts) error) func(*GetSchemaOpts) error {
	return func(config *GetSchemaOpts) error {
		headers := ImageMutationOpts{Headers: config.Headers}
		if err := apply(&headers); err != nil {
			return err
		}
		config.Headers = headers.Headers
		return nil
	}
}

func copyGetSchemaOpts(value GetSchemaOpts) GetSchemaOpts {
	value.Headers = maps.Clone(value.Headers)
	if value.Headers == nil {
		value.Headers = make(map[string]string)
	}
	return value
}

func parseGetSchemaOptions(options []GetSchemaOption) (GetSchemaOpts, error) {
	config := copyGetSchemaOpts(GetSchemaOpts{})
	for _, apply := range calls.Translate(options, getSchemaCalls) {
		if apply == nil {
			return config, uploadInvalid("nil schema discovery option")
		}
		candidate := copyGetSchemaOpts(config)
		if err := apply(&candidate); err != nil {
			return config, err
		}
		config = copyGetSchemaOpts(candidate)
	}
	headers, err := imageMutationHeaders(config.Headers, false, "")
	config.Headers = headers
	return config, err
}
