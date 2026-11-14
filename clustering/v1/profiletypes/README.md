# Profiletypes

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

See `Example_profiletypesFragment1` in [example_profiletypes_fragment1_test.go](example_profiletypes_fragment1_test.go), compiled by `go test`.

```go
values, err := api.All(ctx,
    profiletypes.WithListOptions(profiletypes.ListOpts{Limit: 20}),
    request.Query("vendor_filter", "enabled"),
)
```

See `Example_profiletypesFragment3` in [example_profiletypes_fragment3_test.go](example_profiletypes_fragment3_test.go), compiled by `go test`.

See `example_profiletypes1_test.go` in [example_profiletypes1_test.go](example_profiletypes1_test.go), compiled by `go test`.

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#profile-types-profile-types)
- [Listing](../listing/README.md)
- [Clustering Catalog List Controls Test](../../../api/clustering_catalog_list_controls_test.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)
