# Containers

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListContainers(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ObjectStorage(ctx)
	if err != nil {
		return err
	}
	for container, err := range service.Containers.Resources.List(ctx,
		resource.WithPageSize(100),
		resource.WithMaxItems(20),
		resource.WithPaginated(false),
	) {
		if err != nil {
			return err
		}
		fmt.Println(container.Name, container.Count, container.Bytes)
	}
	return nil
}
```

```python
container = conn.object_store.get_container_metadata("books")
print(container.metadata.get("book"))
refreshed = conn.object_store.set_container_metadata(
    container, book="Moby Dick", note="literal %2F 雪"
)
conn.object_store.delete_container_metadata(refreshed, ["note"])
```

```go
package main

import (
    "context"
    "fmt"

    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func changeContainerMetadata(ctx context.Context, api *containers.API, name string) error {
    getOptions := []containers.GetMetadataOption{
        containers.WithGetMetadataOpts(containers.GetMetadataOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        containers.WithGetMetadataHeaders(map[string]string{"X-Trace-Label": "metadata"}),
        request.Header("X-Request-Label", "container-metadata"),
        request.Custom(func(c *containers.GetMetadataOpts) error { v := false; c.Newest = &v; return nil }),
    }
    before, err := api.GetMetadata(ctx, name, getOptions...)
    if before != nil { fmt.Println(before.StatusCode) }
    if err != nil { return err }
    if before.Metadata.ObjectCount != nil { fmt.Println(*before.Metadata.ObjectCount) }

    mutationOptions := []containers.MetadataOption{
        containers.WithMetadataOpts(containers.MetadataOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        containers.WithMetadataHeaders(map[string]string{"X-Trace-Label": "metadata"}),
        request.Header("X-Request-Label", "metadata-mutation"),
    }
    acknowledgement, err := api.SetMetadata(ctx, name, map[string]string{
        "book": "Moby Dick", "note": "literal %2F 雪",
    }, mutationOptions...)
    if acknowledgement != nil { fmt.Println(acknowledgement.StatusCode) }
    if err != nil { return err }

    refreshOptions := append([]containers.GetMetadataOption(nil), getOptions...)
    refreshOptions = append(refreshOptions, request.Custom(func(c *containers.GetMetadataOpts) error { c.Newest = nil; return nil }))
    refreshed, err := api.GetMetadata(ctx, name, refreshOptions...)
    if err != nil { return err }
    fmt.Println(refreshed.Metadata.Values["book"])

    removed, err := api.DeleteMetadata(ctx, name, []string{"note"}, mutationOptions...)
    if removed != nil { fmt.Println(removed.StatusCode) }
    return err
}

func main() {}
```

```python
created = conn.object_store.create_container(
    "books-lifecycle", metadata={"owner": "ops", "note": "literal %2F 雪"}
)
conn.object_store.delete_container(created, ignore_missing=False)
conn.object_store.delete_container("books-lifecycle")
```

```go
package main

import (
    "context"
    "fmt"

    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func containerLifecycle(ctx context.Context, api *containers.API, name string) error {
    createOptions := []containers.CreateContainerOption{
        containers.WithCreateContainerOpts(containers.CreateContainerOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
            Metadata: map[string]string{"owner": "initial"},
        }),
        containers.WithCreateContainerHeaders(map[string]string{"X-Trace-Label": "container-create"}),
        request.Header("X-Request-Label", "create"),
        containers.WithCreateContainerMetadata(map[string]string{
            "owner": "ops", "note": "literal %2F 雪",
        }),
    }
    created, err := api.CreateContainer(ctx, name, createOptions...)
    if created != nil { fmt.Println(created.StatusCode) }
    if err != nil { return err }

    deleteOptions := []containers.DeleteContainerOption{
        containers.WithDeleteContainerOpts(containers.DeleteContainerOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        containers.WithDeleteContainerHeaders(map[string]string{"X-Trace-Label": "container-delete"}),
        request.Header("X-Request-Label", "delete"),
        request.IgnoreMissing(false),
    }
    deleted, err := api.DeleteContainer(ctx, name, deleteOptions...)
    if deleted != nil { fmt.Println(deleted.StatusCode, deleted.IgnoredMissing) }
    if err != nil { return err }

    quietOptions := append([]containers.DeleteContainerOption(nil), deleteOptions...)
    quietOptions = append(quietOptions, request.IgnoreMissing(true))
    missing, err := api.DeleteContainer(ctx, name, quietOptions...)
    if missing != nil { fmt.Println(missing.StatusCode, missing.IgnoredMissing) }
    return err
}

func main() {}
```

## References

- [Listing](../../../docs/listing.md)
- [Objects](../objects/README.md)
- [Native Scope List Controls Test](../../../api/native_scope_list_controls_test.go)
- [Swift Listing Contracts Test](../../../api/swift_listing_contracts_test.go)
- [pinned Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L189)
- [ Base](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L68)
- [Storage Container Services](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-container-services.inc#L240)
- [Base](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/controllers/base.py#L1953)
- [Server](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/container/server.py#L59)
- [Metadata Contracts Test](metadata_contracts_test.go)
- [Storage Container Services](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-container-services.inc#L117)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L154)
- [Db](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/db.py#L1015)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1519)
- [Lifecycle Core Test](lifecycle_core_test.go)
- [Lifecycle Options Test](lifecycle_options_test.go)
- [Lifecycle Contracts Test](lifecycle_contracts_test.go)
- [Connection Objectstorage Container Lifecycle Test](../../../connectiontest/connection_objectstorage_container_lifecycle_test.go)
- [Swift Container Lifecycle Test](../../../internal/cmd/sdkgen/swift_container_lifecycle_test.go)
