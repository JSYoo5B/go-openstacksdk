# Trunk List Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/trunks`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilter`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilters`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithBodyFilter`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithBodyFilters`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithName`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithStatus`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPageSize`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPaginated`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk.Connection.Network`
- `github.com/JSYoo5B/go-openstacksdk.Connection.NetworkV2`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithRecordListPaginated`

- SDK-owned Resources.List/All classify all14 canonical query attributes and18 accepted spellings and exactly two non-query Body properties from pinned Trunk. Effective10 descriptors use Resource+TagMixin C3 MRO6, not NetworkResource; sourceSHA6/AST35 independently anchor defaults/accessors/types and query classification. Native created_at/updated_at/revision_number do not become semantic attributes.
  Tests: `api/trunk_list_filters_test.go:TestTrunkListFiltersEntireQueryAndAliasDescriptor`, `internal/cmd/sdkgen/trunk_filters_test.go:TestTrunkPythonManifestSeparatesQuerySubportsFromIndependentJSONIdentity`, `internal/cmd/sdkgen/trunk_filters_test.go:TestTrunkPythonPinnedSourceRequiresLiveSixSHAAndThirtyFiveASTProof`.
- Query is_admin_state_up->admin_state_up and three tag aliases are transposed. Bulk canonical keys win by presence over wire aliases including null/false/empty, while individual options targeting one wire key use last-wins. Scalars preserve exact numbers, lowercase bools and empty strings; scalar arrays repeat values with null entries omitted and nil/empty presence retained.
  Tests: `api/trunk_list_filters_test.go:TestTrunkListFiltersEntireQueryAndAliasDescriptor`, `connection_trunk_filters_test.go:TestConnectionTrunkFiltersSharedClientAndDistinctQueryBodyNamespaces`, `internal/cmd/sdkgen/trunk_filters_test.go:TestTrunkPythonManifestSeparatesQuerySubportsFromIndependentJSONIdentity`.
- Local id and tenant_id compare original lowercase JSON fields; missing/null match null, empty strings remain distinct and literal missing/null ID never falls back to Name. Native ID/TenantID still return zero strings and reject incompatible nonnull known types before matching. Project response alias does not infer local tenant from native ProjectID.
  Tests: `api/trunk_list_filters_test.go:TestTrunkListFiltersRawDescriptorsAndNativeProjection`, `connection_trunk_filters_test.go:TestConnectionTrunkFiltersSharedClientAndDistinctQueryBodyNamespaces`.
- Semantic project_id/name/status/port_id/sub_ports/description/is_admin_state_up/tags stay server-only despite disagreeing returned values; sub_ports list and administrative bool response descriptors do not add local normalization. WithName retains server hint+exact local name, WithStatus retains case-insensitive local target using final last-wins status query, raw status alone is wire-only.
  Tests: `api/trunk_list_filters_test.go:TestTrunkListFiltersEntireQueryAndAliasDescriptor`, `api/trunk_list_filters_test.go:TestTrunkListFiltersNativeAndIdentitySurfaceIsolation`, `connection_trunk_filters_test.go:TestConnectionTrunkFiltersPreflightFinalStatusAndNativeSurfaces`.
- Library-owned semantic options snapshot maps/raw JSON/nested scalar arrays and can be reused concurrently. Bulk options replace only their own namespace; nil/empty clears preserve explicit Body/raw query. Only final selected known values validate, allowing later overwrite/clear to remove earlier invalid values.
  Tests: `api/trunk_list_filters_test.go:TestTrunkListFiltersSnapshotsReplacementAndConcurrentReuse`, `resource/filters_test.go:TestSemanticFiltersValidateOnlyFinalSelectedValues`.
- Unknown semantic names are discarded even with unserializable values, including native-only timestamps/revision/sort. Nine source controls fail lazily before HTTP; query objects/nested arrays and unsupported explicit Body keys reject. Same-target semantic/raw query/page/name/status or explicit Body collisions reject, while raw id/tenant query and local Body namespaces are independent.
  Tests: `api/trunk_list_filters_test.go:TestTrunkListFiltersLazyPreflightAndNamespaceCollisions`, `connection_trunk_filters_test.go:TestConnectionTrunkFiltersPreflightFinalStatusAndNativeSurfaces`, `resource/filters_test.go:TestSemanticFiltersNamespaceCollisionsAndIndependentClear`.
- MaxItems counts raw rows before local Body/name/status filtering, and SinglePage or consumer break stops before continuation without synthetic wire limit hints. Caps suppress only unconsumed local selector work; native whole-page malformed strings/bools/ints/arrays/nested Subport/date fields remain errors even beyond a cap, with All returning nil rather than partial results.
  Tests: `api/trunk_list_filters_test.go:TestTrunkListFiltersControlsAndNativeWholePageFailures`, `connection_trunk_filters_test.go:TestConnectionTrunkFiltersPreflightFinalStatusAndNativeSurfaces`, `internal/cmd/sdkgen/trunk_filters_test.go:TestTrunkBodyNativeSchemaGuardsPlainModelsBuilderAndInheritedContinuation`.
- Native Trunk13/Subport3 fields and plain own-method graphs remain unchanged; native standard RFC3339 time.Time accepts Z/offsets and rejects no-zone strings. Missing/null known fields use native zeros, null nested Subport elements are zero structs; consumed null Body rows reject, query-only null rows return native zero models and empty object rows remain valid.
  Tests: `api/trunk_list_filters_test.go:TestTrunkListFiltersRawDescriptorsAndNativeProjection`, `api/trunk_list_filters_test.go:TestTrunkListFiltersControlsAndNativeWholePageFailures`, `internal/cmd/sdkgen/trunk_filters_test.go:TestTrunkBodyNativeSchemaGuardsPlainModelsBuilderAndInheritedContinuation`, `internal/cmd/sdkgen/trunk_filters_test.go:TestTrunkBodyFiltersAcceptActualCompiledPlainNestedAndRequestMethodGraph`.
- Native list HTTP200/204/300 and inherited top-level links.next are retained; plural trunks_links/top-level next do not gain support. Foreign continuations are followed by native pager; cycles/malformed links/late HTTP/decode/cancellation terminate. JSON-empty204 produces native EOF; non-JSON204 is empty. Matching duplicates stay distinct list rows.
  Tests: `api/trunk_list_filters_test.go:TestTrunkListFiltersPagingLiveSourceAndNativeStatus`, `internal/cmd/sdkgen/trunk_filters_test.go:TestTrunkDependenciesGuardInheritedLinksWholePageNumbersCodesAndRoute`.
- Leaf and cached Connection.Network/NetworkV2 Trunk facades retain the same original client/Provider, configured headers/microversion, ResourceBase prefix, middleware and live token changes across pages. No parent GET, automatic mutation, follow-up fetch or shared-client rewrite is introduced.
  Tests: `api/trunk_list_filters_test.go:TestTrunkListFiltersPagingLiveSourceAndNativeStatus`, `connection_trunk_filters_test.go:TestConnectionTrunkFiltersSharedClientAndDistinctQueryBodyNamespaces`, `connection_trunk_filters_test.go:TestConnectionTrunkFiltersPreflightFinalStatusAndNativeSurfaces`.
- Ordinary typed native List/Get/FindIdentity/explicit Ref and existing tags/subport actions remain independent from semantic List filters. Raw wire id/tenant/status/vendor/revision values remain intact in ordinary and Body lanes; native nested full models and action envelopes are returned without fabricating response metadata or a ResourceAdapter API.
  Tests: `api/trunk_list_filters_test.go:TestTrunkListFiltersNativeAndIdentitySurfaceIsolation`, `internal/cmd/sdkgen/trunk_filters_test.go:TestTrunkFilterEmissionOwnsBothQueryLanesAndPreservesPriorTenBindings`, `internal/cmd/sdkgen/trunk_filters_test.go:TestTrunkBodyFiltersAcceptActualCompiledPlainNestedAndRequestMethodGraph`.
- Generation fails closed without live sourceSHA6/fresh completeAST35 equality and schema/hash proof.20 reachable native declarations (leaf10/root2/pager8) plus exactly one resourcePath=trunks constant guard builder/result/plain nested model/inherited pager/UseNumber/codes/route drift. Both iterator query maps deep-copy repeated/nil values; prior ten manifest bytes and emitted function ASTs remain unchanged.
  Tests: `internal/cmd/sdkgen/trunk_filters_test.go:TestTrunkPythonManifestSeparatesQuerySubportsFromIndependentJSONIdentity`, `internal/cmd/sdkgen/trunk_filters_test.go:TestTrunkBodyNativeSchemaGuardsPlainModelsBuilderAndInheritedContinuation`, `internal/cmd/sdkgen/trunk_filters_test.go:TestTrunkFilterEmissionOwnsBothQueryLanesAndPreservesPriorTenBindings`, `internal/cmd/sdkgen/trunk_filters_test.go:TestTrunkPythonPinnedSourceRequiresLiveSixSHAAndThirtyFiveASTProof`, `internal/cmd/sdkgen/trunk_filters_test.go:TestTrunkDependenciesGuardInheritedLinksWholePageNumbersCodesAndRoute`, `internal/cmd/sdkgen/trunk_filters_test.go:TestTrunkBodyFiltersAcceptActualCompiledPlainNestedAndRequestMethodGraph`.
- Lazy generator and complete AllRecords use the original selected service without GET/identity enrichment; repeated ID/name rows remain separate. AllRecords discards partial rows on late page/HTTP/cancellation/source errors.
  Tests: `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordLateErrorsDiscardAllAndDuplicatesStayRows`, `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordOptionOwnershipLazyReuseAndPreflight`.
- Reviewed Trunk Resource+TagMixin effective10 Body descriptors reused directly: missingnull/tags[], explicitnull, project_id alias absence, ordered canonical/wire alias overwrite, CPython boolean truthiness, untyped nested subport list elements and unknown extension/timestamp/raw precision; no NetworkResource revision/Header descriptors.
  Tests: `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordDescriptorsSubportsTagsAndIndependentProof`.
- Exact14 server declarations and tag aliases use canonical query presence before wire aliases includingnull; Source LIST fields values remain original. Unknown kwargs discard before value serialization/validation. id/tenant_id alone are declared nonquery local Body filters.
  Tests: `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordDeclaredQueryAliasesAndUnknownDiscard`, `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordLocalFiltersAliasesAndOrderedFailure`.
- Ordered local recursive dict filtering keeps Source falsey/empty-dict/truthy-scalar/mapping.get error and bool-number equality; decimal truthiness is CPython binary64 while local comparison retains documented exact-decimal JSON profile.
  Tests: `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordLocalFiltersAliasesAndOrderedFailure`, `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordConcurrentSnapshotsAndPaginationTruthiness`.
- Raw-row cap precedes filters and avoids unconsumed bad rows/next; first-page hint, explicitzero-limit/cap, paginated truthiness, consumerbreak and short-page retained Wire marker are owned by existing REST paging. Mutating Resource IDs never changes cursor.
  Tests: `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordRawCapMarkerAndConsumerStop`, `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordConcurrentSnapshotsAndPaginationTruthiness`.
- Source plural/top-level/HTTP/relative-versioned continuations and dictionary compatibility use fixed origin/collection/filter/cursor guards. Top-level next binary64 underflow terminates with actual page preserved; truthy nonstrings and retargeting failures retain whole-page receipt.
  Tests: `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordPagingRepresentationsAndSourceNextTruthiness`, `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordFixedContinuationGuardsAndWholePageReceipt`.
- Go acceptedfinal200..399, representation-required204 and no ignored ordinary LIST400/403/404. Malformed/missing/null/scalar rows and cls/connection/microversion/_synchronized existing-constructor collisions fail with actual acceptedstatus/body/header proof. Source final<400 informational boundary is documented.
  Tests: `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordMalformedConstructorStatusAndPermissionProof`.
- Concrete map/raw/pointer/order/header snapshots and per-iteration capture beforecallbacks are reusable and concurrent. Retained callback state cannot mutate active request. Source/API replacement, nil/canceled context, negative controls, malformed known JSON, unsafe keys and unsupported route/session/JMESPath/unknown-parameter controls fail beforeHTTP. Known JSON builder failures add ErrInvalidOption while preserving caller marshal causes before physical HTTP.
  Tests: `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordOptionOwnershipLazyReuseAndPreflight`, `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordConcurrentSnapshotsAndPaginationTruthiness`, `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordKnownMarshalErrorsPreflightAndUnknownDiscard`.
- Standalone explicit microversion governs every page and empty removes versionheader; matching configured header is replaced only on ownedcopy. LIST Accept defaultsjson with explicit safe override. Retry route/body/header/absence/version guards preserve native rejection/custom cause; clean retry and reauthentication retain live token.
  Tests: `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordExplicitAndAbsentVersionsAcrossPages`, `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordNativeRetryOwnershipAndCleanReauthentication`.
- Physical accepted200 and rejected400/401/403/404/409/429/500/503 mixed Read/Close/customCancel/stickySource errors preserve all causes, actualreceipt, exactlyoneClose/HTTP and no retry/reauth replay under willing callbacks. Unauthorized response follows the actual native ErrUnableToReauthenticate.ErrOriginal receipt and ErrReauth physical/source/cancel causes; the native error intentionally has no Unwrap.
  Tests: `network/v2/extensions/trunks/python_list_contracts_test.go:TestPythonTrunkListRecordPhysicalMixedFaultsNoReplayAndActualReceipt`.

Python/Go differences:

- Go uses frozen concrete WithFilter/WithFilters and explicit Body options instead of Python kwargs and mutable descriptors. Bulk canonical keys win by presence and individual options are last-wins. Raw WithQuery remains an independent explicit wire extension; unknown semantic names discard. Trunk project query and deprecated tenant local Body stay separate.
- Original JSON comparison occurs only after native whole-page extraction. Known string/bool/int/nested Subport/time fields can reject before matching, including later rows past a cap. Returned models remain native projections; original local id/tenant spelling/presence does not expand native model permissiveness.
- All two local descriptors are untyped JSON: no bool/int/list formatter is added. Go does not coerce caller JSON values and distinguishes bool from numbers; Python equality differs. Dictionary subset and ordered-array machinery is shared but native known local strings limit which returned domains reach matching. Queried sub_ports/is_admin_state_up response coercion does not become local filtering.
- Semantic query arrays become repeated scalar URL values rather than arbitrary Python container formatting; nil/empty arrays retain key presence while omitting values. Reserved source controls do not expose caller session/headers/microversion/base_path or JMESPath overrides.
- Go raw cap counts before filtering like Python but omits its limit hint and stops continuation immediately. Native whole-page decoding/HTTP200204300/foreign following/inherited links.next and JSON-empty204 EOF remain. Body-consumed null rows reject; query-only native zero rows remain.
- Concrete immutable TrunkListRecord is an alias of existing TrunkFindRecord and reuses its reviewed Body/query inventory through shared Source opt-in Engine.List. It does not complete mutable Python Resource/cache/dirty/session/subclass/Connection lifecycles or independently change earlier Get/Find/CRUD/Subport mappings.
- Each list iteration physically observes pages, retaining live selected provider auth. Pinned optional Proxy cache_enabled GET caching/invalidation remains the separate shared session/cache foundation; no response cache or cache parity is claimed for this finite named record API.
- Python kwargs become concrete functional options with snapshotted JSON and sorted bulk Go map keys. Sequential query/local filter order is retained; unknown kwargs discard before serialization. allow_unknown_params/JMESPath/conflicting-attrs/route/session/runtime constructor controls are explicit preflighterrors, not an unrestricted Python adapter surface.
- Public Trunk sub_ports/tag list descriptors preserve scalar wrapping/null and untyped nested JSON. Gophercloud native Subport typed extraction, timestamps and revision remain independently callable native surfaces. Raw extension unknown values remain separate actual Wire and Resource JSON; plain/computed/control shadowing and automatic Connection current_location are not full Python equivalence.
- Source integer versus bool interoperability is retained by shared exact-decimal equality plus bool-number matching; scalar fraction/exponent truthiness uses binary64. Full CPython float rounding/nonfinite/custom-object comparison is not claimed.
- Common Engine completion range is final200..399. Source exceptions.raise_from_response accepts final<400, so exposed informational100..199 representations are outside this Go profile; ordinary clients consume interim1xx. Every rejected400..599 actualphysical cause remains observable.
- Same-origin fixed-collection/filter continuation and one nonempty Wire marker query value intentionally restrict Python continuation retargeting. Dictionary links are explicit compatibility. Caps/single-page observe only their declared scope; AllRecords discards partials on late failure rather than claiming unseen-page success.
- Trunk has no class maximum microversion: no discovery is introduced and captured selected/absent version remains. Safe explicit Headers/microversion options own LIST policy and do not grant project/admin authority. Stock Neutron27 project ADMIN_OR_PROJECT_READER classifies the list as core user; live deployment policy remains actual server enforcement.
- Trunk List opts StopFaultRetryCallbacks explicitly and independently of shared Source projection policy. Gophercloud unauthorized reauthentication guard errors retain receipt/cause in ErrOriginal/ErrReauth public fields, as v2.15.0 intentionally does not Unwrap those two independent errors.
- Go fixes the SDK-owned location view to null while preserving actual values in Wire. Source Computed location may accept explicit row kwargs in LIST construction; mutable Connection-derived location remains unresolved under SDK-R1.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--name", default="")
parser.add_argument("--port", default="")
parser.add_argument("--max-items", type=int, default=0)
parser.add_argument("--single-page", action="store_true")
args = parser.parse_args()

conn = openstack.connect(cloud=args.cloud)
query = {"paginated": not args.single_page, "max_items": args.max_items}
if args.name:
    query["name"] = args.name
if args.port:
    query["port_id"] = args.port
for trunk in conn.network.trunks(**query):
    print(json.dumps(trunk.to_dict(), indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    name := flag.String("name", "", "server-side trunk name query")
    port := flag.String("port", "", "parent port ID query")
    cap := flag.Int("max-items", 0, "raw row cap; 0 means unlimited")
    single := flag.Bool("single-page", false, "observe first page only")
    version := flag.String("microversion", "", "optional selected service version")
    flag.Parse()

    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    connOptions := []sdk.ConnectionOption{sdk.WithCloud(*cloud)}
    if *version != "" {
        connOptions = append(connOptions, sdk.WithMicroversion(sdk.Network, *version))
    }
    conn, err := sdk.Connect(ctx, connOptions...)
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    options := []trunks.RecordListOption{
        trunks.WithRecordListMaxItems(*cap),
        trunks.WithRecordListPaginated(!*single),
    }
    if *name != "" { options = append(options, trunks.WithRecordListQuery("name", *name)) }
    if *port != "" { options = append(options, trunks.WithRecordListQuery("port_id", *port)) }
    rows, err := service.Trunks.AllRecords(ctx, options...)
    if err != nil { log.Fatal(err) }
    for _, row := range rows {
        body, err := json.MarshalIndent(row.Resource.Body, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(body))
        fmt.Printf("HTTP %d request-id=%s\n", row.StatusCode,
            row.Header.Get("X-Openstack-Request-Id"))
    }
}
```

## References

- [Install](../../docs/install.md)
- [Trunk](https://opendev.org/openstack/neutron/src/tag/27.0.0/neutron/conf/policies/trunk.py)
