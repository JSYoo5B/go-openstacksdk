# Stacks

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_inStack` in [example_test.go](example_test.go), compiled by `go test`.

See `Example_forStack` in [example_test.go](example_test.go), compiled by `go test`.

## References

- [..](../README.md)
- [Heat Stacks Contracts Test](../../../api/heat_stacks_contracts_test.go)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)

## Native API calls

The generated `API` methods are the unchanged native entry points and take the stack name and ID, or a single identity string, as separate path segments. They are covered by `api/heat_native_stacks_test.go`.

See `ExampleAPI` in [example_test.go](example_test.go), compiled by `go test`.

Limitations: an empty non-nil `Tags` slice is sent as `"tags": ""`, while `nil` tags and an empty `Parameters` map are omitted. Native `Delete` does not ignore 404. A canceled context fails before any request. Mutating calls are never retried.
