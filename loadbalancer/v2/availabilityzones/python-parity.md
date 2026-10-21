# Python parity: Octavia availability zones

Pinned Python `openstack/load_balancer/v2/availability_zone.py` declares `base_path=/lbaas/availabilityzones`, `resource_key=availability_zone`, a query mapping of `name`, `description`, `availability_zone_profile_id` and `enabled` (Python `is_enabled`), and the default `id`/`name` identity. Because a zone has no UUID, a Python proxy call with a string argument addresses `/lbaas/availabilityzones/<value>`, which is the zone name.

Deliberate Go differences (all covered by the tests named in `api/sdk_reviews.json`):

- Concrete request types replace `**attrs`; Resource objects are replaced by `AvailabilityZone` plus raw `Body`/`Header`/`StatusCode`. Python's `is_enabled` is `Enabled`.
- Python `update_availability_zone` with no changed attribute performs no request and returns the local object. Go rejects an empty update before any request.
- Python `delete_*` and `find_*` default to `ignore_missing=True` (deprecated). Go `Delete` reports a missing zone; `Remove` (shared lookup policy) ignores it by default.
- Python `find_*` and Go `FindIdentity` both GET first and then list by name; Python's GET-first also swallows 400/403, Go's shared identity policy decides which statuses fall back to the list.
- Go validates required create fields locally and sends explicit success codes (201/200/204) instead of accepting any 2xx.
- Python sends only dirty fields from an attrs mapping; Go sends exactly the typed fields that are non-nil plus extensions, so `false`, `0` and `""` are preserved.
