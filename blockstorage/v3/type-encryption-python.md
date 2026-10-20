# Type Encryption Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/find_type`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithRecordFindIgnoreMissing`

- GET-first type lookup uses lowercase is_public=none, returns the Source-shaped declared view, and does not list on successful GET.
  Tests: `blockstorage/v3/volumetypes/python_find_records_test.go:TestPythonTypeFindRecordGetFirstUsesLowercaseNone`.
- Only clean 400/403/404 trigger list fallback without a name hint; exact ID or name is matched over the complete list while unknown fields stay in Wire.
  Tests: `blockstorage/v3/volumetypes/python_find_records_test.go:TestPythonTypeFindRecordFallsBackWithoutNameFilterAndMatchesID`.
- Continuation duplicates and late HTTP failures remain errors; default missing returns nil and strict missing reports NotFound after successful empty list evidence.
  Tests: `blockstorage/v3/volumetypes/python_find_records_test.go:TestPythonTypeFindRecordScansContinuationAndRejectsLateErrorOrDuplicate`, `blockstorage/v3/volumetypes/python_find_records_test.go:TestPythonTypeFindRecordMissingPoliciesAndPreflightGuard`.
- Physical rejected body/Close failures do not turn into fallback; source-changing callbacks fail before another request.
  Tests: `blockstorage/v3/volumetypes/python_find_records_test.go:TestPythonTypeFindRecordPhysicalRejectedBodyDoesNotFallback`, `blockstorage/v3/volumetypes/python_find_records_test.go:TestPythonTypeFindRecordMissingPoliciesAndPreflightGuard`.
- Explicit Go extra kwargs snapshot request values and split local nested Source bool/number filters from server query.
  Tests: `blockstorage/v3/volumetypes/python_find_records_test.go:TestPythonTypeFindRecordQueryExtensionLocalFilterAndSnapshot`.

Python/Go differences:

- Python Resource/Munch mutable identity/cache/session/subclass hooks and connection-based location remain shared SDK scope; these owned APIs snapshot input and return a Source-shaped record with raw Wire and actual HTTP evidence.
- Go uses one selected Connection microversion and UTF-8 escaped string route identities; it does not auto-negotiate per-resource microversions or emulate arbitrary Python objects.
- Go option callbacks and headers are concrete owned extensions; malformed known JSON, nil callbacks and source changes fail before HTTP.
- Type proxy exposes only ignore_missing in Source; Go additionally exposes owned query/header extensions while always forcing is_public=none.
- Go requires the result identity to be a nonempty string, supports dictionary links as a compatibility extension, and fixes continuation origin/path/filter identity.

### `python:block_storage/v3/create_type_encryption`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.CreateEncryptionWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithEncryptionAttribute`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithEncryptionAttributeOpts`

- Create emits only supplied declared Body fields, distinguishing omitted from explicit null/zero/false/empty and sending an empty encryption envelope when no attrs are given. Untyped encryption values and unknown raw response fields retain exact JSON types and large numbers.
  Tests: `blockstorage/v3/volumetypes/python_encryption_contracts_test.go:TestPythonEncryptionAttributesPreservePresenceAndResponseTypes`.
- Options own map/raw bytes/header snapshots, unknown attrs are discarded, and bodyless accepted responses preserve the supplied seed.
  Tests: `blockstorage/v3/volumetypes/python_encryption_contracts_test.go:TestPythonEncryptionOptionsSnapshotInputsAndIgnoreUnknown`.
- Invalid options or a changed source are stopped before HTTP; accepted Read and Close failure causes retain actual body/header/status without resend.
  Tests: `blockstorage/v3/volumetypes/python_encryption_contracts_test.go:TestPythonEncryptionPreflightGuardsOptionsAndMalformedJSON`, `blockstorage/v3/volumetypes/python_encryption_contracts_test.go:TestPythonEncryptionAcceptedPhysicalFailuresKeepEvidence`.

Python/Go differences:

- Python Resource/Munch mutable identity/cache/session/subclass hooks and connection-based location remain shared SDK scope; these owned APIs snapshot input and return a Source-shaped record with raw Wire and actual HTTP evidence.
- Go uses one selected Connection microversion and UTF-8 escaped string route identities; it does not auto-negotiate per-resource microversions or emulate arbitrary Python objects.
- Go option callbacks and headers are concrete owned extensions; malformed known JSON, nil callbacks and source changes fail before HTTP.

### `python:block_storage/v3/update_type_encryption`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.UpdateEncryptionFromType`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.UpdateEncryptionRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.GetEncryptionRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithEncryptionAttribute`

- The volume_type form GETs flat encryption and PUTs only dirty fields to its alternate encryption ID, preserving explicit null/zero and omitted seed values.
  Tests: `blockstorage/v3/volumetypes/python_encryption_contracts_test.go:TestPythonEncryptionUpdateFetchesAndSendsOnlyDirtyFields`.
- Empty, Source-equal and id-only changes fetch but skip PUT; Python recursive bool/number equality leaves unchanged values clean.
  Tests: `blockstorage/v3/volumetypes/python_encryption_contracts_test.go:TestPythonEncryptionUpdateEqualEmptyAndIDOnlySkipPut`.
- The record form skips GET, owns a snapshot and does not mutate the caller; bodyless accepted PUT retains the updated seed.
  Tests: `blockstorage/v3/volumetypes/python_encryption_contracts_test.go:TestPythonEncryptionRecordUpdateSnapshotsCallerAndBody`.
- Options/source preflight and actual accepted response physical failures keep original causes and evidence.
  Tests: `blockstorage/v3/volumetypes/python_encryption_contracts_test.go:TestPythonEncryptionPreflightGuardsOptionsAndMalformedJSON`, `blockstorage/v3/volumetypes/python_encryption_contracts_test.go:TestPythonEncryptionAcceptedPhysicalFailuresKeepEvidence`.

Python/Go differences:

- Python Resource/Munch mutable identity/cache/session/subclass hooks and connection-based location remain shared SDK scope; these owned APIs snapshot input and return a Source-shaped record with raw Wire and actual HTTP evidence.
- Go uses one selected Connection microversion and UTF-8 escaped string route identities; it does not auto-negotiate per-resource microversions or emulate arbitrary Python objects.
- Go option callbacks and headers are concrete owned extensions; malformed known JSON, nil callbacks and source changes fail before HTTP.
- Go record updates use explicit attributes and copy the input record; mutating Resource then implicit commit is the separately tracked shared mutable Resource surface. Parent-form options are prepared before GET as a Go preflight improvement.

### `python:block_storage/v3/find_qos_spec`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithRecordFindOpts`

- A successful member GET stops lookup; Source string ID conversion preserves large-number Wire evidence, untyped fields and allowed unknown Body attributes.
  Tests: `blockstorage/v3/qos/python_find_records_test.go:TestPythonQoSFindRecordGetFirstConvertsIDAndKeepsUnknownFields`.
- GET forwards Requests-encoded kwargs while list splits declared server query from known Body nested filters, ignoring unknown list kwargs and preserving recursive Python bool/number equality.
  Tests: `blockstorage/v3/qos/python_find_records_test.go:TestPythonQoSFindRecordSplitsGETKwargsAndListServerBodyFilters`.
- Clean 400/403/404 fallback finds exact IDs or names over all pages; default and strict missing, other member statuses and late duplicate failures follow shared FindIdentity.
  Tests: `blockstorage/v3/qos/python_find_records_test.go:TestPythonQoSFindRecordDefaultsMissingAndFallbackCodes`, `blockstorage/v3/qos/python_find_records_test.go:TestPythonQoSFindRecordFindsIDAndRejectsLateDuplicate`.
- Owned kwargs survive caller mutation; max_items caps raw rows before local filters and supplies a list limit hint while retaining GET kwargs.
  Tests: `blockstorage/v3/qos/python_find_records_test.go:TestPythonQoSFindRecordOwnsOptionsAndCapsBeforeBodyFilter`.
- Malformed options/source/cancellation stop before HTTP; rejected GET and accepted list Read/Close causes retain actual evidence without a false successful lookup.
  Tests: `blockstorage/v3/qos/python_find_records_test.go:TestPythonQoSFindRecordGuardAndQueryFailuresBeforeHTTP`, `blockstorage/v3/qos/python_find_records_test.go:TestPythonQoSFindRecordPhysicalRejectedBodyAndLateAcceptedCloseKeepEvidence`.

Python/Go differences:

- Python Resource/Munch mutable identity/cache/session/subclass hooks and connection-based location remain shared SDK scope; these owned APIs snapshot input and return a Source-shaped record with raw Wire and actual HTTP evidence.
- Go uses one selected Connection microversion and UTF-8 escaped string route identities; it does not auto-negotiate per-resource microversions or emulate arbitrary Python objects.
- Go option callbacks and headers are concrete owned extensions; malformed known JSON, nil callbacks and source changes fail before HTTP.
- Go max_items accepts nonnegative int and paginated bool; arbitrary Python control values are not emulated. Go id kwargs are rejected before HTTP because Source would collide with the find identity constructor.
- Go requires nonempty result string IDs, supports dictionary links as a compatibility extension, and fixes continuation origin/path/filter identity.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
volume_type = conn.block_storage.find_type("fast")
qos = conn.block_storage.find_qos_spec("gold", project_id=project_id)

created = conn.block_storage.create_type_encryption(
    volume_type, provider="luks", control_location="front-end")
updated = conn.block_storage.update_type_encryption(
    volume_type=volume_type, key_size=512, cipher=None)
updated = conn.block_storage.update_type_encryption(
    encryption=updated, key_size=0)
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "strconv"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printResource(v *resource.RawResource) {
    if v == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(v, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    typeName := flag.String("type", "fast", "volume type ID or exact name")
    qosName := flag.String("qos", "", "optional QoS spec ID or exact name; stock policy needs admin")
    project := flag.String("project", "", "optional QoS project filter")
    create := flag.Bool("create-encryption", false, "create encryption on the found type")
    keySize := flag.String("key-size", "", "update encryption key size; 0 is explicit")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    volumeType, err := service.VolumeTypes.FindRecord(ctx, *typeName,
        volumetypes.WithRecordFindIgnoreMissing(false))
    if err != nil { log.Fatal(err) }
    printResource(volumeType.Resource)
    var typeID string
    if err := json.Unmarshal(volumeType.Resource.Body["id"], &typeID); err != nil { log.Fatal(err) }
    if *qosName != "" {
        options := []qos.RecordFindOption{}
        if *project != "" { options = append(options, qos.WithRecordFindQuery("project_id", *project)) }
        specification, err := service.QoS.FindRecord(ctx, *qosName, options...)
        if err != nil { log.Fatal(err) }
        if specification == nil { fmt.Println("QoS: null") } else { printResource(specification.Resource) }
    }
    if *create {
        encryption, err := service.VolumeTypes.CreateEncryptionWithAttributes(ctx, typeID,
            volumetypes.WithEncryptionAttribute("provider", "luks"),
            volumetypes.WithEncryptionAttribute("control_location", "front-end"))
        if err != nil { log.Fatal(err) }
        printResource(encryption.Resource)
    }
    if *keySize != "" {
        size, err := strconv.Atoi(*keySize)
        if err != nil { log.Fatal(err) }
        encryption, err := service.VolumeTypes.UpdateEncryptionFromType(ctx, typeID,
            volumetypes.WithEncryptionAttribute("key_size", size))
        if err != nil { log.Fatal(err) }
        printResource(encryption.Resource)
    }
}
```

## References

- [Policy](https://docs.openstack.org/cinder/2025.2/configuration/block-storage/policy.html)
- [Install](../../docs/install.md)
- [Python Encryption Contracts Test](volumetypes/python_encryption_contracts_test.go)
- [Python Find Records Test](volumetypes/python_find_records_test.go)
- [Python Find Records Test](qos/python_find_records_test.go)
