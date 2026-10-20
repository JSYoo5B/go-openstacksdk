# Attachments Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/create_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithRecordCreateField`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithRecordCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithRecordCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithRecordCreateModeSupported`

- CreateRecord posts the Source attachment envelope to summary /attachments, with mandatory volume_id renamed to volume_uuid. Omitted instance creates a reservation without instance_uuid; truthy instance renames to instance_uuid while explicit JSON falsey instances retain the original instance key. Unknown Source kwargs are ignored and known fields preserve omission/null/false/zero/empty/large JSON values.
  Tests: `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordCreateReservationAndOwnedPartialEvidence`, `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordCreateInstanceTruthinessAndModeVersions`.
- Declared wire/attribute aliases map in option order. Input volume_id is rejected as a duplicate positional value; unordered raw maps reject conflicting field aliases. Source response aliases merge in JSON member order; Source connection_info dict descriptor shows nonnull nonobject as{} while Wire preserves the actual value.
  Tests: `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordCreateAliasOrderAndPreparation`, `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordCreateReservationAndOwnedPartialEvidence`.
- Falsey mode automatically uses shared versionless finite Cinder discovery and Source Supports(maximum,minimum,selected,3.54). Both advertised bounds must include3.54 and any configured default must finite-major-match it. Selected3.80 with minimum3.60, missing bounds or maximum below3.54 preserves mode; global latest preserves it while3.latest can match. An absent default selects at most Source ceiling3.71 without shared client mutation, while an explicit3.80 is retained.
  Tests: `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordCreateInstanceTruthinessAndModeVersions`, `blockstorage/v3/attachments/python_mode_discovery_test.go:TestPythonAttachmentRecordModeRequiresAdvertisedRangeAndSelectedDefault`, `blockstorage/v3/attachments/python_mode_discovery_test.go:TestPythonAttachmentRecordModeFiniteDiscoveryFallbackKeepsUnknownSupport`.
- Raw owned AttachmentRecord distinguishes the ten declared Body properties plus nullable location view from original Wire and whole HTTP Envelope. Accepted partial response overlays supplied fields, blank/malformed JSON keeps seeds as Source does, and valid nonobject/null/wrongUTF8 errors retain actual ACK. No native model coercion or native required-field validation is imposed.
  Tests: `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordCreateReservationAndOwnedPartialEvidence`, `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordCreateResponseTranslationAndStatusPolicy`.
- SDK-owned concrete options snapshot raw fields, headers and nullable controls. Source capture guards provider/type/route/version across option callbacks, retries and physical response Read/Close; custom cancellation and callback/native causes remain attributable without hidden retries or client mutation.
  Tests: `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordSourceChangesCancellationAndRetryOwnership`, `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordConcurrentOptionReuse`.
- Original success policy accepts200..399. Accepted partial receipts preserve actual Envelope/Header/StatusCode and response errors without inventing decoded fields. The shared rejection observer preserves native400..599 status/body/header plus physical Read/Close causes.
  Tests: `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordPhysicalErrorsPreserveStatusBodyAndCauses`.
- Capability discovery observations remain separate from POST ACK. Physical discovery Read/Close/decode/invalid-bound failures and sticky source/cancellation failures stop before POST and return owned Discovery-only proof with attachment StatusCode0 and no fabricated Envelope/Resource/Wire. A changed source restored at Close remains invalid. Negotiated operation version headers or their absence stay fixed across native retry/physical attempts; later POST rejection/accepted-body faults preserve earlier discovery plus actual native/ACK evidence.
  Tests: `blockstorage/v3/attachments/python_mode_discovery_test.go:TestPythonAttachmentRecordModeDiscoveryFailureRetainsOnlyDiscoveryProof`, `blockstorage/v3/attachments/python_mode_discovery_test.go:TestPythonAttachmentRecordModeDiscoverySourceAndCancellationStopBeforePost`, `blockstorage/v3/attachments/python_mode_discovery_test.go:TestPythonAttachmentRecordModeProtectsNegotiatedHeadersAndPostAcknowledgement`.

Python/Go differences:

- Go accepts an explicit validated nonblank UTF-8 volume ID and immutable AttachmentRecord; Python accepts mutable Resource/Munch values and mutates its created Resource. Response Resource/Wire/Envelope are separate evidence channels and location is null without a Python Connection location synthesis.
- Ordered field options resolve aliases by last assignment; an unordered raw Fields map with both aliases is rejected. The positional volume ID cannot be overridden, matching the Source duplicate volume_id call conflict.
- Malformed UTF-8 response JSON fails instead of requests decoding replacement characters. Shared Source/client/REST guards prohibit source or fixed-request ownership changes during callbacks/retries.
- Mutable Resource/session/cache/subclass/base_path behavior remains in global SDK-R1/C1/S1 review scope; this named proxy mapping does not promote those separately inventoried declarations.
- For a supplied falsey mode Go automatically reads the guarded Cinder version endpoint/root and applies separately advertised bounds plus selected-default support. An absent default is selected at Source3.71 ceiling; explicit selected versions are retained. Explicit ModeSupported is an optional caller capability override that bypasses this discovery and retains configured request policy. Source generally uses session discovery caching; owned Go observations are per operation and do not mutate its shared client. Other Resource/session negotiation remains SDK-S1 scope.

### `python:block_storage/v3/attachments`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithRecordListMicroversion`

- ListRecords lazily GETs summary /attachments rather than native /attachments/detail, follows attachments_links and preserves actual per-page wire/declared fields and JSON alias ordering. id/status/project_id/volume_id/instance_id are server queries, all_projects maps all_tenants, server alias is accepted and client alias takes precedence even when null. Requests encoding preserves Python boolean spelling and iterable expansion; declared nonserver attribute names filter locally and unknown/wire-only Body aliases are ignored.
  Tests: `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordListSummaryQueriesAndLinkedPages`, `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordListServerAliasPrecedence`, `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordListLocalFilteringAndRawRowCap`.
- Raw-row MaxItems is applied before local filtering and becomes an initial wire limit hint when limit is missing/zero; cap/break/SinglePage stops before next-link interpretation. Source singleton plural objects and marker fallback on nonempty short pages are supported, zero limit does not enable fallback and empty page stops.
  Tests: `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordListLocalFilteringAndRawRowCap`, `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordListMarkerFallbackAndSingleton`.
- Continuation stays on fixed origin/collection with unchanged filters; repeated markers, added/changed filters and invalid plural envelopes retain actual HTTP evidence. Null rows and absent plural keys fail without pretending list success.
  Tests: `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordListContinuationAndEnvelopeFailures`.
- Per-list3.80 version/header policy applies to a copied Source without original client mutation, Accept defaults application/json and forbidden authentication headers/nil option/negative controls/multi-marker/invalid known JSON fail before HTTP.
  Tests: `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordListVersionHeaderOwnershipAndPreflight`, `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordListSummaryQueriesAndLinkedPages`.
- SDK-owned concrete options snapshot raw fields, headers and nullable controls. Source capture guards provider/type/route/version across option callbacks, retries and physical response Read/Close; custom cancellation and callback/native causes remain attributable without hidden retries or client mutation.
  Tests: `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordSourceChangesCancellationAndRetryOwnership`, `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordConcurrentOptionReuse`.
- Original success policy accepts200..399. Accepted partial receipts preserve actual Envelope/Header/StatusCode and response errors without inventing decoded fields. The shared rejection observer preserves native400..599 status/body/header plus physical Read/Close causes.
  Tests: `blockstorage/v3/attachments/python_records_contracts_test.go:TestPythonAttachmentRecordPhysicalErrorsPreserveStatusBodyAndCauses`.

Python/Go differences:

- Owned immutable AttachmentRecord replaces mutable Python Attachment objects and exposes nullable declared fields separately from lossless Wire/page Envelope proof. Source dict descriptor adaptation applies only to Resource connection_info.
- Go list controls reject negative limits/caps and require a single numeric limit/nonblank marker. The local JSON filter distinguishes booleans from numbers and treats invalid nested object shapes as nonmatches instead of Python attribute errors.
- Fixed REST continuation guards reject cross-origin/path changes, arbitrary query/filter additions and cursor loops. Dictionary links are an explicit Go compatibility extension beyond pinned Python paginator support.
- Configured/per-list microversion is copied directly rather than session negotiated; there is no artificial3.71 ceiling. Mutable Resource/session/cache/subclass/base_path behavior remains in global SDK-R1/C1/S1 scope.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
reserved = conn.block_storage.create_attachment(volume_id)
attached = conn.block_storage.create_attachment(
    volume_id, instance=server_id, connector={"host": "compute-1"}, mode="rw")
for attachment in conn.block_storage.attachments(volume_id=volume_id):
    print(attachment.id, attachment.attach_mode)
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
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    volume := flag.String("volume", "", "volume ID")
    instance := flag.String("instance", "", "optional attaching server ID")
    version := flag.String("microversion", "3.71", "Cinder microversion supported by the deployment")
    reserve := flag.Bool("reserve", false, "create a volume attachment reservation")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud), sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Attachments
    for value, err := range api.ListRecords(ctx,
        attachments.WithRecordListQuery("volume_id", *volume),
        attachments.WithRecordListMaxItems(100)) {
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(value.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    }
    if *reserve {
        options := []attachments.RecordCreateOption{}
        if *instance != "" {
            options = append(options, attachments.WithRecordCreateField("instance", *instance))
        }
        value, err := api.CreateRecord(ctx, *volume, options...)
        if err != nil { log.Fatal(err) }
        data, err := json.MarshalIndent(value.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    }
}
```

## References

- [Attachment proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1495-L1551)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/attachment.py)
- [Install](../../docs/install.md)
- [Python Records Contracts Test](attachments/python_records_contracts_test.go)
