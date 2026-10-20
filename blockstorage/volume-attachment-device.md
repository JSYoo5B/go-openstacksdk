# Volume Attachment Device

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_block_storage/BlockStorageCloudMixin/get_volume_attach_device`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.GetVolumeAttachDevice`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.GetVolumeAttachDeviceFields`

- The direct cloud helper maps to two pure package functions: typed GetVolumeAttachDevice accepts the closed four-model constraint covering both SDK-owned models and native Cinder v2/v3 Volume pointers, and GetVolumeAttachDeviceFields consumes supplied raw JSON fields. Neither takes clients/context/Ref/options nor performs lookup, HTTP, refresh or status validation; existing workflow VolumeInfo values require no caller adapter.
  Tests: `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceOwnedModelsUseLiteralCurrentSelectionWithoutCloudState`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceNativeV2V3ModelsExposeDeclaredZeroStringPresenceLoss`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDevicePureOwnershipAndTypedCurrentVersusOriginalRawEvidence`.
- Typed nil pointers for all four accepted concrete types and nil Attachments produce local invalid-input errors; a nonnil empty attachment slice returns nil success. Current supplied owned attachments remain authoritative even when original volume Body lacks attachments or contains stale/null evidence.
  Tests: `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceOwnedModelsUseLiteralCurrentSelectionWithoutCloudState`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceOwnedManualPresenceAndTypedCurrentAuthority`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceNativeV2V3ModelsExposeDeclaredZeroStringPresenceLoss`.
- Owned selection reads attachment order and compares current ServerID literally with case-sensitive equality, including empty strings, spaces and path/query punctuation. Returned device text is unchanged; unmatched rows do not require a device, and unrelated volume ID/status and attachment metadata do not affect selection.
  Tests: `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceOwnedModelsUseLiteralCurrentSelectionWithoutCloudState`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceOwnedManualPresenceAndTypedCurrentAuthority`.
- The first owned match immediately returns a copied nonnil empty/nonempty device or nil for explicit null. Missing canonical device on that first match errors with no later duplicate fallback. Visited nil rows or absent/wrong-case canonical server_id error in order; schema-invalid rows after an accepted empty/null match remain unvisited.
  Tests: `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceOwnedFirstMatchStopsOnEmptyNullOrMissingAndVisitsOnlyPrefix`.
- Manual owned records with Body nil interpret nil ServerID as explicit null/nonmatch and matching nil Device as explicit null success. A nonnil Body distinguishes missing consumed canonical keys; present canonical keys with nil typed pointers represent current null without decoding stale raw values.
  Tests: `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceOwnedFirstMatchStopsOnEmptyNullOrMissingAndVisitsOnlyPrefix`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceOwnedManualPresenceAndTypedCurrentAuthority`.
- Nonnull caller-supplied/current owned ServerID and Device override original absent/stale/wrong-type Body fields. Typed-current selection never silently hydrates original raw evidence; the explicit raw companion reading VolumeInfo.Body observes the original response rather than later edited typed values.
  Tests: `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceOwnedManualPresenceAndTypedCurrentAuthority`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDevicePureOwnershipAndTypedCurrentVersusOriginalRawEvidence`.
- Both native Cinder model versions expose their current ordered string fields directly without an adapter or invented raw proof. Omitted/null/empty native Device becomes a copied nonnil empty string, and zero ServerID can match an empty selector; the native decoder case-folding and presence loss are declared representation differences.
  Tests: `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceNativeV2V3ModelsExposeDeclaredZeroStringPresenceLoss`.
- Raw access reads only the exact top-level attachments key and ignores unrelated supplied field types, including invalid raw JSON in unconsumed fields. Nil/missing/wrong-case top-level input and a present nil RawMessage are local consumed-input errors; unrelated id/status/attachment metadata need no owned-model schema validation.
  Tests: `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceRawIterableBoundaryAndLocalConsumedInputErrors`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceRawFirstMatchPreservesTypingCastValuesAndNumericPrecision`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceWholeModelDecoderAndDirectRawHaveExplicitDifferentBoundaries`.
- Raw attachments valid empty array, object or string return nil success as ordinary plain JSON mapping iterables. Null/bool/number and nonempty object/string fail locally; arrays preserve ordered prefix consumption and reject visited null/nonobject rows or missing exact server_id/device only when those fields are consumed.
  Tests: `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceRawIterableBoundaryAndLocalConsumedInputErrors`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceRawFirstMatchPreservesTypingCastValuesAndNumericPrecision`.
- Raw attachments syntax and UTF-8 are validated across the entire supplied JSON before selection. Empty/whitespace-only bytes, trailing extra JSON, malformed tail syntax and invalid UTF-8 including an unvisited tail return local errors; JSON SyntaxError remains available in the error chain.
  Tests: `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceRawIterableBoundaryAndLocalConsumedInputErrors`.
- Raw row server_id must have the exact canonical key. Null and every nonstring JSON value are nonmatches against the supplied Go string, without text coercion or device inspection. JSON strings compare literally and case-sensitively, including punctuation and empty selectors.
  Tests: `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceRawIterableBoundaryAndLocalConsumedInputErrors`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceRawFirstMatchPreservesTypingCastValuesAndNumericPrecision`.
- The first raw matching device is returned unchanged as a copied nonnull RawMessage for strings, empty strings, escaped strings, bool, zero, arbitrary-precision integer/decimal, arrays and objects. Null match and no match both return nil success. Missing first matched device cannot fall back, and a schema-invalid tail after a match is not visited.
  Tests: `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceRawIterableBoundaryAndLocalConsumedInputErrors`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceRawFirstMatchPreservesTypingCastValuesAndNumericPrecision`.
- All consumed-input failures use resource.OperationError with the actual public operation name, Resource volume, ErrInvalidOption identity and visited key/index where applicable. Parsing causes are preserved, and local failures never manufacture ResponseError or NotFoundError.
  Tests: `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceOwnedModelsUseLiteralCurrentSelectionWithoutCloudState`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceOwnedFirstMatchStopsOnEmptyNullOrMissingAndVisitsOnlyPrefix`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceOwnedManualPresenceAndTypedCurrentAuthority`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceNativeV2V3ModelsExposeDeclaredZeroStringPresenceLoss`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceRawIterableBoundaryAndLocalConsumedInputErrors`.
- Accessors do not mutate supplied model fields, attachments, Body, headers or status evidence. Returned owned/native device pointers and raw JSON bytes are independent copies; caller mutation of results does not change a repeated selection. Concurrent mutation of the supplied mutable input is outside the contract.
  Tests: `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceNativeV2V3ModelsExposeDeclaredZeroStringPresenceLoss`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDevicePureOwnershipAndTypedCurrentVersusOriginalRawEvidence`.
- Raw prefix consumption is independent from existing strict whole-model decoding: ignored wrong-type known fields and schema-invalid tails may be accepted by the direct raw helper after the first match, while the unchanged owned observation/VolumeInfo decoders reject full malformed models. JSON syntax/UTF-8 validity remains a whole-input boundary.
  Tests: `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceRawIterableBoundaryAndLocalConsumedInputErrors`, `blockstorage/volume_attachment_device_test.go:TestVolumeAttachDeviceWholeModelDecoderAndDirectRawHaveExplicitDifferentBoundaries`.

Python/Go differences:

- This direct declaration is pinned to openstacksdk ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe; native accepted models are pinned to Gophercloud v2.15.0. Only get_volume_attach_device is mapped here. The helper uses no self/session/cache calls; separately inventoried Resource descriptors, attachment workflows, native operations and other SDK declarations retain independent review statuses.
- Python volume/server_id duck-typed values are represented by a library-owned closed Go generic constraint for four existing model pointers plus an explicit ordinary-JSON raw-fields companion. Go serverID is a string and equality is literal; custom mapping/equality/iterables, sets/tuples and user-defined object behavior are outside this Go representation, rather than requirements for caller interfaces or adapters.
- Python typing.cast returns the matched device unchanged, including nonstrings, with Python object identity. The Go typed function returns copied string/null values already representable in the selected model; the raw function returns copied arbitrary nonnull JSON values, preserving bytes and numeric precision without coercion. Neither returns aliasing Python object identity.
- Go owned mutable current fields are authoritative; passive original Body is consulted only for canonical-key presence when a typed pointer is nil. Manual Body nil treats nullable pointers as explicit null, while nonnil Body can retain omission. Calling the raw companion with volume.Body explicitly chooses original evidence and never silently overrides edited typed values.
- Native Gophercloud v2/v3 Volume attachment ServerID/Device strings collapse omitted/null/empty evidence, and native decoding can fold key case or eagerly reject other known fields/timestamps. The typed accessor uses those current strings literally, including zero ServerID matching an empty selector and nonnil empty Device, without reconstructing lost null/presence proof. Raw fields are the explicit presence-sensitive alternative.
- Go raw input validates full attachments JSON syntax and UTF-8 before ordered field consumption; a schema-invalid unvisited tail can be ignored, but syntax/encoding errors anywhere fail. Python operates on already supplied objects and has no corresponding JSON parser boundary. Go decoding/string equality follows Go JSON/UTF-8 representation; arbitrary Python Unicode/custom object semantics are not silently reproduced.
- Python plain dict empty attachments []/{} or empty string iterates empty, which the raw Go function represents. Inherited Python v2 attachments Body is untyped, whereas v3 type=list can wrap a nonnull object/string into a singleton list. Descriptor conversion is independent Resource behavior rather than an additional HTTP/control flow in this direct helper. Go typed nil attachments is invalid evidence and nonnil empty slices represent an empty list.
- Python missing consumed keys/noniterable values raise KeyError/TypeError and runtime cast performs no validation. Go uses local OperationError wrapping ErrInvalidOption with function/resource/key/index context and JSON causes; no HTTP/NotFound evidence is invented. The nil successful result deliberately joins first matched null with no match, and empty string remains a nonnil first match.
- Existing owned whole-model strict decoding remains independent from this direct accessor. It validates all rows and known field types before selection, while raw direct access examines only fields consumed by the pinned helper after a full JSON syntax/UTF-8 check. This mapping does not promote whole-Resource/session/cache or whole-SDK completeness.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "encoding/json"
    "errors"
    "fmt"

    volumesv2 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v2/volumes"
    volumesv3 "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"

    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func text(value string) *string { return &value }

func device(value *string, err error) string {
    if err != nil { panic(err) }
    if value == nil { return "nil" }
    return fmt.Sprintf("%q", *value)
}

func rawDevice(value json.RawMessage, err error) string {
    if err != nil { panic(err) }
    if value == nil { return "nil" }
    return string(value)
}

func invalid(label string, err error) {
    var operation *resource.OperationError
    if !errors.As(err, &operation) || !errors.Is(err, resource.ErrInvalidOption) {
        panic(fmt.Sprintf("unexpected error: %v", err))
    }
    fmt.Printf("%s %s %s %t\n", label, operation.Operation,
        operation.Resource, errors.Is(err, resource.ErrInvalidOption))
}

func main() {
    owned := &blockstorage.VolumeInfo{Attachments: []*blockstorage.AttachVolumeRecord{
        {ServerID: text("server"), Device: text("")},
        {ServerID: text("server"), Device: text("/dev/vdb")},
    }}
    fmt.Println("owned first", device(blockstorage.GetVolumeAttachDevice(owned, "server")))
    fmt.Println("owned no match", device(blockstorage.GetVolumeAttachDevice(owned, "SERVER")))
    empty := &blockstorage.VolumeInfo{Attachments: []*blockstorage.AttachVolumeRecord{}}
    fmt.Println("owned empty", device(blockstorage.GetVolumeAttachDevice(empty, "server")))
    null := &blockstorage.VolumeInfo{Attachments: []*blockstorage.AttachVolumeRecord{
        {ServerID: text("server"), Device: nil}, // Manual Body:nil means explicit null.
    }}
    fmt.Println("owned null", device(blockstorage.GetVolumeAttachDevice(null, "server")))

    var current blockstorage.AttachVolumeObservation
    if err := json.Unmarshal([]byte(`{"attachments":[{"server_id":"server","device":"/dev/original"}]}`), &current); err != nil {
        panic(err)
    }
    current.Attachments[0].Device = text("/dev/current")
    fmt.Println("typed current", device(blockstorage.GetVolumeAttachDevice(&current, "server")))
    fmt.Println("raw original", rawDevice(blockstorage.GetVolumeAttachDeviceFields(current.Body, "server")))

    nativeV2 := &volumesv2.Volume{Attachments: []volumesv2.Attachment{
        {ServerID: "server", Device: ""},
    }}
    nativeV3 := &volumesv3.Volume{Attachments: []volumesv3.Attachment{
        {ServerID: "server", Device: "/dev/vdc"},
    }}
    fmt.Println("native v2", device(blockstorage.GetVolumeAttachDevice(nativeV2, "server")))
    fmt.Println("native v3", device(blockstorage.GetVolumeAttachDevice(nativeV3, "server")))

    raw := map[string]json.RawMessage{
        "attachments": json.RawMessage(`[{"server_id":"server","device":false}]`),
        "status": json.RawMessage("unrelated invalid JSON"), // This field is not consumed.
    }
    fmt.Println("raw false", rawDevice(blockstorage.GetVolumeAttachDeviceFields(raw, "server")))
    raw["attachments"] = json.RawMessage(`[{"server_id":"server","device":9007199254740993123456789}]`)
    fmt.Println("raw number", rawDevice(blockstorage.GetVolumeAttachDeviceFields(raw, "server")))
    raw["attachments"] = json.RawMessage(`[{"server_id":"server","device":{"path":"/dev/vdb"}}]`)
    fmt.Println("raw object", rawDevice(blockstorage.GetVolumeAttachDeviceFields(raw, "server")))
    raw["attachments"] = json.RawMessage(`[{"server_id":"server","device":null}]`)
    fmt.Println("raw null", rawDevice(blockstorage.GetVolumeAttachDeviceFields(raw, "server")))

    tail := []byte(`{"attachments":[{"server_id":"server","device":"/dev/vdb"},null]}`)
    var fields map[string]json.RawMessage
    if err := json.Unmarshal(tail, &fields); err != nil { panic(err) }
    fmt.Println("raw short circuit", rawDevice(blockstorage.GetVolumeAttachDeviceFields(fields, "server")))
    var whole blockstorage.VolumeInfo
    fmt.Println("whole decoder rejects tail", json.Unmarshal(tail, &whole) != nil)

    _, err := blockstorage.GetVolumeAttachDevice((*blockstorage.VolumeInfo)(nil), "server")
    invalid("typed nil error", err)
    fields["attachments"] = json.RawMessage(`[{"server_id":"server"}]`)
    _, err = blockstorage.GetVolumeAttachDeviceFields(fields, "server")
    invalid("raw missing error", err)
}
```

```python
volume = {"attachments": [
    {"server_id": "server", "device": ""},
    {"server_id": "server", "device": "/dev/vdb"},
]}
device = conn.get_volume_attach_device(volume, "server")  # "": first match
```

## References

- [Reference](README.md)
- [..](../README.md)
- [cloud helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L381-L396)
- [v2 Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/volume.py#L44)
- [v3 Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L54)
- [field conversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86-L111)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v2/volumes/results.go#L11-L31)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/results.go#L12-L33)
