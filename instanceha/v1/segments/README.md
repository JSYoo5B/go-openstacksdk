# Masakari segments

Python `conn.instance_ha.segments()`, `get_segment`, `create_segment`,
`update_segment` and `delete_segment` correspond to this API's `List`, `Get`,
`Create`, `Update` and `Delete`. `conn.ha` is Python's alias for `instance_ha`.

```go
api := segments.New(client) // catalog service type "instance-ha", v1 endpoint
segment, err := api.Create(ctx, segments.CreateOpts{
    Name: "compute-a", RecoveryMethod: "auto", ServiceType: "COMPUTE",
})
if err != nil { return err }
name := "compute-primary"
segment, err = api.Update(ctx, resource.ID(segment.UUID),
    segments.UpdateOpts{Name: &name})
if err != nil { return err }
err = api.Delete(ctx, resource.ID(segment.UUID)) // missing is ignored
```

The route identity is `UUID`. `ID` preserves the separate raw database ID;
integer/string forms, large integers, null and omission remain available.
`Resources.Find(ctx, resource.Name("compute-a"))` matches the name exactly and
rejects duplicates. Segment responses preserve `Body`, `Header` and `StatusCode`.
Accepted responses with invalid JSON/envelopes retain evidence in
`resource.ResponseError`, so mutations are not silently retried.

`ListOpts` provides limit/marker, sorting, recovery method, service type and
enabled filters. Use `WithListOptions`; `WithListQuery` carries additional
server-validated filters and cannot overwrite concrete filters.
`API.All(ctx, options...)` consumes the same typed iterator and options.
`Resources.List/All` also support shared name/query/page-size options.
Automatic iteration follows explicit body or HTTP continuation links, preserving
the collection, origin and filters. Without a continuation link it stops at the
returned page; callers may pass an explicit marker. Numeric-ID versus UUID marker
fallback is not asserted without controller evidence.

An omitted selected microversion means 1.0. Explicit numeric `1.N` versions are
preserved; `enabled` in Create, Update or List requires >=1.2 before any lookup or
HTTP. Python's Segment resource can negotiate up to 1.2; Go requires an explicit
selection for this feature. `latest`, a wrong service type and conflicting
case-insensitive source version headers fail preflight. A manual blank-Type
client with an explicit selection needs a matching `OpenStack-API-Version:
instance-ha 1.N` header. No operation upgrades the source client.

Optional pointer fields omit nil and preserve false/empty values. WithOptions
helpers snapshot their inputs; WithField helpers snapshot extension JSON and
protect core/read-only fields. Version and authentication headers are SDK-owned.
Segment has no status field. Shared deletion waits work; status waits require an
explicit supported string attribute rather than an invented status.

The supported recovery values are `auto`, `reserved_host`, `auto_priority` and
`rh_priority`; server policy validates them. HTTP contracts use the pinned
[openstacksdk Segment](https://opendev.org/openstack/openstacksdk/src/commit/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/instance_ha/v1/segment.py)
and the [Masakari API reference](https://docs.openstack.org/api-ref/instance-ha/).
