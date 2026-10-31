package objects

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// DeleteObjectOpts owns both phases' preferences. A nil StaticLargeObject
// probes fresh HEAD; a known flag skips discovery. IgnoreMissing defaults true.
// VersionID selects the same literal version for discovery and deletion.
type DeleteObjectOpts struct {
	Headers                                  map[string]string
	StaticLargeObject, IgnoreMissing, Newest *bool
	VersionID                                string
}

// DeleteObjectOption is one option of DeleteObject: the WithDeleteObject*
// options, request.Header (validated and merged like a one-entry
// WithDeleteObjectHeaders), request.IgnoreMissing or request.Custom over
// *DeleteObjectOpts. Other shared calls, including request.Headers (the
// WithDeleteObjectHeaders map merges instead of replacing), fail with
// resource.ErrInvalidOption before HTTP.
type DeleteObjectOption = request.DeleteCall

var deleteObjectCalls = calls.Spec[DeleteObjectOpts]{
	Header:        calls.MergeHeader[DeleteObjectOpts](WithDeleteObjectHeaders),
	IgnoreMissing: calls.Pointer(func(cfg *DeleteObjectOpts) **bool { return &cfg.IgnoreMissing }),
}

func WithDeleteObjectOpts(value DeleteObjectOpts) DeleteObjectOption {
	snapshot := cloneDeleteObjectOpts(value)
	return request.Custom(func(cfg *DeleteObjectOpts) error { *cfg = cloneDeleteObjectOpts(snapshot); return nil })
}
func WithDeleteObjectHeaders(values map[string]string) DeleteObjectOption {
	snapshot := cloneMetadataHeaders(values)
	return request.Custom(func(cfg *DeleteObjectOpts) error {
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		if _, err := validateDeleteObjectHeaders(snapshot); err != nil {
			return err
		}
		return mergeMetadataHeaders(cfg.Headers, snapshot)
	})
}
func WithDeleteObjectStaticLargeObject(value bool) DeleteObjectOption {
	return request.Custom(func(cfg *DeleteObjectOpts) error { cfg.StaticLargeObject = cloneDeleteObjectBool(&value); return nil })
}
func WithoutDeleteObjectStaticLargeObject() DeleteObjectOption {
	return request.Custom(func(cfg *DeleteObjectOpts) error { cfg.StaticLargeObject = nil; return nil })
}
func WithoutDeleteObjectIgnoreMissing() DeleteObjectOption {
	return request.Custom(func(cfg *DeleteObjectOpts) error { cfg.IgnoreMissing = nil; return nil })
}
func WithDeleteObjectVersionID(value string) DeleteObjectOption {
	return request.Custom(func(cfg *DeleteObjectOpts) error { cfg.VersionID = value; return nil })
}
func WithDeleteObjectNewest(value bool) DeleteObjectOption {
	return request.Custom(func(cfg *DeleteObjectOpts) error { cfg.Newest = cloneDeleteObjectBool(&value); return nil })
}
func WithoutDeleteObjectNewest() DeleteObjectOption {
	return request.Custom(func(cfg *DeleteObjectOpts) error { cfg.Newest = nil; return nil })
}

func cloneDeleteObjectBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	owned := *value
	return &owned
}
func cloneDeleteObjectOpts(value DeleteObjectOpts) DeleteObjectOpts {
	value.Headers = cloneMetadataHeaders(value.Headers)
	value.StaticLargeObject = cloneDeleteObjectBool(value.StaticLargeObject)
	value.IgnoreMissing = cloneDeleteObjectBool(value.IgnoreMissing)
	value.Newest = cloneDeleteObjectBool(value.Newest)
	return value
}
func (p *preparedDeleteObject) applyOptions(ctx context.Context, options []DeleteObjectOption) (DeleteObjectOpts, error) {
	cfg := cloneDeleteObjectOpts(DeleteObjectOpts{})
	for _, option := range calls.Translate(options, deleteObjectCalls) {
		if err := p.guard(ctx); err != nil {
			return cfg, err
		}
		if option == nil {
			return cfg, metadataInvalid("nil DeleteObject option")
		}
		callback := cloneDeleteObjectOpts(cfg)
		err := option(&callback)
		cfg = cloneDeleteObjectOpts(callback)
		if err = joinMetadataErrors(err, p.guard(ctx)); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
