# Metadefnamespaces

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

type NamespaceEvidence struct {
    Created *metadefnamespaces.Namespace
    Read *metadefnamespaces.Namespace
    Rows []*metadefnamespaces.Namespace
    All []*metadefnamespaces.Namespace
    Updated *metadefnamespaces.Namespace
    Deleted *metadefnamespaces.Acknowledgement
}

func manageNamespace(ctx context.Context, service *image.Service, namespace, replacementNamespace, owner, resourceType string, pageLimit int) (out NamespaceEvidence, err error) {
    api := service.API.MetadefNamespaces
    description := "First line\nSecond line"
    protected := false
    out.Created, err = api.Create(ctx, namespace,
        metadefnamespaces.WithCreateOpts(metadefnamespaces.CreateOpts{
            Description: &description, Protected: &protected,
        }),
        request.Header("X-Request-Source", "example"),
        metadefnamespaces.WithCreateHeaders(map[string]string{"X-Trace": "create"}),
        request.Custom(func(c *metadefnamespaces.CreateOpts) error { v := "Example"; c.DisplayName = &v; return nil }),
        request.Custom(func(c *metadefnamespaces.CreateOpts) error { v := description; c.Description = &v; return nil }),
        request.Custom(func(c *metadefnamespaces.CreateOpts) error { v := "private"; c.Visibility = &v; return nil }),
        request.Custom(func(c *metadefnamespaces.CreateOpts) error { v := owner; c.Owner = &v; return nil }),
        request.Custom(func(c *metadefnamespaces.CreateOpts) error { v := false; c.Protected = &v; return nil }))
    if err != nil { return out, err }

    out.Read, err = api.Get(ctx, namespace,
        metadefnamespaces.WithGetOpts(metadefnamespaces.GetOpts{}),
        request.Header("X-Request-Source", "example"),
        metadefnamespaces.WithGetHeaders(map[string]string{"X-Trace": "get"}),
        request.Custom(func(c *metadefnamespaces.GetOpts) error { v := resourceType; c.ResourceType = &v; return nil }))
    if err != nil { return out, err }

    listOptions := []metadefnamespaces.ListOption{
        metadefnamespaces.WithListOpts(metadefnamespaces.ListOpts{}),
        request.Header("X-Request-Source", "example"),
        metadefnamespaces.WithListHeaders(map[string]string{"X-Trace": "list"}),
        request.Limit(pageLimit),
        request.Marker(""),
        request.Custom(func(c *metadefnamespaces.ListOpts) error { c.Visibility = "private"; return nil }),
        request.Custom(func(c *metadefnamespaces.ListOpts) error { c.SortKey = "created_at"; return nil }),
        request.Custom(func(c *metadefnamespaces.ListOpts) error { c.SortDir = "desc"; return nil }),
        request.Custom(func(c *metadefnamespaces.ListOpts) error { c.ResourceTypes = "OS::Glance::Image,OS::Nova::Server"; return nil }),
        request.MaxItems(20),
        request.Custom(func(c *metadefnamespaces.ListOpts) error { c.SinglePage = false; return nil }),
    }
    for row, listErr := range api.List(ctx, listOptions...) {
        if listErr != nil { return out, listErr }
        out.Rows = append(out.Rows, row)
    }
    out.All, err = api.All(ctx, listOptions...)
    if err != nil { return out, err }

    out.Updated, err = api.Update(ctx, namespace,
        metadefnamespaces.WithUpdateOpts(metadefnamespaces.UpdateOpts{}),
        request.Header("X-Request-Source", "example"),
        metadefnamespaces.WithUpdateHeaders(map[string]string{"X-Trace": "update"}),
        request.Custom(func(c *metadefnamespaces.UpdateOpts) error { v := replacementNamespace; c.Namespace = &v; return nil }),
        request.Custom(func(c *metadefnamespaces.UpdateOpts) error { v := "Example"; c.DisplayName = &v; return nil }),
        request.Custom(func(c *metadefnamespaces.UpdateOpts) error { v := description; c.Description = &v; return nil }),
        request.Custom(func(c *metadefnamespaces.UpdateOpts) error { v := "private"; c.Visibility = &v; return nil }),
        request.Custom(func(c *metadefnamespaces.UpdateOpts) error { v := owner; c.Owner = &v; return nil }),
        request.Custom(func(c *metadefnamespaces.UpdateOpts) error { v := false; c.Protected = &v; return nil }))
    if err != nil { return out, err }

    out.Deleted, err = api.Delete(ctx, replacementNamespace,
        metadefnamespaces.WithDeleteOpts(metadefnamespaces.DeleteOpts{}),
        request.Header("X-Request-Source", "example"),
        metadefnamespaces.WithDeleteHeaders(map[string]string{"X-Trace": "delete"}),
        request.IgnoreMissing(false))
    return out, err
}
```

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

```go
package example

import (
    "context"
    "encoding/json"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefnamespaces"
)

func createNestedNamespace(ctx context.Context, service *image.Service, namespace string) (*metadefnamespaces.Namespace, error) {
    description := "A compute shape"
    prefix, target := "hw:", "metadata"
    properties := map[string]metadefnamespaces.PropertyDefinition{
        "mode": {
            Type: "string", Title: "",
            Attributes: map[string]json.RawMessage{
                "enum": json.RawMessage(`["small","large"]`),
                "default": json.RawMessage(`"small"`),
            },
        },
    }
    objects := []metadefnamespaces.ObjectDefinition{{
        Name: "shape", Description: &description,
        Properties: map[string]metadefnamespaces.PropertyDefinition{
            "vcpus": {
                Type: "integer", Title: "VCPUs",
                Attributes: map[string]json.RawMessage{"minimum": json.RawMessage(`0`)},
            },
        },
        Required: []string{"vcpus"},
    }}
    tags := []metadefnamespaces.TagDefinition{{Name: "example"}}
    associations := []metadefnamespaces.ResourceTypeAssociationDefinition{{
        Name: "OS::Nova::Server", Prefix: &prefix, PropertiesTarget: &target,
    }}
    return service.API.MetadefNamespaces.Create(ctx, namespace,
        metadefnamespaces.WithCreateOpts(metadefnamespaces.CreateOpts{}),
        metadefnamespaces.WithCreateProperties(properties),
        metadefnamespaces.WithCreateObjects(objects),
        metadefnamespaces.WithCreateTags(tags),
        metadefnamespaces.WithCreateResourceTypeAssociations(associations))
}
```

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
