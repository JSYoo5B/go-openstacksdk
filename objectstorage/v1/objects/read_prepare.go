package objects

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type preparedObjectRead struct {
	metadata *preparedMetadata
	target   string
	buffer   int
}

func (a *API) captureObjectRead(ctx context.Context, container, object string) (*preparedObjectRead, error) {
	metadata, err := a.captureMetadata(ctx, container, object)
	if err != nil {
		return nil, err
	}
	if _, err := validateObjectReadHeaders(metadata.source.MoreHeaders); err != nil {
		return nil, metadataContextError(ctx, err)
	}
	return &preparedObjectRead{metadata: metadata, target: metadata.target}, nil
}

func (a *API) prepareObjectRead(ctx context.Context, container, object string, options []ObjectReadOption) (*preparedObjectRead, error) {
	p, err := a.captureObjectRead(ctx, container, object)
	if err != nil {
		return nil, err
	}
	if err := p.finish(ctx, options); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *preparedObjectRead) finish(ctx context.Context, options []ObjectReadOption) error {
	cfg, optionErr := applyObjectReadOptions(options)
	if err := joinMetadataErrors(optionErr, p.check(ctx)); err != nil {
		return metadataContextError(ctx, err)
	}
	headers, query, err := objectReadInputs(cfg)
	if err != nil {
		return metadataContextError(ctx, err)
	}
	for key, value := range headers {
		p.metadata.client.MoreHeaders[key] = value
	}
	if len(query) != 0 {
		p.target += "?" + query.Encode()
	}
	p.buffer = cfg.BufferSize
	if p.buffer == 0 {
		p.buffer = 32 * 1024
	}
	return nil
}

func (p *preparedObjectRead) check(ctx context.Context) error {
	baseErr := p.metadata.check(ctx)
	_, headerErr := validateObjectReadHeaders(p.metadata.source.MoreHeaders)
	return metadataContextError(ctx, joinMetadataErrors(baseErr, headerErr))
}

func validateObjectReadHeaders(headers map[string]string) (map[string]string, error) {
	owned, err := validateMetadataHeaders(headers)
	if err != nil {
		return nil, err
	}
	for key := range owned {
		switch strings.ToLower(key) {
		case "range", "if-range", "if-match", "if-none-match", "if-modified-since", "if-unmodified-since", "x-service-token", "cookie":
			return nil, metadataInvalid("object read header %q is SDK owned", key)
		}
	}
	return owned, nil
}

func objectReadInputs(cfg ObjectReadOpts) (map[string]string, url.Values, error) {
	if cfg.BufferSize < 0 || cfg.BufferSize > 16*1024*1024 {
		return nil, nil, metadataInvalid("object read buffer must be zero or between 1 and 16 MiB")
	}
	headers, err := validateObjectReadHeaders(cfg.Headers)
	if err != nil {
		return nil, nil, err
	}
	for key, value := range map[string]string{"If-Match": cfg.IfMatch, "If-None-Match": cfg.IfNoneMatch, "Range": cfg.Range} {
		if !metadataFieldValue(value) {
			return nil, nil, metadataInvalid("invalid object read field %q", key)
		}
		if value != "" {
			headers[key] = value
		}
	}
	for key, value := range map[string]*bool{"X-Newest": cfg.Newest} {
		if value != nil {
			headers[key] = strconv.FormatBool(*value)
		}
	}
	for key, value := range map[string]*time.Time{"If-Modified-Since": cfg.IfModifiedSince, "If-Unmodified-Since": cfg.IfUnmodifiedSince} {
		if value != nil {
			utc := value.UTC()
			if utc.Year() < 1 || utc.Year() > 9999 {
				return nil, nil, metadataInvalid("object read date %q is outside years 1 through 9999", key)
			}
			headers[key] = utc.Format(http.TimeFormat)
		}
	}
	query := make(url.Values)
	for key, value := range map[string]string{"filename": cfg.Filename, "multipart-manifest": cfg.MultipartManifest, "symlink": cfg.Symlink, "version-id": cfg.VersionID} {
		if !utf8.ValidString(value) {
			return nil, nil, metadataInvalid("invalid object read query %q", key)
		}
		for _, b := range []byte(value) {
			if b < 32 || b == 127 {
				return nil, nil, metadataInvalid("invalid control in object read query %q", key)
			}
		}
		if value != "" {
			query.Set(key, value)
		}
	}
	return headers, query, nil
}
