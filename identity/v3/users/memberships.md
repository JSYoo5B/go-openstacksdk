# Memberships

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:identity/v3/users.ListProjects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ListProjects`

- GET users/{userID}/projects or groups retains the selected endpoint prefix, native auth token and empty initial query; each collection uses its own Project/Group type and known/unknown native model fields.
  Tests: `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsDecodeTheirOwnModelsAcrossPages`.
- Native links.next advances the typed iterator across pages; empty 200 collections and 204 responses yield no rows.
  Tests: `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsDecodeTheirOwnModelsAcrossPages`, `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsEmptyAndNoContent`.
- HTTP 403, malformed known fields and cancellation are yielded as errors; consumer break prevents further page requests, and cancellation before iteration prevents the first request.
  Tests: `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsPropagateErrorsAndStopEarly`.
- A later-page HTTP/decode failure or cancellation leaves the already-yielded row observable and produces one terminal error.
  Tests: `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsRetainRowsBeforeContinuationFailure`.
- Generation follows the returned foreign page owner before local name fallbacks, preserves the existing users and local Identity v2 extension extractors, and retains the pinned Trove imported Instance extractor.
  Tests: `internal/cmd/sdkgen/page_extractors_test.go:TestDeclaredPageExtractorUsesReturnedPageInsteadOfNamesOrNestedFields`, `internal/cmd/sdkgen/page_extractors_test.go:TestPinnedForeignPagesAndLocalWrapperGenerateCompatibleIterators`, `api/identity_user_memberships_test.go:TestIdentityLocalUserAndExtensionPageExtractorsRemainSeparate`.

Python/Go differences:

- The native Pager becomes a context-bound lazy iter.Seq2 of pointers to the native Project alias; consumers can stop without fetching the next page and receive partial rows before a terminal error.
- The formerly generated *User return type and ExtractUsers assertion were incorrect for this native page. The public iterator now returns *Project; callers using the old element type must update it.

### `gophercloud:identity/v3/users.ListGroups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ListGroups`

- GET users/{userID}/projects or groups retains the selected endpoint prefix, native auth token and empty initial query; each collection uses its own Project/Group type and known/unknown native model fields.
  Tests: `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsDecodeTheirOwnModelsAcrossPages`.
- Native links.next advances the typed iterator across pages; empty 200 collections and 204 responses yield no rows.
  Tests: `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsDecodeTheirOwnModelsAcrossPages`, `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsEmptyAndNoContent`.
- HTTP 403, malformed known fields and cancellation are yielded as errors; consumer break prevents further page requests, and cancellation before iteration prevents the first request.
  Tests: `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsPropagateErrorsAndStopEarly`.
- A later-page HTTP/decode failure or cancellation leaves the already-yielded row observable and produces one terminal error.
  Tests: `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsRetainRowsBeforeContinuationFailure`.
- Generation follows the returned foreign page owner before local name fallbacks, preserves the existing users and local Identity v2 extension extractors, and retains the pinned Trove imported Instance extractor.
  Tests: `internal/cmd/sdkgen/page_extractors_test.go:TestDeclaredPageExtractorUsesReturnedPageInsteadOfNamesOrNestedFields`, `internal/cmd/sdkgen/page_extractors_test.go:TestPinnedForeignPagesAndLocalWrapperGenerateCompatibleIterators`, `api/identity_user_memberships_test.go:TestIdentityLocalUserAndExtensionPageExtractorsRemainSeparate`.

Python/Go differences:

- The native Pager becomes a context-bound lazy iter.Seq2 of pointers to the native Group alias; consumers can stop without fetching the next page and receive partial rows before a terminal error.
- The formerly generated *User return type and ExtractUsers assertion were incorrect for this native page. The public iterator now returns *Group; callers using the old element type must update it.

### `python:identity/v3/user_projects`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ListProjects`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ListProjectRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithProjectListOptions`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithProjectListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithProjectListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithProjectListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithProjectListHeader`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithProjectListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithProjectListFilter`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.WithProjectListFilters`

- GET users/{userID}/projects or groups retains the selected endpoint prefix, native auth token and empty initial query; each collection uses its own Project/Group type and known/unknown native model fields.
  Tests: `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsDecodeTheirOwnModelsAcrossPages`.
- Native links.next advances the typed iterator across pages; empty 200 collections and 204 responses yield no rows.
  Tests: `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsDecodeTheirOwnModelsAcrossPages`, `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsEmptyAndNoContent`.
- HTTP 403, malformed known fields and cancellation are yielded as errors; consumer break prevents further page requests, and cancellation before iteration prevents the first request.
  Tests: `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsPropagateErrorsAndStopEarly`.
- A later-page HTTP/decode failure or cancellation leaves the already-yielded row observable and produces one terminal error.
  Tests: `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsRetainRowsBeforeContinuationFailure`.
- The additive SDK-owned ListProjectRecords iterator is lazy and applies options once per iteration while preserving the fixed explicit user parent and selected Identity endpoint/token. All eleven canonical queries and fifteen spellings are classified from the pinned UserProject inheritance; canonical bulk values win over wire aliases, including null/false/empty values, while unknown or superseded invalid values are discarded. Individually repeated aliases use the final value, input maps/scalar slices remain owned, and operation-local headers and the explicit identity 3.13 microversion reach HTTP. The static descriptor is independently linked to the UserProject manifest, whose exact class/URI/body/query/MRO and six fixed source hashes are verified against fresh AST extraction; User/Project native collection metadata cannot substitute for this owned binding.
  Tests: `api/identity_user_project_records_test.go:TestIdentityUserProjectRecordsSemanticQueriesAliasesAndOwnedOptions`, `identity/v3/users/project_records_descriptor_test.go:TestUserProjectRecordDescriptorMatchesPinnedOwnedManifest`, `internal/cmd/sdkgen/python_user_project_filters_test.go:TestUserProjectPythonFilterManifestKeepsOwnedParentAndInheritedDescriptors`, `internal/cmd/sdkgen/python_user_project_filters_test.go:TestUserProjectPythonFilterVerificationReadsFreshSourceAndRejectsProofDrift`.
- The four declared local Body filters id/description/options/links are evaluated after server query classification without becoming wire parameters; nested objects use subset matching and query-only names are left to the server. Each successful row owns separate Resource and actual Wire raw fields, exact extension numbers, status and headers. Resource binds the requested user_id and maps nonnull nonobject options to an empty object while preserving absent/null options; Wire retains the response user_id and original false/array values. View, wire and rows do not alias. The cached Connection Identity/IdentityV3 service and its Users API share the selected provider/resource base and use live authentication.
  Tests: `api/identity_user_project_records_test.go:TestIdentityUserProjectRecordsLocalBodiesNormalizedViewAndIndependentWire`.
- The owned iterator preserves initial filters across dictionary links.next, rel/href links or projects_links, top-level next and HTTP Link continuations. Explicit limits use the last raw ID for marker fallback even on short nonempty pages, and empty pages terminate. SinglePage consumes only the first page; MaxItems supplies an omitted wire limit hint and caps raw rows before Body matching, leaving trailing malformed unconsumed rows untouched. Consumed malformed rows preserve whole-page HTTP evidence and any prior matching row. Shared continuation validation rejects foreign collection paths, widened filters, conflicting next sources and cycles rather than adopting Python precedence; shared accepted 204 handling terminates without JSON or continuation decoding.
  Tests: `api/identity_user_project_records_test.go:TestIdentityUserProjectRecordsContinuationFormsRawCapsAndSinglePage`, `internal/rest/list_test.go:TestListDictionaryLinksAreExplicitAndGuarded`, `internal/rest/list_test.go:TestListContinuationCannotWidenCollectionFiltersOrMarkers`, `internal/rest/list_test.go:TestListHTTPLinksAreOptInAndConflictingNextLinksFail`, `internal/rest/list_test.go:TestListCyclesRejectRepeatedMarkerBeforeHTTP`, `internal/rest/list_no_content_test.go:TestListAcceptedNoContentTerminatesWithoutDecodingOrContinuation`.
- Lazy preflight rejects nil options, invalid explicit user IDs, duplicate parent user_id in semantic/raw query, semantic-versus-wire collisions and selected object query values before HTTP; a source-changing callback stops subsequent callbacks and requests. Pre-cancel and cancellation after a yielded row preserve both context.Canceled and the custom cause. Accepted read/Close/decode failures retain whole response body/status/header and original causes with one physical request and one body Close; later native 403 or malformed-page failures retain the earlier row and one terminal error. The reused per-attempt SourceGuard stops native retries after source changes and preserves accepted HTTP evidence.
  Tests: `api/identity_user_project_records_test.go:TestIdentityUserProjectRecordsPreflightOwnedFaultsAndPartialErrors`, `internal/rest/collection_source_guard_test.go:TestCollectionSourceGuardStopsRetryAcrossOperations`, `internal/rest/collection_source_guard_test.go:TestCollectionSourceGuardKeepsAcceptedResponseEvidence`.

Python/Go differences:

- Python accepts a user ID or User Resource without fetching that user first; Go ListProjectRecords requires an explicit nonempty single unescaped URL path segment. Both the semantic and raw user_id query are rejected before HTTP to keep the parent fixed; Python's public user_projects raises a duplicate-keyword TypeError for an overriding user_id. Native ListProjects remains queryless with its Project alias and existing pager/signature.
- The named result maps to an owned UserProjectRecord with independent RawResource Resource/Wire fields, rather than a mutable Python UserProject Resource. Resource copies actual fields, binds the selected user_id and applies only the local options dict conversion (nonnull nonobject becomes {}, absent/null stays distinct). It does not synthesize response is_enabled aliases, Python boolean truthiness, missing tags=[], absent descriptor defaults, computed location, connection/cache state, or strip/passively fetch self. These broader Resource lifecycle capabilities remain in the original SDK goal; mapping this list does not enable UserProject CRUD or change native Project CRUD.
- Semantic fields use the pinned eleven-query/four-Body classification and canonical bulk alias precedence. Go query scalars use lowercase bools/exact JSON numbers/repeated scalar arrays, omit null URL values and reject selected objects; this is an explicit typed serialization policy rather than Python/requests dynamic stringification. Unknown semantic values are discarded, whereas explicit unknown wire queries are a Go extension; semantic-versus-typed/raw query overlaps are errors even for equal values. Local Body matching compares exact JSON types and recursive subsets of nonempty objects, deliberately distinguishing booleans from numbers unlike Python equality.
- Typed Limit=0 and empty Marker omit those wire keys, MaxItems=0 is unlimited, negative limits/caps are rejected, and nil Paginated means all pages. Limits and markers that are sent must be a positive single limit and nonempty single marker. MaxItems counts raw rows and provides an omitted limit hint before local filtering. Concrete header and microversion options are supported; nil microversion preserves the selected Identity client and no Python discovery/_max_microversion=None selection is added. Python-only base_path, deprecated jmespath_filters, allow_unknown_params, resource_type and session controls are reserved instead of exposing arbitrary paths or Python execution.
- The owned operation accepts strict HTTP200 projects arrays and native-compatible HTTP204 empty results; it requires nonnull object rows instead of Python's dynamic non-list-to-singleton envelope handling and adapter-wide accepted statuses. Raw JSON preserves extension precision and null/omission independently of native Project known-field decoding. Accepted response read/Close/decode failures are not resent by the SDK composition; provider-configured native retries and reauthentication remain separate.
- The shared owned pager supports Python rel/href/top-level-next/HTTP-Link/marker routes and adds native dictionary links.next/projects_links compatibility. It validates all advertised channels and rejects conflicting destinations, foreign origins/paths, replacement filters, repeated markers or URLs; Python uses ordered continuation precedence and may normalize paths differently. Raw string IDs are required only when a fallback marker is needed. Cap, consumer break and empty-page termination preserve lazy behavior; shared helper tests supplement the four operation HTTP groups rather than claiming every helper boundary was independently repeated for this endpoint.

### `python:identity/v3/user_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ListGroups`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/users.API.ListGroupRecords`

- GET users/{userID}/projects or groups retains the selected endpoint prefix, native auth token and empty initial query; each collection uses its own Project/Group type and known/unknown native model fields.
  Tests: `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsDecodeTheirOwnModelsAcrossPages`.
- Native links.next advances the typed iterator across pages; empty 200 collections and 204 responses yield no rows.
  Tests: `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsDecodeTheirOwnModelsAcrossPages`, `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsEmptyAndNoContent`.
- HTTP 403, malformed known fields and cancellation are yielded as errors; consumer break prevents further page requests, and cancellation before iteration prevents the first request.
  Tests: `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsPropagateErrorsAndStopEarly`.
- A later-page HTTP/decode failure or cancellation leaves the already-yielded row observable and produces one terminal error.
  Tests: `api/identity_user_memberships_test.go:TestIdentityUserMembershipListsRetainRowsBeforeContinuationFailure`.
- The additive ListGroupRecords iterator uses the explicit user parent and selected cached Identity/IdentityV3 service without an eager user or group lookup. Its public signature has no options/query, and the first GET retains the endpoint/resource-base prefix, live authentication, selected client headers and no body/query. Repeating the lazy sequence issues a fresh request. Successful UserGroupRecord values retain independent actual Wire and parent-bound Resource raw fields, HTTP status/header, nullable known fields and exact large vendor numbers; Resource sets only user_id and does not apply Project's options dict conversion to Group options:false.
  Tests: `api/identity_user_group_records_test.go:TestIdentityUserGroupRecordsOwnedViewAndCachedConnection`.
- The owned Group binding consumes default advertised dictionary links.next, rel/href links, groups_links, top-level next and HTTP Link continuations, retaining the fixed collection path and live token on the second page. Its first request has no query. The first advertised next may introduce one positive server limit, which is retained if later next links omit it; a later introduction, changed limit, zero/negative/duplicate limit or added nonpagination filter is rejected with the actual accepted page evidence. A server-introduced limit does not enable marker fallback: a short second page without next ends after two requests. Empty advertised pages and actual204 yield no rows; consumer break stops before unconsumed rows or another request. Shared membership paging validates conflicting channels, foreign paths/origins, widened filters outside that first server-limit allowance and repeated marker/URL cycles; no initial caller query or artificial marker/limit request is added.
  Tests: `api/identity_user_group_records_test.go:TestIdentityUserGroupRecordsContinuationFormsAndConsumerStop`, `internal/rest/list_test.go:TestListFirstServerLimitIsOptInAndFixedAcrossContinuations`, `internal/rest/list_test.go:TestListDictionaryLinksAreExplicitAndGuarded`, `internal/rest/list_test.go:TestListContinuationCannotWidenCollectionFiltersOrMarkers`, `internal/rest/list_test.go:TestListHTTPLinksAreOptInAndConflictingNextLinksFail`, `internal/rest/list_test.go:TestListCyclesRejectRepeatedMarkerBeforeHTTP`.
- Invalid explicit IDs, a nil API and pre-cancelled custom causes yield one terminal error without HTTP. Accepted read/Close faults preserve complete body/header/status and the original cause with one physical request and one Close; actual403 preserves native status/body/headers. Later malformed objects, cancellation after a yielded row and late source replacement preserve the prior row and one terminal failure. Source drift during the first or later accepted transport retains the accepted response evidence and rejects a successful record; context cancellation preserves context.Canceled and the custom cause.
  Tests: `api/identity_user_group_records_test.go:TestIdentityUserGroupRecordsPreflightFaultsAndPartialResults`.

Python/Go differences:

- Python user_groups accepts a user ID or User Resource and performs no user fetch; Go ListGroupRecords requires an explicit nonempty single unescaped URL path segment and validates it before HTTP. There is no public **query/options argument in either mapping, so the inherited Group query declaration and Resource.list controls are not invented as missing inputs for this named function. Native ListGroups remains unchanged with its Group alias, signature and pager.
- The named result maps to SDK-owned UserGroupRecord with independent RawResource Resource/Wire fields, rather than a mutable Python UserGroup Resource. Resource copies actual JSON and binds the requested user_id; Wire keeps the actual response user_id or omission. RawResource retains nullable known fields, extension precision and unknown raw values instead of applying native Group string/Links decoding or Python Group's disabled unknown-Body-retention/default/computed-location/connection policies. Group vendor options:false remains false; only UserProject uses the options dict descriptor normalization. Passive self/links data are retained without fetching or resource-routing methods.
- The owned operation uses strict HTTP200 groups arrays plus native-compatible HTTP204 emptiness, requiring nonnull object rows instead of Python adapter-wide successful statuses and dynamic non-list-to-singleton envelopes. Selected Identity headers/microversion/provider policy are preserved; this entry does not run Python discovery/_max_microversion=None selection or expose operation-local version/list controls.
- Shared owned membership paging supports source rel/href/top-level-next/HTTP-Link routes and adds groups_links/native dictionary links.next compatibility. The first request remains queryless; the first advertised next may add one positive server limit, which is fixed thereafter and retained when later links omit it. This does not activate marker fallback because the original source Resource.list call captures initial limit=None. Every advertised channel is validated, with conflicting destinations, foreign origins/paths, other added/replaced filters, invalid or later-changed limits and cycles rejected rather than following Python's ordered continuation precedence and path normalization. The entry creates no initial marker/limit or fallback request and stops on empty pages or when no next is advertised.
- Accepted response read/Close/decode/source failures retain original HTTP evidence and are not resent by SDK composition; provider-configured native retry and reauthentication are separate. Explicit user IDs, RawResource/result ownership and fixed source guards are intentional Go mapping policies. Full mutable Resource/session/cache and inherited Group actions remain in the original SDK goal and are not promoted by completion of this named list; UserGroup list-only capabilities do not disable native Group CRUD.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
user_id = "user-id"

for project in conn.identity.user_projects(user_id):
    print(project.id, project.name, project.domain_id)

for group in conn.identity.user_groups(user_id):
    print(group.id, group.name, group.domain_id)
```

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s USER_ID", os.Args[0])
	}
	userID := os.Args[1]
	conn, err := sdk.Connect(ctx)
	if err != nil {
		return err
	}
	identity, err := conn.Identity(ctx)
	if err != nil {
		return err
	}

	for project, err := range identity.Users.ListProjects(ctx, userID) {
		if err != nil {
			return fmt.Errorf("list user projects: %w", err)
		}
		fmt.Printf("project: %s %s domain=%s\n", project.ID, project.Name, project.DomainID)
	}
	for group, err := range identity.Users.ListGroups(ctx, userID) {
		if err != nil {
			return fmt.Errorf("list user groups: %w", err)
		}
		fmt.Printf("group: %s %s domain=%s\n", group.ID, group.Name, group.DomainID)
	}
	return nil
}
```

```python
import openstack

conn = openstack.connect()
for project in conn.identity.user_projects(
    "user-id", is_enabled=True, limit=20, max_items=50
):
    print(project.id, project.name, project.domain_id)
```

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/identity/v3/users"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: %s USER_ID", os.Args[0])
	}
	conn, err := sdk.Connect(ctx)
	if err != nil {
		return err
	}
	identity, err := conn.Identity(ctx)
	if err != nil {
		return err
	}
	for record, err := range identity.Users.ListProjectRecords(ctx, os.Args[1],
		users.WithProjectListOptions(users.ListProjectRecordsOpts{Limit: 20, MaxItems: 50}),
		users.WithProjectListFilter("is_enabled", true),
	) {
		if err != nil {
			return fmt.Errorf("list user project records: %w", err)
		}
		data, err := json.MarshalIndent(map[string]any{
			"resource": record.Resource,
			"wire": record.Wire,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}
	for record, err := range identity.Users.ListGroupRecords(ctx, os.Args[1]) {
		if err != nil {
			return fmt.Errorf("list user group records: %w", err)
		}
		data, err := json.MarshalIndent(map[string]any{
			"resource": record.Resource,
			"wire": record.Wire,
		}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	}
	return nil
}
```

## References

- [user_projects](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1112-L1131)
- [user_groups](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1414-L1429)
- [UserProject manifest](../../../api/openstacksdk/resources/identity/v3/user_project.json)
