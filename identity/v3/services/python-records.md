# Python Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/services`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.ServiceRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.RecordListGenerator.JMESPathIdentity`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.RecordListGenerator.MarshalJSON`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.ProxyListRecords`

- All with resource.WithQuery forwards name, type and limit to GET services as Python services(**query) does for mapped parameters and follows links.next.
  Tests: `identity/v3/services/python_parity_test.go:TestPythonServiceListQueryAndPaging`.
- Pinned Service nullable id/name/description/type/links/is_enabled descriptors retain untyped JSON, bool string truthiness and response canonical/wire last-member precedence; unknown Body is absent from view but physical Wire/page/header/status/Clone stay independently owned.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceRecordsDefaultProjectionPresenceAndIndependentPhysicalEvidence`, `identity/v3/services/python_records_test.go:TestPythonServiceRecordsBoolAliasResponseOrderAndLocalCanonicalKey`.
- Only name/type and inherited limit/marker are server query. Canonical is_enabled and other declared Body values are ordered loaded local filters; undeclared enabled wire query and unknown kwargs discard before encoding. Recursive empty-dictionary/truthy-scalar and loaded float equality match Source.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceRecordsDeclaredQueryAndOrderedLoadedLocalFilters`, `identity/v3/services/python_records_test.go:TestPythonServiceRecordsBoolAliasResponseOrderAndLocalCanonicalKey`.
- Source dictionary root links ignored with key-presence precedence, positive limit marker from last raw wire ID even short pages, max_items raw prefilter cap, empty terminal, lazy break and reusable captured iterator work through shared pager.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceRecordsRawMarkerShortPagesCapBreakAndReusableIterator`, `identity/v3/services/python_records_test.go:TestPythonServiceRecordsSourceDictionaryRootPresenceAndActualContinuationGuards`.
- Immutable option snapshots/typed or raw formal headers/MV/base path and one-pass conflicting attrs match the Source kwargs control cycle. Singleton plural rows, missing plural/204/malformed/constructor collision and local filter failure retain actual accepted receipt; nil/source/cancel preflight errors have no fabricated receipt.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceRecordsOptionSnapshotsFormalsAndPreflightSourceGuards`, `identity/v3/services/python_records_test.go:TestPythonServiceRecordsFormalRestoreHeaderVersionUnknownAndConstructorReceipt`, `identity/v3/services/python_records_test.go:TestPythonServiceRecordsMalformedRowsAndLocalFilterFaultRetainAcceptedPage`.
- Proxy truthy string JMES sees opaque unconsumed generator before HTTP, preserving null projection/type/constants/identity/nested generators and explicit length type-error; omitted/falsey/nonstring keep lazy generator. Typed record iterator requires polymorphic counterpart when return type changes.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceRecordsProxyGeneratorJMESPreservesUnreadAndNestedIdentity`.
- Default standalone location is SourceNoConnection null. Immutable explicit current project facts ignore undeclared remote project/location; invalid JSON scope validates before HTTP, while literal Wire remains original.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceRecordsExplicitLocationOwnsCurrentScopeWithoutUnknownProjectInference`, `identity/v3/services/python_records_test.go:TestPythonServiceResolveRecordOptionsSnapshotCancellationAndExplicitLocation`.
- Physical read/close/mixed fault matrix preserves actual response and causes, native401 pointer ErrOriginal/ErrReauth and one Close/request, with zero Retry/Reauth/Backoff. Clean401/503/429/498 renewal/retry preserve captured query/header/MV and live token.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceRecordsPhysicalFaultsKeepNative401AndActualReceiptWithoutCallbacks`, `identity/v3/services/python_records_test.go:TestPythonServiceRecordsCleanReauthRetryAndBackoffKeepFrozenIdentityRequest`.

Python/Go differences:

- Python takes an ID string or Resource and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model pointer.
- Concrete library-owned immutable JSON descriptors/options/results and captured client/context replace mutable Python Resource/session/subclass objects. Native typed Service/List/All/Find/CRUD contracts are preserved; the SDK user implements no builders or decoder interfaces.
- Direct factory SourceNoConnection has default location:null; explicit immutable CloudLocation/current scope is a Go convenience, and root Connection bridges supply selected identity client plus current location. Whole SDK-R1/C1/S1 mutable Resource/cache/Connection/session/discovery lifecycle remains tracked separately; these three mappings do not claim full SDK supported parity.
- Go preserves actual view/wire/page/header/status evidence and error causes instead of mutable Python object identity or process-global warnings. Protected fixed route/collection/filter/header policy, strict UTF-8/nonnegative pagination/cycle detection and HTTP Link interoperability intentionally bound broader Source adapters and Requests uri/url-link behavior.
- Go JSON marshalling/map-backed option snapshots use deterministic lexical object key order where Python dict has insertion order; ordered WithQuery/raw dictionary filter members retain caller order. Pure JMES JSON/number/syntax policies are concrete Go mappings rather than every Python runtime object/function type behavior.
- iter.Seq2 can repeat a captured request on another iteration instead of a one-shot Python generator. Proxy result Value may retain opaque/nested generator identity and JSON is nil if nonserializable. Python opaque repr/to_string(@) is deliberately bounded by Go serialization errors; no independent Python JMES parser/runtime parity is asserted.

### `python:cloud/_identity/IdentityCloudMixin/search_services`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewServiceReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.ServiceReadCloud.RawAPI`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity.ServiceReadCloud.SearchServices`

- search_services lists with the same GET /v3/services as list_services before filtering locally.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityServiceHelpers`.
- Exact cloud search services has no query kwargs: queryless list materializes every page before identifier exact/glob and ordered local filtering. Duplicate IDs remain and selected ServiceRecord/header/view is a detached clone of physical Inventory.
  Tests: `identity/service_record_cloud_test.go:TestPythonServiceCloudSearchQuerylessAndLocalGlobSubset`.
- Source falsey filters preserve declared rows/null name Python None comparison; truthy string JMES uses materialized rows and returns arbitrary JSON independently of typed Services/actual Inventory. Source loaded numeric nested mapping equality and ordered short circuit are reused.
  Tests: `identity/service_record_cloud_test.go:TestPythonServiceCloudSearchStringExpressionsUseMaterializedRows`, `identity/service_record_cloud_test.go:TestPythonServiceCloudFalseyFiltersPreserveDeclaredRows`, `identity/service_record_cloud_test.go:TestPythonServiceCloudOrderedMismatchHidesUnreachedBadShape`.
- Reached missing/nested-shape/invalid expression errors follow all pages, keep actual Inventory and never fabricate ResponseError. Late HTTP failure occurs before local filtering and retains native status/header/body/method/URL plus prior row proof.
  Tests: `identity/service_record_cloud_test.go:TestPythonServiceCloudLocalErrorsFollowEveryPageAndKeepInventory`, `identity/service_record_cloud_test.go:TestPythonServiceCloudLateHTTPFailurePrecedesInvalidLocalFilter`.
- Cloud immutable filters/headers/location preflight captures original source before options; current location is computed before filtering, ignores undeclared remote ownership, and owner drift/caller custom cancellation/nil options prevent later callbacks and HTTP.
  Tests: `identity/service_record_cloud_test.go:TestPythonServiceCloudLocationBeforeFilterAndOwnedArguments`, `identity/service_record_cloud_test.go:TestPythonServiceCloudOptionsGuardCallbacksAndSelectedSourceSnapshot`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Concrete library-owned immutable JSON descriptors/options/results and captured client/context replace mutable Python Resource/session/subclass objects. Native typed Service/List/All/Find/CRUD contracts are preserved; the SDK user implements no builders or decoder interfaces.
- Direct factory SourceNoConnection has default location:null; explicit immutable CloudLocation/current scope is a Go convenience, and root Connection bridges supply selected identity client plus current location. Whole SDK-R1/C1/S1 mutable Resource/cache/Connection/session/discovery lifecycle remains tracked separately; these three mappings do not claim full SDK supported parity.
- Go preserves actual view/wire/page/header/status evidence and error causes instead of mutable Python object identity or process-global warnings. Protected fixed route/collection/filter/header policy, strict UTF-8/nonnegative pagination/cycle detection and HTTP Link interoperability intentionally bound broader Source adapters and Requests uri/url-link behavior.
- Go JSON marshalling/map-backed option snapshots use deterministic lexical object key order where Python dict has insertion order; ordered WithQuery/raw dictionary filter members retain caller order. Pure JMES JSON/number/syntax policies are concrete Go mappings rather than every Python runtime object/function type behavior.
- Source warnings/deprecations are not process-global output. SearchServices returns an explicit ServiceSearchResult with polymorphic JSON and physical Inventory instead of a nominal Python list that may actually be scalar/object/string/null. No expression result receives invented ServiceRecord/HTTP metadata.

### `python:cloud/_identity/IdentityCloudMixin/get_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewServiceReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.ServiceReadCloud.RawAPI`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithServiceReadLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity.ServiceReadCloud.GetService`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordResolveOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordResolveHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordResolveMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.WithRecordResolveLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/services.API.ResolveRecord`

- get_service(name_or_id) with the default filters=None sends GET /v3/services/<x>, falls back to GET /v3/services?name=<x> after 404 and returns nil when nothing matches.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityServiceHelpers`.
- Omitted or literal JSON null filters use the Source closed-signature finder: member GET100..399 success, only clean400/403/404 fallback to name hint and exact ID-or-name, missing nil and duplicate ambiguity. Fallback consumes after first match but stops at second; no raw arbitrary fetch kwargs are exposed by this named Source call.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceResolveRecordClosedFinderMemberAndCleanStatusFallback`, `identity/v3/services/python_records_test.go:TestPythonServiceResolveRecordMissingDuplicatesAndGetOnlyVersion`.
- Get default fallback preserves exact name rather than glob, accepted original member/list record proof and missing JSON null. {}false/0/[]/empty-string are non-None filters and use queryless search; filtered selection honors glob/dictionaries and never deduplicates repeated IDs.
  Tests: `identity/service_record_cloud_test.go:TestPythonServiceCloudGetNullFilterMemberVersusAllOtherFalseySearchBranches`, `identity/service_record_cloud_test.go:TestPythonServiceCloudGetDefaultFallbackIsLiteralNameNotGlob`, `identity/service_record_cloud_test.go:TestPythonServiceCloudGetDeprecatedFiltersDuplicatesSearchGlobAndMissing`.
- Deprecated expression output follows actual Python truth/len/integer-index control: falsey returns None; >1 raises ambiguity; one-element array or one-codepoint string selects JSON; truthy scalar len and one-member object integer-key indexing raise local errors with actual Inventory only.
  Tests: `identity/service_record_cloud_test.go:TestPythonServiceCloudGetDeprecatedExpressionTruthLengthAndIntegerIndex`.
- Explicit GET-only MV override does not change fallback LIST selected version. Closed resolve option/header/location snapshot, nil/source/cancellation guard, physical403 close failure without fallback or retries preserve native evidence and caller causes.
  Tests: `identity/v3/services/python_records_test.go:TestPythonServiceResolveRecordMissingDuplicatesAndGetOnlyVersion`, `identity/v3/services/python_records_test.go:TestPythonServiceResolveRecordPhysicalForbiddenDoesNotFallThrough`, `identity/v3/services/python_records_test.go:TestPythonServiceResolveRecordOptionsSnapshotCancellationAndExplicitLocation`.
- Get options execute once using captured selected source before callbacks. Explicit immutable current location remains independent of raw project/location; deprecated complete-list lateHTTP beats local malformed filters and unreachable invalid filter on empty selection stays unvisited.
  Tests: `identity/service_record_cloud_test.go:TestPythonServiceCloudGetOptionCallbackOnceCapturedSourceAndOwnedLocation`, `identity/service_record_cloud_test.go:TestPythonServiceCloudGetLateHTTPBeforeDeprecatedLocalErrorsAndEmptySelection`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.find does the ID GET and list fallback in one call and also falls back after GET 400 or 403; Go chains Find(resource.ID(x), resource.WithIgnoreMissing()) and Find(resource.Name(x), resource.WithIgnoreMissing()), which falls through only after a 404.
- The Python list match compares the ID or the name and raises DuplicateResource; the Go name lookup compares only the name and returns resource.ErrAmbiguous.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- Concrete library-owned immutable JSON descriptors/options/results and captured client/context replace mutable Python Resource/session/subclass objects. Native typed Service/List/All/Find/CRUD contracts are preserved; the SDK user implements no builders or decoder interfaces.
- Direct factory SourceNoConnection has default location:null; explicit immutable CloudLocation/current scope is a Go convenience, and root Connection bridges supply selected identity client plus current location. Whole SDK-R1/C1/S1 mutable Resource/cache/Connection/session/discovery lifecycle remains tracked separately; these three mappings do not claim full SDK supported parity.
- Go preserves actual view/wire/page/header/status evidence and error causes instead of mutable Python object identity or process-global warnings. Protected fixed route/collection/filter/header policy, strict UTF-8/nonnegative pagination/cycle detection and HTTP Link interoperability intentionally bound broader Source adapters and Requests uri/url-link behavior.
- Go JSON marshalling/map-backed option snapshots use deterministic lexical object key order where Python dict has insertion order; ordered WithQuery/raw dictionary filter members retain caller order. Pure JMES JSON/number/syntax policies are concrete Go mappings rather than every Python runtime object/function type behavior.
- Go ServiceGetResult.Service exists only for actual ordinary selected resources; expression output is JSON Value plus actual Inventory. Missing success is JSON null; failed result Value is absent. DeprecatedFilters records Source filters-is-not-None branch instead of emitting Python process-global warnings.
- RecordResolveOpts headers/MV/location are explicit Go conveniences, not Source get_service/find_service kwargs. The original named closed-signature Cloud method cannot reach raw Resource.fetch**query branches; the generic Source Find/Resource gaps remain separate inventory operations.
- Default closed finder exposes selected record/error proof but not every discarded fallback row in Inventory; deprecated search branch retains actual prior Inventory. Pure Go scalar/index/type errors are wrapped concrete invalid-option/ambiguity errors rather than Python TypeError/KeyError/SDKException classes.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import os
import openstack


def print_value(value):
    if hasattr(value, "to_dict"):
        value = value.to_dict()
    print(json.dumps(value, indent=2, ensure_ascii=False))


def main():
    conn = openstack.connect()
    identifier = os.environ["IDENTITY_SERVICE_ID"]
    for row in conn.identity.services(
        type="compute", is_enabled=True, max_items=5
    ):
        print_value(row)
    print_value(conn.identity.services(jmespath_filters="[*].id"))
    print_value(conn.search_services(name_or_id="n*", filters="length(@)"))
    print_value(conn.get_service(identifier))
    print_value(conn.get_service(identifier, filters={}))


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
	"os"

	openstack "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/services"
)

func printValue(value any) {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", encoded)
}
func main() {
	ctx := context.Background()
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	svc, err := conn.IdentityV3(ctx)
	if err != nil {
		log.Fatal(err)
	}
	location, err := conn.CurrentLocation()
	if err != nil {
		log.Fatal(err)
	}
	identifier := os.Getenv("IDENTITY_SERVICE_ID")
	if identifier == "" {
		log.Fatal("set IDENTITY_SERVICE_ID to a readable service ID or name")
	}
	rows, err := svc.Services.AllRecords(ctx,
		services.WithRecordListQuery("type", "compute"),
		services.WithRecordListQuery("is_enabled", true),
		services.WithRecordListMaxItems(5),
		services.WithRecordListLocation(location),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range rows {
		printValue(row.Resource)
	}
	proxy, err := svc.Services.ProxyListRecords(ctx, services.WithRecordListJMESPath("[*].id"))
	if err != nil {
		log.Fatal(err)
	}
	printValue(proxy.JSON)
	cloud := identity.NewServiceReadCloud(svc.Services)
	found, err := cloud.SearchServices(ctx, "n*",
		identity.WithServiceReadFilters(json.RawMessage(`"length(@)"`)),
		identity.WithServiceReadLocation(location),
	)
	if err != nil {
		log.Fatal(err)
	}
	printValue(found.Value)
	direct, err := cloud.GetService(ctx, identifier, identity.WithServiceReadLocation(location))
	if err != nil {
		log.Fatal(err)
	}
	printValue(direct.Value)
	searched, err := cloud.GetService(ctx, identifier,
		identity.WithServiceReadFilters(json.RawMessage(`{}`)),
		identity.WithServiceReadLocation(location),
	)
	if err != nil {
		log.Fatal(err)
	}
	printValue(searched.Value)
}
```

## References

- [Service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/service.py)
- [Proxy.services/find_service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1274)
- [Cloud search_services/get_service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L600)
- [Policy](https://docs.openstack.org/keystone/2026.1/configuration/policy.html)
- [Service](https://docs.openstack.org/keystone/2026.1/_modules/keystone/common/policies/service.html)
