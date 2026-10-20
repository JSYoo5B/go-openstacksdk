# APIversions

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:compute/apiversions.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/apiversions.API.List`

- List GETs the endpoint cut before its first version segment (dropping query) as one single page, decodes versions and returns native pager status errors without operation context.
  Tests: `compute/apiversions/native_contracts_test.go:TestNativeComputeAPIVersionsUseTheUnversionedRoot`, `compute/apiversions/native_contracts_test.go:TestNativeComputeAPIVersionsStatusesAndMissingVersion`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/apiversions call only; Python resource models and cloud helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams); native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/apiversions.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/apiversions.API.Get`

- Get GETs {root}/{version trimmed of a trailing slash}/, decodes the version object, returns ErrVersionNotFound with operation context when it is absent and accepts only 200.
  Tests: `compute/apiversions/native_contracts_test.go:TestNativeComputeAPIVersionsUseTheUnversionedRoot`, `compute/apiversions/native_contracts_test.go:TestNativeComputeAPIVersionsStatusesAndMissingVersion`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/apiversions call only; Python resource models and cloud helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams); native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/apiversions/requests.go)
- [Microversions](../../docs/microversions.md)
