# Metadata Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--volume", default="")
parser.add_argument("--snapshot", default="")
parser.add_argument("--action", choices=["fetch", "get", "set", "delete", "clear", "noop"], default="fetch")
parser.add_argument("--metadata", default='{"sdk-example":"python"}')
parser.add_argument("--key", default="sdk-example")
args = parser.parse_args()
if not args.volume and not args.snapshot:
    parser.error("provide --volume or --snapshot")

conn = openstack.connect()
for kind, identity in [("volume", args.volume), ("snapshot", args.snapshot)]:
    if not identity:
        continue
    proxy = conn.block_storage
    if args.action in ("fetch", "get"):
        record = getattr(proxy, f"{args.action}_{kind}_metadata")(identity)
        print(kind, "metadata", record.metadata)
    elif args.action == "set":
        record = getattr(proxy, f"set_{kind}_metadata")(identity, **json.loads(args.metadata))
        print(kind, "cached request delta", record.metadata)
    else:
        keys = None if args.action == "clear" else ([] if args.action == "noop" else [args.key])
        result = getattr(proxy, f"delete_{kind}_metadata")(identity, keys=keys)
        print(kind, "delete return", result)  # Source returns None.
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func show(kind string, declared *resource.RawResource, envelope json.RawMessage, status int) {
    metadata := "<no declared view>"
    if declared != nil { metadata = string(declared.Body["metadata"]) }
    fmt.Printf("%s metadata=%s status=%d actual_body=%q\n", kind, metadata, status, envelope)
}

func main() {
    volumeID := flag.String("volume", "", "existing volume ID")
    snapshotID := flag.String("snapshot", "", "existing snapshot ID")
    action := flag.String("action", "fetch", "fetch, get, set, delete, clear, or noop")
    metadataJSON := flag.String("metadata", `{"sdk-example":"go"}`, "metadata JSON object")
    key := flag.String("key", "sdk-example", "one metadata key to delete")
    flag.Parse()
    if *volumeID == "" && *snapshotID == "" { log.Fatal("provide -volume or -snapshot") }
    switch *action { case "fetch", "get", "set", "delete", "clear", "noop": default: log.Fatal("unknown action") }
    // RawMessage values preserve JSON null/false/zero and large integers.
    var members map[string]json.RawMessage
    if err := json.Unmarshal([]byte(*metadataJSON), &members); err != nil { log.Fatal(err) }
    values := make(map[string]any, len(members))
    for name, raw := range members { values[name] = raw }
    var keys []string
    if *action == "noop" { keys = []string{} }
    if *action == "delete" { keys = []string{*key} }
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }

    if *volumeID != "" {
        var record *volumes.VolumeResourceRecord
        switch *action {
        case "fetch": record, err = service.Volumes.FetchMetadataRecord(ctx, *volumeID)
        case "get": record, err = service.Volumes.GetMetadataRecord(ctx, *volumeID)
        case "set": record, err = service.Volumes.SetMetadataRecord(ctx, *volumeID, values,
            request.Header("X-Client-Request-Id", "metadata-volume"))
        default:
            var deleted *volumes.MetadataRecordDeletion
            deleted, err = service.Volumes.DeleteMetadataRecord(ctx, *volumeID, keys)
            if deleted != nil {
                record = deleted.Record
                fmt.Printf("volume clear=%t completed_cached_keys=%v accepted_receipts=%d\n", deleted.Cleared, deleted.DeletedKeys, len(deleted.Responses))
            }
        }
        if record != nil { show("volume", record.Resource, record.Envelope, record.StatusCode) }
        if err != nil { log.Fatal(err) }
    }
    if *snapshotID != "" {
        var record *snapshots.SnapshotRecord
        switch *action {
        case "fetch": record, err = service.Snapshots.FetchMetadataRecord(ctx, *snapshotID)
        case "get": record, err = service.Snapshots.GetMetadataRecord(ctx, *snapshotID)
        case "set": record, err = service.Snapshots.SetMetadataRecord(ctx, *snapshotID, values,
            request.Header("X-Client-Request-Id", "metadata-snapshot"))
        default:
            var deleted *snapshots.MetadataRecordDeletion
            deleted, err = service.Snapshots.DeleteMetadataRecord(ctx, *snapshotID, keys)
            if deleted != nil {
                record = deleted.Record
                fmt.Printf("snapshot clear=%t completed_cached_keys=%v accepted_receipts=%d\n", deleted.Cleared, deleted.DeletedKeys, len(deleted.Responses))
            }
        }
        if record != nil { show("snapshot", record.Resource, record.Envelope, record.StatusCode) }
        if err != nil { log.Fatal(err) }
    }
}
```

## References

- [Cinder API](https://docs.openstack.org/api-ref/block-storage/v3/)
- [Policy](https://docs.openstack.org/cinder/latest/configuration/block-storage/policy.html)
- [Python Metadata Test](volumes/python_metadata_test.go)
- [Python Metadata Test](snapshots/python_metadata_test.go)
