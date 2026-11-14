# Clusterpolicies

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
binding = conn.clustering.get_cluster_policy("POLICY_NAME_OR_ID", "CLUSTER_NAME_OR_ID")
for binding in conn.clustering.cluster_policies("CLUSTER_NAME_OR_ID", is_enabled=False):
    print(binding.id, binding.policy_id)
```

```go
api := clusterpolicies.New(client)
scope, err := api.InCluster(ctx, resource.ID("CLUSTER_NAME_OR_ID"))
if err != nil { return err }
binding, err := scope.Get(ctx, "POLICY_NAME_OR_ID")
if err != nil { return err }
fmt.Println(binding.ID, binding.PolicyID, binding.ClusterID, binding.URIClusterID)

for binding, err := range scope.List(ctx, request.Custom(func(c *request.Config[clusterpolicies.ListOpts]) error { v := false; c.Options.Enabled = &v; return nil })) {
    if err != nil { return err }
    fmt.Println(binding.PolicyName, binding.IsEnabled)
}
```

```go
bindings, err := scope.All(ctx,
    request.Custom(func(c *request.Config[clusterpolicies.ListOpts]) error { v := false; c.Options.Enabled = &v; return nil }),
    request.Custom(func(c *request.Config[clusterpolicies.ListOpts]) error { c.Options.PolicyName = "POLICY_NAME"; return nil }),
    request.Custom(func(c *request.Config[clusterpolicies.ListOpts]) error { c.Options.PolicyType = "senlin.policy.scaling-1.0"; return nil }))
if err != nil { return err }
fmt.Println(len(bindings))
```

```go
err = scope.WaitForDelete(ctx, resource.ID("POLICY_NAME_OR_ID"),
    resource.WithTimeout(30*time.Second),
    resource.WithPollInterval(time.Second))
if err != nil { return err }
```

```go
listingAPI := clusterpolicies.New(client)
listingScope, err := listingAPI.InCluster(ctx, resource.ID("CLUSTER_ID"))
if err != nil { return err }
values, err := listingScope.All(ctx, request.MaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingScope.List(ctx, request.Paginated(false)) {
    if err != nil { return err }
    fmt.Println(value.PolicyID)
}
```

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func FilterClusterPolicies(ctx context.Context, conn *sdk.Connection, cluster resource.Ref) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	scope, err := service.ClusterPolicies.InCluster(ctx, cluster)
	if err != nil {
		return err
	}
	for binding, err := range scope.List(ctx,
		clusterpolicies.WithListFilter("is_enabled", false),
		clusterpolicies.WithListFilter("data", map[string]any{"team": "infra"}),
		request.MaxItems(50), request.Paginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(binding.ID, binding.PolicyID, binding.URIClusterID)
	}
	return nil
}
```

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#cluster-policies)
- [16.0.0 cluster policy controller](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/cluster_policies.py)
- [16.0.0 router](https://github.com/openstack-archive/senlin/blob/16.0.0/senlin/api/openstack/v1/router.py)
- [Listing](../listing/README.md)
- [Clustering Catalog List Controls Test](../../../api/clustering_catalog_list_controls_test.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)
