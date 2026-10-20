# Python Parity

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v2/extensions`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/extensions.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/extensions.API.List`

- List sends one GET extensions and yields the rows of extensions.values with alias, name, namespace, updated, description and links, matching the custom Python Extension.list.
  Tests: `identity/v2/extensions/python_parity_test.go:TestPythonExtensionListAndGet`.

Python/Go differences:

- Python Extension.list reads response JSON without status checking; the Go pager accepts only 200, 204 and 300 and returns other statuses as errors.
- Go yields the Gophercloud common Extension model, where Python updated_at is the Updated string; Python yields Extension Resources.

### `python:identity/v2/get_extension`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/extensions.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/extensions.API.Get`

- Get(alias) sends GET extensions/{alias}, unwraps the extension envelope and returns a 404 error where Python raises NotFoundException.
  Tests: `identity/v2/extensions/python_parity_test.go:TestPythonExtensionListAndGet`.

Python/Go differences:

- Python accepts an alias string or Extension Resource; Go accepts only the alias string.
- Python accepts any status below 400; the native Get accepts only 200 and returns nil without error when the extension envelope is missing.

### `python:identity/v2/create_role`

Status: `unsupported`.

Unresolved boundaries:

- Keystone removed the v2.0 API in Queens and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:identity/v2/delete_role`

Status: `unsupported`.

Unresolved boundaries:

- Keystone removed the v2.0 API in Queens and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:identity/v2/find_role`

Status: `unsupported`.

Unresolved boundaries:

- Keystone removed the v2.0 API in Queens and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:identity/v2/get_role`

Status: `unsupported`.

Unresolved boundaries:

- Keystone removed the v2.0 API in Queens and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:identity/v2/update_role`

Status: `unsupported`.

Unresolved boundaries:

- Keystone removed the v2.0 API in Queens and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:identity/v2/roles`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/roles.API.List`

- List sends the same single GET OS-KSADM/roles as Python roles() without arguments and decodes id, name and description, but does not follow roles_links.
  Tests: `identity/v2/roles/python_parity_test.go:TestPythonRoleListDefaultRequest`.

Python/Go differences:

- Go returns native Role values decoded without JSON tags; Python returns Role Resources with is_enabled converted by BoolStr.

Unresolved boundaries:

- Keystone removed the v2.0 API in Queens and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:identity/v2/create_tenant`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.API.Create`

- Create sends POST tenants with the {"tenant": {...}} envelope, the request Python create_tenant sends, and decodes the fields Python exposes.
  Tests: `identity/v2/tenants/python_parity_test.go:TestPythonTenantCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v2/native-calls.md.
- Python takes keyword Resource attributes and returns a mutable Resource; Go takes typed CreateOpts and returns the native Gophercloud model pointer.
- tenants.CreateOpts.Name is required, so Go rejects before HTTP the nameless {"tenant": {}} request Python would send; Python attrs map to Name, Description and Enabled.

### `python:identity/v2/delete_tenant`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove(resource.ID(id)) sends DELETE tenants/{id}, succeeds on 404 like the Python ignore_missing=True default, and WithMissingError returns resource.ErrNotFound like ignore_missing=False.
  Tests: `identity/v2/tenants/python_parity_test.go:TestPythonTenantDeleteIgnoreMissing`.

Python/Go differences:

- Python returns None and accepts any status below 400; Go returns only an error and the native Delete accepts 202 and 204.
- Calling the generated Delete directly returns the 404 error, so only Remove matches the Python ignore_missing=True default.

### `python:identity/v2/find_tenant`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Find(resource.ID(x), WithIgnoreMissing) followed on nil by Find(resource.Name(x)) sends GET tenants/{x} and then the unfiltered tenants list, returns the ID or unique name match, nil when missing with WithIgnoreMissing, resource.ErrNotFound when strict and resource.ErrAmbiguous for duplicate names.
  Tests: `identity/v2/tenants/python_parity_test.go:TestPythonTenantFindIDThenNameFallback`.

Python/Go differences:

- Go needs two Find calls; Python Resource.find does the ID GET and the list fallback in one call.
- Python falls back to the list after GET 400, 403 or 404; Go Find with WithIgnoreMissing returns nil only for 404, so the caller must continue to the name lookup after a 400 or 403 error.
- The Python list match compares both ID and name; the Go name lookup compares only the name. Duplicates raise DuplicateResource in Python and resource.ErrAmbiguous in Go.
- The Go name lookup follows tenants_links like Python's list fallback.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.

### `python:identity/v2/get_tenant`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.API.Get`

- Get sends GET tenants/{id}, the request Python get_tenant sends, and decodes the fields Python exposes.
  Tests: `identity/v2/tenants/python_parity_test.go:TestPythonTenantCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v2/native-calls.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.

### `python:identity/v2/tenants`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.WithListOptions`

- List sends GET tenants, follows tenants_links rel=next and sends ListOpts limit and marker as query, but stops when a limited page has no next link.
  Tests: `identity/v2/tenants/python_parity_test.go:TestPythonTenantListRequests`.

Python/Go differences:

- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.

Unresolved boundaries:

- Keystone removed the v2.0 API in Queens and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:identity/v2/update_tenant`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.WithUpdateField`

- Update sends PUT tenants/{id} with the {"tenant": {...}} envelope including the id that Proxy._update leaves dirty when given WithUpdateField("id", id), the request Python update_tenant sends, and decodes the fields Python exposes.
  Tests: `identity/v2/tenants/python_parity_test.go:TestPythonTenantCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v2/native-calls.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Go UpdateOpts omits an empty name and nil pointers and WithUpdateField cannot override typed fields, so Python name="" or description=None values cannot be sent.
- The id body key is added explicitly with tenants.WithUpdateField; Keystone ignores it.

### `python:identity/v2/create_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.API.Create`

- Create sends POST users with the {"user": {...}} envelope, the request Python create_user sends, and decodes the fields Python exposes.
  Tests: `identity/v2/users/python_parity_test.go:TestPythonUserCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v2/native-calls.md.
- Python takes keyword Resource attributes and returns a mutable Resource; Go takes typed CreateOpts and returns the native Gophercloud model pointer.
- users.CreateOpts needs Name or Username, so Go rejects before HTTP a Python create_user without a name; Go can also send username and tenantId, which Python drops as unknown attributes.

### `python:identity/v2/delete_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove(resource.ID(id)) sends DELETE users/{id}, succeeds on 404 like the Python ignore_missing=True default, and WithMissingError returns resource.ErrNotFound like ignore_missing=False.
  Tests: `identity/v2/users/python_parity_test.go:TestPythonUserDeleteIgnoreMissing`.

Python/Go differences:

- Python returns None and accepts any status below 400; Go returns only an error and the native Delete accepts 202 and 204.
- Calling the generated Delete directly returns the 404 error, so only Remove matches the Python ignore_missing=True default.

### `python:identity/v2/find_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Find(resource.ID(x), WithIgnoreMissing) followed on nil by Find(resource.Name(x)) sends GET users/{x} and then the unfiltered users list, returns the ID or unique name match, nil when missing with WithIgnoreMissing, resource.ErrNotFound when strict and resource.ErrAmbiguous for duplicate names.
  Tests: `identity/v2/users/python_parity_test.go:TestPythonUserFindIDThenNameFallback`.

Python/Go differences:

- Go needs two Find calls; Python Resource.find does the ID GET and the list fallback in one call.
- Python falls back to the list after GET 400, 403 or 404; Go Find with WithIgnoreMissing returns nil only for 404, so the caller must continue to the name lookup after a 400 or 403 error.
- The Python list match compares both ID and name; the Go name lookup compares only the name. Duplicates raise DuplicateResource in Python and resource.ErrAmbiguous in Go.
- The Go name lookup reads a single GET users page and does not follow users_links, which Python's list fallback would follow.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.

### `python:identity/v2/get_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.API.Get`

- Get sends GET users/{id}, the request Python get_user sends, and decodes the fields Python exposes.
  Tests: `identity/v2/users/python_parity_test.go:TestPythonUserCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v2/native-calls.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- The native User model reads only tenant_id from responses and matches other fields case-insensitively without JSON tags.

### `python:identity/v2/users`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.API.List`

- List sends the same single GET users as Python users() without arguments, does not follow users_links, and the users collection rejects query options with resource.ErrUnsupported before HTTP.
  Tests: `identity/v2/users/python_parity_test.go:TestPythonUserListSinglePage`.

Python/Go differences:

- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.

Unresolved boundaries:

- Keystone removed the v2.0 API in Queens and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:identity/v2/update_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.WithUpdateField`

- Update sends PUT users/{id} with the {"user": {...}} envelope including the id that Proxy._update leaves dirty when given WithUpdateField("id", id), the request Python update_user sends, and decodes the fields Python exposes.
  Tests: `identity/v2/users/python_parity_test.go:TestPythonUserCreateGetUpdateRequests`.

Python/Go differences:

- Python accepts any status below 400 through raise_from_response; the native call accepts only its Gophercloud OkCodes listed in identity/v2/native-calls.md.
- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Go UpdateOpts omits empty strings and nil pointers and WithUpdateField cannot override typed fields, so Python empty or None attribute values cannot be sent.
- The id body key is added explicitly with users.WithUpdateField; Keystone ignores it.

### `python:identity/v2/wait_for_status`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Wait`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithStatusAttribute`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithUnlimitedWait`

- Tenant and User WaitFor without a status attribute fail with resource.ErrUnsupported before HTTP, like the Python AttributeError for a missing status attribute.
  Tests: `identity/v2/tenants/python_parity_test.go:TestPythonTenantWaitHelpers`, `identity/v2/users/python_parity_test.go:TestPythonUserWaitHelpers`.
- Tenant WaitFor with WithStatusAttribute("name") compares the selected string field case-insensitively after a GET of the fixed ID.
  Tests: `identity/v2/tenants/python_parity_test.go:TestPythonTenantWaitHelpers`.

Python/Go differences:

- Go defaults to a five-minute timeout; Python wait=None waits forever, which Go expresses with resource.WithUnlimitedWait.

Unresolved boundaries:

- Keystone removed the v2.0 API in Queens and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

### `python:identity/v2/wait_for_delete`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v2/tenants.API.WaitForDeletion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v2/users.API.WaitForDeletion`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.WaitDeleted`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithTimeout`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPollInterval`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithProgressCallback`

- Tenant and User WaitForDeletion with WithTimeout(120s) repeat GET on the fixed ID until 404, report progress 0 after each nonterminal observation and then succeed.
  Tests: `identity/v2/tenants/python_parity_test.go:TestPythonTenantWaitHelpers`, `identity/v2/users/python_parity_test.go:TestPythonUserWaitHelpers`.

Python/Go differences:

- Go defaults to a five-minute timeout while the identity v2 Proxy default is wait=120; resource.WithTimeout reproduces it.

Unresolved boundaries:

- Keystone removed the v2.0 API in Queens and no supported deployment serves it, so the operation is excluded from the SDK target instead of being implemented.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
enabled := true
tenant, err := tenants.New(client).Create(ctx, tenants.CreateOpts{Name: "demo", Description: "d", Enabled: &enabled})
tenant, err = tenants.New(client).Update(ctx, "t-1", tenants.UpdateOpts{Name: "renamed"}, tenants.WithUpdateField("id", "t-1"))
```

```go
api := tenants.New(client)
tenant, err := api.Find(ctx, resource.ID(nameOrID), resource.WithIgnoreMissing())
if err == nil && tenant == nil {
	tenant, err = api.Find(ctx, resource.Name(nameOrID), resource.WithIgnoreMissing())
}
```

## References

- [identity v2 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py)
- [Native Calls](native-calls.md)
- [`extensions`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L29)
- [`get_extension`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L36)
- [`create_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L53)
- [`delete_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L64)
- [`find_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L95)
- [`get_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L114)
- [`roles`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L126)
- [`update_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L136)
- [`create_tenant`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L150)
- [`delete_tenant`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L161)
- [`find_tenant`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L192)
- [`get_tenant`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L211)
- [`tenants`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L223)
- [`update_tenant`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L233)
- [`create_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L249)
- [`delete_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L260)
- [`find_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L291)
- [`get_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L310)
- [`users`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L322)
- [`update_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L332)
- [`wait_for_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L346)
- [`wait_for_delete`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L385)
- [Extension.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/extension.py)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2489)
