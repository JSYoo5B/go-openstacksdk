# Python Parity Federation

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/create_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.API.Create`

- Create sends POST policies with {"policy": {blob, type, ...Extra}} carrying the same keys as Python's dirty Policy attributes and decodes id, type, blob and links with project_id and user_id in Extra.
  Tests: `identity/v3/policies/python_parity_test.go:TestPythonPolicyCreateGetUpdate`.

Python/Go differences:

- Go requires non-empty type and a non-nil blob and rejects a type longer than 255 characters before HTTP; Python sends the request and Keystone rejects it with 400.
- Python exposes project_id and user_id as attributes; Go returns them in Policy.Extra.
- Python accepts any status below 400; the generated Go call accepts only the native codes (create 201, get/update 200, delete 202/204) and reports others as gophercloud.ErrUnexpectedResponseCode wrapped in resource.OperationError.
- Keystone v3 has no microversion; neither side sends a microversion header.

### `python:identity/v3/delete_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- Remove and resource.IgnoreMissing(Delete) send one DELETE policies/{id}, return nil on 204 and 404 like ignore_missing=True and keep 409; WithMissingError reports resource.ErrNotFound and plain Delete keeps the 404 like ignore_missing=False.
  Tests: `identity/v3/policies/python_parity_test.go:TestPythonPolicyDeleteIgnoreMissing`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go takes the ID string or a resource.Ref.
- Python accepts any status below 400; the generated Go call accepts only the native codes (create 201, get/update 200, delete 202/204) and reports others as gophercloud.ErrUnexpectedResponseCode wrapped in resource.OperationError.

### `python:identity/v3/get_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`

- Get and Find(resource.ID) send GET policies/{id}, decode the same policy fields and report a 404 as a Gophercloud 404 or resource.ErrNotFound like Python NotFoundException.
  Tests: `identity/v3/policies/python_parity_test.go:TestPythonPolicyCreateGetUpdate`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go takes the ID string or a resource.Ref.
- Python exposes project_id and user_id as attributes; Go returns them in Policy.Extra.
- Python accepts any status below 400; the generated Go call accepts only the native codes (create 201, get/update 200, delete 202/204) and reports others as gophercloud.ErrUnexpectedResponseCode wrapped in resource.OperationError.

### `python:identity/v3/policies`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.API.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.WithRecordListFilter`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.WithRecordListHeader`

- List sends GET policies without a query and follows links.next, All forwards type, limit and marker as wire query, and WithBodyFilter and WithName return resource.ErrUnsupported.
  Tests: `identity/v3/policies/python_parity_test.go:TestPythonPolicyListFilters`.
- Owned ListRecords classifies only limit/marker as server query and applies seven canonical nullable untyped Body filters locally, including recursive blob/links subsets; unknown filter marshalers are never invoked and remote extensions remain in independent Wire/Envelope receipts.
  Tests: `identity/v3/policies/python_list_records_test.go:TestPythonPolicyRecordsBodyFiltersStayLocal`, `identity/v3/policies/python_list_records_test.go:TestPythonPolicyRecordsFiltersAllDeclaredFieldsAndEarlyBreak`.
- Source-shaped rel/href paging, partial AllRecords on a malformed later row, raw caps before filters, explicit zero limit, lazy repeated snapshots, sticky selected-client changes and cancellation preserve the actual boundary.
  Tests: `identity/v3/policies/python_list_records_test.go:TestPythonPolicyRecordsPaginationPartialAndRawCap`, `identity/v3/policies/python_list_records_test.go:TestPythonPolicyRecordsOwnReusableOptionsAndRemainLazy`, `identity/v3/policies/python_list_records_test.go:TestPythonPolicyRecordsGuardAndCancellation`.

Python/Go differences:

- Legacy generated List/All retain native wire query behavior; owned ListRecords/AllRecords provide the actual local Policy Body filters while only limit/marker go to the server.
- The concrete owned list profile does not expose arbitrary Proxy kwargs, conflicting attrs, custom paths, microversion overrides or JMESPath. Finite JSON, guarded fixed paths/cycles, immutable receipts and Go error iteration are documented adaptations; full Resource/Connection/cache/runtime, SDK-R1/C1/S1, real authorization and release gaps remain global.

### `python:identity/v3/update_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/policies.WithUpdateField`

- Update sends PATCH policies/{id} with {"policy": {...}} holding only the given type, blob and Extra keys, including an empty blob via WithUpdateField and null via Extra, and decodes the updated policy.
  Tests: `identity/v3/policies/python_parity_test.go:TestPythonPolicyCreateGetUpdate`.

Python/Go differences:

- Python skips the request when no attribute changed; Go always sends PATCH, with {"policy": {}} for empty options.
- Empty Type and Blob are omitted by UpdateOpts; an empty blob is sent through WithUpdateField and an empty type through Extra.
- Python accepts a Resource instance or an ID string; Go takes the ID string or a resource.Ref.
- Python accepts any status below 400; the generated Go call accepts only the native codes (create 201, get/update 200, delete 202/204) and reports others as gophercloud.ErrUnexpectedResponseCode wrapped in resource.OperationError.

### `python:identity/v3/delete_mapping`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.API.DeleteMapping`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- resource.IgnoreMissing(DeleteMapping) sends one DELETE OS-FEDERATION/mappings/{id}, returns nil on 204 and 404 like ignore_missing=True and keeps 409; plain DeleteMapping keeps the 404 like ignore_missing=False.
  Tests: `identity/v3/federation/python_parity_test.go:TestPythonMappingDeleteIgnoreMissing`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go takes the ID string or a resource.Ref.
- Python accepts any status below 400; the generated Go call accepts only the native codes (create 201, get/update 200, delete 202/204) and reports others as gophercloud.ErrUnexpectedResponseCode wrapped in resource.OperationError.
- Mappings have no resource.Collection, so the ignore_missing default needs the explicit resource.IgnoreMissing wrapper.

## References

- [`identity/v3/_proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py)
- [`policies/python_parity_test.go`](policies/python_parity_test.go)
- [`federation/python_parity_test.go`](federation/python_parity_test.go)
- [native-admin-b.md](native-admin-b.md)
- [Python List Records](policies/python-list-records.md)
- [Source record](federation/python-source-mapping.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L912-L1015)
- [`policy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/policy.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2913-L3020)
- [`mapping.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/mapping.py)
- [Python Source Mapping](federation/python-source-mapping.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2725-L2912)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L3022-L3150)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L3218-L3340)
- [`identity_provider.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/identity_provider.py)
- [`federation_protocol.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/federation_protocol.py)
- [`service_provider.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/service_provider.py)
