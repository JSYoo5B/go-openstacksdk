# Cloud Router Interface Remove Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--router", required=True)
parser.add_argument("--subnet-json", default="null")
parser.add_argument("--port-json", default="null")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    result = conn.remove_router_interface(
        args.router,
        subnet_id=json.loads(args.subnet_json),
        port_id=json.loads(args.port_json),
    )
    print(json.dumps(result))
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

func run(ctx context.Context, cloud, router, subnetJSON, portJSON string) error {
 conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
 if err != nil { return err }
 service, err := conn.Network(ctx)
 if err != nil { return err }
 result, err := service.RemoveRouterInterface(ctx, router,
  network.WithRouterInterfaceRemoveSubnetID(json.RawMessage(subnetJSON)),
  network.WithRouterInterfaceRemovePortID(json.RawMessage(portJSON)))
 if result != nil {
  fmt.Fprintf(os.Stderr, "PUT status=%d request-id=%q\n",
   result.StatusCode, result.Header.Get("X-Request-Id"))
 }
 if err != nil { return err }
 _, err = fmt.Fprintln(os.Stdout, string(result.Value))
 return err
}

func main() {
 cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
 router := flag.String("router", "", "literal router ID, not a name lookup")
 subnet := flag.String("subnet-json", "null", "subnet_id as JSON")
 port := flag.String("port-json", "null", "port_id as JSON; a truthy port wins")
 flag.Parse()
 if *router == "" {
  fmt.Fprintln(os.Stderr, "router is required")
  os.Exit(2)
 }
 ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
 defer cancel()
 if err := run(ctx, *cloud, *router, *subnet, *port); err != nil {
  fmt.Fprintln(os.Stderr, err)
  os.Exit(1)
 }
}
```

## References

- [Install](../docs/install.md)
- [Router](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/router.py#L384-L398)
