# Objects

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
scope, err := service.Objects.InContainer(ctx, resource.ID("assets"))
if err != nil { return err }
object, err := scope.Get(ctx, "images/logo ?.png")
if err != nil { return err }
fmt.Println(object.Name, object.Bytes, object.ContentType, object.Metadata)

for object, err := range scope.List(ctx, resource.WithPageSize(100)) {
    if err != nil { return err }
    fmt.Println(object.Name, object.Hash)
}

header, err := scope.Create(ctx, "docs/readme.txt", objects.CreateOpts{
    Content: strings.NewReader("hello"),
    Metadata: map[string]string{"Owner": "sdk"},
})
if err != nil { return err }
_ = header

download, err := scope.Download(ctx, resource.ID("docs/readme.txt"))
if err != nil { return err }
defer download.Close()
_, err = io.Copy(destination, download)
if err != nil { return err }
```

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func ListObjects(ctx context.Context, conn *sdk.Connection, containerName string) error {
	service, err := conn.ObjectStorage(ctx)
	if err != nil {
		return err
	}
	scope, err := service.Objects.InContainer(ctx, resource.ID(containerName))
	if err != nil {
		return err
	}
	for object, err := range scope.List(ctx,
		resource.WithPageSize(100),
		resource.WithMaxItems(20),
		resource.WithPaginated(false),
	) {
		if err != nil {
			return err
		}
		fmt.Println(object.Container, object.Name, object.Subdir, object.Bytes)
	}
	return nil
}
```

```go
package examples

import (
	"context"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func ChangeObjectMetadata(ctx context.Context, conn *sdk.Connection, newest bool) error {
	service, err := conn.ObjectStorage(ctx)
	if err != nil {
		return err
	}
	container, key := "assets", "docs/readme ?.txt"
	getOptions := []objects.GetMetadataOption{
		objects.WithGetMetadataOpts(objects.GetMetadataOpts{
			Headers: map[string]string{"X-Trans-Id-Extra": "sdk-read"},
		}),
		request.Header("X-Example", "read"),
		objects.WithGetMetadataHeaders(map[string]string{"X-Example-Trace": "metadata"}),
	}
	if newest {
		getOptions = append(getOptions, request.Custom(func(c *objects.GetMetadataOpts) error { v := true; c.Newest = &v; return nil }))
	} else {
		getOptions = append(getOptions, request.Custom(func(c *objects.GetMetadataOpts) error { c.Newest = nil; return nil }))
	}
	before, err := service.Objects.GetMetadata(ctx, container, key, getOptions...)
	if err != nil {
		if before != nil {
			fmt.Println("HEAD", before.StatusCode)
		}
		return err
	}
	if before.Metadata.ContentLength != nil {
		fmt.Println("object bytes", *before.Metadata.ContentLength)
	}
	mutationOptions := []objects.MetadataOption{
		objects.WithMetadataOpts(objects.MetadataOpts{
			Headers: map[string]string{"X-Trans-Id-Extra": "sdk-write"},
		}),
		request.Header("Content-Disposition", "attachment"),
		objects.WithMetadataHeaders(map[string]string{"Cache-Control": "private"}),
	}
	change, err := service.Objects.SetMetadata(ctx, container, key,
		map[string]string{"owner": "sdk", "empty": ""}, mutationOptions...)
	printObjectMetadataPhases(change)
	if err != nil {
		return err
	}
	change, err = service.Objects.DeleteMetadata(ctx, container, key,
		[]string{"owner"}, mutationOptions...)
	printObjectMetadataPhases(change)
	if err != nil {
		return err
	}
	refreshed, err := service.Objects.GetMetadata(ctx, container, key, getOptions...)
	if err != nil {
		return err
	}
	fmt.Println("stored empty value", refreshed.Metadata.Values["empty"])
	return nil
}

func printObjectMetadataPhases(change *objects.MetadataChangeResult) {
	if change == nil {
		return
	}
	if change.Before != nil {
		fmt.Println("before HEAD", change.Before.StatusCode)
	}
	if change.Acknowledgement != nil {
		fmt.Println("POST", change.Acknowledgement.StatusCode)
	}
}
```

```python
obj = conn.object_store.get_object_metadata(
    "docs/readme ?.txt", container="assets"
)
same = conn.object_store.set_object_metadata(obj, owner="sdk", empty="")
assert same is obj
conn.object_store.delete_object_metadata(obj, keys=["owner"])
refreshed = conn.object_store.get_object_metadata(obj)
print(refreshed.metadata)
```

## Limitations

- Reads, uploads and waits report what one request observed. Swift replicas, eventual consistency, proxy caching and middleware decide when other clients see a change; a `HEAD` 404 or a matching metadata value is not a lock or a cluster-wide guarantee.
- Segmented uploads cannot be published atomically or rolled back. Unique segment prefixes and the returned cleanup evidence show what was written.
- `StreamObject` returns an `io.ReadCloser`: close it, and read it from one goroutine at a time.
- Text reads decode only UTF-8, a byte-order mark or a supported declared charset. Undeclared non-UTF-8 text returns `ErrUnsupported` with the original bytes; see [no charset guessing](../../../docs/sdk-support-ledger.md#deliberate-runtime-differences).

## References

- [`copy_object`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L531-L533)
- [Test Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/tests/unit/object_store/v1/test_proxy.py#L251-L252)
- [Listing](../../../docs/listing.md)
- [Containers](../containers/README.md)
- [Native Scope List Controls Test](../../../api/native_scope_list_controls_test.go)
- [Obj](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L222)
- [Storage Object Services](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-object-services.inc#L631)
- [Server](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/obj/server.py#L665)
- [Metadata Contracts Test](metadata_contracts_test.go)
- [atomic header projection](metadata_contracts_test.go)
