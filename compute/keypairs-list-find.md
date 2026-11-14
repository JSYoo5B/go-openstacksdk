# Keypairs List Find

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect()
for keypair in conn.compute.keypairs(type="ssh"):
    print(keypair.to_dict())

found = conn.compute.find_keypair("workstation", ignore_missing=True)
if found is not None:
    print(found.to_dict())
```

See `Example_keypairsListFind` in [example_keypairs_list_find_test.go](example_keypairs_list_find_test.go), compiled by `go test`.

## References

- [Nova policy](https://docs.openstack.org/nova/latest/configuration/policy.html)
- [Compute](https://docs.openstack.org/api-ref/compute/#list-keypairs)
- [Console Selection](console-selection.md)
- [Proxy keypairs/find_keypair](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L910-L978)
- [Keypair](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/keypair.py)
- [Keypairs Console](keypairs-console.md)
- [Keypair leaf API](v2/keypairs/README.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
