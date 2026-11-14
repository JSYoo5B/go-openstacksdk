# Metadefresourcetypes

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/image"
    types "github.com/JSYoo5B/go-openstacksdk/image/v2/metadefresourcetypes"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

type AssociationEvidence struct {
    GlobalRows []*types.ResourceType
    GlobalAll  []*types.ResourceType
    Created    *types.Association
    Rows       []*types.Association
    All        []*types.Association
    Deleted    *types.Acknowledgement
}

func manageResourceTypeAssociation(
    ctx context.Context,
    service *image.Service,
    namespace, resourceType, prefix, target string,
) (*AssociationEvidence, error) {
    evidence := &AssociationEvidence{}
    api := service.API.MetadefResourceTypes
    listOptions := []types.ListOption{
        types.WithListOpts(types.ListOpts{MaxItems: 20}),
        request.Header("X-Example", "resource-types"),
        types.WithListHeaders(map[string]string{"X-Request-Group": "example"}),
        request.MaxItems(20),
    }
    for row, err := range api.List(ctx, listOptions...) {
        if err != nil {
            return evidence, err
        }
        evidence.GlobalRows = append(evidence.GlobalRows, row)
    }
    var err error
    evidence.GlobalAll, err = api.All(ctx, listOptions...)
    if err != nil {
        return evidence, err
    }
    scope, err := api.InNamespace(ctx, namespace)
    if err != nil {
        return evidence, err
    }
    evidence.Created, err = scope.Create(ctx, resourceType,
        types.WithCreateOpts(types.CreateOpts{}),
        request.Header("X-Example", "association"),
        types.WithCreateHeaders(map[string]string{"X-Request-Group": "example"}),
        request.Custom(func(c *types.CreateOpts) error { v := prefix; c.Prefix = &v; return nil }),
        request.Custom(func(c *types.CreateOpts) error { v := target; c.PropertiesTarget = &v; return nil }),
    )
    if err != nil {
        return evidence, err
    }
    for row, err := range scope.List(ctx, listOptions...) {
        if err != nil {
            return evidence, err
        }
        evidence.Rows = append(evidence.Rows, row)
    }
    evidence.All, err = scope.All(ctx, listOptions...)
    if err != nil {
        return evidence, err
    }
    evidence.Deleted, err = scope.Delete(ctx, resourceType,
        types.WithDeleteOpts(types.DeleteOpts{}),
        request.Header("X-Example", "association"),
        types.WithDeleteHeaders(map[string]string{"X-Request-Group": "example"}),
        request.IgnoreMissing(false),
    )
    return evidence, err
}
```

```python
def resource_type_association(conn, namespace, resource_type, prefix, target):
    global_types = list(conn.image.metadef_resource_types(paginated=False))
    created = conn.image.create_metadef_resource_type_association(
        namespace,
        name=resource_type,
        prefix=prefix,
        properties_target=target,
    )
    associations = list(
        conn.image.metadef_resource_type_associations(namespace, paginated=False)
    )
    conn.image.delete_metadef_resource_type_association(
        resource_type, namespace, ignore_missing=False
    )
    return global_types, created, associations
```

## References

- [Metadefs Index](https://docs.openstack.org/api-ref/image/v2/metadefs-index.html#metadata-definition-resource-types)
- [Metadef Resource Types](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_resource_types.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1696-L1786)
- [Metadef Resource Type](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/metadef_resource_type.py)
- [schema](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_resource_types.py#L263-L317)
- [index·show](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/metadef_resource_types.py#L49-L119)
- [Core Test](core_test.go)
- [Options Test](options_test.go)
- [Contracts Test](contracts_test.go)
- [Connection Image Metadef Resource Types Test](../../../connectiontest/connection_image_metadef_resource_types_test.go)
- [Glance Metadef Resource Types Test](../../../internal/cmd/sdkgen/glance_metadef_resource_types_test.go)
- [Metadef Resource Types Records](../../metadef-resource-types-records.md)
- [Metadef Resource Type Association Mutations](../../metadef-resource-type-association-mutations.md)
