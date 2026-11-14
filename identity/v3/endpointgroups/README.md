# Endpointgroups

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
group = conn.identity.create_endpoint_group(name="web", filters={"interface": "public"})
conn.identity.associate_project_with_endpoint_group(group, project_id)
for endpoint in conn.identity.endpoint_group_endpoints(group):
    print(endpoint.url)
conn.identity.delete_endpoint_group(group)
```

See `ExampleAPI_Create` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.

## References

- [`projectendpoints`](../projectendpoints)
