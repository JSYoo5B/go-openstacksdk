# Python Mapping Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/get_mapping`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.API.GetMappingRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.WithMappingRecordReadHeaders`

- GetMappingRecord sends GET OS-FEDERATION/mappings/{id} with the ID escaped as one segment, accepts every status from 200 through 399, reads the mapping envelope or root object and keeps schema_version and the raw rules JSON (including rule keys outside the native MappingRule structs) in typed fields, Body and Wire.
  Tests: `identity/v3/federation/python_mapping_records_test.go:TestPythonMappingRecordGetKeepsSchemaVersionAndRawRules`.
- HTTP 400 and above return the native unexpected-status error wrapped in resource.OperationError with Resource mappings and Operation Get.
  Tests: `identity/v3/federation/python_mapping_records_test.go:TestPythonMappingRecordGetKeepsSchemaVersionAndRawRules`.

Python/Go differences:

- Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes an ID string, escapes it as one URL segment, rejects blank, dot and control-character IDs before HTTP and returns an owned record with typed fields plus raw Body, Wire, Envelope, Header and StatusCode.
- Python converts schema_version to str and rules to list; Go keeps the JSON values as returned.
- Per-request header options are a Go extension; authentication and content headers stay SDK-owned, and the Resource location attribute and deprecation warnings are not reproduced.

### `python:identity/v3/mappings`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.API.ListMappingRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.API.AllMappingRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.WithMappingRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.WithMappingRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.WithMappingRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.WithMappingRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.WithMappingRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.WithMappingRecordListOpts`

- ListMappingRecords sends GET OS-FEDERATION/mappings with Accept application/json and no query by default, filters id, name, rules and schema_version locally because Mapping declares no server query, ignores unknown names, and keeps schema_version and raw rules per row.
  Tests: `identity/v3/federation/python_mapping_records_test.go:TestPythonMappingRecordListFiltersPagingFindUpdateDelete`.
- limit and marker are sent like Python and, without a next link, the list repeats with marker set to the last row ID until an empty page; list failures keep the status in resource.OperationError Operation List.
  Tests: `identity/v3/federation/python_mapping_records_test.go:TestPythonMappingRecordListFiltersPagingFindUpdateDelete`.

Python/Go differences:

- Python returns mutable Mapping Resources; Go yields owned records with typed ID and SchemaVersion, raw Rules, Body, Wire and page evidence.
- Go follows a string Keystone links.next URL as a compatibility extension; the pinned Python paginator ignores that dictionary shape and relies on limit/marker continuation.
- Local Body filters compare JSON values, so Go distinguishes bool from number and does not apply Python type coercion; nested dictionary filters match subsets like Resource.list _dict_filter.
- Per-request header options are a Go extension; authentication and content headers stay SDK-owned, and the Resource location attribute and deprecation warnings are not reproduced.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := federation.New(identityClient)
m, err := api.GetMappingRecord(ctx, "corp-mapping")
if err != nil { return err }
fmt.Println(*m.SchemaVersion, string(m.Rules))
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2982-L3020)
- [`mapping.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/mapping.py)
