# Defaulttypes

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/default_types`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/defaulttypes.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/defaulttypes.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/defaulttypes.API.All`

- List checks the 3.67 support through one Cinder discovery GET (and version_match of a configured microversion), fails with ErrUnsupported and no list request otherwise, and sends GET default-types at min(3.67, server max) or the configured microversion.
  Tests: `blockstorage/v3/defaulttypes/api_test.go:TestDefaultTypeListRequiresMicroversion367`.

Python/Go differences:

- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- An unsupported microversion is resource.ErrUnsupported instead of SDKException; Go returns a lazy iter.Seq2 instead of a generator.

### `python:block_storage/v3/show_default_type`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/defaulttypes.API.Get`

- Get sends GET default-types/{project_id} at the negotiated 3.67 microversion and decodes the default_type envelope over the seeded project id.
  Tests: `blockstorage/v3/defaulttypes/api_test.go:TestDefaultTypeGetSetUnset`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts only the ID string and URL-escapes it as one path segment, rejecting blank, dot and control-character IDs before HTTP, while Python urljoin does not escape.
- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.

### `python:block_storage/v3/set_default_type`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/defaulttypes.API.Set`

- Set sends PUT default-types/{project_id} with {"default_type": {"volume_type": id}}, or volume_type_id for an empty id as _prepare_request_body does, and wraps 404 with operation context.
  Tests: `blockstorage/v3/defaulttypes/api_test.go:TestDefaultTypeGetSetUnset`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts only the ID string and URL-escapes it as one path segment, rejecting blank, dot and control-character IDs before HTTP, while Python urljoin does not escape.
- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- Python accepts a Type instance or ID for the volume type; Go accepts the ID string.

### `python:block_storage/v3/unset_default_type`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/defaulttypes.API.Unset`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/defaulttypes.WithDeleteIgnoreMissing`

- Unset sends DELETE default-types/{project_id} at the negotiated 3.67 microversion, ignores a clean 404 like Proxy._delete and returns 403 with operation context.
  Tests: `blockstorage/v3/defaulttypes/api_test.go:TestDefaultTypeGetSetUnset`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts only the ID string and URL-escapes it as one path segment, rejecting blank, dot and control-character IDs before HTTP, while Python urljoin does not escape.
- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- WithDeleteIgnoreMissing(false) is a Go extension; Python unset_default_type always ignores a 404.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
conn.block_storage.set_default_type(project_id, volume_type_id)
default = conn.block_storage.show_default_type(project_id)
conn.block_storage.unset_default_type(project_id)
```

```go
api := defaulttypes.New(volumeClient)
if _, err := api.Set(ctx, projectID, volumeTypeID); err != nil { return err }
dt, err := api.Get(ctx, projectID)
if err != nil { return err }
fmt.Println(*dt.VolumeTypeID)
err = api.Unset(ctx, projectID)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L738-L818)
- [`default_type.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/default_type.py)
