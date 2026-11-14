# Image Record Actions

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
conn.image.deactivate_image("image-id")

# conn.image.reactivate_image("image-id")
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
    action := flag.String("action", "deactivate", "deactivate or reactivate")
    fromRecord := flag.Bool("record", false, "explicitly fetch an SDK record first")
    viaService := flag.Bool("service", false, "use image.Service for the selected POST")
    flag.Parse()
    if *imageID == "" { log.Fatal("-image-id is required") }
    if *action != "deactivate" && *action != "reactivate" {
        log.Fatal("-action must be deactivate or reactivate")
    }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *action, *fromRecord, *viaService); err != nil {
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

func run(ctx context.Context, cloud, imageID, action string, fromRecord, viaService bool) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    input := image.ImageRecordActionRequest{ID: imageID}
    var service *image.Service
    if fromRecord || viaService {
        service, err = conn.Image(ctx)
        if err != nil { return err }
    }
    if fromRecord {
        record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
        if err != nil { return err }
        input = image.ImageRecordActionRequest{Record: record}
    }
    options := []image.ImageRecordActionOption{
        image.WithImageRecordActionOpts(image.ImageRecordActionOpts{
            Headers: map[string]string{"X-Request-Source": "image-record-actions-example"},
        }),
        request.Header("X-SDK-Example-Action", action),
        image.WithImageRecordActionHeaders(map[string]string{"X-SDK-Example-Mode": "owned"}),
    }
    var result *image.ImageRecordActionResult
    if viaService {
        if action == "deactivate" {
            result, err = service.DeactivateImageRecord(ctx, input, options...)
        } else {
            result, err = service.ReactivateImageRecord(ctx, input, options...)
        }
    } else if action == "deactivate" {
        result, err = conn.DeactivateImageRecord(ctx, input, options...)
    } else {
        result, err = conn.ReactivateImageRecord(ctx, input, options...)
    }
    return printAction(result, err)
}

func printAction(result *image.ImageRecordActionResult, operationErr error) error {
    value := map[string]any{"partial": result != nil && operationErr != nil}
    if result != nil {
        value["acknowledgement"] = result.Acknowledgement
        if result.Record != nil {
            value["record"] = map[string]any{
                "resource": result.Record.Resource,
                "wire": result.Record.Wire,
                "earlier_receipt_status_code": result.Record.StatusCode,
                "earlier_receipt_body_bytes": len(result.Record.Envelope),
                "import_methods": result.Record.ImportMethods,
            }
        }
    }
    if operationErr != nil { value["error"] = operationErr.Error() }
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return errors.Join(operationErr, err) }
    fmt.Println(string(body))
    return operationErr
}
```

## References

- [DeactivateImage/ReactivateImage](mutations.md)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L301-L328)
- [Image Actions](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_actions.py#L47-L102)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
- [Install](../docs/install.md)
- [Image Records](image-records.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1117-L1137)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L602)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L262-L282)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L209-L285)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L785-L806)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1112-L1189)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L429-L437)
