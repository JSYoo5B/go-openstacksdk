# Python openstacksdk parity: Shared File System resource locks, groups, instances, subnets, limits, services

Python is read from the pinned openstacksdk source; fixture tests are not deployment evidence. These SDK-owned packages fill routes Gophercloud v2.15.0 lacks: [resourcelocks](resourcelocks/README.md), [sharegroups](sharegroups/README.md), [sharegroupsnapshots](sharegroupsnapshots/README.md), [shareinstances](shareinstances/README.md), [snapshotinstances](snapshotinstances/README.md), [sharenetworksubnets](sharenetworksubnets/README.md), [limits](limits/README.md), the service actions in `services` (`Enable`, `Disable`, `EnsureShares`) and `snapshots.API.SetMetadata`/`ReplaceMetadata`.

## Shared differences

- Python takes an ID or Resource plus `**attrs`/`**query` and returns mutable Resources; Go uses concrete options, explicit references and typed snapshots with wire `Evidence`.
- Python clamps each resource to `_max_microversion` (2.81 locks, 2.55 groups and group snapshots) and negotiates up to it. Go sends only the configured version, checks a documented floor locally and never clamps or upgrades.
- Python accepts any status below 400 for most calls; Go accepts only the listed success codes (deletes and actions also accept 200/202/204).
- Python `ignore_missing=True` delete default is the default of `Remove`; `resource.WithMissingError()` is `ignore_missing=False`. Only 404 is swallowed.
- Python silently drops query keys that are not in a resource's query mapping and filters Body attributes client-side. Go sends typed server filters and rejects extension queries colliding with them; it never filters locally except through Collection options.
- Python paginates with `marker` when a `limit` is given. Manila list routes here return one response; Go yields that response and never invents marker/offset continuation. Caps are local (`WithListMaxItems`).
- Python update calls with no changed attribute send nothing; Go rejects an update that supplies nothing.
- Python proxy calls that hit the adapter without `raise_from_response` (share-group-snapshot `reset_status` and `members`, `ensure_shares`) ignore error statuses; Go reports every non-success response.

## Per operation

- Resource locks: Python moves `resource_type` through `__conflicting_attrs` and sends it in the body/query; Go has a typed `ResourceType`. Go requires `resource_id` and `resource_type` locally.
- Share groups: Python declares `share_group_snapshot_id`; Go sends the API member `source_share_group_snapshot_id`. `share_groups()` has no `details` and lists `/share-groups`; `ListDetail` is a Go addition. Collection lists use `/detail`.
- Share group snapshots: Python `reset_share_group_snapshot_status` and `share_group_snapshot_members` first GET the snapshot; Go performs one request.
- Share instances: Python `delete_share_instance` is the `force_delete` action; Go `ForceDelete`/`Remove`. The docstring filters `export_location_id/path` are dropped by Python; Go sends them.
- Snapshot instances: details default true as in Python. `snapshot_id`/`project_id` are sent by Go, dropped by Python.
- Subnets: body envelope `share-network-subnet`. Python `delete_share_network_subnet_metadata` iterates `set(keys)` in arbitrary order, ignores `ignore_missing`, and clears everything for `keys=None`; Go deduplicates in first-seen order, stops at the first failure, and clearing is `ReplaceMetadata` with an empty map. Python returns the input metadata; Go returns the service's response.
- Limits: the pinned Python `Limit` declares `maxTotal*`/`total*Used` as top-level members although Manila nests them under `absolute`; per source reading Python therefore yields a Limit with those attributes unset. Go decodes `absolute` and `rate` (not verified against a server).
- Services: Python `disable_service` omits a falsy reason; Go sends an explicitly supplied empty reason. Python requires no explicit host/binary validation; Go requires both non-empty. `EnsureShares` is acknowledged only; the back end may still be ensuring shares. Version floors 2.7 (enable/disable), 2.83 (reason), 2.86 (ensure-shares) are Go-side checks.
- Snapshot metadata: `set_share_snapshot_metadata(replace=...)` maps to `SetMetadata` (POST) and `ReplaceMetadata` (PUT); Python returns the input, Go the service response.

Success codes and floors are taken from the Python resources and the Manila API reference as understood by the authors; none are verified against a live deployment.
