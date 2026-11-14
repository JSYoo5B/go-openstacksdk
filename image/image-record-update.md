# Image Record Update

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")

updated = conn.image.update_image("image-id", name="release-image")

same = conn.image.update_image(updated, name=updated.name)
assert same is updated
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
    name := flag.String("name", "release-image", "new image name")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *name); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func printRecord(label string, value *image.ImageRecord, operationErr error) error {
    output := map[string]any{"step": label, "partial": value != nil && operationErr != nil}
    if value != nil {
        output["resource"] = value.Resource
        output["wire"] = value.Wire
        output["receipt_status_code"] = value.StatusCode
        output["receipt_body_bytes"] = len(value.Envelope)
        output["import_methods"] = value.ImportMethods
    }
    if operationErr != nil { output["error"] = operationErr.Error() }
    body, marshalErr := json.MarshalIndent(output, "", "  ")
    if marshalErr != nil { return errors.Join(operationErr, marshalErr) }
    fmt.Println(string(body))
    return operationErr
}

func run(ctx context.Context, cloud, imageID, name string) error {
    if imageID == "" { return fmt.Errorf("-image-id is required") }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }

    updated, err := conn.UpdateImageRecord(ctx, image.ImageRecordUpdateRequest{
        ID: imageID,
        Attributes: map[string]any{"name": name},
    }, request.Header("X-Request-Source", "image-record-update-example"))
    if err := printRecord("update by ID", updated, err); err != nil { return err }

    // name has no descriptor conversion: use the returned raw JSON spelling.
    sameName := json.RawMessage(updated.Resource.Body["name"])
    unchanged, err := service.UpdateImageRecord(ctx, image.ImageRecordUpdateRequest{
        Record: updated,
    }, image.WithImageRecordAttributes(map[string]any{"name": sameName}))
    return printRecord("reuse record with same name", unchanged, err)
}
```

## References

- [Update](update.md)
- [native images.Update](v2/images/update.md)
- [Image proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1056-L1068)
- [Proxy._update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L716-L748)
- [Image Records](image-records.md)
- [Install](../docs/install.md)
- [Image Records Model](image_records_model.go)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1251-L1273)
- [Pyproject](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/pyproject.toml#L20)
- [Source packing·flatten](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1207-L1229)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L439-L463)
- [Image Record Update Test](image_record_update_test.go)
- [Image Record Update Integration Test](../connectiontest/image_record_update_integration_test.go)
- [Diff Test](../internal/jsonpatch/diff_test.go)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L213-L223)
- [commit/no-op](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1875-L1937)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1394)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
