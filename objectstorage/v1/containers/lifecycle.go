package containers

import (
	"context"
	"net/http"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// CreateContainer creates or updates a literal container with one bodyless PUT.
// It preserves the actual 201 or 202 acknowledgement without fetching state.
func (a *API) CreateContainer(ctx context.Context, container string, options ...CreateContainerOption) (*ContainerResponse, error) {
	p, err := a.captureMetadata(ctx, container)
	if err != nil {
		return nil, request.Wrap("CreateContainer", "containers", err)
	}
	cfg, err := applyCreateContainerOptions(options)
	var headers map[string]string
	if err == nil {
		headers, err = metadataInput(cfg.Metadata)()
	}
	if err = p.finish(ctx, cfg.Headers, err); err != nil {
		return nil, request.Wrap("CreateContainer", "containers", err)
	}
	for key, value := range headers {
		p.client.MoreHeaders[key] = value
	}
	response, err := rest.DoJSONGuarded(ctx, p.client, p.check, http.MethodPut, p.target, nil, nil, http.StatusCreated, http.StatusAccepted)
	err = p.observe(ctx, response, err)
	if response == nil {
		return nil, request.Wrap("CreateContainer", "containers", err)
	}
	result := &ContainerResponse{Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}
	return result, request.Wrap("CreateContainer", "containers", err)
}

// DeleteContainer deletes a literal container. Missing containers are quiet by
// default, with their actual 404 response returned and IgnoredMissing set only
// after response handling succeeds. Nonempty containers remain server errors.
func (a *API) DeleteContainer(ctx context.Context, container string, options ...DeleteContainerOption) (*ContainerResponse, error) {
	p, err := a.captureMetadata(ctx, container)
	if err != nil {
		return nil, request.Wrap("DeleteContainer", "containers", err)
	}
	cfg, err := applyDeleteContainerOptions(options)
	if err = p.finish(ctx, cfg.Headers, err); err != nil {
		return nil, request.Wrap("DeleteContainer", "containers", err)
	}
	codes := []int{http.StatusNoContent}
	if cfg.IgnoreMissing == nil || *cfg.IgnoreMissing {
		codes = append(codes, http.StatusNotFound)
	}
	response, err := rest.DoJSONGuarded(ctx, p.client, p.check, http.MethodDelete, p.target, nil, nil, codes...)
	err = p.observe(ctx, response, err)
	if response == nil {
		return nil, request.Wrap("DeleteContainer", "containers", err)
	}
	result := &ContainerResponse{Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode, IgnoredMissing: err == nil && response.StatusCode == http.StatusNotFound}
	return result, request.Wrap("DeleteContainer", "containers", err)
}
