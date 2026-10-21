# Bare Metal Python-level operations (nodes, ports, allocations, machines)

This guide describes SDK-owned helpers that reproduce the pinned Python
openstacksdk behavior of the Bare Metal proxy and cloud layer. The helpers send
exactly one request per call, never replay a mutation and treat an
acknowledgement (202/204) only as acceptance, never as completion. Deadlines and
cancellation come from the `context.Context`; local cancellation does not roll
back remote work. Tests live in `api/r2_ironic_c_*_test.go`.

## Microversions

Python asserts a minimum version per call. The helpers use two rules from the
pinned source:

- *Assert* (`inject_nmi`, VIF, firmware, vmedia, provision state, reset
  interfaces, soft power, create with a provision state): a configured client
  microversion is the actual version; an unset one is replaced by the highest
  version shared with the endpoint, at most 1.109 (one discovery read). The call
  fails with `resource.ErrUnsupported` before the request if the actual version
  is below the minimum.
- *Pick* (boot mode, secure boot, traits): a configured version that matches the
  minimum is kept, a lower one fails, and an unset one is replaced by the minimum
  when the endpoint range contains it.

The call then sends that version. Operations without a minimum send the
configured client microversion unchanged; Python would send the negotiated
maximum. Use `WithLatestMicroversion` or `WithMicroversionRange` on the
connection for that behavior.

## Nodes (`baremetal/v1/nodes`)

| Python | Go |
|---|---|
| `create_node` | `API.CreateWithProvisionState` (state `enroll`, `manageable`, `available`; never sent in the body; `manageable` follows with `manage` and a wait) |
| `update_node(id, **attrs)` | `API.UpdateAttributes` (every attribute is an `add` operation sorted by path; maintenance routed to the maintenance endpoint), `API.UpdateFromOriginal` (structural difference against an earlier read) |
| `patch_node` | `API.PatchNode` with `PatchOp`, `WithResetInterfaces` |
| `set_node_provision_state` | `API.SetProvisionState`, `ProvisionStateRequest`, `WithProvisionStateWait` |
| `wait_for_nodes_provision_state` | `API.AwaitProvisionState`, `API.AwaitProvisionStates`, `StateReached`, `WithAbortOnFailedState`, `WithContinueOnFailure` |
| `set_node_power_state` | `API.SetPowerState`, `WithPowerStateWait` |
| `inject_nmi_to_node` | `API.InjectNMIChecked` |
| `list_node_vifs` / `detach_vif_from_node` | `API.ListVIFIDs` / `API.DetachVIF` |
| `detach_vmedia_from_node` | `API.DetachVirtualMediaJSON` |
| `list_node_firmware` / `list_node_vendor_passthru` | `API.ListFirmwareRaw` / `API.ListVendorPassthruRaw` |
| `set_node_boot_mode` / `set_node_secure_boot` | `API.SetBootMode` / `API.SetSecureBoot` |
| `set_node_traits` / `remove_node_trait` | `API.SetTraits` / `API.RemoveTrait` |

Behavior notes:

- `UpdateAttributes` names attributes by Python attribute or wire name. Unknown
  names, the identity field and two names for one field fail before HTTP, where
  Python silently drops unknown names. A truthy `maintenance_reason` without
  maintenance fails; with a false `maintenance` the reason is dropped. Only 200
  (PATCH) and 202 (maintenance) succeed; a 409 conflict is returned once, while
  Python retries it by default.
- `PatchNode` converts only the first path segment from attribute name to wire
  name. An empty patch sends nothing (`PatchResult.Sent` is false). With
  `WithResetInterfaces` the query is `reset_interfaces=True` or `False`, as
  Python encodes the boolean, and microversion 1.45 is asserted.
- `UpdateFromOriginal` computes a sorted structural difference like jsonpatch
  (objects by key, arrays by index) but never emits `move` operations.
- `SetProvisionState` validates the request before HTTP (config drive only with
  active or rebuild, clean steps only with clean, deploy steps only with active or
  rebuild, service steps only with service, rescue password only with rescue, a
  runbook only with clean or service and not with steps). `nil` slices and a nil
  rescue password are omitted; empty ones are sent. Config drive and runbook are
  sent only when non-empty, like the Python truthiness checks. Minimum versions:
  verb table, config drive map 1.56, rebuild drive 1.35, deploy steps 1.69,
  runbook 1.92. It returns the node read after the request, or after the wait.
  Python accepts any 2xx; Go accepts 202 only.
- `StateReached` treats an empty provision state as "available", fails on a state
  ending in " failed" or equal to "error", and on "enroll" with a `last_error`
  while "manageable" is expected. The errors are `*ProvisionFailureError`
  (`errors.Is` with `resource.ErrFailedState`).
- `AwaitProvisionStates` reads every pending node in order each round. By default
  the first failure is returned with the result so far. `WithContinueOnFailure`
  is Python's `fail=False`: failures and a context deadline are collected in
  `ProvisionWaitResult`.
- `DetachVirtualMediaJSON` sends the Python wire form: `DELETE /vmedia` with the
  JSON body `{"device_types": "a,b"}`. The existing `DetachVirtualMedia` sends a
  `device_types` query parameter instead. This SDK does not verify that a server
  treats the two forms alike; use the method that matches your server. HTTP 400
  stays an error in both, as in Python (which only logs it).
- `DetachVIF` and `RemoveTrait` treat HTTP 400 as an already-absent resource and
  return `false` with a nil error unless the ignore-missing option is false.
- `ListFirmwareRaw` and `ListVendorPassthruRaw` return unchanged JSON, including
  unknown fields.

## Ports, allocations, conductors

- `ports.API.GetRecord`, `FindRecord`, `ListRecords`, `AllRecords`,
  `ProxyListRecords`, `allocations.API.GetRecord` and
  `conductors.API.GetRecord` follow the node record contracts in
  `nodes/python-record-reads.md`: field names map to wire names and are sent as a
  comma-separated `fields` query, versions are capped at 1.101 (ports), 1.60
  (allocations) and 1.49 (conductors) when the client has none, and a missing
  resource stays a strict 404. Port find tries `GET /ports/{id}` and falls back
  to the detailed list (no name query) with duplicate detection. Port list
  accepts declared query names and aliases (`node_id` is sent as `node_uuid`) and
  discards names Python does not declare, such as `sort_key`.
- `ports.API.UpdateAttributes` / `PatchPort` and
  `allocations.API.UpdateAttributes` / `PatchAllocation` use the same patch rules
  as the node helpers. The allocation PATCH binding is SDK-owned because
  Gophercloud has none; it needs microversion 1.57, which the client must carry.
- `allocations.API.WaitForActive` reads immediately and then per interval until
  the state is not `allocating`; `error` fails with `*AllocationFailedError`
  unless `WithIgnoreError(true)`.
- `drivers.API.ListVendorPassthruRaw` reads the driver first (404 attribution)
  and then the method list.

## Machines and generic waits (`baremetal/v1`)

- `Service.NodeSetProvisionState`, `ActivateNode`, `DeactivateNode`: config drive
  is passed through as a string; the SDK does not read paths or build images (the
  pinned code does not either, despite its docstring). Timeouts default to 3600 s
  (set state) and 1200 s (activate, deactivate) and apply to the whole call; use
  `WithMachineTimeout` or `WithMachineUnlimitedWait`.
- `Service.InspectMachine` always waits (Python ignores its `wait` argument),
  moves an unassociated `available` machine through manage/inspect/provide and
  refuses associated or wrongly-staged machines with `ErrMachineState`.
- `Service.RegisterMachine` creates the node (initial state `enroll` unless the
  target is `available`), manages and waits, waits for the reservation lock,
  creates ports and optionally provides. On any later failure it deletes the
  created ports and the node using a detached context bounded to 30 seconds and
  reports the outcome in `*RegisterError`.
- `Service.UnregisterMachine` refuses active/cleaning states, waits for the lock,
  deletes the first port of each address and the node; 404 on deletes is ignored.
- `Service.UpdateMachine` returns the changed attribute paths (sorted, attribute
  names) and the updated node; no difference means no write.
- `WaitForStatus` and `WaitForDelete` wrap `resource.Collection.Wait` and
  `WaitDeleted` with the Python defaults (2 s interval, failure state `ERROR`;
  delete 120 s). Python's default attribute `status` does not exist on Bare
  Metal resources; the collection's status accessor is used.
