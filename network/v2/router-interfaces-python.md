# Router Interfaces Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

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

See `Example_routerInterfaces` in [example_router_interfaces_test.go](example_router_interfaces_test.go), compiled by `go test`.

## References

- [L3 Db](https://github.com/openstack/neutron/blob/27.0.0/neutron/db/l3_db.py)
- [Neutron 27.0.0 L3 extension](https://github.com/openstack/neutron/blob/27.0.0/neutron/extensions/l3.py)
- [neutron-lib 3.21.1 ACTION_MAP](https://github.com/openstack/neutron-lib/blob/3.21.1/neutron_lib/api/definitions/l3.py)
- [Router Actions Python](router-actions-python.md)
- [Router](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/router.py)
- [Install](../../docs/install.md)
