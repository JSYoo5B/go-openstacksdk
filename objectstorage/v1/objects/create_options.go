package objects

import (
	"context"
	"strings"
)

type CreateObjectOpts struct {
	Headers, Metadata         map[string]string
	MD5, SHA256               string
	SegmentSize               *int64
	UseSLO, GenerateChecksums *bool
}
type CreateObjectOption func(*CreateObjectOpts) error

func WithCreateObjectOpts(value CreateObjectOpts) CreateObjectOption {
	snapshot := cloneCreateObjectOpts(value)
	return func(cfg *CreateObjectOpts) error { *cfg = cloneCreateObjectOpts(snapshot); return nil }
}
func WithCreateObjectHeader(key, value string) CreateObjectOption {
	return WithCreateObjectHeaders(map[string]string{key: value})
}
func WithCreateObjectHeaders(values map[string]string) CreateObjectOption {
	snapshot := cloneMetadataHeaders(values)
	return func(cfg *CreateObjectOpts) error {
		if _, err := validateCreateObjectHeaders(snapshot); err != nil {
			return err
		}
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		return mergeMetadataHeaders(cfg.Headers, snapshot)
	}
}
func WithCreateObjectMetadata(values map[string]string) CreateObjectOption {
	snapshot := cloneMetadataHeaders(values)
	return func(cfg *CreateObjectOpts) error {
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
func WithCreateObjectMetadataValue(key, value string) CreateObjectOption {
	return WithCreateObjectMetadata(map[string]string{key: value})
}
func WithCreateObjectMD5(value string) CreateObjectOption {
	return func(cfg *CreateObjectOpts) error { cfg.MD5 = value; return nil }
}
func WithCreateObjectSHA256(value string) CreateObjectOption {
	return func(cfg *CreateObjectOpts) error { cfg.SHA256 = value; return nil }
}
func WithCreateObjectSegmentSize(value int64) CreateObjectOption {
	return func(cfg *CreateObjectOpts) error { owned := value; cfg.SegmentSize = &owned; return nil }
}
func WithoutCreateObjectSegmentSize() CreateObjectOption {
	return func(cfg *CreateObjectOpts) error { cfg.SegmentSize = nil; return nil }
}
func WithCreateObjectUseSLO(value bool) CreateObjectOption {
	return func(cfg *CreateObjectOpts) error { cfg.UseSLO = cloneDeleteObjectBool(&value); return nil }
}
func WithoutCreateObjectUseSLO() CreateObjectOption {
	return func(cfg *CreateObjectOpts) error { cfg.UseSLO = nil; return nil }
}
func WithCreateObjectGenerateChecksums(value bool) CreateObjectOption {
	return func(cfg *CreateObjectOpts) error { cfg.GenerateChecksums = cloneDeleteObjectBool(&value); return nil }
}
func WithoutCreateObjectGenerateChecksums() CreateObjectOption {
	return func(cfg *CreateObjectOpts) error { cfg.GenerateChecksums = nil; return nil }
}

func cloneCreateObjectOpts(value CreateObjectOpts) CreateObjectOpts {
	value.Headers, value.Metadata = cloneMetadataHeaders(value.Headers), cloneMetadataHeaders(value.Metadata)
	value.UseSLO, value.GenerateChecksums = cloneDeleteObjectBool(value.UseSLO), cloneDeleteObjectBool(value.GenerateChecksums)
	if value.SegmentSize != nil {
		owned := *value.SegmentSize
		value.SegmentSize = &owned
	}
	return value
}
func (p *preparedCreateObject) applyCreateOptions(ctx context.Context, options []CreateObjectOption) (CreateObjectOpts, error) {
	cfg := CreateObjectOpts{}
	for _, option := range options {
		if err := p.guard(ctx); err != nil {
			return cfg, err
		}
		if option == nil {
			return cfg, metadataInvalid("nil CreateObject option")
		}
		callback := cloneCreateObjectOpts(cfg)
		err := option(&callback)
		cfg = cloneCreateObjectOpts(callback)
		if err = joinMetadataErrors(err, p.guard(ctx)); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
