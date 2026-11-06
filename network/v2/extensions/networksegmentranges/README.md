# Networksegmentranges

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_network_segment_range`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/request.Field`
- `github.com/JSYoo5B/go-openstacksdk/request.Headers`
- `github.com/JSYoo5B/go-openstacksdk/request.Microversion`

- POST /network_segment_ranges sends only provided attrs (explicit null/false kept, unknown attrs kept, microversion attr dropped) under network_segment_range, an empty create sends {}, and the response projects bool/int/dict/list casts with unknown keys and null defaults.
  Tests: `network/v2/extensions/networksegmentranges/python_records_test.go:TestPythonSegmentRangeCreateSendsProvidedAttrs`.
- Route-control attrs fail before HTTP and a 409 rejection keeps the native status with Create operation context.
  Tests: `network/v2/extensions/networksegmentranges/python_records_test.go:TestPythonSegmentRangeCreateSendsProvidedAttrs`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.

### `python:network/v2/delete_network_segment_range`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/request.IgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/request.Headers`
- `github.com/JSYoo5B/go-openstacksdk/request.Microversion`

- Bodyless DELETE ignores a clean 404 by default, request.IgnoreMissing(false) returns it, and 409 stays an error with Delete context.
  Tests: `network/v2/extensions/networksegmentranges/python_records_test.go:TestPythonSegmentRangeDeleteIgnoresMissingByDefault`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- Python returns None; Go returns a DeleteResult with the actual status, header and body, and IgnoredMissing=true when the default ignore_missing suppressed a clean 404.

### `python:network/v2/find_network_segment_range`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges.WithFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/request.IgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/request.Query`
- `github.com/JSYoo5B/go-openstacksdk/request.Headers`
- `github.com/JSYoo5B/go-openstacksdk/request.Microversion`

- Find GETs the literal ID first, falls back on clean 400/404 to a complete list with the name query hint plus find query, returns nil or ErrNotFound when missing, ErrAmbiguous for duplicates and never falls back on 500.
  Tests: `network/v2/extensions/networksegmentranges/python_records_test.go:TestPythonSegmentRangeFindFallsBackToNameList`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- find returns nil, nil for a clean miss by default (Python None) and resource.ErrNotFound/ErrAmbiguous instead of NotFoundException/DuplicateResource.
- When following next links Go keeps the original filters and fails on a link that changes them or repeats a page; Python follows the href as is. Local Body filters compare JSON numbers exactly rather than with Python float rounding.

### `python:network/v2/get_network_segment_range`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges.WithGetFields`
- `github.com/JSYoo5B/go-openstacksdk/request.Headers`
- `github.com/JSYoo5B/go-openstacksdk/request.Microversion`

- GET /network_segment_ranges/{id} with repeated fields query, escaped literal ID, 2xx decode with the ID seed and 400/403/404/500 native errors with Get context; blank IDs fail before HTTP.
  Tests: `network/v2/extensions/networksegmentranges/python_records_test.go:TestPythonSegmentRangeGetFieldsStatusesAndEscapedID`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.

### `python:network/v2/network_segment_ranges`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/request.Query`
- `github.com/JSYoo5B/go-openstacksdk/request.Headers`
- `github.com/JSYoo5B/go-openstacksdk/request.Microversion`
- `github.com/JSYoo5B/go-openstacksdk/request.Limit`
- `github.com/JSYoo5B/go-openstacksdk/request.Marker`
- `github.com/JSYoo5B/go-openstacksdk/request.MaxItems`
- `github.com/JSYoo5B/go-openstacksdk/request.Paginated`

- List sends declared query parameters, filters other declared Body attrs (id) locally, discards unknown kwargs, sends Accept: application/json, follows network_segment_ranges_links and limit/marker continuation to an empty page, and paginated=False reads one page.
  Tests: `network/v2/extensions/networksegmentranges/python_records_test.go:TestPythonSegmentRangeListQueryFiltersAndLinksPaging`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- When following next links Go keeps the original filters and fails on a link that changes them or repeats a page; Python follows the href as is. Local Body filters compare JSON numbers exactly rather than with Python float rounding.

### `python:network/v2/update_network_segment_range`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networksegmentranges.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/request.Field`
- `github.com/JSYoo5B/go-openstacksdk/request.Headers`
- `github.com/JSYoo5B/go-openstacksdk/request.Microversion`

- PUT /network_segment_ranges/{id} sends only changed attrs; no attrs is the commit no-op without HTTP; id attr is rejected; 404 keeps native status with Update context.
  Tests: `network/v2/extensions/networksegmentranges/python_records_test.go:TestPythonSegmentRangeUpdatePutsChangedAttrs`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
rng = conn.network.create_network_segment_range(
    name="vlan-100", network_type="vlan", physical_network="physnet1",
    minimum=100, maximum=199, shared=True)
conn.network.update_network_segment_range(rng, maximum=299)
for item in conn.network.network_segment_ranges(network_type="vlan"):
    print(item.name, item.available)
conn.network.delete_network_segment_range(rng)
```

```go
api := networksegmentranges.New(networkClient)
rng, err := api.Create(ctx,
    request.Field("name", "vlan-100"),
    request.Field("network_type", "vlan"),
    request.Field("physical_network", "physnet1"),
    request.Field("minimum", 100),
    request.Field("maximum", 199),
    request.Field("shared", true))
if err != nil {
    return err
}
id := rng.Text("id")
if _, err := api.Update(ctx, id, request.Field("maximum", 299)); err != nil {
    return err
}
for item, err := range api.List(ctx, request.Query("network_type", "vlan")) {
    if err != nil {
        return err
    }
    fmt.Println(item.Text("name"), string(item.Resource.Body["available"]))
}
_, err = api.Delete(ctx, id)
return err
```
