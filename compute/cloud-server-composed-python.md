# Composed cloud server helpers

These `Connection` methods compose existing Compute, Neutron and floating-IP
APIs the way the pinned Python cloud layer composes them. Each is a separate,
documented helper; none of them is a claim of full cloud parity. Review
statuses live in `api/sdk_reviews.json`.

| Python | Go |
| --- | --- |
| `list_servers(bare=True)` | `Connection.ListServerRecords` |
| `search_servers` | `Connection.SearchServerRecords` |
| `get_server_by_id` | `Connection.GetServerByID` |
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

`SearchServerRecords` consumes every row, then applies the cloud name-or-ID
phase (exact or glob) and the local dictionary or JMESPath filters. Nothing is
pushed to Nova unless it is an explicit list option. A JMESPath expression
returns arbitrary JSON in `Value` with `Expression` true and no `Records`. When
a page fails, the rows consumed so far are returned in `Inventory` beside the
error. Search and list never expand addresses (Python `bare=True`).

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

`DeleteServer` returns `(false, nil)` for a missing server or a 404 DELETE
and sends a single DELETE otherwise. `WithCloudServerDeleteIPs` looks up and
deletes each floating address (rows tagged `OS-EXT-IPS:type == floating`, in
order) through `GetFloatingIP` and `DeleteFloatingIP` with the configured retry
(default 1) before the server DELETE; a floating IP that is not deleted aborts
before the server DELETE. With wait, absence is observed for the timeout; once
the DELETE was accepted, a wait failure returns `(true, err)`.

## Security groups

Both helpers take identities (names or IDs). The server is found first with
`FindRecord`, then every group is resolved in order with
`GetSecurityGroupRecord` (Neutron, or Nova when the catalog cleanly lacks
Neutron). A missing server or group returns `(false, nil)` before any action.
Add sends one `addSecurityGroup` action per group using the group name, else its
ID. Remove tolerates a 404 on one action, continues, and returns false at the end;
any other error stops the sequence. No action is replayed.

## Aggregates

`SearchAggregates` lists every aggregate and filters locally. The filter view is
rebuilt from the typed Go aggregate with Python attribute names; timestamps are
re-rendered and `location` and unknown wire fields are absent.

## Limits

- Python's `secgroup_source` gate and pre-resolved group objects are not modelled.
- `detailed=True` hostvars (`get_openstack_vars`, `get_server_meta`) have no Go
  equivalent; `get_server_public_ip` style expansion remains the separate,
  independently reviewed `ExpandServerInterfaces`.
- Evidence is local HTTP fixtures, not a live cloud.
