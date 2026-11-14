# Metadefproperties

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

type PropertyEvidence struct {
    Created *metadefproperties.Property
    Read *metadefproperties.Property
    Rows []*metadefproperties.Property
    All []*metadefproperties.Property
    Updated *metadefproperties.Property
    Deleted *metadefproperties.Acknowledgement
    Cleared *metadefproperties.Acknowledgement
}

func manageProperties(ctx context.Context, service *image.Service, namespace, name, replacementName string, clearNamespace bool) (out PropertyEvidence, err error) {
    scope, err := service.API.MetadefProperties.InNamespace(ctx, namespace)
    if err != nil { return out, err }
    propertyType, title := "string", "Hypervisor Type"
    description := "First line\nSecond line"
    attributes := map[string]any{"enum": []string{"kvm", "qemu"}, "default": "kvm"}
    out.Created, err = scope.Create(ctx, name,
        metadefproperties.WithCreateOpts(metadefproperties.CreateOpts{
            Type: &propertyType, Title: &title,
            Attributes: map[string]json.RawMessage{"enum": json.RawMessage(`["kvm","qemu"]`)},
        }),
        request.Header("X-Request-Source", "example"),
        metadefproperties.WithCreateHeaders(map[string]string{"X-Trace": "create"}),
        request.Custom(func(c *metadefproperties.CreateOpts) error { v := propertyType; c.Type = &v; return nil }),
        request.Custom(func(c *metadefproperties.CreateOpts) error { v := title; c.Title = &v; return nil }),
        request.Custom(func(c *metadefproperties.CreateOpts) error { v := description; c.Description = &v; return nil }),
        metadefproperties.WithCreateAttributes(attributes),
        metadefproperties.WithCreateAttribute("readonly", false))
    if err != nil { return out, err }

    out.Read, err = scope.Get(ctx, name,
        metadefproperties.WithGetOpts(metadefproperties.GetOpts{}),
        request.Header("X-Request-Source", "example"),
        metadefproperties.WithGetHeaders(map[string]string{"X-Trace": "get"}),
        request.Custom(func(c *metadefproperties.GetOpts) error { v := ""; c.ResourceType = &v; return nil }))
    if err != nil { return out, err }

    listOptions := []metadefproperties.ListOption{
        metadefproperties.WithListOpts(metadefproperties.ListOpts{}),
        request.Header("X-Request-Source", "example"),
        metadefproperties.WithListHeaders(map[string]string{"X-Trace": "list"}),
        request.MaxItems(20),
    }
    for row, listErr := range scope.List(ctx, listOptions...) {
        if listErr != nil { return out, listErr }
        out.Rows = append(out.Rows, row)
    }
    out.All, err = scope.All(ctx, listOptions...)
    if err != nil { return out, err }

    out.Updated, err = scope.Update(ctx, name,
        metadefproperties.WithUpdateOpts(metadefproperties.UpdateOpts{}),
        request.Header("X-Request-Source", "example"),
        metadefproperties.WithUpdateHeaders(map[string]string{"X-Trace": "update"}),
        request.Custom(func(c *metadefproperties.UpdateOpts) error { v := replacementName; c.Name = &v; return nil }),
        request.Custom(func(c *metadefproperties.UpdateOpts) error { v := propertyType; c.Type = &v; return nil }),
        request.Custom(func(c *metadefproperties.UpdateOpts) error { v := title; c.Title = &v; return nil }),
        request.Custom(func(c *metadefproperties.UpdateOpts) error { v := description; c.Description = &v; return nil }),
        metadefproperties.WithUpdateAttributes(attributes),
        metadefproperties.WithUpdateAttribute("readonly", false))
    if err != nil { return out, err }

    out.Deleted, err = scope.Delete(ctx, replacementName,
        metadefproperties.WithDeleteOpts(metadefproperties.DeleteOpts{}),
        request.Header("X-Request-Source", "example"),
        metadefproperties.WithDeleteHeaders(map[string]string{"X-Trace": "delete"}),
        request.IgnoreMissing(false))
    if err != nil || !clearNamespace { return out, err }
    out.Cleared, err = scope.DeleteAll(ctx,
        metadefproperties.WithDeleteAllOpts(metadefproperties.DeleteAllOpts{}),
        request.Header("X-Request-Source", "example"),
        metadefproperties.WithDeleteAllHeaders(map[string]string{"X-Trace": "delete-all"}))
    return out, err
}
```

```python
def manage_properties(conn, namespace, name, replacement_name, clear_namespace=False):
    created = conn.image.create_metadef_property(
        namespace, name=name, type="string", title="Hypervisor Type",
        description="First line\nSecond line", enum=["kvm", "qemu"], default="kvm", readonly=False)
    fetched = conn.image.get_metadef_property(name, namespace)
    rows = list(conn.image.metadef_properties(namespace))
    updated = conn.image.update_metadef_property(
        name, namespace, name=replacement_name, type="string", title="Hypervisor Type",
        description="First line\nSecond line", enum=["kvm", "qemu"], default="kvm", readonly=False)
    conn.image.delete_metadef_property(replacement_name, namespace, ignore_missing=False)
    if clear_namespace:
        conn.image.delete_all_metadef_properties(namespace)
    return created, fetched, rows, updated
```

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefproperties"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func deleteOwnedProperties(ctx context.Context, scope *metadefproperties.NamespaceScope, id string, clearNamespace bool) (member, collection *metadefproperties.Acknowledgement, err error) {
    member, err = scope.DeleteRecord(ctx, metadefproperties.RecordRequest{ID: id},
        request.IgnoreMissing(false),
        request.Header("X-Request-Source", "owned-delete-example"),
    )
    if err != nil || !clearNamespace { return member, nil, err }
    collection, err = scope.DeleteAllRecords(ctx,
        request.Header("X-Request-Source", "owned-delete-example"),
    )
    return member, collection, err
}
```

```python
def delete_owned_properties(conn, namespace, id, clear_namespace=False):
    conn.image.delete_metadef_property(id, namespace, ignore_missing=False)
    if clear_namespace:
        conn.image.delete_all_metadef_properties(namespace)
```

## References

- [Metadef Property Records](../../metadef-property-records.md)
- [proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1790-L1947)
- [Metadef Properties](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_properties.py#L254-L389)
- [Schema.raw](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/schema.py#L86-L103)
- [property schema](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_namespaces.py#L692-L806)
- [WSME PropertyType](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/model/metadef_property_type.py#L23-L62)
- [controller.update](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_properties.py#L185-L221)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1874-L1937)
- [server.show](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_properties.py#L98-L146)
- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#show-property-definition)
- [MetadefProperty.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_property.py#L86-L185)
- [Metadef Properties](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_properties.py#L64-L96)
- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#list-properties)
- [core tests](core_test.go)
- [option tests](options_test.go)
- [Contracts Test](contracts_test.go)
- [Connection contract](../../../connectiontest/connection_image_metadef_properties_test.go)
- [generator contract](../../../internal/cmd/sdkgen/glance_metadef_properties_test.go)
- [Metadef Property Record List](../../metadef-property-record-list.md)
- [Metadef Property Record Write](../../metadef-property-record-write.md)
