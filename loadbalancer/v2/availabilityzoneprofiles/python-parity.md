# Python parity: Octavia availability zone profiles

Pinned Python `openstack/load_balancer/v2/availability_zone_profile.py` declares `base_path=/lbaas/availabilityzoneprofiles`, `resource_key=availability_zone_profile`, body fields `id`, `name`, `provider_name`, `availability_zone_data` and a query mapping of the same four names.

Deliberate Go differences (covered by the tests named in `api/sdk_reviews.json`):

- Concrete request types replace `**attrs`; Resource objects are replaced by `AvailabilityZoneProfile` plus raw `Body`/`Header`/`StatusCode`.
- Python `update_availability_zone_profile` with no changed attribute performs no request; Go rejects an empty update before any request.
- Python `delete_*`/`find_*` default to `ignore_missing=True`; Go `Delete` reports a missing profile and `Remove` ignores it by default.
- Python's GET-first `find` swallows 400/403 and falls back to a list; Go's shared identity policy decides the fallback statuses. Duplicate names fail in both.
- Go validates required create fields locally, sends explicit success codes (201/200/204), and preserves empty strings in updates.
