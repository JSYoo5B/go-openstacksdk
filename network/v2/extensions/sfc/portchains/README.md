# Portchains

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
pc = conn.network.create_sfc_port_chain(
    name="web-chain", port_pair_groups=[ppg.id], flow_classifiers=[fc.id],
    chain_parameters={"correlation": "mpls", "symmetric": False})
conn.network.update_sfc_port_chain(pc, flow_classifiers=[fc.id, fc2.id])
conn.network.delete_sfc_port_chain(pc)
```

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9367-L9499)
- [`sfc_port_chain.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/sfc_port_chain.py)
