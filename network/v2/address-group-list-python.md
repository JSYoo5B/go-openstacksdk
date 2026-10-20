# Address Group List Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/address_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilter`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilters`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithBodyFilter`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithBodyFilters`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPaginated`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPageSize`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithName`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithStatus`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk.Connection.Network`
- `github.com/JSYoo5B/go-openstacksdk.Connection.NetworkV2`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithRecordListPaginated`

- Ordinary AddressGroup.Resources.List/All now classify the entire pinned direct declaration through SDK-owned WithFilter/WithFilters: eight canonical and accepted query attributes, three non-query Body attributes id/tenant_id/addresses and nine reserved list controls. AddressGroup directly inherits Resource without NetworkResource or TagMixin. Query name/description/project_id are server-only; no exact local name/project predicate, implicit GET or caller builder is added. Generated leaf and cached Network/NetworkV2 facades share the descriptor.
  Tests: `api/address_group_list_filters_test.go:TestAddressGroupListFiltersEntireQueryDescriptor`, `api/address_group_list_filters_test.go:TestAddressGroupListFiltersRawBodyDescriptorAndPresence`, `connection_address_group_filters_test.go:TestConnectionAddressGroupFiltersSharedFacadesAndRawTenant`.
- Semantic query values encode scalar strings, lowercase booleans and original JSON-number text; flat scalar arrays preserve repeated order while omitting null elements. Null and empty arrays emit no URL values but retain key presence. Unknown semantic names and their encoding errors are discarded; selected maps/nested query arrays fail. The deprecated tenant_id remains a local property: project_id's response descriptor alias does not introduce a tenant_id query alias.
  Tests: `api/address_group_list_filters_test.go:TestAddressGroupListFiltersEntireQueryDescriptor`, `api/address_group_list_filters_test.go:TestAddressGroupListFiltersLazyPreflightAndNamespaceCollisions`, `connection_address_group_filters_test.go:TestConnectionAddressGroupFiltersSharedFacadesAndRawTenant`, `resource/filters_test.go:TestSemanticFiltersCanonicalPrecedencePresenceAndExactQueryEncoding`.
- All three local selectors read original row JSON after native extraction. Literal raw id presence/null/empty stays independent of native typed identity. tenant_id has no fallback to project_id and preserves exact large numbers and unknown nested objects despite being absent from the native model. Missing/null match null, empty strings and arrays remain distinct; arrays compare complete ordered length and elements without CIDR normalization, and objects recursively match subsets of nonempty actual objects. Decimal equality is exact and booleans remain distinct from numbers.
  Tests: `api/address_group_list_filters_test.go:TestAddressGroupListFiltersRawBodyDescriptorAndPresence`, `connection_address_group_filters_test.go:TestConnectionAddressGroupFiltersSharedFacadesAndRawTenant`, `resource/body_record_test.go:TestBodyRecordPairingPreservesOriginalRowsAndOwnsFields`, `resource/body_record_test.go:TestBodyRecordFieldNormalizationPresenceAndCauses`, `internal/jsonfilter/filter_test.go:TestMatchFiltersArraysCompareExactly`, `internal/jsonfilter/filter_test.go:TestMatchFiltersExactDecimalNumbersAndHugeExponents`, `internal/jsonfilter/filter_test.go:TestMatchFiltersDeliberatelyDistinguishesBooleansAndNumbers`, `api/network_body_filters_test.go:TestNetworkBodyFiltersNativeValueSemantics`.
- Matching no longer projects the native Addresses slice: a raw null address element stays null for comparison although the returned native []string decodes that element as an empty string. Returned rows still use the unchanged five-field native model, including independent ProjectID, rather than filter values or a synthetic raw model. Entire address arrays missing/null both match null; ordered values, length and empty arrays remain explicit. Native id/name/description/project_id string types and addresses string-array decoding remain authoritative.
  Tests: `api/address_group_list_filters_test.go:TestAddressGroupListFiltersRawBodyDescriptorAndPresence`, `api/address_group_list_filters_test.go:TestAddressGroupListFiltersControlsAndNativeWholePageFailures`, `api/network_body_filters_test.go:TestNetworkBodyFiltersLocalWireAndNativeSurfaces`.
- Options own nested JSON/query snapshots for independent concurrent reuse. Individual semantic options are last-wins; bulk replace/nil/empty clear affect only semantic state and validate only final selected values, allowing superseded encoding errors to disappear. Same-wire semantic/raw/name/page-size and same-Body semantic/explicit destinations fail lazily before HTTP, including null/empty/equal values. Reserved controls, selected invalid values, unsupported bindings and canceled contexts yield terminal preflight errors. Raw query and explicit Body namespaces remain independent when their destination kinds differ.
  Tests: `api/address_group_list_filters_test.go:TestAddressGroupListFiltersSnapshotsReplacementAndConcurrentReuse`, `api/address_group_list_filters_test.go:TestAddressGroupListFiltersLazyPreflightAndNamespaceCollisions`, `connection_address_group_filters_test.go:TestConnectionAddressGroupFiltersPreflightAndNativeNameHint`, `resource/filters_test.go:TestSemanticFiltersSnapshotsConcurrentReuseAndServerOnlyName`, `resource/filters_test.go:TestSemanticFiltersValidateOnlyFinalSelectedValues`, `resource/filters_test.go:TestSemanticFiltersRejectSelectedErrorsAndReservedControlsBeforeIteration`, `resource/filters_test.go:TestSemanticFiltersNamespaceCollisionsAndIndependentClear`, `resource/body_filters_test.go:TestCollectionBodyFiltersOptionApplicationOwnsRawBytes`, `resource/body_filters_test.go:TestCollectionBodyFiltersSnapshotsBulkAliasesAndConcurrentReuse`.
- Native whole-page extraction precedes raw matching and cap consumption, so malformed known fields after a cap remain terminal with native decode causes. A null row consumed by the Body lane is invalid, while query-only null rows preserve the native zero model and an empty object remains valid. After extraction succeeds a cap or early break can avoid consuming a later null row or continuation. MaxItems counts raw rows before local Body/name predicates without refill or an inferred wire limit; first-page and break stop before continuation, and All returns nil on terminal errors.
  Tests: `api/address_group_list_filters_test.go:TestAddressGroupListFiltersControlsAndNativeWholePageFailures`, `resource/body_record_test.go:TestBodyRecordPairingRejectsUnprovenShapeAndCount`, `resource/body_record_test.go:TestCollectionBodyRecordLaneIsSelectedOnlyForPreparedFilters`, `resource/body_filters_test.go:TestCollectionBodyFiltersRunAfterRawCapsOnBothIteratorPaths`, `api/network_body_filters_test.go:TestNetworkBodyFiltersRawControlsNameAndConsumerBreak`, `api/network_body_filters_test.go:TestNetworkBodyFiltersNativePageErrorsAndLateCancellation`.
- Ordinary native listing retains HTTP200/204/300 and rejects201. A successfully parsed204 is empty, while empty application/json204 retains native EOF. address_groups_links rel=next supplies continuation; top-level links.next alone is ignored, matching duplicates remain ordinary list rows and foreign advertised URLs retain native following. Later HTTP/decode/foreign errors, cycles and cancellation remain terminal. Configured headers/current microversion, shared ProviderClient/live token, HTTP middleware, context and ResourceBase reverse prefix remain in use across pages.
  Tests: `api/address_group_list_filters_test.go:TestAddressGroupListFiltersPagingLiveSourceAndNativeStatus`, `connection_address_group_filters_test.go:TestConnectionAddressGroupFiltersSharedFacadesAndRawTenant`, `api/network_body_filters_test.go:TestNetworkBodyFiltersNativePageErrorsAndLateCancellation`.
- Ordinary raw WithQuery preserves complete wire queries, including id/tenant_id/addresses/status/vendor keys, separately from semantic classification. Existing WithName retains a server hint plus exact local predicate; semantic queried name remains server-only. WithStatus stays unsupported because the native model has no Status, while raw status stays wire-only on both normal and Body lanes. Generated native typed List and existing FindIdentity keep their separate signatures/raw query policy and do not acquire ambient semantic or Body options.
  Tests: `api/address_group_list_filters_test.go:TestAddressGroupListFiltersNativeAndIdentitySurfaceIsolation`, `connection_address_group_filters_test.go:TestConnectionAddressGroupFiltersPreflightAndNativeNameHint`, `api/network_body_filters_test.go:TestNetworkBodyFiltersLocalWireAndNativeSurfaces`.
- Generation requires the checked-in AddressGroup manifest and a live audited -openstacksdk-source checkout. Five exact source SHA files are validated before isolated AST-only extraction and full manifest comparison, including parser minor version, direct Resource C3 MRO, pagination defaults, declared query/Body classification, unknown discard and canonical precedence. Sixteen AST anchors independently prove project_id's descriptor alias, deprecated tenant_id, list-typed addresses and inherited ID/accessor behavior; no SDK imports or fabricated tenant query alias are used. Missing/stale source or manifest fails closed and prior four manifests retain their exact bytes.
  Tests: `internal/cmd/sdkgen/address_group_filters_test.go:TestAddressGroupPythonFilterManifestKeepsQueryAndLocalTenantIdentity`, `internal/cmd/sdkgen/address_group_filters_test.go:TestAddressGroupPythonFilterPinnedSourceRequiresLiveAliasAndCoercionProof`.
- The raw semantic plan requires exact native AddressGroup5/ListOpts9/Link2 field/type/tag shapes, no own model methods, the value query-builder signature and own linked-page/extractor contracts. Fourteen receiver-qualified native/root/pagination declaration hashes guard routes, native extraction, original GetBody pairing and UseNumber precision. Emission retains complete repeat/nil/empty query maps on both lanes, raw status policy, prior audited bindings and distinct explicit-Body/semantic inventory. Identity opt-ins remain twenty, Body bindings eight and semantic bindings five.
  Tests: `internal/cmd/sdkgen/address_group_filters_test.go:TestAddressGroupBodyFilterNativeContractRejectsModelBuilderPagerAndSourceDrift`, `internal/cmd/sdkgen/address_group_filters_test.go:TestAddressGroupSemanticFilterEmissionPreservesRawFieldsAndExistingBindings`, `internal/cmd/sdkgen/address_group_filters_test.go:TestAddressGroupRawDependencyLoaderPreservesNumbersAndNativeLinkHelpers`, `internal/cmd/sdkgen/address_group_filters_test.go:TestAddressGroupBodyFilterAcceptsActualCompiledNativeSchemasAndOwnMethods`.
- Lazy bodyless GET at /address-groups follows the actual Source server QueryParameters including fields unchanged, list-only local id/tenant_id/addresses filters, no identity/name hint and discarded unknown values without MarshalJSON. Source list Accept defaults application/json over captured Accept, explicit option Accept overrides. Existing exact six descriptors/project alias/list conversion/null defaults/unknown extensions are reused with independent Resource/Wire/whole Envelope/Header/status channels; no tag/NetworkResource headers are synthesized.
  Tests: `network/v2/extensions/security/addressgroups/python_list_records_contracts_test.go:TestPythonAddressGroupListSourceProjectionQueryAndEvidence`.
- Frozen common Engine.List/REST pager supplies plural/body/RFCLink/versioned-path/short-page marker continuation, retained actual scalar JSON marker conversion, complete pagination, raw caps before local filters, single page/break without next request and late native/page errors. SourceFindPolicy opt-in uses binary64 next truthiness; ±1e-400 stop cleanly while truthy nonstring next retains exact actual whole page proof. Dictionary next links are an explicit Go improvement; changed origin/path/filter/cycle are errors.
  Tests: `network/v2/extensions/security/addressgroups/python_list_records_contracts_test.go:TestPythonAddressGroupListPaginationCapsNextTruthinessAndLateErrors`.
- Actual final200..399 representations, strict other native status errors, singular plural-object/array schema, missing/null/malformed/nonobject/UTF8 page faults, constructor cls null/false plus connection/microversion/_synchronized errors and Source recursive dict truthiness remain proven. Exact-decimal local numeric comparison is separately bounded from Python binary64 truthiness and kept explicitly documented.
  Tests: `network/v2/extensions/security/addressgroups/python_list_records_contracts_test.go:TestPythonAddressGroupListConstructorDecodeAndLocalFilterFailures`.
- Read+Close/cancel cause/source-changed-and-restored-on-Close on accepted/rejected list responses preserve every physical cause, original whole body/Header/status, close exactly once, perform exactly one actual HTTP and never become empty/success/retry.
  Tests: `network/v2/extensions/security/addressgroups/python_list_records_contracts_test.go:TestPythonAddressGroupListPhysicalFaultsRetainActualProofWithoutReplay`.
- Native retry cannot alter/delete/Omit selected protocol version or add absent/foreign version, auth/framing or JSON to the bodyless GET. Original native503 proof and optional callback cause stay visible with no resend. Ordinary new retry Header/live-token/reauthentication changes remain native; exactly owned route/query is reused.
  Tests: `network/v2/extensions/security/addressgroups/python_list_records_contracts_test.go:TestPythonAddressGroupListRetryHeadersVersionAndReauthOwnership`.
- Source capture precedes callbacks; bulk maps/raw JSON/headers/version/options and retained callback results are independently owned, invalid context/options/control/header/JSON/paging/source values fail before actual HTTP with causes, unknown bulk raw values are discarded. Immutable options produce independent records under8 concurrent calls.
  Tests: `network/v2/extensions/security/addressgroups/python_list_records_contracts_test.go:TestPythonAddressGroupListOptionsPreflightSnapshotsAndConcurrentReuse`.
- Direct List per-call explicit version applies to every list page, replacing captured matching version headers on only the private operation. Explicit empty means version absence; original selected client/default/header stay unchanged and no class discovery is added.
  Tests: `network/v2/extensions/security/addressgroups/python_list_records_contracts_test.go:TestPythonAddressGroupListExplicitVersionOwnsEveryPageAndOriginalHeaders`.

Python/Go differences:

- Complete declared classification is provided through concrete SDK-owned semantic and explicit Body options. Query name/description/project_id stay server-only and tenant_id remains local, distinct from project_id's descriptor alias. WithName has its own hint and local predicate; raw query keys are explicit wire extensions and semantic filtering does not change typed List or FindIdentity.
- Individual semantic options use last-wins targets and bulk replace/clear affects only semantic state. Only final selected values are validated, while explicit Body errors keep their own behavior. Equal/null/empty same-destination collisions fail rather than permissive kwargs merging. Query scalar arrays become repeated values, null elements are omitted, booleans are lowercase, JSON numbers retain text and unknown semantic names/errors are discarded.
- Matching retains original id/tenant_id/addresses values and exact large unknown tenant JSON; returned values remain the five-field native model. Raw null array elements differ from the native empty-string projection. Python addresses(type=list) may wrap non-list response values into lists; the native decoder instead rejects incompatible known fields before matching and before a cap.
- Exact JSON equality distinguishes bools from numbers and strings from numbers. Recursive object subsets preserve the pinned empty-actual-object nonmatch; nonobject actual values are nonmatches rather than potential Python attribute-access errors. Arrays require complete order/length/element equality and do not normalize CIDRs or apply object-subset matching inside arrays.
- Native200/204/300, plural-links continuation and foreign URL following remain intact. Empty application/json204 can fail with native EOF. Local caps count raw rows like the pinned implementation but do not infer its wire limit; first-page/break stop before continuation and All discards partial rows on terminal failures. No same-origin/fixed-query or synthesized marker policy is added.
- Raw Body selectors run only for consumed selected fields after native whole-page decoding. A consumed null row is invalid on the Body lane; query-only null rows preserve the native zero model. SDK comparisons do not replace Python Resource defaults/alias/unknown response state or provide a tracked/cache lifecycle. Source/manifest/native drift fails closed without activating additional native resources or identity capabilities.
- One bounded direct Source list callable mapping reuses existing AddressGroupFindRecord (public AddressGroupListRecord alias), verified six Body/query tables and frozen common neutronfind.Engine.List with SourceFindPolicy=true. Existing Find/native Get/Create/Update/Delete/actions are not changed or recounted.
- Go JSON-domain inputs/results and concrete options own source/policy/route/version/header/paging state. Full Python mutable Resource dirty/cache/session/Adapter/Connection/computed location and object subclass behavior remain global tasks, rather than being claimed through one list row. Literal observed unknown location follows existing record model; complete Connection-derived location calculation is not synthesized.
- Shared local equality retains exact decimal JSON numeric values, not Python JSON binary64 equality; integer9007199254740992 versus filter9007199254740993.0 is explicitly different. Scalar dict/next/pagination truthiness does use Source binary64 underflow/overflow. Raw Wire/Envelope numbers remain exact. Shared Requests repr/Unicode numeric representation runtime boundaries stay documented.
- Bulk maps sort filter keys while individual options preserve insertion order. SDK constructor/route/session/cache/Computed-location controls, deprecated jmespath_filters and __conflicting_attrs overload are rejected instead of being silently mapped into Body/server query; mutable/restoration/JMESPath compatibility remains separately tracked.
- Owned standard links:{next:URL} compatibility extends pinned Source dict-loop behavior. Origin/path/filter/cycle changes are rejected; scalar markers must expand to one nonempty transport value. Limit/max_items are finite nonnegative Go integers and typed pagination is bool; raw paginated follows JSON truthiness, rather than arbitrary Python object controls.
- Go handles final service representations200..399 and retains other actual native status evidence; arbitrary Python Adapter returning final1xx is outside the engine policy. Explicit empty per-call version means absent version headers and immutable retry contracts are library-owned Go behavior.
- Stock Neutron27.0.0 get_address_group policy permits admin/owning-project reader or shared groups and has no service-only branch. Source unknown shared/tags/all_projects query does not grant roles. SDK fixtures do not establish live endpoint/backend/policy acceptance.
- Go fixes the SDK-owned location view to null while preserving actual values in Wire. Source Computed location may accept explicit row kwargs in LIST construction; mutable Connection-derived location remains unresolved under SDK-R1.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
rows = list(conn.network.address_groups(
    project_id="PROJECT_ID",
    addresses=["10.0.0.0/24"],
    max_items=20,
))
for row in rows:
    print(row.to_dict())
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    project := flag.String("project", "", "optional server project filter")
    address := flag.String("address", "", "optional exact one-address list filter")
    cap := flag.Int("max-items", 20, "raw row cap; zero is unbounded")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    options := []addressgroups.RecordListOption{
        addressgroups.WithRecordListMaxItems(*cap),
    }
    if *project != "" {
        options = append(options, addressgroups.WithRecordListQuery("project_id", *project))
    }
    if *address != "" {
        options = append(options, addressgroups.WithRecordListQuery("addresses", []string{*address}))
    }
    for row, err := range network.SecurityAddressGroups.ListRecords(ctx, options...) {
        if err != nil { log.Fatal(err) }
        raw, err := json.MarshalIndent(row.Resource.Body, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Printf("HTTP %d\n%s\n", row.StatusCode, raw)
    }
}
```

## References

- [Install](../../docs/install.md)
- [Address Group](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/address_group.py)
