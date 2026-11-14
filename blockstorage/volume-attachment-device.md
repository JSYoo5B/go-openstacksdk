# Volume Attachment Device

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "encoding/json"
    "errors"
    "fmt"

    volumesv2 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v2/volumes"
    volumesv3 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"

    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func text(value string) *string { return &value }

func device(value *string, err error) string {
    if err != nil { panic(err) }
    if value == nil { return "nil" }
    return fmt.Sprintf("%q", *value)
}

func rawDevice(value json.RawMessage, err error) string {
    if err != nil { panic(err) }
    if value == nil { return "nil" }
    return string(value)
}

func invalid(label string, err error) {
    var operation *resource.OperationError
    if !errors.As(err, &operation) || !errors.Is(err, resource.ErrInvalidOption) {
        panic(fmt.Sprintf("unexpected error: %v", err))
    }
    fmt.Printf("%s %s %s %t\n", label, operation.Operation,
        operation.Resource, errors.Is(err, resource.ErrInvalidOption))
}

func main() {
    owned := &blockstorage.VolumeInfo{Attachments: []*blockstorage.AttachVolumeRecord{
        {ServerID: text("server"), Device: text("")},
        {ServerID: text("server"), Device: text("/dev/vdb")},
    }}
    fmt.Println("owned first", device(blockstorage.GetVolumeAttachDevice(owned, "server")))
    fmt.Println("owned no match", device(blockstorage.GetVolumeAttachDevice(owned, "SERVER")))
    empty := &blockstorage.VolumeInfo{Attachments: []*blockstorage.AttachVolumeRecord{}}
    fmt.Println("owned empty", device(blockstorage.GetVolumeAttachDevice(empty, "server")))
    null := &blockstorage.VolumeInfo{Attachments: []*blockstorage.AttachVolumeRecord{
        {ServerID: text("server"), Device: nil}, // Manual Body:nil means explicit null.
    }}
    fmt.Println("owned null", device(blockstorage.GetVolumeAttachDevice(null, "server")))

    var current blockstorage.AttachVolumeObservation
    if err := json.Unmarshal([]byte(`{"attachments":[{"server_id":"server","device":"/dev/original"}]}`), &current); err != nil {
        panic(err)
    }
    current.Attachments[0].Device = text("/dev/current")
    fmt.Println("typed current", device(blockstorage.GetVolumeAttachDevice(&current, "server")))
    fmt.Println("raw original", rawDevice(blockstorage.GetVolumeAttachDeviceFields(current.Body, "server")))

    nativeV2 := &volumesv2.Volume{Attachments: []volumesv2.Attachment{
        {ServerID: "server", Device: ""},
    }}
    nativeV3 := &volumesv3.Volume{Attachments: []volumesv3.Attachment{
        {ServerID: "server", Device: "/dev/vdc"},
    }}
    fmt.Println("native v2", device(blockstorage.GetVolumeAttachDevice(nativeV2, "server")))
    fmt.Println("native v3", device(blockstorage.GetVolumeAttachDevice(nativeV3, "server")))

    raw := map[string]json.RawMessage{
        "attachments": json.RawMessage(`[{"server_id":"server","device":false}]`),
        "status": json.RawMessage("unrelated invalid JSON"), // This field is not consumed.
    }
    fmt.Println("raw false", rawDevice(blockstorage.GetVolumeAttachDeviceFields(raw, "server")))
    raw["attachments"] = json.RawMessage(`[{"server_id":"server","device":9007199254740993123456789}]`)
    fmt.Println("raw number", rawDevice(blockstorage.GetVolumeAttachDeviceFields(raw, "server")))
    raw["attachments"] = json.RawMessage(`[{"server_id":"server","device":{"path":"/dev/vdb"}}]`)
    fmt.Println("raw object", rawDevice(blockstorage.GetVolumeAttachDeviceFields(raw, "server")))
    raw["attachments"] = json.RawMessage(`[{"server_id":"server","device":null}]`)
    fmt.Println("raw null", rawDevice(blockstorage.GetVolumeAttachDeviceFields(raw, "server")))

    tail := []byte(`{"attachments":[{"server_id":"server","device":"/dev/vdb"},null]}`)
    var fields map[string]json.RawMessage
    if err := json.Unmarshal(tail, &fields); err != nil { panic(err) }
    fmt.Println("raw short circuit", rawDevice(blockstorage.GetVolumeAttachDeviceFields(fields, "server")))
    var whole blockstorage.VolumeInfo
    fmt.Println("whole decoder rejects tail", json.Unmarshal(tail, &whole) != nil)

    _, err := blockstorage.GetVolumeAttachDevice((*blockstorage.VolumeInfo)(nil), "server")
    invalid("typed nil error", err)
    fields["attachments"] = json.RawMessage(`[{"server_id":"server"}]`)
    _, err = blockstorage.GetVolumeAttachDeviceFields(fields, "server")
    invalid("raw missing error", err)
}
```

```python
volume = {"attachments": [
    {"server_id": "server", "device": ""},
    {"server_id": "server", "device": "/dev/vdb"},
]}
device = conn.get_volume_attach_device(volume, "server")  # "": first match
```

## References

- [Reference](README.md)
- [..](../README.md)
- [cloud helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L381-L396)
- [v2 Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/volume.py#L44)
- [v3 Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L54)
- [field conversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86-L111)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v2/volumes/results.go#L11-L31)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/results.go#L12-L33)
