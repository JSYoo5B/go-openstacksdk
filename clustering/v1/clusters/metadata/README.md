# Metadata

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

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
    }, request.Header("X-Request-Id", "metadata-change"))
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
