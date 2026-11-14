# QoS Policy Records Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--identity", default="application-qos")
parser.add_argument("--project", default="")
args = parser.parse_args()
query = {"is_shared": False}
if args.project:
    query["project_id"] = args.project

with openstack.connect(cloud=args.cloud) as conn:
    policy = conn.network.find_qos_policy(
        args.identity, ignore_missing=False, **query
    )
    print(json.dumps(policy.to_dict(), indent=2))
    for policy in conn.network.qos_policies(**query):
        print(json.dumps(policy.to_dict(), indent=2))
```

See `Example_qosPolicyRecords` in [example_qos_policy_records_test.go](example_qos_policy_records_test.go), compiled by `go test`.

## References

- [Neutron 27.0.0 QoS policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)
