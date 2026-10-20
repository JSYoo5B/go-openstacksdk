# Localips

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_local_ip`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localips.API.Create`

- Create sends POST local_ips with {"local_ip": attrs} carrying only the given attributes, keeps unknown and null attributes, drops the microversion constructor key, and decodes the Python Resource view from a 201 response.
  Tests: `network/v2/extensions/localips/python_records_test.go:TestPythonLocalIPCreatePostsGivenAttrs`.
- Create returns 4xx and 5xx statuses as gophercloud.ErrUnexpectedResponseCode wrapped in resource.OperationError for localips.
  Tests: `network/v2/extensions/localips/python_records_test.go:TestPythonLocalIPCreatePostsGivenAttrs`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts ID strings only.
- Go escapes each path segment once and rejects blank, ".", ".." and control-character IDs before HTTP; Python inserts IDs into the path unescaped.
- Python negotiates the network microversion; Go sends OpenStack-API-Version only when the selected client has a microversion, and per-call headers and microversion options are Go extensions.
- Python matches attribute names case-insensitively in _consume_attrs; Go maps only exact attribute or wire names and keeps every other key as an unknown body attribute.
- Python returns a Resource object; Go returns a typed struct whose Resource.Body holds the Python to_dict view as raw JSON, with Wire, Envelope, Header and StatusCode holding the actual response.

### `python:network/v2/delete_local_ip`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localips.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localips.WithDeleteIgnoreMissing`

- Delete sends a bodyless DELETE local_ips/{id}, returns nil for 204 and a clean 404 by default, and returns 404 with WithDeleteIgnoreMissing(false) and 409 as wrapped errors.
  Tests: `network/v2/extensions/localips/python_records_test.go:TestPythonLocalIPDeleteIgnoresMissingByDefault`.
- Delete sends no If-Match header unless the caller supplies one with WithDeleteHeaders.
  Tests: `network/v2/extensions/localips/python_records_test.go:TestPythonLocalIPUpdateIgnoresRevisionPrecondition`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts ID strings only.
- Go escapes each path segment once and rejects blank, ".", ".." and control-character IDs before HTTP; Python inserts IDs into the path unescaped.
- Python negotiates the network microversion; Go sends OpenStack-API-Version only when the selected client has a microversion, and per-call headers and microversion options are Go extensions.
- Python returns None; Go returns only an error, nil for success or an ignored clean 404.
- Python if_revision has no effect because LocalIP is not a NetworkResource, so no If-Match header is sent; Go has no if_revision option and sends If-Match only when the caller supplies it as a header.

### `python:network/v2/find_local_ip`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localips.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localips.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localips.WithFindQuery`

- Find sends GET local_ips/{name_or_id} with the find query first and, only after a clean 400, 403 or 404, lists local_ips?name={name_or_id} and matches exact ID or name; a miss returns nil by default and resource.NotFoundError with WithFindIgnoreMissing(false), duplicates and other statuses are wrapped errors.
  Tests: `network/v2/extensions/localips/python_records_test.go:TestPythonLocalIPFindFallsBackToNameList`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts ID strings only.
- Go escapes each path segment once and rejects blank, ".", ".." and control-character IDs before HTTP; Python inserts IDs into the path unescaped.
- Python negotiates the network microversion; Go sends OpenStack-API-Version only when the selected client has a microversion, and per-call headers and microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose Resource.Body holds the Python to_dict view as raw JSON, with Wire, Envelope, Header and StatusCode holding the actual response.
- Go does not emit the find_* ignore_missing deprecation warning.

### `python:network/v2/get_local_ip`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localips.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localips.WithGetFields`

- Get sends GET local_ips/{id} with the escaped ID and maps WithGetFields attribute names to repeated fields query values, sending no query for empty fields.
  Tests: `network/v2/extensions/localips/python_records_test.go:TestPythonLocalIPGetMapsFields`.
- Get returns non-success statuses, including 404, as wrapped errors without an ignore-missing default.
  Tests: `network/v2/extensions/localips/python_records_test.go:TestPythonLocalIPGetMapsFields`.
- The LocalIP view keeps revision_number untyped and has no tenant_id alias for project_id.
  Tests: `network/v2/extensions/localips/python_records_test.go:TestPythonLocalIPProjectionKeepsUntypedRevisionAndNoTenantAlias`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts ID strings only.
- Go escapes each path segment once and rejects blank, ".", ".." and control-character IDs before HTTP; Python inserts IDs into the path unescaped.
- Python negotiates the network microversion; Go sends OpenStack-API-Version only when the selected client has a microversion, and per-call headers and microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose Resource.Body holds the Python to_dict view as raw JSON, with Wire, Envelope, Header and StatusCode holding the actual response.

### `python:network/v2/local_ips`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localips.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localips.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localips.WithListQuery`

- List sends GET local_ips with Accept: application/json, encodes declared server query names (sort_key, sort_dir, name, description, fields, project_id, network_id, local_port_id, local_ip_address, ip_mode, limit, marker), filters Body names (id, created_at, revision_number, updated_at) locally, ignores unknown names and follows local_ips_links next links. With a limit and no links it continues with the last row ID as marker until an empty page, and a failing page is a wrapped error.
  Tests: `network/v2/extensions/localips/python_records_test.go:TestPythonLocalIPListQueryFiltersAndLinks`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts ID strings only.
- Go escapes each path segment once and rejects blank, ".", ".." and control-character IDs before HTTP; Python inserts IDs into the path unescaped.
- Python negotiates the network microversion; Go sends OpenStack-API-Version only when the selected client has a microversion, and per-call headers and microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose Resource.Body holds the Python to_dict view as raw JSON, with Wire, Envelope, Header and StatusCode holding the actual response.
- Go returns a lazy iter.Seq2 that sends the requests again on every iteration; All collects it and discards partial rows on failure.

### `python:network/v2/update_local_ip`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localips.API.Update`

- Update sends PUT local_ips/{id} with {"local_ip": attrs} and no If-Match header, sends a caller If-Match header only when given, sends no request without attributes, and rejects id, local_ip and if_revision attributes before HTTP.
  Tests: `network/v2/extensions/localips/python_records_test.go:TestPythonLocalIPUpdateIgnoresRevisionPrecondition`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts ID strings only.
- Go escapes each path segment once and rejects blank, ".", ".." and control-character IDs before HTTP; Python inserts IDs into the path unescaped.
- Python negotiates the network microversion; Go sends OpenStack-API-Version only when the selected client has a microversion, and per-call headers and microversion options are Go extensions.
- Python matches attribute names case-insensitively in _consume_attrs; Go maps only exact attribute or wire names and keeps every other key as an unknown body attribute.
- Python returns a Resource object; Go returns a typed struct whose Resource.Body holds the Python to_dict view as raw JSON, with Wire, Envelope, Header and StatusCode holding the actual response.
- Bound proxy argument names passed as attributes raise TypeError in Python; Go rejects them as resource.ErrInvalidOption before HTTP.
- Python if_revision has no effect because LocalIP is not a NetworkResource, so no If-Match header is sent; Go has no if_revision option and sends If-Match only when the caller supplies it as a header.

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L1796-L1941)
- [`local_ip.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/local_ip.py)
- [`localipassociations`](../localipassociations)
