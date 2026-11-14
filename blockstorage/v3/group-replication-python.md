# Group Replication Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("group", help="literal group ID")
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.38")
    parser.add_argument("--action", choices=["enable", "disable", "failover"],
                        default="enable")
    parser.add_argument("--allow-attached-volume", action="store_true")
    parser.add_argument("--secondary-backend-id", default=None)
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud,
        block_storage_default_microversion=args.microversion)
    if args.action == "enable":
        conn.block_storage.enable_group_replication(args.group)
    elif args.action == "disable":
        conn.block_storage.disable_group_replication(args.group)
    else:
        conn.block_storage.failover_group_replication(
            args.group,
            allowed_attached_volume=args.allow_attached_volume,
            secondary_backend_id=args.secondary_backend_id,
        )
    print("action call completed without exception")

if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "time"

    "github.com/JSYoo5B/go-openstacksdk/request"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    id := flag.String("group", "", "literal group ID")
    action := flag.String("action", "enable", "enable/disable/failover")
    version := flag.String("microversion", "3.38", "selected Cinder version")
    allow := flag.Bool("allow-attached-volume", false, "allow attached volumes")
    backend := flag.String("secondary-backend-id", "", "optional backend ID")
    sourceWire := flag.Bool("pinned-source-wire", false, "replay the fixed Source body")
    flag.Parse()
    backendPresent := false
    flag.Visit(func(f *flag.Flag) { if f.Name == "secondary-backend-id" { backendPresent = true } })
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    var value *groups.ReplicationRecordResult
    switch *action {
    case "enable":
        value, err = service.Groups.EnableReplicationRecord(ctx, *id,
            request.Custom(func(c *groups.RecordReplicationOpts) error { c.PinnedSourceWire = *sourceWire; return nil }))
    case "disable":
        value, err = service.Groups.DisableReplicationRecord(ctx, *id,
            request.Custom(func(c *groups.RecordReplicationOpts) error { c.PinnedSourceWire = *sourceWire; return nil }))
    case "failover":
        options := []groups.RecordFailoverOption{
            request.Custom(func(c *groups.RecordFailoverOpts) error { v := *allow; c.AllowedAttachedVolume = &v; return nil }),
            request.Custom(func(c *groups.RecordFailoverOpts) error { c.PinnedSourceWire = *sourceWire; return nil }),
        }
        if backendPresent { options = append(options, request.Custom(func(c *groups.RecordFailoverOpts) error { v := *backend; c.SecondaryBackendID = &v; return nil })) }
        value, err = service.Groups.FailoverReplicationRecord(ctx, *id, options...)
    default:
        log.Fatal("unknown action")
    }
    if value != nil {
        fmt.Printf("reference=%q actual_http=%d actual_body=%q discovery=%d\n",
            value.Reference, value.StatusCode, value.Envelope, len(value.Discovery))
    }
    if err != nil { log.Fatal(err) }
}
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2275-L2319)
- [Group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py#L101-L140)
- [Group leaf](groups-python.md)
- [V3](https://docs.openstack.org/api-ref/block-storage/v3/#group-replication-groups-action)
- [Group Actions](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/group_actions.py#L81-L116)
- [Policy](https://docs.openstack.org/cinder/2025.2/configuration/block-storage/policy.html)
- [Groups](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/schemas/groups.py#L132-L173)
- [controller](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/v3/groups.py#L339-L404)
- [Install](../../docs/install.md)
