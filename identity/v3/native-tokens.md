# Native Tokens

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:identity/v3/tokens.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens.WithCreateField`

- Create POSTs {"auth":{"identity":...,"scope":...}} to auth/tokens for password (name+domain or user ID), token and application credential methods with project, domain or system scope, keeps extension fields in the auth object only, rejects nil options, no auth method and any header extension before HTTP because native Create never calls ToTokenV3HeadersMap, reads the token ID from X-Subject-Token and RFC3339 expires_at, still sends the provider X-Auth-Token because native OmitHeaders runs before it is applied, and accepts only 201/202.
  Tests: `identity/v3/tokens/native_contracts_test.go:TestNativeTokenCreateBodiesScopesAndSubjectHeader`, `identity/v3/tokens/native_contracts_test.go:TestNativeTokenStatusesAndPreflight`, `internal/cmd/sdkgen/page_extractors_test.go:TestTokenScopeBuildersDelegateWithoutExtensionFields`, `internal/cmd/sdkgen/page_extractors_test.go:TestUnreadBuilderMethodsDisableTheirExtensions`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/tokens call only; the Connection authentication flow and Python identity Resource parity are tracked separately.
- The generated scope builder delegates without merging extension fields, so extensions appear once in the auth object; native URL construction, OkCodes, header handling and decoding are unchanged.

### `gophercloud:identity/v3/tokens.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens.API.Get`

- Get GETs auth/tokens with X-Subject-Token, reads the ID from the response subject header and accepts only 200/203.
  Tests: `identity/v3/tokens/native_contracts_test.go:TestNativeTokenGetValidateRevoke`, `identity/v3/tokens/native_contracts_test.go:TestNativeTokenStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/tokens call only; the Connection authentication flow and Python identity Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams or Validate) and the documented extension merge; native URL construction, OkCodes, header handling and decoding are unchanged.

### `gophercloud:identity/v3/tokens.Validate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens.API.Validate`

- Validate HEADs auth/tokens with X-Subject-Token, returns true for 200/204 and false without error for 404, and returns other status errors as native errors without operation context.
  Tests: `identity/v3/tokens/native_contracts_test.go:TestNativeTokenGetValidateRevoke`, `identity/v3/tokens/native_contracts_test.go:TestNativeTokenStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/tokens call only; the Connection authentication flow and Python identity Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams or Validate) and the documented extension merge; native URL construction, OkCodes, header handling and decoding are unchanged.

### `gophercloud:identity/v3/tokens.Revoke`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens.API.Revoke`

- Revoke DELETEs auth/tokens with X-Subject-Token, decodes the bodyless response into an empty Token and accepts only 202/204.
  Tests: `identity/v3/tokens/native_contracts_test.go:TestNativeTokenGetValidateRevoke`, `identity/v3/tokens/native_contracts_test.go:TestNativeTokenStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/tokens call only; the Connection authentication flow and Python identity Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams or Validate) and the documented extension merge; native URL construction, OkCodes, header handling and decoding are unchanged.

### `gophercloud:identity/v3/catalog.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/catalog.API.List`

- List GETs auth/catalog and yields catalog entries with their endpoints, stops at a null links.next, yields nothing for an empty catalog and reports native page status errors.
  Tests: `identity/v3/tokens/native_contracts_test.go:TestNativeCatalogList`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/catalog call only; the Connection authentication flow and Python identity Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams or Validate) and the documented extension merge; native URL construction, OkCodes, header handling and decoding are unchanged.

## References

- [tokens](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/tokens/requests.go)
- [catalog](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/catalog/requests.go)
- [Install](../../docs/install.md)
