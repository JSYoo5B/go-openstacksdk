# Image Record Properties

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/update_image_properties`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.SetImageProperties`
- `github.com/JSYoo5B/go-openstacksdk/image.WithSetImagePropertiesOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithSetImagePropertiesHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithSetImagePropertiesHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithSetImagePropertiesProperty`
- `github.com/JSYoo5B/go-openstacksdk/image.WithSetImagePropertiesProperties`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.UpdateImagePropertiesRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.UpdateImagePropertiesRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordPropertiesOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordPropertiesHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordPropertiesHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordProperty`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordProperties`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordPropertiesJSON`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordPropertyMeta`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordPropertiesMetaJSON`

- Legacy typed add-per-key profile only: Concrete Properties raw map emits one v2.1 add/upsert per exact root key sorted by Go string order in a single PATCH. Direct ID has no GET/RMW/same-value comparison or dirty cached Resource dependency.
  Tests: `image/update_core_test.go:TestUpdateImageCoreSortedPropertiesAndEmptyPatch`, `image/update_contracts_test.go:TestImageUpdateContractsPatchAndProperties`, `connection_image_update_test.go:TestConnectionImageUpdateFacadeOwnsPatchAndResponse`.
- Legacy typed add-per-key profile only: Nil/empty Properties sends nonnil [] and receives actual 200 ImageInfo without inferring Python boolean/noop or server save. Empty fields are distinct from missing state; supplied JSON null stays explicit data.
  Tests: `image/update_core_test.go:TestUpdateImageCoreSortedPropertiesAndEmptyPatch`, `image/update_contracts_test.go:TestImageUpdateContractsWireAndReference`, `image/update_contracts_test.go:TestImageUpdateContractsPatchAndProperties`.
- Legacy typed add-per-key profile only: FullOpts replaces wholeconfig; plural Properties replaces whole map; singular Property replaces one raw key. Header singular canonical replace and plural merge retain lastwins without taking source ownership.
  Tests: `image/update_options_test.go:TestUpdateImageOptionsReplacementAppendAndTypedFields`, `image/update_contracts_test.go:TestImageUpdateContractsOptionsAndSource`.
- Legacy typed add-per-key profile only: Raw values copy and preserve exact numeric/null JSON semantics; malformedUTF8/syntax/nil/empty values stop before Name/PATCH. any Property serializes once at factory with originalkeyUTF8 check and retained marshal cause; standardGoJSON is not Python conversion.
  Tests: `image/update_core_test.go:TestUpdateImageCoreCompletePreflight`, `image/update_options_test.go:TestUpdateImageOptionsFactoriesAndNilPresence`, `image/update_options_test.go:TestUpdateImageOptionsReusablePoliciesAndFactoryCauses`, `image/update_contracts_test.go:TestImageUpdateContractsPatchAndProperties`, `image/update_contracts_test.go:TestImageUpdateContractsOptionsAndSource`.
- Legacy typed add-per-key profile only: Field keys are nonemptyUTF8 and Python-strip-edge stable including U+001C..U+001F, interior literal controls retained. Root JSON Pointer escapes tilde/slash once with no URL-percent decode or aliases.
  Tests: `image/update_core_test.go:TestUpdateImageCoreCompletePreflight`, `image/update_contracts_test.go:TestImageUpdateContractsPatchAndProperties`.
- Legacy typed add-per-key profile only: Canonical/root properties keys may be forwarded directly without a forced properties object or readonly/range/type/schema inference. Default custom string-only schema, reserved/readonly permission failures, configured schema and backend limitations remain server-owned.
  Tests: `image/update_options_test.go:TestUpdateImageOptionsReplacementAppendAndTypedFields`, `image/update_contracts_test.go:TestImageUpdateContractsPatchAndProperties`.
- Legacy typed add-per-key profile only: Required direct literal ID and explicit Go Name extension share option-beforelookup/captured resolver/fixed PATCH policy. Private v2.1 Content-Type and Accept installation occurs after Name so source/option header policy reaches lookup unchanged.
  Tests: `image/update_core_test.go:TestUpdateImageCoreExactNameAndSource`, `image/update_contracts_test.go:TestImageUpdateContractsWireAndReference`, `image/update_contracts_test.go:TestImageUpdateContractsOptionsAndSource`.
- Legacy typed add-per-key profile only: Helper maps/raw bytes and initialized callback state are copied before/after callbacks, preserving full replacement/reuse/parallel safety and nil/custom/source/context failures without HTTP or client header mutation.
  Tests: `image/update_options_test.go:TestUpdateImageOptionsCallbacksOwnRetainedState`, `image/update_options_test.go:TestUpdateImageOptionsReusablePoliciesAndFactoryCauses`, `image/update_contracts_test.go:TestImageUpdateContractsOptionsAndSource`.
- Legacy typed add-per-key profile only: Reused atomic ImageInfo retains actual200 raw evidence/25 nullablecanonical fields/passive timestamps and ID/noncanonical Properties; no synthetic IDs/header pseudoJSON/outputschema validation or post-PATCH detail GET.
  Tests: `image/update_core_test.go:TestUpdateImageCoreFixedOrderedPatchAndMedia`, `image/update_contracts_test.go:TestImageUpdateContractsCanonicalResponse`, `connection_image_update_test.go:TestConnectionImageUpdateFacadeOwnsPatchAndResponse`.
- Legacy typed add-per-key profile only: Accepted failure retains whole ResponseError and nil result/Closeonce/noreplay/context and custom causes; live original auth/fixedsource and native advanced header/retry/reauth/redirect policy remain. Python properties seed/coercion/meta/kernel-ramdisk lookup/bool/cache/session semantics remain partial.
  Tests: `image/update_core_test.go:TestUpdateImageCoreAcceptedFailureEvidence`, `image/update_core_test.go:TestUpdateImageCoreNativePrebodyOwnership`, `image/update_contracts_test.go:TestImageUpdateContractsCanonicalResponse`, `image/update_contracts_test.go:TestImageUpdateContractsNativePolicy`, `internal/cmd/sdkgen/glance_image_update_test.go:TestGlanceImageUpdatePreservesNativeAndRegistry`.
- One captured Glance v2 source, provider, route, headers, location, options and sticky outer guard owns the complete helper. Clone SDK private current/original/dirty and passive receipts before callbacks. Reject ambiguous ID+Record or handcrafted private-state-less record. Public view/receipt changes never change the target or property seed.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesSnapshotsCoverDiscoveryAndCommitWithoutRecapture`, `image/image_record_properties_test.go:TestImageRecordPropertiesControlsParserAndIdentityFailBeforePatch`, `image/image_record_properties_test.go:TestImageRecordPropertiesOptionAndContextGuardsStopLaterCallbacks`.
- Source _get_resource(Image,input) performs existing._update({}) before kwargs; it materializes descriptor/canonical backing dict and current location and resets plain ImportMethods to []. Descriptor/location failure precedes all lookup and PATCH. Do not infer or clean pending body.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesRespectInitialProjectionResolverCopyConversionOrder`, `image/image_record_properties_test.go:TestImageRecordPropertiesBooleanUsesCanonicalComparisonAndCachedSeed`.
- Source literal ID/None creates a bare Image, does not fetch it, and properties defaults None; .copy() then fails. Truthy kernel/ramdisk resolution nevertheless precedes this copy failure. A Go literal ID may return a shaped invalid-input error but must not silently add a detail GET.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesCachedCopyShapeBoundaries`, `image/image_record_properties_test.go:TestImageRecordPropertiesRespectInitialProjectionResolverCopyConversionOrder`.
- Process kwargs in insertion order. Truthy kernel/ramdisk transforms to kernel_id/ramdisk_id before property-map assignment. Collisions with directly supplied *_id are last-value wins at the original first key position. Raw ordered JSON object supports Source order; Go map inputs have documented deterministic sorted order.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesResolveOrderedAliasesAfterCompleteLists`, `image/image_record_properties_test.go:TestImageRecordPropertiesConcreteFactoriesOwnValuesAndReplacementOrder`.
- Only truthy v with original key kernel/ramdisk calls resolver; false/null/zero/empty strings/arrays/objects keep the original key and later ordinary conversion. Meta kernel/ramdisk values do not trigger resolver. JSON-domain PythonTruthy/PythonString supply ordinary truthiness/pattern values.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesFalsyAliasesAndRawMetaSkipResolution`, `image/image_record_properties_test.go:TestImageRecordPropertiesResolveOrderedAliasesAfterCompleteLists`.
- Each resolver call separately consumes every owned image list page with default empty query and no hidden-image retry. Cloud list_images filters each status.lower()==deleted during list consumption; status must be a string. A later page/status/projection/IO/source error is terminal even if an earlier row would match.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesResolveOrderedAliasesAfterCompleteLists`, `image/image_record_properties_test.go:TestImageRecordPropertiesResolverMissingAndLateFailuresAreTerminal`, `image/image_record_properties_test.go:TestImageRecordPropertiesCompleteListFailuresPrecedeNumericAliasSelection`.
- After complete list collection, cloudfilter.Select(rows,pattern,nil,guard) matches each row exact canonical backing ID/name or Python fnmatch, preserves source order and duplicates. Choose the first matched row; no ambiguity error, direct detail GET, exact-name query or Image.find hidden fallback. Missing selection gives raw null.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesResolveOrderedAliasesAfterCompleteLists`, `image/image_record_properties_test.go:TestImageRecordPropertiesResolverMissingAndLateFailuresAreTerminal`.
- Copy the complete private current properties dictionary before overlays; do not use flattened Wire, public Resource edits or only newly supplied values. Null/missing/scalars fail .copy(); JSON [] permits empty False when no changed insertion, changed string-key insertion fails and nonempty arrays fail later ** expansion. Preserve Source order of resolver-before-copy errors.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesCachedCopyShapeBoundaries`, `image/image_record_properties_test.go:TestImageRecordPropertiesRetainCachedKeysAndDelegateDeclaredPromotion`.
- For min_disk/min_ram/size/virtual_size use Python int(v), not Resource descriptor conversion. Booleans become 1/0, arbitrary-precision integer tokens retain value, finite JSON float64 values truncate, signed/space/underscore/Unicode decimal strings parse. Null/containers/invalid strings/nonfinite overflow fail before PATCH. Reuse PythonIntegerString and numeric portion of DescriptorIntegerJSON with explicit bool/error handling.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesIntegerConversionIsHelperInt`.
- is_protected/protected/tags remain raw JSON. Every other nonnull kwargs value uses whole-value PythonString; null stays null. Strings, boolean spellings, nested ordered container repr, integer and float spelling follow the existing documented JSON/float/Unicode representation bounds. Go any factories capture standard Go JSON once.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesRawValuesPythonStringsAndMetaLast`.
- Only after every kwargs conversion finishes does raw meta overlay overwrite keys; even an invalid kwargs value overwritten by meta fails first. Absent/null/empty meta defaults {} in the owned object profile. Truthy nonmapping Source update forms/custom iterables are explicit dynamic-object bounds, not fabricated mappings.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesRawValuesPythonStringsAndMetaLast`, `image/image_record_properties_test.go:TestImageRecordPropertiesRespectInitialProjectionResolverCopyConversionOrder`, `image/image_record_properties_test.go:TestImageRecordPropertiesConcreteFactoriesOwnValuesAndReplacementOrder`.
- Compare against freshly materialized canonical descriptor-backed dict.get(k,None), with recursive EqualPythonJSON bool/numeric equivalence. Do not compare raw component data, remote aliases through __getitem__, or nested custom cached properties. Absent custom key compares as null: custom foo=None can leave cached foo unchanged.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesBooleanUsesCanonicalComparisonAndCachedSeed`.
- Begin with all cached custom properties and add only unequal normalized kwargs/meta values. Cached keys matching declared canonical/wire aliases become ordinary Body fields when delegated; unknown keys repack into properties. A supplied properties attribute and other retained custom keys must use existing Source packing precedence, not a generic deep merge or add-per-field PATCH.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesRetainCachedKeysAndDelegateDeclaredPromotion`.
- If prepared img_props is falsy return False with no Update/HTTP. Otherwise call full update and return True only after success. True can mean no PATCH when components remain clean; pending unrelated body state can be committed; a dirty state can submit an empty [] diff. Do not reinterpret boolean as server save or actual request sent.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesBooleanUsesCanonicalComparisonAndCachedSeed`, `image/image_record_properties_test.go:TestImageRecordPropertiesBooleanControlsPendingCommitAndRetargeting`.
- Call shared owned update preparation/commit, discarding id dirtiness but retaining current ID and any other pending dirty state. Raw meta id can retarget when non-ID changes remain. Source image/self/resource_type/value kwargs collisions fail before PATCH; base_path route override, microversion/constructor/runtime controls stay explicit fixed-route Go profile bounds and must fail locally rather than become arbitrary URL/header changes.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesControlsParserAndIdentityFailBeforePatch`, `image/image_record_properties_test.go:TestImageRecordPropertiesBooleanControlsPendingCommitAndRetargeting`.
- Reuse current/original component diff, RFC6902 execution ordering, wire alias transposition, property flattening, exact 200..399 accepted profile, private v2.1 Content-Type and empty Accept. Source Resource translation rejects >=400. No detail GET or polling. Accepted object overlay cleans baseline; invalid JSON syntax preserves pending components; valid nonobject/descriptor failures retain real response evidence.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesSharedAcceptedTranslationAndPendingResponses`, `image/image_record_properties_test.go:TestImageRecordPropertiesBooleanControlsPendingCommitAndRetargeting`, `image/image_record_update_test.go:TestImageRecordUpdatePropertyReplacementFlatteningAndPointerEscapes`, `image/image_record_update_test.go:TestImageRecordUpdateValidResponseCleansAndInvalidSyntaxKeepsPending`, `image/image_record_update_test.go:TestImageRecordUpdateNoopUsesRawPresenceAndPythonEquality`.
- Retain native HTTP, marshal/parser/read/Close and context/source causes with phase identity and actual accepted receipt; close response exactly once and never replay a PATCH merely to recover decoding/evidence. Lookups can have happened before local later failure. Immutable Go result may return prepared/partial record with error, unlike mutating Python exceptions.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesAcceptedHandlingErrorsReturnFalsePartialReceiptWithoutReplay`, `image/image_record_properties_test.go:TestImageRecordPropertiesResolverMissingAndLateFailuresAreTerminal`, `image/image_record_properties_test.go:TestImageRecordPropertiesNativeRejectionsAndRetryOwnership`.
- Concrete With functions provide raw kwargs, raw meta, headers and whole opts without caller builder interfaces. Factory/callback/full-opts replacement, last wins, copied slices/maps/raw bytes, cancellation, source mutation and live original provider authentication cover Service and Connection facade consistently. Kernel/ramdisk resolution is implemented inside this layer using owned listing/filter reuse, not a developer-supplied resolver builder.
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesConcreteFactoriesOwnValuesAndReplacementOrder`, `image/image_record_properties_test.go:TestImageRecordPropertiesSnapshotsCoverDiscoveryAndCommitWithoutRecapture`, `image/image_record_properties_test.go:TestImageRecordPropertiesOptionAndContextGuardsStopLaterCallbacks`, `image_record_properties_integration_test.go:TestConnectionOwnedImagePropertiesSharesSourceResolverAndCommit`, `image_record_properties_integration_test.go:TestConnectionOwnedImagePropertiesRejectsInvalidInvocation`.

Python/Go differences:

- Principal helper semantics: the pinned Source copies cached properties and returns bool(prepared img_props) after update succeeds, so True can occur without PATCH and False can leave pending unrelated state uncommitted. Owned Go preserves that success boolean as Updated, with an independent Record; Updated is false on an error even when accepted partial PATCH evidence is returned. Literal ID does not fetch a property seed and fails its missing/null .copy() boundary, while truthy kernel/ramdisk full-list resolution occurs before that copy failure. SetImageProperties remains the separate legacy add-per-key profile and implements none of this public helper graph.
- Legacy typed add-per-key profile only: Pinned Python reuses mutable Resource and commits sorted original/current-body diff only when dirty; Go sends explicit owned ordered operations even for [] and returns actual ImageInfo. Cached same-value/noop/Resource mutation/Adapter session and cache parity remain.
- Legacy typed add-per-key profile only: Go exact raw values and typed helpers are deliberate input shape choices, not Python descriptor coercion/aliases or server Schema/readonly/reserved/range/enum/property-protection parity. Backend location effects do not imply cloud rollback or atomicity.
- Legacy typed add-per-key profile only: Go Name lookup is an explicit native resolver extension, direct safe literal IDs do not fetch first. Accepted raw evidence/atomic ImageInfo and fixedsource/cause-safe ownership differ from Python/native model normalization; native ABI remains unchanged.
- Legacy typed add-per-key profile only: Go rejects pointer token edges that Python server strip would rename, including U+001C..U+001F; native UpdateImageProperty interpolates unescaped Names and remains separate. v2.0 draft4 media and broad Resource patch/session behavior are not replicated.
- Legacy typed add-per-key profile only: Python update_image_properties copies cached property seed, kwargs int/raw/string conversion and meta precedence, truthy kernel/ramdisk Connection lookup and bool return (which can be true without PATCH); Go SetImageProperties does none of these.
- The owned JSON-domain helper snapshots SDK-produced private current/original/dirty state and passive receipts; public view and receipt edits do not become inputs. It performs current65 descriptor projection/location and ImportMethods reset, cached dictionary/list shape checks, ordered kwargs conversion/raw meta, canonical recursive Python comparison and shared update/commit. Python reuses mutable Resource/Munch/subclasses and arbitrary objects; original/header/URI/default/backing-dict/constructor lifecycle remains SDK-R1, Adapter caching/invalidation remains SDK-C1, and dynamic session/microversion/class/exception/metrics/transport behavior remains SDK-S1 in docs/implementation-plan.md. This is go_mapping rather than full supported runtime equivalence.
- Raw JSON kwargs preserve insertion order and first-position/last-value duplicate semantics; Go map helpers use deterministic sorted keys and any factories capture standard Go JSON once. int(v) uses explicit bool1/0, arbitrary-precision integer tokens, finite float64 truncation and PythonIntegerString rather than descriptor fallback-zero; PythonString follows the existing finite JSON/float/Unicode representation profile. Truthy arbitrary iterable meta and non-JSON custom objects are not accepted. Identity and resolver status strings enforce complete UTF8 and paired escaped Unicode, unlike Python's broader string domain.
- Each truthy kernel/ramdisk alias consumes every fixed Glance v2 list page, filters string status.lower()==deleted, then uses the first exact-ID/name-or-fnmatch match with no ambiguity error or missing-match exception. Missing returns raw null; late page/status/descriptor/read/Close/source errors remain terminal before copy/commit. Source cloud helpers also have dynamic version/session/cache/normalization branches outside this fixed-v2 profile. No target GET, Image.find hidden retry, developer resolver builder, authorization/status preflight, polling or cleanup is added.
- Source kwargs bound-argument collisions and constructor/runtime controls are finite Go local errors; fixed routing rejects base_path, microversion, connection and similar controls instead of consuming them as arbitrary session/URL behavior. Raw meta id may retarget the shared update when other changes remain, with strict identity validation before equality and id dirtiness removed. The existing current/original RFC6902 engine, wire aliases/property packing, private v2.1 Content-Type and empty Accept remain shared with UpdateImageRecord; exact Python jsonpatch library byte order and arbitrary control/descriptor runtimes remain documented differences.
- Accepted200..399 response object translation cleans the shared baseline, while tolerated invalid JSON syntax preserves pending state; shape/conversion/physical handling failures retain real receipts and causes without an automatic PATCH replay. Lookup requests can have occurred before a later local error, and Go may return a partial submitted Record where Python raises after mutating its Resource. Native400..599 proof, live provider auth and original retry/reauth hooks are preserved. Ordinary property modification is core-user default ADMIN_OR_PROJECT_MEMBER; public visibility and location removal are separate admin field branches governed by the server. No Python SDK runtime, authenticated cloud or backend mutation was executed, and public create_image's complete mandatory graph remains unresolved.

### `python:cloud/_image/ImageCloudMixin/update_image_properties`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.UpdateCloudImageProperties`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordPropertiesOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordProperty`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordProperties`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordPropertyMeta`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordPropertiesMetaJSON`

- UpdateCloudImageProperties selects image or name_or_id: a Record wins over both string forms, otherwise a nonempty literal ID wins over NameOrID, and NameOrID is passed on as a literal identity without any name lookup or HTTP.
  Tests: `image/image_record_cloud_properties_test.go:TestCloudImagePropertiesSelectsImageOrNameOrIDAndDelegates`.
- The selected image is delegated unchanged to the owned UpdateImagePropertiesRecord helper with the same options, returning its Updated/Record result: cached property copy, kernel/ramdisk lookup, conversions, raw meta precedence and conditional automatic PATCH. Literal identities keep the lower helper's missing cached-properties failure before HTTP, and errors are renamed to the Cloud operation without nesting.
  Tests: `image/image_record_cloud_properties_test.go:TestCloudImagePropertiesSelectsImageOrNameOrIDAndDelegates`, `image/image_record_properties_test.go:TestImageRecordPropertiesBooleanUsesCanonicalComparisonAndCachedSeed`, `image/image_record_properties_test.go:TestImageRecordPropertiesRawValuesPythonStringsAndMetaLast`, `image/image_record_properties_test.go:TestImageRecordPropertiesBooleanControlsPendingCommitAndRetargeting`.

Python/Go differences:

- Python's image argument may be an Image object or a string; Go separates Record and ID fields and keeps NameOrID for name_or_id. Python truthiness of an empty string is represented by empty Go strings.
- Properties and meta use the existing concrete ImageRecordPropertiesOption values instead of Python **properties and meta dictionaries; the lower helper's documented Go differences apply unchanged.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
record = conn.image.get_image("image-id")
updated = conn.image.update_image_properties(
    record,
    team="platform",
    min_ram="512",
    meta={"example_enabled": False},
)
print(updated)
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
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required literal image ID")
    properties := flag.String("properties", `{"team":"platform","min_ram":"512"}`, "ordered JSON keyword object")
    meta := flag.String("meta", `{"example_enabled":false}`, "raw JSON overlay object")
    viaService := flag.Bool("service", false, "call image.Service instead of Connection")
    flag.Parse()
    if *imageID == "" { log.Fatal("-image-id is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, json.RawMessage(*properties), json.RawMessage(*meta), *viaService); err != nil {
        var native gophercloud.ErrUnexpectedResponseCode
        if errors.As(err, &native) {
            fmt.Fprintf(os.Stderr, "native HTTP %d, response bytes=%d\n", native.Actual, len(native.Body))
        }
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "accepted HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, imageID string, properties, meta json.RawMessage, viaService bool) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    record, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
    if err != nil { return err }
    input := image.ImageRecordPropertiesRequest{Record: record}
    options := []image.ImageRecordPropertiesOption{
        image.WithImageRecordPropertiesOpts(image.ImageRecordPropertiesOpts{
            Headers: map[string]string{"X-Request-Source": "image-record-properties-example"},
            Properties: properties,
            Meta: meta,
        }),
        image.WithImageRecordPropertiesHeader("X-SDK-Example", "owned-properties"),
    }
    var result *image.ImageRecordPropertiesResult
    if viaService {
        result, err = service.UpdateImagePropertiesRecord(ctx, input, options...)
    } else {
        result, err = conn.UpdateImagePropertiesRecord(ctx, input, options...)
    }
    value := map[string]any{"partial": result != nil && err != nil}
    if result != nil {
        value["updated"] = result.Updated
        if result.Record != nil {
            value["record"] = map[string]any{
                "resource": result.Record.Resource,
                "wire": result.Record.Wire,
                "receipt_status_code": result.Record.StatusCode,
                "receipt_body_bytes": len(result.Record.Envelope),
                "import_methods": result.Record.ImportMethods,
            }
        }
    }
    if err != nil { value["error"] = err.Error() }
    body, marshalErr := json.MarshalIndent(value, "", "  ")
    if marshalErr != nil { return errors.Join(err, marshalErr) }
    fmt.Println(string(body))
    return err
}
```

## References

- [SetImageProperties](update.md)
- [Install](../docs/install.md)
- [Image Record Update](image-record-update.md)
- [UpdateImageRecord](image-record-update.md)
- [Policy](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L213-L229)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L80-L93)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L94-L103)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L159-L172)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1139-L1179)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L159-L176)
- [ Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L84-L120)
- [ Image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_image.py#L221-L246)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
