# QoS Rule Types Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--operation", choices=["get", "find", "list"], default="list")
parser.add_argument("--name", default="bandwidth_limit")
parser.add_argument("--fields")
parser.add_argument("--type")
parser.add_argument("--all-rules", action="store_true")
parser.add_argument("--all-supported", action="store_true")
parser.add_argument("--max-items", type=int, default=0)
parser.add_argument("--strict", action="store_true")
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
fields = args.fields.split(",") if args.fields else None
if args.operation == "get":
    value = conn.network.get_qos_rule_type(args.name, fields=fields)
    print(value.to_dict())
elif args.operation == "find":
    # The pinned public find method has no **query or fields argument.
    value = conn.network.find_qos_rule_type(args.name, ignore_missing=not args.strict)
    print(None if value is None else value.to_dict())
else:
    query = {}
    if fields:
        query["fields"] = fields
    if args.type is not None:
        query["type"] = args.type
    if args.all_rules:
        query["all_rules"] = True
    if args.all_supported:
        query["all_supported"] = True
    if args.max_items:
        query["max_items"] = args.max_items
    for value in conn.network.qos_rule_types(**query):
        print(value.to_dict())
```

See `Example_qosRuleTypes` in [example_qos_rule_types_test.go](example_qos_rule_types_test.go), compiled by `go test`.

## References

- [Policy](https://docs.openstack.org/neutron/2025.2/configuration/policy.html)
- [V2](https://docs.openstack.org/api-ref/network/v2/#qos-rule-types)
- [Install](../../docs/install.md)
- [three proxy declarations](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5212-L5285)
- [QoSRuleType schema](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_rule_type.py#L16-L42)
- [Resource alternate id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L654-L672)
