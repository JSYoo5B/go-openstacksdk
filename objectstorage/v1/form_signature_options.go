package v1

import (
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GenerateFormSignatureOpts owns signing inputs and preferences for any metadata reads.
type GenerateFormSignatureOpts struct {
	Key       []byte
	Digest    TempURLDigest
	Timestamp *time.Time
	Headers   map[string]string
	Newest    *bool
}

// GenerateFormSignatureOption is one option of GenerateFormSignature: the
// WithGenerateFormSignature* options, request.Header (validated and merged
// like a one-entry WithGenerateFormSignatureHeaders) or request.Custom over
// *GenerateFormSignatureOpts. Other shared calls, including request.Headers
// (the WithGenerateFormSignatureHeaders map merges instead of replacing),
// fail with resource.ErrInvalidOption before HTTP.
type GenerateFormSignatureOption = request.GetCall

var generateFormSignatureCalls = calls.Spec[GenerateFormSignatureOpts]{Header: calls.MergeHeader[GenerateFormSignatureOpts](WithGenerateFormSignatureHeaders)}

func WithGenerateFormSignatureOpts(value GenerateFormSignatureOpts) GenerateFormSignatureOption {
	snapshot := cloneGenerateFormSignatureOpts(value)
	return request.Custom(func(cfg *GenerateFormSignatureOpts) error {
		*cfg = cloneGenerateFormSignatureOpts(snapshot)
		return nil
	})
}
func WithGenerateFormSignatureKey(value []byte) GenerateFormSignatureOption {
	snapshot := cloneSigningKey(value)
	return request.Custom(func(cfg *GenerateFormSignatureOpts) error { cfg.Key = cloneSigningKey(snapshot); return nil })
}
func WithGenerateFormSignatureDigest(value TempURLDigest) GenerateFormSignatureOption {
	return request.Custom(func(cfg *GenerateFormSignatureOpts) error { cfg.Digest = value; return nil })
}
func WithGenerateFormSignatureTimestamp(value time.Time) GenerateFormSignatureOption {
	return request.Custom(func(cfg *GenerateFormSignatureOpts) error { owned := value; cfg.Timestamp = &owned; return nil })
}
func WithGenerateFormSignatureHeaders(values map[string]string) GenerateFormSignatureOption {
	snapshot := cloneTempURLKeyHeaders(values)
	return request.Custom(func(cfg *GenerateFormSignatureOpts) error {
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		return mergeTempURLKeyHeaders(cfg.Headers, snapshot)
	})
}
func WithGenerateFormSignatureNewest(value bool) GenerateFormSignatureOption {
	return request.Custom(func(cfg *GenerateFormSignatureOpts) error { owned := value; cfg.Newest = &owned; return nil })
}
func WithoutGenerateFormSignatureNewest() GenerateFormSignatureOption {
	return request.Custom(func(cfg *GenerateFormSignatureOpts) error { cfg.Newest = nil; return nil })
}

func cloneGenerateFormSignatureOpts(value GenerateFormSignatureOpts) GenerateFormSignatureOpts {
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
func applyGenerateFormSignatureOptions(options []GenerateFormSignatureOption) (GenerateFormSignatureOpts, error) {
	cfg := cloneGenerateFormSignatureOpts(GenerateFormSignatureOpts{})
	for _, option := range calls.Translate(options, generateFormSignatureCalls) {
		if option == nil {
			return cfg, tempURLKeyInvalid("nil signing option")
		}
		callback := cloneGenerateFormSignatureOpts(cfg)
		err := option(&callback)
		cfg = cloneGenerateFormSignatureOpts(callback)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
