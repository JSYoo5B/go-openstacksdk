# Python Mutations

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_security_group_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.API.CreateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithIfMatch`

- POST singular security_group_rule envelope with supplied attrs; no attrs sends {}. Pinned SecurityGroupRule._prepare_request removes remote_address_group_id only when its JSON value is falsy (null/false/numeric zero/empty string/list/object); truthy string/container/nonzero exact number values remain.
  Tests: `network/v2/extensions/security/rules/python_mutations_contracts_test.go:TestPythonSecurityRuleCreatePresenceAliasesAndRemoteAddressGroup`.
- Canonical ether_type maps to ethertype through the pinned Body MRO map; unknown/query-only attrs remain Body. Constructor snapshots preserve empty strings, null, false and numeric zero without Direction/EtherType/SecGroupID native required validation. Invalid row/options/Header/query/Resource-control inputs fail before HTTP.
  Tests: `network/v2/extensions/security/rules/python_mutations_contracts_test.go:TestPythonSecurityRuleCreatePresenceAliasesAndRemoteAddressGroup`, `network/v2/extensions/security/rules/python_mutations_contracts_test.go:TestPythonSecurityRuleOptionSnapshotsHeadersAndPreflight`, `network/v2/extensions/security/rules/python_mutations_contracts_test.go:TestPythonSecurityRuleBulkRejectsInvalidRowsBeforeHTTP`.
- Inherited if_match stays out of Body and supports string/string-list/null (None), explicit empty Header, case-insensitive configured If-Match replacement on a cloned client and reusable snapshots. Null/non-string list elements and selected CRLF Header are rejected before HTTP.
  Tests: `network/v2/extensions/security/rules/python_mutations_contracts_test.go:TestPythonSecurityRuleOptionSnapshotsHeadersAndPreflight`, `network/v2/extensions/security/rules/python_mutations_contracts_test.go:TestPythonSecurityRuleBulkBodyOrderLastHeaderAndSnapshots`.
- HTTP 200..399 succeeds under the Go profile; native single/slice decoders are retained. Native 204 yields nil result/nil error without reading HTTP Body or synthesizing request values. HTTP status/body/header, native JSON errors and custom context causes before callbacks/after HTTP remain observable through OperationError.
  Tests: `network/v2/extensions/security/rules/python_mutations_contracts_test.go:TestPythonSecurityRuleMutationStatusesDecodeAndContextCauses`.

Python/Go differences:

- A callable functional JSON-attrs API uses native Go result models and does not change existing typed Create/CreateBulk validation or result extraction.
- Source returns a mutable Resource for singular create and a Resource generator after bulk HTTP/response.json(); this Go bulk method returns an eager native decoded slice. Success is the explicit 200..399 SDK profile. Native 204 returns nil,nil in both methods, while Source bulk unconditional response.json() fails on a bodyless 204. Request-state/partial-response fallback and unknown response attrs are not synthesized.
- Raw If-Match supports only JSON string/string-list/null and rejects selected CRLF values and null/non-string list elements before HTTP; arbitrary Python object str() coercion is unavailable.
- Resource/Proxy control attrs base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location, generic query options and arbitrary Header options are rejected. Configured Gophercloud client/session policies are retained.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Full mutable Resource/Munch instance input, identity, dirty/cache state, request attrs on partial/empty responses and unknown returned fields remain separate from the callable Go profile.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Complete Proxy/Resource control kwargs, Python descriptor access/conversion, computed location/metadata, dynamic microversion and adapter/session/cache/exception policy remain separate.

### `python:network/v2/create_security_group_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.API.CreateBulkWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithIfMatch`

- A nonempty variadic sequence of []AttributeOption rows sends one ordered POST plural security_group_rules envelope; empty row is {} and empty total rows fails before HTTP. All row option slices are copied before callbacks. Source bulk prepares unwrapped rows so falsy remote_address_group_id values remain. Source uses the last row Header only; earlier invalid Headers still fail during row preparation. Configured Header is retained when final row has none.
  Tests: `network/v2/extensions/security/rules/python_mutations_contracts_test.go:TestPythonSecurityRuleBulkBodyOrderLastHeaderAndSnapshots`, `network/v2/extensions/security/rules/python_mutations_contracts_test.go:TestPythonSecurityRuleBulkRejectsInvalidRowsBeforeHTTP`.
- Canonical ether_type maps to ethertype through the pinned Body MRO map; unknown/query-only attrs remain Body. Constructor snapshots preserve empty strings, null, false and numeric zero without Direction/EtherType/SecGroupID native required validation. Invalid row/options/Header/query/Resource-control inputs fail before HTTP.
  Tests: `network/v2/extensions/security/rules/python_mutations_contracts_test.go:TestPythonSecurityRuleCreatePresenceAliasesAndRemoteAddressGroup`, `network/v2/extensions/security/rules/python_mutations_contracts_test.go:TestPythonSecurityRuleOptionSnapshotsHeadersAndPreflight`, `network/v2/extensions/security/rules/python_mutations_contracts_test.go:TestPythonSecurityRuleBulkRejectsInvalidRowsBeforeHTTP`.
- Inherited if_match stays out of Body and supports string/string-list/null (None), explicit empty Header, case-insensitive configured If-Match replacement on a cloned client and reusable snapshots. Null/non-string list elements and selected CRLF Header are rejected before HTTP.
  Tests: `network/v2/extensions/security/rules/python_mutations_contracts_test.go:TestPythonSecurityRuleOptionSnapshotsHeadersAndPreflight`, `network/v2/extensions/security/rules/python_mutations_contracts_test.go:TestPythonSecurityRuleBulkBodyOrderLastHeaderAndSnapshots`.
- HTTP 200..399 succeeds under the Go profile; native single/slice decoders are retained. Native 204 yields nil result/nil error without reading HTTP Body or synthesizing request values. HTTP status/body/header, native JSON errors and custom context causes before callbacks/after HTTP remain observable through OperationError.
  Tests: `network/v2/extensions/security/rules/python_mutations_contracts_test.go:TestPythonSecurityRuleMutationStatusesDecodeAndContextCauses`.

Python/Go differences:

- A callable functional JSON-attrs API uses native Go result models and does not change existing typed Create/CreateBulk validation or result extraction.
- Source returns a mutable Resource for singular create and a Resource generator after bulk HTTP/response.json(); this Go bulk method returns an eager native decoded slice. Success is the explicit 200..399 SDK profile. Native 204 returns nil,nil in both methods, while Source bulk unconditional response.json() fails on a bodyless 204. Request-state/partial-response fallback and unknown response attrs are not synthesized.
- Raw If-Match supports only JSON string/string-list/null and rejects selected CRLF values and null/non-string list elements before HTTP; arbitrary Python object str() coercion is unavailable.
- Resource/Proxy control attrs base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location, generic query options and arbitrary Header options are rejected. Configured Gophercloud client/session policies are retained.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Full mutable Resource/Munch instance input, identity, dirty/cache state, request attrs on partial/empty responses and unknown returned fields remain separate from the callable Go profile.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Complete Proxy/Resource control kwargs, Python descriptor access/conversion, computed location/metadata, dynamic microversion and adapter/session/cache/exception policy remain separate.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
conn.network.create_security_group_rule(
    security_group_id=group_id, direction="ingress", ether_type="IPv4",
    protocol=None, port_range_min=0, remote_address_group_id="",
)

conn.network.create_security_group_rules([
    dict(security_group_id=group_id, direction="ingress", ether_type="IPv4", remote_address_group_id=""),
    dict(security_group_id=group_id, direction="egress", ether_type="IPv6", remote_address_group_id=None),
])
```

```go
package main

import (
    "context"
    "fmt"
    "os"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
    if err != nil { panic(err) }
    service, err := conn.Network(ctx)
    if err != nil { panic(err) }
    api := service.API.SecurityRules
    if os.Getenv("NEUTRON_CREATE") != "1" {
        fmt.Println("set NEUTRON_CREATE=1 and SECURITY_GROUP_ID (optionally NEUTRON_BULK=1)")
        return
    }
    first := []rules.AttributeOption{
        rules.WithAttribute("security_group_id", os.Getenv("SECURITY_GROUP_ID")),
        rules.WithAttribute("direction", "ingress"), rules.WithAttribute("ether_type", "IPv4"),
        rules.WithAttribute("protocol", nil), rules.WithAttribute("remote_address_group_id", ""),
    }
    if os.Getenv("NEUTRON_BULK") == "1" {
        second := []rules.AttributeOption{
            rules.WithAttribute("security_group_id", os.Getenv("SECURITY_GROUP_ID")),
            rules.WithAttribute("direction", "egress"), rules.WithAttribute("ether_type", "IPv6"),
            rules.WithAttribute("remote_address_group_id", nil),
        }
        values, err := api.CreateBulkWithAttributes(ctx, first, second)
        if err != nil { panic(err) }
        fmt.Printf("%#v\n", values)
        return
    }
    value, err := api.CreateWithAttributes(ctx, first...)
    if err != nil { panic(err) }
    fmt.Printf("%#v\n", value)
}
```
