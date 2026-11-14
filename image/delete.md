# Delete

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func deleteWholeImage(ctx context.Context, client *gophercloud.ServiceClient, id string) (*image.DeleteImageResult, error) {
    return image.New(client).DeleteImage(ctx, resource.ID(id))
}
```

```python
conn.image.delete_image(image_id)
```

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func deleteStoreCopy(ctx context.Context, client *gophercloud.ServiceClient, id, storeID string) (*image.DeleteImageResult, error) {
    ignoreMissing := false
    return image.New(client).DeleteImage(ctx, resource.ID(id),
        image.WithDeleteImageOpts(image.DeleteImageOpts{
            StoreID: &storeID,
            IgnoreMissing: &ignoreMissing,
        }),
    )
}
```

```python
conn.image.delete_image(image_id, store=store_id, ignore_missing=False)
```

## References

- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#delete-image-from-store)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#delete-image)
- [Proxy.delete_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L970-L997)
- [Store.delete_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/service_info.py#L46-L75)
- [Delete Contracts Test](delete_contracts_test.go)
- [Delete Core Test](delete_core_test.go)
- [Delete Options Test](delete_options_test.go)
- [Glance Delete Test](../internal/cmd/sdkgen/glance_delete_test.go)
