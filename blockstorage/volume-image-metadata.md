# Volume Image Metadata

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/set_volume_image_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageMetadataOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageMetadataValue`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageMetadataRaw`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeImageMetadataOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.SetVolumeImageMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageMetadataOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageMetadataValue`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageMetadataRaw`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeImageMetadataOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.SetVolumeImageMetadata`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SetVolumeImageMetadata`

- Set sends exactly one os-set_image_metadata action with metadata object, including omitted/nil/empty maps as {}. Literal empty/control/path-like strings and raw null/false/exact large/nested JSON members are preserved in the body; no lookup/model refresh/wait is added. Admitted opaque 200/203/204/302/399 responses complete acknowledgement without JSON parsing.
  Tests: `blockstorage/volume_image_metadata_test.go:TestSetVolumeImageMetadataPreservesEmptyMapsRawValuesAndOwnedFactories`, `blockstorage/volume_image_metadata_test.go:TestVolumeImageMetadataServiceFacadePreservesNativeSetAndPublicPreflight`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataUsesCachedCinderAndPreservesAllEmptyAndDuplicateIntents`.
- String/raw/full factories replace policy; Value alone merges one string member. HTTP-free reusable Prepare owns maps/member bytes and fresh empty defaults, validates only final UTF-8 keys/strings/complete JSON values, permits later repair of invalid members, and returns zero policy on failures.
  Tests: `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataOptionFactoriesOwnInputsReturnedValuesAndReplacementOrder`, `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataPrepareValidatesOnlyFinalMapsAndPresentKeys`, `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataPrepareNilCallbacksAndCancellationKeepZeroPolicyAndEveryCause`.
- Originals snapshot their callback slice, run once in order, copy intermediate maps/raw bytes and join callback/custom-context causes. Direct selected Source/current-ID admission precedes originals; sticky ResourceBase mutation stops a later restoration callback before HTTP.
  Tests: `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataOriginalsOwnCallbackSliceIntermediateMapsAndKeySlices`, `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataPrepareNilCallbacksAndCancellationKeepZeroPolicyAndEveryCause`, `blockstorage/volume_image_metadata_test.go:TestSetVolumeImageMetadataPreservesEmptyMapsRawValuesAndOwnedFactories`, `blockstorage/volume_image_metadata_test.go:TestVolumeImageMetadataServiceFacadePreservesNativeSetAndPublicPreflight`.
- Connection preflights context/provider, prepares originals once before current-ID/cached Cinder selection, ignores configured CurrentLocation, and detaches retained callback maps through the getter. Local/callback/getter/cached wrong-service failures preserve operation/cause and create no HTTP proof.
  Tests: `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataUsesCachedCinderAndPreservesAllEmptyAndDuplicateIntents`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataOriginalsRunOnceBeforeGetterAndRetainedOptionsStayDetached`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataPreflightOptionsAndGetterFailuresKeepEveryCause`.
- Nonempty selected 3.60/3.90 is literal and skips discovery; common ordinary action selection has no minimum gate and cap3.71. Malformed admitted unselected discovery blocks set with only its actual Discovery/ResponseError, while valid common batch negotiation is independently exercised by delete.
  Tests: `blockstorage/volume_image_metadata_test.go:TestSetVolumeImageMetadataPreservesEmptyMapsRawValuesAndOwnedFactories`, `blockstorage/volume_image_metadata_test.go:TestVolumeImageMetadataNegotiatesOnceAndStopsAtCurrentNativeFailureWithoutBorrowedProof`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataUsesCachedCinderAndPreservesAllEmptyAndDuplicateIntents`.
- Accepted Read/Close/source/custom-context failures retain independently owned current Applied/error proof with Completedfalse and no SDK replay. Live auth/reauth/bounded retry keep original body/version/admission; retry/physical307 body/framing/URL/version changes are blocked and native rejection cannot fabricate Applied. Service native SetImageMetadata nil=>null/200-only/error-only remains independent.
  Tests: `blockstorage/volume_image_metadata_transport_test.go:TestVolumeImageMetadataAcceptedFailuresRetainOnlyCurrentPhaseProofAndJoinedCauses`, `blockstorage/volume_image_metadata_transport_test.go:TestVolumeImageMetadataNativeRetriesAndPhysicalMutationGuardsKeepBatchPolicy`, `blockstorage/volume_image_metadata_test.go:TestVolumeImageMetadataServiceFacadePreservesNativeSetAndPublicPreflight`.

Python/Go differences:

- Python **metadata supplies a kwargs dictionary, including empty {}. Go string/raw/full factories replace request policy and Value merges; this is request assembly, not a claim that Cinder atomically replaces its stored image metadata.
- Go raw metadata permits owned ordinary JSON null/bool/numbers/containers without float preconversion, but rejects invalid UTF-8/incomplete JSON/nil raw members and excludes arbitrary Python objects/datetime/UUID/custom serializer behavior. Map member order and numeric/string/escape serialization are Go JSON representation boundaries, not Python mutable kwargs identity.
- Native API.SetImageMetadata remains independently map[string]string, nil metadata:null, HTTP200/error-only; ordinary MetadataIn childroutes are unchanged.
- Go safe explicit current volume ID and owned option/result values replace Source supplied Resource/dict/Munch/None/dynamic inherited ID and constructor/descriptor/location/session side effects. No name finder, CurrentLocation, refresh, state wait or model conversion is introduced.
- The Source helper returns None after response status<400. Go finite100..399 admission exposes owned actual stage responses and Completed acknowledgement; this does not prove the final server/backend metadata state.
- Go finite operation-owned ordinary discovery/cap3.71 and literal nonempty selected version differ from complete Keystoneauth endpoint-data cache/canonicalization/exception ordering. Empty Go selection cannot represent every distinct Python default None/empty value. Original client selection is not rewritten.
- Fixed method/URL/serialized-body/framing/version and captured Source constrain native retry/redirect while live provider authentication is allowed. Native original rejected HTTP remains native; discarded rejected HTTP body IO is not claimed fully reconstructed.
- Only the actual pinned v3 Proxy target is classified as go_mapping. Native/v2/Resource/MetadataMixin/ordinary MetadataIn and shared discovery dependencies are not promoted. Source/static comparison and Root offline Go HTTP/example checks do not imply Python/dependency/authenticated OpenStack execution or full SDK completion.

### `python:block_storage/v3/delete_volume_image_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageMetadataDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageMetadataDeleteKeys`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WithVolumeImageMetadataDeleteAll`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.PrepareVolumeImageMetadataDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.DeleteVolumeImageMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageMetadataDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageMetadataDeleteKeys`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithVolumeImageMetadataDeleteAll`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.PrepareVolumeImageMetadataDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.DeleteVolumeImageMetadata`
- `github.com/JSYoo5B/go-openstacksdk.Connection.DeleteVolumeImageMetadata`

- Keys pointer nil/default means all; a nonnil pointer including nil/empty slice or zero-argument WithKeys means explicit none with zero HTTP/discovery after local admission. Explicit UTF-8 keys, including empty/control/path-like text, preserve order and duplicates as body literals with no GET; every final key is validated before earlier actions.
  Tests: `blockstorage/volume_image_metadata_test.go:TestDeleteVolumeImageMetadataExplicitKeysKeepOrderDuplicatesAndEagerValidation`, `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataOptionFactoriesOwnInputsReturnedValuesAndReplacementOrder`, `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataPrepareValidatesOnlyFinalMapsAndPresentKeys`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataUsesCachedCinderAndPreservesAllEmptyAndDuplicateIntents`.
- Default all intentionally corrects the pinned ordinary-metadata iteration bug: fixed-ID bodyless member GET admits sub400 but requires canonical volume object and reads only volume_image_metadata. Missing/null/empty image map emits no POST; nonobject/bad envelope/malformed JSON keeps actual Observed/error proof. Image keys are snapshotted/sorted, ignoring ordinary metadata, returned ID and unrelated typed fields.
  Tests: `blockstorage/volume_image_metadata_test.go:TestDeleteVolumeImageMetadataAllReadsOnlyCanonicalImageFieldAndKeepsObservation`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataUsesCachedCinderAndPreservesAllEmptyAndDuplicateIntents`.
- Owned factories/full replacement/All/Keys and reusable HTTP-free Prepare preserve pointer presence, slice order/duplicates and returned-policy independence. Original callback slice and intermediate key backing arrays stay detached; final repaired keys succeed, and nil/callback/cancellation failures return zero policy with every cause.
  Tests: `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataOptionFactoriesOwnInputsReturnedValuesAndReplacementOrder`, `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataOriginalsOwnCallbackSliceIntermediateMapsAndKeySlices`, `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataPrepareValidatesOnlyFinalMapsAndPresentKeys`, `blockstorage/volume_image_metadata_options_test.go:TestVolumeImageMetadataPrepareNilCallbacksAndCancellationKeepZeroPolicyAndEveryCause`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataOriginalsRunOnceBeforeGetterAndRetainedOptionsStayDetached`.
- Direct Source/current-ID admission precedes original callbacks even for explicit none. Connection preflight prepares originals once before current-ID/cached Cinder selection, consumes no CurrentLocation, and validates unsafe IDs/invalid final keys/getter/cached source failures without member/action HTTP or borrowed proof.
  Tests: `blockstorage/volume_image_metadata_test.go:TestDeleteVolumeImageMetadataExplicitKeysKeepOrderDuplicatesAndEagerValidation`, `blockstorage/volume_image_metadata_test.go:TestVolumeImageMetadataServiceFacadePreservesNativeSetAndPublicPreflight`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataUsesCachedCinderAndPreservesAllEmptyAndDuplicateIntents`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataOriginalsRunOnceBeforeGetterAndRetainedOptionsStayDetached`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataPreflightOptionsAndGetterFailuresKeepEveryCause`.
- A nonempty selected version skips discovery; otherwise one ordinary cap3.71 negotiation supplies the entire batch GET/POST version and separate actual Discovery. First native action failure stops later keys, retains earlier Deleted/optional Observed, records Failed key with nil Response while the returned error retains current native status/body/header, and cannot borrow admitted ResponseError proof.
  Tests: `blockstorage/volume_image_metadata_test.go:TestVolumeImageMetadataNegotiatesOnceAndStopsAtCurrentNativeFailureWithoutBorrowedProof`, `connectiontest/connection_volume_image_metadata_test.go:TestConnectionVolumeImageMetadataUsesCachedCinderAndPreservesAllEmptyAndDuplicateIntents`, `blockstorage/volume_image_metadata_transport_test.go:TestVolumeImageMetadataNativeRetriesAndPhysicalMutationGuardsKeepBatchPolicy`.
- Per-key opaque acknowledgement bodies/headers and member/error proofs own independent bytes. Accepted IO/source/custom cancellation retains only the current Observed or Failed.Response plus earlier completed keys, without replay/rollback; Completed only means helper acknowledgement. Live auth and bounded retry preserve batch policy, physical bodyless GET/POST/URL/version/framing mutations are blocked, and rejected injected GET Body Close runs once with close cause retained.
  Tests: `blockstorage/volume_image_metadata_test.go:TestDeleteVolumeImageMetadataExplicitKeysKeepOrderDuplicatesAndEagerValidation`, `blockstorage/volume_image_metadata_test.go:TestDeleteVolumeImageMetadataAllReadsOnlyCanonicalImageFieldAndKeepsObservation`, `blockstorage/volume_image_metadata_transport_test.go:TestVolumeImageMetadataAcceptedFailuresRetainOnlyCurrentPhaseProofAndJoinedCauses`, `blockstorage/volume_image_metadata_transport_test.go:TestVolumeImageMetadataNativeRetriesAndPhysicalMutationGuardsKeepBatchPolicy`.

Python/Go differences:

- Pinned keys=None calls Volume.delete_image_metadata, which loops inherited ordinary self.metadata rather than volume_image_metadata. ID-only missing ordinary metadata defaults to None and implies iteration TypeError by static inspection; no Python execution is asserted. Go intentionally repairs this branch with fresh fixed-ID image-field GET and sorted snapshot keys.
- The repaired all path is not an atomic clear: keys added after observation are not included, concurrent changes can cause a later failure, earlier deletions remain, and there is no implicit merge/write/retry/rollback promise.
- Python non-None keys can be an arbitrary lazy iterable, string character iteration or generator failing after earlier POSTs. Go narrows to an eager owned UTF-8 string slice; nonnil pointer to nil/empty slice explicitly means none. Final key validation occurs before any mutation and duplicate order is preserved.
- Source selects microversion separately for every key action using mutable session state. Go negotiates/captures one version for the whole batch, including the added default-all GET; this deliberate stable policy does not reproduce per-key session mutation.
- Observed/Discovery/Deleted/Failed and current native/error proof are distinct owned phases. A failure before attempting the next key can leave Failed nil, while an attempted native rejected action has Failed with nil Response. Earlier acknowledgement is not borrowed as current evidence.
- Go safe explicit current volume ID and owned option/result values replace Source supplied Resource/dict/Munch/None/dynamic inherited ID and constructor/descriptor/location/session side effects. No name finder, CurrentLocation, refresh, state wait or model conversion is introduced.
- The Source helper returns None after response status<400. Go finite100..399 admission exposes owned actual stage responses and Completed acknowledgement; this does not prove the final server/backend metadata state.
- Go finite operation-owned ordinary discovery/cap3.71 and literal nonempty selected version differ from complete Keystoneauth endpoint-data cache/canonicalization/exception ordering. Empty Go selection cannot represent every distinct Python default None/empty value. Original client selection is not rewritten.
- Fixed method/URL/serialized-body/framing/version and captured Source constrain native retry/redirect while live provider authentication is allowed. Native original rejected HTTP remains native; discarded rejected HTTP body IO is not claimed fully reconstructed.
- Only the actual pinned v3 Proxy target is classified as go_mapping. Native/v2/Resource/MetadataMixin/ordinary MetadataIn and shared discovery dependencies are not promoted. Source/static comparison and Root offline Go HTTP/example checks do not imply Python/dependency/authenticated OpenStack execution or full SDK completion.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    if err := run(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run() error {
    cloudName, volumeID := os.Getenv("CLOUD_NAME"), os.Getenv("VOLUME_ID")
    if cloudName == "" || volumeID == "" { return fmt.Errorf("CLOUD_NAME and VOLUME_ID are required") }
    action, path := os.Getenv("IMAGE_METADATA_ACTION"), os.Getenv("IMAGE_METADATA_PATH")
    if action != "set" && action != "delete" { return fmt.Errorf("IMAGE_METADATA_ACTION must be set or delete") }
    if path == "" { path = "connection" }
    if path != "connection" && path != "direct" && path != "service" { return fmt.Errorf("unknown IMAGE_METADATA_PATH %q", path) }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    input := blockstorage.VolumeActionRequest{VolumeID: volumeID}
    var set blockstorage.VolumeImageMetadataOpts
    var deletion blockstorage.VolumeImageMetadataDeleteOpts
    if action == "set" {
        var values map[string]json.RawMessage
        if text, supplied := os.LookupEnv("IMAGE_METADATA_JSON"); supplied {
            if err := json.Unmarshal([]byte(text), &values); err != nil { return fmt.Errorf("IMAGE_METADATA_JSON: %w", err) }
        }
        options := []blockstorage.VolumeImageMetadataOption{blockstorage.WithVolumeImageMetadataRaw(values)}
        if key, supplied := os.LookupEnv("IMAGE_METADATA_KEY"); supplied {
            options = append(options, blockstorage.WithVolumeImageMetadataValue(key, os.Getenv("IMAGE_METADATA_VALUE")))
        }
        prepared, err := blockstorage.PrepareVolumeImageMetadataOptions(ctx, options...)
        if err != nil { return err }
        set = prepared
    } else {
        var options []blockstorage.VolumeImageMetadataDeleteOption
        if text, supplied := os.LookupEnv("IMAGE_METADATA_KEYS_JSON"); supplied {
            var keys []string
            if err := json.Unmarshal([]byte(text), &keys); err != nil { return fmt.Errorf("IMAGE_METADATA_KEYS_JSON: %w", err) }
            options = append(options, blockstorage.WithVolumeImageMetadataDeleteKeys(keys...))
        }
        prepared, err := blockstorage.PrepareVolumeImageMetadataDeleteOptions(ctx, options...)
        if err != nil { return err }
        deletion = prepared
    }
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    if action == "set" {
        result, err := applySet(ctx, conn, path, input, set)
        inspectSet(result)
        inspectError(err)
        return err
    }
    result, err := applyDelete(ctx, conn, path, input, deletion)
    inspectDelete(result)
    inspectError(err)
    return err
}

func applySet(ctx context.Context, conn *sdk.Connection, path string, input blockstorage.VolumeActionRequest, policy blockstorage.VolumeImageMetadataOpts) (*blockstorage.VolumeActionResult, error) {
    option := blockstorage.WithVolumeImageMetadataOptions(policy)
    if path == "connection" { return conn.SetVolumeImageMetadata(ctx, input, option) }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    if path == "direct" { return blockstorage.SetVolumeImageMetadata(ctx, cinder.RawClient(), input, option) }
    return cinder.Volumes.SetVolumeImageMetadata(ctx, input.VolumeID, option)
}

func applyDelete(ctx context.Context, conn *sdk.Connection, path string, input blockstorage.VolumeActionRequest, policy blockstorage.VolumeImageMetadataDeleteOpts) (*blockstorage.VolumeImageMetadataDeleteResult, error) {
    option := blockstorage.WithVolumeImageMetadataDeleteOptions(policy)
    if path == "connection" { return conn.DeleteVolumeImageMetadata(ctx, input, option) }
    cinder, err := conn.BlockStorageV3(ctx)
    if err != nil { return nil, err }
    if path == "direct" { return blockstorage.DeleteVolumeImageMetadata(ctx, cinder.RawClient(), input, option) }
    return cinder.Volumes.DeleteVolumeImageMetadata(ctx, input.VolumeID, option)
}

func inspectSet(result *blockstorage.VolumeActionResult) {
    if result == nil { return }
    fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
    for index, page := range result.Discovery {
        if page != nil { fmt.Println("admitted discovery:", index, page.StatusCode, len(page.Body)) }
    }
    if result.Applied != nil { fmt.Printf("admitted set: HTTP %d; opaque body %q\n", result.Applied.StatusCode, result.Applied.Body) }
    fmt.Println("helper acknowledgement completed:", result.Completed)
}

func inspectDelete(result *blockstorage.VolumeImageMetadataDeleteResult) {
    if result == nil { return }
    fmt.Printf("volume ID: %q; microversion: %q\n", result.VolumeID, result.Microversion)
    for index, page := range result.Discovery {
        if page != nil { fmt.Println("admitted discovery:", index, page.StatusCode, len(page.Body)) }
    }
    if result.Observed != nil { fmt.Printf("admitted member GET: HTTP %d; body %q\n", result.Observed.StatusCode, result.Observed.Body) }
    for _, entry := range result.Deleted {
        if entry.Response != nil { fmt.Printf("acknowledged key %q: HTTP %d; opaque body %q\n", entry.Key, entry.Response.StatusCode, entry.Response.Body) }
    }
    if result.Failed != nil {
        fmt.Printf("failed key: %q\n", result.Failed.Key)
        if page := result.Failed.Response; page != nil { fmt.Printf("admitted failed-key response: HTTP %d; opaque body %q\n", page.StatusCode, page.Body) }
    }
    fmt.Println("helper acknowledgement completed:", result.Completed)
}

func inspectError(err error) {
    if err == nil { return }
    var proof *resource.ResponseError
    if errors.As(err, &proof) { fmt.Println("current admitted error response:", proof.StatusCode, len(proof.Body)) }
}
```

```python
import openstack

conn = openstack.connect(cloud="devstack")
conn.block_storage.set_volume_image_metadata(volume_id, hw_disk_bus="scsi")
conn.block_storage.set_volume_image_metadata(volume_id, **{})
conn.block_storage.delete_volume_image_metadata(volume_id, keys=["z", "a", "z"])
conn.block_storage.delete_volume_image_metadata(volume_id, keys=[])
# Pinned all-path calls Volume.delete_image_metadata, which loops ordinary metadata.
conn.block_storage.delete_volume_image_metadata(volume)
```

## References

- [Proxy set/delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1164-L1197)
- [Volume image metadata actions](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L189-L207)
- [Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/volume.py#L196-L200)
- [Volumes](v3/volumes/README.md)
- [Volume Actions](volume-actions.md)
- [Reference](README.md)
