# Locations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_locations_test.go` in [example_locations_test.go](example_locations_test.go), compiled by `go test`.

```python
def add_and_read_locations(conn, image_id, location_url, validation_data=None):
    created = conn.image.add_image_location(
        image_id, location_url, validation_data=validation_data
    )
    locations = list(conn.image.image_locations(image_id))
    return created, locations
```

## References

- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#add-location)
- [serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1882-L1883)
- [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1161-L1257)
- [Config](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/common/config.py#L414)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#get-location)
- [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1259-L1280)
- [Locations Contracts Test](locations_contracts_test.go)
- [core tests](locations_core_test.go)
- [option tests](locations_options_test.go)
- [Connection test](../connectiontest/connection_image_locations_test.go)
- [generator test](../internal/cmd/sdkgen/glance_locations_test.go)
