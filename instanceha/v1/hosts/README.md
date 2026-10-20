# Masakari hosts

Python `conn.instance_ha.hosts(segment)`, `create_host`, `get_host`,
`update_host` and `delete_host` correspond to a fixed Go Segment scope.

```go
scope, err := hosts.New(client).InSegment(ctx, resource.Name("compute-a"))
if err != nil { return err }
host, err := scope.Create(ctx, hosts.CreateOpts{
    Name: "compute-01", Type: "COMPUTE", ControlAttributes: "SSH",
})
if err != nil { return err }
maintenance := true
host, err = scope.Update(ctx, resource.ID(host.UUID),
    hosts.UpdateOpts{OnMaintenance: &maintenance})
if err != nil { return err }
err = scope.Delete(ctx, resource.ID(host.UUID))
```

An explicit Segment UUID binds without a GET; names resolve exactly once, with
duplicate detection. The scope fixes the Segment UUID for every host request.
List `failover_segment_id` must match it; body/query extensions cannot replace
the parent. `Host.SegmentID` records that URI parent independently of server
`FailoverSegmentID`. Host UUID is the route identity; raw `ID` is the database ID.

`List` and typed `All` accept `WithListOptions` for limit/marker, sorting, type,
maintenance and reservation; `Resources.All` accepts shared collection options.
`Resources.Find` resolves an exact host name within this segment. `Delete`
ignores 404 by default; `resource.WithMissingError()` requires existence. Other
HTTP failures and context causes are preserved.

Nil optional pointers omit fields; false values are serialized. WithOptions
snapshot typed inputs, and WithField snapshot extension JSON while protecting
core/read-only/parent fields. Requests use the documented string control
attributes. Responses retain `ControlAttributes` as raw JSON, including the
object form present in pinned Python fixtures. The update schema for control
attributes was not verified; `WithUpdateField("control_attributes", "SSH")`
is an explicit server-validated escape hatch rather than a claimed typed update
capability.

Responses retain raw fields/nulls/large numeric IDs, timestamps, headers/status,
and a typed nested Segment. Accepted decode failures expose HTTP evidence through
`resource.ResponseError`. Host has no status field; shared deletion waits work,
and a custom status wait requires a real exported string attribute.

Version validation uses the selected numeric 1.N (empty means 1.0), rejects
conflicting source version headers before lookup/request, and never upgrades the
client. Automatic pagination follows explicit server continuation links while
preserving parent, origin and filters. Without such a link it stops at the page;
an explicit marker can be supplied. Server marker fallback is not yet verified.

Source contracts: pinned [Host](https://opendev.org/openstack/openstacksdk/src/commit/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/instance_ha/v1/host.py)
and [Masakari API reference](https://docs.openstack.org/api-ref/instance-ha/).
