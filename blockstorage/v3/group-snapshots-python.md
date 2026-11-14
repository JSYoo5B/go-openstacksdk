# Group Snapshots Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--action", choices=["get", "list", "create", "delete"], default="get")
parser.add_argument("--id", default="group-snapshot-id")
parser.add_argument("--group", default="group-id")
args = parser.parse_args()
conn = openstack.connect()
if args.action == "get":
    print(conn.block_storage.get_group_snapshot(args.id).to_dict())
elif args.action == "list":
    for item in conn.block_storage.group_snapshots(details=False, status="available", limit=10):
        print(item.to_dict())
elif args.action == "create":
    print(conn.block_storage.create_group_snapshot(group_id=args.group, name="sdk-example").to_dict())
else:
    print(conn.block_storage.delete_group_snapshot(args.id, ignore_missing=False))
```

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "time"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func show(value *groupsnapshots.GroupSnapshotRecord) {
    if value == nil { return }
    if value.Resource != nil {
        fmt.Printf("id=%s name=%s status=%s\n", value.Resource.Body["id"], value.Resource.Body["name"], value.Resource.Body["status"])
    }
    fmt.Printf("actual_http=%d discovery=%d actual_body=%q\n", value.StatusCode, len(value.Discovery), value.Envelope)
}

func main() {
    action := flag.String("action", "get", "get, list, create, or delete")
    id := flag.String("id", "group-snapshot-id", "existing group snapshot ID")
    group := flag.String("group", "group-id", "existing source group ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := groupsnapshots.New(cinder.RawClient())
    switch *action {
    case "get":
        value, err := api.GetRecord(ctx, *id)
        show(value)
        if err != nil { log.Fatal(err) }
    case "list":
        for value, err := range api.ListRecords(ctx,
            request.Custom(func(c *groupsnapshots.RecordListOpts) error { v := false; c.Details = &v; return nil }),
            request.Query("status", "available"),
            request.Limit(10)) {
            if err != nil { log.Fatal(err) }
            show(value)
        }
    case "create":
        value, err := api.CreateRecord(ctx, map[string]any{"group_id": *group, "name": "sdk-example"})
        show(value)
        if err != nil { log.Fatal(err) }
    case "delete":
        receipt, err := api.DeleteRecord(ctx, *id, request.IgnoreMissing(false))
        if receipt != nil { fmt.Printf("actual_http=%d ignored404=%t actual_body=%q\n", receipt.StatusCode, receipt.IgnoredMissing, receipt.Envelope) }
        if err != nil { log.Fatal(err) }
    default:
        log.Fatal("unknown action")
    }
}
```

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group_snapshot.py)
- [Policy](https://docs.openstack.org/cinder/latest/configuration/block-storage/policy.html)
- [V3](https://docs.openstack.org/api-ref/block-storage/v3/#group-snapshots-group-snapshots)
- [Python Records Test](groupsnapshots/python_records_test.go)
