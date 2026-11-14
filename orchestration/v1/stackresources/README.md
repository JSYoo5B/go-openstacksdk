# Stackresources

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
scope, err := service.StackResources.InStack(ctx, resource.Name("app"))
if err != nil { return err }
item, err := scope.Find(ctx, resource.Name("web"))
if err != nil { return err }
fmt.Println(item.Name, item.LogicalID, item.PhysicalID, item.Owner)

id, err := scope.ResolveID(ctx, resource.Name("web"))
if err != nil { return err }
detail, err := scope.Get(ctx, id)
if err != nil { return err }
fmt.Println(detail.Attributes, detail.Detailed)
```

```go
scope, err := service.StackResources.ForStack(stacks.StackIdentity{
    Name: "app", ID: "stack-uuid",
})
if err != nil { return err }
items, err := scope.All(ctx, stackresources.WithNestedDepth(1),
    stackresources.WithDetails(true))
if err != nil { return err }
for _, item := range items {
    identity, err := item.Identity()
    if err != nil { return err }
    child, err := service.StackResources.ForResource(identity)
    if err != nil { return err }
    metadata, err := child.Metadata(ctx)
    if err != nil { return err }
    fmt.Println(identity.Stack.Name, identity.Stack.ID, identity.Name, metadata)
}
```

```go
node, err := service.StackResources.ForResource(stackresources.ResourceIdentity{
    Stack: stacks.StackIdentity{Name: "app", ID: "stack-uuid"},
    Name: "web",
})
if err != nil { return err }
if err := node.MarkUnhealthy(ctx, false,
    stackresources.WithHealthReason("recovered")); err != nil { return err }
if _, err := node.Wait(ctx, "CHECK_COMPLETE"); err != nil { return err }
```

## References

- [stack resolver](../stacks/README.md)
- [..](../README.md)
- [Heat Stackresources Contracts Test](../../../api/heat_stackresources_contracts_test.go)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)
- [Heat 2026.1 ResourceController](https://docs.openstack.org/heat/2026.1/_modules/heat/api/openstack/v1/resources.html)
- [Heat resource formatter](https://docs.openstack.org/heat/2026.1/_modules/heat/engine/api.html#format_stack_resource)
- [Heat resource API reference](https://docs.openstack.org/api-ref/orchestration/v1/#stack-resources)
- [Python Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/orchestration/v1/resource.py)
- [Proxy.resources](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/orchestration/v1/_proxy.py#L370-L403)

## Native API calls

The generated `API` methods address Heat paths directly and are covered by `api/heat_native_stackresources_test.go`.

```go
api := stackresources.New(client)

// GET /stacks/{name}/resources; the redirect to the canonical stack is followed.
resources, err := api.Find(ctx, "app")

// Global resource-type operations (not tied to a stack).
for name, err := range api.ListTypes(ctx) { if err != nil { panic(err) }; fmt.Println(*name) }
schema, err := api.Schema(ctx, "OS::Nova::Server")
template, err := api.Template(ctx, "OS::Nova::Server") // indented JSON bytes

// The path segment is sent unchanged, so a physical resource ID can be targeted directly.
err = api.MarkUnhealthy(ctx, "app", id, physicalID, stackresources.MarkUnhealthyOpts{MarkUnhealthy: true, ResourceStatusReason: "bad disk"})
```

Limitations: `ListTypes` is one non-paged request that preserves the server order. `MarkUnhealthy` accepts HTTP 200/202/204, sends `mark_unhealthy: false` when unset and omits an empty reason. Extensions cannot replace the typed `mark_unhealthy` field. Unlike the scoped `StackScope.MarkUnhealthy`, the native call performs no GET preflight or owner check.
