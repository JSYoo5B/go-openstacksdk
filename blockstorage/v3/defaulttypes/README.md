# Defaulttypes

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
conn.block_storage.set_default_type(project_id, volume_type_id)
default = conn.block_storage.show_default_type(project_id)
conn.block_storage.unset_default_type(project_id)
```

```go
api := defaulttypes.New(volumeClient)
if _, err := api.Set(ctx, projectID, volumeTypeID); err != nil { return err }
dt, err := api.Get(ctx, projectID)
if err != nil { return err }
fmt.Println(*dt.VolumeTypeID)
err = api.Unset(ctx, projectID)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L738-L818)
- [`default_type.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/default_type.py)
