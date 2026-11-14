# Security Find Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
group = conn.network.find_security_group("web", project_id="PROJECT_ID")
rule = conn.network.find_security_group_rule("RULE_ID", security_group_id="GROUP_ID")
strict = conn.network.find_security_group("web", ignore_missing=False)
print(None if group is None else group.to_dict())
```

See `Example_securityFind` in [example_security_find_test.go](example_security_find_test.go), compiled by `go test`.

## References

- [Security Group proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6598-L6623)
- [Rule proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6768-L6793)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2489-L2582)
- [Install](../../docs/install.md)
