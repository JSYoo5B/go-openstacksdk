# Tasks

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
  Tests: `image/tasks_options_test.go:TestTaskOptionsReplacementPresenceAndLastWins`, `image/tasks_options_test.go:TestTaskOptionsCallbacksAndCompleteValidation`, `image/tasks_options_test.go:TestTaskOptionsParallelReusableHelpers`, `image/tasks_contracts_test.go:TestTaskContractsOwnedOptionsAndSource`, `connection_image_tasks_test.go:TestConnectionImageTaskFacadeSharesClientAndOwnsInput`.
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

### `python:image/v2/get_task`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.GetTask`
- `github.com/JSYoo5B/go-openstacksdk/image.WithGetTaskOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithGetTaskHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithGetTaskHeaders`

- One direct taskID child GET200, captured target and original liveauth. ID nonempty validUTF8 safe URI literal with no UUID/max length gate; escaped percent/spaces/query/hash accepted, dot/dotdot/slash/backslash/control rejected before callback/HTTP. No Ref/Name/find/fallback/ignoreMissing or Self/Schema/Location retargeting.
  Tests: `image/tasks_core_test.go:TestTaskCoreFixedRoutesAndObjectInput`, `image/tasks_core_test.go:TestTaskCoreCompletePreflight`, `image/tasks_contracts_test.go:TestTaskContractsFixedRoutesAndInput`, `image/tasks_contracts_test.go:TestTaskContractsOwnedOptionsAndSource`.
- WithGetTaskOpts whole replacement and headers lastwins; input option slice/config/header ownership, callbackonce/cause preservation and pre/post source/context checks ensure fixed requested identity.
  Tests: `image/tasks_options_test.go:TestTaskOptionsReplacementPresenceAndLastWins`, `image/tasks_options_test.go:TestTaskOptionsCallbacksAndCompleteValidation`, `image/tasks_contracts_test.go:TestTaskContractsOwnedOptionsAndSource`.
- Canonical exactcase optional nullable TaskInfo strings/date/raw objects preserve actual fullGet sparse/nullable states without native time/map coercion, inputID seeding, status normalization or routefollowing; wrongcanonicaltype/wholeinvalidUTF8/schema shape fails atomically.
  Tests: `image/tasks_core_test.go:TestTaskCoreCanonicalPresenceAndAtomicModels`, `image/tasks_contracts_test.go:TestTaskContractsCanonicalModelsAndRawOwnership`.
- Strict actual200; accepted read/Close/customcause/source/context/decoder errors retain wholeactual bytes/header/status inResponseError and niltyped, Closeonce/noacceptedGET replay. Native404 including maskedauthorization stays error; noabsenceproof/suppression.
  Tests: `image/tasks_core_test.go:TestTaskCoreAcceptedEvidenceAndStrictStatus`, `image/tasks_contracts_test.go:TestTaskContractsResponseOwnershipAndNativeHooks`.
- Configured native prebody retry/reauth/backoff/sametargetredirect retains explicitproviderpolicy and native ErrOriginal/ErrReauth fields; changedtarget/method/ownedrequestbody or acceptedstatus expansion cannot replace fixed ownership. Existing native generatedTask/WaitForTask code stays byteexact; no implicitwait.
  Tests: `image/tasks_core_test.go:TestTaskCoreFixedSourceAndPrebodyPolicy`, `image/tasks_contracts_test.go:TestTaskContractsOwnedOptionsAndSource`, `image/tasks_contracts_test.go:TestTaskContractsResponseOwnershipAndNativeHooks`, `internal/cmd/sdkgen/glance_tasks_test.go:TestGlanceTaskFacadePreservesNativeAndRegistry`.
- GetTask sends one GET tasks/{id} and returns id, type, status, input and result like Python _get(Task).
  Tests: `image/python_rejudge_test.go:TestPythonRejudgeImageLocationsAndTaskDefaultRequests`.

Python/Go differences:

- Pinned Python mutable Resource/dirty body attributes/same-object returns and configured Adapter/cache/session/microversion/header machinery remain broader than fixed concrete function options and independently owned TaskInfo.
- Go TaskInfo exact canonical optional string/date pointers and nullable raw object Input/Result preserve raw precision and literal timestamps; native Task time.Time/map[string]any and pinned Python untyped result/date values differ. No type/status normalization or cached/input/ID seeding.
- Pinned Resource.fetch alwaysAdapterGET and canmutate same providedResource; configured Adapter cache may affect actualtransport. Go creates fresh ownedTaskInfo eachcall and strict404 canmask authorization.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Go has no client-side GET response cache or skip_cache control; every call sends its request.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

### `python:image/v2/tasks`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.Tasks`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.AllTasks`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListTasksOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListTasksHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListTasksHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListTasksLimit`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListTasksMarker`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListTasksType`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListTasksStatus`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListTasksSortKey`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListTasksSortDir`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListTasksMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListTasksSinglePage`

- Tasks is lazy reusable iter.Seq2 with construction-owned option slice and perexecution prepared headers/Limitpointer, FullOpts replacement/lastwins/callbackonce/causes. MaxItems0 unlimited, negativeinvalid; AllTasks returns nonnil empty or wholecollected results, nil/error without partialslice.
  Tests: `image/tasks_options_test.go:TestTaskOptionsReplacementPresenceAndLastWins`, `image/tasks_options_test.go:TestTaskOptionsParallelReusableHelpers`, `image/tasks_contracts_test.go:TestTaskContractsAdvertisedPagingAndLocalCaps`, `image/tasks_contracts_test.go:TestTaskContractsOwnedOptionsAndSource`.
- Concrete sixquery limit/marker/type/status/sort_key/sort_dir; nilLimitomit/zeroemit/nonnegative; emptystringsomit/nonemptyUTF8controlfree literal, onlysortdirasc/desc gate. Native Typejson tagignored byqbuilder differs from explicit facadeTypewire; localcap does not invent wirelimit.
  Tests: `image/tasks_core_test.go:TestTaskCoreCompletePreflight`, `image/tasks_options_test.go:TestTaskOptionsCallbacksAndCompleteValidation`, `image/tasks_core_test.go:TestTaskCoreAdvertisedPagesAndLocalConsumption`, `image/tasks_contracts_test.go:TestTaskContractsFixedRoutesAndInput`, `image/tasks_contracts_test.go:TestTaskContractsOwnedOptionsAndSource`.
- Strict200 and canonical nonnull tasksarray with wholeUTF8/syntax before consumedrow decode; sparse pinned liststub Input/Result/Message remainnil/raw only. Canonical TaskInfo fields/no requestidentityseeding/no perrowGET; wrong consumedrowtype fails while unused malformedrows maybeskipped bycap/break.
  Tests: `image/tasks_core_test.go:TestTaskCoreCanonicalPresenceAndAtomicModels`, `image/tasks_core_test.go:TestTaskCoreAdvertisedPagesAndLocalConsumption`, `image/tasks_contracts_test.go:TestTaskContractsCanonicalModelsAndRawOwnership`, `image/tasks_contracts_test.go:TestTaskContractsAdvertisedPagingAndLocalCaps`.
- Only canonical bodynext advances under captured exact reverseprefix/sameorigin collection; normalize /v2/tasks, preserve initialquery except single advancing marker. Reject foreign/malformed/userinfo/fragment/encoded-dot/duplicate/unknown/droppedquery and cycles before follow; no HTTPLink/first/schema/rowself orlastIDfallback.
  Tests: `image/tasks_core_test.go:TestTaskCoreAdvertisedPagesAndLocalConsumption`, `image/tasks_contracts_test.go:TestTaskContractsAdvertisedPagingAndLocalCaps`.
- Empty pageends; positive localcap/earlybreak/SinglePage stop without unusedrow/nextvalidation orfollow. Iterator keeps deliveredrows at caller when latererror, AllTasks returnsnil/error. No localJSONSchema/runtime policy validation, Python general clientbodyfilters/JMESPath/max_items-to-limit continuation remains broader.
  Tests: `image/tasks_core_test.go:TestTaskCoreAdvertisedPagesAndLocalConsumption`, `image/tasks_contracts_test.go:TestTaskContractsAdvertisedPagingAndLocalCaps`.
- Capture source/provider/Type/Endpoint/base/Microversion at each iteration, ordinary sourceheaders initial beforecallbacks and refreshed onnextpage with stableoptionheaderoverride; original provider authlive. Recheck source/context afterbody and before/afterconsumedrow, no borrowed source mutation.
  Tests: `image/tasks_core_test.go:TestTaskCoreAdvertisedPagesAndLocalConsumption`, `image/tasks_core_test.go:TestTaskCoreFixedSourceAndPrebodyPolicy`, `image/tasks_contracts_test.go:TestTaskContractsAdvertisedPagingAndLocalCaps`, `image/tasks_contracts_test.go:TestTaskContractsOwnedOptionsAndSource`.
- Accepted read/Close/source/context/customcause/root/model/next errors preserve actualwhole response proof, Closeonce/noaccepted replay; original nativeunexpectedstatus/RetryFunc/reauth/backoff/identicalredirect ownedmethodURLbody/status guard remains. No automatictaskcompletion/wait/cleanup/versiondiscovery.
  Tests: `image/tasks_core_test.go:TestTaskCoreAcceptedEvidenceAndStrictStatus`, `image/tasks_core_test.go:TestTaskCoreFixedSourceAndPrebodyPolicy`, `image/tasks_contracts_test.go:TestTaskContractsResponseOwnershipAndNativeHooks`, `internal/cmd/sdkgen/glance_tasks_test.go:TestGlanceTaskFacadePreservesNativeAndRegistry`.

Python/Go differences:

- Pinned Python mutable Resource/dirty body attributes/same-object returns and configured Adapter/cache/session/microversion/header machinery remain broader than fixed concrete function options and independently owned TaskInfo.
- Go TaskInfo exact canonical optional string/date pointers and nullable raw object Input/Result preserve raw precision and literal timestamps; native Task time.Time/map[string]any and pinned Python untyped result/date values differ. No type/status normalization or cached/input/ID seeding.
- Server policy/owner/expiry and deprecated tasks access remain server-owned; no discovery/Schema/authorization gate or authenticated cloud execution/full SDK parity claim.
- Pinned Python lazypager strips self, supports generalBodyfilters, max_items limit hint, body/HTTPLink/fallbackpagination and mutableResource/sessionframework. Go only guardedcanonical bodynext/localcap/sixquery, sparseRows and nil/errorAll behavior are explicitbounded differences.
- Pinned server list Type filter accepts import only though Create Schema/domain supports3 types; facade no enum gate allows server rejection. Serverconfiguredlimit/default/clamp/owner/expiry remainserver-owned.

Unresolved boundaries:

- Pinned Python mutable Resource/dirty body attributes/same-object returns and configured Adapter/cache/session/microversion/header machinery remain broader than fixed concrete function options and independently owned TaskInfo.
- Go TaskInfo exact canonical optional string/date pointers and nullable raw object Input/Result preserve raw precision and literal timestamps; native Task time.Time/map[string]any and pinned Python untyped result/date values differ. No type/status normalization or cached/input/ID seeding.
- Server policy/owner/expiry and deprecated tasks access remain server-owned; no discovery/Schema/authorization gate or authenticated cloud execution/full SDK parity claim.
- Pinned Python lazypager strips self, supports generalBodyfilters, max_items limit hint, body/HTTPLink/fallbackpagination and mutableResource/sessionframework. Go only guardedcanonical bodynext/localcap/sixquery, sparseRows and nil/errorAll behavior are explicitbounded differences.
- Pinned server list Type filter accepts import only though Create Schema/domain supports3 types; facade no enum gate allows server rejection. Serverconfiguredlimit/default/clamp/owner/expiry remainserver-owned.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package examples

import (
    "context"
    "encoding/json"

    "github.com/JSYoo5B/go-openstacksdk/image"
)

func createTask(ctx context.Context, svc *image.Service) (*image.TaskInfo, error) {
    return svc.CreateTask(ctx, "import",
        image.WithCreateTaskOpts(image.CreateTaskOpts{
            Input: map[string]json.RawMessage{"import_from_format": json.RawMessage(`"qcow2"`)},
        }),
        image.WithCreateTaskHeader("X-Request-Id", "task-create-example"),
        image.WithCreateTaskHeaders(map[string]string{"X-Example": "create"}),
        image.WithCreateTaskInput(map[string]any{
            "import_from": "https://example.invalid/image.qcow2",
            "import_from_format": "qcow2",
            "image_properties": map[string]any{
                "container_format": "bare", "disk_format": "qcow2",
            },
        }))
}

func getTask(ctx context.Context, svc *image.Service, taskID string) (*image.TaskInfo, error) {
    return svc.GetTask(ctx, taskID,
        image.WithGetTaskOpts(image.GetTaskOpts{}),
        image.WithGetTaskHeader("X-Request-Id", "task-get-example"),
        image.WithGetTaskHeaders(map[string]string{"X-Example": "get"}))
}

func listTasks(ctx context.Context, svc *image.Service, marker string) ([]*image.TaskInfo, error) {
    options := []image.ListTasksOption{
        image.WithListTasksOpts(image.ListTasksOpts{}),
        image.WithListTasksHeader("X-Request-Id", "task-list-example"),
        image.WithListTasksHeaders(map[string]string{"X-Example": "list"}),
        image.WithListTasksLimit(20),
        image.WithListTasksMarker(marker),
        image.WithListTasksType("import"),
        image.WithListTasksStatus("pending"),
        image.WithListTasksSortKey("created_at"),
        image.WithListTasksSortDir("desc"),
        image.WithListTasksMaxItems(40),
        image.WithListTasksSinglePage(false),
    }
    rows := make([]*image.TaskInfo, 0)
    for task, err := range svc.Tasks(ctx, options...) {
        if err != nil {
            return nil, err
        }
        rows = append(rows, task)
    }
    return rows, nil
}

func allTasks(ctx context.Context, svc *image.Service) ([]*image.TaskInfo, error) {
    return svc.AllTasks(ctx, image.WithListTasksMaxItems(40))
}
```

```python
def task_examples(conn):
    created = conn.image.create_task(type="import", input={
        "import_from": "https://example.invalid/image.qcow2",
        "import_from_format": "qcow2",
        "image_properties": {"container_format": "bare", "disk_format": "qcow2"},
    })
    fetched = conn.image.get_task(created.id)
    rows = list(conn.image.tasks(type="import", status="pending", limit=20))
    return created, fetched, rows
```

## References

- [Tasks](v2/tasks/README.md)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#tasks)
- [Tasks](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/tasks.py#L70-L155)
- [Tasks](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/tasks.py#L354-L430)
- [`WithCreateTaskNullInput`](python-create-task.md)
- [Task Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/task.py#L16-L53)
- [fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1842)
- [Tasks](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/tasks.py#L308-L351)
- [native v2.15.0 ListOpts](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/tasks/requests.go#L36-L72)
- [Tasks Core Test](tasks_core_test.go)
- [Tasks Options Test](tasks_options_test.go)
- [Tasks Contracts Test](tasks_contracts_test.go)
- [Connection Image Tasks Test](../connection_image_tasks_test.go)
- [Glance Tasks Test](../internal/cmd/sdkgen/glance_tasks_test.go)
