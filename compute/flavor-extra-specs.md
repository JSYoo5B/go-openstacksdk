# Flavor Extra Specs

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import sys
import openstack
from openstack.compute.v2.flavor import Flavor

conn = openstack.connect()
existing = Flavor.existing(id=sys.argv[1], extra_specs={"inline": "old"})
updated = conn.compute.fetch_flavor_extra_specs(existing)
print(updated.extra_specs)
print(updated.to_dict())
```

See `Example_flavorExtraSpecs` in [example_flavor_extra_specs_test.go](example_flavor_extra_specs_test.go), compiled by `go test`.

## References

- [Console Selection](console-selection.md)
- [User Read APIs](user-read-apis.md)
- [`fetch_flavor_extra_specs`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L334)
- [`Flavor.fetch_extra_specs`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/flavor.py#L186)
- [`fields._convert_type`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86)
- [`_BaseComponent.__get__`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L208)
- [`ListExtraSpecs`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/flavors/requests.go#L277)
- [`Extract`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/flavors/results.go#L220)
- [Compute Flavor Extra Specs Fetch Test](../api/compute_flavor_extra_specs_fetch_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Implementation Plan](../docs/implementation-plan.md)
