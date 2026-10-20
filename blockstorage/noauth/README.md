# Noauth

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:blockstorage/noauth.NewBlockStorageNoAuthV2`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/noauth.API.NewBlockStorageNoAuthV2`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/noauth.WithNewBlockStorageNoAuthV2Options`

- NewBlockStorageNoAuthV2 makes no HTTP request and returns a block-storage ServiceClient sharing the provider, whose endpoint is the normalized CinderEndpoint plus the project half of a user:project provider token and a trailing slash; a missing endpoint or a token without exactly one colon returns the native error without operation context, while a nil option is wrapped.
  Tests: `blockstorage/noauth/native_contracts_test.go:TestNativeNoAuthClientEndpointAndProvider`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/noauth constructor only; openstacksdk has no noauth Cinder constructor and Python connection-level auth is tracked separately.
- The SDK adds only request.Wrap operation context for option errors; native endpoint construction and validation are unchanged.

### `gophercloud:blockstorage/noauth.NewBlockStorageNoAuthV3`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/noauth.API.NewBlockStorageNoAuthV3`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/noauth.WithNewBlockStorageNoAuthV3Options`

- NewBlockStorageNoAuthV3 makes no HTTP request and returns a block-storage ServiceClient sharing the provider, whose endpoint is the normalized CinderEndpoint plus the project half of a user:project provider token and a trailing slash; a missing endpoint or a token without exactly one colon returns the native error without operation context, while a nil option is wrapped.
  Tests: `blockstorage/noauth/native_contracts_test.go:TestNativeNoAuthClientEndpointAndProvider`.

Python/Go differences:

- This declaration maps the Gophercloud native blockstorage/noauth constructor only; openstacksdk has no noauth Cinder constructor and Python connection-level auth is tracked separately.
- The SDK adds only request.Wrap operation context for option errors; native endpoint construction and validation are unchanged.

## References

- [noauth](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/noauth/requests.go)
- [`noauth.NewClient`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/noauth/requests.go)
