package objects

import (
	"context"
	"strings"
)

type DirectoryMarkerOpts struct {
	Headers, Metadata map[string]string
}
type DirectoryMarkerOption func(*DirectoryMarkerOpts) error

func WithDirectoryMarkerOpts(value DirectoryMarkerOpts) DirectoryMarkerOption {
	snapshot := cloneDirectoryMarkerOpts(value)
	return func(cfg *DirectoryMarkerOpts) error { *cfg = cloneDirectoryMarkerOpts(snapshot); return nil }
}
func WithDirectoryMarkerHeader(key, value string) DirectoryMarkerOption {
	return WithDirectoryMarkerHeaders(map[string]string{key: value})
}
func WithDirectoryMarkerHeaders(values map[string]string) DirectoryMarkerOption {
	snapshot := cloneMetadataHeaders(values)
	return func(cfg *DirectoryMarkerOpts) error {
		if _, err := validateDirectoryMarkerHeaders(snapshot); err != nil {
			return err
		}
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		return mergeMetadataHeaders(cfg.Headers, snapshot)
	}
}
func WithDirectoryMarkerMetadata(values map[string]string) DirectoryMarkerOption {
	snapshot := cloneMetadataHeaders(values)
	return func(cfg *DirectoryMarkerOpts) error {
		owned, err := metadataInput(snapshot)()
		if err != nil {
			return err
		}
		if cfg.Metadata == nil {
			cfg.Metadata = make(map[string]string)
		}
		for key, value := range owned {
			for previous := range cfg.Metadata {
				if strings.EqualFold(previous, key) {
					delete(cfg.Metadata, previous)
				}
			}
			cfg.Metadata[key] = value
		}
		return nil
	}
}
func WithDirectoryMarkerMetadataValue(key, value string) DirectoryMarkerOption {
	return WithDirectoryMarkerMetadata(map[string]string{key: value})
}

func cloneDirectoryMarkerOpts(value DirectoryMarkerOpts) DirectoryMarkerOpts {
	value.Headers, value.Metadata = cloneMetadataHeaders(value.Headers), cloneMetadataHeaders(value.Metadata)
	return value
}
func validateDirectoryMarkerHeaders(values map[string]string) (map[string]string, error) {
	headers, err := validateCreateObjectHeaders(values)
	if err != nil {
		return nil, err
	}
	return headers, directoryMarkerHeaderPolicy(headers)
}
func directoryMarkerHeaderPolicy(values map[string]string) error {
	for key := range values {
		if strings.EqualFold(key, "X-Detect-Content-Type") {
			return metadataInvalid("directory marker header %q is SDK owned", key)
		}
	}
	return nil
}
func (p *preparedCreateObject) applyDirectoryMarkerOptions(ctx context.Context, options []DirectoryMarkerOption) (DirectoryMarkerOpts, error) {
	cfg := DirectoryMarkerOpts{}
	for _, option := range options {
		if err := p.guard(ctx); err != nil {
			return cfg, err
		}
		if option == nil {
			return cfg, metadataInvalid("nil directory marker option")
		}
		callback := cloneDirectoryMarkerOpts(cfg)
		err := option(&callback)
		cfg = cloneDirectoryMarkerOpts(callback)
		if err = joinMetadataErrors(err, p.guard(ctx)); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
