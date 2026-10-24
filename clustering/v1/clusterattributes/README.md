# Clusterattributes

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:clustering/v1/collect_cluster_attrs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterattributes.API.InCluster`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterattributes.Scope.List`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterattributes.Scope.All`

- The fixed escaped JSONPath collection requires selected numeric Senlin1.2+, trims the path and rejects empty/None before parent HTTP, confirms the canonical parent once and rechecks the source/version before each collection page.
  Tests: `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesJSONPathRemainsOneFixedSegment`, `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesMinimumVersionAndPathPreflight`, `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesParentCanonicalizationAndGateRecheck`, `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesExactParentNameUsesCanonicalRawField`.
- Both published HTTP202 and Senlin16.0.0 controller/WSGI HTTP200 decode cluster_attributes synchronously. Exact lowercase id/value fields preserve object/array/scalar, huge numbers and null-versus-omitted values with independent raw Body/Header/status; incidental Location/action does not create a Submission or another request.
  Tests: `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesSynchronousCodesAndRawValues`, `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesCanonicalRawFieldsOwnIdentityAndValue`, `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesResponseErrorsPreserveEvidence`.
- The list-only iterator is lazy, stops on break, follows only explicit links retaining its fixed parent and escaped path, and checks version changes before another page. Initial queries and fabricated Get/Delete/status capabilities are rejected, preserving cancellation and whole malformed-response evidence.
  Tests: `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesLazyExplicitPagingAndVersionGuard`, `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesListOnlyCapabilitiesAndQueries`, `api/clustering_cluster_attributes_test.go:TestClusteringClusterAttributesResponseErrorsPreserveEvidence`.

Python/Go differences:

- Pinned _proxy.py:630-648 declares **query but forwards only cluster_id and path to _list. Every supplied query/control is ignored by that proxy. Go correspondingly exposes no List/All query options and rejects initial Resources queries; absent query forwarding is not a missing direct collect proxy contract.
- Go performs a parent confirmation request or exact Name search and fixes its canonical raw ID. Python directly interpolates the supplied parent/path URI values. Go trims and escapes the path as one segment and requires selected numeric1.2+ instead of relying on implicit adapter negotiation and server validation.
- The public API-ref says HTTP202, while the audited Senlin16.0.0 router has no cluster_collect success override and WSGI defaults to HTTP200. Both are accepted as synchronous read results, with no fabricated action state and no live-cloud completion claim.
- Go requires the documented array and canonical valid node id and stores fixed URIClusterID/URIPath separately; raw value bytes preserve missing/null and exact numeric precision. Python Resource.list can wrap singleton collections, removes self, overlays URI fields and constructs mutable descriptor-backed Resource objects.
- Go follows only same-origin, same-collection explicit links, retaining the fixed escaped path and detecting invalid continuation evidence. It does not guess node-ID marker fallback or expose arbitrary Resource routes.
- Python passes cluster_id=cluster to Proxy._list unchanged and formats it into /clusters/%(cluster_id)s/attrs/%(path)s, so only an ID string yields a valid URL even though the signature accepts a Cluster; Go resolves the cluster once.
- Direct ClusterAttr.list controls (paginated, limit/marker with Resource.id fallback, max_items, base_path, microversion, headers, node_id and attr_value Body filters) are not reachable through this proxy method because it drops **query; they are inherited-Resource features outside this operation, and Go accepts only canonical wire id/value fields.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
for attr in conn.clustering.collect_cluster_attrs("CLUSTER_ID", "$.status"):
    print(attr.node_id, attr.attr_value)
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
scope, err := service.ClusterAttributes.InCluster(ctx,
    resource.ID("CLUSTER_NAME_OR_ID"), "$.status")
if err != nil { return err }
for attr, err := range scope.List(ctx) {
    if err != nil { return err }
    fmt.Println(attr.NodeID, string(attr.Value))
}
```

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#collect-attributes-across-a-cluster)
- [16.0.0 router](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/router.py)
- [cluster controller](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/clusters.py)
- [WSGI Resource](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/common/wsgi.py)
- [conductor](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/conductor/service.py)
