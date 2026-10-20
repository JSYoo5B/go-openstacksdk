# Cloud Project Group Get Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_identity/IdentityCloudMixin/get_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetGroupRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewGroupReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.GroupReadCloud.GetGroup`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/groups.API.ResolveRecord`

- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Default/None uses a closed domain_id-only Source finder: literal GET, only clean400403404 fallback with name/domain hints, exact ID/name, default nil missing, duplicate error, GET-only per-call MV and actual descriptor/Wire/page/header/status evidence. Caller options/domain/header/location bytes snapshot before sticky API/client/provider/route/version/context/private outer guard; physical faults retain native401 ErrOriginal value and all causes with no fallback/auth/retry/backoff hiding. Group search forwards raw domain including None/false/zero and collects every Source row/page before ordered dict/loaded equality or materialized-list JMES; loaded truthiness/len/integer index0 retains false/0/empty selected values and reports duplicate/type errors without invented record receipt.
  Tests: `identity/group_record_get_test.go:TestCloudGetGroupNoneVersusPresentFalseySearch`, `identity/group_record_get_test.go:TestCloudGetGroupLateNativeFailureBeforeLocalFilter`, `identity/group_record_get_test.go:TestCloudGetGroupStickyOptionsCancelAndOuterGuard`, `identity/group_record_get_test.go:TestCloudGetGroupPhysicalCloseSourceCancelKeepActualNativeEvidence`, `identity/group_record_get_test.go:TestCloudGetGroupJMESFirstTruthLenAndIndexKeepsActualInventory`, `identity/group_record_get_test.go:TestCloudGetGroupOwnedRawDomainLocationAndLoadedNestedFilter`, `identity/v3/groups/python_closed_lookup_test.go:TestClosedGroupDomainSeedAndActualSourceEvidence`, `identity/v3/groups/python_closed_lookup_test.go:TestClosedGroupFallbackVersionsExactIDNameAndDefaultMissing`, `identity/v3/groups/python_closed_lookup_test.go:TestClosedGroupOwnedCallbacksRawDomainAndPreflight`, `identity/v3/groups/python_closed_lookup_test.go:TestClosedGroupPhysicalFaultKeepsNative401ValueAndAllCauses`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- Generic raw Resource.fetch/header/session kwargs and mutable Resource/Connection/cache/version metadata remain SDK-R1/C1/S1. Full named Proxy find_project/find_group stays unresolved; closed ResolveRecord exposes only identity/domain_id and owned Go Header/MV/location conveniences.
- Go result separates actual Inventory/record from local arbitrary JSON, with nil Value/record for missing and DeprecatedFilters marker instead of Python warning categories. Literal typed identity, safe route/cursor/blank guards and JSON-bounded operands are intentional Go bounds; no object passthrough or fabricated HTTP metadata.
- Source Project truthy string JMES fails at dict(**filters) despite documentation; Group materialized JMES supports arbitrary JSON and Source First shape/errors. Private scalar wrapper is a local common-predicate adaptation and never an HTTP resource.
- Verified Keystone28.0.0 Project member GET allows admin/system/domain reader and own-project user; fallback/list needs admin/system/domain reader. Group member/list allows admin/system reader or target Group domain reader. Deployment overrides and live acceptance are not established by fixtures.

### `python:cloud/_identity/IdentityCloudMixin/get_project`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetProjectRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewProjectReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.ProjectReadCloud.GetProject`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/projects.API.ResolveRecord`

- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Default/None uses a closed domain_id-only Source finder: literal GET, only clean400403404 fallback with name/domain hints, exact ID/name, default nil missing, duplicate error, GET-only per-call MV and actual descriptor/Wire/page/header/status evidence. Caller options/domain/header/location bytes snapshot before sticky API/client/provider/route/version/context/private outer guard; physical faults retain native401 ErrOriginal value and all causes with no fallback/auth/retry/backoff hiding. Project actual list_projects kwargs-expansion precedes _filter_list: truthy nonmapping/string filters fail before HTTP; falsey filters still select deprecated branch. Truthy raw domain overrides only server domain and preserves original local filters. Private wrapper reuses common actual-truthiness-first mapping for scalar literal Proxy rows without invented metadata.
  Tests: `identity/project_record_get_test.go:TestCloudGetProjectNoneVersusPresentFalseySearch`, `identity/project_record_get_test.go:TestCloudGetProjectLateNativeFailureBeforeLocalFilter`, `identity/project_record_get_test.go:TestCloudGetProjectStickyOptionsCancelAndOuterGuard`, `identity/project_record_get_test.go:TestCloudGetProjectPhysicalCloseSourceCancelKeepActualNativeEvidence`, `identity/project_record_get_test.go:TestCloudGetProjectTruthyFilterKwargsExpansionFailsBeforeHTTP`, `identity/project_record_get_test.go:TestCloudGetProjectProxyLiteralFalseyRowsFollowDictFilterBeforeShape`, `identity/project_record_get_test.go:TestCloudGetProjectRawDomainTruthyOverrideKeepsOriginalLocalFilters`, `identity/v3/projects/python_closed_lookup_test.go:TestClosedProjectDomainSeedAndActualSourceEvidence`, `identity/v3/projects/python_closed_lookup_test.go:TestClosedProjectFallbackVersionsExactIDNameAndDefaultMissing`, `identity/v3/projects/python_closed_lookup_test.go:TestClosedProjectOwnedCallbacksRawDomainAndPreflight`, `identity/v3/projects/python_closed_lookup_test.go:TestClosedProjectPhysicalFaultKeepsNative401ValueAndAllCauses`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- Generic raw Resource.fetch/header/session kwargs and mutable Resource/Connection/cache/version metadata remain SDK-R1/C1/S1. Full named Proxy find_project/find_group stays unresolved; closed ResolveRecord exposes only identity/domain_id and owned Go Header/MV/location conveniences.
- Go result separates actual Inventory/record from local arbitrary JSON, with nil Value/record for missing and DeprecatedFilters marker instead of Python warning categories. Literal typed identity, safe route/cursor/blank guards and JSON-bounded operands are intentional Go bounds; no object passthrough or fabricated HTTP metadata.
- Source Project truthy string JMES fails at dict(**filters) despite documentation; Group materialized JMES supports arbitrary JSON and Source First shape/errors. Private scalar wrapper is a local common-predicate adaptation and never an HTTP resource.
- Verified Keystone28.0.0 Project member GET allows admin/system/domain reader and own-project user; fallback/list needs admin/system/domain reader. Group member/list allows admin/system reader or target Group domain reader. Deployment overrides and live acceptance are not established by fixtures.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--kind", choices=["project", "group"], default="project")
    parser.add_argument("--name", required=True)
    parser.add_argument("--domain-id", default="")
    parser.add_argument("--filters", default="")
    args = parser.parse_args()
    conn = openstack.connect(cloud=args.cloud)
    filters = json.loads(args.filters) if args.filters else None
    domain = args.domain_id or None
    if args.kind == "project":
        value = conn.get_project(args.name, filters=filters, domain_id=domain)
    else:
        value = conn.get_group(args.name, filters=filters, domain_id=domain)
    if hasattr(value, "to_dict"):
        value = value.to_dict()
    print(json.dumps(value, indent=2, ensure_ascii=False))


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
	"github.com/JSYoo5B/go-openstacksdk/identity"
)

func main() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	kind := flag.String("kind", "project", "project or group")
	name := flag.String("name", "", "literal name or ID")
	domain := flag.String("domain-id", "", "optional domain UUID")
	filters := flag.String("filters", "", "optional JSON filters")
	flag.Parse()
	if *name == "" {
		log.Fatal("-name is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
	if err != nil {
		log.Fatal(err)
	}
	var value json.RawMessage
	switch *kind {
	case "project":
		var options []identity.ProjectGetOption
		if *domain != "" {
			options = append(options, identity.WithProjectGetDomainID(*domain))
		}
		if *filters != "" {
			options = append(options, identity.WithProjectGetFilters(json.RawMessage(*filters)))
		}
		result, err := conn.GetProjectRecord(ctx, *name, options...)
		if err != nil {
			log.Fatal(err)
		}
		value = result.Value
	case "group":
		var options []identity.GroupGetOption
		if *domain != "" {
			options = append(options, identity.WithGroupGetDomainID(*domain))
		}
		if *filters != "" {
			options = append(options, identity.WithGroupGetFilters(json.RawMessage(*filters)))
		}
		result, err := conn.GetGroupRecord(ctx, *name, options...)
		if err != nil {
			log.Fatal(err)
		}
		value = result.Value
	default:
		log.Fatal("-kind must be project or group")
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
}
```

## References

- [`get_project/get_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py)
- [`_utils._filter_list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py)
- [Project](v3/projects/python-record-reads.md)
- [Group](v3/groups/python-records.md)
- [Project](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/project.py)
- [Group](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/group.py)
- [Base](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/base.py)
