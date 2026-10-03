package accounts

import (
	"context"

	"gophercloudsdk/internal/rest"
	"gophercloudsdk/request"
)

// SetTempURLKey acknowledges one metadata POST. Empty keys remove the selected
// key; use GetMetadata explicitly when refreshed account state is needed.
func (a *API) SetTempURLKey(ctx context.Context, key string, options ...SetTempURLKeyOption) (*MetadataResponse, error) {
	p, err := a.captureMetadata(ctx)
	if err != nil {
		return nil, request.Wrap("SetTempURLKey", "accounts", err)
	}
	if !metadataFieldValue(key) {
		return nil, request.Wrap("SetTempURLKey", "accounts", metadataContextError(ctx, metadataInvalid("invalid Temp URL key field value")))
	}
	cfg, err := applySetTempURLKeyOptions(options)
	if err = p.finish(ctx, cfg.Headers, err); err != nil {
		return nil, request.Wrap("SetTempURLKey", "accounts", err)
	}
	suffix := "Temp-URL-Key"
	if cfg.Secondary {
		suffix += "-2"
	}
	headers, err := metadataInput(map[string]string{suffix: key})()
	if err != nil {
		return nil, request.Wrap("SetTempURLKey", "accounts", metadataContextError(ctx, err))
	}
	for name, value := range headers {
		p.client.MoreHeaders[name] = value
	}
	response, err := rest.DoJSON(ctx, p.client, "POST", p.endpoint, nil, nil, 204)
	err = p.observe(ctx, response, err)
	if response == nil {
		return nil, request.Wrap("SetTempURLKey", "accounts", err)
	}
	result := &MetadataResponse{Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	return result, request.Wrap("SetTempURLKey", "accounts", err)
}
