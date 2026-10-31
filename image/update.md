# Update

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
  Tests: `image/update_core_test.go:TestUpdateImageCoreSortedPropertiesAndEmptyPatch`, `image/update_contracts_test.go:TestImageUpdateContractsPatchAndProperties`, `connectiontest/connection_image_update_test.go:TestConnectionImageUpdateFacadeOwnsPatchAndResponse`.
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
  Tests: `image/update_core_test.go:TestUpdateImageCoreFixedOrderedPatchAndMedia`, `image/update_contracts_test.go:TestImageUpdateContractsCanonicalResponse`, `connectiontest/connection_image_update_test.go:TestConnectionImageUpdateFacadeOwnsPatchAndResponse`.
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
  Tests: `image/image_record_properties_test.go:TestImageRecordPropertiesConcreteFactoriesOwnValuesAndReplacementOrder`, `image/image_record_properties_test.go:TestImageRecordPropertiesSnapshotsCoverDiscoveryAndCommitWithoutRecapture`, `image/image_record_properties_test.go:TestImageRecordPropertiesOptionAndContextGuardsStopLaterCallbacks`, `connectiontest/image_record_properties_integration_test.go:TestConnectionOwnedImagePropertiesSharesSourceResolverAndCommit`, `connectiontest/image_record_properties_integration_test.go:TestConnectionOwnedImagePropertiesRejectsInvalidInvocation`.

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

### `gophercloud:image/v2/images.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithUpdateOptions`

- Generated API accepts concrete UpdateOpts and the SDK owns the upstream builder and Extract invocation. All nine native concrete Patch types retain order and raw zero/false/empty values. Property add/replace includes string value; remove omits it, and property Names retain native unescaped JSON Pointer paths.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateConcretePatchesAndOrder`.
- Nil/allocated-empty UpdateOpts both issue PATCH JSON[], nil tags is null and allocated-empty tags is[]. WithUpdateOptions replaces the whole base set, including nil replacement. Custom inherited Patch map extension remains available; the SDK does not claim it removed the element interface.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateConcretePatchesAndOrder`, `image/v2/images/update_contracts_test.go:TestNativeImageUpdateEmptyReplacementAndInheritedPatchInterface`.
- Nil/error option and nonempty generic body/query/header/argument extensions fail before HTTP with OperationError context and original cause. Custom inherited Patch JSON marshaling error also fails before transport with its original cause; concrete UpdateOpts does not expose arbitrary external UpdateOptsBuilder.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateRejectsRequestExtensionsAndMarshalFailuresBeforeHTTP`.
- Native Update uses service ResourceBase, PATCH and v2.1 JSON-patch content type with default OkCodes200 only. Native microversion/source headers/live auth/context, rejected-response RetryFunc and accepted JSON-decoder retry remain upstream policies. Default rejected201/203/204/404/503 preserve actual status/expected/method/URL/body/header in native error.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateConcretePatchesAndOrder`, `image/v2/images/update_contracts_test.go:TestNativeImageUpdateStrictHTTPContextAndRetryPolicy`.
- Extract preserves native Image field/timestamp/unknown Properties handling, float64-based size and unknown number decoding, last import-method/store-ID header values and original response identity. It does not inject declared raw view, Envelope/status or an owned acknowledgement into the native result.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateExtractsRichImageUnknownPropertiesAndLastHeaders`.
- Valid empty object returns native Image, valid JSONnull may return nil,nil. Body decode failures and invalid known types preserve upstream error plus possible partial Image; OperationError keeps causes inspectable without fabricating owned ResponseError or replacing partial fields.
  Tests: `image/v2/images/update_contracts_test.go:TestNativeImageUpdateExtractorErrorsKeepNativePartialResults`.

Python/Go differences:

- This mapping closes the pinned Gophercloud Update declaration with library-owned outer builder/options/result handling. It does not close Python update_image: native explicit patches always send HTTP, preserve caller order and default strict200 instead of mutable Resource dirty/no-op/sorted-diff/descriptor behavior.
- The SDK accepts concrete UpdateOpts instead of arbitrary external UpdateOptsBuilder. Its inherited Patch element interface remains optional for arbitrary maps; concrete nine Patch types cover the pinned built-in requests. Nil Patch elements retain native unsafe behavior; input slice/map/custom object snapshots and arbitrary callback safety are not added by this facade.
- Native IDs go through ServiceURL, property Names are interpolated without JSON Pointer escaping, schema/enum/range/readonly/policy/permission checks remain server-owned, and configured native client/hooks can change retry/header/response policies. Owned literal/route/body/source guards and ACK contracts are separate APIs.
- Native Image extraction retains its typed/null/date/float64/unknown-key behavior and possible partial pointer plus error or nil,nil. Accepted body Close/decoder/retry policy and response evidence are not promoted to the owned raw ImageRecord guarantees.
- The current/original raw body snapshot foundation preserves sparse presence before descriptor projection for future owned commit. It is not a mutable Resource dirty/header/cache/session state machine, and Python update_image remains unresolved with SDK-R1/C1/S1 work outstanding.
- Ordinary project-owned modify_image is a core-user candidate under the pinned server default; public visibility, owner and location operations can require additional admin policy. Local HTTP tests/builds establish client contracts, not deployed permissions/storage or actual cloud outcomes.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package examples

import (
    "context"
    "encoding/json"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func updateQueuedImageExample(ctx context.Context, svc *image.Service, imageID, ownerID string) (*image.ImageInfo, error) {
    return svc.UpdateImage(ctx, resource.ID(imageID),
        image.WithUpdateImageOpts(image.UpdateImageOpts{
            Headers: map[string]string{"X-Example": "ordered-image-patch"},
        }),
        image.WithUpdateImageHeaders(map[string]string{"X-Trace-Id": "image-update"}),
        image.WithUpdateImageHeader("X-Request-Id", "image-update-example"),
        image.WithUpdateImageChanges(image.ImagePatch{
            Op: "add", Path: "/build", Value: json.RawMessage(`"draft"`),
        }),
        image.WithUpdateImageChange(image.ImagePatch{
            Op: "replace", Path: "/build", Value: json.RawMessage(`"ready"`),
        }),
        image.WithUpdateImageName("release-image"),
        image.WithUpdateImageVisibility("private"),
        image.WithUpdateImageProtected(false),
        image.WithUpdateImageHidden(false),
        image.WithUpdateImageOwner(ownerID),
        image.WithUpdateImageContainerFormat("bare"),
        image.WithUpdateImageDiskFormat("qcow2"),
        image.WithUpdateImageMinDisk(0),
        image.WithUpdateImageMinRAM(0),
        image.WithUpdateImageTags("release", "ready"),
        image.WithUpdateImageField("build~origin", "ci"),
        image.WithUpdateImageFields(map[string]json.RawMessage{
            "build/id": json.RawMessage(`"2026.10"`),
            "legacy": json.RawMessage(`"temporary"`),
        }),
        image.WithUpdateImageRemoveField("legacy"),
    )
}

func setImagePropertiesExample(ctx context.Context, svc *image.Service, imageID string) (*image.ImageInfo, error) {
    return svc.SetImageProperties(ctx, resource.ID(imageID),
        image.WithSetImagePropertiesOpts(image.SetImagePropertiesOpts{
            Headers: map[string]string{"X-Example": "property-upsert"},
        }),
        image.WithSetImagePropertiesHeaders(map[string]string{"X-Trace-Id": "image-properties"}),
        image.WithSetImagePropertiesHeader("X-Request-Id", "image-properties-example"),
        image.WithSetImagePropertiesProperties(map[string]json.RawMessage{
            "team": json.RawMessage(`"platform"`),
            "release": json.RawMessage(`"2026"`),
        }),
        image.WithSetImagePropertiesProperty("release", "2026.10"),
    )
}
```

```python
from openstack.image.v2.image import Image

def update_image_example(conn, image_id):
    cached = Image.existing(id=image_id, name="old-name", team="platform")
    updated = conn.image.update_image(cached, name="release-image", is_protected=False)
    did_update = conn.image.update_image_properties(
        image=updated, meta={"team": "platform"}, min_ram=0,
    )
    return updated, did_update
```

## References

- [ImageInfo](images.md)
- [generated/native Update](v2/images/README.md)
- [Images Images V2](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-images-v2.inc#L559-L682)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L617-L655)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L657-L713)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1406-L1442)
- [parser](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1450-L1551)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1056-L1068)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881-L2011)
- [`update_image_properties`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1139-L1179)
- [native Request](https://github.com/gophercloud/gophercloud/blob/v2.15.0/service_client.go#L144-L163)
- [Update Core Test](update_core_test.go)
- [Update Options Test](update_options_test.go)
- [Update Contracts Test](update_contracts_test.go)
- [Connection Image Update Test](../connectiontest/connection_image_update_test.go)
- [Glance Image Update Test](../internal/cmd/sdkgen/glance_image_update_test.go)
