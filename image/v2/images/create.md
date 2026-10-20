# Create

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:image/v2/images.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithCreateField`

- Generated API.Create serializes CreateOpts with the pinned BuildRequestBody, omitting zero values while sending explicit pointer false, then copies Properties over declared keys and appends non-colliding WithCreateField extension JSON with exact raw bytes before one POST to native images with JSON headers and service MoreHeaders.
  Tests: `image/v2/images/create_contracts_test.go:TestNativeImageCreateBodyPropertiesExtensionsAndProjection`.
- A 201 response uses the shared Extract header merge and Image.UnmarshalJSON projection. Missing required name, nil options and extensions colliding with core or existing body keys fail before HTTP; other statuses keep native ErrUnexpectedResponseCode with Expected [201] under Create/images context without an owned receipt.
  Tests: `image/v2/images/create_contracts_test.go:TestNativeImageCreateBodyPropertiesExtensionsAndProjection`, `image/v2/images/create_contracts_test.go:TestNativeImageCreatePreflightAndStrictStatus`.

Python/Go differences:

- This declaration maps Gophercloud native metadata Create only; it is not Python create_image parity, which CreateImageRecord and CreateCloudImageRecord cover.
- WithCreateField is a Go extension that rejects collisions instead of overwriting; native body construction, OkCodes and response projection are unchanged.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
visibility := images.ImageVisibilityPrivate
image, err := service.API.Images.Create(ctx, images.CreateOpts{
    Name: "ubuntu-24.04", DiskFormat: "qcow2", ContainerFormat: "bare",
    Visibility: &visibility, Tags: []string{"base"},
    Properties: map[string]string{"os_distro": "ubuntu"},
}, images.WithCreateField("hw_qemu_guest_agent", "yes"))
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
