# Update

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/image/v2/images"
    "github.com/gophercloud/gophercloud/v2"
)

func main() {
    cloud := flag.String("cloud", "", "required clouds.yaml cloud name")
    imageID := flag.String("image-id", "", "required native image ID")
    name := flag.String("name", "release-image", "new image name")
    tag := flag.String("tag", "reviewed", "one tag replacing the whole tag set")
    flag.Parse()

    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *imageID, *name, *tag); err != nil {
        var native gophercloud.ErrUnexpectedResponseCode
        if errors.As(err, &native) {
            fmt.Fprintf(os.Stderr, "HTTP %d, response bytes=%d\n", native.Actual, len(native.Body))
        }
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, imageID, name, tag string) error {
    if cloud == "" || imageID == "" {
        return fmt.Errorf("-cloud and -image-id are required")
    }
    conn, err := openstack.Connect(ctx, openstack.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.Image(ctx)
    if err != nil { return err }

    patches := images.UpdateOpts{
        images.ReplaceImageName{NewName: name},
        images.UpdateVisibility{Visibility: images.ImageVisibilityPrivate},
        images.ReplaceImageHidden{NewHidden: false},
        images.ReplaceImageProtected{NewProtected: false},
        images.ReplaceImageMinDisk{NewMinDisk: 0},
        images.ReplaceImageMinRam{NewMinRam: 0},
        images.ReplaceImageTags{NewTags: []string{tag}},
        images.UpdateImageProperty{Op: images.AddOp, Name: "sdk_demo_build", Value: "draft"},
        images.UpdateImageProperty{Op: images.ReplaceOp, Name: "sdk_demo_build", Value: "ready"},
        images.UpdateImageProperty{Op: images.AddOp, Name: "sdk_demo_temporary", Value: "temporary"},
        images.UpdateImageProperty{Op: images.RemoveOp, Name: "sdk_demo_temporary"},
    }
    // WithUpdateOptions replaces the base nil array with these concrete patches.
    value, operationErr := service.API.Images.Update(ctx, imageID, nil, images.WithUpdateOptions(patches))
    output := map[string]any{
        "image": value,
        "partial": value != nil && operationErr != nil,
    }
    if operationErr != nil { output["error"] = operationErr.Error() }
    body, marshalErr := json.MarshalIndent(output, "", "  ")
    if marshalErr != nil { return errors.Join(operationErr, marshalErr) }
    fmt.Println(string(body))
    return operationErr
}
```

```python
import openstack

conn = openstack.connect(cloud="dev")
updated = conn.image.update_image("image-id", name="release-image", is_protected=False)
same = conn.image.update_image(updated, name=updated.name)
assert same is updated
```

## References

- [Update](../../update.md)
- [Image Records](../../image-records.md)
- [Install](../../../docs/install.md)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L260-L418)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L236-L248)
- [Results](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/results.go#L161-L183)
- [Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1056-L1068)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1875-L1937)
- [owned UpdateImageRecord](../../image-record-update.md)
- [Source pyproject](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/pyproject.toml#L15-L20)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L81-L116)
- [API policy](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/policy.py#L203-L229)
- [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L657-L718)
- [Image](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/policies/image.py#L160-L171)
- [Contracts Test](../../../api/contracts_test.go)
- [Update Contracts Test](../../update_contracts_test.go)
- [Mutation Contracts Test](../../mutation_contracts_test.go)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)
