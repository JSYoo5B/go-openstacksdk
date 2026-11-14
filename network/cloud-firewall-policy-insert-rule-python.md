# Cloud Firewall Policy Insert Rule Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--identity", required=True)
parser.add_argument("--rule", required=True)
parser.add_argument("--after")
parser.add_argument("--before")
parser.add_argument("--filters", default="{}")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    result = conn.insert_rule_into_policy(
        args.identity, args.rule, insert_after=args.after,
        insert_before=args.before, filters=json.loads(args.filters))
    print(json.dumps(result.to_dict(), indent=2))
```

See `Example_cloudFirewallPolicyInsertRule` in [example_cloud_firewall_policy_insert_rule_test.go](example_cloud_firewall_policy_insert_rule_test.go), compiled by `go test`.

## References

- [Install](../docs/install.md)
