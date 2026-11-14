# Images

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package examples

import (
    "context"
    "net/url"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func getImageExample(ctx context.Context, svc *image.Service, imageID string) (*image.ImageInfo, error) {
    return svc.GetImage(ctx, resource.ID(imageID),
        image.WithGetImageOpts(image.GetImageOpts{}),
        image.WithGetImageHeaders(map[string]string{"X-Example": "image-get"}),
        request.Header("X-Request-Id", "image-get-example"),
    )
}

func classicImagesExample(ctx context.Context, svc *image.Service, imageName, ownerID string) ([]*image.ImageInfo, error) {
    options := []image.ListImagesOption{
        image.WithListImagesOpts(image.ListImagesOpts{MaxItems: 0}),
        image.WithListImagesHeaders(map[string]string{"X-Example": "image-list"}),
        request.Header("X-Request-Id", "image-list-example"),
        request.Limit(2),
        request.Marker(""),
        request.Custom(func(c *image.ListImagesOpts) error { v := imageName; c.Name = &v; return nil }),
        request.Custom(func(c *image.ListImagesOpts) error { v := "private"; c.Visibility = &v; return nil }),
        request.Custom(func(c *image.ListImagesOpts) error { v := "all"; c.MemberStatus = &v; return nil }),
        request.Custom(func(c *image.ListImagesOpts) error { v := ownerID; c.Owner = &v; return nil }),
        request.Custom(func(c *image.ListImagesOpts) error { v := "active"; c.Status = &v; return nil }),
        request.Custom(func(c *image.ListImagesOpts) error { v := int64(0); c.SizeMin = &v; return nil }),
        request.Custom(func(c *image.ListImagesOpts) error { v := int64(1 << 40); c.SizeMax = &v; return nil }),
        request.Custom(func(c *image.ListImagesOpts) error { v := false; c.Protected = &v; return nil }),
        request.Custom(func(c *image.ListImagesOpts) error { v := false; c.Hidden = &v; return nil }),
        image.WithListImagesSortKeys("name", "created_at"),
        image.WithListImagesSortDirs("asc", "desc"),
        image.WithListImagesTags("ready", "approved"),
        request.Custom(func(c *image.ListImagesOpts) error { v := "gte:2020-01-01T00:00:00Z"; c.CreatedAt = &v; return nil }),
        request.Custom(func(c *image.ListImagesOpts) error { v := "lte:2030-01-01T00:00:00Z"; c.UpdatedAt = &v; return nil }),
        request.Custom(func(c *image.ListImagesOpts) error { v := "bare"; c.ContainerFormat = &v; return nil }),
        request.Custom(func(c *image.ListImagesOpts) error { v := "qcow2"; c.DiskFormat = &v; return nil }),
        image.WithListImagesFilters(url.Values{
            "os_distro": {"ubuntu"}, "unused": {"remove-me"},
        }),
        image.WithListImagesFilter("architecture", "x86_64"),
        image.WithListImagesFilter("unused"),
        request.MaxItems(20),
        request.Custom(func(c *image.ListImagesOpts) error { c.SinglePage = false; return nil }),
    }
    rows := make([]*image.ImageInfo, 0)
    for value, err := range svc.ListImages(ctx, options...) {
        if err != nil {
            return nil, err
        }
        rows = append(rows, value)
    }
    return rows, nil
}

func modernImagesExample(ctx context.Context, svc *image.Service, imageID string) ([]*image.ImageInfo, error) {
    return svc.AllImages(ctx,
        request.Custom(func(c *image.ListImagesOpts) error { v := imageID; c.ID = &v; return nil }),
        request.Custom(func(c *image.ListImagesOpts) error { v := "name:asc,created_at:desc"; c.Sort = &v; return nil }),
        request.MaxItems(20),
    )
}
```

```python
def images_example(conn, image_id, image_name, owner_id):
    image = conn.image.get_image(image_id)
    rows = list(conn.image.images(
        name=image_name, owner=owner_id, tag=["ready", "approved"],
        sort_key=["name", "created_at"], sort_dir=["asc", "desc"], limit=2,
    ))
    return image, rows
```

## References

- [generated/native Image API](v2/images/README.md)
- [Images Images V2](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-images-v2.inc#L243-L308)
- [Images Images V2](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-images-v2.inc#L353-L550)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L517-L585)
- [Config](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/common/config.py#L282-L329)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1034-L1055)
- [Locations](locations.md)
- [serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1756-L1827)
- [Images](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L1855-L1871)
- [controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L564-L574)
- [Images Core Test](images_core_test.go)
- [Images Options Test](images_options_test.go)
- [Images Contracts Test](images_contracts_test.go)
- [Connection Image Read Test](../connectiontest/connection_image_read_test.go)
- [Glance Image Read Test](../internal/cmd/sdkgen/glance_image_read_test.go)
