# Endpointgroups

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
group = conn.identity.create_endpoint_group(name="web", filters={"interface": "public"})
conn.identity.associate_project_with_endpoint_group(group, project_id)
for endpoint in conn.identity.endpoint_group_endpoints(group):
    print(endpoint.url)
conn.identity.delete_endpoint_group(group)
```

```go
api := endpointgroups.New(identityClient)
group, err := api.Create(ctx,
    request.Field("name", "web"),
    request.Field("filters", map[string]string{"interface": "public"}))
if err != nil {
    return err
}
id := group.Text("id")
if err := api.AssociateProject(ctx, id, projectID); err != nil {
    return err
}
for endpoint, err := range api.ListEndpoints(ctx, id) {
    if err != nil {
        return err
    }
    fmt.Println(endpoint.Text("url"))
}
return api.Delete(ctx, id)
```

## References

- [`projectendpoints`](../projectendpoints)
