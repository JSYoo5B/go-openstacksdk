# QoS Minimum Packet Rate Records Python

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
parser.add_argument("--mode", choices=("get", "find", "list"), default="list")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "get":
        value = conn.network.get_qos_minimum_packet_rate_rule(args.rule, args.policy)
        print(json.dumps(value.to_dict(), indent=2))
    elif args.mode == "find":
        value = conn.network.find_qos_minimum_packet_rate_rule(
            args.rule, args.policy, ignore_missing=False
        )
        print(json.dumps(value.to_dict(), indent=2))
    else:
        for value in conn.network.qos_minimum_packet_rate_rules(
            args.policy, direction="any", max_items=100
        ):
            print(json.dumps(value.to_dict(), indent=2))
```

See `Example_qosMinimumPacketRateRecords` in [example_qos_minimum_packet_rate_records_test.go](example_qos_minimum_packet_rate_records_test.go), compiled by `go test`.

## References

- [Install](../../docs/install.md)
- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L488)
- [parent-owner base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L65)
