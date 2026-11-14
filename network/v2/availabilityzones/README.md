# Availabilityzones

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect()
zones = list(conn.network.availability_zones(name="az-a"))
for zone in zones:
    print(zone.name, zone.resource, zone.state)
```

See `Example_availabilityzones` in [example_availabilityzones_test.go](example_availabilityzones_test.go), compiled by `go test`.

## References

- [availability_zones proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [Availability Zone](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/availability_zone.py)
- [Records Test](records_test.go)
