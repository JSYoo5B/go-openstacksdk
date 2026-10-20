# Credentials Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/create_credential`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.API.CreateRecord`

- POST /credentials with credential envelope; only known id/name/blob/project_id/type/user_id Body attrs, no native required-field or string coercion; omitted fields absent and explicit null/false/empty preserved.
  Tests: `identity/v3/credentials/python_records_test.go:TestPythonCredentialCreateRawBodyAndOwnedViews`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP`.
- Response singular unpack/root object, known response overlay on seeded create attrs; original raw blob/types/numbers/unknown Wire with independent Resource/Envelope/Header.
  Tests: `identity/v3/credentials/python_records_test.go:TestPythonCredentialCreateRawBodyAndOwnedViews`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP`.
- Native generated ABI remains unchanged; capture owned identity client facts and callback policy before physical requests. OperationError preserves original cause and accepted handling errors keep real ResponseError proof.
  Tests: `identity/v3/credentials/python_records_test.go:TestPythonCredentialCreateRawBodyAndOwnedViews`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP`.

Python/Go differences:

- Correct generic Credential /credentials binding; candidate inventory also proposes ec2credentials, but /users/{user_id}/credentials/OS-EC2 is a distinct native API and is not used to close this Python operation.
- Bounded Go profile uses explicit escaped UTF-8 string IDs and owned record fields, not Resource/Munch/same-instance reuse/cache/session/subclass/Connection arbitrary hook behavior; location null. Full mutable Resource surface and Source update_credential are separately pending.
- Library source/route/header guards and valid JSON policy, concrete nonnegative typed limit/caps; arbitrary Python adapter/microversion/kwargs/JMESPath are outside this callable Go profile.

### `python:identity/v3/credentials`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.API.ListRecords`

- GET /credentials. Source type/user_id are wire query encoded by shared Requests JSON-domain expansion (null omitted, repeated arrays, object keys and True spelling); other known Body fields local JSON filters; unknown kwargs ignored.
  Tests: `identity/v3/credentials/python_records_test.go:TestPythonCredentialListServerQueryAndLocalJSONFilters`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialListPagingLazinessAndWireMarker`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP`.
- Lazy reusable list, owned opts and row views, raw-row cap before local filters, max_items hint replaces absent/zero limit, marker and paginated false, singleton page, Source array/top-level/HTTP next, positive initial limit short-page fallback, empty page end; consumer mutation cannot change original-wire marker.
  Tests: `identity/v3/credentials/python_records_test.go:TestPythonCredentialListServerQueryAndLocalJSONFilters`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialListPagingLazinessAndWireMarker`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP`.
- Native generated ABI remains unchanged; capture owned identity client facts and callback policy before physical requests. OperationError preserves original cause and accepted handling errors keep real ResponseError proof.
  Tests: `identity/v3/credentials/python_records_test.go:TestPythonCredentialListServerQueryAndLocalJSONFilters`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialListPagingLazinessAndWireMarker`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP`.

Python/Go differences:

- Correct generic Credential /credentials binding; candidate inventory also proposes ec2credentials, but /users/{user_id}/credentials/OS-EC2 is a distinct native API and is not used to close this Python operation.
- Bounded Go profile uses explicit escaped UTF-8 string IDs and owned record fields, not Resource/Munch/same-instance reuse/cache/session/subclass/Connection arbitrary hook behavior; location null. Full mutable Resource surface and Source update_credential are separately pending.
- Library source/route/header guards and valid JSON policy, concrete nonnegative typed limit/caps; arbitrary Python adapter/microversion/kwargs/JMESPath are outside this callable Go profile.
- Dictionary links.next is an intentional Go compatibility extension because pinned Source generic paginator ignores that shape. Fixed REST rejects cross-origin/path/new-page-key/filter changes, conflicting continuations and marker cycles; does not claim Source arbitrary version-prefix/cross-path behavior.
- JSON filters distinguish bool and number; malformed nested nonobjects do not match instead of Source attribute-error behavior.

### `python:identity/v3/get_credential`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.API.GetRecord`

- Literal GET /credentials/{escapedID}; no parent/user lookup/query. Singular credential unpack or root object; blank/malformed syntax accepted and seeded literal ID retained; valid nonobject JSON is accepted-response handling failure with real status/body/header.
  Tests: `identity/v3/credentials/python_records_test.go:TestPythonCredentialGetTranslationAndStatuses`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialAcceptedResponseProofAndEscapedMember`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialRejectedResponseFailuresStopDeleteAndFind`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- Declared id/name/blob/project_id/type/user_id plus null location, missing fields null, untyped raw blob/numbers preserved and owned Wire keeps unknown data.
  Tests: `identity/v3/credentials/python_records_test.go:TestPythonCredentialGetTranslationAndStatuses`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialAcceptedResponseProofAndEscapedMember`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialRejectedResponseFailuresStopDeleteAndFind`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- Native generated ABI remains unchanged; capture owned identity client facts and callback policy before physical requests. OperationError preserves original cause and accepted handling errors keep real ResponseError proof.
  Tests: `identity/v3/credentials/python_records_test.go:TestPythonCredentialGetTranslationAndStatuses`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialAcceptedResponseProofAndEscapedMember`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialRejectedResponseFailuresStopDeleteAndFind`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- GET400/403/404 and Delete404 observe rejected Read/Close/context/source faults through shared REST; only clean rejection may establish ignoreMissing/fallback. Identical native retry stop-hook return remains a clean status; mixed retry errors preserve causes and stop after one physical request.
  Tests: `identity/v3/credentials/python_records_test.go:TestPythonCredentialGetTranslationAndStatuses`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialAcceptedResponseProofAndEscapedMember`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialRejectedResponseFailuresStopDeleteAndFind`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook`.

Python/Go differences:

- Correct generic Credential /credentials binding; candidate inventory also proposes ec2credentials, but /users/{user_id}/credentials/OS-EC2 is a distinct native API and is not used to close this Python operation.
- Bounded Go profile uses explicit escaped UTF-8 string IDs and owned record fields, not Resource/Munch/same-instance reuse/cache/session/subclass/Connection arbitrary hook behavior; location null. Full mutable Resource surface and Source update_credential are separately pending.
- Library source/route/header guards and valid JSON policy, concrete nonnegative typed limit/caps; arbitrary Python adapter/microversion/kwargs/JMESPath are outside this callable Go profile.

### `python:identity/v3/delete_credential`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.API.DeleteRecord`

- Literal DELETE /credentials/{escapedID}; actual200..399 accepted with opaque body, IgnoreMissing true by default suppresses only clean404, explicit false preserves native404. Retry callback/read/close/transport/cancel/source failures mixed with404 remain visible.
  Tests: `identity/v3/credentials/python_records_test.go:TestPythonCredentialDeleteMissingPolicyAndMixedFailure`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialRejectedResponseFailuresStopDeleteAndFind`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- Native generated ABI remains unchanged; capture owned identity client facts and callback policy before physical requests. OperationError preserves original cause and accepted handling errors keep real ResponseError proof.
  Tests: `identity/v3/credentials/python_records_test.go:TestPythonCredentialDeleteMissingPolicyAndMixedFailure`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialRejectedResponseFailuresStopDeleteAndFind`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- GET400/403/404 and Delete404 observe rejected Read/Close/context/source faults through shared REST; only clean rejection may establish ignoreMissing/fallback. Identical native retry stop-hook return remains a clean status; mixed retry errors preserve causes and stop after one physical request.
  Tests: `identity/v3/credentials/python_records_test.go:TestPythonCredentialDeleteMissingPolicyAndMixedFailure`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialRejectedResponseFailuresStopDeleteAndFind`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook`.

Python/Go differences:

- Correct generic Credential /credentials binding; candidate inventory also proposes ec2credentials, but /users/{user_id}/credentials/OS-EC2 is a distinct native API and is not used to close this Python operation.
- Bounded Go profile uses explicit escaped UTF-8 string IDs and owned record fields, not Resource/Munch/same-instance reuse/cache/session/subclass/Connection arbitrary hook behavior; location null. Full mutable Resource surface and Source update_credential are separately pending.
- Library source/route/header guards and valid JSON policy, concrete nonnegative typed limit/caps; arbitrary Python adapter/microversion/kwargs/JMESPath are outside this callable Go profile.

### `python:identity/v3/find_credential`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials.API.FindRecord`

- GET first without user_id or arbitrary query. Only Source formal IgnoreMissing exposed via concrete owned option; clean400/403/404 fallback to complete exact ID/name list with no name wire hint.
  Tests: `identity/v3/credentials/python_records_test.go:TestPythonCredentialFindDefaultFallbackAndAllPages`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialRejectedResponseFailuresStopDeleteAndFind`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- Default absent nil,nil, strict absent NotFoundError, duplicate AmbiguousError, direct seeded malformed-JSON success has no fallback; accepted nonobject errors do not fallback; later-page error overrides unique first-page match. Shared FindIdentity reused.
  Tests: `identity/v3/credentials/python_records_test.go:TestPythonCredentialFindDefaultFallbackAndAllPages`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialRejectedResponseFailuresStopDeleteAndFind`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- Native generated ABI remains unchanged; capture owned identity client facts and callback policy before physical requests. OperationError preserves original cause and accepted handling errors keep real ResponseError proof.
  Tests: `identity/v3/credentials/python_records_test.go:TestPythonCredentialFindDefaultFallbackAndAllPages`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialRejectedResponseFailuresStopDeleteAndFind`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook`.
- GET400/403/404 and Delete404 observe rejected Read/Close/context/source faults through shared REST; only clean rejection may establish ignoreMissing/fallback. Identical native retry stop-hook return remains a clean status; mixed retry errors preserve causes and stop after one physical request.
  Tests: `identity/v3/credentials/python_records_test.go:TestPythonCredentialFindDefaultFallbackAndAllPages`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialOptionsOwnInputsAndRejectBeforeHTTP`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialRejectedResponseFailuresStopDeleteAndFind`, `identity/v3/credentials/python_records_test.go:TestPythonCredentialCleanRejectionsRemainEligibleAfterNativeRetryHook`.

Python/Go differences:

- Correct generic Credential /credentials binding; candidate inventory also proposes ec2credentials, but /users/{user_id}/credentials/OS-EC2 is a distinct native API and is not used to close this Python operation.
- Bounded Go profile uses explicit escaped UTF-8 string IDs and owned record fields, not Resource/Munch/same-instance reuse/cache/session/subclass/Connection arbitrary hook behavior; location null. Full mutable Resource surface and Source update_credential are separately pending.
- Library source/route/header guards and valid JSON policy, concrete nonnegative typed limit/caps; arbitrary Python adapter/microversion/kwargs/JMESPath are outside this callable Go profile.
- Dictionary links.next is an intentional Go compatibility extension because pinned Source generic paginator ignores that shape. Fixed REST rejects cross-origin/path/new-page-key/filter changes, conflicting continuations and marker cycles; does not claim Source arbitrary version-prefix/cross-path behavior.
- JSON filters distinguish bool and number; malformed nested nonobjects do not match instead of Source attribute-error behavior.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
created = conn.identity.create_credential(
    user_id=user_id, type="ec2", project_id=project_id,
    blob='{"access":"ACCESS","secret":"SECRET"}')
credentials = conn.identity.credentials(user_id=user_id, type="ec2")
found = conn.identity.find_credential(credential_id)
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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials"
)

func printRecord(v *credentials.CredentialRecord) {
    if v == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(v.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    user := flag.String("user", "", "credential owner user ID")
    project := flag.String("project", "", "credential project ID")
    kind := flag.String("type", "ec2", "credential blob type")
    id := flag.String("id", "", "credential ID to GET and find")
    blob := flag.String("blob", "{}", "credential blob string")
    create := flag.Bool("create", false, "create a generic Credential")
    remove := flag.String("delete", "", "credential ID to delete")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Credentials
    for v, err := range api.ListRecords(ctx,
        credentials.WithRecordListType(*kind),
        credentials.WithRecordListUserID(*user),
        credentials.WithRecordListMaxItems(100)) {
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    if *id != "" {
        v, err := api.GetRecord(ctx, *id)
        if err != nil { log.Fatal(err) }
        printRecord(v)
        found, err := api.FindRecord(ctx, *id)
        if err != nil { log.Fatal(err) }
        printRecord(found)
    }
    if *create {
        v, err := api.CreateRecord(ctx,
            credentials.WithRecordCreateField("user_id", *user),
            credentials.WithRecordCreateField("project_id", *project),
            credentials.WithRecordCreateField("type", *kind),
            credentials.WithRecordCreateField("blob", *blob))
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    if *remove != "" {
        if err := api.DeleteRecord(ctx, *remove,
            credentials.WithRecordDeleteIgnoreMissing(false)); err != nil { log.Fatal(err) }
    }
}
```

## References

- [Install](../../docs/install.md)
- [Python Records Test](credentials/python_records_test.go)
- [Native Contracts Test](credentials/native_contracts_test.go)
