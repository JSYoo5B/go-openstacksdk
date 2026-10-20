# Python Update Record

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/update_service_forced_down`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.API.UpdateForcedDownRecord`

- From microversion 2.53 UpdateForcedDownRecord sends PUT os-services/{id} with forced_down true or false for a given id, resolves a missing id through GET os-services?binary=&host= requiring exactly one row, returns resource.ErrNotFound or resource.ErrAmbiguous for zero or several rows, and rejects a target without an id or both host and binary before HTTP.
  Tests: `compute/v2/services/python_update_record_test.go:TestPythonComputeServiceUpdateForcedDownRecordFromMicroversion253`.
- Below microversion 2.53 UpdateForcedDownRecord sends PUT os-services/force-down with host and binary, adds forced_down and pins microversion 2.11 when the client selects 2.11 or later, keeps the client microversion without forced_down below 2.11, keeps the id-less response in Fields without a typed Service, rejects a missing host or binary before HTTP and wraps a 404 in resource.OperationError.
  Tests: `compute/v2/services/python_update_record_test.go:TestPythonComputeServiceUpdateForcedDownRecordBelowMicroversion253`.

Python/Go differences:

- Python decides the 2.53 and 2.11 branches with supports_microversion against the server range and default microversion and sends min(Service _max_microversion 2.69, server maximum); Go decides and sends with the one microversion configured or negotiated on the Connection client.
- Python accepts a Service whose id, host and binary it reuses; Go takes ForcedDownRecordTarget values and treats empty strings as absent, so an empty host or binary that Python would send as null below 2.53 is rejected before HTTP.
- Python returns None below 2.53; Go returns ServiceRecord with the raw response fields, headers and status and a nil Service.
- Python's host and binary lookup would follow a next link from services_links, links, next or a Link header; Go reads only the single os-services response that Nova returns.

### `python:compute/v2/update_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.WithUpdateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.WithUpdateRecordStatus`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.WithUpdateRecordDisabledReason`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.WithUpdateRecordForcedDown`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.WithUpdateRecordField`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.WithUpdateRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/request.Null`
- `github.com/JSYoo5B/go-openstacksdk/request.Present`

- From microversion 2.53 UpdateRecord sends PUT os-services/{id} with only the supplied attributes in an unwrapped body, including forced_down false, empty status and disabled_reason, null disabled_reason and extension attributes, accepts any status below 400, decodes enveloped and flat responses, and sends nothing when no attribute is supplied.
  Tests: `compute/v2/services/python_update_record_test.go:TestPythonComputeServiceUpdateRecordSendsFalseAndEmptyValues`.
- A client microversion below 2.53, unset or latest fails with resource.ErrUnsupported before HTTP, extension fields that collide with typed keys or id, nil options and a blank id fail with resource.ErrInvalidOption before HTTP, and a 409 response is returned as the Gophercloud status error wrapped in resource.OperationError.
  Tests: `compute/v2/services/python_update_record_test.go:TestPythonComputeServiceUpdateRecordRejectsInputsVersionsAndErrorStatuses`.

Python/Go differences:

- Python sends min(Service _max_microversion 2.69, server maximum) discovered per request and checks 2.53 against the server range and default microversion; Go sends and checks the one microversion configured or negotiated on the Connection client.
- Python accepts a Service or an ID string; Go takes the service ID string and rejects a blank one before HTTP.
- Python maps attribute names such as availability_zone and is_forced_down to server keys and drops undeclared keyword attributes; WithUpdateRecordField sends any extension key verbatim as a server key except the typed keys and id.
- Python returns the Resource merged from request attributes and the response; Go returns ServiceRecord with the native typed decode, the raw response object fields, headers and status, and Sent false when no request was needed.

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)
- [`service.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/service.py)
