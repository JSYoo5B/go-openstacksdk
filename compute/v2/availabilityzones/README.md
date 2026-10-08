# Nova availability zones

`API.ListRecords(ctx)` reads ordinary availability zones lazily as owned
`AvailabilityZoneRecord` values. `Resource` exposes passive JSON `id`, `name`,
`state`, `hosts` and `location`; missing fields are null. Canonical name/state
and wire `zoneName`/`zoneState` aliases follow source dictionary order.
`Wire`, row `Envelope`, `Header` and `StatusCode` preserve actual evidence.

Use `compute.Service.ListAvailabilityZones` to include the Connection's
recorded location, and `ListAvailabilityZoneNames` for the eager Cloud name
operation. Concrete header/microversion options need no builder implementation.
The [Python/Go comparison and standalone main](../../availability-zones.md)
describe defaults, pagination, partial results and failure suppression.

Existing `List` returns native typed DTOs through the native `AllPages`
single-page path. It ignores advertised continuations, retains native decode
and status rules, and validates the whole page before publishing a row.
`ListDetail` is the existing native privileged route; its complete SDK contract
and the Proxy's detail branch remain queued for the core admin phase.
