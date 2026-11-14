# Delete

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api, err := conn.ImageV2(ctx)
if err != nil { return err }
```

```go
err := service.API.Images.Delete(ctx, imageID)
if err != nil {
    var native gophercloud.ErrUnexpectedResponseCode
    if errors.As(err, &native) {
        fmt.Printf("HTTP %d, expected=%v, response bytes=%d\n",
            native.Actual, native.Expected, len(native.Body))
    }
    return err
}
```

## References

- [Image Record Delete](../../image-record-delete.md)
- [Install](../../../docs/install.md)
- [upstream Delete](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L222-L226)
- [OkCodes](https://github.com/gophercloud/gophercloud/blob/v2.15.0/provider_client.go#L582-L597)
- [native accepted drain](https://github.com/gophercloud/gophercloud/blob/v2.15.0/provider_client.go#L570-L580)
- [Provider Client](https://github.com/gophercloud/gophercloud/blob/v2.15.0/provider_client.go#L458-L535)
- [Delete Contracts Test](delete_contracts_test.go)
