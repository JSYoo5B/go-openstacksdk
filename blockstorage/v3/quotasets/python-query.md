# Python Query

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect()
quota = conn.block_storage.get_quota_set(
    "project-id", usage=True, fields=["volumes", "gigabytes"]
)
print(quota.volumes, quota.usage, quota.reservation)
```

See `Example_query` in [example_query_test.go](example_query_test.go), compiled by `go test`.

## References

- [block storage proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [Quotas](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/quotas.py)
- [Base](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/base.py)
- [controller](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/contrib/quotas.py)
- [quota class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/quota_set.py)
- [Quota Set](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/quota_set.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [Python Query Test](python_query_test.go)
- [Read Test](../../../internal/quotafetch/read_test.go)
