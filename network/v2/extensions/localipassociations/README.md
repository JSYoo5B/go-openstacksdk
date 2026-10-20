# Localipassociations

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_local_ip_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localipassociations.API.Create`

- Create sends POST local_ips/{local_ip_id}/port_associations with {"port_association": attrs}, escapes the parent once, rejects local_ip and local_ip_id attributes and invalid parents before HTTP, reads a response local_ip_id back as the bound parent and wraps a 409.
  Tests: `network/v2/extensions/localipassociations/python_records_test.go:TestPythonLocalIPAssociationCreatePostsUnderParent`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts ID strings only.
- Go escapes each path segment once and rejects blank, ".", ".." and control-character IDs before HTTP; Python inserts IDs into the path unescaped.
- Python negotiates the network microversion; Go sends OpenStack-API-Version only when the selected client has a microversion, and per-call headers and microversion options are Go extensions.
- Python matches attribute names case-insensitively in _consume_attrs; Go maps only exact attribute or wire names and keeps every other key as an unknown body attribute.
- Python returns a Resource object; Go returns a typed struct whose Resource.Body holds the Python to_dict view as raw JSON, with Wire, Envelope, Header and StatusCode holding the actual response.
- Go takes the local IP ID as the first argument of every method, so the argument order differs from get_local_ip_association(local_ip_association, local_ip).

### `python:network/v2/delete_local_ip_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localipassociations.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localipassociations.WithDeleteIgnoreMissing`

- Delete sends a bodyless DELETE local_ips/{local_ip_id}/port_associations/{fixed_port_id}, ignores a clean 404 by default, and returns 404 with WithDeleteIgnoreMissing(false) and 409 as wrapped errors.
  Tests: `network/v2/extensions/localipassociations/python_records_test.go:TestPythonLocalIPAssociationDeleteIgnoresMissingByDefault`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts ID strings only.
- Go escapes each path segment once and rejects blank, ".", ".." and control-character IDs before HTTP; Python inserts IDs into the path unescaped.
- Python negotiates the network microversion; Go sends OpenStack-API-Version only when the selected client has a microversion, and per-call headers and microversion options are Go extensions.
- Python returns None; Go returns only an error, nil for success or an ignored clean 404.
- Python if_revision has no effect because LocalIPAssociation is not a NetworkResource; Go has no if_revision option.

### `python:network/v2/find_local_ip_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localipassociations.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localipassociations.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localipassociations.WithFindQuery`

- Find sends GET local_ips/{local_ip_id}/port_associations/{name_or_id}?local_ip_id={local_ip_id} with the find query first and, only after a clean 400, 403 or 404, lists the associations without a name hint and matches exact id or name Body values; a miss returns nil by default and resource.NotFoundError with WithFindIgnoreMissing(false), and local_ip query names are rejected.
  Tests: `network/v2/extensions/localipassociations/python_records_test.go:TestPythonLocalIPAssociationFindForwardsParent`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts ID strings only.
- Go escapes each path segment once and rejects blank, ".", ".." and control-character IDs before HTTP; Python inserts IDs into the path unescaped.
- Python negotiates the network microversion; Go sends OpenStack-API-Version only when the selected client has a microversion, and per-call headers and microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose Resource.Body holds the Python to_dict view as raw JSON, with Wire, Envelope, Header and StatusCode holding the actual response.
- Go takes the local IP ID as the first argument of every method, so the argument order differs from get_local_ip_association(local_ip_association, local_ip).
- Go does not emit the find_* ignore_missing deprecation warning.

### `python:network/v2/get_local_ip_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localipassociations.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localipassociations.WithGetFields`

- Get sends GET local_ips/{local_ip_id}/port_associations/{id} with fields and no local_ip_id query, keeps the requested ID when the response omits id, and wraps a 404.
  Tests: `network/v2/extensions/localipassociations/python_records_test.go:TestPythonLocalIPAssociationGetUsesMemberRoute`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts ID strings only.
- Go escapes each path segment once and rejects blank, ".", ".." and control-character IDs before HTTP; Python inserts IDs into the path unescaped.
- Python negotiates the network microversion; Go sends OpenStack-API-Version only when the selected client has a microversion, and per-call headers and microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose Resource.Body holds the Python to_dict view as raw JSON, with Wire, Envelope, Header and StatusCode holding the actual response.
- Go takes the local IP ID as the first argument of every method, so the argument order differs from get_local_ip_association(local_ip_association, local_ip).

### `python:network/v2/local_ip_associations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localipassociations.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localipassociations.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/localipassociations.WithListQuery`

- List sends GET local_ips/{local_ip_id}/port_associations with Accept: application/json, encodes fixed_port_id, fixed_ip, host, fields, sort and paging query names, filters id, name and local_ip_address locally, follows port_associations_links, sets LocalIPID on rows without adding local_ip_id to the Body view, and rejects local_ip and local_ip_id query names.
  Tests: `network/v2/extensions/localipassociations/python_records_test.go:TestPythonLocalIPAssociationListQueryFiltersAndLinks`.
- With a limit and no links a row without id cannot supply a marker, so List fails with resource.ErrInvalidOption after the first page; paginated=false reads one page.
  Tests: `network/v2/extensions/localipassociations/python_records_test.go:TestPythonLocalIPAssociationListMarkerNeedsID`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts ID strings only.
- Go escapes each path segment once and rejects blank, ".", ".." and control-character IDs before HTTP; Python inserts IDs into the path unescaped.
- Python negotiates the network microversion; Go sends OpenStack-API-Version only when the selected client has a microversion, and per-call headers and microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose Resource.Body holds the Python to_dict view as raw JSON, with Wire, Envelope, Header and StatusCode holding the actual response.
- Go takes the local IP ID as the first argument of every method, so the argument order differs from get_local_ip_association(local_ip_association, local_ip).
- When a limit is given without links and rows lack id, Python raises the endless-pagination SDKException (after one extra marker-less request if an initial marker was given) while Go stops with resource.ErrInvalidOption at the first page that needs a marker.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
assoc = conn.network.create_local_ip_association(lip, fixed_port_id=port.id)
for item in conn.network.local_ip_associations(lip, host="compute-1"):
    print(item.fixed_ip)
conn.network.delete_local_ip_association(lip, port.id)
```

```go
api := localipassociations.New(networkClient)
_, err := api.Create(ctx, lipID, localipassociations.WithCreateField("fixed_port_id", portID))
if err != nil { return err }
for item, err := range api.List(ctx, lipID, localipassociations.WithListQuery("host", "compute-1")) {
    if err != nil { return err }
    fmt.Println(*item.FixedIP)
}
err = api.Delete(ctx, lipID, portID)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L1943-L2109)
- [`local_ip_association.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/local_ip_association.py)
- [`localips`](../localips)
