# Python Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/roles`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.RoleRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordListLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordListJMESPath`

- List sends the single GET roles that Python roles() sends without arguments, maps the Python query names domain_id and name through ListOpts or WithListQuery, sends a limit without a marker follow-up, and rejects local Body filters before HTTP.
  Tests: `identity/v3/roles/python_parity_test.go:TestPythonRoleListRequests`.
- Bodyless default roles GET and explicit SourceStatusPolicy accept actual199. Six Role Body descriptors/default/null/options-dict cast discard unknowns only from declared view and retain all actual Wire/page/header/status evidence; Clone channels are independent.
  Tests: `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsDefaultProjectionPresenceAndIndependentPhysicalEvidence`.
- Only name/domain_id/limit/marker go to the server with Requests repeated and bool/null values. Ordered local description/options/links/id filters use CPython loaded equality and recursive subset, empty-dict matches truthy scalar/list, and unknown kwargs discard before marshal.
  Tests: `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsDeclaredQueryAndOrderedLoadedLocalFilters`.
- Explicit positive limit repeats short nonempty pages without links with last raw ID; local-excluded rows count in marker/raw cap; break stops before another request and reusable iterator repeats guarded captured calls.
  Tests: `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsRawMarkerShortPagesCapBreakAndReusableIterator`.
- Source Keystone dictionary-next is ignored, root links presence masks plural links, root null/foreign URL/query change/cycle retain actual whole page proof.
  Tests: `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsSourceDictionaryRootPresenceAndActualContinuationGuards`.
- Unconsumed Proxy generator JMESPath returns null/literals/opaque identity/nested generator before HTTP; array type errors do not consume it. Known formal options, snapshots, reserved/collision/unsafe path/cap validation, source mutation and custom callback cancel fail without fake response.
  Tests: `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsProxyGeneratorJMESPreservesUnreadAndNestedIdentity`, `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsOptionSnapshotsFormalsAndPreflightSourceGuards`.
- Immutable typed location retains recorded current scope, ignores unknown project_id, and physical malformed rows/nonmapping local filter preserve accepted203 page.
  Tests: `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsExplicitLocationOwnsCurrentScopeWithoutUnknownProjectInference`, `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsMalformedRowsAndLocalFilterFaultRetainAcceptedPage`.
- Accepted/rejected200/400/403/404/401/429/498/503 read/Close/both/restored-source+custom-cancel matrix keeps one physical HTTP/Close, actual native or accepted proof, all causes and zero retry/backoff/reauth. Clean401/503/429/498 preserve fixed request/version/absent conditions and live renewed token.
  Tests: `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsPhysicalFaultsKeepNative401AndActualReceiptWithoutCallbacks`, `identity/v3/roles/python_records_test.go:TestPythonRoleRecordsCleanReauthRetryAndBackoffKeepFrozenIdentityRequest`.

Python/Go differences:

- Python reads the Keystone links dict without a rel key and never follows links.next; the native pager follows links.next when Keystone sets it.
- Python sends bool query values as True/False text; typed Go ListOpts send true/false, and WithListQuery reproduces the Python text.
- Python drops unknown query names; Go sends every WithListQuery key to the server.
- roles.ListOpts.Filters adds a stray "-" query key next to the comparator filters; Python cannot send comparator filters at all.
- Existing native Role/List/All/Get/FindIdentity remain unchanged. Concrete functional options and SDK-owned profile descriptors/defaults/decoders avoid caller builders; declared Resource JSON and separate actual Wire/Envelope/header/status are defensively owned.
- Selected Keystone v3 API is required; automatic v2 ensure_service_version conversion is not added. Standalone native client computed location is null; explicit immutable location supplies cloud/region/current project without inference. Role has no project_id/zone descriptors, unknown response facts do not replace that scope and the typed convenience normalizes zone to null.
- Source root link precedence and ignored Keystone dictionary-next are opt-in on this new profile. Fixed origin/path/query/header/version, stronger cycles, representable safe markers and nonnegative Go caps are deliberate Go policies; RFC HTTP Link corrects the pinned Requests next uri/url mismatch.
- Go iterator reuse repeats the captured guarded operation instead of Python generator single-use state. Pure Proxy expressions still run SDK option/source preflight; JSON serialization is separate from opaque/nested generator results.
- SDK-R1/C1/S1 remain separately tracked Connection-wide synchronized Resource/runtime, optional response cache and session/version negotiation. This finite named mapping is not supported full SDK parity or authenticated live-cloud execution. No new find_role mapping or closure of its raw Resource.fetch/control/header kwargs is claimed.

### `python:cloud/_identity/IdentityCloudMixin/list_roles`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewRoleReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.RoleReadCloud.ListRoles`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithRecordListQuery`

- Source cloud list_roles expands ordinary Proxy kwargs into the owned list classifier, preserves server name/domain and local options filter, and exposes independent JSON Value/Roles/actual Inventory with203 header/status.
  Tests: `identity/role_record_cloud_test.go:TestPythonRoleCloudListForwardsOwnedProxyQueryAndLocalBodyFilters`.
- list() conversion preserves JSON literal lists/string iteration/object key lists and opaque nested generator with no invented HTTP/Role association. Noniterable null/number/bool and opaque array functions fail before HTTP.
  Tests: `identity/role_record_cloud_test.go:TestPythonRoleCloudListLiteralOpaqueAndNonIterableBranchesNeverFabricateHTTP`.
- Captured source headers precede list option callbacks. Actual late503 body/header/method/URL preserves prior Inventory while callback runs once; nil/canceled preflight makes zero requests.
  Tests: `identity/role_record_cloud_test.go:TestPythonRoleCloudListOptionCallbacksSnapshotSourceAndLateNativeProof`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.list sends one GET without a limit and ignores the Keystone links dict; Go All/List follows a non-null links.next URL.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- Existing native Role/List/All/Get/FindIdentity remain unchanged. Concrete functional options and SDK-owned profile descriptors/defaults/decoders avoid caller builders; declared Resource JSON and separate actual Wire/Envelope/header/status are defensively owned.
- Selected Keystone v3 API is required; automatic v2 ensure_service_version conversion is not added. Standalone native client computed location is null; explicit immutable location supplies cloud/region/current project without inference. Role has no project_id/zone descriptors, unknown response facts do not replace that scope and the typed convenience normalizes zone to null.
- Source root link precedence and ignored Keystone dictionary-next are opt-in on this new profile. Fixed origin/path/query/header/version, stronger cycles, representable safe markers and nonnegative Go caps are deliberate Go policies; RFC HTTP Link corrects the pinned Requests next uri/url mismatch.
- Go iterator reuse repeats the captured guarded operation instead of Python generator single-use state. Pure Proxy expressions still run SDK option/source preflight; JSON serialization is separate from opaque/nested generator results.
- SDK-R1/C1/S1 remain separately tracked Connection-wide synchronized Resource/runtime, optional response cache and session/version negotiation. This finite named mapping is not supported full SDK parity or authenticated live-cloud execution. No new find_role mapping or closure of its raw Resource.fetch/control/header kwargs is claimed.
- RoleListResult.Value is []any and JSON is populated only if every value is JSON-representable. Object-valued Proxy expressions convert to deterministically sorted Go keys instead of Python insertion order; literal/opaque values do not acquire synthesized transport evidence.

### `python:cloud/_identity/IdentityCloudMixin/search_roles`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity.RoleReadCloud.SearchRoles`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleSearchOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleSearchDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleSearchFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleSearchLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleSearchHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleSearchMicroversion`

- Search lists all pages with only domain query then exact/glob ID/name and ordered recursive loaded-JSON filters. Duplicate records stay in selection, selected records clone away from actual Inventory and203 header/status.
  Tests: `identity/role_record_cloud_test.go:TestPythonRoleCloudSearchDomainServerAndLocalGlobSubset`.
- Truthful materialized string JMESPath yields arbitrary JSON and nil Roles with Expression=true. Name is selected first, empty selection is still evaluated, falsey filters retain declared rows and late HTTP fails before invalid local expression.
  Tests: `identity/role_record_cloud_test.go:TestPythonRoleCloudSearchStringExpressionsUseMaterializedRows`, `identity/role_record_cloud_test.go:TestPythonRoleCloudFalseyFiltersPreserveDeclaredRows`, `identity/role_record_cloud_test.go:TestPythonRoleCloudLateHTTPFailurePrecedesInvalidLocalFilter`.
- Local missing keys/shape errors occur after actual list and retain Inventory; insertion-order mismatch avoids evaluating later invalid shape. No fake HTTP error is added for purely local selection failures.
  Tests: `identity/role_record_cloud_test.go:TestPythonRoleCloudLocalErrorsFollowEveryPageAndKeepInventory`, `identity/role_record_cloud_test.go:TestPythonRoleCloudOrderedMismatchHidesUnreachedBadShape`.
- Explicit typed owned location is applied before local filters and Wire remains original; source callback snapshots/mutations/nil/cancel are guarded throughout the facade and list dependency.
  Tests: `identity/role_record_cloud_test.go:TestPythonRoleCloudLocationBeforeFilterAndOwnedArguments`, `identity/role_record_cloud_test.go:TestPythonRoleCloudOptionsGuardCallbacksAndSelectedSourceSnapshot`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Existing native Role/List/All/Get/FindIdentity remain unchanged. Concrete functional options and SDK-owned profile descriptors/defaults/decoders avoid caller builders; declared Resource JSON and separate actual Wire/Envelope/header/status are defensively owned.
- Selected Keystone v3 API is required; automatic v2 ensure_service_version conversion is not added. Standalone native client computed location is null; explicit immutable location supplies cloud/region/current project without inference. Role has no project_id/zone descriptors, unknown response facts do not replace that scope and the typed convenience normalizes zone to null.
- Source root link precedence and ignored Keystone dictionary-next are opt-in on this new profile. Fixed origin/path/query/header/version, stronger cycles, representable safe markers and nonnegative Go caps are deliberate Go policies; RFC HTTP Link corrects the pinned Requests next uri/url mismatch.
- Go iterator reuse repeats the captured guarded operation instead of Python generator single-use state. Pure Proxy expressions still run SDK option/source preflight; JSON serialization is separate from opaque/nested generator results.
- SDK-R1/C1/S1 remain separately tracked Connection-wide synchronized Resource/runtime, optional response cache and session/version negotiation. This finite named mapping is not supported full SDK parity or authenticated live-cloud execution. No new find_role mapping or closure of its raw Resource.fetch/control/header kwargs is claimed.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import openstack


def main():
    conn = openstack.connect()
    for role in conn.identity.roles(description="operations", limit=2,
                                    max_items=2):
        print(json.dumps(role.to_dict(), indent=2, ensure_ascii=False))

    listed = conn.list_roles(description="operations")
    print(json.dumps([role.to_dict() for role in listed], indent=2))
    selected = conn.search_roles("read*", filters={"options": {"immutable": False}})
    print(json.dumps([role.to_dict() for role in selected], indent=2))
    count = conn.search_roles(filters="length(@)")
    print(json.dumps(count, indent=2))


if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/roles"
)

func printJSON(value any) {
    encoded, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(encoded))
}

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }

    for role, err := range service.Roles.ListRecords(ctx,
        roles.WithRecordListQuery("description", "operations"),
        roles.WithRecordListLimit(2), roles.WithRecordListMaxItems(2)) {
        if err != nil { log.Fatal(err) }
        printJSON(role.Resource)
    }
    cloud := identity.NewRoleReadCloud(service.Roles)
    listed, err := cloud.ListRoles(ctx,
        roles.WithRecordListQuery("description", "operations"))
    if err != nil { log.Fatal(err) }
    printJSON(listed.JSON)
    selected, err := cloud.SearchRoles(ctx, "read*",
        identity.WithRoleSearchFilters(json.RawMessage(`{"options":{"immutable":false}}`)))
    if err != nil { log.Fatal(err) }
    printJSON(selected.Value)
    count, err := cloud.SearchRoles(ctx, "",
        identity.WithRoleSearchFilters(json.RawMessage(`"length(@)"`)))
    if err != nil { log.Fatal(err) }
    printJSON(count.Value)
}
```

## References

- [openstacksdk ef55d7d](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/role.py)
- [Source Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)
- [Keystone28.0.0 Role policy](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/role.py#L99-L119)
- [controller](https://github.com/openstack/keystone/blob/28.0.0/keystone/api/roles.py#L45-L72)
- [Base](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/base.py)
- [Policy](https://docs.openstack.org/keystone/2025.2/configuration/policy.html)
- [Python Record Find](python-record-find.md)
