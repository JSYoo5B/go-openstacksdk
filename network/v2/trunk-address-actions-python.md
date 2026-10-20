# Trunk Address Actions Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/add_addresses_to_address_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.AddAddressesRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithAddressActionOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithAddressActionSeed`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithAddressActionHeaders`

- One envelope-free PUT uses the fixed resource/action route; explicit null/false/zero/empty and nested unknown extension values are preserved without native required-field/default coercion. Trunk removal alone follows Source list(subports) for JSON arrays, ordered object keys and Unicode strings and rejects noniterables before HTTP.
  Tests: `network/v2/extensions/security/addressgroups/python_actions_contracts_test.go:TestPythonAddressActionsRoutesInputsAndEmptyValues`.
- The declared Source Resource projection is independent of actual Wire/Envelope/header/status and submitted Input; seeded attributes survive absent response fields, typed list/bool/default/nullable descriptors and project alias rules apply without native timestamp/numeric coercion. Trunk direct Body update differs from AddressGroup envelope/alias/unknown translation.
  Tests: `network/v2/extensions/security/addressgroups/python_actions_contracts_test.go:TestPythonAddressActionsSourceProjectionAndActualWire`, `network/v2/extensions/security/addressgroups/python_actions_contracts_test.go:TestPythonAddressActionsBodylessSuccessKeepsSourcePolicy`.
- Concrete header/seed options and input JSON are snapshotted; caller and retained-callback mutation cannot alter the applied request or view. Nil options, invalid IDs, malformed seed and protected auth headers fail before HTTP.
  Tests: `network/v2/extensions/security/addressgroups/python_actions_contracts_test.go:TestPythonAddressActionsSnapshotsHeaderPolicyAndInvalidOptions`.
- Callback/custom JSON marshaler cancellation and selected-source mutation preserve callback/context/source causes before HTTP. Actual accepted/rejected physical Read/Close/cancel/source-restored-on-Close errors retain original body/status/headers and never resend or become clean missing.
  Tests: `network/v2/extensions/security/addressgroups/python_actions_contracts_test.go:TestPythonAddressActionsCanceledOrChangedOptionsAndMarshalAreTerminal`, `network/v2/extensions/security/addressgroups/python_actions_contracts_test.go:TestPythonAddressActionsPhysicalAcceptedAndRejectedErrorsRetainProof`.
- A retry hook changing submitted JSON is terminal and retains the initial native HTTP rejection; Source successful statuses 200..399, literal one-time ID escaping and clean 404 rejection remain observable.
  Tests: `network/v2/extensions/security/addressgroups/python_actions_contracts_test.go:TestPythonAddressActionsRetryMutationKeepsOriginalRejection`, `network/v2/extensions/security/addressgroups/python_actions_contracts_test.go:TestPythonAddressActionsCleanSourceStatusPolicyAndEscapedID`.

Python/Go differences:

- Go returns an owned declared Resource view with null location and separate actual Wire/Envelope/Input/header/status rather than mutating a Python Resource; complete Connection cloud-location behavior remains separately tracked.
- The explicit string route ID cannot be replaced by a cached seed. Seed aliases that collide are rejected rather than depending on Go map order; ordinary Go map iterable order is its JSON encoding order while RawMessage preserves object insertion order.
- Inputs use the JSON domain; Python arbitrary iterables/custom objects are not implemented as Go builder interfaces. Optional owned headers/cache-seed are Go conveniences and retain shared auth/source protections.
- AddressGroup Source translator discards self and normalizes known aliases/unknown attributes. Bodyless/non-JSON success returns a seed view with actual HTTP metadata and nil Wire.

### `python:network/v2/add_trunk_subports`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.AddSubportsRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithSubportActionOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithSubportActionSeed`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithSubportActionHeaders`

- One envelope-free PUT uses the fixed resource/action route; explicit null/false/zero/empty and nested unknown extension values are preserved without native required-field/default coercion. Trunk removal alone follows Source list(subports) for JSON arrays, ordered object keys and Unicode strings and rejects noniterables before HTTP.
  Tests: `network/v2/extensions/trunks/python_actions_contracts_test.go:TestPythonTrunkActionsRoutesInputsAndEmptyValues`.
- The declared Source Resource projection is independent of actual Wire/Envelope/header/status and submitted Input; seeded attributes survive absent response fields, typed list/bool/default/nullable descriptors and project alias rules apply without native timestamp/numeric coercion. Trunk direct Body update differs from AddressGroup envelope/alias/unknown translation.
  Tests: `network/v2/extensions/trunks/python_actions_contracts_test.go:TestPythonTrunkActionsSourceProjectionAndActualWire`, `network/v2/extensions/trunks/python_actions_contracts_test.go:TestPythonTrunkActionsBodylessSuccessKeepsSourcePolicy`.
- Concrete header/seed options and input JSON are snapshotted; caller and retained-callback mutation cannot alter the applied request or view. Nil options, invalid IDs, malformed seed and protected auth headers fail before HTTP.
  Tests: `network/v2/extensions/trunks/python_actions_contracts_test.go:TestPythonTrunkActionsSnapshotsHeaderPolicyAndInvalidOptions`.
- Callback/custom JSON marshaler cancellation and selected-source mutation preserve callback/context/source causes before HTTP. Actual accepted/rejected physical Read/Close/cancel/source-restored-on-Close errors retain original body/status/headers and never resend or become clean missing.
  Tests: `network/v2/extensions/trunks/python_actions_contracts_test.go:TestPythonTrunkActionsCanceledOrChangedOptionsAndMarshalAreTerminal`, `network/v2/extensions/trunks/python_actions_contracts_test.go:TestPythonTrunkActionsPhysicalAcceptedAndRejectedErrorsRetainProof`.
- A retry hook changing submitted JSON is terminal and retains the initial native HTTP rejection; Source successful statuses 200..399, literal one-time ID escaping and clean 404 rejection remain observable.
  Tests: `network/v2/extensions/trunks/python_actions_contracts_test.go:TestPythonTrunkActionsRetryMutationKeepsOriginalRejection`, `network/v2/extensions/trunks/python_actions_contracts_test.go:TestPythonTrunkActionsCleanSourceStatusPolicyAndEscapedID`.

Python/Go differences:

- Go returns an owned declared Resource view with null location and separate actual Wire/Envelope/Input/header/status rather than mutating a Python Resource; complete Connection cloud-location behavior remains separately tracked.
- The explicit string route ID cannot be replaced by a cached seed. Seed aliases that collide are rejected rather than depending on Go map order; ordinary Go map iterable order is its JSON encoding order while RawMessage preserves object insertion order.
- Inputs use the JSON domain; Python arbitrary iterables/custom objects are not implemented as Go builder interfaces. Optional owned headers/cache-seed are Go conveniences and retain shared auth/source protections.
- Source direct trunk _body.update does not normalize response aliases/unpack an envelope/collect fresh unknown attrs; those exact HTTP fields stay available in Wire. Bodyless accepted trunk responses still require JSON and return an actual-response decode error.

### `python:network/v2/delete_trunk_subports`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.RemoveSubportsRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithSubportActionOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithSubportActionSeed`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithSubportActionHeaders`

- One envelope-free PUT uses the fixed resource/action route; explicit null/false/zero/empty and nested unknown extension values are preserved without native required-field/default coercion. Trunk removal alone follows Source list(subports) for JSON arrays, ordered object keys and Unicode strings and rejects noniterables before HTTP.
  Tests: `network/v2/extensions/trunks/python_actions_contracts_test.go:TestPythonTrunkActionsRoutesInputsAndEmptyValues`.
- The declared Source Resource projection is independent of actual Wire/Envelope/header/status and submitted Input; seeded attributes survive absent response fields, typed list/bool/default/nullable descriptors and project alias rules apply without native timestamp/numeric coercion. Trunk direct Body update differs from AddressGroup envelope/alias/unknown translation.
  Tests: `network/v2/extensions/trunks/python_actions_contracts_test.go:TestPythonTrunkActionsSourceProjectionAndActualWire`, `network/v2/extensions/trunks/python_actions_contracts_test.go:TestPythonTrunkActionsBodylessSuccessKeepsSourcePolicy`.
- Concrete header/seed options and input JSON are snapshotted; caller and retained-callback mutation cannot alter the applied request or view. Nil options, invalid IDs, malformed seed and protected auth headers fail before HTTP.
  Tests: `network/v2/extensions/trunks/python_actions_contracts_test.go:TestPythonTrunkActionsSnapshotsHeaderPolicyAndInvalidOptions`.
- Callback/custom JSON marshaler cancellation and selected-source mutation preserve callback/context/source causes before HTTP. Actual accepted/rejected physical Read/Close/cancel/source-restored-on-Close errors retain original body/status/headers and never resend or become clean missing.
  Tests: `network/v2/extensions/trunks/python_actions_contracts_test.go:TestPythonTrunkActionsCanceledOrChangedOptionsAndMarshalAreTerminal`, `network/v2/extensions/trunks/python_actions_contracts_test.go:TestPythonTrunkActionsPhysicalAcceptedAndRejectedErrorsRetainProof`.
- A retry hook changing submitted JSON is terminal and retains the initial native HTTP rejection; Source successful statuses 200..399, literal one-time ID escaping and clean 404 rejection remain observable.
  Tests: `network/v2/extensions/trunks/python_actions_contracts_test.go:TestPythonTrunkActionsRetryMutationKeepsOriginalRejection`, `network/v2/extensions/trunks/python_actions_contracts_test.go:TestPythonTrunkActionsCleanSourceStatusPolicyAndEscapedID`.

Python/Go differences:

- Go returns an owned declared Resource view with null location and separate actual Wire/Envelope/Input/header/status rather than mutating a Python Resource; complete Connection cloud-location behavior remains separately tracked.
- The explicit string route ID cannot be replaced by a cached seed. Seed aliases that collide are rejected rather than depending on Go map order; ordinary Go map iterable order is its JSON encoding order while RawMessage preserves object insertion order.
- Inputs use the JSON domain; Python arbitrary iterables/custom objects are not implemented as Go builder interfaces. Optional owned headers/cache-seed are Go conveniences and retain shared auth/source protections.
- Source direct trunk _body.update does not normalize response aliases/unpack an envelope/collect fresh unknown attrs; those exact HTTP fields stay available in Wire. Bodyless accepted trunk responses still require JSON and return an actual-response decode error.

### `python:network/v2/remove_addresses_from_address_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.RemoveAddressesRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithAddressActionOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithAddressActionSeed`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithAddressActionHeaders`

- One envelope-free PUT uses the fixed resource/action route; explicit null/false/zero/empty and nested unknown extension values are preserved without native required-field/default coercion. Trunk removal alone follows Source list(subports) for JSON arrays, ordered object keys and Unicode strings and rejects noniterables before HTTP.
  Tests: `network/v2/extensions/security/addressgroups/python_actions_contracts_test.go:TestPythonAddressActionsRoutesInputsAndEmptyValues`.
- The declared Source Resource projection is independent of actual Wire/Envelope/header/status and submitted Input; seeded attributes survive absent response fields, typed list/bool/default/nullable descriptors and project alias rules apply without native timestamp/numeric coercion. Trunk direct Body update differs from AddressGroup envelope/alias/unknown translation.
  Tests: `network/v2/extensions/security/addressgroups/python_actions_contracts_test.go:TestPythonAddressActionsSourceProjectionAndActualWire`, `network/v2/extensions/security/addressgroups/python_actions_contracts_test.go:TestPythonAddressActionsBodylessSuccessKeepsSourcePolicy`.
- Concrete header/seed options and input JSON are snapshotted; caller and retained-callback mutation cannot alter the applied request or view. Nil options, invalid IDs, malformed seed and protected auth headers fail before HTTP.
  Tests: `network/v2/extensions/security/addressgroups/python_actions_contracts_test.go:TestPythonAddressActionsSnapshotsHeaderPolicyAndInvalidOptions`.
- Callback/custom JSON marshaler cancellation and selected-source mutation preserve callback/context/source causes before HTTP. Actual accepted/rejected physical Read/Close/cancel/source-restored-on-Close errors retain original body/status/headers and never resend or become clean missing.
  Tests: `network/v2/extensions/security/addressgroups/python_actions_contracts_test.go:TestPythonAddressActionsCanceledOrChangedOptionsAndMarshalAreTerminal`, `network/v2/extensions/security/addressgroups/python_actions_contracts_test.go:TestPythonAddressActionsPhysicalAcceptedAndRejectedErrorsRetainProof`.
- A retry hook changing submitted JSON is terminal and retains the initial native HTTP rejection; Source successful statuses 200..399, literal one-time ID escaping and clean 404 rejection remain observable.
  Tests: `network/v2/extensions/security/addressgroups/python_actions_contracts_test.go:TestPythonAddressActionsRetryMutationKeepsOriginalRejection`, `network/v2/extensions/security/addressgroups/python_actions_contracts_test.go:TestPythonAddressActionsCleanSourceStatusPolicyAndEscapedID`.

Python/Go differences:

- Go returns an owned declared Resource view with null location and separate actual Wire/Envelope/Input/header/status rather than mutating a Python Resource; complete Connection cloud-location behavior remains separately tracked.
- The explicit string route ID cannot be replaced by a cached seed. Seed aliases that collide are rejected rather than depending on Go map order; ordinary Go map iterable order is its JSON encoding order while RawMessage preserves object insertion order.
- Inputs use the JSON domain; Python arbitrary iterables/custom objects are not implemented as Go builder interfaces. Optional owned headers/cache-seed are Go conveniences and retain shared auth/source protections.
- AddressGroup Source translator discards self and normalizes known aliases/unknown attributes. Bodyless/non-JSON success returns a seed view with actual HTTP metadata and nil Wire.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
trunk = conn.network.add_trunk_subports(
    trunk_id,
    [{"port_id": child_port_id, "segmentation_type": "vlan", "segmentation_id": 100}],
)
trunk = conn.network.delete_trunk_subports(trunk, [{"port_id": child_port_id}])
group = conn.network.add_addresses_to_address_group(group_id, ["192.0.2.0/24"])
group = conn.network.remove_addresses_from_address_group(group, ["192.0.2.0/24"])
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    trunkID := flag.String("trunk", "", "trunk ID")
    addPort := flag.String("add-subport", "", "subport to add")
    removePort := flag.String("remove-subport", "", "subport to remove")
    segmentationID := flag.Int("segmentation-id", 100, "VLAN segmentation ID")
    groupID := flag.String("group", "", "address group ID")
    addAddress := flag.String("add-address", "", "address or CIDR to add")
    removeAddress := flag.String("remove-address", "", "address or CIDR to remove")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    var trunk *trunks.TrunkActionRecord
    if *addPort != "" {
        if *trunkID == "" { log.Fatal("-trunk is required") }
        trunk, err = service.Trunks.AddSubportsRecord(ctx, *trunkID,
            []map[string]any{{"port_id": *addPort, "segmentation_type": "vlan",
                "segmentation_id": *segmentationID}})
        if err != nil { log.Fatal(err) }
        printJSON(trunk.Resource)
    }
    if *removePort != "" {
        if *trunkID == "" { log.Fatal("-trunk is required") }
        var options []trunks.SubportActionOption
        if trunk != nil {
            options = append(options, trunks.WithSubportActionSeed(trunk.Resource))
        }
        trunk, err = service.Trunks.RemoveSubportsRecord(ctx, *trunkID,
            []map[string]any{{"port_id": *removePort}}, options...)
        if err != nil { log.Fatal(err) }
        printJSON(trunk.Resource)
    }
    var group *addressgroups.AddressActionRecord
    if *addAddress != "" {
        if *groupID == "" { log.Fatal("-group is required") }
        group, err = service.SecurityAddressGroups.AddAddressesRecord(ctx,
            *groupID, []string{*addAddress})
        if err != nil { log.Fatal(err) }
        printJSON(group.Resource)
    }
    if *removeAddress != "" {
        if *groupID == "" { log.Fatal("-group is required") }
        var options []addressgroups.AddressActionOption
        if group != nil {
            options = append(options, addressgroups.WithAddressActionSeed(group.Resource))
        }
        group, err = service.SecurityAddressGroups.RemoveAddressesRecord(ctx,
            *groupID, []string{*removeAddress}, options...)
        if err != nil { log.Fatal(err) }
        printJSON(group.Resource)
    }
}
```

## References

- [Install](../../docs/install.md)
- [Neutron 27.0.0 trunk policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/trunk.py)
- [address group policy](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/address_group.py)
- [Base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py)
- [Policy](https://docs.openstack.org/neutron/2026.2/configuration/policy.html)
