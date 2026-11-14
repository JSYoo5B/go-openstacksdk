# Image Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
image = conn.image.get_image("image-id")
print(image.to_dict(), image.image_import_methods)

for image in conn.image.images(status="active", limit=10, max_items=20):
    print(image.to_dict())

image = conn.image.find_image("ubuntu", ignore_missing=False)
print(image.to_dict())
```

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required image ID for literal GET")
    find := flag.String("find", "", "optional exact image name or ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *find); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func printRecord(value *image.ImageRecord) error {
    body, err := json.MarshalIndent(map[string]any{
        "resource": value.Resource, "wire": value.Wire,
        "status_code": value.StatusCode, "import_methods": value.ImportMethods,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(body))
    return nil
}

func run(ctx context.Context, cloud, imageID, nameOrID string) error {
    if imageID == "" { return fmt.Errorf("-image-id is required") }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }

    value, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID},
        request.Header("X-Request-Source", "image-records-example"))
    if err != nil { return err }
    if err := printRecord(value); err != nil { return err }

    for value, readErr := range service.ListImageRecords(ctx,
        image.WithImageRecordListFilter("status", "active"),
        request.Limit(10), request.MaxItems(20)) {
        if readErr != nil { return readErr }
        if err := printRecord(value); err != nil { return err }
    }

    if nameOrID != "" {
        found, err := service.FindImageRecord(ctx, nameOrID,
            request.IgnoreMissing(false))
        if err != nil { return err }
        if err := printRecord(found); err != nil { return err }
    }
    return nil
}
```

## References

- [Download API](download.md)
- [Images](images.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1013-L1054)
- [Install](../docs/install.md)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L45-L63)
- [QueryParameters validation·transpose](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L301-L382)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L79-L260)
- [Fields](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86-L128)
- [properties packing](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1221-L1230)
- [fetch translation](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1356-L1394)
- [constructor collector](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L808-L839)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L429-L437)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L494-L531)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2524-L2582)
- [Image Record Get Test](image_record_get_test.go)
- [Image Record Lists Test](image_record_lists_test.go)
- [Image Record Find Test](image_record_find_test.go)
- [Connection Image Records Test](../connectiontest/connection_image_records_test.go)
- [Python String Test](../internal/cloudfilter/python_string_test.go)
