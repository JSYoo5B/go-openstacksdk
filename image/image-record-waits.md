# Image Record Waits

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

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
  Tests: `image/image_record_wait_test.go:TestImageRecordWaitStatusOwnsOneBudgetLocationHeadersAndLiteralID`, `image/image_record_wait_test.go:TestImageRecordWaitCompletePreflightAndConcreteReplacement`, `connectiontest/connection_image_record_waits_test.go:TestConnectionImageRecordWaitsShareOwnedSourceAndLatestObservation`.
- Fresh owned GET accepts actual200..399 and reuses the declared Image projector with previous private Body64 as overlay seed. Invalid JSON syntax retains seed Body, valid object packing can reset properties, import-method headers are read afresh, and actual Wire/Envelope/Header/status remain independent from Resource. Nonobject/descriptor/physical processing failures retain their response evidence.
  Tests: `image/image_record_wait_test.go:TestImageRecordWaitAcceptedJSONToleranceAndBodyStateErrors`, `image/image_record_get_test.go:TestImageRecordGetPackingAliasesAndSeedOverlay`, `connectiontest/connection_image_record_waits_test.go:TestConnectionImageRecordWaitsShareOwnedSourceAndLatestObservation`.
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
  Tests: `connectiontest/connection_image_record_waits_test.go:TestConnectionImageRecordWaitsShareOwnedSourceAndLatestObservation`, `image/image_record_wait_test.go:TestImageRecordWaitDeleteFreshFetchAnd404OwnedReturn`, `image/image_record_wait_test.go:TestImageRecordWaitStatusOwnsOneBudgetLocationHeadersAndLiteralID`.
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

```python
import openstack

conn = openstack.connect(cloud="dev")
seed = conn.image.get_image("image-id")

def show_progress(progress):
    print("progress:", progress)

ready = conn.image.wait_for_status(
    seed, "active", failures=["ERROR"], interval=2, wait=None,
    attribute="status", callback=show_progress,
)
print(ready.to_dict())

last = conn.image.wait_for_delete(
    ready, interval=2, wait=120, callback=show_progress,
)
print(last.to_dict())
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
    imageID := flag.String("image-id", "", "required image ID")
    target := flag.String("target", "active", "target status")
    waitDelete := flag.Bool("wait-delete", false, "also observe deletion; does not request deletion")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *target, *waitDelete); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func printRecord(value *image.ImageRecord) error {
    body, err := json.MarshalIndent(map[string]any{
        "resource": value.Resource, "wire": value.Wire,
        "status_code": value.StatusCode, "import_methods": value.ImportMethods,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(body))
    return nil
}

func waitResult(value *image.ImageRecord, waitErr error) error {
    if waitErr != nil {
        if value != nil {
            fmt.Fprintln(os.Stderr, "wait incomplete; latest projected observation follows")
            if err := printRecord(value); err != nil { return errors.Join(waitErr, err) }
        }
        return waitErr
    }
    return printRecord(value)
}

func run(ctx context.Context, cloud, imageID, target string, waitDelete bool) error {
    if imageID == "" { return fmt.Errorf("-image-id is required") }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    seed, err := service.GetImageRecord(ctx, image.ImageRecordRequest{ID: imageID})
    if err != nil { return err }

    callback := func(progress int) { fmt.Fprintf(os.Stderr, "progress: %d\n", progress) }
    ready, err := service.WaitForImageRecordStatus(ctx, seed, target,
        image.WithImageRecordWaitTimeout(2*time.Minute),
        image.WithImageRecordWaitPollInterval(2*time.Second),
        image.WithImageRecordWaitCallback(callback),
        image.WithImageRecordWaitHeader("X-Request-Source", "image-record-waits-example"))
    if err := waitResult(ready, err); err != nil { return err }

    if waitDelete {
        last, err := service.WaitForImageRecordDelete(ctx, ready,
            image.WithImageRecordWaitCallback(callback))
        if err := waitResult(last, err); err != nil { return err }
        fmt.Println("deletion observed; returned record retains the last successful receipt")
    }
    return nil
}
```

## References

- [ImageRecord](image-records.md)
- [Image proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L2299-L2363)
- [Resource wait](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2585-L2722)
- [fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1837)
- [timeout iterator](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L99)
- [Install](../docs/install.md)
- [Image Records](image-records.md)
- [Service Waits](../docs/service-waits.md)
- [Glance Policy Priorities](../docs/glance-policy-priorities.md)
- [Image Record Wait Test](image_record_wait_test.go)
- [Connection Image Record Waits Test](../connectiontest/connection_image_record_waits_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
