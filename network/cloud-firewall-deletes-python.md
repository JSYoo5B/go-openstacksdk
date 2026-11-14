# Cloud Firewall Deletes Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

The filters JSON may carry Source Find kwargs such as `{"headers": {"X-Trace": "t"}, "microversion": "2.5", "max_items": 1}`: headers become an initial GET query value and fallback LIST headers, microversion versions only the initial GET, and max_items is compared before each fallback row.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--kind", choices=("rule", "policy", "group"), default="rule")
parser.add_argument("--name", required=True)
parser.add_argument("--filters", default="{}")
args = parser.parse_args()
filters = json.loads(args.filters)
with openstack.connect(cloud=args.cloud) as conn:
    if args.kind == "rule":
        result = conn.delete_firewall_rule(args.name, filters=filters)
    elif args.kind == "policy":
        result = conn.delete_firewall_policy(args.name, filters=filters)
    else:
        result = conn.delete_firewall_group(args.name, filters=filters)
    print(json.dumps(result))
```

See `Example_cloudFirewallDeletes` in [example_cloud_firewall_deletes_test.go](example_cloud_firewall_deletes_test.go), compiled by `go test`.

## References

- [Policy Sample](https://docs.openstack.org/neutron-fwaas/2025.2/configuration/policy-sample.html)
