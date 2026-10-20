# Tapflows

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_tap_flow`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapflows.API.Create`

- Create sends POST taas/tap_flows with {"tap_flow": attrs} carrying only the given attributes, keeps unknown and null attributes, drops the microversion constructor key, and decodes the Python Resource view from a 201 response.
  Tests: `network/v2/extensions/taas/tapflows/python_records_test.go:TestPythonTapFlowCreatePostsGivenAttrs`.
- Create, Get and Update accept every 200-399 status and return 4xx and 5xx statuses as gophercloud.ErrUnexpectedResponseCode wrapped in resource.OperationError for tapflows.
  Tests: `network/v2/extensions/taas/tapflows/python_records_test.go:TestPythonTapFlowStatusesCarryOperationContext`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts ID strings only.
- Go escapes each path segment once and rejects blank, ".", ".." and control-character IDs before HTTP; Python inserts IDs into the path unescaped.
- Python negotiates the network microversion; Go sends OpenStack-API-Version only when the selected client has a microversion, and per-call headers and microversion options are Go extensions.
- Python matches attribute names case-insensitively in _consume_attrs; Go maps only exact attribute or wire names and keeps every other key as an unknown body attribute.
- Python returns a Resource object; Go returns a typed struct whose Resource.Body holds the Python to_dict view as raw JSON, with Wire, Envelope, Header and StatusCode holding the actual response.

### `python:network/v2/get_tap_flow`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapflows.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapflows.WithGetFields`

- Get sends GET taas/tap_flows/{id} with the escaped ID and maps WithGetFields attribute names to repeated fields query values, sending no query for empty fields.
  Tests: `network/v2/extensions/taas/tapflows/python_records_test.go:TestPythonTapFlowGetMapsFields`.
- Get returns non-success statuses, including 404, as wrapped errors without an ignore-missing default.
  Tests: `network/v2/extensions/taas/tapflows/python_records_test.go:TestPythonTapFlowStatusesCarryOperationContext`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts ID strings only.
- Go escapes each path segment once and rejects blank, ".", ".." and control-character IDs before HTTP; Python inserts IDs into the path unescaped.
- Python negotiates the network microversion; Go sends OpenStack-API-Version only when the selected client has a microversion, and per-call headers and microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose Resource.Body holds the Python to_dict view as raw JSON, with Wire, Envelope, Header and StatusCode holding the actual response.

### `python:network/v2/find_tap_flow`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapflows.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapflows.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapflows.WithFindQuery`

- Find sends GET taas/tap_flows/{name_or_id} with the find query first and, only after a clean 400, 403 or 404, lists taas/tap_flows?name={name_or_id} and matches exact ID or name; a miss returns nil by default and resource.NotFoundError with WithFindIgnoreMissing(false), duplicates and other statuses are wrapped errors.
  Tests: `network/v2/extensions/taas/tapflows/python_records_test.go:TestPythonTapFlowFindFallsBackToNameList`, `network/v2/extensions/taas/tapflows/python_records_test.go:TestPythonTapFlowFindMissing`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts ID strings only.
- Go escapes each path segment once and rejects blank, ".", ".." and control-character IDs before HTTP; Python inserts IDs into the path unescaped.
- Python negotiates the network microversion; Go sends OpenStack-API-Version only when the selected client has a microversion, and per-call headers and microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose Resource.Body holds the Python to_dict view as raw JSON, with Wire, Envelope, Header and StatusCode holding the actual response.
- Go does not emit the find_* ignore_missing deprecation warning.

### `python:network/v2/tap_flows`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapflows.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapflows.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapflows.WithListQuery`

- List sends GET taas/tap_flows with Accept: application/json, encodes declared server query names (sort_key, sort_dir, fields, name, project_id, limit, marker), filters Body names (id, description, tenant_id, tap_service_id, direction, status, source_port) locally, ignores unknown names and follows tap_flows_links next links.
  Tests: `network/v2/extensions/taas/tapflows/python_records_test.go:TestPythonTapFlowListQueryFiltersAndLinks`.
- With a limit and no links List continues with the last row ID as marker until an empty page, paginated=false reads one page, and a failing page is a wrapped error.
  Tests: `network/v2/extensions/taas/tapflows/python_records_test.go:TestPythonTapFlowListMarkerContinuation`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts ID strings only.
- Go escapes each path segment once and rejects blank, ".", ".." and control-character IDs before HTTP; Python inserts IDs into the path unescaped.
- Python negotiates the network microversion; Go sends OpenStack-API-Version only when the selected client has a microversion, and per-call headers and microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose Resource.Body holds the Python to_dict view as raw JSON, with Wire, Envelope, Header and StatusCode holding the actual response.
- Go returns a lazy iter.Seq2 that sends the requests again on every iteration; All collects it and discards partial rows on failure.

### `python:network/v2/update_tap_flow`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapflows.API.Update`

- Update sends PUT taas/tap_flows/{id} with {"tap_flow": attrs} for the given attributes only, sends no request when no attribute is given, and rejects bound argument names before HTTP.
  Tests: `network/v2/extensions/taas/tapflows/python_records_test.go:TestPythonTapFlowUpdatePutsDirtyAttrs`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts ID strings only.
- Go escapes each path segment once and rejects blank, ".", ".." and control-character IDs before HTTP; Python inserts IDs into the path unescaped.
- Python negotiates the network microversion; Go sends OpenStack-API-Version only when the selected client has a microversion, and per-call headers and microversion options are Go extensions.
- Python matches attribute names case-insensitively in _consume_attrs; Go maps only exact attribute or wire names and keeps every other key as an unknown body attribute.
- Python returns a Resource object; Go returns a typed struct whose Resource.Body holds the Python to_dict view as raw JSON, with Wire, Envelope, Header and StatusCode holding the actual response.
- Bound proxy argument names passed as attributes raise TypeError in Python; Go rejects them as resource.ErrInvalidOption before HTTP.

### `python:network/v2/delete_tap_flow`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapflows.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapflows.WithDeleteIgnoreMissing`

- Delete sends a bodyless DELETE taas/tap_flows/{id}, returns nil for 204 and a clean 404 by default, and returns 404 with WithDeleteIgnoreMissing(false) and 409 as wrapped errors.
  Tests: `network/v2/extensions/taas/tapflows/python_records_test.go:TestPythonTapFlowDeleteIgnoresMissingByDefault`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts ID strings only.
- Go escapes each path segment once and rejects blank, ".", ".." and control-character IDs before HTTP; Python inserts IDs into the path unescaped.
- Python negotiates the network microversion; Go sends OpenStack-API-Version only when the selected client has a microversion, and per-call headers and microversion options are Go extensions.
- Python returns None; Go returns only an error, nil for success or an ignored clean 404.

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L8985-L9062)
- [`tap_flow.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/tap_flow.py)
- [`tapmirrors`](../tapmirrors)
- [`tapservices`](../tapservices)
