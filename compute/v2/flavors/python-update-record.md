# Python Update Record

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/update_flavor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithUpdateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithUpdateRecordDescription`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithUpdateRecordField`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithUpdateRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/request.Null`
- `github.com/JSYoo5B/go-openstacksdk/request.Present`

- UpdateRecord sends PUT flavors/{id} with only the supplied attributes in the flavor envelope, including description null, an empty description and extension attributes, accepts any status below 400, decodes enveloped, flat and empty responses, and sends nothing when no attribute is supplied.
  Tests: `compute/v2/flavors/python_update_record_test.go:TestPythonFlavorUpdateRecordSendsNullAndEmptyDescription`.
- Extension fields that collide with description or the id identity, nil options and a blank flavor id fail before HTTP, and a 409 response is returned as the Gophercloud status error wrapped in resource.OperationError.
  Tests: `compute/v2/flavors/python_update_record_test.go:TestPythonFlavorUpdateRecordRejectsInputsAndErrorStatuses`.

Python/Go differences:

- Python accepts a Flavor or an ID string; Go takes the flavor ID string and rejects a blank one before HTTP.
- Python sends min(Flavor _max_microversion 2.61, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python maps attribute names such as is_public to server keys and drops undeclared keyword attributes; WithUpdateRecordField sends any extension key verbatim as a server key except description and id.
- Python returns the Resource merged from request attributes and the response; Go returns FlavorUpdateRecord with the native typed decode, the raw response object fields, headers and status, and Sent false when no request was needed.

## References

- [`update_flavor`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)
- [`flavor.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/flavor.py)
