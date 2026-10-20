# QoS Minimum Bandwidth Records Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/find_qos_minimum_bandwidth_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.FindMinimumBandwidthRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthFindHeaders`

- GET-first uses literal escaped parent and rule, fixed qos_policy_id in member query, raw original kwargs and constructor Body seeds. Only clean400/403/404 fallback to same parent; no default name server hint. Default logical absence is nil and explicit ignore_missing false returns ErrNotFound only after complete successful search.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthGetNoFallbackAndFindMissingCompleteSearch`, `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthFindBodySeedsAndListLocalFilters`.
- Exact ID/name list search sees later duplicates/errors before success. Parent URI seed is excluded from Body; actual member unknown qos_policy_id is projected through fixed URI while Wire keeps raw receipt. Only limit/marker are server query fields and declared Body kwargs are local filters.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthFindLaterDuplicateAndListFailureRemainTerminal`, `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthFindBodySeedsAndListLocalFilters`.
- Find per-call microversion governs member only and fallback retains captured selected default including canonical selected header alias. Clean native retry succeeds and unsafe retry request/header/version/status mutations preserve original proof.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthVersionPhasesCleanRetryAndOwnedHeaderPolicy`.
- Exact class Resource MRO has untyped min_kbps/direction plus inherited id/name, all missing null, unknown received Body retained, no invented TagMixin/NetworkResource Header/default descriptors. Actual literal row, parent URI and physical response snapshots remain separate.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthListLazyQueryLocalFiltersAndUnknowns`.
- Final200..399 completion and every actual400..599 native rejection preserve status/body/header. Accepted malformed shape, Read/Close/custom cancellation/captured Source faults preserve actual receipt and causes; one physical Close. Faulty404 never becomes clean fallback.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthAllStatusAndMalformedReceiptBoundaries`, `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthPhysicalReadCloseCancelAndSourceProof`.
- Concrete options snapshot JSON/maps/pointers/callback outputs, capture fixed selected Source before callbacks, reject nil/malformed parent/proxy/constructor/route controls before HTTP. Safe owned headers/version do not mutate the selected client or enable retry route/status/auth/version changes.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthOptionSnapshotsAndPreflight`, `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthVersionPhasesCleanRetryAndOwnedHeaderPolicy`.
- Source computed location ignores forged response location in Get/Find/List and remains an owned null. Member unknown microversion reads actual chosen fetch version/null, not a forged response token; Wire and Envelope retain literal received values independently.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthComputedLocationCannotBeForgedInResponse`.
- Declared min_kbps is untyped and default-null: list filters use Source local recursive mapping/empty-dict/float-truthiness/bool-number semantics with explicit exact-decimal comparison boundary. List existing-constructor collisions preserve whole physical page receipt.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthListConstructorAndUntypedDictionaryFilters`.
- Known custom JSON marshal failures stop before HTTP with ErrInvalidOption and original MarshalerError/cause; undeclared max_kbps/tags list values are discarded without serialization.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthKnownMarshalCausesPreflightAndUnknownDiscard`.
- Explicit private StopFaultRetryCallbacks=true blocks willing native retry/reauth hooks after compound read/Close/sticky Source/custom cancellation faults. One physical HTTP/Close preserves accepted proof or rejected native Method/URL/status/body/header; unauthorized native ErrOriginal/ErrReauth fields retain independent receipts and causes.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthMixedFaultsStopRetryAndReauthCallbacks`.

Python/Go differences:

- Parent-first Go arguments follow native rule APIs. Parent is a literal policy ID and QoSPolicyID represents URI state outside the default Body view. Mutable Python Resource/Connection/cache/lifecycle is not fully reproduced; computed location is an owned null even when response Body attempts to forge it, with literal value retained in Wire.
- JSON-domain options and deterministic sorted bulk map keys replace arbitrary Python kwargs/runtime objects. Source default QueryParameters only limit/marker; min_kbps/direction/id/name local filtering differs from native upstream rule LIST query builders.
- Local numeric equality uses shared exact-decimal JSON, with bool-number equality and CPython float truthiness/falsey next handling. All CPython binary64 rounding/nonfinite/custom-object comparisons are not claimed.
- HTTP200..399 is the completion success range; Source final<400 informational100..199 is outside this profile. Actual400..599 and accepted/rejected physical causes stay observable.
- Paging restricts same origin, fixed parent collection and original filters; exact versioned-path alias only. Dictionary continuation links are a Go compatibility extension, and rewritten/foreign Source continuations are errors with actual proof.
- Safe Header options and Get Microversion options are explicit Go conveniences absent from the public pinned get proxy. Find per-call version applies to GET only; fallback keeps captured selected default and standalone List version controls all pages.
- Stock Neutron27 get_policy_minimum_bandwidth_rule is project-scoped ADMIN_OR_PARENT_OWNER_READER. Shared QoSPolicy visibility does not alone grant rule read access; fixture403 preservation does not prove live deployment authorization. Other rule kinds/Create/Update/Delete are excluded from this wave.
- This finite new profile explicitly opts StopFaultRetryCallbacks=true. Native unauthorized reauthentication errors intentionally do not Unwrap the two independent ErrOriginal and ErrReauth proofs; public fields preserve them. Unsupported JMESPath/allow_unknown_params and proxy/constructor controls are preflight errors.

### `python:network/v2/get_qos_minimum_bandwidth_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetMinimumBandwidthRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthGetMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthGetHeaders`

- Exact public get proxy constructs literal parent URI without a parent fetch or member parent query, performs one rule GET, seeds only identity, and returns every actual member error without fallback or ignore-missing. Get has no Source query kwargs; safe Header/version convenience is explicit Go extension.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthGetNoFallbackAndFindMissingCompleteSearch`.
- URI qos_policy_id is outside default to_dict unless member response introduces an unknown Body key, then Source getattr(URI) projects the bound parent. Literal raw unknown values, untyped defaults and actual response metadata remain separately owned.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthMemberLiteralRoutesSourceQueryAndURIView`.
- Exact class Resource MRO has untyped min_kbps/direction plus inherited id/name, all missing null, unknown received Body retained, no invented TagMixin/NetworkResource Header/default descriptors. Actual literal row, parent URI and physical response snapshots remain separate.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthListLazyQueryLocalFiltersAndUnknowns`.
- Final200..399 completion and every actual400..599 native rejection preserve status/body/header. Accepted malformed shape, Read/Close/custom cancellation/captured Source faults preserve actual receipt and causes; one physical Close. Faulty404 never becomes clean fallback.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthAllStatusAndMalformedReceiptBoundaries`, `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthPhysicalReadCloseCancelAndSourceProof`.
- Concrete options snapshot JSON/maps/pointers/callback outputs, capture fixed selected Source before callbacks, reject nil/malformed parent/proxy/constructor/route controls before HTTP. Safe owned headers/version do not mutate the selected client or enable retry route/status/auth/version changes.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthOptionSnapshotsAndPreflight`, `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthVersionPhasesCleanRetryAndOwnedHeaderPolicy`.
- Source computed location ignores forged response location in Get/Find/List and remains an owned null. Member unknown microversion reads actual chosen fetch version/null, not a forged response token; Wire and Envelope retain literal received values independently.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthComputedLocationCannotBeForgedInResponse`.
- Explicit private StopFaultRetryCallbacks=true blocks willing native retry/reauth hooks after compound read/Close/sticky Source/custom cancellation faults. One physical HTTP/Close preserves accepted proof or rejected native Method/URL/status/body/header; unauthorized native ErrOriginal/ErrReauth fields retain independent receipts and causes.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthMixedFaultsStopRetryAndReauthCallbacks`.

Python/Go differences:

- Parent-first Go arguments follow native rule APIs. Parent is a literal policy ID and QoSPolicyID represents URI state outside the default Body view. Mutable Python Resource/Connection/cache/lifecycle is not fully reproduced; computed location is an owned null even when response Body attempts to forge it, with literal value retained in Wire.
- JSON-domain options and deterministic sorted bulk map keys replace arbitrary Python kwargs/runtime objects. Source default QueryParameters only limit/marker; min_kbps/direction/id/name local filtering differs from native upstream rule LIST query builders.
- Local numeric equality uses shared exact-decimal JSON, with bool-number equality and CPython float truthiness/falsey next handling. All CPython binary64 rounding/nonfinite/custom-object comparisons are not claimed.
- HTTP200..399 is the completion success range; Source final<400 informational100..199 is outside this profile. Actual400..599 and accepted/rejected physical causes stay observable.
- Paging restricts same origin, fixed parent collection and original filters; exact versioned-path alias only. Dictionary continuation links are a Go compatibility extension, and rewritten/foreign Source continuations are errors with actual proof.
- Safe Header options and Get Microversion options are explicit Go conveniences absent from the public pinned get proxy. Find per-call version applies to GET only; fallback keeps captured selected default and standalone List version controls all pages.
- Stock Neutron27 get_policy_minimum_bandwidth_rule is project-scoped ADMIN_OR_PARENT_OWNER_READER. Shared QoSPolicy visibility does not alone grant rule read access; fixture403 preservation does not prove live deployment authorization. Other rule kinds/Create/Update/Delete are excluded from this wave.
- This finite new profile explicitly opts StopFaultRetryCallbacks=true. Native unauthorized reauthentication errors intentionally do not Unwrap the two independent ErrOriginal and ErrReauth proofs; public fields preserve them. Unsupported JMESPath/allow_unknown_params and proxy/constructor controls are preflight errors.

### `python:network/v2/qos_minimum_bandwidth_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ListMinimumBandwidthRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.AllMinimumBandwidthRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthListPaginated`

- Lazy list uses fixed literal parent without lookup. Only default limit/marker transpose to server query; Body id/name/direction/bandwidth fields filter locally. Unknown kwargs including fields/tags are discarded before transport-value evaluation, even function/malformed unknown values.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthListRawCapSinglePageAndEarlyStop`.
- Source list injects URI parent then consumes it outside Body; Wire owns foreign original row value and whole page proof. Positive raw cap counts before filtering, limit=0 plus cap creates limit hint; single page and early break avoid another request.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthListLazyQueryLocalFiltersAndUnknowns`, `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthListRawCapSinglePageAndEarlyStop`.
- Complete pager handles short-page ID marker fallback, relative and exact versioned continuations, Source falsey float next and malformed truthy next proofs. Parent/origin/filter retargeting fails with actual page evidence; iterator yields earlier rows but All drops partial rows after late failure.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthListContinuationCannotChangeParentOrFilters`, `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthListMarkerFallbackAndFalseyNumericNext`, `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthListPartialIterationAndAllDropLateError`.
- Exact class Resource MRO has untyped min_kbps/direction plus inherited id/name, all missing null, unknown received Body retained, no invented TagMixin/NetworkResource Header/default descriptors. Actual literal row, parent URI and physical response snapshots remain separate.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthMemberLiteralRoutesSourceQueryAndURIView`, `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthListLazyQueryLocalFiltersAndUnknowns`.
- Final200..399 completion and every actual400..599 native rejection preserve status/body/header. Accepted malformed shape, Read/Close/custom cancellation/captured Source faults preserve actual receipt and causes; one physical Close. Faulty404 never becomes clean fallback.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthAllStatusAndMalformedReceiptBoundaries`, `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthPhysicalReadCloseCancelAndSourceProof`.
- Concrete options snapshot JSON/maps/pointers/callback outputs, capture fixed selected Source before callbacks, reject nil/malformed parent/proxy/constructor/route controls before HTTP. Safe owned headers/version do not mutate the selected client or enable retry route/status/auth/version changes.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthOptionSnapshotsAndPreflight`, `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthVersionPhasesCleanRetryAndOwnedHeaderPolicy`.
- Source computed location ignores forged response location in Get/Find/List and remains an owned null. Member unknown microversion reads actual chosen fetch version/null, not a forged response token; Wire and Envelope retain literal received values independently.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthComputedLocationCannotBeForgedInResponse`.
- Declared min_kbps is untyped and default-null: list filters use Source local recursive mapping/empty-dict/float-truthiness/bool-number semantics with explicit exact-decimal comparison boundary. List existing-constructor collisions preserve whole physical page receipt.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthListConstructorAndUntypedDictionaryFilters`.
- Known custom JSON marshal failures stop before HTTP with ErrInvalidOption and original MarshalerError/cause; undeclared max_kbps/tags list values are discarded without serialization.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthKnownMarshalCausesPreflightAndUnknownDiscard`.
- Explicit private StopFaultRetryCallbacks=true blocks willing native retry/reauth hooks after compound read/Close/sticky Source/custom cancellation faults. One physical HTTP/Close preserves accepted proof or rejected native Method/URL/status/body/header; unauthorized native ErrOriginal/ErrReauth fields retain independent receipts and causes.
  Tests: `network/v2/extensions/qos/rules/python_minimum_bandwidth_contracts_test.go:TestPythonMinimumBandwidthMixedFaultsStopRetryAndReauthCallbacks`.

Python/Go differences:

- Parent-first Go arguments follow native rule APIs. Parent is a literal policy ID and QoSPolicyID represents URI state outside the default Body view. Mutable Python Resource/Connection/cache/lifecycle is not fully reproduced; computed location is an owned null even when response Body attempts to forge it, with literal value retained in Wire.
- JSON-domain options and deterministic sorted bulk map keys replace arbitrary Python kwargs/runtime objects. Source default QueryParameters only limit/marker; min_kbps/direction/id/name local filtering differs from native upstream rule LIST query builders.
- Local numeric equality uses shared exact-decimal JSON, with bool-number equality and CPython float truthiness/falsey next handling. All CPython binary64 rounding/nonfinite/custom-object comparisons are not claimed.
- HTTP200..399 is the completion success range; Source final<400 informational100..199 is outside this profile. Actual400..599 and accepted/rejected physical causes stay observable.
- Paging restricts same origin, fixed parent collection and original filters; exact versioned-path alias only. Dictionary continuation links are a Go compatibility extension, and rewritten/foreign Source continuations are errors with actual proof.
- Safe Header options and Get Microversion options are explicit Go conveniences absent from the public pinned get proxy. Find per-call version applies to GET only; fallback keeps captured selected default and standalone List version controls all pages.
- Stock Neutron27 get_policy_minimum_bandwidth_rule is project-scoped ADMIN_OR_PARENT_OWNER_READER. Shared QoSPolicy visibility does not alone grant rule read access; fixture403 preservation does not prove live deployment authorization. Other rule kinds/Create/Update/Delete are excluded from this wave.
- This finite new profile explicitly opts StopFaultRetryCallbacks=true. Native unauthorized reauthentication errors intentionally do not Unwrap the two independent ErrOriginal and ErrReauth proofs; public fields preserve them. Unsupported JMESPath/allow_unknown_params and proxy/constructor controls are preflight errors.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--policy", required=True)
parser.add_argument("--rule", default="")
parser.add_argument("--mode", choices=("find", "get", "list"), default="list")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "find":
        rule = conn.network.find_qos_minimum_bandwidth_rule(
            args.rule, args.policy, ignore_missing=False
        )
        print(json.dumps(rule.to_dict(), indent=2))
    elif args.mode == "get":
        rule = conn.network.get_qos_minimum_bandwidth_rule(args.rule, args.policy)
        print(json.dumps(rule.to_dict(), indent=2))
    else:
        for rule in conn.network.qos_minimum_bandwidth_rules(
            args.policy, direction="egress", max_items=100
        ):
            print(json.dumps(rule.to_dict(), indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func run(ctx context.Context, cloud, mode, parent, identity string) (any, error) {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return nil, err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return nil, err }
    switch mode {
    case "find":
        value, err := network.QoSRules.FindMinimumBandwidthRecord(
            ctx, parent, identity, rules.WithMinimumBandwidthFindIgnoreMissing(false),
        )
        if err != nil || value == nil { return nil, err }
        return value.Resource, nil
    case "get":
        value, err := network.QoSRules.GetMinimumBandwidthRecord(ctx, parent, identity)
        if err != nil { return nil, err }
        return value.Resource, nil
    case "list":
        values, err := network.QoSRules.AllMinimumBandwidthRecords(
            ctx, parent, rules.WithMinimumBandwidthListQuery("direction", "egress"),
            rules.WithMinimumBandwidthListMaxItems(100),
        )
        if err != nil { return nil, err }
        rows := make([]*resource.RawResource, 0, len(values))
        for _, value := range values { rows = append(rows, value.Resource) }
        return rows, nil
    default:
        return nil, fmt.Errorf("unknown mode %q", mode)
    }
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    parent := flag.String("policy", "", "literal QoS policy ID")
    identity := flag.String("rule", "", "literal rule ID or Find identity")
    mode := flag.String("mode", "list", "find, get or list")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    value, err := run(ctx, *cloud, *mode, *parent, *identity)
    if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    if err := encoder.Encode(value); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

## References

- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py#L413)
- [Base](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/base.py#L65)
