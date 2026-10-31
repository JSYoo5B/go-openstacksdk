# Member Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:image/v2/find_member`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.FindImageMember`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageMemberOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageMemberHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageMemberHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageMemberIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.FindImageMemberRecord`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageMemberRecordOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageMemberRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageMemberRecordHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithFindImageMemberRecordIgnoreMissing`

- All methods require an explicit parent image Ref; direct imageID has only fixed member endpoint, explicit parentName uses unchanged native exact-name resolver. No Member parent recovery, memberName/project lookup, response-ID synthesis or response-driven routes.
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_core_test.go:TestImageMembersCoreFixedRoutesAndActualEvidence`, `connectiontest/connection_image_members_test.go:TestConnectionImageMembersShareClientAndConcreteDefaults`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
- Canonical ImageMember requires validUTF8 nonnull JSON object. Optional image_id/member_id/status/schema/created_at/updated_at are strict nullable strings; missing/null nil, empty present, literal dates unparsed. Unknown/case-decoy/large extension numbers are independent raw fields, links passive/nil typed. Mismatched response IDs and arbitrary response status remain passive; atomic schema errors retain original actual200 proof.
  Tests: `image/members_contracts_test.go:TestImageMembersCanonicalRawModelsAndFiniteList`, `image/members_core_test.go:TestImageMembersCoreCanonicalPresenceAndAtomicDecoder`.
- Source/context/client/provider/prefix/ordinary headers captured before usercallbacks. MemberID is nonempty safeUTF8 single unescaped segment; status input exactly pending/accepted/rejected; invalid inputs/protectedheaders/nilerrorcallbacks fail before parentName/memberHTTP. ParentName failures remain errors despite missing defaults.
  Tests: `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_core_test.go:TestImageMembersCoreCompletePreflightAndParentMissing`, `image/members_options_test.go:TestImageMembersOptionsValidationAndDefaults`.
- Concrete FullOpts replaces configuration, Header/Headers merge; map/pointer/helper input/optionslice/percallback configs independently copied and callbacks applied once. Prepared route/headers stay fixed with live originalprovider token; callback/source/provider and postName changes rejected. Reuse and parallel helpers retain owned settings.
  Tests: `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_core_test.go:TestImageMembersCoreCapturedSourceAndRequestOwnership`, `image/members_options_test.go:TestImageMembersOptionsSnapshotsReplacementAndLastWins`, `image/members_options_test.go:TestImageMembersOptionsCallbacksPointersAndSliceOwnership`, `image/members_options_test.go:TestImageMembersOptionsParallelReusableHelpers`.
- Accepted200 typed failures read/Close/context/UTF8/JSON/schema return nil model plus ownedResponseError retaining actual bytes/header/status and original read/Close/ctx.Err/customcause. Remove actual204 yields opaque actualack including nonnilack+error on handlingfailure. BodyClose once, no acceptedbody replay; rejectedstatuses keepnative ownership.
  Tests: `image/members_contracts_test.go:TestImageMembersAcceptedEvidenceAndMutationNoReplay`, `image/members_core_test.go:TestImageMembersCoreOwnedBodyFailuresAndPhysical404`.
- Shared REST/fixedrequest keeps configured native prebody retry/reauth/backoff and identical-target redirect callbacks, blocks changedmethod/origin/path/query before transport. Owned serializedbody/RawBody/KeepResponseBody/JSONResponse cannot mutate; nilbody differsJSONnull, same serializedreplacement allowed. Actual status gate remains200/204 despite hook-expandedOkCodes. Native reauth ErrOriginal/ErrReauth fields and existing native memberABI/scopes unchanged.
  Tests: `image/members_contracts_test.go:TestImageMembersNativeHooksAndCompatibility`, `image/members_core_test.go:TestImageMembersCoreCapturedSourceAndRequestOwnership`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
- Find is literal fixedmember GET only, no400/403/LIST/memberName fallback or ID equality assertion. IgnoreMissing nil/defaulttrue accepts only owned fullyhandled GET404=>nil,nil, failure retains404 ResponseError. False preserves strict200/native404 callback/retry; parentName/logical/nested404 neverignored. Server may mask authorization404, so nil is not absence proof.
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_contracts_test.go:TestImageMembersAcceptedEvidenceAndMutationNoReplay`, `image/members_core_test.go:TestImageMembersCoreOwnedBodyFailuresAndPhysical404`, `image/members_contracts_test.go:TestImageMembersNativeHooksAndCompatibility`, `image/members_core_test.go:TestImageMembersCoreCompletePreflightAndParentMissing`.
- Owned Find seeds the exact member string as id and accepts actual 200..399 GET. Invalid JSON/empty accepted bodies retain seeded defaults with Wire nil; valid nonobject/whole UTF-8 faults fail. Known Body response attrs overlay the seed with ordered alias/null semantics while parent URI/location stay fixed.
  Tests: `image/member_record_find_test.go:TestFindImageMemberRecordSeededGetAndTolerance`.
- Only clean final native direct-GET 400/403/404 enables the shared query-free member list. Resource.Collection.FindIdentity matches exact declared ID or inherited name, detects a second match immediately, scans later pages after the first candidate, discards candidates on later failure and never refetches a selected row.
  Tests: `image/member_record_find_test.go:TestFindImageMemberRecordCompatibleFallbackAndUniqueness`.
- Rejected GET 400/403/404 Read/Close/source failures remain terminal with original native proof. Glance explicitly preserves an identical native RetryFunc return as a clean final rejection; extra hook causes, wrapped/nested errors, accepted IO faults and SDK status-policy failures cannot enable fallback. The default retry policy of unrelated workflows remains unchanged.
  Tests: `image/member_record_find_test.go:TestFindImageMemberRecordNativeHooksAndTerminalFailures`, `image/member_records_test.go:TestImageMemberRecordsReadCloseAndStickyGuardEvidence`, `blockstorage/snapshot_mutation_transport_test.go:TestVolumeSnapshotMutationDelete404RequiresCleanReadCloseAndGuard`, `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairCreateAndDeleteOutcomes`.
- IgnoreMissing defaults to true only after successful exhaustive lookup finds no match; false exposes ErrNotFound and duplicate/physical/list/parent-resolution failures remain errors. Present id:null suppresses alternate identity while inherited name can still match. The original nonblank UTF-8 member string is compared unchanged and escaped as one GET segment.
  Tests: `image/member_record_find_test.go:TestFindImageMemberRecordCompatibleFallbackAndUniqueness`, `image/member_record_find_test.go:TestFindImageMemberRecordPreflightAndLegacyBoundaries`.
- One operation captures source, headers, options and current location before callbacks and shares the prepared parent/client between GET and fallback pages. Callback ownership and original provider/token are preserved; current source changes are checked at each physical and yield boundary.
  Tests: `image/member_record_find_test.go:TestFindImageMemberRecordOptionsLocationAndFallbackSnapshot`, `image/member_records_test.go:TestImageMemberRecordsReadCloseAndStickyGuardEvidence`.
- Cached Connection Image and ImageV2 calls share native clients, option factory snapshots and location across the complete GET-list search; selected rows retain actual page status/proof. Existing literal FindImageMember and generated scoped Members remain unchanged.
  Tests: `connectiontest/connection_image_member_records_test.go:TestConnectionImageMemberRecordsShareLocationAndFallback`, `image/member_record_find_test.go:TestFindImageMemberRecordPreflightAndLegacyBoundaries`.

Python/Go differences:

- Existing FindImageMember retains its seven legacy literal-GET contracts above; FindImageMemberRecord supplies owned GET-first exact ID/name search and duplicate detection without changing that ABI.
- Go returns independent ImageMemberRecord values rather than reusing mutable cached Python Member instances. Explicit parent-name resolution and request header options are Go extensions. ErrNotFound/ErrAmbiguous replace Python exception classes and the upcoming-default warning is not emitted.
- Only selected rejected GET 400/403/404 bodies receive physical IO observation; other native rejected statuses/list attempts retain existing native IO policy. Whole UTF-8, protected headers, safe literal input and fixed-route continuation bounds remain explicit Go differences.
- Glance server may mask authorization as 404; missing nil is not proof of absence. Source inspection/local HTTP tests do not establish authenticated cloud behavior, and full Resource/session/cache/discovery remains a separate SDK objective.

### `python:image/v2/members`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image.Service.ListImageMembers`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImageMembersOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImageMembersHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImageMembersHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithListImageMembersMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.AllImageMembers`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.ListImageMemberRecords`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.AllImageMemberRecords`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListHeader`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListFilter`
- `github.com/JSYoo5B/go-openstacksdk/image.WithImageMemberRecordListFilters`

- All methods require an explicit parent image Ref; direct imageID has only fixed member endpoint, explicit parentName uses unchanged native exact-name resolver. No Member parent recovery, memberName/project lookup, response-ID synthesis or response-driven routes.
  Tests: `image/members_contracts_test.go:TestImageMembersFixedParentRoutesAndBodies`, `image/members_core_test.go:TestImageMembersCoreFixedRoutesAndActualEvidence`, `connectiontest/connection_image_members_test.go:TestConnectionImageMembersShareClientAndConcreteDefaults`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
- Finite collection requires whole validUTF8 nonnull JSONobject and canonical nonnull membersarray. Consumed ImageMember rows atomically decode nullable string fields/literaldates and own independent raw unknownfields/header/status; unusedrowcanonical errors skipped oncap/earlybreak. No envelope/schema/link pagination or response-ID routing.
  Tests: `image/members_contracts_test.go:TestImageMembersCanonicalRawModelsAndFiniteList`, `image/members_core_test.go:TestImageMembersCoreCanonicalPresenceAndAtomicDecoder`, `image/members_core_test.go:TestImageMembersCoreFiniteLazyListAndConsumedRows`.
- Source/context/client/provider/prefix/ordinary headers captured before usercallbacks. MemberID is nonempty safeUTF8 single unescaped segment; status input exactly pending/accepted/rejected; invalid inputs/protectedheaders/nilerrorcallbacks fail before parentName/memberHTTP. ParentName failures remain errors despite missing defaults.
  Tests: `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_core_test.go:TestImageMembersCoreCompletePreflightAndParentMissing`, `image/members_options_test.go:TestImageMembersOptionsValidationAndDefaults`.
- Concrete FullOpts replaces configuration, Header/Headers merge; map/pointer/helper input/optionslice/percallback configs independently copied and callbacks applied once. Prepared route/headers stay fixed with live originalprovider token; callback/source/provider and postName changes rejected. Reuse and parallel helpers retain owned settings.
  Tests: `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_core_test.go:TestImageMembersCoreCapturedSourceAndRequestOwnership`, `image/members_options_test.go:TestImageMembersOptionsSnapshotsReplacementAndLastWins`, `image/members_options_test.go:TestImageMembersOptionsCallbacksPointersAndSliceOwnership`, `image/members_options_test.go:TestImageMembersOptionsParallelReusableHelpers`.
- Collection acceptedread/Close/context/wholeUTF8/JSON/envelopeschema failure yields terminalnilrow+ownedResponseError retaining full original200 bytes/header/status and causes. Consumed malformedrow retains same fullproof; All discards partialrows. BodyClose once and no acceptedbody replay.
  Tests: `image/members_contracts_test.go:TestImageMembersAcceptedEvidenceAndMutationNoReplay`, `image/members_core_test.go:TestImageMembersCoreOwnedBodyFailuresAndPhysical404`, `image/members_contracts_test.go:TestImageMembersCanonicalRawModelsAndFiniteList`, `image/members_core_test.go:TestImageMembersCoreFiniteLazyListAndConsumedRows`.
- Shared REST/fixedrequest keeps configured native prebody retry/reauth/backoff and identical-target redirect callbacks, blocks changedmethod/origin/path/query before transport. Owned serializedbody/RawBody/KeepResponseBody/JSONResponse cannot mutate; nilbody differsJSONnull, same serializedreplacement allowed. Actual status gate remains200/204 despite hook-expandedOkCodes. Native reauth ErrOriginal/ErrReauth fields and existing native memberABI/scopes unchanged.
  Tests: `image/members_contracts_test.go:TestImageMembersNativeHooksAndCompatibility`, `image/members_core_test.go:TestImageMembersCoreCapturedSourceAndRequestOwnership`, `internal/cmd/sdkgen/glance_members_test.go:TestGlanceMemberWorkflowsPreserveNativeScopeAndConcreteDefaults`.
- List snapshot is lazy periteration; each iteration makes one finite body/query-freeGET200 after required parentresolution. Required canonical members nonnullarray, wholeUTF8/JSON before rows; rowcanonical decode onlywhenconsumed. MaxItems zero unlimited,negative invalid,positive localrawrowcap; earlybreak/cap skipsunusedrowerrors/no continuation/query/fanout. Eachrow owns metadata/header/raw, source/context checked perrow. All helper returns nonnil[] onempty and discards partial onerror.
  Tests: `image/members_contracts_test.go:TestImageMembersCanonicalRawModelsAndFiniteList`, `image/members_core_test.go:TestImageMembersCoreFiniteLazyListAndConsumedRows`, `image/members_contracts_test.go:TestImageMembersOwnedOptionsAndPreflight`, `image/members_options_test.go:TestImageMembersOptionsValidationAndDefaults`.
- Owned member records add the pinned nine-field declared view beside unchanged finite typed APIs. Untyped Body values and ordered member/member_id aliases preserve raw precision, canonical alias precedence and present null ID; fixed parent URI and current location override passive response fields. Resource, Wire, Envelope, headers and identity own independent bytes.
  Tests: `image/member_records_test.go:TestImageMemberRecordsProjectionAndOwnedReceipts`, `image/member_records_test.go:TestImageMemberRecordsIdentityAliasesAndLocation`.
- Public Proxy.members drops all query kwargs. The owned default is query-free with generic pagination enabled; explicit Go options add server limit/marker, raw-row cap, pagination, headers and declared local filters. Unknown filter keys are ignored before encoding and exact JSON semantic filtering reuses the shared selector.
  Tests: `image/member_records_test.go:TestImageMemberRecordsPreflightAndExtensionFilters`, `image/member_records_test.go:TestImageMemberRecordsLazyRepeatedOptionSnapshots`.
- Shared guarded pagination follows fixed collection links and HTTP Link, ignores Source-falsey top-level next values and permits initial-positive-limit wire-marker fallback. Unused rows/continuations are skipped at raw cap or early break; partial All results survive later errors, and nil/empty/member-object/array envelopes follow the declared collection policy.
  Tests: `image/member_records_test.go:TestImageMemberRecordsPagingAndRawConsumption`, `internal/rest/falsey_next_test.go:TestListFalseyNextIsOptInAndAllowsHeaderOrMarkerFallback`, `internal/rest/falsey_next_test.go:TestListTruthyNonstringNextAndDefaultFalseyTypeKeepExactPageEvidence`.
- Owned pages accept actual 200..399 and require whole UTF-8 JSON with the mandatory members envelope. Consumed constructor binding collisions and malformed rows are terminal with actual page proof; accepted Read/Close/context/source faults retain independent receipt and cause without replay.
  Tests: `image/member_records_test.go:TestImageMemberRecordsEnvelopeFailuresAndActualStatus`, `image/member_records_test.go:TestImageMemberRecordsReadCloseAndStickyGuardEvidence`.
- Options/helpers and iteration slices snapshot maps and pointers, full options replace and merge helpers use last-wins. Each lazy iteration captures source and headers before callbacks/location, resolves an explicit parent once, retains live provider token, and makes observed source faults sticky across Read and Close.
  Tests: `image/member_records_test.go:TestImageMemberRecordsLazyRepeatedOptionSnapshots`, `image/member_records_test.go:TestImageMemberRecordsPreflightAndExtensionFilters`, `image/member_records_test.go:TestImageMemberRecordsReadCloseAndStickyGuardEvidence`.
- Both Connection Image/ImageV2 entries retain the same native service client and current location through owned list and GET-list fallback. Legacy finite member list, literal Find and generated native member ABI remain separate and unchanged.
  Tests: `connectiontest/connection_image_member_records_test.go:TestConnectionImageMemberRecordsShareLocationAndFallback`, `image/member_records_test.go:TestImageMemberRecordsPreserveLegacyFiniteMemberList`, `image/member_record_find_test.go:TestFindImageMemberRecordPreflightAndLegacyBoundaries`.

Python/Go differences:

- Existing ListImageMembers/AllImageMembers retain their seven legacy contracts above; ListImageMemberRecords/AllImageMemberRecords supply the owned pinned view and generic pager without changing that ABI.
- Go limit/marker/max-items/paginated/header/local-filter options and explicit parent-name resolution are extensions; the pinned public members proxy drops every query kwarg. Go exposes fixed image_id URI provenance in the declared view although Python to_dict excludes URI fields by default.
- Whole UTF-8, protected headers, safe literal parent identity, fixed-origin/path/query pagination and conflicting-link rejection are Go bounds. Exact decimal truthiness can differ from Python binary underflow; physical Close faults remain errors.
- Mutable Python Resource/cache/session/discovery identity remains a separate whole-SDK objective. These named contracts use Source inspection and local HTTP tests; authenticated OpenStack/Python runtime behavior was not executed.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
image_id = "image-id"

for member in conn.image.members(image_id):
    print(member.to_dict())

member = conn.image.find_member("project-id", image_id, ignore_missing=False)
print(member.to_dict())
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
    imageID := flag.String("image", "", "literal parent image ID")
    find := flag.String("find", "", "optional member ID or inherited name")
    status := flag.String("status", "", "optional local member status filter")
    limit := flag.Int("limit", 0, "optional server page limit")
    maxItems := flag.Int("max-items", 0, "maximum consumed rows before filtering")
    flag.Parse()
    if *imageID == "" { log.Fatal("-image is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *find, *status, *limit, *maxItems); err != nil {
        var response *resource.ResponseError
        if errors.As(err, &response) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", response.StatusCode, len(response.Body))
        }
        log.Fatal(err)
    }
}

func printRecord(record *image.ImageMemberRecord) error {
    value := map[string]any{"resource": record.Resource, "status_code": record.StatusCode}
    body, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(body))
    return nil
}

func run(ctx context.Context, cloud, imageID, find, status string, limit, maxItems int) error {
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }
    parent := resource.ID(imageID)
    options := []image.ImageMemberRecordListOption{
        image.WithImageMemberRecordListOpts(image.ImageMemberRecordListOpts{Limit: limit, MaxItems: maxItems}),
        image.WithImageMemberRecordListHeader("X-Request-Source", "member-records-example"),
    }
    if status != "" {
        options = append(options, image.WithImageMemberRecordListFilter("status", status))
    }
    for record, readErr := range service.ListImageMemberRecords(ctx, parent, options...) {
        if readErr != nil { return readErr }
        if err := printRecord(record); err != nil { return err }
    }
    if find != "" {
        record, err := service.FindImageMemberRecord(ctx, parent, find,
            image.WithFindImageMemberRecordIgnoreMissing(false))
        if err != nil { return err }
        return printRecord(record)
    }
    return nil
}
```

## References

- [Members](members.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1289-L1354)
- [Install](../docs/install.md)
- [Member](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/member.py#L16-L43)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2524-L2582)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L251-L278)
- [Member Records Test](member_records_test.go)
- [Member Record Find Test](member_record_find_test.go)
- [Falsey Next Test](../internal/rest/falsey_next_test.go)
- [Members Contracts Test](members_contracts_test.go)
