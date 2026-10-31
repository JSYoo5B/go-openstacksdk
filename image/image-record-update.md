# Image Record Update

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/update_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.UpdateImage`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageChange`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageChanges`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageName`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageVisibility`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageProtected`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageHidden`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageOwner`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageContainerFormat`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageDiskFormat`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageMinDisk`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageMinRAM`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageTags`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageField`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageFields`
- `github.com/JSYoo5B/go-openstacksdk/image.WithUpdateImageRemoveField`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.UpdateImageRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.UpdateImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordAttributes`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordHeaders`

- Required Ref uses direct safe literal ID without GET or existing exact native Name resolution then fixed PATCH; values/options validate before Name HTTP. v2.1 media and Accept are set on private client after Name, source untouched, actual 200 only.
  Tests: `image/update_core_test.go:TestUpdateImageCoreFixedOrderedPatchAndMedia`, `image/update_core_test.go:TestUpdateImageCoreExactNameAndSource`, `image/update_core_test.go:TestUpdateImageCoreCompletePreflight`, `image/update_contracts_test.go:TestImageUpdateContractsWireAndReference`, `image/update_contracts_test.go:TestImageUpdateContractsOptionsAndSource`, `connectiontest/connection_image_update_test.go:TestConnectionImageUpdateFacadeOwnsPatchAndResponse`.
- Ordered concrete ImagePatch array retains duplicate paths and submitted sequence. FullOpts/plural Changes replace; singular Change and typed/Fields helpers append. Nil/empty inputs send nonnil [] PATCH and return real ImageInfo; no dirty comparison/dedup/local noop.
  Tests: `image/update_core_test.go:TestUpdateImageCoreFixedOrderedPatchAndMedia`, `image/update_core_test.go:TestUpdateImageCoreSortedPropertiesAndEmptyPatch`, `image/update_options_test.go:TestUpdateImageOptionsReplacementAppendAndTypedFields`, `image/update_contracts_test.go:TestImageUpdateContractsPatchAndProperties`.
- Exact add/replace require present nonempty validUTF8 JSON raw bytes including explicit null; remove Value strictly nil and nonnil empty stays invalid. Generic JSON Pointer validates escapes/nonnull decoded tokens/Python-strip stable edges including U+001C..U+001F, with no percent decoding or client path-depth gate.
  Tests: `image/update_core_test.go:TestUpdateImageCoreCompletePreflight`, `image/update_options_test.go:TestUpdateImageOptionsFactoriesAndNilPresence`, `image/update_contracts_test.go:TestImageUpdateContractsPatchAndProperties`.
- Typed ten mutable-field helpers use exact wire add/upsert, preserve string empty/bool false/int64 zero/negative, Tags zeroargs nonnil [], arbitrary Fields/Field/RemoveField use escaped exact root keys. Schema/type/range/readonly/reserved/locations validation remains server-owned.
  Tests: `image/update_options_test.go:TestUpdateImageOptionsReplacementAppendAndTypedFields`, `image/update_contracts_test.go:TestImageUpdateContractsPatchAndProperties`.
- Immediate helper snapshots and callback configs own map/slice/raw bytes, nil/custom/marshal causes preserved, full replacement/header merge canonical lastwins and reusable parallel policies retain independent input. any helper snapshots encoding/json once; no original-key UTF8 normalization.
  Tests: `image/update_options_test.go:TestUpdateImageOptionsFactoriesAndNilPresence`, `image/update_options_test.go:TestUpdateImageOptionsCallbacksOwnRetainedState`, `image/update_options_test.go:TestUpdateImageOptionsReusablePoliciesAndFactoryCauses`, `image/update_contracts_test.go:TestImageUpdateContractsOptionsAndSource`.
- Actual successful PATCH uses unchanged ImageInfo atomic25 canonical types plus independent Body/noncanonical raw Properties/header/status, passive ID/date/URLs and raw precision/null with no request seed/header pseudoJSON/forced properties object.
  Tests: `image/update_core_test.go:TestUpdateImageCoreFixedOrderedPatchAndMedia`, `image/update_contracts_test.go:TestImageUpdateContractsCanonicalResponse`, `connectiontest/connection_image_update_test.go:TestConnectionImageUpdateFacadeOwnsPatchAndResponse`.
- Accepted read/Close/context/customcause/source/JSON/model failures return nil model with whole actual ResponseError and Close once/no replay; fixed provider/target/microversion/liveauth and native advanced MoreHeaders/prebody retry/reauth/redirect/body/status policy preserve existing boundaries.
  Tests: `image/update_core_test.go:TestUpdateImageCoreAcceptedFailureEvidence`, `image/update_core_test.go:TestUpdateImageCoreNativePrebodyOwnership`, `image/update_contracts_test.go:TestImageUpdateContractsCanonicalResponse`, `image/update_contracts_test.go:TestImageUpdateContractsNativePolicy`, `internal/cmd/sdkgen/glance_image_update_test.go:TestGlanceImageUpdatePreservesNativeAndRegistry`.
- Owned request selects a literal ID or SDK-produced private-state Record with concrete attributes/options, no implicit GET/name lookup, and raw current/original presence independent from projected defaults or public view edits. Recursive Python JSON-domain same-value comparison suppresses clean commits; missing versus null/list remains changed and dirty is sticky. ID dirty is discarded without restoring identity: clean ID-only changes no-op, pending/non-ID changes route by current ID and may include /id. Literal attribute ID collisions and Go fixed-profile control names reject before HTTP.
  Tests: `image/image_record_update_test.go:TestImageRecordUpdateNoopUsesRawPresenceAndPythonEquality`, `image/image_record_update_test.go:TestImageRecordUpdateMissingNullAndGetterConversionsCreateRawChanges`, `image/image_record_update_test.go:TestImageRecordUpdateIDOnlyRetargetAndPrivateOriginal`, `image/image_record_update_test.go:TestImageRecordUpdatePreflightIncludesSourceControlsAndFinalDescriptors`.
- Owned updates retain raw scalar/string/null/list values before descriptor projection and transpose canonical components to wire aliases, collapsing owner aliases to owner. Incoming unknown attributes replace the prior properties component; explicit incoming properties and unknowns pack together. Truthy property dictionaries flatten after declared wire fields and can override patch values without choosing the route; truthy strings remain properties while other excluded shapes may remain dirty with empty diff. Escaped JSON pointers support exact arbitrary object keys including empty keys.
  Tests: `image/image_record_update_test.go:TestImageRecordUpdateLiteralSparseAutomaticPatchAndReceipts`, `image/image_record_update_test.go:TestImageRecordUpdateMissingNullAndGetterConversionsCreateRawChanges`, `image/image_record_update_test.go:TestImageRecordUpdatePropertyReplacementFlatteningAndPointerEscapes`.
- Owned PATCH compares complete flattened original/current JSON rather than selecting dirty upserts. Shared deterministic RFC6902 add/remove/replace covers nested objects, arrays, shape/null/presence and escaped pointers, with exact numerical equality and boolean/number distinction. Arrays retain a deterministic common subsequence and executable indices; no global path sorting or move/copy byte equivalence is promised. Empty diffs serialize as [] and remain sendable when dirty requires commit.
  Tests: `internal/jsonpatch/diff_test.go:TestDiffAppliesNestedAndArrayEdits`, `internal/jsonpatch/diff_test.go:TestDiffHasDeterministicConcreteOperations`, `internal/jsonpatch/diff_test.go:TestDiffKeepsArrayIndicesExecutableAcrossTenAndRepeatedRemoval`, `internal/jsonpatch/diff_test.go:TestDiffUsesExactJSONNumberEquality`, `internal/jsonpatch/diff_test.go:TestDiffCompleteDocumentMatrixAppliesDeterministically`, `image/image_record_update_test.go:TestImageRecordUpdatePropertyReplacementFlatteningAndPointerEscapes`, `image/image_record_update_test.go:TestImageRecordUpdateValidResponseCleansAndInvalidSyntaxKeepsPending`.
- Valid object responses overlay submitted current, fetch-pack properties including {}, preserve omitted declared fields, and establish a clean full-current baseline. Tolerated empty/invalid JSON keeps current/original/pending state for explicit subsequent calls, including constructor attributes from invalid-JSON GetImageRecord. ImportMethods resets during preparation and consumes a plain input attribute without Body dirtiness, then is replaced by actual response headers. No-op returns an independent Record with prior receipt and no fabricated HTTP.
  Tests: `image/image_record_update_test.go:TestImageRecordUpdateValidResponseCleansAndInvalidSyntaxKeepsPending`, `image/image_record_update_test.go:TestImageRecordUpdateReusesPendingInvalidJSONGetWithoutSyntheticDefaults`, `image/image_record_update_test.go:TestImageRecordUpdateConsumesPlainImportMethodsWithoutBodyCommit`, `image/image_record_update_test.go:TestImageRecordUpdateNoopUsesRawPresenceAndPythonEquality`.
- Record/source/attributes/options are captured before location mutation callbacks; concrete options overlay request attributes with reusable independent maps/raw bytes, apply once, and preserve custom marshal/option/context causes. Fixed literal PATCH has v2.1 Content-Type, blank Accept and actual200..399 acceptance. Guarded native retry retains ordinary headers/live authentication but cannot alter owned representation; source drift including restored drift stays sticky and accepted responses are not automatically replayed.
  Tests: `image/image_record_update_test.go:TestImageRecordUpdateSourceOptionsAndAttributeSnapshots`, `image/image_record_update_test.go:TestImageRecordUpdateCapturesOptionsAndTopLevelAttributesBeforeMarshal`, `image/image_record_update_test.go:TestImageRecordUpdateRetryKeepsOwnedPatchHeadersAndLiveAuthentication`, `image/image_record_update_test.go:TestImageRecordUpdateAccepts200Through399WithoutAutomaticReplay`, `image/image_record_update_test.go:TestImageRecordUpdateReadCloseStickyGuardAndRejectionProof`.
- Accepted Read/Close/context/source/decode/projection failures return an independent submitted pending Record plus cause-safe actual ResponseError evidence and passive Wire when decodable, without baseline clean or accepted replay. Preflight/no-accepted-response failures return nil Record; HTTP rejection retains native proof. Raw state, public view, receipt/header/bytes and inputs do not alias. Connection delegates through shared Image/ImageV2 binding, captured location/live auth, and clean returned records can be reused without HTTP.
  Tests: `image/image_record_update_test.go:TestImageRecordUpdateAcceptedMalformedObjectAndProjectionRetainEvidence`, `image/image_record_update_test.go:TestImageRecordUpdateReadCloseStickyGuardAndRejectionProof`, `image/image_record_update_test.go:TestImageRecordUpdatePendingRecordsDoNotAliasInputsOrReceipts`, `connectiontest/image_record_update_integration_test.go:TestConnectionOwnedImageUpdateSharesBindingSourceAndCleanLifecycle`.

Python/Go differences:

- Legacy explicit UpdateImage profile only: Pinned Python reuses mutable Resource and commits sorted original/current-body diff only when dirty; Go sends explicit owned ordered operations even for [] and returns actual ImageInfo. Cached same-value/noop/Resource mutation/Adapter session and cache parity remain.
- Legacy explicit UpdateImage profile only: Go exact raw values and typed helpers are deliberate input shape choices, not Python descriptor coercion/aliases or server Schema/readonly/reserved/range/enum/property-protection parity. Backend location effects do not imply cloud rollback or atomicity.
- Legacy explicit UpdateImage profile only: Go Name lookup is an explicit native resolver extension, direct safe literal IDs do not fetch first. Accepted raw evidence/atomic ImageInfo and fixedsource/cause-safe ownership differ from Python/native model normalization; native ABI remains unchanged.
- Legacy explicit UpdateImage profile only: Go rejects pointer token edges that Python server strip would rename, including U+001C..U+001F; native UpdateImageProperty interpolates unescaped Names and remains separate. v2.0 draft4 media and broad Resource patch/session behavior are not replicated.
- Owned UpdateImageRecord is an immutable JSON-domain Image profile: private raw baseline and projected view are separate and the returned independent record must be passed to the next call. Caller mutation/same-instance, arbitrary Resource/Munch/subclass, shared descriptor defaults/history and complete Header/URI lifecycle remain SDK-R1; Adapter cache/conditional GET is SDK-C1 and dynamic session/base_path/microversion/exception/conflict policy is SDK-S1, tracked in docs/implementation-plan.md.
- Pinned Resource globally sorts jsonpatch output paths; Go intentionally preserves executable array-index operation order and emits deterministic add/remove/replace without move/copy or minimum-operation promises. This is a separate policy difference from the floating Python jsonpatch dependency. Go RFC JSON diff equates 1 and 1.0 and distinguishes booleans/numbers; Python library scalar serialization and array Python equality can differ. Dirty same-value comparison separately uses recursive Python JSON-domain bool/number equivalence with exact decimals, not arbitrary Python float/object semantics.
- Go raw attributes use encoding/json JSON and sorted map aliases; Python kwargs insertion order, duplicate-key/surrogate/parser spelling, arbitrary Python objects and float rounding are not runtime-equivalent. Fixed-profile direct self/image/connection/_synchronized/base_path/microversion names reject before HTTP even where a supplied Python Image may pack connection/_synchronized as properties; nested explicit properties retain ordinary data. Literal UTF8 ID is validated/escaped once; broad dynamic routing controls are not encoded as attributes.
- Owned v2.1 PATCH accepts actual200..399 with blank Accept and preserves actual receipt plus submitted pending snapshot on accepted processing failures. Python mutates the same instance before commit and may partially overlay before later conversion failure. The Go return policy is deliberate; neither nil error in tolerated invalid syntax nor a partial Record proves server schema/readonly/property-protection success or rollback. No real Python/jsonpatch/cloud mutation was executed.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")

updated = conn.image.update_image("image-id", name="release-image")

same = conn.image.update_image(updated, name=updated.name)
assert same is updated
```

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required literal image ID")
    name := flag.String("name", "release-image", "new image name")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *name); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func printRecord(label string, value *image.ImageRecord, operationErr error) error {
    output := map[string]any{"step": label, "partial": value != nil && operationErr != nil}
    if value != nil {
        output["resource"] = value.Resource
        output["wire"] = value.Wire
        output["receipt_status_code"] = value.StatusCode
        output["receipt_body_bytes"] = len(value.Envelope)
        output["import_methods"] = value.ImportMethods
    }
    if operationErr != nil { output["error"] = operationErr.Error() }
    body, marshalErr := json.MarshalIndent(output, "", "  ")
    if marshalErr != nil { return errors.Join(operationErr, marshalErr) }
    fmt.Println(string(body))
    return operationErr
}

func run(ctx context.Context, cloud, imageID, name string) error {
    if imageID == "" { return fmt.Errorf("-image-id is required") }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }

    updated, err := conn.UpdateImageRecord(ctx, image.ImageRecordUpdateRequest{
        ID: imageID,
        Attributes: map[string]any{"name": name},
    }, image.WithImageRecordHeader("X-Request-Source", "image-record-update-example"))
    if err := printRecord("update by ID", updated, err); err != nil { return err }

    // name has no descriptor conversion: use the returned raw JSON spelling.
    sameName := json.RawMessage(updated.Resource.Body["name"])
    unchanged, err := service.UpdateImageRecord(ctx, image.ImageRecordUpdateRequest{
        Record: updated,
    }, image.WithImageRecordAttributes(map[string]any{"name": sameName}))
    return printRecord("reuse record with same name", unchanged, err)
}
```

## References

- [Update](update.md)
- [native images.Update](v2/images/update.md)
- [Image proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1056-L1068)
- [Proxy._update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L716-L748)
- [Image Records](image-records.md)
- [Install](../docs/install.md)
- [Image Records Model](image_records_model.go)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1251-L1273)
- [Pyproject](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/pyproject.toml#L20)
- [Source packing·flatten](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1207-L1229)
- [Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image.py#L439-L463)
- [Image Record Update Test](image_record_update_test.go)
- [Image Record Update Integration Test](../connectiontest/image_record_update_integration_test.go)
- [Diff Test](../internal/jsonpatch/diff_test.go)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L213-L223)
- [commit/no-op](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1875-L1937)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1394)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
