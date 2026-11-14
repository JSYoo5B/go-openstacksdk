# QoS Minimum Bandwidth Records Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--policy", required=True)
parser.add_argument("--rule", default="")
parser.add_argument("--mode", choices=("find", "get", "list"), default="list")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "find":
        rule = conn.network.find_qos_minimum_bandwidth_rule(
            args.rule, args.policy, ignore_missing=False
        )
        print(json.dumps(rule.to_dict(), indent=2))
    elif args.mode == "get":
        rule = conn.network.get_qos_minimum_bandwidth_rule(args.rule, args.policy)
        print(json.dumps(rule.to_dict(), indent=2))
    else:
        for rule in conn.network.qos_minimum_bandwidth_rules(
            args.policy, direction="egress", max_items=100
        ):
            print(json.dumps(rule.to_dict(), indent=2))
```

See `Example_qosMinimumBandwidthRecords` in [example_qos_minimum_bandwidth_records_test.go](example_qos_minimum_bandwidth_records_test.go), compiled by `go test`.

## References

- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L413)
- [Base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L65)
