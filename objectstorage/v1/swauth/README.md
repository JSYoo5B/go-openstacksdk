# Swauth

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:objectstorage/v1/swauth.Auth`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/swauth.API.Auth`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/swauth.WithAuthOptions`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/swauth.WithAuthHeader`

- Auth GETs auth/v1.0 under the provider IdentityBase with required X-Auth-User and X-Auth-Key headers plus the provider X-Auth-Token and extension headers, accepts only 200, and reads Token, StorageURL and CDNURL from the X-Auth-Token, X-Storage-Url and X-CDN-Management-Url response headers; missing credentials, core-header extensions and nil options fail before HTTP.
  Tests: `objectstorage/v1/swauth/native_contracts_test.go:TestNativeSwauthAuthHeadersAndResult`, `objectstorage/v1/swauth/native_contracts_test.go:TestNativeSwauthStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/swauth call only; openstacksdk relies on keystoneauth plugins for TempAuth and that parity is tracked separately.
- The SDK adds only request.Wrap operation context and the documented header extension merge; native URL construction, OkCodes and header decoding are unchanged.

### `gophercloud:objectstorage/v1/swauth.NewObjectStorageV1`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/swauth.API.NewObjectStorageV1`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/swauth.WithNewObjectStorageV1Options`

- NewObjectStorageV1 performs one swauth Auth request and returns a ServiceClient with an empty Type whose endpoint is the normalized X-Storage-Url, sharing the ProviderClient and writing the swauth token into its TokenID so every client on that provider switches token; native authentication and validation errors return without operation context and leave the token unchanged, while a nil option is wrapped.
  Tests: `objectstorage/v1/swauth/native_contracts_test.go:TestNativeSwauthNewObjectStorageV1SharesProviderToken`.

Python/Go differences:

- This declaration maps the Gophercloud native objectstorage/v1/swauth constructor only; openstacksdk relies on keystoneauth plugins for TempAuth and that parity is tracked separately.
- The SDK adds only request.Wrap operation context for option errors; the native shared-provider token write is preserved and documented.

## References

- [swauth](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/objectstorage/v1/swauth/requests.go)
