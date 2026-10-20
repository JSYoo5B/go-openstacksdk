# Python List Query

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/tap_mirrors`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.WithListQueryValues`
- `github.com/JSYoo5B/go-openstacksdk/request.WithQueryValues`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithName`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPageSize`

- List with WithListQueryValues sends GET taas/tap_mirrors with fields, sort_key and sort_dir repeated once per value in order alongside name and limit, keeps the repeated keys when following the tap_mirrors_links next href with marker, lets a later option replace earlier values of the key and drops a key given no values.
  Tests: `network/v2/extensions/taas/tapmirrors/python_list_query_test.go:TestPythonTapMirrorListRepeatedQueryValues`.
- WithListQueryValues with a blank key fails before HTTP with resource.ErrInvalidOption wrapped in a List resource.OperationError, and a 400 list response is returned as the Gophercloud status error after sending the repeated sort_key values.
  Tests: `network/v2/extensions/taas/tapmirrors/python_list_query_test.go:TestPythonTapMirrorListRepeatedQueryRejectsInputAndStatus`.
- request.WithQueryValues snapshots the values, replaces an earlier single value, removes the key when given no values, and replaces the typed serialization of that key through ExtendQuery.
  Tests: `request/query_values_test.go:TestQueryValuesRepeatKeysAndExtendTypedQuery`.
- All sends GET taas/tap_mirrors with name, project_id and limit query values, follows the tap_mirrors_links next href with marker and decodes each mirror; a full page without links ends the listing after one request.
  Tests: `network/v2/extensions/taas/tapmirrors/python_parity_test.go:TestPythonTapMirrorListQueryAndPaging`.

Python/Go differences:

- Python sends only declared query keys (name, project_id, sort_key, sort_dir, fields, limit, marker), filters other Body attributes such as port_id locally and drops unknown keys; Go forwards every WithListQuery, WithListQueryValues or resource.WithQuery key to the server.
- Repeated query keys are available only through API.List; API.All and resource.WithQuery still send one value per key, so combining repeated fields with max_items means stopping the List iterator after that many rows.
- Python continues with marker=<last id> after a limit page without links; Go follows only tap_mirrors_links.
- Python returns TapMirror Resources whose project_id falls back to tenant_id; Go yields *tapmirrors.TapMirror with separate ProjectID and TenantID, and List HTTP errors are Gophercloud errors.
- Python passes the directions dict through unchanged; Go Directions decodes IN/OUT tunnel IDs from JSON strings of integers and fails on unquoted numbers.
- Neutron has no microversion, so neither SDK sends a version header.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
for mirror, err := range api.List(ctx,
	tapmirrors.WithListQuery("name", "mirror"),
	tapmirrors.WithListQueryValues("fields", "id", "name"),
	tapmirrors.WithListQueryValues("sort_key", "name", "id"),
	tapmirrors.WithListQueryValues("sort_dir", "asc", "desc"),
) {
	...
}
```

## References

- [`tap_mirrors(**query)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [`TapMirror`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/tap_mirror.py)
