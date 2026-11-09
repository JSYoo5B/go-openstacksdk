package v1

import (
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GenerateTempURLOpts owns signing inputs and preferences for any metadata reads.
type GenerateTempURLOpts struct {
	Key       []byte
	Digest    TempURLDigest
	Timestamp *time.Time
	Headers   map[string]string
	Newest    *bool
	Absolute  bool
	Prefix    bool
	ISO8601   bool
	IPRange   string
}

// GenerateTempURLOption is one option of GenerateTempURL: the
// WithGenerateTempURL* options, request.Header (validated and merged like a
// one-entry WithGenerateTempURLHeaders) or request.Custom over
// *GenerateTempURLOpts. Other shared calls, including request.Headers (the
// WithGenerateTempURLHeaders map merges instead of replacing), fail with
// resource.ErrInvalidOption before HTTP.
type GenerateTempURLOption = request.GetCall

var generateTempURLCalls = calls.Spec[GenerateTempURLOpts]{Header: calls.MergeHeader[GenerateTempURLOpts](WithGenerateTempURLHeaders)}

func WithGenerateTempURLOpts(value GenerateTempURLOpts) GenerateTempURLOption {
	snapshot := cloneGenerateTempURLOpts(value)
	return request.Custom(func(cfg *GenerateTempURLOpts) error { *cfg = cloneGenerateTempURLOpts(snapshot); return nil })
}
func WithGenerateTempURLKey(value []byte) GenerateTempURLOption {
	snapshot := cloneSigningKey(value)
	return request.Custom(func(cfg *GenerateTempURLOpts) error { cfg.Key = cloneSigningKey(snapshot); return nil })
}
func WithGenerateTempURLHeaders(values map[string]string) GenerateTempURLOption {
	snapshot := cloneTempURLKeyHeaders(values)
	return request.Custom(func(cfg *GenerateTempURLOpts) error {
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		return mergeTempURLKeyHeaders(cfg.Headers, snapshot)
	})
}

func cloneGenerateTempURLOpts(value GenerateTempURLOpts) GenerateTempURLOpts {
	value.Key = cloneSigningKey(value.Key)
	value.Headers = cloneTempURLKeyHeaders(value.Headers)
	if value.Timestamp != nil {
		owned := *value.Timestamp
		value.Timestamp = &owned
	}
	if value.Newest != nil {
		owned := *value.Newest
		value.Newest = &owned
	}
	return value
}
func applyGenerateTempURLOptions(options []GenerateTempURLOption) (GenerateTempURLOpts, error) {
	cfg := cloneGenerateTempURLOpts(GenerateTempURLOpts{})
	for _, option := range calls.Translate(options, generateTempURLCalls) {
		if option == nil {
			return cfg, tempURLKeyInvalid("nil signing option")
		}
		callback := cloneGenerateTempURLOpts(cfg)
		err := option(&callback)
		cfg = cloneGenerateTempURLOpts(callback)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
