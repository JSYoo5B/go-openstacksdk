# Cloud QoS Dscp Python

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
    parser.add_argument("--policy", required=True)
    parser.add_argument("--rule", default="")
    parser.add_argument("--mode", choices=("list", "search", "get"), default="list")
    parser.add_argument("--filters", default="null")
    args = parser.parse_args()
    with openstack.connect(cloud=args.cloud) as conn:
        filters = json.loads(args.filters)
        if args.mode == "list":
            values = conn.list_qos_dscp_marking_rules(args.policy, filters)
        elif args.mode == "search":
            values = conn.search_qos_dscp_marking_rules(args.policy, args.rule or None, filters)
        else:
            if not args.rule:
                parser.error("--rule required for get")
            values = [conn.get_qos_dscp_marking_rule(args.policy, args.rule)]
        for value in values:
            print(json.dumps(value.to_dict() if hasattr(value, "to_dict") else value,
                             indent=2, ensure_ascii=False))

if __name__ == "__main__":
    main()
```

See `Example_cloudQosDscp` in [example_cloud_qos_dscp_test.go](example_cloud_qos_dscp_test.go), compiled by `go test`.

## References

- [ Network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py)
- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)
- [Cloud QoS Dscp Test](cloud_qos_dscp_test.go)
