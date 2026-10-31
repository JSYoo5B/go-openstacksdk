package objects

import (
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// ObjectReadOpts owns ordinary headers and explicit GET inputs. Range and
// conditional strings are literal header values; their semantics belong to
// Swift. BufferSize controls GetObject and DownloadObject, not stream Read.
type ObjectReadOpts struct {
	Headers                            map[string]string
	Newest                             *bool
	IfMatch, IfNoneMatch, Range        string
	IfModifiedSince, IfUnmodifiedSince *time.Time
	Filename, MultipartManifest        string
	Symlink, VersionID                 string
	BufferSize                         int
}

// ObjectReadOption is one option of GetObject, DownloadObject and
// StreamObject: the WithObjectRead* options, request.Header (validated and
// merged like a one-entry WithObjectReadHeaders) or request.Custom over
// *ObjectReadOpts. Other shared calls, including request.Headers (the
// WithObjectReadHeaders map merges instead of replacing), fail with
// resource.ErrInvalidOption before HTTP.
type ObjectReadOption = request.GetCall

var objectReadCalls = calls.Spec[ObjectReadOpts]{Header: calls.MergeHeader[ObjectReadOpts](WithObjectReadHeaders)}

func WithObjectReadOpts(value ObjectReadOpts) ObjectReadOption {
	snapshot := cloneObjectReadOpts(value)
	return request.Custom(func(cfg *ObjectReadOpts) error {
		*cfg = cloneObjectReadOpts(snapshot)
		return nil
	})
}

func WithObjectReadHeaders(values map[string]string) ObjectReadOption {
	snapshot := cloneMetadataHeaders(values)
	return request.Custom(func(cfg *ObjectReadOpts) error {
		owned, err := validateObjectReadHeaders(snapshot)
		if err != nil {
			return err
		}
		return mergeMetadataHeaders(cfg.Headers, owned)
	})
}

func WithObjectReadNewest(value bool) ObjectReadOption {
	return request.Custom(func(cfg *ObjectReadOpts) error {
		owned := value
		cfg.Newest = &owned
		return nil
	})
}

func WithoutObjectReadNewest() ObjectReadOption {
	return request.Custom(func(cfg *ObjectReadOpts) error { cfg.Newest = nil; return nil })
}

func WithObjectReadIfMatch(value string) ObjectReadOption {
	return request.Custom(func(cfg *ObjectReadOpts) error { cfg.IfMatch = value; return nil })
}

func WithObjectReadIfNoneMatch(value string) ObjectReadOption {
	return request.Custom(func(cfg *ObjectReadOpts) error { cfg.IfNoneMatch = value; return nil })
}

func WithObjectReadIfModifiedSince(value time.Time) ObjectReadOption {
	return request.Custom(func(cfg *ObjectReadOpts) error {
		owned := value
		cfg.IfModifiedSince = &owned
		return nil
	})
}

func WithoutObjectReadIfModifiedSince() ObjectReadOption {
	return request.Custom(func(cfg *ObjectReadOpts) error { cfg.IfModifiedSince = nil; return nil })
}

func WithObjectReadIfUnmodifiedSince(value time.Time) ObjectReadOption {
	return request.Custom(func(cfg *ObjectReadOpts) error {
		owned := value
		cfg.IfUnmodifiedSince = &owned
		return nil
	})
}

func WithoutObjectReadIfUnmodifiedSince() ObjectReadOption {
	return request.Custom(func(cfg *ObjectReadOpts) error { cfg.IfUnmodifiedSince = nil; return nil })
}

func WithObjectReadRange(value string) ObjectReadOption {
	return request.Custom(func(cfg *ObjectReadOpts) error { cfg.Range = value; return nil })
}

func WithObjectReadFilename(value string) ObjectReadOption {
	return request.Custom(func(cfg *ObjectReadOpts) error { cfg.Filename = value; return nil })
}

func WithObjectReadMultipartManifest(value string) ObjectReadOption {
	return request.Custom(func(cfg *ObjectReadOpts) error { cfg.MultipartManifest = value; return nil })
}

func WithObjectReadSymlink(value string) ObjectReadOption {
	return request.Custom(func(cfg *ObjectReadOpts) error { cfg.Symlink = value; return nil })
}

func WithObjectReadVersionID(value string) ObjectReadOption {
	return request.Custom(func(cfg *ObjectReadOpts) error { cfg.VersionID = value; return nil })
}

func WithObjectReadBufferSize(value int) ObjectReadOption {
	return request.Custom(func(cfg *ObjectReadOpts) error { cfg.BufferSize = value; return nil })
}

func cloneObjectReadOpts(value ObjectReadOpts) ObjectReadOpts {
	value.Headers = cloneMetadataHeaders(value.Headers)
	if value.Newest != nil {
		owned := *value.Newest
		value.Newest = &owned
	}
	if value.IfModifiedSince != nil {
		owned := *value.IfModifiedSince
		value.IfModifiedSince = &owned
	}
	if value.IfUnmodifiedSince != nil {
		owned := *value.IfUnmodifiedSince
		value.IfUnmodifiedSince = &owned
	}
	return value
}

func applyObjectReadOptions(options []ObjectReadOption) (ObjectReadOpts, error) {
	cfg := cloneObjectReadOpts(ObjectReadOpts{})
	for _, option := range calls.Translate(options, objectReadCalls) {
		if option == nil {
			return cfg, metadataInvalid("nil object read option")
		}
		callback := cloneObjectReadOpts(cfg)
		err := option(&callback)
		cfg = cloneObjectReadOpts(callback)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
