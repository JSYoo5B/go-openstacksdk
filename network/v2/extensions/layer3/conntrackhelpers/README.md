# Conntrackhelpers

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/conntrack_helpers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithListPaginated`

- List sends only limit/marker as server query, filters declared Body attrs locally, discards unknown kwargs, follows conntrack_helpers_links and marker continuation, tags rows with RouterID, rejects router_id before HTTP and reports 404 with List context.
  Tests: `network/v2/extensions/layer3/conntrackhelpers/python_records_test.go:TestPythonConntrackListFiltersLocallyAndPages`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- Go takes the router ID first (Create/List/Get/Update/Delete(ctx, routerID, ...)); Python get/update/delete take the helper first and the router second. The router is never fetched, as Python _get_resource only extracts its ID.
- When following next links Go keeps the original filters and fails on a link that changes them or repeats a page; Python follows the href as is. Local Body filters compare JSON numbers exactly rather than with Python float rounding.

### `python:network/v2/create_conntrack_helper`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithCreateMicroversion`

- POST /routers/{router_id}/conntrack_helpers sends only declared attrs (unknown attrs discarded because the class does not allow them), keeps undeclared response keys only in Wire, exposes RouterID, rejects router_id and blank router before HTTP and reports 409 with Create context.
  Tests: `network/v2/extensions/layer3/conntrackhelpers/python_records_test.go:TestPythonConntrackCreateDropsUnknownAttrs`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- Go takes the router ID first (Create/List/Get/Update/Delete(ctx, routerID, ...)); Python get/update/delete take the helper first and the router second. The router is never fetched, as Python _get_resource only extracts its ID.

### `python:network/v2/get_conntrack_helper`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithGetMicroversion`

- GET /routers/{router_id}/conntrack_helpers/{id} sends no query and reports 404 with Get context.
  Tests: `network/v2/extensions/layer3/conntrackhelpers/python_records_test.go:TestPythonConntrackGetUpdateDelete`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- Go takes the router ID first (Create/List/Get/Update/Delete(ctx, routerID, ...)); Python get/update/delete take the helper first and the router second. The router is never fetched, as Python _get_resource only extracts its ID.

### `python:network/v2/update_conntrack_helper`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithUpdateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithUpdateMicroversion`

- PUT sends only declared changed attrs, ignores if_revision without If-Match (not a NetworkResource), the unknown-only update sends nothing, and 404 keeps Update context.
  Tests: `network/v2/extensions/layer3/conntrackhelpers/python_records_test.go:TestPythonConntrackGetUpdateDelete`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- Go takes the router ID first (Create/List/Get/Update/Delete(ctx, routerID, ...)); Python get/update/delete take the helper first and the router second. The router is never fetched, as Python _get_resource only extracts its ID.

### `python:network/v2/delete_conntrack_helper`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/conntrackhelpers.WithDeleteMicroversion`

- Bodyless DELETE ignores a clean 404 by default, WithDeleteIgnoreMissing(false) returns it and 204 is a plain receipt.
  Tests: `network/v2/extensions/layer3/conntrackhelpers/python_records_test.go:TestPythonConntrackGetUpdateDelete`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- Go takes the router ID first (Create/List/Get/Update/Delete(ctx, routerID, ...)); Python get/update/delete take the helper first and the router second. The router is never fetched, as Python _get_resource only extracts its ID.
- Python returns None; Go returns a DeleteResult with the actual status, header and body, and IgnoredMissing=true when the default ignore_missing suppressed a clean 404.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
helper = conn.network.create_conntrack_helper(
    router, helper="ftp", protocol="tcp", port=21)
conn.network.update_conntrack_helper(helper, router, port=2121)
for item in conn.network.conntrack_helpers(router, helper="ftp"):
    print(item.id, item.port)
conn.network.delete_conntrack_helper(helper, router)
```

```go
api := conntrackhelpers.New(networkClient)
helper, err := api.Create(ctx, routerID,
    conntrackhelpers.WithCreateField("helper", "ftp"),
    conntrackhelpers.WithCreateField("protocol", "tcp"),
    conntrackhelpers.WithCreateField("port", 21))
if err != nil {
    return err
}
id := helper.Text("id")
if _, err := api.Update(ctx, routerID, id, conntrackhelpers.WithUpdateField("port", 2121)); err != nil {
    return err
}
for item, err := range api.List(ctx, routerID, conntrackhelpers.WithListQuery("helper", "ftp")) {
    if err != nil {
        return err
    }
    fmt.Println(item.Text("id"), string(item.Resource.Body["port"]))
}
_, err = api.Delete(ctx, routerID, id)
return err
```
