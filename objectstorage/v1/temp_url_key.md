# Temp Url Key

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:object_store/v1/set_account_temp_url_key`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.API.SetTempURLKey`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithSetTempURLKeyOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithSetTempURLKeyHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithSetTempURLKeyHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts.WithSetTempURLKeySecondary`

- One bodyless exact-account-Endpoint POST204 sends only selected primary or secondary X-Account-Meta-Temp-URL-Key header, transmits empty key literally, and performs no implicit HEAD/refresh.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsWireAndSelection`, `objectstorage/v1/accounts/temp_url_key_core_test.go:TestAccountTempURLKeyCoreWireAndStatus`, `connectiontest/connection_objectstorage_temp_url_key_test.go:TestConnectionTempURLKeySharesClientAndSnapshotsEachWorkflow`.
- Concrete full/header/header-map/secondary options own factory and callback snapshots, replace full configuration and apply canonical header overwrite; default false selects primary.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsPresenceAndOptions`, `objectstorage/v1/accounts/temp_url_key_options_test.go:TestAccountTempURLKeyOptionsSnapshotsAndReplacement`, `objectstorage/v1/accounts/temp_url_key_options_test.go:TestAccountTempURLKeyOptionsParallelAndInvalidHeaders`.
- Source/context/literal key preflight preserves invalid-option and cancellation causes before HTTP; setter never sends a body or asks consumer builders.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsPreflightAndSource`, `objectstorage/v1/accounts/temp_url_key_core_test.go:TestAccountTempURLKeyCorePreflightAndCallbackCause`.
- Actual accepted204 opaque body/header/status survives Read/Close/customcontext failure in MetadataResponse+ResponseError; close once and no accepted-body replay.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsPhaseEvidence`, `objectstorage/v1/accounts/temp_url_key_core_test.go:TestAccountTempURLKeyCoreAcceptedFailuresAndNativeHooks`.
- Native prebody retry/reauth/bodyless ownership and configured advanced headers remain bounded; unexpected native successes reject actual status without changing native Update/Get/CreateTempURL ABI.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsNativePolicy`, `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsWireAndSelection`, `objectstorage/v1/accounts/temp_url_key_core_test.go:TestAccountTempURLKeyCoreAcceptedFailuresAndNativeHooks`, `internal/cmd/sdkgen/swift_temp_url_key_test.go:TestSwiftTempURLKeyPreservesNativeAndRegistry`.
- SetTempURLKey sends one POST with X-Account-Meta-Temp-URL-Key or X-Container-Meta-Temp-URL-Key (the -2 header when secondary), the metadata POST Python set_temp_url_key sends.
  Tests: `objectstorage/v1/python_rejudge_test.go:TestPythonRejudgeSwiftSetTempURLKeyRequests`.

Python/Go differences:

- Python proxy returns None after setter wrapper implicit POST→HEAD refresh; Go returns single strict204 acknowledgement and requires explicit GetMetadata for refreshed state.
- Python Resource|string cache/descriptor/str(any) behavior and generic success policy are not replicated.
- Python follows the POST with a HEAD that refreshes a Resource it then discards; Go sends only the POST.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

### `python:object_store/v1/set_container_temp_url_key`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.API.SetTempURLKey`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithSetTempURLKeyOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithSetTempURLKeyHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithSetTempURLKeyHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers.WithSetTempURLKeySecondary`

- One bodyless literal container POST204 uses captured effective ResourceBase with once-escaped Unicode/percent/space/colon/question/hash name and correct X-Container-Meta-Temp-URL-Key slot; no account prefix/discovery/implicit HEAD.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsWireAndSelection`, `objectstorage/v1/containers/temp_url_key_core_test.go:TestContainerTempURLKeyCoreWireAndStatus`, `connectiontest/connection_objectstorage_temp_url_key_test.go:TestConnectionTempURLKeySharesClientAndSnapshotsEachWorkflow`.
- Concrete full/header/header-map/secondary options own factory and callback snapshots, replace full configuration and apply canonical header overwrite; default false selects primary.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsPresenceAndOptions`, `objectstorage/v1/containers/temp_url_key_options_test.go:TestContainerTempURLKeyOptionsSnapshotsAndReplacement`, `objectstorage/v1/containers/temp_url_key_options_test.go:TestContainerTempURLKeyOptionsParallelAndInvalidHeaders`.
- Source/context/literal key preflight preserves invalid-option and cancellation causes before HTTP; setter never sends a body or asks consumer builders.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsPreflightAndSource`, `objectstorage/v1/containers/temp_url_key_core_test.go:TestContainerTempURLKeyCorePreflightAndCallbackCause`.
- Actual accepted204 opaque body/header/status survives Read/Close/customcontext failure in MetadataResponse+ResponseError; close once and no accepted-body replay.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsPhaseEvidence`, `objectstorage/v1/containers/temp_url_key_core_test.go:TestContainerTempURLKeyCoreAcceptedFailuresAndNativeHooks`.
- Native prebody retry/reauth/bodyless ownership and configured advanced headers remain bounded; unexpected native successes reject actual status without changing native Update/Get/CreateTempURL ABI.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsNativePolicy`, `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsWireAndSelection`, `objectstorage/v1/containers/temp_url_key_core_test.go:TestContainerTempURLKeyCoreAcceptedFailuresAndNativeHooks`, `internal/cmd/sdkgen/swift_temp_url_key_test.go:TestSwiftTempURLKeyPreservesNativeAndRegistry`.
- SetTempURLKey sends one POST with X-Account-Meta-Temp-URL-Key or X-Container-Meta-Temp-URL-Key (the -2 header when secondary), the metadata POST Python set_temp_url_key sends.
  Tests: `objectstorage/v1/python_rejudge_test.go:TestPythonRejudgeSwiftSetTempURLKeyRequests`.

Python/Go differences:

- Python proxy returns None after setter wrapper implicit POST→HEAD refresh; Go returns single strict204 acknowledgement and requires explicit GetMetadata for refreshed state.
- Python Resource|string cache/descriptor/str(any) behavior and generic success policy are not replicated.
- Python follows the POST with a HEAD that refreshes a Resource it then discards; Go sends only the POST.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

### `python:object_store/v1/get_temp_url_key`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.Service.GetTempURLKey`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGetTempURLKeyOpts`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGetTempURLKeyContainer`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGetTempURLKeyHeader`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGetTempURLKeyHeaders`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithGetTempURLKeyNewest`
- `github.com/JSYoo5B/go-openstacksdk/objectstorage/v1.WithoutGetTempURLKeyNewest`

- Fresh container HEAD204 prefers nonempty secondary then primary; only successful absent/empty keys cause account HEAD204 with same preference. Account-only omits container HTTP and any HTTP failure stops fallback.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsWireAndSelection`, `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsPhaseEvidence`, `objectstorage/v1/temp_url_key_core_test.go:TestTempURLKeyCoreSelectionAndProof`, `objectstorage/v1/temp_url_key_core_test.go:TestTempURLKeyCoreStopsOnErrors`, `connectiontest/connection_objectstorage_temp_url_key_test.go:TestConnectionTempURLKeySharesClientAndSnapshotsEachWorkflow`.
- No usable key gives nil Key/zero selection flags, while raw full metadata distinguishes absent versus explicit empty; whitespace and Unicode remain literal selected UTF8 bytes.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsPresenceAndOptions`, `objectstorage/v1/temp_url_key_core_test.go:TestTempURLKeyCoreSelectionAndProof`.
- Full/header/header-map/container/newest/without-newest options own maps and pointers, replace final configuration, preserve explicitfalse/true on both phases and emptyContainer account-only reset.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsPresenceAndOptions`, `objectstorage/v1/temp_url_key_options_test.go:TestTempURLKeyOptionsSnapshotsAndReplacement`, `objectstorage/v1/temp_url_key_options_test.go:TestTempURLKeyOptionsParallelReuseAndPreflight`.
- Joint account/container custom/remove prefixes and auth/framing/newest are protected; original ASCII tokens/aliases/UTF8 values and source/context/name validation precede HTTP.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsPreflightAndSource`, `objectstorage/v1/temp_url_key_core_test.go:TestTempURLKeyCorePreflightAndAcceptedSourceGuard`, `objectstorage/v1/temp_url_key_options_test.go:TestTempURLKeyOptionsParallelReuseAndPreflight`.
- Original Service/API/client/provider/Endpoint/rawResourceBase/type/version identities stay fixed across callbacks and accepted responses; late drift clears current Metadata, retains rawproof and prevents account fallback. Captured ordinary headers and original live auth persist.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsPreflightAndSource`, `objectstorage/v1/temp_url_key_core_test.go:TestTempURLKeyCorePreflightAndAcceptedSourceGuard`, `objectstorage/v1/temp_url_key_core_test.go:TestTempURLKeyCoreCapturedHeadersAndNativeSigner`, `connectiontest/connection_objectstorage_temp_url_key_test.go:TestConnectionTempURLKeySharesClientAndSnapshotsEachWorkflow`.
- Full atomic metadata projection rejects malformed unrelated metadata/counters/duplicate recognized fields even with usable key; signed int64/literal dates/unknown opaque body evidence are preserved.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsPhaseEvidence`, `objectstorage/v1/temp_url_key_core_test.go:TestTempURLKeyCoreStopsOnErrors`.
- Result allocation follows first accepted raw proof only; later account native/transport or accepted handling failure retains earlier container proof and any currentaccepted proof, with Key/selection flags zero and underlying causes retained.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsPhaseEvidence`, `objectstorage/v1/temp_url_key_core_test.go:TestTempURLKeyCoreStopsOnErrors`.
- Native configured retries/body ownership/actual-success-code/fixedtarget/auth policies remain bounded, and native CreateTempURL keeps its explicit noHTTP and primary-only signer ABI.
  Tests: `objectstorage/v1/temp_url_key_contracts_test.go:TestTempURLKeyContractsNativePolicy`, `objectstorage/v1/temp_url_key_core_test.go:TestTempURLKeyCoreCapturedHeadersAndNativeSigner`, `internal/cmd/sdkgen/swift_temp_url_key_test.go:TestSwiftTempURLKeyPreservesNativeAndRegistry`.
- GetTempURLKey HEADs the container (when given) and then the account, preferring Temp-URL-Key-2 over Temp-URL-Key at each level, the order Python get_temp_url_key uses.
  Tests: `objectstorage/v1/python_rejudge_test.go:TestPythonRejudgeSwiftTempURLKeySelection`.

Python/Go differences:

- Go concrete owned result instead of mutable Resource or bytes|string cache/coercion; strictHEAD204 and full atomic metadata validation deliberately exceed Python generic handling.
- Python final explicit falsey empty primary can remain empty; Go exposes nil Key with raw empty-header evidence.
- Go returns typed result values with copied raw response evidence instead of a mutable Python Resource, so there is no Resource descriptor coercion, dirty-attribute tracking or seeded-object merge.
- Go has no client-side GET response cache or skip_cache control; every call sends its request.
- Python raises openstack.exceptions classes; Go returns wrapped Gophercloud or SDK errors, with resource.NotFoundError or the native 404 cause where documented.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
conn.object_store.set_account_temp_url_key(account_secret)
conn.object_store.set_container_temp_url_key(
    "books", container_secret, secondary=True
)
key = conn.object_store.get_temp_url_key("books")
```

```go
package main

import (
    "context"
    "fmt"

    objectstorage "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
)

func tempURLKeys(ctx context.Context, service *objectstorage.Service, name, accountSecret, containerSecret string) error {
    accountOptions := []accounts.SetTempURLKeyOption{
        accounts.WithSetTempURLKeyOpts(accounts.SetTempURLKeyOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        accounts.WithSetTempURLKeyHeaders(map[string]string{"X-Trace-Label": "account-key"}),
        accounts.WithSetTempURLKeyHeader("X-Request-Label", "account"),
        accounts.WithSetTempURLKeySecondary(false),
    }
    accountAck, err := service.Accounts.SetTempURLKey(ctx, accountSecret, accountOptions...)
    if accountAck != nil { fmt.Println(accountAck.StatusCode) }
    if err != nil { return err }

    containerOptions := []containers.SetTempURLKeyOption{
        containers.WithSetTempURLKeyOpts(containers.SetTempURLKeyOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        containers.WithSetTempURLKeyHeaders(map[string]string{"X-Trace-Label": "container-key"}),
        containers.WithSetTempURLKeyHeader("X-Request-Label", "container"),
        containers.WithSetTempURLKeySecondary(true),
    }
    containerAck, err := service.Containers.SetTempURLKey(ctx, name, containerSecret, containerOptions...)
    if containerAck != nil { fmt.Println(containerAck.StatusCode) }
    if err != nil { return err }

    getOptions := []objectstorage.GetTempURLKeyOption{
        objectstorage.WithGetTempURLKeyOpts(objectstorage.GetTempURLKeyOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        objectstorage.WithGetTempURLKeyContainer(name),
        objectstorage.WithGetTempURLKeyHeaders(map[string]string{"X-Trace-Label": "key-discovery"}),
        objectstorage.WithGetTempURLKeyHeader("X-Request-Label", "read"),
        objectstorage.WithGetTempURLKeyNewest(true),
        objectstorage.WithoutGetTempURLKeyNewest(),
        objectstorage.WithGetTempURLKeyNewest(false),
    }
    selected, err := service.GetTempURLKey(ctx, getOptions...)
    if selected != nil {
        if selected.Container != nil { fmt.Println(selected.Container.StatusCode) }
        if selected.Account != nil { fmt.Println(selected.Account.StatusCode) }
    }
    if err != nil { return err }
    fmt.Println(selected.Key != nil, selected.FromContainer, selected.Secondary)
    return nil
}

func main() {}
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L923)
- [ Base](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L68)
- [Tempurl](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/tempurl.py)
- [account core](accounts/temp_url_key_core_test.go)
- [options](accounts/temp_url_key_options_test.go)
- [container core](containers/temp_url_key_core_test.go)
- [options](containers/temp_url_key_options_test.go)
- [getter core](temp_url_key_core_test.go)
- [options](temp_url_key_options_test.go)
- [Temp Url Key Contracts Test](temp_url_key_contracts_test.go)
- [Connection Objectstorage Temp Url Key Test](../../connectiontest/connection_objectstorage_temp_url_key_test.go)
- [Swift Temp Url Key Test](../../internal/cmd/sdkgen/swift_temp_url_key_test.go)
