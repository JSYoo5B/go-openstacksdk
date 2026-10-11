package containers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/gophercloud/gophercloud/v2"
)

// MetadataRefreshResult separates the actual POST acknowledgement from the
// subsequent HEAD. A failed refresh does not undo or replay an acknowledged POST.
type MetadataRefreshResult struct {
	Acknowledgement *MetadataResponse
	Refreshed       *GetMetadataResult
}

// SetMetadataAndRefresh implements the Python container setter's default POST
// followed by HEAD. JSON-domain values use Python string spellings; keys may be
// suffixes, exact prefixes or audited system aliases. WithMetadataRefresh(false)
// skips the final HEAD; existing SetMetadata remains a one-POST acknowledgement.
func (a *API) SetMetadataAndRefresh(ctx context.Context, container string, values map[string]any, options ...MetadataRefreshOption) (*MetadataRefreshResult, error) {
	p, err := a.captureMetadata(ctx, container)
	if err != nil {
		return nil, request.Wrap("SetMetadataAndRefresh", "containers", err)
	}
	headers, conversionErr := pythonMetadataHeaders(values)
	if err = joinMetadataErrors(conversionErr, p.check(ctx)); err != nil {
		return nil, request.Wrap("SetMetadataAndRefresh", "containers", metadataContextError(ctx, err))
	}
	cfg, err := applyMetadataRefreshOptions(append([]MetadataRefreshOption(nil), options...))
	if err = p.finish(ctx, cfg.Headers, err); err != nil {
		return nil, request.Wrap("SetMetadataAndRefresh", "containers", err)
	}
	readClient := *p.client
	readClient.MoreHeaders = cloneMetadataHeaders(p.client.MoreHeaders)
	for key, value := range headers {
		p.client.MoreHeaders[key] = value
	}
	result := &MetadataRefreshResult{}
	ack, err := pythonMetadataExchange(ctx, p, p.client, "POST", p.target)
	if ack != nil {
		result.Acknowledgement = &MetadataResponse{Body: append([]byte(nil), ack.Body...), Header: ack.Header.Clone(), StatusCode: ack.StatusCode}
	}
	if err != nil {
		return result, request.Wrap("SetMetadataAndRefresh", "containers", err)
	}
	if cfg.Refresh != nil && !*cfg.Refresh {
		return result, nil
	}
	response, err := pythonMetadataExchange(ctx, p, &readClient, "HEAD", p.target)
	if response != nil {
		result.Refreshed = &GetMetadataResult{Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
		if err == nil {
			result.Refreshed.Metadata, err = projectMetadata(response.Header)
			if err = joinMetadataErrors(err, p.check(ctx)); err != nil {
				result.Refreshed.Metadata = nil
				err = response.Fail(err)
			}
		}
	}
	return result, request.Wrap("SetMetadataAndRefresh", "containers", err)
}

// DeleteMetadataKeys follows Python's bodyless POST, including an empty keys
// iterable. Exact prefixed names are supported; duplicates collapse to one key.
func (a *API) DeleteMetadataKeys(ctx context.Context, container string, keys []string, options ...MetadataOption) (*MetadataResponse, error) {
	values := make(map[string]any, len(keys))
	for _, key := range append([]string(nil), keys...) {
		values[key] = ""
	}
	p, err := a.captureMetadata(ctx, container)
	if err != nil {
		return nil, request.Wrap("DeleteMetadataKeys", "containers", err)
	}
	headers, conversionErr := pythonMetadataHeaders(values)
	if err = joinMetadataErrors(conversionErr, p.check(ctx)); err != nil {
		return nil, request.Wrap("DeleteMetadataKeys", "containers", metadataContextError(ctx, err))
	}
	cfg, err := applyMetadataOptions(append([]MetadataOption(nil), options...))
	if err = p.finish(ctx, cfg.Headers, err); err != nil {
		return nil, request.Wrap("DeleteMetadataKeys", "containers", err)
	}
	for key, value := range headers {
		p.client.MoreHeaders[key] = value
	}
	response, err := pythonMetadataExchange(ctx, p, p.client, "POST", p.target)
	if response == nil {
		return nil, request.Wrap("DeleteMetadataKeys", "containers", err)
	}
	result := &MetadataResponse{Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	return result, request.Wrap("DeleteMetadataKeys", "containers", err)
}

func pythonMetadataHeaders(values map[string]any) (map[string]string, error) {
	result := make(map[string]string, len(values))
	for key, value := range values {
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		text, err := cloudfilter.PythonString(raw)
		if err != nil {
			return nil, err
		}
		name := key
		if wire, present := pythonMetadataSystem[key]; present {
			name = wire
		} else {
			systemWire := false
			for _, wire := range pythonMetadataSystem {
				if key == wire {
					systemWire = true
					break
				}
			}
			if !systemWire && !strings.HasPrefix(name, "X-Container-Meta-") {
				name = "X-Container-Meta-" + name
			}
		}
		if !metadataToken(name) || !metadataFieldValue(text) {
			return nil, metadataInvalid("invalid Python metadata header %q", key)
		}
		name = http.CanonicalHeaderKey(name)
		if _, present := result[name]; present {
			return nil, metadataInvalid("ambiguous Python metadata aliases %q", key)
		}
		result[name] = text
	}
	return result, nil
}

func pythonMetadataExchange(ctx context.Context, p *preparedMetadata, client *gophercloud.ServiceClient, method, target string) (*rest.Response, error) {
	codes := make([]int, 200)
	rejected := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
		rejected[i] = 400 + i
	}
	response, err := rest.DoJSONGuardedRejections(ctx, client, p.check, method, target, nil, nil, rest.RejectionPolicy{Codes: rejected}, codes...)
	return response, p.observe(ctx, response, err)
}

// Audited Source Container._system_metadata: all eight aliases, including history_location.
var pythonMetadataSystem = map[string]string{
	"content_type":             "content-type",
	"is_content_type_detected": "x-detect-content-type",
	"versions_location":        "x-versions-location",
	"history_location":         "x-history-location",
	"read_ACL":                 "x-container-read",
	"write_ACL":                "x-container-write",
	"sync_to":                  "x-container-sync-to",
	"sync_key":                 "x-container-sync-key",
}

// MetadataRefreshOpts owns ordinary headers and the Source refresh default.
type MetadataRefreshOpts struct {
	Headers map[string]string
	Refresh *bool
}
type MetadataRefreshOption func(*MetadataRefreshOpts) error

func WithMetadataRefresh(value bool) MetadataRefreshOption {
	return func(cfg *MetadataRefreshOpts) error { owned := value; cfg.Refresh = &owned; return nil }
}
func WithMetadataRefreshOpts(value MetadataRefreshOpts) MetadataRefreshOption {
	snapshot := cloneMetadataRefreshOpts(value)
	return func(cfg *MetadataRefreshOpts) error { *cfg = cloneMetadataRefreshOpts(snapshot); return nil }
}
func WithMetadataRefreshHeader(key, value string) MetadataRefreshOption {
	return WithMetadataRefreshHeaders(map[string]string{key: value})
}
func WithMetadataRefreshHeaders(headers map[string]string) MetadataRefreshOption {
	snapshot := cloneMetadataHeaders(headers)
	return func(cfg *MetadataRefreshOpts) error { return mergeMetadataHeaders(cfg.Headers, snapshot) }
}
func cloneMetadataRefreshOpts(value MetadataRefreshOpts) MetadataRefreshOpts {
	value.Headers = cloneMetadataHeaders(value.Headers)
	if value.Refresh != nil {
		owned := *value.Refresh
		value.Refresh = &owned
	}
	return value
}
func applyMetadataRefreshOptions(options []MetadataRefreshOption) (MetadataRefreshOpts, error) {
	cfg := cloneMetadataRefreshOpts(MetadataRefreshOpts{})
	for _, option := range options {
		if option == nil {
			return cfg, metadataInvalid("nil refresh metadata option")
		}
		callback := cloneMetadataRefreshOpts(cfg)
		err := option(&callback)
		cfg = cloneMetadataRefreshOpts(callback)
		if err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}
