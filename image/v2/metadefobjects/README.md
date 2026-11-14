# Metadefobjects

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefobjects"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

type ObjectEvidence struct {
    Created *metadefobjects.Object
    Read *metadefobjects.Object
    Rows []*metadefobjects.Object
    All []*metadefobjects.Object
    Updated *metadefobjects.Object
    Deleted *metadefobjects.Acknowledgement
    Cleared *metadefobjects.Acknowledgement
}

func manageObjects(ctx context.Context, service *image.Service, namespace, name, replacementName string, clearNamespace bool) (out ObjectEvidence, err error) {
    scope, err := service.API.MetadefObjects.InNamespace(ctx, namespace)
    if err != nil { return out, err }
    description := "First line\nSecond line"
    properties := map[string]json.RawMessage{
        "vcpus": json.RawMessage(`{"type":"integer","title":"Virtual CPUs","minimum":1}`),
    }
    required := []string{"vcpus"}
    out.Created, err = scope.Create(ctx, name,
        metadefobjects.WithCreateOpts(metadefobjects.CreateOpts{Description: &description}),
        request.Header("X-Request-Source", "example"),
        metadefobjects.WithCreateHeaders(map[string]string{"X-Trace": "create"}),
        request.Custom(func(c *metadefobjects.CreateOpts) error { v := description; c.Description = &v; return nil }),
        metadefobjects.WithCreateProperties(properties),
        metadefobjects.WithCreateRequired(required))
    if err != nil { return out, err }

    out.Read, err = scope.Get(ctx, name,
        metadefobjects.WithGetOpts(metadefobjects.GetOpts{}),
        request.Header("X-Request-Source", "example"),
        metadefobjects.WithGetHeaders(map[string]string{"X-Trace": "get"}))
    if err != nil { return out, err }

    listOptions := []metadefobjects.ListOption{
        metadefobjects.WithListOpts(metadefobjects.ListOpts{}),
        request.Header("X-Request-Source", "example"),
        metadefobjects.WithListHeaders(map[string]string{"X-Trace": "list"}),
        request.MaxItems(20),
    }
    for row, listErr := range scope.List(ctx, listOptions...) {
        if listErr != nil { return out, listErr }
        out.Rows = append(out.Rows, row)
    }
    out.All, err = scope.All(ctx, listOptions...)
    if err != nil { return out, err }

    out.Updated, err = scope.Update(ctx, name,
        metadefobjects.WithUpdateOpts(metadefobjects.UpdateOpts{}),
        request.Header("X-Request-Source", "example"),
        metadefobjects.WithUpdateHeaders(map[string]string{"X-Trace": "update"}),
        request.Custom(func(c *metadefobjects.UpdateOpts) error { v := replacementName; c.Name = &v; return nil }),
        request.Custom(func(c *metadefobjects.UpdateOpts) error { v := description; c.Description = &v; return nil }),
        metadefobjects.WithUpdateProperties(properties),
        metadefobjects.WithUpdateRequired(required))
    if err != nil { return out, err }

    out.Deleted, err = scope.Delete(ctx, replacementName,
        metadefobjects.WithDeleteOpts(metadefobjects.DeleteOpts{}),
        request.Header("X-Request-Source", "example"),
        metadefobjects.WithDeleteHeaders(map[string]string{"X-Trace": "delete"}),
        request.IgnoreMissing(false))
    if err != nil || !clearNamespace { return out, err }
    out.Cleared, err = scope.DeleteAll(ctx,
        metadefobjects.WithDeleteAllOpts(metadefobjects.DeleteAllOpts{}),
        request.Header("X-Request-Source", "example"),
        metadefobjects.WithDeleteAllHeaders(map[string]string{"X-Trace": "delete-all"}))
    return out, err
}
```

```python
def manage_objects(conn, namespace, name, replacement_name, clear_namespace=False):
    properties = {"vcpus": {"type": "integer", "title": "Virtual CPUs", "minimum": 1}}
    created = conn.image.create_metadef_object(
        namespace, name=name, description="First line\nSecond line",
        properties=properties, required=["vcpus"])
    fetched = conn.image.get_metadef_object(name, namespace)
    rows = list(conn.image.metadef_objects(namespace))
    updated = conn.image.update_metadef_object(
        name, namespace, name=replacement_name,
        description="First line\nSecond line", properties=properties, required=["vcpus"])
    deleted = conn.image.delete_metadef_object(
        replacement_name, namespace, ignore_missing=False)
    cleared = conn.image.delete_all_metadef_objects(namespace) if clear_namespace else None
    return created, fetched, rows, updated, deleted, cleared
```

## References

- [Metadef Object Namespace Record Lists](../../metadef-object-namespace-record-lists.md)
- [proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1543-L1692)
- [  Init  ](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/__init__.py#L561-L601)
- [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_objects.py#L174-L221)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1874-L1937)
- [repository.list](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/__init__.py#L625-L632)
- [serializer.index](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_objects.py#L430-L435)
- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#list-objects)
- [Contracts Test](contracts_test.go)
- [Core Test](core_test.go)
- [Options Test](options_test.go)
- [Connection Image Metadef Objects Test](../../../connectiontest/connection_image_metadef_objects_test.go)
- [Glance Metadef Objects Test](../../../internal/cmd/sdkgen/glance_metadef_objects_test.go)
