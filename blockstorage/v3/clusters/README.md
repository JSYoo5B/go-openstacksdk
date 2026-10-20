# Clusters

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/get_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.API.Get`

- Get sends GET clusters/{name} at min(3.26, server max), seeds the requested value as id and decodes the cluster envelope, wrapping 404 with operation context.
  Tests: `blockstorage/v3/clusters/api_test.go:TestClusterGetNegotiatesMicroversion326`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts only the ID string and URL-escapes it as one path segment, rejecting blank, dot and control-character IDs before HTTP, while Python urljoin does not escape.
- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.

### `python:block_storage/v3/find_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.WithFindIgnoreMissing`

- Find sends GET clusters/{name_or_id} and after a clean 404 lists clusters/detail?name={name_or_id} matching id or name, returning nil by default and ErrNotFound with WithFindIgnoreMissing(false).
  Tests: `blockstorage/v3/clusters/api_test.go:TestClusterFindFallsBackToDetailListByName`.

Python/Go differences:

- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- The Resource location attribute and the find_* deprecation warning are not reproduced.
- A missing cluster with ignore_missing=False is resource.ErrNotFound instead of NotFoundException.

### `python:block_storage/v3/clusters`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.API.All`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.WithListDetails`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.WithListQuery`

- List sends GET clusters/detail by default or clusters with WithListDetails(false), maps the Cluster QueryParameters to Requests-encoded query and filters undeclared-query Body attributes such as state locally.
  Tests: `blockstorage/v3/clusters/api_test.go:TestClusterListDetailsQueryAndLocalFilters`.

Python/Go differences:

- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- Go returns a lazy iter.Seq2 (or All for a slice) instead of a generator; local filters compare JSON values without Python type conversion.

### `python:block_storage/v3/enable_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.API.Enable`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.NewTarget`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.Cluster.Target`

- Enable sends PUT clusters/enable with {"name": name} plus a nonempty binary at the configured client microversion only (no discovery) and decodes the cluster envelope.
  Tests: `blockstorage/v3/clusters/api_test.go:TestClusterEnableDisableUseConfiguredMicroversion`.

Python/Go differences:

- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- Per-request header options are a Go extension; authentication, microversion and content headers stay SDK owned.
- Python takes a Cluster instance or string; Go takes a Target with Name and Binary, where a nil Name reproduces the {"name": null} body Python sends for a plain string.

### `python:block_storage/v3/disable_cluster`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.API.Disable`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.WithDisableReason`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.NewTarget`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/clusters.Cluster.Target`

- Disable sends PUT clusters/disable with name, a nonempty binary and a nonempty disabled_reason at the configured client microversion and wraps 400 with operation context.
  Tests: `blockstorage/v3/clusters/api_test.go:TestClusterEnableDisableUseConfiguredMicroversion`.

Python/Go differences:

- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- Per-request header options are a Go extension; authentication, microversion and content headers stay SDK owned.
- Python takes a Cluster instance or string; Go takes a Target with Name and Binary, where a nil Name reproduces the {"name": null} body Python sends for a plain string.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
cluster = conn.block_storage.get_cluster("cluster1@lvm")
conn.block_storage.disable_cluster(cluster, reason="maintenance")
```

```go
api := clusters.New(volumeClient)
cluster, err := api.Get(ctx, "cluster1@lvm")
if err != nil { return err }
_, err = api.Disable(ctx, cluster.Target(), clusters.WithDisableReason("maintenance"))
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1957-L2065)
- [`cluster.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/cluster.py)
