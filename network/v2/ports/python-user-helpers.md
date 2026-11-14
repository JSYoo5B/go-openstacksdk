# Python User Helpers

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
    parser.add_argument("--mode", choices=("values", "subnet", "bulk"), default="values")
    parser.add_argument("--filters", default="{}")
    parser.add_argument("--subnet", default="")
    parser.add_argument("--data", default="[]")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    if args.mode == "values":
        values = conn.list_ports(filters=json.loads(args.filters))
        for value in values:
            print(json.dumps(value.to_dict() if hasattr(value, "to_dict") else value, ensure_ascii=False, indent=2))
    elif args.mode == "subnet":
        if not args.subnet:
            parser.error("--subnet required")
        for value in conn.network.get_subnet_ports(args.subnet):
            print(json.dumps(value.to_dict(), ensure_ascii=False, indent=2))
    else:
        for value in conn.network.create_ports(json.loads(args.data)):
            print(json.dumps(value.to_dict(), ensure_ascii=False, indent=2))

if __name__ == "__main__":
    main()
```

See `Example_userHelpers` in [example_user_helpers_test.go](example_user_helpers_test.go), compiled by `go test`.

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/tree/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe)
- [Python List Records](python-list-records.md)
- [Source cloud `list_ports`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L202)
- [Source loop](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L4047)
- [Source create_ports](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3894)
- [V2](https://docs.openstack.org/api-ref/network/v2/#bulk-create-ports)
- [Neutron 27 Pecan BodyValidation](https://github.com/openstack/neutron/blob/27.0.0/neutron/pecan_wsgi/hooks/body_validation.py)
- [Policy hooks](https://github.com/openstack/neutron/blob/27.0.0/neutron/pecan_wsgi/hooks/policy_enforcement.py)
- [Plugin](https://github.com/openstack/neutron/blob/27.0.0/neutron/plugins/ml2/plugin.py)
- [Port](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/port.py)
