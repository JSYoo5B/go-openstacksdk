# Metadata Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := objects.New(swiftClient)
change, err := api.SetMetadataFromFresh(ctx, "assets", "folder/file ?.txt",
    map[string]any{"owner": "ops", "empty": "", "is_content_type_detected": true},
    objects.WithMetadataHeaders(map[string]string{"X-Trans-Id-Extra": "metadata"}),
)
if err != nil {
    return err
}
_ = change.Acknowledgement

removed, err := api.DeleteMetadataAndRefresh(ctx, "assets", "folder/file ?.txt", []string{"owner"})
if err != nil {
    return err
}
if !removed.NoOp {
    observed := removed.Refreshed.Metadata.Values
    _ = observed
}
```

## References

- [Python Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L563-L640)
- [Object.set_metadata](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L222-L258)
- [Object.delete_metadata](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/obj.py#L260-L320)
