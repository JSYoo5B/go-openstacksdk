# Python Quota Query

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect()
quota = conn.compute.get_quota_set(
    "project-id", usage=True, user_id="user-id", fields=["cores", "ram"]
)
print(quota.cores, quota.usage, quota.reservation)
```

See `Example_quotaQuery` in [example_quota_query_test.go](example_quota_query_test.go), compiled by `go test`.

## References

- [compute proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)
- [Quota Sets](https://github.com/openstack/nova/blob/31.0.0/nova/policies/quota_sets.py)
- [base rule](https://github.com/openstack/nova/blob/31.0.0/nova/policies/base.py)
- [controller](https://github.com/openstack/nova/blob/31.0.0/nova/api/openstack/compute/quota_sets.py)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)
- [quota class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/quota_set.py)
- [Quota Set](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/quota_set.py)
- [Python Quota Query Test](python_quota_query_test.go)
- [Read Test](../../../internal/quotafetch/read_test.go)
