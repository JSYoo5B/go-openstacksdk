package accounts

// SetTempURLKeyOpts selects the primary key by default and owns ordinary headers.
type SetTempURLKeyOpts struct {
	Headers   map[string]string
	Secondary bool
}
type SetTempURLKeyOption func(*SetTempURLKeyOpts) error

func WithSetTempURLKeyOpts(value SetTempURLKeyOpts) SetTempURLKeyOption {
	snapshot := cloneSetTempURLKeyOpts(value)
	return func(cfg *SetTempURLKeyOpts) error { *cfg = cloneSetTempURLKeyOpts(snapshot); return nil }
}
func WithSetTempURLKeyHeader(key, value string) SetTempURLKeyOption {
	return WithSetTempURLKeyHeaders(map[string]string{key: value})
}
func WithSetTempURLKeyHeaders(values map[string]string) SetTempURLKeyOption {
	snapshot := cloneMetadataHeaders(values)
	return func(cfg *SetTempURLKeyOpts) error {
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		return mergeMetadataHeaders(cfg.Headers, snapshot)
	}
}
func WithSetTempURLKeySecondary(value bool) SetTempURLKeyOption {
	return func(cfg *SetTempURLKeyOpts) error { cfg.Secondary = value; return nil }
}

func cloneSetTempURLKeyOpts(value SetTempURLKeyOpts) SetTempURLKeyOpts {
	value.Headers = cloneMetadataHeaders(value.Headers)
	return value
}
func applySetTempURLKeyOptions(options []SetTempURLKeyOption) (SetTempURLKeyOpts, error) {
	cfg := cloneSetTempURLKeyOpts(SetTempURLKeyOpts{})
	for _, option := range options {
		if option == nil {
			return cfg, metadataInvalid("nil Temp URL key option")
		}
		callback := cloneSetTempURLKeyOpts(cfg)
		err := option(&callback)
		cfg = cloneSetTempURLKeyOpts(callback)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
