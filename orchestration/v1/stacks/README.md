# Stacks

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
scope, err := service.Stacks.InStack(ctx, resource.Name("app"))
if err != nil { return err }
identity := scope.Identity()
fmt.Println(identity.Name, identity.ID)

stack, err := scope.Get(ctx, stacks.WithResolveOutputs(false))
if err != nil { return err }
fmt.Println(stack.Name, stack.ID, stack.Status, stack.Detailed)
```

```go
scope, err := service.Stacks.ForStack(stacks.StackIdentity{
    Name: "app",
    ID: "stack-uuid",
})
if err != nil { return err }
if err := scope.UpdatePatch(ctx, stacks.UpdateOpts{
    Parameters: map[string]any{"size": "large"},
    Tags: []string{"ops", "test"},
}); err != nil { return err }
if _, err := scope.Wait(ctx, "UPDATE_COMPLETE"); err != nil { return err }
```

```go
if err := scope.Delete(ctx); err != nil { return err }
if err := scope.WaitDeleted(ctx); err != nil { return err }
```

## References

- [..](../README.md)
- [Heat Stacks Contracts Test](../../../api/heat_stacks_contracts_test.go)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)

## Native API calls

The generated `API` methods are the unchanged native entry points and take the stack name and ID, or a single identity string, as separate path segments. They are covered by `api/heat_native_stacks_test.go`.

```go
api := stacks.New(client)

// Name-or-ID lookup: one GET /stacks/{identity}; Heat's redirect is followed.
stack, err := api.Find(ctx, "app")

// Single-page list with every typed filter; booleans that are false are omitted.
for stack, err := range api.List(ctx, stacks.WithListOptions(stacks.ListOpts{Name: "app", AllTenants: true, Tags: "a,b"})) {
    if err != nil { panic(err) }
    fmt.Println(stack.ID)
}

// PUT requires a template; PATCH does not. Update accepts HTTP 201/202, UpdatePatch accepts 200/202/204.
err = api.Update(ctx, "app", id, stacks.UpdateOpts{TemplateOpts: &stacks.Template{TE: stacks.TE{Bin: tpl}}})
err = api.UpdatePatch(ctx, "app", id, stacks.UpdateOpts{Parameters: map[string]any{"size": 2}})

// Delete accepts HTTP 202/204 and returns every other status as an error. Abandon requires HTTP 200.
err = api.Delete(ctx, "app", id)
abandoned, err := api.Abandon(ctx, "app", id)
```

Limitations: an empty non-nil `Tags` slice is sent as `"tags": ""`, while `nil` tags and an empty `Parameters` map are omitted. Native `Delete` does not ignore 404. A canceled context fails before any request. Mutating calls are never retried.
