# Octavia providers

`providers.API` wraps Gophercloud v2.15.0 `loadbalancer/v2/providers` for `/lbaas/providers`. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `List` | `GET /lbaas/providers` | 200 | Follows `providers_links`; `ListOpts.Fields` repeat the `fields` query key |
| `ListFlavorCapabilities` | `GET .../{provider}/flavor_capabilities` | 200 | Returns `[]Capability` (name, description); `Fields` supported |
| `ListAvailabilityZoneCapabilities` | `GET .../{provider}/availability_zone_capabilities` | 200 | Returns `[]Capability`; `Fields` supported |

Extension queries use `WithListQuery`, `WithListFlavorCapabilitiesQuery` and `WithListAvailabilityZoneCapabilitiesQuery`; nil options fail with `resource.ErrInvalidOption` before any request. A native 404 for an unknown provider stays an HTTP response error. These are read-only calls; there is no `resource.Collection` for providers.

Cancelling a context stops the local call.
