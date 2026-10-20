# Python Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/extensions`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListConflictingAttrs`

- List sends GET extensions and decodes alias, name, description and the raw updated string.
  Tests: `network/v2/extensions/python_parity_core_test.go:TestPythonParityCoreExtensionListAndGetWithoutFallback`.
- Source-shaped extension list implements all six effective Body fields/defaults with direct Resource MRO, actual alias alternate ID by primary presence, canonical/wire updated alias ordering and unknown body/raw timestamps. Nested links type=list/list_type=dict preserves top null, wraps nonlist and converts every nonmapping element including null to an empty dictionary, independently of actual Wire/whole-page/Header/status.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsDefaultMROAliasIDAndNestedLinksCast`.
- Inherited limit/marker are the only server queries. All declared Body attrs including id/alias/name/description/links/updated_at are owned local filters; unsupported fields/wire updated/unknown kwargs are discarded before value encoding. Dictionary subset, bool-number and loaded CPython float equality are shared library behavior. No member fetch/name query hint is invented.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionListRecordsAllBodyFiltersAndQueryControlsAreLibraryOwned`.
- Shared inherited pager handles actual next/Link/plural links/marker with original primary-id presence and alias alternate only when absent. Raw cap counts excluded rows and stops before unconsumed constructor/foreign continuation. Actual primary null/empty never switches to alias; page/late failures preserve whole actual receipt and AllRecords is atomic.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionListAlternateMarkerPrivateProofAndRawCaps`, `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsMalformedEnvelopeAndLateErrorsKeepActualProof`.
- Actual Source Proxy receives an opaque unconsumed generator for deprecated JMESPath: projection null, literal JSON, identity equality, runtime errors and nested generators preserve zero HTTP until consumption. Original expression remains separate from restored unknown JMES; one-pass conflicting attrs and concrete header/version/base-path controls use the shared normalizer. No scalar/object expression is fabricated into a Resource or successful HTTP receipt.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionProxyJMESGeneratorResultsAndSourceConflictControls`.
- Source/option snapshots, initialized callback maps and concurrently reusable factories retain request ownership. Nil/bound duplicate options and malformed constructor/query/base paths fail before HTTP. Physical read/Close/cancel/source faults and native pointer401 retain all causes/actual evidence without invoking native retry/reauth; clean retry/token renewal preserves semantics and forbidden auth mutation cannot turn original503 into success.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsSnapshotsNilMapsConcurrentReuseAndPreflight`, `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsPhysicalFaultsNeverFallbackRetryOrReauth`, `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsCleanRetryTokenRenewalAndProtocolGuard`.
- Source JSON paging honors present root links over plural links, ignores dictionary values, selects first literal lowercase rel next, falls back to top-level next and preserves actual malformed links receipt. HTTP Link interoperability remains explicit.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsSourceLinkPrecedenceAndActualMalformedProof`.

Python/Go differences:

- Python can send the default limit and marker query keys; the Go extension list accepts no query.
- Go reads one page and does not follow extensions_links; Neutron returns the extension list in one response.
- Python filters Body attributes such as name locally; the Go caller filters the returned rows.
- Owned Extension records are sibling to existing native Get/List and preserve full declared/default JSON and actual evidence. Extension derives from Resource directly, with no invented NetworkResource Header/tag/revision declarations.
- Go immutable eager JSON view differs from Source lazy/mutable Resource/Computed/current_location/cache/dirty lifecycle and arbitrary class/session injection. location is SDK-owned null and Wire remains actual. Known runtime/constructor/control replacement is bounded explicitly rather than a claim of full shared Resource parity.
- Default/selected network microversion has no Extension class maximum to negotiate. Concrete safe header maps and fixed source/origin/path/filter/continuation policies bound broad Source adapters/URI percent-formatting. Empty path means default in Go, and HTTP Link interoperability is explicit. Source100..399 acceptance and JSON link precedence are opted in; invalid UTF8 is rejected rather than replaced.
- Primitive scalar/list/dictionary descriptors, nested links dict element null handling and CPython-loaded numeric equality use shared library contracts. Source runtime custom objects/subclasses and configurable integer-string runtime constraints are outside this JSON interface; unknown actual JSON/large integers remain raw and no user decoder/builder is required.
- Neutron27 extension controller alias/list routes have no dedicated ADMIN gate. Catalog/auth middleware and deployment extension visibility remain authoritative; fixture/source/race/main evidence does not assert anonymous access, elevated roles or enabled drivers/backends.
- Typed ListRecords/AllRecords yield resource-only values; ProxyListRecords owns arbitrary JSON or nested opaque generator results. The full Source generator branch is preserved rather than evaluated over collected arrays, and mutable Python iterator/session lifecycles remain separate shared SDK foundations.

### `python:network/v2/find_extension`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordFindListBasePath`

- Get sends GET extensions/{name} and returns the 404 without listing extensions.
  Tests: `network/v2/extensions/python_parity_core_test.go:TestPythonParityCoreExtensionListAndGetWithoutFallback`.
- GET-first Source find escapes the literal identity segment, preserves raw member kwargs/constructor seed, consumes formals and projects actual response aliases/defaults without confusing seed with Wire. Actual explicit null id wins over requested seed; accepted empty/nonJSON responses retain seed/default Resource with Wire nil and independent actual HTTP evidence.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionFindMemberSeedExplicitNullAndPrivateActualEvidence`.
- Only clean member400/403/404 trigger fallback. Extension has no name query so no automatic name hint; complete list searches exact alias-derived ID or name and detects late duplicates. Successful full absence is defaultnil,nil; strictErrNotFound, LIST403 and local Body conditions retain distinct phases without hiding permission or late/physical failures.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionFindAliasNameFallbackHasNoServerNameHintAndDetectsLateAmbiguity`, `network/v2/extensions/python_record_contracts_test.go:TestExtensionFindMissingForbiddenAndLocalBodyFiltersPreserveTheirPhases`.
- Owned ListBasePath is a fallback-only route override with cloned pointer; the member stays at default extensions/{literal}. Explicit Find microversion affects only GET and fallback list keeps original selected/default version. Shared safe header policy, named/typed formal values and immutable selected client remain consistent through both phases.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionFindListBasePathAndGetOnlyVersionAreIndependent`.
- Library-owned options/initialized maps and source guards surround callbacks. Bound identity/ignore_missing/class controls reject before application marshaling, unknown response JSON remains visible, and nested links cast uses the common descriptor. Physical/cancel/source failures never become clean404 fallback or missing success, native401 pointer retains actual receipt/handling causes, and clean renewal/retry preserves live tokens with fixed protocol ownership.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsSnapshotsNilMapsConcurrentReuseAndPreflight`, `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsPhysicalFaultsNeverFallbackRetryOrReauth`, `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsCleanRetryTokenRenewalAndProtocolGuard`.
- Extension six-control collection GET retains unknown constructor seed while distinguishing physical Wire/Envelope; alias-only GET response leaves requested seed id, explicit null id wins, fallback alias-derived ID and updated/updated_at literal and links element projections remain resource-specific. Malformed selected envelopes retain actual accepted response proof.
  Tests: `network/v2/extensions/fetch_controls_test.go:TestExtensionFindFetchControlsCollectionKeepsConstructorSeedAndAliasProjection`, `network/v2/extensions/fetch_controls_test.go:TestExtensionFindFetchControlsEnvelopeNullIDAndMalformedProof`.
- Raw headers remain GET mapping-key query values and become fallback HTTP headers with case-insensitive raw-header precedence over typed headers; fallback base path is independent, invalid truthy headers fail only after rejected GET reaches LIST, and no invented name query hint is added for Extension.
  Tests: `network/v2/extensions/fetch_controls_test.go:TestExtensionFindFetchControlsRawHeadersUseGETQueryThenFallbackTransport`, `network/v2/extensions/fetch_controls_test.go:TestExtensionFindFetchControlsRawHeadersValidateOnlyAtFallback`.
- Raw options are owned, repeated calls invoke the stateless reader without cache headers, custom HTTP error causes retain physical proof and clean fallback semantics, while read/Close/cancel faults remain terminal without fallback.
  Tests: `network/v2/extensions/fetch_controls_test.go:TestExtensionFindFetchControlsOwnOptionsWithoutCacheAndKeepHTTPCause`, `network/v2/extensions/fetch_controls_test.go:TestExtensionFindFetchControlsPhysicalFaultsNeverFallback`, `internal/neutronfind/find_fetch_route_test.go:TestFindWithFetchControlsDefaultRouteKeepsCanonicalContinuationAlias`, `internal/neutronfind/find_fetch_route_test.go:TestFindWithFetchControlsExplicitRouteKeepsContinuationInsideSelectedCollection`.

Python/Go differences:

- Owned Extension records are sibling to existing native Get/List and preserve full declared/default JSON and actual evidence. Extension derives from Resource directly, with no invented NetworkResource Header/tag/revision declarations.
- Go immutable eager JSON view differs from Source lazy/mutable Resource/Computed/current_location/cache/dirty lifecycle and arbitrary class/session injection. location is SDK-owned null and Wire remains actual. Known runtime/constructor/control replacement is bounded explicitly rather than a claim of full shared Resource parity.
- Default/selected network microversion has no Extension class maximum to negotiate. Concrete safe header maps and fixed source/origin/path/filter/continuation policies bound broad Source adapters/URI percent-formatting. An empty typed fallback-only ListBasePath retains the selected path, whereas the new raw base_path="" selects the service root. HTTP Link interoperability is explicit. Source100..399 acceptance and JSON link precedence are opted in; invalid UTF8 is rejected rather than replaced.
- Primitive scalar/list/dictionary descriptors, nested links dict element null handling and CPython-loaded numeric equality use shared library contracts. Source runtime custom objects/subclasses and configurable integer-string runtime constraints are outside this JSON interface; unknown actual JSON/large integers remain raw and no user decoder/builder is required.
- Neutron27 extension controller alias/list routes have no dedicated ADMIN gate. Catalog/auth middleware and deployment extension visibility remain authoritative; fixture/source/race/main evidence does not assert anonymous access, elevated roles or enabled drivers/backends.
- Go takes explicit string identity and owned concrete option snapshots; arbitrary Python Resource/Adapter objects, overridden constructors and runtime kwargs/control replacement are separately tracked shared contracts. Returned seeded member ID is the Source constructor view and is not represented as actual Wire identity.
- Owned FindRecord explicitly opts into six raw fetch controls while preserving Extension unknown constructor Body, alias-derived alternate ID and updated/updated_at literal projection. URI formatting is limited to %(id)s and %%, with finite UTF-8 JSON/string identities and string-valued fallback headers in safe service-local routes. Other leaf/native find profiles remain unchanged. skip_cache invokes the stateless reader without an invented cache header, and bypass of caller custom transport cache is not claimed; HTTP messages preserve Go causes rather than Python exception classes.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import types
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("list", "find", "proxy"), default="list")
parser.add_argument("--extension", default="qos")
parser.add_argument("--strict", action="store_true")
parser.add_argument("--jmes", default="")
parser.add_argument("--list-base-path")
args = parser.parse_args()

with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "find":
        query = {"skip_cache": True, "error_message": "Extension lookup failed"}
        if args.list_base_path is not None:
            query["list_base_path"] = args.list_base_path
        value = conn.network.find_extension(
            args.extension, ignore_missing=not args.strict, **query
        )
        print(json.dumps(None if value is None else value.to_dict(), indent=2))
    else:
        query = {}
        if args.mode == "list":
            query["alias"] = args.extension
        else:
            query["jmespath_filters"] = args.jmes
        result = conn.network.extensions(**query)
        if isinstance(result, types.GeneratorType):
            for value in result:
                print(json.dumps(value.to_dict(), indent=2))
        else:
            print(result)
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    mode := flag.String("mode", "list", "list, find or proxy")
    identity := flag.String("extension", "qos", "alias or exact display name")
    strict := flag.Bool("strict", false, "return an error when missing")
    expression := flag.String("jmes", "", "deprecated Proxy expression")
    listPath := flag.String("list-base-path", "", "Find fallback service-relative path")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := network.Extensions
    switch *mode {
    case "find":
        options := []extensions.RecordFindOption{
            extensions.WithRecordFindIgnoreMissing(!*strict),
            extensions.WithRecordFindQuery("skip_cache", true),
            extensions.WithRecordFindQuery("error_message", "Extension lookup failed"),
        }
        if *listPath != "" {
            options = append(options, extensions.WithRecordFindListBasePath(*listPath))
        }
        value, err := api.FindRecord(ctx, *identity, options...)
        if err != nil { log.Fatal(err) }
        if value == nil { printJSON(nil); return }
        printJSON(value.Resource)
    case "list":
        for value, err := range api.ListRecords(
            ctx, extensions.WithRecordListQuery("alias", *identity),
        ) {
            if err != nil { log.Fatal(err) }
            printJSON(value.Resource)
        }
    case "proxy":
        result, err := api.ProxyListRecords(ctx, extensions.WithRecordListJMESPath(*expression))
        if err != nil { log.Fatal(err) }
        if result.JSON != nil { fmt.Println(string(result.JSON)); return }
        generator, ok := result.Value.(*extensions.RecordListGenerator)
        if !ok { fmt.Printf("opaque expression result: %T\n", result.Value); return }
        for value, err := range generator.Records {
            if err != nil { log.Fatal(err) }
            printJSON(value.Resource)
        }
    default:
        log.Fatal("mode must be list, find or proxy")
    }
}
```

## References

- [Extensions](https://github.com/openstack/neutron/blob/27.0.0/neutron/api/extensions.py#L100-L127)
- [Install](../../../docs/install.md)
- [Extension class](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/extension.py#L16)
- [find/list proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L1588-L1626)
- [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2489)
- [list element cast](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86-L113)
