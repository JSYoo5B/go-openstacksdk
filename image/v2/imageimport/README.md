# Imageimport

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func importStagedImage(ctx context.Context, client *gophercloud.ServiceClient, id string) (*imageimport.ImportResult, error) {
    return imageimport.New(client).ImportImage(ctx, resource.ID(id))
}
```

```python
image = conn.image.get_image(image_id)
response = conn.image.import_image(image)  # method='glance-direct'
```

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/images"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func importKnownFromWeb(ctx context.Context, client *gophercloud.ServiceClient, image *images.Image, uri string) (*imageimport.ImportResult, error) {
    return imageimport.New(client).ImportKnownImage(ctx, image,
        request.Custom(func(c *imageimport.ImportOpts) error { c.Method = "web-download"; return nil }),
        request.Custom(func(c *imageimport.ImportOpts) error { c.URI = uri; return nil }),
        imageimport.WithImportStores("fast", "reliable"),
        request.Custom(func(c *imageimport.ImportOpts) error { v := false; c.AllStoresMustSucceed = &v; return nil }),
    )
}
```

```python
response = conn.image.import_image(
    image, method="web-download", uri=uri,
    stores=["fast", "reliable"], all_stores_must_succeed=False)
```

## References

- [Proxy.import_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L440-L536)
- [Image.import_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L368-L427)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#import-an-image)
- [Glance Import Contracts Test](../../../api/glance_import_contracts_test.go)
- [Import Options Test](import_options_test.go)
- [Import Core Test](import_core_test.go)
- [Glance Import Test](../../../internal/cmd/sdkgen/glance_import_test.go)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/imageimport/requests.go)
