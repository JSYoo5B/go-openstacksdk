# Localipassociations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
assoc = conn.network.create_local_ip_association(lip, fixed_port_id=port.id)
for item in conn.network.local_ip_associations(lip, host="compute-1"):
    print(item.fixed_ip)
conn.network.delete_local_ip_association(lip, port.id)
```

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L1943-L2109)
- [`local_ip_association.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/local_ip_association.py)
- [`localips`](../localips)
