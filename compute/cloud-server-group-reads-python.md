# Cloud Server Group Reads Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_compute/ComputeCloudMixin/search_server_groups`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.SearchServerGroups`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetServerGroup`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudFilters`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudExpression`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudHeaders`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.ListCloudRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.FindCloudRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SearchServerGroups`

- Search consumes every default-project page before exact/glob name/ID and ordered local mapping/JMES filters, without name/filter/all-projects pushdown. Canonical member_ids is used for builtin dict cloud filters; wire members and unknown attrs are not rewritten to valid keys. Source untyped fields stay raw, rules alone gets dictionary getter cast.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsSearchExactGlobOrderedDictCanonicalAndNoPushdown`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsCompleteSearchAndDefaultFindStopsSecondDuplicate`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsListDescriptorsAliasesUnknownStatusAndVersion`.
- Absent/null filters use default find; every nonnull value including false/0/empty dict/list/string enters deprecated complete search. Filtered marker exposes the branch. Primitive JMES results are preserved in Value, not invented as a Resource; filtered get follows loaded truthiness, len, index0, retaining false/zero/empty selected elements and nil only for selected null.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsGetPresenceDefaultFindVersusDeprecatedSearch`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsJMESArbitraryJSONAndFilteredGetIndexSemantics`.
- Default find member GET falls back only for clean400/403/404; exact ID/name equality has no fnmatch conversion. Missing defaults nil. It stops on second exact match before a later HTTP page, while filtered get fully consumes later failures before ambiguity. Observed member and consumed fallback Inventory retain separate actual proof.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsDefaultExactNoGlobMissingEmptyCollectionAndNativeErrors`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsCompleteSearchAndDefaultFindStopsSecondDuplicate`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsFindEarlyDuplicateEmptyTrailingAndLateFailure`.
- Actual Source empty string is supported as fixed trailing-slash collection GET with an empty ID seed; it is not collapsed with Source None InvalidRequest. Empty fallback never treats decoded null IDs/names as literal empty strings. Other IDs use owned literal path segments with explicit safe Go constraints.
  Tests: `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsFindEarlyDuplicateEmptyTrailingAndLateFailure`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsEmptyExactFallbackNeverMatchesNullAsEmptyString`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsDiscoveryCeilingSelectedSessionAndLiteralSegments`.
- Selected Nova microversion has precedence; unspecified discovery caps against ServerGroup2.64/min/max using existing shared selection and new strict physical fault opt-in. New member/list Source100..399 leaves old native and FindRecord baselines unchanged. Actual records/default views and Wire/whole Envelope/Header/Status remain owned; CurrentLocation never enters query and foreign project names clear.
  Tests: `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsDiscoveryCeilingSelectedSessionAndLiteralSegments`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsListDescriptorsAliasesUnknownStatusAndVersion`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsOwnsOptionsRecordsHeadersAndCurrentLocation`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsCleanRetryReauthStrictStatusAndHeaderCapture`.
- Captured service/client/group binding and option/header/location snapshots precede callbacks. Nil malformed options/context/headers/location and custom cancellation fail preflight. Mutable callback snapshots cannot alter later applied filters/headers or stored response evidence.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsOwnsOptionsRecordsHeadersAndCurrentLocation`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsPreflightSelectedAPIOptionsAndCustomCancellation`.
- Physical read/Close/cancel/restored source faults in search/member/discovery for200/401/404/429/503 preserve actual proof and prevent user retry/reauth/backoff, fallback or missing. Native401 value ErrOriginal/ErrReauth remain available. Clean503 retry and401 token renewal remain guarded. OkCodes expansion cannot establish success or404 absence under the original policy.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsPhysicalFaultsCannotFallbackRetryOrBecomeMissing`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsCleanRetryReauthStrictStatusAndHeaderCapture`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsExpandedRejected404AndPhysicalFailureDoNotFallback`.
- New Source opt-in closes finite JSON float loading in local mapping, loaded falsey first selection and exact binary finite operands for JMES comparisons without rounding arbitrary integers. Source two-float9007199254740992.0/9007199254740993.0 comparison is supported; this does not claim full arithmetic, serialization, nonfinite or Resource class equality/function namespace.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsLoadedNumericMappingAndJMESPredicate`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsJMESArbitraryJSONAndFilteredGetIndexSemantics`.
- Connection exposes this cloud read through its cached service and SDK-owned location defaults. Cancellation is checked before catalog discovery. Only Neutron list helpers turn clean missing catalog endpoints into [] before evaluating options; searches retain catalog errors. Caller-provided options retain their explicit values.
  Tests: `connection_cycle8_reads_test.go:TestConnectionCycle8CancelledReadsNeverDiscover`.

Python/Go differences:

- Concrete library-owned Go options/carriers expose Value plus actual Groups/Group/Observed/Inventory instead of only mutable Python Resource/lists. JMES arbitrary output has no fabricated row association. Filtered marks deprecated get-search selection instead of Python warnings.warn side effects.
- Source None name is not representable by the required Go string argument: Source get(None) raisesInvalidRequest without HTTP, while explicit Go empty string is supported as the actual Source trailing collection GET. Search empty is equivalent to Source absent identifier. Safe literal ID/control/traversal/whitespace policy is explicit and never silently replaces an ID.
- Owned header/version/location snapshots, complete UTF8 JSON and JSON object wire order are Go policy. Existing mutation/API caches and native read/find baselines remain intact. Custom runtime objects/mutable Resource/class/adapter/cache/session/current_location foundation differences remain SDK-R1/SDK-C1/SDK-S1 with supported increment0, separately from the two reachable named JMES gaps retained above.
- Nova32 index/show policy is project PROJECT_READER_OR_ADMIN; all_projects index requires ADMIN. Cloud methods keep all_projectsFalse and do not expand privileges. Runtime/mock/race/main proof is not actual authenticated backend, role, ownership or deployment-policy acceptance.
- Actual imported pinned cloud search/get/list→Nova Proxy→Resource list/find/constructor/descriptor→cloudfilter/JMESPath ran under Python3.14.8 in core114 + class/numeric/empty25 + arithmetic8 =147 cases. Only Adapter HTTP/get_endpoint_data/connection were fixture substitutions. Source files and supplemental references are pinned, with no live backend. Go validation covers 15 basic groups; unresolved named class/numeric namespaces prevent completed/full-supported promotion.
- Source None name is not representable by the required Go string argument: Source get(None) raisesInvalidRequest without HTTP, while explicit Go empty string is supported as the actual Source trailing collection GET. Search empty is equivalent to Source absent identifier. Safe literal ID/control/traversal/whitespace policy is explicit and never silently replaces an ID. Literal db0* member GET uses Go single-segment db0%2A encoding versus Source Requests retained star; both default find paths use exact matching, not a glob.

Unresolved boundaries:

- Named JMESPath Source ServerGroup subclass typing/equality remains unresolved. Actual pinned Source type([0]) is object but keys([0])/values([0])/length([0]) raise JMESPathTypeError unknown/ServerGroup; Go JSON map loses that concrete validator. Resource.__eq__ is false against an otherwise identical plain canonical10field object, so Source [?@ == identical_literal].id is [] and filtered get None, whereas structural Go maps can match. Nested rules/metadata dicts are distinct from the Resource class. Actual147-case reference includes these reachable operations; they are not hidden as only generic SDK-R1.
- Named full Python numeric/arithmetic/serialization/nonfinite JMES namespace remains unresolved. Finite binary-loaded comparison and mixed int/float predicate handling are implemented, but arithmetic/serialization retain the existing exact-number engine rather than a complete CPython typed numeric model. Actual Source sum(`[9007199254740992.0,1]`) returns9007199254740992.0; to_string(`1e23`) returns"1e+23". Source literal1e400 search returns inf and filtered get raisesTypeError, while Go loaded JSON rejects nonfinite as unsupported. These named paths stay open, not a claim of full filter parity.

### `python:cloud/_compute/ComputeCloudMixin/get_server_group`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.SearchServerGroups`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetServerGroup`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudFilters`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudExpression`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudHeaders`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroupCloudCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.ListCloudRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servergroups.API.FindCloudRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetServerGroup`

- Without filters get_server_group(name_or_id) is find_server_group, and FindRecord sends the same GET os-server-groups/{id} with the list fallback.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudServerGroupHelpers`, `compute/v2/servergroups/python_find_contracts_test.go:TestPythonGroupFindRecordUsesGETThenExactIDOrNameAndAllProjects`.
- Search consumes every default-project page before exact/glob name/ID and ordered local mapping/JMES filters, without name/filter/all-projects pushdown. Canonical member_ids is used for builtin dict cloud filters; wire members and unknown attrs are not rewritten to valid keys. Source untyped fields stay raw, rules alone gets dictionary getter cast.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsSearchExactGlobOrderedDictCanonicalAndNoPushdown`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsCompleteSearchAndDefaultFindStopsSecondDuplicate`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsListDescriptorsAliasesUnknownStatusAndVersion`.
- Absent/null filters use default find; every nonnull value including false/0/empty dict/list/string enters deprecated complete search. Filtered marker exposes the branch. Primitive JMES results are preserved in Value, not invented as a Resource; filtered get follows loaded truthiness, len, index0, retaining false/zero/empty selected elements and nil only for selected null.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsGetPresenceDefaultFindVersusDeprecatedSearch`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsJMESArbitraryJSONAndFilteredGetIndexSemantics`.
- Default find member GET falls back only for clean400/403/404; exact ID/name equality has no fnmatch conversion. Missing defaults nil. It stops on second exact match before a later HTTP page, while filtered get fully consumes later failures before ambiguity. Observed member and consumed fallback Inventory retain separate actual proof.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsDefaultExactNoGlobMissingEmptyCollectionAndNativeErrors`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsCompleteSearchAndDefaultFindStopsSecondDuplicate`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsFindEarlyDuplicateEmptyTrailingAndLateFailure`.
- Actual Source empty string is supported as fixed trailing-slash collection GET with an empty ID seed; it is not collapsed with Source None InvalidRequest. Empty fallback never treats decoded null IDs/names as literal empty strings. Other IDs use owned literal path segments with explicit safe Go constraints.
  Tests: `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsFindEarlyDuplicateEmptyTrailingAndLateFailure`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsEmptyExactFallbackNeverMatchesNullAsEmptyString`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsDiscoveryCeilingSelectedSessionAndLiteralSegments`.
- Selected Nova microversion has precedence; unspecified discovery caps against ServerGroup2.64/min/max using existing shared selection and new strict physical fault opt-in. New member/list Source100..399 leaves old native and FindRecord baselines unchanged. Actual records/default views and Wire/whole Envelope/Header/Status remain owned; CurrentLocation never enters query and foreign project names clear.
  Tests: `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsDiscoveryCeilingSelectedSessionAndLiteralSegments`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsListDescriptorsAliasesUnknownStatusAndVersion`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsOwnsOptionsRecordsHeadersAndCurrentLocation`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsCleanRetryReauthStrictStatusAndHeaderCapture`.
- Captured service/client/group binding and option/header/location snapshots precede callbacks. Nil malformed options/context/headers/location and custom cancellation fail preflight. Mutable callback snapshots cannot alter later applied filters/headers or stored response evidence.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsOwnsOptionsRecordsHeadersAndCurrentLocation`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsPreflightSelectedAPIOptionsAndCustomCancellation`.
- Physical read/Close/cancel/restored source faults in search/member/discovery for200/401/404/429/503 preserve actual proof and prevent user retry/reauth/backoff, fallback or missing. Native401 value ErrOriginal/ErrReauth remain available. Clean503 retry and401 token renewal remain guarded. OkCodes expansion cannot establish success or404 absence under the original policy.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsPhysicalFaultsCannotFallbackRetryOrBecomeMissing`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsCleanRetryReauthStrictStatusAndHeaderCapture`, `compute/v2/servergroups/python_cloud_reads_test.go:TestSourceCloudGroupReadsExpandedRejected404AndPhysicalFailureDoNotFallback`.
- New Source opt-in closes finite JSON float loading in local mapping, loaded falsey first selection and exact binary finite operands for JMES comparisons without rounding arbitrary integers. Source two-float9007199254740992.0/9007199254740993.0 comparison is supported; this does not claim full arithmetic, serialization, nonfinite or Resource class equality/function namespace.
  Tests: `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsLoadedNumericMappingAndJMESPredicate`, `compute/cloud_server_group_reads_test.go:TestCloudServerGroupReadsJMESArbitraryJSONAndFilteredGetIndexSemantics`.
- Connection exposes this cloud read through its cached service and SDK-owned location defaults. Cancellation is checked before catalog discovery. Only Neutron list helpers turn clean missing catalog endpoints into [] before evaluating options; searches retain catalog errors. Caller-provided options retain their explicit values.
  Tests: `connection_cycle8_reads_test.go:TestConnectionCycle8CancelledReadsNeverDiscover`.

Python/Go differences:

- Concrete library-owned Go options/carriers expose Value plus actual Groups/Group/Observed/Inventory instead of only mutable Python Resource/lists. JMES arbitrary output has no fabricated row association. Filtered marks deprecated get-search selection instead of Python warnings.warn side effects.
- Source None name is not representable by the required Go string argument: Source get(None) raisesInvalidRequest without HTTP, while explicit Go empty string is supported as the actual Source trailing collection GET. Search empty is equivalent to Source absent identifier. Safe literal ID/control/traversal/whitespace policy is explicit and never silently replaces an ID.
- Owned header/version/location snapshots, complete UTF8 JSON and JSON object wire order are Go policy. Existing mutation/API caches and native read/find baselines remain intact. Custom runtime objects/mutable Resource/class/adapter/cache/session/current_location foundation differences remain SDK-R1/SDK-C1/SDK-S1 with supported increment0, separately from the two reachable named JMES gaps retained above.
- Nova32 index/show policy is project PROJECT_READER_OR_ADMIN; all_projects index requires ADMIN. Cloud methods keep all_projectsFalse and do not expand privileges. Runtime/mock/race/main proof is not actual authenticated backend, role, ownership or deployment-policy acceptance.
- Actual imported pinned cloud search/get/list→Nova Proxy→Resource list/find/constructor/descriptor→cloudfilter/JMESPath ran under Python3.14.8 in core114 + class/numeric/empty25 + arithmetic8 =147 cases. Only Adapter HTTP/get_endpoint_data/connection were fixture substitutions. Source files and supplemental references are pinned, with no live backend. Go validation covers 15 basic groups; unresolved named class/numeric namespaces prevent completed/full-supported promotion.
- Source None name is not representable by the required Go string argument: Source get(None) raisesInvalidRequest without HTTP, while explicit Go empty string is supported as the actual Source trailing collection GET. Search empty is equivalent to Source absent identifier. Safe literal ID/control/traversal/whitespace policy is explicit and never silently replaces an ID. Literal db0* member GET uses Go single-segment db0%2A encoding versus Source Requests retained star; both default find paths use exact matching, not a glob.

Unresolved boundaries:

- Named JMESPath Source ServerGroup subclass typing/equality remains unresolved. Actual pinned Source type([0]) is object but keys([0])/values([0])/length([0]) raise JMESPathTypeError unknown/ServerGroup; Go JSON map loses that concrete validator. Resource.__eq__ is false against an otherwise identical plain canonical10field object, so Source [?@ == identical_literal].id is [] and filtered get None, whereas structural Go maps can match. Nested rules/metadata dicts are distinct from the Resource class. Actual147-case reference includes these reachable operations; they are not hidden as only generic SDK-R1.
- Named full Python numeric/arithmetic/serialization/nonfinite JMES namespace remains unresolved. Finite binary-loaded comparison and mixed int/float predicate handling are implemented, but arithmetic/serialization retain the existing exact-number engine rather than a complete CPython typed numeric model. Actual Source sum(`[9007199254740992.0,1]`) returns9007199254740992.0; to_string(`1e23`) returns"1e+23". Source literal1e400 search returns inf and filtered get raisesTypeError, while Go loaded JSON rejects nonfinite as unsupported. These named paths stay open, not a claim of full filter parity.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--operation", choices=("search", "get"), default="search")
parser.add_argument("--name")
parser.add_argument("--filters", help="JSON dictionary, string expression or other Source value")
args = parser.parse_args()
kwargs = {}
if args.filters is not None:
    kwargs["filters"] = json.loads(args.filters)

def view(value):
    if hasattr(value, "to_dict"):
        return value.to_dict()
    if isinstance(value, list):
        return [view(item) for item in value]
    return value

with openstack.connect(cloud=args.cloud) as conn:
    if args.operation == "search":
        result = conn.search_server_groups(args.name, **kwargs)
    else:
        result = conn.get_server_group(args.name, **kwargs)
    print(json.dumps(view(result), indent=2))
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
	"github.com/JSYoo5B/go-openstacksdk/compute"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	operation := flag.String("operation", "search", "search or get")
	name := flag.String("name", "", "name, literal ID or search glob")
	filters := flag.String("filters", "", "complete JSON filter; omitted means None")
	flag.Parse()
	if *operation != "search" && *operation != "get" {
		log.Fatal("unknown operation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		log.Fatal(err)
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		log.Fatal(err)
	}
	options := []compute.ServerGroupCloudReadOption{compute.WithServerGroupCloudCurrentLocation(location)}
	if *filters != "" {
		options = append(options, compute.WithServerGroupCloudFilters(json.RawMessage(*filters)))
	}
	var value json.RawMessage
	if *operation == "search" {
		result, cause := service.SearchServerGroups(ctx, *name, options...)
		err = cause
		if result != nil {
			value = result.Value
		}
	} else {
		result, cause := service.GetServerGroup(ctx, *name, options...)
		err = cause
		if result != nil {
			value = result.Value
		}
	}
	if err != nil {
		log.Fatal(err)
	}
	if value == nil {
		value = json.RawMessage("null")
	}
	data, err := json.MarshalIndent(json.RawMessage(value), "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

## References

- [Install](../docs/install.md)
- [Server Groups](https://github.com/openstack/nova/blob/32.0.0/nova/policies/server_groups.py#L49-L85)
- [search](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L216-L242)
- [get](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L728-L772)
- [filter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L43-L145)
- [ServerGroup](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_group.py)
- [Resource.find/equality](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
