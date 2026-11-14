# Serviceproviders

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import argparse
import json
import types
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--service-type")
parser.add_argument("--default", choices=("any", "yes", "no"), default="any")
parser.add_argument("--jmes")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    query = {}
    if args.service_type is not None:
        query["service_type"] = args.service_type
    if args.default != "any":
        query["is_default"] = args.default == "yes"
    if args.jmes is not None:
        query["jmespath_filters"] = args.jmes
    result = conn.network.service_providers(**query)
    if isinstance(result, types.GeneratorType):
        for value in result:
            print(json.dumps(value.to_dict(), indent=2))
    else:
        print(result)
```

See `Example_serviceproviders` in [example_serviceproviders_test.go](example_serviceproviders_test.go), compiled by `go test`.

## References

- [Service Type](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/service_type.py#L22-L41)
- [Install](../../../docs/install.md)
- [ServiceProvider](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/service_provider.py#L16)
- [Proxy.service_providers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7274)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)
