# Image Record Import

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
record = conn.image.get_image("image-id")
response = conn.image.import_image(
    record,
    method="web-download",
    uri="https://images.example.org/image.qcow2",
    stores=["fast", "archive"],
    all_stores_must_succeed=False,
)
print(response.status_code)
```

```sh
go run . -cloud dev -image-id ID
go run . -cloud dev -image-id ID -method web-download -uri https://images.example.org/image.qcow2 -stores fast,archive -all-stores-must-succeed=false
go run . -cloud dev -image-id ID -service -method glance-download -remote-region RegionTwo -remote-image-id REMOTE_ID -remote-interface public
go run . -cloud dev -image-id ID -method copy-image -store archive
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
    "strings"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required literal image ID")
    viaService := flag.Bool("service", false, "call image.Service instead of Connection")
    method := flag.String("method", "glance-direct", "server import method")
    uri := flag.String("uri", "", "web-download source URI")
    region := flag.String("remote-region", "", "remote Glance region")
    remoteID := flag.String("remote-image-id", "", "remote Glance image ID")
    remoteInterface := flag.String("remote-interface", "", "remote Glance service interface")
    store := flag.String("store", "", "singular store; explicit empty string is retained")
    stores := flag.String("stores", "", "comma-separated plural stores, preserving order and duplicates")
    allStores := flag.Bool("all-stores", false, "explicit all_stores value")
    allMust := flag.Bool("all-stores-must-succeed", false, "explicit all_stores_must_succeed value")
    flag.Parse()
    present := make(map[string]bool)
    flag.Visit(func(value *flag.Flag) { present[value.Name] = true })
    if *imageID == "" { log.Fatal("-image-id is required") }
    options := []image.ImageRecordImportOption{
        image.WithImageRecordImportOpts(image.ImageRecordImportOpts{
            Headers: map[string]string{"X-Request-Source": "image-record-import-example"},
        }),
        image.WithImageRecordImportMethod(*method),
        image.WithImageRecordImportURI(*uri),
        image.WithImageRecordImportRemoteRegion(*region),
        image.WithImageRecordImportRemoteImageID(*remoteID),
        image.WithImageRecordImportRemoteServiceInterface(*remoteInterface),
    }
    if present["store"] {
        options = append(options, image.WithImageRecordImportStore(image.ImageRecordImportStore{ID: *store}))
    }
    if present["stores"] {
        selected := make([]image.ImageRecordImportStore, 0)
        for _, id := range strings.Split(*stores, ",") { selected = append(selected, image.ImageRecordImportStore{ID: id}) }
        options = append(options, image.WithImageRecordImportStores(selected...))
    }
    if present["all-stores"] { options = append(options, image.WithImageRecordImportAllStores(*allStores)) }
    if present["all-stores-must-succeed"] { options = append(options, image.WithImageRecordImportAllStoresMustSucceed(*allMust)) }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *viaService, options); err != nil {
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

func run(ctx context.Context, cloud, imageID string, viaService bool, options []image.ImageRecordImportOption) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
    if err != nil { return err }
    input := image.ImageRecordImportRequest{Record: record}
    var result *image.ImageRecordImportResult
    if viaService {
        result, err = service.ImportImageRecord(ctx, input, options...)
    } else {
        result, err = conn.ImportImageRecord(ctx, input, options...)
    }
    return printImport(result, err)
}

func printImport(result *image.ImageRecordImportResult, operationErr error) error {
    value := map[string]any{"partial": result != nil && operationErr != nil}
    if result != nil {
        if ack := result.Acknowledgement; ack != nil {
            // Opaque response bytes may not be JSON. []byte renders as base64.
            value["acknowledgement"] = map[string]any{
                "image_id": ack.ImageID, "body": []byte(ack.Body),
                "header": ack.Header, "status_code": ack.StatusCode,
            }
        }
        if result.Record != nil {
            value["record"] = map[string]any{
                "resource": result.Record.Resource,
                "wire": result.Record.Wire,
                "status_code": result.Record.StatusCode,
                "envelope_bytes": len(result.Record.Envelope),
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

- [Image Record Stage](image-record-stage.md)
- [Install](../docs/install.md)
- [`API.ImageImport.ImportImage/ImportKnownImage`](v2/imageimport/README.md)
- [CreateAndImport](create-import.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L440-L536)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L368-L436)
- [Image Records](image-records.md)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L317-L390)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L80-L93)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L330-L343)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
