package v1

// GetTempURLKeyOpts owns both read phases' preferences. Empty Container reads
// only account metadata; nil Newest leaves that header absent.
type GetTempURLKeyOpts struct {
	Container string
	Headers   map[string]string
	Newest    *bool
}
type GetTempURLKeyOption func(*GetTempURLKeyOpts) error

func WithGetTempURLKeyOpts(value GetTempURLKeyOpts) GetTempURLKeyOption {
	snapshot := cloneGetTempURLKeyOpts(value)
	return func(cfg *GetTempURLKeyOpts) error { *cfg = cloneGetTempURLKeyOpts(snapshot); return nil }
}
func WithGetTempURLKeyContainer(value string) GetTempURLKeyOption {
	return func(cfg *GetTempURLKeyOpts) error { cfg.Container = value; return nil }
}
func WithGetTempURLKeyHeader(key, value string) GetTempURLKeyOption {
	return WithGetTempURLKeyHeaders(map[string]string{key: value})
}
func WithGetTempURLKeyHeaders(values map[string]string) GetTempURLKeyOption {
	snapshot := cloneTempURLKeyHeaders(values)
	return func(cfg *GetTempURLKeyOpts) error {
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		return mergeTempURLKeyHeaders(cfg.Headers, snapshot)
	}
}
func WithGetTempURLKeyNewest(value bool) GetTempURLKeyOption {
	return func(cfg *GetTempURLKeyOpts) error { owned := value; cfg.Newest = &owned; return nil }
}
func WithoutGetTempURLKeyNewest() GetTempURLKeyOption {
	return func(cfg *GetTempURLKeyOpts) error { cfg.Newest = nil; return nil }
}

func cloneTempURLKeyHeaders(value map[string]string) map[string]string {
	result := make(map[string]string, len(value))
	for key, value := range value {
		result[key] = value
	}
	return result
}
func cloneGetTempURLKeyOpts(value GetTempURLKeyOpts) GetTempURLKeyOpts {
	value.Headers = cloneTempURLKeyHeaders(value.Headers)
	if value.Newest != nil {
		owned := *value.Newest
		value.Newest = &owned
	}
	return value
}
func applyGetTempURLKeyOptions(options []GetTempURLKeyOption) (GetTempURLKeyOpts, error) {
	cfg := cloneGetTempURLKeyOpts(GetTempURLKeyOpts{})
	for _, option := range options {
		if option == nil {
			return cfg, tempURLKeyInvalid("nil Temp URL key option")
		}
		callback := cloneGetTempURLKeyOpts(cfg)
		err := option(&callback)
		cfg = cloneGetTempURLKeyOpts(callback)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
