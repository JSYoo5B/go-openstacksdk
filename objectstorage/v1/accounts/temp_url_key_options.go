package accounts

import (
	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// SetTempURLKeyOpts selects the primary key by default and owns ordinary headers.
type SetTempURLKeyOpts struct {
	Headers   map[string]string
	Secondary bool
}

// SetTempURLKeyOption is one option of SetTempURLKey: the WithSetTempURLKey*
// options, request.Header (validated and merged like a one-entry
// WithSetTempURLKeyHeaders) or request.Custom over *SetTempURLKeyOpts. Other
// shared calls, including request.Headers (the WithSetTempURLKeyHeaders map
// merges instead of replacing), fail with resource.ErrInvalidOption before
// HTTP.
type SetTempURLKeyOption = request.WriteCall

var setTempURLKeyCalls = calls.Spec[SetTempURLKeyOpts]{Header: calls.MergeHeader[SetTempURLKeyOpts](WithSetTempURLKeyHeaders)}

func WithSetTempURLKeyOpts(value SetTempURLKeyOpts) SetTempURLKeyOption {
	snapshot := cloneSetTempURLKeyOpts(value)
	return request.Custom(func(cfg *SetTempURLKeyOpts) error { *cfg = cloneSetTempURLKeyOpts(snapshot); return nil })
}
func WithSetTempURLKeyHeaders(values map[string]string) SetTempURLKeyOption {
	snapshot := cloneMetadataHeaders(values)
	return request.Custom(func(cfg *SetTempURLKeyOpts) error {
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		return mergeMetadataHeaders(cfg.Headers, snapshot)
	})
}
func WithSetTempURLKeySecondary(value bool) SetTempURLKeyOption {
	return request.Custom(func(cfg *SetTempURLKeyOpts) error { cfg.Secondary = value; return nil })
}

func cloneSetTempURLKeyOpts(value SetTempURLKeyOpts) SetTempURLKeyOpts {
	value.Headers = cloneMetadataHeaders(value.Headers)
	return value
}
func applySetTempURLKeyOptions(options []SetTempURLKeyOption) (SetTempURLKeyOpts, error) {
	cfg := cloneSetTempURLKeyOpts(SetTempURLKeyOpts{})
	for _, option := range calls.Translate(options, setTempURLKeyCalls) {
		if option == nil {
			return cfg, metadataInvalid("nil Temp URL key option")
		}
		callback := cloneSetTempURLKeyOpts(cfg)
		err := option(&callback)
		cfg = cloneSetTempURLKeyOpts(callback)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
