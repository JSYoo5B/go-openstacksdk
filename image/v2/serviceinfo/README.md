# Serviceinfo

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func discoverStores(ctx context.Context, conn *openstack.Connection) ([]*serviceinfo.Store, error) {
    service, err := conn.ImageV2(ctx)
    if err != nil {
        return nil, err
    }
    return service.ServiceInfo.AllStores(ctx,
        request.Custom(func(c *request.Config[serviceinfo.ListStoresOpts]) error { c.Options.Details = true; return nil }),
        request.MaxItems(20),
    )
}
```

```python
for store in conn.image.stores(details=True, max_items=20):
    print(store.id, store.description, store.is_default)
```

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
)

func discoverImportMethods(ctx context.Context, client *gophercloud.ServiceClient) (*serviceinfo.ImportInfo, error) {
    return image.New(client).API.ServiceInfo.GetImportInfo(ctx)
}
```

```python
info = conn.image.get_import_info()
methods = info.import_methods
print(methods)
```

## References

- [Records](records.md)
- [Python Store Details](python-store-details.md)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#list-stores)
- [detail API](https://docs.openstack.org/api-ref/image/v2/index.html#list-stores-detail)
- [basic controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L53-L93)
- [detail controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/discovery.py#L165-L195)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#import-methods-and-values-discovery)
- [Proxy.stores/get_import_info](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L2273-L2295)
- [Store/Import](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/service_info.py#L22-L44)
- [Contracts Test](contracts_test.go)
- [Core Test](core_test.go)
- [Options Test](options_test.go)
- [Connection Image Serviceinfo Test](../../../connectiontest/connection_image_serviceinfo_test.go)
- [Glance Serviceinfo Test](../../../internal/cmd/sdkgen/glance_serviceinfo_test.go)
- [Response Validation Test](../../../internal/rest/response_validation_test.go)
- [Usage](usage.md)
