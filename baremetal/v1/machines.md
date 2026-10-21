# Machine helpers

`Service` offers helpers that correspond to the pinned openstacksdk cloud layer (`BaremetalCloudMixin`). Each helper is an explicit composition of the node and port APIs. Nothing replays a mutation, waits implicitly, or treats an acknowledgement as completion. A machine is identified by a name or UUID string that is placed in the URL without a lookup, except in the find helpers.

| Python cloud method | Go |
|---|---|
| `list_machines` | `Service.ListMachines` |
| `get_machine` | `Service.GetMachine` (`nil, nil` when missing) |
| `get_machine_by_mac` | `Service.GetMachineByMAC` |
| `list_nics` | `Service.ListNICs` |
| `list_nics_for_machine` | `Service.ListNICsForMachine` |
| `get_nic_by_mac` | `Service.GetNICByMAC` |
| `set_machine_power_on/off/reboot` | `Service.SetMachinePowerOn/Off/Reboot` |
| `set_machine_maintenance_state`, `remove_machine_from_maintenance` | `Service.SetMachineMaintenance`, `Service.RemoveMachineFromMaintenance` |
| `patch_machine`, `set_node_instance_info` | `Service.PatchMachine` |
| `purge_node_instance_info` | `Service.PurgeMachineInstanceInfo` |
| `validate_machine`, `validate_node` | `Service.ValidateMachine`, `Service.ValidateMachineInterfaces` |
| `wait_for_baremetal_node_lock` | `Service.WaitForMachineReservation` |
| `attach_port_to_machine` | `Connection.AttachPortToMachine` |
| `detach_port_from_machine` | `Connection.DetachPortFromMachine` (returns whether a VIF was detached) |
| `list_ports_attached_to_machine` | `Connection.ListPortsAttachedToMachine` |

## Behavior

- Reads use the same record engine as `Nodes.FindRecord` and `Nodes.AllRecords`, so name fallback, duplicate detection and paging follow the node read contracts in [python-record-reads.md](nodes/python-record-reads.md). Port lists use `GET /ports/detail`; a node filter is sent as `node_uuid` and a MAC filter as `address`.
- `ListNICsForMachine`, `GetNICByMAC` and every identity argument reject an empty string before any request, because an omitted filter would otherwise list every port.
- Power helpers send `power on`, `power off` and `rebooting`. Maintenance with a nil or empty reason sends no reason; the Python call sends an explicit null, which the service treats the same way. An explicit empty-string reason is not expressible.
- `PatchMachine` sends the JSON Patch once. HTTP 409 is returned to the caller. Python retries conflicts and 503 by default; Go never replays a mutation.
- `ValidateMachineInterfaces` fails with `*MachineValidationError` only for listed interfaces that are present in the response without a true result. Interfaces absent from the response are ignored, as in Python. Failures are listed in alphabetical order because Go maps are unordered.
- `WaitForMachineReservation` always performs the first read and then polls with the shared wait options (`resource.WithPollInterval`, `resource.WithTimeout`). The lock is best effort.
- `AttachPortToMachine` finds the machine (missing fails with `resource.ErrNotFound`), finds the Network port by ID then exact name (must exist), then sends one POST. `DetachPortFromMachine` treats HTTP 400 as "not attached" and returns `false, nil`, like Python's default `ignore_missing=True`. `ListPortsAttachedToMachine` fails as a whole if any VIF cannot be found.

## Not provided

Provision-state helpers (`node_set_provision_state`, `activate_node`, `deactivate_node`, `inspect_machine`, `register_machine`, `unregister_machine`) need Python's expected-state mapping, wait loops, config-drive building and rollback; they are not implemented. `update_machine` needs a client-side JSON diff of the current node and is not implemented. Use `Nodes.ChangeProvisionState`, `Nodes.WaitForProvisionState` and `Nodes.Update` directly.

## Example

```go
service, err := conn.BareMetalV1(ctx)
if err != nil { return err }
record, err := service.GetMachine(ctx, "node-a")
if err != nil || record == nil { return err }
if err := service.SetMachinePowerOn(ctx, "node-a"); err != nil { return err }
if err := conn.AttachPortToMachine(ctx, "node-a", "port-name"); err != nil { return err }
```

Tests: `api/py_cloud_ironic_machines_test.go`, `api/py_cloud_ironic_vifs_test.go`.
