# Cache

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

type CacheEvidence struct {
    State *image.ImageCache
    Queued *image.ImageCacheAcknowledgement
    Nodes *image.CachedImageNodesResult
    Deleted *image.ImageCacheAcknowledgement
    Cleared *image.ImageCacheAcknowledgement
    Cleaned *image.ImageCacheAcknowledgement
    Pruned *image.CachePruneResult
}

func inspectAndManageCache(ctx context.Context, service *image.Service, ref resource.Ref, target image.CacheTarget) (out CacheEvidence, err error) {
    options := []image.CacheOption{
        image.WithCacheOpts(image.CacheOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        request.Header("X-Cache-Request", "manage"),
        image.WithCacheHeaders(map[string]string{"X-Cache-Trace": "sample"}),
    }
    out.State, err = service.GetImageCache(ctx, options...)
    if err != nil { return out, err }
    out.Queued, err = service.QueueImage(ctx, ref, options...)
    if err != nil { return out, err }
    out.Nodes, err = service.CachedImageNodes(ctx, ref, options...)
    if err != nil { return out, err }
    out.Deleted, err = service.CacheDeleteImage(ctx, ref,
        image.WithCacheDeleteOpts(image.CacheDeleteOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        request.Header("X-Cache-Request", "delete"),
        image.WithCacheDeleteHeaders(map[string]string{"X-Cache-Trace": "sample"}),
        request.IgnoreMissing(false))
    if err != nil { return out, err }
    out.Cleared, err = service.ClearCache(ctx,
        image.WithClearCacheOpts(image.ClearCacheOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        request.Header("X-Cache-Request", "clear"),
        image.WithClearCacheHeaders(map[string]string{"X-Cache-Trace": "sample"}),
        request.Custom(func(c *image.ClearCacheOpts) error { c.Target = target; return nil }))
    if err != nil { return out, err }
    out.Cleaned, err = service.CleanCache(ctx, options...)
    if err != nil { return out, err }
    out.Pruned, err = service.PruneCache(ctx, options...)
    return out, err
}

func cacheTargets() [3]image.CacheTarget {
    return [3]image.CacheTarget{image.CacheBoth, image.CacheOnly, image.QueueOnly}
}
```

```python
def inspect_and_manage_cache(conn, image_id, target="both"):
    state = conn.image.get_image_cache()
    conn.image.queue_image(image_id)
    nodes = conn.image.cached_image_nodes(image_id)
    conn.image.cache_delete_image(image_id, ignore_missing=False)
    conn.image.clear_cache(target=target)
    conn.image.clean_cache()
    pruned = conn.image.prune_cache()
    return state, nodes, pruned
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L97-L155)
- [Cache/CachedImage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/cache.py#L24-L152)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#cache-manage)
- [clear controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/cached_images.py#L219-L240)
- [Cache](https://docs.openstack.org/glance/latest/admin/cache.html#controlling-image-cache-using-v2-api)
- [204 serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/cached_images.py#L349-L366)
- [xattr](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/image_cache/drivers/xattr.py)
- [sqlite](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/image_cache/drivers/sqlite.py)
- [centralized DB](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/sqlalchemy/api.py#L2306-L2318)
- [nodes controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/cached_images.py#L189-L217)
- [queue controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/cached_images.py#L252-L276)
- [Cache Contracts Test](cache_contracts_test.go)
- [core tests](cache_core_test.go)
- [option tests](cache_options_test.go)
- [Connection test](../connectiontest/connection_image_cache_test.go)
- [generator test](../internal/cmd/sdkgen/glance_cache_test.go)
