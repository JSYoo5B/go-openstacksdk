# Metadeftags

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/metadeftags"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

type TagEvidence struct {
    Created *metadeftags.Tag
    Read *metadeftags.Tag
    Updated *metadeftags.Tag
    Set *metadeftags.SetResult
    Rows []*metadeftags.Tag
    All []*metadeftags.Tag
    Deleted *metadeftags.Acknowledgement
    Cleared *metadeftags.Acknowledgement
}

func manageNamespaceTags(ctx context.Context, service *image.Service, namespace, current, replacement string, setNames []string, clearNamespace bool) (out TagEvidence, err error) {
    scope, err := service.API.MetadefTags.InNamespace(ctx, namespace)
    if err != nil { return out, err }
    out.Created, err = scope.Create(ctx, current,
        metadeftags.WithCreateOpts(metadeftags.CreateOpts{}),
        request.Header("X-Request-Source", "example"),
        metadeftags.WithCreateHeaders(map[string]string{"X-Trace": "create"}))
    if err != nil { return out, err }
    out.Read, err = scope.Get(ctx, current,
        metadeftags.WithGetOpts(metadeftags.GetOpts{}),
        request.Header("X-Request-Source", "example"),
        metadeftags.WithGetHeaders(map[string]string{"X-Trace": "get"}))
    if err != nil { return out, err }
    out.Updated, err = scope.Update(ctx, current,
        metadeftags.WithUpdateOpts(metadeftags.UpdateOpts{}),
        request.Header("X-Request-Source", "example"),
        metadeftags.WithUpdateHeaders(map[string]string{"X-Trace": "update"}),
        request.Custom(func(c *metadeftags.UpdateOpts) error { v := replacement; c.Name = &v; return nil }))
    if err != nil { return out, err }
    out.Set, err = scope.Set(ctx, setNames,
        metadeftags.WithSetOpts(metadeftags.SetOpts{}),
        request.Header("X-Request-Source", "example"),
        metadeftags.WithSetHeaders(map[string]string{"X-Trace": "set"}),
        request.Custom(func(c *metadeftags.SetOpts) error { v := true; c.Append = &v; return nil }))
    if err != nil { return out, err }

    listOptions := []metadeftags.ListOption{
        metadeftags.WithListOpts(metadeftags.ListOpts{}),
        request.Header("X-Request-Source", "example"),
        metadeftags.WithListHeaders(map[string]string{"X-Trace": "list"}),
        request.Limit(2),
        request.Marker(""),
        request.Custom(func(c *metadeftags.ListOpts) error { v := "name"; c.SortKey = &v; return nil }),
        request.Custom(func(c *metadeftags.ListOpts) error { v := "asc"; c.SortDir = &v; return nil }),
        request.MaxItems(20),
    }
    for row, listErr := range scope.List(ctx, listOptions...) {
        if listErr != nil { return out, listErr }
        out.Rows = append(out.Rows, row)
    }
    out.All, err = scope.All(ctx, listOptions...)
    if err != nil { return out, err }
    out.Deleted, err = scope.Delete(ctx, replacement,
        metadeftags.WithDeleteOpts(metadeftags.DeleteOpts{}),
        request.Header("X-Request-Source", "example"),
        metadeftags.WithDeleteHeaders(map[string]string{"X-Trace": "delete"}))
    if err != nil || !clearNamespace { return out, err }
    out.Cleared, err = scope.DeleteAll(ctx,
        metadeftags.WithDeleteAllOpts(metadeftags.DeleteAllOpts{}),
        request.Header("X-Request-Source", "example"),
        metadeftags.WithDeleteAllHeaders(map[string]string{"X-Trace": "delete-all"}))
    return out, err
}
```

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
