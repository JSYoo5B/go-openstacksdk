package objects

import (
	"context"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

type CreateObjectOpts struct {
	Headers, Metadata         map[string]string
	MD5, SHA256               string
	SegmentSize               *int64
	UseSLO, GenerateChecksums *bool
}

// CreateObjectOption is one option of CreateObject: the WithCreateObject*
// options, request.Header (validated and merged like a one-entry
// WithCreateObjectHeaders) or request.Custom over *CreateObjectOpts. Other
// shared calls, including request.Headers (the WithCreateObjectHeaders map
// merges instead of replacing), fail with resource.ErrInvalidOption before
// HTTP.
type CreateObjectOption = request.WriteCall

var createObjectCalls = calls.Spec[CreateObjectOpts]{Header: calls.MergeHeader[CreateObjectOpts](WithCreateObjectHeaders)}

func WithCreateObjectOpts(value CreateObjectOpts) CreateObjectOption {
	snapshot := cloneCreateObjectOpts(value)
	return request.Custom(func(cfg *CreateObjectOpts) error { *cfg = cloneCreateObjectOpts(snapshot); return nil })
}
func WithCreateObjectHeaders(values map[string]string) CreateObjectOption {
	snapshot := cloneMetadataHeaders(values)
	return request.Custom(func(cfg *CreateObjectOpts) error {
		if _, err := validateCreateObjectHeaders(snapshot); err != nil {
			return err
		}
		if cfg.Headers == nil {
			cfg.Headers = make(map[string]string)
		}
		return mergeMetadataHeaders(cfg.Headers, snapshot)
	})
}
func WithCreateObjectMetadata(values map[string]string) CreateObjectOption {
	snapshot := cloneMetadataHeaders(values)
	return request.Custom(func(cfg *CreateObjectOpts) error {
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
	})
}
func WithCreateObjectMetadataValue(key, value string) CreateObjectOption {
	return WithCreateObjectMetadata(map[string]string{key: value})
}
func WithCreateObjectMD5(value string) CreateObjectOption {
	return request.Custom(func(cfg *CreateObjectOpts) error { cfg.MD5 = value; return nil })
}
func WithCreateObjectSHA256(value string) CreateObjectOption {
	return request.Custom(func(cfg *CreateObjectOpts) error { cfg.SHA256 = value; return nil })
}
func WithCreateObjectSegmentSize(value int64) CreateObjectOption {
	return request.Custom(func(cfg *CreateObjectOpts) error { owned := value; cfg.SegmentSize = &owned; return nil })
}
func WithoutCreateObjectSegmentSize() CreateObjectOption {
	return request.Custom(func(cfg *CreateObjectOpts) error { cfg.SegmentSize = nil; return nil })
}
func WithCreateObjectUseSLO(value bool) CreateObjectOption {
	return request.Custom(func(cfg *CreateObjectOpts) error { cfg.UseSLO = cloneDeleteObjectBool(&value); return nil })
}
func WithoutCreateObjectUseSLO() CreateObjectOption {
	return request.Custom(func(cfg *CreateObjectOpts) error { cfg.UseSLO = nil; return nil })
}
func WithCreateObjectGenerateChecksums(value bool) CreateObjectOption {
	return request.Custom(func(cfg *CreateObjectOpts) error { cfg.GenerateChecksums = cloneDeleteObjectBool(&value); return nil })
}
func WithoutCreateObjectGenerateChecksums() CreateObjectOption {
	return request.Custom(func(cfg *CreateObjectOpts) error { cfg.GenerateChecksums = nil; return nil })
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
	for _, option := range calls.Translate(options, createObjectCalls) {
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
