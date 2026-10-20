package v1

import (
	"context"
	"net/http"
)

type GetInfoOpts struct {
	Headers map[string]string
}
type GetInfoOption func(*GetInfoOpts) error

func WithGetInfoOpts(value GetInfoOpts) GetInfoOption {
	snapshot := cloneGetInfoOpts(value)
	return func(cfg *GetInfoOpts) error { *cfg = cloneGetInfoOpts(snapshot); return nil }
}
func WithGetInfoHeader(key, value string) GetInfoOption {
	return WithGetInfoHeaders(map[string]string{key: value})
}
func WithGetInfoHeaders(values map[string]string) GetInfoOption {
	snapshot := cloneTempURLKeyHeaders(values)
	return func(cfg *GetInfoOpts) error {
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		return mergeInfoHeaders(cfg.Headers, snapshot)
	}
}

// ObjectSegmentSizeOpts distinguishes the default 1 GiB request from explicit
// zero. Negative sizes are invalid before the capabilities request.
type ObjectSegmentSizeOpts struct {
	Headers map[string]string
	Size    *int64
}
type ObjectSegmentSizeOption func(*ObjectSegmentSizeOpts) error

func WithObjectSegmentSizeOpts(value ObjectSegmentSizeOpts) ObjectSegmentSizeOption {
	snapshot := cloneObjectSegmentSizeOpts(value)
	return func(cfg *ObjectSegmentSizeOpts) error { *cfg = cloneObjectSegmentSizeOpts(snapshot); return nil }
}
func WithObjectSegmentSizeHeader(key, value string) ObjectSegmentSizeOption {
	return WithObjectSegmentSizeHeaders(map[string]string{key: value})
}
func WithObjectSegmentSizeHeaders(values map[string]string) ObjectSegmentSizeOption {
	snapshot := cloneTempURLKeyHeaders(values)
	return func(cfg *ObjectSegmentSizeOpts) error {
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		return mergeInfoHeaders(cfg.Headers, snapshot)
	}
}
func WithObjectSegmentSize(value int64) ObjectSegmentSizeOption {
	return func(cfg *ObjectSegmentSizeOpts) error { owned := value; cfg.Size = &owned; return nil }
}
func WithoutObjectSegmentSize() ObjectSegmentSizeOption {
	return func(cfg *ObjectSegmentSizeOpts) error { cfg.Size = nil; return nil }
}

func cloneGetInfoOpts(value GetInfoOpts) GetInfoOpts {
	value.Headers = cloneTempURLKeyHeaders(value.Headers)
	return value
}
func cloneObjectSegmentSizeOpts(value ObjectSegmentSizeOpts) ObjectSegmentSizeOpts {
	value.Headers = cloneTempURLKeyHeaders(value.Headers)
	if value.Size != nil {
		owned := *value.Size
		value.Size = &owned
	}
	return value
}
func mergeInfoHeaders(dst, values map[string]string) error {
	owned, err := validateInfoHeaders(values)
	if err != nil {
		return err
	}
	for key, value := range owned {
		for previous := range dst {
			if tempURLKeyToken(previous) && http.CanonicalHeaderKey(previous) == key {
				delete(dst, previous)
			}
		}
		dst[key] = value
	}
	return nil
}

func (p *preparedInfo) applyGetInfoOptions(ctx context.Context, options []GetInfoOption) (GetInfoOpts, error) {
	cfg := cloneGetInfoOpts(GetInfoOpts{})
	for _, option := range options {
		if err := p.guard(ctx); err != nil {
			return cfg, err
		}
		if option == nil {
			return cfg, tempURLKeyInvalid("nil GetInfo option")
		}
		callback := cloneGetInfoOpts(cfg)
		err := option(&callback)
		cfg = cloneGetInfoOpts(callback)
		if err = joinTempURLKeyErrors(err, p.guard(ctx)); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
func (p *preparedInfo) applySegmentOptions(ctx context.Context, options []ObjectSegmentSizeOption) (ObjectSegmentSizeOpts, error) {
	cfg := cloneObjectSegmentSizeOpts(ObjectSegmentSizeOpts{})
	for _, option := range options {
		if err := p.guard(ctx); err != nil {
			return cfg, err
		}
		if option == nil {
			return cfg, tempURLKeyInvalid("nil object segment size option")
		}
		callback := cloneObjectSegmentSizeOpts(cfg)
		err := option(&callback)
		cfg = cloneObjectSegmentSizeOpts(callback)
		if err = joinTempURLKeyErrors(err, p.guard(ctx)); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
