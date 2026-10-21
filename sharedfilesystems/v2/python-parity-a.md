# Python openstacksdk parity: Shared File System (part A)

This page records how Go calls compare with the pinned Python `openstack.shared_file_system.v2` proxy methods whose names sort from `accept_share_transfer` through `get_share_metadata_item`, plus `list_share_availability_zones` from the cloud layer. Support status for each operation lives in `api/sdk_reviews.json`; this page explains the contracts. Python is read from the pinned source only. Fixture tests are not deployment evidence.

## Shared differences

- Python proxy methods take an ID string or a Resource and `**attrs`/`**query`; Go uses concrete typed options, explicit `resource.ID`/`resource.Name` references and typed snapshots instead of mutable Resources.
- Python negotiates each resource's microversion up to its `_max_microversion` (for example 2.7 availability zones, 2.37 user messages, 2.47 export locations, 2.55 share groups, 2.81 resource locks, 2.82 access rules). Go sends exactly the version configured on the service client, never upgrades silently, and leaves floors to the service unless a call documents a local check.
- Python accepts every HTTP status below 400. Go accepts only the documented success codes of each route and keeps the native response error otherwise.
- `ignore_missing=True` (the Python default for deletes) is the default of `Remove`, `ForceRemove` and `DeleteMetadata`; `resource.WithMissingError()` is `ignore_missing=False`. `resource.WithIgnoreMissing()` selects it explicitly. Only 404 is ever swallowed.
- Acknowledgement of a create or delete does not prove completion. Use `WaitFor` and `WaitForDeletion`. Mutations are never replayed automatically and cancelling a context does not roll back remote work.

## New Go helpers

| Method | Python behavior |
|---|---|
| `shares.API.ForceRemove`, `snapshots.API.ForceRemove`, `replicas.API.ForceRemove` | `delete_share`, `delete_share_snapshot` and `delete_share_replica` with `force=True`: POST `{"force_delete":null}` to the action route for one ID. 404 is ignored by default. In the pinned Python the replica force path never honors `ignore_missing` (the `try` guards only a dict literal), so a 404 always raises there; Go applies the documented flag. |
| `shares.API.DeleteMetadata` | `delete_share_metadata`: one DELETE per key in order. Missing keys are ignored by default; all other failures are collected, the loop continues and `*shares.MetadataDeleteError` lists the failed keys and causes. Cancellation stops the loop and reports the unattempted keys. |
| `shares.API.ReplaceMetadata` | `set_share_metadata(replace=True)`: PUT `/shares/{id}/metadata` with the whole object; a nil map is sent as `{}`. Only 200 succeeds. `SetMetadata` stays the merging POST. |
| `snapshots.API.GetMetadata`, `GetMetadatum`, `DeleteMetadata` | `fetch_share_snapshot_metadata`, `fetch_share_snapshot_metadata_item`, `delete_share_snapshot_metadata`. Only 200 succeeds, keys are path-escaped, deletion stops at the first failure. |
| `sharetypes.API.UnsetExtraSpecProperty` | `delete_share_type_extra_spec_property`: 404 is an error by default (`ErrNotFound` wrapping the native response), `resource.WithIgnoreMissing()` swallows it. |
| `sharetypes.API.CreateWithExtraSpecs` | `create_share_type(name, extra_specs=...)`: preserves `driver_handles_share_servers=false`, null and empty-string specs. The typed `Create` cannot send `false` because the upstream field is a required bool. Visibility is sent as `os-share-type-access:is_public`. |

## Operation outcomes

Mapped (`go_mapping`): the seven `delete_*` calls for shares, snapshots, replicas, networks, transfers, share types and user messages; `delete_share_metadata`; `fetch_/get_share_metadata[_item]`; `create_share_metadata`; the three share-snapshot metadata calls; `delete_share_type_extra_spec_property`; `export_locations`; `get_export_location`; `get_share`; `get_access_rule`; and `create_share`, `create_share_snapshot`, `create_share_network`, `create_share_replica`, `create_share_transfer`, `accept_share_transfer`, `create_share_type`.

Known differences worth remembering:

- `accept_share_transfer` always sends `clear_access_rules`, including `false`; Go omits `false` (the server default is `false`) and returns only an error.
- `create_share_metadata` sends `{}` for a missing dict; Go `SetMetadata` sends `null` for a nil map, so pass an empty map for the Python wire shape.
- The pinned deprecated aliases (`get_share_metadata`, `get_share_metadata_item`, `create_share_metadata`) emit a deprecation warning that Go does not reproduce.

Unresolved: `availability_zones` and the cloud `list_share_availability_zones` (Go uses the legacy `os-availability-zone` route, Python `/availability-zones`), `find_share` and `find_share_transfer` (Python's ID-then-name fallback that swallows 400/403, summary list source), `create_access_rule` and `delete_access_rule` (Python uses the legacy single-POST action without parent lookups or a 2.45 floor), the quota reads and `access_rules` carried over with their earlier remaining items, and the operations with no Go transport at all: resource locks, share groups and group snapshots, share instances, share network subnets and their metadata, and service enable/disable/ensure-shares.
