# Images

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/wait_for_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.WaitForState`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.WaitForState`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.WaitForImageRecordStatus`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitTimeout`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitPollInterval`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitUnlimited`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitFailureStates`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitAttribute`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitCallback`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitHeaders`

- Generic service WaitForState requires a target and applies exact case-insensitive ERROR, two-second polling and no SDK timeout. The returned model is the actual native observation, not a cached supplied Python Resource.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitDeadlinesDefaultsParentsAndCallerOrdering`, `api/service_wait_contracts_test.go:TestServiceWaitContractsDefaultDeadlinesAndConvenienceTargets`.
- Target state comparison precedes failure comparison. WithFailureStates replaces the exact case-insensitive default list, and an empty list disables state failure checks. error_deleting/error_extending, failed or killed are not inferred as additional service failures. Captured failure-state values remain independent of later caller slice mutations.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitExactFailuresReplacementAndTargetPrecedence`, `api/service_wait_contracts_test.go:TestServiceWaitContractsExactFailuresAndCallerOverrides`.
- SDK defaults are prepended and concrete caller resource.WaitOption values apply last. WithUnlimitedWait removes a finite default, a later positive WithTimeout reinstates a bound, and parent context deadlines and cancellation remain effective. No application status adapter or polling builder is required.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitDeadlinesDefaultsParentsAndCallerOrdering`, `api/service_wait_contracts_test.go:TestServiceWaitContractsDefaultDeadlinesAndConvenienceTargets`.
- The default polling interval is two seconds and a positive caller interval replaces it. Nil context or collection, invalid refs/targets/options, unsupported selected attributes and canceled context fail without HTTP. Callback cancellation and active-fetch cancellation stop before any further poll.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitDefaultTwoSecondIntervalAndCallerShortening`, `internal/servicewait/wait_test.go:TestServiceWaitPreflightAndCallerCancellationBeforeAnotherFetch`, `api/service_wait_contracts_test.go:TestServiceWaitContractsPreflightAndCallerCancellation`.
- Status selection uses an exported string field and can also select deletion completion. Server canonical integer progress reports native values; absent canonical progress reports zero. Cinder-v3 Snapshot uses the noncanonical os-extended-snapshot-attributes:progress string tag, so callback reports zero without parsing that string. Successful terminal, failure and HTTP-error observations do not invoke the callback.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitReusableCallerOptionsRemainOwnedAndSelectAttributes`, `api/service_wait_contracts_test.go:TestServiceWaitContractsStatusAttributesAndNativeProgressLimits`, `api/service_wait_contracts_test.go:TestServiceWaitContractsNameBindingLiveClientAndProgress`, `resource/wait_attributes_test.go:TestWaitProgressCallbackCancellationStopsFurtherGETs`, `resource/wait_attributes_test.go:TestWaitProgressOmitsSuccessFailureAndHTTPError`.
- Explicit ID is stable even when response ID differs. Existing Name lookup resolves once, retains the native list/name-query boundary, and state waiting reports its initial List observation before fixed-ID GETs. Selected ResourceBase/reverse prefix, configured ordinary headers/microversion, caller context/middleware and the original live Provider token remain in use; no response-driven route or model synthesis occurs.
  Tests: `api/service_wait_contracts_test.go:TestServiceWaitContractsNameBindingLiveClientAndProgress`, `resource/wait_identity_test.go:TestWaitKeepsExplicitIDWhenResponseIdentityChanges`, `api/service_wait_contracts_test.go:TestServiceWaitContractsDeletionOnlyObservesAndRejectsTerminalErrors`.
- Fresh lookup failures remain errors rather than a successful target model. The shared read gate preserves original terminal URL/accepted-response/JSON/read/context causes even when nested native404 or adapter-owned ErrNotFound is discoverable. Actual direct403 or transport-wrapped404 terminates the service deletion observation after one request. Successful missing state fetches return a nonpanic error; successful nil deletion fetches complete.
  Tests: `api/service_wait_contracts_test.go:TestServiceWaitContractsDeletionOnlyObservesAndRejectsTerminalErrors`, `resource/read_errors_test.go:TestReadErrorsNeverAddMissingToTerminalDirectOrQueryCauses`, `resource/read_errors_test.go:TestReadErrorsDefendIgnoreMissingAndWaitAgainstAdapterPrewrappedTerminalCauses`, `resource/wait_policy_test.go:TestWaitMissingSuccessfulResponseDoesNotPanic`, `resource/wait_attributes_test.go:TestWaitDeletedNilSuccessfulFetchIsTerminalWithoutCallback`.
- Existing common Collection.Wait/WaitDeleted and generated WaitFor/WaitForDeletion retain their five-minute deadline and original binding failure predicate. Native WaitForStatus retains its error-only callable and native predicate-first behavior, including original caller context; source one-second/exact-case behavior is not replaced by this service policy. This is preservation evidence, not a separate native-operation status promotion.
  Tests: `api/service_wait_contracts_test.go:TestServiceWaitContractsLegacyAndNativeSignaturesStayUnchanged`, `internal/servicewait/wait_test.go:TestServiceWaitDeadlinesDefaultsParentsAndCallerOrdering`.
- Generator discovery records only the six audited Server/Cinder Volume-Snapshot/Image-v2 collections with their service wait policy. Existing generated common WaitFor/WaitForDeletion still delegate unchanged; unrelated Backup, Flavor, Task, network and Image-v1 collections receive no inferred service-default policy.
  Tests: `internal/cmd/sdkgen/service_waits_test.go:TestServiceWaitPoliciesBindOnlyActualStatusCollections`, `internal/cmd/sdkgen/service_waits_test.go:TestServiceWaitPolicyRejectsParentDriftAndDoesNotInferOtherModels`.
- Owned WaitForImageRecordStatus compares the supplied canonical descriptor before polling and returns an independent seed clone without HTTP or progress when the target already matches. Initial target comparison precedes duration and failure-list validation; structural seed/source/options validation still runs.
  Tests: `image/image_record_wait_test.go:TestImageRecordWaitStatusInitialTargetReturnsIndependentSeed`.
- The owned profile uses locale-independent Unicode16 full lowercase for initial, fetched target and failure comparisons, including dotted-I expansion and contextual Final_Sigma. Target success precedes failure, nil failures means ERROR, empty failures disables failure checks, null state remains pending and nonstring selected state is an explicit processing error.
  Tests: `image/image_record_wait_test.go:TestImageRecordWaitStatusFreshSelectionFailureAndPending`, `image/image_record_wait_test.go:TestImageRecordWaitUnicodeStatusComparison`, `internal/cloudfilter/python_lower_test.go:TestPythonLowerFullUnicode16AndOriginalContext`.
- Owned status waiting defaults to no SDK timeout and two-second polling. Explicit timeout0 stops without a GET after a nonmatching initial state; interval0 becomes100ms or the shorter positive timeout. A single budget covers GET, callback and pause while preserving the parent context; concrete timeout/unlimited helpers apply in caller order.
  Tests: `image/image_record_wait_test.go:TestImageRecordWaitZeroIntervalsTimeoutAndParentBudget`, `image/image_record_wait_test.go:TestImageRecordWaitStatusOwnsOneBudgetLocationHeadersAndLiteralID`, `internal/servicewait/wait_test.go:TestServiceWaitDefaultTwoSecondIntervalAndCallerShortening`.
- Owned preparation captures source/location once and independently snapshots seed channels, ordinary headers, duration pointers, failure lists and option slice before callbacks. Each option runs once; escaped input ID remains the route despite response-ID drift and Provider authentication remains live. Concrete replacement and invalid source/seed/attribute/header/config inputs fail before HTTP.
  Tests: `image/image_record_wait_test.go:TestImageRecordWaitStatusOwnsOneBudgetLocationHeadersAndLiteralID`, `image/image_record_wait_test.go:TestImageRecordWaitCompletePreflightAndConcreteReplacement`, `connection_image_record_waits_test.go:TestConnectionImageRecordWaitsShareOwnedSourceAndLatestObservation`.
- Fresh owned GET accepts actual200..399 and reuses the declared Image projector with previous private Body64 as overlay seed. Invalid JSON syntax retains seed Body, valid object packing can reset properties, import-method headers are read afresh, and actual Wire/Envelope/Header/status remain independent from Resource. Nonobject/descriptor/physical processing failures retain their response evidence.
  Tests: `image/image_record_wait_test.go:TestImageRecordWaitAcceptedJSONToleranceAndBodyStateErrors`, `image/image_record_get_test.go:TestImageRecordGetPackingAliasesAndSeedOverlay`, `connection_image_record_waits_test.go:TestConnectionImageRecordWaitsShareOwnedSourceAndLatestObservation`.
- Only nonterminal owned observations invoke synchronous progress0, because Image has no declared progress descriptor. Cancellation, source changes and physical/processing errors stop before another poll; after a successful projection later errors return that latest owned record with the error. Newer physical failures keep their own receipt instead of being replaced by an earlier successful receipt; state404 is an error.
  Tests: `image/image_record_wait_test.go:TestImageRecordWaitCallbacksGuardBeforePauseOrNextGET`, `image/image_record_wait_test.go:TestImageRecordWaitHTTPFailuresAnd404HandlingFaultsAreTerminal`, `image/image_record_wait_test.go:TestImageRecordWaitAcceptedAndRejectedStickyPhysicalGuards`.

Python/Go differences:

- Go accepts explicit resource.Ref and an SDK typed collection and returns a native-model pointer for state waits; Python accepts an existing Resource and may return it immediately from its cached target state without HTTP. Explicit ID performs a fresh lookup and Name retains the existing native resolver before fixing its ID.
- Python nullable status/descriptor access and arbitrary progress values differ from Go exported string/integer model fields. Cinder-v3 Snapshot progress is a native string under os-extended-snapshot-attributes:progress, which does not match the canonical progress selector; callback reports zero without coercing that string. Other absent progress is also reported as zero. Successful missing/null native status fields may already collapse during typed decoding.
- Go requires positive poll intervals and positive bounded timeouts; WithUnlimitedWait represents no SDK timeout explicitly. Python iterate_timeout zero/negative timeout and interval-zero coercion are not reproduced.
- WaitForDelete returns error only, while Python returns an original or fetched Resource. Native HTTP/transport/decoder/context causes remain visible rather than being converted into a synthetic deleted model.
- These service-default methods supplement existing five-minute Collection/leaf waits and preserve native WaitForStatus context-only one-second exact-case/error-only behavior. Create/upload and Cloud workflow policies are separate.
- State waiting can report the initial Name List model; deletion waiting resolves Name without a callback and starts progress at its first fixed-ID GET. An explicit status attribute for deletion is a Go extension; pinned Python delete observes its conventional status field.
- Earlier contracts/differences above describe the preserved Ref/native-model service policies. The additional named immutable ImageRecord profile implements cached initial-target selection, nullable declared status, fresh projection, Python-style zero controls and record-return behavior in the new contracts; the earlier typed limitations do not describe the new record methods.
- Owned ImageRecord waits preserve caller snapshots rather than mutating a supplied Resource instance. Arbitrary Resource subclasses, dirty/component lifecycle and same-instance identity remain SDK-R1; configured Adapter cache/conditional GET remains SDK-C1; dynamic session/adapter/base_path/microversion replacement remains SDK-S1 in docs/implementation-plan.md. These shared SDK objectives remain active and this named mapping does not imply full Python runtime or cloud equivalence.
- Go keeps the escaped nonblank input ID fixed rather than following response-ID mutation into a later route. Initial-target/first404 return preserves seed location; each fresh poll projects the location captured for this operation, whereas Source Image fetch without project_id/availability_zone can retain its previous computed location. Options/source/descriptor preflight runs even for an initial-target shortcut.
- The owned profile takes valid UTF-8 string targets and canonical declared Body64/location/image_import_methods attribute selectors; nullable targetNone, arbitrary Python attribute/progress objects, lone-surrogate JSON/string behavior and other parser/runtime domains are outside this finite profile. Comparisons pin Unicode16 full lowercase; other Python runtime Unicode databases are a documented boundary. Go rejects negative durations, uses monotonic context budgets and explicit unlimited helpers, and cannot interrupt a blocked synchronous callback.
- Image-v1, wait_for_task and Cloud wait_for_image/wait_for_server are separate declarations and are not incidentally promoted by this operation. The existing legacy Ref name-resolution/typed decoder boundary remains preserved alongside literal owned ImageRecord input.

### `python:image/v2/wait_for_delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.WaitForImageRecordDelete`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitTimeout`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitPollInterval`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitUnlimited`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitFailureStates`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitAttribute`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitCallback`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageRecordWaitHeaders`

- SDK-owned WaitForDelete supplies a two-second interval and120-second SDK timeout. Actual404, successful nil or deleted status complete with error-only success. An ERROR observation does not itself complete or fail this deletion waiter; caller failure-state lists do not change deletion-state policy.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitDeadlinesDefaultsParentsAndCallerOrdering`, `api/service_wait_contracts_test.go:TestServiceWaitContractsDefaultDeadlinesAndConvenienceTargets`, `internal/servicewait/wait_test.go:TestServiceWaitDeleteObservesAbsenceWithoutFailureOrMutation`, `api/service_wait_contracts_test.go:TestServiceWaitContractsDeletionOnlyObservesAndRejectsTerminalErrors`, `resource/wait_attributes_test.go:TestWaitDeletedNilSuccessfulFetchIsTerminalWithoutCallback`.
- Deletion waiting sends GET observations and optional initial Name resolution only, never a DELETE request. Name is resolved once without reporting its resolution List to the callback, then the first fixed-ID GET and later nonterminal GETs can report progress. Changed response IDs cannot retarget the poll.
  Tests: `api/service_wait_contracts_test.go:TestServiceWaitContractsDeletionOnlyObservesAndRejectsTerminalErrors`, `internal/servicewait/wait_test.go:TestServiceWaitDeleteObservesAbsenceWithoutFailureOrMutation`, `resource/wait_identity_test.go:TestWaitKeepsExplicitIDWhenResponseIdentityChanges`.
- SDK defaults are prepended and concrete caller resource.WaitOption values apply last. WithUnlimitedWait removes a finite default, a later positive WithTimeout reinstates a bound, and parent context deadlines and cancellation remain effective. No application status adapter or polling builder is required.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitDeadlinesDefaultsParentsAndCallerOrdering`, `api/service_wait_contracts_test.go:TestServiceWaitContractsDefaultDeadlinesAndConvenienceTargets`.
- The default polling interval is two seconds and a positive caller interval replaces it. Nil context or collection, invalid refs/targets/options, unsupported selected attributes and canceled context fail without HTTP. Callback cancellation and active-fetch cancellation stop before any further poll.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitDefaultTwoSecondIntervalAndCallerShortening`, `internal/servicewait/wait_test.go:TestServiceWaitPreflightAndCallerCancellationBeforeAnotherFetch`, `api/service_wait_contracts_test.go:TestServiceWaitContractsPreflightAndCallerCancellation`.
- Status selection uses an exported string field and can also select deletion completion. Server canonical integer progress reports native values; absent canonical progress reports zero. Cinder-v3 Snapshot uses the noncanonical os-extended-snapshot-attributes:progress string tag, so callback reports zero without parsing that string. Successful terminal, failure and HTTP-error observations do not invoke the callback.
  Tests: `internal/servicewait/wait_test.go:TestServiceWaitReusableCallerOptionsRemainOwnedAndSelectAttributes`, `api/service_wait_contracts_test.go:TestServiceWaitContractsStatusAttributesAndNativeProgressLimits`, `api/service_wait_contracts_test.go:TestServiceWaitContractsNameBindingLiveClientAndProgress`, `resource/wait_attributes_test.go:TestWaitProgressCallbackCancellationStopsFurtherGETs`, `resource/wait_attributes_test.go:TestWaitProgressOmitsSuccessFailureAndHTTPError`.
- Explicit ID is stable even when response ID differs. Existing Name lookup resolves once, retains the native list/name-query boundary, and state waiting reports its initial List observation before fixed-ID GETs. Selected ResourceBase/reverse prefix, configured ordinary headers/microversion, caller context/middleware and the original live Provider token remain in use; no response-driven route or model synthesis occurs.
  Tests: `api/service_wait_contracts_test.go:TestServiceWaitContractsNameBindingLiveClientAndProgress`, `resource/wait_identity_test.go:TestWaitKeepsExplicitIDWhenResponseIdentityChanges`, `api/service_wait_contracts_test.go:TestServiceWaitContractsDeletionOnlyObservesAndRejectsTerminalErrors`.
- Terminal failures remain failures rather than deletion completion. The shared read gate preserves original terminal URL/accepted-response/JSON/read/context causes even when nested native404 or adapter-owned ErrNotFound is discoverable. Actual direct403 or transport-wrapped404 terminates the service deletion observation after one request. Successful missing state fetches return a nonpanic error; successful nil deletion fetches complete.
  Tests: `api/service_wait_contracts_test.go:TestServiceWaitContractsDeletionOnlyObservesAndRejectsTerminalErrors`, `resource/read_errors_test.go:TestReadErrorsNeverAddMissingToTerminalDirectOrQueryCauses`, `resource/read_errors_test.go:TestReadErrorsDefendIgnoreMissingAndWaitAgainstAdapterPrewrappedTerminalCauses`, `resource/wait_policy_test.go:TestWaitMissingSuccessfulResponseDoesNotPanic`, `resource/wait_attributes_test.go:TestWaitDeletedNilSuccessfulFetchIsTerminalWithoutCallback`.
- Existing common Collection.Wait/WaitDeleted and generated WaitFor/WaitForDeletion retain their five-minute deadline and original binding failure predicate. Native WaitForStatus retains its error-only callable and native predicate-first behavior, including original caller context; source one-second/exact-case behavior is not replaced by this service policy. This is preservation evidence, not a separate native-operation status promotion.
  Tests: `api/service_wait_contracts_test.go:TestServiceWaitContractsLegacyAndNativeSignaturesStayUnchanged`, `internal/servicewait/wait_test.go:TestServiceWaitDeadlinesDefaultsParentsAndCallerOrdering`.
- Generator discovery records only the six audited Server/Cinder Volume-Snapshot/Image-v2 collections with their service wait policy. Existing generated common WaitFor/WaitForDeletion still delegate unchanged; unrelated Backup, Flavor, Task, network and Image-v1 collections receive no inferred service-default policy.
  Tests: `internal/cmd/sdkgen/service_waits_test.go:TestServiceWaitPoliciesBindOnlyActualStatusCollections`, `internal/cmd/sdkgen/service_waits_test.go:TestServiceWaitPolicyRejectsParentDriftAndDoesNotInferOtherModels`.
- Owned WaitForImageRecordDelete always begins with fresh fixed-ID GET, including an initially deleted seed, and never sends DELETE or List. Defaults remain120-second SDK timeout and two-second polling; this additive record-return API preserves existing typed/error-only/native policies.
  Tests: `image/image_record_wait_test.go:TestImageRecordWaitDeleteFreshFetchAnd404OwnedReturn`, `internal/servicewait/wait_test.go:TestServiceWaitDeadlinesDefaultsParentsAndCallerOrdering`.
- A freshly fetched string deleted completes owned deletion waiting under Unicode16 full lowercase comparison. A fetched null/nonstring status is a processing error. Ordinary failure states do not complete or fail deletion; actual200..399 fetches retain declared projection and receipt policy.
  Tests: `image/image_record_wait_test.go:TestImageRecordWaitAcceptedJSONToleranceAndBodyStateErrors`, `image/image_record_wait_test.go:TestImageRecordWaitDeleteFreshFetchAnd404OwnedReturn`, `internal/servicewait/wait_test.go:TestServiceWaitDeleteObservesAbsenceWithoutFailureOrMutation`.
- First clean final native404 returns an independent initial seed clone;404 after successful GETs returns the latest private overlay record. No404 response body/header/status is synthesized into a successful Image receipt, matching the visible latest-data effect of Python orig_resource being the same mutable fetched object without mutating the Go caller.
  Tests: `image/image_record_wait_test.go:TestImageRecordWaitDeleteFreshFetchAnd404OwnedReturn`.
- Read/Close, context, sticky-source, transport and retry-hook failures associated with404 are terminal rather than deletion completion. Native reauthentication/retry remains active, unchanged plain native RetryFunc errors remain clean, and canceled GETs preserve physical response evidence alongside the context cause. Other rejected statuses keep native HTTP failure policy.
  Tests: `image/image_record_wait_test.go:TestImageRecordWaitHTTPFailuresAnd404HandlingFaultsAreTerminal`, `image/image_record_wait_test.go:TestImageRecordWaitAcceptedAndRejectedStickyPhysicalGuards`, `resource/wait_deleted_guard_test.go:TestWaitDeletedCanceledReadRetainsPhysicalResponseEvidence`.
- Owned deletion uses one captured source/location/header preparation and live Provider authentication through every poll. The escaped input ID stays fixed even when a response id changes or becomes null. Computed location and receipt metadata are excluded from the Body64 polling seed, and all returned channels remain independently owned.
  Tests: `connection_image_record_waits_test.go:TestConnectionImageRecordWaitsShareOwnedSourceAndLatestObservation`, `image/image_record_wait_test.go:TestImageRecordWaitDeleteFreshFetchAnd404OwnedReturn`, `image/image_record_wait_test.go:TestImageRecordWaitStatusOwnsOneBudgetLocationHeadersAndLiteralID`.
- One owned deletion budget covers GET, progress and pause, supports timeout0, interval0 and explicit unlimited selection, and retains parent cancellation/deadline. Completion/errors omit progress. Opt-in common deletion guards check before GET, after read before completion and after progress, preserving fetch evidence and existing nil-guard behavior; later errors return the last successfully projected owned record.
  Tests: `image/image_record_wait_test.go:TestImageRecordWaitZeroIntervalsTimeoutAndParentBudget`, `image/image_record_wait_test.go:TestImageRecordWaitCallbacksGuardBeforePauseOrNextGET`, `resource/wait_deleted_guard_test.go:TestWaitDeletedOptInGuardProtectsEveryCompletionBoundary`, `resource/wait_deleted_guard_test.go:TestWaitDeletedGuardPreservesFetchFailureAndLegacyNilGuard`.

Python/Go differences:

- Go accepts explicit resource.Ref and an SDK typed collection and returns a native-model pointer for state waits; Python accepts an existing Resource and may return it immediately from its cached target state without HTTP. Explicit ID performs a fresh lookup and Name retains the existing native resolver before fixing its ID.
- Python nullable status/descriptor access and arbitrary progress values differ from Go exported string/integer model fields. Cinder-v3 Snapshot progress is a native string under os-extended-snapshot-attributes:progress, which does not match the canonical progress selector; callback reports zero without coercing that string. Other absent progress is also reported as zero. Successful missing/null native status fields may already collapse during typed decoding.
- Go requires positive poll intervals and positive bounded timeouts; WithUnlimitedWait represents no SDK timeout explicitly. Python iterate_timeout zero/negative timeout and interval-zero coercion are not reproduced.
- WaitForDelete returns error only, while Python returns an original or fetched Resource. Native HTTP/transport/decoder/context causes remain visible rather than being converted into a synthetic deleted model.
- These service-default methods supplement existing five-minute Collection/leaf waits and preserve native WaitForStatus context-only one-second exact-case/error-only behavior. Create/upload and Cloud workflow policies are separate.
- State waiting can report the initial Name List model; deletion waiting resolves Name without a callback and starts progress at its first fixed-ID GET. An explicit status attribute for deletion is a Go extension; pinned Python delete observes its conventional status field.
- Earlier contracts/differences above describe the preserved Ref/native-model service policies. The additional named immutable ImageRecord profile implements cached initial-target selection, nullable declared status, fresh projection, Python-style zero controls and record-return behavior in the new contracts; the earlier typed limitations do not describe the new record methods.
- Owned ImageRecord waits preserve caller snapshots rather than mutating a supplied Resource instance. Arbitrary Resource subclasses, dirty/component lifecycle and same-instance identity remain SDK-R1; configured Adapter cache/conditional GET remains SDK-C1; dynamic session/adapter/base_path/microversion replacement remains SDK-S1 in docs/implementation-plan.md. These shared SDK objectives remain active and this named mapping does not imply full Python runtime or cloud equivalence.
- Go keeps the escaped nonblank input ID fixed rather than following response-ID mutation into a later route. Initial-target/first404 return preserves seed location; each fresh poll projects the location captured for this operation, whereas Source Image fetch without project_id/availability_zone can retain its previous computed location. Options/source/descriptor preflight runs even for an initial-target shortcut.
- The owned profile takes valid UTF-8 string targets and canonical declared Body64/location/image_import_methods attribute selectors; nullable targetNone, arbitrary Python attribute/progress objects, lone-surrogate JSON/string behavior and other parser/runtime domains are outside this finite profile. Comparisons pin Unicode16 full lowercase; other Python runtime Unicode databases are a documented boundary. Go rejects negative durations, uses monotonic context budgets and explicit unlimited helpers, and cannot interrupt a blocked synchronous callback.
- Image-v1, wait_for_task and Cloud wait_for_image/wait_for_server are separate declarations and are not incidentally promoted by this operation. The existing legacy Ref name-resolution/typed decoder boundary remains preserved alongside literal owned ImageRecord input.

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

### `gophercloud:image/v2/images.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.RawClient`

- Generated API.Get retains the exact RawClient and sends one GET to native ServiceURL(ResourceBase or Endpoint, images, raw id) with a nil body, default Accept, service MoreHeaders, microversion and live token. Empty IDs, slashes and query delimiters are forwarded without validation or escaping.
  Tests: `image/v2/images/get_contracts_test.go:TestNativeImageGetRoutesHeadersAndNativeProjection`.
- A 200 response uses the pinned Extract header merge and Image.UnmarshalJSON: import methods and store IDs are trimmed and comma split without empty fields, numeric size becomes int64 and null size zero, undeclared keys except self/size/header keys become Properties with the body properties string retained, and times decode to time.Time.
  Tests: `image/v2/images/get_contracts_test.go:TestNativeImageGetRoutesHeadersAndNativeProjection`.
- Only 200 is accepted; other statuses keep the native ErrUnexpectedResponseCode with Expected [200] as the direct cause of a Get/images OperationError without an owned receipt. Decode errors keep native partial Extract results, and caller cancellation stays inspectable.
  Tests: `image/v2/images/get_contracts_test.go:TestNativeImageGetStrictStatusAndDecodeErrors`.

Python/Go differences:

- This declaration maps Gophercloud native Get only; it is not Python get_image parity, owned GetImageRecord descriptor/location projection or Cloud get_image_by_id behavior.
- The SDK adds only request.Wrap operation context; native URL construction, status policy, retries, reauthentication and partial Extract results are unchanged.

### `gophercloud:image/v2/images.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.List`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithListQuery`

- Generated API.List serializes ListOpts through the pinned ToImageListQuery, omitting zero values, repeating tag, formatting created/updated date filters and silently dropping int64 SizeMin/SizeMax in BuildQueryString, then appends WithListQuery extension values; zero options send no query.
  Tests: `image/v2/images/list_contracts_test.go:TestNativeImageListQueryAndExtensions`.
- The lazy SDK stream follows native next links rebased onto the client ServiceURL path, stops on an empty images page even with next, surfaces the native EOF for a bodyless 204, yields earlier rows before a late native HTTP error, returns row decode errors and sends no further page after the caller stops.
  Tests: `image/v2/images/list_contracts_test.go:TestNativeImageListPagingStopsAndErrors`.

Python/Go differences:

- This declaration maps Gophercloud native List only; it is not Python images() descriptor/local-filter/location parity or Cloud list_images behavior.
- The SDK exposes the native pager as iter.Seq2 and keeps native query construction, including the int64 size filter omission; callers can send size_min/size_max with WithListQuery. Iteration errors are native and only option preparation errors receive List/images context.

### `gophercloud:image/v2/images.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithCreateField`

- Generated API.Create serializes CreateOpts with the pinned BuildRequestBody, omitting zero values while sending explicit pointer false, then copies Properties over declared keys and appends non-colliding WithCreateField extension JSON with exact raw bytes before one POST to native images with JSON headers and service MoreHeaders.
  Tests: `image/v2/images/create_contracts_test.go:TestNativeImageCreateBodyPropertiesExtensionsAndProjection`.
- A 201 response uses the shared Extract header merge and Image.UnmarshalJSON projection. Missing required name, nil options and extensions colliding with core or existing body keys fail before HTTP; other statuses keep native ErrUnexpectedResponseCode with Expected [201] under Create/images context without an owned receipt.
  Tests: `image/v2/images/create_contracts_test.go:TestNativeImageCreateBodyPropertiesExtensionsAndProjection`, `image/v2/images/create_contracts_test.go:TestNativeImageCreatePreflightAndStrictStatus`.

Python/Go differences:

- This declaration maps Gophercloud native metadata Create only; it is not Python create_image parity, which CreateImageRecord and CreateCloudImageRecord cover.
- WithCreateField is a Go extension that rejects collisions instead of overwriting; native body construction, OkCodes and response projection are unchanged.

## References

- [Service Waits](../../../docs/service-waits.md)
- [Update](update.md)
- [Create](create.md)
- [List](list.md)
- [Get](get.md)
- [Delete](delete.md)
- [Image Record Delete](../../image-record-delete.md)
