# Flavor Cloud

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(compute_api_version="2.55")
all_flavors = conn.list_flavors()  # get_extra=False
matched = conn.search_flavors("m1.*", filters={"disk": 20})
found = conn.get_flavor("m1.small")  # get_extra=True, ignore_missing=True
for flavor in matched:
    print(flavor.to_dict())
if found is not None:
    print(found.to_dict())
```

See `Example_flavorCloud` in [example_flavor_cloud_test.go](example_flavor_cloud_test.go), compiled by `go test`.

## References

- [Flavor Records](flavor-records.md)
- [owned List](flavor-records.md)
- [owned Flavor](flavor-records.md)
- [Cloud list_flavors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L278-L289)
- [search_flavors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L165-L181)
- [get_flavor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L524-L561)
- [_filter_list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L43-L145)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
