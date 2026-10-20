# Keypairs List Find

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/keypairs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairListFilter`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairListFilters`

- Wrapped and flat rows produce a nine-attribute source-shaped Resource: nested keypair fields override outer fields, canonical name overrides literal id including null/empty, logical id follows name, omitted type defaults to ssh and deleted uses response Boolean truthiness. Resource fields and headers are independent of the complete actual outer-row Wire, raw row Envelope and record headers, including precise extension numbers.
  Tests: `api/compute_keypair_records_test.go:TestComputeKeypairRecordsProjectionSemanticQueriesAndLazyOwnership`.
- Semantic user_id/limit/marker go to the server while id/created_at/is_deleted/fingerprint/name/private_key/public_key/type filter the normalized Resource locally. Unknown attributes and wire deleted as a semantic key are ignored. Option maps/pointers/carriers are owned; list construction is lazy and each repeated iteration applies callbacks once to fresh state.
  Tests: `api/compute_keypair_records_test.go:TestComputeKeypairRecordsProjectionSemanticQueriesAndLazyOwnership`.
- Advertised body links, keypairs_links, top-level next and HTTP Link continue the fixed collection. A filtered-out last row still supplies its normalized name as fallback marker, caller mutations do not change that marker, raw max_items precedes local filtering, single-page and break stop requests, and an empty page stops even with next. A first server-introduced positive limit does not activate marker fallback. Late403 retains already yielded results and actual HTTP evidence.
  Tests: `api/compute_keypair_records_test.go:TestComputeKeypairRecordsContinuationLogicalMarkersCapsAndLateErrors`, `internal/rest/list_test.go:TestListDictionaryLinksAreExplicitAndGuarded`.
- A selected Compute version is retained; otherwise finite Nova discovery selects at most2.10. Explicit empty overrides selection for versionless reads on a private client. Invalid controls/collisions/options and changed source fail before subsequent work; accepted page read/source faults remain ResponseError with actual body/header/status and cannot be replayed or mistaken for missing.
  Tests: `api/compute_keypair_records_test.go:TestComputeKeypairRecordsVersionPreflightAndAcceptedFaultBinding`.
- The plural keypairs envelope accepts an array or one object. Owned accepted204 still requires a list representation rather than silently becoming an empty native page; parsed or physical failures retain accepted response evidence.
  Tests: `api/compute_keypair_records_test.go:TestComputeKeypairRecordsProjectionSemanticQueriesAndLazyOwnership`, `api/compute_keypair_records_test.go:TestComputeKeypairRecordsVersionPreflightAndAcceptedFaultBinding`, `internal/rest/list_test.go:TestListOptionalReaderAndRepresentationsKeepDefaultPolicy`.

Python/Go differences:

- Concrete With-functions and iter.Seq2 replace Python kwargs and its generator. The owned Resource exposes the nine declared keypair Body attributes; computed Connection location and mutable Python Resource methods are not synthesized by this leaf API. Wire and raw row Envelope retain the actual response without normalized aliases/defaults.
- Typed Limit/MaxItems are nonnegative with zero meaning omitted/unbounded; a present pagination limit must be one positive integer and marker one nonempty string. Raw queries override same typed queries; semantic query collisions with typed/raw inputs are rejected rather than silently overwriting them. Unknown semantic fields are discarded; explicit raw query extensions remain separate.
- Operation controls use dedicated options. Arbitrary base_path, deprecated jmespath_filters and allow_unknown_params are explicitly unsupported instead of becoming wire filters. Continuations are guarded to the captured collection route and query; inconsistent/cyclic links fail. Dictionary links.next is an installed Go compatibility extension; the pinned source ordinary-next dictionary does not follow that form.
- Nil operation microversion uses the captured selected version or the shared finite Nova discovery capped at2.10. Explicit empty deliberately performs versionless reads; no mutation of the original client or automatic upgrade to the newer keypair pagination version occurs. This does not reproduce KSA's entire discovery/cache/session system.
- Source JSON fields remain passive raw values, while malformed object shapes, physical accepted faults and Go context/source errors are explicit. The native List/All results and single-page policy remain separate and unchanged.
- Canonical deleted wins a conflicting is_deleted response field deterministically; the pinned Python body component may depend on insertion order when both wire and alias names are present.

### `python:compute/v2/find_keypair`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.FindKeypair`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairFindUserID`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairFindIgnoreMissing`

- Find always attempts the escaped fixed member GET for accepted literal UTF-8 input and optional truthy owner. A successful member response returns its input-name/owner-seeded Resource with response fields overriding seed including null, while Wire and Envelope contain only actual response data. Opaque/malformed/empty accepted bodies retain the seed with nil Wire; valid nonobject/envelope failures retain a receipt beside the error.
  Tests: `api/compute_keypair_find_test.go:TestComputeFindKeypairDirectSeedAndOwnerPreservingFullListFallback`, `resource/find_identity_response_id_test.go:TestCollectionFindIdentityPassiveIDsAreOnlyCompared`.
- Only clean actual400/403/404 permits a full owned list fallback with the same owner, version, headers and live provider token; no name query hint is added. Passive numeric names do not equal a string identity. Unique matches consume the complete list, duplicate matches raise ErrAmbiguous, late list errors remain errors, default missing returns nil,nil and strict missing returns ErrNotFound.409/500 do not fall back.
  Tests: `api/compute_keypair_find_test.go:TestComputeFindKeypairDirectSeedAndOwnerPreservingFullListFallback`.
- Automatic ceiling2.10 selection occurs once across direct GET and fallback, and concrete options apply once for the whole find. Explicit empty preserves the original selected client while issuing both phases versionless with owned option pointers/maps/headers and final missing-policy precedence.
  Tests: `api/compute_keypair_find_test.go:TestComputeFindKeypairDirectSeedAndOwnerPreservingFullListFallback`, `api/compute_keypair_find_test.go:TestComputeFindKeypairOwnedOptionsPreflightAndPhysicalTerminalErrors`.
- Nil/invalid inputs, options, canceled context, outer guard and changed source fail before subsequent callbacks/HTTP. Accepted read/Close/source failures with nested404 are terminal and preserve the actual member receipt, or discovery evidence without fabricating a member result. Native retry source mutation cannot issue another member/fallback; live reauthentication preserves the fixed member and owner.
  Tests: `api/compute_keypair_find_test.go:TestComputeFindKeypairOwnedOptionsPreflightAndPhysicalTerminalErrors`.

Python/Go differences:

- FindKeypair is an additive concrete-option API returning the owned KeypairRecord. Existing Find(resource.Ref), Get and their native KeyPair ABI retain their explicit lookup and strict native decode policies. Default missing is true without Python's deprecation warning.
- Literal identity validation rejects empty/whitespace-only, invalid UTF-8, controls and exact '.' or '..' identities; reserved characters and internal spaces use escaped always-member GET through the audited direct-route hook. Passive response name/id values never choose another target.
- The declared nine Body fields are projected without Python mutable Resource or computed Connection location. Accepted malformed/nonJSON body tolerance preserves a seeded Resource with nil Wire; physical failures and parsed invalid objects return an error with actual accepted receipt. Missing/duplicate failures use Go errors.Is sentinels. Invalid UTF-8 response bytes also use the Go tolerant opaque-body path; Python HTTP text decoding is not reproduced.
- Microversion selection uses the shared finite Nova reader once. Explicit empty is a Go versionless override; source/session discovery caches and warning facilities are not replicated. Foreign-owner access still depends on deployment policy and its HTTP failures remain visible.

### `gophercloud:compute/v2/keypairs.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.List`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithListQuery`

- The unchanged native List binding serializes typed UserID then applies raw wire overrides/extensions using its SDK-owned concrete builder. GET retains selected ResourceBase, live token, source header and both2.55 version headers, with no body. Raw limit is a request extension rather than a native ListOpts field.
  Tests: `api/compute_keypair_records_test.go:TestComputeKeypairListNativeBindingKeepsTypedSinglePageABI`.
- Native nested keypair extraction preserves six string fields and ignores vendor fields; omitted/null type stays empty and null nested keypair yields its zero native model. SinglePageBase does not follow body or HTTP next links. The tested empty204 without JSON Content-Type is an empty native page;200/300 retain normal extraction and201 remains outside native pager codes.
  Tests: `api/compute_keypair_records_test.go:TestComputeKeypairListNativeBindingKeepsTypedSinglePageABI`.
- Unexpected201 and403 preserve native actual/expected200,204,300 statuses, GET method, exact owner-bearing URL, response body/header. Invalid known string types preserve json.UnmarshalTypeError and stop before yielding partially decoded rows.
  Tests: `api/compute_keypair_records_test.go:TestComputeKeypairListNativeBindingKeepsTypedSinglePageABI`.

Python/Go differences:

- The optional upstream ListOptsBuilder is replaced by concrete ListOption/WithListOptions plus a library-owned serializer, and the native pager/extractor are exposed as iter.Seq2. Native ListOpts, six-string KeyPair alias,200/204/300 pager acceptance and SinglePageBase behavior remain intact. Additive ListRecords implements the separately reviewed Python-style list instead of changing this native operation.

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

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
for keypair in conn.compute.keypairs(type="ssh"):
    print(keypair.to_dict())

found = conn.compute.find_keypair("workstation", ignore_missing=True)
if found is not None:
    print(found.to_dict())
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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
)

func main() {
    name := flag.String("find", "", "keypair name to find instead of listing")
    owner := flag.String("owner-user-id", "", "optional owner")
    kind := flag.String("type", "", "optional local list filter, for example ssh")
    maximum := flag.Int("max-items", 0, "raw list row cap; zero is unlimited")
    strict := flag.Bool("strict", false, "return an error when find has no match")
    version := flag.String("microversion", "", "optional operation version; explicit empty is versionless")
    flag.Parse()
    versionSet := false
    flag.Visit(func(f *flag.Flag) {
        if f.Name == "microversion" { versionSet = true }
    })
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *name, *owner, *kind, *maximum, *strict, *version, versionSet); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, name, owner, kind string, maximum int, strict bool, version string, versionSet bool) error {
    conn, err := sdk.Connect(ctx)
    if err != nil { return err }
    service, err := conn.Compute(ctx)
    if err != nil { return err }
    api := service.API.KeyPairs
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    printRecord := func(record *keypairs.KeypairRecord) error {
        return encoder.Encode(map[string]any{
            "resource": record.Resource, "wire": record.Wire,
            "envelope": string(record.Envelope),
            "status_code": record.StatusCode, "header": record.Header,
        })
    }
    if name != "" {
        options := []keypairs.KeypairFindOption{
            keypairs.WithKeypairFindUserID(owner),
            keypairs.WithKeypairFindIgnoreMissing(!strict),
        }
        if versionSet { options = append(options, keypairs.WithKeypairFindMicroversion(version)) }
        record, err := api.FindKeypair(ctx, name, options...)
        if record != nil {
            if printErr := printRecord(record); printErr != nil { return errors.Join(err, printErr) }
        }
        if err != nil { return fmt.Errorf("find keypair: %w", err) }
        if record == nil { fmt.Println("keypair not found") }
        return nil
    }
    options := []keypairs.KeypairListOption{
        keypairs.WithKeypairListUserID(owner),
        keypairs.WithKeypairListMaxItems(maximum),
    }
    if kind != "" { options = append(options, keypairs.WithKeypairListFilter("type", kind)) }
    if versionSet { options = append(options, keypairs.WithKeypairListMicroversion(version)) }
    for record, err := range api.ListRecords(ctx, options...) {
        if err != nil { return fmt.Errorf("list keypairs: %w", err) }
        if err := printRecord(record); err != nil { return err }
    }
    return nil
}
```

## References

- [Nova policy](https://docs.openstack.org/nova/latest/configuration/policy.html)
- [Compute](https://docs.openstack.org/api-ref/compute/#list-keypairs)
- [Console Selection](console-selection.md)
- [Proxy keypairs/find_keypair](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L910-L978)
- [Keypair](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/keypair.py)
- [Keypairs Console](keypairs-console.md)
- [Keypair leaf API](v2/keypairs/README.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
