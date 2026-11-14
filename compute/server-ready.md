# Server Ready

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
server = conn.get_server("web-01")
if server is None:
	raise RuntimeError("server not found")

# Provided status is inspected. Non-ACTIVE returns None; default wait=False.
active = conn.get_active_server(server, auto_ip=True, reuse=True, wait=False, timeout=180)

# Only the supplied ID chooses the target; current state is fetched and polled.
ready = conn.wait_for_server(server, auto_ip=True, reuse=True, timeout=180)
print(ready.id, ready.interface_ip)
```

See `Example_serverReady` in [example_server_ready_test.go](example_server_ready_test.go), compiled by `go test`.

## References

- [AddIPsToServer·AddIPList](server-ip-helpers.md)
- [wait_for_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1359-L1415)
- [get_active_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1417-L1481)
- [attach helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1109-L1123)
- [IP dispatch](server-ip-dispatch.md)
- [Legacy Nova backend](server-nova-floating-ip.md)
- [Server Ip Dispatch](server-ip-dispatch.md)
