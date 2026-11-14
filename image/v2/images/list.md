# List

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
for image, err := range service.API.Images.List(ctx,
    images.WithListOptions(images.ListOpts{Status: images.ImageStatusActive, Limit: 50}),
    request.Query("size_min", "1073741824")) {
    if err != nil {
        return err
    }
    fmt.Println(image.ID, image.Name, image.SizeBytes)
}
```

## References

- [Get](get.md)
- [Image Records](../../image-records.md)
- [Image Record Cloud](../../image-record-cloud.md)
- [ToImageListQuery](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L98-L123)
- [BuildQueryString](https://github.com/gophercloud/gophercloud/blob/v2.15.0/params.go#L418-L421)
- [nextPageURL](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/urls.go#L43-L65)
