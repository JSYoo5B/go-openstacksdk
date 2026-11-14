# Availability Zones

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect(cloud="dev")
for zone in conn.compute.availability_zones():
    print(zone.name, zone.state, zone.hosts)

print(conn.list_availability_zone_names())
print(conn.list_availability_zone_names(unavailable=True))
```

See `Example_availabilityZones` in [example_availability_zones_test.go](example_availability_zones_test.go), compiled by `go test`.

## References

- [Policy](https://docs.openstack.org/nova/2024.1/configuration/policy.html)
- [Implementation Plan](../docs/implementation-plan.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Proxy availability_zones](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L1854-L1874)
- [AvailabilityZone descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/availability_zone.py#L16-L36)
- [Cloud list_availability_zone_names](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L255-L276)
