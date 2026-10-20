# Trove instances

`instances.API` wraps Gophercloud v2.15.0 `db/v1/instances` for `/instances`. Root-access reads are described in [root-access.md](root-access.md). Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /instances` | 200 only | Body wrapped in `instance`; `FlavorRef` and `Size >= 1` are required before any request; empty `Name`, `AvailabilityZone`, `Configuration` and `VolumeType` are omitted; `Networks` become `nics`; `Databases` and `Users` reuse the child builders |
| `Get` | `GET /instances/{id}` | 200 | Typed `Instance`; native 404 stays an HTTP response error |
| `List` | `GET /instances` | 200 (204 is an empty list) | Follows `instances_links` |
| `Delete` | `DELETE /instances/{id}` | 202, 204 | Native 404 stays an HTTP response error |
| `Restart` | `POST /instances/{id}/action` `{"restart":{}}` | 201, 202 | No input validation |
| `Resize` | `POST .../action` `{"resize":{"flavorRef":...}}` | 201, 202 | |
| `ResizeVolume` | `POST .../action` `{"resize":{"volume":{"size":n}}}` | 201, 202 | The size is sent as given, including 0 or negative; Trove validates it |
| `AttachConfigurationGroup` | `PUT /instances/{id}` `{"instance":{"configuration":id}}` | 202 only | |
| `DetachConfigurationGroup` | `PUT /instances/{id}` `{"instance":{}}` | 202 only | |
| `EnableRootUser` | `POST /instances/{id}/root` | 200 only | No request body; returns the generated root user, including its password, in the native `users.User` model |

Extension inputs use `WithCreateField`, merged into the `instance` object. A field extension that names an input already set (for example `name`, `flavorRef`, `volume`, `datastore` or `nics`) fails with `resource.ErrInvalidOption` before any request; nil options fail the same way. Gophercloud offers no extension point for the action, attach/detach, list or delete requests.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`: exact name lookup over the full list (`resource.ErrAmbiguous` for duplicates, `resource.ErrNotFound` when missing), a local `resource.WithStatus` filter that is not sent to Trove, `resource.ErrUnsupported` for `resource.WithQuery`, and ignore-missing deletion by default (`Remove` returns nil for a 404 unless `resource.WithMissingError()` is passed). `API.Delete` itself does not ignore a missing instance.

Acknowledgement of Create, Delete, Restart, Resize, ResizeVolume, Attach or Detach does not imply completion; wait on `Instance.Status` explicitly with `WaitFor`. Cancelling a context stops the local call and does not roll back remote work; mutations, including a root-user enablement that returned an error, are never replayed automatically.
