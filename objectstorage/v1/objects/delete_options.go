package objects

import "context"

// DeleteObjectOpts owns both phases' preferences. A nil StaticLargeObject
// probes fresh HEAD; a known flag skips discovery. IgnoreMissing defaults true.
// VersionID selects the same literal version for discovery and deletion.
type DeleteObjectOpts struct {
	Headers                                  map[string]string
	StaticLargeObject, IgnoreMissing, Newest *bool
	VersionID                                string
}
type DeleteObjectOption func(*DeleteObjectOpts) error

func WithDeleteObjectOpts(value DeleteObjectOpts) DeleteObjectOption {
	snapshot := cloneDeleteObjectOpts(value)
	return func(cfg *DeleteObjectOpts) error { *cfg = cloneDeleteObjectOpts(snapshot); return nil }
}
func WithDeleteObjectHeader(key, value string) DeleteObjectOption {
	return WithDeleteObjectHeaders(map[string]string{key: value})
}
func WithDeleteObjectHeaders(values map[string]string) DeleteObjectOption {
	snapshot := cloneMetadataHeaders(values)
	return func(cfg *DeleteObjectOpts) error {
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		if _, err := validateDeleteObjectHeaders(snapshot); err != nil {
			return err
		}
		return mergeMetadataHeaders(cfg.Headers, snapshot)
	}
}
func WithDeleteObjectStaticLargeObject(value bool) DeleteObjectOption {
	return func(cfg *DeleteObjectOpts) error { cfg.StaticLargeObject = cloneDeleteObjectBool(&value); return nil }
}
func WithoutDeleteObjectStaticLargeObject() DeleteObjectOption {
	return func(cfg *DeleteObjectOpts) error { cfg.StaticLargeObject = nil; return nil }
}
func WithDeleteObjectIgnoreMissing(value bool) DeleteObjectOption {
	return func(cfg *DeleteObjectOpts) error { cfg.IgnoreMissing = cloneDeleteObjectBool(&value); return nil }
}
func WithoutDeleteObjectIgnoreMissing() DeleteObjectOption {
	return func(cfg *DeleteObjectOpts) error { cfg.IgnoreMissing = nil; return nil }
}
func WithDeleteObjectVersionID(value string) DeleteObjectOption {
	return func(cfg *DeleteObjectOpts) error { cfg.VersionID = value; return nil }
}
func WithDeleteObjectNewest(value bool) DeleteObjectOption {
	return func(cfg *DeleteObjectOpts) error { cfg.Newest = cloneDeleteObjectBool(&value); return nil }
}
func WithoutDeleteObjectNewest() DeleteObjectOption {
	return func(cfg *DeleteObjectOpts) error { cfg.Newest = nil; return nil }
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
	for _, option := range options {
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
