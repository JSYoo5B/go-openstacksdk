# Service Actions Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
svc = next(conn.block_storage.services(binary="cinder-volume"))
conn.block_storage.disable_service(svc, reason="maintenance")
for level in conn.block_storage.get_service_log_levels(binary="cinder-api"):
    print(level.host, level.levels)
```

```go
api := services.New(volumeClient)
for svc, err := range api.List(ctx, services.WithListOptions(services.ListOpts{Binary: "cinder-volume"})) {
    if err != nil { return err }
    if _, err := api.Disable(ctx, services.TargetOf(svc), request.Custom(func(c *services.ActionOpts) error { c.Reason = "maintenance"; return nil })); err != nil { return err }
    break
}
levels, err := api.GetLogLevels(ctx, request.Custom(func(c *services.LogOpts) error { v := "cinder-api"; c.Binary = &v; return nil }))
if err != nil { return err }
for _, level := range levels.Levels { fmt.Println(level.Host, level.Levels) }
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L3423-L3616)
- [`service.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/service.py)
