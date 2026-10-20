# Cloud User Role Get Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_identity/IdentityCloudMixin/get_role`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetRoleRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewRoleReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.RoleReadCloud.RawAPI`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleGetDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleGetDomainIDJSON`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleGetFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleGetExpression`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleGetMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithRoleGetLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity.RoleReadCloud.GetRole`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithClosedLookupOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithClosedLookupHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithClosedLookupMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithClosedLookupLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.ResolveRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithClosedLookupDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithClosedLookupDomainIDJSON`

- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Fixed resolver sends raw nullable/false/zero/empty/repeated/object-key domain query and constructor seed, preserves actual response overwrite/descriptor defaults and independently copied view/wire/page/header/status evidence.
  Tests: `identity/v3/roles/python_closed_lookup_test.go:TestPythonRoleClosedLookupDomainSeedRawQueryAndDescriptorEvidence`.
- Source100..399 completion and only clean400/403/404 GET fallback retain name/domain hint, exact ID-or-name selection, missing nil, late page failure and immediate second-match ambiguity. Per-call MV applies GET only; fallback keeps selected source version.
  Tests: `identity/v3/roles/python_closed_lookup_test.go:TestPythonRoleClosedLookupCleanStatusFallbackAndLiteralNameHint`, `identity/v3/roles/python_closed_lookup_test.go:TestPythonRoleClosedLookupMissingDuplicateLatePageAndGetOnlyMicroversion`.
- Closed rawdomain/header/location opts are immutable; nil/context/custom cancellation/source mutation/private guard block work without fake receipt. Malformed accepted representation retains actual response. Existing profiles/old native behavior are untouched.
  Tests: `identity/v3/roles/python_closed_lookup_test.go:TestPythonRoleClosedLookupImmutableOptsSourceCancellationAndExplicitLocation`, `identity/v3/roles/python_closed_lookup_test.go:TestPythonRoleClosedLookupPrivateGuardAndMalformedAcceptedReceipt`.
- Physical accepted/rejected/native401 read/close/mixed matrix preserves actual body/header/status and all causes, suppresses every Retry/Reauth/Backoff callback and prevents403/404 fallback; clean renewal/retry keeps frozen domain/header/MV with live token.
  Tests: `identity/v3/roles/python_closed_lookup_test.go:TestPythonRoleClosedLookupPhysicalFaultStopsFallbackAndNativeCallbacks`, `identity/v3/roles/python_closed_lookup_test.go:TestPythonRoleClosedLookupCleanAuthRetryKeepsFrozenRequest`.
- None/literalnull uses fixed domain-aware finder; non-None falsey filters use search and return its first entity without a final find. Role search always forwards raw nullable/false/zero/empty/repeated domain and does not put nameOrID in server name query.
  Tests: `identity/role_record_get_test.go:TestPythonCloudGetRoleFiltersNoneVersusFalseySearchAndNoFinalFind`, `identity/role_record_get_test.go:TestPythonCloudGetRoleRawFalseyDomainAlwaysForwarded`.
- Role full search and loaded nested/exact/glob predicates keep duplicate IDs, missing and ordered mismatch short circuit, detached actual chosen record; late HTTP precedes local failure and unvisited invalid filters on empty selection remain unvisited.
  Tests: `identity/role_record_get_test.go:TestPythonCloudGetRoleFilteredGlobLoadedMappingDuplicatesAndMissing`, `identity/role_record_get_test.go:TestPythonCloudGetRoleLateHTTPPrecedesLocalErrorAndKeepsActualInventory`.
- Role follows truthiness/len/integer0: None/[null] returnsNone; [false]/[0] retain selected values; one-codepoint string selects itself, >1 is ambiguity, one-member object is integer-key error and truthy scalar has no len. Arbitrary expression JSON has no fake Role record.
  Tests: `identity/role_record_get_test.go:TestPythonCloudGetRolePolymorphicTruthLenAndIntegerZero`.
- Role fixed default finder retains400/403/404 name/domain fallback, missing/duplicate/late proof and GET-only selected override; copied opts/current location and one callback preserve frozen source headers, nil/owner/caller/private guard and physical401 original/native reauth error chain with zero retry callbacks.
  Tests: `identity/role_record_get_test.go:TestPythonCloudGetRoleDefaultFinderFallbackMissingDuplicateAndVersion`, `identity/role_record_get_test.go:TestPythonCloudGetRoleOptsOnceOwnValuesSourceSnapshotAndLocation`, `identity/role_record_get_test.go:TestPythonCloudGetRoleOwnerCancellationPrivateGuardAndPhysical401`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- The named Cloud signatures expose only name_or_id/filters/domain_id; the SDK-owned closed resolver models that reachable subset plus concrete Go Headers/Microversion/Location. Generic named find_user/find_role **query and Resource/adapter cache/session/config lifecycle remain separate unresolved operations; no wholesale Find or supported parity promotion is claimed.
- Concrete immutable JSON descriptor records/options and captured source/context replace Python mutable Resource/dirty/subclass object identity. Source loaded primitive/filter policies reuse existing owned profiles. Strict UTF8/JSON representation/safe routes/headers and bounded pagination/link repairs intentionally differ from Python arbitrary values and malformed-body no-op behavior.
- Direct selected client/factory is SourceNoConnection and defaults location:null. Explicit immutable CloudLocation or root Connection getter bridge supplies current scope, while SDK-R1/C1/S1 whole Resource/Connection/session lifecycle remains tracked outside this named mapping.
- Go wraps concrete invalid-option/ambiguity/native/ResponseError/cancellation causes instead of Python TypeError/KeyError/SDKException class identity or process-global deprecation warnings. Source opaque/Resource JMES function/number/representation behavior is bounded by the documented Go JSON evaluator; this is not a claim of every Python runtime object/function type match.
- Inventory is only actual deprecated search rows; final closed resolver exposes only selected Lookup or an actual error receipt, not every discarded fallback candidate row. Returned record/Value is independent of search observation; expression JSON never receives invented resource/HTTP evidence.
- Role filtered expression returns JSON via Value/Expression with Role=nil; ordinary entities[0] selection is a detached actual RoleRecord. Literalnull and selected [null] are success Value=null, while an error has Value=nil. DeprecatedFilters records Source non-None filter branch without warning output.

### `python:cloud/_identity/IdentityCloudMixin/get_user`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetUserRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity.NewUserReadCloud`
- `github.com/JSYoo5B/go-openstacksdk/identity.UserReadCloud.RawAPI`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserGetDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserGetDomainIDJSON`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserGetFilters`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserGetExpression`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserGetMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity.WithUserGetLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity.UserReadCloud.GetUser`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithClosedLookupOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithClosedLookupHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithClosedLookupMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithClosedLookupLocation`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ResolveRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithClosedLookupDomainID`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithClosedLookupDomainIDJSON`

- Projects, Users, Groups and Roles FindIdentity with WithIdentityFindQuery("domain_id", ...) send GET /<kind>/<name>?domain_id=... and then the list with domain_id and name, and Domains FindIdentity sends the GET then the name list, matching the Proxy.find_* requests behind the cloud get_* helpers.
  Tests: `identity/v3/python_cloud_parity_test.go:TestPythonCloudIdentityUnresolvedPartialSequences`.
- Fixed resolver sends raw nullable/false/zero/empty/repeated/object-key domain query and constructor seed, preserves actual response overwrite/descriptor defaults and independently copied view/wire/page/header/status evidence.
  Tests: `identity/v3/users/python_closed_lookup_test.go:TestPythonUserClosedLookupDomainSeedRawQueryAndDescriptorEvidence`.
- Source100..399 completion and only clean400/403/404 GET fallback retain name/domain hint, exact ID-or-name selection, missing nil, late page failure and immediate second-match ambiguity. Per-call MV applies GET only; fallback keeps selected source version.
  Tests: `identity/v3/users/python_closed_lookup_test.go:TestPythonUserClosedLookupCleanStatusFallbackAndLiteralNameHint`, `identity/v3/users/python_closed_lookup_test.go:TestPythonUserClosedLookupMissingDuplicateLatePageAndGetOnlyMicroversion`.
- Closed rawdomain/header/location opts are immutable; nil/context/custom cancellation/source mutation/private guard block work without fake receipt. Malformed accepted representation retains actual response. Existing profiles/old native behavior are untouched.
  Tests: `identity/v3/users/python_closed_lookup_test.go:TestPythonUserClosedLookupImmutableOptsSourceCancellationAndExplicitLocation`, `identity/v3/users/python_closed_lookup_test.go:TestPythonUserClosedLookupPrivateGuardAndMalformedAcceptedReceipt`.
- Physical accepted/rejected/native401 read/close/mixed matrix preserves actual body/header/status and all causes, suppresses every Retry/Reauth/Backoff callback and prevents403/404 fallback; clean renewal/retry keeps frozen domain/header/MV with live token.
  Tests: `identity/v3/users/python_closed_lookup_test.go:TestPythonUserClosedLookupPhysicalFaultStopsFallbackAndNativeCallbacks`, `identity/v3/users/python_closed_lookup_test.go:TestPythonUserClosedLookupCleanAuthRetryKeepsFrozenRequest`.
- None/literalnull filters use the fixed finder, but every other falsey value triggers deprecated search. Successful search len1 still calls find_user with original identity/domain and returns its actual different record, preserving the pinned missing-return bug and detached search/final proof.
  Tests: `identity/user_record_get_test.go:TestPythonCloudGetUserFiltersNoneVersusAllFalseyPresence`, `identity/user_record_get_test.go:TestPythonCloudGetUserPreservesSuccessfulSearchThenFindBugAndRealReceipts`.
- User search omits falsey domain and supplies non-UUID server name hint; later finder always receives the same domain including None/false/0/empty. UUID normalized names skip list name hint. No arbitrary Source find kwargs are exposed.
  Tests: `identity/user_record_get_test.go:TestPythonCloudGetUserRawDomainTruthinessSearchVersusFixedFinder`.
- User only tests search truth and len before the final finder: one-element object/string/[null]/[false]/[0] still proceeds; falsey returnsNone, >1 ambiguity and truthy scalar len error stop. No entity[0] indexing or expression-to-record evidence is invented.
  Tests: `identity/user_record_get_test.go:TestPythonCloudGetUserPolymorphicSearchUsesLenOnlyNeverIndexZero`.
- Complete search materialization/ordered loaded local predicates preserve missing/duplicate/short-circuit/localerror and late HTTP priority; successful gate followed by missing/duplicate/late final finder keeps actual prior Inventory but no returned User/Value on error.
  Tests: `identity/user_record_get_test.go:TestPythonCloudGetUserSearchStopsOnMissingDuplicateLocalErrorAndLateHTTP`, `identity/user_record_get_test.go:TestPythonCloudGetUserSearchGateThenFinalFinderMissingFallbackAndLateFailure`.
- Options execute once after selected source capture, copy mutable rawdomain/filter/header/current location, stop owner/cancel/private guard callbacks and preserve actual physical403 from the final finder without fallback/retry/reauth.
  Tests: `identity/user_record_get_test.go:TestPythonCloudGetUserOptionsOnceOwnedFilterDomainLocationAndFrozenSource`, `identity/user_record_get_test.go:TestPythonCloudGetUserOptionCancellationOwnerAndPrivateGuardStopWork`, `identity/user_record_get_test.go:TestPythonCloudGetUserPhysicalForbiddenNeverFindFallsThroughOrRetries`.

Python/Go differences:

- Python first calls utils.ensure_service_version to require an identity v3 adapter; Go uses the identity v3 service from Connection.Identity without that check.
- With the deprecated filters argument Python runs search_users but still returns the find_user result.
- Python returns openstacksdk identity Resource objects; Go returns the Gophercloud identity v3 model pointers (enabled is Enabled, and unknown Keystone fields are only in Extra where the model declares it).
- The named Cloud signatures expose only name_or_id/filters/domain_id; the SDK-owned closed resolver models that reachable subset plus concrete Go Headers/Microversion/Location. Generic named find_user/find_role **query and Resource/adapter cache/session/config lifecycle remain separate unresolved operations; no wholesale Find or supported parity promotion is claimed.
- Concrete immutable JSON descriptor records/options and captured source/context replace Python mutable Resource/dirty/subclass object identity. Source loaded primitive/filter policies reuse existing owned profiles. Strict UTF8/JSON representation/safe routes/headers and bounded pagination/link repairs intentionally differ from Python arbitrary values and malformed-body no-op behavior.
- Direct selected client/factory is SourceNoConnection and defaults location:null. Explicit immutable CloudLocation or root Connection getter bridge supplies current scope, while SDK-R1/C1/S1 whole Resource/Connection/session lifecycle remains tracked outside this named mapping.
- Go wraps concrete invalid-option/ambiguity/native/ResponseError/cancellation causes instead of Python TypeError/KeyError/SDKException class identity or process-global deprecation warnings. Source opaque/Resource JMES function/number/representation behavior is bounded by the documented Go JSON evaluator; this is not a claim of every Python runtime object/function type match.
- Inventory is only actual deprecated search rows; final closed resolver exposes only selected Lookup or an actual error receipt, not every discarded fallback candidate row. Returned record/Value is independent of search observation; expression JSON never receives invented resource/HTTP evidence.
- Pinned get_user intentionally discards the single filtered search entity and performs find_user, so a different final resource or final missing/error is preserved. SearchValue/Expression/Inventory describe the gate only; Value/User/Lookup describe the actual final resolution. Process-global deprecation warning is represented by DeprecatedFilters.

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
    user = os.environ["IDENTITY_USER_REF"]
    role = os.environ["IDENTITY_ROLE_REF"]
    domain = os.getenv("IDENTITY_DOMAIN_ID")
    print_value(conn.get_user(user, domain_id=domain))
    print_value(conn.get_user(user, filters={}, domain_id=domain))
    # Pinned Source checks len([None]) then still calls find_user.
    print_value(conn.get_user(user, filters="`[null]`", domain_id=domain))
    print_value(conn.get_role(role, domain_id=domain))
    # Role returns entities[0], so this expression returns None.
    print_value(conn.get_role(role, filters="`[null]`", domain_id=domain))


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

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity"
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
	conn, err := sdk.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	user, role := os.Getenv("IDENTITY_USER_REF"), os.Getenv("IDENTITY_ROLE_REF")
	if user == "" || role == "" {
		log.Fatal("set IDENTITY_USER_REF and IDENTITY_ROLE_REF to readable IDs or names")
	}
	userOpts := []identity.UserGetOption{}
	roleOpts := []identity.RoleGetOption{}
	if domain, ok := os.LookupEnv("IDENTITY_DOMAIN_ID"); ok {
		userOpts = append(userOpts, identity.WithUserGetDomainID(domain))
		roleOpts = append(roleOpts, identity.WithRoleGetDomainID(domain))
	}
	directUser, err := conn.GetUserRecord(ctx, user, userOpts...)
	if err != nil {
		log.Fatal(err)
	}
	printValue(directUser.Value)
	filteredUser, err := conn.GetUserRecord(ctx, user, append(userOpts,
		identity.WithUserGetFilters(json.RawMessage(`{}`)))...)
	if err != nil {
		log.Fatal(err)
	}
	printValue(filteredUser.Value)
	pinnedUser, err := conn.GetUserRecord(ctx, user, append(userOpts,
		identity.WithUserGetExpression("`[null]`"))...)
	if err != nil {
		log.Fatal(err)
	}
	printValue(pinnedUser.Value)
	directRole, err := conn.GetRoleRecord(ctx, role, roleOpts...)
	if err != nil {
		log.Fatal(err)
	}
	printValue(directRole.Value)
	filteredRole, err := conn.GetRoleRecord(ctx, role, append(roleOpts,
		identity.WithRoleGetExpression("`[null]`"))...)
	if err != nil {
		log.Fatal(err)
	}
	printValue(filteredRole.Value)
}
```

## References

- [get_user](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L344)
- [get_role](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1422)
- [Proxy find_user/find_role](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py)
- [User](https://docs.openstack.org/keystone/2026.1/_modules/keystone/common/policies/user.html)
- [Role](https://docs.openstack.org/keystone/2026.1/_modules/keystone/common/policies/role.html)
- [User](v3/users/python_records.md)
- [Role](v3/roles/python-records.md)
