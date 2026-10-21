# Python openstacksdk mapping for Masakari (instance HA v1)

Verified Go counterparts for the pinned `openstack.instance_ha.v1` proxy are
listed in `api/sdk_reviews.json`; tests are in `api/instanceha_*_test.go`.

## Differences that remain open

- Segment, host and notification list calls: the pinned `Resource.list` takes the
  wire `id` of the last row as the marker. When the caller passes `limit` and the
  server omits a next link, it requests one more page with that marker. Go
  follows only advertised links and does not implement this fallback, and the
  server's marker identity (database id versus UUID) is unverified.
- `description: null` is expressible in Python segment create/update; the Go
  concrete option treats a nil pointer as omission and protects the core field
  against extension fields.
- Host update `control_attributes` remains a server-validated extension.

## Waiters

`wait_for_status` and `wait_for_delete` take a Resource in Python and can return
a cached target state without fetching. Go waiters take a fixed UUID reference,
poll through the typed getter and return a fresh model (or only an error for
deletion).
