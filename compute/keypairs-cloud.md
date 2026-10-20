# Keypairs Cloud

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_compute/ComputeCloudMixin/list_keypairs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.ListKeypairs`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithKeypairQueryOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithKeypairQueryFilters`

- ListKeypairs eagerly consumes the existing owned KeyPairs.ListRecords inventory in page order, preserving duplicate records. Value and Keypairs represent a completed list only after full successful consumption; a late page failure retains prior yielded Inventory records and original physical HTTP evidence but cannot return a completed list. Inventory contains the leaf's first-stage filtered yielded records, not every physical raw page row.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairQueryCompositionAndPresence`, `api/compute_keypair_records_test.go:TestComputeKeypairRecordsContinuationLogicalMarkersCapsAndLateErrors`.
- Absent/null or falsey filters use the unfiltered source-list branch, while a dictionary reuses the leaf's server query and declared normalized Body filters. UserID, limit and marker remain server parameters, the eight other canonical attributes use the existing local matcher and raw max_items precedes that matcher. Source-shaped wrapped/flat/default/nullable Resource projection, complete actual row Wire/Envelope and guarded paging remain the same leaf engine rather than native single-page All.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairQueryCompositionAndPresence`, `api/compute_keypair_records_test.go:TestComputeKeypairRecordsProjectionSemanticQueriesAndLazyOwnership`, `api/compute_keypair_records_test.go:TestComputeKeypairRecordsContinuationLogicalMarkersCapsAndLateErrors`.
- The common Cloud read preparation snapshots filters JSON, typed version/pagination pointers and callback header maps once for the eager operation. The captured selected version, live provider token, leaf cap/versionless policy and original service binding remain guarded. Nil options, invalid filters, negative caps, source mutation and callback cancellation fail before HTTP; original callback, context and custom-cause errors remain observable.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairOptionsOwnershipAndTerminalErrors`, `api/compute_keypair_records_test.go:TestComputeKeypairRecordsVersionPreflightAndAcceptedFaultBinding`.
- The shared Cloud location stage is captured once before option callbacks and owned separately for each yielded Resource. The Cloud Search composition fixture proves the same inventory preparation preserves the current cloud/project location for nested local filtering, ignores later caller pointer/raw-byte mutation, sends no location query, and leaves actual per-row Wire/Envelope and completed JSON Value independently owned. Standalone Cloud create uses null location when no adapter exists.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairOptionsOwnershipAndTerminalErrors`, `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairCreateAndDeleteOutcomes`.

Python/Go differences:

- Python returns an eager list of connected mutable Keypairs. Go returns complete normalized JSON Value/Keypairs plus separately retained partial Inventory on error. Each record carries an owned Resource and actual Wire/raw row receipt; generic mutable Resource methods and session/cache facilities retain their own SDK scope.
- Cloud Resource adds a current configured/recorded Connection location to the existing nine Body attributes; standalone services without that dependency use JSON null. Location never grants authority to a URL, and actual response Wire/Envelope remains unchanged. Source/session computed attribute hydration is represented by this finite library-owned view rather than arbitrary mutable Resource kwargs.
- The mapped leaf's documented concrete controls and finite discovery are reused: typed nonnegative cap, guarded continuations, explicit-empty versionless override and default selected-or-ceiling2.10. Dictionary links.next is a Go compatibility extension, not the pinned source ordinary-next dictionary behavior. Arbitrary base_path, deprecated list jmespath_filters and allow_unknown_params remain explicitly unsupported shared controls rather than wire filters.
- Only list_keypairs is mapped here. Source Cloud search and get add independent second-stage selection, while native single-page typed List/All and Proxy/keypairs keep their separate reviewed declarations. Own-user access is core user behavior; foreign-owner access is left to actual Nova policy and its HTTP failures.

### `python:cloud/_compute/ComputeCloudMixin/search_keypairs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.SearchKeypairs`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithKeypairQueryFilters`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithKeypairQueryExpression`

- Dictionary search first consumes ListRecords using declared server/Body filters, then applies the same original ordered JSON dictionary again to the complete normalized inventory. A requested user_id is therefore a server query and a second-stage response predicate. A first Body mismatch removes the row before later Cloud keys; among surviving rows an earlier mismatch leaves later missing attributes unconsumed while an earlier missing key returns a local error, rather than unknown-discard at the second stage.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairQueryCompositionAndPresence`, `internal/cloudfilter/mapping_test.go:TestCloudFilterOrderedMappingConsumesOnlyReachedKeysAndKeepsDuplicateFirstPosition`, `internal/cloudfilter/mapping_test.go:TestCloudFilterNestedEmptyObjectsUseActualTruthinessBeforeShapeOrMissingKeys`.
- Identity selection runs exact name/id and Python-style glob matching over normalized logical identities before the local filter phase. Numeric logical identities are stringified for matching, and matching order and duplicates remain intact. A string filter is not forwarded to ListRecords and the shared JMESPath engine can return arbitrary JSON Value; expression output never invents an associated Keypair record.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairQueryCompositionAndPresence`, `internal/cloudfilter/identifier_test.go:TestCloudFilterIdentifiersUsePythonJSONDomainStringAndContainerRepresentation`, `internal/cloudfilter/results_test.go:TestCloudFilterExpressionBridgeReturnsArbitraryOwnedJSONWithoutInventedRowIndices`.
- Search never resolves or returns an early match before complete inventory. A later403 or accepted read/Close nested404 stays an original error with partial Inventory/actual receipt and no completed Value or selected records. Options and guarded source facts are prepared once for all pages; source/callback cancellation prevents subsequent work and preserves all original causes.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairQueryCompositionAndPresence`, `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairOptionsOwnershipAndTerminalErrors`.
- Configured current location is captured once before options, participates in the second-stage nested Cloud dictionary filter without becoming a wire parameter, and is independently owned per row. Callback mutation of the original cloud/region/project pointers or raw bytes cannot alter the captured view; returned row location edits do not alter another row, actual Wire/Envelope or completed Value.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairOptionsOwnershipAndTerminalErrors`.

Python/Go differences:

- Source annotations say list, but JMESPath can return a scalar/object/null. Go preserves that result explicitly as Value; Keypairs is populated only when source rows retain an actual record association. Error results retain first-stage yielded Inventory rather than discarding all prior HTTP work.
- Raw UTF-8 JSON preserves dictionary insertion order and duplicate-key last value at the first insertion position. Unknown first-stage semantic attributes are discarded like Resource.list, but reached second-stage missing attributes are errors like _utils._filter_list; the source docstring's broad invalid-filter-ignored wording is not used to suppress the literal helper's error.
- Dictionary list controls use the existing dedicated typed bridge, including pagination/cap/microversion/headers, then remain in the unchanged second-stage dictionary if reached. Positive-limit/safe-marker/nonnegative-cap and header restrictions are deliberate leaf Go policies. Unsupported base_path/deprecated list jmespath_filters/allow_unknown_params and arbitrary non-JSON Python objects remain separate shared SDK controls, while the directly declared Cloud string JMES filters are implemented here.
- Normalized current Connection location is a library-owned tenth Cloud view field, standalone null, while the leaf remains nine fields and actual Wire/Envelope does not change. Location/session mutability and every inherited Resource declaration are separate SDK scope, not additional undeclared parameters of this search.

### `python:cloud/_compute/ComputeCloudMixin/get_keypair`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetKeypair`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithKeypairQueryFilters`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithKeypairQueryUserID`

- Absent or explicit JSON null filters delegate to the existing FindKeypair with the supplied owner. Even a present empty dictionary or empty string delegates to complete Cloud search instead of member lookup and intentionally does not forward UserID. This follows filters-is-not-None rather than filters truthiness.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairQueryCompositionAndPresence`, `api/compute_keypair_find_test.go:TestComputeFindKeypairDirectSeedAndOwnerPreservingFullListFallback`.
- Filtered get waits for complete inventory before shared First selection: an empty result is missing, multiple matching records produce ErrAmbiguous, and a one-item JMES result containing false remains present false Value with no invented Keypair association. Arbitrary expression truthiness/length/integer-index failures remain errors instead of forcing every expression into a resource list.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairQueryCompositionAndPresence`, `internal/cloudfilter/results_test.go:TestCloudFilterFirstFollowsTruthyLengthAndIntegerIndexWithoutRetestingSelectedFalsyValue`.
- The unfiltered branch retains the leaf's always-member escaped GET, default ignored missing, clean400/403/404 same-owner complete-list fallback, duplicate and late error policies. Accepted nested404 never becomes a missing value or fallback: filtered inventory read failures retain actual receipt and leave completed Value/Keypair nil. Source/options/version ownership is shared across the whole Cloud operation.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairOptionsOwnershipAndTerminalErrors`, `api/compute_keypair_find_test.go:TestComputeFindKeypairDirectSeedAndOwnerPreservingFullListFallback`, `api/compute_keypair_find_test.go:TestComputeFindKeypairOwnedOptionsPreflightAndPhysicalTerminalErrors`.
- Unfiltered member get attaches one current location snapshot to Resource while preserving the response location in Wire. The same configured fixture proves create captures its own prior location before its option callback and the next get sees the later current location rather than a stale cached value. No additional member GET is introduced for location.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairOptionsOwnershipAndTerminalErrors`.

Python/Go differences:

- Filters pointer presence distinguishes absent from present empty, with JSON null representing Python None. Go replaces the deprecated Python filters warning with concrete options and returns Value plus an optional actual Keypair and partial Inventory. JMES results need not be resources; missing is nil Value and no record while false/zero/empty selected elements are represented by raw JSON.
- UserID is consumed only by unfiltered member Find. Filtered get's ignored owner is the pinned Cloud behavior, not a guessed permission rule. The owner can still be expressed as a dictionary filter, which then participates in both server query and local response comparison.
- Unfiltered leaf identity accepts validated literal UTF-8 names with escaped reserved characters and spaces, rejecting empty/control/dot identities. Typed operation controls/versionless override and finite discovery remain the existing documented Go policies. No _utils._get_entity Resource-object/direct-UUID shortcut is asserted because this pinned method does not call that helper.
- Cloud Resource adds finite current location without changing leaf Wire/Envelope or Python mutable-resource identity. Full session/cache/mutable Resource and unsupported inherited list controls retain independent goal tracking, while the named get filters-presence/selection branches are mapped explicitly.

### `python:cloud/_compute/ComputeCloudMixin/create_keypair`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.CreateKeypair`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithCloudKeypairPublicKey`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithCloudKeypairCreateHeader`

- Cloud CreateKeypair always supplies name as literal body data, including an empty or path-like body name, and supplies public_key only for a nonempty string. Empty public key is omitted rather than forwarded as the leaf's explicit empty attribute. The library composes the existing one fixed keypair-enveloped collection POST without lookup, wait or cleanup.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairCreateAndDeleteOutcomes`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateDirtyBodyAliasesAndNativeCompatibility`.
- The existing owned creation result preserves original input seed/defaults and actual response overlay, including absent type=ssh only in Resource, present null authority, logical name/id aliases, independently owned Wire and physical receipt. HTTP and accepted read/Close/source errors remain original causes beside actual receipt instead of a second Cloud creation or synthetic successful result.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairCreateAndDeleteOutcomes`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateResponseSeedWireAndPassiveErrors`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateOwnedOptionsSourceAndAcceptedFailures`.
- Creation captures configured Cloud location once before its option callback and keeps that snapshot in Resource when the callback changes the caller cloud pointer. Location is omitted from the transmitted body, while the actual response location remains unchanged in Wire and Envelope. A service without the adapter projects null location.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairOptionsOwnershipAndTerminalErrors`, `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairCreateAndDeleteOutcomes`.

Python/Go differences:

- Python name/public_key string or None inputs become a required Go name argument and optional concrete PublicKey string option; empty represents the source falsey/None omission. Full nullable/type/user_id/vendor body control remains available through the separate versioned CreateKeypair, not invented arguments of the Cloud convenience.
- Go returns the owned KeypairRecord and adds finite current Connection location to Resource only; standalone location is null. It does not return a connected mutable Python object or populate view defaults/current location in actual Wire/Envelope.
- The versioned create engine retains selected microversion rather than adding Python session discovery for this mutation. Its documented final200..399/accepted opaque-body tolerance/strict parsed-object/physical receipt policies and deterministic alias choices are reused. Nova still validates names/types/owner permissions; mutable session/discovery declarations remain independent SDK scope.
- This row closes the two-input Cloud creation transform only and does not incidentally promote native/Proxy create, server creation, keypair deletion or unrelated service workflow declarations.

### `python:cloud/_compute/ComputeCloudMixin/delete_keypair`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.DeleteKeypair`

- Cloud DeleteKeypair forces strict missing handling on the existing owned bodyless fixed-name DELETE. Accepted success returns true,nil; only classified clean actual404 for the exact DELETE method/member URL returns false,nil. No preliminary name resolution, GET, wait or cleanup is introduced, and HTTP403 remains an error with original physical cause.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairCreateAndDeleteOutcomes`, `api/compute_keypair_delete_test.go:TestComputeKeypairDeleteOwnedDefaultsOwnerAndNativeCompatibility`.
- Accepted Close nested404 remains a terminal ResponseError with the real202 body/header and nested original cause, never ordinary false,nil. A missing response combined with captured-source failure or cancellation/custom cause remains an error. Source guard and actual method/URL evidence prevent an unrelated native404 from being reclassified as Cloud missing.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairCreateAndDeleteOutcomes`, `api/compute_keypair_delete_test.go:TestComputeKeypairDeleteAcceptedFailuresAndNativeRetryGuards`.
- The configured create/get/delete fixture proves DeleteKeypair performs its single fixed DELETE without invoking the CloudLocation dependency or fetching a record. A retry callback returning native404 remains an error with its original retry cause and does not become ordinary false,nil.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairOptionsOwnershipAndTerminalErrors`, `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairCreateAndDeleteOutcomes`.

Python/Go differences:

- The bool result matches Cloud success/missing polarity, while Go errors.Is/errors.As preserves original failure causes. The standalone versioned DeleteKeypair still defaults to ignored missing and remains error-only; this Cloud wrapper always chooses strict false explicitly.
- The mapped leaf's safe validated single-segment string name and selected microversion policy remain in place rather than arbitrary Python Resource or session-default negotiation. This wrapper has no user_id argument, lookup, deletion completion wait or detached cancellation bypass.
- Final200..399 passive-body handling uses the existing owned mutation engine; native strict202/204 and Remove lookup semantics retain their separate declarations. Accepted read/Close/source/context/other404 faults are intentionally kept visible rather than caught as Python broad NotFound alone.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
all_keys = conn.list_keypairs()
work_keys = conn.search_keypairs("work-*", filters={"type": "ssh"})
key = conn.get_keypair("workstation", user_id="owner-user-id")
created = conn.create_keypair("workstation", public_key="ssh-ed25519 ...")
deleted = conn.delete_keypair("workstation")
```

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
)

func main() {
    operation := flag.String("operation", "list", "list, search, get, create, delete")
    name := flag.String("name", "", "keypair name or search pattern")
    filters := flag.String("filters", "", "optional filters as JSON")
    owner := flag.String("owner-user-id", "", "owner for unfiltered get")
    publicKey := flag.String("public-key", "", "optional public key for create")
    flag.Parse()
    filtersSet := false
    flag.Visit(func(f *flag.Flag) {
        if f.Name == "filters" { filtersSet = true }
    })
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *operation, *name, *filters, filtersSet, *owner, *publicKey); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, operation, name, filters string, filtersSet bool, owner, publicKey string) error {
    conn, err := sdk.Connect(ctx)
    if err != nil { return err }
    service, err := conn.Compute(ctx)
    if err != nil { return err }
    options := []compute.KeypairQueryOption{compute.WithKeypairQueryUserID(owner)}
    if filtersSet {
        options = append(options, compute.WithKeypairQueryFilters(json.RawMessage(filters)))
    }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    printResult := func(value any, callErr error) error {
        return errors.Join(callErr, encoder.Encode(value))
    }
    switch operation {
    case "list":
        result, err := service.ListKeypairs(ctx, options...)
        return printResult(result, err)
    case "search":
        result, err := service.SearchKeypairs(ctx, name, options...)
        return printResult(result, err)
    case "get":
        result, err := service.GetKeypair(ctx, name, options...)
        return printResult(result, err)
    case "create":
        result, err := service.CreateKeypair(ctx, name, compute.WithCloudKeypairPublicKey(publicKey))
        return printResult(result, err)
    case "delete":
        deleted, err := service.DeleteKeypair(ctx, name)
        return printResult(map[string]bool{"deleted": deleted}, err)
    default:
        return fmt.Errorf("unknown operation %q", operation)
    }
}
```

## References

- [Keypairs List Find](keypairs-list-find.md)
- [Testing](../docs/testing.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
