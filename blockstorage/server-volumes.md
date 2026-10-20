# Server Volumes

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_block_storage/BlockStorageCloudMixin/get_volumes`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetVolumes`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.GetVolumes`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareGetVolumesOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithGetVolumesOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithGetVolumesServerFields`
- `github.com/JSYoo5B/go-openstacksdk/resource.RawResource.Decode`
- `github.com/JSYoo5B/go-openstacksdk/resource.RawResource.Clone`

- GetVolumes maps the supplied server ID through GetVolumesRequest.ServerID as a literal local JSON string, including empty/space/punctuation values. The package and Connection workflows return RawResource matches plus independent admitted Pages evidence; no UUID/Ref/name validation, Nova selection, server lookup, member GET, refresh, wait or SDK deadline is introduced.
  Tests: `blockstorage/get_volumes_test.go:TestGetVolumesCompletePagingPreservesAttachmentDuplicatesAndDistinctRawRows`, `blockstorage/get_volumes_test.go:TestGetVolumesV3ListifiedAttachmentsAndFullJSONServerEqualityIgnoreDeviceSchema`, `connection_get_volumes_test.go:TestConnectionGetVolumesUsesOnlyCachedCinderAndOriginalOptionsOnce`.
- GetVolumesOpts.ServerFields nil pointer uses the request string, while a nonnil pointer selects an owned raw-fields map, including nil map for explicit missing evidence. WithGetVolumesServerFields and whole-replacement WithGetVolumesOptions snapshot pointers/maps/raw bytes and original callbacks apply once in order; no id/key/JSON validation occurs before source consumption.
  Tests: `blockstorage/get_volumes_test.go:TestGetVolumesOptionsSnapshotsReplacementOrderedCallbacksAndDelayedFields`, `blockstorage/get_volumes_test.go:TestGetVolumesConsumesAssociationAndServerEvidenceOnlyAfterCompleteList`, `blockstorage/get_volumes_test.go:TestGetVolumesV3ListifiedAttachmentsAndFullJSONServerEqualityIgnoreDeviceSchema`.
- PrepareGetVolumesOptions applies original options locally once with context guards and snapshots all retained maps/raw bytes without HTTP. Direct package captures required Cinder source before callbacks and guards context/source during preparation and execution. Provided server id is validated lazily only after list materialization and a visited attachment server_id.
  Tests: `blockstorage/get_volumes_test.go:TestGetVolumesOptionsSnapshotsReplacementOrderedCallbacksAndDelayedFields`, `blockstorage/get_volumes_test.go:TestGetVolumesPreflightAndCapturedSourceFactsBeforeOptionsOrSubsequentPages`, `blockstorage/get_volumes_test.go:TestGetVolumesConsumesAssociationAndServerEvidenceOnlyAfterCompleteList`.
- Connection validates context/receiver, prepares original callbacks once before selecting cached Cinder v3 and delegates complete owned options without rerunning originals. No Nova service is initialized; copied execution clients preserve current provider authentication and configured selected microversion.
  Tests: `connection_get_volumes_test.go:TestConnectionGetVolumesUsesOnlyCachedCinderAndOriginalOptionsOnce`, `connection_get_volumes_test.go:TestConnectionGetVolumesPreparesOptionsBeforeGetterAndDelaysUnusedServerFields`, `connection_get_volumes_test.go:TestConnectionGetVolumesInvalidEntryNilOptionsAndOptionCancellationSelectNoService`, `connection_get_volumes_test.go:TestConnectionGetVolumesGetterCancellationKeepsOriginalCause`.
- Captured source facts Provider/Endpoint/ResourceBase/Type/Microversion are sticky throughout callbacks, HTTP and selection. Ordinary headers are captured independently and current provider tokens remain live; empty Type is normalized only on the private copied Cinder source. Source drift or caller cancellation is terminal and retains native causes without borrowing stale response evidence.
  Tests: `blockstorage/get_volumes_test.go:TestGetVolumesNativeAuthBackoffRetryKeepsBodylessExactListAndLiveHeaders`, `blockstorage/get_volumes_test.go:TestGetVolumesPreflightAndCapturedSourceFactsBeforeOptionsOrSubsequentPages`, `blockstorage/get_volumes_test.go:TestGetVolumesAcceptedPageFailuresKeepCurrentProofAndLaterRejectionsNeverBorrowOldPage`, `connection_get_volumes_test.go:TestConnectionGetVolumesUsesOnlyCachedCinderAndOriginalOptionsOnce`.
- The initial fixed request is bodyless GET200 /volumes/detail with no filters/all_projects/name/status/server query and no cap/first-page shortcut. Every page up to actual EOF and every raw volume object is completely materialized before any attachments or supplied server fields are consumed; later page failures take precedence over earlier association/server errors.
  Tests: `blockstorage/get_volumes_test.go:TestGetVolumesCompletePagingPreservesAttachmentDuplicatesAndDistinctRawRows`, `blockstorage/get_volumes_test.go:TestGetVolumesConsumesAssociationAndServerEvidenceOnlyAfterCompleteList`, `blockstorage/get_volumes_test.go:TestGetVolumesContinuationScopeCyclesAndCanonicalEnvelopeUseActualPageProof`.
- Accepted list pages require complete valid UTF-8 JSON with canonical nonnull volumes array and object rows, without typed known-field decoding. Each returned raw volume owns its Body/header/status evidence; wrong-type id/status/device/timestamp or extensions are preserved rather than eagerly schema-validated.
  Tests: `blockstorage/get_volumes_test.go:TestGetVolumesCompletePagingPreservesAttachmentDuplicatesAndDistinctRawRows`, `blockstorage/get_volumes_test.go:TestGetVolumesContinuationScopeCyclesAndCanonicalEnvelopeUseActualPageProof`, `blockstorage/get_volumes_test.go:TestGetVolumesAcceptedPageFailuresKeepCurrentProofAndLaterRejectionsNeverBorrowOldPage`, `resource/raw_resource_test.go:TestRawResourceAcceptsFlexibleCanonicalObjectsAndFailedDecodeIsAtomic`.
- Canonical volumes [] terminates immediately as pinned Resource.list EOF and does not inspect or request its advertised next links, including malformed or unsafe links. The collector judges raw array length without native IsEmpty/ExtractVolumes. On nonempty pages, continuation extraction delegates to actual pinned native VolumePage.NextPageURL for the exact canonical volumes_links envelope; relative next URLs resolve against the current URL and remain on the initial escaped collection path and origin/authority. Userinfo/fragment/opaque/malformed query/path changes and canonical URL cycles including query reordering are rejected before the next request. Advertised query keys remain allowed; no synthetic marker, HTTP Link, envelope substitution or local restart is invented.
  Tests: `blockstorage/get_volumes_test.go:TestGetVolumesContinuationScopeCyclesAndCanonicalEnvelopeUseActualPageProof`, `blockstorage/get_volumes_test.go:TestGetVolumesCompletePagingPreservesAttachmentDuplicatesAndDistinctRawRows`.
- Each physically admitted200 response contributes independently owned Pages Body/header/status before read/Close/decode/source/context failure. Rejected HTTP/native/transport failures add no successful page and use their own actual evidence, never an old200 ResponseError. Fixed request ownership and native auth/reauth/retry behavior are preserved.
  Tests: `blockstorage/get_volumes_test.go:TestGetVolumesAcceptedPageFailuresKeepCurrentProofAndLaterRejectionsNeverBorrowOldPage`, `blockstorage/get_volumes_test.go:TestGetVolumesNativeAuthBackoffRetryKeepsBodylessExactListAndLiveHeaders`.
- Raw response/attachment shape or missing-key failures carry their actual originating page response evidence even when detected after all pages materialize. Provided-server missing/invalid id is local ErrInvalidOption without invented ResponseError. On every error Volumes remains nil, while actual admitted Pages remains available; only full success assigns a new nonnil result slice, including empty.
  Tests: `blockstorage/get_volumes_test.go:TestGetVolumesConsumesAssociationAndServerEvidenceOnlyAfterCompleteList`, `blockstorage/get_volumes_test.go:TestGetVolumesAcceptedPageFailuresKeepCurrentProofAndLaterRejectionsNeverBorrowOldPage`, `blockstorage/get_volumes_test.go:TestGetVolumesContinuationScopeCyclesAndCanonicalEnvelopeUseActualPageProof`.
- Actual Cinder v3 attachments type=list consumption is preserved: arrays retain order, nonnull nonarray wraps one row, and null/missing fails iteration. A wrapped object with canonical server_id is consumable, whereas wrapped empty object/string/nonobject rows fail. This differs deliberately from the separate pure device accessor plain-mapping empty-iterable interpretation.
  Tests: `blockstorage/get_volumes_test.go:TestGetVolumesV3ListifiedAttachmentsAndFullJSONServerEqualityIgnoreDeviceSchema`.
- Every visited attachment must be an object and consume exact canonical server_id before supplied server id. Missing attachment key errors first when both inputs are malformed. Empty volume lists and empty attachment arrays do not consume missing/invalid supplied server evidence. Raw nil server id value is explicit JSON null; invalid JSON/UTF-8 is delayed until the server id is actually read.
  Tests: `blockstorage/get_volumes_test.go:TestGetVolumesConsumesAssociationAndServerEvidenceOnlyAfterCompleteList`, `blockstorage/get_volumes_test.go:TestGetVolumesV3ListifiedAttachmentsAndFullJSONServerEqualityIgnoreDeviceSchema`, `connection_get_volumes_test.go:TestConnectionGetVolumesPreparesOptionsBeforeGetterAndDelaysUnusedServerFields`.
- Association equality covers ordinary JSON string/null/bool/number/array/object without float64 conversion. New EqualPythonJSON supports recursive Python bool-number equality while preserving exact decimal number normalization; existing EqualJSON/MatchFilters semantics remain unchanged. Default request strings are never coerced into numeric/null/bool values.
  Tests: `blockstorage/get_volumes_test.go:TestGetVolumesV3ListifiedAttachmentsAndFullJSONServerEqualityIgnoreDeviceSchema`, `internal/jsonfilter/python_equal_test.go:TestEqualPythonJSONSupportsRecursiveBooleanNumberEqualityAndCompleteShape`, `internal/jsonfilter/python_equal_test.go:TestEqualPythonJSONPreservesExactDecimalPrecisionAndLargeExponents`, `internal/jsonfilter/python_equal_test.go:TestEqualPythonJSONErrorsOwnInputsAndExistingFiltersKeepBooleanNumberDistinction`.
- Every matching attachment appends the same corresponding owned RawResource pointer in volume and attachment order. Duplicate associations are retained without deduplication/break, while separate source rows with the same id remain distinct. Device is never consumed, regardless of absent/null/empty/bool/object values, and volume ID/status/timestamp do not filter results.
  Tests: `blockstorage/get_volumes_test.go:TestGetVolumesCompletePagingPreservesAttachmentDuplicatesAndDistinctRawRows`, `blockstorage/get_volumes_test.go:TestGetVolumesV3ListifiedAttachmentsAndFullJSONServerEqualityIgnoreDeviceSchema`.
- A malformed later attachment fails after an earlier match without exposing a partial Volumes slice. Context/source guards also apply across association processing. A successful result uses a new slice with intentional duplicate row-pointer aliasing; independent per-occurrence mutation is available through explicit Clone.
  Tests: `blockstorage/get_volumes_test.go:TestGetVolumesConsumesAssociationAndServerEvidenceOnlyAfterCompleteList`, `blockstorage/get_volumes_test.go:TestGetVolumesAcceptedPageFailuresKeepCurrentProofAndLaterRejectionsNeverBorrowOldPage`, `blockstorage/get_volumes_test.go:TestGetVolumesPreflightAndCapturedSourceFactsBeforeOptionsOrSubsequentPages`, `connection_get_volumes_test.go:TestConnectionGetVolumesInvalidEntryNilOptionsAndOptionCancellationSelectNoService`.
- RawResource.UnmarshalJSON atomically accepts only a full UTF-8 nonnull JSON object and retains all fields without known-type coercion; MarshalJSON emits its current Body. This library-owned flexible result supports the direct helper device-independent behavior without weakening existing VolumeInfo/native decoders or claiming inherited Python Resource behavior.
  Tests: `resource/raw_resource_test.go:TestRawResourceAcceptsFlexibleCanonicalObjectsAndFailedDecodeIsAtomic`.
- RawResource.Decode atomically projects current Body into a fresh nonnil pointer target, preserves its strict custom known-field decoder behavior and commits caller state only on success. Directly declared exported exact Metadata field seeding transfers independently owned source evidence to compatible SDK targets; projection errors are local and do not manufacture HTTP response failures.
  Tests: `resource/raw_resource_test.go:TestRawResourceDecodeProjectsStrictTypedModelAtomicallyWithIndependentMetadata`, `resource/raw_resource_test.go:TestRawResourceDecodeRejectsInvalidTargetAndRetainsLocalConversionCauses`.
- RawResource.Clone deeply owns Body/raw bytes/header/nullable metadata, making explicit independent copies of otherwise repeated matching row pointers. Options, Pages and separate materialized rows likewise own their captured mutable evidence; caller/result mutations do not rewrite in-flight policy or another independently owned proof.
  Tests: `resource/raw_resource_test.go:TestRawResourceCloneOwnsRawHeaderNullableMetadataAndLinks`, `resource/raw_resource_test.go:TestRawResourceDecodeProjectsStrictTypedModelAtomicallyWithIndependentMetadata`, `blockstorage/get_volumes_test.go:TestGetVolumesCompletePagingPreservesAttachmentDuplicatesAndDistinctRawRows`, `blockstorage/get_volumes_test.go:TestGetVolumesOptionsSnapshotsReplacementOrderedCallbacksAndDelayedFields`.
- The pinned cloud cache parameter is deprecated warning-only/no-op and is deliberately omitted in Go. Review maps only this direct get_volumes declaration and its explicit Go policies; separate cloud list/get/id/exists/search, native/Proxy operations and inherited Resource/session/cache remain independently reviewed.
  Tests: `blockstorage/get_volumes_test.go:TestGetVolumesCompletePagingPreservesAttachmentDuplicatesAndDistinctRawRows`, `connection_get_volumes_test.go:TestConnectionGetVolumesUsesOnlyCachedCinderAndOriginalOptionsOnce`, `connection_get_volumes_test.go:TestConnectionGetVolumesPreparesOptionsBeforeGetterAndDelaysUnusedServerFields`.

Python/Go differences:

- Pinned direct source is openstacksdk ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe get_volumes. Go composes selected Cinder v3 detailed-list behavior under Gophercloud v2.15.0 and omits the Python deprecated cache argument/warning, which never changes listing. This unit does not promote list_volumes or any separate get/id/exists/search/native/Proxy/inherited declaration.
- Go request ServerID is an explicit string projection of supplied server[id]; ServerFields represents ordinary JSON plain mapping id values with lazy consumption. Custom Python mapping/equality/iterables, Resource missing-default/server descriptor behavior and arbitrary object values are outside this JSON representation. Explicit null uses present id:null/nil RawMessage, while missing id remains a local error only when consumed.
- Go uses library-owned RawResource to preserve device-independent direct selection and unrelated raw field precision, rather than eagerly forcing known VolumeInfo/native types. Explicit atomic Decode provides stricter typed projection and copied Metadata evidence. No arbitrary Python Resource descriptor/session/cache behavior is implicitly recreated; Cinder v3 attachment type=list wrapping is mapped for this workflow separately from the pure plain-mapping device accessor.
- Go ordinary JSON equality supports recursive bool-number equality as Python does, but exact decimal normalization avoids float64 rounding. This deliberately differs from Python floating-point/custom numeric objects and arbitrary Unicode/equality semantics. Existing collection EqualJSON/MatchFilters behavior remains independent and unchanged.
- Python returns a new list of original Volume object references and raises on failure. Go returns a new slice of owned raw row pointers with duplicate occurrences intentionally aliasing the same row, plus independently owned actual Pages evidence; any error leaves Volumes nil. Explicit Clone produces per-occurrence independent copies, rather than silently deduplicating or splitting repeated pointer identity.
- Go requires complete UTF-8 object pages with canonical nonnull volumes array/object rows; pinned Resource.list can wrap a nonarray resources value and strips a self key during Resource construction, while Go raw results preserve complete row fields. These are explicit canonical-response/raw-ownership boundaries rather than automatic reproduction of inherited Resource hydration. Empty volumes [] is still actual EOF and suppresses all unused next-link processing. Nonempty continuation uses only canonical native volumes_links plus fixed same-origin escaped collection routing/source/context policies; response selection examines consumed attachments/server_id and ignores device/other known field types. Accepted proof survives processing errors without being borrowed for later rejected/blocked requests.
- Go context controls preparation, discovery, HTTP and selection; policy callbacks are owned and run once. Ordinary captured headers coexist with live provider authentication and existing native retries. Direct package does no version discovery/fallback or partial successful selection, and no rollback/replay/member GET is added.
- This go_mapping covers only the direct pinned get_volumes declaration under the explicit documented Go policies. Generic RawResource and jsonfilter additions are library-owned support extensions rather than blanket promotion of unrelated Python Resource/session/cache/native/Proxy or whole-SDK capabilities.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func getForServer(ctx context.Context, conn *sdk.Connection, serverID string) error {
    result, err := conn.GetVolumes(ctx, blockstorage.GetVolumesRequest{ServerID: serverID})
    return inspectResult(result, err)
}

func getWithClient(ctx context.Context, cinder *gophercloud.ServiceClient, serverID string) (*blockstorage.GetVolumesResult, error) {
    return blockstorage.GetVolumes(ctx, cinder,
        blockstorage.GetVolumesRequest{ServerID: serverID})
}

func getForRawServer(ctx context.Context, conn *sdk.Connection) (*blockstorage.GetVolumesResult, error) {
    // An explicit null ID can match a null attachment server_id.
    server := map[string]json.RawMessage{"id": json.RawMessage(`null`)}
    return conn.GetVolumes(ctx, blockstorage.GetVolumesRequest{},
        blockstorage.WithGetVolumesServerFields(server))
}

func prepareServerFields(ctx context.Context, server map[string]json.RawMessage) (blockstorage.GetVolumesOpts, error) {
    // Runs options locally once and owns their maps/raw bytes; no HTTP.
    return blockstorage.PrepareGetVolumesOptions(ctx,
        blockstorage.WithGetVolumesServerFields(server))
}

func getWithPreparedOptions(ctx context.Context, conn *sdk.Connection, prepared blockstorage.GetVolumesOpts) (*blockstorage.GetVolumesResult, error) {
    return conn.GetVolumes(ctx, blockstorage.GetVolumesRequest{},
        blockstorage.WithGetVolumesOptions(prepared))
}

func inspectResult(result *blockstorage.GetVolumesResult, callErr error) error {
    if callErr != nil {
        if result != nil {
            // Volumes is nil on every workflow error. Pages is actual response evidence.
            for i, page := range result.Pages {
                fmt.Printf("accepted page %d: status=%d request=%q bytes=%d\n",
                    i, page.StatusCode, page.Header.Get("X-Openstack-Request-Id"), len(page.Body))
            }
        }
        return callErr
    }
    for i, volume := range result.Volumes {
        fmt.Printf("match %d: raw id=%s raw status=%s\n", i,
            volume.Body["id"], volume.Body["status"])

        var projected blockstorage.VolumeInfo
        if err := volume.Decode(&projected); err != nil {
            // Selection succeeded; this optional stricter typed projection failed.
            fmt.Printf("typed projection %d: %v\n", i, err)
        } else {
            fmt.Printf("typed projection %d: status=%d request=%q\n", i,
                projected.StatusCode, projected.Header.Get("X-Openstack-Request-Id"))
        }

        privateCopy := volume.Clone()
        privateCopy.Body["name"] = json.RawMessage(`"private-copy"`)
        fmt.Printf("copy %d name=%s; original name=%s\n", i,
            privateCopy.Body["name"], volume.Body["name"])
    }
    return nil
}

func independentMatches(result *blockstorage.GetVolumesResult) []*resource.RawResource {
    copies := make([]*resource.RawResource, len(result.Volumes))
    for i, volume := range result.Volumes {
        copies[i] = volume.Clone()
    }
    return copies
}
```

```python
server = {"id": "server-id"}
volumes = conn.get_volumes(server)
for volume in volumes:
    print(volume["id"])
```

## References

- [..](../README.md)
- [Volume Attachment Device](volume-attachment-device.md)
- [Reference](README.md)
- [get_volumes](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L323-L346)
- [list_volumes](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py#L52-L64)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L890-L921)
- [attachments descriptor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L54)
- [list conversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86-L111)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2340-L2359)
- [VolumePage.NextPageURL](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/volumes/results.go#L126-L135)
