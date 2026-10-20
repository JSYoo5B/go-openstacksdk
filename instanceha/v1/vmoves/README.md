# Masakari VMoves

Python `conn.instance_ha.vmoves(notification)` and
`get_vmove(vmove, notification)` map to a fixed Notification scope.

```go
client.Microversion = "1.3" // before constructing/using the API
scope, err := vmoves.New(client).InNotification(ctx,
    resource.ID(notification.UUID))
if err != nil { return err }
state, kind := "ongoing", "evacuation"
moves, err := scope.All(ctx, vmoves.WithListOptions(vmoves.ListOpts{
    Status: &state, Type: &kind,
}))
if err != nil { return err }
for _, move := range moves {
    move, err = scope.Get(ctx, move.UUID)
    if err != nil { return err }
}
```

VMove is read-only: list/get plus shared status/deletion monitoring, with no
create, update, delete or name lookup capability. Both parent and VMove route
identities are UUIDs. The numeric/string raw database `ID` is distinct. Missing
response UUIDs remain missing and are never synthesized from database IDs.
`NotificationID` records the fixed URI parent; `NotificationUUID` retains the
server's body field independently. Parent selectors in queries are rejected.

The path is `/notifications/{notification_uuid}/vmoves`, matching pinned Python;
the API reference's list heading uses an inconsistent singular spelling.
`ServerID`/`ServerName` decode `instance_uuid`/`instance_name`, not internal Python
attribute names. Models preserve raw fields, large IDs, null/omission, optional
messages/host/timestamp strings and HTTP evidence.

`List` and typed `All` accept limit/marker, sorting, type and status through
`WithListOptions`. `WithListQuery` is an explicit server-validated extension that
cannot overwrite core filters or the parent. `Resources.All` accepts shared
collection options. Iterator options are snapshotted at construction. Explicit
server continuation links are followed while preserving parent, origin and
filters; without a link iteration stops at the page. Automatic marker fallback
remains unverified; callers may supply an explicit marker.

VMove requires numeric selected >=1.3 before scope binding and every actual page
or request. Empty selection is 1.0 and is insufficient. A later downgrade or a
conflicting version header fails before HTTP. Go adds this preflight even though
pinned Python VMove omitted a minimum-version declaration. The source client is
never upgraded and symbolic `latest` cannot prove the gate.

`Wait` compares the real status, detects `failed`, and supports custom failure
states, intervals, bounded/unlimited waits and context cancellation. Valid states
include pending, ongoing, succeeded, failed and ignored; types include evacuation,
migration and live_migration. Python status wait is unlimited by default while Go
uses five minutes; request `resource.WithUnlimitedWait()` for Python behavior.
The generic progress callback reports zero because VMove has no progress field.
HTTP/decode errors retain original causes and accepted response evidence.

Sources: pinned [VMove](https://opendev.org/openstack/openstacksdk/src/commit/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/instance_ha/v1/vmove.py),
[1.3 release notes](https://docs.openstack.org/releasenotes/masakari/2023.2.html)
and [Masakari API reference](https://docs.openstack.org/api-ref/instance-ha/).
