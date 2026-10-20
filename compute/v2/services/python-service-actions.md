# Python Service Actions

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/enable_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.API.EnableRecord`

- From client microversion 2.53 EnableRecord sends PUT os-services/{id} with the unwrapped body {"status": "enabled"} for a given ID, ignoring Host and Binary, at the client microversion, accepts every status below 400 and decodes the typed Service and raw Fields.
  Tests: `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsFromMicroversion253`.
- From 2.53 without an ID EnableRecord lists GET os-services?binary=&host=, requires exactly one row (resource.ErrNotFound for none, resource.ErrAmbiguous for several, no PUT), sends only attributes whose listed value differs, and returns the listed row with Sent false and the list header/status when nothing changed.
  Tests: `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsFromMicroversion253`, `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsLookupFailures`.
- Below 2.53, including an unset client microversion, EnableRecord sends PUT os-services/enable with {host, binary} at the client microversion or without version headers, keeps Nova's id-less service response in Fields with a nil typed Service, and wraps a 404 in resource.OperationError{EnableRecord, services} with the Gophercloud status error.
  Tests: `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsBelowMicroversion253`.
- A target without an ID or both Host and Binary from 2.53, a target without both Host and Binary below 2.53 (even with an ID), an unparsable client microversion and a nil API fail with resource.ErrInvalidOption before HTTP, as Python _get_service raises ValueError.
  Tests: `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsLookupFailures`, `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsBelowMicroversion253`.

Python/Go differences:

- Python decides the 2.53 branch with supports_microversion against the server range and default microversion and sends min(Service _max_microversion 2.69, server maximum); Go decides and sends with the one microversion configured or negotiated on the Connection client.
- Python accepts a Service or an ID whose id, host and binary it reuses; Go takes ServiceActionTarget values and treats empty strings as absent, so an empty host or binary that Python would send is rejected before HTTP.
- Python returns the Resource merged from seeded attributes and the response, including the seeded id below 2.53; Go returns ServiceRecord with the response object in Fields, a typed Service from 2.53, and no typed Service for the id-less legacy response.
- Python's host and binary lookup would follow a next link from services_links, links, next or a Link header; Go reads only the single os-services response that Nova returns.

### `python:compute/v2/disable_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.API.DisableRecord`

- From client microversion 2.53 DisableRecord sends PUT os-services/{id} with {"status": "disabled"} plus disabled_reason only when the reason is non-empty, at the client microversion, and accepts every status below 400.
  Tests: `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsFromMicroversion253`.
- From 2.53 without an ID DisableRecord lists services by host and binary, requires exactly one row (resource.ErrAmbiguous wrapped in resource.OperationError{DisableRecord, services} for several), sends only dirty attributes ({"disabled_reason": ...} when only the reason changes), and sends no PUT when status and reason already match.
  Tests: `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsFromMicroversion253`, `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsLookupFailures`.
- Below 2.53 DisableRecord sends PUT os-services/disable with {host, binary}, or os-services/disable-log-reason with disabled_reason added, at the client microversion or without version headers, and keeps Nova's id-less response in Fields with a nil typed Service.
  Tests: `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsBelowMicroversion253`.
- An unparsable client microversion and a target without the required ID or host and binary fail with resource.ErrInvalidOption before HTTP.
  Tests: `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsLookupFailures`.

Python/Go differences:

- Python decides the 2.53 branch with supports_microversion against the server range and default microversion and sends min(Service _max_microversion 2.69, server maximum); Go decides and sends with the one microversion configured or negotiated on the Connection client.
- Python accepts a Service or an ID whose id, host and binary it reuses; Go takes ServiceActionTarget values and treats empty strings, including an empty reason, as absent; Python also treats an empty reason as no reason.
- Python returns the Resource merged from seeded attributes and the response, including the seeded id below 2.53; Go returns ServiceRecord with the response object in Fields, a typed Service from 2.53, and no typed Service for the id-less legacy response.
- Python's host and binary lookup would follow a next link from services_links, links, next or a Link header; Go reads only the single os-services response that Nova returns.

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)
- [Service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/service.py)
- [UpdateForcedDownRecord](python-update-record.md)
