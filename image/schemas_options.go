package image

import "maps"

// GetSchemaOpts supplies ordinary headers for fixed schema discovery requests.
// Authentication, representation and request routing remain SDK-owned.
type GetSchemaOpts struct {
	Headers map[string]string
}

type GetSchemaOption func(*GetSchemaOpts) error

// WithGetSchemaOpts snapshots and replaces the complete typed header policy.
func WithGetSchemaOpts(value GetSchemaOpts) GetSchemaOption {
	snapshot := copyGetSchemaOpts(value)
	return func(config *GetSchemaOpts) error { *config = copyGetSchemaOpts(snapshot); return nil }
}

// WithGetSchemaHeader adds one ordinary header; later applications win by name.
func WithGetSchemaHeader(key, value string) GetSchemaOption {
	return getSchemaHeaderOption(WithImageMutationHeader(key, value))
}

// WithGetSchemaHeaders snapshots and merges ordinary headers. Conflicting
// aliases within the supplied map are rejected.
func WithGetSchemaHeaders(values map[string]string) GetSchemaOption {
	return getSchemaHeaderOption(WithImageMutationHeaders(values))
}

func getSchemaHeaderOption(apply ImageMutationOption) GetSchemaOption {
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
	for _, apply := range options {
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
