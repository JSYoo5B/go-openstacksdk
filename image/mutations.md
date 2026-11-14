# Mutations

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

type MutationProof struct {
    Added       *image.ImageTagResult
    Removed     *image.ImageTagResult
    Deactivated *image.ImageActionResult
    Reactivated *image.ImageActionResult
}

func mutateImage(ctx context.Context, service *image.Service, ref resource.Ref, tag string) (proof MutationProof, err error) {
    option := request.Header("X-Request-Source", "example")
    proof.Added, err = service.AddTag(ctx, ref, tag, option)
    if err != nil {
        return proof, err
    }
    proof.Removed, err = service.RemoveTag(ctx, ref, tag, option)
    if err != nil {
        return proof, err
    }
    proof.Deactivated, err = service.DeactivateImage(ctx, ref, option)
    if err != nil {
        return proof, err
    }
    proof.Reactivated, err = service.ReactivateImage(ctx, ref, option)
    return proof, err
}
```

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
