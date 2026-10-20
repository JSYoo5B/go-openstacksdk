# Trusts Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/create_trust`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithRecordCreateField`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithRecordCreateOpts`

- POST /OS-TRUST/trusts sends declared raw attrs under trust, preserves omitted/null/false/zero/empty values and untyped roles/user/expiry/counts, normalizes is_impersonation/impersonation, ignores unknown attrs and imposes no native required-field validation.
  Tests: `identity/v3/trusts/python_records_test.go:TestPythonTrustCreateRawAttributesAliasesAndDescriptorViews`.
- Actual response fields overlay owned create seed; canonical view applies nonnull bool truthiness while raw Wire/Envelope preserve unknown values and exact integers; all channels are independent.
  Tests: `identity/v3/trusts/python_records_test.go:TestPythonTrustCreateRawAttributesAliasesAndDescriptorViews`, `identity/v3/trusts/python_records_test.go:TestPythonTrustGetResponseTranslationAndBoolDefaults`.
- Owned options isolate maps and retained callback pointers; invalid known JSON/aliases/options and callback errors combined with cancellation or Source mutation stop before HTTP.
  Tests: `identity/v3/trusts/python_records_test.go:TestPythonTrustOptionsOwnershipPreflightAndJoinedCallbacks`.
- Source success statuses and physical accepted/rejected Read/Close/cancel/Source failures preserve actual proof; native retries keep the fixed POST body and invalid callbacks cannot replay changed requests.
  Tests: `identity/v3/trusts/python_records_test.go:TestPythonTrustDeleteMissingAndSourceStatusPolicies`, `identity/v3/trusts/python_records_test.go:TestPythonTrustPhysicalAcceptedAndRejectedFailures`, `identity/v3/trusts/python_records_test.go:TestPythonTrustNativeRetryAndFixedRequestOwnership`.

Python/Go differences:

- The explicit Go profile accepts literal string IDs, owns copied concrete options and returns fresh TrustRecord views; mutable Python Resource/Munch inputs, Connection location, same-instance state and arbitrary session/cache/subclass hooks remain global Resource parity work.
- The Go profile fixes literal UTF-8 IDs as escaped URL segments and guards request ownership/source identity. Accepted and rejected physical Read/Close/cancel/Source failures retain actual HTTP evidence and original causes.
- Conflicting canonical/wire impersonation aliases in an unordered bulk map or response are rejected, rather than depending on Python insertion order; sequential field options normalize aliases and the final value wins.

### `python:identity/v3/delete_trust`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.API.DeleteRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithRecordDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithRecordDeleteHeaders`

- DELETE literal Trust member accepts actual 200..399 and defaults to ignoring clean404; strict missing preserves original native status/body while opaque successful responses need no JSON model.
  Tests: `identity/v3/trusts/python_records_test.go:TestPythonTrustDeleteMissingAndSourceStatusPolicies`.
- Physical accepted/rejected Read/Close/cancel/Source causes remain observable and a mixed404 is never ignored; identical clean native retry rejection remains eligible for default missing suppression.
  Tests: `identity/v3/trusts/python_records_test.go:TestPythonTrustPhysicalAcceptedAndRejectedFailures`, `identity/v3/trusts/python_records_test.go:TestPythonTrustNativeRetryAndFixedRequestOwnership`.
- Owned typed missing/header options and invalid nil options retain common Source/auth/literal target preflight before HTTP.
  Tests: `identity/v3/trusts/python_records_test.go:TestPythonTrustOptionsOwnershipPreflightAndJoinedCallbacks`.

Python/Go differences:

- The explicit Go profile accepts literal string IDs, owns copied concrete options and returns fresh TrustRecord views; mutable Python Resource/Munch inputs, Connection location, same-instance state and arbitrary session/cache/subclass hooks remain global Resource parity work.
- The Go profile fixes literal UTF-8 IDs as escaped URL segments and guards request ownership/source identity. Accepted and rejected physical Read/Close/cancel/Source failures retain actual HTTP evidence and original causes.
- Conflicting canonical/wire impersonation aliases in an unordered bulk map or response are rejected, rather than depending on Python insertion order; sequential field options normalize aliases and the final value wins.

### `python:identity/v3/find_trust`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithRecordFindIgnoreMissing`

- GET-first lookup falls back only after clean 400/403/404, without a name server-query hint, scans all pages for exact ID/name, retains duplicates and late errors, and defaults missing to nil,nil with explicit strict NotFound.
  Tests: `identity/v3/trusts/python_records_test.go:TestPythonTrustFindGetFirstCleanFallbackFullSearchAndMissing`.
- Mixed rejected Read/Close/cancel/Source causes prevent fallback; a native retry hook returning the identical clean HTTP rejection still permits fallback.
  Tests: `identity/v3/trusts/python_records_test.go:TestPythonTrustPhysicalAcceptedAndRejectedFailures`, `identity/v3/trusts/python_records_test.go:TestPythonTrustNativeRetryAndFixedRequestOwnership`.
- Direct accepted malformed JSON retains literal seed while Source descriptor views and invalid/nil Find options retain the owned read contract.
  Tests: `identity/v3/trusts/python_records_test.go:TestPythonTrustFindGetFirstCleanFallbackFullSearchAndMissing`, `identity/v3/trusts/python_records_test.go:TestPythonTrustGetResponseTranslationAndBoolDefaults`, `identity/v3/trusts/python_records_test.go:TestPythonTrustOptionsOwnershipPreflightAndJoinedCallbacks`.

Python/Go differences:

- The explicit Go profile accepts literal string IDs, owns copied concrete options and returns fresh TrustRecord views; mutable Python Resource/Munch inputs, Connection location, same-instance state and arbitrary session/cache/subclass hooks remain global Resource parity work.
- The Go profile fixes literal UTF-8 IDs as escaped URL segments and guards request ownership/source identity. Accepted and rejected physical Read/Close/cancel/Source failures retain actual HTTP evidence and original causes.
- Conflicting canonical/wire impersonation aliases in an unordered bulk map or response are rejected, rather than depending on Python insertion order; sequential field options normalize aliases and the final value wins.

### `python:identity/v3/get_trust`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.API.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithRecordReadHeaders`

- GET literal /OS-TRUST/trusts/{id} preserves singular/root translation, seeded ID, malformed/empty JSON tolerance, accepted nonobject proof, alias normalization and null/default/bool descriptor behavior.
  Tests: `identity/v3/trusts/python_records_test.go:TestPythonTrustGetResponseTranslationAndBoolDefaults`.
- Get uses actual 200..399 policy and retains original native unexpected status evidence and physical accepted/rejected Read/Close/cancel/Source causes.
  Tests: `identity/v3/trusts/python_records_test.go:TestPythonTrustDeleteMissingAndSourceStatusPolicies`, `identity/v3/trusts/python_records_test.go:TestPythonTrustPhysicalAcceptedAndRejectedFailures`.
- Invalid read options/auth headers/IDs fail before HTTP; escaped literal member IDs and configured future version remain unchanged without an invented maximum ceiling.
  Tests: `identity/v3/trusts/python_records_test.go:TestPythonTrustOptionsOwnershipPreflightAndJoinedCallbacks`, `identity/v3/trusts/python_records_test.go:TestPythonTrustEscapedMemberAndConfiguredFutureVersion`.

Python/Go differences:

- The explicit Go profile accepts literal string IDs, owns copied concrete options and returns fresh TrustRecord views; mutable Python Resource/Munch inputs, Connection location, same-instance state and arbitrary session/cache/subclass hooks remain global Resource parity work.
- The Go profile fixes literal UTF-8 IDs as escaped URL segments and guards request ownership/source identity. Accepted and rejected physical Read/Close/cancel/Source failures retain actual HTTP evidence and original causes.
- Conflicting canonical/wire impersonation aliases in an unordered bulk map or response are rejected, rather than depending on Python insertion order; sequential field options normalize aliases and the final value wins.

### `python:identity/v3/trusts`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithRecordListTrustorUserID`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithRecordListTrusteeUserID`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts.WithRecordListHeaders`

- Lazy list classifies trustor/trustee fields as Requests-encoded server queries, canonical declared Body attrs as local descriptor filters, and unknown/wire-only aliases as ignored; max_items counts raw rows before filters.
  Tests: `identity/v3/trusts/python_records_test.go:TestPythonTrustListQueryClassificationLocalFiltersAndRawCap`.
- Each iteration owns copied headers/query/limit and new Source capture; Source advertised next/singletons and positive-limit short-page marker fallback use original wire markers even after consumer mutations, and later-page errors and consumer stop are respected.
  Tests: `identity/v3/trusts/python_records_test.go:TestPythonTrustListPagingLazyOwnershipAndLaterErrors`.
- Lists enforce the actual trust JSON envelope and success policy while preserving accepted/rejected physical Read/Close/cancel/Source failures, with invalid options and negative typed limits rejected before requests.
  Tests: `identity/v3/trusts/python_records_test.go:TestPythonTrustDeleteMissingAndSourceStatusPolicies`, `identity/v3/trusts/python_records_test.go:TestPythonTrustPhysicalAcceptedAndRejectedFailures`, `identity/v3/trusts/python_records_test.go:TestPythonTrustOptionsOwnershipPreflightAndJoinedCallbacks`.

Python/Go differences:

- The explicit Go profile accepts literal string IDs, owns copied concrete options and returns fresh TrustRecord views; mutable Python Resource/Munch inputs, Connection location, same-instance state and arbitrary session/cache/subclass hooks remain global Resource parity work.
- The Go profile fixes literal UTF-8 IDs as escaped URL segments and guards request ownership/source identity. Accepted and rejected physical Read/Close/cancel/Source failures retain actual HTTP evidence and original causes.
- Conflicting canonical/wire impersonation aliases in an unordered bulk map or response are rejected, rather than depending on Python insertion order; sequential field options normalize aliases and the final value wins.
- Local JSON filters distinguish bool from number and reject nonobject nested shape; typed controls reject negative limits/caps. Dictionary next links are an explicit Go pagination extension; arbitrary inherited dynamic/JMESPath controls remain outside this profile.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
created = conn.identity.create_trust(
    trustor_user_id=trustor_id,
    trustee_user_id=trustee_id,
    project_id=project_id,
    is_impersonation=False,
    roles=[{"name": "member"}],
)
for trust in conn.identity.trusts(trustor_user_id=trustor_id):
    print(trust.id, trust.is_impersonation)
found = conn.identity.find_trust(created.id, ignore_missing=False)
conn.identity.delete_trust(created.id, ignore_missing=False)
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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts"
)

func printTrust(value *trusts.TrustRecord) {
    if value == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    trustor := flag.String("trustor", "", "trustor user ID")
    trustee := flag.String("trustee", "", "trustee user ID for creation")
    project := flag.String("project", "", "project ID for creation")
    role := flag.String("role", "member", "role name for creation")
    id := flag.String("id", "", "Trust ID to get and find")
    create := flag.Bool("create", false, "create a Trust")
    remove := flag.String("delete", "", "Trust ID to delete")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Trusts
    options := []trusts.RecordListOption{trusts.WithRecordListMaxItems(100)}
    if *trustor != "" {
        options = append(options, trusts.WithRecordListTrustorUserID(*trustor))
    }
    for value, err := range api.ListRecords(ctx, options...) {
        if err != nil { log.Fatal(err) }
        printTrust(value)
    }
    if *id != "" {
        value, err := api.GetRecord(ctx, *id)
        if err != nil { log.Fatal(err) }
        printTrust(value)
        found, err := api.FindRecord(ctx, *id, trusts.WithRecordFindIgnoreMissing(false))
        if err != nil { log.Fatal(err) }
        printTrust(found)
    }
    if *create {
        value, err := api.CreateRecord(ctx,
            trusts.WithRecordCreateField("trustor_user_id", *trustor),
            trusts.WithRecordCreateField("trustee_user_id", *trustee),
            trusts.WithRecordCreateField("project_id", *project),
            trusts.WithRecordCreateField("is_impersonation", false),
            trusts.WithRecordCreateField("roles", []map[string]string{{"name": *role}}))
        if err != nil { log.Fatal(err) }
        printTrust(value)
    }
    if *remove != "" {
        if err := api.DeleteRecord(ctx, *remove,
            trusts.WithRecordDeleteIgnoreMissing(false)); err != nil { log.Fatal(err) }
    }
}
```

## References

- [Install](../../docs/install.md)
- [Python Records Test](trusts/python_records_test.go)
