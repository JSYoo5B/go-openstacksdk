# Python Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/project_endpoints`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projectendpoints.WithRecordListOpts`

- ListRecords(projectID) sends GET OS-EP-FILTER/projects/{escaped project}/endpoints with Accept application/json, maps interface, region_id and service_id to Requests-encoded server query, filters id, name, is_enabled/enabled, links and url locally, ignores unknown names, and decodes enabled, region_id, links and the URI project_id on every row with raw Body and Wire.
  Tests: `identity/v3/projectendpoints/python_records_test.go:TestPythonProjectEndpointRecordListQueryFiltersAndModel`.
- limit triggers Python marker continuation until an empty page, max_items alone becomes the limit hint and caps rows, a blank project ID fails before HTTP, and 400/403/404/500 keep the status in resource.OperationError Operation List.
  Tests: `identity/v3/projectendpoints/python_records_test.go:TestPythonProjectEndpointRecordListPagingAndStatuses`.

Python/Go differences:

- Python takes a project ID string or Project Resource and returns ProjectEndpoint Resources; Go takes the project ID string and yields owned EndpointRecord values.
- Go follows a string Keystone links.next URL as a compatibility extension; the pinned Python paginator ignores that dictionary shape and relies on limit/marker continuation.
- Local Body filters compare JSON values, so Go distinguishes bool from number and does not apply Python type coercion; nested dictionary filters match subsets like Resource.list _dict_filter.
- Per-request header options are a Go extension; authentication and content headers stay SDK-owned, and the Resource location attribute and deprecation warnings are not reproduced.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := projectendpoints.New(identityClient)
values, err := api.AllRecords(ctx, projectID,
    projectendpoints.WithRecordListQuery("interface", "public"),
    projectendpoints.WithRecordListQuery("is_enabled", true))
```

## References

- [`_proxy.py` project_endpoints](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L537-L555)
- [`endpoint.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/endpoint.py)
