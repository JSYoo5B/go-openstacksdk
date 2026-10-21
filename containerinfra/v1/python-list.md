# Python-style cluster and template lists (Magnum)

`clusters.API.PythonResources()` and `clustertemplates.API.PythonResources()`
return a `resource.Collection` over `GET /clusters` and `GET /clustertemplates`
(the summary routes used by the pinned Python `clusters()` and
`cluster_templates()` generators). Rows are `ListRecord` values holding the
native model plus `resource.Metadata` with the original row fields.

- Continuation follows body links, a truthy `next` field and the `Link`
  header. When an explicit page size (`resource.WithPageSize` or
  `resource.WithQuery("limit", n)`) was sent and a nonempty page advertises no
  link, the next request carries `marker` set to the row's `id`, or `uuid`
  when `id` is absent. An empty page ends the list.
- `resource.WithBodyFilter`/`WithBodyFilters` apply Python's client-side Body
  filters by Python attribute name (`is_floating_ip_enabled`,
  `is_master_lb_enabled`, `status_reason`, `is_public`, `is_hidden`,
  `is_tls_disabled`, `is_registry_enabled` and the wire names). Integer and
  boolean attributes use the audited normalizers; object filters match a
  subset. Unknown names fail before HTTP.
- The native `List`/`ListDetail` calls keep Gophercloud's `next`-only pager.

Tests: `api/r2_small_filters_test.go`.
