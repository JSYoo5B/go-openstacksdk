# Create

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
visibility := images.ImageVisibilityPrivate
image, err := service.API.Images.Create(ctx, images.CreateOpts{
    Name: "ubuntu-24.04", DiskFormat: "qcow2", ContainerFormat: "bare",
    Visibility: &visibility, Tags: []string{"base"},
    Properties: map[string]string{"os_distro": "ubuntu"},
}, request.Field("hw_qemu_guest_agent", "yes"))
if err != nil {
    return err
}
fmt.Println(image.ID, image.Status)
```

## References

- [Image Record Create](../../image-record-create.md)
- [Image Record Cloud](../../image-record-cloud.md)
- [ToImageCreateMap](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L195-L207)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L209-L219)
- [Get](get.md)
- [Delete](delete.md)
