# Python-style paging and attribute updates (Masakari)

These APIs reproduce three behaviors of the pinned openstacksdk `instance_ha`
proxy that the default Go calls deliberately do not.

## Marker fallback paging

`segments.API`, `notifications.API`, `hosts.SegmentScope` and
`vmoves.NotificationScope` each add `ListMarkerFallback` and `AllMarkerFallback`
with the same options and validation as `List`/`All`.

- The next page comes from a body link (`links`, `<plural>_links`, a truthy
  top-level `next`) or a `Link` header, with the same precedence as the pinned
  `Resource._get_next_link`.
- When no link is advertised, the caller passed an explicit `limit` and the page
  was not empty, the next request repeats the query with `marker` set to the
  wire `id` of the last row (a JSON string or integer, rendered literally).
  An empty page ends the list. Without a limit, a page without a link is the
  last page.
- A row without a usable `id`, a repeated page or a repeated marker is an error
  (Python only compares with the previous marker).
- Breaking out of the iterator or canceling the context stops before the next
  request.

The Masakari controller's marker identity (database id versus UUID) is not
verified, so the extra request can be rejected by a deployment. That is why the
fallback is a separate, explicit method and `List` keeps following advertised
links only.

## Explicit `null` and omission for segments and hosts

`segments.API.CreateAttributes`/`UpdateAttributes` and
`hosts.SegmentScope.UpdateAttributes` take structs of `request.Optional` fields.
The zero value is omitted, `request.Null` sends JSON `null` (Python sends
`description=None` as `null`) and `request.Present` keeps `false`, zero and
empty strings. Extension fields (`WithAttributeField`) cannot replace a concrete
attribute or the protected `id`, `uuid`, timestamp and parent fields; reserved
headers are rejected. All validation happens before HTTP. `enabled` still needs
microversion 1.2. Host `control_attributes` is arbitrary JSON because the
controller's update schema is not verified by this SDK; the controller
validates it.

Differences from Python: a call with no attribute is rejected before HTTP (Python
skips the PUT when nothing is dirty and returns the seed resource), names are
resolved by an exact list lookup (Python sends the string as the identifier),
and the result is a decoded model with raw response evidence rather than a
mutable Resource.

Tests: `api/r2_small_instanceha_test.go`.
