# Group Snapshot Find Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("identity")
parser.add_argument("--strict", action="store_true")
parser.add_argument("--summary", action="store_true")
args = parser.parse_args()
conn = openstack.connect()
value = conn.block_storage.find_group_snapshot(
    args.identity,
    ignore_missing=not args.strict,
    details=not args.summary,
)
print(None if value is None else value.to_dict())
```

```go
package main

import (
    "context"
    "errors"
    "flag"
    "fmt"
    "log"
    "time"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    identity := flag.String("identity", "group-snapshot-id", "group snapshot ID or exact name")
    strict := flag.Bool("strict", false, "report completed search absence as an error")
    summary := flag.Bool("summary", false, "use summary list only on fallback")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := groupsnapshots.New(cinder.RawClient())
    value, err := api.FindRecord(ctx, *identity,
        request.IgnoreMissing(!*strict),
        request.Custom(func(c *groupsnapshots.RecordFindOpts) error { v := !*summary; c.Details = &v; return nil }))
    if err != nil {
        if value != nil {
            fmt.Printf("actual_http=%d actual_body=%q discovery=%d\n", value.StatusCode, value.Envelope, len(value.Discovery))
        }
        if errors.Is(err, resource.ErrNotFound) { log.Fatal("completed search found no group snapshot") }
        if errors.Is(err, resource.ErrAmbiguous) { log.Fatal("more than one ID/name match") }
        log.Fatal(err)
    }
    if value == nil { fmt.Println("no matching group snapshot"); return }
    fmt.Printf("id=%s name=%s status=%s\n", value.Resource.Body["id"], value.Resource.Body["name"], value.Resource.Body["status"])
    fmt.Printf("actual_http=%d actual_body=%q discovery=%d\n", value.StatusCode, value.Envelope, len(value.Discovery))
}
```

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2690-L2719)
- [Policy](https://docs.openstack.org/cinder/latest/configuration/block-storage/policy.html)
