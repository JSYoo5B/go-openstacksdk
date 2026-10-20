# Application Credentials Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/access_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.API.ListAccessRuleRecords`

- Library-owned literal user ID route and captured identity Source; no parent GET for resource.ID scopes; concrete owned options and guarded physical requests; native generated ABI unchanged.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialListFiltersRawCapAndPaging`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialListContinuationsAndLazyErrors`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Meaningful OperationError preserves actual accepted 200..399 response handling errors and original rejected HTTP / transport / read / close / cancellation causes.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialListFiltersRawCapAndPaging`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialListContinuationsAndLazyErrors`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Declared Body JSON filters run locally; only limit/marker are wire query. Unknown filters discarded. Null versus empty preserved, credential unrestricted uses Source JSON truthiness. Owned raw Wire keeps unknown fields and precise numbers.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialListFiltersRawCapAndPaging`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialListContinuationsAndLazyErrors`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Lazy reusable iterator, raw observed-row cap before local filters, max_items limit hint replaces absent/zero initial limit, paginated=false single page, singleton object page, Source array/top-level/HTTP next and positive-initial-limit short-page marker fallback; empty page stops.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialListFiltersRawCapAndPaging`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialListContinuationsAndLazyErrors`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.

Python/Go differences:

- Bounded Go callable profile uses explicit string IDs and owned Record DTOs, not arbitrary Python Resource/Munch identity, same-instance mutation, cache/session/subclass/Connection hooks. location is null; full Resource/session surface remains separate.
- Valid UTF-8 / JSON fields and escaped member path segments; guarded fixed service routes and protected headers; arbitrary Source adapter/microversion/route choices are not accepted.
- JSON filter comparisons distinguish bool from number; malformed nested nonobjects fail matching rather than raising Python attribute errors.
- Dictionary links.next is a Go compatibility extension: pinned Source paginator does not recognize that shape. Fixed continuation preserves initial filters, rejects cross-origin/path changes, new arbitrary page keys, conflict/cycles; Source arbitrary version-prefix and cross-path continuation are broader.
- All Python list control kwargs, JMESPath and negative dynamic values are not provided; concrete Limit/Marker/MaxItems/Paginated/Headers/declared JSON Filters cover this binding.

### `python:identity/v3/application_credentials`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.API.ListCredentialRecords`

- Library-owned literal user ID route and captured identity Source; no parent GET for resource.ID scopes; concrete owned options and guarded physical requests; native generated ABI unchanged.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialListFiltersRawCapAndPaging`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialListContinuationsAndLazyErrors`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Meaningful OperationError preserves actual accepted 200..399 response handling errors and original rejected HTTP / transport / read / close / cancellation causes.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialListFiltersRawCapAndPaging`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialListContinuationsAndLazyErrors`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Declared Body JSON filters run locally; only limit/marker are wire query. Unknown filters discarded. Null versus empty preserved, credential unrestricted uses Source JSON truthiness. Owned raw Wire keeps unknown fields and precise numbers.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialListFiltersRawCapAndPaging`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialListContinuationsAndLazyErrors`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Lazy reusable iterator, raw observed-row cap before local filters, max_items limit hint replaces absent/zero initial limit, paginated=false single page, singleton object page, Source array/top-level/HTTP next and positive-initial-limit short-page marker fallback; empty page stops.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialListFiltersRawCapAndPaging`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialListContinuationsAndLazyErrors`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.

Python/Go differences:

- Bounded Go callable profile uses explicit string IDs and owned Record DTOs, not arbitrary Python Resource/Munch identity, same-instance mutation, cache/session/subclass/Connection hooks. location is null; full Resource/session surface remains separate.
- Valid UTF-8 / JSON fields and escaped member path segments; guarded fixed service routes and protected headers; arbitrary Source adapter/microversion/route choices are not accepted.
- JSON filter comparisons distinguish bool from number; malformed nested nonobjects fail matching rather than raising Python attribute errors.
- Dictionary links.next is a Go compatibility extension: pinned Source paginator does not recognize that shape. Fixed continuation preserves initial filters, rejects cross-origin/path changes, new arbitrary page keys, conflict/cycles; Source arbitrary version-prefix and cross-path continuation are broader.
- All Python list control kwargs, JMESPath and negative dynamic values are not provided; concrete Limit/Marker/MaxItems/Paginated/Headers/declared JSON Filters cover this binding.

### `python:identity/v3/create_application_credential`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.API.CreateCredentialRecord`

- Library-owned literal user ID route and captured identity Source; no parent GET for resource.ID scopes; concrete owned options and guarded physical requests; native generated ABI unchanged.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialCreateAndOwnedViews`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Meaningful OperationError preserves actual accepted 200..399 response handling errors and original rejected HTTP / transport / read / close / cancellation causes.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialCreateAndOwnedViews`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- POST singular application_credential envelope with explicit name; supplied known Body attrs only, unknown attrs discarded before serialization, fixed name/user_id collision rejected. Omitted unrestricted not injected. Explicit null/false/empty retained and untyped expires_at/roles/access_rules preserved.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialCreateAndOwnedViews`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Accepted blank/malformed JSON tolerates Source ValueError branch and retains seeded attrs; valid nonobject errors with actual response evidence. Response declared fields overlay seed; fixed user_id view and independent Wire/Envelope/Header.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialCreateAndOwnedViews`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.

Python/Go differences:

- Bounded Go callable profile uses explicit string IDs and owned Record DTOs, not arbitrary Python Resource/Munch identity, same-instance mutation, cache/session/subclass/Connection hooks. location is null; full Resource/session surface remains separate.
- Valid UTF-8 / JSON fields and escaped member path segments; guarded fixed service routes and protected headers; arbitrary Source adapter/microversion/route choices are not accepted.

### `python:identity/v3/delete_access_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.API.DeleteAccessRuleRecord`

- Library-owned literal user ID route and captured identity Source; no parent GET for resource.ID scopes; concrete owned options and guarded physical requests; native generated ABI unchanged.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialDeleteDefaultAndMixedFailures`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Meaningful OperationError preserves actual accepted 200..399 response handling errors and original rejected HTTP / transport / read / close / cancellation causes.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialDeleteDefaultAndMixedFailures`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- DELETE literal member, actual200..399 accepted, opaque body never JSON decoded. IgnoreMissing defaults true; only clean404 suppressed, strict false preserves original native404 body/header; joined retry/read/close/cancel/source failures visible.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialDeleteDefaultAndMixedFailures`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Rejected 400/403/404 body Read/Close, cancellation cause and source changes retain native HTTP evidence. Only clean permitted rejections, including an unchanged native rejection from a retry callback, may be ignored or enter Find fallback; mixed failures are terminal and never replay an accepted response. Common identity lookup preserves the original HTTP error when cancellation occurs during GET or enrichment.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialRejectedResponseFailuresStopDeleteAndFind`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook`, `resource/find_identity_test.go:TestCollectionFindIdentityCancellationRetainsPhysicalLookupAndEnrichmentErrors`.

Python/Go differences:

- Bounded Go callable profile uses explicit string IDs and owned Record DTOs, not arbitrary Python Resource/Munch identity, same-instance mutation, cache/session/subclass/Connection hooks. location is null; full Resource/session surface remains separate.
- Valid UTF-8 / JSON fields and escaped member path segments; guarded fixed service routes and protected headers; arbitrary Source adapter/microversion/route choices are not accepted.

### `python:identity/v3/delete_application_credential`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.API.DeleteCredentialRecord`

- Library-owned literal user ID route and captured identity Source; no parent GET for resource.ID scopes; concrete owned options and guarded physical requests; native generated ABI unchanged.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialDeleteDefaultAndMixedFailures`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Meaningful OperationError preserves actual accepted 200..399 response handling errors and original rejected HTTP / transport / read / close / cancellation causes.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialDeleteDefaultAndMixedFailures`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- DELETE literal member, actual200..399 accepted, opaque body never JSON decoded. IgnoreMissing defaults true; only clean404 suppressed, strict false preserves original native404 body/header; joined retry/read/close/cancel/source failures visible.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialDeleteDefaultAndMixedFailures`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Rejected 400/403/404 body Read/Close, cancellation cause and source changes retain native HTTP evidence. Only clean permitted rejections, including an unchanged native rejection from a retry callback, may be ignored or enter Find fallback; mixed failures are terminal and never replay an accepted response. Common identity lookup preserves the original HTTP error when cancellation occurs during GET or enrichment.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialRejectedResponseFailuresStopDeleteAndFind`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook`, `resource/find_identity_test.go:TestCollectionFindIdentityCancellationRetainsPhysicalLookupAndEnrichmentErrors`.

Python/Go differences:

- Bounded Go callable profile uses explicit string IDs and owned Record DTOs, not arbitrary Python Resource/Munch identity, same-instance mutation, cache/session/subclass/Connection hooks. location is null; full Resource/session surface remains separate.
- Valid UTF-8 / JSON fields and escaped member path segments; guarded fixed service routes and protected headers; arbitrary Source adapter/microversion/route choices are not accepted.

### `python:identity/v3/find_application_credential`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.API.FindCredentialRecord`

- Library-owned literal user ID route and captured identity Source; no parent GET for resource.ID scopes; concrete owned options and guarded physical requests; native generated ABI unchanged.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialFindSourceDefaultsAndQueries`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Meaningful OperationError preserves actual accepted 200..399 response handling errors and original rejected HTTP / transport / read / close / cancellation causes.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialFindSourceDefaultsAndQueries`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- GET first on escaped literal identity with fixed user_id query and supplied string query. Direct response overlays id and known Body query seeds. Clean400/403/404 allow list fallback; transport/source/cancellation/body failures never fallback or get suppressed.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialFindSourceDefaultsAndQueries`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- No automatic name query hint for this class. Complete exact ID-or-name list uses shared FindIdentity; default missing nil,nil, strict IgnoreMissing false NotFoundError, duplicates AmbiguousError, later-page HTTP failure preserved. Known Body query becomes local string filter; only limit/marker stay wire.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialFindSourceDefaultsAndQueries`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Rejected 400/403/404 body Read/Close, cancellation cause and source changes retain native HTTP evidence. Only clean permitted rejections, including an unchanged native rejection from a retry callback, may be ignored or enter Find fallback; mixed failures are terminal and never replay an accepted response. Common identity lookup preserves the original HTTP error when cancellation occurs during GET or enrichment.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialRejectedResponseFailuresStopDeleteAndFind`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook`, `resource/find_identity_test.go:TestCollectionFindIdentityCancellationRetainsPhysicalLookupAndEnrichmentErrors`.

Python/Go differences:

- Bounded Go callable profile uses explicit string IDs and owned Record DTOs, not arbitrary Python Resource/Munch identity, same-instance mutation, cache/session/subclass/Connection hooks. location is null; full Resource/session surface remains separate.
- Valid UTF-8 / JSON fields and escaped member path segments; guarded fixed service routes and protected headers; arbitrary Source adapter/microversion/route choices are not accepted.
- Shared IdentityFindOption accepts string-valued service query; arbitrary Python JSON object/list query/filter and pagination/transport kwargs are not accepted. Explicit fixed id/user_id query replacements are preflight errors. Unsupported shared Details/AllProjects/ExtraSpecs controls remain unsupported.
- Dictionary links.next Go extension and fixed-origin/path/query continuation differ from broader Source generic paginator; see guide.

### `python:identity/v3/get_access_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.API.GetAccessRuleRecord`

- Library-owned literal user ID route and captured identity Source; no parent GET for resource.ID scopes; concrete owned options and guarded physical requests; native generated ABI unchanged.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialGetTranslationAndStatuses`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Meaningful OperationError preserves actual accepted 200..399 response handling errors and original rejected HTTP / transport / read / close / cancellation causes.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialGetTranslationAndStatuses`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- GET literal member without user_id query; singular-key unpack or root object; Source ValueError blank/malformed JSON tolerance with seeded literal id, valid nonobject error. Declared view null defaults, fixed user_id URI view, independent raw Wire/Envelope/Header; no strict native timestamp coercion.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialGetTranslationAndStatuses`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Rejected 400/403/404 body Read/Close, cancellation cause and source changes retain native HTTP evidence. Only clean permitted rejections, including an unchanged native rejection from a retry callback, may be ignored or enter Find fallback; mixed failures are terminal and never replay an accepted response. Common identity lookup preserves the original HTTP error when cancellation occurs during GET or enrichment.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialRejectedResponseFailuresStopDeleteAndFind`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook`, `resource/find_identity_test.go:TestCollectionFindIdentityCancellationRetainsPhysicalLookupAndEnrichmentErrors`.

Python/Go differences:

- Bounded Go callable profile uses explicit string IDs and owned Record DTOs, not arbitrary Python Resource/Munch identity, same-instance mutation, cache/session/subclass/Connection hooks. location is null; full Resource/session surface remains separate.
- Valid UTF-8 / JSON fields and escaped member path segments; guarded fixed service routes and protected headers; arbitrary Source adapter/microversion/route choices are not accepted.

### `python:identity/v3/get_application_credential`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials.API.GetCredentialRecord`

- Library-owned literal user ID route and captured identity Source; no parent GET for resource.ID scopes; concrete owned options and guarded physical requests; native generated ABI unchanged.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialGetTranslationAndStatuses`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Meaningful OperationError preserves actual accepted 200..399 response handling errors and original rejected HTTP / transport / read / close / cancellation causes.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialGetTranslationAndStatuses`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- GET literal member without user_id query; singular-key unpack or root object; Source ValueError blank/malformed JSON tolerance with seeded literal id, valid nonobject error. Declared view null defaults, fixed user_id URI view, independent raw Wire/Envelope/Header; no strict native timestamp coercion.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialGetTranslationAndStatuses`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialOptionsOwnershipAndPreflight`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialLiteralScopesAndResponseProof`.
- Rejected 400/403/404 body Read/Close, cancellation cause and source changes retain native HTTP evidence. Only clean permitted rejections, including an unchanged native rejection from a retry callback, may be ignored or enter Find fallback; mixed failures are terminal and never replay an accepted response. Common identity lookup preserves the original HTTP error when cancellation occurs during GET or enrichment.
  Tests: `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialRejectedResponseFailuresStopDeleteAndFind`, `identity/v3/applicationcredentials/python_records_test.go:TestPythonApplicationCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook`, `resource/find_identity_test.go:TestCollectionFindIdentityCancellationRetainsPhysicalLookupAndEnrichmentErrors`.

Python/Go differences:

- Bounded Go callable profile uses explicit string IDs and owned Record DTOs, not arbitrary Python Resource/Munch identity, same-instance mutation, cache/session/subclass/Connection hooks. location is null; full Resource/session surface remains separate.
- Valid UTF-8 / JSON fields and escaped member path segments; guarded fixed service routes and protected headers; arbitrary Source adapter/microversion/route choices are not accepted.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
credentials = conn.identity.application_credentials(user_id, name="ci", max_items=100)
created = conn.identity.create_application_credential(
    user_id, "ci", description="", unrestricted=False, roles=[{"name": "member"}])
found = conn.identity.find_application_credential(user_id, "ci")
rule = conn.identity.get_access_rule(user_id, rule_id)
# conn.identity.delete_application_credential(user_id, created.id)
# conn.identity.delete_access_rule(user_id, rule.id, ignore_missing=False)
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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printRecord(v *applicationcredentials.ApplicationCredentialRecord) {
    if v == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(v.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    user := flag.String("user", "", "literal Keystone user ID")
    name := flag.String("name", "ci", "exact credential name or ID")
    credential := flag.String("credential", "", "credential ID to GET")
    rule := flag.String("rule", "", "access rule ID to GET")
    create := flag.Bool("create", false, "create an application credential")
    deleteCredential := flag.String("delete-credential", "", "credential ID to delete")
    deleteRule := flag.String("delete-rule", "", "access rule ID to delete")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    api := service.ApplicationCredentials
    scope, err := api.InUser(ctx, resource.ID(*user))
    if err != nil { log.Fatal(err) }
    rules, err := api.AccessRules(ctx, resource.ID(*user))
    if err != nil { log.Fatal(err) }
    for v, err := range scope.ListRecords(ctx,
        applicationcredentials.WithRecordListFilter("name", *name),
        applicationcredentials.WithRecordListMaxItems(100)) {
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    found, err := scope.FindRecord(ctx, *name)
    if err != nil { log.Fatal(err) }
    printRecord(found)
    if *credential != "" {
        v, err := scope.GetRecord(ctx, *credential)
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    for v, err := range rules.ListRecords(ctx) {
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    if *rule != "" {
        v, err := rules.GetRecord(ctx, *rule)
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    if *create {
        v, err := scope.CreateRecord(ctx, *name,
            applicationcredentials.WithRecordCreateField("description", ""),
            applicationcredentials.WithRecordCreateField("unrestricted", false),
            applicationcredentials.WithRecordCreateField("roles", []map[string]string{{"name": "member"}}))
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    if *deleteCredential != "" {
        if err := scope.DeleteRecord(ctx, *deleteCredential); err != nil { log.Fatal(err) }
    }
    if *deleteRule != "" {
        if err := rules.DeleteRecord(ctx, *deleteRule,
            applicationcredentials.WithRecordDeleteIgnoreMissing(false)); err != nil { log.Fatal(err) }
    }
}
```

## References

- [Install](../../docs/install.md)
- [Python Records Test](applicationcredentials/python_records_test.go)
