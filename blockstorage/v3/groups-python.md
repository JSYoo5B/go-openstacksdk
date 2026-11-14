# Groups Python

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
    parser.add_argument("--microversion", default="3.38")
    parser.add_argument("--action", choices=["list", "get", "create", "clone", "update", "delete"], default="list")
    parser.add_argument("--group", default="")
    parser.add_argument("--name", default="sample")
    parser.add_argument("--group-type", default="")
    parser.add_argument("--volume-types", default="")
    parser.add_argument("--source-group", default="")
    parser.add_argument("--source-snapshot", default="")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud,
        block_storage_default_microversion=args.microversion)
    api = conn.block_storage
    if args.action == "list":
        for value in api.groups():
            print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))
        return
    if args.action == "get":
        value = api.get_group(args.group, list_volume=True)
    elif args.action == "create":
        value = api.create_group(group_type=args.group_type,
            volume_types=args.volume_types.split(","), name=args.name)
    elif args.action == "clone":
        # Both IDs are required parameters of the pinned classmethod.
        value = api.create_group_from_source(
            group_snapshot_id=args.source_snapshot or None,
            source_group_id=args.source_group or None,
            name=args.name)
    elif args.action == "update":
        value = api.update_group(args.group, name=args.name)
    else:
        api.delete_group(args.group, delete_volumes=False, ignore_missing=False)
        return
    print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))

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
    "strings"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printGroup(value *resource.RawResource) {
    raw, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(raw))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    version := flag.String("microversion", "3.38", "Cinder selected version")
    action := flag.String("action", "list", "list/get/create/clone/update/delete")
    id := flag.String("group", "", "group ID")
    name := flag.String("name", "sample", "name for create/clone/update")
    groupType := flag.String("group-type", "", "existing group-type name or ID")
    volumeTypes := flag.String("volume-types", "", "comma-separated volume-type IDs")
    sourceGroup := flag.String("source-group", "", "source group ID for clone")
    sourceSnapshot := flag.String("source-snapshot", "", "source group-snapshot ID for clone")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := groups.New(service.RawClient())
    var value *groups.GroupRecord
    switch *action {
    case "list":
        for row, err := range api.ListRecords(ctx) {
            if err != nil { log.Fatal(err) }
            printGroup(row.Resource)
        }
        return
    case "get":
        value, err = api.GetRecord(ctx, *id, request.Custom(func(c *groups.RecordGetOpts) error { v := true; c.ListVolume = &v; return nil }))
    case "create":
        value, err = api.CreateRecord(ctx,
            request.Field("group_type", *groupType),
            request.Field("volume_types", strings.Split(*volumeTypes, ",")),
            request.Field("name", *name))
    case "clone":
        var snapshot, source any
        if *sourceSnapshot != "" { snapshot = *sourceSnapshot }
        if *sourceGroup != "" { source = *sourceGroup }
        value, err = api.CreateFromSourceRecord(ctx,
            groups.WithRecordSourceGroupSnapshotID(snapshot),
            groups.WithRecordSourceSourceGroupID(source),
            groups.WithRecordSourceName(*name))
    case "update":
        value, err = api.UpdateRecord(ctx, *id, request.Field("name", *name))
    case "delete":
        err = api.DeleteRecord(ctx, *id, request.IgnoreMissing(false))
        if err != nil { log.Fatal(err) }
        return
    default:
        log.Fatal("unknown action")
    }
    if err != nil { log.Fatal(err) }
    printGroup(value.Resource)
}
```

## References

- [Group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2071-L2250)
- [Install](../../docs/install.md)
- [Group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py#L65)
- [Groups](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/groups.py)
- [group:delete](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/group_actions.py)
- [API](https://github.com/openstack/cinder/blob/27.0.0/cinder/group/api.py#L183-L200)
- [Python Records Test](groups/python_records_test.go)
- [Python Policy Test](groups/python_policy_test.go)
