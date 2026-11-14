# Metadata Refresh Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `ExampleAPI_SetMetadataAndRefresh` in [example_metadata_refresh_test.go](example_metadata_refresh_test.go), compiled by `go test`.

## References

- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L125-L147)
- [BaseResource.set_metadata](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L73-L92)
