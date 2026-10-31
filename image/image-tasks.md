# Image Tasks

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/image_tasks`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.ImageTasks`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.AllImageTasks`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImageTasksOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImageTasksHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImageTasksHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImageTasksMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.TaskRecords`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.AllTaskRecords`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithTaskRecordsOpts`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithTaskRecordsQuery`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithTaskRecordsHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithTaskRecordsMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithTaskRecordsLocation`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithTaskRecordsLimit`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithTaskRecordsMarker`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithTaskRecordsMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithTaskRecordsPaginated`

- Required validated image Ref fixes parent route. DirectID has no metadata GET/UUID/localmaxlength gate; literalUTF8 safe segments escapeonce. Name is explicitGo extension using captured existing exact image resolver with uniqueness/missing/nativepagination semantics; no Python literal string lookup equivalence.
  Tests: `image/image_tasks_core_test.go:TestImageTaskCoreFixedRouteAndFullRecords`, `image/image_tasks_core_test.go:TestImageTaskCoreCompletePreflight`, `image/image_tasks_core_test.go:TestImageTaskCoreExactNameAndSourceGuards`, `image/image_tasks_contracts_test.go:TestImageTaskContractsFixedRouteAndParentIdentity`, `image/image_tasks_contracts_test.go:TestImageTaskContractsOwnedOptionsAndSource`.
- Lazy ImageTasks copies option slice atconstruction, prepares perexecution; concrete Headers/MaxItems0unlimited/positivecap/negativeerror, FullOpts replacement/lastwins, initialized callback headerconfig copiedbeforeafter, immediate headerfactory snapshots and reusable/parallel execution without caller mutation.
  Tests: `image/image_tasks_options_test.go:TestImageTaskOptionsImmediateHeaderSnapshots`, `image/image_tasks_options_test.go:TestImageTaskOptionsReplacementAndLastWins`, `image/image_tasks_options_test.go:TestImageTaskOptionsCallbacksLazyAndOwned`, `image/image_tasks_options_test.go:TestImageTaskOptionsParallelReusablePolicies`, `image/image_tasks_contracts_test.go:TestImageTaskContractsLazyConsumptionAndLocalCaps`, `image/image_tasks_contracts_test.go:TestImageTaskContractsOwnedOptionsAndSource`.
- One finite bodyless/queryfree fixed images/{id}/tasks actual200 periteration. WholeUTF8/syntax plus canonicalnonnulltasksarray, consumedrows only; localcap/earlybreak skipunusedinvalidrows. next/Link/first/schema/self entirelypassive/no validation/follow/fallback/wirehint; Allnonnilempty ornilerror withoutpartials.
  Tests: `image/image_tasks_core_test.go:TestImageTaskCoreFixedRouteAndFullRecords`, `image/image_tasks_core_test.go:TestImageTaskCoreFiniteConsumptionAndEnvelope`, `image/image_tasks_contracts_test.go:TestImageTaskContractsFixedRouteAndParentIdentity`, `image/image_tasks_contracts_test.go:TestImageTaskContractsLazyConsumptionAndLocalCaps`.
- Distinct ImageTaskInfo embeds unchangedTaskInfo but explicitly decodes deletion atomically: deleted nullableboolpresence/deleted_atliteralstring; canonical optional13strings/rawobjectInputResult/unknownprecision/Header/status independentlyowned. NoID/defaultseeding; actualrequest_id/user_id only, officialsample request-id/user rawaliases remainpassive.
  Tests: `image/image_tasks_core_test.go:TestImageTaskCoreDeletionPresenceAndAtomicDecoder`, `image/image_tasks_core_test.go:TestImageTaskCoreFixedRouteAndFullRecords`, `image/image_tasks_contracts_test.go:TestImageTaskContractsCanonicalDetailAndRawOwnership`, `connectiontest/connection_image_tasks_associated_test.go:TestConnectionImageAssociatedTasksSharesClientAndResolvesName`.
- Capturedoriginalsource/provider/Type/Endpoint/base/Microversion and ordinaryheaders fixedbeforecallbacks throughName resolver/singleGET; stableoptionheadersoverride/originalproviderliveauth. Recheckbeforework/afterresolver/acceptedbody/beforeafterconsumedrows, no source/provider retarget orborrowedsource mutation.
  Tests: `image/image_tasks_core_test.go:TestImageTaskCoreExactNameAndSourceGuards`, `image/image_tasks_options_test.go:TestImageTaskOptionsCallbacksLazyAndOwned`, `image/image_tasks_contracts_test.go:TestImageTaskContractsOwnedOptionsAndSource`, `connectiontest/connection_image_tasks_associated_test.go:TestConnectionImageAssociatedTasksSharesClientAndResolvesName`.
- Strictactual200; acceptedread/Close/context/customcause/source/envelope/model errors retain wholeactualResponseError andnilrow, closeonce/no accepted replay. Nativeunexpectedstatus and configuredprebodyretry/reauth/backoff/identicaltargetredirect/body/status guards remain; direct404 stays error andName resolver nativeownership remainsseparate.
  Tests: `image/image_tasks_core_test.go:TestImageTaskCoreOwnedErrorsAndNativePolicy`, `image/image_tasks_contracts_test.go:TestImageTaskContractsResponseOwnershipAndNativeHooks`.
- Existing TaskInfo/nativegeneralTask/generatedaliases/WaitForTask byteexact and nativeimages has no associatedendpoint; upperaugmentation needsno syntheticnative/catalog/resource/registry operation. Version2.12 sourceendpoint andPython2.17maxcap distinction, no version/policy/Schema discovery or task wait.
  Tests: `image/image_tasks_core_test.go:TestImageTaskCoreFixedRouteAndFullRecords`, `image/image_tasks_contracts_test.go:TestImageTaskContractsFixedRouteAndParentIdentity`, `internal/cmd/sdkgen/glance_image_tasks_test.go:TestGlanceImageAssociatedTasksPreservesNativeAndRegistry`.
- Literal parent escaped once with no parent lookup; Source image_id Body local filter excludes missing/null/foreign rows and keeps returned image_id separate from fixed parent.
  Tests: `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsLiteralParentAndSourceBodyFilter`, `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsRawDescriptorsDefaultsAndProof`.
- Owned untyped Source16 Body/default/computed location projection, arbitrary scalar/list/null input/result/date/deleted/large integers, unknown/self/actualheaders only Wire, independent Resource/Wire/Envelope/Header/discovery receipt.
  Tests: `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsRawDescriptorsDefaultsAndProof`, `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsLocationUsesCurrentScopeNotOwner`.
- Pinned tasks envelope list/singleton, strict Go JSON/UTF8/wrongshape, Source list constructor collisions retain actual accepted wholepage status/body/headers and nonnil empty success.
  Tests: `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsEnvelopeSingletonAndConstructorProof`.
- Canonical Body local filtering, unknown discard, limit/marker server query, Source dict truthiness/subset/mapping.get errors and bool-number equality; binary64 decimal underflow with documented exact decimal equality profile.
  Tests: `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsLocalFiltersUnknownAndQueryControls`.
- Retained inherited list plural/next/HTTP links/exact v2 alias/dictionarycompatibility, stop-onempty, shortpage marker, raw cap before filters and unconsumed invalid row/next, falsey binary64 next, origin/collection/filter/cycle and latepage actual error proof.
  Tests: `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsMarkerCapsAndUnconsumedRows`, `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsContinuationAndLateErrors`.
- Lazy per-execution concrete owned opts/rawbytes/pointers/maps/location/optionslice and retainedcallback isolation, reusable concurrent outputs, invalid control/header/query/identity/cap before HTTP; cancellation/source/type causes retained.
  Tests: `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsLazySnapshotsReusableAndConcurrent`, `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsPreflightAndContextCauses`, `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsLocationUsesCurrentScopeNotOwner`.
- Captured selected microversion preserved; absent default finite major2 discovery/Source2.17 ceiling, API v2.17 ID does not imply microversion maximum, explicit empty/selectedlatest policies and separate actual discovery proof; explicit list headers never leak into discovery.
  Tests: `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsSelectedVersionAndAcceptOwnership`, `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsDiscoveryCeilingAndAPIMinorIsNotMicroversion`, `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsDiscoveryFailuresRetainActualProof`.
- Parent missing/denied statuses remain actual errors, no ignore_missing/find fallback; bounded Go final200..399 versus pinned Source<400, native informational199 rejection proof.
  Tests: `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsMissingErrorsAndBoundedStatuses`.
- Source LIST AcceptJSON default/explicit safe override; clean native retry/reauth fixed route/query/body/version; invalid body/header/custom callback/required-absent auth/framing/foreign version and Omitversion preserve original native rejection and causes before resend. OkCodes hook changes can allow retry2 but final actual response remains checked against original200..399 acceptance, including actual second400 rejection.
  Tests: `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsSelectedVersionAndAcceptOwnership`, `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsNativeRetryAndReauthentication`, `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsRetryForbiddenHeadersAndOmittedVersion`.
- Accepted/rejected physical Read/Close/both/cancel/transient Source restored errors preserve actual envelope/body/header/status, all causes and exactly one physical Close; no resend/fallback.
  Tests: `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsPhysicalProofAndStickySource`.
- Unselected If-Match/If-None-Match/If-Modified-Since/If-Unmodified-Since stay required-absent during retry; captured/per-call selected values remain fixed and changed/omitted conditions fail before resend. Actual original503/body/header and custom callback cause survive independently.
  Tests: `image/v2/images/python_tasks_contracts_test.go:TestPythonImageTasksRecordsConditionalHeadersAreOwned`.

Python/Go differences:

- Pinned Python Resource._get_id uses literalstring or Resource.id despite name prose. Go requires Ref and offers explicit Name resolver with native exactlookup requests; absent/ambiguous/forbidden parent errors remainerrors.
- Pinned Python mutableResource/untypedBody/sessionAdapter/configuredcache/microversion/generalpager behavior is broader than fixed concrete functionoptions and independentlyowned canonical ImageTaskInfo. No Resource dirty/cache/session or same-object parity claim.
- SQL JSONEncodedDict reads arbitraryJSON and Python Body values are untyped, while unchanged TaskInfo restricts Input/Result to nullable objects. Nonobject response values are owned200 decode errors, not full JSON parity; literal date/boolshape semantics are explicitGo policy.
- Actual endpoint is finite queryfree fullDBrow response. Go ignores passive next/Link/schema regardlesstype and applieslocalcap only; Python generalResource list mayusebodyfilters/max_items-to-limit hints and broaderpagercontinuation. Server visibility/expiry/deletion/order and authorization remainserver-owned.
- Official API since2.12 and Python max2.17 do not imply a client minimum/version capability gate. No discovery/Schema/timecoercion/automatic wait/task cleanup/livecloud or wholeSDK completion evidence.
- Pinned image_tasks proxy accepts only image selector and does no name lookup despite prose. Go literal UTF8 string selector and concrete Query/Headers/Microversion/Location/list controls are explicit SDK conveniences inspired by inherited Resource.list, not extra Python proxy kwargs. Invalid positional/constructor controls are rejected rather than runtime Python call errors.
- Declared ImageTasks untyped16 Body fields plus computed location are owned snapshots separate from actual row/wholepage receipt. Mutable Resource/dirty/cache/same-object/class/subclass/session and global discovery caching remain independent shared SDK foundations; this named record does not claim them complete.
- Default current location uses recorded native Keystone auth scope without HTTP; cloud/region/auth-config names are not invented. Optional CloudLocation is snapshotted and preserves explicit zone. Task owner is not a project descriptor and does not overwrite current location.
- Existing typed image.Service.ImageTasks remains unchanged: finite queryfree HTTP200, typed ImageTaskInfo scalar/date/bool/object limits and explicit Ref Name resolver. New images.API.TaskRecords provides Source literal parent, mandatory image_id Body localfilter, untyped JSON and retained pager as a sibling.
- Sequential Query and literal response dictionary order is preserved; bulk maps/new callback keys are sorted because Go maps have no kwargs insertion order. Source dict truthiness uses binary64 decimals while equality uses shared exact decimal JSON plus bool-number behavior, not arbitrary Python float rounding/nonfinite/runtime objects.
- Go typed/raw limit requires one nonnegative machine integer and marker one nonblank UTF8 value. Safe literal route, request origin/collection/filter/protocol and retry ownership reject broad Source callback/continuation retargeting. Exact relative /v2/images/parent/tasks alias is supported; dictionary links are explicit Go compatibility. Cap bounds raw rows before local filtering and may omit later matches.
- Selected client or explicit Microversion is preserved. Absent default uses finite eligible Glance major2 advertisement and ceiling2.17; API minor ID is not maxmicroversion and bounds-free stock Glance stays headerless. No minimum2.12 feature gate, Schema lookup, automatic task wait/cleanup or persistent KSA/session cache is claimed.
- Pinned Source raise_from_response returns for status<400; Go intentionally accepts final200..399 and rejects informational100..199 with actual native receipt. Strict UTF8 differs from Requests replacement decoding, and malformed/empty/wrongshape list JSON remains a terminal accepted-response error.
- Stock policy evidence is pinned Glance controller57f7dd9 get_image check plus official Glance2025.2 policy documentation, not a claim that57f7dd9 is a31.0.0 release tag. Image-scoped tasks is user reader subject to visibility; external /tasks tasks_api_access admin is outscope. Fixture and compile validation is distinct from live deployment behavior.
- Conditional request headers are fixed if captured or explicitly selected, and otherwise required absent; callback injection/retargeting is a bounded Go guard. Native OkCodes hook changes are not blanket preflight-rejected: replay may occur, with final response checked against original200..399 range and actual outside-range receipt retained.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package examples

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func imageTaskExamples(ctx context.Context, svc *image.Service, imageID string) ([]*image.ImageTaskInfo, error) {
    options := []image.ListImageTasksOption{
        image.WithListImageTasksOpts(image.ListImageTasksOpts{MaxItems: 0}),
        image.WithListImageTasksHeader("X-Request-Id", "image-tasks-example"),
        image.WithListImageTasksHeaders(map[string]string{"X-Example": "tasks"}),
        image.WithListImageTasksMaxItems(20),
    }
    rows := make([]*image.ImageTaskInfo, 0)
    for task, err := range svc.ImageTasks(ctx, resource.ID(imageID), options...) {
        if err != nil {
            return nil, err
        }
        rows = append(rows, task)
    }
    return rows, nil
}

func allImageTaskExamples(ctx context.Context, svc *image.Service, imageID string) ([]*image.ImageTaskInfo, error) {
    return svc.AllImageTasks(ctx, resource.ID(imageID), image.WithListImageTasksMaxItems(20))
}
```

```python
def image_tasks_example(conn, image_id):
    return list(conn.image.image_tasks(image_id))
```

## References

- [Tasks](tasks.md)
- [Images Images V2](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-images-v2.inc#L311-L350)
- [ImageTasks Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image_tasks.py#L20-L61)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1181-L1194)
- [Controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L599-L614)
- [API](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/sqlalchemy/api.py#L1834-L1873)
- [SQL formatter](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/sqlalchemy/api.py#L2018-L2043)
- [Image Tasks Core Test](image_tasks_core_test.go)
- [Image Tasks Options Test](image_tasks_options_test.go)
- [Image Tasks Contracts Test](image_tasks_contracts_test.go)
- [Connection Image Tasks Associated Test](../connectiontest/connection_image_tasks_associated_test.go)
- [Glance Image Tasks Test](../internal/cmd/sdkgen/glance_image_tasks_test.go)
