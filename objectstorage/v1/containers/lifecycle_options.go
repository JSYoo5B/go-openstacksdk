package containers

// CreateContainerOpts owns ordinary headers and custom metadata for a PUT.
type CreateContainerOpts struct {
	Headers  map[string]string
	Metadata map[string]string
}
type CreateContainerOption func(*CreateContainerOpts) error

// DeleteContainerOpts owns ordinary headers and an optional missing policy.
// A nil IgnoreMissing defaults to true.
type DeleteContainerOpts struct {
	Headers       map[string]string
	IgnoreMissing *bool
}
type DeleteContainerOption func(*DeleteContainerOpts) error

func WithCreateContainerOpts(value CreateContainerOpts) CreateContainerOption {
	snapshot := cloneCreateContainerOpts(value)
	return func(cfg *CreateContainerOpts) error { *cfg = cloneCreateContainerOpts(snapshot); return nil }
}
func WithCreateContainerHeader(key, value string) CreateContainerOption {
	return WithCreateContainerHeaders(map[string]string{key: value})
}
func WithCreateContainerHeaders(values map[string]string) CreateContainerOption {
	snapshot := cloneMetadataHeaders(values)
	return func(cfg *CreateContainerOpts) error { return mergeMetadataHeaders(cfg.Headers, snapshot) }
}

// WithCreateContainerMetadata replaces the complete custom metadata input.
func WithCreateContainerMetadata(values map[string]string) CreateContainerOption {
	snapshot := cloneMetadataHeaders(values)
	return func(cfg *CreateContainerOpts) error { cfg.Metadata = cloneMetadataHeaders(snapshot); return nil }
}
func WithDeleteContainerOpts(value DeleteContainerOpts) DeleteContainerOption {
	snapshot := cloneDeleteContainerOpts(value)
	return func(cfg *DeleteContainerOpts) error { *cfg = cloneDeleteContainerOpts(snapshot); return nil }
}
func WithDeleteContainerHeader(key, value string) DeleteContainerOption {
	return WithDeleteContainerHeaders(map[string]string{key: value})
}
func WithDeleteContainerHeaders(values map[string]string) DeleteContainerOption {
	snapshot := cloneMetadataHeaders(values)
	return func(cfg *DeleteContainerOpts) error { return mergeMetadataHeaders(cfg.Headers, snapshot) }
}
func WithDeleteContainerIgnoreMissing(value bool) DeleteContainerOption {
	return func(cfg *DeleteContainerOpts) error { owned := value; cfg.IgnoreMissing = &owned; return nil }
}

func cloneCreateContainerOpts(value CreateContainerOpts) CreateContainerOpts {
	value.Headers = cloneMetadataHeaders(value.Headers)
	value.Metadata = cloneMetadataHeaders(value.Metadata)
	return value
}
func cloneDeleteContainerOpts(value DeleteContainerOpts) DeleteContainerOpts {
	value.Headers = cloneMetadataHeaders(value.Headers)
	if value.IgnoreMissing != nil {
		owned := *value.IgnoreMissing
		value.IgnoreMissing = &owned
	}
	return value
}
func applyCreateContainerOptions(options []CreateContainerOption) (CreateContainerOpts, error) {
	cfg := cloneCreateContainerOpts(CreateContainerOpts{})
	for _, option := range options {
		if option == nil {
			return cfg, metadataInvalid("nil create container option")
		}
		callback := cloneCreateContainerOpts(cfg)
		err := option(&callback)
		cfg = cloneCreateContainerOpts(callback)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
func applyDeleteContainerOptions(options []DeleteContainerOption) (DeleteContainerOpts, error) {
	cfg := cloneDeleteContainerOpts(DeleteContainerOpts{})
	for _, option := range options {
		if option == nil {
			return cfg, metadataInvalid("nil delete container option")
		}
		callback := cloneDeleteContainerOpts(cfg)
		err := option(&callback)
		cfg = cloneDeleteContainerOpts(callback)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
