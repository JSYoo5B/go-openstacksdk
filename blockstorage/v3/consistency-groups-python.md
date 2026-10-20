# Consistency Groups Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/consistency_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups.API.ListRecords`

- Lazy details defaulttrue, only limit/marker/offset/sort_dir/sort_key/sort declared server query; unknown/all_tenants discarded, declared Body local filters, raw row caps and guarded shared pager incl stock singular links as explicit Go improvement.
  Tests: `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupListDeclaredQueryLocalFiltersAndRawCap`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupListStockSingularLinksOffsetsMarkersAndBreak`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupListContinuationRejectsChangedScopeAndCursor`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupListDictionaryFiltersDefaultsAndShallowEquality`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupListConstructorAndShapesKeepActualPage`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupListExactDecimalPolicyIsExplicit`.
- Original selected/source/provider facts, owned options/headers/body/case-normalized request policy, uncapped selectedversions/no defaultdiscovery, fixed version/header absence and native retry ownership; original physical/cancellation/source causes and ErrUnableToReauthenticate public ErrOriginal/ErrReauth receipt are preserved.
  Tests: `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupPreflightControlsAndJoinedOptionCauses`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupEveryOptionFamilyStartsWithOwnedHeaders`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupPhysicalReceiptsStopHooksAndRetainAllCauses`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupNativeRetryOwnsPayloadVersionAndFraming`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupClean401AndOrdinary503RetainOwnedPOST`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupSourceAdmissionAndSelectedAliasesRemainUncapped`.
- Page next truthiness uses existing cloudfilter.PythonJSONTruthy: Source binary64 underflow±1e-400 falsey ignored, nonzero1e-300 and overflow±1e400 truthy nonstring cause actual203 page error with no second listHTTP; local filter exact decimals remain explicit separate policy. LIST-only reserved helper rejects Source/Proxy controls by raw presence before Marshaler/HTTP for null/false/zero bulk/callback/single options, preserving typed controls and unknown case-sensitive discard.
  Tests: `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupListNextUsesPythonBinary64TruthinessAndActualPage`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupListReservedControlsFailBeforeMarshalAndHTTP`.

Python/Go differences:

- Owned standalone stateless records do not reproduce persistent Python Resource/Munch/subclass/Connection dirty state or session cache; location is null. Repeated Get/Find performs HTTP. Source warnings/dynamic TypeError/arbitrary objects are explicit shared boundaries, not advertised lifecycle parity.
- Go identities and numeric cursor controls are stricter; origin/project/collection/filter lock, cursor progress and conflicting continuation validation intentionally reject unsafe or ambiguous continuation mutations. Dictionary links are an explicit Go compatibility extension.
- Local filter equality/truthiness preserves exact decimals instead of Python binary64 rounding/underflow; deterministic sorted filters short circuit first false. Actual Wire/Envelope numbers remain unchanged.
- Per-call Headers/Microversion are explicit owned Go policy extensions. Find microversion is GET-only; empty Find override inherits original selected policy rather than forwarding Source explicit empty string. No class ceiling/default discovery is introduced.
- Stock singular consistencygroup_links is supported as explicit Go pagination improvement. Pinned Resource._get_next_link directly reads generic links/plural links/next/HTTP Link and does not directly read the stock singular key; limit-based Source marker fallback is distinct.
- Raw generic LIST kwargs for named controls are rejected with ErrInvalidOption and must use concrete typed options. No allow_unknown_params override is exposed; deprecated JMESPath/conflicting dynamic kwargs remain SDK-J1 rather than silent discard/full parity. Mutation attr semantics unchanged by list-only helper.

### `python:block_storage/v3/create_consistency_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups.API.CreateRecord`

- Known partial attrs with absent/null/zero/false/empty distinctions, owned map/pointer snapshots; raw comma volume_types remains request string, descriptor List conversion returned view only.
  Tests: `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupCreateRawPartialNullAndSnapshotOwnership`.
- Original selected/source/provider facts, owned options/headers/body/case-normalized request policy, uncapped selectedversions/no defaultdiscovery, fixed version/header absence and native retry ownership; original physical/cancellation/source causes and ErrUnableToReauthenticate public ErrOriginal/ErrReauth receipt are preserved.
  Tests: `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupPreflightControlsAndJoinedOptionCauses`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupEveryOptionFamilyStartsWithOwnedHeaders`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupPhysicalReceiptsStopHooksAndRetainAllCauses`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupNativeRetryOwnsPayloadVersionAndFraming`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupClean401AndOrdinary503RetainOwnedPOST`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupSourceAdmissionAndSelectedAliasesRemainUncapped`.

Python/Go differences:

- Owned standalone stateless records do not reproduce persistent Python Resource/Munch/subclass/Connection dirty state or session cache; location is null. Repeated Get/Find performs HTTP. Source warnings/dynamic TypeError/arbitrary objects are explicit shared boundaries, not advertised lifecycle parity.
- Go identities and numeric cursor controls are stricter; origin/project/collection/filter lock, cursor progress and conflicting continuation validation intentionally reject unsafe or ambiguous continuation mutations. Dictionary links are an explicit Go compatibility extension.
- Local filter equality/truthiness preserves exact decimals instead of Python binary64 rounding/underflow; deterministic sorted filters short circuit first false. Actual Wire/Envelope numbers remain unchanged.
- Per-call Headers/Microversion are explicit owned Go policy extensions. Find microversion is GET-only; empty Find override inherits original selected policy rather than forwarding Source explicit empty string. No class ceiling/default discovery is introduced.

### `python:block_storage/v3/create_consistency_group_from_source`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups.API.CreateFromSourceRecord`

- POST consistencygroups/create_from_src with consistencygroup-from-src, four null defaults, falsey empty IDs→null, literal optional sourceIDs and name/description empty preserved; no prerequisite lookup/request-seed, exactly-one-source remains actual server400 validation.
  Tests: `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupCreateFromSourceNullDefaultsAndLiteralReferences`.
- Original selected/source/provider facts, owned options/headers/body/case-normalized request policy, uncapped selectedversions/no defaultdiscovery, fixed version/header absence and native retry ownership; original physical/cancellation/source causes and ErrUnableToReauthenticate public ErrOriginal/ErrReauth receipt are preserved.
  Tests: `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupPreflightControlsAndJoinedOptionCauses`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupEveryOptionFamilyStartsWithOwnedHeaders`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupPhysicalReceiptsStopHooksAndRetainAllCauses`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupNativeRetryOwnsPayloadVersionAndFraming`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupClean401AndOrdinary503RetainOwnedPOST`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupSourceAdmissionAndSelectedAliasesRemainUncapped`.

Python/Go differences:

- Owned standalone stateless records do not reproduce persistent Python Resource/Munch/subclass/Connection dirty state or session cache; location is null. Repeated Get/Find performs HTTP. Source warnings/dynamic TypeError/arbitrary objects are explicit shared boundaries, not advertised lifecycle parity.
- Go identities and numeric cursor controls are stricter; origin/project/collection/filter lock, cursor progress and conflicting continuation validation intentionally reject unsafe or ambiguous continuation mutations. Dictionary links are an explicit Go compatibility extension.
- Local filter equality/truthiness preserves exact decimals instead of Python binary64 rounding/underflow; deterministic sorted filters short circuit first false. Actual Wire/Envelope numbers remain unchanged.
- Per-call Headers/Microversion are explicit owned Go policy extensions. Find microversion is GET-only; empty Find override inherits original selected policy rather than forwarding Source explicit empty string. No class ceiling/default discovery is introduced.
- Optional sourceID/name/description inputs are nullable Go strings; arbitrary Source dynamic objects/Resource reference overloads are not public Go inputs. Empty source IDs map to Source proxy falsey null. Request emits both null IDs when omitted so server can actually reject, rather than adding unrequested preflight exclusivity.

### `python:block_storage/v3/delete_consistency_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups.API.DeleteRecord`

- POST literal /{id}/delete forcefalse by default; corrected stock consistencygroup key, explicit PinnedSourceWire reproduces pinned Source consistency_group defect with actual400; no ignore_missing/default404 suppression; accepted body ignored.
  Tests: `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupDeleteCorrectedBodyReplayAndStrictMissing`.
- Original selected/source/provider facts, owned options/headers/body/case-normalized request policy, uncapped selectedversions/no defaultdiscovery, fixed version/header absence and native retry ownership; original physical/cancellation/source causes and ErrUnableToReauthenticate public ErrOriginal/ErrReauth receipt are preserved.
  Tests: `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupPreflightControlsAndJoinedOptionCauses`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupEveryOptionFamilyStartsWithOwnedHeaders`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupPhysicalReceiptsStopHooksAndRetainAllCauses`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupNativeRetryOwnsPayloadVersionAndFraming`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupClean401AndOrdinary503RetainOwnedPOST`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupSourceAdmissionAndSelectedAliasesRemainUncapped`.

Python/Go differences:

- Owned standalone stateless records do not reproduce persistent Python Resource/Munch/subclass/Connection dirty state or session cache; location is null. Repeated Get/Find performs HTTP. Source warnings/dynamic TypeError/arbitrary objects are explicit shared boundaries, not advertised lifecycle parity.
- Go identities and numeric cursor controls are stricter; origin/project/collection/filter lock, cursor progress and conflicting continuation validation intentionally reject unsafe or ambiguous continuation mutations. Dictionary links are an explicit Go compatibility extension.
- Local filter equality/truthiness preserves exact decimals instead of Python binary64 rounding/underflow; deterministic sorted filters short circuit first false. Actual Wire/Envelope numbers remain unchanged.
- Per-call Headers/Microversion are explicit owned Go policy extensions. Find microversion is GET-only; empty Find override inherits original selected policy rather than forwarding Source explicit empty string. No class ceiling/default discovery is introduced.
- Tracked Source bug: fixed Python delete sends consistency_group but stock controller requires consistencygroup. Go correcteddefault plus explicit PinnedSourceWire replay; literal Source body is never falsely claimed accepted.

### `python:block_storage/v3/find_consistency_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups.API.FindRecord`

- GET-first clean400/403/404 fallback to full detailed/summary list without server name query; exact string ID-or-name, sameID/late duplicates, complete stream missing, actual duplicate-page receipt and terminal physical/accepted/late errors.
  Tests: `blockstorage/v3/consistencygroups/python_find_contracts_test.go:TestPythonCinderConsistencyGroupFindRecordCleanFallbackHasNoServerNameQuery`, `blockstorage/v3/consistencygroups/python_find_contracts_test.go:TestPythonCinderConsistencyGroupFindRecordLateDuplicatesKeepAcceptedPage`, `blockstorage/v3/consistencygroups/python_find_contracts_test.go:TestPythonCinderConsistencyGroupFindRecordLateFailureDoesNotReturnFirstMatch`, `blockstorage/v3/consistencygroups/python_find_contracts_test.go:TestPythonCinderConsistencyGroupFindRecordRejectedReadCloseNeverFallsBack`, `blockstorage/v3/consistencygroups/python_find_contracts_test.go:TestPythonCinderConsistencyGroupFindRecordGetOverrideFallbackOriginalVersionAndHeaders`.
- Original selected/source/provider facts, owned options/headers/body/case-normalized request policy, uncapped selectedversions/no defaultdiscovery, fixed version/header absence and native retry ownership; original physical/cancellation/source causes and ErrUnableToReauthenticate public ErrOriginal/ErrReauth receipt are preserved.
  Tests: `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupPreflightControlsAndJoinedOptionCauses`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupEveryOptionFamilyStartsWithOwnedHeaders`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupPhysicalReceiptsStopHooksAndRetainAllCauses`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupNativeRetryOwnsPayloadVersionAndFraming`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupClean401AndOrdinary503RetainOwnedPOST`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupSourceAdmissionAndSelectedAliasesRemainUncapped`.
- Fallback LIST uses Source binary64 next truthiness. Falsey underflow preserves actual matching203 result after member404; truthy nonzero/overflow nonstring next yields actual203 ResponseError and no successful prior candidate or extra HTTP.
  Tests: `blockstorage/v3/consistencygroups/python_find_contracts_test.go:TestPythonCinderConsistencyGroupFindNextUsesPythonBinary64TruthinessAndActualPage`.

Python/Go differences:

- Owned standalone stateless records do not reproduce persistent Python Resource/Munch/subclass/Connection dirty state or session cache; location is null. Repeated Get/Find performs HTTP. Source warnings/dynamic TypeError/arbitrary objects are explicit shared boundaries, not advertised lifecycle parity.
- Go identities and numeric cursor controls are stricter; origin/project/collection/filter lock, cursor progress and conflicting continuation validation intentionally reject unsafe or ambiguous continuation mutations. Dictionary links are an explicit Go compatibility extension.
- Local filter equality/truthiness preserves exact decimals instead of Python binary64 rounding/underflow; deterministic sorted filters short circuit first false. Actual Wire/Envelope numbers remain unchanged.
- Per-call Headers/Microversion are explicit owned Go policy extensions. Find microversion is GET-only; empty Find override inherits original selected policy rather than forwarding Source explicit empty string. No class ceiling/default discovery is introduced.

### `python:block_storage/v3/get_consistency_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups.API.GetRecord`

- Literal member GET without query or default discovery; declared9 Body/default-null/List view, original wire/unknown/big numbers, independent actual receipts and accepted decode/partial errors.
  Tests: `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupGetLiteralProjectionDefaultVersionAndNoCache`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupAcceptedDecoderKeepsActualEnvelopeAndSeeds`.
- Original selected/source/provider facts, owned options/headers/body/case-normalized request policy, uncapped selectedversions/no defaultdiscovery, fixed version/header absence and native retry ownership; original physical/cancellation/source causes and ErrUnableToReauthenticate public ErrOriginal/ErrReauth receipt are preserved.
  Tests: `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupPreflightControlsAndJoinedOptionCauses`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupEveryOptionFamilyStartsWithOwnedHeaders`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupPhysicalReceiptsStopHooksAndRetainAllCauses`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupNativeRetryOwnsPayloadVersionAndFraming`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupClean401AndOrdinary503RetainOwnedPOST`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupSourceAdmissionAndSelectedAliasesRemainUncapped`.

Python/Go differences:

- Owned standalone stateless records do not reproduce persistent Python Resource/Munch/subclass/Connection dirty state or session cache; location is null. Repeated Get/Find performs HTTP. Source warnings/dynamic TypeError/arbitrary objects are explicit shared boundaries, not advertised lifecycle parity.
- Go identities and numeric cursor controls are stricter; origin/project/collection/filter lock, cursor progress and conflicting continuation validation intentionally reject unsafe or ambiguous continuation mutations. Dictionary links are an explicit Go compatibility extension.
- Local filter equality/truthiness preserves exact decimals instead of Python binary64 rounding/underflow; deterministic sorted filters short circuit first false. Actual Wire/Envelope numbers remain unchanged.
- Per-call Headers/Microversion are explicit owned Go policy extensions. Find microversion is GET-only; empty Find override inherits original selected policy rather than forwarding Source explicit empty string. No class ceiling/default discovery is introduced.

### `python:block_storage/v3/update_consistency_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups.API.UpdateRecord`

- PUT literal member with known dirty attrs, discarded dirty id; empty/unknown-only/id-only returns local seed without HTTP; stock3.6 blank support is selected policy/server rejection, not invented automatic discovery/support gate.
  Tests: `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupUpdateDirtyNoopBlankVersionAndResponseMerge`.
- Original selected/source/provider facts, owned options/headers/body/case-normalized request policy, uncapped selectedversions/no defaultdiscovery, fixed version/header absence and native retry ownership; original physical/cancellation/source causes and ErrUnableToReauthenticate public ErrOriginal/ErrReauth receipt are preserved.
  Tests: `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupPreflightControlsAndJoinedOptionCauses`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupEveryOptionFamilyStartsWithOwnedHeaders`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupPhysicalReceiptsStopHooksAndRetainAllCauses`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupNativeRetryOwnsPayloadVersionAndFraming`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupClean401AndOrdinary503RetainOwnedPOST`, `blockstorage/v3/consistencygroups/python_record_contracts_test.go:TestPythonCinderConsistencyGroupSourceAdmissionAndSelectedAliasesRemainUncapped`.

Python/Go differences:

- Owned standalone stateless records do not reproduce persistent Python Resource/Munch/subclass/Connection dirty state or session cache; location is null. Repeated Get/Find performs HTTP. Source warnings/dynamic TypeError/arbitrary objects are explicit shared boundaries, not advertised lifecycle parity.
- Go identities and numeric cursor controls are stricter; origin/project/collection/filter lock, cursor progress and conflicting continuation validation intentionally reject unsafe or ambiguous continuation mutations. Dictionary links are an explicit Go compatibility extension.
- Local filter equality/truthiness preserves exact decimals instead of Python binary64 rounding/underflow; deterministic sorted filters short circuit first false. Actual Wire/Envelope numbers remain unchanged.
- Per-call Headers/Microversion are explicit owned Go policy extensions. Find microversion is GET-only; empty Find override inherits original selected policy rather than forwarding Source explicit empty string. No class ceiling/default discovery is introduced.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default=None)
    parser.add_argument("--action", choices=["list", "get", "find", "create", "clone", "update", "delete"], default="list")
    parser.add_argument("--group", default="")
    parser.add_argument("--name", default="sample")
    parser.add_argument("--volume-types", default="")
    parser.add_argument("--source-group", default="")
    parser.add_argument("--source-snapshot", default="")
    parser.add_argument("--summary", action="store_true")
    parser.add_argument("--strict", action="store_true")
    parser.add_argument("--force", action="store_true")
    args = parser.parse_args()
    policy = {}
    if args.microversion is not None:
        policy["block_storage_default_microversion"] = args.microversion
    api = openstack.connect(cloud=args.cloud, **policy).block_storage
    if args.action == "list":
        for value in api.consistency_groups(details=not args.summary):
            print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))
        return
    if args.action == "get":
        value = api.get_consistency_group(args.group)
    elif args.action == "find":
        value = api.find_consistency_group(args.group,
            ignore_missing=not args.strict, details=not args.summary)
    elif args.action == "create":
        value = api.create_consistency_group(volume_types=args.volume_types,
            name=args.name)
    elif args.action == "clone":
        value = api.create_consistency_group_from_source(
            consistency_group=args.source_group or None,
            consistency_group_snapshot=args.source_snapshot or None,
            name=args.name)
    elif args.action == "update":
        value = api.update_consistency_group(args.group, name=args.name)
    else:
        # This exact pinned helper sends consistency_group rather than the
        # stock controller's consistencygroup and can receive HTTP400.
        api.delete_consistency_group(args.group, force=args.force)
        return
    print(json.dumps(None if value is None else value.to_dict(),
                     indent=2, ensure_ascii=False))


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
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/consistencygroups"
)

func show(value *consistencygroups.ConsistencyGroupRecord) {
    if value == nil { fmt.Println("null"); return }
    raw, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(raw))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    version := flag.String("microversion", "", "optional selected Cinder version")
    action := flag.String("action", "list", "list/get/find/create/clone/update/delete")
    id := flag.String("group", "", "literal ID, or exact identity for find")
    name := flag.String("name", "sample", "name for create/clone/update")
    volumeTypes := flag.String("volume-types", "", "comma-separated legacy volume types")
    sourceGroup := flag.String("source-group", "", "source consistency-group ID")
    sourceSnapshot := flag.String("source-snapshot", "", "source consistency-group-snapshot ID")
    summary := flag.Bool("summary", false, "summary list/find")
    strict := flag.Bool("strict", false, "strict find missing")
    force := flag.Bool("force", false, "also request deletion of member volumes")
    sourceWire := flag.Bool("pinned-source-wire", false, "replay known Source delete spelling")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    policy := []sdk.ConnectionOption{sdk.WithCloud(*cloud)}
    if *version != "" { policy = append(policy, sdk.WithMicroversion(sdk.BlockStorage, *version)) }
    conn, err := sdk.Connect(ctx, policy...)
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := consistencygroups.New(service.RawClient())
    var value *consistencygroups.ConsistencyGroupRecord
    switch *action {
    case "list":
        for row, err := range api.ListRecords(ctx, consistencygroups.WithRecordListDetails(!*summary)) {
            if err != nil { log.Fatal(err) }
            show(row)
        }
        return
    case "get":
        value, err = api.GetRecord(ctx, *id)
    case "find":
        value, err = api.FindRecord(ctx, *id,
            consistencygroups.WithRecordFindIgnoreMissing(!*strict),
            consistencygroups.WithRecordFindDetails(!*summary))
    case "create":
        value, err = api.CreateRecord(ctx,
            consistencygroups.WithRecordCreateField("volume_types", *volumeTypes),
            consistencygroups.WithRecordCreateField("name", *name))
    case "clone":
        value, err = api.CreateFromSourceRecord(ctx,
            consistencygroups.WithRecordSourceConsistencyGroupID(*sourceGroup),
            consistencygroups.WithRecordSourceConsistencyGroupSnapshotID(*sourceSnapshot),
            consistencygroups.WithRecordSourceName(*name))
    case "update":
        value, err = api.UpdateRecord(ctx, *id, consistencygroups.WithRecordUpdateField("name", *name))
    case "delete":
        err = api.DeleteRecord(ctx, *id, consistencygroups.WithRecordDeleteForce(*force),
            consistencygroups.WithRecordDeletePinnedSourceWire(*sourceWire))
        if err != nil { log.Fatal(err) }
        return
    default:
        log.Fatal("unknown action")
    }
    if err != nil { log.Fatal(err) }
    show(value)
}
```

## References

- [ConsistencyGroup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/consistency_group.py)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2322-L2504)
- [Groups Python](groups-python.md)
- [Config](https://github.com/openstack/cinder/blob/27.0.0/cinder/common/config.py#L59-L61)
- [v3 router](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/v3/router.py#L165-L171)
- [Install](../../docs/install.md)
- [stock v3 update](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/v3/consistencygroups.py#L34-L100)
- [stock controller](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/contrib/consistencygroups.py#L61-L76)
- [view builder](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/views/consistencygroups.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1418-L1445)
- [group policy](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/groups.py)
- [delete policy](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/group_actions.py#L58-L68)
- [Python Record Contracts Test](consistencygroups/python_record_contracts_test.go)
- [Python Find Contracts Test](consistencygroups/python_find_contracts_test.go)
- [Python Parity](python-parity.md)
