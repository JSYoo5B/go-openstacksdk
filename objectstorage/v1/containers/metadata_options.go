package containers

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GetMetadataOpts owns optional headers and an explicit X-Newest preference.
type GetMetadataOpts struct {
	Headers map[string]string
	Newest  *bool
}

// GetMetadataOption is one option of GetMetadata: the WithGetMetadata*
// options, request.Header (validated and merged like a one-entry
// WithGetMetadataHeaders) or request.Custom over *GetMetadataOpts. Other
// shared calls, including request.Headers (the WithGetMetadataHeaders map
// merges instead of replacing), fail with resource.ErrInvalidOption before
// HTTP.
type GetMetadataOption = request.GetCall

var getMetadataCalls = calls.Spec[GetMetadataOpts]{Header: calls.MergeHeader[GetMetadataOpts](WithGetMetadataHeaders)}

// MetadataOpts owns ordinary headers for metadata mutations.
type MetadataOpts struct{ Headers map[string]string }

// MetadataOption is one option of the metadata mutations (SetMetadata,
// DeleteMetadata and their refresh forms): the WithMetadata* options,
// request.Header (validated and merged like a one-entry WithMetadataHeaders)
// or request.Custom over *MetadataOpts. Other shared calls, including
// request.Headers (the WithMetadataHeaders map merges instead of replacing),
// fail with resource.ErrInvalidOption before HTTP.
type MetadataOption = request.WriteCall

var metadataCalls = calls.Spec[MetadataOpts]{Header: calls.MergeHeader[MetadataOpts](WithMetadataHeaders)}

func WithGetMetadataOpts(value GetMetadataOpts) GetMetadataOption {
	snapshot := cloneGetMetadataOpts(value)
	return request.Custom(func(cfg *GetMetadataOpts) error { *cfg = cloneGetMetadataOpts(snapshot); return nil })
}
func WithGetMetadataHeaders(values map[string]string) GetMetadataOption {
	snapshot := cloneMetadataHeaders(values)
	return request.Custom(func(cfg *GetMetadataOpts) error { return mergeMetadataHeaders(cfg.Headers, snapshot) })
}
func WithGetMetadataNewest(value bool) GetMetadataOption {
	return request.Custom(func(cfg *GetMetadataOpts) error { owned := value; cfg.Newest = &owned; return nil })
}
func WithoutGetMetadataNewest() GetMetadataOption {
	return request.Custom(func(cfg *GetMetadataOpts) error { cfg.Newest = nil; return nil })
}
func WithMetadataOpts(value MetadataOpts) MetadataOption {
	snapshot := cloneMetadataOpts(value)
	return request.Custom(func(cfg *MetadataOpts) error { *cfg = cloneMetadataOpts(snapshot); return nil })
}
func WithMetadataHeaders(values map[string]string) MetadataOption {
	snapshot := cloneMetadataHeaders(values)
	return request.Custom(func(cfg *MetadataOpts) error { return mergeMetadataHeaders(cfg.Headers, snapshot) })
}

func cloneMetadataHeaders(value map[string]string) map[string]string {
	result := make(map[string]string, len(value))
	for key, value := range value {
		result[key] = value
	}
	return result
}
func cloneGetMetadataOpts(value GetMetadataOpts) GetMetadataOpts {
	value.Headers = cloneMetadataHeaders(value.Headers)
	if value.Newest != nil {
		owned := *value.Newest
		value.Newest = &owned
	}
	return value
}
func cloneMetadataOpts(value MetadataOpts) MetadataOpts {
	value.Headers = cloneMetadataHeaders(value.Headers)
	return value
}
func applyGetMetadataOptions(options []GetMetadataOption) (GetMetadataOpts, error) {
	cfg := cloneGetMetadataOpts(GetMetadataOpts{})
	for _, option := range calls.Translate(options, getMetadataCalls) {
		if option == nil {
			return cfg, metadataInvalid("nil metadata option")
		}
		callback := cloneGetMetadataOpts(cfg)
		err := option(&callback)
		cfg = cloneGetMetadataOpts(callback)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
func applyMetadataOptions(options []MetadataOption) (MetadataOpts, error) {
	cfg := cloneMetadataOpts(MetadataOpts{})
	for _, option := range calls.Translate(options, metadataCalls) {
		if option == nil {
			return cfg, metadataInvalid("nil metadata option")
		}
		callback := cloneMetadataOpts(cfg)
		err := option(&callback)
		cfg = cloneMetadataOpts(callback)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
