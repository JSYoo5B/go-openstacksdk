# Image Record Stage

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
record = conn.image.get_image("image-id")
record = conn.image.stage_image(record, filename="image.qcow2")

# with open("image.qcow2", "rb") as data:
#     record = conn.image.stage_image(record, data=data, size=0)
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
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required literal image ID")
    filename := flag.String("filename", "", "required binary image filename")
    borrow := flag.Bool("borrow", false, "open a caller-owned file and lend its reader")
    viaService := flag.Bool("service", false, "call image.Service instead of Connection")
    infer := flag.Bool("infer-size", true, "infer total seekable length and restore cursor")
    size := flag.Int64("size", 0, "explicit signed X-OpenStack-Image-Size")
    flag.Parse()
    explicitSize := false
    flag.Visit(func(value *flag.Flag) { if value.Name == "size" { explicitSize = true } })
    if *imageID == "" || *filename == "" { log.Fatal("-image-id and -filename are required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *filename, *borrow, *viaService, *infer, *size, explicitSize); err != nil {
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

func run(ctx context.Context, cloud, imageID, filename string, borrow, viaService, infer bool, size int64, explicitSize bool) (err error) {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
    if err != nil { return err }
    input := image.ImageRecordStageRequest{Record: record, Filename: filename}
    mode := "filename"
    if borrow {
        data, openErr := os.Open(filename)
        if openErr != nil { return openErr }
        defer func() { err = errors.Join(err, data.Close()) }()
        input = image.ImageRecordStageRequest{Record: record, Data: data}
        mode = "borrowed-reader"
    }
    options := []image.ImageRecordStageOption{
        image.WithImageRecordStageOpts(image.ImageRecordStageOpts{
            Headers: map[string]string{"X-Request-Source": "image-record-stage-example"},
        }),
        request.Header("X-SDK-Example-Mode", mode),
        image.WithImageRecordStageHeaders(map[string]string{"X-SDK-Example-Operation": "stage"}),
        image.WithImageRecordStageSizeInference(infer),
    }
    if explicitSize {
        options = append(options, request.Custom(func(c *image.ImageRecordStageOpts) error { v := size; c.Size = &v; return nil }))
    } else {
        options = append(options, request.Custom(func(c *image.ImageRecordStageOpts) error { c.Size = nil; return nil }))
    }
    var result *image.ImageRecordStageResult
    if viaService {
        result, err = service.StageImageRecord(ctx, input, options...)
    } else {
        result, err = conn.StageImageRecord(ctx, input, options...)
    }
    return printStage(result, err)
}

func printStage(result *image.ImageRecordStageResult, operationErr error) error {
    value := map[string]any{"partial": result != nil && operationErr != nil}
    if result != nil {
        value["staged"] = result.Staged
        value["metadata"] = result.Metadata
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

- [`API.ImageData.StageImage/StageKnownImage`](v2/imagedata/README.md)
- [Install](../docs/install.md)
- [Image Records](image-records.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L538-L581)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L325-L366)
- [Utils](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L462-L491)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1403)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1837)
- [Image Data](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_data.py#L330-L360)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L80-L93)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
