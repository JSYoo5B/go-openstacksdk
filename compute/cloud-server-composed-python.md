# Composed cloud server helpers

These `Connection` methods compose existing Compute, Neutron and floating-IP
APIs the way the pinned Python cloud layer composes them. Each is a separate,
documented helper; none of them is a claim of full cloud parity. Review
statuses live in `api/sdk_reviews.json`.

| Python | Go |
| --- | --- |
| `list_servers(bare=True)` | `Connection.ListServerRecords` |
| `list_servers` (default `bare=False`, `detailed`) | `Connection.ListServers` |
| `search_servers(bare=True)` | `Connection.SearchServerRecords` |
| `search_servers` (default `bare=False`, `detailed`) | `Connection.SearchServers` |
| `get_server` | `Connection.GetServer` |
| `get_server_by_id` | `Connection.GetServerByID` |
| `get_openstack_vars` | `Connection.GetOpenStackVars` |
| `get_server_meta` | `Connection.GetServerMeta`, `CloudServerGroups` |
| `update_server` | `Connection.UpdateServer` |
| `rebuild_server` | `Connection.RebuildServer` |
| `delete_server` | `Connection.DeleteServer` |
| `add_server_security_groups` / `remove_server_security_groups` | `Connection.AddServerSecurityGroups` / `RemoveServerSecurityGroups` |
| `search_aggregates` | `Connection.SearchAggregates` |

## Listing and searching

`servers.API.ListCloudRecords` streams `GET /servers/detail` (or `/servers`
with `WithRecordFindDetails(false)`) as `ServerFindRecord` values that carry the
Source Resource view, the received row and the page receipt. `all_tenants=True`
is sent only with `WithRecordFindAllProjects(true)`. Microversion negotiation,
header policy, paging and location projection are shared with `FindRecord`.
The full Python `Proxy.servers` query mapping, client Body filters, paging
fallbacks and per-call controls are described in
`compute/v2/servers-list-python.md`.

`SearchServerRecords` consumes every row, then applies the cloud name-or-ID
phase (exact or glob) and the local dictionary or JMESPath filters. Nothing is
pushed to Nova unless it is an explicit list option. A JMESPath expression
returns arbitrary JSON in `Value` with `Expression` true and no `Records`. When
a page fails, the rows consumed so far are returned in `Inventory` beside the
error. These record readers never expand addresses (Python `bare=True`);
`ListServers`, `SearchServers` and `GetServer` below apply the default
expansion.

## Typed getters and actions

`GetServerByID` returns `(nil, nil)` for a 404, otherwise the server expanded
through `ExpandServerInterfaces` unless `WithCloudServerBare(true)` is set (a Go
extension; Python always expands here). An expansion failure is returned beside
the retained server.

`UpdateServer` finds the server strictly, then sends one PUT containing only the
supplied `name` and `description`. An explicit `Null` description is sent as
JSON null. An empty or null name is rejected before HTTP because the typed update
builder cannot send it. With no attribute nothing is sent (`Sent` false, `Server`
nil), as Python's commit does.

`RebuildServer` rejects an empty image ID before HTTP (Python raises
`TypeError`), sends `imageRef` and a nonempty `adminPass`, and with
`WithCloudServerWait(true)` polls until ACTIVE (ERROR fails) within the timeout
(default 180 seconds, `WithCloudServerUnlimitedWait` for Python's `None`). The
returned admin password, else the supplied one, is restored on the waited
server. The rebuild is never replayed; a failed wait returns the action result
beside the error.

`DeleteServer` finds the server like `find_server` with its defaults (GET by
identity, then the detail list filtered by `name` only). It returns
`(false, nil)` for a missing server or a 404 DELETE and sends a single DELETE
otherwise. `WithCloudServerDeleteIPs` looks up and deletes each floating address
(rows tagged `OS-EXT-IPS:type == floating`, in order) through `GetFloatingIP`
and `DeleteFloatingIP` with the configured retry (default 1) before the server
DELETE, using the Neutron or Nova (`os-floating-ips`) backend the Connection
selects. An address whose lookup reports not found or finds nothing is skipped;
a floating IP that is not deleted aborts before the server DELETE. When the
Connection's `floating_ip_source` is `none` (`WithServerAddressPolicy` or
clouds.yaml), the cleanup is skipped without any request, as Python's
`_has_floating_ips` gate does; per-call floating-IP options do not change the
gate. With wait, absence is observed for the timeout; once the DELETE was
accepted, a wait failure returns `(true, err)`.

## Expansion and hostvars

Python's `bare=False` default passes every returned server through
`_expand_server`: `meta.add_server_interfaces` normally, and
`meta.get_hostvars_from_server` with `detailed=True`. The Go helpers return a
`CloudServerResult` whose `Interfaces` holds the `ExpandServerInterfaces` view
and whose `Hostvars` holds the detailed expansion:

- `WithCloudServerBare(true)` (or `WithCloudServerActionBare(true)` for
  `RebuildServer`) skips both expansions.
- `WithCloudServerDetailed(true)` (`WithCloudServerActionDetailed(true)` for
  `RebuildServer`) builds hostvars for `GetServer`, `ListServers`,
  `SearchServers`, `UpdateServer` and `RebuildServer`. `GetServerByID` rejects
  it before HTTP because `get_server_by_id` only adds interfaces.
- `GetServer` finds with `FindRecord` and details forced on (a missing server is
  `(nil, nil)`). With `WithCloudServerFilters` it lists bare, filters locally
  and expands the single match; several matches are a
  `resource.AmbiguousError`, and a JMESPath value that is not a row selection
  is rejected.
- `ListServers` expands rows in list order and returns the results built
  before a list or expansion failure beside the error. `SearchServers` filters
  over `CloudServerResultView`, so filters can use `public_v4`,
  `interface_ip` or, when detailed, `flavor.name`.

`GetOpenStackVars(ctx, server, addressOptions...)` follows
`get_hostvars_from_server` in Python order and never modifies the supplied
server:

1. `ExpandServerInterfaces` (Neutron supplementation, then public, private and
   interface addresses).
2. Flavor name: a truthy flavor `id` is looked up with
   `flavors.API.FindIdentity` (a missing flavor sets no name); otherwise an
   embedded `original_name` is used. `Flavor.NameSource` reports which.
3. Security groups: `GET servers/{id}` and `GET servers/{id}/os-security-groups`.
   Python stores this list on the supplied server, not in the mapping, so Go
   returns it in `SecurityGroups` and records a failure in
   `SecurityGroupsError` instead of returning it.
4. Image name: a server without an image object (Nova's `""` for a server
   booted from a volume) sets `Image.BootFromVolume` and the mapping image
   `{"id": ""}`; otherwise a truthy image `id` is resolved with
   `image.Service.GetImageRecordName` and a truthy result becomes the name.
5. Volumes: `GetVolumes` for the server, each with the raw `device` of its
   first attachment. A catalog without a block-storage endpoint skips the step
   (`VolumeService` false); other listing failures are recorded in
   `VolumesError`. A first attachment without a `device` key is an error, as
   Python's `KeyError` is.

Interface, flavor and image failures are returned beside the partial result;
cancellation always is. `json.Marshal` of `CloudServerHostvars` renders the
mapping: the record's Source Resource view when the server came from a record
lookup, otherwise the typed server JSON plus a computed `location`, with
`addresses`, `public_v4`, `public_v6`, `private_v4`, `interface_ip`,
`access_ipv4`, `access_ipv6`, the named `flavor` and `image`, and `volumes`
set over it.

`GetServerMeta` adds `CloudServerGroups(hostvars)`: cloud, region,
`cloud_region`, the `group` metadata value, each nonempty comma-separated
`groups` entry, `instance-ID`, `flavor-NAME` and `image-NAME` when a name was
set, then `meta-KEY_VALUE` for every metadata item. Cloud and region come from
`CurrentLocation` (unknown values render as `""`). Metadata items use sorted
key order because the typed metadata is a Go map. Python reads `az` from a key
the Server Resource mapping does not carry, so no availability-zone groups are
produced. Name values that are not strings use Python's rendering for
`None`/`True`/`False` and compact JSON otherwise.

## Security groups

Both helpers take identities (names or IDs). The `secgroup_source`
configuration (`WithSecurityGroupSource` or clouds.yaml, default `neutron`) is
checked first, like `_has_secgroups`: any value other than `neutron` or `nova`
in any casing, including null, returns `network.ErrUnavailableSecurityGroups`
before any request. The server is then found with `FindRecord` (as
`get_server(bare=True)` uses `find_server`), and every group is resolved in
order with `GetSecurityGroupRecord`. Neutron is used only for the exact source
`neutron` while the network service is enabled (`has_network`) and present in
the catalog; otherwise Nova `os-security-groups` is used. A missing server or
group returns `(false, nil)` before any action.
Add sends one `addSecurityGroup` action per group using the group name, else its
ID. Remove tolerates a 404 on one action, continues, and returns false at the end;
any other error stops the sequence. No action is replayed.

## Aggregates

`SearchAggregates` lists every aggregate and filters locally. The filters run
over the Source Aggregate attributes built from each row Nova returned:
Python names (`deleted` becomes `is_deleted`), null and absent members as null,
timestamp strings exactly as sent, and the computed `location`, whose `zone` is
the aggregate's `availability_zone`. For example, `{"availability_zone": null}`
selects aggregates created without an availability zone, as in Python.

## Limits

- Pre-resolved group objects are not accepted; groups are identity strings.
- The expansion reuses `ExpandServerInterfaces`, whose own
  `get_server_public_ip` review stays unresolved for the separate cloud
  Create/Get/Wait lifecycle; the expansion contracts used here are tested.
- Python hostvars are a Munch of a mutable Server Resource (attribute names,
  class attributes and Resource-typed flavor/image values that always contain
  a `name` key, giving `flavor-None` groups). Go returns typed fields and an
  explicit JSON projection instead; see the SDK-R1 boundary in
  `docs/sdk-support-ledger.md`. The hostvars security-group lookup does not
  apply the `secgroup_source` gate, and Python's cache is not modelled.
- Evidence is local HTTP fixtures, not a live cloud.
