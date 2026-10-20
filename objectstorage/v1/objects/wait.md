# Wait

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:object_store/v1/wait_for_delete`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitPollInterval`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitTimeout`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitUnlimitedWait`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitStatusAttribute`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitStatusHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitFailureStates`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitProgressCallback`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutObjectWaitProgressCallback`

- Fresh bodyless HEAD addresses escaped literal container/object under captured ResourceBase. Clean physical404 alone sets Complete/Deleted, while200/204 continue; delete Status stays nil and valid status/failure choices do not determine completion. The waiter does not submit DELETE.
  Tests: `objectstorage/v1/objects/wait_core_test.go:TestObjectWaitCoreFreshPollingAndCallbacks`, `objectstorage/v1/objects/wait_contracts_test.go:TestObjectWaitContractsLiteralHEADAndDeletion`, `objectstorage/v1/objects/wait_options_test.go:TestObjectWaitOptionsPreflightAndCauses`.
- Read/Close faults, sticky forbidden source mutations and transport errors carrying nested404 do not complete deletion. Wire response plus transport error retains actual status/header proof without inventing acknowledgement or consumed body; rejected responses keep native proof. Accepted fault proof remains independently owned.
  Tests: `objectstorage/v1/objects/wait_core_test.go:TestObjectWaitCoreFailureEvidenceAndCallbackCauses`, `objectstorage/v1/objects/wait_contracts_test.go:TestObjectWaitContractsDirtyMissingAndAcceptedFaults`.
- Defaults are interval2s/delete120s, UnlimitedWait removes only the SDK deadline, and parent/SDK cancellation interrupts pauses. Last retains the last physically started phase rather than poll history; Polls counts logical exchanges, including a later canceled exchange with no physical request, which preserves previous Last.
  Tests: `objectstorage/v1/objects/wait_core_test.go:TestObjectWaitCoreTimersAndLastActualPhase`, `objectstorage/v1/objects/wait_contracts_test.go:TestObjectWaitContractsDefaultsCallbacksAndCancellation`.
- Full option factories snapshot maps, duration pointers, failure slices and callbacks; later caller or retained callback-option mutation cannot alter prepared polls. Canonical header overlays and reusable factories keep independent acknowledgement/attempt proof, including concurrent calls.
  Tests: `objectstorage/v1/objects/wait_options_test.go:TestObjectWaitOptionsSnapshots`, `objectstorage/v1/objects/wait_options_test.go:TestObjectWaitOptionsDefaultsSelectorsAndResets`, `objectstorage/v1/objects/wait_options_test.go:TestObjectWaitOptionsParallelReuse`.
- Nil options/context, canceled context, invalid durations, malformed selectors/failure values, conflicting full selectors, duplicate aliases and reserved request headers fail before HTTP. Source/target preflight runs before option callbacks; callback, source and cancellation causes remain inspectable together.
  Tests: `objectstorage/v1/objects/wait_options_test.go:TestObjectWaitOptionsPreflightAndCauses`.
- A clean nonterminal observation calls the synchronous callback once with progress0; terminal observations do not. Callback error/cancellation/source faults preserve the actual response without committing that poll status. WithoutObjectWaitProgressCallback clears the callback.
  Tests: `objectstorage/v1/objects/wait_core_test.go:TestObjectWaitCoreFreshPollingAndCallbacks`, `objectstorage/v1/objects/wait_core_test.go:TestObjectWaitCoreFailureEvidenceAndCallbackCauses`, `objectstorage/v1/objects/wait_options_test.go:TestObjectWaitOptionsDefaultsSelectorsAndResets`, `objectstorage/v1/objects/wait_contracts_test.go:TestObjectWaitContractsDefaultsCallbacksAndCancellation`.
- Native retry/backoff/reauth can create multiple physical attempts within one logical poll without rerunning option callbacks. Captured ordinary headers remain stable, live auth refreshes, and rejected callback-mutable response bodies/headers do not alias retained proof. Native/body/redirect guards reject conditional headers, changed accepted codes and injected bodies; the waiter-specific Range policy does not change ordinary CreateObject retry behavior.
  Tests: `objectstorage/v1/objects/wait_contracts_test.go:TestObjectWaitContractsNativeReplaySnapshotsAndLiveGuards`, `objectstorage/v1/objects/wait_core_test.go:TestObjectWaitCoreHeaderPolicyIsolation`.
- Connection ObjectStorage/ObjectStorageV1 exposes the same objects facade and original service client, captures configuration afresh per workflow, refreshes live auth between polls and owns acknowledgement/attempt evidence independently.
  Tests: `connection_objectstorage_wait_test.go:TestConnectionSwiftObjectWaitSharesClientAndFreshCapture`.
- Generated Swift native/service/registry files remain byte-exact and accounts/containers/objects gain no generated Collections; the service guide references both concrete waiters, HEAD, delete120s and objects/wait.md.
  Tests: `internal/cmd/sdkgen/swift_object_wait_test.go:TestSwiftObjectWaitPreservesNativeAndSharedWaiters`.

Python/Go differences:

- Go requires literal container/object and observes bodyless HEAD200/204/404; clean physical404 alone completes deletion. Python generic Resource fetch uses GET and can succeed on falsy resource, deleted status or caught NotFoundException; ordinary Swift Object declares no status. Neither waiter submits DELETE.
- Go returns bounded last physically-started poll attempts and independently owned raw acknowledgement/error proof plus logical Polls/Complete/Deleted, while Python returns the mutable Resource. Status remains nil for delete and status/failure selectors do not determine completion.
- Go keeps interval2s/deleteSDK120s with positive duration helpers, pointer0 or UnlimitedWait and caller context deadlines/cancellation; Python iterate_timeout uses wall-clock/numeric coercion, accepts None and rewrites zero interval. Existing Collection.WaitDeleted five-minute/default generic policy stays separate.
- Go snapshots typed maps/pointers/slices and callbacks, guards fixed original source/body/status codes, preserves live auth and excludes Range/If-* plus existing X-Newest reservations. Python Resource.fetch does not forward prepared request headers to session.get. Go synchronous progress0 callbacks can return errors and clear via nil/Without.

Unresolved boundaries:

- Python Resource identity/container precedence, dirty descriptor/metadata state, adapter/session/cache/microversion behavior and generic exception-class equivalence are not reproduced wholesale.
- Python generic nil/deleted completion and arbitrary resource/status/progress callbacks differ from the concrete clean HEAD404 observation and last-only Go evidence.
- HEAD404 observes absence at that route/response; replica races, middleware permissions, proxy caching and cluster-wide deletion completion remain server concerns. Native/read Newest options are separate; this waiter adds no newest extension.

### `python:object_store/v1/wait_for_status`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitPollInterval`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitTimeout`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitUnlimitedWait`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitStatusAttribute`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitStatusHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitFailureStates`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithObjectWaitProgressCallback`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects.WithoutObjectWaitProgressCallback`

- Default/unknown/numeric/bool status attributes are unsupported before HTTP. All supported string attribute mappings and explicit response-header selectors perform fresh HEAD, including captured name/container literals; missing selected header returns unsupported with the real response and nil Status.
  Tests: `objectstorage/v1/objects/wait_core_test.go:TestObjectWaitCoreSelectorsAndUnsupported`, `objectstorage/v1/objects/wait_contracts_test.go:TestObjectWaitContractsSelectedStatusAndFailurePolicy`.
- Only the selected header projects: duplicate case aliases, multiple/zero values, invalid UTF8 and controls fail atomically with raw response proof and nil Status. Unrelated size/date/header values do not participate; present empty is a distinct nonnil value and can match an empty target.
  Tests: `objectstorage/v1/objects/wait_core_test.go:TestObjectWaitCoreProjectionAndTerminalStates`, `objectstorage/v1/objects/wait_contracts_test.go:TestObjectWaitContractsProjectionIsAtomic`, `objectstorage/v1/objects/wait_contracts_test.go:TestObjectWaitContractsSelectedStatusAndFailurePolicy`.
- Target and failure comparisons ignore case and target wins over failure. Nil failures reset to ERROR; nonnil empty and no-argument failure helper disable failures. A clean failure keeps observed Status and typed failed-state proof; clean404 is a status-wait error. Selector helpers clear each other so the last selection wins, while conflicting full selectors reject.
  Tests: `objectstorage/v1/objects/wait_core_test.go:TestObjectWaitCoreProjectionAndTerminalStates`, `objectstorage/v1/objects/wait_options_test.go:TestObjectWaitOptionsDefaultsSelectorsAndResets`, `objectstorage/v1/objects/wait_contracts_test.go:TestObjectWaitContractsSelectedStatusAndFailurePolicy`.
- Status wait defaults to interval2s with no SDK deadline; explicit positive timeout/unlimited options and caller context remain effective. Pause timeout preserves the prior clean observation. If the next logical exchange is canceled before any physical request, Polls increases while prior actual Last and clean Status remain; later dirty/callback processing commits no new status.
  Tests: `objectstorage/v1/objects/wait_core_test.go:TestObjectWaitCoreTimersAndLastActualPhase`, `objectstorage/v1/objects/wait_core_test.go:TestObjectWaitCoreFailureEvidenceAndCallbackCauses`, `objectstorage/v1/objects/wait_contracts_test.go:TestObjectWaitContractsDefaultsCallbacksAndCancellation`.
- Accepted read/Close/source faults and rejected or transport error responses stop with actual last-phase evidence and no completed or newly projected state. Acknowledgement, physical attempt and returned response-error proof do not share mutable header/body storage.
  Tests: `objectstorage/v1/objects/wait_core_test.go:TestObjectWaitCoreProjectionAndTerminalStates`, `objectstorage/v1/objects/wait_core_test.go:TestObjectWaitCoreFailureEvidenceAndCallbackCauses`, `objectstorage/v1/objects/wait_contracts_test.go:TestObjectWaitContractsDirtyMissingAndAcceptedFaults`.
- Full option factories snapshot maps, duration pointers, failure slices and callbacks; later caller or retained callback-option mutation cannot alter prepared polls. Canonical header overlays and reusable factories keep independent acknowledgement/attempt proof, including concurrent calls.
  Tests: `objectstorage/v1/objects/wait_options_test.go:TestObjectWaitOptionsSnapshots`, `objectstorage/v1/objects/wait_options_test.go:TestObjectWaitOptionsDefaultsSelectorsAndResets`, `objectstorage/v1/objects/wait_options_test.go:TestObjectWaitOptionsParallelReuse`.
- Nil options/context, canceled context, invalid durations, malformed selectors/failure values, conflicting full selectors, duplicate aliases and reserved request headers fail before HTTP. Source/target preflight runs before option callbacks; callback, source and cancellation causes remain inspectable together.
  Tests: `objectstorage/v1/objects/wait_options_test.go:TestObjectWaitOptionsPreflightAndCauses`.
- A clean nonterminal observation calls the synchronous callback once with progress0; terminal observations do not. Callback error/cancellation/source faults preserve the actual response without committing that poll status. WithoutObjectWaitProgressCallback clears the callback.
  Tests: `objectstorage/v1/objects/wait_core_test.go:TestObjectWaitCoreFreshPollingAndCallbacks`, `objectstorage/v1/objects/wait_core_test.go:TestObjectWaitCoreFailureEvidenceAndCallbackCauses`, `objectstorage/v1/objects/wait_options_test.go:TestObjectWaitOptionsDefaultsSelectorsAndResets`, `objectstorage/v1/objects/wait_contracts_test.go:TestObjectWaitContractsDefaultsCallbacksAndCancellation`.
- Native retry/backoff/reauth can create multiple physical attempts within one logical poll without rerunning option callbacks. Captured ordinary headers remain stable, live auth refreshes, and rejected callback-mutable response bodies/headers do not alias retained proof. Native/body/redirect guards reject conditional headers, changed accepted codes and injected bodies; the waiter-specific Range policy does not change ordinary CreateObject retry behavior.
  Tests: `objectstorage/v1/objects/wait_contracts_test.go:TestObjectWaitContractsNativeReplaySnapshotsAndLiveGuards`, `objectstorage/v1/objects/wait_core_test.go:TestObjectWaitCoreHeaderPolicyIsolation`.
- Connection ObjectStorage/ObjectStorageV1 exposes the same objects facade and original service client, captures configuration afresh per workflow, refreshes live auth between polls and owns acknowledgement/attempt evidence independently.
  Tests: `connection_objectstorage_wait_test.go:TestConnectionSwiftObjectWaitSharesClientAndFreshCapture`.
- Generated Swift native/service/registry files remain byte-exact and accounts/containers/objects gain no generated Collections; the service guide references both concrete waiters, HEAD, delete120s and objects/wait.md.
  Tests: `internal/cmd/sdkgen/swift_object_wait_test.go:TestSwiftObjectWaitPreservesNativeAndSharedWaiters`.

Python/Go differences:

- Swift Object has no default status/progress descriptor. Python default getattr(status) raises AttributeError before fetch; Go default/unsupported selector returns ErrUnsupported before HTTP. Go accepts library-owned known string attributes or explicit response header, performs fresh HEAD even for captured name/container and does not implement the Python cached initial-match shortcut.
- Go compares one strict relevant single-value UTF8/control-safe header or literal string using EqualFold, preserving absence/empty/case. Missing selected response header retains real HEAD proof with unsupported error; target match precedes failures. Python uses arbitrary getattr and str.lower over mutable Resource descriptor values fetched by GET.
- Go retains source interval2s/statusSDKunlimited and nilFailureStates ERROR; explicit empty disables failures. Typed positive time helpers, pointer0/unlimited context policy and synchronous func(int)error callback differ from Python numeric/wall-clock/None interval and no modeled callback return error.
- Go returns nullable Status/Complete and bounded last physically-started phase instead of mutable Resource, preserving accepted/rejected raw evidence and joined processing/source/context causes. Failed current phase or callback commits no new status; later pause timeout can retain last clean value. Full opts/selectors/headers own snapshots, last helper selection wins and fixed original source/live auth stay guarded. A later logical exchange stopped before any physical request retains the prior actual Last and clean Status while Polls still counts that logical exchange.

Unresolved boundaries:

- Python arbitrary Resource attributes, descriptor coercion, unknown/dirty state, cached initial fast path and GET object-content/session/cache behavior remain distinct from the concrete HEAD/string-selector workflow.
- Python lower normalization and exception classes are not identical to Go EqualFold/typed errors; callbacks cannot emulate arbitrary mutable Resource progress behavior because ordinary Object has no progress.
- Application-selected metadata/header state is caller/server data and no standard Swift lifecycle status is invented. Matching observation is not an atomic lock, publication/consistency guarantee or whole proxy parity completion.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func showWait(result *objects.ObjectWaitResult, err error) {
    if result != nil {
        status := "<absent>"
        if result.Status != nil {
            status = *result.Status
        }
        fmt.Printf("polls=%d complete=%v deleted=%v status=%q\n",
            result.Polls, result.Complete, result.Deleted, status)
        if result.Last != nil {
            for _, attempt := range result.Last.Attempts {
                fmt.Printf("physical=%d error=%v\n", attempt.PhysicalAttempt, attempt.Error)
                if attempt.Response != nil {
                    fmt.Printf("HTTP%d headers=%v body=%q\n",
                        attempt.Response.StatusCode, attempt.Response.Header, attempt.Response.Body)
                }
            }
            if result.Last.Acknowledgement != nil {
                fmt.Printf("last acknowledgement HTTP%d\n", result.Last.Acknowledgement.StatusCode)
            }
        }
    }
    if err != nil {
        fmt.Println(err)
        if errors.Is(err, resource.ErrUnsupported) {
            fmt.Println("unsupported status source")
        }
        var proof *resource.ResponseError
        if errors.As(err, &proof) {
            fmt.Printf("error proof HTTP%d headers=%v body=%q\n",
                proof.StatusCode, proof.Header, proof.Body)
        }
    }
}

func waitExamples(ctx context.Context, conn *sdk.Connection) {
    service, err := conn.ObjectStorageV1(ctx)
    if err != nil {
        showWait(nil, err)
        return
    }
    interval := 2 * time.Second
    policy := []objects.ObjectWaitOption{
        objects.WithObjectWaitOpts(objects.ObjectWaitOpts{
            Headers: map[string]string{"X-Client": "object-wait"},
            Interval: &interval,
        }),
        objects.WithObjectWaitHeader("X-Trace", "initial"),
        objects.WithObjectWaitHeaders(map[string]string{"x-trace": "last"}),
        objects.WithObjectWaitPollInterval(2 * time.Second),
        objects.WithObjectWaitTimeout(time.Minute),
        objects.WithObjectWaitProgressCallback(func(progress int) error {
            fmt.Printf("nonterminal observation, progress=%d\n", progress)
            return nil
        }),
    }
    statusOptions := append([]objects.ObjectWaitOption{}, policy...)
    statusOptions = append(statusOptions,
        objects.WithObjectWaitStatusHeader("X-Object-Meta-State"),
        objects.WithObjectWaitFailureStates("ERROR", "FAILED"),
    )
    result, err := service.Objects.WaitForStatus(ctx, "backups", "archive.tar", "READY", statusOptions...)
    showWait(result, err)

    // A known string attribute can also be observed, such as the MIME type.
    attributeOptions := append([]objects.ObjectWaitOption{}, policy...)
    attributeOptions = append(attributeOptions,
        objects.WithObjectWaitStatusAttribute("content_type"),
        objects.WithObjectWaitFailureStates(),
        objects.WithoutObjectWaitProgressCallback(),
    )
    result, err = service.Objects.WaitForStatus(ctx, "backups", "archive.tar", "application/octet-stream", attributeOptions...)
    showWait(result, err)

    // UnlimitedWait removes the SDK deadline; this caller still has 30 seconds.
    deleteCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    deleteOptions := append([]objects.ObjectWaitOption{}, policy...)
    deleteOptions = append(deleteOptions,
        objects.WithObjectWaitUnlimitedWait(),
        objects.WithoutObjectWaitProgressCallback(),
    )
    result, err = service.Objects.WaitForDelete(deleteCtx, "backups", "already-deleted.bin", deleteOptions...)
    showWait(result, err)
}

func main() {}
```

## References

- [pinned proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L1247-L1306)
- [Resource waits](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2585-L2715)
- [Resource.fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1837)
- [Object descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L27-L213)
- [iterate_timeout](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53-L101)
- [Collection.WaitDeleted](../../../resource/README.md)
