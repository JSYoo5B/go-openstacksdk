# Usage

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:compute/v2/usage.SingleTenant`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/usage.API.SingleTenant`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/usage.WithSingleTenantOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/usage.WithSingleTenantQuery`

- SingleTenant GETs os-simple-tenant-usage/{tenantID} with offset-less wall-clock start/end, limit, marker and extension query, yields one tenant_usage per page following tenant_usage_links next, treats a missing tenant_usage as empty, surfaces io.EOF for a bodyless 204, rejects a nil option before HTTP and returns native pager status errors without operation context.
  Tests: `compute/v2/usage/native_contracts_test.go:TestNativeSingleTenantUsageQueryPagingAndDecode`, `compute/v2/usage/native_contracts_test.go:TestNativeSingleTenantUsageStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/usage call only; Python resource models and cloud helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams); native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/usage/requests.go)
