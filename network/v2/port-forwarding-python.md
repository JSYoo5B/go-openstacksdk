# Port Forwarding Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
created = conn.network.create_floating_ip_port_forwarding(
    floating_ip_id, internal_port_id=port_id,
    internal_ip_address="10.0.0.5", internal_port=22,
    external_port=2222, protocol="tcp", description="SSH")
selected = conn.network.get_floating_ip_port_forwarding(
    floating_ip_id, created.id, fields=["id", "external_port"])
for forwarding in conn.network.floating_ip_port_forwardings(
    floating_ip_id, protocol="tcp", description="SSH"):
    print(forwarding.id)
found = conn.network.find_floating_ip_port_forwarding(floating_ip_id, created.id)
#     floating_ip_id, created.id, ignore_missing=False)
```

See `Example_portForwarding` in [example_port_forwarding_test.go](example_port_forwarding_test.go), compiled by `go test`.

## References

- [Install](../../docs/install.md)
- [Reference](README.md)
