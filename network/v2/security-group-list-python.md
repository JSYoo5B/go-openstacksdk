# Security Group List Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/security_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilter`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilters`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithBodyFilter`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithBodyFilters`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithName`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPageSize`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPaginated`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk.Connection.Network`
- `github.com/JSYoo5B/go-openstacksdk.Connection.NetworkV2`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordListPaginated`

- Pinned direct Proxy.security_groups delegates query to SecurityGroup Resource.list. Ordinary SecurityGroups.Resources.List/All and API.All provide SDK-owned WithFilter/WithFilters classification and explicit WithBodyFilter predicates without caller builders or callbacks. Leaf and cached Connection.Network().API/NetworkV2 facades share the existing native service/provider client; no handwritten top-level SecurityGroup collection or public ResourceAdapter is added.
  Tests: `api/security_group_list_filters_test.go:TestSecurityGroupListFiltersEntireQueryAndAliasDescriptor`, `connection_security_group_filters_test.go:TestConnectionSecurityGroupFiltersSharedClientAndOriginalRules`, `internal/cmd/sdkgen/security_group_filters_test.go:TestSecurityGroupFilterEmissionPreservesOwnedFullQueryPagersAndPriorNineBindings`.
- Seventeen canonical query attributes and twenty-one accepted canonical/wire spellings include inherited limit/marker/tags and shared/tag aliases. Both project_id and tenant_id are explicit separate server query keys; the response project alias does not collapse them. Inherited revision_number, stateful, is_shared, id and name are also query-only. Only created_at, updated_at and security_group_rules are local Body properties.
  Tests: `api/security_group_list_filters_test.go:TestSecurityGroupListFiltersEntireQueryAndAliasDescriptor`, `api/security_group_list_filters_test.go:TestSecurityGroupListFiltersLazyPreflightAndNamespaceCollisions`, `connection_security_group_filters_test.go:TestConnectionSecurityGroupFiltersPreflightAndNativeSurfaces`, `internal/cmd/sdkgen/security_group_filters_test.go:TestSecurityGroupPythonManifestKeepsQueryAliasesAndOnlyThreeJSONBodyProperties`.
- Semantic bulk canonical query names take precedence by presence over wire aliases even for null/false/empty values. Individual same-wire options are last-wins. Scalar arrays retain repeated URL values and exact number spelling while omitting null members; nil/empty arrays omit URL values without losing option-key presence. Query objects/nested arrays and nine source controls reject lazily. Unknown semantic names are discarded before value validation.
  Tests: `api/security_group_list_filters_test.go:TestSecurityGroupListFiltersEntireQueryAndAliasDescriptor`, `api/security_group_list_filters_test.go:TestSecurityGroupListFiltersLazyPreflightAndNamespaceCollisions`, `resource/filters_test.go:TestSemanticFiltersValidateOnlyFinalSelectedValues`.
- Semantic query conditions conflict before HTTP with raw query/page-size/name hints for the same wire key; semantic and explicit Body conditions for the same raw field also conflict even when values agree. Explicit Body filters support exactly the three original response keys, with no queried name/id/tenant/revision/stateful/shared selectors. Raw WithQuery remains an independent literal wire extension. WithStatus remains unsupported because the native model has no Status field.
  Tests: `api/security_group_list_filters_test.go:TestSecurityGroupListFiltersLazyPreflightAndNamespaceCollisions`, `connection_security_group_filters_test.go:TestConnectionSecurityGroupFiltersPreflightAndNativeSurfaces`, `resource/filters_test.go:TestSemanticFiltersNamespaceCollisionsAndIndependentClear`.
- All three local fields use original JSON without response-only boolean or integer normalization. Missing/null rules match null, empty arrays remain distinct, and null rule members stay null for comparison while the returned native Rules contains zero structs. Original date strings are compared literally after native time decoding; equal instants with different spellings need not match.
  Tests: `api/security_group_list_filters_test.go:TestSecurityGroupListFiltersRawDescriptorsAndNativeProjection`, `connection_security_group_filters_test.go:TestConnectionSecurityGroupFiltersSharedClientAndOriginalRules`, `internal/nativefind/security_group_bodies_test.go:TestNativeSecurityGroupBodiesOwnQueryRowsAndLiveSource`.
- Original rule arrays retain unknown nested keys, precise large JSON numbers and complete ordered elements after successful native decoding. Whole-array objects compare fully rather than as recursive subsets; standalone object filters use the existing subset policy on nonempty actual objects. Native Rules, Stateful, ProjectID/TenantID, revision integers and group/rule timestamp projections are returned unchanged. Known native fields may reject before any raw comparison.
  Tests: `api/security_group_list_filters_test.go:TestSecurityGroupListFiltersRawDescriptorsAndNativeProjection`, `connection_security_group_filters_test.go:TestConnectionSecurityGroupFiltersSharedClientAndOriginalRules`, `internal/nativefind/security_group_bodies_test.go:TestNativeSecurityGroupBodiesControlsAndWholePageDecoding`.
- Options deeply snapshot maps, lists and raw JSON, permit independent concurrent reuse, and validate only final selected values. Semantic bulk replacement/clear changes only its namespace; raw query and explicit Body conditions remain independent. Mutating returned rules cannot mutate saved options or later responses. Invalid options and canceled contexts produce an iterator error before HTTP rather than silent truncation.
  Tests: `api/security_group_list_filters_test.go:TestSecurityGroupListFiltersSnapshotsReplacementAndConcurrentReuse`, `api/security_group_list_filters_test.go:TestSecurityGroupListFiltersLazyPreflightAndNamespaceCollisions`, `internal/nativefind/security_group_bodies_test.go:TestNativeSecurityGroupBodiesLazyPreflightAndConcurrentReuse`.
- Local max_items counts raw rows before Body/name predicates without match refill or wire-limit inference. First-page mode and consumer break stop before continuation. Native extraction validates every row of the current page, including malformed known group/rule fields after a raw cap. Consumed null rows with Body filters reject, query-only null rows keep native zero models, and empty object rows remain valid.
  Tests: `api/security_group_list_filters_test.go:TestSecurityGroupListFiltersControlsAndNativeWholePageFailures`, `internal/nativefind/security_group_bodies_test.go:TestNativeSecurityGroupBodiesControlsAndWholePageDecoding`, `connection_security_group_filters_test.go:TestConnectionSecurityGroupFiltersPreflightAndNativeSurfaces`.
- The SDK-owned raw pager preserves the hyphenated security-groups route and underscore security_groups envelope, native 200/204/300 and security_groups_links last-rel-next behavior. Non-JSON204 completes empty; JSON-empty204 produces native EOF. Top-level next does not acquire support. Later HTTP/native decode failures, cycles, foreign continuation following and cancellation remain terminal; All does not return a partial slice on error.
  Tests: `api/security_group_list_filters_test.go:TestSecurityGroupListFiltersPagingLiveSourceAndNativeStatus`, `internal/nativefind/security_group_bodies_test.go:TestNativeSecurityGroupBodiesNativeContinuationCodesAndErrors`, `connection_security_group_filters_test.go:TestConnectionSecurityGroupFiltersSharedClientAndOriginalRules`.
- The new Body iterator uses a frozen complete url.Values map and the original live service/provider clients, configured headers, microversion, middleware and ResourceBase prefix. Repeated, nil/empty and unknown wire values are preserved by the existing raw-query path rather than concrete native ListOpts conversion. The unchanged ordinary iterator remains the no-Body path; list conditions do not manufacture item GETs or rule requests.
  Tests: `internal/nativefind/security_group_bodies_test.go:TestNativeSecurityGroupBodiesOwnQueryRowsAndLiveSource`, `internal/nativefind/security_group_bodies_test.go:TestNativeSecurityGroupBodiesLazyPreflightAndConcurrentReuse`, `connection_security_group_filters_test.go:TestConnectionSecurityGroupFiltersSharedClientAndOriginalRules`.
- Semantic unknown status is discarded, while raw status/shared/fields/vendor queries remain literal wire extensions. Typed native List/Get, existing FindIdentity, group CRUD and rule APIs retain their separate request/model contracts; this list unit does not add lookup, mutation, status-model or rule-fetch capabilities.
  Tests: `api/security_group_list_filters_test.go:TestSecurityGroupListFiltersNativeAndIdentitySurfaceIsolation`, `api/security_group_list_filters_test.go:TestSecurityGroupListFiltersLazyPreflightAndNamespaceCollisions`, `connection_security_group_filters_test.go:TestConnectionSecurityGroupFiltersPreflightAndNativeSurfaces`.
- The complete classifier requires seven live pinned Python file SHAs and thirty-nine independent AST anchors proving exact C3 inheritance, all twelve effective descriptors/defaults/types/aliases, inherited queried revision and tag mapping, None access, conversion and direct Proxy controls. Fresh isolated AST extraction must match the checked-in manifest. Native gates cover twenty-one declaration hashes, rootPath and NoZ constants, full model/ListOpts/nested Rule/result/page schemas and receiver signatures, including the unchanged typed List BuildQueryString policy. Direct Result.ExtractInto is reachable; struct/slice pointer extractors are not. Source hashes do not themselves implement inherited cache/session/continuation behavior.
  Tests: `internal/cmd/sdkgen/security_group_filters_test.go:TestSecurityGroupPythonManifestKeepsQueryAliasesAndOnlyThreeJSONBodyProperties`, `internal/cmd/sdkgen/security_group_filters_test.go:TestSecurityGroupBodyNativeSchemaGuardsConcreteListAndBothTimestampDecoders`, `internal/cmd/sdkgen/security_group_filters_test.go:TestSecurityGroupFilterEmissionPreservesOwnedFullQueryPagersAndPriorNineBindings`, `internal/cmd/sdkgen/security_group_filters_test.go:TestSecurityGroupPythonPinnedSourceRequiresCompleteLiveClassificationProof`, `internal/cmd/sdkgen/security_group_filters_test.go:TestSecurityGroupDependenciesGuardNestedRuleWholePageNumbersCodesAndConstants`, `internal/cmd/sdkgen/security_group_filters_test.go:TestSecurityGroupBodyFiltersAcceptActualCompiledGroupRuleAndConcreteRequestGraph`.
- Pinned proxy security_groups delegates to exact SecurityGroup list, with lazy SDK-owned ListRecords/AllRecords and no member GET/name hint/discovery; reused existing MRO12 Body fields plus Header/computed view, common Engine.List SourceFindPolicy/StopFaultRetryCallbacks.
  Tests: `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsRawDescriptorsDefaultsAndOwnedEvidence`, `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsSelectedVersionAndListHeaders`, `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsPublicFixtureAndCancellation`.
- Owned Source null/tag[] defaults, untyped stateful/date/name/raw shapes, project_id fallback only absent, raw shared bool/binary64 truthiness and Source arbitrary precision/Unicode/bool int descriptor; unknown extensions/self/Wire/Envelope/Header/page-status independent, computed rawlocation cannot forge location.
  Tests: `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsRawDescriptorsDefaultsAndOwnedEvidence`, `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsAliasAndIntegerPresence`.
- Seventeen canonical server query keys and21 canonical/wire spellings: inherited limit/marker, tags/shared aliases canonical wins includingnull; project_id andtenant_id remain distinct, original fields/repeated ID values/false/zero preserved, only created_at/updated_at/security_group_rules local and unknown/Header kwargs discarded beforeencoding.
  Tests: `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsQueryAliasesAndNullCanonicalPriority`, `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsLocalFiltersAndOrderedShortCircuit`.
- Source local dict recursive/truthiness/mapping.get error and sequential predicate shortcircuit; descriptor list values use exact JSON plus bool-number equality, binary64 underflow truthiness, with documented exact decimal equality profile.
  Tests: `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsLocalFiltersAndOrderedShortCircuit`.
- Full retained pager plural/next/HTTP/dictionary/exactv2alias/falseynext/stopempty/shortpage marker, rawcap beforelocalfilter and unconsumedbadrow/next; foreignorigin/collection/filter/cycle and latepermission actualproof, AllRecords discards partial and successfulempty nonnil.
  Tests: `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsPagingRepresentationsAndFixedRoute`, `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsRawCapsMarkerAndUnconsumedRows`.
- Plural envelope list/singleton with consumed wrongshape/null/malformed/empty/UTF8 and Source cls/connection/microversion/_synchronized row constructor errors preserving entire actual acceptedpage receipt; unknown uppercaseunderscore Header key remains unknownBody and pageHTTPHeader does not seed row descriptor.
  Tests: `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsEnvelopeAndConstructorErrorsKeepPage`, `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsRawDescriptorsDefaultsAndOwnedEvidence`.
- Concrete options/maps/rawbytes/pointers/slice and retainedcallback isolation per lazyexecution, reusable concurrent ownedrecords; invalidknownJSON/control/cap/header beforeHTTP, explicit deprecatedjmespath/conflictingconstructor boundary, cancellation/type/Source errors retain all causes.
  Tests: `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsPreflightAndIgnoredValues`, `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsLazySnapshotsAndConcurrentReuse`, `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsPublicFixtureAndCancellation`.
- Selected/default explicitempty/latest network microversion preserved on allpages without newdiscovery, Source listAcceptJSON default and explicitsafeHeaderoverride, actual conditionalHeader remains requesttransport instead of invented rowdescriptor.
  Tests: `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsSelectedVersionAndListHeaders`.
- Bounded final200..399 Go policy versus pinnedSource<400, actual informational199 nativeproof; clean native retry/reauth preserves originalrequest, invalidbody/version/auth/Omitheader staysoriginalrejection, OkCodes retry2 stillenforces originalacceptance with actual400 receipt.
  Tests: `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsNativeStatusPolicy`, `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsNativeRetryAndOriginalAcceptance`.
- Physical accepted/rejected Read/Close/both/cancel/stickySource restored errors preserve actualpage/nativebody/header/status, allcauses and exactlyoneClose; StopFaultRetryCallbacks keeps callback countzero and noresend.
  Tests: `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsPhysicalErrorsDoNotInvokeRetry`.
- Shared SourceFindPolicy conditional-absence guard: unselected4 conditional headers stay absent; captured/per-call conditions survive cleanretry without row Header descriptor seed. Retry injection/valuechange/OmitHeaders fail before secondHTTP with actualoriginal503/body/headers and customcallback cause.
  Tests: `network/v2/extensions/security/groups/python_list_contracts_test.go:TestPythonSecurityGroupListRecordsConditionalRequestOwnership`.

Python/Go differences:

- Go uses concrete frozen WithFilter/WithFilters and explicit Body options instead of Python keyword arguments and mutable Resource descriptors. Canonical query keys win by presence in bulk maps; individual options are last-wins. Both tenant_id/project_id remain query attributes despite the response alias. Unknown semantic names are discarded, while raw WithQuery remains an explicit wire extension.
- Original JSON is available only after successful native whole-page decoding. Known group/rule strings, bools, ints, nested arrays and timestamps can reject before filtering, including rows past a cap. Unknown rule members retain exact numbers and null array members for comparison without changing the returned native zero/projection policy.
- Python list descriptors wrap scalar values, while the Go raw lane does not. Arrays compare full ordered elements; standalone dictionaries retain subset matching on nonempty actual objects. Go bool/number equality is distinct from Python, and original date spellings are compared literally. All three local descriptors are raw JSON; queried int/bool descriptors introduce no new local coercion.
- Known semantic query arrays become repeated URL values rather than arbitrary Python container formatting. Nil/empty arrays preserve option-key presence while omitting values. Reserved source controls are not interpreted as per-call session/base-path/header/microversion or JMESPath options.
- Raw-row caps omit Python Resource.list wire-limit hints and stop before continuation. Native whole-page extraction, native linked pagination including foreign following, and JSON-empty204 EOF remain unchanged. Body-filter null rows reject; query-only native zero rows remain accepted.
- New Source-shaped record list is sibling to existing native typed List/All, generated Resources classifier and owned FindRecord; no existing API is recounted or changed. Private profile reuses existing verified12-field canonical/wire inventory and shared neutronfind Engine rather than adding a decoder/transport framework.
- Go concrete JSON options/owned Resource/Wire/Envelope snapshots provide the reviewed named list boundary; Python mutable Resource/class/subclass/cache/dirty/same-object/session and Connection current-location synthesis remain independent SDK-R1/C1/S1 foundations. Location is the explicit owned null view, not inferred cloud/auth-config facts.
- Unknown Source list kwargs including Header-only values are discarded before JSON transport encoding. Raw Source session/base_path/resource_type/constructor/control replacement, __conflicting_attrs restoration and deprecated JMESPath expression runtime are explicitly rejected and documented shared framework differences; no silent expression success or complete Python Proxy parity claim.
- Sequential predicates and raw response constructor aliases preserve JSON dictionary order; bulk maps/new callback keys are sorted. Numeric equality uses common exact decimal JSON plus bool-number behavior; decimal truthiness uses CPython binary64 underflow and shared integer descriptor conversion, not arbitrary Python runtime rounding/nonfinite/custom objects.
- Go limit/cap require nonnegative machine integers and marker one nonblank UTF8 value. Safe fixed origin/collection/filter/protocol route and exact /v2.0/security-groups alias restrict Source broad continuation retargeting; dictionary links are Gocompatibility. Raw cap counts excluded rows and explicit cap/singlepage observes only that range.
- Source LIST Header descriptor comes from row canonical exact if_match/case-insensitive wire if-match, not actual wholepage HTTP headers. Actual Header and Wire remain separate; uppercase IF_MATCH remains unknown Body extension. Computed location ignores raw wire spoofing under SourceFindPolicy.
- Pinned Source accepts status<400; Go deliberately handles final200..399 and rejects informational100..199 with actual native proof. UTF8 is strict versus Requests replacement decoding. Native OkCodes hooks can reach retry2 but final original acceptance remains enforced; no blanket preflight rejection claim.
- Fixed Neutron27 policy permits project-reader or shared-security-group visibility paths and stock collection controller filters authorization per row. SDK adds no client authorization gate or all-project permission inference. Fixture/race/main build evidence is distinct from authenticated deployment policy/endpoints, and write/admin API parity is not part of this list row.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument('--cloud', required=True)
parser.add_argument('--name')
parser.add_argument('--limit', type=int)
parser.add_argument('--max-items', type=int, default=0)
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
query = {'max_items': args.max_items}
if args.name is not None:
    query['name'] = args.name
if args.limit is not None:
    query['limit'] = args.limit
for group in conn.network.security_groups(**query):
    print(json.dumps(group.to_dict(), ensure_ascii=False))
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

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups"
)

func main() {
    cloud := flag.String("cloud", os.Getenv("OS_CLOUD"), "clouds.yaml cloud")
    name := flag.String("name", "", "optional server name filter")
    limit := flag.Int("limit", -1, "page limit; -1 omits it")
    cap := flag.Int("max-items", 0, "raw row cap; zero is unlimited")
    paginated := flag.Bool("paginated", true, "follow all retained pages")
    flag.Parse()
    if err := run(*cloud, *name, *limit, *cap, *paginated); err != nil {
        log.Fatal(err)
    }
}

func run(cloud, name string, limit, maxItems int, paginated bool) error {
    if limit < -1 { return fmt.Errorf("-limit must be -1 or nonnegative") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    network, err := conn.NetworkV2(ctx)
    if err != nil { return err }
    options := []groups.RecordListOption{
        groups.WithRecordListMaxItems(maxItems),
        groups.WithRecordListPaginated(paginated),
        groups.WithRecordListHeaders(map[string]string{"X-Request-Source": "security-group-list-example"}),
    }
    if name != "" { options = append(options, groups.WithRecordListQuery("name", name)) }
    if limit >= 0 { options = append(options, groups.WithRecordListLimit(limit)) }
    out := json.NewEncoder(os.Stdout)
    for record, readErr := range network.SecurityGroups.ListRecords(ctx, options...) {
        if readErr != nil { return readErr }
        if err := out.Encode(map[string]any{
            "resource": record.Resource.Body,
            "wire": record.Wire.Body,
            "envelope": record.Envelope,
            "headers": record.Header,
            "status_code": record.StatusCode,
        }); err != nil { return err }
    }
    return nil
}
```

## References

- [`ef55d7d`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6647-L6665)
- [`SecurityGroup`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/security_group.py)
- [Security Group](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/security_group.py#L125-L153)
- [extension](https://github.com/openstack/neutron/blob/27.0.0/neutron/extensions/securitygroup.py)
- [stock Controller._items](https://github.com/openstack/neutron/blob/27.0.0/neutron/api/v2/base.py#L292-L378)
- [Python List Contracts Test](extensions/security/groups/python_list_contracts_test.go)
