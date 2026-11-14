# Python Onboard Network Subnets

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--pool", required=True)
parser.add_argument("--network", required=True)
args = parser.parse_args()
with openstack.connect() as conn:
    result = conn.network.onboard_network_subnets(args.pool, args.network)
    print(json.dumps(result, indent=2))
```

See `Example_onboardNetworkSubnets` in [example_onboard_network_subnets_test.go](example_onboard_network_subnets_test.go), compiled by `go test`.

## References

- [Proxy.onboard_network_subnets](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [SubnetPool.subnet_onboard](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet_pool.py)
