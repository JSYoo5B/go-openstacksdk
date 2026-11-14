# Image Record Create

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
record = conn.image.create_image(
    name="ubuntu-example",
    filename="ubuntu.qcow2",
    visibility="private",
    allow_duplicates=True,
)

record = conn.image.create_image(
    name="binary-example",
    data=b"image bytes",
    validate_checksum=True,
)

record = conn.image.create_image(
    name="import-example",
    filename="ubuntu.qcow2",
    use_import=True,
    import_method="glance-direct",
)
```

```sh
go run . -cloud dev -scenario direct -file ubuntu.qcow2 -name ubuntu-example
go run . -cloud dev -scenario bytes -file ubuntu.qcow2 -name binary-example -service
go run . -cloud dev -scenario import -file ubuntu.qcow2 -name import-example
go run . -cloud task-cloud -scenario task -file ubuntu.qcow2 -name task-example -wait=true
go run . -cloud dev -scenario metadata -name metadata-example
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
)

type createImage func(context.Context, image.ImageRecordCreateRequest, ...image.ImageRecordCreateOption) (*image.ImageRecordCreateResult, error)

func main() {
    cloud := flag.String("cloud", "", "clouds.yaml entry; omission uses OS_CLOUD/OS_* settings")
    scenario := flag.String("scenario", "direct", "direct, bytes, import, task, or metadata")
    filename := flag.String("file", "", "local image file for direct/bytes/import/task")
    name := flag.String("name", "image-example", "image name")
    viaService := flag.Bool("service", false, "call image.Service instead of Connection")
    wait := flag.Bool("wait", true, "wait for Task completion in the task scenario")
    forceTask := flag.Bool("force-task-policy", false, "replace cloud image policy with explicit Task policy")
    flag.Parse()
    if *scenario != "metadata" && *filename == "" {
        log.Fatal("this scenario requires -file")
    }
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *scenario, *filename, *name, *viaService, *wait, *forceTask); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, scenario, filename, name string, viaService, wait, forceTask bool) error {
    options := []openstack.ConnectionOption{}
    if cloud != "" {
        options = append(options, openstack.WithCloud(cloud))
    }
    if forceTask {
        options = append(options, openstack.WithImageCreatePolicy(
            image.WithImageCreatePolicyTasks(true),
            image.WithImageCreatePolicyObjectStoreEnabled(true),
        ))
    }
    conn, err := openstack.Connect(ctx, options...)
    if err != nil { return err }
    var create createImage = conn.CreateImageRecord
    if viaService {
        service, serviceErr := conn.Image(ctx)
        if serviceErr != nil { return serviceErr }
        create = service.CreateImageRecord
    }
    var result *image.ImageRecordCreateResult
    switch scenario {
    case "direct":
        result, err = direct(ctx, create, filename, name)
    case "bytes":
        result, err = validatedBytes(ctx, create, filename, name)
    case "import":
        result, err = imported(ctx, create, filename, name)
    case "task":
        result, err = task(ctx, create, filename, name, wait)
    case "metadata":
        result, err = metadata(ctx, create, name)
    default:
        return fmt.Errorf("unknown scenario %q", scenario)
    }
    return show(result, err)
}

func direct(ctx context.Context, create createImage, filename, name string) (*image.ImageRecordCreateResult, error) {
    return create(ctx, image.ImageRecordCreateRequest{
        Name: name, Filename: filename, Attributes: map[string]any{"visibility": "private"},
    }, image.WithImageRecordCreateAllowDuplicates(true))
}

func validatedBytes(ctx context.Context, create createImage, filename, name string) (*image.ImageRecordCreateResult, error) {
    data, err := os.ReadFile(filename)
    if err != nil { return nil, err }
    return create(ctx, image.ImageRecordCreateRequest{
        Name: name, Data: image.ImageRecordCreateBytes(data),
    }, image.WithImageRecordCreateValidateChecksum(true))
}

func imported(ctx context.Context, create createImage, filename, name string) (*image.ImageRecordCreateResult, error) {
    return create(ctx, image.ImageRecordCreateRequest{Name: name, Filename: filename},
        image.WithImageRecordCreateAllowDuplicates(true),
        image.WithImageRecordCreateUseImport(true),
        image.WithImageRecordCreateImportOptions(image.WithImageRecordImportMethod("glance-direct")),
    )
}

func task(ctx context.Context, create createImage, filename, name string, wait bool) (*image.ImageRecordCreateResult, error) {
    return create(ctx, image.ImageRecordCreateRequest{Name: name, Filename: filename},
        image.WithImageRecordCreateAllowDuplicates(true),
        image.WithImageRecordCreateContainer("images"),
        image.WithImageRecordCreateWait(wait),
        image.WithImageRecordCreateTimeout(300),
    )
}

func metadata(ctx context.Context, create createImage, name string) (*image.ImageRecordCreateResult, error) {
    return create(ctx, image.ImageRecordCreateRequest{
        Name: name, Attributes: map[string]any{"min_disk": "0", "hw_qemu_guest_agent": false},
    }, image.WithImageRecordCreateAllowDuplicates(true),
        image.WithImageRecordCreateTags([]string{"sdk-example"}),
        image.WithImageRecordCreateMeta(map[string]any{"vendor_raw": map[string]any{"enabled": false}}),
    )
}

func recordSummary(record *image.ImageRecord) any {
    if record == nil { return nil }
    return map[string]any{
        "resource": record.Resource, "wire": record.Wire,
        "status_code": record.StatusCode, "header": record.Header,
        "envelope_bytes": []byte(record.Envelope), "import_methods": record.ImportMethods,
    }
}

func taskSummary(value *image.ImageRecordCreateTask) any {
    if value == nil { return nil }
    return map[string]any{
        "resource": value.Resource, "status_code": value.StatusCode,
        "header": value.Header, "envelope_bytes": []byte(value.Envelope),
    }
}

func show(result *image.ImageRecordCreateResult, operationErr error) error {
    value := map[string]any{"partial": result != nil && operationErr != nil}
    if operationErr != nil { value["error"] = operationErr.Error() }
    if result != nil {
        value["outcome"], value["reused"], value["warnings"] = result.Outcome, result.Reused, result.Warnings
        value["record"], value["updated"] = recordSummary(result.Record), recordSummary(result.Updated)
        value["task"], value["task_diagnostic"] = taskSummary(result.Task), taskSummary(result.TaskDiagnostic)
        value["checksum"], value["imported"] = result.Checksum, result.Imported
        for phase, receipt := range map[string]*image.ImageUploadResponse{
            "created": result.Created, "uploaded": result.Uploaded, "staged": result.Staged,
            "checksum_fetched": result.ChecksumFetched, "task_image_fetched": result.TaskImageFetched,
            "task_diagnostic_response": result.TaskDiagnosticResponse,
        } {
            if receipt != nil { value[phase] = receipt }
        }
        if result.TaskWait != nil {
            value["task_wait"] = map[string]any{
                "task": taskSummary(result.TaskWait.Task), "original_task": taskSummary(result.TaskWait.OriginalTask),
                "fetches": result.TaskWait.Fetches, "recreations": result.TaskWait.Recreations,
                "fetch_responses": result.TaskWait.FetchResponses, "recreation_responses": result.TaskWait.RecreationResponses,
            }
        }
        if result.Swift != nil {
            value["swift"] = map[string]any{
                "container_discovery": result.Swift.ContainerDiscovery,
                "container_created": result.Swift.ContainerCreated,
                "container_fetched": result.Swift.ContainerFetched,
                "object": result.Swift.Object, "cleanup": result.Swift.Cleanup,
            }
        }
        if result.Cleanup != nil { value["image_cleanup_ack"] = result.Cleanup.Acknowledgement }
    }
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return errors.Join(operationErr, err) }
    fmt.Println(string(body))
    return operationErr
}
```

## References

- [create_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L178-L436)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Implementation Plan](../docs/implementation-plan.md)
- [Upload](upload-image.md)
- [CreateAndImport](create-import.md)
- [UploadImageRecord](image-record-upload.md)
- [StageImageRecord](image-record-stage.md)
- [ImportImageRecord](image-record-import.md)
- [Install](../docs/install.md)
- [Source filename helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L50-L63)
- [Create](../objectstorage/v1/objects/create.md)
