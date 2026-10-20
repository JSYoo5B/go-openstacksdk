# Group Reset Status Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/reset_group_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.API.ResetStatusRecord`

- ResetStatusRecord posts {"reset_status": {"status": status}} to groups/{id}/action at min(3.38, server max) or the configured microversion, accepts every status below 400 and wraps 400+ with operation context while keeping the actual response.
  Tests: `blockstorage/v3/groups/python_reset_test.go:TestGroupResetStatusRecordPostsAction`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts only the ID string and URL-escapes it as one path segment, rejecting blank, dot and control-character IDs before HTTP, while Python urljoin does not escape.
- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Per-request header options are a Go extension; authentication, microversion and content headers stay SDK owned.
- Python returns None; Go returns a ResetStatusRecordResult with the reference and actual response.

### `python:block_storage/v3/reset_group_state`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groups.API.ResetStatusRecord`

- The deprecated alias sends the same POST groups/{id}/action reset_status request through ResetStatusRecord.
  Tests: `blockstorage/v3/groups/python_reset_test.go:TestGroupResetStatusRecordPostsAction`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts only the ID string and URL-escapes it as one path segment, rejecting blank, dot and control-character IDs before HTTP, while Python urljoin does not escape.
- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- The RemovedInSDK50Warning deprecation warning is not reproduced.
- Python returns None; Go returns a ResetStatusRecordResult with the reference and actual response.

### `python:block_storage/v3/reset_group_snapshot_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.API.ResetStatusRecord`

- ResetStatusRecord posts {"reset_status": {"status": status}} to group_snapshots/{id}/action with the empty Accept header of GroupSnapshot._action at min(3.29, server max) or the configured microversion, accepting every status below 400.
  Tests: `blockstorage/v3/groupsnapshots/python_reset_test.go:TestGroupSnapshotResetStatusRecordPostsActionWithEmptyAccept`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts only the ID string and URL-escapes it as one path segment, rejecting blank, dot and control-character IDs before HTTP, while Python urljoin does not escape.
- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Per-request header options are a Go extension; authentication, microversion and content headers stay SDK owned.
- Python returns None; Go returns a ResetStatusRecordResult with the reference and actual response.

### `python:block_storage/v3/reset_group_snapshot_state`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/groupsnapshots.API.ResetStatusRecord`

- The deprecated alias sends the same POST group_snapshots/{id}/action reset_status request with an empty Accept header through ResetStatusRecord.
  Tests: `blockstorage/v3/groupsnapshots/python_reset_test.go:TestGroupSnapshotResetStatusRecordPostsActionWithEmptyAccept`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts only the ID string and URL-escapes it as one path segment, rejecting blank, dot and control-character IDs before HTTP, while Python urljoin does not escape.
- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- The RemovedInSDK50Warning deprecation warning is not reproduced.
- Python returns None; Go returns a ResetStatusRecordResult with the reference and actual response.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
conn.block_storage.reset_group_status(group_id, "available")
conn.block_storage.reset_group_snapshot_status(snapshot_id, "error")
```

```go
if _, err := groups.New(volumeClient).ResetStatusRecord(ctx, groupID, "available"); err != nil { return err }
if _, err := groupsnapshots.New(volumeClient).ResetStatusRecord(ctx, snapshotID, "error"); err != nil { return err }
```

## References

- [`_proxy.py` reset_group_status](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2250-L2272)
- [`reset_group_snapshot_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2756-L2786)
- [`group.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group.py)
- [`group_snapshot.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group_snapshot.py)
