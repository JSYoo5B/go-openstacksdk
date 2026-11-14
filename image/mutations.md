# Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `example_mutations_test.go` in [example_mutations_test.go](example_mutations_test.go), compiled by `go test`.

```python
def mutate_image(conn, image_id, tag):
    conn.image.add_tag(image_id, tag)
    conn.image.remove_tag(image_id, tag)
    conn.image.deactivate_image(image_id)
    conn.image.reactivate_image(image_id)
```

## References

- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#add-image-tag)
- [tag controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_tags.py#L47-L101)
- [Index](https://docs.openstack.org/api-ref/image/v2/index.html#deactivate-image)
- [Image Actions](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/image_actions.py#L47-L111)
- [Mutation Core Test](mutation_core_test.go)
- [Mutation Contracts Test](mutation_contracts_test.go)
- [Mutation Options Test](mutation_options_test.go)
- [Connection Image Mutations Test](../connectiontest/connection_image_mutations_test.go)
- [Glance Mutations Test](../internal/cmd/sdkgen/glance_mutations_test.go)
