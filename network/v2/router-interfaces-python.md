# Router Interfaces Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
attached = conn.network.add_interface_to_router("ROUTER_ID", subnet="SUBNET_ID")
detached = conn.network.remove_interface_from_router("ROUTER_ID", port="PORT_ID")

added = conn.network.update_router("ROUTER_ID", external_gateway_info={
    "network_id": "EXTERNAL_NETWORK_ID"
})
removed = conn.network.update_router("ROUTER_ID", external_gateway_info=None)

# conn.network.add_gateway_to_router("ROUTER_ID", network_id="EXTERNAL_NETWORK_ID")
# conn.network.remove_gateway_from_router("ROUTER_ID")
```

```sh
go run . -cloud dev -router ROUTER_ID -action add-interface -subnet SUBNET_ID
go run . -cloud dev -router ROUTER_ID -action remove-interface -port PORT_ID
go run . -cloud dev -router ROUTER_ID -action add-gateway -network EXTERNAL_NETWORK_ID
go run . -cloud dev -router ROUTER_ID -action remove-gateway
```

```go
package main

import (
	"github.com/JSYoo5B/go-openstacksdk/request"
    "context"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    id := flag.String("router", "", "literal router ID")
    action := flag.String("action", "", "add-interface, remove-interface, add-gateway, remove-gateway")
    port := flag.String("port", "", "interface port ID; wins subnet")
    subnet := flag.String("subnet", "", "interface subnet ID")
    network := flag.String("network", "", "single external gateway network ID")
    advertise := flag.Bool("advertise-host", false, "add-interface only; requires supporting backend")
    legacy := flag.Bool("legacy-source-route", false, "gateway only; requires backend legacy action support")
    flag.Parse()
    if *id == "" { log.Fatal("-router is required") }
    switch *action {
    case "add-interface", "remove-interface":
        if *port == "" && *subnet == "" { log.Fatal("provide -port or -subnet") }
        if *legacy { log.Fatal("-legacy-source-route is a gateway option") }
        if *action == "remove-interface" && *advertise { log.Fatal("-advertise-host is add-interface only") }
    case "add-gateway":
        if *network == "" { log.Fatal("-network is required") }
    case "remove-gateway":
        if *network != "" && !*legacy { log.Fatal("stock remove-gateway accepts no network attribute") }
    default: log.Fatal("select a supported -action")
    }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    var result *routers.RouterJSONResult
    switch *action {
    case "add-interface", "remove-interface":
        options := []routers.RouterInterfaceOption{}
        if *port != "" { options = append(options, routers.WithRouterInterfacePort(*port)) }
        if *subnet != "" { options = append(options, routers.WithRouterInterfaceSubnet(*subnet)) }
        if *action == "add-interface" {
            options = append(options, routers.WithRouterInterfaceAdvertiseHost(*advertise))
            result, err = service.Routers.AddInterfaceRecord(ctx, *id, options...)
        } else {
            result, err = service.Routers.RemoveInterfaceRecord(ctx, *id, options...)
        }
    case "add-gateway", "remove-gateway":
        options := []routers.RouterGatewayOption{request.Custom(func(c *routers.RouterGatewayOpts) error { c.LegacySourceRoutes = *legacy; return nil })}
        if *network != "" { options = append(options, request.Field("network_id", *network)) }
        if *action == "add-gateway" {
            result, err = service.Routers.AddGatewayRecord(ctx, *id, options...)
        } else {
            result, err = service.Routers.RemoveGatewayRecord(ctx, *id, options...)
        }
    }
    if result != nil {
        fmt.Printf("HTTP status=%d request_id=%s\n", result.StatusCode, result.Header.Get("X-Request-Id"))
        fmt.Printf("actual envelope=%s\n", result.Envelope)
        if result.Value != nil { fmt.Printf("complete JSON value=%s\n", result.Value) }
    }
    if err != nil { log.Fatal(err) }
}
```

## References

- [L3 Db](https://github.com/openstack/neutron/blob/27.0.0/neutron/db/l3_db.py)
- [Neutron 27.0.0 L3 extension](https://github.com/openstack/neutron/blob/27.0.0/neutron/extensions/l3.py)
- [neutron-lib 3.21.1 ACTION_MAP](https://github.com/openstack/neutron-lib/blob/3.21.1/neutron_lib/api/definitions/l3.py)
- [Router Actions Python](router-actions-python.md)
- [Router](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/router.py)
- [Install](../../docs/install.md)
