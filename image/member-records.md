# Member Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
image_id = "image-id"

for member in conn.image.members(image_id):
    print(member.to_dict())

member = conn.image.find_member("project-id", image_id, ignore_missing=False)
print(member.to_dict())
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
    imageID := flag.String("image", "", "literal parent image ID")
    find := flag.String("find", "", "optional member ID or inherited name")
    status := flag.String("status", "", "optional local member status filter")
    limit := flag.Int("limit", 0, "optional server page limit")
    maxItems := flag.Int("max-items", 0, "maximum consumed rows before filtering")
    flag.Parse()
    if *imageID == "" { log.Fatal("-image is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *find, *status, *limit, *maxItems); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func printRecord(record *image.ImageMemberRecord) error {
    value := map[string]any{"resource": record.Resource, "status_code": record.StatusCode}
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(body))
    return nil
}

func run(ctx context.Context, cloud, imageID, find, status string, limit, maxItems int) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    parent := resource.ID(imageID)
    options := []image.ImageMemberRecordListOption{
        image.WithImageMemberRecordListOpts(image.ImageMemberRecordListOpts{Limit: limit, MaxItems: maxItems}),
        request.Header("X-Request-Source", "member-records-example"),
    }
    if status != "" {
        options = append(options, image.WithImageMemberRecordListFilter("status", status))
    }
    for record, readErr := range service.ListImageMemberRecords(ctx, parent, options...) {
        if readErr != nil { return readErr }
        if err := printRecord(record); err != nil { return err }
    }
    if find != "" {
        record, err := service.FindImageMemberRecord(ctx, parent, find,
            request.IgnoreMissing(false))
        if err != nil { return err }
        return printRecord(record)
    }
    return nil
}
```

## References

- [Members](members.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1289-L1354)
- [Install](../docs/install.md)
- [Member](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/member.py#L16-L43)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2524-L2582)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L251-L278)
- [Member Records Test](member_records_test.go)
- [Member Record Find Test](member_record_find_test.go)
- [Falsey Next Test](../internal/rest/falsey_next_test.go)
- [Members Contracts Test](members_contracts_test.go)
