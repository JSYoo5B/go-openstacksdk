# QoS Rule Types Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/find_qos_rule_type`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordFindHeaders`

- Four declared Resource Body fields id/name/type/drivers with nullable defaults, Source drivers list boxing, arbitrary untyped nested driver/parameter values and independently owned raw extensions/HTTP evidence. SDK-owned location=null and selected runtime member microversion do not overwrite Wire.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeGetFieldsDefaultsAndOwnedActualEvidence`, `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeLiteralIDPriorityAndAlternateListID`, `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeNestedDriverParametersRemainExactRawJSON`.
- Known options are captured and cloned around SDK callbacks; invalid IDs/UTF-8/known JSON/source/version/auth-route controls fail before HTTP. Lists remain lazy and isolated per iteration.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeRecordOptionsPreflightAndLazyCloneIsolation`.
- New Source profile explicitly StopFaultRetryCallbacks:true. Accepted and all400..599 rejected Read/Close/sticky source/custom cancellation faults are terminal and preserve actual status/body/header without native hook invocation/replay. Clean503 retry retains owned selected version, ordinary native headers and live token; mutated owned version is terminal.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeCompoundPhysicalFaultsAreTerminalWithNoRetryCallback`, `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeNativeCleanRetryAndOwnedHeaderMutation`.
- No arbitrary find kwargs/name hint: only GET-first then clean400/403/404 permits a complete exact id-or-name search. Type is the alternate list id only when literal id is absent. Default missing nil, strict ErrNotFound, complete multipage duplicate ErrAmbiguous.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeFindCleanFallbackDefaultsAndCompleteUniqueness`.
- Per-call explicit version governs only member GET; fallback LIST returns to configured version. Successful direct member preserves requested id and independently projects actual type and selected/null runtime microversion.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeFindGetOnlyVersionOverrideAndLiteralMemberSeed`.
- Server/decode/physical/source/context errors never become missing, wrong found result or lookup fallback.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeErrorsNeverBecomeFalseFoundOrFallback`, `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeCompoundPhysicalFaultsAreTerminalWithNoRetryCallback`.
- All first custom and nil-reset factory callbacks receive initialized Header maps; LIST Query remains initialized. Retained callback maps cannot mutate selected live request/local filters/client. Physical401 with willing originalReauthFunc preserves native pointer ErrUnableToReauthenticate.ErrOriginal actualGET method/URL/status/body/Header and ErrReauth mixedRead/Close/stickySource/customcancel with oneHTTP/oneClose/zeroRetryandReauth hooks.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeCustomCallbacksReceiveInitializedOwnedMaps`, `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeCompoundPhysicalFaultsAreTerminalWithNoRetryCallback`.

Python/Go differences:

- Go accepts literal nonblank UTF-8 identities and concrete owned record/options rather than Python Resource/Munch/subclass selectors or mutable object lifecycle. Connection location is always an ownednull per SDK-R1, including provided list computed kwargs; Wire keeps the actual value.
- Common Neutron reader admits200..399; Source accepts<400, so synthetic100..199 is an explicit Go admission boundary. All400..599 rejections retain physical handling/source/context causes; normal native retry/auth semantics remain.
- Explicit header/microversion transport setters are Go additions. Source public find_qos_rule_type accepts only rule_type_name and ignore_missing, so the Go Find DTO deliberately has no arbitrary Query/fields support.
- Go guards the captured source, immutable member/list routes, selected version or absence and SDK-owned headers through native callbacks/attempts. Received id/type/extensions never select another endpoint; ordinary native retry header/live-token changes remain allowed.
- Pagination validates fixed origin/path/filters/cursor progress and one nonempty query value for a raw Wire id-or-type marker. Literal id presence, including null/false/empty, takes priority. Source permissive adapter and arbitrary iterable/object marker serialization are not fully reproduced; dictionary next links are a Go extension.
- Local ordinary JSON equality uses exact decimals rather than Python binary64 rounding/underflow:9007199254740992 versus9007199254740993.0 and0 versus1e-4000 are unequal. Control/next float truthiness separately preserves Source binary64 behavior.
- Unknown list kwargs are discarded before evaluating their values; only declared server query fields and local Body id/name filters participate. Unknown response extensions do not automatically become query capabilities.
- No actual deployment/extension/driver support, general cache/session behavior, arbitrary Python runtime attributes/callbacks, full shared Resource parity or whole SDK completion is claimed.

### `python:network/v2/get_qos_rule_type`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.API.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordGetFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordGetMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordGetHeaders`

- Four declared Resource Body fields id/name/type/drivers with nullable defaults, Source drivers list boxing, arbitrary untyped nested driver/parameter values and independently owned raw extensions/HTTP evidence. SDK-owned location=null and selected runtime member microversion do not overwrite Wire.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeGetFieldsDefaultsAndOwnedActualEvidence`, `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeLiteralIDPriorityAndAlternateListID`, `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeNestedDriverParametersRemainExactRawJSON`.
- Known options are captured and cloned around SDK callbacks; invalid IDs/UTF-8/known JSON/source/version/auth-route controls fail before HTTP. Lists remain lazy and isolated per iteration.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeRecordOptionsPreflightAndLazyCloneIsolation`.
- New Source profile explicitly StopFaultRetryCallbacks:true. Accepted and all400..599 rejected Read/Close/sticky source/custom cancellation faults are terminal and preserve actual status/body/header without native hook invocation/replay. Clean503 retry retains owned selected version, ordinary native headers and live token; mutated owned version is terminal.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeCompoundPhysicalFaultsAreTerminalWithNoRetryCallback`, `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeNativeCleanRetryAndOwnedHeaderMutation`.
- One literal escaped member GET; nil/empty fields omitted, nonempty fields retain order/duplicates/unknown names and do not seed Resource. Requested member id remains when actual response omits id; actual id includingnull replaces it.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeGetFieldsDefaultsAndOwnedActualEvidence`, `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeLiteralIDPriorityAndAlternateListID`.
- Get does not perform missing lookup fallback and retains native rejected or accepted malformed/nonobject response evidence.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeErrorsNeverBecomeFalseFoundOrFallback`.
- All first custom and nil-reset factory callbacks receive initialized Header maps; LIST Query remains initialized. Retained callback maps cannot mutate selected live request/local filters/client. Physical401 with willing originalReauthFunc preserves native pointer ErrUnableToReauthenticate.ErrOriginal actualGET method/URL/status/body/Header and ErrReauth mixedRead/Close/stickySource/customcancel with oneHTTP/oneClose/zeroRetryandReauth hooks.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeCustomCallbacksReceiveInitializedOwnedMaps`, `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeCompoundPhysicalFaultsAreTerminalWithNoRetryCallback`.

Python/Go differences:

- Go accepts literal nonblank UTF-8 identities and concrete owned record/options rather than Python Resource/Munch/subclass selectors or mutable object lifecycle. Connection location is always an ownednull per SDK-R1, including provided list computed kwargs; Wire keeps the actual value.
- Common Neutron reader admits200..399; Source accepts<400, so synthetic100..199 is an explicit Go admission boundary. All400..599 rejections retain physical handling/source/context causes; normal native retry/auth semantics remain.
- Explicit header/microversion transport setters are Go additions. Source public find_qos_rule_type accepts only rule_type_name and ignore_missing, so the Go Find DTO deliberately has no arbitrary Query/fields support.
- Go guards the captured source, immutable member/list routes, selected version or absence and SDK-owned headers through native callbacks/attempts. Received id/type/extensions never select another endpoint; ordinary native retry header/live-token changes remain allowed.
- Pagination validates fixed origin/path/filters/cursor progress and one nonempty query value for a raw Wire id-or-type marker. Literal id presence, including null/false/empty, takes priority. Source permissive adapter and arbitrary iterable/object marker serialization are not fully reproduced; dictionary next links are a Go extension.
- Local ordinary JSON equality uses exact decimals rather than Python binary64 rounding/underflow:9007199254740992 versus9007199254740993.0 and0 versus1e-4000 are unequal. Control/next float truthiness separately preserves Source binary64 behavior.
- Unknown list kwargs are discarded before evaluating their values; only declared server query fields and local Body id/name filters participate. Unknown response extensions do not automatically become query capabilities.
- No actual deployment/extension/driver support, general cache/session behavior, arbitrary Python runtime attributes/callbacks, full shared Resource parity or whole SDK completion is claimed.

### `python:network/v2/qos_rule_types`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordListPaginated`

- Four declared Resource Body fields id/name/type/drivers with nullable defaults, Source drivers list boxing, arbitrary untyped nested driver/parameter values and independently owned raw extensions/HTTP evidence. SDK-owned location=null and selected runtime member microversion do not overwrite Wire.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeGetFieldsDefaultsAndOwnedActualEvidence`, `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeLiteralIDPriorityAndAlternateListID`, `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeNestedDriverParametersRemainExactRawJSON`.
- Known options are captured and cloned around SDK callbacks; invalid IDs/UTF-8/known JSON/source/version/auth-route controls fail before HTTP. Lists remain lazy and isolated per iteration.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeRecordOptionsPreflightAndLazyCloneIsolation`.
- New Source profile explicitly StopFaultRetryCallbacks:true. Accepted and all400..599 rejected Read/Close/sticky source/custom cancellation faults are terminal and preserve actual status/body/header without native hook invocation/replay. Clean503 retry retains owned selected version, ordinary native headers and live token; mutated owned version is terminal.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeCompoundPhysicalFaultsAreTerminalWithNoRetryCallback`, `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeNativeCleanRetryAndOwnedHeaderMutation`.
- Server fields limit/marker/type/drivers/fields/all_rules/all_supported map Source values; local id/name filters inspect projected Body, unknown kwargs discarded before malformed value evaluation; raw cap precedes filtering and provides initial limit hint.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeListQueryClassificationRawCapAndUnknownDiscard`, `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeListLocalIdentityUsesAlternateTypeAndExactJSONPolicy`.
- Owned shared pager handles body/HTTP/dictionary next links and short-page raw marker fallback. Literal Wire id includingfalse takes precedence over alternate type; null primary id is not repaired by type and fails with actual page proof. Break stops the next request.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeListPagingUsesAlternateAndLiteralWireIDPriority`, `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeListLinkFormsBreakAndConstructorControlCollisions`.
- List rows duplicate Source existing constructor controls connection/microversion/_synchronized/cls by presence even null/false; accepted page proof is retained and partial AllRecords results discarded.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeListLinkFormsBreakAndConstructorControlCollisions`.
- All first custom and nil-reset factory callbacks receive initialized Header maps; LIST Query remains initialized. Retained callback maps cannot mutate selected live request/local filters/client. Physical401 with willing originalReauthFunc preserves native pointer ErrUnableToReauthenticate.ErrOriginal actualGET method/URL/status/body/Header and ErrReauth mixedRead/Close/stickySource/customcancel with oneHTTP/oneClose/zeroRetryandReauth hooks.
  Tests: `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeCustomCallbacksReceiveInitializedOwnedMaps`, `network/v2/extensions/qos/ruletypes/python_records_contracts_test.go:TestPythonQoSRuleTypeCompoundPhysicalFaultsAreTerminalWithNoRetryCallback`.

Python/Go differences:

- Go accepts literal nonblank UTF-8 identities and concrete owned record/options rather than Python Resource/Munch/subclass selectors or mutable object lifecycle. Connection location is always an ownednull per SDK-R1, including provided list computed kwargs; Wire keeps the actual value.
- Common Neutron reader admits200..399; Source accepts<400, so synthetic100..199 is an explicit Go admission boundary. All400..599 rejections retain physical handling/source/context causes; normal native retry/auth semantics remain.
- Explicit header/microversion transport setters are Go additions. Source public find_qos_rule_type accepts only rule_type_name and ignore_missing, so the Go Find DTO deliberately has no arbitrary Query/fields support.
- Go guards the captured source, immutable member/list routes, selected version or absence and SDK-owned headers through native callbacks/attempts. Received id/type/extensions never select another endpoint; ordinary native retry header/live-token changes remain allowed.
- Pagination validates fixed origin/path/filters/cursor progress and one nonempty query value for a raw Wire id-or-type marker. Literal id presence, including null/false/empty, takes priority. Source permissive adapter and arbitrary iterable/object marker serialization are not fully reproduced; dictionary next links are a Go extension.
- Local ordinary JSON equality uses exact decimals rather than Python binary64 rounding/underflow:9007199254740992 versus9007199254740993.0 and0 versus1e-4000 are unequal. Control/next float truthiness separately preserves Source binary64 behavior.
- Unknown list kwargs are discarded before evaluating their values; only declared server query fields and local Body id/name filters participate. Unknown response extensions do not automatically become query capabilities.
- No actual deployment/extension/driver support, general cache/session behavior, arbitrary Python runtime attributes/callbacks, full shared Resource parity or whole SDK completion is claimed.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--operation", choices=["get", "find", "list"], default="list")
parser.add_argument("--name", default="bandwidth_limit")
parser.add_argument("--fields")
parser.add_argument("--type")
parser.add_argument("--all-rules", action="store_true")
parser.add_argument("--all-supported", action="store_true")
parser.add_argument("--max-items", type=int, default=0)
parser.add_argument("--strict", action="store_true")
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
fields = args.fields.split(",") if args.fields else None
if args.operation == "get":
    value = conn.network.get_qos_rule_type(args.name, fields=fields)
    print(value.to_dict())
elif args.operation == "find":
    # The pinned public find method has no **query or fields argument.
    value = conn.network.find_qos_rule_type(args.name, ignore_missing=not args.strict)
    print(None if value is None else value.to_dict())
else:
    query = {}
    if fields:
        query["fields"] = fields
    if args.type is not None:
        query["type"] = args.type
    if args.all_rules:
        query["all_rules"] = True
    if args.all_supported:
        query["all_supported"] = True
    if args.max_items:
        query["max_items"] = args.max_items
    for value in conn.network.qos_rule_types(**query):
        print(value.to_dict())
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
)

func printType(value *ruletypes.QoSRuleTypeRecord) {
    if value == nil { fmt.Println("not found"); return }
    data, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    operation := flag.String("operation", "list", "get/find/list")
    name := flag.String("name", "bandwidth_limit", "literal rule type name")
    fields := flag.String("fields", "", "comma-separated fields")
    typeFilter := flag.String("type", "", "server type filter")
    allRules := flag.Bool("all-rules", false, "server all_rules control")
    allSupported := flag.Bool("all-supported", false, "server all_supported control")
    maxItems := flag.Int("max-items", 0, "raw row cap; zero is unlimited")
    strict := flag.Bool("strict", false, "find reports missing")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := service.QoSRuleTypes
    var selected []string
    if *fields != "" { selected = strings.Split(*fields, ",") }
    switch *operation {
    case "get":
        value, err := api.GetRecord(ctx, *name, ruletypes.WithRecordGetFields(selected...))
        if err != nil { log.Fatal(err) }
        printType(value)
    case "find":
        value, err := api.FindRecord(ctx, *name, ruletypes.WithRecordFindIgnoreMissing(!*strict))
        if err != nil { log.Fatal(err) }
        printType(value)
    case "list":
        options := []ruletypes.RecordListOption{ruletypes.WithRecordListMaxItems(*maxItems)}
        if len(selected) != 0 { options = append(options, ruletypes.WithRecordListQuery("fields", selected)) }
        if *typeFilter != "" { options = append(options, ruletypes.WithRecordListQuery("type", *typeFilter)) }
        if *allRules { options = append(options, ruletypes.WithRecordListQuery("all_rules", true)) }
        if *allSupported { options = append(options, ruletypes.WithRecordListQuery("all_supported", true)) }
        for value, err := range api.ListRecords(ctx, options...) {
            if err != nil { log.Fatal(err) }
            printType(value)
        }
    default: log.Fatal("unknown operation")
    }
}
```

## References

- [Policy](https://docs.openstack.org/neutron/2025.2/configuration/policy.html)
- [V2](https://docs.openstack.org/api-ref/network/v2/#qos-rule-types)
- [Install](../../docs/install.md)
- [three proxy declarations](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5212-L5285)
- [QoSRuleType schema](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_rule_type.py#L16-L42)
- [Resource alternate id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L654-L672)
