# Python Update Record

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:placement/v1/update_resource_class`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.WithUpdateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.WithUpdateRecordName`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.WithUpdateRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/request.Present`
- `github.com/JSYoo5B/go-openstacksdk/request.Null`

- UpdateRecord sends PUT resource_classes/{name} at placement 1.2 even when the client selects 1.39, with {"name": new}, {} without a new name or {"name": null}, accepts 200, 201 and 204, decodes the 200 body into the typed ResourceClass and raw fields while an empty body leaves both nil, and leaves the client microversion unchanged.
  Tests: `placement/v1/resourceclasses/python_update_record_test.go:TestPythonResourceClassUpdateRecordRenames`.
- A 409 response is returned as the Gophercloud status error wrapped in resource.OperationError, a non-object JSON body is returned with resource.ResponseError, and invalid names, a conflicting version header, nil options and a nil API fail before HTTP.
  Tests: `placement/v1/resourceclasses/python_update_record_test.go:TestPythonResourceClassUpdateRecordRejections`.

Python/Go differences:

- Python sends min(1.2, server maximum) or the session default microversion; Go always sends 1.2 for this call because 1.7 and later turn the PUT into a bodyless ensure-exists call.
- Python accepts a ResourceClass or a name string; Go takes the current name string.
- Python returns the Resource merged from request attributes and the response, so its id keeps the old name; Go returns ResourceClassRecord with the native typed decode, the raw response object fields, headers and status.

## References

- [`update_resource_class(resource_class, **attrs)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/_proxy.py)
- [`ResourceClass`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/resource_class.py)
