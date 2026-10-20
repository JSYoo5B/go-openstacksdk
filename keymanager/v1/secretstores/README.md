# Secretstores

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:key_manager/v1/secret_stores`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores.API.List`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores.API.All`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores.WithListGlobalDefault`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores.WithListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores.WithListFilter`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores.WithListFilters`

- Pinned _proxy.py:361-373 delegates the optional **query to SecretStore Resource.list and returns a lazy generator. Go List/All performs GET secret-stores accepting HTTP200 with a secret_stores array. All nine declared query names (name, status, global_default, crypto_plugin, secret_store_plugin, created, updated, limit and marker) remain server conditions rather than extra local predicates. WithListFilter/WithListFilters discards unknown semantic names, preserves explicit false/empty/repeated scalar query values and keeps raw WithListQuery extensions independent. Typed and semantic value/slice snapshots, lazy iterator slice ownership, independent reuse and concurrent library-option reuse are exercised.
  Tests: `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresListQuerySnapshotsAndConcurrentReuse`, `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresSemanticQueryOwnershipAndBodySeparation`, `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresSemanticLazyControlsCollisionsAndClear`.
- The five declared non-query Body attributes are evaluated from original fields: id prefers literal key presence including null/empty or otherwise the full original ref; created_at/updated_at select original created/updated text; secret_store_ref keeps the full ref; secret_store_id independently formats the last original path component even when literal id exists. Null/omission and empty remain distinct. Models retain canonical typed fields and raw row evidence without following foreign references; case-variant fields stay extensions and cannot replace canonical decoding. A selected empty-ref formatter fails once with ErrInvalidOption while a literal-id condition succeeds on that row; this local condition error does not synthesize a new HTTP ResponseError.
  Tests: `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresSingletonPathsModelsAndPassiveReferences`, `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresSemanticBodyAttributesUseOriginalFields`.
- A supplied limit, including a generated max-items limit hint, enables marker fallback on short nonempty pages using the literal canonical id if present or the FULL original secret_store_ref otherwise. A fully filtered short page still advances with that original full-ref marker; null/empty literal id never falls back to the ref and yields accepted response evidence if a continuation marker is needed. With no limit or hint only advertised links are followed, and empty pages stop.
  Tests: `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresMarkerFallbackUsesFullRawReference`, `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresListControlsAndLazyRowEvidence`, `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresSemanticRawCapAndContinuation`.
- MaxItems applies to consumed raw rows before local matching, using a limit hint only if limit is absent and preserving explicit limit. Filtered rows consume the cap rather than causing a refill page: cap-one yields no matches and cap-two yields one on the fixture's first page. Cap, single-page, consumer break and empty-page stops perform no extra HTTP and do not follow ignored continuations. Invalid consumed models or malformed whole-page JSON retain full accepted response body/header/status; All discards partial output on terminal errors while List can expose completed rows before a later HTTP404.
  Tests: `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresListControlsAndLazyRowEvidence`, `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresSemanticRawCapAndContinuation`.
- Advertised next/plural-links/HTTP Link continuation preserves filters and the same origin/exact collection path; foreign or changed-path/filter links and cycles fail. Later HTTP/cancellation/source-type/read errors remain terminal with their causes and no SDK composition resend or partial All result. Shared provider/middleware/token and lazy negative/nil/unsupported/control-query preflight are preserved. Semantic reserved controls, typed/raw query collisions including selected null presence, and selected JSON encoding failures stop before HTTP; superseded/cleared errors and unknown values are discarded. Capture before user options rejects endpoint replacement before HTTP, and a bulk clear replaces only semantic state while retaining raw query. Pre-cancel and an option which cancels with a custom cause and returns its own error preserve context.Canceled, the custom cause and the original option error before HTTP.
  Tests: `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresContinuationGuardsAndTerminalFailures`, `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresLiveSourcePreflightAndReadFailures`, `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresSemanticLazyControlsCollisionsAndClear`, `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresSemanticRawCapAndContinuation`.
- Connection.KeyManagerV1/KeyManager caches the authenticated key-manager facade and exposes native Secrets plus SDK-owned SecretStores and Quotas on the same ServiceClient/Provider. The cached SecretStores.All consumes a declared local secret_store_id condition using the shared reverse ResourceBase, rotated token and configured headers without an extra lookup or reference follow. Cancellation performs no HTTP. The independent registry advertises the actual owned APIs without inventing native quota/secret-store operations or publicly exposing the hidden semantic-filter Collection as Resources CRUD/Find/Wait.
  Tests: `connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedSharedClientAndFixedProject`, `connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedCancellationAndStrictMissing`, `internal/cmd/sdkgen/keymanager_services_test.go:TestKeyManagerSDKOwnedRegistryAndActualCapabilities`.

Python/Go differences:

- Go returns caller-owned SecretStore values with typed strings/boolean pointers and exact original Body/Header/StatusCode. Omitted and null pointer fields both become nil while Body preserves presence, null, unknown fields and precise extension numbers. Only exact canonical wire keys populate typed fields; case variants remain extensions. Python untyped field values and mutable Resource descriptor/default/seeded/dirty/cache state are broader and remain separate goal scope.
- Go convenience ID prefers exact literal id including null/empty, otherwise extracts the final raw path component of an absolute secret_store_ref. Semantic id and marker instead use literal raw id or the full alternate ref; semantic secret_store_id formats the reference independently. Ref formatting preserves Unicode/space/percent spelling, excludes query/fragment and never causes HTTP. Go URL parsing differs from Python urlsplit at some malformed authorities/control normalization, and selected-only formatting differs from eager Python property/to_dict evaluation.
- WithListFilter/WithListFilters maps the declared nine identity query names and five local Body attributes without caller builders or predicates; unknown semantic names are discarded. WithListQuery remains a separate explicit wire extension with repeated/empty values and overriding a same-key typed query. Later individual semantic assignments win; a bulk nil/empty replaces only semantic state. Same-key semantic and typed/raw query selections fail before HTTP rather than permissive dynamic merging.
- Semantic scalar arrays become repeated query values, booleans use lowercase text, numbers retain JSON text, and null/empty arrays retain selection presence without URL values. Query objects/nested arrays and selected non-JSON values are rejected. Local common JSON comparison preserves exact decimal numbers, original timestamps and null/missing semantics but distinguishes booleans from numbers, unlike Python True==1; returned rows still use the documented typed decoder.
- Go accepts strict HTTP200 and a secret_stores array, while generic Python Resource.list accepts final status<400 and can wrap a non-list plural value. Decode/read model failures retain accepted HTTP evidence; selected local formatter failures preserve the condition cause without synthesizing ResponseError. Configured Provider authentication/reauthentication/native retry remains its own transport policy. Exact origin/path/filter/cycle guards and immediate cap/first-page termination are explicit Go policies; fixture proof does not establish live Barbican marker support.
- SDK list controls require dedicated concrete options. Deprecated Proxy._list jmespath_filters and __conflicting_attrs recovery, optional GET request caching and inherited per-call session/resource_type/base_path/microversion/header/allow_unknown_params injection are separate common compatibility scope, not falsely implemented by the declared field classifier. The selected configured client/version is retained rather than replaced by an invented generic resource facade.
- Completion is restricted to the finite named python:key_manager/v1/secret_stores operation and its documented Go mapping. Other SecretStore getters, native operations, Resource lifecycle and independent list/CRUD/Find/Wait declarations are reviewed separately. Barbican 2024.1 project-reader new-default policy differs from Train admin policy and deployment overrides; no automatic privilege change is made. Offline HTTP tests and main compilation do not imply live-cloud or Python execution.

### `python:key_manager/v1/get_global_default_secret_store`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores.API.GetGlobalDefault`

- Pinned _proxy.py:374-382 is a no-input named getter returning one SecretStore through the fixed 'global-default' selector. Go GetGlobalDefault(ctx) reads the deployment global default using one GET secret-stores/global-default, HTTP200 and a ROOT object. There is no project/name lookup, response-ref follow-up, extra wait or implicit preferred fallback. Original client/provider, configured headers and exact raw/canonical typed fields are retained; response values/headers/body are caller-owned and passive foreign references never trigger HTTP.
  Tests: `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresSingletonPathsModelsAndPassiveReferences`, `connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedSharedClientAndFixedProject`.
- The shared singleton engine's representative preferred-route table rejects unexpected201/204, malformed/null/array root JSON and invalid canonical typed fields while preserving native HTTP or accepted ResponseError evidence. Actual HTTP404 remains strict ErrNotFound with native body/header/status; HTTP403 propagates. A transport error containing native404 stays a transport failure instead of becoming resource absence. Existing nil/invalid-source and standard canceled-context preflight performs no HTTP; the SDK composition does not resend because of these failures, while configured Provider retry/reauthentication remains a separate transport policy.
  Tests: `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresStrictSingletonErrorsAndAcceptedEvidence`, `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresLiveSourcePreflightAndReadFailures`, `connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedCancellationAndStrictMissing`.
- Connection.KeyManagerV1/KeyManager caches the authenticated key-manager facade and exposes native Secrets plus SDK-owned SecretStores and Quotas on the same ServiceClient/Provider. Both singleton getters use the reverse ResourceBase, tokens rotated after facade creation and configured source headers on that shared client, with no extra catalog lookup. Standard cancellation performs no HTTP and missing preferred causes no global-default fallback. The independent registry advertises the actual owned APIs without inventing native quota/secret-store operations or Resources CRUD/Find/Wait capabilities.
  Tests: `connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedSharedClientAndFixedProject`, `connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedCancellationAndStrictMissing`, `internal/cmd/sdkgen/keymanager_services_test.go:TestKeyManagerSDKOwnedRegistryAndActualCapabilities`.

Python/Go differences:

- Go returns caller-owned SecretStore values with typed strings/boolean pointers and exact original Body/Header/StatusCode. Omitted and null pointer fields both become nil while Body preserves presence, null, unknown fields and precise extension numbers. Only exact canonical wire keys populate typed fields; case variants remain extensions. Python untyped descriptor coercion and mutable/default/seeded/cache/dirty Resource state remain separate common SDK scope.
- For this singleton Python Proxy._get_resource seeds id='global-default', and Resource._translate_response updates existing Body state, so an omitted response literal id retains that selector as Resource.id. Go does not seed response metadata with the request selector: its convenience ID prefers response literal id including null/empty, otherwise passively extracts the final raw component of absolute secret_store_ref. Python secret_store_id independently formats the ref. This differs from an unseeded list Resource.id's full alternate reference; no response reference becomes a new request target.
- Go accepts exactly HTTP200 and a non-null ROOT JSON object, then validates canonical typed fields and any reference used to derive its convenience ID. Generic Python fetch accepts final status<400 and can retain its seeded resource when JSON parsing raises ValueError; arbitrary untyped field values and eager formatter/property behavior are broader. Strict typed decode/canonical-key/passive-ref validation with owned raw evidence is the documented Go mapping.
- Configured Provider authentication, reauthentication, transport/native retry and the selected service endpoint/ResourceBase/header/version are reused. No-input fixed-selector APIs provide no per-call base_path/session/adapter/header/microversion/skip_cache override. Generic fetch/request caching, descriptor/model overload and lifecycle compatibility remain separate original goal scope rather than unrelated blockers for these two named getters.
- HTTP/error table faults use the shared engine's representative preferred-route fixture; both global and preferred routes have direct ordinary/Connection success evidence. This does not claim every fault was re-executed for both selectors. Offline fixture/example evidence does not imply live-cloud/Python execution or complete SecretStore List/CRUD/Find/Wait support.
- Barbican 2024.1 new-default policy permits project-reader secretstores:get_global_default/get_preferred, while Train defaults require admin and deployment overrides may return403. Go neither assumes effective privilege nor automatically changes roles; these read-only getters do not imply preferred-backend administration.

### `python:key_manager/v1/get_preferred_secret_store`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores.API.GetPreferred`

- Pinned _proxy.py:384-392 is a no-input named getter returning one SecretStore through the fixed 'preferred' selector. Go GetPreferred(ctx) reads the current authenticated project's preferred backend using one GET secret-stores/preferred, HTTP200 and a ROOT object. There is no project/name lookup, response-ref follow-up, extra wait or implicit global-default fallback. Original client/provider, configured headers and exact raw/canonical typed fields are retained; response values/headers/body are caller-owned and passive foreign references never trigger HTTP.
  Tests: `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresSingletonPathsModelsAndPassiveReferences`, `connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedSharedClientAndFixedProject`.
- The shared singleton engine's representative preferred-route table rejects unexpected201/204, malformed/null/array root JSON and invalid canonical typed fields while preserving native HTTP or accepted ResponseError evidence. Actual HTTP404 remains strict ErrNotFound with native body/header/status; HTTP403 propagates. A transport error containing native404 stays a transport failure instead of becoming resource absence. Existing nil/invalid-source and standard canceled-context preflight performs no HTTP; the SDK composition does not resend because of these failures, while configured Provider retry/reauthentication remains a separate transport policy.
  Tests: `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresStrictSingletonErrorsAndAcceptedEvidence`, `api/keymanager_secretstores_test.go:TestKeyManagerSecretStoresLiveSourcePreflightAndReadFailures`, `connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedCancellationAndStrictMissing`.
- Connection.KeyManagerV1/KeyManager caches the authenticated key-manager facade and exposes native Secrets plus SDK-owned SecretStores and Quotas on the same ServiceClient/Provider. Both singleton getters use the reverse ResourceBase, tokens rotated after facade creation and configured source headers on that shared client, with no extra catalog lookup. Standard cancellation performs no HTTP and missing preferred causes no global-default fallback. The independent registry advertises the actual owned APIs without inventing native quota/secret-store operations or Resources CRUD/Find/Wait capabilities.
  Tests: `connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedSharedClientAndFixedProject`, `connection_keymanager_sdk_owned_test.go:TestConnectionKeyManagerSDKOwnedCancellationAndStrictMissing`, `internal/cmd/sdkgen/keymanager_services_test.go:TestKeyManagerSDKOwnedRegistryAndActualCapabilities`.

Python/Go differences:

- Go returns caller-owned SecretStore values with typed strings/boolean pointers and exact original Body/Header/StatusCode. Omitted and null pointer fields both become nil while Body preserves presence, null, unknown fields and precise extension numbers. Only exact canonical wire keys populate typed fields; case variants remain extensions. Python untyped descriptor coercion and mutable/default/seeded/cache/dirty Resource state remain separate common SDK scope.
- For this singleton Python Proxy._get_resource seeds id='preferred', and Resource._translate_response updates existing Body state, so an omitted response literal id retains that selector as Resource.id. Go does not seed response metadata with the request selector: its convenience ID prefers response literal id including null/empty, otherwise passively extracts the final raw component of absolute secret_store_ref. Python secret_store_id independently formats the ref. This differs from an unseeded list Resource.id's full alternate reference; no response reference becomes a new request target.
- Go accepts exactly HTTP200 and a non-null ROOT JSON object, then validates canonical typed fields and any reference used to derive its convenience ID. Generic Python fetch accepts final status<400 and can retain its seeded resource when JSON parsing raises ValueError; arbitrary untyped field values and eager formatter/property behavior are broader. Strict typed decode/canonical-key/passive-ref validation with owned raw evidence is the documented Go mapping.
- Configured Provider authentication, reauthentication, transport/native retry and the selected service endpoint/ResourceBase/header/version are reused. No-input fixed-selector APIs provide no per-call base_path/session/adapter/header/microversion/skip_cache override. Generic fetch/request caching, descriptor/model overload and lifecycle compatibility remain separate original goal scope rather than unrelated blockers for these two named getters.
- HTTP/error table faults use the shared engine's representative preferred-route fixture; both global and preferred routes have direct ordinary/Connection success evidence. This does not claim every fault was re-executed for both selectors. Offline fixture/example evidence does not imply live-cloud/Python execution or complete SecretStore List/CRUD/Find/Wait support.
- Barbican 2024.1 new-default policy permits project-reader secretstores:get_global_default/get_preferred, while Train defaults require admin and deployment overrides may return403. Go neither assumes effective privilege nor automatically changes roles; these read-only getters do not imply preferred-backend administration.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores"
)

func ReadBackends(ctx context.Context, client *gophercloud.ServiceClient) (*secretstores.SecretStore, *secretstores.SecretStore, error) {
	api := secretstores.New(client)
	global, err := api.GetGlobalDefault(ctx)
	if err != nil {
		return nil, nil, err
	}
	preferred, err := api.GetPreferred(ctx)
	return global, preferred, err
}
```

```go
package example

import (
	"context"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores"
)

func ListNonDefaultBackends(ctx context.Context, client *gophercloud.ServiceClient) ([]*secretstores.SecretStore, error) {
	return secretstores.New(client).All(ctx,
		secretstores.WithListOptions(secretstores.ListOpts{Status: "ACTIVE", Limit: 10}),
		secretstores.WithListGlobalDefault(false),
		secretstores.WithListMaxItems(5),
		secretstores.WithListPaginated(false),
	)
}
```

```python
stores = conn.key_manager.secret_stores(
    name="backend-pattern", secret_store_id="store-id", limit=10,
)
```

```go
stores, err := service.SecretStores.All(ctx,
    secretstores.WithListFilter("name", "backend-pattern"),
    secretstores.WithListFilter("secret_store_id", "store-id"),
    secretstores.WithListOptions(secretstores.ListOpts{Limit: 10}),
)
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "net/http"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretstores"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    name := flag.String("name", "", "server name query")
    id := flag.String("id", "", "local secret_store_id condition")
    timeout := flag.Duration("timeout", time.Minute, "authentication and list timeout")
    flag.Parse()
    if err := run(*cloud, *name, *id, *timeout); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(cloud, name, id string, timeout time.Duration) error {
    if timeout <= 0 {
        return fmt.Errorf("timeout must be positive")
    }
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil {
        return err
    }
    service, err := conn.KeyManagerV1(ctx)
    if err != nil {
        return err
    }
    options := []secretstores.ListOption{
        secretstores.WithListOptions(secretstores.ListOpts{Limit: 10}),
    }
    if name != "" {
        options = append(options, secretstores.WithListFilter("name", name))
    }
    if id != "" {
        options = append(options, secretstores.WithListFilter("secret_store_id", id))
    }
    stores, err := service.SecretStores.All(ctx, options...)
    if err != nil {
        return err
    }
    type item struct {
        ID         string                     `json:"id"`
        StatusCode int                        `json:"status_code"`
        Header     http.Header                `json:"header"`
        Body       map[string]json.RawMessage `json:"body"`
    }
    output := make([]item, 0, len(stores))
    for _, store := range stores {
        output = append(output, item{
            ID: store.ID,
            StatusCode: store.StatusCode,
            Header: store.Header,
            Body: store.Body,
        })
    }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(output)
}
```

## References

- [Keymanager Secretstores Test](../../../api/keymanager_secretstores_test.go)
- [Store Backends](https://docs.openstack.org/barbican/wallaby/api/reference/store_backends.html)
- [Policy](https://docs.openstack.org/barbican/2024.1/configuration/policy.html)
- [Policy](https://docs.openstack.org/barbican/train/configuration/policy.html)
