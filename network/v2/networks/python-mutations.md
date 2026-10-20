# Python Mutations

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/update_network`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.UpdateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithIfMatch`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRevision`

- The concrete Go update for this operation emits exact revision_number=N for typed zero/positive/signed values, omits a nil generated condition and preserves the native route/body/response envelope. This all-nine Go matrix is evidence of Go behavior; Python conditional similarity or difference is established from the pinned class/helper sources.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- Final typed replacement selects body and condition together, preserves allowed body extensions, and rejects unsupported query/header/nil options before HTTP. These concrete option semantics do not emulate Python mutable Resource/attrs state.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- Native accepted-code and412 policies remain inspectable with no SDK unconditional fallback; native401/429/503 replay retains the prepared condition/body and live auth despite caller pointer mutation. Python adapter/session retry/error semantics remain a documented difference.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Representative Network API exercises the shared native escape path: service-wide If-Match override, RetryFunc explicit condition override and OmitHeaders removal remain effective. This is one Network fixture, not separate per-API raw-override coverage for all nine.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsRawNativeHeaderEscapePolicy`.
- Representative Network API exercises native307 redirect refusal or following with retained PUT/body/live auth and callback-overridden If-Match. This is one shared-path fixture, not per-API redirect coverage for the other eight.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeRedirectPolicy`.
- Literal-ID update with no Body/Header options performs no HTTP and returns a new ID-only native model. Supplied attrs send one PUT networks/{id}, with no lookup. Raw If-Match/WithRevision alone sends an empty network envelope. WithRevision(0) sends revision_number=0, overrides raw if_match regardless option order, discards invalid raw Header values before validation, and later revisions win.
  Tests: `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkMutationBodiesDefaultsAndAliases`, `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkRevisionHeaderOnlyPriorityAndClientIsolation`.
- WithAttribute snapshots JSON at constructor time; reusable options preserve false/zero/null/empty values, later wire-key values win, and aliases use the pinned generated Body MRO table: is_admin_state_up=>admin_state_up; is_router_external=>router:external; is_shared=>shared; is_port_security_enabled=>port_security_enabled; subnet_ids=>subnets; provider_network_type=>provider:network_type; provider_segmentation_id=>provider:segmentation_id; ipv4_address_scope_id=>ipv4_address_scope. Unknown and query-only attrs stay in Body. Invalid option/serialization/capability and Resource control attrs fail before HTTP.
  Tests: `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkMutationBodiesDefaultsAndAliases`, `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkOptionSnapshotsHeaderRoutingAndPreflight`.
- Inherited if_match Header is separate from Body. String values, string lists joined by comma-space, explicit empty list and null->None are supported, with constructor snapshots and configured client Header isolation. Create accepts raw If-Match; update accepts revision/If-Match, and explicitly selected conditions replace case-insensitive configured If-Match on a cloned client.
  Tests: `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkOptionSnapshotsHeaderRoutingAndPreflight`, `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkRevisionHeaderOnlyPriorityAndClientIsolation`.
- SDK HTTP success codes are 200..399 with unchanged native typed extraction. Network native Extract returns an ID-less zero model pointer and nil error on 204. Results are not seeded with request attrs/ID. OperationError retains native decode errors, HTTP status/body/header and pre-canceled/option-callback/post-response context.Canceled plus custom context.Cause.
  Tests: `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkMutationStatusesDecodeAndContextCauses`, `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkRevisionHeaderOnlyPriorityAndClientIsolation`.

Python/Go differences:

- A functional callable JSON mutation profile uses native Go models and a literal update ID, while existing native Create/Update ABI, required validations and result decoders are unchanged.
- The explicit SDK success profile is 200..399 and strict native decode remains. Network native Extract returns an ID-less zero model pointer and nil error on 204. Source can retain a mutable request Resource on empty/invalid JSON success; native Go results do not store response unknown fields, merge request attrs, or apply descriptor access/coercion.
- WithRevision accepts int64 (including zero, negative and exact max-int64), rather than arbitrary-precision Python values. Raw If-Match is a JSON string/string-list/null profile; arbitrary Python object str() coercion is unavailable. A selected Header containing CR/LF is rejected before HTTP. Raw Header values discarded by revision are not validated.
- Body controls base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location are rejected. Update id and if_revision must use the literal ID parameter and WithRevision rather than Body extensions. Configured Gophercloud session/client policy is retained.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Full Python Resource/Munch instance input, returning the same mutable instance, comparison against existing dirty/cache state and retaining request attrs or unknown response attrs remain separate from this callable Go profile.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Complete Proxy/Resource control kwargs, computed location/metadata, dynamic microversion, Python descriptor access/conversion and adapter/session/cache/exception policy remain separate.

### `python:network/v2/create_network`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.CreateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithIfMatch`

- POST networks sends only supplied Body attrs in the singular network envelope. Omitted attrs send an empty object, no native defaults, no required prevalidation. Neutron validates missing required values.
  Tests: `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkMutationBodiesDefaultsAndAliases`.
- WithAttribute snapshots JSON at constructor time; reusable options preserve false/zero/null/empty values, later wire-key values win, and aliases use the pinned generated Body MRO table: is_admin_state_up=>admin_state_up; is_router_external=>router:external; is_shared=>shared; is_port_security_enabled=>port_security_enabled; subnet_ids=>subnets; provider_network_type=>provider:network_type; provider_segmentation_id=>provider:segmentation_id; ipv4_address_scope_id=>ipv4_address_scope. Unknown and query-only attrs stay in Body. Invalid option/serialization/capability and Resource control attrs fail before HTTP.
  Tests: `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkMutationBodiesDefaultsAndAliases`, `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkOptionSnapshotsHeaderRoutingAndPreflight`.
- Inherited if_match Header is separate from Body. String values, string lists joined by comma-space, explicit empty list and null->None are supported, with constructor snapshots and configured client Header isolation. Create accepts raw If-Match; update accepts revision/If-Match, and explicitly selected conditions replace case-insensitive configured If-Match on a cloned client.
  Tests: `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkOptionSnapshotsHeaderRoutingAndPreflight`, `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkRevisionHeaderOnlyPriorityAndClientIsolation`.
- SDK HTTP success codes are 200..399 with unchanged native typed extraction. Network native Extract returns an ID-less zero model pointer and nil error on 204. Results are not seeded with request attrs/ID. OperationError retains native decode errors, HTTP status/body/header and pre-canceled/option-callback/post-response context.Canceled plus custom context.Cause.
  Tests: `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkMutationStatusesDecodeAndContextCauses`, `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkRevisionHeaderOnlyPriorityAndClientIsolation`.

Python/Go differences:

- A functional callable JSON mutation profile uses native Go models and a literal update ID, while existing native Create/Update ABI, required validations and result decoders are unchanged.
- The explicit SDK success profile is 200..399 and strict native decode remains. Network native Extract returns an ID-less zero model pointer and nil error on 204. Source can retain a mutable request Resource on empty/invalid JSON success; native Go results do not store response unknown fields, merge request attrs, or apply descriptor access/coercion.
- WithRevision accepts int64 (including zero, negative and exact max-int64), rather than arbitrary-precision Python values. Raw If-Match is a JSON string/string-list/null profile; arbitrary Python object str() coercion is unavailable. A selected Header containing CR/LF is rejected before HTTP. Raw Header values discarded by revision are not validated.
- Body controls base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location are rejected. Update id and if_revision must use the literal ID parameter and WithRevision rather than Body extensions. Configured Gophercloud session/client policy is retained.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Full Python Resource/Munch instance input, returning the same mutable instance, comparison against existing dirty/cache state and retaining request attrs or unknown response attrs remain separate from this callable Go profile.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Complete Proxy/Resource control kwargs, computed location/metadata, dynamic microversion, Python descriptor access/conversion and adapter/session/cache/exception policy remain separate.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
conn.network.create_network(name="", is_admin_state_up=False, subnet_ids=[])
conn.network.update_network(network_id, name="", is_router_external=False, if_revision=0)
conn.network.update_network(resource_id)
conn.network.update_network(resource_id, if_revision=0)
```

```go
package main

import (
    "context"
    "fmt"
    "os"
    "strconv"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
    if err != nil { panic(err) }
    service, err := conn.Network(ctx)
    if err != nil { panic(err) }
    api := service.API.Networks
    if id := os.Getenv("NEUTRON_UPDATE_ID"); id != "" {
        options := []networks.AttributeOption{networks.WithAttribute("name", ""), networks.WithAttribute("is_router_external", false)}
        if value := os.Getenv("NEUTRON_REVISION"); value != "" {
            revision, err := strconv.ParseInt(value, 10, 64)
            if err != nil { panic(err) }
            options = append(options, networks.WithRevision(revision))
        }
        value, err := api.UpdateWithAttributes(ctx, id, options...)
        if err != nil { panic(err) }
        fmt.Printf("%#v\n", value)
        return
    }
    if os.Getenv("NEUTRON_CREATE") == "1" {
        value, err := api.CreateWithAttributes(ctx, networks.WithAttribute("name", ""), networks.WithAttribute("is_admin_state_up", false), networks.WithAttribute("subnet_ids", []string{}))
        if err != nil { panic(err) }
        fmt.Printf("%#v\n", value)
        return
    }
    fmt.Println("set NEUTRON_UPDATE_ID (optionally NEUTRON_REVISION) or NEUTRON_CREATE=1")
}
```
