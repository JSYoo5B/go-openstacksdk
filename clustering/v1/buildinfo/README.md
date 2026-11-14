# Buildinfo

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
info, err := buildinfo.New(client).Get(ctx)
if err != nil {
    return err
}
if info.API != nil && info.API.Revision != nil {
    fmt.Println(*info.API.Revision)
}
```

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#build-information-build-info)
