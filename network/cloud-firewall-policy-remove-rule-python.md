# Cloud Firewall Policy Remove Rule Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--policy", required=True)
parser.add_argument("--rule", required=True)
parser.add_argument("--filters", default="{}")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    result = conn.remove_rule_from_policy(args.policy, args.rule, filters=json.loads(args.filters))
    print(json.dumps(result.to_dict(), indent=2))
```

See `Example_cloudFirewallPolicyRemoveRule` in [example_cloud_firewall_policy_remove_rule_test.go](example_cloud_firewall_policy_remove_rule_test.go), compiled by `go test`.

## References

- [Install](../docs/install.md)
