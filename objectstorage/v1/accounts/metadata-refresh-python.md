# Metadata Refresh Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := accounts.New(swiftClient)
result, err := api.SetMetadataAndRefresh(ctx, map[string]any{
    "owner": "ops",
    "enabled": false,
    "X-Account-Meta-External": "value",
})
if err != nil {
    return err
}
observed := result.Refreshed.Metadata.Values["owner"]
_ = observed
```

## References

- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L125-L147)
- [BaseResource.set_metadata](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L73-L92)
