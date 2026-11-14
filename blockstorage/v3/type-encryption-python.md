# Type Encryption Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
volume_type = conn.block_storage.find_type("fast")
qos = conn.block_storage.find_qos_spec("gold", project_id=project_id)

created = conn.block_storage.create_type_encryption(
    volume_type, provider="luks", control_location="front-end")
updated = conn.block_storage.update_type_encryption(
    volume_type=volume_type, key_size=512, cipher=None)
updated = conn.block_storage.update_type_encryption(
    encryption=updated, key_size=0)
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "strconv"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printResource(v *resource.RawResource) {
    if v == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(v, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    typeName := flag.String("type", "fast", "volume type ID or exact name")
    qosName := flag.String("qos", "", "optional QoS spec ID or exact name; stock policy needs admin")
    project := flag.String("project", "", "optional QoS project filter")
    create := flag.Bool("create-encryption", false, "create encryption on the found type")
    keySize := flag.String("key-size", "", "update encryption key size; 0 is explicit")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    volumeType, err := service.VolumeTypes.FindRecord(ctx, *typeName,
        request.IgnoreMissing(false))
    if err != nil { log.Fatal(err) }
    printResource(volumeType.Resource)
    var typeID string
    if err := json.Unmarshal(volumeType.Resource.Body["id"], &typeID); err != nil { log.Fatal(err) }
    if *qosName != "" {
        options := []qos.RecordFindOption{}
        if *project != "" { options = append(options, request.Query("project_id", *project)) }
        specification, err := service.QoS.FindRecord(ctx, *qosName, options...)
        if err != nil { log.Fatal(err) }
        if specification == nil { fmt.Println("QoS: null") } else { printResource(specification.Resource) }
    }
    if *create {
        encryption, err := service.VolumeTypes.CreateEncryptionWithAttributes(ctx, typeID,
            volumetypes.WithEncryptionAttribute("provider", "luks"),
            volumetypes.WithEncryptionAttribute("control_location", "front-end"))
        if err != nil { log.Fatal(err) }
        printResource(encryption.Resource)
    }
    if *keySize != "" {
        size, err := strconv.Atoi(*keySize)
        if err != nil { log.Fatal(err) }
        encryption, err := service.VolumeTypes.UpdateEncryptionFromType(ctx, typeID,
            volumetypes.WithEncryptionAttribute("key_size", size))
        if err != nil { log.Fatal(err) }
        printResource(encryption.Resource)
    }
}
```

## References

- [Policy](https://docs.openstack.org/cinder/2025.2/configuration/block-storage/policy.html)
- [Install](../../docs/install.md)
- [Python Encryption Contracts Test](volumetypes/python_encryption_contracts_test.go)
- [Python Find Records Test](volumetypes/python_find_records_test.go)
- [Python Find Records Test](qos/python_find_records_test.go)
