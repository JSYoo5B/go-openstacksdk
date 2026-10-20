package v1

import "time"

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
type GenerateTempURLOption func(*GenerateTempURLOpts) error

func WithGenerateTempURLOpts(value GenerateTempURLOpts) GenerateTempURLOption {
	snapshot := cloneGenerateTempURLOpts(value)
	return func(cfg *GenerateTempURLOpts) error { *cfg = cloneGenerateTempURLOpts(snapshot); return nil }
}
func WithGenerateTempURLKey(value []byte) GenerateTempURLOption {
	snapshot := cloneSigningKey(value)
	return func(cfg *GenerateTempURLOpts) error { cfg.Key = cloneSigningKey(snapshot); return nil }
}
func WithGenerateTempURLDigest(value TempURLDigest) GenerateTempURLOption {
	return func(cfg *GenerateTempURLOpts) error { cfg.Digest = value; return nil }
}
func WithGenerateTempURLTimestamp(value time.Time) GenerateTempURLOption {
	return func(cfg *GenerateTempURLOpts) error { owned := value; cfg.Timestamp = &owned; return nil }
}
func WithGenerateTempURLHeader(key, value string) GenerateTempURLOption {
	return WithGenerateTempURLHeaders(map[string]string{key: value})
}
func WithGenerateTempURLHeaders(values map[string]string) GenerateTempURLOption {
	snapshot := cloneTempURLKeyHeaders(values)
	return func(cfg *GenerateTempURLOpts) error {
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		return mergeTempURLKeyHeaders(cfg.Headers, snapshot)
	}
}
func WithGenerateTempURLNewest(value bool) GenerateTempURLOption {
	return func(cfg *GenerateTempURLOpts) error { owned := value; cfg.Newest = &owned; return nil }
}
func WithoutGenerateTempURLNewest() GenerateTempURLOption {
	return func(cfg *GenerateTempURLOpts) error { cfg.Newest = nil; return nil }
}
func WithGenerateTempURLAbsolute(value bool) GenerateTempURLOption {
	return func(cfg *GenerateTempURLOpts) error { cfg.Absolute = value; return nil }
}
func WithGenerateTempURLPrefix(value bool) GenerateTempURLOption {
	return func(cfg *GenerateTempURLOpts) error { cfg.Prefix = value; return nil }
}
func WithGenerateTempURLISO8601(value bool) GenerateTempURLOption {
	return func(cfg *GenerateTempURLOpts) error { cfg.ISO8601 = value; return nil }
}
func WithGenerateTempURLIPRange(value string) GenerateTempURLOption {
	return func(cfg *GenerateTempURLOpts) error { cfg.IPRange = value; return nil }
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
	for _, option := range options {
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
