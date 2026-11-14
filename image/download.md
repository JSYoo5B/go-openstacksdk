# Download

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "io"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func downloadTo(ctx context.Context, client *gophercloud.ServiceClient, id string, output io.Writer) (*image.DownloadImageResult, error) {
    return image.New(client).DownloadTo(ctx, resource.ID(id), output)
}
```

```python
response = conn.image.download_image(image_id, output=output, stream=True)
```

```go
package example

import (
    "bytes"
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func downloadByName(ctx context.Context, client *gophercloud.ServiceClient, name string) ([]byte, *image.DownloadImageResult, error) {
    var output bytes.Buffer
    chunk := 64 * 1024
    result, err := image.New(client).DownloadTo(ctx, resource.Name(name), &output,
        image.WithDownloadImageOpts(image.DownloadImageOpts{
            ChunkSize: &chunk,
            StorePreferences: []string{"fast", "archive", "fast"},
        }),
    )
    return output.Bytes(), result, err
}
```

```python
image = conn.image.find_image(name, ignore_missing=False)
response = conn.image.download_image(
    image, output=output, stream=True, chunk_size=64 * 1024,
    store_preferences=["fast", "archive", "fast"])
```

## References

- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#download-binary-image-data)
- [Proxy.download_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L919-L968)
- [DownloadMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/_download.py#L26-L183)
- [Proxy.request](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L209-L285)
- [ImageData.Download](v2/imagedata/README.md)
- [CreateAndImport](create-import.md)
- [Service.Upload](README.md)
- [Download Contracts Test](download_contracts_test.go)
- [Download Core Test](download_core_test.go)
- [Download Options Test](download_options_test.go)
- [Glance Download Test](../internal/cmd/sdkgen/glance_download_test.go)
