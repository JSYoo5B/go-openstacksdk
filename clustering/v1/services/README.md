# Services

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
services = conn.clustering.services()
```

See `Example_list` in [example_services_fragment1_test.go](example_services_fragment1_test.go), compiled by `go test`.

See `Example_all` in [example_services_fragment2_test.go](example_services_fragment2_test.go), compiled by `go test`.

See `example_services1_test.go` in [example_services1_test.go](example_services1_test.go), compiled by `go test`.

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#services-services)
- [Listing](../listing/README.md)
- [Clustering Catalog List Controls Test](../../../api/clustering_catalog_list_controls_test.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)
