# Waiting

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
cluster = conn.clustering.get_cluster("CLUSTER_ID")
cluster = conn.clustering.wait_for_status(
    cluster, "ACTIVE", failures=["ERROR"], interval=2, wait=180,
)
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
cluster, err := service.Clusters.WaitForStatus(ctx,
    resource.ID("CLUSTER_ID"), "ACTIVE",
    resource.WithTimeout(3*time.Minute))
if err != nil { return err }
fmt.Println(cluster.ID, cluster.Status, cluster.Header, cluster.Body)
```

```go
profile, err := service.Profiles.WaitForStatus(ctx,
    resource.ID("PROFILE_ID"), "renamed-profile",
    resource.WithStatusAttribute("name"),
    resource.WithFailureStates(),
    resource.WithTimeout(30*time.Second))
if err != nil { return err }
fmt.Println(profile.Name)
```

```python
conn.clustering.delete_node(node)
node = conn.clustering.wait_for_delete(node)
```

```go
submission, err := service.Nodes.Delete(ctx, resource.ID("NODE_ID"))
if err != nil { return err }
fmt.Println(submission.ActionID)
if err := service.Nodes.WaitForDelete(ctx, resource.ID("NODE_ID")); err != nil {
    return err
}
```

```go
err = service.Nodes.WaitForDelete(ctx, resource.ID("NODE_ID"),
    resource.WithTimeout(10*time.Minute),
    resource.WithUnlimitedWait(),
    resource.WithPollInterval(time.Second),
    resource.WithProgressCallback(func(progress int) error { fmt.Println(progress); return nil }))
if err != nil { return err }
```

## References

- [ClusterPolicy scope](../clusterpolicies/README.md)
- [ClusterAttributes](../clusterattributes/README.md)
- [Senlin proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L1350)
- [Resource wait helpers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591)
- [timeout iterator](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53)
- [Clustering Wait Contracts Test](../../../api/clustering_wait_contracts_test.go)
- [Clustering Cluster Policies Test](../../../api/clustering_cluster_policies_test.go)
