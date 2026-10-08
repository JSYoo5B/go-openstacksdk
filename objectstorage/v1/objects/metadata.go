package objects

import (
	"context"
	"net/http"
	"strconv"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GetMetadata reads the object metadata at the current route. Swift may follow
// a symlink; mutations instead read the link itself and reject observed links.
func (a *API) GetMetadata(ctx context.Context, container, object string, options ...GetMetadataOption) (*GetMetadataResult, error) {
	p, err := a.captureMetadata(ctx, container, object)
	if err != nil {
		return nil, request.Wrap("GetMetadata", "objects", err)
	}
	cfg, err := applyGetMetadataOptions(options)
	if err = p.finish(ctx, cfg.Headers, err); err != nil {
		return nil, request.Wrap("GetMetadata", "objects", err)
	}
	if cfg.Newest != nil {
		p.client.MoreHeaders["X-Newest"] = strconv.FormatBool(*cfg.Newest)
	}
	result, _, err := p.read(ctx, false)
	return result, request.Wrap("GetMetadata", "objects", err)
}

// SetMetadata reads fresh custom metadata, merges these literal values, then
// posts the complete map. Empty values are stored. Concurrent writes may be lost.
// Use GetMetadata explicitly when the resulting state is needed.
func (a *API) SetMetadata(ctx context.Context, container, object string, metadata map[string]string, options ...MetadataOption) (*MetadataChangeResult, error) {
	return a.changeMetadata(ctx, container, object, "SetMetadata", metadataInput(metadata), false, options)
}

// DeleteMetadata reads fresh custom metadata and posts it without these keys.
// Even empty or absent keys perform the read and write; this is not atomic.
func (a *API) DeleteMetadata(ctx context.Context, container, object string, keys []string, options ...MetadataOption) (*MetadataChangeResult, error) {
	return a.changeMetadata(ctx, container, object, "DeleteMetadata", metadataKeys(append([]string(nil), keys...)), true, options)
}

func (a *API) changeMetadata(ctx context.Context, container, object, method string, input func() (map[string]string, error), remove bool, options []MetadataOption) (*MetadataChangeResult, error) {
	p, err := a.captureMetadata(ctx, container, object)
	if err != nil {
		return nil, request.Wrap(method, "objects", err)
	}
	values, err := input()
	if err != nil {
		return nil, request.Wrap(method, "objects", metadataContextError(ctx, err))
	}
	cfg, err := applyMetadataOptions(options)
	if err = p.finish(ctx, nil, err); err != nil {
		return nil, request.Wrap(method, "objects", err)
	}
	writeHeaders, err := validateMetadataHeaders(cfg.Headers)
	if err != nil {
		return nil, request.Wrap(method, "objects", metadataContextError(ctx, err))
	}
	before, response, err := p.read(ctx, true)
	if before == nil {
		return nil, request.Wrap(method, "objects", err)
	}
	result := &MetadataChangeResult{Before: before}
	if err != nil {
		return result, request.Wrap(method, "objects", err)
	}
	merged := cloneMetadataHeaders(before.Metadata.Values)
	for key, value := range values {
		if remove {
			delete(merged, key)
		} else {
			merged[key] = value
		}
	}
	// ServiceClient.MoreHeaders wins over request headers. Own the final POST
	// overlay on a private client, preserving the original read/source snapshot.
	headers := mutableMetadataHeaders(before.Metadata)
	for key, value := range p.headers {
		headers[key] = value
	}
	for key, value := range writeHeaders {
		headers[key] = value
	}
	for key, value := range merged {
		headers[http.CanonicalHeaderKey("X-Object-Meta-"+key)] = value
	}
	if err = p.check(ctx); err != nil {
		return result, request.Wrap(method, "objects", response.Fail(err))
	}
	client := *p.client
	client.MoreHeaders = headers
	ack, err := rest.DoJSON(ctx, &client, "POST", p.target, nil, nil, 202)
	err = p.observe(ctx, ack, err)
	if ack != nil {
		result.Acknowledgement = &MetadataResponse{Body: append([]byte(nil), ack.Body...), Header: ack.Header.Clone(), StatusCode: ack.StatusCode}
	}
	return result, request.Wrap(method, "objects", err)
}

func (p *preparedMetadata) read(ctx context.Context, mutation bool) (*GetMetadataResult, *rest.Response, error) {
	target := p.target
	if mutation {
		target += "?symlink=get"
	}
	response, err := rest.DoJSON(ctx, p.client, "HEAD", target, nil, nil, 200)
	err = p.observe(ctx, response, err)
	if response == nil {
		return nil, nil, err
	}
	result := &GetMetadataResult{Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	if err == nil {
		if mutation {
			err = rejectMetadataSymlink(response.Header)
		}
		var metadata *MetadataInfo
		if err == nil {
			metadata, err = projectMetadata(response.Header)
		}
		err = joinMetadataErrors(err, p.check(ctx))
		if err == nil {
			result.Metadata = metadata
		} else {
			err = response.Fail(err)
		}
	}
	return result, response, err
}
