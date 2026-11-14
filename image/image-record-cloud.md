# Image Record Cloud

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
visible = conn.list_images()
everything = conn.list_images(show_all=True)
ubuntu = conn.search_images("ubuntu-*", filters={"visibility": "public"})
image = conn.get_image("ubuntu-24.04")  # find_image, ignore_missing=True
by_id = conn.get_image_by_id("4c1e7d5a-0b6f-4a28-9a5e-7b0f3c1d2e9f")
print(len(visible), len(everything), [i.id for i in ubuntu])
if image is not None:
    print(image.to_dict(), by_id.status)
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
    "time"

    "github.com/JSYoo5B/go-openstacksdk/request"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    pattern := flag.String("pattern", "*", "image name or ID glob")
    showAll := flag.Bool("show-all", false, "include shared images that are not accepted")
    identity := flag.String("image", "", "optional image name or ID for Cloud get_image")
    id := flag.String("id", "", "optional image ID for strict get_image_by_id")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *pattern, *showAll, *identity, *id); err != nil {
        log.Fatal(err)
    }
}

func recordValue(record *image.ImageRecord) any {
    if record == nil {
        return nil
    }
    return map[string]any{
        "resource": record.Resource, "wire": record.Wire,
        "status_code": record.StatusCode, "import_methods": record.ImportMethods,
    }
}

func printQuery(operation string, result *image.ImageRecordQueryResult) error {
    if result == nil {
        return nil
    }
    data, err := json.MarshalIndent(map[string]any{
        "operation": operation, "value": result.Value,
        "image_count": len(result.Images), "inventory_count": len(result.Inventory),
    }, "", "  ")
    if err != nil {
        return err
    }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, pattern string, showAll bool, identity, id string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil {
        return err
    }
    service, err := conn.Image(ctx)
    if err != nil {
        return err
    }
    all, err := service.AllCloudImageRecords(ctx, request.Custom(func(c *image.ImageRecordQueryOpts) error { v := showAll; c.ShowAll = &v; return nil }))
    if printErr := printQuery("list_images", all); printErr != nil {
        return errors.Join(err, printErr)
    }
    if err != nil {
        return fmt.Errorf("list images: %w", err)
    }
    matched, err := service.SearchImageRecords(ctx, pattern)
    if printErr := printQuery("search_images", matched); printErr != nil {
        return errors.Join(err, printErr)
    }
    if err != nil {
        return fmt.Errorf("search images: %w", err)
    }
    if identity != "" {
        found, err := service.GetCloudImageRecord(ctx, identity)
        if err != nil {
            return fmt.Errorf("get image: %w", err)
        }
        data, err := json.MarshalIndent(map[string]any{"operation": "get_image", "value": found.Value, "record": recordValue(found.Image)}, "", "  ")
        if err != nil {
            return err
        }
        fmt.Println(string(data))
    }
    if id != "" {
        record, err := service.GetImageRecordByID(ctx, id)
        if err != nil {
            return fmt.Errorf("get image by id: %w", err)
        }
        data, err := json.MarshalIndent(map[string]any{"operation": "get_image_by_id", "record": recordValue(record)}, "", "  ")
        if err != nil {
            return err
        }
        fmt.Println(string(data))
    }
    return nil
}
```

## References

- [Image Records](image-records.md)
- [Install](../docs/install.md)
- [Image Record Download](image-record-download.md)
- [Image Record Properties](image-record-properties.md)
- [Image Record Delete](image-record-delete.md)
- [Image Record Create](image-record-create.md)
- [Cinder Proxy create_image](../blockstorage/volume-upload-image.md)
- [Cloud list_images·search_images·get_image·get_image_by_id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L84-L166)
- [get_image_exclude·get_image_name·get_image_id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L221-L246)
- [wait_for_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L248-L263)
- [download_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L168-L219)
- [update_image_properties](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L432-L445)
- [delete_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L265-L313)
- [create_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L315-L430)
- [iterate_timeout](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L101)
- [_filter_list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L43-L145)
