# Imagedata

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "io"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func stageExistingImage(ctx context.Context, client *gophercloud.ServiceClient, id string, data io.Reader) (*imagedata.StageImageResult, error) {
    return imagedata.New(client).StageImage(ctx, resource.ID(id), data)
}
```

```python
image = conn.image.get_image(image_id)
image = conn.image.stage_image(image, data=data)
```

```go
package example

import (
    "context"
    "io"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/images"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func stageKnownImage(ctx context.Context, client *gophercloud.ServiceClient, image *images.Image, data io.Reader, remainingBytes int64) (*imagedata.StageImageResult, error) {
    return imagedata.New(client).StageKnownImage(ctx, image, data,
        request.Custom(func(c *imagedata.StageOpts) error { v := remainingBytes; c.Size = &v; return nil }),
        request.Header("X-Trace", "stage-image"),
    )
}
```

```python
image = conn.image.stage_image(image, data=data, size=remaining_bytes)
```

## References

- [Proxy.stage_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L538-L581)
- [Image.stage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L325-L366)
- [get_file_size](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L462-L491)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#stage-binary-image-data)
- [Imageimport](../imageimport/README.md)
- [Glance Stage Contracts Test](../../../api/glance_stage_contracts_test.go)
- [Stage Core Test](stage_core_test.go)
- [Stage Options Test](stage_options_test.go)
- [Glance Stage Test](../../../internal/cmd/sdkgen/glance_stage_test.go)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/imagedata/requests.go)
- [Image Record Download](../../image-record-download.md)
