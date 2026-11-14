# Rbac Policy Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--action", choices=["list", "get", "find"], default="list")
    parser.add_argument("--id", default="")
    parser.add_argument("--fields", default="")
    parser.add_argument("--object-type", default="network")
    parser.add_argument("--target-project", default="")
    parser.add_argument("--strict", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    fields = args.fields.split(",") if args.fields else None
    if args.action == "get":
        value = conn.network.get_rbac_policy(args.id, fields=fields)
    else:
        query = {"object_type": args.object_type}
        if fields:
            query["fields"] = fields
        if args.target_project:
            query["target_project_id"] = args.target_project
        if args.action == "list":
            for value in conn.network.rbac_policies(**query):
                print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))
            return
        value = conn.network.find_rbac_policy(
            args.id, ignore_missing=not args.strict, **query)
    if value is None:
        print("not found")
    else:
        print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_rbacPolicy` in [example_rbac_policy_test.go](example_rbac_policy_test.go), compiled by `go test`.

## References

- [RBACPolicy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/rbac_policy.py)
- [proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5432-L5500)
- [Neutron RBAC API](https://docs.openstack.org/api-ref/network/v2/#rbac-policies)
- [Install](../../docs/install.md)
- [Rbac](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/rbac.py)
- [Python Records Contracts Test](extensions/rbacpolicies/python_records_contracts_test.go)
