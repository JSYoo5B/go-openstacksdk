# Python bare metal chassis, port group and volume operations

Applies to `chassis`, `portgroups`, `volumeconnectors` and `volumetargets` in this directory. Pinned openstacksdk behavior compared: `create_*`, `get_*`, `find_*`, `update_*`, `patch_*`, `delete_*` and the list generators (`chassis`, `port_groups`, `volume_connectors`, `volume_targets`).

Shared differences, each a deliberate Go design choice:

- Python takes `**attrs` and returns mutable Resource objects; Go uses concrete option types and returns typed models with raw `Body`, `Header` and `StatusCode`.
- Python caps the negotiated microversion per resource (chassis 1.8, port groups 1.26, volume connectors and targets 1.32) and sends it per request. Go sends only the client's configured `Microversion` and never raises it.
- Python retries HTTP 409/503 automatically and treats every status below 400 as success. Go never replays a call and accepts exactly 201 (create), 200 (get, update, list) and 204 (delete).
- `get_*(fields=...)` maps SDK attribute names to wire names; Go `WithGetFields` takes wire names. `patch_*` and `update_*` translate SDK attribute paths to wire paths; Go patches use wire paths (`/node_uuid`, `/standalone_ports_supported`).
- `update_*(**attrs)` sends one `add` per attribute (sorted); Go `FieldPatches` builds the same list. `update_*`/`patch_*` with nothing to change return the object without a request; Go fails an empty patch before HTTP.
- `delete_*` returns the resource or None with ignore_missing=True; Go `Remove` returns an error only, ignoring a 404 by default for an ID reference, while `Delete` keeps the native 404.
- `find_*` is deprecated in Python and emits a warning; Go `FindIdentity` has no warning. It GETs by identity, falls back to a list scan on 400, 403 or 404, reports duplicates as `resource.AmbiguousError`, and returns nil without error when nothing matches unless `resource.WithIdentityFindIgnoreMissing(false)`. Python passes `details=True` as a GET query parameter for volume connectors and targets (not a documented Ironic parameter) and drops it for their fallback list; Go never sends it and scans the detailed list by default.
- Python list `sort_key`/`sort_dir` are rejected for these resources by the Resource query mapping; Go exposes them as typed filters because Ironic supports them. Python accepts `details=True`; Go has `ListDetail`. `Fields` with a detailed list fails locally.
- Required create fields are not validated locally; the server decides.
