# Python shared-file-system operation mapping (read, action, update and wait operations)

This guide records how the Go Manila APIs map to the pinned Python openstacksdk `shared_file_system/v2` proxy for the operations below. Contracts are verified against HTTP fixtures in `api/py_manila_b_*_test.go`; a fixture is not a deployment acceptance test. Operations without a Go counterpart are listed at the end and stay unresolved.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions.

## Shared differences

- Python takes an ID or a mutable `Resource` and returns a `Resource`; Go takes ID strings and returns Gophercloud typed models. Response members outside the model are dropped unless the API documents raw access.
- Python sends the session default microversion, or the smaller of the resource `_max_microversion` and the server maximum when no default is set (user messages cap at 2.37). Go sends only the microversion configured or negotiated on the service client and never upgrades it.
- Python raises `NotFoundException` for a missing resource. Go `Get`, `Delete` and action methods return the native HTTP error (`gophercloud.ResponseCodeIs(err, 404)`); only the `Collection` methods (`Find`, `All`, `Remove`, `WaitFor`, `WaitForDeletion`) add `resource.ErrNotFound` and ignore-missing deletion.
- Python accepts any 2xx response; Go accepts only the documented success codes (202 for actions, 200 for updates and reads) and returns the native unexpected-status error otherwise. No mutation is replayed automatically.
- Python list calls send only the keys in the resource query mapping and filter other body attributes on the client; Go sends its typed Gophercloud filters (or `WithListQuery` extensions) to the server. Python re-requests with a `marker` query when a `limit` is set; Go follows the Gophercloud offset paging that Manila uses. Share-network lists issue one extra probe request at the maximum offset when no limit is set.

## Reads and lists

| Python operation | Go API | Wire contract |
|---|---|---|
| `get_share_network`, `get_share_replica`, `get_share_snapshot`, `get_share_transfer`, `get_share_type`, `get_user_message` | `sharenetworks`, `replicas`, `snapshots`, `sharetransfers`, `sharetypes`, `messages` `API.Get` | One `GET` on `/share-networks/{id}`, `/share-replicas/{id}`, `/snapshots/{id}`, `/share-transfers/{id}`, `/types/{id}`, `/messages/{id}` |
| `shares(details)` | `shares.API.ListDetail` (`details=True`, default) and `shares.API.List` (`details=False`) | `/shares/detail` and `/shares` with the same filters |
| `share_networks(details)` | `sharenetworks.API.ListDetail` and `sharenetworks.API.List` | `/share-networks/detail` and `/share-networks` |
| `share_snapshots(details)` | `snapshots.API.ListDetail` and `snapshots.API.List` | `/snapshots/detail` and `/snapshots` |
| `share_replicas` | `replicas.API.List` | `/share-replicas` |
| `share_transfers(details)` | `sharetransfers.API.List` (default `details=False`) and `ListDetail` | `/share-transfers` and `/share-transfers/detail` |
| `share_types` | `sharetypes.API.List` | `/types`; `is_public` and JSON-encoded `extra_specs` go through `WithListQuery` |
| `user_messages` | `messages.API.List` | `/messages` |
| `services` | `services.API.List` | `/services`; the pinned native typed filters are not encoded, so use `WithListQuery("binary"\|"zone"\|"host"\|"state"\|"status", value)` (Python aliases `name` to `binary` and `availability_zone` to `zone`) |
| `storage_pools(details)` | `schedulerstats.API.ListDetail` (default) and `List` | `/scheduler-stats/pools/detail` and `/scheduler-stats/pools`; filters `pool`, `backend`, `host`, `capabilities`, `share_type` use `WithListQuery`/`WithListDetailQuery` |

`shares.API.List`, `snapshots.API.List` and `sharenetworks.API.List` are SDK additions that return the summary view (identifier, name and links on current Manila releases); the absent fields are zero values, not server-reported zeros.

## Actions

- `promote_share_replica`, `resync_share_replica`, `reset_share_replica_status`, `reset_share_replica_state`: `POST /share-replicas/{id}/action` with `promote`, `resync`, `reset_status` and `reset_replica_state`. Go `Promote` sends `{"promote":{}}` where Python sends `{"promote":null}`; `Resync` and the reset bodies are identical. Only 202 succeeds.
- `reset_share_status`, `reset_share_snapshot_status`: `POST /shares/{id}/action` and `/snapshots/{id}/action` with `{"reset_status":{"status":...}}`; an empty status fails before HTTP.
- `unmanage_share`, `revert_share_to_snapshot`, `soft_delete_share`, `restore_share`: Python first `GET`s the share and then posts the action. Go posts only the action (`Unmanage`, `Revert`, `SoftDelete`, `Restore`); the pre-fetch carries no request data, so a missing share fails on the action instead. `SoftDelete` and `Restore` need Manila 2.69 or later.
- `resize_share`: `shares.API.Resize(ctx, id, newSize, ...)` performs the same `GET`, compares sizes and sends one extend (with `"force":true` only through `WithResizeForce`) or shrink action; equal sizes and disabled directions send nothing. Python compares an integer size; Go compares the decoded `Size`. The decision uses an unlocked read.
- `manage_share`, `manage_share_snapshot`: `shares.API.Manage` and `snapshots.API.Manage` post the required members and optional members (`name`, `description`, `share_type`, `driver_options`, `is_public`, `share_server_id`) to `/shares/manage` and `/snapshots/manage`. Optional pointers and maps are omitted only when nil, so empty strings, `false` and empty maps are sent. `WithManageField` adds extension members and rejects core names. 200 and 202 succeed. `snapshots.API.Unmanage` implements `unmanage_share_snapshot` without a pre-fetch.
- `set_share_metadata` and the deprecated `update_share_metadata`: `shares.API.SetMetadata` (`POST`, merge) and `shares.API.ReplaceMetadata` (`PUT`, replace). Python returns the share with the metadata it sent; Go returns the metadata echoed by the service. The pinned Gophercloud `UpdateMetadata` also posts and therefore does not replace; use `ReplaceMetadata`.

## Updates

`update_share`, `update_share_snapshot`, `update_share_type` and the quota updates commit only the supplied attributes with `PUT` in the resource envelope. Go `Update` methods take pointer options (empty string and `false` are sent, nil is omitted) and `WithUpdateField` extensions for attributes the typed options lack, such as `name` and `description` on shares and snapshots. `update_share_type_extra_specs` posts `{"extra_specs": {...}}` through `sharetypes.API.SetExtraSpecs`.

`update_quota_set`, `update_quota_class_set` and `revert_quota_set` map to `quotasets.ProjectQuotaScope.Update`/`Reset` and `quotaclasssets.QuotaClassScope.Update`. Python always uses the `quota-sets` and `quota-class-sets` routes; Go uses the legacy `os-` routes through Manila 2.6 and the plain routes from 2.7, and requires a numeric microversion. Go sends `force` only when set. Python `revert_quota_set` ignores a missing quota by default; Go `Reset` reports 404 unless `WithResetIgnoreMissing(true)` is passed.

## Waits

`wait_for_status` and `wait_for_delete` map to `WaitFor` and `WaitForDeletion` on the share, snapshot, replica and message collections. Python polls `fetch` on a supplied resource and returns it immediately when its cached status already matches; Go always performs a fresh `GET` on an explicit ID (or resolves a name once). Python defaults are an unlimited status wait, a 120 second delete wait, a two second interval and the failure list `['ERROR']`; Go defaults to five minutes, two seconds and each collection's own failure predicate (for shares: statuses starting with `error`, ending with `fail`/`failed`, or `killed`). `WithUnlimitedWait`, `WithTimeout`, `WithPollInterval` and `WithFailureStates("ERROR")` select the Python behavior per call. Deletion completes on a 404 or a `deleted` status, and authorization failures end the wait. Python returns the resource and raises `ResourceFailure`/`ResourceTimeout`; Go returns the model (or only an error for deletion) and wait failure or context errors.

## Unresolved operations

The Go SDK has no package, type or option set for these Python operations: `limits`, `share_groups`, `update_share_group`, `share_group_snapshots`, `share_group_snapshot_members`, `update_share_group_snapshot`, `reset_share_group_snapshot_status`, `share_instances`, `reset_share_instance_status`, `share_snapshot_instances`, `get_share_snapshot_instance`, `share_network_subnets`, `get_share_network_subnet`, `set_share_network_subnet_metadata`, `set_share_snapshot_metadata`, `resource_locks` and `update_resource_lock`. `update_share_network` is also unresolved: the typed `NeutronNetID`, `NeutronSubnetID` and `NovaNetID` fields are omitted when empty and cannot be overridden by extensions, so an empty or null value cannot be sent.
