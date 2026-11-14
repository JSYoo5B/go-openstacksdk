# Router Actions Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
router_id = "ROUTER_ID"
route_body = {"router": {"routes": [
    {"destination": "192.0.2.0/24", "nexthop": "10.0.0.9"}
]}}
gateway_body = {"router": {"external_gateways": [
    {"network_id": "EXTERNAL_NETWORK_ID"}
]}}

added = conn.network.add_extra_routes_to_router(router_id, route_body)
removed = conn.network.remove_extra_routes_from_router(router_id, route_body)
gateway_added = conn.network.add_external_gateways(router_id, gateway_body)
gateway_updated = conn.network.update_external_gateways(router_id, gateway_body)
gateway_removed = conn.network.remove_external_gateways(router_id, gateway_body)
print(added.to_dict())
```

```sh
go run . -cloud dev -router ROUTER_ID -action add-routes \
  -body '{"router":{"routes":[{"destination":"192.0.2.0/24","nexthop":"10.0.0.9"}]}}'
go run . -cloud dev -router ROUTER_ID -action add-gateways \
  -body '{"router":{"external_gateways":[{"network_id":"EXTERNAL_NETWORK_ID"}]}}'
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    id := flag.String("router", "", "literal router ID")
    action := flag.String("action", "", "add-routes, remove-routes, add-gateways, update-gateways, remove-gateways")
    body := flag.String("body", "", "complete JSON action body, including router envelope")
    flag.Parse()
    if *id == "" || !json.Valid([]byte(*body)) { log.Fatal("-router and a valid JSON -body are required") }
    switch *action {
    case "add-routes", "remove-routes", "add-gateways", "update-gateways", "remove-gateways":
    default: log.Fatal("select a supported -action")
    }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Routers
    raw := json.RawMessage(*body)
    var result *routers.RouterActionRecord
    switch *action {
    case "add-routes": result, err = api.AddExtraRoutesRecord(ctx, *id, raw)
    case "remove-routes": result, err = api.RemoveExtraRoutesRecord(ctx, *id, raw)
    case "add-gateways": result, err = api.AddExternalGatewaysRecord(ctx, *id, raw)
    case "update-gateways": result, err = api.UpdateExternalGatewaysRecord(ctx, *id, raw)
    case "remove-gateways": result, err = api.RemoveExternalGatewaysRecord(ctx, *id, raw)
    }
    if result != nil {
        fmt.Printf("HTTP status=%d request_id=%s\n", result.StatusCode, result.Header.Get("X-Request-Id"))
        fmt.Printf("actual envelope=%s\n", result.Envelope)
        if result.Resource != nil {
            view, encodeErr := json.MarshalIndent(result.Resource, "", "  ")
            if encodeErr != nil { log.Fatal(encodeErr) }
            fmt.Printf("Source descriptor view=%s\n", view)
        }
    }
    if err != nil { log.Fatal(err) }
}
```

```go
cached := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{
    "name": json.RawMessage(`"cached-edge"`),
    "routes": json.RawMessage(`[]`),
}}}
row, err := api.AddExtraRoutesRecord(ctx, "ROUTER_ID", json.RawMessage(`{"router":{"routes":[]}}`),
    routers.WithRouterActionSeed(cached),
    request.Headers(map[string]string{"X-Client-Trace": "router-action"}))
```

## References

- [Router](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/router.py)
- [Install](../../docs/install.md)
- [Implementation Plan](../../docs/implementation-plan.md)
