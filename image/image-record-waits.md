# Image Record Waits

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
seed = conn.image.get_image("image-id")

def show_progress(progress):
    print("progress:", progress)

ready = conn.image.wait_for_status(
    seed, "active", failures=["ERROR"], interval=2, wait=None,
    attribute="status", callback=show_progress,
)
print(ready.to_dict())

last = conn.image.wait_for_delete(
    ready, interval=2, wait=120, callback=show_progress,
)
print(last.to_dict())
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
    imageID := flag.String("image-id", "", "required image ID")
    target := flag.String("target", "active", "target status")
    waitDelete := flag.Bool("wait-delete", false, "also observe deletion; does not request deletion")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *target, *waitDelete); err != nil {
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

func waitResult(value *image.ImageRecord, waitErr error) error {
    if waitErr != nil {
        if value != nil {
            fmt.Fprintln(os.Stderr, "wait incomplete; latest projected observation follows")
            if err := printRecord(value); err != nil { return errors.Join(waitErr, err) }
        }
        return waitErr
    }
    return printRecord(value)
}

func run(ctx context.Context, cloud, imageID, target string, waitDelete bool) error {
    if imageID == "" { return fmt.Errorf("-image-id is required") }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    seed, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
    if err != nil { return err }

    callback := image.WithImageRecordWait(resource.WithProgressCallback(func(progress int) error {
        fmt.Fprintf(os.Stderr, "progress: %d\n", progress)
        return nil
    }))
    ready, err := service.WaitForImageRecordStatus(ctx, seed, target,
        image.WithImageRecordWaitTimeout(2*time.Minute),
        image.WithImageRecordWaitPollInterval(2*time.Second),
        callback,
        request.Header("X-Request-Source", "image-record-waits-example"))
    if err := waitResult(ready, err); err != nil { return err }

    if waitDelete {
        last, err := service.WaitForImageRecordDelete(ctx, ready,
            callback)
        if err := waitResult(last, err); err != nil { return err }
        fmt.Println("deletion observed; returned record retains the last successful receipt")
    }
    return nil
}
```

## References

- [ImageRecord](image-records.md)
- [Image proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L2299-L2363)
- [Resource wait](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2585-L2722)
- [fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1837)
- [timeout iterator](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L99)
- [Install](../docs/install.md)
- [Image Records](image-records.md)
- [Service Waits](../docs/service-waits.md)
- [Image Record Wait Test](image_record_wait_test.go)
- [Connection Image Record Waits Test](../connectiontest/connection_image_record_waits_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
