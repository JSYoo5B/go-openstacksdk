# Conntrackhelpers

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
helper = conn.network.create_conntrack_helper(
    router, helper="ftp", protocol="tcp", port=21)
conn.network.update_conntrack_helper(helper, router, port=2121)
for item in conn.network.conntrack_helpers(router, helper="ftp"):
    print(item.id, item.port)
conn.network.delete_conntrack_helper(helper, router)
```

```go
api := conntrackhelpers.New(networkClient)
helper, err := api.Create(ctx, routerID,
    request.Field("helper", "ftp"),
    request.Field("protocol", "tcp"),
    request.Field("port", 21))
if err != nil {
    return err
}
id := helper.Text("id")
if _, err := api.Update(ctx, routerID, id, request.Field("port", 2121)); err != nil {
    return err
}
for item, err := range api.List(ctx, routerID, request.Query("helper", "ftp")) {
    if err != nil {
        return err
    }
    fmt.Println(item.Text("id"), string(item.Resource.Body["port"]))
}
_, err = api.Delete(ctx, routerID, id)
return err
```
