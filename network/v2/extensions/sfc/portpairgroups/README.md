# Portpairgroups

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
ppg = conn.network.create_sfc_port_pair_group(
    name="fw-group", port_pairs=[pp.id],
    port_pair_group_parameters={"lb_fields": ["ip_src"]})
conn.network.update_sfc_port_pair_group(ppg, port_pairs=[pp.id, pp2.id])
conn.network.delete_sfc_port_pair_group(ppg)
```

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9630-L9767)
- [`sfc_port_pair_group.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/sfc_port_pair_group.py)
