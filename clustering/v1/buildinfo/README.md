# Buildinfo

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:clustering/v1/get_build_info`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/buildinfo.API.Get`

- The build-info singleton has no fabricated ID or body and retains separate API/engine revisions, large raw values and accepted HTTP evidence.
  Tests: `api/clustering_discovery_test.go:TestClusteringDiscoveryBuildInfoSingletonAndRawEvidence`.
- Native403/404, invalid accepted resource bodies, canceled contexts and changed redirect targets preserve errors and prevent unauthorized extra HTTP.
  Tests: `api/clustering_discovery_test.go:TestClusteringDiscoveryHTTPAndDecodeErrorsRetainEvidence`, `api/clustering_discovery_test.go:TestClusteringDiscoveryInvalidInputsCancelAndRedirectBeforeSecondRequest`.

Python/Go differences:

- Go takes explicit type-name strings or concrete options with context/error and returns typed snapshots retaining raw JSON and HTTP evidence.
- Go validates the documented endpoint-specific envelope instead of Python permissive flat/empty response fallback; resources are fetched through the API rather than mutated and implicitly committed.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
info, err := buildinfo.New(client).Get(ctx)
if err != nil {
    return err
}
if info.API != nil && info.API.Revision != nil {
    fmt.Println(*info.API.Revision)
}
```

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#build-information-build-info)
