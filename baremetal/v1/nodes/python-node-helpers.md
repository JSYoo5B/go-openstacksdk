# Python node helper contracts

This guide describes the node helpers that map Python openstacksdk
`baremetal` proxy methods onto Go. See the [support ledger](../../../docs/sdk-support-ledger.md)
for status definitions. Evidence is limited to local HTTP fixtures; it is not
live-cloud validation.

Shared differences from the Python proxy:

- Nodes are addressed by a string name or UUID. There is no Resource object, no
  dirty tracking and no in-place refresh.
- The Ironic microversion comes from the wrapped `ServiceClient.Microversion`.
  The helpers never negotiate or assert a version.
- The helpers never replay a request. Python retries HTTP 409 and 503 for some
  calls; the Go helpers send each mutation once and surface the error.
- Success codes are exact (for example 202 for maintenance), whereas Python
  accepts any status below 400.
- Deadlines come from the `context.Context`. There is no separate timeout
  value; expiry stays observable through `errors.Is(err, context.DeadlineExceeded)`.

## `ValidateRequired` (`python:baremetal/v1/validate_node`)

`API.ValidateRequired(ctx, id, required)` issues `GET /nodes/{id}/validate`. For each
interface the server reports that is listed in `required` and has a falsy
`result` (false, null or missing), it returns `*ValidationError`
(`errors.Is(err, ErrValidationFailed)`). Interfaces absent from the response
are not checked, and a nil or empty `required` checks nothing. Use
`DefaultValidationInterfaces()` for the Python default of boot, deploy and
power. The reason text is `no reason` when the key is missing and `None` when it
is null, as in Python. Failures are sorted by interface name because Go maps are
unordered. The per-interface results are also returned together with a
`*ValidationError`.

## Maintenance (`set_node_maintenance`, `unset_node_maintenance`)

`API.SetMaintenanceAndGet(ctx, id, reason)` sends `PUT /nodes/{id}/maintenance`
with `{"reason": ...}` and then reads the node. A nil reason is sent as JSON
`null` (the Python default), an empty string is sent as `""`. A failed PUT
performs no read. If the read fails after an acknowledged PUT, the maintenance
change remains in effect remotely. `API.UnsetMaintenanceAndGet` sends
`DELETE /nodes/{id}/maintenance` and then reads the node.

## Waits (`wait_for_node_power_state`, `wait_for_node_reservation`)

`API.WaitForPowerState(ctx, id, expected, opts...)` reads the node immediately,
then every `WithWaitInterval` (default two seconds), until `PowerState` equals
`expected`, and returns the matching node. A read error ends the wait.

`API.WaitForReservation(ctx, snapshot, opts...)` mirrors the Python behavior of
consulting only the object it is given: a snapshot without a `Reservation` is
returned without any HTTP request, and a reserved snapshot is re-read until the
reservation is empty. Release is best effort; a conductor may lock the node
again immediately.

## Boot device (`set_node_boot_device`)

`API.SetBootDevice` already sends both `boot_device` and an explicit
`persistent` value, including `false`, and accepts only HTTP 204.

## Not mapped

Other Python baremetal proxy operations remain unresolved in the review ledger;
see each review's `remaining` list.
