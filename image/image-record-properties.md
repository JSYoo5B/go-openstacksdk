# Image Record Properties

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
record = conn.image.get_image("image-id")
updated = conn.image.update_image_properties(
    record,
    team="platform",
    min_ram="512",
    meta={"example_enabled": False},
)
print(updated)
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
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required literal image ID")
    properties := flag.String("properties", `{"team":"platform","min_ram":"512"}`, "ordered JSON keyword object")
    meta := flag.String("meta", `{"example_enabled":false}`, "raw JSON overlay object")
    viaService := flag.Bool("service", false, "call image.Service instead of Connection")
    flag.Parse()
    if *imageID == "" { log.Fatal("-image-id is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, json.RawMessage(*properties), json.RawMessage(*meta), *viaService); err != nil {
        var native gophercloud.ErrUnexpectedResponseCode
        if errors.As(err, &native) {
            fmt.Fprintf(os.Stderr, "native HTTP %d, response bytes=%d\n", native.Actual, len(native.Body))
        }
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "accepted HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, imageID string, properties, meta json.RawMessage, viaService bool) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
    if err != nil { return err }
    input := image.ImageRecordPropertiesRequest{Record: record}
    options := []image.ImageRecordPropertiesOption{
        image.WithImageRecordPropertiesOpts(image.ImageRecordPropertiesOpts{
            Headers: map[string]string{"X-Request-Source": "image-record-properties-example"},
            Properties: properties,
            Meta: meta,
        }),
        request.Header("X-SDK-Example", "owned-properties"),
    }
    var result *image.ImageRecordPropertiesResult
    if viaService {
        result, err = service.UpdateImagePropertiesRecord(ctx, input, options...)
    } else {
        result, err = conn.UpdateImagePropertiesRecord(ctx, input, options...)
    }
    value := map[string]any{"partial": result != nil && err != nil}
    if result != nil {
        value["updated"] = result.Updated
        if result.Record != nil {
            value["record"] = map[string]any{
                "resource": result.Record.Resource,
                "wire": result.Record.Wire,
                "receipt_status_code": result.Record.StatusCode,
                "receipt_body_bytes": len(result.Record.Envelope),
                "import_methods": result.Record.ImportMethods,
            }
        }
    }
    if err != nil { value["error"] = err.Error() }
    body, marshalErr := json.MarshalIndent(value, "", "  ")
    if marshalErr != nil { return errors.Join(err, marshalErr) }
    fmt.Println(string(body))
    return err
}
```

## References

- [SetImageProperties](update.md)
- [Install](../docs/install.md)
- [Image Record Update](image-record-update.md)
- [UpdateImageRecord](image-record-update.md)
- [Policy](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L213-L229)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L80-L93)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L94-L103)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L159-L172)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1139-L1179)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L159-L176)
- [ Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L84-L120)
- [ Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L221-L246)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
