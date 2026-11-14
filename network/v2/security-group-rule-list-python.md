# Security Group Rule List Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import os
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--group", default="")
    parser.add_argument("--direction", default="")
    parser.add_argument("--name", default="")
    parser.add_argument("--page-size", type=int, default=-1)
    parser.add_argument("--max-items", type=int, default=0)
    parser.add_argument("--single-page", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    query = {"max_items": args.max_items, "paginated": not args.single_page}
    if args.group:
        query["security_group_id"] = args.group
    if args.direction:
        query["direction"] = args.direction
    if args.name:
        query["name"] = args.name
    if args.page_size >= 0:
        query["limit"] = args.page_size
    for rule in conn.network.security_group_rules(**query):
        print(json.dumps(rule.to_dict(), ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_securityGroupRuleList` in [example_security_group_rule_list_test.go](example_security_group_rule_list_test.go), compiled by `go test`.

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6820-L6842)
- [Source SecurityGroupRule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/security_group_rule.py)
- [Source Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866-L916)
- [Neutron 27.0.0 policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/security_group.py#L228-L247)
- [stock DB](https://github.com/openstack/neutron/blob/27.0.0/neutron/db/securitygroups_db.py#L1021-L1050)
- [controller](https://github.com/openstack/neutron/blob/27.0.0/neutron/api/v2/base.py#L292-L331)
- [Python List Contracts Test](extensions/security/rules/python_list_contracts_test.go)
