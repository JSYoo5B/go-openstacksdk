package image

import (
	"maps"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// ImageLocationValidation supplies literal hash values to the server's location
// task. The server owns algorithm, digest and content verification policy.
type ImageLocationValidation struct {
	OSHashAlgo  string
	OSHashValue string
}

// AddImageLocationOpts supplies an optional hash pair and ordinary headers.
// Nil ValidationData sends an empty object and leaves hashing to the server.
type AddImageLocationOpts struct {
	ValidationData *ImageLocationValidation
	Headers        map[string]string
}

// AddImageLocationOption is one option of AddImageLocation:
// WithAddImageLocationOpts, WithImageLocationValidation,
// WithAddImageLocationHeaders (merges), request.Header (merges one header) or
// request.Custom over *AddImageLocationOpts. Other shared calls, including
// request.Headers and request.Field, fail with resource.ErrInvalidOption
// before HTTP.
type AddImageLocationOption = request.WriteCall

// GetImageLocationsOpts supplies ordinary headers for one finite locations GET.
type GetImageLocationsOpts struct {
	Headers map[string]string
}

// GetImageLocationsOption is one option of GetImageLocations:
// WithGetImageLocationsOpts, WithGetImageLocationsHeaders (merges),
// request.Header (merges one header) or request.Custom over
// *GetImageLocationsOpts. Other shared calls, including request.Headers, fail
// with resource.ErrInvalidOption before HTTP.
type GetImageLocationsOption = request.GetCall

var (
	addImageLocationCalls  = calls.Spec[AddImageLocationOpts]{Header: calls.MergeHeader[AddImageLocationOpts](WithAddImageLocationHeaders)}
	getImageLocationsCalls = calls.Spec[GetImageLocationsOpts]{Header: calls.MergeHeader[GetImageLocationsOpts](WithGetImageLocationsHeaders)}
)

// WithAddImageLocationOpts snapshots and replaces the complete typed policy.
func WithAddImageLocationOpts(value AddImageLocationOpts) AddImageLocationOption {
	snapshot := copyAddImageLocationOpts(value)
	return request.Custom(func(config *AddImageLocationOpts) error { *config = copyAddImageLocationOpts(snapshot); return nil })
}

// WithImageLocationValidation supplies both literal hash fields. No algorithm,
// hex, whitespace or case normalization is performed by the SDK.
func WithImageLocationValidation(algorithm, value string) AddImageLocationOption {
	return request.Custom(func(config *AddImageLocationOpts) error {
		config.ValidationData = &ImageLocationValidation{OSHashAlgo: algorithm, OSHashValue: value}
		return nil
	})
}

// WithAddImageLocationHeaders snapshots and merges ordinary headers, rejecting
// conflicting aliases within the supplied map.
func WithAddImageLocationHeaders(values map[string]string) AddImageLocationOption {
	apply := mergeImageMutationHeaders(values)
	return request.Custom(func(config *AddImageLocationOpts) error {
		return applyImageLocationHeader(&config.Headers, apply)
	})
}

// WithGetImageLocationsOpts snapshots and replaces the complete typed policy.
func WithGetImageLocationsOpts(value GetImageLocationsOpts) GetImageLocationsOption {
	snapshot := copyGetImageLocationsOpts(value)
	return request.Custom(func(config *GetImageLocationsOpts) error { *config = copyGetImageLocationsOpts(snapshot); return nil })
}

// WithGetImageLocationsHeaders snapshots and merges ordinary headers, rejecting
// conflicting aliases within the supplied map.
func WithGetImageLocationsHeaders(values map[string]string) GetImageLocationsOption {
	apply := mergeImageMutationHeaders(values)
	return request.Custom(func(config *GetImageLocationsOpts) error {
		return applyImageLocationHeader(&config.Headers, apply)
	})
}

func applyImageLocationHeader(headers *map[string]string, apply func(*ImageMutationOpts) error) error {
	config := ImageMutationOpts{Headers: *headers}
	if err := apply(&config); err != nil {
		return err
	}
	*headers = config.Headers
	return nil
}

func copyAddImageLocationOpts(value AddImageLocationOpts) AddImageLocationOpts {
	value.Headers = copyGetImageLocationsOpts(GetImageLocationsOpts{Headers: value.Headers}).Headers
	if value.ValidationData != nil {
		pair := *value.ValidationData
		value.ValidationData = &pair
	}
	return value
}

func copyGetImageLocationsOpts(value GetImageLocationsOpts) GetImageLocationsOpts {
	value.Headers = maps.Clone(value.Headers)
	if value.Headers == nil {
		value.Headers = make(map[string]string)
	}
	return value
}

func parseAddImageLocationOptions(options []AddImageLocationOption) (AddImageLocationOpts, error) {
	config := copyAddImageLocationOpts(AddImageLocationOpts{})
	for _, apply := range calls.Translate(options, addImageLocationCalls) {
		if apply == nil {
			return config, uploadInvalid("nil add image location option")
		}
		candidate := copyAddImageLocationOpts(config)
		if err := apply(&candidate); err != nil {
			return config, err
		}
		config = copyAddImageLocationOpts(candidate)
	}
	if pair := config.ValidationData; pair != nil {
		if pair.OSHashAlgo == "" || pair.OSHashValue == "" || !utf8.ValidString(pair.OSHashAlgo) || !utf8.ValidString(pair.OSHashValue) {
			return config, uploadInvalid("location validation requires two nonempty valid UTF-8 hash fields")
		}
	}
	headers, err := imageMutationHeaders(config.Headers, false, "")
	config.Headers = headers
	return config, err
}

func parseGetImageLocationsOptions(options []GetImageLocationsOption) (GetImageLocationsOpts, error) {
	config := copyGetImageLocationsOpts(GetImageLocationsOpts{})
	for _, apply := range calls.Translate(options, getImageLocationsCalls) {
		if apply == nil {
			return config, uploadInvalid("nil get image locations option")
		}
		candidate := copyGetImageLocationsOpts(config)
		if err := apply(&candidate); err != nil {
			return config, err
		}
		config = copyGetImageLocationsOpts(candidate)
	}
	headers, err := imageMutationHeaders(config.Headers, false, "")
	config.Headers = headers
	return config, err
}
