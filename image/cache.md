# Cache

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_cache_test.go` in [example_cache_test.go](example_cache_test.go), compiled by `go test`.

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
