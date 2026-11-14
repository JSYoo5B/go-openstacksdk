# Floating Ip Attach

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
server = conn.compute.find_server("web-01", ignore_missing=False)
server = conn.add_ip_list(
    server, ips=["198.51.100.10"],
    fixed_address="10.0.0.10", wait=True, timeout=60,
)
```

See `Example_floatingIpAttach` in [example_floating_ip_attach_test.go](example_floating_ip_attach_test.go), compiled by `go test`.

## References

- [AddIPsToServer·AddIPList](../compute/server-ip-helpers.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1342-L1391)
- [Server Ip Dispatch](../compute/server-ip-dispatch.md)
- [Compute legacy Nova backend](../compute/server-nova-floating-ip.md)
- [Floating Ip Ensure](floating-ip-ensure.md)
- [Implementation Plan](../docs/implementation-plan.md)
