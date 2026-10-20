# Extensions

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:compute/v2/extensions.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/extensions.API.List`

- List GETs extensions as one single page ignoring links, yields nothing for an empty array, surfaces io.EOF for a bodyless 204 and returns native pager status errors without operation context.
  Tests: `compute/v2/extensions/native_contracts_test.go:TestNativeComputeExtensionsRoutesDecodeAndActionURL`, `compute/v2/extensions/native_contracts_test.go:TestNativeComputeExtensionsStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/extensions call only; Python resource models and cloud helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams); native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/extensions.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/extensions.API.Get`

- Get GETs extensions/{alias}, decodes the extension envelope with updated kept as a string and accepts only 200.
  Tests: `compute/v2/extensions/native_contracts_test.go:TestNativeComputeExtensionsRoutesDecodeAndActionURL`, `compute/v2/extensions/native_contracts_test.go:TestNativeComputeExtensionsStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/extensions call only; Python resource models and cloud helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams); native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/extensions.ActionURL`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/extensions.API.ActionURL`

- ActionURL formats ResourceBase servers/{id}/action without sending a request.
  Tests: `compute/v2/extensions/native_contracts_test.go:TestNativeComputeExtensionsRoutesDecodeAndActionURL`, `compute/v2/extensions/native_contracts_test.go:TestNativeComputeExtensionsStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/extensions call only; Python resource models and cloud helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams); native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [compute extensions](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/extensions/delegate.go)
