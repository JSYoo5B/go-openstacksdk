package v1

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GetTempURLKeyOpts owns both read phases' preferences. Empty Container reads
// only account metadata; nil Newest leaves that header absent.
type GetTempURLKeyOpts struct {
	Container string
	Headers   map[string]string
	Newest    *bool
}

// GetTempURLKeyOption is one option of GetTempURLKey: the WithGetTempURLKey*
// options, request.Header (validated and merged like a one-entry
// WithGetTempURLKeyHeaders) or request.Custom over *GetTempURLKeyOpts. Other
// shared calls, including request.Headers (the WithGetTempURLKeyHeaders map
// merges instead of replacing), fail with resource.ErrInvalidOption before
// HTTP.
type GetTempURLKeyOption = request.GetCall

var getTempURLKeyCalls = calls.Spec[GetTempURLKeyOpts]{Header: calls.MergeHeader[GetTempURLKeyOpts](WithGetTempURLKeyHeaders)}

func WithGetTempURLKeyOpts(value GetTempURLKeyOpts) GetTempURLKeyOption {
	snapshot := cloneGetTempURLKeyOpts(value)
	return request.Custom(func(cfg *GetTempURLKeyOpts) error { *cfg = cloneGetTempURLKeyOpts(snapshot); return nil })
}
func WithGetTempURLKeyContainer(value string) GetTempURLKeyOption {
	return request.Custom(func(cfg *GetTempURLKeyOpts) error { cfg.Container = value; return nil })
}
func WithGetTempURLKeyHeaders(values map[string]string) GetTempURLKeyOption {
	snapshot := cloneTempURLKeyHeaders(values)
	return request.Custom(func(cfg *GetTempURLKeyOpts) error {
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		return mergeTempURLKeyHeaders(cfg.Headers, snapshot)
	})
}
func WithGetTempURLKeyNewest(value bool) GetTempURLKeyOption {
	return request.Custom(func(cfg *GetTempURLKeyOpts) error { owned := value; cfg.Newest = &owned; return nil })
}
func WithoutGetTempURLKeyNewest() GetTempURLKeyOption {
	return request.Custom(func(cfg *GetTempURLKeyOpts) error { cfg.Newest = nil; return nil })
}

func cloneTempURLKeyHeaders(value map[string]string) map[string]string {
	result := make(map[string]string, len(value))
	for key, value := range value {
		result[key] = value
	}
	return result
}
func cloneGetTempURLKeyOpts(value GetTempURLKeyOpts) GetTempURLKeyOpts {
	value.Headers = cloneTempURLKeyHeaders(value.Headers)
	if value.Newest != nil {
		owned := *value.Newest
		value.Newest = &owned
	}
	return value
}
func applyGetTempURLKeyOptions(options []GetTempURLKeyOption) (GetTempURLKeyOpts, error) {
	cfg := cloneGetTempURLKeyOpts(GetTempURLKeyOpts{})
	for _, option := range calls.Translate(options, getTempURLKeyCalls) {
		if option == nil {
			return cfg, tempURLKeyInvalid("nil Temp URL key option")
		}
		callback := cloneGetTempURLKeyOpts(cfg)
		err := option(&callback)
		cfg = cloneGetTempURLKeyOpts(callback)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
