# Image Record Create

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/create_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.CreateAndImport`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCreateImportOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCreateImportMetadata`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCreateImportStage`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCreateImportImport`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCreateImportStageOptions`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCreateImportImportOptions`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCreateImportWait`
- `github.com/JSYoo5B/go-openstacksdk/image.WithoutCreateImportWait`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imagedata.API.PrepareStageOptions`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport.API.PrepareImportOptions`
- `github.com/JSYoo5B/go-openstacksdk.Connection.CreateImageRecord`
- `github.com/JSYoo5B/go-openstacksdk.WithImageCreatePolicy`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.CreateImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateContainer`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateMD5`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateSHA256`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateDiskFormat`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateContainerFormat`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateTags`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateDisableVendorAgent`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateAllowDuplicates`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateWait`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateTimeout`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateValidateChecksum`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateUseImport`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateSize`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateMeta`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateAttributes`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateImportOptions`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateSwiftHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateSwiftHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.ImageRecordCreateBytes`
- `github.com/JSYoo5B/go-openstacksdk/image.ImageRecordCreateText`
- `github.com/JSYoo5B/go-openstacksdk/image.ImageRecordCreateReader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageCreatePolicyOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageCreatePolicyFormat`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageCreatePolicyTasks`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageCreatePolicyVendorAgent`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageCreatePolicyObjectStoreEnabled`
- `github.com/JSYoo5B/go-openstacksdk/image.PrepareImageCreatePolicy`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.CreateImageImportObject`

- CreateAndImport defaults to qcow2/bare/private and glance-direct with waiting disabled. Direct execution performs actual creation201, staging204, post-stage metadata200 and import202 without an initial image lookup, duplicate/discovery request or synthetic active result. Phase responses and fixed ImageID are retained separately.
  Tests: `image/create_import_contracts_test.go:TestCreateAndImportDefaultDirectSequenceAndEvidence`, `image/create_import_core_test.go:TestCreateImportFixedCanonicalRouteAndIndependentActualEvidence`.
- Concrete metadata, stage, import and wait policies, source/reference validity, reader requirements and protected namespaces are checked before the first create POST or input read. Typed nil, invalid source/context, unknown fresh workflow method, conflicting selector and invalid field/header/size/wait policies stop locally.
  Tests: `image/create_import_contracts_test.go:TestCreateAndImportCompletePreflight`, `image/create_import_options_test.go:TestCreateImportWaitOptionsHaveConcretePreflightPolicy`, `image/create_import_core_test.go:TestCreateImportPreflightValidatesEveryPhaseBeforeCreationOrReading`.
- PrepareStageOptions validates the selected source, applies a caller callback once and returns independently owned concrete size/header settings without HTTP or reader use. Retained callback configuration cannot later alter the prepared snapshot; invalid configured source headers are rejected before callbacks.
  Tests: `image/v2/imagedata/stage_prepare_test.go:TestPrepareStageOptionsFreezesOnceWithoutRequests`.
- PrepareImportOptions is a local-only source-aware snapshot seam. It preserves owned JSON extension numbers and store slices after one callback application, rejects conflicting configured store headers and selected plural/all-store controls, and permits an explicit singular-store overlay without changing the shared source.
  Tests: `image/v2/imageimport/import_prepare_test.go:TestPrepareImportOptionsFreezesJSONAndSourceConflictsWithoutRequests`.
- The original lowercase creation id must be a nonempty safe route segment and becomes the fixed workflow ImageID. Typed case aliases, foreign Location and later response IDs do not retarget staging, import or wait. Direct creation also requires exact canonical queued status before consuming Data.
  Tests: `image/create_import_contracts_test.go:TestCreateAndImportCanonicalIdentityAndPostStageFormats`, `image/create_import_contracts_test.go:TestCreateAndImportDefaultDirectSequenceAndEvidence`, `image/create_import_core_test.go:TestCreateImportFixedCanonicalRouteAndIndependentActualEvidence`.
- Direct import reads required lowercase container_format and disk_format from the actual post-stage metadata body, independently of returned typed case aliases or stale creation formats. Missing/null/empty canonical decision fields preserve the actual accepted response and stop before import.
  Tests: `image/create_import_contracts_test.go:TestCreateAndImportCanonicalIdentityAndPostStageFormats`, `image/create_import_core_test.go:TestCreateImportCanonicalFailureRetainsAcceptedCreationAndStage`.
- The workflow metadata policy reuses the existing upload builder, retaining optional false and explicit min_disk/min_ram zero and root JSON extensions without replacing protected core fields. Full and namespace helpers snapshot pointers, slices, maps and JSON. Deferred leaf callback batches run once during preparation and can be cleared by later concrete helpers; reusable policies remain independent. Invalid custom JSON retains the original MarshalerError and caller cause alongside ErrInvalidOption.
  Tests: `image/create_import_contracts_test.go:TestCreateAndImportPreparedSnapshotsAndCallbacksOnce`, `image/create_import_options_test.go:TestCreateImportOptionsOwnReplacementAndReusableSnapshots`, `image/create_import_options_test.go:TestCreateImportDeferredPhaseCallbacksAreAppliedOnceAndCanBeCleared`, `image/create_import_options_test.go:TestCreateImportMetadataReusesUploadBuilderAndPreservesZeroFields`, `image/create_import_options_test.go:TestCreateImportMetadataRetainsCustomJSONFailureCause`.
- Fresh web-download and glance-download workflows create metadata and submit import with no reader or staging phase. Required URI/remote inputs and unused stage controls are guarded before creation. Remote formats come from the actual creation response, plural stores retain repeats and explicit false flags, and singular Store remains header-only. Copy-image and unknown fresh methods are unsupported while the existing leaf passthrough remains separate.
  Tests: `image/create_import_contracts_test.go:TestCreateAndImportRemoteModesAndSelectors`, `image/create_import_contracts_test.go:TestCreateAndImportCompletePreflight`.
- Direct data is borrowed at its current cursor without input Close/Seek/size inspection/buffering or semantic replay. Stage headers and metadata POST-only headers are kept in their selected phases, while size is header-only and explicit zero can be retained. The binary provider replay policy stays distinct from configured JSON requests.
  Tests: `image/create_import_contracts_test.go:TestCreateAndImportDefaultDirectSequenceAndEvidence`, `image/create_import_contracts_test.go:TestCreateAndImportPreparedSnapshotsAndCallbacksOnce`, `image/create_import_contracts_test.go:TestCreateAndImportLiveProviderSourceBoundariesAndNativeIsolation`.
- Before accepted creation201 the result is nil; accepted creation read/decode/invariant errors retain Created evidence. Later stage/read/followup/import failures preserve prior actual201/204/200/202 proof and original errors without automatic delete, recreation, recovery GET or fabricated model. Response bodies/headers and models from different phases are independently owned.
  Tests: `image/create_import_contracts_test.go:TestCreateAndImportPartialProofAndTerminalFailures`, `image/create_import_contracts_test.go:TestCreateAndImportDefaultDirectSequenceAndEvidence`, `image/create_import_core_test.go:TestCreateImportCanonicalFailureRetainsAcceptedCreationAndStage`.
- Optional active waiting uses concrete SDK-owned policy: nil Wait disables, selected nil timeout/interval use five minutes/two seconds, zero timeout removes the SDK deadline, nil failures select killed/deleted and nonnil empty failures disable them. The parent context bounds all phases; only an actual successful wait populates Ready and prior accepted phase proof remains on failure.
  Tests: `image/create_import_options_test.go:TestCreateImportWaitOptionsHaveConcretePreflightPolicy`, `image/create_import_contracts_test.go:TestCreateAndImportExplicitWaitAndFailureRetention`, `image/create_import_core_test.go:TestCreateImportOptionalWaitUsesPreparedDefaultBudgetAndFixedID`.
- The workflow captures one original-provider client selection and validates its source between phases and waiter polls. Canonical decisions and actual native header/model extraction stay separate from raw evidence; native Create/Get/Stage/Upload and existing Service.Upload remain independently callable without changing generated native declarations or inventory. Generation checks required native CreateOpts/Image field types and canonical JSON tags, rejects drift, and documents the upper workflow independently of native image API declarations.
  Tests: `image/create_import_contracts_test.go:TestCreateAndImportLiveProviderSourceBoundariesAndNativeIsolation`, `image/create_import_core_test.go:TestCreateImportFixedCanonicalRouteAndIndependentActualEvidence`, `internal/cmd/sdkgen/glance_create_import_test.go:TestGlanceCreateImportKeepsNativeImageAPIAndDocumentsUpperService`, `internal/cmd/sdkgen/glance_create_import_test.go:TestGlanceCreateImportRejectsNativeMetadataAndResponseFieldDrift`.
- Whole modern CreateImageRecord defaults container only when omitted to images, falsey formal disk_format to frozen cloud image_format (qcow2 by default), falsey container_format to bare and inserts tags only when truthy; it does not inject private visibility. Connection merged configuration and explicit concrete policy snapshots preserve raw presence/nulls and lazy Task configuration.
  Tests: `image/image_record_create_workflow_test.go:TestImageRecordCreateWorkflowDefaultsUsePythonTruthiness`, `image/image_record_create_policy_test.go:TestImageCreatePolicyDefaultsRetainExplicitRawAndServicePresence`, `connection_image_create_policy_test.go:TestCloudImageConfigurationMergesDefaultsAndFreezesSourceValues`.
- Tagged None/empty/nonempty bytes, UTF-8 text and borrowed Reader preserve Source truthiness versus presence. Truthy filename+data rejects before hashes/discovery; both falsey invoke the actual name/file helper including leading-dot splitext and the pinned second-branch isfile(name) typo. Filename plus present falsey data can fail later in Swift after container establishment.
  Tests: `image/image_record_create_data_test.go:TestImageRecordCreateDataFactoriesPreserveNoneEmptyAndTruthiness`, `image/image_record_create_data_test.go:TestImageRecordCreateDataOwnsBytesAndBorrowsReaderWithoutCaptureIO`, `image/image_record_create_workflow_test.go:TestImageRecordCreatePrefaceFilenameInferencePreservesSourceBugAndDotfiles`, `image/image_record_create_workflow_test.go:TestImageRecordCreatePrefaceTruthyFilenameAndDataFailBeforeHashOrDiscovery`, `image/image_record_create_swift_test.go:TestImageRecordCreateTaskWholeContainerEnsureAndLateEmptyDataConflict`.
- Preface checksum validation permits truthy bytes/file and rejects truthy text/Reader before IO or duplicate lookup. Both MD5/SHA256 are calculated only when neither supplied hash is truthy; supplied one hash prevents computation of the other. File hashing uses guarded chunks and closes its own handle before later phases; empty falsey bytes do not compute hashes.
  Tests: `image/image_record_create_workflow_test.go:TestImageRecordCreatePrefaceChecksumValidationComputesBothOrNeither`, `image/image_record_create_workflow_test.go:TestImageRecordCreatePrefaceChecksumRejectsTextAndBorrowedReaderWithoutReading`, `image/image_record_create_workflow_test.go:TestImageRecordCreatePrefaceConfiguredFormatTypeIsCheckedOnlyDuringInference`.
- Unless allow_duplicates is truthy, reuse runs the complete prepared GET-first/visible-list/hidden-list find graph, including exact name/raw ID, duplicate and terminal error rules. Modern owner_specified.openstack hash key presence overrides legacy shade fallback even when null; at least one expected hash must be truthy, each supplied truthy hash must match Python JSON equality, and reuse returns before vendor/property/upload/Task availability errors.
  Tests: `image/image_record_create_workflow_test.go:TestImageRecordCreateWorkflowDuplicateHashesRespectModernPresence`, `image/image_record_create_workflow_test.go:TestImageRecordCreateWorkflowReuseStopsBeforeVendorPropertiesAndUploadConflicts`, `image/image_record_create_workflow_test.go:TestImageRecordCreateWorkflowDiscoveryTraversesVisibleThenHidden`, `image/image_record_create_workflow_test.go:TestImageRecordCreateWorkflowDuplicateDiscoveryFailureDoesNotCreate`, `image/image_record_find_test.go:TestFindImageRecordRejectedOwnershipFaultsAreTerminal`.
- After reuse, raw kwargs are overwritten by selected frozen vendor mapping/pair-list values; legacy properties is popped as a mapping, synthesized raw MD5/SHA256 (falsey becomes empty string) and object container/name overwrite it, then its contents override kwargs. Source/vendor input maps stay unchanged, malformed selected shapes fail at this phase, and explicit vendor disable avoids those late errors.
  Tests: `image/image_record_create_workflow_test.go:TestImageRecordCreateWorkflowMetadataConversionVendorPropertiesAndMetaOrder`, `image/image_record_create_workflow_test.go:TestImageRecordCreateWorkflowVendorFlagAndPairListFollowSourceUpdate`, `image/image_record_create_workflow_test.go:TestImageRecordCreateWorkflowLegacyPropertiesShapeFailsAfterDiscovery`, `image/image_record_create_policy_test.go:TestImageCreatePolicyRawVendorOverrideDefersShapeChecksAndTypedHelperClearsIt`.
- Whole metadata and non-Task upload reuse exact property int/string/raw/None conversion: integer keys including min_disk/min_ram/size/virtual_size convert before Meta and None fails; other values stringify with Python JSON policy except preserved tags/protection/null. Meta overrides without conversion, formal name is forced last, and raw constructor hooks can override the resulting attributes with no final format/default recheck.
  Tests: `image/image_record_create_workflow_test.go:TestImageRecordCreateWorkflowIntegerConversionPrecedesMetaOverride`, `image/image_record_create_workflow_test.go:TestImageRecordCreateWorkflowMetadataConversionVendorPropertiesAndMetaOrder`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadChecksumUsesFlatMetaBeforeConstructorHook`.
- Only truthy filename, tagged data or import_method selects upload. use_import/URI/remote/store flags alone remain metadata-only and do not trigger Task configuration, store validation or timeout. Selected upload alone warns on mere is_public key presence, removes it and sets root public/private by Python truthiness; configured Task plus explicit truthy use_import rejects before Swift.
  Tests: `image/image_record_create_workflow_test.go:TestImageRecordCreateWorkflowFalseyDataKeepsMetadataOnlyBranch`, `image/image_record_create_workflow_test.go:TestImageRecordCreateWorkflowIsPublicWarningBelongsToSelectedUploadOnly`, `image/image_record_create_swift_test.go:TestImageRecordCreateTaskWholeAvailabilityIsLazyAndRawConfigIsBool`.
- Metadata-only uses the same private all64-Body raw Image constructor/create engine as owned Upload, including explicit id, aliases, properties packing, __conflicting_attrs, captured location, pending/original/dirty state and independent sparse/opaque accepted POST proof. It imposes no returned ID, status or formats gate and performs no binary/import/Task/wait/final GET.
  Tests: `image/image_record_create_workflow_test.go:TestImageRecordCreateWorkflowMetadataOnlyAcceptsSparseAndOpaqueSuccess`, `image/image_record_upload_test.go:TestImageRecordUploadAllDeclaredFieldsSubmitRawWireWithoutProjectionDefaults`, `image/image_record_upload_test.go:TestImageRecordUploadConflictingAttributesAndPropertiesFollowSourceConstructorOrder`, `image/image_record_upload_test.go:TestImageRecordUploadInvalidMetadataSyntaxPreservesConstructorDirtyIDAndLaterCommit`, `image/image_record_upload_test.go:TestImageRecordUploadMetadataTranslationFailuresKeepActualCreatedEvidence`.
- After duplicate reuse and non-Task store controls, a selected filename opens before property conversion or metadata POST whenever data is falsey. Metadata/create or unsupported-method failures remain outside upload cleanup. SDK handles close once on every exit and are removed from retained Record.data; Task retains its separate file/SLO path and ignores Meta and Size.
  Tests: `image/image_record_create_upload_test.go:TestImageRecordCreateUploadFilenameOpenedBeforeConversionAndNotRetainedClosed`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadStoreConflictPrecedesFileOpenAndMetadataConversion`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadImportMethodRejectionAfterCreateHasNoCleanup`, `image/image_record_create_swift_test.go:TestImageRecordCreateTaskWholeSuccessMergesUnconvertedPropertiesAndFinallyDeletesObject`.
- Non-Task direct upload assigns data unconditionally including None, then performs one opaque file PUT without a queued gate or response Body overlay. Explicit Size accepts Source Python integers including bool with signed arbitrary-precision decimal/True/False headers; non-integer JSON fails inside cleanup. Omitted/null size infers total length only for SDK filename or borrowed file-like Reader and restores its cursor; Bytes/Text never gain Go-only seek inference. Provider binary retry/reauth/backoff/redirect replay is disabled.
  Tests: `image/image_record_create_upload_test.go:TestImageRecordCreateUploadDirectAcceptsOpaqueResponsesWithoutQueuedGateWaitOrRefresh`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadSizePreservesPythonIntegersAndRejectsOtherTypesInsideCleanup`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadReaderSizeRestoresCursorAndBorrowedLifetime`, `image/image_record_stage_test.go:TestImageRecordStageSizeInferenceKeepsSeekFailuresAndESPIPEUnknown`, `image/image_record_stage_test.go:TestImageRecordStageBinaryRejectionsNeverReplayReauthenticateOrBackoff`.
- Non-Task create ignores Wait and Timeout, returning the created/uploaded/imported private image without active waiting or mandatory final GET. Only the independently selected legacy checksum comparison adds one metadata fetch. Connection uses the same concrete whole operation and source without adding lifecycle requests.
  Tests: `image/image_record_create_upload_test.go:TestImageRecordCreateUploadDirectAcceptsOpaqueResponsesWithoutQueuedGateWaitOrRefresh`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadImportMethodSpecificForwardingAndNoBinaryRead`, `connectiontest/image_record_create_integration_test.go:TestConnectionOwnedImageCreateComposesPolicyAndKeepsSwiftLazy`.
- Import capability is checked after accepted metadata creation and unconditional data assignment, using exact creation OpenStack-image-import-methods membership with no trimming/case normalization. Unsupported methods retain the created record/receipt, consume no data and perform no stage/import/cleanup; method defaults to glance-direct only after import selection.
  Tests: `image/image_record_create_upload_test.go:TestImageRecordCreateUploadImportMethodRejectionAfterCreateHasNoCleanup`.
- Truthy plural Stores/all_stores/all_stores_must_succeed imply use_import; all_stores with truthy plural stores rejects before file open/metadata creation. Nonnil Stores, including an empty slice, suppress both all-store flags before import compilation, while an empty body stores list may be omitted. Store identity snapshots/order/duplicates and fixed compatibility header policy reuse the owned Import compiler.
  Tests: `image/image_record_create_upload_test.go:TestImageRecordCreateUploadStoreImplicationAndEmptyStorePresence`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadStoreConflictPrecedesFileOpenAndMetadataConversion`, `image/image_record_create_options_test.go:TestImageRecordCreateOptionsMergeReplaceAndEmptyStorePresenceRemainDistinct`.
- glance-direct stages raw data once without a queued gate or mandatory metadata GET. Its opaque accepted response translates headers only, preserving raw Body/dirty/data while exposing the stage receipt. Subsequent Proxy-style empty _update resets ImportMethods and projects the same private image before import formats/controls are checked.
  Tests: `image/image_record_create_upload_test.go:TestImageRecordCreateUploadStoreImplicationAndEmptyStorePresence`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadImportFormatsFailAfterStageAndTriggerImageCleanup`, `image/image_record_stage_test.go:TestImageRecordStageInvalidFetchPreservesPendingBodyForLaterCommit`.
- Advertised glance-direct, web-download, glance-download, copy-image and custom methods use the complete owned raw Import compiler without image/store/method discovery. Only exact web-download forwards URI; only exact glance-download forwards a complete truthy remote tuple. Stores preserve raw IDs and order, formats use Python truthiness after empty update, and invalid selected controls retain prior stage/create evidence and invoke selected cleanup.
  Tests: `image/image_record_create_upload_test.go:TestImageRecordCreateUploadImportMethodSpecificForwardingAndNoBinaryRead`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadStoreImplicationAndEmptyStorePresence`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadImportFormatsFailAfterStageAndTriggerImageCleanup`, `image/image_record_import_test.go:TestImageRecordImportPluralStoreConstructorsPreserveRawIDsOrderDuplicatesAndMissingNull`.
- Native HTTP import failures are explicit Go errors even though the Source branch can ignore a returned no-raise ACK. Accepted import Body is opaque and preserved separately; read/close/guard faults keep actual ACK proof, native rejections invent no accepted receipt, and either selected failure reaches the same non-Task cleanup boundary.
  Tests: `image/image_record_create_upload_test.go:TestImageRecordCreateUploadNativeFailuresAndCleanMissingCleanupPreserveOriginalEvidence`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadAcceptedResponseFaultAndCleanupFaultAreJoinedWithReceipts`, `image/image_record_create_admin_test.go:TestImageRecordCreateWholeNativeAdminDenialKeepsBranchCleanupBoundary`.
- Optional remote checksum GET runs only when ValidateChecksum and either final flat hash after property conversion and Meta overrides are truthy. Expected values are captured before the constructor conflicting hook. A falsey actual checksum is accepted with Compared/Matched false; otherwise exact Python JSON equality with MD5 OR SHA256 decides. No os_hash/download/content-proof substitution or extra final fetch is added.
  Tests: `image/image_record_create_upload_test.go:TestImageRecordCreateUploadChecksumUsesFlatMetaBeforeConstructorHook`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadFalseyRemoteChecksumRemainsUnverified`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadChecksumFailureDeletesFetchedPrivateCurrentID`.
- Only failures inside binary/stage/import/checksum trigger whole-image cleanup using the current private ID, including an ID changed by the checksum fetch. Cleanup constructs an ID-only resource and suppresses only a clean final native404; POST/unsupported/pre-block failures cause no DELETE. Original and cleanup causes/actual receipts are joined, and failed/canceled/sticky source guards never use detached cleanup contexts.
  Tests: `image/image_record_create_upload_test.go:TestImageRecordCreateUploadImportMethodRejectionAfterCreateHasNoCleanup`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadNativeFailuresAndCleanMissingCleanupPreserveOriginalEvidence`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadChecksumFailureDeletesFetchedPrivateCurrentID`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadAcceptedResponseFaultAndCleanupFaultAreJoinedWithReceipts`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadRestoredSourceFailureNeverUsesDetachedCleanup`, `image/image_record_create_admin_test.go:TestImageRecordCreateWholeNativeAdminDenialKeepsBranchCleanupBoundary`, `image/image_record_create_deadline_test.go:TestImageRecordCreateNativeRequestDeadlineKeepsParentLiveForCleanup`, `image/image_record_create_deadline_test.go:TestImageRecordCreateTaskWaitKeepsCallerCancellationAndSourceFailureSticky`.
- Task resolves object-store availability/configured service lazily after reuse and only when upload and cloud Tasks are selected. Raw has_object_store is type-checked as bool at this branch; disabled/clean absence reports unavailable while other provider/auth/source errors propagate. Metadata/direct/reused operations do not construct Swift or observe its branch-only configuration.
  Tests: `image/image_record_create_swift_test.go:TestImageRecordCreateTaskWholeAvailabilityIsLazyAndRawConfigIsBool`, `image/image_record_create_workflow_test.go:TestImageRecordCreateWorkflowReuseStopsBeforeVendorPropertiesAndUploadConflicts`, `connectiontest/image_record_create_integration_test.go:TestConnectionOwnedImageCreateComposesPolicyAndKeepsSwiftLazy`, `image/image_record_create_policy_test.go:TestImageCreatePolicyRawObjectStoreOverrideDefersAvailabilityChecksAndTypedHelperClearsIt`.
- Task ensures its derived container through actual broad200..399 HEAD: existing returns, clean physical404 causes PUT then HEAD. Container receipts stay independent, invalid/faulted responses remain errors and no container rollback is introduced. Captured Swift source-only registration and outer operation guard remain active during options/auth/retry/response Read/Close.
  Tests: `image/image_record_create_swift_test.go:TestImageRecordCreateTaskWholeContainerEnsureAndLateEmptyDataConflict`, `image/image_record_create_swift_test.go:TestImageRecordCreateTaskWholeNoWaitKeepsRawTaskAndAllSourceSuccessCodes`, `objectstorage/v1/containers/image_import_guard_test.go:TestImageImportContainerMetadataOuterGuardStopsNativeReplay`, `objectstorage/v1/containers/image_import_guard_test.go:TestImageImportContainerOuterGuardReadFailureRemainsAfterCloseRestore`, `objectstorage/v1/containers/image_import_guard_test.go:TestImageImportContainerSourceOnlyRegistrationPersistsWithoutRecursion`.
- Task bytes/text/borrowed Reader use a single ordinary Swift PUT without pre-read/Seek/Close/spool/replay. File input reuses capability discovery/staleness and ordinary or default-SLO engines. All uploads own octet-stream, x-delete-after86400 and x-object-meta-x-sdk-autocreated=true; raw-data hashes are ignored, and filename/data presence conflict remains late after ensure.
  Tests: `objectstorage/v1/objects/image_import_source_test.go:TestImageImportObjectBorrowedDataUsesOnlyOneOrdinaryPUT`, `objectstorage/v1/objects/image_import_source_test.go:TestImageImportObjectPresentNilDataIsEmptyPUT`, `objectstorage/v1/objects/image_import_source_test.go:TestImageImportObjectBorrowedDataNeverRunsNativeReplayPolicies`, `image/image_record_create_swift_test.go:TestImageRecordCreateTaskWholeNoWaitKeepsRawTaskAndAllSourceSuccessCodes`, `image/image_record_create_swift_test.go:TestImageRecordCreateTaskWholeFilenameReusesSwiftCapabilityStaleAndArbitraryHashProfile`.
- Modern synthesized absent filename hashes are non-None empty strings, so the reused Swift profile sets GenerateChecksums=false. Missing HEAD sends no calculated hash metadata; an existing no-hash HEAD uses comparison-only fallback. Supplied filename hashes cross the concrete valid UTF-8 string bridge without hex restrictions; raw-data input does not convert or use supplied hashes.
  Tests: `objectstorage/v1/objects/image_import_source_test.go:TestImageImportObjectFilenameReusesStaleAndArbitraryHashPolicy`, `image/image_record_create_swift_test.go:TestImageRecordCreateTaskWholeFilenameReusesSwiftCapabilityStaleAndArbitraryHashProfile`, `objectstorage/v1/objects/create_core_test.go:TestObjectCreateCoreChecksumDecisions`.
- Task filename upload reuses the actual Swift immutable spool, capability/stale policies, segment scheduling/native physical retry, exact-owned cleanup and manifest ambiguity handling. The image composition accepts its broad Source200..399 acknowledgements (including segment202) through a private profile while standalone Swift behavior stays separate; no second SLO implementation or Reader-to-spool substitution is introduced.
  Tests: `objectstorage/v1/objects/image_import_source_test.go:TestImageImportObjectFilenameReusesSLOWithBroadSourceAcknowledgements`, `objectstorage/v1/objects/image_import_source_test.go:TestImageImportObjectFilenameNativeRetryCannotReplaceImageControls`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadSegmentRoundsAndExactCleanup`, `objectstorage/v1/objects/create_upload_test.go:TestObjectCreateUploadManifestAttemptsAndAmbiguity`, `objectstorage/v1/objects/create_contracts_test.go:TestCreateObjectContractsCleanupAndStickyAmbiguity`.
- Task POST type import/input import_from container/name/image_properties formal name follows object creation without image metadata POST. No-wait returns raw Task with absent/null/nonstring ID/status/result accepted, Source declared descriptors and captured location, exact accepted receipt and no object cleanup; server TTL remains. Object/Task creation failures before wait also do not enter finally cleanup.
  Tests: `image/image_record_create_swift_test.go:TestImageRecordCreateTaskWholeNoWaitKeepsRawTaskAndAllSourceSuccessCodes`, `image/image_record_create_swift_test.go:TestImageRecordCreateTaskWholeFailuresBeforeWaitLeaveObjectAndContainer`, `image/image_record_create_task_test.go:TestImageRecordCreateTaskNoWaitPreservesRawMissingAndNonstringDescriptors`, `image/image_record_create_task_test.go:TestImageRecordCreateTaskConstructorAndSparseTranslationUseTaskDescriptors`, `image/image_record_create_admin_test.go:TestImageRecordCreateWholeNativeAdminDenialKeepsBranchCleanupBoundary`.
- Task wait returns seeded success before GET, identity and timeout validation; otherwise it uses the original default3600-second budget and2-second polls with parent context. JSON null is unbounded, zero/negative expires and numeric/bool/fractional timeout follows finite Go timer bounds. Only exact396 recreates fetched raw type/input within the same budget; recreated success still pauses/GETs, and OriginalTask preserves the outer pre-recreation object for diagnostics. The Task budget context is child-scoped; expiration returns its deadline error without poisoning parent-scope source invariants.
  Tests: `image/image_record_create_task_test.go:TestImageRecordCreateTaskWaitSeedSuccessPrecedesIDAndTimeoutValidation`, `image/image_record_create_task_test.go:TestImageRecordCreateTaskWaitTimeoutDomainAndFixedBudget`, `image/image_record_create_task_test.go:TestImageRecordCreateTaskWaitExact396RecreatesRawInputsAndGetsRecreatedSuccess`, `image/image_record_create_task_test.go:TestImageRecordCreateTaskWaitMissingRecreationFieldsBecomeExplicitNull`, `image/image_record_create_task_test.go:TestImageRecordCreateTaskWaitDefaultPauseHonorsParentAndFractionalBudget`, `image/image_record_create_task_test.go:TestImageRecordCreateTaskWaitPreservesOriginalDiagnosticObjectAcrossRecreations`, `image/image_record_create_deadline_test.go:TestImageRecordCreateTaskOwnDeadlineKeepsParentLiveForFinallyCleanup`, `image/image_record_create_deadline_test.go:TestImageRecordCreateNativeRequestDeadlineKeepsParentLiveForCleanup`, `image/image_record_create_deadline_test.go:TestImageRecordCreateTaskWaitKeepsCallerCancellationAndSourceFailureSticky`.
- Task success takes raw result.image_id, fetches the complete image, copies fetched properties and overwrites them with original unconverted create properties, retaining root tags/selected legacy visibility while removing root formats. Meta/Size are ignored and the shared normalized automatic PATCH applies with no conversion, name re-force or resolver. Task/Image location comes from captured Source-style current-location collection.
  Tests: `image/image_record_create_swift_test.go:TestImageRecordCreateTaskWholeSuccessMergesUnconvertedPropertiesAndFinallyDeletesObject`, `image/image_record_create_swift_test.go:TestImageRecordCreateTaskWholeFailureDiagnosesOriginalTaskOnlyForResourceFailure`, `image/image_record_create_task_test.go:TestImageRecordCreateTaskConstructorAndSparseTranslationUseTaskDescriptors`.
- Only an observed non396 Task failure is a ResourceFailure and triggers an extra GET of the outer original Task for diagnostic evidence; HTTP/type/timeout/source failures do not. Every entered wait exit performs Swift metadata HEAD/delete/SLO cleanup with joined errors, including success/update failure, and never rolls back image/container. No-wait and pre-wait object/Task failures remain outside this finally. Every entered wait exit invokes the existing Swift cleanup engine using the original parent context. Actual HEAD/DELETE remains possible after an internal wait timeout when the parent/source are live; parent cancellation or genuine source failure stops physical cleanup and preserves its cause, without detached context.
  Tests: `image/image_record_create_swift_test.go:TestImageRecordCreateTaskWholeFailureDiagnosesOriginalTaskOnlyForResourceFailure`, `image/image_record_create_swift_test.go:TestImageRecordCreateTaskWholeFailuresBeforeWaitLeaveObjectAndContainer`, `image/image_record_create_swift_test.go:TestImageRecordCreateTaskWholeJoinsCleanupFailureAndStopsAfterSwiftSourceDrift`, `image/image_record_create_swift_test.go:TestImageRecordCreateTaskWholeSLOCleanupUsesExistingBulkDeleteEvidence`, `image/image_record_create_task_test.go:TestImageRecordCreateTaskWaitPreservesOriginalDiagnosticObjectAcrossRecreations`, `image/image_record_create_admin_test.go:TestImageRecordCreateWholeNativeAdminDenialKeepsBranchCleanupBoundary`, `image/image_record_create_deadline_test.go:TestImageRecordCreateTaskOwnDeadlineKeepsParentLiveForFinallyCleanup`, `image/image_record_create_deadline_test.go:TestImageRecordCreateNativeRequestDeadlineKeepsParentLiveForCleanup`, `image/image_record_create_deadline_test.go:TestImageRecordCreateTaskWaitKeepsCallerCancellationAndSourceFailureSticky`.
- Concrete With factories and operation preparation independently own raw/maps/bytes/store/header/config snapshots, apply nested callbacks once and stop after each sticky guard. Glance/Connection/provider/base/type/microversion/API source remains captured across all phases; lazy Swift source-only guards avoid recursive registries and cover native auth/retry/Read/Close. Borrowed lifetimes and actual accepted phase receipts remain independent and no fabricated ready/active/proof is returned. Sticky source failures and caller cancellation remain authoritative; temporary request/wait lifetime contexts must not be cached as permanent source-binding failures. HTTP/native request timer cancellation is returned per request rather than stored as source drift; normal response Close timer cancellation does not turn successful Swift upload into failure. Genuine Source drift and caller cancellation, including mixed timeout errors, remain sticky.
  Tests: `image/image_record_create_options_test.go:TestImageRecordCreateWholeOptionsCaptureNestedRawMapsStoresAndReuseIndependently`, `image/image_record_create_options_test.go:TestImageRecordCreateNestedImportCallbacksRunOnceAndStopAfterEachGuard`, `image/image_record_create_options_test.go:TestImageRecordCreateOptionAndMarshalerGuardsStopLaterCallbacks`, `image/image_record_create_workflow_test.go:TestImageRecordCreateWorkflowCallbacksAndSourceAreCapturedOnce`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadRestoredSourceFailureNeverUsesDetachedCleanup`, `image/image_record_create_task_test.go:TestImageRecordCreateTaskNativeRetryFreezesRawSubmissionAndRetainsLiveAuthentication`, `objectstorage/v1/objects/image_import_source_test.go:TestImageImportObjectOuterGuardRetainsRequestReadAndAcceptedResponseEvidence`, `objectstorage/v1/objects/image_import_metadata_guard_test.go:TestImageImportObjectMetadataOuterGuardStopsNativeRetryAndReauthentication`, `objectstorage/v1/objects/image_import_metadata_guard_test.go:TestImageImportObjectMetadataReadGuardKeepsAcceptedEvidenceBeforeCloseRestores`, `objectstorage/v1/containers/image_import_guard_test.go:TestImageImportContainerMetadataOuterGuardStopsNativeReplay`, `image/image_record_create_deadline_test.go:TestImageRecordCreateTaskOwnDeadlineKeepsParentLiveForFinallyCleanup`, `image/image_record_create_deadline_test.go:TestImageRecordCreateNativeRequestDeadlineKeepsParentLiveForCleanup`, `image/image_record_create_deadline_test.go:TestImageRecordCreateTaskWaitKeepsCallerCancellationAndSourceFailureSticky`.
- Whole create forwards ordinary own-project metadata/import/update and admin-scoped public visibility/other-project owner/copy-image/configured external Task controls through the same operation without client role gates. Native server policy/quota/store/URI/backend errors remain authoritative; one create_image declaration is counted, not each branch, and local accepted transport proof does not establish deployment outcome.
  Tests: `image/image_record_create_workflow_test.go:TestImageRecordCreateWorkflowIsPublicWarningBelongsToSelectedUploadOnly`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadImportMethodSpecificForwardingAndNoBinaryRead`, `image/image_record_create_upload_test.go:TestImageRecordCreateUploadNativeFailuresAndCleanMissingCleanupPreserveOriginalEvidence`, `image/image_record_create_task_test.go:TestImageRecordCreateTaskNativeRejectionsPreserveStatusWithoutResourceFailure`, `image/image_record_create_admin_test.go:TestImageRecordCreateWholeForwardsOwnerAndVisibilityToServerPolicy`, `image/image_record_create_admin_test.go:TestImageRecordCreateWholeNativeAdminDenialKeepsBranchCleanupBoundary`.
- Preserve all12 legacy CreateAndImport contract objects/test refs and its explicit fresh-image/queued/post-stage/default/wait/no-cleanup profile; add the complete modern whole Service and Connection creation graph separately. Public whole-project/service guides and executable standalone/Connection examples must cover all modern routes and explicit differences. Promotion is allowed only after actual focused/full checks, generator zero drift, local/remote consumers and Source/contract/test inventories are verified; global SDK-R1/C1/S1 and standalone Swift/Task rows remain unresolved.
  Tests: `connectiontest/image_record_create_integration_test.go:TestConnectionOwnedImageCreateComposesPolicyAndKeepsSwiftLazy`, `connectiontest/image_record_create_integration_test.go:TestConnectionOwnedImageCreateRejectsInvalidInvocation`, `internal/cmd/sdkgen/glance_create_import_test.go:TestGlanceCreateImportKeepsNativeImageAPIAndDocumentsUpperService`, `internal/cmd/sdkgen/glance_create_import_test.go:TestGlanceCreateImportRejectsNativeMetadataAndResponseFieldDrift`.

Python/Go differences:

- Legacy CreateAndImport profile: CreateAndImport is an explicit fresh-image import workflow with qcow2/bare/private, glance-direct and no-wait defaults. Python create_image additionally selects metadata-only/direct-file/configured Swift task paths and uses cloud format/vendor/container configuration.
- Legacy CreateAndImport profile: Go prepares and freezes concrete metadata/stage/import/wait policy before POST or reader consumption, using local-only SDK seams and separate callback namespaces. Python kwargs/meta conversion, descriptor values and mutable Resource/session behavior are broader.
- Legacy CreateAndImport profile: Python non-task _upload_image_put does not receive create_image wait or timeout. Go optional active wait is an explicit five-minute/two-second policy, with caller overrides and parent context, rather than a reproduction of timeout3600 for that source branch.
- Legacy CreateAndImport profile: Go preserves actual accepted201/204/200/202 phase evidence and performs no cleanup DELETE after failure. Python checks response import-method capability after creation, stages without this post-stage metadata GET and deletes upload/import/checksum failures inside its try block.
- Legacy CreateAndImport profile: Go canonical created ID and original lowercase post-stage format decisions reject missing/unsafe data while typed case aliases and passive Location cannot choose the route. Go retains native typed projection/header behavior alongside independent raw response evidence.
- Legacy CreateAndImport profile: Go explicit plural stores and optional false flags are preserved together, while Python stores-is-not-None suppresses both all-store flags. Existing leaf singular Store is header-only rather than Python header plus one-element stores body.
- Legacy CreateAndImport profile: Go borrows explicit io.Reader without filename/name inference, hashing, automatic whole-file size or checksum validation. Binary staging disables SDK replay callbacks and redirects; caller RoundTripper internals and blocking Read release remain caller responsibilities.
- Modern whole CreateImageRecord covers the complete finite JSON/source-fixed create_image graph. Legacy CreateAndImport remains a separately named stricter fresh-image import profile with its preserved12 contracts; its defaults/queued/followupGET/active-wait/no-cleanup differences do not describe modern creation.
- Go owns immutable tagged data, raw JSON/maps/headers/store/config snapshots and independent Image/Task/phase/cleanup receipts, rather than mutating arbitrary Python Resource/Munch/subclasses. Raw Body/current/original/dirty presence, constructor hook ordering, Source-style captured current location and header collector reset reuse owned Resource helpers; full dynamic class/component/default alias/header/URI/backing lifecycle remains SDK-R1.
- Go native400..599 import rejection and retained response causes are explicit improvements over Source returned no-raise import ACK. Errors within selected binary/import/checksum block delete current private image ID; no successful import is equated with backend completion or active status.
- Go closes SDK-owned filenames once and clears their retained data reference; Source opens its upload file without a comparable owned closure. Borrowed Readers stay open/current-cursor, binary replay disabled, and blocked caller Read or custom RoundTripper internal replay still requires caller cooperation.
- Go joins original and cleanup faults rather than Source masking or cloud SDKException suppression, preserves actual partial receipts and keeps source/context failure sticky. Cleanup uses the same captured context/source and is never detached to bypass a failed binding.
- Source signed arbitrary Python integer/bool Size is retained in the JSON domain, including bool True/False; float/string/container size rejects inside cleanup. Size inference is restricted to Source file-like filename/Reader capability, excluding artificial Go Bytes/Text seekability. Task timeout null/zero/bool/number/fraction is retained but positive timer values beyond Go duration range are saturated and tiny positives bounded to1ns.
- Swift filename composition reuses unique segment prefixes, immutable replay sections, exact-owned cleanup and ambiguity policies instead of reproducing the Source FileSegment cursor bug. Image-only profile broad200..399 and segment202 differ from unchanged standalone Swift gates. Arbitrary valid UTF-8 string hashes are accepted without hex restrictions; nonstring filename hashes lie outside the concrete Swift string bridge, while raw-data hashes are ignored like Source.
- The actual pinned filename-extension inference typo is preserved. No corrected inference mode or extra default/private/queued/active gates are introduced.
- Only clean disabled/object-store absence becomes unavailable; mixed/auth/IO/source errors propagate rather than broad Source has_service suppression. Formal Go request fields and options are separate concrete namespaces where Python duplicate-argument binding cannot be represented; fixed string-key JSON, complete UTF-8/paired-surrogate routing, safe literal IDs/header/source controls and JSON serializer order remain explicit bounds.
- Native server policy including own-project modify/community visibility and admin public/other-owner/copy-image/tasks access, store availability, Swift ACL/quota/URI authorization and eventual backend processing remain server-owned. Local tests cannot establish live cloud success or universal policy overrides.
- Captured provider/origin/base/type/microversion and source-only nested guards preserve current operation. Full Adapter caching/conditional GET/invalidation/metrics/global request configuration remain SDK-C1; arbitrary session/adapter/response/microversion/base-path injection remains SDK-S1. The global shared objectives and standalone Swift/Task declaration rows are not promoted by this finite modern operation.
- Request-owned Task/HTTP timer expiry does not invalidate live parent-source invariants; caller cancellation or real binding/source drift, including joined timeout errors, stays sticky and cannot be bypassed by detached cleanup. Ordinary response closure can cancel native request timers without manufacturing failed Swift uploads.
- The modern import phase reuses the owned ImportImageRecord policy: configured legacy store headers are discarded from the copied request client, then an explicit singular selection supplies its fixed header. Source/shared header maps stay unchanged.

### `python:cloud/_image/ImageCloudMixin/create_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.CreateCloudImageRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.PrepareImageRecordCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordCreateOpts`

- Without a volume CreateCloudImageRecord runs the owned CreateImageRecord graph with one prepared option snapshot and returns its Record when wait is falsey; with a volume it maps truthy allow_duplicates to force and Python-falsey container/disk formats to the Cinder Proxy defaults before CreateVolumeImageRecord.
  Tests: `connectiontest/connection_cloud_image_create_test.go:TestConnectionCloudImageCreateBranchesAndWait`, `connectiontest/connection_cloud_image_create_test.go:TestConnectionCloudImageCreateVolumeBodyAndPreflight`.
- A truthy wait repeats Cloud get_image on the created id: missing images keep polling, a found status other than exact queued or saving, including null, is returned, and WaitLookups/WaitLast keep the evidence.
  Tests: `connectiontest/connection_cloud_image_create_test.go:TestConnectionCloudImageCreateBranchesAndWait`.
- The iterate_timeout deadline uses the raw create timeout (default 3600 seconds, null unlimited); on timeout the image is deleted through Cloud delete_image with wait and the DeadlineExceeded error keeps the cleanup result and created record.
  Tests: `connectiontest/connection_cloud_image_create_test.go:TestConnectionCloudImageCreateBranchesAndWait`.
- Truthy non-string volume formats, non-numeric timeouts with wait, nil options and negative poll intervals fail before HTTP.
  Tests: `connectiontest/connection_cloud_image_create_test.go:TestConnectionCloudImageCreateVolumeBodyAndPreflight`.

Python/Go differences:

- Python returns an Image or None and raises ResourceTimeout or the cleanup exception; Go returns CloudImageCreateResult with Created, Volume, wait and Cleanup evidence and joins the timeout with any cleanup failure.
- Python keyword arguments become the concrete ImageRecordCreateOption values; the volume is a literal VolumeID and PollInterval is a Go extension for the Cloud wait and its cleanup.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
record = conn.image.create_image(
    name="ubuntu-example",
    filename="ubuntu.qcow2",
    visibility="private",
    allow_duplicates=True,
)

record = conn.image.create_image(
    name="binary-example",
    data=b"image bytes",
    validate_checksum=True,
)

record = conn.image.create_image(
    name="import-example",
    filename="ubuntu.qcow2",
    use_import=True,
    import_method="glance-direct",
)
```

```sh
go run . -cloud dev -scenario direct -file ubuntu.qcow2 -name ubuntu-example
go run . -cloud dev -scenario bytes -file ubuntu.qcow2 -name binary-example -service
go run . -cloud dev -scenario import -file ubuntu.qcow2 -name import-example
go run . -cloud task-cloud -scenario task -file ubuntu.qcow2 -name task-example -wait=true
go run . -cloud dev -scenario metadata -name metadata-example
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
)

type createImage func(context.Context, image.ImageRecordCreateRequest, ...image.ImageRecordCreateOption) (*image.ImageRecordCreateResult, error)

func main() {
    cloud := flag.String("cloud", "", "clouds.yaml entry; omission uses OS_CLOUD/OS_* settings")
    scenario := flag.String("scenario", "direct", "direct, bytes, import, task, or metadata")
    filename := flag.String("file", "", "local image file for direct/bytes/import/task")
    name := flag.String("name", "image-example", "image name")
    viaService := flag.Bool("service", false, "call image.Service instead of Connection")
    wait := flag.Bool("wait", true, "wait for Task completion in the task scenario")
    forceTask := flag.Bool("force-task-policy", false, "replace cloud image policy with explicit Task policy")
    flag.Parse()
    if *scenario != "metadata" && *filename == "" {
        log.Fatal("this scenario requires -file")
    }
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *scenario, *filename, *name, *viaService, *wait, *forceTask); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, scenario, filename, name string, viaService, wait, forceTask bool) error {
    options := []openstack.ConnectionOption{}
    if cloud != "" {
        options = append(options, openstack.WithCloud(cloud))
    }
    if forceTask {
        options = append(options, openstack.WithImageCreatePolicy(
            image.WithImageCreatePolicyTasks(true),
            image.WithImageCreatePolicyObjectStoreEnabled(true),
        ))
    }
    conn, err := openstack.Connect(ctx, options...)
    if err != nil { return err }
    var create createImage = conn.CreateImageRecord
    if viaService {
        service, serviceErr := conn.Image(ctx)
        if serviceErr != nil { return serviceErr }
        create = service.CreateImageRecord
    }
    var result *image.ImageRecordCreateResult
    switch scenario {
    case "direct":
        result, err = direct(ctx, create, filename, name)
    case "bytes":
        result, err = validatedBytes(ctx, create, filename, name)
    case "import":
        result, err = imported(ctx, create, filename, name)
    case "task":
        result, err = task(ctx, create, filename, name, wait)
    case "metadata":
        result, err = metadata(ctx, create, name)
    default:
        return fmt.Errorf("unknown scenario %q", scenario)
    }
    return show(result, err)
}

func direct(ctx context.Context, create createImage, filename, name string) (*image.ImageRecordCreateResult, error) {
    return create(ctx, image.ImageRecordCreateRequest{
        Name: name, Filename: filename, Attributes: map[string]any{"visibility": "private"},
    }, image.WithImageRecordCreateAllowDuplicates(true))
}

func validatedBytes(ctx context.Context, create createImage, filename, name string) (*image.ImageRecordCreateResult, error) {
    data, err := os.ReadFile(filename)
    if err != nil { return nil, err }
    return create(ctx, image.ImageRecordCreateRequest{
        Name: name, Data: image.ImageRecordCreateBytes(data),
    }, image.WithImageRecordCreateValidateChecksum(true))
}

func imported(ctx context.Context, create createImage, filename, name string) (*image.ImageRecordCreateResult, error) {
    return create(ctx, image.ImageRecordCreateRequest{Name: name, Filename: filename},
        image.WithImageRecordCreateAllowDuplicates(true),
        image.WithImageRecordCreateUseImport(true),
        image.WithImageRecordCreateImportOptions(image.WithImageRecordImportMethod("glance-direct")),
    )
}

func task(ctx context.Context, create createImage, filename, name string, wait bool) (*image.ImageRecordCreateResult, error) {
    return create(ctx, image.ImageRecordCreateRequest{Name: name, Filename: filename},
        image.WithImageRecordCreateAllowDuplicates(true),
        image.WithImageRecordCreateContainer("images"),
        image.WithImageRecordCreateWait(wait),
        image.WithImageRecordCreateTimeout(300),
    )
}

func metadata(ctx context.Context, create createImage, name string) (*image.ImageRecordCreateResult, error) {
    return create(ctx, image.ImageRecordCreateRequest{
        Name: name, Attributes: map[string]any{"min_disk": "0", "hw_qemu_guest_agent": false},
    }, image.WithImageRecordCreateAllowDuplicates(true),
        image.WithImageRecordCreateTags([]string{"sdk-example"}),
        image.WithImageRecordCreateMeta(map[string]any{"vendor_raw": map[string]any{"enabled": false}}),
    )
}

func recordSummary(record *image.ImageRecord) any {
    if record == nil { return nil }
    return map[string]any{
        "resource": record.Resource, "wire": record.Wire,
        "status_code": record.StatusCode, "header": record.Header,
        "envelope_bytes": []byte(record.Envelope), "import_methods": record.ImportMethods,
    }
}

func taskSummary(value *image.ImageRecordCreateTask) any {
    if value == nil { return nil }
    return map[string]any{
        "resource": value.Resource, "status_code": value.StatusCode,
        "header": value.Header, "envelope_bytes": []byte(value.Envelope),
    }
}

func show(result *image.ImageRecordCreateResult, operationErr error) error {
    value := map[string]any{"partial": result != nil && operationErr != nil}
    if operationErr != nil { value["error"] = operationErr.Error() }
    if result != nil {
        value["outcome"], value["reused"], value["warnings"] = result.Outcome, result.Reused, result.Warnings
        value["record"], value["updated"] = recordSummary(result.Record), recordSummary(result.Updated)
        value["task"], value["task_diagnostic"] = taskSummary(result.Task), taskSummary(result.TaskDiagnostic)
        value["checksum"], value["imported"] = result.Checksum, result.Imported
        for phase, receipt := range map[string]*image.ImageUploadResponse{
            "created": result.Created, "uploaded": result.Uploaded, "staged": result.Staged,
            "checksum_fetched": result.ChecksumFetched, "task_image_fetched": result.TaskImageFetched,
            "task_diagnostic_response": result.TaskDiagnosticResponse,
        } {
            if receipt != nil { value[phase] = receipt }
        }
        if result.TaskWait != nil {
            value["task_wait"] = map[string]any{
                "task": taskSummary(result.TaskWait.Task), "original_task": taskSummary(result.TaskWait.OriginalTask),
                "fetches": result.TaskWait.Fetches, "recreations": result.TaskWait.Recreations,
                "fetch_responses": result.TaskWait.FetchResponses, "recreation_responses": result.TaskWait.RecreationResponses,
            }
        }
        if result.Swift != nil {
            value["swift"] = map[string]any{
                "container_discovery": result.Swift.ContainerDiscovery,
                "container_created": result.Swift.ContainerCreated,
                "container_fetched": result.Swift.ContainerFetched,
                "object": result.Swift.Object, "cleanup": result.Swift.Cleanup,
            }
        }
        if result.Cleanup != nil { value["image_cleanup_ack"] = result.Cleanup.Acknowledgement }
    }
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return errors.Join(operationErr, err) }
    fmt.Println(string(body))
    return operationErr
}
```

## References

- [create_image](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L178-L436)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Implementation Plan](../docs/implementation-plan.md)
- [Upload](upload-image.md)
- [CreateAndImport](create-import.md)
- [UploadImageRecord](image-record-upload.md)
- [StageImageRecord](image-record-stage.md)
- [ImportImageRecord](image-record-import.md)
- [Install](../docs/install.md)
- [Source filename helper](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L50-L63)
- [Create](../objectstorage/v1/objects/create.md)
