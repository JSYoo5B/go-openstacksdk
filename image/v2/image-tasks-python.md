# Image Tasks Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument('--cloud', required=True)
parser.add_argument('--image', required=True)
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
for task in conn.image.image_tasks(args.image):
    print(json.dumps(task.to_dict(), ensure_ascii=False))
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/images"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
    cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud name")
    imageID := flag.String("image", "", "literal image ID")
    status := flag.String("status", "", "optional local status filter")
    limit := flag.Int("limit", -1, "server page hint; -1 omits it")
    cap := flag.Int("max-items", 0, "raw rows consumed before local filtering; 0 is unlimited")
    paginated := flag.Bool("paginated", true, "follow retained continuation")
    flag.Parse()
    if err := run(*cloud, *imageID, *status, *limit, *cap, *paginated); err != nil {
        log.Fatal(err)
    }
}

func run(cloud, imageID, status string, limit, maxItems int, paginated bool) error {
    if imageID == "" { return fmt.Errorf("-image is required") }
    if limit < -1 { return fmt.Errorf("-limit must be -1 or nonnegative") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.ImageV2(ctx)
    if err != nil { return err }
    options := []images.TaskRecordsOption{
        request.Headers(map[string]string{"X-Request-Source": "image-tasks-example"}),
        request.MaxItems(maxItems),
        request.Paginated(paginated),
    }
    if limit >= 0 { options = append(options, request.Limit(limit)) }
    if status != "" { options = append(options, request.Query("status", status)) }
    out := json.NewEncoder(os.Stdout)
    for record, readErr := range service.Images.TaskRecords(ctx, imageID, options...) {
        if readErr != nil { return readErr }
        if err := out.Encode(map[string]any{
            "resource": record.Resource.Body,
            "wire": record.Wire.Body,
            "envelope": record.Envelope,
            "headers": record.Header,
            "status_code": record.StatusCode,
            "microversion": record.Microversion,
            "discovery": record.Discovery,
        }); err != nil { return err }
    }
    return nil
}
```

## References

- [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1181-L1194)
- [`57f7dd9`](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L599-L614)
- [Glance Policy](https://docs.openstack.org/glance/2025.2/configuration/glance_policy.html)
- [`ImageTasks`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image_tasks.py#L22-L64)
- [`Resource.list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2359)
- [`image.Service.ImageTasks`/`AllImageTasks`](../image-tasks.md)
