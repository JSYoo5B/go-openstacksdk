# Cloud QoS Minimum Bandwidth Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/list_qos_minimum_bandwidth_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.ListQoSMinimumBandwidthRules`
- `github.com/JSYoo5B/go-openstacksdk/network.WithMinimumBandwidthCloudOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithMinimumBandwidthCloudCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network.WithMinimumBandwidthCloudFiltersJSON`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ProxyMinimumBandwidthRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthProxyQuery`

- Every call consumes the full fresh qos extension inventory before GET-first exact policy Find; early qos match does not suppress later alias/page faults. Missing or ambiguous policy stops rule I/O with actual already observed gate/policy evidence, no fabricated policy response.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthGateConsumesAllExtensionsAndStopsBeforePolicy`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPolicyMissingAndAmbiguityRetainActualGate`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFreshCompleteGatePolicyNameLookupAndSourceLinkPrecedence`.
- Selected original network service/client/source is captured before owned option callbacks; cancellation causes and source mutation remain terminal. Nil context/service fails before discovery or invented response. Location and raw JSON options are cloned and callbacks run once.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFilterOwnershipCallbackGuardAndFalseyCases`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPreflightAndInvalidLocationHaveHonestEvidence`.
- Actual HTTP physical rule faults and late pages retain partial Inventory plus native response bytes/header/status in error, with no successful Value/Rules. Real Content-Length faults at 200/203/401/404/429/503 preserve pointer401 ErrOriginal VALUE response and ErrReauth EOF; retry/backoff/reauth fault hooks remain zero.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPhysical401RulePageAndLateFailurePreserveStageProof`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPhysicalLatePagesKeepActualHeaderStatusAndNoFaultHooks`.
- Falsey cloud filters become empty kwargs only after extension/policy lookup; truthy nonmappings fail Source **kwargs expansion before rule HTTP. Source Body filters are local while server queries are only declared limit/marker. Full paging and CPython JSON-loaded equality are owned by the concrete Source family profile.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthListOwnsGateParentPagingAndRawFields`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFilterOwnershipCallbackGuardAndFalseyCases`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSearchKeepsRuleIDPhaseAndFilterPresence`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFalseyMaxItemsAndTemplateOnlyJSONArguments`.
- Reachable Proxy/Resource formals headers, microversion, paginated/max_items, nullable/empty BasePath and JSON-domain %(key)s route binding are handled by an owned family Source lane; one-pass conflicting attrs restores the URI parent and preserves original JMES expression. Formal duplicates fail before rule I/O. Legacy typed list policy remains unchanged.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSourceFormalFiltersAndRestoredURIBeforeJMES`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSourceProxyLiteralsBasePathAndOldTypedBoundary`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFalseyMaxItemsAndTemplateOnlyJSONArguments`.
- Full JMESPath evaluates the opaque generator before rule collection. Identity/default returns actual rules; array/string/dict literals return arbitrary list() values with zero rule HTTP and no invented records. Nested generators remain lazy, guarded, location-bound and represented in Values without fake JSON Value.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSourceProxyLiteralsBasePathAndOldTypedBoundary`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPolymorphicFalseySearchAndNestedGeneratorMembership`.
- Rule Resource default/unknown Body, raw Wire, envelope/header/status and URI parent remain distinct. Standalone location null and owned connected current_location are applied before filtering; undeclared project_id/availability_zone cannot change the configured project/zone. Selected Rules, Values, Inventory, LastRule and JSON Value do not alias exported rule metadata/bytes.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthListOwnsGateParentPagingAndRawFields`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPreflightAndInvalidLocationHaveHonestEvidence`.

Python/Go differences:

- Go returns an owned result carrier with real Extensions/Policy/Inventory/LastRule evidence, correlated Rules/Rule and independent Values/Value; literal/opaque output never fabricates response Resource/Wire/Header/status. Failed page receipt remains in the actual error rather than a fictional failed resource.
- Source kwargs are restricted to JSON-domain data, policy/rule identities to safe nonblank Go strings, HTTP headers to string-valued mappings, microversion/base_path to nullable strings and truthy max_items to nonnegative Go integers or true. Go validates these typed/ownership boundaries and service-relative URI safety before some Python dynamic errors; unsafe origin/query/fragment/traversal and non-string numeric %-route formatting are explicit errors.
- Dictionary-key iterable results and key-based JMESPath functions use reviewed Go lexical order instead of Python insertion order. JMES deprecation warnings are omitted. Opaque generator repr/to_string involving Python process addresses and non-JSON dynamic object/session/mutable Resource behavior remain SDK-R1/C1/S1 scope; no equivalent process address string is fabricated.
- Explicit CurrentLocation is an owned configured snapshot; the rule classes have no project_id/availability_zone descriptors, so unknown remote fields cannot override it. Root Connection wrappers may supply this library-owned option automatically. Existing paging bounds/cycle checks/HTTP Link repair and fixed request guards remain documented Go improvements. Authenticated live Neutron acceptance is separate from testhelper HTTP/Source AST proof.

### `python:cloud/_network/NetworkCloudMixin/search_qos_minimum_bandwidth_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.SearchQoSMinimumBandwidthRules`
- `github.com/JSYoo5B/go-openstacksdk/network.WithMinimumBandwidthCloudOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithMinimumBandwidthCloudCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network.WithMinimumBandwidthCloudFiltersJSON`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ProxyMinimumBandwidthRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthProxyQuery`

- Every call consumes the full fresh qos extension inventory before GET-first exact policy Find; early qos match does not suppress later alias/page faults. Missing or ambiguous policy stops rule I/O with actual already observed gate/policy evidence, no fabricated policy response.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthGateConsumesAllExtensionsAndStopsBeforePolicy`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPolicyMissingAndAmbiguityRetainActualGate`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFreshCompleteGatePolicyNameLookupAndSourceLinkPrecedence`.
- Selected original network service/client/source is captured before owned option callbacks; cancellation causes and source mutation remain terminal. Nil context/service fails before discovery or invented response. Location and raw JSON options are cloned and callbacks run once.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFilterOwnershipCallbackGuardAndFalseyCases`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPreflightAndInvalidLocationHaveHonestEvidence`.
- Actual HTTP physical rule faults and late pages retain partial Inventory plus native response bytes/header/status in error, with no successful Value/Rules. Real Content-Length faults at 200/203/401/404/429/503 preserve pointer401 ErrOriginal VALUE response and ErrReauth EOF; retry/backoff/reauth fault hooks remain zero.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPhysical401RulePageAndLateFailurePreserveStageProof`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPhysicalLatePagesKeepActualHeaderStatusAndNoFaultHooks`.
- Falsey cloud filters become empty kwargs only after extension/policy lookup; truthy nonmappings fail Source **kwargs expansion before rule HTTP. Source Body filters are local while server queries are only declared limit/marker. Full paging and CPython JSON-loaded equality are owned by the concrete Source family profile.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthListOwnsGateParentPagingAndRawFields`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFilterOwnershipCallbackGuardAndFalseyCases`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSearchKeepsRuleIDPhaseAndFilterPresence`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFalseyMaxItemsAndTemplateOnlyJSONArguments`.
- Reachable Proxy/Resource formals headers, microversion, paginated/max_items, nullable/empty BasePath and JSON-domain %(key)s route binding are handled by an owned family Source lane; one-pass conflicting attrs restores the URI parent and preserves original JMES expression. Formal duplicates fail before rule I/O. Legacy typed list policy remains unchanged.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSourceFormalFiltersAndRestoredURIBeforeJMES`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSourceProxyLiteralsBasePathAndOldTypedBoundary`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFalseyMaxItemsAndTemplateOnlyJSONArguments`.
- Full JMESPath evaluates the opaque generator before rule collection. Identity/default returns actual rules; array/string/dict literals return arbitrary list() values with zero rule HTTP and no invented records. Nested generators remain lazy, guarded, location-bound and represented in Values without fake JSON Value.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSourceProxyLiteralsBasePathAndOldTypedBoundary`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPolymorphicFalseySearchAndNestedGeneratorMembership`.
- Cloud search first finishes list, then uses exact ID/name or fnmatch glob and ordered/nested Source getter dictionary filtering, including loaded float precision. Unknown attributes fail lazily with actual final successful receipt; an earlier mismatch skips later invalid keys. Top-level truthy string filters remain unreachable JMES because list **kwargs fails first.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSearchKeepsRuleIDPhaseAndFilterPresence`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthOrderedNestedCloudFiltersAndLoadedFloatEquality`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFilterOwnershipCallbackGuardAndFalseyCases`.
- Without identifier selection, Source skips falsey JSON literal values before dictionary lookup. Identifier get() errors happen before that skip. Unknown-key membership on a nested opaque generator consumes its actual rules before failure and retains those actual Inventory/receipts.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPolymorphicFalseySearchAndNestedGeneratorMembership`.
- Rule Resource default/unknown Body, raw Wire, envelope/header/status and URI parent remain distinct. Standalone location null and owned connected current_location are applied before filtering; undeclared project_id/availability_zone cannot change the configured project/zone. Selected Rules, Values, Inventory, LastRule and JSON Value do not alias exported rule metadata/bytes.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthListOwnsGateParentPagingAndRawFields`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPreflightAndInvalidLocationHaveHonestEvidence`.

Python/Go differences:

- Go returns an owned result carrier with real Extensions/Policy/Inventory/LastRule evidence, correlated Rules/Rule and independent Values/Value; literal/opaque output never fabricates response Resource/Wire/Header/status. Failed page receipt remains in the actual error rather than a fictional failed resource.
- Source kwargs are restricted to JSON-domain data, policy/rule identities to safe nonblank Go strings, HTTP headers to string-valued mappings, microversion/base_path to nullable strings and truthy max_items to nonnegative Go integers or true. Go validates these typed/ownership boundaries and service-relative URI safety before some Python dynamic errors; unsafe origin/query/fragment/traversal and non-string numeric %-route formatting are explicit errors.
- Dictionary-key iterable results and key-based JMESPath functions use reviewed Go lexical order instead of Python insertion order. JMES deprecation warnings are omitted. Opaque generator repr/to_string involving Python process addresses and non-JSON dynamic object/session/mutable Resource behavior remain SDK-R1/C1/S1 scope; no equivalent process address string is fabricated.
- Explicit CurrentLocation is an owned configured snapshot; the rule classes have no project_id/availability_zone descriptors, so unknown remote fields cannot override it. Root Connection wrappers may supply this library-owned option automatically. Existing paging bounds/cycle checks/HTTP Link repair and fixed request guards remain documented Go improvements. Authenticated live Neutron acceptance is separate from testhelper HTTP/Source AST proof.

### `python:cloud/_network/NetworkCloudMixin/get_qos_minimum_bandwidth_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetMinimumBandwidthRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.GetQoSMinimumBandwidthRule`
- `github.com/JSYoo5B/go-openstacksdk/network.WithMinimumBandwidthCloudOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithMinimumBandwidthCloudCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetSourceMinimumBandwidthRecord`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Every call consumes the full fresh qos extension inventory before GET-first exact policy Find; early qos match does not suppress later alias/page faults. Missing or ambiguous policy stops rule I/O with actual already observed gate/policy evidence, no fabricated policy response.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthGateConsumesAllExtensionsAndStopsBeforePolicy`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPolicyMissingAndAmbiguityRetainActualGate`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFreshCompleteGatePolicyNameLookupAndSourceLinkPrecedence`.
- Selected original network service/client/source is captured before owned option callbacks; cancellation causes and source mutation remain terminal. Nil context/service fails before discovery or invented response. Location and raw JSON options are cloned and callbacks run once.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFilterOwnershipCallbackGuardAndFalseyCases`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPreflightAndInvalidLocationHaveHonestEvidence`.
- Actual HTTP physical rule faults and late pages retain partial Inventory plus native response bytes/header/status in error, with no successful Value/Rules. Real Content-Length faults at 200/203/401/404/429/503 preserve pointer401 ErrOriginal VALUE response and ErrReauth EOF; retry/backoff/reauth fault hooks remain zero.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPhysical401RulePageAndLateFailurePreserveStageProof`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPhysicalLatePagesKeepActualHeaderStatusAndNoFaultHooks`.
- Named member getter resolves policy with ignore_missing=False for minimum bandwidth. Then one literal escaped member GET uses Source status/projection, no rule Find/list fallback or suppressed rule404. Nonstring policy identity fails with actual policy response proof.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthGetResolvesPolicyOnceAndKeeps404`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthGetterPolicyStrictnessAndMalformedPolicyReceipt`.
- Rule Resource default/unknown Body, raw Wire, envelope/header/status and URI parent remain distinct. Standalone location null and owned connected current_location are applied before filtering; undeclared project_id/availability_zone cannot change the configured project/zone. Selected Rules, Values, Inventory, LastRule and JSON Value do not alias exported rule metadata/bytes.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthListOwnsGateParentPagingAndRawFields`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPreflightAndInvalidLocationHaveHonestEvidence`.

Python/Go differences:

- Go returns an owned result carrier with real Extensions/Policy/Inventory/LastRule evidence, correlated Rules/Rule and independent Values/Value; literal/opaque output never fabricates response Resource/Wire/Header/status. Failed page receipt remains in the actual error rather than a fictional failed resource.
- Source kwargs are restricted to JSON-domain data, policy/rule identities to safe nonblank Go strings, HTTP headers to string-valued mappings, microversion/base_path to nullable strings and truthy max_items to nonnegative Go integers or true. Go validates these typed/ownership boundaries and service-relative URI safety before some Python dynamic errors; unsafe origin/query/fragment/traversal and non-string numeric %-route formatting are explicit errors.
- Dictionary-key iterable results and key-based JMESPath functions use reviewed Go lexical order instead of Python insertion order. JMES deprecation warnings are omitted. Opaque generator repr/to_string involving Python process addresses and non-JSON dynamic object/session/mutable Resource behavior remain SDK-R1/C1/S1 scope; no equivalent process address string is fabricated.
- Explicit CurrentLocation is an owned configured snapshot; the rule classes have no project_id/availability_zone descriptors, so unknown remote fields cannot override it. Root Connection wrappers may supply this library-owned option automatically. Existing paging bounds/cycle checks/HTTP Link repair and fixed request guards remain documented Go improvements. Authenticated live Neutron acceptance is separate from testhelper HTTP/Source AST proof.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import os
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
    parser.add_argument("--policy", required=True)
    parser.add_argument("--rule", default="")
    parser.add_argument("--mode", choices=("list", "search", "get"), default="list")
    parser.add_argument("--filters", default="null")
    args = parser.parse_args()
    with openstack.connect(cloud=args.cloud) as conn:
        filters = json.loads(args.filters)
        if args.mode == "list":
            values = conn.list_qos_minimum_bandwidth_rules(args.policy, filters)
        elif args.mode == "search":
            values = conn.search_qos_minimum_bandwidth_rules(args.policy, args.rule or None, filters)
        else:
            if not args.rule:
                parser.error("--rule required for get")
            values = [conn.get_qos_minimum_bandwidth_rule(args.policy, args.rule)]
        for value in values:
            print(json.dumps(value.to_dict() if hasattr(value, "to_dict") else value,
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
 "os"
 "time"

 sdk "github.com/JSYoo5B/go-openstacksdk"
 "github.com/JSYoo5B/go-openstacksdk/network"
)
func main() {
 cloud:=flag.String("cloud",os.Getenv("OS_CLOUD"),"clouds.yaml cloud")
 policy:=flag.String("policy","","QoS policy ID or name")
 rule:=flag.String("rule","","rule ID or fnmatch pattern")
 mode:=flag.String("mode","list","list, search or get")
 filters:=flag.String("filters","null","JSON-domain Source filters")
 flag.Parse()
 if *policy=="" {log.Fatal("-policy required")}
 ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second)
 defer cancel()
 var connect []sdk.ConnectionOption
 if *cloud!=""{connect=append(connect,sdk.WithCloud(*cloud))}
 conn,err:=sdk.Connect(ctx,connect...)
 if err!=nil{log.Fatal(err)}
 service,err:=conn.Network(ctx)
 if err!=nil{log.Fatal(err)}
 location,err:=conn.CurrentLocation()
 if err!=nil{log.Fatal(err)}
 options:=[]network.MinimumBandwidthCloudOption{network.WithMinimumBandwidthCloudCurrentLocation(location)}
 var result *network.MinimumBandwidthCloudResult
 switch *mode {
 case "list":
  options=append(options,network.WithMinimumBandwidthCloudFiltersJSON(json.RawMessage(*filters)))
  result,err=service.ListQoSMinimumBandwidthRules(ctx,*policy,options...)
 case "search":
  options=append(options,network.WithMinimumBandwidthCloudFiltersJSON(json.RawMessage(*filters)))
  result,err=service.SearchQoSMinimumBandwidthRules(ctx,*policy,*rule,options...)
 case "get":
  if *rule==""{log.Fatal("-rule required for get")}
  result,err=service.GetQoSMinimumBandwidthRule(ctx,*policy,*rule,options...)
 default:log.Fatal("mode must be list, search or get")
 }
 if err!=nil{
  if result!=nil{fmt.Fprintf(os.Stderr,"observed extensions=%d rules=%d\n",len(result.Extensions),len(result.Inventory))}
  log.Fatal(err)
 }
 if result.Value==nil{fmt.Printf("opaque result: %d values\n",len(result.Values));return}
 fmt.Println(string(result.Value))
}
```

## References

- [ Network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py)
- [QoS](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/qos.py)
- [Cloud QoS Minimum Bandwidth Test](cloud_qos_minimum_bandwidth_test.go)
