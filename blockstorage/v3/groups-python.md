# Groups Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/create_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordCreateField`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordCreateMicroversion`

- POST groups uses known raw dirty attrs in group envelope with null/false/zero/empty and unknown discard before marshalling; response retains request known seeds, server overrides and raw proof without request-side list coercion.
  Tests: `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupCreatePartialNullUnknownAndOptionSnapshots`, `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupOptionsPreflightUnknownDiscardAndStickySource`.
- Native retry cannot change fixed selected versions, reserved headers, body Content-Type/JSON/retention or omit an owned header; original503 and callback causes remain oneHTTP. Ordinary headers/live token retry remains native.
  Tests: `blockstorage/v3/groups/python_policy_test.go:TestPythonCinderGroupNativeRetryOwnedVersionReservedHeadersAndBody`, `blockstorage/v3/groups/python_policy_test.go:TestPythonCinderGroupNativeRetryOrdinaryHeadersLiveTokenAndOriginalStatus`.
- Bound classmethod cls input is a reserved control, including explicit null/false in individual Field/Query options and bulk raw Opts. It fails before discovery/mutation without synthetic HTTP proof.
  Tests: `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupClassInputsAreBoundControlsBeforeDiscovery`.

Python/Go differences:

- Go accepts a literal nonblank UTF-8 ID and owned JSON-domain record/options rather than mutable Resource/Munch/subclasses. Connection-derived computed location is null in the standalone leaf; cross-call dirty state and dynamic session customization remain shared SDK-R1/C1/S1 boundaries.
- Owned records are independent snapshots, not shared Proxy/session HTTP response cache entries. Repeated Get performs HTTP; zero-HTTP response-cache hits/invalidation are not claimed.
- Go preserves actual admitted/rejected read/Close/cancel/source/callback evidence, owns version/body/reserved protocol headers through native callbacks and rejects route/control overrides. Ordinary retry headers and live provider tokens retain native policy; Python arbitrary Adapter behavior is not reproduced.
- Original HTTP100..399 admission and physical400..599 rejections are retained. InvalidUTF8 and valid nonobject JSON are actual-response errors; accepted empty/malformed JSON preserves Source seed behavior.

### `python:block_storage/v3/create_group_from_source`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.API.CreateFromSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordSourceGroupSnapshotID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordSourceSourceGroupID`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordSourceName`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordSourceDescription`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordSourceHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordSourceMicroversion`

- POST groups/action create-from-src requires both nullable source IDs and defaults name/description to null. Fresh Source response has no request seed, while supplied fields and options own their bytes.
  Tests: `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupCreateFromSourceRequiredNullableIDsAndFreshResponse`.

Python/Go differences:

- Go accepts a literal nonblank UTF-8 ID and owned JSON-domain record/options rather than mutable Resource/Munch/subclasses. Connection-derived computed location is null in the standalone leaf; cross-call dirty state and dynamic session customization remain shared SDK-R1/C1/S1 boundaries.
- Owned records are independent snapshots, not shared Proxy/session HTTP response cache entries. Repeated Get performs HTTP; zero-HTTP response-cache hits/invalidation are not claimed.
- Go preserves actual admitted/rejected read/Close/cancel/source/callback evidence, owns version/body/reserved protocol headers through native callbacks and rejects route/control overrides. Ordinary retry headers and live provider tokens retain native policy; Python arbitrary Adapter behavior is not reproduced.
- Original HTTP100..399 admission and physical400..599 rejections are retained. InvalidUTF8 and valid nonobject JSON are actual-response errors; accepted empty/malformed JSON preserves Source seed behavior.

### `python:block_storage/v3/delete_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.API.DeleteRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordDeleteVolumes`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordDeleteMicroversion`

- Delete uses POST member action delete/delete-volumes:false, not DELETE. Explicit true and strict ignore_missing:false are owned; only clean404 is suppressed and successful body is not decoded.
  Tests: `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupDeleteActionDefaultsStrictMissingAndStatuses`.
- Rejected404 read/Close/callback or expanded original admission never establishes suppressible absence; exact native method/status/body/header and original causes remain.
  Tests: `blockstorage/v3/groups/python_policy_test.go:TestPythonCinderGroupPhysicalAcceptedRejectedAndCancellationProof`, `blockstorage/v3/groups/python_policy_test.go:TestPythonCinderGroupNativeRetryOrdinaryHeadersLiveTokenAndOriginalStatus`.

Python/Go differences:

- Go accepts a literal nonblank UTF-8 ID and owned JSON-domain record/options rather than mutable Resource/Munch/subclasses. Connection-derived computed location is null in the standalone leaf; cross-call dirty state and dynamic session customization remain shared SDK-R1/C1/S1 boundaries.
- Owned records are independent snapshots, not shared Proxy/session HTTP response cache entries. Repeated Get performs HTTP; zero-HTTP response-cache hits/invalidation are not claimed.
- Go preserves actual admitted/rejected read/Close/cancel/source/callback evidence, owns version/body/reserved protocol headers through native callbacks and rejects route/control overrides. Ordinary retry headers and live provider tokens retain native policy; Python arbitrary Adapter behavior is not reproduced.
- Original HTTP100..399 admission and physical400..599 rejections are retained. InvalidUTF8 and valid nonobject JSON are actual-response errors; accepted empty/malformed JSON preserves Source seed behavior.

### `python:block_storage/v3/get_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.API.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordGetListVolume`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordGetField`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordGetMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordGetHeaders`

- Literal ID GET preserves omission versus explicit list_volume=False/True and explicit selected microversion above3.38; known untyped/list Source descriptors, null defaults and raw unknown/large-number proof are independent.
  Tests: `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupGetProjectionListVolumeAndOwnedProof`.
- Source deprecated declared attr kwargs seed response state without GET query/body; server known fields override seeds. Invalid envelope/UTF8 retains actual response; clean404 is terminal without Find fallback.
  Tests: `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupGetDeclaredDeprecatedAttrsSeedWithoutWireQuery`, `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupGetEnvelopeSeedsAndActualDecodeErrors`.
- An absent default negotiates at most3.38 via existing Cinder discovery, with separate actual discovery evidence; denied discovery stops before member HTTP.
  Tests: `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupDiscoveryCeilingSelectedOverridesAndOwnedEvidence`.
- Bound classmethod cls input is a reserved control, including explicit null/false in individual Field/Query options and bulk raw Opts. It fails before discovery/mutation without synthetic HTTP proof.
  Tests: `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupClassInputsAreBoundControlsBeforeDiscovery`.
- Unknown cls in a direct accepted member response is retained in Wire/Envelope and omitted from declared Resource fields; it is not a list constructor collision.
  Tests: `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupListClassCollisionKeepsAcceptedPageAndDirectUnknown`.

Python/Go differences:

- Go accepts a literal nonblank UTF-8 ID and owned JSON-domain record/options rather than mutable Resource/Munch/subclasses. Connection-derived computed location is null in the standalone leaf; cross-call dirty state and dynamic session customization remain shared SDK-R1/C1/S1 boundaries.
- Owned records are independent snapshots, not shared Proxy/session HTTP response cache entries. Repeated Get performs HTTP; zero-HTTP response-cache hits/invalidation are not claimed.
- Go preserves actual admitted/rejected read/Close/cancel/source/callback evidence, owns version/body/reserved protocol headers through native callbacks and rejects route/control overrides. Ordinary retry headers and live provider tokens retain native policy; Python arbitrary Adapter behavior is not reproduced.
- Original HTTP100..399 admission and physical400..599 rejections are retained. InvalidUTF8 and valid nonobject JSON are actual-response errors; accepted empty/malformed JSON preserves Source seed behavior.
- Go warns neither on renamed/deprecated Python kwargs nor Resource deprecation. Known deprecated attrs can be seeded with RecordGetOpts.Fields/WithRecordGetField; ID/route/session control attrs are SDK owned and rejected before HTTP.

### `python:block_storage/v3/groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordListDetails`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordListMaxItems`

- Lazy default details route, canonical all_projects wire alias precedence, discarded unknown/list_volume kwargs, Source known Body local filters and descriptor/default conversion are library owned. Raw cap/hint stops before unconsumed rows or links.
  Tests: `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupListDeclaredAliasesLocalFiltersAndRawCap`, `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupListLocalDictionaryFailuresKeepActualPage`.
- Summary route, groups_links/next/HTTP Link/dictionary links and consumer break use common guarded REST paging. MarkerOrOffsetPagination follows advertised offset even with no initial offset or an initial marker, drops the previous cursor family, then falls back to the last raw Wire ID with limit. Origin/project collection/initial filters stay locked and repeat/backward offsets fail with the actual accepted page receipt.
  Tests: `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupListDetailsLinksMarkersOffsetsAndConsumerBreak`, `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupListShapesConstructorCollisionsAndActualLateError`, `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupListAdvertisedOffsetTransitionsKeepScopeAndFallback`, `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupListOffsetContinuationRejectsScopeAndCursorRegressions`, `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupListClassCollisionKeepsAcceptedPageAndDirectUnknown`.
- Bound classmethod cls input is a reserved control, including explicit null/false in individual Field/Query options and bulk raw Opts. It fails before discovery/mutation without synthetic HTTP proof.
  Tests: `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupClassInputsAreBoundControlsBeforeDiscovery`.
- Declared Body local filters use the common exact decimal and exact nonzero truthiness policy, preserving wire numbers/proof while explicitly differing from Source binary64 rounding and underflow.
  Tests: `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupListDeclaredNumericFiltersUseExactDecimalPolicy`.

Python/Go differences:

- Go accepts a literal nonblank UTF-8 ID and owned JSON-domain record/options rather than mutable Resource/Munch/subclasses. Connection-derived computed location is null in the standalone leaf; cross-call dirty state and dynamic session customization remain shared SDK-R1/C1/S1 boundaries.
- Owned records are independent snapshots, not shared Proxy/session HTTP response cache entries. Repeated Get performs HTTP; zero-HTTP response-cache hits/invalidation are not claimed.
- Go preserves actual admitted/rejected read/Close/cancel/source/callback evidence, owns version/body/reserved protocol headers through native callbacks and rejects route/control overrides. Ordinary retry headers and live provider tokens retain native policy; Python arbitrary Adapter behavior is not reproduced.
- Original HTTP100..399 admission and physical400..599 rejections are retained. InvalidUTF8 and valid nonobject JSON are actual-response errors; accepted empty/malformed JSON preserves Source seed behavior.
- Go evaluates unordered bulk local filters in sorted key order and first-false short-circuits; Python retains kwargs insertion order. Recursive truthy-scalar empty-dict and nonempty mapping errors are preserved.
- Dictionary links are a Go compatibility extension. Source advertised offset and no-link marker fallback are retained through shared MarkerOrOffsetPagination; Go requires a single cursor family per request and strictly forward offsets across the stream, with cycle/origin/project/filter locks. Numeric pagination controls are validated rather than delegated to server acceptance.
- Only fixed declared mapping and Body attrs are supported. Source docs list_volume/search_opts/detailed kwargs are actually undeclared and discarded by this fixed resource; get list_volume has an explicit owned option.
- Local equality/truthiness preserves exact JSON decimals instead of Python binary64 decoding. 9007199254740992 != 9007199254740993.0 in Go (Source float rounding makes them equal); 9007199254740993 == 9007199254740993.0 in Go (Source int/float differs); 1e-400 stays truthy in Go (Source float underflows to0.0). Declared id raw local-filter regressions document all three; Wire/Envelope numbers remain unchanged.

### `python:block_storage/v3/update_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordUpdateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordUpdateMicroversion`

- Partial PUT groups/id sends known dirty attrs. Empty/unknown-only/ID-only update skips mutation and discovery and returns status0 local seed without synthetic HTTP proof; response known values override supplied fields.
  Tests: `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupUpdateDirtyNoopAndResponseMerge`.
- Bound classmethod cls input is a reserved control, including explicit null/false in individual Field/Query options and bulk raw Opts. It fails before discovery/mutation without synthetic HTTP proof.
  Tests: `blockstorage/v3/groups/python_records_test.go:TestPythonCinderGroupClassInputsAreBoundControlsBeforeDiscovery`.

Python/Go differences:

- Go accepts a literal nonblank UTF-8 ID and owned JSON-domain record/options rather than mutable Resource/Munch/subclasses. Connection-derived computed location is null in the standalone leaf; cross-call dirty state and dynamic session customization remain shared SDK-R1/C1/S1 boundaries.
- Owned records are independent snapshots, not shared Proxy/session HTTP response cache entries. Repeated Get performs HTTP; zero-HTTP response-cache hits/invalidation are not claimed.
- Go preserves actual admitted/rejected read/Close/cancel/source/callback evidence, owns version/body/reserved protocol headers through native callbacks and rejects route/control overrides. Ordinary retry headers and live provider tokens retain native policy; Python arbitrary Adapter behavior is not reproduced.
- Original HTTP100..399 admission and physical400..599 rejections are retained. InvalidUTF8 and valid nonobject JSON are actual-response errors; accepted empty/malformed JSON preserves Source seed behavior.
- Update accepts a literal ID with new known attrs, not a synchronized mutable existing Resource baseline. Repeated same attrs are sent again. Dirty body id is excluded and ID-only is no-op; Python string selector with duplicate id kwargs would raise dynamic TypeError, while mutable Resource commit discards dirty id.

### `python:block_storage/v3/find_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordFindDetails`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.WithRecordFindHeaders`

- Literal escaped member GET runs before list and returns successful Source untyped/null/numeric id/name without synthetic string validation; known descriptors and unknown raw proof are the existing Group record profile.
  Tests: `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordDirectUntypedProjectionAndLiteralIdentity`.
- Only a clean physical400/403/404 can fall back. The fixed Group mapping has no server name query; details defaults true with /groups/detail and false uses /groups. Every page is checked by exact string id OR name, including advertised forward offset continuation and ID-only matches.
  Tests: `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordCleanFallbackHasNoServerNameQuery`, `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordCleanRetryOriginalKeepsFallback`.
- IgnoreMissing defaults true and suppresses only logical absence after a successful complete list. Strict false returns ErrNotFound; actual list403/404/503 remains native status/body/header and is never suppressed.
  Tests: `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordMissingRequiresSuccessfulList`.
- A second matching row, including identical id, raises AmbiguousError with the actual accepted duplicate page. A later rejection/read/Close or constructor error discards the earlier candidate and preserves actual physical evidence.
  Tests: `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordLateDuplicatesKeepAcceptedPage`, `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordLateFailureDoesNotReturnFirstMatch`, `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordAcceptedErrorsAndListConstructorAreTerminal`.
- Rejected member read/Close400/403/404 cannot become clean fallback or invoke RetryFunc; accepted202 read errors containing nested400/403/404 likewise remain terminal actual202 response errors with original close/native causes and zero retry hooks.
  Tests: `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordRejectedReadCloseNeverFallsBack`, `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordAcceptedPhysicalCausesCannotBecomeFallback`.
- Bool/string pointers, per-call header maps and retained custom option pointers are owned snapshots for both phases. Nil context/API/option, blank identity, invalid version and reserved headers fail before discovery. Option errors, original source mutation and custom cancellation causes join rather than replace one another. Default/nil Headers become a fresh empty owned map for the first custom callback; repeated GET-to-LIST runs, retained callback mutation and an independent original-header snapshot preserve ownership.
  Tests: `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordOptionsOwnBothPhaseSnapshots`, `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordPreflightAndStickyOriginalSource`, `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordFirstCustomHeaderOptionOwnsDefaultAndNilMaps`.
- A single original cloudread capture guards both private per-phase copies. GET-only microversion override never propagates to fallback, which retains the original selected version. Empty override inherits original. Captured Accept applies to GET while Source LIST defaultapplication/json overrides it unless per-call Accept is supplied.
  Tests: `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordGetOverrideFallbackOriginalVersionAndHeaders`.
- Absent original version negotiates to at most3.38 and records actual discovery separately. A default GET reuses the original negotiation for fallback; an explicit GET override does not fabricate discovery and fallback negotiates the original default. Discovery403 is terminal and cannot be mistaken for member403 fallback.
  Tests: `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordDiscoveryEvidenceFollowsEachVersionPhase`, `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordDiscoveryDenialCannotBecomeMemberFallback`.
- A transient source endpoint mutation between actual Read and Close stays sticky despite restoration and preserves the accepted200 receipt.
  Tests: `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordAcceptedTransientSourceMutationKeepsReceipt`.
- Library-owned selected version values/all three version-header roles, reserved authentication/framing absence and owned OmitHeaders checks apply to native RetryFunc. Mutation stops before second HTTP with original503 evidence. Ordinary X-Retry and live provider token refresh remain supported; clean RetryFunc-return-original404 retains fallback.
  Tests: `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordNativeRetryOwnsVersionAndPreservesOrdinaryHeaders`, `blockstorage/v3/groups/python_find_test.go:TestPythonCinderGroupFindRecordCleanRetryOriginalKeepsFallback`.

Python/Go differences:

- The fixed proxy exposes name_or_id, ignore_missing and details only. Go additionally exposes typed GET-only inherited Resource.find microversion and ordinary per-call headers, without arbitrary kwargs/query/route/session controls.
- Go accepts a literal nonblank UTF-8 string and rejects dot segments/Unicode controls before HTTP. Details/IgnoreMissing are owned typed nullable bool options rather than Python dynamically truthy values. Python deprecation warnings are not emitted.
- Group records are independent JSON-domain snapshots; Source mutable Resource/Munch/subclasses, Connection-derived computed location, persistent dirty state and response-cache hits/invalidation remain shared SDK-R1/C1/S1 boundaries. location=null and every Find makes HTTP.
- Reuses the committed Group declared body/list descriptors, null defaults, exact wire numeric preservation and guarded MarkerOrOffsetPagination. Go uses strict origin/project/filter locks, forward single-family cursors and dictionary-link compatibility extensions.
- Terminal physical/source/cancel/callback faults are explicitly preserved and clean fallback is narrower than arbitrary Python transport customization. Owned version/reserved headers survive native callbacks; ordinary retry headers and live X-Auth-Token refresh retain native policy.
- The explicit empty GET override is a documented Go inherit-default policy. Pinned Source Resource.fetch forwards an explicitly empty microversion string; this Go option is not claimed as identical underlying fetch behavior.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.38")
    parser.add_argument("--action", choices=["list", "get", "create", "clone", "update", "delete"], default="list")
    parser.add_argument("--group", default="")
    parser.add_argument("--name", default="sample")
    parser.add_argument("--group-type", default="")
    parser.add_argument("--volume-types", default="")
    parser.add_argument("--source-group", default="")
    parser.add_argument("--source-snapshot", default="")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud,
        block_storage_default_microversion=args.microversion)
    api = conn.block_storage
    if args.action == "list":
        for value in api.groups():
            print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))
        return
    if args.action == "get":
        value = api.get_group(args.group, list_volume=True)
    elif args.action == "create":
        value = api.create_group(group_type=args.group_type,
            volume_types=args.volume_types.split(","), name=args.name)
    elif args.action == "clone":
        # Both IDs are required parameters of the pinned classmethod.
        value = api.create_group_from_source(
            group_snapshot_id=args.source_snapshot or None,
            source_group_id=args.source_group or None,
            name=args.name)
    elif args.action == "update":
        value = api.update_group(args.group, name=args.name)
    else:
        api.delete_group(args.group, delete_volumes=False, ignore_missing=False)
        return
    print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))


if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "strings"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printGroup(value *resource.RawResource) {
    raw, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(raw))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    version := flag.String("microversion", "3.38", "Cinder selected version")
    action := flag.String("action", "list", "list/get/create/clone/update/delete")
    id := flag.String("group", "", "group ID")
    name := flag.String("name", "sample", "name for create/clone/update")
    groupType := flag.String("group-type", "", "existing group-type name or ID")
    volumeTypes := flag.String("volume-types", "", "comma-separated volume-type IDs")
    sourceGroup := flag.String("source-group", "", "source group ID for clone")
    sourceSnapshot := flag.String("source-snapshot", "", "source group-snapshot ID for clone")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := groups.New(service.RawClient())
    var value *groups.GroupRecord
    switch *action {
    case "list":
        for row, err := range api.ListRecords(ctx) {
            if err != nil { log.Fatal(err) }
            printGroup(row.Resource)
        }
        return
    case "get":
        value, err = api.GetRecord(ctx, *id, groups.WithRecordGetListVolume(true))
    case "create":
        value, err = api.CreateRecord(ctx,
            groups.WithRecordCreateField("group_type", *groupType),
            groups.WithRecordCreateField("volume_types", strings.Split(*volumeTypes, ",")),
            groups.WithRecordCreateField("name", *name))
    case "clone":
        var snapshot, source any
        if *sourceSnapshot != "" { snapshot = *sourceSnapshot }
        if *sourceGroup != "" { source = *sourceGroup }
        value, err = api.CreateFromSourceRecord(ctx,
            groups.WithRecordSourceGroupSnapshotID(snapshot),
            groups.WithRecordSourceSourceGroupID(source),
            groups.WithRecordSourceName(*name))
    case "update":
        value, err = api.UpdateRecord(ctx, *id, groups.WithRecordUpdateField("name", *name))
    case "delete":
        err = api.DeleteRecord(ctx, *id, groups.WithRecordDeleteIgnoreMissing(false))
        if err != nil { log.Fatal(err) }
        return
    default:
        log.Fatal("unknown action")
    }
    if err != nil { log.Fatal(err) }
    printGroup(value.Resource)
}
```

## References

- [Group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2071-L2250)
- [Install](../../docs/install.md)
- [Group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py#L65)
- [Groups](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/groups.py)
- [group:delete](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/group_actions.py)
- [API](https://github.com/openstack/cinder/blob/27.0.0/cinder/group/api.py#L183-L200)
- [Python Records Test](groups/python_records_test.go)
- [Python Policy Test](groups/python_policy_test.go)
- [Python Parity](python-parity.md)
