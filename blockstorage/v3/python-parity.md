# Python Parity

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/get_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.Get`

- Volumes.Get sends GET volumes/{id} like Proxy._get and decodes name, size, multiattach, tenant, host and metadata.
  Tests: `blockstorage/v3/volumes/python_parity_test.go:TestPythonVolumeGetCreateUpdateMatchProxyRequests`.

Python/Go differences:

- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/create_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithCreateHintOpts`

- Volumes.Create posts the volume envelope with wire names such as imageRef and multiattach and puts OS-SCH-HNT:scheduler_hints outside it, as Volume._prepare_request_body does.
  Tests: `blockstorage/v3/volumes/python_parity_test.go:TestPythonVolumeGetCreateUpdateMatchProxyRequests`.

Python/Go differences:

- Python attribute names (image_id, source_volume_id, is_multiattach, scheduler_hints) become CreateOpts fields or WithCreateField wire keys and WithCreateHintOpts in Go.
- Gophercloud validates scheduler hint fields such as same_host as UUIDs before sending; Python forwards the dict unchanged.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/update_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithUpdateField`

- Volumes.Update sends PUT volumes/{id} with only the supplied name and description in the volume envelope.
  Tests: `blockstorage/v3/volumes/python_parity_test.go:TestPythonVolumeGetCreateUpdateMatchProxyRequests`.

Python/Go differences:

- Python commit sends no request when no attribute is dirty; Go Update always sends PUT, even with an empty envelope.
- Attributes outside UpdateOpts need WithUpdateField with the wire key.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/delete_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithDeleteQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ForceDelete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- Remove ignores 404 by default and WithMissingError reports NotFound; Delete with WithDeleteQuery reproduces Python's cascade=False&force=False (or True) query and ForceDelete posts os-force_delete for the pre-3.23 force path.
  Tests: `blockstorage/v3/volumes/python_parity_test.go:TestPythonVolumeDeleteDefaultsCascadeForceAndMissing`.
- Wrapping the direct delete in resource.IgnoreMissing turns a plain HTTP 404 into success like Python's default ignore_missing=True, keeps other failures such as 409 as errors, and leaves the DELETE request unchanged.
  Tests: `resource/missing_test.go:TestIgnoreMissingDropsOnlyPlain404`, `blockstorage/v3/volumes/python_ignore_missing_test.go:TestPythonDeleteIgnoreMissingVolume`.

Python/Go differences:

- At microversion 3.23 or later Python always sends cascade and force as query parameters; Remove sends none and relies on the same server defaults.
- Python chooses the force action or query from microversion support; the Go caller chooses ForceDelete or the force query explicitly.
- ForceDelete sends {"os-force_delete": ""} where Python sends null.
- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.
- Python ignore_missing=True is the default; Go selects it by wrapping the call in resource.IgnoreMissing, and the bare call returns the 404 like ignore_missing=False.

### `python:block_storage/v3/volumes`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ListWithDetails`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ListSummary`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.ListSummaryWithControl`

- Volumes.List always requests volumes/detail, sends name, status and all_tenants=True as given and follows volumes_links.
  Tests: `blockstorage/v3/volumes/python_parity_test.go:TestPythonVolumeListDefaultUsesDetailRouteOnly`.
- details=false selects GET /volumes, details=true and existing List retain /volumes/detail; selected query/native options, ResourceBase and authentication are preserved and linked native pages yield typed summary rows.
  Tests: `blockstorage/v3/volumes/summary_test.go:TestPythonVolumeSummarySelectsDetailsAndPreservesQuery`.
- Shared raw-row MaxItems and SinglePage never rewrite an explicit wire limit; consumer break stops continuation and whole native page decode errors remain observable beyond a cap.
  Tests: `blockstorage/v3/volumes/summary_test.go:TestPythonVolumeSummaryLocalControlsAndConsumerBreak`.
- Invalid receiver/context/control/options do not issue HTTP, callback failure and cancellation cause remain joined, and a later page native HTTP error preserves prior rows and ListSummary operation identity.
  Tests: `blockstorage/v3/volumes/summary_test.go:TestPythonVolumeSummaryPreflightAndPartialFailures`.

Python/Go differences:

- Python all_projects=True sends all_tenants=True; typed Go AllTenants sends all_tenants=true, so WithListQuery("all_tenants", "True") is needed for the identical string.
- Python filters returned rows locally for Body attributes outside the query mapping; Go sends only query parameters and leaves local filtering to the caller.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.
- The shared native summary cap is local and does not supply Python max_items wire-limit hints; native linked URL/query and whole-page decode policy remain explicit. The aggregate /volumes/summary API is a different operation.

### `python:block_storage/v3/init_volume_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.InitializeWithConnector`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithConnectorHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithConnectorHeaders`

- The owned direct-ID action sends the exact Source envelope and arbitrary nested connector JSON, including wwnns arrays/nqn/extensions and nil/null/false/zero/empty values. Source is captured before custom MarshalJSON and JSON is snapshotted before options, so callbacks cannot change transmitted caller data; headers are concrete guarded inputs and native typed connector APIs remain unchanged.
  Tests: `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNestedInputAndOptionSnapshots`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNullEmptyAndRawConnectionInfo`.
- Action200..399 follows Source _action. Initialization requires the JSON response connection_info field and retains its raw object/null/scalar/array without Source cast coercion; termination accepts opaque success bytes. Actual discovery/action body/header/status are separate receipts, never Resource state or invented connector input.
  Tests: `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorResponseShapeAndSourceStatuses`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNegotiationPreservesActualDiscovery`.
- The shared Cinder negotiation honors a selected version without GET, otherwise selects server bounds and Source Volume3.71 ceiling with actual discovery receipts. Guards preserve accepted/rejected physical read/Close/cancel/source errors, sticky route changes, ordinary native retries and original HTTP causes while rejecting mutated retry JSON/version. Source and typed options remain library-owned.
  Tests: `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorCapturesSourceBeforeMarshalAndOptions`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorPhysicalFaultsKeepReceiptsAndCauses`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNativeRetryOwnsBodyAndVersion`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNegotiationPreservesActualDiscovery`.

Python/Go differences:

- Go accepts a literal validated ID and a map[string]any JSON connector instead of mutable Python Resource; arbitrary connector contents remain untyped. Unsupported Go JSON values fail before HTTP.
- Go returns explicit action/discovery receipts; Python initialize returns its unchecked connection_info JSON value and terminate returns None. Neither proves local attachment completion.
- The SDK reuses finite Cinder discovery when no client version is selected, capped at Source Volume3.71; explicitly selected versions retain existing session/client policy. JSON/route/version ownership and physical handling causes are guarded Go behavior.

### `python:block_storage/v3/terminate_volume_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.API.TerminateWithConnector`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithConnectorHeader`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumes.WithConnectorHeaders`

- The owned direct-ID action sends the exact Source envelope and arbitrary nested connector JSON, including wwnns arrays/nqn/extensions and nil/null/false/zero/empty values. Source is captured before custom MarshalJSON and JSON is snapshotted before options, so callbacks cannot change transmitted caller data; headers are concrete guarded inputs and native typed connector APIs remain unchanged.
  Tests: `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNestedInputAndOptionSnapshots`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNullEmptyAndRawConnectionInfo`.
- Action200..399 follows Source _action. Initialization requires the JSON response connection_info field and retains its raw object/null/scalar/array without Source cast coercion; termination accepts opaque success bytes. Actual discovery/action body/header/status are separate receipts, never Resource state or invented connector input.
  Tests: `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorResponseShapeAndSourceStatuses`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNegotiationPreservesActualDiscovery`.
- The shared Cinder negotiation honors a selected version without GET, otherwise selects server bounds and Source Volume3.71 ceiling with actual discovery receipts. Guards preserve accepted/rejected physical read/Close/cancel/source errors, sticky route changes, ordinary native retries and original HTTP causes while rejecting mutated retry JSON/version. Source and typed options remain library-owned.
  Tests: `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorCapturesSourceBeforeMarshalAndOptions`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorPhysicalFaultsKeepReceiptsAndCauses`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNativeRetryOwnsBodyAndVersion`, `blockstorage/v3/volumes/python_connector_test.go:TestPythonVolumeConnectorNegotiationPreservesActualDiscovery`.

Python/Go differences:

- Go accepts a literal validated ID and a map[string]any JSON connector instead of mutable Python Resource; arbitrary connector contents remain untyped. Unsupported Go JSON values fail before HTTP.
- Go returns explicit action/discovery receipts; Python initialize returns its unchecked connection_info JSON value and terminate returns None. Neither proves local attachment completion.
- The SDK reuses finite Cinder discovery when no client version is selected, capped at Source Volume3.71; explicitly selected versions retain existing session/client policy. JSON/route/version ownership and physical handling causes are guarded Go behavior.

### `python:block_storage/v3/get_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.Get`

- Snapshots.Get sends GET snapshots/{id} and decodes the extended progress and project attributes.
  Tests: `blockstorage/v3/snapshots/python_parity_test.go:TestPythonSnapshotGetCreateUpdateMatchProxyRequests`.

Python/Go differences:

- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/snapshots`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.ListDetail`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithListDetailOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithListDetailQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithListOptions`

- ListDetail (details=True) and List (details=False) send the Python routes with name, volume_id, limit and all_tenants=True queries and follow snapshots_links.
  Tests: `blockstorage/v3/snapshots/python_parity_test.go:TestPythonSnapshotListDetailsAllProjectsAndQuery`.

Python/Go differences:

- Python all_projects=True sends all_tenants=True; typed Go AllTenants sends all_tenants=true, so WithListQuery("all_tenants", "True") is needed for the identical string.
- Python filters returned rows locally for Body attributes outside the query mapping; Go sends only query parameters and leaves local filtering to the caller.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/create_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithCreateField`

- Snapshots.Create posts the snapshot envelope with volume_id, name, description, force and metadata.
  Tests: `blockstorage/v3/snapshots/python_parity_test.go:TestPythonSnapshotGetCreateUpdateMatchProxyRequests`.

Python/Go differences:

- Python is_forced=False is sent as "force": false; Go omits a false Force and the server default is the same.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/update_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.WithUpdateField`

- Snapshots.Update sends PUT snapshots/{id} with only the supplied fields.
  Tests: `blockstorage/v3/snapshots/python_parity_test.go:TestPythonSnapshotGetCreateUpdateMatchProxyRequests`.

Python/Go differences:

- Python commit sends no request when no attribute is dirty; Go Update always sends PUT, even with an empty envelope.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/delete_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.ForceDelete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove ignores 404, WithMissingError and plain Delete keep it, and ForceDelete posts os-force_delete for force=True.
  Tests: `blockstorage/v3/snapshots/python_parity_test.go:TestPythonSnapshotDeleteIgnoresMissingAndForceUsesAction`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- force=True posts os-force_delete with null and microversion 3.65 in Python; Go ForceDelete sends {} with the client microversion.
- Python force=True does not apply ignore_missing; Go ForceDelete likewise returns the HTTP error.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/reset_snapshot_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.ResetStatus`

- ResetStatus posts {"os-reset_status": {"status": ...}} to snapshots/{id}/action and sends microversion 3.65 when the client is set to it.
  Tests: `blockstorage/v3/snapshots/python_parity_test.go:TestPythonSnapshotStatusActionsWithPinnedMicroversion`.

Python/Go differences:

- Snapshot._action always sends microversion 3.65; Go sends the client microversion, so a client copy with Microversion 3.65 is needed for the same header.
- Go returns only an error, matching Python's None.

### `python:block_storage/v3/reset_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.ResetStatus`

- The deprecated alias sends the same os-reset_status action as ResetStatus.
  Tests: `blockstorage/v3/snapshots/python_parity_test.go:TestPythonSnapshotStatusActionsWithPinnedMicroversion`.

Python/Go differences:

- Python emits a RemovedInSDK50Warning before delegating; Go has no alias.
- Snapshot._action always sends microversion 3.65; Go sends the client microversion.

### `python:block_storage/v3/set_snapshot_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/snapshots.API.UpdateStatus`

- UpdateStatus posts os-update_snapshot_status with status and includes progress only when given.
  Tests: `blockstorage/v3/snapshots/python_parity_test.go:TestPythonSnapshotStatusActionsWithPinnedMicroversion`.

Python/Go differences:

- Snapshot._action always sends microversion 3.65; Go sends the client microversion.

### `python:block_storage/v3/backups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.ListDetail`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithListDetailQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithListOptions`

- ListDetail (details=True) and List (details=False) send the Python routes with name, status, volume_id and all_tenants=True queries and follow backups_links.
  Tests: `blockstorage/v3/backups/python_parity_test.go:TestPythonBackupListDetailsAndQuery`.

Python/Go differences:

- ListDetailOpts has no typed name, status or volume_id filters; WithListDetailQuery adds them.
- Python all_projects=True sends all_tenants=True; typed Go AllTenants sends all_tenants=true, so WithListQuery("all_tenants", "True") is needed for the identical string.
- Python filters returned rows locally for Body attributes outside the query mapping; Go sends only query parameters and leaves local filtering to the caller.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/get_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.Get`

- Backups.Get sends GET backups/{id} and decodes incremental, project, object count, metadata and availability zone.
  Tests: `blockstorage/v3/backups/python_parity_test.go:TestPythonBackupGetCreateUpdateMatchProxyRequests`.

Python/Go differences:

- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/create_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithCreateField`

- Backups.Create posts the backup envelope with incremental (Python's renamed is_incremental), force, container, snapshot_id, metadata and availability_zone.
  Tests: `blockstorage/v3/backups/python_parity_test.go:TestPythonBackupGetCreateUpdateMatchProxyRequests`.

Python/Go differences:

- Go omits false Force and Incremental values that Python would send explicitly; the server defaults are the same.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/update_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.WithUpdateField`

- Backups.Update sends PUT backups/{id} with only the supplied fields.
  Tests: `blockstorage/v3/backups/python_parity_test.go:TestPythonBackupGetCreateUpdateMatchProxyRequests`.

Python/Go differences:

- Python commit sends no request when no attribute is dirty; Go Update always sends PUT, even with an empty envelope.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/delete_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.ForceDelete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove ignores 404, WithMissingError and plain Delete keep it, and ForceDelete posts os-force_delete for force=True.
  Tests: `blockstorage/v3/backups/python_parity_test.go:TestPythonBackupDeleteForceAndResetAlias`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- force=True posts os-force_delete with null and microversion 3.64 in Python; Go ForceDelete sends {} with the client microversion.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/reset_backup`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.ResetBackupStatus`

- The deprecated alias maps to ResetBackupStatus, which posts os-reset_status and forces microversion 3.64 like Backup._action.
  Tests: `blockstorage/v3/backups/python_parity_test.go:TestPythonBackupDeleteForceAndResetAlias`, `blockstorage/backup_proxy_action_contract_test.go:TestResetVolumeBackupStatusForwardsEveryUTF8StringWithForcedMicroversionAndOpaqueAck`.

Python/Go differences:

- Python emits a RemovedInSDK50Warning and returns None; Go returns a ResetBackupStatusResult with the applied response and Completed flag.

### `python:block_storage/v3/fetch_backup_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.PythonMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.PythonMetadataProfile.FetchMetadata`

- Fetch creates a resource from an explicit ID without a preliminary member GET, GETs the metadata child route, accepts the Source final200..399 status family, parses one response object and observes only its exact metadata JSON field; absence/null/large integers/extension JSON are independent from physical proof.
  Tests: `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileFetchAndLegacyAliasKeepObservedJSON`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileFetchOmissionNullAndDecodeFailures`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileKeepsSourceStatusesAndStock404WithoutFallback`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileAcceptedAndRejectedBodyFaultsKeepPhysicalCauses`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfilePreflightOwnsOptionsAndPreservesWireCauses`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileSnapshotsHeadersConcurrentReuseAndSourceChanges`.

Python/Go differences:

- Compatibility is explicitly selected by API.PythonMetadata(); these child routes are absent from stock Cinder27.0.0 router c36f40684e140a1492b00dd4812e8dce17fb2ebf. A physical stock404 remains an error and does not trigger fallback to ordinary backup GET/PUT.
- Go accepts an explicit validated ID string and returns owned physical evidence instead of a mutable Python Backup Resource; it neither mutates a native Backup cache nor synthesizes observed response metadata from submitted values.
- Go uses the configured client microversion and ordinary library-owned header options rather than Python session negotiation; Source MetadataMixin adds no per-request3.43 gate.
- Fetch requires an UTF-8 JSON object root and stores metadata as owned rawJSON: omission is nil and explicitnull is nonnil null; arbitrary metadata shapes and duplicate literals are retained without float conversion.
- Stock Cinder 27.0.0 registers no /backups/{id}/metadata child routes; the SDK-owned PythonMetadata profile reproduces the pinned Python request shape, error behavior and physical 404, and deployment support of those routes is not verified and not counted as stock Cinder support.

### `python:block_storage/v3/get_backup_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.PythonMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.PythonMetadataProfile.GetMetadata`

- Deprecated Source alias delegates to fetch; the explicit Go profile emits the same GET and retains operation identity GetMetadata.
  Tests: `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileFetchAndLegacyAliasKeepObservedJSON`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileKeepsSourceStatusesAndStock404WithoutFallback`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileAcceptedAndRejectedBodyFaultsKeepPhysicalCauses`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfilePreflightOwnsOptionsAndPreservesWireCauses`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileSnapshotsHeadersConcurrentReuseAndSourceChanges`.

Python/Go differences:

- Compatibility is explicitly selected by API.PythonMetadata(); these child routes are absent from stock Cinder27.0.0 router c36f40684e140a1492b00dd4812e8dce17fb2ebf. A physical stock404 remains an error and does not trigger fallback to ordinary backup GET/PUT.
- Go accepts an explicit validated ID string and returns owned physical evidence instead of a mutable Python Backup Resource; it neither mutates a native Backup cache nor synthesizes observed response metadata from submitted values.
- Go uses the configured client microversion and ordinary library-owned header options rather than Python session negotiation; Source MetadataMixin adds no per-request3.43 gate.
- Fetch requires an UTF-8 JSON object root and stores metadata as owned rawJSON: omission is nil and explicitnull is nonnil null; arbitrary metadata shapes and duplicate literals are retained without float conversion.
- Deprecated GoDoc replaces the Python RemovedInSDK50Warning; no runtime warning is emitted.
- Stock Cinder 27.0.0 registers no /backups/{id}/metadata child routes; the SDK-owned PythonMetadata profile reproduces the pinned Python request shape, error behavior and physical 404, and deployment support of those routes is not verified and not counted as stock Cinder support.

### `python:block_storage/v3/set_backup_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.PythonMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.PythonMetadataProfile.SetMetadata`

- Source MetadataMixin POSTs metadata envelope including explicit{} for nil/empty input, adds no3.43 microversion gate and does not JSON-decode the response; Go owns input before callbacks and preserves opaque accepted-response evidence.
  Tests: `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileSetOwnsInputAndOpaqueAcknowledgement`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileKeepsSourceStatusesAndStock404WithoutFallback`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileAcceptedAndRejectedBodyFaultsKeepPhysicalCauses`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfilePreflightOwnsOptionsAndPreservesWireCauses`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileSnapshotsHeadersConcurrentReuseAndSourceChanges`.

Python/Go differences:

- Compatibility is explicitly selected by API.PythonMetadata(); these child routes are absent from stock Cinder27.0.0 router c36f40684e140a1492b00dd4812e8dce17fb2ebf. A physical stock404 remains an error and does not trigger fallback to ordinary backup GET/PUT.
- Go accepts an explicit validated ID string and returns owned physical evidence instead of a mutable Python Backup Resource; it neither mutates a native Backup cache nor synthesizes observed response metadata from submitted values.
- Go uses the configured client microversion and ordinary library-owned header options rather than Python session negotiation; Source MetadataMixin adds no per-request3.43 gate.
- Go uses map[string]string for the documented string kwargs; opaque response is returned without manufacturing the Python input-based mutable cache.
- Stock Cinder 27.0.0 registers no /backups/{id}/metadata child routes; the SDK-owned PythonMetadata profile reproduces the pinned Python request shape, error behavior and physical 404, and deployment support of those routes is not verified and not counted as stock Cinder support.

### `python:block_storage/v3/delete_backup_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.API.PythonMetadata`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/backups.PythonMetadataProfile.DeleteMetadata`

- Source keys=None PUTs metadata:{}, nonnil empty keys issue noHTTP, and other keys DELETE sequentially preserving duplicates until first error. Go retains ordered prior acknowledgements, captures literal key segments and never falls back to ordinary backup member GET/PUT.
  Tests: `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileDeleteNilEmptyLiteralKeysAndPartialFailure`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileKeepsSourceStatusesAndStock404WithoutFallback`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileAcceptedAndRejectedBodyFaultsKeepPhysicalCauses`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfilePreflightOwnsOptionsAndPreservesWireCauses`, `blockstorage/v3/backups/metadata_python_test.go:TestPythonBackupMetadataProfileSnapshotsHeadersConcurrentReuseAndSourceChanges`.

Python/Go differences:

- Compatibility is explicitly selected by API.PythonMetadata(); these child routes are absent from stock Cinder27.0.0 router c36f40684e140a1492b00dd4812e8dce17fb2ebf. A physical stock404 remains an error and does not trigger fallback to ordinary backup GET/PUT.
- Go accepts an explicit validated ID string and returns owned physical evidence instead of a mutable Python Backup Resource; it neither mutates a native Backup cache nor synthesizes observed response metadata from submitted values.
- Go uses the configured client microversion and ordinary library-owned header options rather than Python session negotiation; Source MetadataMixin adds no per-request3.43 gate.
- Go validates and escapes literal keys once; it returns partial HTTP acknowledgements rather than None and does not reproduce mutable-resource cache removal KeyError behavior.
- Stock Cinder 27.0.0 registers no /backups/{id}/metadata child routes; the SDK-owned PythonMetadata profile reproduces the pinned Python request shape, error behavior and physical 404, and deployment support of those routes is not verified and not counted as stock Cinder support.

### `python:block_storage/v3/get_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.API.Get`

- Attachments.Get sends GET attachments/{id} and decodes instance, volume_id, status and attach_mode.
  Tests: `blockstorage/v3/attachments/python_parity_test.go:TestPythonAttachmentCrudAndCompleteMatchProxyRequests`.

Python/Go differences:

- Python reads the mode attribute from a mode key while Go reads attach_mode.
- Attachments need microversion 3.27 or later; Python negotiates up to 3.71 while Go requires the client microversion to be configured.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.

### `python:block_storage/v3/update_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.WithUpdateField`

- Attachments.Update sends PUT attachments/{id} with the connector envelope.
  Tests: `blockstorage/v3/attachments/python_parity_test.go:TestPythonAttachmentCrudAndCompleteMatchProxyRequests`.

Python/Go differences:

- UpdateOpts.Connector has no omitempty, so an update without connector sends "connector": null where Python omits the key.
- Attachments need microversion 3.27 or later; Python negotiates up to 3.71 while Go requires the client microversion to be configured.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.

### `python:block_storage/v3/complete_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.API.Complete`

- Attachments.Complete posts os-complete to attachments/{id}/action.
  Tests: `blockstorage/v3/attachments/python_parity_test.go:TestPythonAttachmentCrudAndCompleteMatchProxyRequests`.

Python/Go differences:

- Python sends {"os-complete": "<attachment id>"}; Go sends {"os-complete": null}. The target comes from the URL in both.
- Go accepts only 204.
- Attachments need microversion 3.27 or later; Python negotiates up to 3.71 while Go requires the client microversion to be configured.

### `python:block_storage/v3/delete_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/attachments.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE attachments/{id}, ignores 404 by default and reports NotFound with WithMissingError.
  Tests: `blockstorage/v3/attachments/python_parity_test.go:TestPythonAttachmentCrudAndCompleteMatchProxyRequests`, `blockstorage/v3/attachments/python_parity_test.go:TestPythonAttachmentDeleteIgnoresMissing`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- Go Delete accepts only 200.
- Attachments need microversion 3.27 or later; Python negotiates up to 3.71 while Go requires the client microversion to be configured.

### `python:block_storage/v3/get_transfer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordReadHeaders`

- Transfers.Get sends GET os-volume-transfer/{id} and decodes name, volume_id and created_at.
  Tests: `blockstorage/v3/transfers/python_parity_test.go:TestPythonTransferGetAcceptDeleteUseLegacyRoute`.
- GetRecord sends bodyless GET to the discovered transfer route, seeds literal ID only for the declared Resource, retains timezone dates/unknown Wire fields and actual HTTP response proof.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferModernGetAcceptDeleteAndCleanRetries`.
- Advertised bounds and explicit/default microversion selection remain independent without mutating the shared client.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferDiscoveryAndVersionRouting`.
- Invalid route IDs/options and sticky client mutations fail before HTTP.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferInvalidOptionsAndStickySelection`.
- Every Record request observes all HTTP400..599 rejected bodies; mixed Read/Close faults retain exact native status/body/headers plus physical causes and prevent observation resend, refetch or suppression, independently of find/delete clean absence policy.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferAllOperationsRetainRejectedHTTPFaults`.

Python/Go differences:

- Go always uses the legacy /os-volume-transfer route, which Python uses only when microversion 3.55 is unavailable; with 3.55 Python uses /volume-transfers.
- Go decodes created_at only without a time zone suffix.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.
- Each Record operation performs finite shared Cinder discovery rather than relying on a Python adapter discovery cache; accepted discovery evidence is separate from actual transfer response proof.
- TransferRecord exposes owned declared Python-shaped Resource fields and original Wire/Envelope/headers/status rather than a mutable session-bound Python Resource; location is nullable null.
- Library-owned option factories snapshot JSON/maps/slices; invalid route IDs, conflicting headers, changed service sources and unsafe continuation authority are rejected before another HTTP request.

### `python:block_storage/v3/accept_transfer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.Accept`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.AcceptRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordAcceptAuthKey`

- Transfers.Accept posts {"accept": {"auth_key": ...}} to os-volume-transfer/{id}/accept and returns the transfer from the response.
  Tests: `blockstorage/v3/transfers/python_parity_test.go:TestPythonTransferGetAcceptDeleteUseLegacyRoute`.
- AcceptRecord uses discovered modern/legacy route, sends the exact accept/auth_key envelope and constructs the response Resource without seeding the input ID.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferModernGetAcceptDeleteAndCleanRetries`.
- Option ownership and invalid preflight prevent HTTP on nil options or invalid input.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferInvalidOptionsAndStickySelection`.
- Every Record request observes all HTTP400..599 rejected bodies; mixed Read/Close faults retain exact native status/body/headers plus physical causes and prevent observation resend, refetch or suppression, independently of find/delete clean absence policy.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferAllOperationsRetainRejectedHTTPFaults`.

Python/Go differences:

- Go always uses the legacy /os-volume-transfer route, which Python uses only when microversion 3.55 is unavailable; with 3.55 Python uses /volume-transfers.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.
- Each Record operation performs finite shared Cinder discovery rather than relying on a Python adapter discovery cache; accepted discovery evidence is separate from actual transfer response proof.
- TransferRecord exposes owned declared Python-shaped Resource fields and original Wire/Envelope/headers/status rather than a mutable session-bound Python Resource; location is nullable null.
- Library-owned option factories snapshot JSON/maps/slices; invalid route IDs, conflicting headers, changed service sources and unsafe continuation authority are rejected before another HTTP request.

### `python:block_storage/v3/delete_transfer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.API.DeleteRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers.WithRecordDeleteIgnoreMissing`

- Remove sends DELETE os-volume-transfer/{id}, ignores 404 by default and reports NotFound with WithMissingError.
  Tests: `blockstorage/v3/transfers/python_parity_test.go:TestPythonTransferGetAcceptDeleteUseLegacyRoute`.
- DeleteRecord uses the discovered route, ignores only clean404 by default and preserves an identical native rejection returned by a retry hook; strict missing retains the original native404.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferModernGetAcceptDeleteAndCleanRetries`.
- Rejected body Read/Close faults, cancellation and changed sources are never suppressed as missing and retain native proof and physical causes.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferFindDeleteRejectedReadCloseAndCancellation`.
- Every Record request observes all HTTP400..599 rejected bodies; mixed Read/Close faults retain exact native status/body/headers plus physical causes and prevent observation resend, refetch or suppression, independently of find/delete clean absence policy.
  Tests: `blockstorage/v3/transfers/python_records_test.go:TestPythonTransferAllOperationsRetainRejectedHTTPFaults`.

Python/Go differences:

- Go always uses the legacy /os-volume-transfer route, which Python uses only when microversion 3.55 is unavailable; with 3.55 Python uses /volume-transfers.
- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- Each Record operation performs finite shared Cinder discovery rather than relying on a Python adapter discovery cache; accepted discovery evidence is separate from actual transfer response proof.
- TransferRecord exposes owned declared Python-shaped Resource fields and original Wire/Envelope/headers/status rather than a mutable session-bound Python Resource; location is nullable null.
- Library-owned option factories snapshot JSON/maps/slices; invalid route IDs, conflicting headers, changed service sources and unsafe continuation authority are rejected before another HTTP request.
- Strict deletion returns the native Gophercloud404 cause (ResponseCodeIs) rather than synthesizing a missing-resource sentinel.

### `python:block_storage/v3/get_type`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.Get`

- VolumeTypes.Get sends GET types/{id} and decodes extra_specs, os-volume-type-access:is_public and qos_specs_id.
  Tests: `blockstorage/v3/volumetypes/python_parity_test.go:TestPythonTypeCrudAndExtraSpecsMatchProxyRequests`.

Python/Go differences:

- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/types`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithListQuery`

- VolumeTypes.List sends is_public=true with VisibilityPublic (Cinder's meaning of an absent is_public), is_public=None by default, and forwards is_public=none, all_tenants=True and sort.
  Tests: `blockstorage/v3/volumetypes/python_parity_test.go:TestPythonTypeListVisibilityAndQuery`.

Python/Go differences:

- Python types() sends no is_public, which Cinder treats as true; Go always sends is_public and defaults to None (public and private), so ListOpts{IsPublic: VisibilityPublic} is needed for Python's default result.
- Go follows volume_type_links while Python continues by marker only when limit is given, so multi-page behavior can differ.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/create_type`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithCreateField`

- VolumeTypes.Create posts the volume_type envelope with name, description, extra_specs and os-volume-type-access:is_public.
  Tests: `blockstorage/v3/volumetypes/python_parity_test.go:TestPythonTypeCrudAndExtraSpecsMatchProxyRequests`.

Python/Go differences:

- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/update_type`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.WithUpdateField`

- VolumeTypes.Update sends PUT types/{id} with only the supplied fields.
  Tests: `blockstorage/v3/volumetypes/python_parity_test.go:TestPythonTypeCrudAndExtraSpecsMatchProxyRequests`.

Python/Go differences:

- Python commit sends no request when no attribute is dirty; Go Update always sends PUT, even with an empty envelope.
- Python sends is_public under os-volume-type-access:is_public while Go UpdateOpts.IsPublic sends is_public; WithUpdateField reproduces the Python key.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/delete_type`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE types/{id}, ignores 404 by default and reports NotFound with WithMissingError.
  Tests: `blockstorage/v3/volumetypes/python_parity_test.go:TestPythonTypeCrudAndExtraSpecsMatchProxyRequests`, `blockstorage/v3/volumetypes/python_parity_test.go:TestPythonTypeDeleteIgnoresMissing`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/update_type_extra_specs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.CreateExtraSpecs`

- CreateExtraSpecs posts {"extra_specs": {...}} to types/{id}/extra_specs and returns the server's extra_specs map.
  Tests: `blockstorage/v3/volumetypes/python_parity_test.go:TestPythonTypeCrudAndExtraSpecsMatchProxyRequests`.

Python/Go differences:

- With no attributes Python sends nothing and returns Type(extra_specs={}); Go CreateExtraSpecs posts an empty map.
- Python returns a Type built from the ID and extra_specs; Go returns map[string]string.
- Python raises ValueError for non-string values; Go's map[string]string enforces this at compile time.

### `python:block_storage/v3/delete_type_extra_specs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.DeleteExtraSpec`

- Calling DeleteExtraSpec per key sends one DELETE types/{id}/extra_specs/{key} per key in order.
  Tests: `blockstorage/v3/volumetypes/python_parity_test.go:TestPythonTypeCrudAndExtraSpecsMatchProxyRequests`.

Python/Go differences:

- Python loops internally over keys; the Go caller loops and stops at the first error like Python.
- Go accepts only 202.

### `python:block_storage/v3/get_type_access`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.ListAccesses`

- ListAccesses sends GET types/{id}/os-volume-type-access and yields volume_type_id/project_id pairs.
  Tests: `blockstorage/v3/volumetypes/python_parity_test.go:TestPythonTypeAccessActions`.

Python/Go differences:

- Python returns the raw volume_type_access list of dicts; Go yields VolumeTypeAccess values with only VolumeTypeID and ProjectID.

### `python:block_storage/v3/add_type_access`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.AddAccess`

- AddAccess posts {"addProjectAccess": {"project": ...}} to types/{id}/action.
  Tests: `blockstorage/v3/volumetypes/python_parity_test.go:TestPythonTypeAccessActions`.

Python/Go differences:

- Python accepts a Project object or ID; Go takes the project ID string.
- Go accepts only 202.

### `python:block_storage/v3/remove_type_access`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.RemoveAccess`

- RemoveAccess posts {"removeProjectAccess": {"project": ...}} to types/{id}/action.
  Tests: `blockstorage/v3/volumetypes/python_parity_test.go:TestPythonTypeAccessActions`.

Python/Go differences:

- Python accepts a Project object or ID; Go takes the project ID string.
- Go accepts only 202.

### `python:block_storage/v3/get_type_encryption`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.GetEncryption`

- GetEncryption sends GET types/{id}/encryption and decodes the flat show body including encryption_id.
  Tests: `blockstorage/v3/volumetypes/python_parity_test.go:TestPythonTypeEncryptionGetDeleteAndFullBodies`.

Python/Go differences:

- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/delete_type_encryption`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.GetEncryption`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/volumetypes.API.DeleteEncryption`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- GetEncryption followed by DeleteEncryption with its EncryptionID sends the same GET and DELETE types/{id}/encryption/{encryption_id}; DeleteEncryption keeps the 404 error.
  Tests: `blockstorage/v3/volumetypes/python_parity_test.go:TestPythonTypeEncryptionGetDeleteAndFullBodies`.
- Wrapping the direct delete in resource.IgnoreMissing turns a plain HTTP 404 into success like Python's default ignore_missing=True, keeps other failures such as 409 as errors, and leaves the DELETE request unchanged.
  Tests: `resource/missing_test.go:TestIgnoreMissingDropsOnlyPlain404`, `blockstorage/v3/volumetypes/python_ignore_missing_test.go:TestPythonDeleteIgnoreMissingTypeEncryption`.

Python/Go differences:

- Python accepts either a TypeEncryption resource or volume_type and fetches the encryption itself; Go needs the type ID and encryption ID.
- Python ignore_missing=True is the default; Go selects it by wrapping the call in resource.IgnoreMissing, and the bare call returns the 404 like ignore_missing=False.

### `python:block_storage/v3/create_qos_spec`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithCreateField`

- QoS.Create posts the qos_specs envelope with name, consumer and spec keys flattened, as Python's unknown body attributes are.
  Tests: `blockstorage/v3/qos/python_parity_test.go:TestPythonQoSCrudMatchesProxyRequests`.

Python/Go differences:

- Python spec keys are passed as extra keyword attributes; Go takes them in CreateOpts.Specs or WithCreateField.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/get_qos_spec`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.Get`

- QoS.Get sends GET qos-specs/{id} and decodes name, consumer and specs.
  Tests: `blockstorage/v3/qos/python_parity_test.go:TestPythonQoSCrudMatchesProxyRequests`.

Python/Go differences:

- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/qos_specs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithListQuery`

- QoS.List sends qos-specs with the given limit and follows qos_specs_links.
  Tests: `blockstorage/v3/qos/python_parity_test.go:TestPythonQoSAssociationsKeysAndList`.

Python/Go differences:

- Python filters returned rows locally for Body attributes outside the query mapping; Go sends only query parameters and leaves local filtering to the caller.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

### `python:block_storage/v3/update_qos_spec`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithUpdateField`

- QoS.Update sends PUT qos-specs/{id} with the flattened specs and returns the response qos_specs map.
  Tests: `blockstorage/v3/qos/python_parity_test.go:TestPythonQoSCrudMatchesProxyRequests`.

Python/Go differences:

- Python commit sends no request when no attribute is dirty; Go Update always sends PUT, even with an empty envelope.
- Python returns a QoSSpec resource; Go returns map[string]string.

### `python:block_storage/v3/delete_qos_spec`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithDeleteQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- Delete with WithDeleteQuery sends Python's force=False or force=True query; Remove ignores 404 and WithMissingError reports NotFound.
  Tests: `blockstorage/v3/qos/python_parity_test.go:TestPythonQoSCrudMatchesProxyRequests`, `blockstorage/v3/qos/python_parity_test.go:TestPythonQoSDeleteMissingPolicies`.
- Wrapping the direct delete in resource.IgnoreMissing turns a plain HTTP 404 into success like Python's default ignore_missing=True, keeps other failures such as 409 as errors, and leaves the DELETE request unchanged.
  Tests: `resource/missing_test.go:TestIgnoreMissingDropsOnlyPlain404`, `blockstorage/v3/qos/python_ignore_missing_test.go:TestPythonDeleteIgnoreMissingQoSSpec`.

Python/Go differences:

- Python always sends force=False; Remove sends no force query and relies on the same server default.
- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- Python ignore_missing=True is the default; Go selects it by wrapping the call in resource.IgnoreMissing, and the bare call returns the 404 like ignore_missing=False.

### `python:block_storage/v3/associate_qos_spec`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.Associate`

- Associate sends GET qos-specs/{id}/associate?vol_type_id=....
  Tests: `blockstorage/v3/qos/python_parity_test.go:TestPythonQoSAssociationsKeysAndList`.

Python/Go differences:

- Go rejects an empty volume type ID before sending.
- Go accepts only 202.

### `python:block_storage/v3/disassociate_qos_spec`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.Disassociate`

- Disassociate sends GET qos-specs/{id}/disassociate?vol_type_id=....
  Tests: `blockstorage/v3/qos/python_parity_test.go:TestPythonQoSAssociationsKeysAndList`.

Python/Go differences:

- Go rejects an empty volume type ID before sending.
- Go accepts only 202.

### `python:block_storage/v3/disassociate_all_qos_spec`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.DisassociateAll`

- DisassociateAll sends GET qos-specs/{id}/disassociate_all.
  Tests: `blockstorage/v3/qos/python_parity_test.go:TestPythonQoSAssociationsKeysAndList`.

Python/Go differences:

- Go accepts only 202.

### `python:block_storage/v3/delete_qos_spec_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.DeleteKeys`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.WithDeleteKeysOptions`

- DeleteKeys sends PUT qos-specs/{id}/delete_keys with {"keys": [...]}.
  Tests: `blockstorage/v3/qos/python_parity_test.go:TestPythonQoSAssociationsKeysAndList`.

Python/Go differences:

- Without WithDeleteKeysOptions Go sends {"keys": null}.
- Go accepts only 202.

### `python:block_storage/v3/qos_spec_associations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/qos.API.ListAssociations`

- ListAssociations sends GET qos-specs/{id}/associations and yields association_type, id and name.
  Tests: `blockstorage/v3/qos/python_parity_test.go:TestPythonQoSAssociationsKeysAndList`.

Python/Go differences:

- Go reads only the first page while Python would follow pagination links.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud typed model, so response fields outside that model are dropped.

### `python:block_storage/v3/availability_zones`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/availabilityzones.API.List`

- AvailabilityZones.List sends one GET os-availability-zone and yields zoneName and zoneState.available.
  Tests: `blockstorage/v3/availabilityzones/python_parity_test.go:TestPythonZoneListMatchesProxyRequest`.

Python/Go differences:

- Go ZoneState decodes only available; Python keeps the whole zoneState dict.
- Python sends min(resource _max_microversion, server maximum) or the session default; Go sends the one microversion configured or negotiated on the Connection client.

## References

- [block_storage v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [SDK Support Ledger](../../docs/sdk-support-ledger.md)
- [Connector Python](connector-python.md)
- [Attachments Python](attachments-python.md)
- [Transfers Python](transfers-python.md)
- [Type Encryption Python](type-encryption-python.md)
- [Volume Summary Python](volume-summary-python.md)
- [Backup Metadata Python](backup-metadata-python.md)
- [Type/QoS](type-encryption-python.md)
- [Transfer](transfers-python.md)
- [Policy](https://docs.openstack.org/cinder/2025.2/configuration/block-storage/policy.html)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L1495-L1552)
- [27.0.0 list controller](https://github.com/openstack/cinder/blob/27.0.0/cinder/api/v3/attachments.py#L65-L106)
- [Connector](connector-python.md)
- [Attachment](attachments-python.md)
