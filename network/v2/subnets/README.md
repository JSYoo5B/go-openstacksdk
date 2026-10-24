# Subnets

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/subnets`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.New`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilter`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilters`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithBodyFilter`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithBodyFilters`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPaginated`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.RecordListGenerator.JMESPathIdentity`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.RecordListGenerator.MarshalJSON`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListConflictingAttrs`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListJMESPath`

- All nine declared non-query Body fields are explicitly filterable in ordinary Resources.List/All through leaf and cached Connection.Network.API paths. Native extraction completes first, then original rows are paired by count and index without extra HTTP or ID lookup. The public native model alias is retained.
  Tests: `api/network_subnet_raw_body_filters_test.go:TestNetworkSubnetRawBodyFiltersNineFieldsAndOriginalValues`, `resource/body_record_test.go:TestBodyRecordPairingPreservesOriginalRowsAndOwnsFields`, `resource/body_record_test.go:TestBodyRecordPairingRejectsUnprovenShapeAndCount`.
- Raw JSON preserves precision beyond float64, unknown pool/route keys, null array elements, missing/null versus empty values and original accepted timestamp text. Untyped prefixlen values keep number/string/bool/fraction/object/array types; no response string-to-number coercion is applied. Generic object filters use recursive subset and arrays require full ordered values.
  Tests: `api/network_subnet_raw_body_filters_test.go:TestNetworkSubnetRawBodyFiltersExactNumbersAndUntypedPrefix`, `api/network_subnet_raw_body_filters_test.go:TestNetworkSubnetRawBodyFiltersNineFieldsAndOriginalValues`, `internal/jsonfilter/integer_test.go:TestIntegerJSONPreservesExactValueAndBoundedExponentMemory`.
- Explicit WithBodyFilter/WithBodyFilters own construction snapshots, per-iteration values, independent concurrent reuse, canonical last-wins, bulk replacement/clear and prefix_length alias conflicts. Clearing explicit Body conditions selects the original typed iterator. Unknown/unsupported/non-JSON explicit Body options fail lazily before HTTP; semantic options have their separate unknown-discard policy.
  Tests: `api/network_subnet_raw_body_filters_test.go:TestNetworkSubnetRawBodyFiltersSnapshotsAliasesAndConcurrentReuse`, `api/network_subnet_raw_body_filters_test.go:TestNetworkSubnetRawBodyFiltersLazyPreflightAndCapabilities`, `resource/body_record_test.go:TestCollectionBodyRecordLaneIsSelectedOnlyForPreparedFilters`.
- Raw controls apply before local Body/name filters without refilling matches. First-page, early break, native plural links, later HTTP failures, cycle/foreign continuation and cancellation remain terminal. Valid untyped prefix JSON beyond the cap is not rejected or normalized.
  Tests: `api/network_subnet_raw_body_filters_test.go:TestNetworkSubnetRawBodyFiltersControlsAndContinuation`.
- Native modeled array/pool/route/timestamp/revision errors are observable from the whole current page even beyond the cap or a name mismatch, retaining native causes. Raw query stays wire-only, typed status remains unsupported, and native typed List/FindIdentity plus the existing QoS float64 projection are unchanged.
  Tests: `api/network_subnet_raw_body_filters_test.go:TestNetworkSubnetRawBodyFiltersNativeWholePageFailures`, `api/network_subnet_raw_body_filters_test.go:TestNetworkSubnetRawBodyFiltersWireAndNativeIsolation`, `resource/body_record_test.go:TestBodyRecordFieldNormalizationPresenceAndCauses`.
- Generation fails closed on exact native Subnet/nested-field schema, custom decoder, own linked-page methods, extractor and receiver-qualified native/pagination declaration hashes including the UseNumber decoding proof. Only nine canonical Subnet fields are inventoried; existing four typed Body bindings and twenty identity/query capabilities remain intact.
  Tests: `internal/cmd/sdkgen/body_filter_collections_test.go:TestBodyFilterCollectionGateRejectsNativeShapeAndUnauditedFields`, `internal/cmd/sdkgen/body_filter_collections_test.go:TestBodyFilterCollectionSelectorsMarshalOnlyTheSelectedNativeField`, `internal/cmd/sdkgen/body_filter_collections_test.go:TestBodyFilterCollectionStillRequiresPinnedNativeContracts`, `internal/cmd/sdkgen/body_filter_collections_test.go:TestBodyFilterCollectionInventoryHasOnlyOwnedCanonicalFields`.
- Ordinary Resources.List/All classify the complete pinned Subnet semantic attribute set through SDK-owned WithFilter/WithFilters: 24 canonical query attributes and 30 accepted canonical/wire spellings, plus all nine declared non-query Body attributes. Query name/id/tags remain server-only. prefix_length selects original prefixlen JSON, tenant_id and revision_number stay local despite native query tags, and raw prefixlen is not a semantic attribute. Both generated leaf and cached Connection.Network.API paths use the same descriptor.
  Tests: `api/network_subnet_semantic_filters_test.go:TestNetworkSubnetSemanticFiltersEntireQueryDescriptor`, `api/network_subnet_semantic_filters_test.go:TestNetworkSubnetSemanticFiltersNineLocalBodyClassifications`, `resource/filters_test.go:TestSemanticFiltersDescriptorGateAndWireAliasBodyOverlap`.
- Individual semantic options are last-wins by destination; bulk WithFilters replaces only the semantic namespace and canonical Python query names beat their wire aliases even for nil, false and empty values. Nil/empty bulk maps clear semantic filters. Input snapshots, copied iteration state and concurrent reuse are owned by the SDK, and only final selected values are validated: overwritten or discarded encoding errors do not become terminal failures.
  Tests: `api/network_subnet_semantic_filters_test.go:TestNetworkSubnetSemanticFiltersEncodingAndCanonicalPrecedence`, `api/network_subnet_semantic_filters_test.go:TestNetworkSubnetSemanticFiltersSnapshotsReplacementAndConcurrentReuse`, `resource/filters_test.go:TestSemanticFiltersSnapshotsConcurrentReuseAndServerOnlyName`, `resource/filters_test.go:TestSemanticFiltersValidateOnlyFinalSelectedValues`.
- Semantic query scalars encode as strings, lowercase booleans and exact JSON number text; flat scalar arrays preserve repeated order while null elements produce no URL value. A selected null or empty array still retains presence for precedence/collision checks. Unknown semantic names are discarded without validating their captured value. Selected query objects/nested arrays/invalid JSON fail lazily before HTTP, and all nine source list-control names are reserved for dedicated options or return unsupported.
  Tests: `api/network_subnet_semantic_filters_test.go:TestNetworkSubnetSemanticFiltersEncodingAndCanonicalPrecedence`, `api/network_subnet_semantic_filters_test.go:TestNetworkSubnetSemanticFiltersUnknownReservedAndNamespaceCollisions`, `resource/filters_test.go:TestSemanticFiltersCanonicalPrecedencePresenceAndExactQueryEncoding`, `resource/filters_test.go:TestSemanticFiltersRejectSelectedErrorsAndReservedControlsBeforeIteration`.
- Raw WithQuery and explicit WithBodyFilter namespaces remain independent. Same-wire semantic query/raw query/page-size or WithName-hint collisions, and same-canonical semantic/explicit Body collisions, fail before HTTP in either option order even for identical or null values. Clearing semantic options leaves raw/explicit Body options intact. Native typed List and FindIdentity do not acquire semantic options, and unaudited bindings reject even a semantic clear.
  Tests: `api/network_subnet_semantic_filters_test.go:TestNetworkSubnetSemanticFiltersUnknownReservedAndNamespaceCollisions`, `api/network_subnet_semantic_filters_test.go:TestNetworkSubnetSemanticFiltersRawAndNativeSurfaceIsolation`, `resource/filters_test.go:TestSemanticFiltersNamespaceCollisionsAndIndependentClear`, `resource/filters_test.go:TestSemanticFiltersDescriptorGateAndWireAliasBodyOverlap`.
- Combined semantic query and raw local Body conditions retain native paging and whole-page extraction. Raw max-items caps apply before local matches without refill or a wire limit hint; single-page/early break stop before continuation. Query fields on advertised continuation links remain intact and later HTTP/cycle/cancellation or native typed-row errors remain terminal, including a malformed typed row beyond the cap.
  Tests: `api/network_subnet_semantic_filters_test.go:TestNetworkSubnetSemanticFiltersControlsAndTerminalPages`.
- Semantic generation requires -openstacksdk-source and the independent checked-in api/openstacksdk/resources/network/v2/subnet.json manifest. Seven reviewed live-source SHA-256 hashes are checked before isolated AST-only extraction; C3 MRO, inherited declarations, pagination defaults, tag aliases, unknown-discard/canonical-precedence policy, all descriptors and nine controls are re-extracted and compared with the complete manifest and fifteen AST anchors. Parser minor-version or source/manifest drift fails closed before generation. The existing full native raw-Body gate remains required; semantic query/body/reserved inventory is distinct from explicit canonical Body metadata and emitted only for Subnet.
  Tests: `internal/cmd/sdkgen/python_filters_test.go:TestPythonFilterExtractionIndependentMROQueryAndBody`, `internal/cmd/sdkgen/python_filters_test.go:TestPythonFilterExtractionDetectsInheritedAndImplementationDrift`, `internal/cmd/sdkgen/python_filters_test.go:TestPythonFilterManifestCannotReplaceLiveSourceProof`, `internal/cmd/sdkgen/python_filters_test.go:TestPythonFilterDescriptorSelectiveEmissionAndInventory`, `internal/cmd/sdkgen/python_filters_test.go:TestPythonFilterPinnedCheckoutVerification`.
- New opt-in source profile clones pinned Subnet/NetworkResource descriptor and query declarations; missing defaults, bool/int/list casting, ordered canonical/wire response aliases, project/tenant fallback, unknown Body and large integer tokens are owned. Row if_match is a constructor Header and actual page Header remains separate. Baseline native/Find/Resources policies stay unchanged.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsOwnedSparseDefaultsAndOrderedAliases`, `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsDescriptorsRowHeadersAndIndependentActualEvidence`.
- Canonical/wire Query mapping is pushed down without an invented name hint; non-query declared Body attributes filter loaded descriptors locally with Python loaded JSON equality. Unknown kwargs are discarded before application marshaling. Source max_items/limit raw behavior remains incomplete and is explicitly excluded below.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsPushdownLocalLoadedFiltersAndUnknownMarshalDiscard`.
- Proxy._list evaluates JMESPath against the actual opaque Resource.list generator, before Resource control normalization or HTTP. Index/projection/type expressions produce null without HTTP; literals preserve arbitrary JSON; @ returns the same generator, nested to_array keeps it unconsumed, and length errors before HTTP. ListRecords/AllRecords are ordinary resource-only convenience APIs.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsProxyGeneratorJMESAndLazyResourceControls`.
- Original bound self/resource_type/cls/session collisions and restored cls/session/base_path/paginated conflicts are eager. Truthy __conflicting_attrs restores ordered kwargs once; restored self/resource_type are discarded as unknown Resource attrs and restored JMES does not replace original expression. Resource headers/base_path/microversion are deferred until actual generator consumption, so an opaque empty literal avoids invalid controls entirely.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsOneLevelRestorationBindingAndImmutableOptions`, `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsProxyGeneratorJMESAndLazyResourceControls`.
- Only explicit empty string base_path produces an empty HTTP0 generator; null selects default, false/zero/list/object fails at consumption. Context/source checks still run before an empty generator returns. Literal same-origin service-relative path policy remains narrower than arbitrary Python formatting.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsEmptyPathOnlyNullDefaultAndDelayedCancellation`.
- Per-call microversion None keeps captured default and every other falsey JSON value disables it. Ordinary JSON numeric/string/list/object-key iterable normalization follows KSA conventions including true/1 -> 1.0, [2,4] -> 2.4, v2.4 -> 2.4, latest and major.latest rejection. Python overflow infinity/latest conversion remains unsupported.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsMicroversionJSONDomainAndDefaultDisable`.
- Source status100..399, plural singleton objects, required accepted JSON parsing, row constructor/descriptor faults, root links precedence, ignored dictionary next, HTTP/limit/raw-marker continuation use the shared guarded pager. Actual late-page body/read/Close failures preserve original native/ResponseError evidence and callbacks are suppressed only for physical faults; clean token renewal retains captured headers. Dynamic cap exact-boundary semantics remain named gaps.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsSourceLinkPrecedenceAndRawMarkerMaxItems`, `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsMalformedAcceptedPagesKeepPhysicalProof`, `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsLatePhysicalFaultsNative401AndNoFaultReplay`, `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsCleanReauthenticationAndCapturedHeaders`.
- The explicit Source-shaped list lane now accepts JSON falsey, true, negative and fractional max_items values, counts raw rows before local filtering, and defers truthy nonnumeric comparison errors until a nonempty raw row; falsey/container and raw limit hints preserve their separate wire forms.
  Tests: `network/v2/subnets/python_raw_max_items_test.go:TestPythonSubnetRawMaxItemsJSONDomainAndDeferredComparison`, `network/v2/subnets/python_raw_max_items_test.go:TestPythonSubnetRawMaxItemsFalseyAndRawLimitHint`, `network/v2/subnets/python_raw_max_items_test.go:TestPythonSubnetRawMaxItemsOpaqueJMESAndOwnedJSON`.
- The cap is checked before each next raw-row constructor. Exact-page cap boundaries may fetch a following page and expose its HTTP/read/Close/context failure; malformed rows beyond the cap are ignored and filtered rows count without refill. Legacy Find/native/typed pager policies remain separate.
  Tests: `network/v2/subnets/python_raw_max_items_test.go:TestPythonSubnetRawMaxItemsExactPageBoundaryObservesNextResponse`, `network/v2/subnets/python_raw_max_items_test.go:TestPythonSubnetRawMaxItemsCountsFilteredRowsAndStopsBeforeConstructor`.
- ProxyListRecords and the cloud list lanes: Raw limit values reach the wire with Requests spelling (False, -1, 1.5, x, repeated values) and Resource.list keeps the original limit for its loop: only a Python-truthy limit performs the marker GET after a nonempty page without a next link, so false/0/0.0/""/null send one GET. base_path="" never enters Resource.list's while loop: no HTTP and no KSA microversion normalization, so an invalid raw microversion is unobserved and the result is empty; a nonempty route still rejects it before HTTP.
  Tests: `network/v2/subnets/python_source_list_limits_test.go:TestPythonSubnetProxyRawLimitTruthiness`, `network/v2/subnets/python_source_list_limits_test.go:TestPythonSubnetProxyEmptyBasePathSkipsVersionNormalization`.

Python/Go differences:

- Go provides SDK-owned WithFilter/WithFilters for complete declared semantic classification, without caller builders or predicates, while WithBodyFilter/WithBodyFilters remain explicit local options and WithQuery remains independent wire data. Python canonical query names win in a bulk map; individual Go options use last-wins and nil/empty bulk replacement clears only semantic state. prefix_length is the semantic Python attribute; explicit Body options also accept raw prefixlen, which is unknown and discarded in the semantic namespace.
- Original page rows preserve unknown nested fields, JSON null elements, timestamp text, missing/null and exact numbers for local comparison; returned values remain native typed Subnets. Go distinguishes bool and number, requires exact ordered arrays/full inner objects and does not perform Python scalar-to-list coercion.
- The untyped Python prefix_length Body descriptor is preserved as raw JSON rather than inventing integer validation or converting response strings into numbers. revision_number is inherited type=int, but the native whole-page int decoder is retained first, so numeric strings/fractional/exponent spellings remain native decode boundaries.
- Native accepted statuses, full-current-page extraction and advertised subnets_links remain intact. max_items caps consumed raw rows locally without an inferred Python limit hint. Native typed List, FindIdentity and the prior four typed Body projections retain their existing behavior.
- Go defines scalar/repeated query encoding explicitly: booleans are lowercase, JSON number text is preserved, null elements are omitted and empty/null query values retain option presence without URL values. Query objects and nested arrays are rejected; no Subnet CSV format or native-q-tag-derived classification is inferred. Unknown semantic attributes are discarded, while explicit unknown Body options are errors and raw wire extensions remain caller-controlled.
- Semantic query name is a server filter rather than a local exact-name predicate. WithName retains its query hint and local exact match; duplicate ownership of the same query or Body destination is rejected across semantic/raw/page/hint/explicit namespaces, including equal values. This is an intentional Go conflict policy rather than permissive Python kwargs merging.
- An explicit semantic clear still requires an audited descriptor. Reserved session/paginated/base_path/allow_unknown_params/microversion/headers/max_items/resource_type/jmespath_filters names are not treated as field payloads. Generation additionally requires a live audited Python source checkout and its recorded AST parser minor version.
- Source-shaped lazy Subnet records and ordinary cloud list/search orchestration are owned by the SDK; existing native List/Resources/Find bindings retain their baseline policies.
- SDK APIs are concrete functional options rather than application builders/interfaces. JSON/individual options preserve source order; unspecified Go map order and cloud literal dictionary key results use deterministic lexical order. UTF8 JSON ownership, cancellation/source validation, typed positive controls and same-origin literal relative paths are explicit Go policies.
- Actual HTTP data is separated from Resource defaults/computed location and JMES literals. LastResponse identifies the last observed yielded resource/error receipt and is not a final-page receipt for empty or fully filtered pages. Native401 preserves ErrOriginal and ErrReauth despite absent Unwrap; physical faults are never presented as clean retry.
- Pinned Source evidence uses CPython3.14.8 actual cloud/Proxy/Resource/KSA code with identified adapter/config/HTTP fixtures, not a live backend or runtime authorization. Independent six fixture sets total696 observations including348 Subnet observations; KSA overflow-edge is header normalization only. Neutron27 stock get_subnet project-read policy and admin-only segment visibility do not guarantee live deployment permissions.
- Raw max_items behavior and exact-boundary cap timing are owned only by the explicit Source-shaped list path; historical typed/native contracts above retain their original boundaries.
- Only standard finite JSON numbers are accepted: NaN, Infinity and overflowing literals such as 1e400 return ErrUnsupported with the original response, where Python's json loads them as floats, and a latest microversion must be written as "latest"; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Results are immutable owned snapshots rather than Python's mutable Resource with adapter, session, cache and dirty state (SDK-R1/C1/S1); see Deliberate runtime differences in docs/sdk-support-ledger.md.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
subnets = conn.network.subnets(
    network_id="network-a",
    is_dhcp_enabled=False,
    prefix_length=24,
    dns_nameservers=["192.0.2.53"],
    max_items=100,
)
for subnet in subnets:
    print(subnet.id, subnet.cidr)
```

```go
package examples

import (
    "context"
    "fmt"

    networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListSubnets(ctx context.Context, service *networkv2.Service) error {
    for subnet, err := range service.Subnets.Resources.List(ctx,
        resource.WithFilters(map[string]any{
            "network_id": "network-a",
            "is_dhcp_enabled": false,
            "prefix_length": 24,
            "dns_nameservers": []string{"192.0.2.53"},
        }),
        resource.WithMaxItems(100),
    ) {
        if err != nil { return err }
        fmt.Println(subnet.ID, subnet.CIDR)
    }
    return nil
}
```

## References

- [Python Raw Max Items](python-raw-max-items.md)
- [AST manifest](../../../api/openstacksdk/resources/network/v2/subnet.json)
- [SDKgen](../../../internal/cmd/sdkgen/README.md)
- [Listing](../../../docs/listing.md)
- [Finding Identities](../../../docs/finding-identities.md)
- [Network Subnet Semantic Filters Test](../../../api/network_subnet_semantic_filters_test.go)
- [Network Subnet Raw Body Filters Test](../../../api/network_subnet_raw_body_filters_test.go)
