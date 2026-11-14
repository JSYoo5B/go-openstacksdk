# Consistency Group Snapshots Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("operation", choices=["get", "find", "list", "create", "delete"])
parser.add_argument("--identity")
parser.add_argument("--group")
parser.add_argument("--name")
parser.add_argument("--attrs", default="{}")
parser.add_argument("--query", default="{}")
parser.add_argument("--strict", action="store_true")
parser.add_argument("--summary", action="store_true")
args = parser.parse_args()
conn = openstack.connect()
if args.operation in {"get", "find", "delete"} and args.identity is None:
    parser.error("--identity is required")
if args.operation == "get":
    value = conn.block_storage.get_consistency_group_snapshot(args.identity)
    print(value.to_dict())
elif args.operation == "find":
    value = conn.block_storage.find_consistency_group_snapshot(
        args.identity, ignore_missing=not args.strict, details=not args.summary
    )
    print(None if value is None else value.to_dict())
elif args.operation == "list":
    for value in conn.block_storage.consistency_group_snapshots(
        details=not args.summary, **json.loads(args.query)
    ):
        print(value.to_dict())
elif args.operation == "create":
    attrs = json.loads(args.attrs)
    if args.group is not None:
        attrs["consistencygroup_id"] = args.group
    if args.name is not None:
        attrs["name"] = args.name
    print(conn.block_storage.create_consistency_group_snapshot(**attrs).to_dict())
else:
    conn.block_storage.delete_consistency_group_snapshot(
        args.identity, ignore_missing=not args.strict
    )
    print("delete returned")
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

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
    operation := flag.String("operation", "list", "get, find, list, create or delete")
    identity := flag.String("identity", "", "literal ID, or exact ID/name for find")
    group := flag.String("group", "", "source consistency group ID for create")
    name := flag.String("name", "", "create name, including an explicit empty string")
    attrsJSON := flag.String("attrs", "{}", "create JSON object")
    queryJSON := flag.String("query", "{}", "list JSON query object")
    strict := flag.Bool("strict", false, "report missing find/delete as an error")
    summary := flag.Bool("summary", false, "list summary, also on find fallback")
    flag.Parse()
    supplied := make(map[string]bool)
    flag.Visit(func(f *flag.Flag) { supplied[f.Name] = true })
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := consistencygroupsnapshots.New(cinder.RawClient())
    show := func(value *consistencygroupsnapshots.ConsistencyGroupSnapshotRecord, err error) {
        if value != nil {
            fmt.Printf("actual_http=%d actual_body=%q\n", value.StatusCode, value.Envelope)
        }
        if err != nil { log.Fatal(err) }
        if value == nil { fmt.Println("no matching consistency group snapshot"); return }
        raw, err := json.MarshalIndent(value.Resource.Body, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(raw))
    }
    switch *operation {
    case "get":
        show(api.GetRecord(ctx, *identity))
    case "find":
        show(api.FindRecord(ctx, *identity,
            request.IgnoreMissing(!*strict),
            request.Custom(func(c *consistencygroupsnapshots.RecordFindOpts) error { v := !*summary; c.Details = &v; return nil })))
    case "list":
        var query map[string]json.RawMessage
        if err := json.Unmarshal([]byte(*queryJSON), &query); err != nil { log.Fatal(err) }
        for value, err := range api.ListRecords(ctx,
            consistencygroupsnapshots.WithRecordListOpts(consistencygroupsnapshots.RecordListOpts{Query: query}),
            request.Custom(func(c *consistencygroupsnapshots.RecordListOpts) error { v := !*summary; c.Details = &v; return nil })) {
            show(value, err)
        }
    case "create":
        var attrs map[string]any
        decoder := json.NewDecoder(strings.NewReader(*attrsJSON))
        decoder.UseNumber()
        if err := decoder.Decode(&attrs); err != nil { log.Fatal(err) }
        if attrs == nil { attrs = make(map[string]any) }
        if supplied["group"] { attrs["consistencygroup_id"] = *group }
        if supplied["name"] { attrs["name"] = *name }
        show(api.CreateRecord(ctx, attrs))
    case "delete":
        receipt, err := api.DeleteRecord(ctx, *identity,
            request.IgnoreMissing(!*strict))
        if receipt != nil {
            fmt.Printf("actual_http=%d actual_body=%q ignored_missing=%t\n",
                receipt.StatusCode, receipt.Envelope, receipt.IgnoredMissing)
        }
        if err != nil { log.Fatal(err) }
    default:
        log.Fatal("unknown operation: ", *operation)
    }
}
```

```sh
python consistency_snapshots.py list --query '{"limit": 20, "status": "available"}'
go run ./cmd/consistency-snapshots -operation list -query '{"limit": 20, "status": "available"}'
python consistency_snapshots.py create --group source-consistency-group-id --name nightly
go run ./cmd/consistency-snapshots -operation create -group source-consistency-group-id -name nightly
python consistency_snapshots.py find --identity nightly --strict
go run ./cmd/consistency-snapshots -operation find -identity nightly -strict
```

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2503-L2640)
- [Consistency Group Snapshot](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/consistency_group_snapshot.py#L16-L49)
- [Cgsnapshots](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/contrib/cgsnapshots.py)
- [default standard_extensions](https://github.com/openstack/cinder/blob/27.0.0/cinder/common/config.py)
- [API](https://github.com/openstack/cinder/blob/27.0.0/cinder/group/api.py)
- [Group Snapshots](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/group_snapshots.py)
