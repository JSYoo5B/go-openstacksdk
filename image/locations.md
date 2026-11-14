# Locations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func addDefaultLocation(ctx context.Context, service *image.Service, ref resource.Ref, locationURL string) (*image.AddImageLocationResult, error) {
    return service.AddImageLocation(ctx, ref, locationURL)
}

func addValidatedThenGetLocations(ctx context.Context, service *image.Service, ref resource.Ref, locationURL, algorithm, value string) (ack *image.AddImageLocationResult, locations *image.ImageLocationsResult, err error) {
    ack, err = service.AddImageLocation(ctx, ref, locationURL,
        image.WithAddImageLocationOpts(image.AddImageLocationOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        image.WithImageLocationValidation(algorithm, value),
        request.Header("X-Location-Request", "add"))
    if err != nil {
        return ack, nil, err
    }
    locations, err = service.GetImageLocations(ctx, ref,
        image.WithGetImageLocationsOpts(image.GetImageLocationsOpts{
            Headers: map[string]string{"X-Request-Source": "example"},
        }),
        request.Header("X-Location-Request", "get"))
    return ack, locations, err
}
```

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
