package objects

import "context"

type IsObjectStaleOpts struct {
	Headers     map[string]string
	MD5, SHA256 string
}
type IsObjectStaleOption func(*IsObjectStaleOpts) error

func WithIsObjectStaleOpts(value IsObjectStaleOpts) IsObjectStaleOption {
	snapshot := cloneIsObjectStaleOpts(value)
	return func(cfg *IsObjectStaleOpts) error { *cfg = cloneIsObjectStaleOpts(snapshot); return nil }
}
func WithIsObjectStaleHeader(key, value string) IsObjectStaleOption {
	return WithIsObjectStaleHeaders(map[string]string{key: value})
}
func WithIsObjectStaleHeaders(values map[string]string) IsObjectStaleOption {
	snapshot := cloneMetadataHeaders(values)
	return func(cfg *IsObjectStaleOpts) error {
		if _, err := validateCreateObjectHeaders(snapshot); err != nil {
			return err
		}
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		return mergeMetadataHeaders(cfg.Headers, snapshot)
	}
}
func WithIsObjectStaleMD5(value string) IsObjectStaleOption {
	return func(cfg *IsObjectStaleOpts) error { cfg.MD5 = value; return nil }
}
func WithIsObjectStaleSHA256(value string) IsObjectStaleOption {
	return func(cfg *IsObjectStaleOpts) error { cfg.SHA256 = value; return nil }
}
func cloneIsObjectStaleOpts(value IsObjectStaleOpts) IsObjectStaleOpts {
	value.Headers = cloneMetadataHeaders(value.Headers)
	return value
}
func (p *preparedCreateObject) applyStaleOptions(ctx context.Context, options []IsObjectStaleOption) (IsObjectStaleOpts, error) {
	cfg := IsObjectStaleOpts{}
	for _, option := range options {
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
