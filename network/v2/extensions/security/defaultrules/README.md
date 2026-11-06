# Defaultrules

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_default_security_group_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/request.Field`
- `github.com/JSYoo5B/go-openstacksdk/request.Headers`
- `github.com/JSYoo5B/go-openstacksdk/request.Microversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules.WithCreateIfMatch`

- POST /default-security-group-rules maps ether_type to ethertype, sends the if_match Header list joined with ', ' as If-Match, projects int/bool casts, revision_number and the if_match view, rejects non-string if_match before HTTP and reports 409 with Create context.
  Tests: `network/v2/extensions/security/defaultrules/python_records_test.go:TestPythonDefaultRuleCreateMapsEtherTypeAndIfMatch`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- if_match attrs must be a string, string list or null; Python would str() any other value while Go rejects it before HTTP.

### `python:network/v2/delete_default_security_group_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/request.IgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/request.Headers`
- `github.com/JSYoo5B/go-openstacksdk/request.Microversion`

- Bodyless DELETE ignores a clean 404 by default, request.IgnoreMissing(false) returns it and 204 is a plain receipt.
  Tests: `network/v2/extensions/security/defaultrules/python_records_test.go:TestPythonDefaultRuleGetAndDelete`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- Python returns None; Go returns a DeleteResult with the actual status, header and body, and IgnoredMissing=true when the default ignore_missing suppressed a clean 404.

### `python:network/v2/find_default_security_group_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules.WithFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/request.IgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/request.Query`
- `github.com/JSYoo5B/go-openstacksdk/request.Headers`
- `github.com/JSYoo5B/go-openstacksdk/request.Microversion`

- Find GETs the literal ID with the find query, falls back on a clean 404 to a complete list without a name hint (no name QueryParameter) and returns nil or ErrNotFound when missing.
  Tests: `network/v2/extensions/security/defaultrules/python_records_test.go:TestPythonDefaultRuleFindHasNoNameHint`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- find returns nil, nil for a clean miss by default (Python None) and resource.ErrNotFound/ErrAmbiguous instead of NotFoundException/DuplicateResource.
- When following next links Go keeps the original filters and fails on a link that changes them or repeats a page; Python follows the href as is. Local Body filters compare JSON numbers exactly rather than with Python float rounding.

### `python:network/v2/get_default_security_group_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules.WithGetFields`
- `github.com/JSYoo5B/go-openstacksdk/request.Headers`
- `github.com/JSYoo5B/go-openstacksdk/request.Microversion`

- GET /default-security-group-rules/{id} maps fields attribute names to wire names (ether_type to ethertype) and reports 404 with Get context.
  Tests: `network/v2/extensions/security/defaultrules/python_records_test.go:TestPythonDefaultRuleGetAndDelete`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.

### `python:network/v2/default_security_group_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultrules.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/request.Query`
- `github.com/JSYoo5B/go-openstacksdk/request.Headers`
- `github.com/JSYoo5B/go-openstacksdk/request.Microversion`
- `github.com/JSYoo5B/go-openstacksdk/request.Limit`
- `github.com/JSYoo5B/go-openstacksdk/request.Marker`
- `github.com/JSYoo5B/go-openstacksdk/request.MaxItems`
- `github.com/JSYoo5B/go-openstacksdk/request.Paginated`

- List maps ether_type to the ethertype query, encodes bools as True, filters name locally, discards unknown kwargs, follows default_security_group_rules_links and reports 401 with List context.
  Tests: `network/v2/extensions/security/defaultrules/python_records_test.go:TestPythonDefaultRuleListMapsQueryAliases`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- When following next links Go keeps the original filters and fails on a link that changes them or repeats a page; Python follows the href as is. Local Body filters compare JSON numbers exactly rather than with Python float rounding.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
rule = conn.network.create_default_security_group_rule(
    direction="ingress", ether_type="IPv4", protocol="tcp",
    port_range_min=22, port_range_max=22, used_in_default_sg=True)
for item in conn.network.default_security_group_rules(ether_type="IPv4"):
    print(item.id, item.port_range_min)
conn.network.delete_default_security_group_rule(rule)
```

```go
api := defaultrules.New(networkClient)
rule, err := api.Create(ctx,
    request.Field("direction", "ingress"),
    request.Field("ether_type", "IPv4"),
    request.Field("protocol", "tcp"),
    request.Field("port_range_min", 22),
    request.Field("port_range_max", 22),
    request.Field("used_in_default_sg", true))
if err != nil {
    return err
}
for item, err := range api.List(ctx, request.Query("ether_type", "IPv4")) {
    if err != nil {
        return err
    }
    fmt.Println(item.Text("id"), string(item.Resource.Body["port_range_min"]))
}
_, err = api.Delete(ctx, rule.Text("id"))
return err
```
