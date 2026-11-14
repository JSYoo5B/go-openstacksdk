package image_test

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

type CacheEvidenceCache struct {
	State   *image.ImageCache
	Queued  *image.ImageCacheAcknowledgement
	Nodes   *image.CachedImageNodesResult
	Deleted *image.ImageCacheAcknowledgement
	Cleared *image.ImageCacheAcknowledgement
	Cleaned *image.ImageCacheAcknowledgement
	Pruned  *image.CachePruneResult
}

func inspectAndManageCache(ctx context.Context, service *image.Service, ref resource.Ref, target image.CacheTarget) (out CacheEvidenceCache, err error) {
	options := []image.CacheOption{
		image.WithCacheOpts(image.CacheOpts{
			Headers: map[string]string{"X-Request-Source": "example"},
		}),
		request.Header("X-Cache-Request", "manage"),
		image.WithCacheHeaders(map[string]string{"X-Cache-Trace": "sample"}),
	}
	out.State, err = service.GetImageCache(ctx, options...)
	if err != nil {
		return out, err
	}
	out.Queued, err = service.QueueImage(ctx, ref, options...)
	if err != nil {
		return out, err
	}
	out.Nodes, err = service.CachedImageNodes(ctx, ref, options...)
	if err != nil {
		return out, err
	}
	out.Deleted, err = service.CacheDeleteImage(ctx, ref,
		image.WithCacheDeleteOpts(image.CacheDeleteOpts{
			Headers: map[string]string{"X-Request-Source": "example"},
		}),
		request.Header("X-Cache-Request", "delete"),
		image.WithCacheDeleteHeaders(map[string]string{"X-Cache-Trace": "sample"}),
		request.IgnoreMissing(false))
	if err != nil {
		return out, err
	}
	out.Cleared, err = service.ClearCache(ctx,
		image.WithClearCacheOpts(image.ClearCacheOpts{
			Headers: map[string]string{"X-Request-Source": "example"},
		}),
		request.Header("X-Cache-Request", "clear"),
		image.WithClearCacheHeaders(map[string]string{"X-Cache-Trace": "sample"}),
		request.Custom(func(c *image.ClearCacheOpts) error { c.Target = target; return nil }))
	if err != nil {
		return out, err
	}
	out.Cleaned, err = service.CleanCache(ctx, options...)
	if err != nil {
		return out, err
	}
	out.Pruned, err = service.PruneCache(ctx, options...)
	return out, err
}

func cacheTargetsCache() [3]image.CacheTarget {
	return [3]image.CacheTarget{image.CacheBoth, image.CacheOnly, image.QueueOnly}
}
