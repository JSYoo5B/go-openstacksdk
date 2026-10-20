# Native Admin A

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:identity/v3/domains.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithCreateField`

- Create POSTs domains with a domain envelope, requires name, omits empty optional fields, sends a non-nil enabled false, merges extension fields inside the envelope while rejecting core-key collisions and nil options before HTTP, decodes the domain envelope and accepts only 201.
  Tests: `identity/v3/domains/native_contracts_test.go:TestNativeDomainRoutesBodiesPagingAndDecode`, `identity/v3/domains/native_contracts_test.go:TestNativeDomainStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/domains call only; Python openstacksdk identity Domain proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/domains.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Get`

- Get GETs domains/{id}, decodes the domain envelope (a missing or null envelope is nil without error) and accepts only 200.
  Tests: `identity/v3/domains/native_contracts_test.go:TestNativeDomainRoutesBodiesPagingAndDecode`, `identity/v3/domains/native_contracts_test.go:TestNativeDomainStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/domains call only; Python openstacksdk identity Domain proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/domains.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithUpdateField`

- Update PATCHes domains/{id} with a domain envelope (an empty update sends an empty envelope and a pointer to an empty description clears it), merges extension fields inside the envelope, rejects core-key collisions and nil options before HTTP and accepts only 200.
  Tests: `identity/v3/domains/native_contracts_test.go:TestNativeDomainRoutesBodiesPagingAndDecode`, `identity/v3/domains/native_contracts_test.go:TestNativeDomainStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/domains call only; Python openstacksdk identity Domain proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/domains.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Delete`

- Delete sends DELETE domains/{id} and accepts the native 202 or 204 default.
  Tests: `identity/v3/domains/native_contracts_test.go:TestNativeDomainRoutesBodiesPagingAndDecode`, `identity/v3/domains/native_contracts_test.go:TestNativeDomainStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/domains call only; Python openstacksdk identity Domain proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/domains.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithListQuery`

- List GETs domains with sorted enabled, name, limit and extension query values, follows the Keystone links.next string until null, yields nothing for an empty page, ends bodyless 204 with io.EOF, reports 404 as the native pager error and rejects a nil option before HTTP.
  Tests: `identity/v3/domains/native_contracts_test.go:TestNativeDomainRoutesBodiesPagingAndDecode`, `identity/v3/domains/native_contracts_test.go:TestNativeDomainStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/domains call only; Python openstacksdk identity Domain proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/domains.ListAvailable`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.ListAvailable`

- ListAvailable GETs auth/domains without options, follows the Keystone links.next string until null, yields nothing for an empty page, ends bodyless 204 with io.EOF and reports 404 as the native pager error.
  Tests: `identity/v3/domains/native_contracts_test.go:TestNativeDomainRoutesBodiesPagingAndDecode`, `identity/v3/domains/native_contracts_test.go:TestNativeDomainStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/domains call only; Python openstacksdk identity Domain proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/groups.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithCreateField`

- Create POSTs groups with a group envelope, requires name, flattens Extra keys into the envelope, merges extension fields there while rejecting core-key or Extra collisions and nil options before HTTP, collects unknown response keys into Extra and accepts only 201.
  Tests: `identity/v3/groups/native_contracts_test.go:TestNativeGroupRoutesBodiesPagingAndDecode`, `identity/v3/groups/native_contracts_test.go:TestNativeGroupStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/groups call only; Python openstacksdk identity Group proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/groups.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.Get`

- Get GETs groups/{id}, decodes the group envelope with remaining keys or a nested extra object as Extra and accepts only 200.
  Tests: `identity/v3/groups/native_contracts_test.go:TestNativeGroupRoutesBodiesPagingAndDecode`, `identity/v3/groups/native_contracts_test.go:TestNativeGroupStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/groups call only; Python openstacksdk identity Group proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/groups.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithUpdateField`

- Update PATCHes groups/{id} with a group envelope, sends a pointer to an empty description, flattens Extra keys (nil as null) and extension fields into the envelope, rejects nil options before HTTP and accepts only 200.
  Tests: `identity/v3/groups/native_contracts_test.go:TestNativeGroupRoutesBodiesPagingAndDecode`, `identity/v3/groups/native_contracts_test.go:TestNativeGroupStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/groups call only; Python openstacksdk identity Group proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/groups.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.Delete`

- Delete sends DELETE groups/{id} and accepts the native 202 or 204 default.
  Tests: `identity/v3/groups/native_contracts_test.go:TestNativeGroupRoutesBodiesPagingAndDecode`, `identity/v3/groups/native_contracts_test.go:TestNativeGroupStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/groups call only; Python openstacksdk identity Group proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/groups.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.WithListQuery`

- List GETs groups with sorted domain_id, name and extension query values, follows the Keystone links.next string until null, yields nothing for an empty page, ends bodyless 204 with io.EOF, reports 404 as the native pager error and rejects a nil option before HTTP. A non-empty Filters map adds NAME__COMPARATOR keys (malformed keys fail before HTTP with an unwrapped InvalidListFilter) but native BuildQueryString also leaks a stray '-' parameter holding the map's text.
  Tests: `identity/v3/groups/native_contracts_test.go:TestNativeGroupRoutesBodiesPagingAndDecode`, `identity/v3/groups/native_contracts_test.go:TestNativeGroupStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/groups call only; Python openstacksdk identity Group proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/projects.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithCreateField`

- Create POSTs projects with a project envelope, requires name, sends enabled/is_domain pointers, tags and options, flattens Extra keys and extension fields into the envelope while rejecting core-key or Extra collisions and nil options before HTTP, collects unknown keys including links into Extra and accepts the native 201 or 202 POST default.
  Tests: `identity/v3/projects/native_admin_test.go:TestNativeProjectAdminRoutesBodiesPagingAndDecode`, `identity/v3/projects/native_admin_test.go:TestNativeProjectAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/projects call only; Python openstacksdk identity Project proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/projects.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.Get`

- Get GETs projects/{id}, decodes the project envelope with tags, options and remaining keys as Extra and accepts only 200.
  Tests: `identity/v3/projects/native_admin_test.go:TestNativeProjectAdminRoutesBodiesPagingAndDecode`, `identity/v3/projects/native_admin_test.go:TestNativeProjectAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/projects call only; Python openstacksdk identity Project proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/projects.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithUpdateField`

- Update PATCHes projects/{id} with a project envelope where a pointer to an empty tag list sends tags [] and a pointer to an empty description clears it, merges Extra and extension fields into the envelope, rejects nil options before HTTP and accepts only 200.
  Tests: `identity/v3/projects/native_admin_test.go:TestNativeProjectAdminRoutesBodiesPagingAndDecode`, `identity/v3/projects/native_admin_test.go:TestNativeProjectAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/projects call only; Python openstacksdk identity Project proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/projects.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.Delete`

- Delete sends DELETE projects/{id} and accepts the native 202 or 204 default.
  Tests: `identity/v3/projects/native_admin_test.go:TestNativeProjectAdminRoutesBodiesPagingAndDecode`, `identity/v3/projects/native_admin_test.go:TestNativeProjectAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/projects call only; Python openstacksdk identity Project proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/projects.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithListQuery`

- List GETs projects with sorted domain_id, enabled, is_domain, name, parent_id, tags, tags-any, not-tags, not-tags-any, limit and extension query values, follows the Keystone links.next string until null, yields nothing for an empty page, ends bodyless 204 with io.EOF, reports 404 as the native pager error and rejects a nil option before HTTP. A non-empty Filters map adds NAME__COMPARATOR keys (malformed keys fail before HTTP with an unwrapped InvalidListFilter) but native BuildQueryString also leaks a stray '-' parameter holding the map's text.
  Tests: `identity/v3/projects/native_admin_test.go:TestNativeProjectAdminRoutesBodiesPagingAndDecode`, `identity/v3/projects/native_admin_test.go:TestNativeProjectAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/projects call only; Python openstacksdk identity Project proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/projects.ListTags`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.ListTags`

- ListTags GETs projects/{id}/tags, decodes the top-level tags list, accepts only 200 and returns a non-nil zero Tags together with an error.
  Tests: `identity/v3/projects/native_admin_test.go:TestNativeProjectAdminRoutesBodiesPagingAndDecode`, `identity/v3/projects/native_admin_test.go:TestNativeProjectAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/projects call only; Python openstacksdk identity Project proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/projects.ModifyTags`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.ModifyTags`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithModifyTagsOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.WithModifyTagsField`

- ModifyTags PUTs projects/{id}/tags with an unwrapped tags body (an empty list is omitted, so it sends {}) and top-level extension fields, rejects a core tags extension before HTTP, decodes only top-level projects and links, accepts only 200 and returns a non-nil zero ProjectTags together with an error.
  Tests: `identity/v3/projects/native_admin_test.go:TestNativeProjectAdminRoutesBodiesPagingAndDecode`, `identity/v3/projects/native_admin_test.go:TestNativeProjectAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/projects call only; Python openstacksdk identity Project proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/projects.DeleteTags`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.DeleteTags`

- DeleteTags sends DELETE projects/{id}/tags and accepts only 204.
  Tests: `identity/v3/projects/native_admin_test.go:TestNativeProjectAdminRoutesBodiesPagingAndDecode`, `identity/v3/projects/native_admin_test.go:TestNativeProjectAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/projects call only; Python openstacksdk identity Project proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/users.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithCreateField`

- Create POSTs users with a user envelope, requires name, sends password, options and enabled pointers, flattens Extra keys and extension fields into the envelope while rejecting core-key collisions and nil options before HTTP, decodes string enabled values and the zone-less password_expires_at layout and accepts only 201.
  Tests: `identity/v3/users/native_admin_test.go:TestNativeUserAdminRoutesBodiesPagingAndDecode`, `identity/v3/users/native_admin_test.go:TestNativeUserAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/users call only; Python openstacksdk identity User proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/users.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.Get`

- Get GETs users/{id}, decodes enabled from bool, string or null, password_expires_at from the zone-less microsecond layout (null is zero, a trailing Z fails), keeps remaining keys except password_expires_at in Extra and accepts only 200.
  Tests: `identity/v3/users/native_admin_test.go:TestNativeUserAdminRoutesBodiesPagingAndDecode`, `identity/v3/users/native_admin_test.go:TestNativeUserAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/users call only; Python openstacksdk identity User proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/users.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithUpdateField`

- Update PATCHes users/{id} with a user envelope including a pointer to an empty description, enabled false and password, flattens Extra (nil as null) and extension fields into the envelope while rejecting Extra collisions and nil options before HTTP and accepts only 200.
  Tests: `identity/v3/users/native_admin_test.go:TestNativeUserAdminRoutesBodiesPagingAndDecode`, `identity/v3/users/native_admin_test.go:TestNativeUserAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/users call only; Python openstacksdk identity User proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/users.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.Delete`

- Delete sends DELETE users/{id} and accepts the native 202 or 204 default.
  Tests: `identity/v3/users/native_admin_test.go:TestNativeUserAdminRoutesBodiesPagingAndDecode`, `identity/v3/users/native_admin_test.go:TestNativeUserAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/users call only; Python openstacksdk identity User proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/users.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithListQuery`

- List GETs users with sorted domain_id, enabled, idp_id, name, password_expires_at, protocol_id, unique_id and extension query values, follows the Keystone links.next string until null, yields nothing for an empty page, ends bodyless 204 with io.EOF, reports 404 as the native pager error and rejects a nil option before HTTP. A non-empty Filters map adds NAME__COMPARATOR keys (malformed keys fail before HTTP with an unwrapped InvalidListFilter) but native BuildQueryString also leaks a stray '-' parameter holding the map's text.
  Tests: `identity/v3/users/native_admin_test.go:TestNativeUserAdminRoutesBodiesPagingAndDecode`, `identity/v3/users/native_admin_test.go:TestNativeUserAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/users call only; Python openstacksdk identity User proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/users.ListInGroup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ListInGroup`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithListInGroupOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithListInGroupQuery`

- ListInGroup GETs groups/{group}/users with the users ListOpts and extension query values, follows the Keystone links.next string until null, yields nothing for an empty page, ends bodyless 204 with io.EOF, reports 404 as the native pager error and rejects nil options before HTTP while malformed filter keys fail with an unwrapped InvalidListFilter.
  Tests: `identity/v3/users/native_admin_test.go:TestNativeUserAdminRoutesBodiesPagingAndDecode`, `identity/v3/users/native_admin_test.go:TestNativeUserAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/users call only; Python openstacksdk identity User proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/users.AddToGroup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.AddToGroup`

- AddToGroup PUTs groups/{group}/users/{user} without a body and accepts only 204.
  Tests: `identity/v3/users/native_admin_test.go:TestNativeUserAdminRoutesBodiesPagingAndDecode`, `identity/v3/users/native_admin_test.go:TestNativeUserAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/users call only; Python openstacksdk identity User proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/users.IsMemberOfGroup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.IsMemberOfGroup`

- IsMemberOfGroup sends HEAD groups/{group}/users/{user}, returns true for 204 and false without error for 404, and reports every other status as a native error expecting 204 or 404.
  Tests: `identity/v3/users/native_admin_test.go:TestNativeUserAdminRoutesBodiesPagingAndDecode`, `identity/v3/users/native_admin_test.go:TestNativeUserAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/users call only; Python openstacksdk identity User proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/users.RemoveFromGroup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.RemoveFromGroup`

- RemoveFromGroup sends DELETE groups/{group}/users/{user} and accepts only 204.
  Tests: `identity/v3/users/native_admin_test.go:TestNativeUserAdminRoutesBodiesPagingAndDecode`, `identity/v3/users/native_admin_test.go:TestNativeUserAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/users call only; Python openstacksdk identity User proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/roles.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithCreateField`

- Create POSTs roles with a role envelope, requires name, sends domain_id, description and options, flattens Extra and extension fields into the envelope while rejecting core-key collisions and nil options before HTTP, decodes null domain_id as empty, copies description into Extra alongside remaining keys and accepts only 201.
  Tests: `identity/v3/roles/native_contracts_test.go:TestNativeRoleRoutesBodiesPagingAndDecode`, `identity/v3/roles/native_contracts_test.go:TestNativeRoleStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/roles call only; Python openstacksdk identity Role proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/roles.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Get`

- Get GETs roles/{id}, decodes the role envelope with options and Extra (remaining keys plus a description copy) and accepts only 200.
  Tests: `identity/v3/roles/native_contracts_test.go:TestNativeRoleRoutesBodiesPagingAndDecode`, `identity/v3/roles/native_contracts_test.go:TestNativeRoleStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/roles call only; Python openstacksdk identity Role proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/roles.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithUpdateField`

- Update PATCHes roles/{id} with a role envelope including a pointer to an empty description, options and Extra (nil as null) plus extension fields, rejects nil options before HTTP and accepts only 200.
  Tests: `identity/v3/roles/native_contracts_test.go:TestNativeRoleRoutesBodiesPagingAndDecode`, `identity/v3/roles/native_contracts_test.go:TestNativeRoleStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/roles call only; Python openstacksdk identity Role proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/roles.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Delete`

- Delete sends DELETE roles/{id} and accepts the native 202 or 204 default.
  Tests: `identity/v3/roles/native_contracts_test.go:TestNativeRoleRoutesBodiesPagingAndDecode`, `identity/v3/roles/native_contracts_test.go:TestNativeRoleStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/roles call only; Python openstacksdk identity Role proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/roles.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithListQuery`

- List GETs roles with sorted domain_id, name and extension query values, follows the Keystone links.next string until null, yields nothing for an empty page, ends bodyless 204 with io.EOF, reports 404 as the native pager error and rejects a nil option before HTTP. A non-empty Filters map adds NAME__COMPARATOR keys (malformed keys fail before HTTP with an unwrapped InvalidListFilter) but native BuildQueryString also leaks a stray '-' parameter holding the map's text.
  Tests: `identity/v3/roles/native_contracts_test.go:TestNativeRoleRoutesBodiesPagingAndDecode`, `identity/v3/roles/native_contracts_test.go:TestNativeRoleStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/roles call only; Python openstacksdk identity Role proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/roles.Assign`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Assign`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithAssignOptions`

- Assign PUTs {projects|domains}/{id}/{users|groups}/{id}/roles/{role} or system/{users|groups}/{id}/roles/{role} without a body, requires exactly one of user/group and exactly one of project/domain/system before HTTP (ErrMissingInput with operation context), lets WithAssignOptions replace the positional options and accepts only 204.
  Tests: `identity/v3/roles/native_contracts_test.go:TestNativeRoleAssignmentRoutesQueryAndDecode`, `identity/v3/roles/native_contracts_test.go:TestNativeRoleStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/roles call only; Python openstacksdk identity Role proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/roles.Validate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Validate`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithValidateOptions`

- Validate sends HEAD to the same project, domain or system role assignment path, applies the same exactly-one actor and scope checks before HTTP and accepts only 204.
  Tests: `identity/v3/roles/native_contracts_test.go:TestNativeRoleAssignmentRoutesQueryAndDecode`, `identity/v3/roles/native_contracts_test.go:TestNativeRoleStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/roles call only; Python openstacksdk identity Role proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/roles.Unassign`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.Unassign`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithUnassignOptions`

- Unassign sends DELETE to the same project, domain or system role assignment path, applies the same exactly-one actor and scope checks before HTTP and accepts only 204.
  Tests: `identity/v3/roles/native_contracts_test.go:TestNativeRoleAssignmentRoutesQueryAndDecode`, `identity/v3/roles/native_contracts_test.go:TestNativeRoleStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/roles call only; Python openstacksdk identity Role proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/roles.ListAssignmentsOnResource`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.ListAssignmentsOnResource`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithListAssignmentsOnResourceOptions`

- ListAssignmentsOnResource GETs {projects|domains}/{id}/{users|groups}/{id}/roles or system/{users|groups}/{id}/roles, follows the Keystone links.next string, ends bodyless 204 with io.EOF, reports 404 as the native pager error, wraps a nil option but yields an unwrapped ErrMissingInput without HTTP when called without options.
  Tests: `identity/v3/roles/native_contracts_test.go:TestNativeRoleAssignmentRoutesQueryAndDecode`, `identity/v3/roles/native_contracts_test.go:TestNativeRoleStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/roles call only; Python openstacksdk identity Role proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/roles.ListAssignments`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.ListAssignments`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithListAssignmentsOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithListAssignmentsQuery`

- ListAssignments GETs role_assignments with sorted group.id, role.id, scope.domain.id, scope.project.id, scope.system, user.id, effective, include_names, include_subtree (pointer booleans sent as true/false text) and extension query values, follows the Keystone links.next string, decodes role, scope (System only for system scope), user and group, ends bodyless 204 with io.EOF and rejects a nil option before HTTP.
  Tests: `identity/v3/roles/native_contracts_test.go:TestNativeRoleAssignmentRoutesQueryAndDecode`, `identity/v3/roles/native_contracts_test.go:TestNativeRoleStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/roles call only; Python openstacksdk identity Role proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/roles.CreateRoleInferenceRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.CreateRoleInferenceRule`

- CreateRoleInferenceRule PUTs roles/{prior}/implies/{implied} without a body, decodes the top-level role_inference prior_role/implies and links, accepts only 201 and returns a non-nil zero rule with an error.
  Tests: `identity/v3/roles/native_contracts_test.go:TestNativeRoleInferenceRoutesAndDecode`, `identity/v3/roles/native_contracts_test.go:TestNativeRoleStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/roles call only; Python openstacksdk identity Role proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/roles.GetRoleInferenceRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.GetRoleInferenceRule`

- GetRoleInferenceRule GETs roles/{prior}/implies/{implied}, decodes the top-level role_inference and links, accepts only 200 and returns a non-nil zero rule together with an error.
  Tests: `identity/v3/roles/native_contracts_test.go:TestNativeRoleInferenceRoutesAndDecode`, `identity/v3/roles/native_contracts_test.go:TestNativeRoleStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/roles call only; Python openstacksdk identity Role proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/roles.DeleteRoleInferenceRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.DeleteRoleInferenceRule`

- DeleteRoleInferenceRule sends DELETE roles/{prior}/implies/{implied} and accepts only 204.
  Tests: `identity/v3/roles/native_contracts_test.go:TestNativeRoleInferenceRoutesAndDecode`, `identity/v3/roles/native_contracts_test.go:TestNativeRoleStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/roles call only; Python openstacksdk identity Role proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/roles.ListRoleInferenceRules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.ListRoleInferenceRules`

- ListRoleInferenceRules GETs role_inferences once (links.next is kept in Links but not followed), decodes each prior_role with its implies list, accepts only 200 and returns a non-nil zero list together with an error.
  Tests: `identity/v3/roles/native_contracts_test.go:TestNativeRoleInferenceRoutesAndDecode`, `identity/v3/roles/native_contracts_test.go:TestNativeRoleStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/roles call only; Python openstacksdk identity Role proxy and Resource parity are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/osinherit.Assign`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Assign`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.WithAssignOptions`

- Assign PUTs OS-INHERIT/{projects|domains}/{id}/{users|groups}/{id}/roles/{role}/inherited_to_projects without a body, requires exactly one of user/group and exactly one of project/domain before HTTP (ErrMissingInput with operation context), lets WithAssignOptions replace the positional options and accepts only 204.
  Tests: `identity/v3/osinherit/native_contracts_test.go:TestNativeInheritRoutes`, `identity/v3/osinherit/native_contracts_test.go:TestNativeInheritStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/osinherit call only; Python openstacksdk inherited role assignment helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/osinherit.Validate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Validate`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.WithValidateOptions`

- Validate sends HEAD to the same OS-INHERIT inherited_to_projects path, applies the same exactly-one actor and target checks before HTTP and accepts only 204.
  Tests: `identity/v3/osinherit/native_contracts_test.go:TestNativeInheritRoutes`, `identity/v3/osinherit/native_contracts_test.go:TestNativeInheritStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/osinherit call only; Python openstacksdk inherited role assignment helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:identity/v3/osinherit.Unassign`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.API.Unassign`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/osinherit.WithUnassignOptions`

- Unassign sends DELETE to the same OS-INHERIT inherited_to_projects path, applies the same exactly-one actor and target checks before HTTP and accepts only 204.
  Tests: `identity/v3/osinherit/native_contracts_test.go:TestNativeInheritRoutes`, `identity/v3/osinherit/native_contracts_test.go:TestNativeInheritStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native identity/v3/osinherit call only; Python openstacksdk inherited role assignment helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [domains](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/domains/requests.go)
- [groups](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/groups/requests.go)
- [projects](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/projects/requests.go)
- [users](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/users/requests.go)
- [roles](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/roles/requests.go)
- [osinherit](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/osinherit/requests.go)
- [Native Trusts](native-trusts.md)
- [Memberships](users/memberships.md)
