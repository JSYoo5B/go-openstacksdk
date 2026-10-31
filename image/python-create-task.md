# Python Create Task

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/create_task`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.CreateTask`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCreateTaskOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCreateTaskInput`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCreateTaskNullInput`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCreateTaskHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithCreateTaskHeaders`

- WithCreateTaskNullInput makes CreateTask send POST tasks with {"type": ..., "input": null}, clears an earlier input object, is overridden by a later WithCreateTaskInput, is also selected by CreateTaskOpts.NullInput, and a null input in the 201 response leaves TaskInfo.Input nil with the raw null kept in Metadata.Body.
  Tests: `image/python_create_task_test.go:TestPythonCreateTaskSendsNullInput`.
- CreateTaskOpts with both NullInput and a non-nil Input fails before HTTP as resource.ErrInvalidOption wrapped in a CreateTask resource.OperationError, and a 400 response to a null input is returned as the Gophercloud status error with operation context.
  Tests: `image/python_create_task_test.go:TestPythonCreateTaskNullInputRejections`.
- One fixed captured image-service collection POST201 with explicit type/input only. Type nonemptyUTF8 literal without local enum/control gate; nilInput defaults{} rather than top-levelnull. Pinned nullable Schema/controller.get discrepancy is explicit deliberateGo policy; no extra GET/child/wait/cleanup.
  Tests: `image/tasks_core_test.go:TestTaskCoreFixedRoutesAndObjectInput`, `image/tasks_core_test.go:TestTaskCoreCompletePreflight`, `image/tasks_contracts_test.go:TestTaskContractsFixedRoutesAndInput`.
- Raw CreateTaskOpts Input independently owns top-level keys/raw validUTF8 JSON values including null/exact large numeric tokens; arbitrary inputkeys allowed. WithCreateTaskInput anymap snapshots JSON and validates original top-level keyUTF8 before marshal; nil resets{} and stored marshal cause returns during invocation; nested any uses encoding/json conventions.
  Tests: `image/tasks_options_test.go:TestTaskOptionsImmediateFactoriesAndRawSnapshots`, `image/tasks_options_test.go:TestTaskOptionsCallbacksAndCompleteValidation`, `image/tasks_contracts_test.go:TestTaskContractsFixedRoutesAndInput`, `image/tasks_contracts_test.go:TestTaskContractsOwnedOptionsAndSource`.
- WithCreateTaskOpts whole replacement, field/header helpers lastwins, option callbacks once and recursive raw/header/config snapshots prevent retained aliases; errors preserve original callback/marshal/context cause beforeHTTP; reused helpers/parallel calls do not share prepared bodies.
  Tests: `image/tasks_options_test.go:TestTaskOptionsReplacementPresenceAndLastWins`, `image/tasks_options_test.go:TestTaskOptionsCallbacksAndCompleteValidation`, `image/tasks_options_test.go:TestTaskOptionsParallelReusableHelpers`, `image/tasks_contracts_test.go:TestTaskContractsOwnedOptionsAndSource`, `connectiontest/connection_image_tasks_test.go:TestConnectionImageTaskFacadeSharesClientAndOwnsInput`.
- Canonical TaskInfo wholeUTF8 nonnull object, exactcase optional strings/nullable raw Input/Result maps, literal dates and unknown precision retained. Atomic wrongtype failure, independently owned raw map/Metadata.Body/header/status; no responseID equality or request-driven synthesized fields.
  Tests: `image/tasks_core_test.go:TestTaskCoreCanonicalPresenceAndAtomicModels`, `image/tasks_contracts_test.go:TestTaskContractsCanonicalModelsAndRawOwnership`.
- Strict actual201 accepted only. Accepted read/Close/source/context/customcause/decode failures return niltyped plus whole actualResponseError evidence, Closeonce and no acceptedPOST replay; native rejectedstatus ownership unchanged.
  Tests: `image/tasks_core_test.go:TestTaskCoreAcceptedEvidenceAndStrictStatus`, `image/tasks_contracts_test.go:TestTaskContractsResponseOwnershipAndNativeHooks`.
- Original client/provider/Type/Endpoint/base/Microversion and ordinary headers captured before callbacks; context/source recheck beforeHTTP/afterbody, original provider liveauth remains. Existing prebody retries/reauth/backoff/identicaltargetredirect plus owned method/URL/body/status guards retained, advanced native header policy remains.
  Tests: `image/tasks_core_test.go:TestTaskCoreFixedSourceAndPrebodyPolicy`, `image/tasks_contracts_test.go:TestTaskContractsOwnedOptionsAndSource`, `image/tasks_contracts_test.go:TestTaskContractsResponseOwnershipAndNativeHooks`.

Python/Go differences:

- Python sends every declared Task attribute it receives, such as status, owner or message, and may omit type or input; Go always sends exactly the type and input keys, which are the only values Glance's controller passes to task creation.
- Python sends any input value, including a list or a string; Go sends a JSON object, {} by default, or null with WithCreateTaskNullInput.
- Python sends an empty type string to the server; Go rejects an empty or invalid UTF-8 type before HTTP.
- Python accepts every status below 400 and ignores a non-JSON body; Go accepts only 201 and returns a non-object body as a resource.ResponseError.
- Python returns a Task Resource merged from the request attributes and the response with the self key dropped; Go returns TaskInfo with only response fields, typed as optional strings and raw JSON maps, plus headers, status and the raw body.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
task, err := svc.CreateTask(ctx, "import", image.WithCreateTaskNullInput())
```

## References

- [`create_task(**attrs)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py)
- [`Task`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/task.py)
- [tasks.md](tasks.md)
