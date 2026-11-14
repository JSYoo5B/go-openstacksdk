# Image Record Delete

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
conn.image.delete_image("image-id", ignore_missing=False)

admin = openstack.connect(cloud="admin")
admin.image.delete_image("other-image-id", store="store-id", ignore_missing=False)
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
    imageID := flag.String("image-id", "", "required literal image ID")
    storeID := flag.String("store", "", "optional explicit store ID; omission deletes whole image")
    ignoreMissing := flag.Bool("ignore-missing", true, "ignore a clean final native404")
    fromRecord := flag.Bool("record", false, "explicitly fetch an SDK record before deletion")
    viaService := flag.Bool("service", false, "use image.Service instead of Connection for the DELETE")
    flag.Parse()
    storeSelected := false
    flag.Visit(func(value *flag.Flag) {
        if value.Name == "store" { storeSelected = true }
    })
    if *imageID == "" { log.Fatal("-image-id is required") }
    if storeSelected && *storeID == "" { log.Fatal("an explicitly selected store must have a nonempty ID") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *storeID, storeSelected,
        *ignoreMissing, *fromRecord, *viaService); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func printDelete(result *image.ImageRecordDeleteResult, operationErr error) error {
    value := map[string]any{"partial": result != nil && operationErr != nil}
    if result == nil && operationErr == nil { value["ignored_missing"] = true }
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

func run(ctx context.Context, cloud, imageID, storeID string,
    storeSelected, ignoreMissing, fromRecord, viaService bool) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    input := image.ImageRecordDeleteRequest{ID: imageID}
    if fromRecord {
        service, err := conn.Image(ctx)
        if err != nil { return err }
        record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
        if err != nil { return err }
        input = image.ImageRecordDeleteRequest{Record: record}
    }
    options := []image.ImageRecordDeleteOption{
        request.IgnoreMissing(ignoreMissing),
        request.Header("X-Request-Source", "image-record-delete-example"),
    }
    if storeSelected { options = append(options, image.WithImageRecordDeleteStoreID(storeID)) }
    var result *image.ImageRecordDeleteResult
    if viaService {
        service, err := conn.Image(ctx)
        if err != nil { return err }
        result, err = service.DeleteImageRecord(ctx, input, options...)
        return printDelete(result, err)
    }
    result, err = conn.DeleteImageRecord(ctx, input, options...)
    return printDelete(result, err)
}
```

## References

- [DeleteImage](delete.md)
- [native Delete](v2/images/delete.md)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L38-L51)
- [Router](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/router.py#L442-L446)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L769-L773)
- [Policy](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L224-L229)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L159-L172)
- [Install](../docs/install.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L970-L997)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2094-L2152)
- [Service Info](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/service_info.py#L46-L75)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L429-L437)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1394)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L563-L714)
