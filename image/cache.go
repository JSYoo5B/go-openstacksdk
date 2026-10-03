package image

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

// GetImageCache reads one complete cache snapshot without paging or discovery.
func (s *Service) GetImageCache(ctx context.Context, options ...CacheOption) (*ImageCache, error) {
	prepared, err := s.prepareCommonCache(ctx, nil, options)
	if err != nil {
		return nil, wrapImageMutationError(ctx, "GetImageCache", err)
	}
	response, err := rest.DoJSON(ctx, prepared.client, http.MethodGet, prepared.base+"cache", nil, nil, http.StatusOK)
	if err != nil {
		return nil, wrapImageMutationError(ctx, "GetImageCache", err)
	}
	var value ImageCache
	err = json.Unmarshal(response.Body, &value)
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		return nil, wrapImageMutationError(ctx, "GetImageCache", response.Fail(err))
	}
	value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
	for _, image := range value.CachedImages {
		image.Header, image.StatusCode = response.Header.Clone(), response.StatusCode
	}
	return &value, nil
}

// QueueImage requests caching of one image. The 202 acknowledges queueing;
// existence, active state and deployment capability remain server decisions.
func (s *Service) QueueImage(ctx context.Context, ref resource.Ref, options ...CacheOption) (*ImageCacheAcknowledgement, error) {
	prepared, err := s.prepareCommonCache(ctx, &ref, options)
	if err != nil {
		return nil, wrapImageMutationError(ctx, "QueueImage", err)
	}
	response, err := rest.DoJSON(ctx, prepared.client, http.MethodPut, prepared.base+"cache/"+url.PathEscape(prepared.id), nil, nil, http.StatusAccepted)
	return cacheAcknowledgement(prepared.id, nil, response, http.StatusAccepted), wrapImageMutationError(ctx, "QueueImage", err)
}

// CacheDeleteImage removes one cached or queued entry. The default ignores only
// a fully read and closed actual DELETE 404; all name lookup failures remain errors.
func (s *Service) CacheDeleteImage(ctx context.Context, ref resource.Ref, options ...CacheDeleteOption) (*ImageCacheAcknowledgement, error) {
	options = append([]CacheDeleteOption(nil), options...)
	var policy CacheDeleteOpts
	prepared, err := s.prepareCache(ctx, &ref, func() (map[string]string, error) {
		var err error
		policy, err = parseCacheDeleteOptions(options)
		return policy.Headers, err
	})
	if err != nil {
		return nil, wrapImageMutationError(ctx, "CacheDeleteImage", err)
	}
	codes := []int{http.StatusNoContent}
	if *policy.IgnoreMissing {
		codes = append(codes, http.StatusNotFound)
	}
	response, err := rest.DoJSON(ctx, prepared.client, http.MethodDelete, prepared.base+"cache/"+url.PathEscape(prepared.id), nil, nil, codes...)
	if response != nil && response.StatusCode == http.StatusNotFound {
		if err == nil && ctx.Err() != nil {
			err = response.Fail(ctx.Err())
		}
		return nil, wrapImageMutationError(ctx, "CacheDeleteImage", err)
	}
	return cacheAcknowledgement(prepared.id, nil, response, http.StatusNoContent), wrapImageMutationError(ctx, "CacheDeleteImage", err)
}

// ClearCache clears both cached and queued images by default. It omits the
// target header for CacheBoth and does not invent counts discarded by the server.
func (s *Service) ClearCache(ctx context.Context, options ...ClearCacheOption) (*ImageCacheAcknowledgement, error) {
	options = append([]ClearCacheOption(nil), options...)
	var policy ClearCacheOpts
	prepared, err := s.prepareCache(ctx, nil, func() (map[string]string, error) {
		var err error
		policy, err = parseClearCacheOptions(options)
		return policy.Headers, err
	})
	if err != nil {
		return nil, wrapImageMutationError(ctx, "ClearCache", err)
	}
	var headers map[string]string
	if policy.Target != CacheBoth {
		value := "cache"
		if policy.Target == QueueOnly {
			value = "queue"
		}
		headers = map[string]string{cacheTargetHeader: value}
	}
	response, err := rest.DoJSON(ctx, prepared.client, http.MethodDelete, prepared.base+"cache", nil, headers, http.StatusNoContent)
	return cacheAcknowledgement("", &policy.Target, response, http.StatusNoContent), wrapImageMutationError(ctx, "ClearCache", err)
}

// CachedImageNodes reads passive node reference strings once. It neither follows
// those URLs nor sends requests to other nodes.
func (s *Service) CachedImageNodes(ctx context.Context, ref resource.Ref, options ...CacheOption) (*CachedImageNodesResult, error) {
	prepared, err := s.prepareCommonCache(ctx, &ref, options)
	if err != nil {
		return nil, wrapImageMutationError(ctx, "CachedImageNodes", err)
	}
	response, err := rest.DoJSON(ctx, prepared.client, http.MethodGet, prepared.base+"cache/nodes/"+url.PathEscape(prepared.id), nil, nil, http.StatusOK)
	if err != nil {
		return nil, wrapImageMutationError(ctx, "CachedImageNodes", err)
	}
	nodes, err := cacheStringArray(response.Body)
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		return nil, wrapImageMutationError(ctx, "CachedImageNodes", response.Fail(err))
	}
	return &CachedImageNodesResult{ImageID: prepared.id, Nodes: nodes, Body: append([]byte(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}, nil
}

// CleanCache requests cleanup of invalid or stalled entries. Its actual 200 is
// an opaque acknowledgement, not a JSON count or local cache-state guarantee.
func (s *Service) CleanCache(ctx context.Context, options ...CacheOption) (*ImageCacheAcknowledgement, error) {
	prepared, err := s.prepareCommonCache(ctx, nil, options)
	if err != nil {
		return nil, wrapImageMutationError(ctx, "CleanCache", err)
	}
	response, err := rest.DoJSON(ctx, prepared.client, http.MethodPost, prepared.base+"cache/clean", nil, nil, http.StatusOK)
	return cacheAcknowledgement("", nil, response, http.StatusOK), wrapImageMutationError(ctx, "CleanCache", err)
}

// PruneCache requests pruning and reads the server's explicit integer counts.
func (s *Service) PruneCache(ctx context.Context, options ...CacheOption) (*CachePruneResult, error) {
	prepared, err := s.prepareCommonCache(ctx, nil, options)
	if err != nil {
		return nil, wrapImageMutationError(ctx, "PruneCache", err)
	}
	response, err := rest.DoJSON(ctx, prepared.client, http.MethodPost, prepared.base+"cache/prune", nil, nil, http.StatusOK)
	if err != nil {
		return nil, wrapImageMutationError(ctx, "PruneCache", err)
	}
	var value CachePruneResult
	err = json.Unmarshal(response.Body, &value)
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		return nil, wrapImageMutationError(ctx, "PruneCache", response.Fail(err))
	}
	value.Header, value.StatusCode = response.Header.Clone(), response.StatusCode
	return &value, nil
}

func (s *Service) prepareCommonCache(ctx context.Context, ref *resource.Ref, options []CacheOption) (*preparedImageMutation, error) {
	options = append([]CacheOption(nil), options...)
	return s.prepareCache(ctx, ref, func() (map[string]string, error) {
		policy, err := parseCacheOptions(options)
		return policy.Headers, err
	})
}
