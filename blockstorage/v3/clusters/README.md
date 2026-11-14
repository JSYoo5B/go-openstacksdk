# Clusters

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
cluster = conn.block_storage.get_cluster("cluster1@lvm")
conn.block_storage.disable_cluster(cluster, reason="maintenance")
```

```go
api := clusters.New(volumeClient)
cluster, err := api.Get(ctx, "cluster1@lvm")
if err != nil { return err }
_, err = api.Disable(ctx, cluster.Target(), request.Custom(func(c *clusters.ActionOpts) error { c.Reason = "maintenance"; return nil }))
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1957-L2065)
- [`cluster.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/cluster.py)
