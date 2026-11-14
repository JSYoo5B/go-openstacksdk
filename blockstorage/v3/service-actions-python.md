# Service Actions Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
svc = next(conn.block_storage.services(binary="cinder-volume"))
conn.block_storage.disable_service(svc, reason="maintenance")
for level in conn.block_storage.get_service_log_levels(binary="cinder-api"):
    print(level.host, level.levels)
```

See `ExampleAPI_GetLogLevels` in [example_guide_test.go](services/example_guide_test.go), compiled by `go test`.

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3423-L3616)
- [`service.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/service.py)
