# Cloud Subnet Lists Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=["list", "search"], default="list")
parser.add_argument("--name", default="")
parser.add_argument("--filters", default="{}")
args = parser.parse_args()
filters = json.loads(args.filters)
with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "list":
        values = conn.list_subnets(filters=filters)
    else:
        values = conn.search_subnets(args.name, filters=filters)
    print(json.dumps([
        value.to_dict() if hasattr(value, "to_dict") else value
        for value in values
    ], indent=2))
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
 "github.com/JSYoo5B/go-openstacksdk/network"
)

func run(ctx context.Context, cloud, mode, name, filters string) error {
 conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
 if err != nil { return err }
 options := []network.SubnetReadOption{network.WithSubnetReadFiltersJSON(json.RawMessage(filters))}
 var result *network.SubnetRecords
 switch mode {
 case "list": result, err = conn.ListSubnetRecords(ctx, options...)
 case "search": result, err = conn.SearchSubnetRecords(ctx, name, options...)
 default: return fmt.Errorf("mode must be list or search")
 }
 if result != nil && result.LastResponse != nil {
  fmt.Fprintf(os.Stderr, "last observed resource/error status=%d request-id=%q partial-resources=%d\n",
   result.LastResponse.StatusCode, result.LastResponse.Header.Get("X-Request-Id"), len(result.Inventory))
 }
 if err != nil { return err }
 encoded, err := json.MarshalIndent(result.Value, "", "  ")
 if err != nil { return err }
 _, err = fmt.Fprintln(os.Stdout, string(encoded))
 return err
}

func main() {
 cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
 mode := flag.String("mode", "list", "list or search")
 name := flag.String("name", "", "literal search name, including UUID or glob characters")
 filters := flag.String("filters", `{}`, "Source cloud filter JSON")
 flag.Parse()
 ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
 defer cancel()
 if err := run(ctx, *cloud, *mode, *name, *filters); err != nil {
  fmt.Fprintln(os.Stderr, err)
  os.Exit(1)
 }
}
```

## References

- [Install](../docs/install.md)
- [Subnet](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/subnet.py#L121-L176)
