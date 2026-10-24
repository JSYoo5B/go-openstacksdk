# Python `Proxy.servers` listing

`servers.API.ListCloudRecords` is the Go counterpart of the pinned
`openstack/compute/v2/_proxy.py` `Proxy.servers(details=True,
all_projects=False, **query)`, which delegates to `openstack/proxy.py`
`Proxy._list` and `openstack/resource.py` `Resource.list` over the
`openstack/compute/v2/server.py` `Server` resource. The cloud
`list_servers`, `search_servers` and `OpenStackInventory.list_hosts` paths
consume the same rows (see `compute/cloud-server-composed-python.md` and
`docs/connection-r2-misc.md`). The native `servers.API.List`/`All` iterators
remain the Gophercloud-shaped alternative described in `docs/listing.md`.

## Routes and scope

- The default is `GET /servers/detail`; `WithRecordFindDetails(false)` selects
  `GET /servers` (the Python `details=False` alternative).
- `WithRecordFindAllProjects(true)` sends `all_tenants=True`; `false` or no
  option sends nothing. A raw `all_tenants` query is sent unchanged when no
  typed all-projects option is supplied; combining the typed option with a raw
  `all_tenants` key fails before HTTP (Python would let the typed value win).
- The selected token is sent as is; Nova decides whether the caller may list
  other projects.

## Query parameters

`WithRecordFindQuery(key, value)` accepts every client-side name of the pinned
`Server._query_mapping` and its Nova name, including the implicit `limit` and
`marker` and the tag filters:

| Client name | Nova name |
| --- | --- |
| `access_ipv4`, `access_ipv6` | `access_ip_v4`, `access_ip_v6` |
| `has_config_drive` | `config_drive` |
| `deleted_only` | `deleted` |
| `compute_host` | `host` |
| `is_soft_deleted` | `soft_deleted` |
| `ipv4_address`, `ipv6_address` | `ip`, `ip6` |
| `changes_since`, `changes_before` | `changes-since`, `changes-before` |
| `id` | `uuid` |
| `any_tags`, `not_tags`, `not_any_tags` | `tags-any`, `not-tags`, `not-tags-any` |

Every other declared name (`auto_disk_config`, `availability_zone`,
`created_at`, `description`, `flavor`, `hostname`, `image`, `kernel_id`,
`key_name`, `launch_index`, `launched_at`, `locked`, `locked_by`, `name`,
`node`, `power_state`, `progress`, `project_id`, `ramdisk_id`,
`reservation_id`, `root_device_name`, `status`, `task_state`,
`terminated_at`, `user_id`, `vm_state`, `sort_key`, `sort_dir`,
`pinned_availability_zone`, `tags`, `limit`, `marker`) is sent under the same
name.

- When both names are supplied the client-side name wins, and an explicit
  client-side `null` suppresses the parameter, as `QueryParameters._transpose`
  and Requests do.
- Values follow Requests encoding: `true`/`false` become `True`/`False`,
  arrays repeat the parameter and `null` items are dropped.
- Unknown keys that are not `Server` Body attributes are silently dropped, as
  `Resource.list` validates with `allow_unknown_params=True`.
- `Server` Body attributes that are not query parameters (for example
  `host_status`, `is_locked` or `metadata`) become client filters: they are
  not sent and each row is compared locally with Python equality, with a JSON
  object filter matching a nested subset.

## Paging and caps

- Continuation follows `servers_links`, an api-wg `links` value, a top-level
  `next` (including the exact relative `/v2.1/servers[/detail]` alias) or an
  HTTP `Link` header. Without a link, a caller `limit` (or the
  `WithRecordFindMaxItems` hint) keeps requesting with the last received ID as
  `marker` until an empty page.
- `WithRecordFindMaxItems(n)` sends `limit=n` when no limit (or a zero limit)
  is given and counts every received row, including rows rejected by a client
  filter. Like `Resource.list`, the cap is checked before each further row, so
  a cap reached exactly at a page boundary still requests the next page and
  reports its failure, but yields none of its rows.
- `WithRecordFindPaginated(false)` reads exactly one page.
- A continuation that repeats a previous page fails with the actual page
  receipt after the rows already yielded.

## Per-call controls

- `WithRecordFindMicroversion` overrides the negotiated microversion for this
  call only; without it the captured client microversion is used, or the
  minimum of the server maximum and `Server._max_microversion` 2.100 is
  negotiated.
- `WithRecordFindHeaders` adds request headers; `Accept: application/json` is
  the default and may be replaced. Microversion headers must use
  `WithRecordFindMicroversion`.
- The Python control keyword arguments (`paginated`, `max_items`,
  `microversion`, `headers`, `details`, `all_projects`, `base_path`) are typed
  options in Go; passing them as raw query keys fails before HTTP. The
  deprecated `Proxy._list` `jmespath_filters` argument is also rejected before
  HTTP; the cloud `SearchServers` expression filter is the supported
  alternative.

## Rows

Each `ServerFindRecord` carries the Source `Resource` view (the declared
`Server` Body descriptors, aliases, defaults, nested Flavor/Image/volume
conversions and computed location shared with `FindRecord`, see
`compute/v2/server-find-python.md`), the received `Wire` row, the page
`Envelope`, `Header`, `StatusCode` and the selected `Microversion`. The views
are independent copies.

## Limitations

- Python returns mutable `Server` Resources with session, cache and dirty
  state. That runtime is the shared SDK-R1/C1/S1 boundary of
  `docs/sdk-support-ledger.md` and is not reproduced.
- Go validates `limit` as a nonnegative integer; Python forwards arbitrary
  `limit` and `max_items` values.
- Continuation links that change the origin, collection path or caller
  filters are rejected with the actual page receipt instead of being followed.
- Final 200..399 representations are accepted; informational statuses are
  rejected with the actual response.
