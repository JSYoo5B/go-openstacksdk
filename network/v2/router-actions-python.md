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

See `Example_routerActions1` in [example_router_actions1_test.go](example_router_actions1_test.go), compiled by `go test`.

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
