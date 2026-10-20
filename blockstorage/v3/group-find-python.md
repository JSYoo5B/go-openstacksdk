# Group Find Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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
    parser.add_argument("identity", help="group ID or exact name")
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--microversion", default="3.38")
    parser.add_argument("--strict", action="store_true")
    parser.add_argument("--summary", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(
        cloud=args.cloud,
        block_storage_default_microversion=args.microversion,
    )
    value = conn.block_storage.find_group(
        args.identity,
        ignore_missing=not args.strict,
        details=not args.summary,
    )
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
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    identity := flag.String("identity", "", "group ID or exact name")
    version := flag.String("microversion", "3.38", "selected Cinder default")
    strict := flag.Bool("strict", false, "return an error when not found")
    summary := flag.Bool("summary", false, "use summary fallback list")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.BlockStorage, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    value, err := service.Groups.FindRecord(ctx, *identity,
        groups.WithRecordFindIgnoreMissing(!*strict),
        groups.WithRecordFindDetails(!*summary))
    if err != nil { log.Fatal(err) }
    if value == nil { fmt.Println("null"); return }
    raw, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(raw))
}
```

```go
value, err := service.Groups.FindRecord(ctx, "sample",
    groups.WithRecordFindMicroversion("3.90"),
    groups.WithRecordFindHeaders(map[string]string{"X-Trace": "group-find"}))
```

## References

- [find_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2126-L2155)
- [Group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py#L26-L66)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2489-L2582)
- [Groups Python](groups-python.md)
- [Install](../../docs/install.md)
- [Groups](https://github.com/openstack/cinder/blob/27.0.0/cinder/policies/groups.py)
