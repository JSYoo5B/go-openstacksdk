# Ndpproxies

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_ndp_proxy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithCreateMicroversion`

- POST /ndp_proxies sends provided attrs (null and unknown attrs kept) under ndp_proxy, keeps the untyped revision_number and unknown tenant_id, rejects constructor controls before HTTP and reports 400 with Create context.
  Tests: `network/v2/extensions/layer3/ndpproxies/python_records_test.go:TestPythonNDPProxyCreateSendsProvidedAttrs`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.

### `python:network/v2/get_ndp_proxy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithGetFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithGetMicroversion`

- GET /ndp_proxies/{id} sends repeated fields, escapes the literal ID and reports 404 with Get context.
  Tests: `network/v2/extensions/layer3/ndpproxies/python_records_test.go:TestPythonNDPProxyGetUpdateAndDelete`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.

### `python:network/v2/find_ndp_proxy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithFindMicroversion`

- Find GETs the literal ID, falls back on clean 403/404 to a complete name-hinted list that follows ndp_proxies_links, and returns nil or ErrNotFound when missing.
  Tests: `network/v2/extensions/layer3/ndpproxies/python_records_test.go:TestPythonNDPProxyFindFallsBackToCompleteNameList`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- find returns nil, nil for a clean miss by default (Python None) and resource.ErrNotFound/ErrAmbiguous instead of NotFoundException/DuplicateResource.
- When following next links Go keeps the original filters and fails on a link that changes them or repeats a page; Python follows the href as is. Local Body filters compare JSON numbers exactly rather than with Python float rounding.

### `python:network/v2/delete_ndp_proxy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithDeleteMicroversion`

- Bodyless DELETE ignores a clean 404 by default, WithDeleteIgnoreMissing(false) returns it and 204 is a plain receipt.
  Tests: `network/v2/extensions/layer3/ndpproxies/python_records_test.go:TestPythonNDPProxyGetUpdateAndDelete`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- Python returns None; Go returns a DeleteResult with the actual status, header and body, and IgnoredMissing=true when the default ignore_missing suppressed a clean 404.

### `python:network/v2/ndp_proxies`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithListMaxItems`

- List reads only the first page despite ndp_proxies_links, sends declared query parameters, filters undeclared-query Body attrs locally, rejects a paginated kwarg before HTTP and reports 500 with List context.
  Tests: `network/v2/extensions/layer3/ndpproxies/python_records_test.go:TestPythonNDPProxyListReadsOnlyTheFirstPage`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- ndp_proxies always lists with paginated=False, so Go reads only the first page; a caller paginated kwarg is the Python duplicate-keyword TypeError and Go rejects it before HTTP with resource.ErrInvalidOption.
- When following next links Go keeps the original filters and fails on a link that changes them or repeats a page; Python follows the href as is. Local Body filters compare JSON numbers exactly rather than with Python float rounding.

### `python:network/v2/update_ndp_proxy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithUpdateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/ndpproxies.WithUpdateMicroversion`

- PUT /ndp_proxies/{id} sends only changed attrs, the empty update sends nothing and 409 keeps Update context.
  Tests: `network/v2/extensions/layer3/ndpproxies/python_records_test.go:TestPythonNDPProxyGetUpdateAndDelete`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
proxy = conn.network.create_ndp_proxy(
    router_id=router.id, port_id=port.id, ip_address="2001:db8::10")
conn.network.update_ndp_proxy(proxy, description="web")
for item in conn.network.ndp_proxies(router_id=router.id):
    print(item.ip_address)
conn.network.delete_ndp_proxy(proxy)
```

```go
api := ndpproxies.New(networkClient)
proxy, err := api.Create(ctx,
    ndpproxies.WithCreateField("router_id", routerID),
    ndpproxies.WithCreateField("port_id", portID),
    ndpproxies.WithCreateField("ip_address", "2001:db8::10"))
if err != nil {
    return err
}
id := proxy.Text("id")
if _, err := api.Update(ctx, id, ndpproxies.WithUpdateField("description", "web")); err != nil {
    return err
}
for item, err := range api.List(ctx, ndpproxies.WithListQuery("router_id", routerID)) {
    if err != nil {
        return err
    }
    fmt.Println(item.Text("ip_address"))
}
_, err = api.Delete(ctx, id)
return err
```
