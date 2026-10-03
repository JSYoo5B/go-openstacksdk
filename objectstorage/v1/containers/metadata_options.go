package containers

// GetMetadataOpts owns optional headers and an explicit X-Newest preference.
type GetMetadataOpts struct {
	Headers map[string]string
	Newest  *bool
}
type GetMetadataOption func(*GetMetadataOpts) error

// MetadataOpts owns ordinary headers for metadata mutations.
type MetadataOpts struct{ Headers map[string]string }
type MetadataOption func(*MetadataOpts) error

func WithGetMetadataOpts(value GetMetadataOpts) GetMetadataOption {
	snapshot := cloneGetMetadataOpts(value)
	return func(cfg *GetMetadataOpts) error { *cfg = cloneGetMetadataOpts(snapshot); return nil }
}
func WithGetMetadataHeader(key, value string) GetMetadataOption {
	return WithGetMetadataHeaders(map[string]string{key: value})
}
func WithGetMetadataHeaders(values map[string]string) GetMetadataOption {
	snapshot := cloneMetadataHeaders(values)
	return func(cfg *GetMetadataOpts) error { return mergeMetadataHeaders(cfg.Headers, snapshot) }
}
func WithGetMetadataNewest(value bool) GetMetadataOption {
	return func(cfg *GetMetadataOpts) error { owned := value; cfg.Newest = &owned; return nil }
}
func WithoutGetMetadataNewest() GetMetadataOption {
	return func(cfg *GetMetadataOpts) error { cfg.Newest = nil; return nil }
}
func WithMetadataOpts(value MetadataOpts) MetadataOption {
	snapshot := cloneMetadataOpts(value)
	return func(cfg *MetadataOpts) error { *cfg = cloneMetadataOpts(snapshot); return nil }
}
func WithMetadataHeader(key, value string) MetadataOption {
	return WithMetadataHeaders(map[string]string{key: value})
}
func WithMetadataHeaders(values map[string]string) MetadataOption {
	snapshot := cloneMetadataHeaders(values)
	return func(cfg *MetadataOpts) error { return mergeMetadataHeaders(cfg.Headers, snapshot) }
}

func cloneMetadataHeaders(value map[string]string) map[string]string {
	result := make(map[string]string, len(value))
	for key, value := range value {
		result[key] = value
	}
	return result
}
func cloneGetMetadataOpts(value GetMetadataOpts) GetMetadataOpts {
	value.Headers = cloneMetadataHeaders(value.Headers)
	if value.Newest != nil {
		owned := *value.Newest
		value.Newest = &owned
	}
	return value
}
func cloneMetadataOpts(value MetadataOpts) MetadataOpts {
	value.Headers = cloneMetadataHeaders(value.Headers)
	return value
}
func applyGetMetadataOptions(options []GetMetadataOption) (GetMetadataOpts, error) {
	cfg := cloneGetMetadataOpts(GetMetadataOpts{})
	for _, option := range options {
		if option == nil {
			return cfg, metadataInvalid("nil metadata option")
		}
		callback := cloneGetMetadataOpts(cfg)
		err := option(&callback)
		cfg = cloneGetMetadataOpts(callback)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
func applyMetadataOptions(options []MetadataOption) (MetadataOpts, error) {
	cfg := cloneMetadataOpts(MetadataOpts{})
	for _, option := range options {
		if option == nil {
			return cfg, metadataInvalid("nil metadata option")
		}
		callback := cloneMetadataOpts(cfg)
		err := option(&callback)
		cfg = cloneMetadataOpts(callback)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
