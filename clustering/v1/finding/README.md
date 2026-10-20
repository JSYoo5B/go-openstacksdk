# Finding

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:clustering/v1/find_profile`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.WithFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.WithFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.WithFindHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.WithFindMicroversion`

- The declared string identity is sent to the direct controller GET without a UUID heuristic. A successful response returns immediately without a list and may resolve a name or short ID to a different canonical returned ID.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFiveFacadesDirectGETAndCanonicalFields`.
- The default compatible policy falls back only on direct GET400/403/404;404-only and disabled fallback are owned Go selections. Native401/409/5xx, accepted-response decode/read and transport errors stay errors. Default missing returns nil,nil with strict override; list403/404 never become ignored absence, and successful empty fallback does not attach a suppressed GET403 cause.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFallbackPoliciesAndNativeStatuses`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityRawIDOrNameMatchingAndMissingDefaults`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityAcceptedMalformedAndConsumedRowEvidence`.
- Fallback sends a literal name query and matches exact raw lowercase id OR name, including an ID match with a different name. All advertised pages are checked after the first candidate; a second match is ambiguous even for a repeated same ID, and later errors or cancellation prevent partial success. No limit, cap, marker, or single-page hint is invented.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityRawIDOrNameMatchingAndMissingDefaults`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityScansLaterPagesForDuplicatesAndFailures`, `api/clustering_find_identity_test.go:TestClusteringFindIdentitySourceAndContextRecheckedBeforeFallbackAndContinuation`.
- Canonical response id/name are validated and normalized from exact raw fields rather than case-insensitive typed aliases. Invalid identities on nonmatching consumed rows, malformed accepted envelopes and read failures retain whole-response body/header/status evidence without fallback.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFiveFacadesDirectGETAndCanonicalFields`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityAcceptedMalformedAndConsumedRowEvidence`.
- Concrete options snapshot pointer/maps before HTTP; later settings win and concurrent siblings remain isolated. SDK-owned header/version/auth controls apply the same selection to GET and every fallback page while preserving live Provider tokens, original prefix, source revalidation and context cancellation. Body/query/foreign arguments, unsafe route identities, protected headers and invalid policies/versions fail before HTTP; existing explicit Ref Find stays deterministic.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityOptionsSnapshotsLastWinsAndConcurrentReuse`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityPreflightProtectsIdentityAndRequestControls`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityPhasesFreezeSelectionAndReadLiveToken`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityExistingExplicitReferencesRemainDeterministic`.

Python/Go differences:

- The pinned five clustering find proxy declarations accept only name_or_id:string and ignore_missing:boolean, without arbitrary kwargs. FindIdentity maps these declared inputs to context/error and SDK-owned functional options; inherited Resource.find query/URI/cache surfaces are tracked separately from the direct proxy operation.
- Default ignored absence matches the pinned Python proxy; explicit ID/Name API.Find remains available and Resources.Find keeps the shared strict default. Go additionally offers404-only and disabled fallback policies. SDK route input must be a safe single unescaped segment; other names can use existing Find(resource.Name(value)).
- Go enforces the declared singular/plural envelopes and canonical safe response id plus optional string/null name. Exact raw keys override case-variant scalar shadows, unknown fields and exact numbers remain in Body, and malformed extension types may remain typed decode errors rather than Python permissive/cache mutation behavior.
- FindIdentity headers and numeric microversion apply consistently to both GET and list phases; pinned Resource.find passes its per-call microversion only to GET. Go protects SDK auth/version/transport headers, rejects arbitrary lookup body/query/arguments, checks original and effective clients, and adds no Find-specific retry or cache.
- All advertised continuations retain same origin, collection path and fixed name query with cycle checks. The SDK does not manufacture a wire page limit; without an explicit limit the pinned source likewise has no implicit marker fallback. Whole-page validation can report malformed rows before returning a candidate or duplicate.

### `python:clustering/v1/find_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.WithFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.WithFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.WithFindHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.WithFindMicroversion`

- The declared string identity is sent to the direct controller GET without a UUID heuristic. A successful response returns immediately without a list and may resolve a name or short ID to a different canonical returned ID.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFiveFacadesDirectGETAndCanonicalFields`.
- The default compatible policy falls back only on direct GET400/403/404;404-only and disabled fallback are owned Go selections. Native401/409/5xx, accepted-response decode/read and transport errors stay errors. Default missing returns nil,nil with strict override; list403/404 never become ignored absence, and successful empty fallback does not attach a suppressed GET403 cause.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFallbackPoliciesAndNativeStatuses`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityRawIDOrNameMatchingAndMissingDefaults`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityAcceptedMalformedAndConsumedRowEvidence`.
- Fallback sends a literal name query and matches exact raw lowercase id OR name, including an ID match with a different name. All advertised pages are checked after the first candidate; a second match is ambiguous even for a repeated same ID, and later errors or cancellation prevent partial success. No limit, cap, marker, or single-page hint is invented.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityRawIDOrNameMatchingAndMissingDefaults`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityScansLaterPagesForDuplicatesAndFailures`, `api/clustering_find_identity_test.go:TestClusteringFindIdentitySourceAndContextRecheckedBeforeFallbackAndContinuation`.
- Canonical response id/name are validated and normalized from exact raw fields rather than case-insensitive typed aliases. Invalid identities on nonmatching consumed rows, malformed accepted envelopes and read failures retain whole-response body/header/status evidence without fallback.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFiveFacadesDirectGETAndCanonicalFields`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityAcceptedMalformedAndConsumedRowEvidence`.
- Concrete options snapshot pointer/maps before HTTP; later settings win and concurrent siblings remain isolated. SDK-owned header/version/auth controls apply the same selection to GET and every fallback page while preserving live Provider tokens, original prefix, source revalidation and context cancellation. Body/query/foreign arguments, unsafe route identities, protected headers and invalid policies/versions fail before HTTP; existing explicit Ref Find stays deterministic.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityOptionsSnapshotsLastWinsAndConcurrentReuse`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityPreflightProtectsIdentityAndRequestControls`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityPhasesFreezeSelectionAndReadLiveToken`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityExistingExplicitReferencesRemainDeterministic`.

Python/Go differences:

- The pinned five clustering find proxy declarations accept only name_or_id:string and ignore_missing:boolean, without arbitrary kwargs. FindIdentity maps these declared inputs to context/error and SDK-owned functional options; inherited Resource.find query/URI/cache surfaces are tracked separately from the direct proxy operation.
- Default ignored absence matches the pinned Python proxy; explicit ID/Name API.Find remains available and Resources.Find keeps the shared strict default. Go additionally offers404-only and disabled fallback policies. SDK route input must be a safe single unescaped segment; other names can use existing Find(resource.Name(value)).
- Go enforces the declared singular/plural envelopes and canonical safe response id plus optional string/null name. Exact raw keys override case-variant scalar shadows, unknown fields and exact numbers remain in Body, and malformed extension types may remain typed decode errors rather than Python permissive/cache mutation behavior.
- FindIdentity headers and numeric microversion apply consistently to both GET and list phases; pinned Resource.find passes its per-call microversion only to GET. Go protects SDK auth/version/transport headers, rejects arbitrary lookup body/query/arguments, checks original and effective clients, and adds no Find-specific retry or cache.
- All advertised continuations retain same origin, collection path and fixed name query with cycle checks. The SDK does not manufacture a wire page limit; without an explicit limit the pinned source likewise has no implicit marker fallback. Whole-page validation can report malformed rows before returning a candidate or duplicate.

### `python:clustering/v1/find_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithFindHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithFindMicroversion`

- The declared string identity is sent to the direct controller GET without a UUID heuristic. A successful response returns immediately without a list and may resolve a name or short ID to a different canonical returned ID.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFiveFacadesDirectGETAndCanonicalFields`.
- The default compatible policy falls back only on direct GET400/403/404;404-only and disabled fallback are owned Go selections. Native401/409/5xx, accepted-response decode/read and transport errors stay errors. Default missing returns nil,nil with strict override; list403/404 never become ignored absence, and successful empty fallback does not attach a suppressed GET403 cause.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFallbackPoliciesAndNativeStatuses`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityRawIDOrNameMatchingAndMissingDefaults`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityAcceptedMalformedAndConsumedRowEvidence`.
- Fallback sends a literal name query and matches exact raw lowercase id OR name, including an ID match with a different name. All advertised pages are checked after the first candidate; a second match is ambiguous even for a repeated same ID, and later errors or cancellation prevent partial success. No limit, cap, marker, or single-page hint is invented.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityRawIDOrNameMatchingAndMissingDefaults`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityScansLaterPagesForDuplicatesAndFailures`, `api/clustering_find_identity_test.go:TestClusteringFindIdentitySourceAndContextRecheckedBeforeFallbackAndContinuation`.
- Canonical response id/name are validated and normalized from exact raw fields rather than case-insensitive typed aliases. Invalid identities on nonmatching consumed rows, malformed accepted envelopes and read failures retain whole-response body/header/status evidence without fallback.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFiveFacadesDirectGETAndCanonicalFields`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityAcceptedMalformedAndConsumedRowEvidence`.
- Concrete options snapshot pointer/maps before HTTP; later settings win and concurrent siblings remain isolated. SDK-owned header/version/auth controls apply the same selection to GET and every fallback page while preserving live Provider tokens, original prefix, source revalidation and context cancellation. Body/query/foreign arguments, unsafe route identities, protected headers and invalid policies/versions fail before HTTP; existing explicit Ref Find stays deterministic.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityOptionsSnapshotsLastWinsAndConcurrentReuse`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityPreflightProtectsIdentityAndRequestControls`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityPhasesFreezeSelectionAndReadLiveToken`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityExistingExplicitReferencesRemainDeterministic`.

Python/Go differences:

- The pinned five clustering find proxy declarations accept only name_or_id:string and ignore_missing:boolean, without arbitrary kwargs. FindIdentity maps these declared inputs to context/error and SDK-owned functional options; inherited Resource.find query/URI/cache surfaces are tracked separately from the direct proxy operation.
- Default ignored absence matches the pinned Python proxy; explicit ID/Name API.Find remains available and Resources.Find keeps the shared strict default. Go additionally offers404-only and disabled fallback policies. SDK route input must be a safe single unescaped segment; other names can use existing Find(resource.Name(value)).
- Go enforces the declared singular/plural envelopes and canonical safe response id plus optional string/null name. Exact raw keys override case-variant scalar shadows, unknown fields and exact numbers remain in Body, and malformed extension types may remain typed decode errors rather than Python permissive/cache mutation behavior.
- FindIdentity headers and numeric microversion apply consistently to both GET and list phases; pinned Resource.find passes its per-call microversion only to GET. Go protects SDK auth/version/transport headers, rejects arbitrary lookup body/query/arguments, checks original and effective clients, and adds no Find-specific retry or cache.
- All advertised continuations retain same origin, collection path and fixed name query with cycle checks. The SDK does not manufacture a wire page limit; without an explicit limit the pinned source likewise has no implicit marker fallback. Whole-page validation can report malformed rows before returning a candidate or duplicate.

### `python:clustering/v1/find_node`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.WithFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.WithFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.WithFindHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.WithFindMicroversion`

- The declared string identity is sent to the direct controller GET without a UUID heuristic. A successful response returns immediately without a list and may resolve a name or short ID to a different canonical returned ID.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFiveFacadesDirectGETAndCanonicalFields`.
- The default compatible policy falls back only on direct GET400/403/404;404-only and disabled fallback are owned Go selections. Native401/409/5xx, accepted-response decode/read and transport errors stay errors. Default missing returns nil,nil with strict override; list403/404 never become ignored absence, and successful empty fallback does not attach a suppressed GET403 cause.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFallbackPoliciesAndNativeStatuses`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityRawIDOrNameMatchingAndMissingDefaults`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityAcceptedMalformedAndConsumedRowEvidence`.
- Fallback sends a literal name query and matches exact raw lowercase id OR name, including an ID match with a different name. All advertised pages are checked after the first candidate; a second match is ambiguous even for a repeated same ID, and later errors or cancellation prevent partial success. No limit, cap, marker, or single-page hint is invented.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityRawIDOrNameMatchingAndMissingDefaults`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityScansLaterPagesForDuplicatesAndFailures`, `api/clustering_find_identity_test.go:TestClusteringFindIdentitySourceAndContextRecheckedBeforeFallbackAndContinuation`.
- Canonical response id/name are validated and normalized from exact raw fields rather than case-insensitive typed aliases. Invalid identities on nonmatching consumed rows, malformed accepted envelopes and read failures retain whole-response body/header/status evidence without fallback.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFiveFacadesDirectGETAndCanonicalFields`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityAcceptedMalformedAndConsumedRowEvidence`.
- Concrete options snapshot pointer/maps before HTTP; later settings win and concurrent siblings remain isolated. SDK-owned header/version/auth controls apply the same selection to GET and every fallback page while preserving live Provider tokens, original prefix, source revalidation and context cancellation. Body/query/foreign arguments, unsafe route identities, protected headers and invalid policies/versions fail before HTTP; existing explicit Ref Find stays deterministic.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityOptionsSnapshotsLastWinsAndConcurrentReuse`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityPreflightProtectsIdentityAndRequestControls`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityPhasesFreezeSelectionAndReadLiveToken`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityExistingExplicitReferencesRemainDeterministic`.

Python/Go differences:

- The pinned five clustering find proxy declarations accept only name_or_id:string and ignore_missing:boolean, without arbitrary kwargs. FindIdentity maps these declared inputs to context/error and SDK-owned functional options; inherited Resource.find query/URI/cache surfaces are tracked separately from the direct proxy operation.
- Default ignored absence matches the pinned Python proxy; explicit ID/Name API.Find remains available and Resources.Find keeps the shared strict default. Go additionally offers404-only and disabled fallback policies. SDK route input must be a safe single unescaped segment; other names can use existing Find(resource.Name(value)).
- Go enforces the declared singular/plural envelopes and canonical safe response id plus optional string/null name. Exact raw keys override case-variant scalar shadows, unknown fields and exact numbers remain in Body, and malformed extension types may remain typed decode errors rather than Python permissive/cache mutation behavior.
- FindIdentity headers and numeric microversion apply consistently to both GET and list phases; pinned Resource.find passes its per-call microversion only to GET. Go protects SDK auth/version/transport headers, rejects arbitrary lookup body/query/arguments, checks original and effective clients, and adds no Find-specific retry or cache.
- All advertised continuations retain same origin, collection path and fixed name query with cycle checks. The SDK does not manufacture a wire page limit; without an explicit limit the pinned source likewise has no implicit marker fallback. Whole-page validation can report malformed rows before returning a candidate or duplicate.

### `python:clustering/v1/find_receiver`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.WithFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.WithFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.WithFindHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.WithFindMicroversion`

- The declared string identity is sent to the direct controller GET without a UUID heuristic. A successful response returns immediately without a list and may resolve a name or short ID to a different canonical returned ID.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFiveFacadesDirectGETAndCanonicalFields`.
- The default compatible policy falls back only on direct GET400/403/404;404-only and disabled fallback are owned Go selections. Native401/409/5xx, accepted-response decode/read and transport errors stay errors. Default missing returns nil,nil with strict override; list403/404 never become ignored absence, and successful empty fallback does not attach a suppressed GET403 cause.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFallbackPoliciesAndNativeStatuses`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityRawIDOrNameMatchingAndMissingDefaults`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityAcceptedMalformedAndConsumedRowEvidence`.
- Fallback sends a literal name query and matches exact raw lowercase id OR name, including an ID match with a different name. All advertised pages are checked after the first candidate; a second match is ambiguous even for a repeated same ID, and later errors or cancellation prevent partial success. No limit, cap, marker, or single-page hint is invented.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityRawIDOrNameMatchingAndMissingDefaults`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityScansLaterPagesForDuplicatesAndFailures`, `api/clustering_find_identity_test.go:TestClusteringFindIdentitySourceAndContextRecheckedBeforeFallbackAndContinuation`.
- Canonical response id/name are validated and normalized from exact raw fields rather than case-insensitive typed aliases. Invalid identities on nonmatching consumed rows, malformed accepted envelopes and read failures retain whole-response body/header/status evidence without fallback.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityFiveFacadesDirectGETAndCanonicalFields`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityAcceptedMalformedAndConsumedRowEvidence`.
- Concrete options snapshot pointer/maps before HTTP; later settings win and concurrent siblings remain isolated. SDK-owned header/version/auth controls apply the same selection to GET and every fallback page while preserving live Provider tokens, original prefix, source revalidation and context cancellation. Body/query/foreign arguments, unsafe route identities, protected headers and invalid policies/versions fail before HTTP; existing explicit Ref Find stays deterministic.
  Tests: `api/clustering_find_identity_test.go:TestClusteringFindIdentityOptionsSnapshotsLastWinsAndConcurrentReuse`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityPreflightProtectsIdentityAndRequestControls`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityPhasesFreezeSelectionAndReadLiveToken`, `api/clustering_find_identity_test.go:TestClusteringFindIdentityExistingExplicitReferencesRemainDeterministic`.

Python/Go differences:

- The pinned five clustering find proxy declarations accept only name_or_id:string and ignore_missing:boolean, without arbitrary kwargs. FindIdentity maps these declared inputs to context/error and SDK-owned functional options; inherited Resource.find query/URI/cache surfaces are tracked separately from the direct proxy operation.
- Default ignored absence matches the pinned Python proxy; explicit ID/Name API.Find remains available and Resources.Find keeps the shared strict default. Go additionally offers404-only and disabled fallback policies. SDK route input must be a safe single unescaped segment; other names can use existing Find(resource.Name(value)).
- Go enforces the declared singular/plural envelopes and canonical safe response id plus optional string/null name. Exact raw keys override case-variant scalar shadows, unknown fields and exact numbers remain in Body, and malformed extension types may remain typed decode errors rather than Python permissive/cache mutation behavior.
- FindIdentity headers and numeric microversion apply consistently to both GET and list phases; pinned Resource.find passes its per-call microversion only to GET. Go protects SDK auth/version/transport headers, rejects arbitrary lookup body/query/arguments, checks original and effective clients, and adds no Find-specific retry or cache.
- All advertised continuations retain same origin, collection path and fixed name query with cycle checks. The SDK does not manufacture a wire page limit; without an explicit limit the pinned source likewise has no implicit marker fallback. Whole-page validation can report malformed rows before returning a candidate or duplicate.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
profile = conn.clustering.find_profile("worker_template")
policy = conn.clustering.find_policy("placement")
cluster = conn.clustering.find_cluster("workers")
node = conn.clustering.find_node("worker_1")
receiver = conn.clustering.find_receiver("scale_out")
```

```go
package example

import (
    "context"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
)

func FindSenlinResources(ctx context.Context, conn *sdk.Connection) error {
    service, err := conn.Clustering(ctx)
    if err != nil { return err }
    profile, err := service.Profiles.FindIdentity(ctx, "worker_template")
    if err != nil { return err }
    policy, err := service.Policies.FindIdentity(ctx, "placement")
    if err != nil { return err }
    cluster, err := service.Clusters.FindIdentity(ctx, "workers")
    if err != nil { return err }
    node, err := service.Nodes.FindIdentity(ctx, "worker_1")
    if err != nil { return err }
    receiver, err := service.Receivers.FindIdentity(ctx, "scale_out")
    if err != nil { return err }
    if profile != nil { fmt.Println("profile", profile.ID) }
    if policy != nil { fmt.Println("policy", policy.ID) }
    if cluster != nil { fmt.Println("cluster", cluster.ID) }
    if node != nil { fmt.Println("node", node.ID) }
    if receiver != nil { fmt.Println("receiver", receiver.ID) }
    return nil
}
```

```go
package example

import (
    "context"
    "errors"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindClusterStrict(ctx context.Context, conn *sdk.Connection) error {
    service, err := conn.Clustering(ctx)
    if err != nil { return err }
    value, err := service.Clusters.FindIdentity(ctx, "workers",
        clusters.WithFindIgnoreMissing(false),
        clusters.WithFindFallback(resource.FindFallbackNotFoundOnly),
        clusters.WithFindHeader("X-Vendor-Trace", "lookup"),
        clusters.WithFindMicroversion("1.7"))
    if errors.Is(err, resource.ErrNotFound) { return fmt.Errorf("workers cluster: %w", err) }
    if err != nil { return err }
    fmt.Println(value.ID, value.Name, value.Header, value.StatusCode)
    return nil
}
```

## References

- [Listing](../listing/README.md)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2439)
- [Find Identity](../../../internal/senlin/find_identity.go)
- [Find Identity Test](../../../internal/senlin/find_identity_test.go)
- [Clustering Find Identity Test](../../../api/clustering_find_identity_test.go)
