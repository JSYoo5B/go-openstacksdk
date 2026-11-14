# Python Port Bindings

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--port", required=True)
parser.add_argument("--host", required=True)
parser.add_argument("--mode", choices=("create", "list", "activate", "delete"), required=True)
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "create":
        row = conn.network.create_port_binding(
            args.port, host=args.host, profile={"capabilities": ["switchdev"]})
        value = row.to_dict()
    elif args.mode == "list":
        value = [row.to_dict() for row in conn.network.port_bindings(
            args.port, host=args.host)]
    elif args.mode == "activate":
        row = conn.network.activate_port_binding(args.port, args.host)
        value = None if row is None else row.to_dict()
    else:
        conn.network.delete_port_binding(args.port, args.host)
        value = None
    print(json.dumps(value, indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/portbindings"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    port := flag.String("port", "", "Port ID")
    host := flag.String("host", "", "Binding host")
    mode := flag.String("mode", "", "create, list, activate or delete")
    flag.Parse()
    if *port == "" || *host == "" { log.Fatal("port and host are required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := network.PortBindings
    var value any
    switch *mode {
    case "create":
        value, err = api.Create(ctx, *port,
            portbindings.WithAttribute("host", *host),
            portbindings.WithAttribute("profile", map[string]any{"capabilities": []string{"switchdev"}}))
    case "list":
        value, err = api.All(ctx, *port, request.Query("host", *host))
    case "activate":
        value, err = api.Activate(ctx, *port, *host)
    case "delete":
        value, err = api.Delete(ctx, *port, *host)
    default:
        log.Fatal("mode must be create, list, activate or delete")
    }
    if err != nil { log.Fatal(err) }
    encoded, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(encoded))
}
```

## References

- [Install](../../../../docs/install.md)
