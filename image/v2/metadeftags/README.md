# Metadeftags

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_metadeftags_test.go` in [example_metadeftags_test.go](example_metadeftags_test.go), compiled by `go test`.

```python
from openstack.image.v2.metadef_namespace import MetadefNamespace

def proxy_namespace_tags(conn, namespace, tag, clear_namespace=False):
    conn.image.add_tag_to_metadef_namespace(namespace, tag)
    conn.image.remove_tag_from_metadef_namespace(namespace, tag)
    if clear_namespace:
        conn.image.remove_tags_from_metadef_namespace(namespace)

def resource_namespace_tags(conn, namespace, names):
    resource = MetadefNamespace.new(id=namespace)
    resource.set_tags(conn.image, names, append=True)
    resource.fetch_tags(conn.image)
    if names:
        resource.check_tag(conn.image, names[0])
    return resource
```

## References

- [Router](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/router.py#L355-L396)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1487-L1539)
- [Namespace.add_tag·set_tags](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_namespace.py#L109-L152)
- [inherited TagMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/tag.py#L28-L142)
- [Metadef Tags](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_tags.py#L288-L337)
- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#create-tags)
- [SQL create_tags](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/sqlalchemy/metadef_api/tag.py#L114-L141)
- [index](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_tags.py#L139-L183)
- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#list-tags)
- [deserializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_tags.py#L411-L429)
- [Core Test](core_test.go)
- [Options Test](options_test.go)
- [Contracts Test](contracts_test.go)
- [Connection Image Metadef Tags Test](../../../connectiontest/connection_image_metadef_tags_test.go)
- [Glance Metadef Tags Test](../../../internal/cmd/sdkgen/glance_metadef_tags_test.go)
