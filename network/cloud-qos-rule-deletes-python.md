# Cloud QoS Rule Deletes Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import os
import openstack

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--kind", choices=("bandwidth", "dscp", "minimum"), required=True)
    parser.add_argument("--policy", required=True)
    parser.add_argument("--rule", required=True)
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    methods = {
        "bandwidth": conn.delete_qos_bandwidth_limit_rule,
        "dscp": conn.delete_qos_dscp_marking_rule,
        "minimum": conn.delete_qos_minimum_bandwidth_rule,
    }
    print(methods[args.kind](args.policy, args.rule))

if __name__ == "__main__":
    main()
```

See `Example_cloudQosRuleDeletes` in [example_cloud_qos_rule_deletes_test.go](example_cloud_qos_rule_deletes_test.go), compiled by `go test`.

## References

- [`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1723)
- [minimum-bandwidth Source](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2115)
- [Neutron27 QoS policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)
- [base policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py)
