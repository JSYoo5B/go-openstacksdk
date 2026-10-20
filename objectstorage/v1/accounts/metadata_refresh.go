package accounts

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

// SetMetadataAndRefresh implements the Python account setter's default POST
// followed by HEAD. JSON-domain values use Python string spellings; keys may be
// suffixes or the exact X-Account-Meta- prefix. Existing SetMetadata is one POST.
func (a *API) SetMetadataAndRefresh(ctx context.Context, values map[string]any, options ...MetadataOption) (*MetadataRefreshResult, error) {
	p, err := a.captureMetadata(ctx)
	if err != nil {
		return nil, request.Wrap("SetMetadataAndRefresh", "accounts", err)
	}
	headers, conversionErr := pythonMetadataHeaders(values)
	if err = joinMetadataErrors(conversionErr, p.check(ctx)); err != nil {
		return nil, request.Wrap("SetMetadataAndRefresh", "accounts", metadataContextError(ctx, err))
	}
	cfg, err := applyMetadataOptions(append([]MetadataOption(nil), options...))
	if err = p.finish(ctx, cfg.Headers, err); err != nil {
		return nil, request.Wrap("SetMetadataAndRefresh", "accounts", err)
	}
	readClient := *p.client
	readClient.MoreHeaders = cloneMetadataHeaders(p.client.MoreHeaders)
	for key, value := range headers {
		p.client.MoreHeaders[key] = value
	}
	result := &MetadataRefreshResult{}
	ack, err := pythonMetadataExchange(ctx, p, p.client, "POST", p.endpoint)
	if ack != nil {
		result.Acknowledgement = &MetadataResponse{Body: append([]byte(nil), ack.Body...), Header: ack.Header.Clone(), StatusCode: ack.StatusCode}
	}
	if err != nil {
		return result, request.Wrap("SetMetadataAndRefresh", "accounts", err)
	}
	response, err := pythonMetadataExchange(ctx, p, &readClient, "HEAD", p.endpoint)
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
	return result, request.Wrap("SetMetadataAndRefresh", "accounts", err)
}

// DeleteMetadataKeys follows Python's bodyless POST, including an empty keys
// iterable. Exact prefixed names are supported; duplicates collapse to one key.
func (a *API) DeleteMetadataKeys(ctx context.Context, keys []string, options ...MetadataOption) (*MetadataResponse, error) {
	values := make(map[string]any, len(keys))
	for _, key := range append([]string(nil), keys...) {
		values[key] = ""
	}
	p, err := a.captureMetadata(ctx)
	if err != nil {
		return nil, request.Wrap("DeleteMetadataKeys", "accounts", err)
	}
	headers, conversionErr := pythonMetadataHeaders(values)
	if err = joinMetadataErrors(conversionErr, p.check(ctx)); err != nil {
		return nil, request.Wrap("DeleteMetadataKeys", "accounts", metadataContextError(ctx, err))
	}
	cfg, err := applyMetadataOptions(append([]MetadataOption(nil), options...))
	if err = p.finish(ctx, cfg.Headers, err); err != nil {
		return nil, request.Wrap("DeleteMetadataKeys", "accounts", err)
	}
	for key, value := range headers {
		p.client.MoreHeaders[key] = value
	}
	response, err := pythonMetadataExchange(ctx, p, p.client, "POST", p.endpoint)
	if response == nil {
		return nil, request.Wrap("DeleteMetadataKeys", "accounts", err)
	}
	result := &MetadataResponse{Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	return result, request.Wrap("DeleteMetadataKeys", "accounts", err)
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
		if !strings.HasPrefix(name, "X-Account-Meta-") {
			name = "X-Account-Meta-" + name
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
