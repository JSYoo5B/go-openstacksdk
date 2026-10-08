package v1

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// Only phase orchestration is new: leaf APIs retain their strict response policy.
func discoverSigningKey(ctx context.Context, p *preparedTempURLKey, cfg GetTempURLKeyOpts) (*TempURLKeyResult, error) {
	var result *TempURLKeyResult
	if cfg.Container != "" {
		if err := p.check(ctx); err != nil {
			return nil, err
		}
		response, err := containers.New(p.client).GetMetadata(ctx, cfg.Container,
			containers.WithGetMetadataOpts(containers.GetMetadataOpts{Headers: cfg.Headers, Newest: cfg.Newest}))
		var proof *rest.Response
		if response != nil {
			result = &TempURLKeyResult{Container: response}
			proof = &rest.Response{Body: response.Body, Header: response.Header, StatusCode: response.StatusCode}
		}
		err, changed := p.observe(ctx, proof, err)
		if changed && response != nil {
			response.Metadata = nil
		}
		if err != nil {
			return result, clearSigningSelection(result, err)
		}
		if key, secondary := selectTempURLKey(response.Metadata.Values); key != "" {
			result.Key, result.FromContainer, result.Secondary = []byte(key), true, secondary
			return result, nil
		}
	}
	if err := observeSigningDiscovery(ctx, p, result, nil); err != nil {
		return result, err
	}
	response, err := accounts.New(p.client).GetMetadata(ctx,
		accounts.WithGetMetadataOpts(accounts.GetMetadataOpts{Headers: cfg.Headers, Newest: cfg.Newest}))
	var proof *rest.Response
	if response != nil {
		if result == nil {
			result = &TempURLKeyResult{}
		}
		result.Account = response
		proof = &rest.Response{Body: response.Body, Header: response.Header, StatusCode: response.StatusCode}
	}
	err, changed := p.observe(ctx, proof, err)
	if changed && response != nil {
		response.Metadata = nil
	}
	if err != nil {
		return result, clearSigningSelection(result, err)
	}
	if key, secondary := selectTempURLKey(response.Metadata.Values); key != "" {
		result.Key, result.Secondary = []byte(key), secondary
		return result, nil
	}
	return result, fmt.Errorf("%w: no usable Temp URL key", resource.ErrNotFound)
}
func clearSigningSelection(result *TempURLKeyResult, err error) error {
	if result != nil {
		result.Key, result.FromContainer, result.Secondary = nil, false, false
	}
	return err
}
func observeSigningDiscovery(ctx context.Context, p *preparedTempURLKey, result *TempURLKeyResult, err error) error {
	var proof *rest.Response
	if result != nil {
		if response := result.Account; response != nil {
			proof = &rest.Response{Body: response.Body, Header: response.Header, StatusCode: response.StatusCode}
		} else if response := result.Container; response != nil {
			proof = &rest.Response{Body: response.Body, Header: response.Header, StatusCode: response.StatusCode}
		}
	}
	err, changed := p.observe(ctx, proof, err)
	if changed && result != nil {
		if result.Account != nil {
			result.Account.Metadata = nil
		} else if result.Container != nil {
			result.Container.Metadata = nil
		}
	}
	if err != nil {
		return clearSigningSelection(result, err)
	}
	return nil
}
