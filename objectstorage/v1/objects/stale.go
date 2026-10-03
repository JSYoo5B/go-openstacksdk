package objects

import (
	"context"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"gophercloudsdk/request"
)

// IsObjectStale compares only the SDK checksum metadata. Filename is explicit
// and is read only when an existing object needs local comparison digests.
func (a *API) IsObjectStale(ctx context.Context, container, object, filename string, options ...IsObjectStaleOption) (*ObjectStaleResult, error) {
	p, err := a.captureCreateObject(ctx, container, object)
	if err != nil {
		return nil, request.Wrap("IsObjectStale", "objects", err)
	}
	if err = validateCreateObjectFilename(filename); err != nil {
		return nil, request.Wrap("IsObjectStale", "objects", metadataContextError(ctx, err))
	}
	cfg, err := p.applyStaleOptions(ctx, options)
	if err == nil {
		cfg.Headers, err = validateCreateObjectHeaders(cfg.Headers)
	}
	if err == nil {
		err = validateObjectCreateDigest(cfg.MD5, 32)
	}
	if err == nil {
		err = validateObjectCreateDigest(cfg.SHA256, 64)
	}
	if err = joinMetadataErrors(err, p.guard(ctx)); err != nil {
		return nil, request.Wrap("IsObjectStale", "objects", err)
	}
	result, err := p.stale(ctx, cfg.Headers, cfg.MD5, cfg.SHA256, func() (string, string, error) { return p.hashFilename(ctx, filename) })
	return result, request.Wrap("IsObjectStale", "objects", err)
}

func (p *preparedCreateObject) stale(ctx context.Context, headers map[string]string, md, sh string, hash func() (string, string, error)) (*ObjectStaleResult, error) {
	out := p.exchange(ctx, http.MethodHead, p.metadata.target, "discovery", nil, headers, 1, http.StatusOK, http.StatusNoContent, http.StatusNotFound)
	response := out.phase.Acknowledgement
	if response == nil {
		return nil, out.err
	}
	result := &ObjectStaleResult{Discovery: cloneObjectCreateResponse(response), MD5: md, SHA256: sh}
	if out.err != nil {
		return result, out.err
	}
	if response.StatusCode == http.StatusNotFound {
		if err := p.guard(ctx); err != nil {
			return result, objectCreateResponseError(response, err)
		}
		value := true
		result.Stale = &value
		return result, nil
	}
	remoteMD, remoteSH, err := objectStaleDigests(response.Header)
	if err == nil && md == "" && sh == "" {
		md, sh, err = hash()
		result.MD5, result.SHA256 = md, sh
	}
	if err = joinMetadataErrors(err, p.guard(ctx)); err != nil {
		return result, objectCreateResponseError(response, err)
	}
	stale, matched := false, false
	for _, digest := range []struct {
		local  string
		remote *string
	}{{md, remoteMD}, {sh, remoteSH}} {
		if digest.local == "" {
			continue
		}
		if digest.remote == nil || *digest.remote != digest.local {
			stale = true
		} else {
			matched = true
		}
	}
	stale = stale || !matched
	result.RemoteMD5, result.RemoteSHA256, result.Stale = remoteMD, remoteSH, &stale
	return result, nil
}

func objectCreateHeaderValue(headers http.Header, name string) (*string, error) {
	var found bool
	var value string
	for key, values := range headers {
		if !strings.EqualFold(key, name) {
			continue
		}
		if found || len(values) != 1 || !utf8.ValidString(values[0]) {
			return nil, metadataInvalid("object response header %q must have one valid value", name)
		}
		found, value = true, values[0]
		for _, r := range value {
			if unicode.IsControl(r) {
				return nil, metadataInvalid("object response header %q contains a control", name)
			}
		}
	}
	if !found {
		return nil, nil
	}
	return &value, nil
}
func objectStaleDigests(headers http.Header) (*string, *string, error) {
	values := make([]*string, 4)
	for index, name := range []string{"X-Object-Meta-X-Sdk-Md5", "X-Object-Meta-X-Sdk-Sha256", "X-Object-Meta-X-Shade-Md5", "X-Object-Meta-X-Shade-Sha256"} {
		value, err := objectCreateHeaderValue(headers, name)
		if err != nil {
			return nil, nil, err
		}
		values[index] = value
	}
	if values[0] == nil {
		values[0] = values[2]
	}
	if values[1] == nil {
		values[1] = values[3]
	}
	return values[0], values[1], nil
}
