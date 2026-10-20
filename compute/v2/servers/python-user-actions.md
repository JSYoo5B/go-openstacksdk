# Nova user action defaults and optional values

These eight named pinned openstacksdk proxy calls have owned Go APIs on
`conn.ComputeV2(ctx).Servers`. They submit requests directly, without looking up
servers, allocating IPs, listing legacy floating IP inventory, or waiting for a
server state change. A successful call means the action was accepted; cloud
policy and server state still determine whether Nova permits it.

| Pinned openstacksdk | Go | Default wire request |
|---|---|---|
| `clear_server_password(server)` | `api.ClearPassword(ctx, id)` | bodyless DELETE `servers/{id}/os-server-password` |
| `restore_server(server)` | `api.Restore(ctx, id)` | POST `{"restore":null}` |
| `lock_server(server, locked_reason=None)` | `api.LockWithOptions(ctx, id)` | POST `{"lock":null}` |
| `unshelve_server(server)` | `api.UnshelveWithOptions(ctx, id)` | POST `{"unshelve":null}` |
| `add_fixed_ip_to_server(server, network)` | `api.AddFixedIP(ctx, id, networkID)` | POST `{"addFixedIp":{"networkId":networkID}}` |
| `remove_fixed_ip_from_server(server, address)` | `api.RemoveFixedIP(ctx, id, address)` | POST `{"removeFixedIp":{"address":address}}` |
| `add_floating_ip_to_server(server, address, fixed_address=None)` | `api.AddFloatingIP(ctx, id, address)` | POST `{"addFloatingIp":{"address":address}}` |
| `remove_floating_ip_from_server(server, address)` | `api.RemoveFloatingIP(ctx, id, address)` | POST `{"removeFloatingIp":{"address":address}}` |

All POSTs use `servers/{id}/action`. `ClearPassword` removes Nova's stored
password; it does not change the administrator password inside the guest.
`Restore` applies to soft-deleted servers and does not perform a force-delete or
rebuild. These calls report 404 rather than treating it as an ignored missing
resource.

The [official Nova microversion history](https://docs.openstack.org/nova/latest/reference/api-microversion-history.html)
defines reason support at **2.73**, unshelve availability zone strings at
**2.77**, host and explicit null-zone unpinning at **2.91**, and direct fixed or
floating IP action removal at **2.44**. IP actions therefore support **2.1–2.43**.
This differs from `os-floating-ips` inventory removal at 2.36: direct
`AddFloatingIP` can still execute at 2.36–2.43 because it never queries inventory.
The SDK reports `resource.ErrUnsupported` before HTTP when a selected version
cannot execute the requested option or action. It never silently downgrades or
performs additional version discovery. A blank selected version means Nova's
2.1 default; `latest` is allowed for modern actions but rejected for removed IP
actions. Numeric versions must have major 2 and one decimal minor component.

```python
# Select compute_api_version=2.91 in clouds.yaml for this modern example.
conn.compute.clear_server_password(server_id)
conn.compute.restore_server(server_id)
conn.compute.lock_server(server_id, locked_reason="maintenance")
conn.compute.unshelve_server(server_id, host="compute-1")
conn.compute.unshelve_server(server_id, availability_zone=None)

# A separate Compute adapter selected at 2.43 executes the legacy IP actions.
legacy.compute.add_fixed_ip_to_server(server_id, network_id)
legacy.compute.remove_fixed_ip_from_server(server_id, "10.0.0.1")
legacy.compute.add_floating_ip_to_server(
    server_id, "203.0.113.1", fixed_address="10.0.0.1")
legacy.compute.remove_floating_ip_from_server(server_id, "203.0.113.1")
```

## Runnable Go main

Prepare a module using the [installation guide](../../../docs/install.md), save
this as `main.go`, and choose one action per invocation. For example,
`go run . -cloud dev -id SERVER_ID -action lock -value maintenance` sends one
lock action at 2.91. Legacy actions require an explicit version, such as
`go run . -id SERVER_ID -version 2.43 -action add-floating -value 203.0.113.1`.
No action runs when `-action` is omitted.

```go
package main

import (
    "context"
    "flag"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    id := flag.String("id", "", "literal server ID")
    version := flag.String("version", "2.91", "selected Nova microversion")
    action := flag.String("action", "", "clear-password, restore, lock, unshelve-host, unshelve-unpin, add-fixed, remove-fixed, add-floating, remove-floating")
    value := flag.String("value", "", "reason, host, network ID or address")
    flag.Parse()
    if *action == "" { return }
    if *id == "" { log.Fatal("-id is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud), sdk.WithMicroversion(sdk.Compute, *version))
    if err != nil { log.Fatal(err) }
    compute, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    api := compute.Servers
    switch *action {
    case "clear-password":
        err = api.ClearPassword(ctx, *id)
    case "restore":
        err = api.Restore(ctx, *id)
    case "lock":
        err = api.LockWithOptions(ctx, *id, servers.WithLockedReason(*value))
    case "unshelve-host":
        err = api.UnshelveWithOptions(ctx, *id, servers.WithUnshelveHost(*value))
    case "unshelve-unpin":
        err = api.UnshelveWithOptions(ctx, *id, servers.WithUnshelveUnpinnedAvailabilityZone())
    case "add-fixed":
        err = api.AddFixedIP(ctx, *id, *value)
    case "remove-fixed":
        err = api.RemoveFixedIP(ctx, *id, *value)
    case "add-floating":
        err = api.AddFloatingIP(ctx, *id, *value)
    case "remove-floating":
        err = api.RemoveFloatingIP(ctx, *id, *value)
    default:
        log.Fatal("unknown -action")
    }
    if err != nil { log.Fatal(err) }
    log.Print("Nova accepted the request")
}
```

## Options and error evidence

`WithLockedReason("")` includes the empty string; no option or
`WithoutLockedReason()` sends `lock:null`. `WithUnshelveHost("")` omits host, as
Python does. `WithUnshelveAvailabilityZone("")` includes the empty string,
`WithUnshelveUnpinnedAvailabilityZone()` includes JSON null, and
`WithUnshelveDefaultAvailabilityZone()` restores omission. Host-only requests
nest host inside `unshelve`, and a null zone remains inside that object.
`WithFloatingIPFixedAddress("")` includes the empty string while
`WithoutFloatingIPFixedAddress()` restores omission. Later options win.

Concrete `With*Header` helpers use the shared SDK header validation. Auth,
service origin, microversion, and protocol headers retain library ownership.
JSON extension values are copied when `With*Field` is constructed. Unknown
fields nest inside the action object, including when the default action is
null; this additive Go extension behavior is separate from Python's named
parameters. Declared reason, zone, host, address, and network fields cannot be
overridden through extensions. ClearPassword has no body and rejects JSON
extensions. Query and secondary argument extensions are rejected.

The selected source is captured before applying user options, with sticky
checks across callbacks, retries, physical response reads, and Close. Literal
server IDs are escaped once; blank IDs, dot segments, invalid UTF-8, or controls
are rejected. JSON body strings retain empty values and must be valid UTF-8;
Nova validates address formats and lengths.

Like the pinned Python `raise_from_response` calls, these owned APIs accept
final statuses 200–399 without decoding action responses as JSON. Native
Gophercloud calls keep their narrower status rules and existing signatures.
Accepted response read/Close failures retain the actual body, headers, and
status in `resource.ResponseError`. Rejected HTTP statuses preserve the native
`gophercloud.ErrUnexpectedResponseCode` and physical read/Close causes. Context
cancellation and its custom cause, source changes, retry failures, and attempts
to replace SDK-owned request JSON remain visible through `errors.Is`/`As`.

These eight named declarations do not establish complete Python Resource,
session, discovery, every Nova action, or authenticated acceptance parity.
