# Attachments Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
reserved = conn.block_storage.create_attachment(volume_id)
attached = conn.block_storage.create_attachment(
    volume_id, instance=server_id, connector={"host": "compute-1"}, mode="rw")
for attachment in conn.block_storage.attachments(volume_id=volume_id):
    print(attachment.id, attachment.attach_mode)
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    "github.com/JSYoo5B/go-openstacksdk/request"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    volume := flag.String("volume", "", "volume ID")
    instance := flag.String("instance", "", "optional attaching server ID")
    version := flag.String("microversion", "3.71", "Cinder microversion supported by the deployment")
    reserve := flag.Bool("reserve", false, "create a volume attachment reservation")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud), sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Attachments
    for value, err := range api.ListRecords(ctx,
        request.Query("volume_id", *volume),
        request.MaxItems(100)) {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(value.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    }
    if *reserve {
        options := []attachments.RecordCreateOption{}
        if *instance != "" {
            options = append(options, request.Field("instance", *instance))
        }
        value, err := api.CreateRecord(ctx, *volume, options...)
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(value.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    }
}
```

## References

- [Attachment proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1495-L1551)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/attachment.py)
- [Install](../../docs/install.md)
- [Python Records Contracts Test](attachments/python_records_contracts_test.go)
