# Clusterattributes

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

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
