# Native Admin Flavors Servers

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:compute/v2/flavors.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithCreateField`

- Create POSTs flavors with {"flavor":{...}} rejecting only an empty name or nil disk before HTTP (zero ram/vcpus are sent), merges extensions inside the envelope while refusing core keys, decodes the flavor including string swap values and accepts only 200/201.
  Tests: `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminRoutesBodiesAndDecode`, `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/flavors call only; Python compute flavor admin parity and Cloud flavor helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/flavors.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithUpdateField`

- Update PUTs flavors/{id} with {"flavor":{"description":...}} (an empty description leaves {"flavor":{}}), refuses the description extension key before HTTP, decodes the flavor and accepts only 200.
  Tests: `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminRoutesBodiesAndDecode`, `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/flavors call only; Python compute flavor admin parity and Cloud flavor helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/flavors.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.Delete`

- Delete sends DELETE flavors/{id} without a body and accepts only 202/204.
  Tests: `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminRoutesBodiesAndDecode`, `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/flavors call only; Python compute flavor admin parity and Cloud flavor helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/flavors.AddAccess`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.AddAccess`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithAddAccessOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithAddAccessField`

- AddAccess POSTs flavors/{id}/action with {"addTenantAccess":{"tenant":...}} sending an empty tenant as "", merges extensions inside the action object while refusing the tenant key, decodes the full flavor_access list and accepts only 200.
  Tests: `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminRoutesBodiesAndDecode`, `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/flavors call only; Python compute flavor admin parity and Cloud flavor helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/flavors.RemoveAccess`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.RemoveAccess`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithRemoveAccessOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithRemoveAccessField`

- RemoveAccess POSTs flavors/{id}/action with {"removeTenantAccess":{"tenant":...}}, refuses the tenant extension key before HTTP, decodes the remaining flavor_access list (an empty array is an empty slice) and accepts only 200.
  Tests: `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminRoutesBodiesAndDecode`, `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/flavors call only; Python compute flavor admin parity and Cloud flavor helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/flavors.ListAccesses`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.ListAccesses`

- ListAccesses GETs flavors/{id}/os-flavor-access as a single page that ignores links, streams flavor_access rows, yields nothing for an empty array, io.EOF for a bodyless 204 and unwrapped ErrUnexpectedResponseCode outside 200/204/300.
  Tests: `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminRoutesBodiesAndDecode`, `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminAccessListStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/flavors call only; Python compute flavor admin parity and Cloud flavor helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/flavors.CreateExtraSpecs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.CreateExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithCreateExtraSpecsOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithCreateExtraSpecsField`

- CreateExtraSpecs POSTs flavors/{id}/os-extra_specs with {"extra_specs":{...}}, places extensions beside extra_specs at the top level and refuses an extra_specs extension before HTTP, decodes the extra_specs map and accepts only 200.
  Tests: `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminRoutesBodiesAndDecode`, `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/flavors call only; Python compute flavor admin parity and Cloud flavor helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/flavors.UpdateExtraSpec`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.UpdateExtraSpec`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithUpdateExtraSpecOptions`

- UpdateExtraSpec PUTs the single key/value pair to flavors/{id}/os-extra_specs/{key} with the unescaped key, rejects zero or multiple pairs before HTTP, decodes the bare response map and accepts only 200.
  Tests: `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminRoutesBodiesAndDecode`, `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/flavors call only; Python compute flavor admin parity and Cloud flavor helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/flavors.DeleteExtraSpec`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.DeleteExtraSpec`

- DeleteExtraSpec sends DELETE flavors/{id}/os-extra_specs/{key} without a body and accepts only 200.
  Tests: `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminRoutesBodiesAndDecode`, `compute/v2/flavors/native_admin_contracts_test.go:TestNativeFlavorAdminStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/flavors call only; Python compute flavor admin parity and Cloud flavor helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/servers.Evacuate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Evacuate`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithEvacuateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithEvacuateField`

- Evacuate POSTs servers/{id}/action with {"evacuate":{...}} always including onSharedStorage regardless of microversion, merges extensions inside the action while refusing core keys before HTTP, returns adminPass or an empty string for {} or a bodyless 200, and accepts only 200.
  Tests: `compute/v2/servers/native_admin_action_contracts_test.go:TestNativeServerAdminActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_admin_action_contracts_test.go:TestNativeServerAdminActionsStrictStatuses`, `compute/v2/servers/native_admin_action_contracts_test.go:TestNativeServerAdminActionsPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/servers call only; Python compute server admin action parity and Cloud server helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/servers.ForceDelete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ForceDelete`

- ForceDelete POSTs {"forceDelete":""} to servers/{id}/action and accepts only 201/202.
  Tests: `compute/v2/servers/native_admin_action_contracts_test.go:TestNativeServerAdminActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_admin_action_contracts_test.go:TestNativeServerAdminActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/servers call only; Python compute server admin action parity and Cloud server helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/servers.InjectNetworkInfo`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.InjectNetworkInfo`

- InjectNetworkInfo POSTs {"injectNetworkInfo":null} to servers/{id}/action and accepts only 201/202.
  Tests: `compute/v2/servers/native_admin_action_contracts_test.go:TestNativeServerAdminActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_admin_action_contracts_test.go:TestNativeServerAdminActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/servers call only; Python compute server admin action parity and Cloud server helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/servers.LiveMigrate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.LiveMigrate`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithLiveMigrateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithLiveMigrateField`

- LiveMigrate POSTs {"os-migrateLive":{...}} with host always present (null when nil) and block_migration/disk_over_commit omitted when nil, refuses a block_migration "auto" or other core-key extension before HTTP, merges other extensions inside the action and accepts only 201/202.
  Tests: `compute/v2/servers/native_admin_action_contracts_test.go:TestNativeServerAdminActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_admin_action_contracts_test.go:TestNativeServerAdminActionsStrictStatuses`, `compute/v2/servers/native_admin_action_contracts_test.go:TestNativeServerAdminActionsPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/servers call only; Python compute server admin action parity and Cloud server helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/servers.Migrate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Migrate`

- Migrate POSTs {"migrate":null} to servers/{id}/action without a host option and accepts only 201/202.
  Tests: `compute/v2/servers/native_admin_action_contracts_test.go:TestNativeServerAdminActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_admin_action_contracts_test.go:TestNativeServerAdminActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/servers call only; Python compute server admin action parity and Cloud server helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/servers.ResetNetwork`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ResetNetwork`

- ResetNetwork POSTs {"resetNetwork":null} to servers/{id}/action and accepts only 201/202.
  Tests: `compute/v2/servers/native_admin_action_contracts_test.go:TestNativeServerAdminActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_admin_action_contracts_test.go:TestNativeServerAdminActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/servers call only; Python compute server admin action parity and Cloud server helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:compute/v2/servers.ResetState`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ResetState`

- ResetState POSTs {"os-resetState":{"state":...}} passing any state string through unvalidated and accepts only 201/202.
  Tests: `compute/v2/servers/native_admin_action_contracts_test.go:TestNativeServerAdminActionsRoutesBodiesAndDecode`, `compute/v2/servers/native_admin_action_contracts_test.go:TestNativeServerAdminActionsStrictStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/servers call only; Python compute server admin action parity and Cloud server helpers are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/flavors/requests.go)
- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/servers/requests.go)
- [Flavor Property And Native](../flavor-property-and-native.md)
- [Native](servers/native.md)
