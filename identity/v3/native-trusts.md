# Native Trusts

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:identity/v3/trusts.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithCreateField`

- Create POSTs {"trust":{...}} to OS-TRUST/trusts always sending impersonation (false included), formatting expires_at with a literal Z and no UTC conversion, keeping roles and extensions inside the envelope, requires trustee and trustor user IDs and rejects a core-key collision before HTTP, decodes RFC3339 times and accepts only 201.
  Tests: `identity/v3/trusts/native_contracts_test.go:TestNativeTrustRoutesBodiesPagingAndDecode`, `identity/v3/trusts/native_contracts_test.go:TestNativeTrustStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/trusts call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/trusts.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.API.Get`

- Get GETs OS-TRUST/trusts/{id}, decodes the trust with a null deleted_at as zero time and accepts only 200.
  Tests: `identity/v3/trusts/native_contracts_test.go:TestNativeTrustRoutesBodiesPagingAndDecode`, `identity/v3/trusts/native_contracts_test.go:TestNativeTrustStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/trusts call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/trusts.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithListQuery`

- List GETs OS-TRUST/trusts with trustor/trustee filters and extension query values, follows the Keystone links.next string until null and rejects a nil option before HTTP.
  Tests: `identity/v3/trusts/native_contracts_test.go:TestNativeTrustRoutesBodiesPagingAndDecode`, `identity/v3/trusts/native_contracts_test.go:TestNativeTrustStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/trusts call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/trusts.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.API.Delete`

- Delete DELETEs OS-TRUST/trusts/{id} and accepts only 202/204.
  Tests: `identity/v3/trusts/native_contracts_test.go:TestNativeTrustRoutesBodiesPagingAndDecode`, `identity/v3/trusts/native_contracts_test.go:TestNativeTrustStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/trusts call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/trusts.ListRoles`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.API.ListRoles`

- ListRoles GETs OS-TRUST/trusts/{id}/roles and follows the Keystone links.next string until null.
  Tests: `identity/v3/trusts/native_contracts_test.go:TestNativeTrustRoutesBodiesPagingAndDecode`, `identity/v3/trusts/native_contracts_test.go:TestNativeTrustStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/trusts call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/trusts.GetRole`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.API.GetRole`

- GetRole GETs OS-TRUST/trusts/{id}/roles/{role}, decodes the role envelope and accepts only 200.
  Tests: `identity/v3/trusts/native_contracts_test.go:TestNativeTrustRoutesBodiesPagingAndDecode`, `identity/v3/trusts/native_contracts_test.go:TestNativeTrustStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/trusts call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/trusts.CheckRole`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.API.CheckRole`

- CheckRole HEADs OS-TRUST/trusts/{id}/roles/{role} and accepts only the native HEAD default 200.
  Tests: `identity/v3/trusts/native_contracts_test.go:TestNativeTrustRoutesBodiesPagingAndDecode`, `identity/v3/trusts/native_contracts_test.go:TestNativeTrustStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/trusts call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/projects.ListAvailable`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.ListAvailable`

- ListAvailable GETs auth/projects without options, follows the Keystone links.next string until null, yields nothing for an empty list and reports native page status errors.
  Tests: `identity/v3/projects/native_available_test.go:TestNativeProjectListAvailable`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/projects call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/users.ChangePassword`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ChangePassword`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithChangePasswordOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithChangePasswordField`

- ChangePassword POSTs {"user":{"original_password":...,"password":...}} to users/{user}/password always sending both values (empty included) with extensions inside the envelope, rejects a core-key collision and nil option before HTTP, and accepts only 204.
  Tests: `identity/v3/projects/native_available_test.go:TestNativeUserChangePassword`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/users call only; Python identity Resource parity is tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [trusts](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/trusts/requests.go)
- [projects](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/projects/requests.go)
- [users](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/users/requests.go)
- [Memberships](users/memberships.md)
