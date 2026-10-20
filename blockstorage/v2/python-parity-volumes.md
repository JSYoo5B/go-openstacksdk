# Python Parity Volumes

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v2/get_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.Get`

- Volumes.Get sends GET volumes/{id} on the v2 resource base without a microversion header and decodes name, size, bootable, multiattach, host, tenant, consistency group and metadata.
  Tests: `blockstorage/v2/volumes/python_parity_test.go:TestPythonVolumeGetUpdateMatchProxyRequests`.

Python/Go differences:

- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- The v2 Volume model has no volume_image_metadata, os-vol-mig-status-attr or os-volume-replication fields that the Python Volume exposes.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

### `python:block_storage/v2/update_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithUpdateField`

- Volumes.Update sends PUT volumes/{id} with the supplied name, description and extension fields inside the volume envelope, and still sends an empty envelope when nothing is supplied.
  Tests: `blockstorage/v2/volumes/python_parity_test.go:TestPythonVolumeGetUpdateMatchProxyRequests`.

Python/Go differences:

- Python commit sends no request when no attribute is dirty; Go Update always sends PUT, even with an empty envelope.
- Attributes outside UpdateOpts need WithUpdateField with the wire key.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

### `python:block_storage/v2/delete_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithDeleteQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.ForceDelete`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- resource.IgnoreMissing around Volumes.Delete with WithDeleteQuery(cascade, False) sends DELETE volumes/{id}?cascade=False and turns only 404 into success while 409 stays an error, and Remove ignores 404 unless WithMissingError is given.
  Tests: `blockstorage/v2/volumes/python_parity_test.go:TestPythonVolumeDeleteDefaultsCascadeForceAndMissing`.
- WithDeleteQuery(cascade, True) sends cascade=True and ForceDelete posts os-force_delete to volumes/{id}/action for force=True.
  Tests: `blockstorage/v2/volumes/python_parity_test.go:TestPythonVolumeDeleteDefaultsCascadeForceAndMissing`.

Python/Go differences:

- Python always sends cascade=False or cascade=True; Remove sends no query and DeleteOpts{Cascade: true} sends lowercase cascade=true, so the exact Python query needs WithDeleteQuery.
- Python ignore_missing=True is the default; Go selects it with Remove or by wrapping Delete in resource.IgnoreMissing, and the bare Delete returns the 404 like ignore_missing=False.
- ForceDelete sends {"os-force_delete": ""} where Python sends null; Python's force path catches NotFound but its v2 action never raises, whereas Go ForceDelete returns HTTP errors.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

### `python:block_storage/v2/create_volume`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithCreateField`

- Volumes.Create posts the volume envelope with size, name, snapshot_id, imageRef and WithCreateField keys such as multiattach and OS-SCH-HNT:scheduler_hints inside it, but always sends size (0 when unset) and rejects a size override before HTTP.
  Tests: `blockstorage/v2/volumes/python_parity_test.go:TestPythonVolumeCreateEnvelopeAlwaysSendsSize`.

Python/Go differences:

- The v2 Python Volume keeps scheduler_hints as a Body attribute inside the envelope; WithCreateHintOpts puts it outside, so WithCreateField must be used.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/volumes`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.WithListQuery`

- Volumes.List sends GET volumes/detail with the Python query mapping (all_tenants=True, name, status, project_id) and follows volumes_links like the default details=True call.
  Tests: `blockstorage/v2/volumes/python_parity_test.go:TestPythonVolumeListDetailDefaultOnly`.

Python/Go differences:

- Python all_projects=True becomes all_tenants=True; typed ListOpts.AllTenants sends all_tenants=true, so the exact string needs WithListQuery.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/find_volume`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`

- Find by ID with WithIgnoreMissing then Find by Name sends GET volumes/{x} and, after 404, GET volumes/detail?name=x like the default find_volume, while a 403 on the ID lookup is returned as an error.
  Tests: `blockstorage/v2/volumes/python_parity_test.go:TestPythonVolumeFindChainUsesDetailNameQuery`.

Python/Go differences:

- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/extend_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.ExtendSize`

- ExtendSize posts {"os-extend": {"new_size": N}} to volumes/{id}/action and reports a 400 response as an error.
  Tests: `blockstorage/v2/volumes/python_parity_test.go:TestPythonVolumeActionBodiesMatchProxyRequests`.

Python/Go differences:

- Python v2 Volume._action does not call raise_from_response, so HTTP error responses return None silently; Go accepts only the fixed native success status and returns other responses as errors.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

### `python:block_storage/v2/retype_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.ChangeType`

- ChangeType posts os-retype with new_type and migration_policy never or on-demand, matching the Python default never when MigrationPolicyNever is given.
  Tests: `blockstorage/v2/volumes/python_parity_test.go:TestPythonVolumeActionBodiesMatchProxyRequests`.

Python/Go differences:

- Python applies migration_policy=never by default; Go omits migration_policy unless the caller sets MigrationPolicyNever.
- Python new_type accepts a Type object or ID; Go takes the type ID or name string.
- Python v2 Volume._action does not call raise_from_response, so HTTP error responses return None silently; Go accepts only the fixed native success status and returns other responses as errors.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

### `python:block_storage/v2/set_volume_bootable_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.SetBootable`

- SetBootable posts {"os-set_bootable": {"bootable": false}} including an explicit false.
  Tests: `blockstorage/v2/volumes/python_parity_test.go:TestPythonVolumeActionBodiesMatchProxyRequests`.

Python/Go differences:

- Go accepts only HTTP 200 for this action.
- Python v2 Volume._action does not call raise_from_response, so HTTP error responses return None silently; Go accepts only the fixed native success status and returns other responses as errors.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

### `python:block_storage/v2/set_volume_image_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.SetImageMetadata`

- SetImageMetadata posts os-set_image_metadata with the metadata map, including an empty map as {}.
  Tests: `blockstorage/v2/volumes/python_parity_test.go:TestPythonVolumeActionBodiesMatchProxyRequests`.

Python/Go differences:

- Python **metadata always yields a dict; a nil Go map is sent as metadata:null, so callers pass an empty map for no keys.
- Go accepts only HTTP 200 and decodes the response metadata map.
- Python v2 Volume._action does not call raise_from_response, so HTTP error responses return None silently; Go accepts only the fixed native success status and returns other responses as errors.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

### `python:block_storage/v2/attach_volume`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.Attach`

- Attach posts os-attach with mountpoint and either instance_uuid or host_name, as Volume.attach does.
  Tests: `blockstorage/v2/volumes/python_parity_test.go:TestPythonVolumeActionBodiesMatchProxyRequests`.

Python/Go differences:

- Python prefers instance over host_name and raises ValueError when both are missing; Go sends whatever AttachOpts holds, so the caller sets exactly one.
- An empty mountpoint is sent as "" by Python and omitted by Go.
- Python v2 Volume._action does not call raise_from_response, so HTTP error responses return None silently; Go accepts only the fixed native success status and returns other responses as errors.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

### `python:block_storage/v2/detach_volume`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.Detach`

- Detach posts {"os-detach": {"attachment_id": id}} for the default force=False call.
  Tests: `blockstorage/v2/volumes/python_parity_test.go:TestPythonVolumeActionBodiesMatchProxyRequests`.

Python/Go differences:

- Python v2 Volume._action does not call raise_from_response, so HTTP error responses return None silently; Go accepts only the fixed native success status and returns other responses as errors.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/reset_volume_status`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.ResetStatus`

- ResetStatus posts os-reset_status with status, attach_status and migration_status, but sends status as an empty string when only attach_status is supplied.
  Tests: `blockstorage/v2/volumes/python_parity_test.go:TestPythonVolumeActionBodiesMatchProxyRequests`.

Python/Go differences:

- Python v2 Volume._action does not call raise_from_response, so HTTP error responses return None silently; Go accepts only the fixed native success status and returns other responses as errors.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/upload_volume_to_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.UploadImage`

- UploadImage posts os-volume_upload_image with image_name, force, disk_format and container_format and decodes image_id, volume id and status from the os-volume_upload_image response.
  Tests: `blockstorage/v2/volumes/python_parity_test.go:TestPythonVolumeUploadAndCreateImage`.

Python/Go differences:

- Python always sends force, disk_format and container_format, including false and null; Go omits empty values and relies on the server defaults.
- Python returns the raw os-volume_upload_image dict; Go returns the typed VolumeImage.
- Python calls .json() without raise_from_response, so an error body fails with KeyError; Go accepts only HTTP 202 and returns an error.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

### `python:block_storage/v2/create_image`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/volumes.API.UploadImage`
- `github.com/JSYoo5B/go-openstacksdk/image.Service.ExistingImageRecord`

- UploadImage with force=allow_duplicates, disk_format qcow2 and container_format bare followed by image.Service.ExistingImageRecord(image_id) sends one action request and builds the image record without HTTP.
  Tests: `blockstorage/v2/volumes/python_parity_test.go:TestPythonVolumeUploadAndCreateImage`.

Python/Go differences:

- Python fills an empty disk_format from the cloud image_format setting (default qcow2) and container_format with bare; the v2 Go call does not read cloud settings, so the caller passes both values.
- Python sends force:false explicitly; Go omits a false Force.
- The unused wait and timeout parameters have no Go counterpart.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

### `python:block_storage/v2/set_volume_readonly`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/delete_volume_image_metadata`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/unmanage_volume`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/migrate_volume`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/complete_volume_migration`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/get_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.API.Get`

- Snapshots.Get sends GET snapshots/{id} and decodes name, description, volume_id, status, size, metadata and created_at.
  Tests: `blockstorage/v2/snapshots/python_parity_test.go:TestPythonSnapshotGetCreateMatchProxyRequests`.

Python/Go differences:

- The v2 Snapshot model has no force field that the Python Snapshot exposes as is_forced.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

### `python:block_storage/v2/create_snapshot`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.WithCreateField`

- Snapshots.Create posts the snapshot envelope with volume_id, name, description, force and metadata, omits a false force and rejects force as an extension field before HTTP.
  Tests: `blockstorage/v2/snapshots/python_parity_test.go:TestPythonSnapshotGetCreateMatchProxyRequests`.

Python/Go differences:

- Python is_forced=False is sent as "force": false; Go omits a false Force and the server default is the same.
- Go requires CreateOpts.VolumeID before HTTP; Python leaves validation to the server.
- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

### `python:block_storage/v2/delete_snapshot`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Snapshots.Remove sends DELETE snapshots/{id}, ignores 404 by default, keeps 409 as an error and reports NotFound with WithMissingError.
  Tests: `blockstorage/v2/snapshots/python_parity_test.go:TestPythonSnapshotDeleteIgnoreMissingDefault`.

Python/Go differences:

- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/snapshots`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.WithListQuery`

- Snapshots.List sends GET snapshots with the Python query mapping (all_tenants=True, name, volume_id) but reads only one page and ignores snapshots_links.
  Tests: `blockstorage/v2/snapshots/python_parity_test.go:TestPythonSnapshotListAndFindUseSummaryRoute`.

Python/Go differences:

- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/find_snapshot`

Status: `unsupported`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/snapshots.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`

- Find by ID with WithIgnoreMissing then Find by Name sends GET snapshots/{x} and, after 404, GET snapshots?name=x on the summary route.
  Tests: `blockstorage/v2/snapshots/python_parity_test.go:TestPythonSnapshotListAndFindUseSummaryRoute`.

Python/Go differences:

- Python accepts a Resource object or ID and returns a mutable Resource; Go takes an ID string and returns a Gophercloud v2 typed model, so response fields outside that model are dropped.
- Cinder v2 has no microversions: Python sends no version header and the v2 client from BlockStorageV2 has no microversion, so Go sends none either.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/update_snapshot`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/reset_snapshot_status`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/reset_snapshot`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/manage_snapshot`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/unmanage_snapshot`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/delete_qos_spec_metadata`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/get_consistency_group_snapshot`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/find_consistency_group_snapshot`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/consistency_group_snapshots`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/create_consistency_group_snapshot`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

### `python:block_storage/v2/delete_consistency_group_snapshot`

Status: `unsupported`.

Unresolved boundaries:

- Cinder removed the v2 API in Xena (19.0.0) and no supported deployment serves this route, so the operation is excluded from the SDK target instead of being implemented.

## References

- [block_storage v2 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py)
- [Volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/volume.py)
- [Snapshot](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/snapshot.py)
- [Python Parity Test](volumes/python_parity_test.go)
- [Python Parity Test](snapshots/python_parity_test.go)
- [`get_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L594)
- [`find_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L626)
- [`volumes`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L666)
- [`create_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L694)
- [`update_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L705)
- [`delete_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L720)
- [`extend_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L761)
- [`set_volume_readonly`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L773)
- [`retype_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L788)
- [`set_volume_bootable_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L811)
- [`set_volume_image_metadata`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L826)
- [`delete_volume_image_metadata`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L841)
- [`reset_volume_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L861)
- [`attach_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L882)
- [`detach_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L902)
- [`unmanage_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L923)
- [`migrate_volume`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L935)
- [`complete_volume_migration`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L961)
- [`upload_volume_to_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L980)
- [`create_image`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L59)
- [`get_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L91)
- [`find_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L126)
- [`snapshots`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L170)
- [`create_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L201)
- [`update_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L212)
- [`delete_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L225)
- [`reset_snapshot_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L255)
- [`reset_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L269)
- [`manage_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L279)
- [`unmanage_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L291)
- [`delete_qos_spec_metadata`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L1436)
- [`get_consistency_group_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L2231)
- [`find_consistency_group_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L2270)
- [`consistency_group_snapshots`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L2303)
- [`create_consistency_group_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L2330)
- [`delete_consistency_group_snapshot`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/_proxy.py#L2346)
- [..](../README.md)
- [`Volume._action`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v2/volume.py#L109)
- [v3 consistencygroupsnapshots](../v3/consistency-group-snapshots-python.md)
