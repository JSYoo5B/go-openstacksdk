# Cloud Firewall Deletes Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/delete_firewall_rule`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.DeleteFirewallRuleRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteCurrentLocation`

- The strict lookup of delete_firewall_rule sends the filters on GET /fwaas/firewall_rules/{name_or_id} and on the name list and reports a missing resource as resource.ErrNotFound before any DELETE.
  Tests: `api/find_identity_python_finds_test.go:TestPythonCloudGetFirewallForwardsFiltersToBothRequests`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- Library-owned concrete JSON/functional options snapshot raw filters, headers, version and location. Falsey filters become empty kwargs, truthy filters require mapping despite the historical JMES docstring; bound name_or_id/ignore_missing collide by presence before HTTP. No caller builder/interface or extension/has_service gate is needed.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- A closed Source profile reuses Engine.Find with strict ignore_missing=False: initial member GET, clean400/403/404 complete list fallback, declared query versus local Body filters and exact original ID/name match; unknown kwargs reach GET only and explicit name overrides the server hint. Full fallback duplicate/late error blocks DELETE. It uses plain Resource fields without inherited NetworkResource Header/tag defaults.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteMissingStrictLookupAndFallbackErrors`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- Found actual Resource ID selects one strict bodyless DELETE. Clean missing Find, clean LIST404 and clean actual DELETE404 return Deleted=false; normal accepted100..399 DELETE returns true regardless of empty/non-JSON body. Mutation Resource/Wire are not fabricated; lookup and deletion actual envelope/header/status remain separate. Conflicts and non404 rejections remain errors.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteStrictFindThenActualIDAndDelete404IsFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteMissingStrictLookupAndFallbackErrors`.
- Source fetch accepted204/empty or200/malformed response keeps the original requested ID seed and can proceed to DELETE; explicit returned null ID errors before DELETE. Profile SourceStatusPolicy allows100..399, cloned SourceLinkPolicy and PythonLoadedEquality leave old native/generated paths unchanged. Untyped fields remain raw, Group.ports/Policy.firewall_rules wrap scalar list values, Rule Computed.summary reads pinned default while raw summary remains only in Wire.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteAcceptedEmptyLookupAndFalseyReturnedIDBoundaries`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteStrictFindThenActualIDAndDelete404IsFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- Physical accepted200/rejected401/404/503 GET and DELETE faults preserve actual full bytes/header/status and no successful Resource. Three-family×two-method×four-status×Read+Close/cancel/source72lane fixture closes once, invokes no native retry/reauth hook and keeps custom context cause. Native401 pointer does not unwrap: actual value ErrOriginal and ErrReauth read/close causes are checked independently while retaining any outer joined causes.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeletePhysicalFaultsAreNeverMissingAndKeepActualProof`.
- Clean missing classification delegates through the existing resource Collection Get/Delete policies and accepts only a single clean missing chain. Joined terminal callback/physical/source/context errors cannot become false merely because they contain404. A callback-fabricated404 over actual503 remains error; selected original client/API/route facts are guarded around cloned option callbacks and physical operations.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteOptionsSourceGuardsAndFabricated404NeverFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeletePhysicalFaultsAreNeverMissingAndKeepActualProof`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- SDK-R1: Go returns bool plus owned Lookup/Delete receipt views instead of mutable Python Resource/cache/lifecycle objects. All declared getter fields are projected before return, so an unused malformed field may fail earlier. Source informational status and empty/malformed fetch seed behavior are implemented only on this new closed profile; legacy typed/generated APIs keep their old contracts.
- SDK-C1: JSON bytes retain large integers and Source untyped values without native bool/int coercion. Local equality uses shared CPython-loaded numeric policy and exact raw Wire/Envelope proof. Non-JSON custom objects and unrestricted Python runtime/representation/marshal behaviors remain separately tracked; structured Go errors keep actual native status and physical/context causes.
- SDK-S1: Go captures the original Neutron service/client/route and uses safe escaped literal member IDs with fixed protocol/header/origin guards. Arbitrary adapter/class retargeting and URI percent-format are restricted. WithFirewallDeleteHeaders/Microversion are explicit Go transport extras (headers across phases, microversion on initial GET+DELETE while fallback LIST keeps captured default), not a claim that the unimplemented raw Source Find controls are equivalent.
- Source has no FWaaS extension/has_service pre-gate for these deletes, and neither does this workflow. Shared-reader access is not permission to delete: official FWaaS2025.2 sample gives owner project member/admin ordinary delete and separate admin-only shared deletion. SDK does not elevate RBAC or unlink attached rules/policies/groups; backend conflict remains actual409.
- Actual pinned Source156cases, standalone Python AST/byte-exact Go+Python mains and prepared fixture groups are distinct evidence. Owner did not execute Go/Git or authenticate a live cloud. New deletion bool/receipt closure does not remove the reachable rawFind/falsey collection/paging remaining below.

Unresolved boundaries:

- Raw Find kwargs remain incomplete in the common engine: actual Source filters.headers is an ordinary initial GET query then a fallback LIST formal; raw filters.microversion overrides initial GET only while fallback LIST and DELETE return to adapter default. Current raw filters with these names and reserved fetch/constructor controls such as base_path/skip_cache/resource_response_key reject before HTTP rather than reproduce their Source phase-dependent behavior. Concrete Go transport extras are distinct and do not close this named gap.
- Actual Source can DELETE the collection route when a found Resource.id is false/0/empty string/array/object. Current safe member-only workflow rejects these falsey nonnull values before DELETE. Explicit null/resolved None ID or a fallback LIST row with no ID has Source InvalidRequest, whereas an initial fetched response omitting its id member retains the requested ID seed and can DELETE that ID. This is a concrete reachable returned-JSON-ID gap, not generic runtime object injection.
- Find fallback inherits strict Go integer limit/max_items and guarded marker/continuation policy. Source bool/negative/fractional caps/limits, their error phases and a following GET when the cap equals a whole page are not fully reproduced. That can change when later duplicate/HTTP/physical failure is observed and whether DELETE begins; no full named parity is claimed.

### `python:cloud/_network/NetworkCloudMixin/delete_firewall_policy`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.DeleteFirewallPolicyRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteCurrentLocation`

- The strict lookup of delete_firewall_policy sends the filters on GET /fwaas/firewall_policies/{name_or_id} and on the name list and reports a missing resource as resource.ErrNotFound before any DELETE.
  Tests: `api/find_identity_python_finds_test.go:TestPythonCloudGetFirewallForwardsFiltersToBothRequests`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- Library-owned concrete JSON/functional options snapshot raw filters, headers, version and location. Falsey filters become empty kwargs, truthy filters require mapping despite the historical JMES docstring; bound name_or_id/ignore_missing collide by presence before HTTP. No caller builder/interface or extension/has_service gate is needed.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- A closed Source profile reuses Engine.Find with strict ignore_missing=False: initial member GET, clean400/403/404 complete list fallback, declared query versus local Body filters and exact original ID/name match; unknown kwargs reach GET only and explicit name overrides the server hint. Full fallback duplicate/late error blocks DELETE. It uses plain Resource fields without inherited NetworkResource Header/tag defaults.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteMissingStrictLookupAndFallbackErrors`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- Found actual Resource ID selects one strict bodyless DELETE. Clean missing Find, clean LIST404 and clean actual DELETE404 return Deleted=false; normal accepted100..399 DELETE returns true regardless of empty/non-JSON body. Mutation Resource/Wire are not fabricated; lookup and deletion actual envelope/header/status remain separate. Conflicts and non404 rejections remain errors.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteStrictFindThenActualIDAndDelete404IsFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteMissingStrictLookupAndFallbackErrors`.
- Source fetch accepted204/empty or200/malformed response keeps the original requested ID seed and can proceed to DELETE; explicit returned null ID errors before DELETE. Profile SourceStatusPolicy allows100..399, cloned SourceLinkPolicy and PythonLoadedEquality leave old native/generated paths unchanged. Untyped fields remain raw, Group.ports/Policy.firewall_rules wrap scalar list values, Rule Computed.summary reads pinned default while raw summary remains only in Wire.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteAcceptedEmptyLookupAndFalseyReturnedIDBoundaries`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteStrictFindThenActualIDAndDelete404IsFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- Physical accepted200/rejected401/404/503 GET and DELETE faults preserve actual full bytes/header/status and no successful Resource. Three-family×two-method×four-status×Read+Close/cancel/source72lane fixture closes once, invokes no native retry/reauth hook and keeps custom context cause. Native401 pointer does not unwrap: actual value ErrOriginal and ErrReauth read/close causes are checked independently while retaining any outer joined causes.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeletePhysicalFaultsAreNeverMissingAndKeepActualProof`.
- Clean missing classification delegates through the existing resource Collection Get/Delete policies and accepts only a single clean missing chain. Joined terminal callback/physical/source/context errors cannot become false merely because they contain404. A callback-fabricated404 over actual503 remains error; selected original client/API/route facts are guarded around cloned option callbacks and physical operations.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteOptionsSourceGuardsAndFabricated404NeverFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeletePhysicalFaultsAreNeverMissingAndKeepActualProof`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- SDK-R1: Go returns bool plus owned Lookup/Delete receipt views instead of mutable Python Resource/cache/lifecycle objects. All declared getter fields are projected before return, so an unused malformed field may fail earlier. Source informational status and empty/malformed fetch seed behavior are implemented only on this new closed profile; legacy typed/generated APIs keep their old contracts.
- SDK-C1: JSON bytes retain large integers and Source untyped values without native bool/int coercion. Local equality uses shared CPython-loaded numeric policy and exact raw Wire/Envelope proof. Non-JSON custom objects and unrestricted Python runtime/representation/marshal behaviors remain separately tracked; structured Go errors keep actual native status and physical/context causes.
- SDK-S1: Go captures the original Neutron service/client/route and uses safe escaped literal member IDs with fixed protocol/header/origin guards. Arbitrary adapter/class retargeting and URI percent-format are restricted. WithFirewallDeleteHeaders/Microversion are explicit Go transport extras (headers across phases, microversion on initial GET+DELETE while fallback LIST keeps captured default), not a claim that the unimplemented raw Source Find controls are equivalent.
- Source has no FWaaS extension/has_service pre-gate for these deletes, and neither does this workflow. Shared-reader access is not permission to delete: official FWaaS2025.2 sample gives owner project member/admin ordinary delete and separate admin-only shared deletion. SDK does not elevate RBAC or unlink attached rules/policies/groups; backend conflict remains actual409.
- Actual pinned Source156cases, standalone Python AST/byte-exact Go+Python mains and prepared fixture groups are distinct evidence. Owner did not execute Go/Git or authenticate a live cloud. New deletion bool/receipt closure does not remove the reachable rawFind/falsey collection/paging remaining below.

Unresolved boundaries:

- Raw Find kwargs remain incomplete in the common engine: actual Source filters.headers is an ordinary initial GET query then a fallback LIST formal; raw filters.microversion overrides initial GET only while fallback LIST and DELETE return to adapter default. Current raw filters with these names and reserved fetch/constructor controls such as base_path/skip_cache/resource_response_key reject before HTTP rather than reproduce their Source phase-dependent behavior. Concrete Go transport extras are distinct and do not close this named gap.
- Actual Source can DELETE the collection route when a found Resource.id is false/0/empty string/array/object. Current safe member-only workflow rejects these falsey nonnull values before DELETE. Explicit null/resolved None ID or a fallback LIST row with no ID has Source InvalidRequest, whereas an initial fetched response omitting its id member retains the requested ID seed and can DELETE that ID. This is a concrete reachable returned-JSON-ID gap, not generic runtime object injection.
- Find fallback inherits strict Go integer limit/max_items and guarded marker/continuation policy. Source bool/negative/fractional caps/limits, their error phases and a following GET when the cap equals a whole page are not fully reproduced. That can change when later duplicate/HTTP/physical failure is observed and whether DELETE begins; no full named parity is claimed.

### `python:cloud/_network/NetworkCloudMixin/delete_firewall_group`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.DeleteFirewallGroupRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallDeleteCurrentLocation`

- The strict lookup of delete_firewall_group sends the filters on GET /fwaas/firewall_groups/{name_or_id} and on the name list and reports a missing resource as resource.ErrNotFound before any DELETE.
  Tests: `api/find_identity_python_finds_test.go:TestPythonCloudGetFirewallForwardsFiltersToBothRequests`, `api/find_identity_python_finds_test.go:TestPythonFindMissingDefaultsToNilAndStrictIsNotFound`.
- Library-owned concrete JSON/functional options snapshot raw filters, headers, version and location. Falsey filters become empty kwargs, truthy filters require mapping despite the historical JMES docstring; bound name_or_id/ignore_missing collide by presence before HTTP. No caller builder/interface or extension/has_service gate is needed.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- A closed Source profile reuses Engine.Find with strict ignore_missing=False: initial member GET, clean400/403/404 complete list fallback, declared query versus local Body filters and exact original ID/name match; unknown kwargs reach GET only and explicit name overrides the server hint. Full fallback duplicate/late error blocks DELETE. It uses plain Resource fields without inherited NetworkResource Header/tag defaults.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteMissingStrictLookupAndFallbackErrors`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- Found actual Resource ID selects one strict bodyless DELETE. Clean missing Find, clean LIST404 and clean actual DELETE404 return Deleted=false; normal accepted100..399 DELETE returns true regardless of empty/non-JSON body. Mutation Resource/Wire are not fabricated; lookup and deletion actual envelope/header/status remain separate. Conflicts and non404 rejections remain errors.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteStrictFindThenActualIDAndDelete404IsFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteMissingStrictLookupAndFallbackErrors`.
- Source fetch accepted204/empty or200/malformed response keeps the original requested ID seed and can proceed to DELETE; explicit returned null ID errors before DELETE. Profile SourceStatusPolicy allows100..399, cloned SourceLinkPolicy and PythonLoadedEquality leave old native/generated paths unchanged. Untyped fields remain raw, Group.ports/Policy.firewall_rules wrap scalar list values, Rule Computed.summary reads pinned default while raw summary remains only in Wire.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteAcceptedEmptyLookupAndFalseyReturnedIDBoundaries`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteStrictFindThenActualIDAndDelete404IsFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteFiltersOwnJSONAndUntypedLocalProjection`.
- Physical accepted200/rejected401/404/503 GET and DELETE faults preserve actual full bytes/header/status and no successful Resource. Three-family×two-method×four-status×Read+Close/cancel/source72lane fixture closes once, invokes no native retry/reauth hook and keeps custom context cause. Native401 pointer does not unwrap: actual value ErrOriginal and ErrReauth read/close causes are checked independently while retaining any outer joined causes.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeletePhysicalFaultsAreNeverMissingAndKeepActualProof`.
- Clean missing classification delegates through the existing resource Collection Get/Delete policies and accepts only a single clean missing chain. Joined terminal callback/physical/source/context errors cannot become false merely because they contain404. A callback-fabricated404 over actual503 remains error; selected original client/API/route facts are guarded around cloned option callbacks and physical operations.
  Tests: `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeleteOptionsSourceGuardsAndFabricated404NeverFalse`, `network/cloud_firewall_deletes_test.go:TestCloudFirewallDeletePhysicalFaultsAreNeverMissingAndKeepActualProof`.

Python/Go differences:

- Python returns None for a missing resource and raises NotFoundException with ignore_missing=False; Go FindIdentity returns nil, nil by default and resource.ErrNotFound with resource.WithIdentityFindIgnoreMissing(false).
- Python raises DuplicateResource for two matching rows; Go returns resource.ErrAmbiguous after reading every list page.
- Identities that are not safe as one URL segment (spaces, slashes, percent signs) skip the member GET in Go and go straight to the list fallback; Python sends them URL-encoded on the GET first.
- Python returns a Resource object; Go returns the native typed model.
- SDK-R1: Go returns bool plus owned Lookup/Delete receipt views instead of mutable Python Resource/cache/lifecycle objects. All declared getter fields are projected before return, so an unused malformed field may fail earlier. Source informational status and empty/malformed fetch seed behavior are implemented only on this new closed profile; legacy typed/generated APIs keep their old contracts.
- SDK-C1: JSON bytes retain large integers and Source untyped values without native bool/int coercion. Local equality uses shared CPython-loaded numeric policy and exact raw Wire/Envelope proof. Non-JSON custom objects and unrestricted Python runtime/representation/marshal behaviors remain separately tracked; structured Go errors keep actual native status and physical/context causes.
- SDK-S1: Go captures the original Neutron service/client/route and uses safe escaped literal member IDs with fixed protocol/header/origin guards. Arbitrary adapter/class retargeting and URI percent-format are restricted. WithFirewallDeleteHeaders/Microversion are explicit Go transport extras (headers across phases, microversion on initial GET+DELETE while fallback LIST keeps captured default), not a claim that the unimplemented raw Source Find controls are equivalent.
- Source has no FWaaS extension/has_service pre-gate for these deletes, and neither does this workflow. Shared-reader access is not permission to delete: official FWaaS2025.2 sample gives owner project member/admin ordinary delete and separate admin-only shared deletion. SDK does not elevate RBAC or unlink attached rules/policies/groups; backend conflict remains actual409.
- Actual pinned Source156cases, standalone Python AST/byte-exact Go+Python mains and prepared fixture groups are distinct evidence. Owner did not execute Go/Git or authenticate a live cloud. New deletion bool/receipt closure does not remove the reachable rawFind/falsey collection/paging remaining below.

Unresolved boundaries:

- Raw Find kwargs remain incomplete in the common engine: actual Source filters.headers is an ordinary initial GET query then a fallback LIST formal; raw filters.microversion overrides initial GET only while fallback LIST and DELETE return to adapter default. Current raw filters with these names and reserved fetch/constructor controls such as base_path/skip_cache/resource_response_key reject before HTTP rather than reproduce their Source phase-dependent behavior. Concrete Go transport extras are distinct and do not close this named gap.
- Actual Source can DELETE the collection route when a found Resource.id is false/0/empty string/array/object. Current safe member-only workflow rejects these falsey nonnull values before DELETE. Explicit null/resolved None ID or a fallback LIST row with no ID has Source InvalidRequest, whereas an initial fetched response omitting its id member retains the requested ID seed and can DELETE that ID. This is a concrete reachable returned-JSON-ID gap, not generic runtime object injection.
- Find fallback inherits strict Go integer limit/max_items and guarded marker/continuation policy. Source bool/negative/fractional caps/limits, their error phases and a following GET when the cap equals a whole page are not fully reproduced. That can change when later duplicate/HTTP/physical failure is observed and whether DELETE begins; no full named parity is claimed.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--kind", choices=("rule", "policy", "group"), default="rule")
parser.add_argument("--name", required=True)
parser.add_argument("--filters", default="{}")
args = parser.parse_args()
filters = json.loads(args.filters)
with openstack.connect(cloud=args.cloud) as conn:
    if args.kind == "rule":
        result = conn.delete_firewall_rule(args.name, filters=filters)
    elif args.kind == "policy":
        result = conn.delete_firewall_policy(args.name, filters=filters)
    else:
        result = conn.delete_firewall_group(args.name, filters=filters)
    print(json.dumps(result))
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
	"github.com/JSYoo5B/go-openstacksdk/network"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
	kind := flag.String("kind", "rule", "rule, policy or group")
	name := flag.String("name", "", "literal resource ID or exact name to delete")
	filters := flag.String("filters", "{}", "JSON Source Find kwargs")
	flag.Parse()
	if *name == "" {
		log.Fatal("name is required")
	}
	if *kind != "rule" && *kind != "policy" && *kind != "group" {
		log.Fatal("unknown kind")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.Network(ctx)
	if err != nil {
		log.Fatal(err)
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		log.Fatal(err)
	}
	opts := []network.FirewallDeleteOption{network.WithFirewallDeleteFilters(json.RawMessage(*filters)), network.WithFirewallDeleteCurrentLocation(location)}
	var result *network.FirewallDeletion
	switch *kind {
	case "rule":
		result, err = service.DeleteFirewallRuleRecord(ctx, *name, opts...)
	case "policy":
		result, err = service.DeleteFirewallPolicyRecord(ctx, *name, opts...)
	case "group":
		result, err = service.DeleteFirewallGroupRecord(ctx, *name, opts...)
	}
	if err != nil {
		log.Fatal(err)
	}
	value, err := json.Marshal(result.Deleted)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(value))
}
```

## References

- [Policy Sample](https://docs.openstack.org/neutron-fwaas/2025.2/configuration/policy-sample.html)
