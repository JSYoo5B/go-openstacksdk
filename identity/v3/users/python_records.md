# Python Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/users`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.UserRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithRecordListPaginated`

- List sends the single GET users that Python users() sends without arguments, maps the Python query names domain_id, enabled, idp_id, name, password_expires_at, protocol_id and unique_id through ListOpts or WithListQuery, sends a limit without a marker follow-up, and rejects local Body filters before HTTP.
  Tests: `identity/v3/users/python_parity_test.go:TestPythonUserListRequests`.
- Pinned User nullable declarations/casts/defaults, canonical/wire response last-member order, untyped large password-expiration values, options missing{} vs explicitnull, federated scalar wrapping, unknown discard and independent view/wire/page/header/status/Clone match owned Source policy.
  Tests: `identity/v3/users/python_records_test.go:TestPythonUserListRecordsExactDescriptorDefaultsAndResponseAliasOrder`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsLazyQueryAndUntypedView`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsOptionsDefaultNullAndNestedLoadedLocal`.
- Mapped domain/name/password expiry/unique/idp/protocol/is_enabled->enabled plus inherited limit/marker preserve Requests repeated/null/bool values and canonical query precedence; declared nonquery Body kwargs run ordered loaded local filters, while unknown kwargs discard before marshal.
  Tests: `identity/v3/users/python_records_test.go:TestPythonUserListRecordsDeclaredQueriesCanonicalWinsRequestsValues`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsLoadedLocalFilterAndShortCircuit`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsOptionsDefaultNullAndNestedLoadedLocal`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsUnknownDiscardBeforeEncodingAndTypedJMESBoundary`.
- Lazy list defaults, Source root dictionary links ignored with key-presence precedence, short nonempty page positive-limit marker from last raw wire ID, raw prefilter max_items hint/single page/empty terminal and actual unsafe-next proof preserve the named Source list.
  Tests: `identity/v3/users/python_records_test.go:TestPythonUserListRecordsDictionaryLinksIgnoredWithSourcePrecedence`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsFilteredMarkerAndRawMaxItems`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsIteratorKeepsYieldedRowsAndAllDiscards`.
- Copied raw/options/header/MV/base_path and one-pass conflicting attrs, original JMES priority, constructor/formal errors, bound route/context/source guards and explicit100..399 SourceStatusPolicy operate without changing existing native API defaults or inventing discovery.
  Tests: `identity/v3/users/python_records_test.go:TestPythonUserListRecordsOptionsSnapshotFormalRestorationAndHeaders`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsGuardsOptionCancellationAndContextCauses`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsSourceSuccessStatusAndHeaderNoDiscovery`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsSingletonMalformedAndConstructorEvidence`.
- Opaque Source Proxy generator receives truthy JMES before HTTP; null/index/projection/type/literals/type-errors and retained/nested generator identity are distinct from materialized arrays. Omitted/falsey/nonstring filters keep lazy rows; typed iterator explicitly directs truthy string callers to polymorphic method.
  Tests: `identity/v3/users/python_records_test.go:TestPythonUserProxyListOpaqueProjectionAndRetainedGenerators`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsUnknownDiscardBeforeEncodingAndTypedJMESBoundary`.
- SourceNoConnection default location null, explicit immutable configured CloudLocation retains current scope because User has no project/zone Body; default_project_id and unknown project_id/location cannot alter it. Raw receipt remains unchanged and invalid location rejects beforeHTTP.
  Tests: `identity/v3/users/python_records_test.go:TestPythonUserListRecordsLocationIsExplicitImmutableAndDeclared`.
- Physical200/203/401/404/429/503 read/close/mixed faults preserve actual accepted/native rejection proof plus every cause and native401 ErrOriginal/ErrReauth, oneClose/HTTP and zero Retry/Reauth. Clean401/503 preserve captured query/headers/version and live token; Close cancel preserves real receipt and caller context cause.
  Tests: `identity/v3/users/python_records_test.go:TestPythonUserListRecordsPhysicalFailuresPreserveNative401AndStopCallbacks`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsCleanReauthenticationAndRetryKeepPolicy`, `identity/v3/users/python_records_test.go:TestPythonUserListRecordsCloseCancellationKeepsReceiptAndCause`.

Python/Go differences:

- Python reads the Keystone links dict without a rel key and never follows links.next; the native pager follows links.next when Keystone sets it.
- Python sends bool query values as True/False text; typed Go ListOpts send true/false, and WithListQuery reproduces the Python text.
- Python drops unknown query names; Go sends every WithListQuery key to the server.
- users.ListOpts.Filters adds a stray "-" query key next to the comparator filters; Python cannot send comparator filters at all.
- Go owns immutable JSON descriptor records, fixed source/context/options, detached actual evidence and explicit polymorphic result instead of mutable Python Resource/session/subclass objects. Native User/List/All/membership APIs retain their current contracts.
- Direct leaf/selected v3 factory has no Python Connection: computed location defaults null and immutable explicit CloudLocation is a Go convenience. No default_project_id authorization or automatic client/scope/name inference is introduced. Whole SDK-R1/C1/S1 mutable Resource/cache/session/Connection/version configuration remain tracked separately.
- Go safe UTF8/fixed-origin/collection/filter/conditional-header bounds, nonnegative integer pagination, stronger cycles and valid RFC HTTP Link interoperability intentionally bound broader Source adapters and the pinned Requests uri/url link defect. Runtime constructor controls fail as concrete preflight/receipt errors rather than arbitrary Python object mutations.
- iter.Seq2 can repeat the captured request on repeated iteration rather than one-shot Python generator state. Proxy pure expression still performs SDK option/source preflight; JSON carrier is absent for nested generators. No live cloud/full SDK or shared-engine supported status is claimed.

### `python:cloud/_identity/IdentityCloudMixin/list_users`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.All`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewUserReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.UserReadCloud.ListUsers`
- `github.com/JSYoo5B/go-openstacksdk/identity.UserReadCloud.RawAPI`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserListDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserListName`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserListLocation`

- list_users() is GET /v3/users and list_users(domain_id=..., name=...) adds exactly those two query keys.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityProjectUserHelpers`.
- Exact decorated Source cloud list accepts only domain_id/name kwargs; ordinary typed/raw null/false/repeated values are passed to owned users list while all other named kwargs fail before marshal/HTTP. No header/filter/JMES/limit Source kwargs are invented.
  Tests: `identity/user_record_cloud_test.go:TestPythonUserCloudListDefaultAndOnlyDeclaredSourceKwargs`, `identity/user_record_cloud_test.go:TestPythonUserCloudListValidKwargsRejectBeforeMarshalerAndHTTP`.
- Cloud list materializes Source rows without member/project lookup, ignores ordinary Keystone dict next, returns owned declared Value/Users and actual Inventory with no fake receipt, and successful empty list publishes nonnil Users/Inventory.
  Tests: `identity/user_record_cloud_test.go:TestPythonUserCloudListDefaultAndOnlyDeclaredSourceKwargs`, `identity/user_record_cloud_test.go:TestPythonUserCloudListOptionSnapshotsHeadersAndCallbacksOwnMaps`.
- Captured selected service/context around copied map/raw/options/header/version callbacks preserves mutable-input snapshots, initializes callback maps, stops after simultaneous option/cancel/source error and avoids fabricated HTTP evidence; explicit User location never treats default project as ownership.
  Tests: `identity/user_record_cloud_test.go:TestPythonUserCloudListOptionSnapshotsHeadersAndCallbacksOwnMaps`, `identity/user_record_cloud_test.go:TestPythonUserCloudSelectedSourceCancellationAndPreflightCannotCreateReceipt`, `identity/user_record_cloud_test.go:TestPythonUserCloudLocationBeforeFilteringIgnoresDefaultProjectAndRemoteForgery`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python Resource.list sends one GET without a limit and ignores the Keystone links dict; Go All/List follows a non-null links.next URL.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- Go owns immutable JSON descriptor records, fixed source/context/options, detached actual evidence and explicit polymorphic result instead of mutable Python Resource/session/subclass objects. Native User/List/All/membership APIs retain their current contracts.
- Direct leaf/selected v3 factory has no Python Connection: computed location defaults null and immutable explicit CloudLocation is a Go convenience. No default_project_id authorization or automatic client/scope/name inference is introduced. Whole SDK-R1/C1/S1 mutable Resource/cache/session/Connection/version configuration remain tracked separately.
- Go safe UTF8/fixed-origin/collection/filter/conditional-header bounds, nonnegative integer pagination, stronger cycles and valid RFC HTTP Link interoperability intentionally bound broader Source adapters and the pinned Requests uri/url link defect. Runtime constructor controls fail as concrete preflight/receipt errors rather than arbitrary Python object mutations.
- iter.Seq2 can repeat the captured request on repeated iteration rather than one-shot Python generator state. Proxy pure expression still performs SDK option/source preflight; JSON carrier is absent for nested generators. No live cloud/full SDK or shared-engine supported status is claimed.
- Source valid_kwargs domain_id/name restriction is represented explicitly by WithUserListQuery; typed Headers/Microversion/Location are Go transport conveniences, not extra accepted cloud kwargs. The existing native cloud mapping remains available; the new factory materializes owned Source records.

### `python:cloud/_identity/IdentityCloudMixin/search_users`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.List`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewUserReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.UserReadCloud.SearchUsers`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserSearchOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserSearchDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserSearchFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserSearchHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserSearchMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserSearchLocation`

- search_users lists with GET /v3/users carrying name (only for a non-UUID name_or_id) and domain_id before filtering locally.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityProjectUserHelpers`.
- Source UUID hex/braces/prefix/hyphen detection controls only truthy nonUUID server name hint; domain_id is truthy scope, no user/project lookup occurs. Glob name also reaches server literally before exact/glob ID/name local selection.
  Tests: `identity/user_record_cloud_test.go:TestPythonUserCloudSearchExactSourceUUIDDetectionAndNameHint`, `identity/user_record_cloud_test.go:TestPythonUserCloudSearchGlobOrderedLoadedFiltersAndDeclaredMembership`, `identity/user_record_cloud_test.go:TestPythonUserCloudEmptyListAndPythonNoneIdentifier`.
- Search completes all list pages then applies ordered nested mapping with Source membership errors/loaded float and bool equality or materialized-row string JMESPath. Falsey filters preserve rows; reached bad-shape/unknown filters fail afterHTTP; empty identifier selection avoids reached nonmapping filter errors.
  Tests: `identity/user_record_cloud_test.go:TestPythonUserCloudSearchGlobOrderedLoadedFiltersAndDeclaredMembership`, `identity/user_record_cloud_test.go:TestPythonUserCloudSearchFalseyAndReachedInvalidFilters`.
- Source cloud JMESPath returns arbitrary scalar/null/list JSON separately from actual User association, retains actual observed Inventory only, and type/syntax failures never borrow accepted203 proof. This is materialized input distinct from Proxy opaque generator.
  Tests: `identity/user_record_cloud_test.go:TestPythonUserCloudSearchExpressionsMaterializeThenReturnArbitraryJSON`, `identity/user_record_cloud_test.go:TestPythonUserCloudSearchOpaqueLocalErrorsNeverBorrowPhysicalProof`.
- Every page precedes local evaluation, late503 wins over invalid JMES, actual body/header/status remains native rejection and prior Inventory; explicit current User location is applied before local location filtering, ignores default project/server forgery, and invalid location creates noHTTP.
  Tests: `identity/user_record_cloud_test.go:TestPythonUserCloudSearchAllPagesBeforeLocalFailureAndLateHTTPWins`, `identity/user_record_cloud_test.go:TestPythonUserCloudLocationBeforeFilteringIgnoresDefaultProjectAndRemoteForgery`.
- Concrete query/filter/header/MV/location copies plus sticky source/context/options preserve custom cancellation and callback cause, stop later callbacks/HTTP, and keep null/missing/empty-list/Python None string identifier behavior.
  Tests: `identity/user_record_cloud_test.go:TestPythonUserCloudSelectedSourceCancellationAndPreflightCannotCreateReceipt`, `identity/user_record_cloud_test.go:TestPythonUserCloudEmptyListAndPythonNoneIdentifier`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Go owns immutable JSON descriptor records, fixed source/context/options, detached actual evidence and explicit polymorphic result instead of mutable Python Resource/session/subclass objects. Native User/List/All/membership APIs retain their current contracts.
- Direct leaf/selected v3 factory has no Python Connection: computed location defaults null and immutable explicit CloudLocation is a Go convenience. No default_project_id authorization or automatic client/scope/name inference is introduced. Whole SDK-R1/C1/S1 mutable Resource/cache/session/Connection/version configuration remain tracked separately.
- Go safe UTF8/fixed-origin/collection/filter/conditional-header bounds, nonnegative integer pagination, stronger cycles and valid RFC HTTP Link interoperability intentionally bound broader Source adapters and the pinned Requests uri/url link defect. Runtime constructor controls fail as concrete preflight/receipt errors rather than arbitrary Python object mutations.
- iter.Seq2 can repeat the captured request on repeated iteration rather than one-shot Python generator state. Proxy pure expression still performs SDK option/source preflight; JSON carrier is absent for nested generators. No live cloud/full SDK or shared-engine supported status is claimed.
- Search uses a typed Go name/domain string and raw JSON filters rather than arbitrary Python object coercion. JMESPath exact-decimal/lexical object-key/strict grammar/custom-type differences are existing explicit common-engine policies; shared dependency parity is not promoted by this row.
- Value is arbitrary expression JSON with Users only for ordinary correlated selection; Inventory contains actual detached rows on later failure, whereas Python raises without returning partial rows. The selected v3 factory does not automatically negotiate an identity version or infer Connection facts.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import os
import openstack


def main():
    conn = openstack.connect()
    domain = os.environ["USER_QUERY_DOMAIN_ID"]
    for row in conn.identity.users(
        domain_id=domain, is_enabled=True, max_items=10
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))
    listed = conn.list_users(domain_id=domain)
    print(json.dumps([row.to_dict() for row in listed], indent=2))
    searched = conn.search_users(
        name_or_id="team*", domain_id=domain, filters="length(@)"
    )
    print(json.dumps(searched, indent=2))


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
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
)

func printJSON(value any) {
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
	domain := os.Getenv("USER_QUERY_DOMAIN_ID")
	if domain == "" {
		log.Fatal("set USER_QUERY_DOMAIN_ID to an allowed domain ID")
	}

	rows, err := svc.Users.AllRecords(ctx,
		users.WithRecordListQuery("domain_id", domain),
		users.WithRecordListQuery("is_enabled", true),
		users.WithRecordListMaxItems(10),
		users.WithRecordListLocation(location),
	)
	if err != nil {
		log.Fatal(err)
	}
	for _, row := range rows {
		printJSON(row.Resource.Body)
	}

	cloud := identity.NewUserReadCloud(svc.Users)
	listed, err := cloud.ListUsers(ctx,
		identity.WithUserListDomainID(domain),
		identity.WithUserListLocation(location),
	)
	if err != nil {
		log.Fatal(err)
	}
	printJSON(listed.Value)

	searched, err := cloud.SearchUsers(ctx, "team*",
		identity.WithUserSearchDomainID(domain),
		identity.WithUserSearchFilters(json.RawMessage(`"length(@)"`)),
		identity.WithUserSearchLocation(location),
	)
	if err != nil {
		log.Fatal(err)
	}
	printJSON(searched.Value)
}
```

## References

- [User](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/user.py)
- [Proxy.users](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1429)
- [Cloud list_users/search_users](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L290)
- [Policy](https://docs.openstack.org/keystone/2026.1/configuration/policy)
- [Python Record Find](python-record-find.md)
