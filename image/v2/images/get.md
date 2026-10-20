# Get

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:image/v2/images.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.RawClient`

- Generated API.Get retains the exact RawClient and sends one GET to native ServiceURL(ResourceBase or Endpoint, images, raw id) with a nil body, default Accept, service MoreHeaders, microversion and live token. Empty IDs, slashes and query delimiters are forwarded without validation or escaping.
  Tests: `image/v2/images/get_contracts_test.go:TestNativeImageGetRoutesHeadersAndNativeProjection`.
- A 200 response uses the pinned Extract header merge and Image.UnmarshalJSON: import methods and store IDs are trimmed and comma split without empty fields, numeric size becomes int64 and null size zero, undeclared keys except self/size/header keys become Properties with the body properties string retained, and times decode to time.Time.
  Tests: `image/v2/images/get_contracts_test.go:TestNativeImageGetRoutesHeadersAndNativeProjection`.
- Only 200 is accepted; other statuses keep the native ErrUnexpectedResponseCode with Expected [200] as the direct cause of a Get/images OperationError without an owned receipt. Decode errors keep native partial Extract results, and caller cancellation stays inspectable.
  Tests: `image/v2/images/get_contracts_test.go:TestNativeImageGetStrictStatusAndDecodeErrors`.

Python/Go differences:

- This declaration maps Gophercloud native Get only; it is not Python get_image parity, owned GetImageRecord descriptor/location projection or Cloud get_image_by_id behavior.
- The SDK adds only request.Wrap operation context; native URL construction, status policy, retries, reauthentication and partial Extract results are unchanged.

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
