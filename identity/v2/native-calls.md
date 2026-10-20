# Native Calls

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:identity/v2/extensions.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/extensions.API.Get`

- Get sends GET extensions/{alias} on the legacy Keystone v2.0 endpoint, decodes the common extension envelope with a missing or null envelope as a nil result, and accepts only HTTP 200.
  Tests: `identity/v2/extensions/native_contracts_test.go:TestNativeV2ExtensionRoutesValuesEnvelopeAndDecode`, `identity/v2/extensions/native_contracts_test.go:TestNativeV2ExtensionStatusesAndListEnvelope`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/extensions call only; Python openstacksdk identity v2 Extension resources and cloud helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/extensions.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/extensions.API.List`

- List sends one GET extensions and decodes only the v2 {"extensions":{"values":[...]}} envelope as a single page that never follows links; a flat extensions array is a JSON type error, an empty page ends cleanly, 404 fails with Expected [200,204,300] and a bodyless 204 yields io.EOF.
  Tests: `identity/v2/extensions/native_contracts_test.go:TestNativeV2ExtensionRoutesValuesEnvelopeAndDecode`, `identity/v2/extensions/native_contracts_test.go:TestNativeV2ExtensionStatusesAndListEnvelope`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/extensions call only; Python openstacksdk identity v2 Extension resources and cloud helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/roles.AddUser`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/roles.API.AddUser`

- AddUser is an admin-only OS-KSADM call that sends a bodyless PUT tenants/{tenant}/users/{user}/roles/OS-KSADM/{role} and accepts HTTP 200 or 201.
  Tests: `identity/v2/roles/native_contracts_test.go:TestNativeV2RoleRoutesAndDecode`, `identity/v2/roles/native_contracts_test.go:TestNativeV2RoleStatusesAndListPager`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/roles call only; Python openstacksdk identity v2 Role resources and cloud role-assignment helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/roles.DeleteUser`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/roles.API.DeleteUser`

- DeleteUser is an admin-only OS-KSADM call that sends DELETE tenants/{tenant}/users/{user}/roles/OS-KSADM/{role} and accepts HTTP 202 or 204.
  Tests: `identity/v2/roles/native_contracts_test.go:TestNativeV2RoleRoutesAndDecode`, `identity/v2/roles/native_contracts_test.go:TestNativeV2RoleStatusesAndListPager`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/roles call only; Python openstacksdk identity v2 Role resources and cloud role-assignment helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/roles.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/roles.API.List`

- List is an admin-only call that sends one GET OS-KSADM/roles as a single page ignoring roles_links, decodes untagged Role fields case-insensitively (serviceId but not service_id), ends cleanly on an empty page, fails 404 with Expected [200,204,300] and yields io.EOF for a bodyless 204.
  Tests: `identity/v2/roles/native_contracts_test.go:TestNativeV2RoleRoutesAndDecode`, `identity/v2/roles/native_contracts_test.go:TestNativeV2RoleStatusesAndListPager`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/roles call only; Python openstacksdk identity v2 Role resources and cloud role-assignment helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/tenants.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.WithCreateField`

- Create is an admin call that sends POST tenants with {"tenant":{...}} requiring name and omitting empty description and enabled, rejects missing name, core-field extensions and nil options before HTTP, decodes the tenant envelope and accepts HTTP 200 or 201.
  Tests: `identity/v2/tenants/native_contracts_test.go:TestNativeV2TenantRoutesBodiesPagingAndDecode`, `identity/v2/tenants/native_contracts_test.go:TestNativeV2TenantStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/tenants call only; Python openstacksdk identity v2 Tenant resources and cloud project helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/tenants.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.API.Delete`

- Delete is an admin call that sends DELETE tenants/{id} and accepts HTTP 202 or 204.
  Tests: `identity/v2/tenants/native_contracts_test.go:TestNativeV2TenantRoutesBodiesPagingAndDecode`, `identity/v2/tenants/native_contracts_test.go:TestNativeV2TenantStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/tenants call only; Python openstacksdk identity v2 Tenant resources and cloud project helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/tenants.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.API.Get`

- Get is an admin call that sends GET tenants/{id}, decodes the tenant envelope with a missing or null envelope as nil, and accepts only HTTP 200.
  Tests: `identity/v2/tenants/native_contracts_test.go:TestNativeV2TenantRoutesBodiesPagingAndDecode`, `identity/v2/tenants/native_contracts_test.go:TestNativeV2TenantStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/tenants call only; Python openstacksdk identity v2 Tenant resources and cloud project helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/tenants.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.WithListQuery`

- List sends GET tenants with marker and limit plus extra query values, follows only tenants_links rel=next and ignores links.next, rejects nil options and blank query keys before HTTP, ends cleanly on an empty page, fails 404 with Expected [200,204,300] and yields io.EOF for a bodyless 204.
  Tests: `identity/v2/tenants/native_contracts_test.go:TestNativeV2TenantRoutesBodiesPagingAndDecode`, `identity/v2/tenants/native_contracts_test.go:TestNativeV2TenantStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/tenants call only; Python openstacksdk identity v2 Tenant resources and cloud project helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/tenants.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.WithUpdateField`

- Update is an admin call that sends PUT tenants/{id} with {"tenant":{...}} omitting empty fields (an empty UpdateOpts sends {"tenant":{}} and a pointer description may send an empty string), rejects core-field extensions and nil options before HTTP and accepts only HTTP 200.
  Tests: `identity/v2/tenants/native_contracts_test.go:TestNativeV2TenantRoutesBodiesPagingAndDecode`, `identity/v2/tenants/native_contracts_test.go:TestNativeV2TenantStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/tenants call only; Python openstacksdk identity v2 Tenant resources and cloud project helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/tokens.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tokens.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tokens.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tokens.WithCreateField`

- Create sends POST tokens with {"auth":{"passwordCredentials":{username,password}}} when Password is set (dropping TokenID) or {"auth":{"token":{"id"}}} otherwise plus non-empty tenantId and tenantName, rejects a missing username or token, core-field and passwordCredentials extensions and nil options before HTTP, still sends an existing provider X-Auth-Token because native OmitHeaders runs before token injection, decodes access.token (RFC3339Milli expires with a literal Z), access.serviceCatalog and access.user into Authentication, and accepts HTTP 200 or 203.
  Tests: `identity/v2/tokens/native_contracts_test.go:TestNativeV2TokenRoutesBodiesAndAccessDecode`, `identity/v2/tokens/native_contracts_test.go:TestNativeV2TokenStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/tokens call only; Python keystoneauth v2 password and token plugins and Connection session authentication are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/tokens.CreateURL`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tokens.API.CreateURL`

- CreateURL formats ResourceBase tokens without sending a request.
  Tests: `identity/v2/tokens/native_contracts_test.go:TestNativeV2TokenRoutesBodiesAndAccessDecode`, `identity/v2/tokens/native_contracts_test.go:TestNativeV2TokenStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/tokens call only; Python keystoneauth v2 password and token plugins and Connection session authentication are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/tokens.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tokens.API.Get`

- Get is an admin token validation call that sends GET tokens/{token} with the provider X-Auth-Token, decodes the same access envelope into Authentication with unscoped responses leaving tenant, catalog and user empty, fails on a missing or offset expires, and accepts HTTP 200 or 203.
  Tests: `identity/v2/tokens/native_contracts_test.go:TestNativeV2TokenRoutesBodiesAndAccessDecode`, `identity/v2/tokens/native_contracts_test.go:TestNativeV2TokenStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/tokens call only; Python keystoneauth v2 password and token plugins and Connection session authentication are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/tokens.GetURL`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tokens.API.GetURL`

- GetURL formats ResourceBase tokens/{token} without escaping the token and without sending a request.
  Tests: `identity/v2/tokens/native_contracts_test.go:TestNativeV2TokenRoutesBodiesAndAccessDecode`, `identity/v2/tokens/native_contracts_test.go:TestNativeV2TokenStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/tokens call only; Python keystoneauth v2 password and token plugins and Connection session authentication are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/users.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.WithCreateField`

- Create is an admin-only call that sends POST users with {"user":{...}} omitting empty fields, requires name or username with gophercloud.ErrMissingInput before HTTP, rejects core-field extensions and nil options, decodes the user envelope where TenantID reads only tenant_id although the request sends tenantId, and accepts HTTP 200 or 201.
  Tests: `identity/v2/users/native_contracts_test.go:TestNativeV2UserRoutesBodiesAndDecode`, `identity/v2/users/native_contracts_test.go:TestNativeV2UserStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/users call only; Python openstacksdk identity v2 User resources and cloud user helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/users.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.API.Delete`

- Delete is an admin-only call that sends DELETE users/{id}, accepts HTTP 202 or 204 and always returns a nil User because the native call never decodes a response body.
  Tests: `identity/v2/users/native_contracts_test.go:TestNativeV2UserRoutesBodiesAndDecode`, `identity/v2/users/native_contracts_test.go:TestNativeV2UserStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/users call only; Python openstacksdk identity v2 User resources and cloud user helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/users.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.API.Get`

- Get is an admin-only call that sends GET users/{id} with an ID or name, decodes the user envelope with a missing or null envelope as nil, and accepts only HTTP 200.
  Tests: `identity/v2/users/native_contracts_test.go:TestNativeV2UserRoutesBodiesAndDecode`, `identity/v2/users/native_contracts_test.go:TestNativeV2UserStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/users call only; Python openstacksdk identity v2 User resources and cloud user helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/users.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.API.List`

- List is an admin-only call that sends one GET users as a single page ignoring users_links, decodes untagged User fields case-insensitively and tenant_id only, ends cleanly on an empty page, fails 404 with Expected [200,204,300] and yields io.EOF for a bodyless 204.
  Tests: `identity/v2/users/native_contracts_test.go:TestNativeV2UserRoutesBodiesAndDecode`, `identity/v2/users/native_contracts_test.go:TestNativeV2UserStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/users call only; Python openstacksdk identity v2 User resources and cloud user helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/users.ListRoles`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.API.ListRoles`

- ListRoles is an admin-only call that sends one GET tenants/{tenant}/users/{user}/roles as a single page ignoring roles_links, ends cleanly on an empty page, fails 404 with Expected [200,204,300] and yields io.EOF for a bodyless 204.
  Tests: `identity/v2/users/native_contracts_test.go:TestNativeV2UserRoutesBodiesAndDecode`, `identity/v2/users/native_contracts_test.go:TestNativeV2UserStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/users call only; Python openstacksdk identity v2 User resources and cloud user helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/users.ResourceURL`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.API.ResourceURL`

- ResourceURL formats ResourceBase users/{id} without sending a request.
  Tests: `identity/v2/users/native_contracts_test.go:TestNativeV2UserRoutesBodiesAndDecode`, `identity/v2/users/native_contracts_test.go:TestNativeV2UserStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/users call only; Python openstacksdk identity v2 User resources and cloud user helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v2/users.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.WithUpdateField`

- Update is an admin-only call that sends PUT users/{id} with {"user":{...}} omitting empty fields (an empty UpdateOpts sends {"user":{}}), rejects core-field extensions and nil options before HTTP and accepts only HTTP 200.
  Tests: `identity/v2/users/native_contracts_test.go:TestNativeV2UserRoutesBodiesAndDecode`, `identity/v2/users/native_contracts_test.go:TestNativeV2UserStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v2/users call only; Python openstacksdk identity v2 User resources and cloud user helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [extensions](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v2/extensions/requests.go)
- [roles](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v2/roles/requests.go)
- [tenants](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v2/tenants/requests.go)
- [tokens](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v2/tokens/requests.go)
- [users](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v2/users/requests.go)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/common/extensions/results.go)
- [tokens README](tokens/README.md)
