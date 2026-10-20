# Accounts

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:object_store/v1/get_account_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.API.GetMetadata`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithGetMetadataOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithGetMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithGetMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithGetMetadataNewest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithoutGetMetadataNewest`

- One bodyless HEAD targets the exact query-free account Endpoint including reverse prefix/escaped path/trailing slash; ResourceBase does not change it. Only actual204 is accepted, with no Ref/Name/discovery/detail/list fallback.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreRoutesAndEvidence`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsWireAndPresence`, `connection_objectstorage_account_metadata_test.go:TestConnectionAccountMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_account_metadata_test.go:TestSwiftAccountMetadataPreservesNativeAndRegistry`.
- Newest nil omits the owned header, false/true are explicit HEAD values, Without clears, and whole opts replacement resets previous values. Ordinary header singular last overwrite handles earlier ASCII case variants, while an incoming plural or final map rejects aliases.
  Tests: `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsReplacementAndNewest`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsWireAndPresence`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsPreflightAndSnapshots`.
- Factories and each callback independently snapshot initialized Headers/Newest, preserve callback/custom context causes and permit concurrent option reuse without mutating source maps.
  Tests: `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsFactorySnapshots`, `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsRetainedCallbacksAndCauses`, `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsParallelReuseAndSourceHeaders`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsPreflightAndSnapshots`.
- Get projects nonnil lowercased metadata suffix values including Temp URL keys, optional signed exactint64 counters and literal Timestamp atomically. Original non-ASCII token bytes, malformed suffixes, aliases, multi/zero values and malformed counters fail without partial Metadata while preserving raw unknown headers.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreAtomicHeaderProjection`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsHeaderDecodeAndOwnership`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsWireAndPresence`, `connection_objectstorage_account_metadata_test.go:TestConnectionAccountMetadataSharesClientAndOwnsEvidence`.
- Nil context/API/client/provider, wrong service Type, invalid endpoint/header/reserved header/nil callback or cancellation are rejected before HTTP and initial callbacks, retaining typed option/context causes.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreCompletePreflight`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsPreflightAndSnapshots`.
- Actual204 Body/Header/Status is owned independently; read/Close/context/custom cause/source/projection failure retains Get result with nil Metadata and whole ResponseError. Close once, no accepted-body replay and original strconv cause are preserved.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreAcceptedFailures`, `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreAtomicHeaderProjection`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsHeaderDecodeAndOwnership`, `connection_objectstorage_account_metadata_test.go:TestConnectionAccountMetadataSharesClientAndOwnsEvidence`.
- Original source/provider/Endpoint/Type/Microversion and ordinary headers are captured before callbacks; live original auth is used. Fixed method/URL/body/actual204 guards retain native prebody retry/reauth fields and configured same-scope redirects, while advanced RetryFunc.MoreHeaders and existing native Get/Update ABI remain native policy.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreSourceAndTargetBoundaries`, `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreNativePolicyAndCompatibility`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsSourceAndNativePolicy`, `connection_objectstorage_account_metadata_test.go:TestConnectionAccountMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_account_metadata_test.go:TestSwiftAccountMetadataPreservesNativeAndRegistry`.

Python/Go differences:

- Python returns a mutable Account Resource using header descriptors and Adapter/session state; Go returns independently owned raw evidence and atomic nullable MetadataInfo.
- Go Values deliberately includes Temp-URL-Key and Temp-URL-Key-2, which Python known descriptors consume before custom metadata extraction. Literal timestamp/signed nullable counters and stricter atomic invalid-header rejection differ.
- The named Python call has no account input or query options. Go performs a bodyless HEAD at the exact configured account Endpoint and offers additive concrete request headers/Newest; ResourceBase does not select another account. Python Resource.head may use session discovery/state, while Go keeps its configured client and live provider authentication.
- Python header translation accepts statuses below400. Go accepts actual204 only and exposes raw Body/Header/Status plus optional exact signed int64 counters and a literal timestamp, rejecting malformed canonical headers atomically; errors retain typed native/transport or accepted-response causes.
- Python Account known header descriptors and _last_headers are a mutable view; Go returns owned Header rather than descriptor objects. Temp URL keys remain available both in raw Header and intentionally in Values. Other raw headers remain accessible without descriptor coercion. Cache/session translation and setter refresh are separate API scope, and server authorization/metadata limits remain server policy.

### `python:object_store/v1/set_account_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.API.SetMetadata`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithMetadataOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.API.SetMetadataAndRefresh`

- Set sends one bodyless POST with exactly one X-Account-Meta prefix and strict actual204, returning opaque acknowledgement without automatic HEAD, submitted-state projection, account create/delete or clear-all. Nil/empty maps still POST and an empty value requests deletion of only that key.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreRoutesAndEvidence`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsWireAndPresence`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsMutationEvidence`, `connection_objectstorage_account_metadata_test.go:TestConnectionAccountMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_account_metadata_test.go:TestSwiftAccountMetadataPreservesNativeAndRegistry`.
- Caller metadata is snapshotted before callbacks. Suffix-only nonempty ASCII HTTP token keys reject full prefixes and case aliases; literal UTF8 values preserve spaces/percent/Unicode/HTAB/empty without str coercion, trim or URL encoding, and invalid inputs stop before HTTP.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreCompletePreflight`, `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsFactorySnapshots`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsPreflightAndSnapshots`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsWireAndPresence`.
- Concrete mutation full opts replaces all ordinary headers, singular/plural helpers merge last values while rejecting incoming aliases; independently owned callback/factory maps and parallel reuse preserve source storage and option/context causes.
  Tests: `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsFactorySnapshots`, `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsReplacementAndNewest`, `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsRetainedCallbacksAndCauses`, `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsParallelReuseAndSourceHeaders`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsPreflightAndSnapshots`.
- Actual204 Set acknowledgement retains opaque bytes and raw repeated/malformed metadata-looking headers without projection; accepted read/Close/custom context/source failure returns result plus whole independent ResponseError, closes once and never replays an accepted response.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreAcceptedFailures`, `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreSourceAndTargetBoundaries`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsMutationEvidence`.
- Only original actual204 creates Set result: unexpected200/201/202/native/transport failures return nil with native typed evidence. Fixed captured source/liveauth and request/body/status ownership guards preserve original errors, native reauth fields, explicit advanced retry headers and legacy Update201/202/204 acceptance.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreNativePolicyAndCompatibility`, `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreSourceAndTargetBoundaries`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsSourceAndNativePolicy`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsMutationEvidence`, `connection_objectstorage_account_metadata_test.go:TestConnectionAccountMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_account_metadata_test.go:TestSwiftAccountMetadataPreservesNativeAndRegistry`.
- Bodyless POST then default HEAD, including empty metadata, on selected account route; actual final observation is independent from submitted input.
  Tests: `objectstorage/v1/accounts/metadata_refresh_test.go:TestSwiftPythonAccountMetadataSetterRefreshesObservedStateAndSourceValues`, `objectstorage/v1/accounts/metadata_refresh_test.go:TestSwiftPythonAccountMetadataRefreshFailuresKeepAcknowledgementWithoutReplay`, `objectstorage/v1/accounts/metadata_refresh_test.go:TestSwiftPythonAccountMetadataInputsCallbacksAndConcurrentReuseAreOwned`, `objectstorage/v1/accounts/metadata_refresh_test.go:TestSwiftPythonAccountMetadataPreflightAndSourceStatusPolicy`, `objectstorage/v1/accounts/metadata_refresh_test.go:TestSwiftPythonAccountMetadataCapturesSourceBeforeCustomJSONMarshaler`.
- JSON-domain Python string conversion, exact full-prefix convention, Source status200..399 and existing owned concrete options/source/cause-safe REST.
  Tests: `objectstorage/v1/accounts/metadata_refresh_test.go:TestSwiftPythonAccountMetadataSetterRefreshesObservedStateAndSourceValues`, `objectstorage/v1/accounts/metadata_refresh_test.go:TestSwiftPythonAccountMetadataRefreshFailuresKeepAcknowledgementWithoutReplay`, `objectstorage/v1/accounts/metadata_refresh_test.go:TestSwiftPythonAccountMetadataInputsCallbacksAndConcurrentReuseAreOwned`, `objectstorage/v1/accounts/metadata_refresh_test.go:TestSwiftPythonAccountMetadataPreflightAndSourceStatusPolicy`, `objectstorage/v1/accounts/metadata_refresh_test.go:TestSwiftPythonAccountMetadataCapturesSourceBeforeCustomJSONMarshaler`.

Python/Go differences:

- Go returns independently owned physical response/phase proof and accepts context/explicit route names; Python None/same mutable Resource return is part of separate Resource scope.
- Metadata values cover JSON-domain Python string/truthiness conversion; arbitrary __str__, nonJSON objects, float repr and Unicode runtime identity are bounded; colliding incoming Go map wire aliases fail before HTTP.
- Go metadata token/field-value/single-header validation and native per-request transport policy remain explicit boundaries.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: R1/C1/S1 mutable Resource descriptor/cache and Adapter/session behavior remain independently tracked.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Server ACL/quota/encoding/version/SLO/symlink and durable replication are deployment contracts; read/merge/write offers no CAS/rollback/whole-operation replay.

### `python:object_store/v1/delete_account_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.API.DeleteMetadata`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithMetadataOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.API.DeleteMetadataKeys`

- DeleteMetadata is one bodyless POST carrying empty X-Account-Meta values, never HTTP DELETE or X-Remove syntax. Nil/empty keys still POST and do not mean clear-all; only actual204 creates raw acknowledgement and no refresh/fallback is performed.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreRoutesAndEvidence`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsWireAndPresence`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsMutationEvidence`, `connection_objectstorage_account_metadata_test.go:TestConnectionAccountMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_account_metadata_test.go:TestSwiftAccountMetadataPreservesNativeAndRegistry`.
- A key slice is snapshotted before callbacks; suffix-only nonempty ASCII token names reject full prefixes and duplicate/case-colliding keys without coercion or normalization, and all key/header/context/source preflight fails before HTTP.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreCompletePreflight`, `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsFactorySnapshots`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsPreflightAndSnapshots`.
- Delete shares concrete mutation opts with whole replacement and header merge, independent factory/retained-callback storage and reused concurrent options, while preserving callback/source/context causes and reserved-header policy.
  Tests: `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsReplacementAndNewest`, `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsRetainedCallbacksAndCauses`, `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsParallelReuseAndSourceHeaders`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsPreflightAndSnapshots`.
- Accepted204 Delete bytes/header/status and joined read/Close/context/custom causes are retained in independent acknowledgement and ResponseError with one Close and no resend. Mutation headers remain passive and do not prove refreshed metadata state.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreAcceptedFailures`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsMutationEvidence`, `connection_objectstorage_account_metadata_test.go:TestConnectionAccountMetadataSharesClientAndOwnsEvidence`.
- Delete strictly preserves missing404 and other unexpected statuses as native errors with nil result; no IgnoreMissing or account deletion is inferred. Captured source/live original authentication and native prebody/reauth/redirect policy remain guarded for target/body/status with advanced native MoreHeaders retained.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreNativePolicyAndCompatibility`, `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreSourceAndTargetBoundaries`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsMutationEvidence`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsSourceAndNativePolicy`, `connection_objectstorage_account_metadata_test.go:TestConnectionAccountMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_account_metadata_test.go:TestSwiftAccountMetadataPreservesNativeAndRegistry`.
- One bodyless POST including nil/empty keys; duplicate keys collapse, exact full prefix is accepted, final200..399 preserves opaque actual acknowledgement.
  Tests: `objectstorage/v1/accounts/metadata_refresh_test.go:TestSwiftPythonAccountMetadataEmptyAndPrefixedDeleteFollowSourcePOST`, `objectstorage/v1/accounts/metadata_refresh_test.go:TestSwiftPythonAccountMetadataRefreshFailuresKeepAcknowledgementWithoutReplay`, `objectstorage/v1/accounts/metadata_refresh_test.go:TestSwiftPythonAccountMetadataInputsCallbacksAndConcurrentReuseAreOwned`, `objectstorage/v1/accounts/metadata_refresh_test.go:TestSwiftPythonAccountMetadataPreflightAndSourceStatusPolicy`, `objectstorage/v1/accounts/metadata_refresh_test.go:TestSwiftPythonAccountMetadataCapturesSourceBeforeCustomJSONMarshaler`.

Python/Go differences:

- Go returns independently owned physical response/phase proof and accepts context/explicit route names; Python None/same mutable Resource return is part of separate Resource scope.
- Metadata values cover JSON-domain Python string/truthiness conversion; arbitrary __str__, nonJSON objects, float repr and Unicode runtime identity are bounded; colliding incoming Go map wire aliases fail before HTTP.
- Go metadata token/field-value/single-header validation and native per-request transport policy remain explicit boundaries.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: R1/C1/S1 mutable Resource descriptor/cache and Adapter/session behavior remain independently tracked.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Server ACL/quota/encoding/version/SLO/symlink and durable replication are deployment contracts; read/merge/write offers no CAS/rollback/whole-operation replay.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
account = conn.object_store.get_account_metadata()
print(account.metadata)
conn.object_store.set_account_metadata(book="Moby Dick", note="literal %2F 雪")
conn.object_store.delete_account_metadata(["note"])
```

```go
package main

import (
    "context"
    "fmt"

    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts"
)

func changeAccountMetadata(ctx context.Context, api *accounts.API) error {
    getOptions := []accounts.GetMetadataOption{
        accounts.WithGetMetadataOpts(accounts.GetMetadataOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        accounts.WithGetMetadataHeaders(map[string]string{"X-Trace-Label": "metadata"}),
        accounts.WithGetMetadataHeader("X-Request-Label", "account-metadata"),
        accounts.WithGetMetadataNewest(false),
    }
    before, err := api.GetMetadata(ctx, getOptions...)
    if err != nil { return err }
    if before.Metadata.BytesUsed != nil { fmt.Println(*before.Metadata.BytesUsed) }

    mutationOptions := []accounts.MetadataOption{
        accounts.WithMetadataOpts(accounts.MetadataOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        accounts.WithMetadataHeaders(map[string]string{"X-Trace-Label": "metadata"}),
        accounts.WithMetadataHeader("X-Request-Label", "metadata-mutation"),
    }
    acknowledgement, err := api.SetMetadata(ctx, map[string]string{
        "book": "Moby Dick", "note": "literal %2F 雪",
    }, mutationOptions...)
    if acknowledgement != nil {
        fmt.Println(acknowledgement.StatusCode)
    }
    if err != nil { return err }

    refreshOptions := append([]accounts.GetMetadataOption(nil), getOptions...)
    refreshOptions = append(refreshOptions, accounts.WithoutGetMetadataNewest())
    refreshed, err := api.GetMetadata(ctx, refreshOptions...)
    if err != nil { return err }
    fmt.Println(refreshed.Metadata.Values["book"])

    removed, err := api.DeleteMetadata(ctx, []string{"note"}, mutationOptions...)
    if removed != nil { fmt.Println(removed.StatusCode) }
    return err
}

func main() {}
```

## References

- [pinned Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L119)
- [ Base](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L71)
- [Storage Account Services](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-account-services.inc#L120)
- [Base](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/controllers/base.py#L1953)
- [Metadata Contracts Test](metadata_contracts_test.go)
