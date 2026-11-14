# Consistency Groups Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default=None)
    parser.add_argument("--action", choices=["list", "get", "find", "create", "clone", "update", "delete"], default="list")
    parser.add_argument("--group", default="")
    parser.add_argument("--name", default="sample")
    parser.add_argument("--volume-types", default="")
    parser.add_argument("--source-group", default="")
    parser.add_argument("--source-snapshot", default="")
    parser.add_argument("--summary", action="store_true")
    parser.add_argument("--strict", action="store_true")
    parser.add_argument("--force", action="store_true")
    args = parser.parse_args()
    policy = {}
    if args.microversion is not None:
        policy["block_storage_default_microversion"] = args.microversion
    api = openstack.connect(cloud=args.cloud, **policy).block_storage
    if args.action == "list":
        for value in api.consistency_groups(details=not args.summary):
            print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))
        return
    if args.action == "get":
        value = api.get_consistency_group(args.group)
    elif args.action == "find":
        value = api.find_consistency_group(args.group,
            ignore_missing=not args.strict, details=not args.summary)
    elif args.action == "create":
        value = api.create_consistency_group(volume_types=args.volume_types,
            name=args.name)
    elif args.action == "clone":
        value = api.create_consistency_group_from_source(
            consistency_group=args.source_group or None,
            consistency_group_snapshot=args.source_snapshot or None,
            name=args.name)
    elif args.action == "update":
        value = api.update_consistency_group(args.group, name=args.name)
    else:
        # This exact pinned helper sends consistency_group rather than the
        # stock controller's consistencygroup and can receive HTTP400.
        api.delete_consistency_group(args.group, force=args.force)
        return
    print(json.dumps(None if value is None else value.to_dict(),
                     indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
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

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func show(value *consistencygroups.ConsistencyGroupRecord) {
    if value == nil { fmt.Println("null"); return }
    raw, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(raw))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    version := flag.String("microversion", "", "optional selected Cinder version")
    action := flag.String("action", "list", "list/get/find/create/clone/update/delete")
    id := flag.String("group", "", "literal ID, or exact identity for find")
    name := flag.String("name", "sample", "name for create/clone/update")
    volumeTypes := flag.String("volume-types", "", "comma-separated legacy volume types")
    sourceGroup := flag.String("source-group", "", "source consistency-group ID")
    sourceSnapshot := flag.String("source-snapshot", "", "source consistency-group-snapshot ID")
    summary := flag.Bool("summary", false, "summary list/find")
    strict := flag.Bool("strict", false, "strict find missing")
    force := flag.Bool("force", false, "also request deletion of member volumes")
    sourceWire := flag.Bool("pinned-source-wire", false, "replay known Source delete spelling")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    policy := []sdk.ConnectionOption{sdk.WithCloud(*cloud)}
    if *version != "" { policy = append(policy, sdk.WithMicroversion(sdk.BlockStorage, *version)) }
    conn, err := sdk.Connect(ctx, policy...)
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := consistencygroups.New(service.RawClient())
    var value *consistencygroups.ConsistencyGroupRecord
    switch *action {
    case "list":
        for row, err := range api.ListRecords(ctx, request.Custom(func(c *consistencygroups.RecordListOpts) error { v := !*summary; c.Details = &v; return nil })) {
            if err != nil { log.Fatal(err) }
            show(row)
        }
        return
    case "get":
        value, err = api.GetRecord(ctx, *id)
    case "find":
        value, err = api.FindRecord(ctx, *id,
            request.IgnoreMissing(!*strict),
            request.Custom(func(c *consistencygroups.RecordFindOpts) error { v := !*summary; c.Details = &v; return nil }))
    case "create":
        value, err = api.CreateRecord(ctx,
            request.Field("volume_types", *volumeTypes),
            request.Field("name", *name))
    case "clone":
        value, err = api.CreateFromSourceRecord(ctx,
            request.Custom(func(c *consistencygroups.RecordSourceOpts) error { v := *sourceGroup; c.ConsistencyGroupID = &v; return nil }),
            request.Custom(func(c *consistencygroups.RecordSourceOpts) error { v := *sourceSnapshot; c.ConsistencyGroupSnapshotID = &v; return nil }),
            request.Custom(func(c *consistencygroups.RecordSourceOpts) error { v := *name; c.Name = &v; return nil }))
    case "update":
        value, err = api.UpdateRecord(ctx, *id, request.Field("name", *name))
    case "delete":
        err = api.DeleteRecord(ctx, *id, request.Custom(func(c *consistencygroups.RecordDeleteOpts) error { v := *force; c.Force = &v; return nil }),
            request.Custom(func(c *consistencygroups.RecordDeleteOpts) error { c.PinnedSourceWire = *sourceWire; return nil }))
        if err != nil { log.Fatal(err) }
        return
    default:
        log.Fatal("unknown action")
    }
    if err != nil { log.Fatal(err) }
    show(value)
}
```

## References

- [ConsistencyGroup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/consistency_group.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2322-L2504)
- [Groups Python](groups-python.md)
- [Config](https://github.com/openstack/cinder/blob/27.0.0/cinder/common/config.py#L59-L61)
- [v3 router](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/v3/router.py#L165-L171)
- [Install](../../docs/install.md)
- [stock v3 update](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/v3/consistencygroups.py#L34-L100)
- [stock controller](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/contrib/consistencygroups.py#L61-L76)
- [view builder](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/views/consistencygroups.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1418-L1445)
- [group policy](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/groups.py)
- [delete policy](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/group_actions.py#L58-L68)
- [Python Record Contracts Test](consistencygroups/python_record_contracts_test.go)
- [Python Find Contracts Test](consistencygroups/python_find_contracts_test.go)
