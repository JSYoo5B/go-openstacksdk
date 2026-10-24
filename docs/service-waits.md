# Service Waits

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/wait_for_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.WaitForServer`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.WaitForServerState`
- `github.com/JSYoo5B/go-openstacksdk/compute.Servers.WaitForServer`
- `github.com/JSYoo5B/go-openstacksdk/compute.Servers.WaitForServerState`

- WaitForServer supplies ACTIVE, exact case-insensitive ERROR, two-second polling and120-second SDK timeout. WaitForServerState keeps the same server-specific timeout with an explicit target; it is distinct from unlimited generic Compute WaitForState. The returned model is the actual native observation, not a cached supplied Python Resource.
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
- The default call polls GET on the fixed resource path only, never mutates it, and applies the Python defaults: two-second interval, 120 seconds for server and delete waits, no SDK timeout for the generic status wait, and ERROR as the failure state.
  Tests: `api/service_wait_contracts_test.go:TestServiceWaitContractsDefaultDeadlinesAndConvenienceTargets`.

Python/Go differences:

- Go accepts explicit resource.Ref and an SDK typed collection and returns a native-model pointer for state waits; Python accepts an existing Resource and may return it immediately from its cached target state without HTTP. Explicit ID performs a fresh lookup and Name retains the existing native resolver before fixing its ID.
- Python nullable status/descriptor access and arbitrary progress values differ from Go exported string/integer model fields. Cinder-v3 Snapshot progress is a native string under os-extended-snapshot-attributes:progress, which does not match the canonical progress selector; callback reports zero without coercing that string. Other absent progress is also reported as zero. Successful missing/null native status fields may already collapse during typed decoding.
- Go requires positive poll intervals and positive bounded timeouts; WithUnlimitedWait represents no SDK timeout explicitly. Python iterate_timeout zero/negative timeout and interval-zero coercion are not reproduced.
- WaitForDelete returns error only, while Python returns an original or fetched Resource. Native HTTP/transport/decoder/context causes remain visible rather than being converted into a synthetic deleted model.
- These service-default methods supplement existing five-minute Collection/leaf waits and preserve native WaitForStatus context-only one-second exact-case/error-only behavior. Create/upload and Cloud workflow policies are separate.
- State waiting can report the initial Name List model; deletion waiting resolves Name without a callback and starts progress at its first fixed-ID GET. An explicit status attribute for deletion is a Go extension; pinned Python delete observes its conventional status field.
- Python returns the supplied Resource without HTTP when its cached status already matches the target; Go always performs a fresh GET first.
- Python accepts any Resource the compute proxy can fetch; Go waits through typed resource.Collection values (compute.WaitForState and compute.WaitForDelete accept any collection).
- Python raises ResourceTimeout and ResourceFailure; Go returns context deadline and wait failure errors.

### `python:compute/v2/wait_for_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.WaitForState`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.WaitForState`
- `github.com/JSYoo5B/go-openstacksdk/compute.Servers.WaitForState`

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
- The default call polls GET on the fixed resource path only, never mutates it, and applies the Python defaults: two-second interval, 120 seconds for server and delete waits, no SDK timeout for the generic status wait, and ERROR as the failure state.
  Tests: `api/service_wait_contracts_test.go:TestServiceWaitContractsDefaultDeadlinesAndConvenienceTargets`.

Python/Go differences:

- Go accepts explicit resource.Ref and an SDK typed collection and returns a native-model pointer for state waits; Python accepts an existing Resource and may return it immediately from its cached target state without HTTP. Explicit ID performs a fresh lookup and Name retains the existing native resolver before fixing its ID.
- Python nullable status/descriptor access and arbitrary progress values differ from Go exported string/integer model fields. Cinder-v3 Snapshot progress is a native string under os-extended-snapshot-attributes:progress, which does not match the canonical progress selector; callback reports zero without coercing that string. Other absent progress is also reported as zero. Successful missing/null native status fields may already collapse during typed decoding.
- Go requires positive poll intervals and positive bounded timeouts; WithUnlimitedWait represents no SDK timeout explicitly. Python iterate_timeout zero/negative timeout and interval-zero coercion are not reproduced.
- WaitForDelete returns error only, while Python returns an original or fetched Resource. Native HTTP/transport/decoder/context causes remain visible rather than being converted into a synthetic deleted model.
- These service-default methods supplement existing five-minute Collection/leaf waits and preserve native WaitForStatus context-only one-second exact-case/error-only behavior. Create/upload and Cloud workflow policies are separate.
- State waiting can report the initial Name List model; deletion waiting resolves Name without a callback and starts progress at its first fixed-ID GET. An explicit status attribute for deletion is a Go extension; pinned Python delete observes its conventional status field.
- Python returns the supplied Resource without HTTP when its cached status already matches the target; Go always performs a fresh GET first.
- Python accepts any Resource the compute proxy can fetch; Go waits through typed resource.Collection values (compute.WaitForState and compute.WaitForDelete accept any collection).
- Python raises ResourceTimeout and ResourceFailure; Go returns context deadline and wait failure errors.

### `python:compute/v2/wait_for_delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/compute.Servers.WaitForDelete`

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
- The default call polls GET on the fixed resource path only, never mutates it, and applies the Python defaults: two-second interval, 120 seconds for server and delete waits, no SDK timeout for the generic status wait, and ERROR as the failure state.
  Tests: `api/service_wait_contracts_test.go:TestServiceWaitContractsDefaultDeadlinesAndConvenienceTargets`.

Python/Go differences:

- Go accepts explicit resource.Ref and an SDK typed collection and returns a native-model pointer for state waits; Python accepts an existing Resource and may return it immediately from its cached target state without HTTP. Explicit ID performs a fresh lookup and Name retains the existing native resolver before fixing its ID.
- Python nullable status/descriptor access and arbitrary progress values differ from Go exported string/integer model fields. Cinder-v3 Snapshot progress is a native string under os-extended-snapshot-attributes:progress, which does not match the canonical progress selector; callback reports zero without coercing that string. Other absent progress is also reported as zero. Successful missing/null native status fields may already collapse during typed decoding.
- Go requires positive poll intervals and positive bounded timeouts; WithUnlimitedWait represents no SDK timeout explicitly. Python iterate_timeout zero/negative timeout and interval-zero coercion are not reproduced.
- WaitForDelete returns error only, while Python returns an original or fetched Resource. Native HTTP/transport/decoder/context causes remain visible rather than being converted into a synthetic deleted model.
- These service-default methods supplement existing five-minute Collection/leaf waits and preserve native WaitForStatus context-only one-second exact-case/error-only behavior. Create/upload and Cloud workflow policies are separate.
- State waiting can report the initial Name List model; deletion waiting resolves Name without a callback and starts progress at its first fixed-ID GET. An explicit status attribute for deletion is a Go extension; pinned Python delete observes its conventional status field.
- Python returns the supplied Resource without HTTP when its cached status already matches the target; Go always performs a fresh GET first.
- Python accepts any Resource the compute proxy can fetch; Go waits through typed resource.Collection values (compute.WaitForState and compute.WaitForDelete accept any collection).
- Python raises ResourceTimeout and ResourceFailure; Go returns context deadline and wait failure errors.

### `python:block_storage/v2/wait_for_status`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WaitForState`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WaitForAvailable`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.WaitForState`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.WaitForAvailable`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.API.WaitForState`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.API.WaitForAvailable`

- Cinder WaitForState requires a target and WaitForAvailable supplies available; both apply exact case-insensitive error, two-second polling and no SDK timeout. The returned model is the actual native observation, not a cached supplied Python Resource.
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

Python/Go differences:

- Go accepts explicit resource.Ref and an SDK typed collection and returns a native-model pointer for state waits; Python accepts an existing Resource and may return it immediately from its cached target state without HTTP. Explicit ID performs a fresh lookup and Name retains the existing native resolver before fixing its ID.
- Python nullable status/descriptor access and arbitrary progress values differ from Go exported string/integer model fields. Cinder-v3 Snapshot progress is a native string under os-extended-snapshot-attributes:progress, which does not match the canonical progress selector; callback reports zero without coercing that string. Other absent progress is also reported as zero. Successful missing/null native status fields may already collapse during typed decoding.
- Go requires positive poll intervals and positive bounded timeouts; WithUnlimitedWait represents no SDK timeout explicitly. Python iterate_timeout zero/negative timeout and interval-zero coercion are not reproduced.
- WaitForDelete returns error only, while Python returns an original or fetched Resource. Native HTTP/transport/decoder/context causes remain visible rather than being converted into a synthetic deleted model.
- These service-default methods supplement existing five-minute Collection/leaf waits and preserve native WaitForStatus context-only one-second exact-case/error-only behavior. Create/upload and Cloud workflow policies are separate.
- State waiting can report the initial Name List model; deletion waiting resolves Name without a callback and starts progress at its first fixed-ID GET. An explicit status attribute for deletion is a Go extension; pinned Python delete observes its conventional status field.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/wait_for_delete`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.API.WaitForDelete`

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

Python/Go differences:

- Go accepts explicit resource.Ref and an SDK typed collection and returns a native-model pointer for state waits; Python accepts an existing Resource and may return it immediately from its cached target state without HTTP. Explicit ID performs a fresh lookup and Name retains the existing native resolver before fixing its ID.
- Python nullable status/descriptor access and arbitrary progress values differ from Go exported string/integer model fields. Cinder-v3 Snapshot progress is a native string under os-extended-snapshot-attributes:progress, which does not match the canonical progress selector; callback reports zero without coercing that string. Other absent progress is also reported as zero. Successful missing/null native status fields may already collapse during typed decoding.
- Go requires positive poll intervals and positive bounded timeouts; WithUnlimitedWait represents no SDK timeout explicitly. Python iterate_timeout zero/negative timeout and interval-zero coercion are not reproduced.
- WaitForDelete returns error only, while Python returns an original or fetched Resource. Native HTTP/transport/decoder/context causes remain visible rather than being converted into a synthetic deleted model.
- These service-default methods supplement existing five-minute Collection/leaf waits and preserve native WaitForStatus context-only one-second exact-case/error-only behavior. Create/upload and Cloud workflow policies are separate.
- State waiting can report the initial Name List model; deletion waiting resolves Name without a callback and starts progress at its first fixed-ID GET. An explicit status attribute for deletion is a Go extension; pinned Python delete observes its conventional status field.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v3/wait_for_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WaitForState`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WaitForAvailable`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.WaitForState`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.WaitForAvailable`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.WaitForState`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.WaitForAvailable`
- The group, group snapshot, consistency group, consistency group snapshot, cluster, default type, group type, message, capabilities and quota class collections listed in [Proxy waits](python-proxy-waits.md#wait-collections)

- Cinder WaitForState requires a target and WaitForAvailable supplies available; both apply exact case-insensitive error, two-second polling and no SDK timeout. The returned model is the actual native observation, not a cached supplied Python Resource.
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
- Group, group snapshot, consistency group and consistency group snapshot records and clusters expose their status to blockstorage.WaitForState (case-insensitive target, error failure, deleted completion); status-less models fail before HTTP without a selected attribute.
  Tests: `api/cinder_proxy_waits_test.go:TestCinderProxyWaitsRecordAndClusterStatus`, `api/cinder_proxy_waits_test.go:TestCinderProxyWaitsDeleteEveryFetchableResource`.
- Records that keep their declared Resource view as raw JSON (group-family, message, capabilities and quota class records) accept resource.WithStatusAttribute with an exact Python attribute name, compared like a typed field and also driving the deleted check; wire spellings, absent, null and non-string values fail with resource.ErrUnsupported. Group-family records keep their default status accessor. See [raw JSON records](python-proxy-waits.md#raw-json-records).
  Tests: `api/raw_record_wait_attributes_test.go:TestRawRecordWaitAttributesCinder`, `resource/wait_raw_attributes_test.go:TestWaitRawAttributeComparesViewStringsLikeTypedFields`, `resource/wait_raw_attributes_test.go:TestWaitRawAttributeRejectsMissingNullAndNonStringValues`, `resource/wait_raw_attributes_test.go:TestWaitRawAttributeKeepsBindingStatusDefaultAndDeletion`.

Python/Go differences:

- Go accepts explicit resource.Ref and an SDK typed collection and returns a native-model pointer for state waits; Python accepts an existing Resource and may return it immediately from its cached target state without HTTP. Explicit ID performs a fresh lookup and Name retains the existing native resolver before fixing its ID.
- Python nullable status/descriptor access and arbitrary progress values differ from Go exported string/integer model fields. Cinder-v3 Snapshot progress is a native string under os-extended-snapshot-attributes:progress, which does not match the canonical progress selector; callback reports zero without coercing that string. Other absent progress is also reported as zero. Successful missing/null native status fields may already collapse during typed decoding.
- Go requires positive poll intervals and positive bounded timeouts; WithUnlimitedWait represents no SDK timeout explicitly. Python iterate_timeout zero/negative timeout and interval-zero coercion are not reproduced.
- WaitForDelete returns error only, while Python returns an original or fetched Resource. Native HTTP/transport/decoder/context causes remain visible rather than being converted into a synthetic deleted model.
- These service-default methods supplement existing five-minute Collection/leaf waits and preserve native WaitForStatus context-only one-second exact-case/error-only behavior. Create/upload and Cloud workflow policies are separate.
- State waiting can report the initial Name List model; deletion waiting resolves Name without a callback and starts progress at its first fixed-ID GET. An explicit status attribute for deletion is a Go extension; pinned Python delete observes its conventional status field.
- Python nullable descriptor/status/progress coercion and Resource-return behavior are outside the explicit Ref/native-model API (see the differences above); native name-resolution paging limits and model decoders remain unchanged.
- Image-v1, Glance wait_for_task and Cloud wait_for_server/wait_for_image workflows are separate declarations and are not promoted by this operation.
- Limits and BlockStorageSummary have no member ID; Python's inherited fetch requires one, so neither is waitable in Python or Go.
- For records that keep a raw JSON Resource view, a selected attribute is read from each observation: an attribute the view does not carry, a null value or a non-string value fails with resource.ErrUnsupported after the first GET rather than before HTTP, and a null value fails instead of being polled like Python's None until the timeout. Only group-family records select status by default; message, capabilities and quota class records need resource.WithStatusAttribute. Progress callbacks for these records report zero.

### `python:block_storage/v3/wait_for_delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroupsnapshots.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/defaulttypes.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/messages.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/capabilities.API.Resources`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.API.QuotaClassResources`

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
- Every pinned fetchable block_storage/v3 Resource with a member route (volumes, snapshots, backups, attachments, transfers, QoS specs, types, groups, group snapshots, consistency groups and snapshots, clusters, default types, group types, messages, capabilities and quota class sets) has an SDK collection; blockstorage.WaitForDelete polls each fixed member route until 404, and record collections complete on a deleted status.
  Tests: `api/cinder_proxy_waits_test.go:TestCinderProxyWaitsDeleteEveryFetchableResource`, `api/cinder_proxy_waits_test.go:TestCinderProxyWaitsRecordAndClusterStatus`.

Python/Go differences:

- Go accepts explicit resource.Ref and an SDK typed collection and returns a native-model pointer for state waits; Python accepts an existing Resource and may return it immediately from its cached target state without HTTP. Explicit ID performs a fresh lookup and Name retains the existing native resolver before fixing its ID.
- Python nullable status/descriptor access and arbitrary progress values differ from Go exported string/integer model fields. Cinder-v3 Snapshot progress is a native string under os-extended-snapshot-attributes:progress, which does not match the canonical progress selector; callback reports zero without coercing that string. Other absent progress is also reported as zero. Successful missing/null native status fields may already collapse during typed decoding.
- Go requires positive poll intervals and positive bounded timeouts; WithUnlimitedWait represents no SDK timeout explicitly. Python iterate_timeout zero/negative timeout and interval-zero coercion are not reproduced.
- WaitForDelete returns error only, while Python returns an original or fetched Resource. Native HTTP/transport/decoder/context causes remain visible rather than being converted into a synthetic deleted model.
- These service-default methods supplement existing five-minute Collection/leaf waits and preserve native WaitForStatus context-only one-second exact-case/error-only behavior. Create/upload and Cloud workflow policies are separate.
- State waiting can report the initial Name List model; deletion waiting resolves Name without a callback and starts progress at its first fixed-ID GET. An explicit status attribute for deletion is a Go extension; pinned Python delete observes its conventional status field.
- Python nullable descriptor/status/progress coercion and Resource-return behavior are outside the explicit Ref/native-model API (see the differences above); native name-resolution paging limits and model decoders remain unchanged.
- Image-v1, Glance wait_for_task and Cloud wait_for_server/wait_for_image workflows are separate declarations and are not promoted by this operation.
- Limits and BlockStorageSummary have no member ID; Python's inherited fetch requires one, so neither is waitable in Python or Go.

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

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func waitForServer(ctx context.Context, client *gophercloud.ServiceClient, id string) (*servers.Server, error) {
    return servers.New(client).WaitForServer(ctx, resource.ID(id),
        resource.WithTimeout(time.Minute))
}
```

```python
server = conn.compute.wait_for_server(server, wait=60)
```

```go
package example

import (
    "context"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func waitForVolume(ctx context.Context, client *gophercloud.ServiceClient, id string) (*volumes.Volume, error) {
    bounded, cancel := context.WithTimeout(ctx, 10*time.Minute)
    defer cancel()
    return volumes.New(client).WaitForAvailable(bounded, resource.ID(id))
}

func waitForVolumeDeletion(ctx context.Context, client *gophercloud.ServiceClient, id string) error {
    return volumes.New(client).WaitForDelete(ctx, resource.ID(id),
        resource.WithUnlimitedWait())
}
```

```python
volume = conn.block_storage.wait_for_status(volume)  # available, wait=None
conn.block_storage.wait_for_delete(volume, wait=None)
```

## References

- [Image Record Waits](../image/image-record-waits.md)
- [Server Ready](../compute/server-ready.md)
- [Resource](../resource/README.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L3244-L3349)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L2373-L2435)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3775-L3837)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L2299-L2358)
- [Tasks](../image/v2/tasks/README.md)
