# Dscp Marking Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--policy", required=True)
    parser.add_argument("--operation", choices=("list", "get", "find"), default="list")
    parser.add_argument("--rule", default="")
    parser.add_argument("--page-size", type=int, default=0)
    parser.add_argument("--max-items", type=int, default=0)
    parser.add_argument("--mark", type=int)
    parser.add_argument("--strict", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect()
    if args.operation == "get":
        result = conn.network.get_qos_dscp_marking_rule(args.rule, args.policy)
        print(result.to_dict())
    elif args.operation == "find":
        result = conn.network.find_qos_dscp_marking_rule(
            args.rule, args.policy, ignore_missing=not args.strict
        )
        print(None if result is None else result.to_dict())
    else:
        query = {"limit": args.page_size, "max_items": args.max_items}
        if args.mark is not None:
            query["dscp_mark"] = args.mark
        for rule in conn.network.qos_dscp_marking_rules(args.policy, **query):
            print(rule.qos_policy_id, rule.to_dict())

if __name__ == "__main__":
    main()
```

See `Example_dscpMarking` in [example_dscp_marking_test.go](example_dscp_marking_test.go), compiled by `go test`.

## References

- [Source rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_dscp_marking_rule.py)
- [Neutron 27.0.0 policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L337-L357)
- [Python Dscp Marking Contracts Test](python_dscp_marking_contracts_test.go)
