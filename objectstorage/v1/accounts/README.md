# Accounts

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
account = conn.object_store.get_account_metadata()
print(account.metadata)
conn.object_store.set_account_metadata(book="Moby Dick", note="literal %2F 雪")
conn.object_store.delete_account_metadata(["note"])
```

```go
package main

import (
    "context"
    "fmt"

    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func changeAccountMetadata(ctx context.Context, api *accounts.API) error {
    getOptions := []accounts.GetMetadataOption{
        accounts.WithGetMetadataOpts(accounts.GetMetadataOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        accounts.WithGetMetadataHeaders(map[string]string{"X-Trace-Label": "metadata"}),
        request.Header("X-Request-Label", "account-metadata"),
        request.Custom(func(c *accounts.GetMetadataOpts) error { v := false; c.Newest = &v; return nil }),
    }
    before, err := api.GetMetadata(ctx, getOptions...)
    if err != nil { return err }
    if before.Metadata.BytesUsed != nil { fmt.Println(*before.Metadata.BytesUsed) }

    mutationOptions := []accounts.MetadataOption{
        accounts.WithMetadataOpts(accounts.MetadataOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        accounts.WithMetadataHeaders(map[string]string{"X-Trace-Label": "metadata"}),
        request.Header("X-Request-Label", "metadata-mutation"),
    }
    acknowledgement, err := api.SetMetadata(ctx, map[string]string{
        "book": "Moby Dick", "note": "literal %2F 雪",
    }, mutationOptions...)
    if acknowledgement != nil {
        fmt.Println(acknowledgement.StatusCode)
    }
    if err != nil { return err }

    refreshOptions := append([]accounts.GetMetadataOption(nil), getOptions...)
    refreshOptions = append(refreshOptions, request.Custom(func(c *accounts.GetMetadataOpts) error { c.Newest = nil; return nil }))
    refreshed, err := api.GetMetadata(ctx, refreshOptions...)
    if err != nil { return err }
    fmt.Println(refreshed.Metadata.Values["book"])

    removed, err := api.DeleteMetadata(ctx, []string{"note"}, mutationOptions...)
    if removed != nil { fmt.Println(removed.StatusCode) }
    return err
}

func main() {}
```

## References

- [pinned Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L119)
- [ Base](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L71)
- [Storage Account Services](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/api-ref/source/storage-account-services.inc#L120)
- [Base](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/proxy/controllers/base.py#L1953)
- [Metadata Contracts Test](metadata_contracts_test.go)
