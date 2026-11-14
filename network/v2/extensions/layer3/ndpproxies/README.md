# Ndpproxies

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
proxy = conn.network.create_ndp_proxy(
    router_id=router.id, port_id=port.id, ip_address="2001:db8::10")
conn.network.update_ndp_proxy(proxy, description="web")
for item in conn.network.ndp_proxies(router_id=router.id):
    print(item.ip_address)
conn.network.delete_ndp_proxy(proxy)
```

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.
