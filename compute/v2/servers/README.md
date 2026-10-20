# Servers

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:compute/v2/servers.GetPassword`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.GetPassword`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithGetPasswordPrivateKey`

- Default returns the encrypted password; optional RSA decryption and nil-key reset use SDK-owned options.
  Tests: `api/password_contracts_test.go:TestServerPasswordDefaultAndOptionalDecryption`.
- Empty, missing and null passwords become an empty string; typed decoding and HTTP errors retain the original cause.
  Tests: `api/password_contracts_test.go:TestServerPasswordHandlesEmptyResponsesAndErrors`.

Python/Go differences:

- Go returns a string and wrapped error directly instead of PasswordResult; optional RSA decryption is selected with a concrete function option.

### `python:compute/v2/get_server_password`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.GetPassword`

- Default returns the encrypted password; optional RSA decryption and nil-key reset use SDK-owned options.
  Tests: `api/password_contracts_test.go:TestServerPasswordDefaultAndOptionalDecryption`.
- Empty, missing and null passwords become an empty string; typed decoding and HTTP errors retain the original cause.
  Tests: `api/password_contracts_test.go:TestServerPasswordHandlesEmptyResponsesAndErrors`.

Python/Go differences:

- Python accepts a server ID or mutable Server; Go accepts an explicit server ID and returns (string, error), with optional SDK-owned RSA PKCS#1 v1.5 decryption. No server lookup or plaintext decryption is implied by the default call.
- Python missing/null password values return None and its typing cast does not validate other JSON value types. Go missing/null values become an empty string and non-string password values produce the native typed decode error.
- Python Server.get_password uses an explicit session default or Server maximum-supported microversion discovery (Server ceiling2.100). Go preserves the configured Gophercloud service-client microversion without discovery or an automatic upgrade.
- Python raise_from_response accepts statuses below400 before JSON parsing. Go keeps native GetPassword GET200 acceptance and native JSON/HTTP error causes wrapped with GetPassword/servers; this scalar result does not expose a mutable Resource or an owned raw receipt.

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

### `python:compute/v2/fetch_server_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Metadata`

- The server-ID metadata read makes one GET at the selected ResourceBase/reverse-prefix servers/{id}/metadata route, with no query/body or detail/fallback request. Current token, configured header and both selected Compute2.55 version headers reach the request; the selected client remains unchanged.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeServerMetadataReadNativeContract`.
- Native metadata extraction preserves Unicode and empty string values, nonnil empty object maps and missing/null nil maps. A null map member becomes the native empty string. Wrong metadata shape or nonstring values preserve the typed decode cause and native partial-map ABI; malformed JSON returns its original error.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeServerMetadataReadNativeContract`.
- Only native GET200 is accepted. Actual403/404/unexpected203 remain errors with native actual/expected status, GET method, exact URL, body and response header, and no unrelated request or missing-success conversion.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeServerMetadataReadNativeContract`.

Python/Go differences:

- Pinned fetch_server_metadata mutates and returns a seeded Server Resource. Go accepts an explicit server ID and returns a stateless map[string]string projection, with no supplied Server mutation or detail GET. Native missing/null wrapper is nil and an explicit empty object is nonnil; null members become empty strings while nonstring values may return a partial map with a type error. Python arbitrary dictionary/descriptor/seed behavior is not claimed.
- The selected native client policy and strict200 status are preserved rather than adapter-wide Python successful statuses or automatic discovery. Native success results do not add RawResource/Wire/header fields; HTTP errors retain original evidence. Full mutable Resource/session/cache state remains a separate SDK declaration scope.

### `python:compute/v2/get_server_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Metadata`

- The pinned get_server_metadata alias delegates to fetch_server_metadata; both names map to the same server-ID Metadata GET. The actual SDK request preserves the selected reverse ResourceBase, live token, configured header and both Compute2.55 version headers, with no query/body or preliminary server-detail request.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeServerMetadataReadNativeContract`.
- The shared native string-map result preserves Unicode/empty values, nonnil empty maps, nil missing/null wrappers and null-member empty strings. Wrong shapes/nonstring fields preserve the typed cause and native partial result; malformed JSON remains an error.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeServerMetadataReadNativeContract`.
- Native strict GET200 is retained. Actual403/404/unexpected203 expose original actual/expected status, method, URL, body/header and make no fallback or unrelated request; selected service configuration remains unchanged.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeServerMetadataReadNativeContract`.

Python/Go differences:

- Pinned get_server_metadata is the deprecated4.14 alias of fetch_server_metadata and emits a Python warning. Go exposes one Metadata method and does not reproduce the language-specific alias warning; distinct source IDs/fingerprints retain the two named declarations with the same actual HTTP evidence.
- Go accepts an explicit server ID and returns a stateless map[string]string instead of mutating/returning a seeded Server Resource. Native nullable wrapper/member, typed partial error and strict200 behavior are documented Go alternatives; no Python arbitrary mapping/descriptor/adapter-wide status or mutable Resource/session/cache behavior is promoted by this alias.

### `python:compute/v2/get_server_console_output`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ConsoleOutput`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithConsoleOutputLength`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithConsoleOutputOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithConsoleOutputField`

- The owned ConsoleOutput action uses the selected reverse/resource-base servers/{id}/action route with no preliminary GET or query. Omitted/nil length sends an empty action object; explicit0, positive and negative integers are transmitted literally. Default Accept is empty like the source action, while current authentication, configured source header and selected Compute2.87 are preserved. Reused helper options apply once per call and do not replace the native service/provider.
  Tests: `api/compute_console_output_test.go:TestComputeConsoleOutputLiteralLengthOwnedOptionsAndNativeCompatibility`.
- Typed length pointers and JSON extension input are owned snapshots, including custom callback values between options. Last individual length wins; bulk zero options restore omission. Core length extension, empty/non-JSON fields, nil option and unsupported per-call query/header/arguments fail before HTTP. Existing ShowConsoleOutput retains its signature and native0 omission while owned0 is explicit.
  Tests: `api/compute_console_output_test.go:TestComputeConsoleOutputLiteralLengthOwnedOptionsAndNativeCompatibility`.
- Owned final200..399 action statuses include actual200/201/203/300-without-Location. The native output-string projection preserves Unicode/control characters and native case-insensitive field matching; missing/null/empty output yields empty string. Nonstring output, null/array root, malformed/invalid-UTF8 JSON and empty204 fail with accepted status/body/header proof and no accepted-processing retry. Actual400/403/404 and native ShowConsoleOutput203 rejection keep original method/URL/body/header and native expected status policy.
  Tests: `api/compute_console_output_test.go:TestComputeConsoleOutputLiteralLengthOwnedOptionsAndNativeCompatibility`, `api/compute_console_output_test.go:TestComputeConsoleOutputStringProjectionAndHTTPResponseEvidence`.
- Nil API/context, invalid explicit ID, pre-cancelled custom cause and option cancellation/source/API replacement stop before later callbacks or HTTP while preserving original causes. Accepted read/Close-with-nested404 and source/context failures keep actual200 body/header/status, original cause and one physical action rather than missing success or SDK resend. Native401 reauthentication retains the literal body and selected version with two physical requests, one reauth and one custom option application.
  Tests: `api/compute_console_output_test.go:TestComputeConsoleOutputSourceContextAndAcceptedBodyBoundaries`.

Python/Go differences:

- Python accepts server ID/Server Resource and returns the entire decoded response dict. Go requires a validated single URL path-segment ID with valid text/no controls and returns the native output string projection. Missing/null output becomes empty string, native case-insensitive matching is retained, nonstring output and nonnull-UTF8-object violations are explicit errors. Unknown response fields are not exposed through this string result.
- Concrete nullable integer Length preserves the source omitted-versus-explicit0 distinction and forwards negative values literally. Go excludes arbitrary dynamic input coercion and supplies library-owned pointer/JSON snapshots, extension/core-field protection and preflight/source/context guards. Per-call session/query/header/argument overrides are not part of this named API; selected client policy is preserved.
- Owned ConsoleOutput accepts final200..399 like the source action and adds strict object/string decoding with accepted response evidence and no SDK processing-error replay. Native ShowConsoleOutput remains strict200 with its int omitempty and signature unchanged. Provider native retry/reauthentication remains distinct from SDK composition and the general mutable Resource/session/cache goal. No remote-console creation, server lifecycle or unrelated native operation is promoted by this row.

### `python:compute/v2/get_server_console_url`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ConsoleURL`

- ConsoleURL preserves mandatory exact console type and all six legacy action mappings: novnc/xvpvnc to os-getVNCConsole, spice-html5/spice-direct to os-getSPICEConsole, rdp-html5 to os-getRDPConsole and serial to os-getSerialConsole. It performs one fixed selected-ResourceBase POST at servers/{serverID}/action with the exact type body, no query and empty Accept, retaining live token/source headers and selected version. Selected2.100, absent version and spice-direct2.5 remain legacy without discovery, modern fallback or a new local gate. Empty/webmks/unknown/case or whitespace variants fail before HTTP.
  Tests: `api/compute_console_url_test.go:TestComputeConsoleURLLegacyActionsSelectedSourceAndUnsupportedTypes`.
- The actual root object must contain the exact console key. Its object/empty object/null/array/string/false/zero value is returned as independently copied json.RawMessage without a type/url model, input seed, coercion or unknown/large-number loss. Actual response type may differ from the requested type; literal Unicode, escaped controls and URL text remain raw. Owned final200..399 accepts actual200/201/203 and300 without Location.
  Tests: `api/compute_console_url_test.go:TestComputeConsoleURLRawValueAndResponseEvidence`, `api/compute_console_url_test.go:TestComputeConsoleURLLegacyActionsSelectedSourceAndUnsupportedTypes`.
- Missing console or wrong-case key, null/array/string root, empty204, opaque/malformed accepted JSON and invalid UTF8 remain receipt errors rather than nil console success. Accepted errors retain original body/header/status and do not invoke native retry; this legacy getter does not borrow creation JSON tolerance. HTTP400/403/404 preserve native actual/expected200..399 status, POST method, exact URL and original body/header, with no lookup or modern fallback.
  Tests: `api/compute_console_url_test.go:TestComputeConsoleURLRawValueAndResponseEvidence`.
- Nil API/client/context, unsafe server ID, pre-canceled custom-cause context and an outer operation guard fail with zero HTTP. Accepted read, nested404 Close, source drift and cancellation retain physical203 receipt and original causes; nested404 is not ErrNotFound. Accepted processing failures cause no SDK rePOST and the shared fault wrappers confirm one physical request/read/Close where applicable.
  Tests: `api/compute_console_url_test.go:TestComputeConsoleURLSourceContextAndAcceptedFailures`.

Python/Go differences:

- Python accepts a Server Resource or ID and returns the actual dynamically shaped console value. Go requires a validated explicit single-path-segment/text server ID and mandatory type, returning an independently owned json.RawMessage; Python None is represented by raw JSON null. Missing console is an operation-scoped ResponseError rather than Python KeyError, while null/scalar/array values remain successful exact values.
- This legacy getter itself preserves the selected client microversion without advertised discovery or Server2.100 maximum selection. The separate compute.Service.CreateConsole provides advertised/selected modern-versus-legacy dispatch and is reviewed under python:compute/v2/create_console. validate_console_auth_token, cloud console wrappers and general Resource/session lifecycle are not promoted by this getter row.
- Go rejects invalid-UTF8 response bodies with a receipt and does not reproduce Python response charset detection or replacement-character decoding. JSON shape/selection/read/Close/source/context failures preserve actual evidence through Go errors; accepted processing errors are not retried. This strict getter policy is independent from the tolerant accepted-nonJSON policy of CreateConsole/CreateKeypair.

### `gophercloud:compute/v2/servers.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.RawClient`

- Get sends GET servers/{raw id} with service MoreHeaders and live token, accepts 200 and 203, decodes Server with an empty image string as nil Image, and keeps other statuses as native errors under Get/servers context.
  Tests: `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCRUDRoutesBodiesAndDecode`, `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCRUDStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server call only; Python compute server Resource parity and the owned create/wait workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, paging, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.List`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithListQuery`

- List GETs servers/detail with the ListOpts query including all_tenants and extension values, follows the servers_links next href verbatim and sends no further page after the caller stops.
  Tests: `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerListsQueryPagingAndStop`.

Python/Go differences:

- This declaration maps the Gophercloud native server call only; Python compute server Resource parity and the owned create/wait workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, paging, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.ListSimple`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ListSimple`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithListSimpleOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithListSimpleQuery`

- ListSimple GETs servers with the same ListOpts query and follows the servers_links next href verbatim.
  Tests: `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerListsQueryPagingAndStop`.

Python/Go differences:

- This declaration maps the Gophercloud native server call only; Python compute server Resource parity and the owned create/wait workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, paging, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithUpdateOptions`

- Update PUTs {"server": ...} with omitted empty name/access IPs and an optional hostname, returns the decoded server and accepts only 200.
  Tests: `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCRUDRoutesBodiesAndDecode`, `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCRUDStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server call only; Python compute server Resource parity and the owned create/wait workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, paging, retries and decoding are unchanged.

### `gophercloud:compute/v2/servers.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Delete`

- Delete sends DELETE servers/{raw id} and accepts the native default 202 and 204.
  Tests: `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCRUDRoutesBodiesAndDecode`, `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCRUDStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native server call only; Python compute server Resource parity and the owned create/wait workflows are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes, paging, retries and decoding are unchanged.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
encrypted, err := service.Servers.GetPassword(ctx, serverID)
if err != nil { return err }
_ = encrypted

password, err := service.Servers.GetPassword(ctx, serverID,
    servers.WithGetPasswordPrivateKey(privateKey))
if err != nil { return err }
_ = password
```

## References

- [User Read APIs](../../user-read-apis.md)
- [Keypairs Console](../../keypairs-console.md)
- [Python Server.get_password](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server.py)
- [Resource._get_microversion](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [Service Waits](../../../docs/service-waits.md)
- [Native](native.md)
