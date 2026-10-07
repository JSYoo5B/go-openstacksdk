package v1

import (
	"context"

	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/accounts"
	"github.com/JSYoo5B/gophercloudsdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/gophercloudsdk/request"
)

// GetTempURLKey reads fresh metadata, preferring the container's secondary then
// primary key before the account's secondary then primary key. Any read failure
// stops discovery. It does not sign a URL or change native CreateTempURL policy.
func (s *Service) GetTempURLKey(ctx context.Context, options ...GetTempURLKeyOption) (*TempURLKeyResult, error) {
	p, err := s.captureTempURLKey(ctx)
	if err != nil {
		return nil, request.Wrap("GetTempURLKey", "objectstorage", err)
	}
	cfg, err := applyGetTempURLKeyOptions(options)
	if err = p.finish(ctx, cfg, err); err != nil {
		return nil, request.Wrap("GetTempURLKey", "objectstorage", err)
	}
	var result *TempURLKeyResult
	if cfg.Container != "" {
		if err = p.check(ctx); err != nil {
			return nil, request.Wrap("GetTempURLKey", "objectstorage", err)
		}
		response, readErr := containers.New(p.client).GetMetadata(ctx, cfg.Container,
			containers.WithGetMetadataOpts(containers.GetMetadataOpts{Headers: cfg.Headers, Newest: cfg.Newest}))
		var proof *rest.Response
		if response != nil {
			result = &TempURLKeyResult{Container: response}
			proof = &rest.Response{Body: response.Body, Header: response.Header, StatusCode: response.StatusCode}
		}
		err, changed := p.observe(ctx, proof, readErr)
		if changed && response != nil {
			response.Metadata = nil
		}
		if err != nil {
			return result, request.Wrap("GetTempURLKey", "objectstorage", err)
		}
		if key, secondary := selectTempURLKey(response.Metadata.Values); key != "" {
			result.Key, result.FromContainer, result.Secondary = []byte(key), true, secondary
			return result, nil
		}
	}
	if err = p.check(ctx); err != nil {
		return result, request.Wrap("GetTempURLKey", "objectstorage", err)
	}
	response, readErr := accounts.New(p.client).GetMetadata(ctx,
		accounts.WithGetMetadataOpts(accounts.GetMetadataOpts{Headers: cfg.Headers, Newest: cfg.Newest}))
	var proof *rest.Response
	if response != nil {
		if result == nil {
			result = &TempURLKeyResult{}
		}
		result.Account = response
		proof = &rest.Response{Body: response.Body, Header: response.Header, StatusCode: response.StatusCode}
	}
	err, changed := p.observe(ctx, proof, readErr)
	if changed && response != nil {
		response.Metadata = nil
	}
	if err != nil {
		return result, request.Wrap("GetTempURLKey", "objectstorage", err)
	}
	if key, secondary := selectTempURLKey(response.Metadata.Values); key != "" {
		result.Key, result.Secondary = []byte(key), secondary
	}
	return result, nil
}

func selectTempURLKey(values map[string]string) (string, bool) {
	if key := values["temp-url-key-2"]; key != "" {
		return key, true
	}
	return values["temp-url-key"], false
}
