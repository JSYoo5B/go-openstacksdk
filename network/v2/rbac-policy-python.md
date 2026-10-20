# Rbac Policy Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/find_rbac_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordFindMicroversion`

- GET-first passes original kwargs to member query and constructor seed, then clean400/403/404 alone allow complete unique exact ID/name list search. No automatic name server hint exists for RBAC mapping. Default clean missing is nil and explicit strict missing is ErrNotFound.
  Tests: `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyFindRecordFallbackQuerySeedsAndNoNameHint`, `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyFindRecordCleanFallbackMissingAndCompleteUniqueness`, `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyFindRecordNativeClean404RetryStillFallsBackOnce`.
- Member requested fields remain untransposed Find kwargs while fallback LIST transposes target_project_id alias separately and applies declared tenant_id filters. Per-call override affects GET only; LIST returns to configured version.
  Tests: `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyFindRecordFallbackQuerySeedsAndNoNameHint`.
- Constructor/route options preflight, actual body defaults and arbitrary extensions reuse Source projector; physical404 or any read/Close/cancel/source failure stays terminal rather than hidden fallback/missing.
  Tests: `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyRecordOptionsOwnershipAndPreflight`, `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyRecordPhysicalCompoundFaultsAreTerminalAndKeepActualProof`.
- Presence of bound rbac_policy, name_or_id or ignore_missing query controls is rejected before HTTP and before evaluating their JSON value, including null/false/zero. Typed IgnoreMissing remains the public control.
  Tests: `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyFindRecordBoundArgumentsFailBeforeHTTP`.
- An actual unknown response microversion is retained unchanged in Wire while the Source runtime attribute in Resource reflects the selected member version string or null; response values never control routing or request version.
  Tests: `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyMemberMicroversionAttributeUsesSelectedVersion`.

Python/Go differences:

- Go takes owned JSON-domain options/record snapshots and literal nonblank UTF-8 IDs rather than Resource/Munch/subclasses; Connection-derived location is null unless received as an unknown extension. Mutable lifecycle and session response-cache parity are not claimed.
- The shared Neutron reader retains existing200..399 admission; Source accepts<400, so synthetic100..199 responses are a documented Go admission difference. All400..599 rejections retain physical read/Close/cancel/source/callback errors and actual native evidence.
- Go owns fixed source/route, selected network version or absence and reserved protocol/header behavior through native retries. Explicit Get/List microversion/header settings are library extensions; normal retry headers and live provider token remain native.
- Get/List/Find return Source-shaped Body descriptors/defaults plus raw unknown response extensions and independent HTTP proof. Native typed APIs retain their existing contract and are not used to claim these three Python callable mappings.
- Route/session/constructor controls and invalid UTF-8 IDs/known JSON are rejected before HTTP; arbitrary dynamic Python kwargs and adapter behavior are not reproduced.
- Unknown list kwargs are discarded before value evaluation; id/name/tenant_id are local declared Body filters even if the server can filter additional fields. Source arbitrary extension query filtering is not inferred from readable unknown Body extensions.
- Local ordinary JSON number equality uses exact decimals rather than Source binary64 rounding/underflow:9007199254740992 versus9007199254740993.0 is unequal, and1e-4000 versus0 is unequal. Source float truthiness is separately preserved for dict filters/paginated/next controls.
- Ordered functional query options preserve kwargs insertion order; bulk Query maps are sorted. Constructor canonical/wire aliases can therefore differ from unordered Python mappings.
- Pagination keeps fixed origin/path/initial filters and detects loops, validates numeric controls, and requires one nonempty wire ID marker query value. Source permissive adapter and arbitrary iterable/object marker encoding are not fully reproduced. Dictionary links are a Go extension.
- Pinned ignore_missing default true/deprecation warning is kept as behavior; Go emits no Python warning. GET unknown kwargs are preserved JSON-domain constructor/query values and discarded on list fallback.
- Go fixes the SDK-owned location view to null while preserving actual values in Wire. Source Computed location may accept explicit row kwargs in LIST construction; mutable Connection-derived location remains unresolved under SDK-R1.

### `python:network/v2/get_rbac_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.API.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordGetFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordGetMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordGetHeaders`

- Literal member GET transposes nonempty public fields to wire target_tenant while preserving requested order/duplicates/unknown field names; nil/empty fields omit query and do not seed Resource. Received null/false/zero/untyped fields and unknown extensions/defaults remain independent actual proof.
  Tests: `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyGetRecordFieldsAliasesDefaultsAndOwnedEvidence`, `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyGetRecordEnvelopeAndMissingAreTerminal`.
- Source ordered canonical/wire aliases retain response input-order last value; project_id alias tenant_id applies only when absent, not explicit null. No tags/revision/timestamp/if_match Body defaults are inherited.
  Tests: `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyRecordsOrderedResponseAliasesAndNullPrecedence`.
- Options own fields/header/raw bytes; invalid controls and canceled custom cause stop before HTTP. Accepted/rejected compound faults preserve method/status/body/header and original read/Close/sticky source/cancel causes with oneClose and no replay.
  Tests: `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyRecordOptionsOwnershipAndPreflight`, `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyRecordPhysicalCompoundFaultsAreTerminalAndKeepActualProof`.
- An actual unknown response microversion is retained unchanged in Wire while the Source runtime attribute in Resource reflects the selected member version string or null; response values never control routing or request version.
  Tests: `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyMemberMicroversionAttributeUsesSelectedVersion`.

Python/Go differences:

- Go takes owned JSON-domain options/record snapshots and literal nonblank UTF-8 IDs rather than Resource/Munch/subclasses; Connection-derived location is null unless received as an unknown extension. Mutable lifecycle and session response-cache parity are not claimed.
- The shared Neutron reader retains existing200..399 admission; Source accepts<400, so synthetic100..199 responses are a documented Go admission difference. All400..599 rejections retain physical read/Close/cancel/source/callback errors and actual native evidence.
- Go owns fixed source/route, selected network version or absence and reserved protocol/header behavior through native retries. Explicit Get/List microversion/header settings are library extensions; normal retry headers and live provider token remain native.
- Get/List/Find return Source-shaped Body descriptors/defaults plus raw unknown response extensions and independent HTTP proof. Native typed APIs retain their existing contract and are not used to claim these three Python callable mappings.
- Route/session/constructor controls and invalid UTF-8 IDs/known JSON are rejected before HTTP; arbitrary dynamic Python kwargs and adapter behavior are not reproduced.
- Source _get_with_fields is exposed through typed []string Fields. Other constructor/fetch kwargs are unavailable on this public get callable. Default404 remains terminal with no list fallback.
- Go fixes the SDK-owned location view to null while preserving actual values in Wire. Source Computed location may accept explicit row kwargs in LIST construction; mutable Connection-derived location remains unresolved under SDK-R1.

### `python:network/v2/rbac_policies`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithRecordListPaginated`

- Lazy list uses exactly declared query aliases and local nonquery Body filters, preserved omission/null/false/zero, Source constructor defaults and raw unknown extensions; no name hint or inherited NetworkResource/TagMixin fields.
  Tests: `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyListRecordQueriesDefaultsAndUnknownDiscard`, `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyRecordsOrderedResponseAliasesAndNullPrecedence`, `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyListRecordLocalFiltersExactEqualityAndFloatTruthiness`.
- Raw cap and consumer break precede trailing constructor/link errors. Four presence collisions(connection/microversion/_synchronized/cls), Source link/short-page marker behavior, fixed scope and late errors preserve actual page receipt.
  Tests: `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyListRecordPagingFormatsAndOriginalWireMarker`, `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyListRecordRawCapsBreakAndConstructorCollisions`, `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyListRecordLatePageAndContinuationErrorsKeepActualReceipt`.
- LIST Accept overrides configured default with application/json or explicit operation header, explicit version governs all pages, native retry protects owned headers and permits ordinary header/fresh token retry. Physical accepted/rejected faults retain actual evidence and no replay.
  Tests: `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyRecordPhysicalCompoundFaultsAreTerminalAndKeepActualProof`, `network/v2/extensions/rbacpolicies/python_records_contracts_test.go:TestPythonRBACPolicyRecordListHeaderAndNativeRetryOwnership`.

Python/Go differences:

- Go takes owned JSON-domain options/record snapshots and literal nonblank UTF-8 IDs rather than Resource/Munch/subclasses; Connection-derived location is null unless received as an unknown extension. Mutable lifecycle and session response-cache parity are not claimed.
- The shared Neutron reader retains existing200..399 admission; Source accepts<400, so synthetic100..199 responses are a documented Go admission difference. All400..599 rejections retain physical read/Close/cancel/source/callback errors and actual native evidence.
- Go owns fixed source/route, selected network version or absence and reserved protocol/header behavior through native retries. Explicit Get/List microversion/header settings are library extensions; normal retry headers and live provider token remain native.
- Get/List/Find return Source-shaped Body descriptors/defaults plus raw unknown response extensions and independent HTTP proof. Native typed APIs retain their existing contract and are not used to claim these three Python callable mappings.
- Route/session/constructor controls and invalid UTF-8 IDs/known JSON are rejected before HTTP; arbitrary dynamic Python kwargs and adapter behavior are not reproduced.
- Unknown list kwargs are discarded before value evaluation; id/name/tenant_id are local declared Body filters even if the server can filter additional fields. Source arbitrary extension query filtering is not inferred from readable unknown Body extensions.
- Local ordinary JSON number equality uses exact decimals rather than Source binary64 rounding/underflow:9007199254740992 versus9007199254740993.0 is unequal, and1e-4000 versus0 is unequal. Source float truthiness is separately preserved for dict filters/paginated/next controls.
- Ordered functional query options preserve kwargs insertion order; bulk Query maps are sorted. Constructor canonical/wire aliases can therefore differ from unordered Python mappings.
- Pagination keeps fixed origin/path/initial filters and detects loops, validates numeric controls, and requires one nonempty wire ID marker query value. Source permissive adapter and arbitrary iterable/object marker encoding are not fully reproduced. Dictionary links are a Go extension.
- Go fixes the SDK-owned location view to null while preserving actual values in Wire. Source Computed location may accept explicit row kwargs in LIST construction; mutable Connection-derived location remains unresolved under SDK-R1.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="dev")
    parser.add_argument("--action", choices=["list", "get", "find"], default="list")
    parser.add_argument("--id", default="")
    parser.add_argument("--fields", default="")
    parser.add_argument("--object-type", default="network")
    parser.add_argument("--target-project", default="")
    parser.add_argument("--strict", action="store_true")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    fields = args.fields.split(",") if args.fields else None
    if args.action == "get":
        value = conn.network.get_rbac_policy(args.id, fields=fields)
    else:
        query = {"object_type": args.object_type}
        if fields:
            query["fields"] = fields
        if args.target_project:
            query["target_project_id"] = args.target_project
        if args.action == "list":
            for value in conn.network.rbac_policies(**query):
                print(json.dumps(value.to_dict(), indent=2, ensure_ascii=False))
            return
        value = conn.network.find_rbac_policy(
            args.id, ignore_missing=not args.strict, **query)
    if value is None:
        print("not found")
    else:
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies"
)

func printPolicy(value *rbacpolicies.RBACPolicyRecord) {
    if value == nil {
        fmt.Println("not found")
        return
    }
    raw, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(raw))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    action := flag.String("action", "list", "list/get/find")
    id := flag.String("id", "", "RBAC policy literal ID")
    fields := flag.String("fields", "", "comma-separated requested fields")
    objectType := flag.String("object-type", "network", "RBAC object type")
    targetProject := flag.String("target-project", "", "target project filter")
    strict := flag.Bool("strict", false, "find reports missing resource")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := rbacpolicies.New(network.RawClient())
    var selected []string
    if *fields != "" { selected = strings.Split(*fields, ",") }
    switch *action {
    case "get":
        value, err := api.GetRecord(ctx, *id,
            rbacpolicies.WithRecordGetFields(selected...))
        if err != nil { log.Fatal(err) }
        printPolicy(value)
    case "list":
        options := []rbacpolicies.RecordListOption{
            rbacpolicies.WithRecordListQuery("object_type", *objectType),
        }
        if len(selected) != 0 {
            options = append(options, rbacpolicies.WithRecordListQuery("fields", selected))
        }
        if *targetProject != "" {
            options = append(options, rbacpolicies.WithRecordListQuery("target_project_id", *targetProject))
        }
        for value, err := range api.ListRecords(ctx, options...) {
            if err != nil { log.Fatal(err) }
            printPolicy(value)
        }
    case "find":
        options := []rbacpolicies.RecordFindOption{
            rbacpolicies.WithRecordFindIgnoreMissing(!*strict),
            rbacpolicies.WithRecordFindQuery("object_type", *objectType),
        }
        if len(selected) != 0 {
            options = append(options, rbacpolicies.WithRecordFindQuery("fields", selected))
        }
        if *targetProject != "" {
            options = append(options, rbacpolicies.WithRecordFindQuery("target_project_id", *targetProject))
        }
        value, err := api.FindRecord(ctx, *id, options...)
        if err != nil { log.Fatal(err) }
        printPolicy(value)
    default:
        log.Fatal("unknown action")
    }
}
```

## References

- [RBACPolicy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/rbac_policy.py)
- [proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5432-L5500)
- [Neutron RBAC API](https://docs.openstack.org/api-ref/network/v2/#rbac-policies)
- [Install](../../docs/install.md)
- [Rbac](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/rbac.py)
- [Python Records Contracts Test](extensions/rbacpolicies/python_records_contracts_test.go)
