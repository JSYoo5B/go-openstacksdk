# Group Find Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("identity", help="group ID or exact name")
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.38")
    parser.add_argument("--strict", action="store_true")
    parser.add_argument("--summary", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(
        cloud=args.cloud,
        block_storage_default_microversion=args.microversion,
    )
    value = conn.block_storage.find_group(
        args.identity,
        ignore_missing=not args.strict,
        details=not args.summary,
    )
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
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    identity := flag.String("identity", "", "group ID or exact name")
    version := flag.String("microversion", "3.38", "selected Cinder default")
    strict := flag.Bool("strict", false, "return an error when not found")
    summary := flag.Bool("summary", false, "use summary fallback list")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    value, err := service.Groups.FindRecord(ctx, *identity,
        request.IgnoreMissing(!*strict),
        request.Custom(func(c *groups.RecordFindOpts) error { v := !*summary; c.Details = &v; return nil }))
    if err != nil { log.Fatal(err) }
    if value == nil { fmt.Println("null"); return }
    raw, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(raw))
}
```

```go
value, err := service.Groups.FindRecord(ctx, "sample",
    request.Microversion("3.90"),
    request.Headers(map[string]string{"X-Trace": "group-find"}))
```

## References

- [find_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2126-L2155)
- [Group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py#L26-L66)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2489-L2582)
- [Groups Python](groups-python.md)
- [Install](../../docs/install.md)
- [Groups](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/groups.py)
