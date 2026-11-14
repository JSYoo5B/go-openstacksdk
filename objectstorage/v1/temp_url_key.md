# Temp Url Key

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
conn.object_store.set_account_temp_url_key(account_secret)
conn.object_store.set_container_temp_url_key(
    "books", container_secret, secondary=True
)
key = conn.object_store.get_temp_url_key("books")
```

```go
package main

import (
    "context"
    "fmt"

    objectstorage "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/accounts"
    "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/containers"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func tempURLKeys(ctx context.Context, service *objectstorage.Service, name, accountSecret, containerSecret string) error {
    accountOptions := []accounts.SetTempURLKeyOption{
        accounts.WithSetTempURLKeyOpts(accounts.SetTempURLKeyOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        accounts.WithSetTempURLKeyHeaders(map[string]string{"X-Trace-Label": "account-key"}),
        request.Header("X-Request-Label", "account"),
        request.Custom(func(c *accounts.SetTempURLKeyOpts) error { c.Secondary = false; return nil }),
    }
    accountAck, err := service.Accounts.SetTempURLKey(ctx, accountSecret, accountOptions...)
    if accountAck != nil { fmt.Println(accountAck.StatusCode) }
    if err != nil { return err }

    containerOptions := []containers.SetTempURLKeyOption{
        containers.WithSetTempURLKeyOpts(containers.SetTempURLKeyOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        containers.WithSetTempURLKeyHeaders(map[string]string{"X-Trace-Label": "container-key"}),
        request.Header("X-Request-Label", "container"),
        request.Custom(func(c *containers.SetTempURLKeyOpts) error { c.Secondary = true; return nil }),
    }
    containerAck, err := service.Containers.SetTempURLKey(ctx, name, containerSecret, containerOptions...)
    if containerAck != nil { fmt.Println(containerAck.StatusCode) }
    if err != nil { return err }

    getOptions := []objectstorage.GetTempURLKeyOption{
        objectstorage.WithGetTempURLKeyOpts(objectstorage.GetTempURLKeyOpts{
            Headers: map[string]string{"X-Request-Label": "initial"},
        }),
        request.Custom(func(c *objectstorage.GetTempURLKeyOpts) error { c.Container = name; return nil }),
        objectstorage.WithGetTempURLKeyHeaders(map[string]string{"X-Trace-Label": "key-discovery"}),
        request.Header("X-Request-Label", "read"),
        request.Custom(func(c *objectstorage.GetTempURLKeyOpts) error { v := true; c.Newest = &v; return nil }),
        request.Custom(func(c *objectstorage.GetTempURLKeyOpts) error { c.Newest = nil; return nil }),
        request.Custom(func(c *objectstorage.GetTempURLKeyOpts) error { v := false; c.Newest = &v; return nil }),
    }
    selected, err := service.GetTempURLKey(ctx, getOptions...)
    if selected != nil {
        if selected.Container != nil { fmt.Println(selected.Container.StatusCode) }
        if selected.Account != nil { fmt.Println(selected.Account.StatusCode) }
    }
    if err != nil { return err }
    fmt.Println(selected.Key != nil, selected.FromContainer, selected.Secondary)
    return nil
}

func main() {}
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L923)
- [ Base](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_base.py#L68)
- [Tempurl](https://github.com/openstack/swift/blob/5e4b45fc17a59d4edd7079c0da3fe028175def45/swift/common/middleware/tempurl.py)
- [account core](accounts/temp_url_key_core_test.go)
- [options](accounts/temp_url_key_options_test.go)
- [container core](containers/temp_url_key_core_test.go)
- [options](containers/temp_url_key_options_test.go)
- [getter core](temp_url_key_core_test.go)
- [options](temp_url_key_options_test.go)
- [Temp Url Key Contracts Test](temp_url_key_contracts_test.go)
- [Connection Objectstorage Temp Url Key Test](../../connectiontest/connection_objectstorage_temp_url_key_test.go)
- [Swift Temp Url Key Test](../../internal/cmd/sdkgen/swift_temp_url_key_test.go)
