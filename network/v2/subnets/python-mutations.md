# Python Mutations

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/update_subnet`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.UpdateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithIfMatch`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRevision`

- The concrete Go update for this operation emits exact revision_number=N for typed zero/positive/signed values, omits a nil generated condition and preserves the native route/body/response envelope. This all-nine Go matrix is evidence of Go behavior; Python conditional similarity or difference is established from the pinned class/helper sources.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- Final typed replacement selects body and condition together, preserves allowed body extensions, and rejects unsupported query/header/nil options before HTTP. These concrete option semantics do not emulate Python mutable Resource/attrs state.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- Native accepted-code and412 policies remain inspectable with no SDK unconditional fallback; native401/429/503 replay retains the prepared condition/body and live auth despite caller pointer mutation. Python adapter/session retry/error semantics remain a documented difference.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Literal-ID update with no Body/Header options performs no HTTP and returns an ID-only native model. Supplied attrs send a single PUT subnets/{id} without lookup. A raw If-Match or WithRevision alone is dirty and sends {subnet:{}}. WithRevision(0) emits revision_number=0; it overrides raw if_match regardless option order, including invalid discarded raw values, and the last revision wins.
  Tests: `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetMutationBodiesDefaultsAndAliases`, `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetRevisionHeaderOnlyPriorityAndClientIsolation`.
- WithAttribute constructor snapshots supplied JSON and Header values; reusable options preserve zero/false/null/empty values, later wire-key values win, canonical Body aliases map through the existing pinned MRO table, unknown extension and query-only attrs stay in Body. is_dhcp_enabled=>enable_dhcp; prefix_length=>prefixlen; subnet_pool_id=>subnetpool_id; use_default_subnet_pool=>use_default_subnetpool. Invalid options/serialization, non-If-Match headers, query mechanisms, and Resource/Proxy control attrs fail before HTTP.
  Tests: `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetMutationBodiesDefaultsAndAliases`, `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetOptionSnapshotsHeaderRoutingAndPreflight`.
- Inherited if_match Header is separate from Body. String values, string lists joined by comma-space, explicit empty list and null->None are supported, with constructor snapshots and configured client Header isolation. Create accepts raw If-Match; update accepts revision/If-Match, and explicitly selected conditions replace case-insensitive configured If-Match on a cloned client.
  Tests: `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetOptionSnapshotsHeaderRoutingAndPreflight`, `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetRevisionHeaderOnlyPriorityAndClientIsolation`.
- SDK HTTP success codes are 200..399 and native typed result extraction is retained. Subnet native Extract returns nil model and nil error on 204. No response model is seeded with request ID/attrs. HTTP status/body/header, native JSON decode errors, and pre-canceled/option-callback/post-response context.Canceled plus custom context.Cause remain observable through OperationError.
  Tests: `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetMutationStatusesDecodeAndContextCauses`, `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetRevisionHeaderOnlyPriorityAndClientIsolation`.

Python/Go differences:

- A functional callable JSON mutation profile uses native Go models and a literal update ID, while existing native Create/Update ABI, required validations and result decoders are unchanged.
- HTTP 200..399 is the explicit SDK profile; strict native decode behavior and returned known fields are retained. Subnet native Extract returns nil model and nil error on 204. Source retains a mutable Resource request view on empty or invalid JSON success bodies; this Go API does not synthesize request attributes, returned unknown fields or partial-response fallback.
- WithRevision accepts int64 (including zero, negative and exact max-int64), rather than arbitrary-precision Python values. Raw If-Match is a JSON string/string-list/null profile; arbitrary Python object str() coercion is unavailable. A selected Header containing CR/LF is rejected before HTTP. Raw Header values discarded by revision are not validated.
- Body controls base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location are rejected. Update id and if_revision must use the literal ID parameter and WithRevision rather than Body extensions. Configured Gophercloud session/client policy is retained.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Full Python Resource/Munch instance input, returning the same mutable instance, comparison against existing dirty/cache state and retaining request attrs or unknown response attrs remain separate from this callable Go profile.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Complete Proxy/Resource control kwargs, computed location/metadata, dynamic microversion, Python descriptor access/conversion and adapter/session/cache/exception policy remain separate.

### `python:network/v2/create_subnet`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.CreateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithIfMatch`

- POST subnets sends exactly the supplied singular subnet Body envelope. No attributes still sends {subnet:{}} without native required-value prevalidation or synthesized defaults; Neutron decides missing required values.
  Tests: `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetMutationBodiesDefaultsAndAliases`.
- WithAttribute constructor snapshots supplied JSON and Header values; reusable options preserve zero/false/null/empty values, later wire-key values win, canonical Body aliases map through the existing pinned MRO table, unknown extension and query-only attrs stay in Body. is_dhcp_enabled=>enable_dhcp; prefix_length=>prefixlen; subnet_pool_id=>subnetpool_id; use_default_subnet_pool=>use_default_subnetpool. Invalid options/serialization, non-If-Match headers, query mechanisms, and Resource/Proxy control attrs fail before HTTP.
  Tests: `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetMutationBodiesDefaultsAndAliases`, `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetOptionSnapshotsHeaderRoutingAndPreflight`.
- Inherited if_match Header is separate from Body. String values, string lists joined by comma-space, explicit empty list and null->None are supported, with constructor snapshots and configured client Header isolation. Create accepts raw If-Match; update accepts revision/If-Match, and explicitly selected conditions replace case-insensitive configured If-Match on a cloned client.
  Tests: `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetOptionSnapshotsHeaderRoutingAndPreflight`, `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetRevisionHeaderOnlyPriorityAndClientIsolation`.
- SDK HTTP success codes are 200..399 and native typed result extraction is retained. Subnet native Extract returns nil model and nil error on 204. No response model is seeded with request ID/attrs. HTTP status/body/header, native JSON decode errors, and pre-canceled/option-callback/post-response context.Canceled plus custom context.Cause remain observable through OperationError.
  Tests: `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetMutationStatusesDecodeAndContextCauses`, `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetRevisionHeaderOnlyPriorityAndClientIsolation`.

Python/Go differences:

- A functional callable JSON mutation profile uses native Go models and a literal update ID, while existing native Create/Update ABI, required validations and result decoders are unchanged.
- HTTP 200..399 is the explicit SDK profile; strict native decode behavior and returned known fields are retained. Subnet native Extract returns nil model and nil error on 204. Source retains a mutable Resource request view on empty or invalid JSON success bodies; this Go API does not synthesize request attributes, returned unknown fields or partial-response fallback.
- WithRevision accepts int64 (including zero, negative and exact max-int64), rather than arbitrary-precision Python values. Raw If-Match is a JSON string/string-list/null profile; arbitrary Python object str() coercion is unavailable. A selected Header containing CR/LF is rejected before HTTP. Raw Header values discarded by revision are not validated.
- Body controls base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location are rejected. Update id and if_revision must use the literal ID parameter and WithRevision rather than Body extensions. Configured Gophercloud session/client policy is retained.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Full Python Resource/Munch instance input, returning the same mutable instance, comparison against existing dirty/cache state and retaining request attrs or unknown response attrs remain separate from this callable Go profile.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Complete Proxy/Resource control kwargs, computed location/metadata, dynamic microversion, Python descriptor access/conversion and adapter/session/cache/exception policy remain separate.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
conn.network.create_subnet(network_id=network_id, ip_version=4, cidr="10.61.0.0/24", is_dhcp_enabled=False)
conn.network.update_subnet(subnet_id, gateway_ip=None, is_dhcp_enabled=False, if_revision=0)
conn.network.update_subnet(resource_id)
conn.network.update_subnet(resource_id, if_revision=0)
```

```go
package main

import (
    "context"
    "fmt"
    "os"
    "strconv"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/subnets"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
    if err != nil { panic(err) }
    service, err := conn.Network(ctx)
    if err != nil { panic(err) }
    api := service.API.Subnets
    if id := os.Getenv("NEUTRON_UPDATE_ID"); id != "" {
        options := []subnets.AttributeOption{subnets.WithAttribute("gateway_ip", nil), subnets.WithAttribute("is_dhcp_enabled", false)}
        if value := os.Getenv("NEUTRON_REVISION"); value != "" {
            revision, err := strconv.ParseInt(value, 10, 64)
            if err != nil { panic(err) }
            options = append(options, subnets.WithRevision(revision))
        }
        value, err := api.UpdateWithAttributes(ctx, id, options...)
        if err != nil { panic(err) }
        fmt.Printf("%#v\n", value)
        return
    }
    if os.Getenv("NEUTRON_CREATE") == "1" {
        value, err := api.CreateWithAttributes(ctx, subnets.WithAttribute("network_id", os.Getenv("NETWORK_ID")), subnets.WithAttribute("ip_version", 4), subnets.WithAttribute("cidr", "10.61.0.0/24"), subnets.WithAttribute("is_dhcp_enabled", false))
        if err != nil { panic(err) }
        fmt.Printf("%#v\n", value)
        return
    }
    fmt.Println("set NEUTRON_UPDATE_ID (optionally NEUTRON_REVISION) or NEUTRON_CREATE=1")
}
```
