package objects

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type IsObjectStaleOpts struct {
	Headers     map[string]string
	MD5, SHA256 string
}

// IsObjectStaleOption is one option of IsObjectStale: the WithIsObjectStale*
// options, request.Header (validated and merged like a one-entry
// WithIsObjectStaleHeaders) or request.Custom over *IsObjectStaleOpts. Other
// shared calls, including request.Headers (the WithIsObjectStaleHeaders map
// merges instead of replacing), fail with resource.ErrInvalidOption before
// HTTP.
type IsObjectStaleOption = request.GetCall

var isObjectStaleCalls = calls.Spec[IsObjectStaleOpts]{Header: calls.MergeHeader[IsObjectStaleOpts](WithIsObjectStaleHeaders)}

func WithIsObjectStaleOpts(value IsObjectStaleOpts) IsObjectStaleOption {
	snapshot := cloneIsObjectStaleOpts(value)
	return request.Custom(func(cfg *IsObjectStaleOpts) error { *cfg = cloneIsObjectStaleOpts(snapshot); return nil })
}
func WithIsObjectStaleHeaders(values map[string]string) IsObjectStaleOption {
	snapshot := cloneMetadataHeaders(values)
	return request.Custom(func(cfg *IsObjectStaleOpts) error {
		if _, err := validateCreateObjectHeaders(snapshot); err != nil {
			return err
		}
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		return mergeMetadataHeaders(cfg.Headers, snapshot)
	})
}
func WithIsObjectStaleMD5(value string) IsObjectStaleOption {
	return request.Custom(func(cfg *IsObjectStaleOpts) error { cfg.MD5 = value; return nil })
}
func WithIsObjectStaleSHA256(value string) IsObjectStaleOption {
	return request.Custom(func(cfg *IsObjectStaleOpts) error { cfg.SHA256 = value; return nil })
}
func cloneIsObjectStaleOpts(value IsObjectStaleOpts) IsObjectStaleOpts {
	value.Headers = cloneMetadataHeaders(value.Headers)
	return value
}
func (p *preparedCreateObject) applyStaleOptions(ctx context.Context, options []IsObjectStaleOption) (IsObjectStaleOpts, error) {
	cfg := IsObjectStaleOpts{}
	for _, option := range calls.Translate(options, isObjectStaleCalls) {
		if err := p.guard(ctx); err != nil {
			return cfg, err
		}
		if option == nil {
			return cfg, metadataInvalid("nil IsObjectStale option")
		}
		callback := cloneIsObjectStaleOpts(cfg)
		err := option(&callback)
		cfg = cloneIsObjectStaleOpts(callback)
		if err = joinMetadataErrors(err, p.guard(ctx)); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
