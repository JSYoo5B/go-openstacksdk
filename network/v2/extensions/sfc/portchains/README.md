# Portchains

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
pc = conn.network.create_sfc_port_chain(
    name="web-chain", port_pair_groups=[ppg.id], flow_classifiers=[fc.id],
    chain_parameters={"correlation": "mpls", "symmetric": False})
conn.network.update_sfc_port_chain(pc, flow_classifiers=[fc.id, fc2.id])
conn.network.delete_sfc_port_chain(pc)
```

```go
api := portchains.New(networkClient)
pc, err := api.Create(ctx,
    request.Field("name", "web-chain"),
    request.Field("port_pair_groups", []string{ppg.ID}),
    request.Field("flow_classifiers", []string{fc.ID}),
    request.Field("chain_parameters", map[string]any{"correlation": "mpls", "symmetric": false}))
if err != nil { return err }
_, err = api.Update(ctx, pc.ID, request.Field("flow_classifiers", []string{fc.ID, fc2.ID}))
if err != nil { return err }
err = api.Delete(ctx, pc.ID)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L9367-L9499)
- [`sfc_port_chain.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/sfc_port_chain.py)
