# Nova Images Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
images = conn.compute.images(details=False, min_disk=4, max_items=100)
image = conn.compute.get_image(image_id)
found = conn.compute.find_image(image_id)
metadata = conn.compute.fetch_image_metadata(image_id)
conn.compute.set_image_metadata(image_id, owner="ci")
conn.compute.delete_image_metadata(image_id, keys=["owner"])
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "strings"
    "time"

    "github.com/JSYoo5B/go-openstacksdk/request"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/images"
)

func printImage(value *images.ImageRecord) {
    if value == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    id := flag.String("id", "", "Nova legacy image ID")
    owner := flag.String("set-owner", "", "metadata owner to POST")
    keys := flag.String("delete-keys", "", "comma-separated metadata keys")
    clear := flag.Bool("clear-metadata", false, "replace metadata with empty object")
    remove := flag.Bool("delete-image", false, "delete image strictly")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud), sdk.WithMicroversion(sdk.Compute, "2.35"))
    if err != nil { log.Fatal(err) }
    service, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Images
    for value, err := range api.ListRecords(ctx,
        request.Custom(func(c *images.ListOpts) error { v := false; c.Details = &v; return nil }), request.MaxItems(100)) {
        if err != nil { log.Fatal(err) }
        printImage(value)
    }
    if *id == "" { return }
    value, err := api.GetRecord(ctx, *id)
    if err != nil { log.Fatal(err) }
    printImage(value)
    value, err = api.FindRecord(ctx, *id, request.IgnoreMissing(false))
    if err != nil { log.Fatal(err) }
    printImage(value)
    value, err = api.FetchMetadata(ctx, *id)
    if err != nil { log.Fatal(err) }
    printImage(value)
    if *owner != "" {
        value, err = api.SetMetadata(ctx, *id, map[string]any{"owner": *owner})
        if err != nil { log.Fatal(err) }
        printImage(value)
    }
    if *keys != "" {
        if err := api.DeleteMetadata(ctx, *id,
            images.WithDeleteMetadataKeys(strings.Split(*keys, ",")...)); err != nil { log.Fatal(err) }
    }
    if *clear {
        if err := api.DeleteMetadata(ctx, *id); err != nil { log.Fatal(err) }
    }
    if *remove {
        if err := api.DeleteRecord(ctx, *id,
            request.IgnoreMissing(false)); err != nil { log.Fatal(err) }
    }
}
```

## References

- [Compute](https://docs.openstack.org/api-ref/compute/#images-deprecated)
- [Routes](https://github.com/openstack/nova/blob/master/nova/api/openstack/compute/routes.py)
- [API Version Request](https://github.com/openstack/nova/blob/master/nova/api/openstack/api_version_request.py)
- [Install](../../docs/install.md)
- [Image Records Test](images/image_records_test.go)
