# Image Record Upload

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
with open("ubuntu.qcow2", "rb") as source:
    record = conn.image.upload_image(
        container_format="bare",
        disk_format="qcow2",
        data=source,
        name="ubuntu-example",
        visibility="private",
        filename="ubuntu.qcow2",
    )
print(record.id, record.status)
```

```sh
go run . -cloud dev -file ubuntu.qcow2 -name ubuntu-example
go run . -cloud dev -file ubuntu.qcow2 -name ubuntu-example -service -size 0
go run . -cloud dev -file ubuntu.qcow2 -name ubuntu-example -infer-size=false
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
    "path/filepath"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    filename := flag.String("file", "", "required caller-owned image file")
    name := flag.String("name", "ubuntu-example", "image metadata name")
    visibility := flag.String("visibility", "private", "server image visibility")
    container := flag.String("container-format", "bare", "explicit image container format")
    disk := flag.String("disk-format", "qcow2", "explicit image disk format")
    viaService := flag.Bool("service", false, "call image.Service instead of Connection")
    size := flag.Int64("size", 0, "explicit signed size hint; omission uses inference")
    inferSize := flag.Bool("infer-size", true, "infer total seekable length when size is omitted")
    flag.Parse()
    if *filename == "" { log.Fatal("-file is required by this example") }
    sizePresent := false
    flag.Visit(func(value *flag.Flag) {
        if value.Name == "size" { sizePresent = true }
    })
    options := []image.ImageRecordUploadOption{
        image.WithImageRecordUploadOpts(image.ImageRecordUploadOpts{
            Headers: map[string]string{"X-Request-Source": "image-record-upload-example"},
        }),
        image.WithImageRecordUploadContainerFormat(*container),
        image.WithImageRecordUploadDiskFormat(*disk),
        image.WithImageRecordUploadSizeInference(*inferSize),
    }
    if sizePresent { options = append(options, request.Custom(func(c *image.ImageRecordUploadOpts) error { v := *size; c.Size = &v; return nil })) }
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *filename, *name, *visibility, *viaService, options); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, filename, name, visibility string, viaService bool, options []image.ImageRecordUploadOption) (err error) {
    source, err := os.Open(filename)
    if err != nil { return err }
    defer func() { err = errors.Join(err, source.Close()) }()
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    input := image.ImageRecordUploadRequest{
        Data: source,
        Attributes: map[string]any{
            "name": name, "visibility": visibility,
            // This is metadata; the SDK does not open a filename attribute.
            "filename": filepath.Base(filename),
        },
    }
    var result *image.ImageRecordUploadResult
    if viaService {
        service, serviceErr := conn.Image(ctx)
        if serviceErr != nil { return serviceErr }
        result, err = service.UploadImageRecord(ctx, input, options...)
    } else {
        result, err = conn.UploadImageRecord(ctx, input, options...)
    }
    return printUpload(result, err)
}

func printUpload(result *image.ImageRecordUploadResult, operationErr error) error {
    value := map[string]any{"partial": result != nil && operationErr != nil}
    if result != nil {
        for name, receipt := range map[string]*image.ImageUploadResponse{
            "metadata": result.Metadata, "uploaded": result.Uploaded,
        } {
            if receipt != nil {
                value[name] = map[string]any{
                    // Response bytes may be empty or non-JSON; []byte is base64.
                    "body": receipt.Body, "header": receipt.Header,
                    "status_code": receipt.StatusCode,
                }
            }
        }
        if result.Record != nil {
            value["record"] = map[string]any{
                "resource": result.Record.Resource, "wire": result.Record.Wire,
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

- [Image Record Waits](image-record-waits.md)
- [Install](../docs/install.md)
- [Upload Image](upload-image.md)
- [Create Import](create-import.md)
- [Implementation Plan](../docs/implementation-plan.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
