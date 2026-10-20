# Share Attachments Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/create_share_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/shareattachments.API.CreateRecord`

- Caller IDs fix both URI segments and never derive routes from returned uuid/share_id; Capture and owned config precede callbacks/discovery, request Body/header/microversion remains fixed through callbacks and clean authentication renewal. Invalid inputs and observed source changes terminate before discovery or later options.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareGetFixesParentMemberAndProxySeed`, `compute/v2/shareattachments/contracts_test.go:TestShareInvalidOptionsAndIdentityPrecedeDiscovery`, `compute/v2/shareattachments/contracts_test.go:TestShareCleanAuthenticationRenewalRetainsRequest`, `compute/v2/shareattachments/contracts_test.go:TestShareOwnedConfigRepairsNilMapsAndCopiesPolicyAroundCallbacks`.
- Selected microversion is literal and uncapped; missing selected version discovers the advertised range with Source ceiling2.97. Explicit empty override owns required version header absence without mutating shared client. Discovery receipt is separate from member response evidence.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareVersionDiscoveryCeilingAndExplicitSelection`, `compute/v2/shareattachments/contracts_test.go:TestShareExplicitEmptyOverrideOwnsVersionAbsence`.
- Actual accepted/rejected status, body, header and all physical Read/Close causes survive. Four operations by eight statuses by three fault modes prove one HTTP/Close and zero retry, backoff or reauthentication callbacks after physical faults; native401 pointer ErrOriginal/ErrReauth preserved, while clean401 renewal keeps exact route/body/version.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestSharePhysicalFaultsKeepEvidenceAndStopAllHooks`, `compute/v2/shareattachments/contracts_test.go:TestShareCleanAuthenticationRenewalRetainsRequest`.
- POST uses share envelope containing explicit share_id and only supplied declared dirty attributes; omitted default export_location/tag stays absent in request, explicitnull and arbitrary untyped values survive. Source view merges accepted response fields over seed and retains defaults; actual Wire preserves unknown/self/exact numbers independently.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareCreateDirtyBodyAndSeededDescriptors`, `compute/v2/shareattachments/contracts_test.go:TestShareMemberNonJSONRetainsSeedWithoutFakeWire`.

Python/Go differences:

- Go concrete IDs use server.ID/share.ID instead of Python arbitrary Resource inputs. Literal UTF8 identity, fixed route/selected header guards and physical handling-error termination improve safety; this is not arbitrary session/base_path or Resource lifecycle parity.
- Source has six untyped declared Body descriptors plus inherited id/name. Go returns independently owned Resource/Wire/Envelope/header/status, preserves seed/default/null and alternate ID presence, and supplies explicit per-call microversion options. The shared SDK-R1/C1 Connection location and optional HTTP cache contracts are separate; direct client location is null.
- Official Nova31.0.0 stock policy is project reader on index/show and project member on create/delete, checked against instance project. Endpoints need2.97, request accepts share_id and optional tag, tag default belongs to Nova; asynchronous mapping receipt does not prove mount acceptance or complete Manila access workflow.

### `python:compute/v2/get_share_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/shareattachments.API.GetRecord`

- Caller IDs fix both URI segments and never derive routes from returned uuid/share_id; Capture and owned config precede callbacks/discovery, request Body/header/microversion remains fixed through callbacks and clean authentication renewal. Invalid inputs and observed source changes terminate before discovery or later options.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareGetFixesParentMemberAndProxySeed`, `compute/v2/shareattachments/contracts_test.go:TestShareInvalidOptionsAndIdentityPrecedeDiscovery`, `compute/v2/shareattachments/contracts_test.go:TestShareCleanAuthenticationRenewalRetainsRequest`, `compute/v2/shareattachments/contracts_test.go:TestShareOwnedConfigRepairsNilMapsAndCopiesPolicyAroundCallbacks`.
- Selected microversion is literal and uncapped; missing selected version discovers the advertised range with Source ceiling2.97. Explicit empty override owns required version header absence without mutating shared client. Discovery receipt is separate from member response evidence.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareVersionDiscoveryCeilingAndExplicitSelection`, `compute/v2/shareattachments/contracts_test.go:TestShareExplicitEmptyOverrideOwnsVersionAbsence`.
- Actual accepted/rejected status, body, header and all physical Read/Close causes survive. Four operations by eight statuses by three fault modes prove one HTTP/Close and zero retry, backoff or reauthentication callbacks after physical faults; native401 pointer ErrOriginal/ErrReauth preserved, while clean401 renewal keeps exact route/body/version.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestSharePhysicalFaultsKeepEvidenceAndStopAllHooks`, `compute/v2/shareattachments/contracts_test.go:TestShareCleanAuthenticationRenewalRetainsRequest`.
- GET seeds Proxy member id, independently retains caller server URI and actual returned share_id/uuid. Untyped descriptors preserve null/value; missing export_location default empty and inherited fields null; nonJSON accepted body keeps seed with no fabricated Wire while parsed wrongshape returns actual receipt.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareGetFixesParentMemberAndProxySeed`, `compute/v2/shareattachments/contracts_test.go:TestShareMemberNonJSONRetainsSeedWithoutFakeWire`, `compute/v2/shareattachments/contracts_test.go:TestShareMemberParsedInvalidShapeKeepsActualReceipt`.

Python/Go differences:

- Go concrete IDs use server.ID/share.ID instead of Python arbitrary Resource inputs. Literal UTF8 identity, fixed route/selected header guards and physical handling-error termination improve safety; this is not arbitrary session/base_path or Resource lifecycle parity.
- Source has six untyped declared Body descriptors plus inherited id/name. Go returns independently owned Resource/Wire/Envelope/header/status, preserves seed/default/null and alternate ID presence, and supplies explicit per-call microversion options. The shared SDK-R1/C1 Connection location and optional HTTP cache contracts are separate; direct client location is null.
- Official Nova31.0.0 stock policy is project reader on index/show and project member on create/delete, checked against instance project. Endpoints need2.97, request accepts share_id and optional tag, tag default belongs to Nova; asynchronous mapping receipt does not prove mount acceptance or complete Manila access workflow.

### `python:compute/v2/share_attachments`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/shareattachments.API.ListRecords`

- Caller IDs fix both URI segments and never derive routes from returned uuid/share_id; Capture and owned config precede callbacks/discovery, request Body/header/microversion remains fixed through callbacks and clean authentication renewal. Invalid inputs and observed source changes terminate before discovery or later options.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareGetFixesParentMemberAndProxySeed`, `compute/v2/shareattachments/contracts_test.go:TestShareInvalidOptionsAndIdentityPrecedeDiscovery`, `compute/v2/shareattachments/contracts_test.go:TestShareCleanAuthenticationRenewalRetainsRequest`, `compute/v2/shareattachments/contracts_test.go:TestShareOwnedConfigRepairsNilMapsAndCopiesPolicyAroundCallbacks`.
- Selected microversion is literal and uncapped; missing selected version discovers the advertised range with Source ceiling2.97. Explicit empty override owns required version header absence without mutating shared client. Discovery receipt is separate from member response evidence.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareVersionDiscoveryCeilingAndExplicitSelection`, `compute/v2/shareattachments/contracts_test.go:TestShareExplicitEmptyOverrideOwnsVersionAbsence`.
- Actual accepted/rejected status, body, header and all physical Read/Close causes survive. Four operations by eight statuses by three fault modes prove one HTTP/Close and zero retry, backoff or reauthentication callbacks after physical faults; native401 pointer ErrOriginal/ErrReauth preserved, while clean401 renewal keeps exact route/body/version.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestSharePhysicalFaultsKeepEvidenceAndStopAllHooks`, `compute/v2/shareattachments/contracts_test.go:TestShareCleanAuthenticationRenewalRetainsRequest`.
- List is lazy; canonical Body filters apply locally after Source raw row cap and do not become wire query. MaxItems implies Source limit hint, cap prevents nextlink lookahead, marker fallback selects id by presence before alternate share_id; nullprimary marker is terminal with page receipt. Ordinary unknown kwargs are discarded before serialization and reserved Source controls explicitly rejected before caller marshaller. Local equality uses CPython-loaded float and arbitraryprecision integer/Boolean semantics; dictionary conditions use Source actualtruthiness and recursive subset, including missing nested None and truthy nonmapping empty-filter behavior.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareListProjectionLocalCapAndLazyNoLookahead`, `compute/v2/shareattachments/contracts_test.go:TestShareListAlternateMarkerPresenceAndContinuation`, `compute/v2/shareattachments/contracts_test.go:TestShareListNullPrimaryMarkerIsTerminalWithPageReceipt`, `compute/v2/shareattachments/contracts_test.go:TestShareListUnknownDiscardAndControlRejectBeforeMarshaler`, `compute/v2/shareattachments/contracts_test.go:TestShareListActualLinksUseFixedRouteAndSinglePageStopsBeforeLink`, `compute/v2/shareattachments/contracts_test.go:TestShareListForeignContinuationFailsWithActualPage`, `compute/v2/shareattachments/contracts_test.go:TestShareSourceLoadedLocalEqualityAndDictionaryTruthiness`.

Python/Go differences:

- Go concrete IDs use server.ID/share.ID instead of Python arbitrary Resource inputs. Literal UTF8 identity, fixed route/selected header guards and physical handling-error termination improve safety; this is not arbitrary session/base_path or Resource lifecycle parity.
- Source has six untyped declared Body descriptors plus inherited id/name. Go returns independently owned Resource/Wire/Envelope/header/status, preserves seed/default/null and alternate ID presence, and supplies explicit per-call microversion options. The shared SDK-R1/C1 Connection location and optional HTTP cache contracts are separate; direct client location is null.
- Official Nova31.0.0 stock policy is project reader on index/show and project member on create/delete, checked against instance project. Endpoints need2.97, request accepts share_id and optional tag, tag default belongs to Nova; asynchronous mapping receipt does not prove mount acceptance or complete Manila access workflow.
- Source inherits limit/marker and maxItems limit hint but stock Nova31.0.0 index schema accepts no query properties; these options may be server-rejected, default no-query listing works. Go primitive limit/marker types and same-origin/path/cursor guards are stricter than arbitrary Source kwargs.
- Arbitrary JMESPath and session/routing/constructor controls explicitly fail before HTTP; shared SDK-S1 stays separate. Existing Source REST link, marker, raw cap and local filter policies are reused rather than labeling transport wrappers full SDK parity.
- Source marker fallback can place null/false/empty primary id into query params. Go preserves primary presence precedence but requires nonblank string cursor and returns the actual page receipt on failure; it never silently substitutes alternate share_id. TestShareListNullPrimaryMarkerIsTerminalWithPageReceipt proves this deliberate strict continuation difference.

### `python:compute/v2/delete_share_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/shareattachments.API.DeleteRecord`

- Caller IDs fix both URI segments and never derive routes from returned uuid/share_id; Capture and owned config precede callbacks/discovery, request Body/header/microversion remains fixed through callbacks and clean authentication renewal. Invalid inputs and observed source changes terminate before discovery or later options.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareGetFixesParentMemberAndProxySeed`, `compute/v2/shareattachments/contracts_test.go:TestShareInvalidOptionsAndIdentityPrecedeDiscovery`, `compute/v2/shareattachments/contracts_test.go:TestShareCleanAuthenticationRenewalRetainsRequest`, `compute/v2/shareattachments/contracts_test.go:TestShareOwnedConfigRepairsNilMapsAndCopiesPolicyAroundCallbacks`.
- Selected microversion is literal and uncapped; missing selected version discovers the advertised range with Source ceiling2.97. Explicit empty override owns required version header absence without mutating shared client. Discovery receipt is separate from member response evidence.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareVersionDiscoveryCeilingAndExplicitSelection`, `compute/v2/shareattachments/contracts_test.go:TestShareExplicitEmptyOverrideOwnsVersionAbsence`.
- Actual accepted/rejected status, body, header and all physical Read/Close causes survive. Four operations by eight statuses by three fault modes prove one HTTP/Close and zero retry, backoff or reauthentication callbacks after physical faults; native401 pointer ErrOriginal/ErrReauth preserved, while clean401 renewal keeps exact route/body/version.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestSharePhysicalFaultsKeepEvidenceAndStopAllHooks`, `compute/v2/shareattachments/contracts_test.go:TestShareCleanAuthenticationRenewalRetainsRequest`.
- DELETE has no Body, uses server/share member route and preserves literal accepted receipt without Resource fabrication. IgnoreMissing defaults true and suppresses only clean404; strict404/conflict/physical faults return original causal errors.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareDeleteDefaultOnlyCleanMissingAndBodylessSuccess`, `compute/v2/shareattachments/contracts_test.go:TestSharePhysicalFaultsKeepEvidenceAndStopAllHooks`.

Python/Go differences:

- Go concrete IDs use server.ID/share.ID instead of Python arbitrary Resource inputs. Literal UTF8 identity, fixed route/selected header guards and physical handling-error termination improve safety; this is not arbitrary session/base_path or Resource lifecycle parity.
- Source has six untyped declared Body descriptors plus inherited id/name. Go returns independently owned Resource/Wire/Envelope/header/status, preserves seed/default/null and alternate ID presence, and supplies explicit per-call microversion options. The shared SDK-R1/C1 Connection location and optional HTTP cache contracts are separate; direct client location is null.
- Official Nova31.0.0 stock policy is project reader on index/show and project member on create/delete, checked against instance project. Endpoints need2.97, request accepts share_id and optional tag, tag default belongs to Nova; asynchronous mapping receipt does not prove mount acceptance or complete Manila access workflow.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--server", required=True)
parser.add_argument("--share", default="")
parser.add_argument("--mode", choices=("list", "get", "create", "delete"), default="list")
parser.add_argument("--tag", default=None)
args = parser.parse_args()

with openstack.connect(cloud=args.cloud, compute_default_microversion="2.97") as conn:
    if args.mode == "list":
        for value in conn.compute.share_attachments(args.server):
            print(json.dumps(value.to_dict(), indent=2))
    elif args.mode == "get":
        value = conn.compute.get_share_attachment(args.server, args.share)
        print(json.dumps(value.to_dict(), indent=2))
    elif args.mode == "create":
        attrs = {} if args.tag is None else {"tag": args.tag}
        value = conn.compute.create_share_attachment(args.server, args.share, **attrs)
        print(json.dumps(value.to_dict(), indent=2))
    else:
        conn.compute.delete_share_attachment(args.server, args.share)
        print(json.dumps({"delete_requested": args.share}, indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/shareattachments"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    server := flag.String("server", "", "server UUID")
    share := flag.String("share", "", "Manila share UUID")
    mode := flag.String("mode", "list", "list, get, create or delete")
    tag := flag.String("tag", "", "optional guest mount tag for create")
    strict := flag.Bool("strict", false, "return missing attachment errors on delete")
    flag.Parse()
    if *server == "" { log.Fatal("-server is required") }
    if *mode != "list" && *share == "" { log.Fatal("-share is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud),
        sdk.WithMicroversion(sdk.Compute, "2.97"))
    if err != nil { log.Fatal(err) }
    compute, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    switch *mode {
    case "list":
        for value, err := range compute.ShareAttachments.ListRecords(ctx, *server) {
            if err != nil { log.Fatal(err) }
            printJSON(value.Resource)
        }
    case "get":
        value, err := compute.ShareAttachments.GetRecord(ctx, *server, *share)
        if err != nil { log.Fatal(err) }
        printJSON(value.Resource)
    case "create":
        var options []shareattachments.CreateOption
        if *tag != "" {
            options = append(options, shareattachments.WithCreateAttribute("tag", *tag))
        }
        value, err := compute.ShareAttachments.CreateRecord(ctx, *server, *share, options...)
        if err != nil { log.Fatal(err) }
        printJSON(value.Resource)
        fmt.Printf("HTTP %d\n", value.StatusCode)
    case "delete":
        value, err := compute.ShareAttachments.DeleteRecord(ctx, *server, *share,
            shareattachments.WithDeleteIgnoreMissing(!*strict))
        if err != nil { log.Fatal(err) }
        printJSON(value)
    default:
        log.Fatal("unknown -mode")
    }
}
```

## References

- [Compute Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L3422)
- [ShareMapping](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_share.py)
- [Install](../../docs/install.md)
- [Server Shares](https://github.com/openstack/nova/blob/31.0.0/nova/policies/server_shares.py)
- [controller](https://github.com/openstack/nova/blob/31.0.0/nova/api/openstack/compute/server_shares.py)
- [schema](https://github.com/openstack/nova/blob/31.0.0/nova/api/openstack/compute/schemas/server_shares.py)
- [Base](https://github.com/openstack/nova/blob/31.0.0/nova/policies/base.py)
