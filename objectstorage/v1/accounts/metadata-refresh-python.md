# Metadata Refresh Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:object_store/v1/set_account_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.API.SetMetadata`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithMetadataOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithMetadataHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.API.SetMetadataAndRefresh`

- Set sends one bodyless POST with exactly one X-Account-Meta prefix and strict actual204, returning opaque acknowledgement without automatic HEAD, submitted-state projection, account create/delete or clear-all. Nil/empty maps still POST and an empty value requests deletion of only that key.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreRoutesAndEvidence`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsWireAndPresence`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsMutationEvidence`, `connectiontest/connection_objectstorage_account_metadata_test.go:TestConnectionAccountMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_account_metadata_test.go:TestSwiftAccountMetadataPreservesNativeAndRegistry`.
- Caller metadata is snapshotted before callbacks. Suffix-only nonempty ASCII HTTP token keys reject full prefixes and case aliases; literal UTF8 values preserve spaces/percent/Unicode/HTAB/empty without str coercion, trim or URL encoding, and invalid inputs stop before HTTP.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreCompletePreflight`, `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsFactorySnapshots`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsPreflightAndSnapshots`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsWireAndPresence`.
- Concrete mutation full opts replaces all ordinary headers, singular/plural helpers merge last values while rejecting incoming aliases; independently owned callback/factory maps and parallel reuse preserve source storage and option/context causes.
  Tests: `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsFactorySnapshots`, `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsReplacementAndNewest`, `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsRetainedCallbacksAndCauses`, `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsParallelReuseAndSourceHeaders`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsPreflightAndSnapshots`.
- Actual204 Set acknowledgement retains opaque bytes and raw repeated/malformed metadata-looking headers without projection; accepted read/Close/custom context/source failure returns result plus whole independent ResponseError, closes once and never replays an accepted response.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreAcceptedFailures`, `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreSourceAndTargetBoundaries`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsMutationEvidence`.
- Only original actual204 creates Set result: unexpected200/201/202/native/transport failures return nil with native typed evidence. Fixed captured source/liveauth and request/body/status ownership guards preserve original errors, native reauth fields, explicit advanced retry headers and legacy Update201/202/204 acceptance.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreNativePolicyAndCompatibility`, `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreSourceAndTargetBoundaries`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsSourceAndNativePolicy`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsMutationEvidence`, `connectiontest/connection_objectstorage_account_metadata_test.go:TestConnectionAccountMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_account_metadata_test.go:TestSwiftAccountMetadataPreservesNativeAndRegistry`.
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
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreRoutesAndEvidence`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsWireAndPresence`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsMutationEvidence`, `connectiontest/connection_objectstorage_account_metadata_test.go:TestConnectionAccountMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_account_metadata_test.go:TestSwiftAccountMetadataPreservesNativeAndRegistry`.
- A key slice is snapshotted before callbacks; suffix-only nonempty ASCII token names reject full prefixes and duplicate/case-colliding keys without coercion or normalization, and all key/header/context/source preflight fails before HTTP.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreCompletePreflight`, `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsFactorySnapshots`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsPreflightAndSnapshots`.
- Delete shares concrete mutation opts with whole replacement and header merge, independent factory/retained-callback storage and reused concurrent options, while preserving callback/source/context causes and reserved-header policy.
  Tests: `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsReplacementAndNewest`, `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsRetainedCallbacksAndCauses`, `objectstorage/v1/accounts/metadata_options_test.go:TestAccountMetadataOptionsParallelReuseAndSourceHeaders`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsPreflightAndSnapshots`.
- Accepted204 Delete bytes/header/status and joined read/Close/context/custom causes are retained in independent acknowledgement and ResponseError with one Close and no resend. Mutation headers remain passive and do not prove refreshed metadata state.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreAcceptedFailures`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsMutationEvidence`, `connectiontest/connection_objectstorage_account_metadata_test.go:TestConnectionAccountMetadataSharesClientAndOwnsEvidence`.
- Delete strictly preserves missing404 and other unexpected statuses as native errors with nil result; no IgnoreMissing or account deletion is inferred. Captured source/live original authentication and native prebody/reauth/redirect policy remain guarded for target/body/status with advanced native MoreHeaders retained.
  Tests: `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreNativePolicyAndCompatibility`, `objectstorage/v1/accounts/metadata_core_test.go:TestAccountMetadataCoreSourceAndTargetBoundaries`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsMutationEvidence`, `objectstorage/v1/accounts/metadata_contracts_test.go:TestAccountMetadataContractsSourceAndNativePolicy`, `connectiontest/connection_objectstorage_account_metadata_test.go:TestConnectionAccountMetadataSharesClientAndOwnsEvidence`, `internal/cmd/sdkgen/swift_account_metadata_test.go:TestSwiftAccountMetadataPreservesNativeAndRegistry`.
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

```go
api := accounts.New(swiftClient)
result, err := api.SetMetadataAndRefresh(ctx, map[string]any{
    "owner": "ops",
    "enabled": false,
    "X-Account-Meta-External": "value",
})
if err != nil {
    return err
}
observed := result.Refreshed.Metadata.Values["owner"]
_ = observed
```

## References

- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L125-L147)
- [BaseResource.set_metadata](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L73-L92)
