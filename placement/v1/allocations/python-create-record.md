# Python Create Record

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:placement/v1/create_allocations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations.WithCreateRecordHeader`

- At placement 1.38 CreateRecord posts the caller's per-consumer objects unchanged to POST allocations, so mappings, null consumer_generation and an empty allocations map stay inside each consumer, sends {} for a nil or empty map and null for a nil consumer, and accepts 200 and 204.
  Tests: `placement/v1/allocations/python_create_record_test.go:TestPythonAllocationCreateRecordSendsPerConsumerKeys`.
- A 409 response is returned as the Gophercloud status error wrapped in resource.OperationError, and values encoding/json cannot encode, a nil option or a nil API fail before HTTP.
  Tests: `placement/v1/allocations/python_create_record_test.go:TestPythonAllocationCreateRecordRejections`.
- At placement 1.38 Manage sends POST allocations with each consumer's allocations, project_id, user_id, consumer_generation and consumer_type, and an empty non-nil allocations map clears a consumer.
  Tests: `placement/v1/allocations/python_parity_test.go:TestPythonAllocationProxyCalls`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.38.
- Python returns None; Go returns CreateRecordResult with the response headers and status code.
- Python sends no body when allocations is None; Go sends {} for a nil consumer map.

## References

- [`create_allocations(allocations)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/_proxy.py)
- [`Allocation.set`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/allocation.py)
