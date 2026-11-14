# Get

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api, err := conn.ImageV2(ctx)
if err != nil { return err }
```

```go
image, err := service.API.Images.Get(ctx, imageID)
if err != nil {
    var native gophercloud.ErrUnexpectedResponseCode
    if errors.As(err, &native) {
        fmt.Printf("HTTP %d, expected=%v, response bytes=%d\n",
            native.Actual, native.Expected, len(native.Body))
    }
    return err
}
fmt.Println(image.ID, image.Status, image.SizeBytes, image.Properties, image.OpenStackImageImportMethods)
```

## References

- [Image Records](../../image-records.md)
- [Image Record Cloud](../../image-record-cloud.md)
- [upstream Get](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L228-L233)
- [Extract](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/results.go#L161-L179)
- [Image.UnmarshalJSON](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/results.go#L101-L154)
- [Delete](delete.md)
