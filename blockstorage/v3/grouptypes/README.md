# Grouptypes

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/get_group_type`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.New`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.API.Get`

- Get sends GET group_types/{id} at min(3.11, server max) after one discovery GET, or at the configured microversion without discovery, and decodes the group_type envelope or root object with group_specs defaulting to {}.
  Tests: `blockstorage/v3/grouptypes/api_test.go:TestGroupTypeGetNegotiatesMicroversion311`.
- Get accepts every status below 400, wraps 400+ with operation context, fails a non-object JSON body with the accepted response and keeps the seeded id for a non-JSON body.
  Tests: `blockstorage/v3/grouptypes/api_test.go:TestGroupTypeStatusPolicyAndRejections`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts only the ID string and URL-escapes it as one path segment, rejecting blank, dot and control-character IDs before HTTP, while Python urljoin does not escape.
- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- Per-request header options are a Go extension; authentication, microversion and content headers stay SDK owned.

### `python:block_storage/v3/find_group_type`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.WithFindIgnoreMissing`

- Find sends GET group_types/{name_or_id}?is_public=none and after a clean 400/403/404 lists group_types?is_public=none matching exact id or name, returns nil for a missing type by default, ErrNotFound with WithFindIgnoreMissing(false), and AmbiguousError for duplicates.
  Tests: `blockstorage/v3/grouptypes/api_test.go:TestGroupTypeFindUsesIsPublicNone`.

Python/Go differences:

- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- The Resource location attribute and the find_* deprecation warning are not reproduced.
- A missing type with ignore_missing=False is resource.ErrNotFound instead of NotFoundException; duplicates are resource.AmbiguousError instead of DuplicateResource.

### `python:block_storage/v3/group_types`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.API.List`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.API.All`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.WithListPaginated`

- List sends GET group_types with Accept application/json, maps is_public and all_projects (as all_tenants) to Requests-encoded query, filters id/name/description/group_specs locally, ignores unknown names, follows group_types_links and honors limit, marker and paginated.
  Tests: `blockstorage/v3/grouptypes/api_test.go:TestGroupTypeListQueryFiltersAndPaging`.

Python/Go differences:

- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- Go returns a lazy iter.Seq2 (or All for a slice) instead of a generator; local filters compare JSON values without Python type conversion.

### `python:block_storage/v3/create_group_type`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.WithCreateOpts`

- Create posts {"group_type": attrs} with only declared attributes to group_types at the negotiated 3.11 microversion and decodes the response.
  Tests: `blockstorage/v3/grouptypes/api_test.go:TestGroupTypeCreatePostsEnvelope`.

Python/Go differences:

- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- Per-request header options are a Go extension; authentication, microversion and content headers stay SDK owned.
- Python **attrs become WithCreateField options holding raw JSON; aliases of one Body key with different values are rejected before HTTP.

### `python:block_storage/v3/update_group_type`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.WithUpdateOpts`

- Update sends PUT group_types/{id} with only the dirty declared attributes, never the id, and sends neither discovery nor update HTTP when nothing is dirty.
  Tests: `blockstorage/v3/grouptypes/api_test.go:TestGroupTypeUpdateSendsDirtyAttributesOnly`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts only the ID string and URL-escapes it as one path segment, rejecting blank, dot and control-character IDs before HTTP, while Python urljoin does not escape.
- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- Per-request header options are a Go extension; authentication, microversion and content headers stay SDK owned.

### `python:block_storage/v3/delete_group_type`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.WithDeleteIgnoreMissing`

- Delete sends DELETE group_types/{id}, ignores a clean 404 by default, returns it with WithDeleteIgnoreMissing(false) and always returns 409 with operation context.
  Tests: `blockstorage/v3/grouptypes/api_test.go:TestGroupTypeDeleteIgnoresMissingByDefault`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts only the ID string and URL-escapes it as one path segment, rejecting blank, dot and control-character IDs before HTTP, while Python urljoin does not escape.
- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Per-request header options are a Go extension; authentication, microversion and content headers stay SDK owned.

### `python:block_storage/v3/fetch_group_type_group_specs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.API.FetchGroupSpecs`

- FetchGroupSpecs sends GET group_types/{id}/group_specs at the negotiated 3.11 microversion and returns the group type seeded with the id and the response group_specs, failing a non-object or non-JSON body.
  Tests: `blockstorage/v3/grouptypes/api_test.go:TestGroupTypeGroupSpecs`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts only the ID string and URL-escapes it as one path segment, rejecting blank, dot and control-character IDs before HTTP, while Python urljoin does not escape.
- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- The returned GroupType is a new value holding only the id and group_specs rather than the updated input instance.

### `python:block_storage/v3/create_group_type_group_specs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.API.CreateGroupSpecs`

- CreateGroupSpecs posts {"group_specs": specs} to group_types/{id}/group_specs and defaults an absent response group_specs to {}.
  Tests: `blockstorage/v3/grouptypes/api_test.go:TestGroupTypeGroupSpecs`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts only the ID string and URL-escapes it as one path segment, rejecting blank, dot and control-character IDs before HTTP, while Python urljoin does not escape.
- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Python returns a mutable Resource; Go returns a typed struct with raw Body/Wire/Envelope, Header, StatusCode and Discovery evidence.
- The specs are a Go map[string]any marshaled as JSON.

### `python:block_storage/v3/get_group_type_group_specs_property`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.API.GetGroupSpecsProperty`

- GetGroupSpecsProperty sends GET group_types/{id}/group_specs/{prop} and returns the prop member as raw JSON, or JSON null when the member is absent.
  Tests: `blockstorage/v3/grouptypes/api_test.go:TestGroupTypeGroupSpecs`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts only the ID string and URL-escapes it as one path segment, rejecting blank, dot and control-character IDs before HTTP, while Python urljoin does not escape.
- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Python returns the decoded value or None; Go returns a SpecProperty whose Value is raw JSON plus the actual response.

### `python:block_storage/v3/update_group_type_group_specs_property`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.API.UpdateGroupSpecsProperty`

- UpdateGroupSpecsProperty sends PUT group_types/{id}/group_specs/{prop} with {prop: val}, returns the response prop member and fails like the Python KeyError when it is absent.
  Tests: `blockstorage/v3/grouptypes/api_test.go:TestGroupTypeGroupSpecs`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts only the ID string and URL-escapes it as one path segment, rejecting blank, dot and control-character IDs before HTTP, while Python urljoin does not escape.
- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Python returns the decoded value; Go returns a SpecProperty whose Value is raw JSON plus the actual response.

### `python:block_storage/v3/delete_group_type_group_specs_property`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.API.DeleteGroupSpecsProperty`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes.WithDeleteIgnoreMissing`

- DeleteGroupSpecsProperty sends DELETE group_types/{id}/group_specs/{prop}, ignores a clean 404 by default, returns it with WithDeleteIgnoreMissing(false) and returns 403 with operation context.
  Tests: `blockstorage/v3/grouptypes/api_test.go:TestGroupTypeGroupSpecs`.

Python/Go differences:

- Python accepts a Resource instance or an ID string; Go accepts only the ID string and URL-escapes it as one path segment, rejecting blank, dot and control-character IDs before HTTP, while Python urljoin does not escape.
- Python negotiates min(Resource max, server max) through keystoneauth discovery; Go sends the configured client microversion or performs one Cinder version discovery GET with the same cap, and the result keeps the discovery responses.
- Per-request header options are a Go extension; authentication, microversion and content headers stay SDK owned.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
gt = conn.block_storage.create_group_type(name="gold", is_public=True)
conn.block_storage.create_group_type_group_specs(gt, {"consistent_group_snapshot_enabled": "<is> True"})
conn.block_storage.delete_group_type(gt)
```

```go
api := grouptypes.New(volumeClient)
gt, err := api.Create(ctx,
    grouptypes.WithCreateField("name", "gold"),
    grouptypes.WithCreateField("is_public", true))
if err != nil { return err }
_, err = api.CreateGroupSpecs(ctx, gt.ID, map[string]any{"consistent_group_snapshot_enabled": "<is> True"})
if err != nil { return err }
err = api.Delete(ctx, gt.ID)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L2809-L3025)
- [`group_type.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/group_type.py)
