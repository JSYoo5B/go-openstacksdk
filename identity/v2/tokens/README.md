# Tokens

`Tokens.Create/Get` returns SDK-owned Authentication with Token, User, Catalog, raw JSON Body, and Header, preserving additional fields. Keystone v2.0 was removed in Queens and is applicable only to deployments that still expose it. Native request statuses and administrator conditions remain operation-specific.

Use the concrete API and operation-specific options in this package. Parent scopes fix the caller-selected resource; optional request values and missing-resource behavior are defined per operation.

Consult the [support ledger](../../../docs/sdk-support-ledger.md) and [operation reviews](../../../api/sdk_reviews.json) for supported behavior and unresolved boundaries.

## API reference

- [api_generated.go](api_generated.go)
- [authentication.go](authentication.go)

## Usage examples

See `ExampleAPI_Get` in [example_test.go](example_test.go), compiled by `go test`.
