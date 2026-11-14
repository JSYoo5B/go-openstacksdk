# Quotas

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
quota = conn.key_manager.get_quota()
print(quota.secrets)
```

See `Example_string` in [example_quotas1_test.go](example_quotas1_test.go), compiled by `go test`.

See `example_quotas2_test.go` in [example_quotas2_test.go](example_quotas2_test.go), compiled by `go test`.

## References

- [Python Project Records](python-project-records.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L397)
- [Barbican quota API](https://docs.openstack.org/api-ref/key-manager/v1/#quotas)
- [ProjectQuotaValidator](https://github.com/openstack/barbican/blob/master/barbican/common/validators.py)
- [Keymanager Quotas Test](../../../api/keymanager_quotas_test.go)
- [public get_quota](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L457)
- [Quota fields](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/quota.py#L16)
- [Connection Keymanager SDK Owned Test](../../../connectiontest/connection_keymanager_sdk_owned_test.go)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)
