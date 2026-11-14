# Grouptypes

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
gt = conn.block_storage.create_group_type(name="gold", is_public=True)
conn.block_storage.create_group_type_group_specs(gt, {"consistent_group_snapshot_enabled": "<is> True"})
conn.block_storage.delete_group_type(gt)
```

```go
api := grouptypes.New(volumeClient)
gt, err := api.Create(ctx,
    request.Field("name", "gold"),
    request.Field("is_public", true))
if err != nil { return err }
_, err = api.CreateGroupSpecs(ctx, gt.ID, map[string]any{"consistent_group_snapshot_enabled": "<is> True"})
if err != nil { return err }
err = api.Delete(ctx, gt.ID)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2809-L3025)
- [`group_type.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group_type.py)
