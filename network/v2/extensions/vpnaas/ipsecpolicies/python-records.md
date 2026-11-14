# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--name", default="")
    parser.add_argument("--units", default="")
    args = parser.parse_args()
    query = {}
    if args.name:
        query["name"] = args.name
    if args.units:
        query["units"] = args.units
    with openstack.connect(cloud=args.cloud) as conn:
        for row in conn.network.vpn_ipsec_policies(**query):
            print(json.dumps(row.to_dict(), ensure_ascii=False))

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
    "os"
    "time"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies"
)

func run(ctx context.Context, cloud, name, units string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.NetworkV2(ctx)
    if err != nil { return err }
    var options []ipsecpolicies.RecordListOption
    if name != "" { options = append(options, request.Query("name", name)) }
    if units != "" { options = append(options, request.Query("units", units)) }
    for record, err := range service.VPNIPsecPolicies.ListRecords(ctx, options...) {
        if err != nil { return err }
        if err := json.NewEncoder(os.Stdout).Encode(record.Resource.Body); err != nil { return err }
    }
    return nil
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    name := flag.String("name", "", "server-side policy name filter")
    units := flag.String("units", "", "local top-level units Body filter")
    timeout := flag.Duration("timeout", time.Minute, "whole operation deadline")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud, *name, *units); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

```sh
python main.py --cloud devstack --name policy
go run main.go --cloud devstack --name policy
```
