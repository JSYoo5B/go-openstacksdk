# Finding

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

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
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func FindClusterStrict(ctx context.Context, conn *sdk.Connection) error {
    service, err := conn.Clustering(ctx)
    if err != nil { return err }
    value, err := service.Clusters.FindIdentity(ctx, "workers",
        request.IgnoreMissing(false),
        clusters.WithFindFallback(resource.FindFallbackNotFoundOnly),
        request.Header("X-Vendor-Trace", "lookup"),
        request.Microversion("1.7"))
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
