# Image Record Tags

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")

result = conn.image.add_tag("image-id", "reviewed")
assert result is None

image = conn.image.get_image("image-id")
conn.image.remove_tag(image, "reviewed")
print(image.tags)
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
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required literal image ID")
    tag := flag.String("tag", "reviewed", "literal tag to add and then remove")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *tag); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func printResult(value *image.ImageRecordTagResult, operationErr error) error {
    if value != nil {
        output := make(map[string]any)
        if ack := value.Acknowledgement; ack != nil {
            output["acknowledgement"] = map[string]any{
                "image_id": ack.ImageID, "tag": ack.Tag,
                "status_code": ack.StatusCode, "body_bytes": len(ack.Body),
            }
        }
        if value.Record != nil {
            output["resource"] = value.Record.Resource
            output["wire"] = value.Record.Wire
            output["record_status_code"] = value.Record.StatusCode
        }
        body, err := json.MarshalIndent(output, "", "  ")
        if err != nil { return errors.Join(operationErr, err) }
        fmt.Println(string(body))
    }
    return operationErr
}

func run(ctx context.Context, cloud, imageID, tag string) error {
    if imageID == "" { return fmt.Errorf("-image-id is required") }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }

    added, err := service.AddImageRecordTag(ctx, image.ImageRecordTagRequest{ID: imageID}, tag,
        request.Header("X-Request-Source", "image-record-tags-example"))
    if err := printResult(added, err); err != nil { return err }

    seed, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
    if err != nil { return err }
    removed, err := service.RemoveImageRecordTag(ctx, image.ImageRecordTagRequest{Record: seed}, tag,
        image.WithImageMutationHeaders(map[string]string{"X-Request-Source": "image-record-tags-example"}))
    return printResult(removed, err)
}
```

## References

- [ImageRecord](image-records.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1196-L1220)
- [TagMixin add/remove](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/tag.py#L105-L145)
- [Install](../docs/install.md)
- [urljoin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L43-L50)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L602)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L785-L839)
- [Image Record Tag Test](image_record_tag_test.go)
- [Connection Image Record Tags Test](../connectiontest/connection_image_record_tags_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
