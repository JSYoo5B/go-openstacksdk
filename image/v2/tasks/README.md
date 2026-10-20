# Tasks

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/wait_for_task`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/tasks.API.WaitForTask`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/tasks.API.WaitForTaskState`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/tasks.WithTaskWaitOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/tasks.WithTaskWaitTimeout`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/tasks.WithUnlimitedTaskWait`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/tasks.WithTaskWaitPollInterval`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/tasks.WithTaskWaitFailureStates`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/tasks.WithTaskWaitHeader`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/tasks.WithTaskWaitHeaders`

- The SDK-owned WaitForTask starts with a fresh GET of an explicit ID and defaults to success, exact case-insensitive failure, one120-second budget and two-second polling. WaitForTaskState accepts an explicit target. Incidental GET response IDs do not change the current request ID; OriginalID and CurrentID identify actual request routing.
  Tests: `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitFreshReadsAndDefaults`, `image/v2/tasks/wait_options_test.go:TestTaskWaitOptionsDefaultsOrderingEmptyFailuresAndInvalidValues`, `image/v2/tasks/wait_core_test.go:TestTaskWaitCoreTargetFailureSelectionDeadlineAndPreflight`.
- Target comparison precedes failure detection. The default selected failure is failure; custom failures replace it case-insensitively, and an explicit empty list disables state failure and396 recreation. Only a fetched selected failure with the exact case-sensitive message Image cannot be imported. Error code: '396' recreates; wrong punctuation, case or an unselected state never POSTs.
  Tests: `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitSelected396AndExactFetchedInput`.
- Recreation sends only the fetched canonical type and raw input, preserving large JSON numbers, exponents, nested arrays/objects and null. Explicit null and empty-object input remain distinct. A selected396 response with missing input or missing/null/empty type fails before POST, retaining the actual fetched result; no seeded/cached request data is invented.
  Tests: `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitSelected396AndExactFetchedInput`, `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitFetchedSchemaAndInputPresence`, `image/v2/tasks/wait_results_test.go:TestTaskWaitDecodeCanonicalFieldsAndExactNumbers`.
- Each successful creation changes the fixed current ID only after a usable canonical lowercase POST id is validated. Repeated396 transitions keep one unchanged deadline across GET/POST/new-GET rather than starting new120-second budgets. Recreated counts only valid decoded creations; Self and Schema do not redirect requests.
  Tests: `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitRepeated396KeepsBudgetAndPausesAfterCreate`, `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitRecreationErrorsKeepActualEvidence`, `image/v2/tasks/wait_core_test.go:TestTaskWaitCoreOneBudgetRawRecreationFixedRoutesAndLiveToken`.
- The waiter pauses and fetches the new ID even if the actual POST201 model already reports success. A deadline during that pause returns the actual created model and201 proof with the context error, without a success shortcut or another GET. Parent deadlines remain effective under explicit unlimited policy.
  Tests: `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitRepeated396KeepsBudgetAndPausesAfterCreate`, `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitFreshReadsAndDefaults`.
- The outer Task/Body/Header/StatusCode retain the latest successfully decoded real observation. Created separately retains the latest accepted POST201 body/header/status, including read/decode/ID failures with no invented Task. Failed recreation keeps the previous fetched model/route; failed subsequent GET keeps the valid created model/current ID. Created typed maps and raw HTTP evidence are independently copied from outer/source values.
  Tests: `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitRecreationErrorsKeepActualEvidence`, `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitTerminalReadsAndCancellationNeverReplay`, `image/v2/tasks/wait_results_test.go:TestTaskWaitDecodeCanonicalFieldsAndExactNumbers`, `image/v2/tasks/wait_core_test.go:TestTaskWaitCoreFailedCreationRetainsLastTaskAndActualAcceptedProof`.
- The new workflow requires a flat object and canonical lowercase fetched status string; invalid UTF-8, malformed roots, native map/string/time.Time shapes and missing/null/nonstring decision fields preserve actual accepted response evidence and underlying causes. Case-variant known keys stay in raw Body but cannot shadow canonical decisions. The dedicated return model uses json.Number for input/result; ordinary native Get/Create number decoding remains unchanged.
  Tests: `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitFetchedSchemaAndInputPresence`, `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitNativeSurfacesAndCanonicalEvidenceStaySeparate`, `image/v2/tasks/wait_results_test.go:TestTaskWaitDecodeRejectsNativeShapesWithAcceptedEvidence`, `image/v2/tasks/wait_results_test.go:TestTaskWaitDecodeCanonicalFieldsAndExactNumbers`, `image/v2/tasks/wait_core_test.go:TestTaskWaitCoreInvalidUTF8NeverRecreatesOrAcceptsCreatedTask`.
- GET/POST HTTP errors, transport-wrapped native404, accepted-body read/decode failures and cancellation remain terminal workflow errors with their original causes and actual response evidence. They are not missing/completion signals and do not trigger semantic recreation. Accepted read failure is not resent through configured RetryFunc; no response-less model or HTTP metadata is synthesized.
  Tests: `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitTerminalReadsAndCancellationNeverReplay`, `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitRecreationErrorsKeepActualEvidence`.
- Concrete WithTaskWaitOpts snapshots pointer/slice/header inputs and replaces the whole policy; individual helpers apply last. Nil Timeout selects120s, an explicit zero bulk duration or WithUnlimitedTaskWait removes the SDK deadline, and later positive bounds restore it. Nil failures select failure while nonnil empty disables them. Reusable options and retained custom config pointers do not alias later parsed requests or returned models.
  Tests: `image/v2/tasks/wait_options_test.go:TestTaskWaitOptionsSnapshotCustomRetentionAndConcurrentReuse`, `image/v2/tasks/wait_options_test.go:TestTaskWaitOptionsDefaultsOrderingEmptyFailuresAndInvalidValues`, `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitOptionSnapshotsAndConcurrentReuse`.
- Request headers are canonicalized, frozen and applied across the workflow; case-insensitive sequential helpers are last-wins. Conflicting case aliases and protected auth/transport/version controls fail preHTTP. Nil context/source/provider/options, unsafe or Name refs, blank targets/failures and nonpositive helper durations also reject without HTTP. The task namespace provides no generic progress/status-attribute/query/base-path controls.
  Tests: `image/v2/tasks/wait_options_test.go:TestTaskWaitHeadersCanonicalLastWinsAndProtectedOwnership`, `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitPreflightAndClosedOptions`.
- The workflow shares the original Provider for live token rotation, configured middleware, microversion, reauthentication and configured native retry, while capturing the selected ResourceBase task routes and request headers. Configured401 reauth and503 retry preserve the same POST payload and method/target; invalid source changes after options or fetch stop before following phases. Redirects cannot retarget a task request and the original ServiceClient remains unchanged.
  Tests: `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitLiveSourceFixedRoutesAndReauthentication`.
- Existing native Get/Create/List declarations, Task alias and generic collection wait boundary remain separate. Native Get/Create still expose their original float64 map projection, Name and Remove remain unsupported, and the new wait adds no cleanup requests. Generator metadata marks only the audited image-v2 Task collection with task_wait=WaitForTask; parent/payload drift and unrelated resource inference are rejected, without emitting the manual workflow as a native declaration.
  Tests: `api/glance_task_wait_contracts_test.go:TestGlanceTaskWaitNativeSurfacesAndCanonicalEvidenceStaySeparate`, `internal/cmd/sdkgen/glance_task_wait_test.go:TestGlanceTaskWaitBindsActualIDOnlyCollectionWithoutReplacingNativeCalls`, `internal/cmd/sdkgen/glance_task_wait_test.go:TestGlanceTaskWaitRejectsParentAndPayloadModelDrift`.

Python/Go differences:

- Python accepts a supplied mutable Task and may return its cached target state without HTTP. Go accepts explicit ID Ref only, performs a fresh GET, returns TaskWaitResult rather than that Resource, and does not resolve Task names.
- The new Go workflow returns owned raw HTTP proof and typed Task maps decoded with json.Number, an intentional precision improvement. Existing native Task Get/Create remain unchanged and may project those numbers to float64; this is not a global native decoder change.
- Python Task Body descriptors for input/result/time/status are untyped with defaults/cache merge. Go keeps native map/string/time.Time shape validation, exact lowercase decision keys, a required fetched status string and a safe POST id for the new route. Case aliases remain raw extensions, missing selected recreation input is rejected instead of using Python None/cached input, and explicit null input remains valid.
- The shared workflow timeout applies to active requests as well as pauses and remains subordinate to parent context; timeout/cancellation exposes Go context causes rather than Python ResourceTimeout. Positive duration helpers and explicit Unlimited replace Python interval-zero normalization and zero/negative wait behavior.
- TaskWaitResult distinguishes the latest successful observation from the latest accepted creation proof, preserves partial results/errors and uses separately owned evidence. Python normally returns the current Resource or raises; the Go result does not invent a server task after an unaccepted or undecodable response.
- SDK-owned explicit header options snapshot requests and preserve original Provider transport policy. Protected auth/transport/version headers and source/route gates are deliberate Go controls rather than arbitrary session/adapter injection. No guessed task type enum, cloud schema, access permission or minimum microversion is imposed.
- Python returns the supplied Task without HTTP when its status already matches; Go always fetches the task first.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/tasks"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func waitForTask(ctx context.Context, client *gophercloud.ServiceClient, id string) (*tasks.TaskWaitResult, error) {
    return tasks.New(client).WaitForTask(ctx, resource.ID(id))
}
```

```python
task = conn.image.wait_for_task(task)
```

```go
package example

import (
    "context"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/tasks"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func waitForProcessing(ctx context.Context, client *gophercloud.ServiceClient, id string) (*tasks.TaskWaitResult, error) {
    timeout := time.Minute
    interval := 500 * time.Millisecond
    policy := tasks.WithTaskWaitOpts(tasks.TaskWaitOpts{
        Timeout: &timeout,
        PollInterval: &interval,
        FailureStates: []string{},
    })
    return tasks.New(client).WaitForTaskState(ctx, resource.ID(id), "processing", policy)
}
```

```python
task = conn.image.wait_for_task(
    task, status="processing", failures=[], interval=0.5, wait=60)
```

## References

- [`wait_for_task`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L2198-L2269)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#tasks)
- [Service Waits](../../../docs/service-waits.md)
- [Glance Task Wait Contracts Test](../../../api/glance_task_wait_contracts_test.go)
- [Wait Options Test](wait_options_test.go)
- [Wait Results Test](wait_results_test.go)
- [Glance Task Wait Test](../../../internal/cmd/sdkgen/glance_task_wait_test.go)
