# Metadata

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:clustering/v1/fetch_cluster_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.FetchMetadata`

- The functional metadata read uses the supported cluster GET200 object route, retaining an independent exact raw metadata map and original Cluster Body/Header/status. Explicit omitted, null and empty-object metadata remain distinct through Present/Null flags, canonical lowercase metadata owns the view despite case-folded extensions, and incidental Location never causes another request.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataReadViewsPreservePresenceAndRawAuthority`, `connection_senlin_metadata_test.go:TestConnectionSenlinMetadataReadAliasesAndOwnedDelete`.
- Explicit ID references remain supplied controller routes; Name references resolve a canonical lowercase raw id/name once. Invalid input/source/context fails before HTTP, malformed accepted GET metadata preserves the full original response, and native 403/404 errors retain status/body/header with 404 additionally ErrNotFound.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataNameResolutionUsesCanonicalRawFields`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataPreflightRejectsUnsupportedInputs`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataReadFailuresPreserveWholeResponse`.

Python/Go differences:

- Pinned _proxy.py:384-399 constructs or reuses a Cluster Resource and MetadataMixin.fetch_metadata uses GET /clusters/{id}/metadata, updating that mutable Resource only if the response includes metadata (common/metadata.py:33-46). Senlin16.0.0 router has no cluster metadata subresource route. Go implements the functional read intent through the supported GET /clusters/{id} and returns a fresh MetadataView instead of attempting the absent route.
- Go uses explicit ID/Name references and context, validates exact canonical Name-list identities, fixes an explicit request identity independently of incidental response IDs, and exposes raw precision/presence/evidence instead of Python descriptor and cached-instance behavior. It does not claim inherited Resource/cache compatibility or automatic action completion.
- This is the documented stateless Go read mapping: supported cluster GET projection, explicit Ref/context and independently owned raw response/presence fields. Mutable input-resource identity/cache and Python runtime deprecation warnings are Go policy differences, not a claim that the absent stock-server metadata route exists.

### `python:clustering/v1/get_cluster_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.GetMetadata`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.FetchMetadata`

- GetMetadata delegates to FetchMetadata and issues an independent supported cluster GET200 with the same raw precision, omission/null distinction and original response evidence. The alias does not reuse a previous read, follow incidental Location or claim a metadata subresource exists.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataReadViewsPreservePresenceAndRawAuthority`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataReadFailuresPreserveWholeResponse`, `connection_senlin_metadata_test.go:TestConnectionSenlinMetadataReadAliasesAndOwnedDelete`.
- Both read names share source validation and explicit controller identity policy; invalid references, nil/foreign clients, symbolic or malformed selected versions and cancellation remain observable before HTTP.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataPreflightRejectsUnsupportedInputs`.

Python/Go differences:

- Pinned _proxy.py:401-417 emits a deprecation warning and calls fetch_cluster_metadata, whose MetadataMixin GET /clusters/{id}/metadata route is absent from the audited Senlin16.0.0 router. Go keeps a compatibility method alias to its supported cluster GET read and does not emit Python's runtime warning.
- The Go alias returns a fresh typed MetadataView with independent raw values and Cluster evidence rather than updating/reusing the supplied Python Resource instance. Explicit ID/Name and context policies match the Go functional FetchMetadata alternative.
- This is the documented stateless Go read mapping: supported cluster GET projection, explicit Ref/context and independently owned raw response/presence fields. Mutable input-resource identity/cache and Python runtime deprecation warnings are Go policy differences, not a claim that the absent stock-server metadata route exists.

### `python:clustering/v1/set_cluster_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.SetMetadata`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.PythonMetadata`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.PythonMetadataProfile.SetMetadata`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.PythonMetadataProfile.ReplaceMetadata`

- Set snapshots input JSON and protected header-only options before lookup, observes metadata via GET and shallow-merges top-level keys while preserving unrelated exact raw numbers/null/unknown values. One full-object PATCH202 stays on the supplied ID route or the once-resolved canonical Name route, uses the source's current token/version and returns independent Previous/Requested/ResponseCluster/Operation without claiming requested values were applied.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataShallowMergeSnapshotsAndQueuedResult`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataNameResolutionUsesCanonicalRawFields`, `connection_senlin_metadata_test.go:TestConnectionSenlinMetadataSharesSourceAndLocksResolvedRoute`.
- Nil or empty updates and semantically equal exact JSON perform GET but no PATCH; object order and decimal spelling do not fabricate a change, while boolean/number and nested missing/null remain distinct. Serialization, unsupported body/query/arguments, owned headers, source changes and canceled contexts fail with their original causes before another request or no-op success.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataNoOpsUseExactJSONEquality`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataPreflightRejectsUnsupportedInputs`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataRevalidatesSourceBeforeEveryRequest`, `connection_senlin_metadata_test.go:TestConnectionSenlinMetadataNoOpAndSourceRecheck`.
- Malformed GET or accepted PATCH envelopes/metadata, missing/foreign/multiple action Location and cancellation after acceptance retain full original HTTP evidence without a resend or action GET. Wrong status and server failures remain native. The RMW concurrency fixture demonstrates that a concurrent key absent from the observed object is not merged into the replacement and queued acceptance does not imply final state.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataReadFailuresPreserveWholeResponse`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataAcceptedFailuresNeverResendOrFollowLocation`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataCancellationAfterAcceptanceRetainsEvidence`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataReadModifyWriteDoesNotClaimConcurrencyOrCompletion`, `connection_senlin_metadata_test.go:TestConnectionSenlinMetadataAcceptedFailuresKeepOriginalEvidence`.
- The explicit PythonMetadata profile sends POST/PUT {"metadata": ...} and per-key DELETE routes, preserves nil-as-{} and ordering, returns opaque receipts and partial deletion results, and keeps a stock 404 as an error without fallback to GET+PATCH.
  Tests: `api/r2_small_clustering_test.go:TestR2SmallClusteringPythonMetadataRoutesAndBodies`, `api/r2_small_clustering_test.go:TestR2SmallClusteringPythonMetadataStock404PartialDeleteAndRejection`.

Python/Go differences:

- The routes are absent from the audited Senlin router, so stock deployments return 404; the profile is an opt-in to the pinned Python request shape and not a capability claim. The GET+PATCH alternative (API.SetMetadata/DeleteMetadata) remains the stock-compatible path and is never used as fallback.
- Go accepts a validated Senlin cluster ID string and returns opaque receipts instead of a mutable Cluster holding the submitted metadata; metadata is not synthesized from input.
- Key deletion is sequential and not atomic; Go returns the keys acknowledged before the first failure. Python raises on the first failure without partial output.

### `python:clustering/v1/delete_cluster_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.DeleteMetadata`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.WithMetadataHeader`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.PythonMetadata`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.PythonMetadataProfile.DeleteMetadata`

- Nil keys clear the observed metadata with one whole-object empty replacement; a nonnil empty key slice performs no HTTP and fabricates no observation. Selected duplicate or arbitrary string keys are snapshotted before GET, removed together and sent as one PATCH while preserving exact unrelated values. Missing selected keys and already empty/omitted/null metadata perform GET without PATCH.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataSelectedDeleteIsOneOwnedReplacement`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataNoOpsUseExactJSONEquality`, `connection_senlin_metadata_test.go:TestConnectionSenlinMetadataReadAliasesAndOwnedDelete`.
- Header-only capability and source/context validation applies even to the zero-HTTP empty-key branch and after custom option side effects. Name resolution and explicit ID routes remain fixed, native missing/server errors are not hidden, and an accepted replacement retains required action Location/evidence without a retry or automatic follow-up.
  Tests: `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataZeroHTTPDeleteRevalidatesOptionSideEffects`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataPreflightRejectsUnsupportedInputs`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataRevalidatesSourceBeforeEveryRequest`, `api/clustering_cluster_metadata_test.go:TestClusteringClusterMetadataAcceptedFailuresNeverResendOrFollowLocation`, `connection_senlin_metadata_test.go:TestConnectionSenlinMetadataNoOpAndSourceRecheck`.
- The explicit PythonMetadata profile sends POST/PUT {"metadata": ...} and per-key DELETE routes, preserves nil-as-{} and ordering, returns opaque receipts and partial deletion results, and keeps a stock 404 as an error without fallback to GET+PATCH.
  Tests: `api/r2_small_clustering_test.go:TestR2SmallClusteringPythonMetadataRoutesAndBodies`, `api/r2_small_clustering_test.go:TestR2SmallClusteringPythonMetadataStock404PartialDeleteAndRejection`.

Python/Go differences:

- The routes are absent from the audited Senlin router, so stock deployments return 404; the profile is an opt-in to the pinned Python request shape and not a capability claim. The GET+PATCH alternative (API.SetMetadata/DeleteMetadata) remains the stock-compatible path and is never used as fallback.
- Go accepts a validated Senlin cluster ID string and returns opaque receipts instead of a mutable Cluster holding the submitted metadata; metadata is not synthesized from input.
- Key deletion is sequential and not atomic; Go returns the keys acknowledged before the first failure. Python raises on the first failure without partial output.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
cluster = conn.clustering.fetch_cluster_metadata("CLUSTER_NAME_OR_ID")
print(cluster.metadata)
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
view, err := service.Clusters.FetchMetadata(ctx, resource.ID("CLUSTER_NAME_OR_ID"))
if err != nil { return err }
fmt.Println(view.Present, view.Null, string(view.Values["team"]))
```

```python
conn.clustering.set_cluster_metadata("CLUSTER_ID", team="infra", nullable=None)
```

```go
result, err := service.Clusters.SetMetadata(ctx, resource.ID("CLUSTER_ID"),
    map[string]any{
        "team": "infra",
        "nullable": nil,
        "big": json.Number("9007199254740993"),
    }, clusters.WithMetadataHeader("X-Request-Id", "metadata-change"))
if err != nil { return err }
fmt.Println(result.Changed, string(result.Previous["team"]), string(result.Requested["team"]))
if result.Operation != nil { fmt.Println(result.Operation.ActionID) }
```

```python
conn.clustering.delete_cluster_metadata("CLUSTER_ID", keys=["team", "nullable"])
conn.clustering.delete_cluster_metadata("CLUSTER_ID")
```

```go
result, err := service.Clusters.DeleteMetadata(ctx, resource.ID("CLUSTER_ID"),
    []string{"team", "nullable"})
if err != nil { return err }
result, err = service.Clusters.DeleteMetadata(ctx, resource.ID("CLUSTER_ID"), nil)
if err != nil { return err }
fmt.Println(result.Changed)
```

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#update-a-cluster)
- [16.0.0 router](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/router.py)
- [cluster controller](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/clusters.py)
- [conductor](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/conductor/service.py)
