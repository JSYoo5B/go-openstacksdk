package objects

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

type preparedCreateObject struct {
	metadata    *preparedMetadata
	imageImport bool
	// Optional immutable policy is used by scoped HEAD waiters. Existing
	// create/stale operations retain their original header policy.
	headerPolicy func(map[string]string) error
	mu           sync.Mutex
	observed     error
}

func (a *API) captureCreateObject(ctx context.Context, container, object string) (*preparedCreateObject, error) {
	base, err := a.captureMetadata(ctx, container, object)
	if err != nil {
		return nil, err
	}
	if _, err := validateCreateObjectHeaders(base.source.MoreHeaders); err != nil {
		return nil, metadataContextError(ctx, err)
	}
	p := &preparedCreateObject{metadata: base}
	if profile, _ := ctx.Value(imageImportObjectProfileKey{}).(bool); profile {
		p.imageImport = true
		base.extraSource = validateImageImportObjectSourceHeaders
	}
	if err := p.guard(ctx); err != nil {
		return nil, err
	}
	return p, nil
}
func validateCreateObjectHeaders(headers map[string]string) (map[string]string, error) {
	owned, err := validateMetadataHeaders(headers)
	if err != nil {
		return nil, err
	}
	for key := range owned {
		name := strings.ToLower(key)
		if strings.HasPrefix(name, "x-copy-from") || strings.HasPrefix(name, "x-backend-") {
			return nil, metadataInvalid("object create header %q is SDK owned", key)
		}
		switch name {
		case "cookie", "x-service-token", "etag", "x-object-manifest", "x-static-large-object":
			return nil, metadataInvalid("object create header %q is SDK owned", key)
		}
	}
	return owned, nil
}
func (p *preparedCreateObject) check(ctx context.Context) error {
	_, headerErr := validateCreateObjectHeaders(p.metadata.source.MoreHeaders)
	if p.headerPolicy != nil {
		headerErr = joinMetadataErrors(headerErr, p.headerPolicy(p.metadata.source.MoreHeaders))
	}
	return metadataContextError(ctx, joinMetadataErrors(p.metadata.check(ctx), headerErr))
}
func (p *preparedCreateObject) guard(ctx context.Context) error {
	p.mu.Lock()
	observed := p.observed
	p.mu.Unlock()
	return joinMetadataErrors(observed, p.check(ctx))
}
func (p *preparedCreateObject) note(ctx context.Context) error {
	err := p.check(ctx)
	if err != nil {
		p.mu.Lock()
		p.observed = joinMetadataErrors(p.observed, err)
		p.mu.Unlock()
	}
	return err
}
func (p *preparedCreateObject) target(name string) (string, error) {
	if err := validateMetadataObject(name); err != nil {
		return "", err
	}
	target := p.metadata.client.ServiceURL(url.PathEscape(p.metadata.container), url.PathEscape(name))
	return target, rest.ValidateTarget(p.metadata.client, target)
}
func validateCreateObjectFilename(filename string) error {
	if filename == "" || !utf8.ValidString(filename) || strings.ContainsRune(filename, 0) {
		return metadataInvalid("an explicit valid object filename is required")
	}
	return nil
}
func snapshotCreateObjectInput(input CreateObjectInput) (CreateObjectInput, string, error) {
	count, kind := 0, ""
	if input.Data != nil {
		count++
		kind = "bytes"
		owned := make([]byte, len(input.Data))
		copy(owned, input.Data)
		input.Data = owned
	}
	if input.Filename != "" {
		count++
		kind = "file"
		if err := validateCreateObjectFilename(input.Filename); err != nil {
			return input, "", err
		}
	}
	if input.Reader != nil {
		value := reflect.ValueOf(input.Reader)
		switch value.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			if value.IsNil() {
				return input, "", metadataInvalid("nil object reader")
			}
		}
		count++
		kind = "reader"
	}
	if count != 1 {
		return input, "", metadataInvalid("exactly one object Data, Filename or Reader is required")
	}
	return input, kind, nil
}
func validateObjectCreateDigest(value string, length int) error {
	if value == "" {
		return nil
	}
	if len(value) != length {
		return metadataInvalid("object digest must have %d hexadecimal characters", length)
	}
	for _, b := range []byte(value) {
		if !(b >= '0' && b <= '9' || b >= 'a' && b <= 'f' || b >= 'A' && b <= 'F') {
			return metadataInvalid("invalid object digest")
		}
	}
	return nil
}
func (p *preparedCreateObject) finishCreate(ctx context.Context, cfg *CreateObjectOpts, kind string, err error) error {
	err = joinMetadataErrors(err, p.guard(ctx))
	if err != nil {
		return metadataContextError(ctx, err)
	}
	cfg.Headers, err = validateCreateObjectHeaders(cfg.Headers)
	if err == nil {
		cfg.Metadata, err = metadataInput(cfg.Metadata)()
	}
	if err == nil && p.imageImport {
		if !metadataFieldValue(cfg.MD5) || !metadataFieldValue(cfg.SHA256) {
			err = metadataInvalid("invalid image import object hash header value")
		}
	} else if err == nil {
		err = joinMetadataErrors(validateObjectCreateDigest(cfg.MD5, 32), validateObjectCreateDigest(cfg.SHA256, 64))
	}
	if err == nil && cfg.SegmentSize != nil && *cfg.SegmentSize < 0 {
		err = metadataInvalid("negative object segment size")
	}
	if err == nil && kind == "bytes" && cfg.GenerateChecksums != nil && *cfg.GenerateChecksums {
		err = metadataInvalid("checksums cannot be generated for byte Data")
	}
	return metadataContextError(ctx, joinMetadataErrors(err, p.guard(ctx)))
}
func objectCreateUploadHeaders(cfg CreateObjectOpts) map[string]string {
	headers := cloneMetadataHeaders(cfg.Headers)
	if headers == nil {
		headers = make(map[string]string)
	}
	for key, value := range cfg.Metadata {
		headers[http.CanonicalHeaderKey("X-Object-Meta-"+key)] = value
	}
	return headers
}
