# Python Mutations

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/update_port`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.UpdateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithIfMatch`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRevision`

- The concrete Go update for this operation emits exact revision_number=N for typed zero/positive/signed values, omits a nil generated condition and preserves the native route/body/response envelope. This all-nine Go matrix is evidence of Go behavior; Python conditional similarity or difference is established from the pinned class/helper sources.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- Final typed replacement selects body and condition together, preserves allowed body extensions, and rejects unsupported query/header/nil options before HTTP. These concrete option semantics do not emulate Python mutable Resource/attrs state.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- Native accepted-code and412 policies remain inspectable with no SDK unconditional fallback; native401/429/503 replay retains the prepared condition/body and live auth despite caller pointer mutation. Python adapter/session retry/error semantics remain a documented difference.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Literal-ID update with no Body/Header options performs no HTTP and returns an ID-only native model. Supplied attrs send a single PUT ports/{id} without lookup. A raw If-Match or WithRevision alone is dirty and sends {port:{}}. WithRevision(0) emits revision_number=0; it overrides raw if_match regardless option order, including invalid discarded raw values, and the last revision wins.
  Tests: `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortMutationBodiesDefaultsAndAliases`, `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortRevisionHeaderOnlyPriorityAndClientIsolation`.
- WithAttribute constructor snapshots supplied JSON and Header values; reusable options preserve zero/false/null/empty values, later wire-key values win, canonical Body aliases map through the existing pinned MRO table, unknown extension and query-only attrs stay in Body. binding_host_id=>binding:host_id; binding_profile=>binding:profile; is_admin_state_up=>admin_state_up; is_port_security_enabled=>port_security_enabled; security_group_ids=>security_groups. Invalid options/serialization, non-If-Match headers, query mechanisms, and Resource/Proxy control attrs fail before HTTP.
  Tests: `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortMutationBodiesDefaultsAndAliases`, `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortOptionSnapshotsHeaderRoutingAndPreflight`.
- Inherited if_match Header is separate from Body. String values, string lists joined by comma-space, explicit empty list and null->None are supported, with constructor snapshots and configured client Header isolation. Create accepts raw If-Match; update accepts revision/If-Match, and explicitly selected conditions replace case-insensitive configured If-Match on a cloned client.
  Tests: `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortOptionSnapshotsHeaderRoutingAndPreflight`, `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortRevisionHeaderOnlyPriorityAndClientIsolation`.
- SDK HTTP success codes are 200..399 and native typed result extraction is retained. Port native Extract returns an ID-less zero model pointer and nil error on 204. No response model is seeded with request ID/attrs. HTTP status/body/header, native JSON decode errors, and pre-canceled/option-callback/post-response context.Canceled plus custom context.Cause remain observable through OperationError.
  Tests: `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortMutationStatusesDecodeAndContextCauses`, `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortRevisionHeaderOnlyPriorityAndClientIsolation`.

Python/Go differences:

- A functional callable JSON mutation profile uses native Go models and a literal update ID, while existing native Create/Update ABI, required validations and result decoders are unchanged.
- HTTP 200..399 is the explicit SDK profile; strict native decode behavior and returned known fields are retained. Port native Extract returns an ID-less zero model pointer and nil error on 204. Source retains a mutable Resource request view on empty or invalid JSON success bodies; this Go API does not synthesize request attributes, returned unknown fields or partial-response fallback.
- WithRevision accepts int64 (including zero, negative and exact max-int64), rather than arbitrary-precision Python values. Raw If-Match is a JSON string/string-list/null profile; arbitrary Python object str() coercion is unavailable. A selected Header containing CR/LF is rejected before HTTP. Raw Header values discarded by revision are not validated.
- Body controls base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location are rejected. Update id and if_revision must use the literal ID parameter and WithRevision rather than Body extensions. Configured Gophercloud session/client policy is retained.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Full Python Resource/Munch instance input, returning the same mutable instance, comparison against existing dirty/cache state and retaining request attrs or unknown response attrs remain separate from this callable Go profile.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Complete Proxy/Resource control kwargs, computed location/metadata, dynamic microversion, Python descriptor access/conversion and adapter/session/cache/exception policy remain separate.

### `python:network/v2/create_port`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.CreateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithIfMatch`

- POST ports sends exactly the supplied singular port Body envelope. No attributes still sends {port:{}} without native required-value prevalidation or synthesized defaults; Neutron decides missing required values.
  Tests: `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortMutationBodiesDefaultsAndAliases`.
- WithAttribute constructor snapshots supplied JSON and Header values; reusable options preserve zero/false/null/empty values, later wire-key values win, canonical Body aliases map through the existing pinned MRO table, unknown extension and query-only attrs stay in Body. binding_host_id=>binding:host_id; binding_profile=>binding:profile; is_admin_state_up=>admin_state_up; is_port_security_enabled=>port_security_enabled; security_group_ids=>security_groups. Invalid options/serialization, non-If-Match headers, query mechanisms, and Resource/Proxy control attrs fail before HTTP.
  Tests: `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortMutationBodiesDefaultsAndAliases`, `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortOptionSnapshotsHeaderRoutingAndPreflight`.
- Inherited if_match Header is separate from Body. String values, string lists joined by comma-space, explicit empty list and null->None are supported, with constructor snapshots and configured client Header isolation. Create accepts raw If-Match; update accepts revision/If-Match, and explicitly selected conditions replace case-insensitive configured If-Match on a cloned client.
  Tests: `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortOptionSnapshotsHeaderRoutingAndPreflight`, `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortRevisionHeaderOnlyPriorityAndClientIsolation`.
- SDK HTTP success codes are 200..399 and native typed result extraction is retained. Port native Extract returns an ID-less zero model pointer and nil error on 204. No response model is seeded with request ID/attrs. HTTP status/body/header, native JSON decode errors, and pre-canceled/option-callback/post-response context.Canceled plus custom context.Cause remain observable through OperationError.
  Tests: `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortMutationStatusesDecodeAndContextCauses`, `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortRevisionHeaderOnlyPriorityAndClientIsolation`.

Python/Go differences:

- A functional callable JSON mutation profile uses native Go models and a literal update ID, while existing native Create/Update ABI, required validations and result decoders are unchanged.
- HTTP 200..399 is the explicit SDK profile; strict native decode behavior and returned known fields are retained. Port native Extract returns an ID-less zero model pointer and nil error on 204. Source retains a mutable Resource request view on empty or invalid JSON success bodies; this Go API does not synthesize request attributes, returned unknown fields or partial-response fallback.
- WithRevision accepts int64 (including zero, negative and exact max-int64), rather than arbitrary-precision Python values. Raw If-Match is a JSON string/string-list/null profile; arbitrary Python object str() coercion is unavailable. A selected Header containing CR/LF is rejected before HTTP. Raw Header values discarded by revision are not validated.
- Body controls base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location are rejected. Update id and if_revision must use the literal ID parameter and WithRevision rather than Body extensions. Configured Gophercloud session/client policy is retained.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Full Python Resource/Munch instance input, returning the same mutable instance, comparison against existing dirty/cache state and retaining request attrs or unknown response attrs remain separate from this callable Go profile.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Complete Proxy/Resource control kwargs, computed location/metadata, dynamic microversion, Python descriptor access/conversion and adapter/session/cache/exception policy remain separate.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
conn.network.create_port(network_id=network_id, is_admin_state_up=False, security_group_ids=[])
conn.network.update_port(port_id, name="", binding_host_id="", if_revision=0)
conn.network.update_port(resource_id)
conn.network.update_port(resource_id, if_revision=0)
```

```go
package main

import (
    "context"
    "fmt"
    "os"
    "strconv"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/ports"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
    if err != nil { panic(err) }
    service, err := conn.Network(ctx)
    if err != nil { panic(err) }
    api := service.API.Ports
    if id := os.Getenv("NEUTRON_UPDATE_ID"); id != "" {
        options := []ports.AttributeOption{ports.WithAttribute("name", ""), ports.WithAttribute("binding_host_id", "")}
        if value := os.Getenv("NEUTRON_REVISION"); value != "" {
            revision, err := strconv.ParseInt(value, 10, 64)
            if err != nil { panic(err) }
            options = append(options, ports.WithRevision(revision))
        }
        value, err := api.UpdateWithAttributes(ctx, id, options...)
        if err != nil { panic(err) }
        fmt.Printf("%#v\n", value)
        return
    }
    if os.Getenv("NEUTRON_CREATE") == "1" {
        value, err := api.CreateWithAttributes(ctx, ports.WithAttribute("network_id", os.Getenv("NETWORK_ID")), ports.WithAttribute("is_admin_state_up", false), ports.WithAttribute("security_group_ids", []string{}))
        if err != nil { panic(err) }
        fmt.Printf("%#v\n", value)
        return
    }
    fmt.Println("set NEUTRON_UPDATE_ID (optionally NEUTRON_REVISION) or NEUTRON_CREATE=1")
}
```
