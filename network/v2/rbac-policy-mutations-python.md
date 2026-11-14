# Rbac Policy Mutations Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--mode", choices=("create", "update", "delete"), required=True)
parser.add_argument("--policy", default="")
parser.add_argument("--object", default="")
parser.add_argument("--target-project", default="")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "create":
        value = conn.network.create_rbac_policy(
            object_type="network", object_id=args.object,
            action="access_as_shared", target_project_id=args.target_project,
        )
        print(json.dumps(value.to_dict(), indent=2))
    elif args.mode == "update":
        value = conn.network.update_rbac_policy(
            args.policy, target_project_id=args.target_project
        )
        print(json.dumps(value.to_dict(), indent=2))
    else:
        conn.network.delete_rbac_policy(args.policy)
        print(json.dumps({"deleted": args.policy}, indent=2))
```

See `Example_rbacPolicyMutations` in [example_rbac_policy_mutations_test.go](example_rbac_policy_mutations_test.go), compiled by `go test`.

## References

- [Install](../../docs/install.md)
- [Rbac](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/rbac.py#L89)
- [owner base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L32)
- [Stock RBAC schema](https://github.com/openstack/neutron/blob/27.0.0/neutron/extensions/rbac.py#L60)
