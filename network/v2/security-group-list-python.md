# Security Group List Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument('--cloud', required=True)
parser.add_argument('--name')
parser.add_argument('--limit', type=int)
parser.add_argument('--max-items', type=int, default=0)
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
query = {'max_items': args.max_items}
if args.name is not None:
    query['name'] = args.name
if args.limit is not None:
    query['limit'] = args.limit
for group in conn.network.security_groups(**query):
    print(json.dumps(group.to_dict(), ensure_ascii=False))
```

See `Example_securityGroupList` in [example_security_group_list_test.go](example_security_group_list_test.go), compiled by `go test`.

## References

- [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6647-L6665)
- [`SecurityGroup`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/security_group.py)
- [Security Group](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/security_group.py#L125-L153)
- [extension](https://github.com/openstack/neutron/blob/27.0.0/neutron/extensions/securitygroup.py)
- [stock Controller._items](https://github.com/openstack/neutron/blob/27.0.0/neutron/api/v2/base.py#L292-L378)
- [Python List Contracts Test](extensions/security/groups/python_list_contracts_test.go)
