# Nodes

`Nodes.GetVirtualMedia` returns SDK-owned VirtualMedia with image, insertion state, and media types. Node record APIs have their own microversion discovery/selection controls; consult their operation guides instead of inferring behavior from native Get/List. The selected version and actual response remain distinct from advertised maximum versions.

Use the concrete API and operation-specific options in this package. Parent scopes fix the caller-selected resource; optional request values and missing-resource behavior are defined per operation.

Consult the [support ledger](../../../docs/sdk-support-ledger.md) and [operation reviews](../../../api/sdk_reviews.json) for supported behavior and unresolved boundaries.

## Gophercloud-backed node operations

`nodes.API` wraps Gophercloud v2.15.0 `baremetal/v1/nodes` for the `/nodes` routes. Python openstacksdk Resource and workflow equivalence is tracked separately. Every route below is relative to the baremetal endpoint (`/v1`).

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /nodes` | 201, 202 | Flat body (no envelope); `*bool` fields (`AutomatedClean`, `DisablePowerOff`) send an explicit `false` |
| `Get`, `Validate` | `GET /nodes/{id}`, `/validate` | 200 | Typed model; unknown response fields are not retained |
| `Update` | `PATCH /nodes/{id}` | 200 only | `UpdateOpts` is a JSON Patch list; `Value` of `""`, `false`, `0` and `nil` is sent as written, `remove` carries no value |
| `Delete` | `DELETE /nodes/{id}` | 202, 204 | Native 404 stays an HTTP response error |
| `List`, `ListDetail` | `GET /nodes`, `/nodes/detail` | 200 | Follows `nodes_links` next links; `ListDetail` rejects `Fields` before any request |
| `ChangeProvisionState` | `PUT /nodes/{id}/states/provision` | 202 only | `target` is required; configdrive, clean/deploy/service steps and `rescue_password` are omitted when unset |
| `ChangePowerState` | `PUT /nodes/{id}/states/power` | 202 only | `target` is required; `timeout` is omitted when 0 |
| `SetMaintenance`, `UnsetMaintenance` | `PUT`/`DELETE /nodes/{id}/maintenance` | 202 only | An empty `Reason` is omitted; unset sends no body |
| `SetBootDevice`, `GetBootDevice`, `GetSupportedBootDevices` | `/nodes/{id}/management/boot_device[/supported]` | 204 / 200 / 200 | `persistent` is always sent, including `false` |
| `InjectNMI` | `PUT /nodes/{id}/management/inject_nmi` | 204 only | Sends the body `{}` |
| `SetRAIDConfig` | `PUT /nodes/{id}/states/raid` | 204 only | An unset `SizeGB` is sent as `"MAX"` |
| `ListBIOSSettings`, `GetBIOSSetting` | `GET /nodes/{id}/bios[/{setting}]` | 200 | `Detail` and `Fields` together fail before any request |
| `GetVendorPassthruMethods`, `ListVendorPassthruMethods` | `GET /nodes/{id}/vendor_passthru/methods` | 200 | Two result shapes of the same route |
| `CallVendorPassthru` | `{httpMethod} /nodes/{id}/vendor_passthru?method=` | 200, 202, 204 | `Method` is required; the status code is not returned, a 204 yields an empty map |
| `CreateSubscription`, `GetSubscription`, `GetAllSubscriptions`, `DeleteSubscription` | `POST`, `GET`, `GET`, `DELETE` on `.../vendor_passthru?method=` | 200 / 200 / 200 / 200, 202, 204 | `GET` and `DELETE` subscriptions carry a JSON body `{"id":...}`; the vendor method name is taken from `CallVendorPassthruOpts` and is not validated for subscriptions |
| `GetInventory`, `ListFirmware` | `GET /nodes/{id}/inventory`, `/firmware` | 200 | Nullable response fields stay pointers |
| `AttachVirtualMedia`, `DetachVirtualMedia`, `GetVirtualMedia` | `POST`, `DELETE`, `GET /nodes/{id}/vmedia` | 204 / 204 / 200 | `DeviceTypes` become the `device_types` comma-separated query; `GetVirtualMedia` also keeps the raw body |
| `ListVirtualInterfaces`, `AttachVirtualInterface`, `DetachVirtualInterface` | `/nodes/{id}/vifs[/{vif}]` | 200 / 204 / 204 | `PortUUID` with `PortgroupUUID` fails before any request |
| `WaitForProvisionState` | repeated `GET /nodes/{id}` | n/a | Polls about once per second until `provision_state` matches |

Extension inputs: `WithCreateField`, `WithChangeProvisionStateField`, `WithChangePowerStateField`, `WithSetMaintenanceField`, `WithSetBootDeviceField`, `WithSetRAIDConfigField`, `WithAttachVirtualInterfaceField`, `WithAttachVirtualMediaField` add JSON fields; `WithListQuery`, `WithListDetailQuery`, `WithListBIOSSettingsQuery`, `WithDetachVirtualMediaQuery` add query values. A field extension that names a core input fails with `resource.ErrInvalidOption` before any request; nil options fail the same way. `Update` has no field extension.

Limits and exact behavior:

- Microversions: the generated API never chooses one. Gophercloud sends `X-OpenStack-Ironic-API-Version` only when the `ServiceClient.Microversion` it wraps is set. Several options need newer versions (for example BIOS `Detail`/`Fields` 1.74, virtual media read 1.93, structured configdrive 1.56, deploy steps 1.69, node `Health` filter 1.109); the API does not check this and the server decides.
- Typed `ListOpts` booleans (`Maintenance`, `Associated`) are plain `bool`: `false` is omitted. Send an explicit `maintenance=false` through `WithListQuery`.
- Provision-state, power-state and maintenance calls return on the 202 acknowledgement only. They do not poll and acknowledgement does not mean completion; read `provision_state`, `power_state` or the node afterwards. `WaitForProvisionState` does not treat failed states (for example `deploy failed`) as terminal; bound it with a context deadline.
- A cancelled or expired context stops the local call and does not roll back remote work. Failed or uncertain mutations (including 409 and 503 responses) are never replayed automatically.
- `GetBIOSSetting` decodes the body from a `setting` key as Gophercloud does; a body keyed by the setting name decodes to an empty setting.
- `Get`, `Delete` and the other native calls keep a 404 as `gophercloud.ErrUnexpectedResponseCode`. Only `Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection` policies; `Remove` ignores a missing node by default.

## API reference

- [api_generated.go](api_generated.go)
- [python_list_records.go](python_list_records.go)
- [python_list_records_profile.go](python_list_records_profile.go)
- [python_list_records_proxy.go](python_list_records_proxy.go)
- [resources_generated.go](resources_generated.go)
- [virtual_media.go](virtual_media.go)
- [python_record_reads.go](python_record_reads.go)

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
conn, err := sdk.Connect(ctx, sdk.WithCloud("dev"),
    sdk.WithMicroversion("baremetal", "1.93"))
if err != nil { return err }
service, err := conn.BareMetal(ctx)
if err != nil { return err }
media, err := service.Nodes.GetVirtualMedia(ctx, nodeID)
if err != nil { return err }
fmt.Println(media.Image, media.Inserted, media.MediaTypes)
```

## References

- [Source-shaped Node reads](python-record-reads.md)
