# Python Update Record

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/update_aggregate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithUpdateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithUpdateRecordName`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithUpdateRecordAvailabilityZone`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithUpdateRecordField`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithUpdateRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/request.Null`
- `github.com/JSYoo5B/go-openstacksdk/request.Present`

- UpdateRecord sends PUT os-aggregates/{id} with only the supplied attributes in the aggregate envelope, including availability_zone null, an empty name and extension attributes, accepts any status below 400, decodes enveloped and flat responses, and sends nothing when no attribute is supplied.
  Tests: `compute/v2/aggregates/python_update_record_test.go:TestPythonAggregateUpdateRecordSendsNullAndEmptyValues`.
- Extension fields that collide with typed keys or the id identity and nil options fail before HTTP, and a 409 response is returned as the Gophercloud status error wrapped in resource.OperationError.
  Tests: `compute/v2/aggregates/python_update_record_test.go:TestPythonAggregateUpdateRecordRejectsInputsAndErrorStatuses`.

Python/Go differences:

- Python accepts an Aggregate or an ID string; Go takes the integer aggregate ID.
- Python sends min(Aggregate _max_microversion 2.81, server maximum) discovered per request; Go sends the one microversion configured on the client.
- Python drops undeclared keyword attributes; WithUpdateRecordField sends any extension key except the typed keys and id.
- Python returns the Resource merged from request attributes and the response; Go returns AggregateRecord with the native typed decode, the raw response object fields, headers and status, and Sent false when no request was needed.

## References

- [`update_aggregate`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)
