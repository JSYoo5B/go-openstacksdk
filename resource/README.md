# Resource

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
resource.ID("server-id")
resource.Name("web-01")
resource.ID(server.ID)
```

See `Example_listWithEarlyStop` in [example_docs_test.go](../example_docs_test.go), compiled by `go test`.

See `Example_waitOnAttribute` in [example_docs_test.go](../example_docs_test.go), compiled by `go test`.

See `example_resource1_test.go` in [example_resource1_test.go](example_resource1_test.go), compiled by `go test`.

## References

- [Finding Identities](../docs/finding-identities.md)
- [Subnets](../network/v2/subnets/README.md)
- [Listing](../keymanager/v1/secrets/listing/README.md)
- [Listing](../keymanager/v1/containers/listing/README.md)
- [Listing](../keymanager/v1/orders/listing/README.md)
- [Listing](../network/v2/extensions/security/addressgroups/listing/README.md)
- [Listing](../network/v2/extensions/qos/policies/listing/README.md)
- [Listing](../network/v2/extensions/subnetpools/listing/README.md)
- [Listing](../network/v2/networks/listing/README.md)
- [Listing](../network/v2/extensions/layer3/routers/listing/README.md)
- [Listing](../network/v2/extensions/security/groups/listing/README.md)
- [Listing](../network/v2/extensions/trunks/listing/README.md)
- [Listing](../docs/listing.md)
- [Listing](../clustering/v1/listing/README.md)
- [Scoped Resources](../docs/scoped-resources.md)
- [collection_test.go](collection_test.go)
- [Collections Test](../connectiontest/collections_test.go)
