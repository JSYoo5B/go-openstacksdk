# Ndpproxies

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
proxy = conn.network.create_ndp_proxy(
    router_id=router.id, port_id=port.id, ip_address="2001:db8::10")
conn.network.update_ndp_proxy(proxy, description="web")
for item in conn.network.ndp_proxies(router_id=router.id):
    print(item.ip_address)
conn.network.delete_ndp_proxy(proxy)
```

```go
api := ndpproxies.New(networkClient)
proxy, err := api.Create(ctx,
    request.Field("router_id", routerID),
    request.Field("port_id", portID),
    request.Field("ip_address", "2001:db8::10"))
if err != nil {
    return err
}
id := proxy.Text("id")
if _, err := api.Update(ctx, id, request.Field("description", "web")); err != nil {
    return err
}
for item, err := range api.List(ctx, request.Query("router_id", routerID)) {
    if err != nil {
        return err
    }
    fmt.Println(item.Text("ip_address"))
}
_, err = api.Delete(ctx, id)
return err
```
