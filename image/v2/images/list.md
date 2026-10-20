# List

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:image/v2/images.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.List`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.WithListQuery`

- Generated API.List serializes ListOpts through the pinned ToImageListQuery, omitting zero values, repeating tag, formatting created/updated date filters and silently dropping int64 SizeMin/SizeMax in BuildQueryString, then appends WithListQuery extension values; zero options send no query.
  Tests: `image/v2/images/list_contracts_test.go:TestNativeImageListQueryAndExtensions`.
- The lazy SDK stream follows native next links rebased onto the client ServiceURL path, stops on an empty images page even with next, surfaces the native EOF for a bodyless 204, yields earlier rows before a late native HTTP error, returns row decode errors and sends no further page after the caller stops.
  Tests: `image/v2/images/list_contracts_test.go:TestNativeImageListPagingStopsAndErrors`.

Python/Go differences:

- This declaration maps Gophercloud native List only; it is not Python images() descriptor/local-filter/location parity or Cloud list_images behavior.
- The SDK exposes the native pager as iter.Seq2 and keeps native query construction, including the int64 size filter omission; callers can send size_min/size_max with WithListQuery. Iteration errors are native and only option preparation errors receive List/images context.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
for image, err := range service.API.Images.List(ctx,
    images.WithListOptions(images.ListOpts{Status: images.ImageStatusActive, Limit: 50}),
    images.WithListQuery("size_min", "1073741824")) {
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
