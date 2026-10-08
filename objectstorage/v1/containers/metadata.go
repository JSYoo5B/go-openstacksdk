package containers

import (
	"context"
	"strconv"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GetMetadata reads this container's metadata and preserves the actual response.
func (a *API) GetMetadata(ctx context.Context, container string, options ...GetMetadataOption) (*GetMetadataResult, error) {
	p, err := a.captureMetadata(ctx, container)
	if err != nil {
		return nil, request.Wrap("GetMetadata", "containers", err)
	}
	cfg, err := applyGetMetadataOptions(options)
	if err = p.finish(ctx, cfg.Headers, err); err != nil {
		return nil, request.Wrap("GetMetadata", "containers", err)
	}
	if cfg.Newest != nil {
		p.client.MoreHeaders["X-Newest"] = strconv.FormatBool(*cfg.Newest)
	}
	response, err := rest.DoJSON(ctx, p.client, "HEAD", p.target, nil, nil, 204)
	err = p.observe(ctx, response, err)
	if response == nil {
		return nil, request.Wrap("GetMetadata", "containers", err)
	}
	result := &GetMetadataResult{Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if err == nil {
		result.Metadata, err = projectMetadata(response.Header)
		if err != nil {
			err = response.Fail(err)
		}
		if checkErr := p.check(ctx); checkErr != nil {
			result.Metadata = nil
			err = response.Fail(joinMetadataErrors(err, checkErr))
		}
	}
	return result, request.Wrap("GetMetadata", "containers", err)
}

// SetMetadata submits only the given keys. Empty values delete metadata.
// Use GetMetadata explicitly when the updated container state is needed.
func (a *API) SetMetadata(ctx context.Context, container string, metadata map[string]string, options ...MetadataOption) (*MetadataResponse, error) {
	return a.changeMetadata(ctx, container, "SetMetadata", metadataInput(metadata), options)
}

// DeleteMetadata removes the selected metadata keys with one bodyless POST.
func (a *API) DeleteMetadata(ctx context.Context, container string, keys []string, options ...MetadataOption) (*MetadataResponse, error) {
	return a.changeMetadata(ctx, container, "DeleteMetadata", metadataKeys(append([]string(nil), keys...)), options)
}

func (a *API) changeMetadata(ctx context.Context, container, method string, input func() (map[string]string, error), options []MetadataOption) (*MetadataResponse, error) {
	p, err := a.captureMetadata(ctx, container)
	if err != nil {
		return nil, request.Wrap(method, "containers", err)
	}
	headers, err := input()
	if err != nil {
		return nil, request.Wrap(method, "containers", metadataContextError(ctx, err))
	}
	cfg, err := applyMetadataOptions(options)
	if err = p.finish(ctx, cfg.Headers, err); err != nil {
		return nil, request.Wrap(method, "containers", err)
	}
	for key, value := range headers {
		p.client.MoreHeaders[key] = value
	}
	response, err := rest.DoJSON(ctx, p.client, "POST", p.target, nil, nil, 204)
	err = p.observe(ctx, response, err)
	if response == nil {
		return nil, request.Wrap(method, "containers", err)
	}
	result := &MetadataResponse{Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	return result, request.Wrap(method, "containers", err)
}
