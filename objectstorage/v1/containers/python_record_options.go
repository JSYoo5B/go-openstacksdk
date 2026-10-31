package containers

import (
	"context"
	"errors"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// RecordCreateOpts contains ordinary HTTP extensions; Source attrs and metadata
// are supplied separately to CreateRecord and snapshotted before callbacks.
type RecordCreateOpts struct{ Headers map[string]string }

// RecordCreateOption is one option of CreateRecord: the WithRecordCreate*
// options, request.Header (validated and merged like a one-entry
// WithRecordCreateHeaders) or request.Custom over *RecordCreateOpts. Other
// shared calls, including request.Headers (the WithRecordCreateHeaders map
// merges instead of replacing), fail with resource.ErrInvalidOption before
// HTTP.
type RecordCreateOption = request.WriteCall

var recordCreateCalls = calls.Spec[RecordCreateOpts]{Header: calls.MergeHeader[RecordCreateOpts](WithRecordCreateHeaders)}

func WithRecordCreateHeaders(values map[string]string) RecordCreateOption {
	owned := cloneMetadataHeaders(values)
	return request.Custom(func(cfg *RecordCreateOpts) error { return mergeMetadataHeaders(cfg.Headers, owned) })
}

type RecordDeleteOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}

// RecordDeleteOption is one option of DeleteRecord: the WithRecordDelete*
// options, request.Header (validated and merged like a one-entry
// WithRecordDeleteHeaders), request.IgnoreMissing or request.Custom over
// *RecordDeleteOpts. Other shared calls, including request.Headers (the
// WithRecordDeleteHeaders map merges instead of replacing), fail with
// resource.ErrInvalidOption before HTTP.
type RecordDeleteOption = request.DeleteCall

var recordDeleteCalls = calls.Spec[RecordDeleteOpts]{
	Header:        calls.MergeHeader[RecordDeleteOpts](WithRecordDeleteHeaders),
	IgnoreMissing: calls.Pointer(func(cfg *RecordDeleteOpts) **bool { return &cfg.IgnoreMissing }),
}

func WithRecordDeleteHeaders(values map[string]string) RecordDeleteOption {
	owned := cloneMetadataHeaders(values)
	return request.Custom(func(cfg *RecordDeleteOpts) error { return mergeMetadataHeaders(cfg.Headers, owned) })
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
func applyRecordOptions[T any](ctx context.Context, p *preparedMetadata, cfg T, copyValue func(T) T, options []func(*T) error) (T, error) {
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
