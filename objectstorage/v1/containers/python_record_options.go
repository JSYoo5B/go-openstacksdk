package containers

import (
	"context"
	"errors"
)

// RecordCreateOpts contains ordinary HTTP extensions; Source attrs and metadata
// are supplied separately to CreateRecord and snapshotted before callbacks.
type RecordCreateOpts struct{ Headers map[string]string }
type RecordCreateOption func(*RecordCreateOpts) error

func WithRecordCreateHeaders(values map[string]string) RecordCreateOption {
	owned := cloneMetadataHeaders(values)
	return func(cfg *RecordCreateOpts) error { return mergeMetadataHeaders(cfg.Headers, owned) }
}
func WithRecordCreateHeader(key, value string) RecordCreateOption {
	return WithRecordCreateHeaders(map[string]string{key: value})
}

type RecordDeleteOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}
type RecordDeleteOption func(*RecordDeleteOpts) error

func WithRecordDeleteHeaders(values map[string]string) RecordDeleteOption {
	owned := cloneMetadataHeaders(values)
	return func(cfg *RecordDeleteOpts) error { return mergeMetadataHeaders(cfg.Headers, owned) }
}
func WithRecordDeleteHeader(key, value string) RecordDeleteOption {
	return WithRecordDeleteHeaders(map[string]string{key: value})
}
func WithRecordDeleteIgnoreMissing(value bool) RecordDeleteOption {
	return func(cfg *RecordDeleteOpts) error { owned := value; cfg.IgnoreMissing = &owned; return nil }
}
func copyRecordCreate(cfg RecordCreateOpts) RecordCreateOpts {
	cfg.Headers = cloneMetadataHeaders(cfg.Headers)
	return cfg
}
func copyRecordDelete(cfg RecordDeleteOpts) RecordDeleteOpts {
	cfg.Headers = cloneMetadataHeaders(cfg.Headers)
	if cfg.IgnoreMissing != nil {
		owned := *cfg.IgnoreMissing
		cfg.IgnoreMissing = &owned
	}
	return cfg
}
func applyRecordOptions[T any, O ~func(*T) error](ctx context.Context, p *preparedMetadata, cfg T, copyValue func(T) T, options []O) (T, error) {
	for _, option := range options {
		if err := p.check(ctx); err != nil {
			return cfg, err
		}
		if option == nil {
			return cfg, metadataInvalid("nil container record option")
		}
		next := copyValue(cfg)
		err := option(&next)
		if err = errors.Join(err, p.check(ctx)); err != nil {
			return cfg, err
		}
		cfg = copyValue(next)
	}
	return cfg, p.check(ctx)
}
