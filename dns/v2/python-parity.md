# Python openstacksdk DNS mapping

This page lists how the pinned openstacksdk `dns/v2` Proxy and `cloud/_dns` mixin map to the Go DNS packages. Contracts for each operation are recorded in `api/sdk_reviews.json`; detailed wire behavior sits in the package READMEs (`zones`, `recordsets`, `tsigkeys`, `transfer/request`, `transfer/accept`, `quotas`).

Deliberate differences that apply to every mapped operation:

- Go uses concrete option types and explicit references (`resource.ID`, `resource.Name`) instead of `**attrs` and Resource objects. Responses are typed models; unknown response fields are not retained.
- Python `ignore_missing=True` defaults for delete correspond to `Remove`, which ignores a missing resource by default. Python `find_*` defaults to `ignore_missing=True`; the Go `Find` defaults to an error and accepts `resource.WithIgnoreMissing()`.
- The Go list helpers send the native Designate sudo header `X-Auth-Sudo-Tenant-ID` and `X-Auth-All-Projects: true`; Python sends `x-auth-sudo-project-id`.
- There is no Resource cache, dirty tracking or Adapter microversion negotiation; Designate v2 has no microversions.

Not implemented in Go (and not exposed by Gophercloud v2.15.0): blacklists, TLDs, zone exports and imports, floating-IP PTR records, limits, service statuses, the `find_zone_share` helper and the `target_project_id` query of zone share listing, plus every `cloud/_dns` Connection helper. Their reviews stay unresolved.
