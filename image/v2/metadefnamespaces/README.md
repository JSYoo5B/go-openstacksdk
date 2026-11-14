# Metadefnamespaces

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_metadefnamespaces1_test.go` in [example_metadefnamespaces1_test.go](example_metadefnamespaces1_test.go), compiled by `go test`.

```python
def manage_namespace(conn, namespace, replacement_namespace, owner, page_limit):
    created = conn.image.create_metadef_namespace(
        namespace=namespace, display_name="Example",
        description="First line\nSecond line", visibility="private",
        owner=owner, is_protected=False)
    fetched = conn.image.get_metadef_namespace(namespace)
    rows = list(conn.image.metadef_namespaces(
        limit=page_limit, visibility="private", sort_key="created_at",
        sort_dir="desc", resource_types="OS::Glance::Image,OS::Nova::Server"))
    updated = conn.image.update_metadef_namespace(
        namespace, namespace=replacement_namespace, display_name="Example",
        description="First line\nSecond line", visibility="private",
        owner=owner, is_protected=False)
    conn.image.delete_metadef_namespace(replacement_namespace, ignore_missing=False)
    return created, fetched, rows, updated
```

See `example_metadefnamespaces2_test.go` in [example_metadefnamespaces2_test.go](example_metadefnamespaces2_test.go), compiled by `go test`.

```python
def create_namespace_known_containers(conn, namespace):
    return conn.image.create_metadef_namespace(
        namespace=namespace,
        resource_type_associations=[{"name": "OS::Nova::Server"}],
        tags=[{"name": "example"}])

def create_namespace_server_body(conn, namespace):
    # Adapter HTTP: properties/objects are not pinned MetadefNamespace descriptors.
    response = conn.image.post("/metadefs/namespaces", json={
        "namespace": namespace,
        "properties": {"mode": {"type": "string", "title": ""}},
        "objects": [{"name": "shape", "properties": {
            "vcpus": {"type": "integer", "title": "VCPUs"}},
            "required": ["vcpus"]}],
        "tags": [{"name": "example"}],
        "resource_type_associations": [{"name": "OS::Nova::Server"}]})
    response.raise_for_status()
    return response
```

## References

- [Metadef Object Namespace Record Lists](../../metadef-object-namespace-record-lists.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1390-L1485)
- [Metadef Namespace](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_namespace.py)
- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#update-namespace)
- [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L342-L387)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1874-L1937)
- [Metadef Namespaces](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L261-L340)
- [Metadef Namespaces](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L552-L603)
- [Metadef Namespaces](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L647-L661)
- [Contracts Test](contracts_test.go)
- [Core Test](core_test.go)
- [Options Test](options_test.go)
- [Connection Image Metadef Namespaces Test](../../../connectiontest/connection_image_metadef_namespaces_test.go)
- [Glance Metadef Namespaces Test](../../../internal/cmd/sdkgen/glance_metadef_namespaces_test.go)
- [Metadef Namespaces](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L705-L925)
- [WSME PropertyType](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/model/metadef_property_type.py#L25-L57)
- [Metadef Namespace](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_namespace.py#L23-L54)
- [Tag](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/tag.py#L38)
- [Metadef Namespaces](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L133-L261)
- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#create-namespace)
- [Nested Core Test](nested_core_test.go)
- [Nested Options Test](nested_options_test.go)
- [Nested Contracts Test](nested_contracts_test.go)
- [Connection Image Metadef Namespace Create Test](../../../connectiontest/connection_image_metadef_namespace_create_test.go)
- [Glance Metadef Namespace Create Test](../../../internal/cmd/sdkgen/glance_metadef_namespace_create_test.go)
