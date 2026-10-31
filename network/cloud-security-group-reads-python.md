# Cloud Security Group Reads Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network_common/NetworkCommonCloudMixin/search_security_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.SearchSecurityGroupRecords`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.SearchSecurityGroups`
- `github.com/JSYoo5B/go-openstacksdk/network.SecurityGroupCloud.SearchSecurityGroups`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SecurityGroupCloud`
- `github.com/JSYoo5B/go-openstacksdk/network.NewSecurityGroupCloud`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadFiltersJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadExpression`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordProxyQuery`

- SDK-owned concrete Neutron/Nova backend selection follows Source source and network availability. Connection only falls back to Nova on clean absent Neutron catalog endpoint, never HTTP404/cancellation/joined discovery failure. Explicit Source empty/unrecognized feature errors perform no HTTP. Nova source casing and deployment/API/microversion restrictions are explicit.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaSourceSelectionAndNeutron404DoesNotFallback`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaAbsenceFallbackNormalizationAndStrictCompatibility`, `connectiontest/connection_security_group_records_test.go:TestConnectionSecurityGroupRecordsCleanCatalogAbsenceSelectsNova`, `connectiontest/connection_security_group_records_test.go:TestConnectionSecurityGroupRecordsDoNotFallbackOnJoinedFailureOrCancellation`.
- Neutron dictionary filters alone push down concrete Proxy kwargs, then the complete paged inventory receives exact ID/name/fnmatch and original cloud mapping. Top-level string/scalar/list filters request the full list without name/filter query, then JMESPath or Source lazy mapping. Falsey/underflow, ordered/nested keys and loaded numeric equality are preserved.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupSearchDictionaryPushdownThenWholeLocalSelection`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupSearchStringJMESHasNoServerQueryAndArbitraryJSON`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupOrderedNestedAndLoadedFilters`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupFalseyAndNonmappingFiltersAreLocalAfterWholeList`.
- Reachable Proxy headers/version/BasePath/paginated/max_items and conflicting-attrs formals use an opt-in Source family lane. Opaque generator JMESPath is evaluated before HTTP; falsey literal search skip, earlier identifier get failure and nested generator membership consumption retain actual receipts without fabricated Resource/Wire.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupProxyLiteralsAndNestedMembershipDoNotInventRows`, `network/v2/extensions/security/groups/python_list_proxy_test.go:TestPythonSecurityGroupSourceProxyOpaqueJMESAndLegacyTypedBoundary`, `network/v2/extensions/security/groups/python_list_proxy_test.go:TestPythonSecurityGroupSourceProxyFormalsBasePathAndSnapshot`, `network/v2/extensions/security/groups/python_list_proxy_test.go:TestPythonSecurityGroupSourceProxyDuplicateControlsAndLazyContext`.
- One Nova legacy GET is normalized by library-owned Source contracts: raw numeric/string IDs, eager pop/default precedence, artifact removal, rules aliases, properties, strict compatibility, loaded current/foreign project location, Python minimum int conversion and unchanged Source maximum-port string bug. Malformed/partial normalization preserves actual NovaResponse/envelope/status/header.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaAbsenceFallbackNormalizationAndStrictCompatibility`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaDirectGetAndEagerRuleDefaultsPreserveSourceBug`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaMalformedRowsKeepActualPartialProof`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaFalseyContainersAndQueryShapes`.
- Input options/filters/location, normalized Resource/raw Wire, selected records and partial Inventory/LastGroup own independent bytes and headers. Selected original API/client/route is captured before callbacks. Context cancellation/source mutation and real late 200/203/401/404/429/503 body faults preserve original native pointer401 causes and response receipt while stopping retry/backoff/reauth fault hooks and successful partial selection.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupReadOwnsOptionsLocationAndExportedCarriers`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupReadSourceGuardCancellationAndPreflight`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupReadLatePhysicalPagesKeepNativeReceiptsAndStopHooks`, `connectiontest/connection_security_group_records_test.go:TestConnectionSecurityGroupRecordsNeutronDefaultsAndLocationBeforeFilter`.

Python/Go differences:

- Go returns SecurityGroupReadResult with actual backend/Inventory/LastGroup/NovaResponse evidence and independent Groups/Group/Values/Value. JMES arbitrary JSON has Expression=true and no invented record correlation; Nova logical normalized Munch view is distinguished from its actual Nova Wire, not claimed to be a Neutron class instance.
- Go exposes concrete SecurityGroupReadDependencies and typed transport/current-location options, with owned resets and source invariants instead of developer interfaces/builders. It restricts transport headers/version/BasePath/route formatting/max_items and finite JSON input to safe Go representations; explicit Source policy flags are factory data while Connection supplies default backend/location without expanding global configuration scope.
- Go preserves the Source minimum conversion and maximum-port string normalization bug, eager pop/default behavior, source casing and strict compatibility. Nova proxy availability is conditional on actual deployment/API/microversion; modern Nova 2.36 proxy404 is not represented as universally supported fallback.
- Go lexical dict-key/JMESPath key-function order, omitted deprecation/log warnings, escaped service-relative routes, fixed request guards and stronger paging/cycle/Link safety are documented differences. Python arbitrary object/attribute/repr addresses, mutable Resource/dirty/session/cache and lazy materialization behavior remain SDK-R1/C1/S1 generic scope, separate from authenticated live cloud acceptance.

### `python:cloud/_network_common/NetworkCommonCloudMixin/get_security_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetSecurityGroupRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.GetSecurityGroup`
- `github.com/JSYoo5B/go-openstacksdk/network.SecurityGroupCloud.GetSecurityGroup`
- `github.com/JSYoo5B/go-openstacksdk/network.ExistingSecurityGroupRead`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.GetSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SecurityGroupCloud`
- `github.com/JSYoo5B/go-openstacksdk/network.NewSecurityGroupCloud`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadFiltersJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadExpression`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordProxyQuery`

- SDK-owned concrete Neutron/Nova backend selection follows Source source and network availability. Connection only falls back to Nova on clean absent Neutron catalog endpoint, never HTTP404/cancellation/joined discovery failure. Explicit Source empty/unrecognized feature errors perform no HTTP. Nova source casing and deployment/API/microversion restrictions are explicit.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaSourceSelectionAndNeutron404DoesNotFallback`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaAbsenceFallbackNormalizationAndStrictCompatibility`, `connectiontest/connection_security_group_records_test.go:TestConnectionSecurityGroupRecordsCleanCatalogAbsenceSelectsNova`, `connectiontest/connection_security_group_records_test.go:TestConnectionSecurityGroupRecordsDoNotFallbackOnJoinedFailureOrCancellation`.
- Neutron dictionary filters alone push down concrete Proxy kwargs, then the complete paged inventory receives exact ID/name/fnmatch and original cloud mapping. Top-level string/scalar/list filters request the full list without name/filter query, then JMESPath or Source lazy mapping. Falsey/underflow, ordered/nested keys and loaded numeric equality are preserved.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupSearchDictionaryPushdownThenWholeLocalSelection`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupSearchStringJMESHasNoServerQueryAndArbitraryJSON`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupOrderedNestedAndLoadedFilters`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupFalseyAndNonmappingFiltersAreLocalAfterWholeList`.
- Reachable Proxy headers/version/BasePath/paginated/max_items and conflicting-attrs formals use an opt-in Source family lane. Opaque generator JMESPath is evaluated before HTTP; falsey literal search skip, earlier identifier get failure and nested generator membership consumption retain actual receipts without fabricated Resource/Wire.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupProxyLiteralsAndNestedMembershipDoNotInventRows`, `network/v2/extensions/security/groups/python_list_proxy_test.go:TestPythonSecurityGroupSourceProxyOpaqueJMESAndLegacyTypedBoundary`, `network/v2/extensions/security/groups/python_list_proxy_test.go:TestPythonSecurityGroupSourceProxyFormalsBasePathAndSnapshot`, `network/v2/extensions/security/groups/python_list_proxy_test.go:TestPythonSecurityGroupSourceProxyDuplicateControlsAndLazyContext`.
- One Nova legacy GET is normalized by library-owned Source contracts: raw numeric/string IDs, eager pop/default precedence, artifact removal, rules aliases, properties, strict compatibility, loaded current/foreign project location, Python minimum int conversion and unchanged Source maximum-port string bug. Malformed/partial normalization preserves actual NovaResponse/envelope/status/header.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaAbsenceFallbackNormalizationAndStrictCompatibility`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaDirectGetAndEagerRuleDefaultsPreserveSourceBug`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaMalformedRowsKeepActualPartialProof`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaFalseyContainersAndQueryShapes`.
- Input options/filters/location, normalized Resource/raw Wire, selected records and partial Inventory/LastGroup own independent bytes and headers. Selected original API/client/route is captured before callbacks. Context cancellation/source mutation and real late 200/203/401/404/429/503 body faults preserve original native pointer401 causes and response receipt while stopping retry/backoff/reauth fault hooks and successful partial selection.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupReadOwnsOptionsLocationAndExportedCarriers`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupReadSourceGuardCancellationAndPreflight`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupReadLatePhysicalPagesKeepNativeReceiptsAndStopHooks`, `connectiontest/connection_security_group_records_test.go:TestConnectionSecurityGroupRecordsNeutronDefaultsAndLocationBeforeFilter`.
- Default getter searches the complete list and applies Source truthiness/len/index0: empty/falsey means None, one ordinary record means Group, duplicates are typed ambiguity, and expression false/0/null/string/object/scalar cases preserve Source distinction. Explicit UUID-like direct getter ignores filters and keeps member404; it never creates implicit GET-first Find behavior.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupGetDefaultListsAllAndReturnsNoneOneOrAmbiguity`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupGetExpressionFirstScalarAndFalseySemantics`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupDirectUUIDGetterIgnoresSearchFiltersAndKeeps404`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaDirectGetAndEagerRuleDefaultsPreserveSourceBug`.
- Already fetched SecurityGroupRecord, RawResource or JSON dictionary with id present (including null/false/zero) is cloned through public pure ExistingSecurityGroupRead before backend discovery or user callbacks; Connection uses the same helper and does not invent transport proof.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupExistingReferenceBypassesFeatureDiscoveryAndCallbacks`, `connectiontest/connection_security_group_records_test.go:TestConnectionSecurityGroupExistingReferenceBypassesDiscoveryAndOptions`.

Python/Go differences:

- Go returns SecurityGroupReadResult with actual backend/Inventory/LastGroup/NovaResponse evidence and independent Groups/Group/Values/Value. JMES arbitrary JSON has Expression=true and no invented record correlation; Nova logical normalized Munch view is distinguished from its actual Nova Wire, not claimed to be a Neutron class instance.
- Go exposes concrete SecurityGroupReadDependencies and typed transport/current-location options, with owned resets and source invariants instead of developer interfaces/builders. It restricts transport headers/version/BasePath/route formatting/max_items and finite JSON input to safe Go representations; explicit Source policy flags are factory data while Connection supplies default backend/location without expanding global configuration scope.
- Go preserves the Source minimum conversion and maximum-port string normalization bug, eager pop/default behavior, source casing and strict compatibility. Nova proxy availability is conditional on actual deployment/API/microversion; modern Nova 2.36 proxy404 is not represented as universally supported fallback.
- Go lexical dict-key/JMESPath key-function order, omitted deprecation/log warnings, escaped service-relative routes, fixed request guards and stronger paging/cycle/Link safety are documented differences. Python arbitrary object/attribute/repr addresses, mutable Resource/dirty/session/cache and lazy materialization behavior remain SDK-R1/C1/S1 generic scope, separate from authenticated live cloud acceptance.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import os
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default=os.getenv("OS_CLOUD"))
parser.add_argument("--mode", choices=("search", "get"), default="search")
parser.add_argument("--name", default="")
parser.add_argument("--filters", default="null")
parser.add_argument("--existing", default="")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    filters = json.loads(args.filters)
    if args.mode == "get":
        reference = json.loads(args.existing) if args.existing else args.name
        result = conn.get_security_group(reference, filters)
    else:
        result = conn.search_security_groups(args.name or None, filters)
    if hasattr(result, "to_dict"):
        result = result.to_dict()
    elif isinstance(result, list):
        result = [value.to_dict() if hasattr(value, "to_dict") else value
                  for value in result]
    print(json.dumps(result, ensure_ascii=False, indent=2))
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
func main(){
 cloud:=flag.String("cloud",os.Getenv("OS_CLOUD"),"clouds.yaml cloud")
 mode:=flag.String("mode","search","search or get")
 name:=flag.String("name","","name, ID or fnmatch pattern")
 filters:=flag.String("filters","null","raw JSON filters; quote a JMESPath string")
 existing:=flag.String("existing","","already fetched JSON object containing id")
 flag.Parse()
 ctx,cancel:=context.WithTimeout(context.Background(),30*time.Second);defer cancel()
 var connect []sdk.ConnectionOption;if *cloud!=""{connect=append(connect,sdk.WithCloud(*cloud))}
 conn,err:=sdk.Connect(ctx,connect...);if err!=nil{log.Fatal(err)}
 options:=[]network.SecurityGroupReadOption{network.WithSecurityGroupReadFiltersJSON(json.RawMessage(*filters))}
 var result *network.SecurityGroupReadResult
 switch *mode{
 case "search":result,err=conn.SearchSecurityGroupRecords(ctx,*name,options...)
 case "get":var reference any=*name;if *existing!=""{reference=json.RawMessage(*existing)};result,err=conn.GetSecurityGroupRecord(ctx,reference,options...)
 default:log.Fatal("mode must be search or get")
 }
 if err!=nil{if result!=nil{fmt.Fprintf(os.Stderr,"backend=%s observed=%d\n",result.Backend,len(result.Inventory))};log.Fatal(err)}
 if result.Value==nil{fmt.Println("null");return};fmt.Println(string(result.Value))
}
```

## References

- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1834)
- [API Microversion History](https://docs.openstack.org/nova/2026.1/reference/api-microversion-history.html)
- [Secgroups](../compute/v2/secgroups/README.md)
- [Security Group](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/security_group.py)
- [Cloud Security Group Reads Test](cloud_security_group_reads_test.go)
- [Python List Proxy Test](v2/extensions/security/groups/python_list_proxy_test.go)
- [Connection Security Group Records Test](../connectiontest/connection_security_group_records_test.go)
