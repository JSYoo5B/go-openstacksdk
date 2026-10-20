# Native Calls

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:placement/v1/resourceproviders.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithCreateField`

- Create POSTs resource_providers with an unwrapped body that always sends name, omits empty uuid/parent_provider_uuid and merges root extensions (core-key collision or nil option fails before HTTP), sends the caller's OpenStack-API-Version placement header, accepts only 200 (so the pre-1.20 bodyless 201 is an error) and decodes the unwrapped provider.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python create_resource_provider attribute handling and microversion negotiation are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceproviders.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.Get`

- Get GETs resource_providers/{id} with the caller's placement microversion header, accepts only 200 and decodes the unwrapped provider, where {} or null yield a zero value and mistyped fields fail.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python get_resource_provider and find_resource_provider lookup are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceproviders.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.List`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithListQuery`

- List GETs resource_providers as one SinglePageBase page with name/uuid/member_of/resources/in_tree/required single-string filters percent-encoded (WithListQuery replaces a native key; ListOpts139 repeated keys are not reachable), yields the resource_providers rows, treats an empty or missing array as no rows, returns the native pager error unwrapped for 404 and io.EOF for a bodyless 204.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python resource_providers query mapping, 1.39 repeated filters and generator behavior are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceproviders.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithUpdateField`

- Update PUTs resource_providers/{id} with pointer name/parent_provider_uuid omitted when nil, rewrites an empty parent pointer to JSON null, rejects reserved-key extensions or nil options before HTTP, accepts only 200 and decodes the unwrapped provider.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python update_resource_provider attribute handling are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceproviders.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.Delete`

- Delete sends DELETE resource_providers/{id} without a body and accepts the default 202 or 204.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python delete_resource_provider ignore_missing and wait_for_delete are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceproviders.GetUsages`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.GetUsages`

- GetUsages GETs resource_providers/{id}/usages, accepts only 200 and decodes resource_provider_generation and the usages class map.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python fetch_resource_provider_usages are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceproviders.GetAllocations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.GetAllocations`

- GetAllocations GETs resource_providers/{id}/allocations, accepts only 200 and decodes resource_provider_generation and per-consumer resources.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python resource_provider_allocations are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceproviders.GetInventories`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.GetInventories`

- GetInventories GETs resource_providers/{id}/inventories, accepts only 200 and decodes resource_provider_generation and the per-class inventory map with a float32 allocation_ratio.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python resource_provider_inventories are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceproviders.UpdateInventories`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.UpdateInventories`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithUpdateInventoriesOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithUpdateInventoriesField`

- UpdateInventories PUTs resource_providers/{id}/inventories with {resource_provider_generation, inventories} where every inventory number including zero is sent, rejects reserved-key extensions before HTTP, accepts only 200 (a generation conflict 409 stays native) and decodes the inventories response.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python set_resource_provider_inventories are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceproviders.DeleteInventories`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.DeleteInventories`

- DeleteInventories sends DELETE resource_providers/{id}/inventories and accepts only 204.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python delete_resource_provider_inventories are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceproviders.GetInventory`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.GetInventory`

- GetInventory GETs resource_providers/{id}/inventories/{class}, accepts only 200 and decodes the flattened inventory fields with resource_provider_generation.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python get_resource_provider_inventory are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceproviders.UpdateInventory`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.UpdateInventory`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithUpdateInventoryOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithUpdateInventoryField`

- UpdateInventory PUTs resource_providers/{id}/inventories/{class} with the embedded inventory fields flattened beside resource_provider_generation, treats those flattened keys as reserved for extensions, accepts only 200 and decodes the flattened inventory.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python create_resource_provider_inventory and update_resource_provider_inventory are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceproviders.DeleteInventory`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.DeleteInventory`

- DeleteInventory sends DELETE resource_providers/{id}/inventories/{class} and accepts only 204.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python delete_resource_provider_inventory are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceproviders.GetTraits`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.GetTraits`

- GetTraits GETs resource_providers/{id}/traits, accepts only 200 and decodes resource_provider_generation and the trait list.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python get_resource_provider_trait are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceproviders.UpdateTraits`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.UpdateTraits`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithUpdateTraitsOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithUpdateTraitsField`

- UpdateTraits PUTs resource_providers/{id}/traits with {resource_provider_generation, traits} where nil traits are sent as null, rejects reserved-key extensions before HTTP, accepts only 200 and decodes the traits response.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python set_resource_provider_trait are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceproviders.DeleteTraits`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.DeleteTraits`

- DeleteTraits sends DELETE resource_providers/{id}/traits and accepts only 204.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python delete_resource_provider_trait are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceproviders.GetAggregates`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.GetAggregates`

- GetAggregates GETs resource_providers/{id}/aggregates, accepts only 200 and decodes the aggregate list with a pointer generation that is nil when absent.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python get_resource_provider_aggregates and fetch_resource_provider_aggregates are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceproviders.UpdateAggregates`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.UpdateAggregates`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithUpdateAggregatesOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithUpdateAggregatesField`

- UpdateAggregates PUTs resource_providers/{id}/aggregates with the {aggregates, resource_provider_generation} object when the client microversion is empty or minor >= 19 and with only the bare aggregates array (generation and extensions dropped, nil list sends no body) when minor < 19, fails before HTTP for an unparsable microversion such as latest, accepts only 200 and decodes the aggregates response.
  Tests: `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersRoutesBodiesAndDecode`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersMicroversionAndAggregatesBody`, `placement/v1/resourceproviders/native_contracts_test.go:TestNativeResourceProvidersStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceproviders call only; Python set_resource_provider_aggregates are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceclasses.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.WithCreateField`

- Create POSTs resource_classes with {name} where an empty name fails the native required check before HTTP, merges root extensions with core-key protection, accepts only 201 and returns no value.
  Tests: `placement/v1/resourceclasses/native_contracts_test.go:TestNativeResourceClassesRoutesBodiesAndDecode`, `placement/v1/resourceclasses/native_contracts_test.go:TestNativeResourceClassesStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceclasses call only; Python create_resource_class are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceclasses.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.API.Get`

- Get GETs resource_classes/{name}, accepts only 200 and decodes the unwrapped class with links, where {} or null yield a zero value.
  Tests: `placement/v1/resourceclasses/native_contracts_test.go:TestNativeResourceClassesRoutesBodiesAndDecode`, `placement/v1/resourceclasses/native_contracts_test.go:TestNativeResourceClassesStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceclasses call only; Python get_resource_class are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceclasses.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.API.List`

- List GETs resource_classes as one SinglePageBase page without query options, yields resource_classes rows, treats an empty array as no rows, returns the native pager error unwrapped for 404 and io.EOF for a bodyless 204.
  Tests: `placement/v1/resourceclasses/native_contracts_test.go:TestNativeResourceClassesRoutesBodiesAndDecode`, `placement/v1/resourceclasses/native_contracts_test.go:TestNativeResourceClassesStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceclasses call only; Python resource_classes generator are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceclasses.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.API.Update`

- Update sends a bodyless PUT resource_classes/{name} and accepts 201 or 204.
  Tests: `placement/v1/resourceclasses/native_contracts_test.go:TestNativeResourceClassesRoutesBodiesAndDecode`, `placement/v1/resourceclasses/native_contracts_test.go:TestNativeResourceClassesStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceclasses call only; Python update_resource_class are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/resourceclasses.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.API.Delete`

- Delete sends DELETE resource_classes/{name} and accepts only 204.
  Tests: `placement/v1/resourceclasses/native_contracts_test.go:TestNativeResourceClassesRoutesBodiesAndDecode`, `placement/v1/resourceclasses/native_contracts_test.go:TestNativeResourceClassesStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/resourceclasses call only; Python delete_resource_class are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/traits.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/traits.API.Create`

- Create sends a bodyless PUT traits/{name} and accepts 201 or 204.
  Tests: `placement/v1/traits/native_contracts_test.go:TestNativeTraitsRoutesQueryAndDecode`, `placement/v1/traits/native_contracts_test.go:TestNativeTraitsStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/traits call only; Python create_trait are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/traits.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/traits.API.Get`

- Get sends GET traits/{name} as an existence check that accepts only 204 and returns no value.
  Tests: `placement/v1/traits/native_contracts_test.go:TestNativeTraitsRoutesQueryAndDecode`, `placement/v1/traits/native_contracts_test.go:TestNativeTraitsStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/traits call only; Python get_trait are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/traits.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/traits.API.List`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/traits.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/traits.WithListQuery`

- List GETs traits as one SinglePageBase page with name (startswith:/in: kept verbatim and percent-encoded) and a pointer associated filter that sends false, yields trait name strings, treats an empty array as no rows, returns the native pager error unwrapped for 404 and io.EOF for a bodyless 204.
  Tests: `placement/v1/traits/native_contracts_test.go:TestNativeTraitsRoutesQueryAndDecode`, `placement/v1/traits/native_contracts_test.go:TestNativeTraitsStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/traits call only; Python traits generator are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/traits.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/traits.API.Delete`

- Delete sends DELETE traits/{name} and accepts only 204.
  Tests: `placement/v1/traits/native_contracts_test.go:TestNativeTraitsRoutesQueryAndDecode`, `placement/v1/traits/native_contracts_test.go:TestNativeTraitsStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/traits call only; Python delete_trait are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/allocations.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations.API.Get`

- Get GETs allocations/{consumer}, accepts only 200 and decodes the unwrapped allocations with nil project_id/user_id/consumer_generation/consumer_type pointers when the consumer has no allocations.
  Tests: `placement/v1/allocations/native_contracts_test.go:TestNativeAllocationsRoutesBodiesAndDecode`, `placement/v1/allocations/native_contracts_test.go:TestNativeAllocationsStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/allocations call only; Python get_allocation are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/allocations.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations.WithUpdateField`

- Update PUTs allocations/{consumer} with an unwrapped body that always sends allocations, project_id, user_id and consumer_generation (nil as null for a new consumer) and omits empty consumer_type, rejects reserved-key extensions (even omitted ones) or nil options before HTTP, and accepts only 204 so a consumer generation 409 stays native.
  Tests: `placement/v1/allocations/native_contracts_test.go:TestNativeAllocationsRoutesBodiesAndDecode`, `placement/v1/allocations/native_contracts_test.go:TestNativeAllocationsStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/allocations call only; Python create_allocations and update_allocation are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/allocations.Manage`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations.API.Manage`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations.WithManageOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations.WithManageField`

- Manage POSTs allocations with the consumer-keyed UpdateOpts map serialized as is (nil map sends null), places extensions inside the only consumer when exactly one is present and at the root otherwise, and accepts only 204.
  Tests: `placement/v1/allocations/native_contracts_test.go:TestNativeAllocationsRoutesBodiesAndDecode`, `placement/v1/allocations/native_contracts_test.go:TestNativeAllocationsStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/allocations call only; Python batch allocation workflows are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/allocations.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations.API.Delete`

- Delete sends DELETE allocations/{consumer} and accepts the default 202 or 204.
  Tests: `placement/v1/allocations/native_contracts_test.go:TestNativeAllocationsRoutesBodiesAndDecode`, `placement/v1/allocations/native_contracts_test.go:TestNativeAllocationsStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/allocations call only; Python delete_allocation are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:placement/v1/allocationcandidates.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocationcandidates.API.List`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocationcandidates.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocationcandidates.WithListQuery`

- List GETs allocation_candidates with repeated required/member_of/same_subtree keys, single resources/in_tree/group_policy/limit/root_required values and suffixed resource group keys, all sorted and percent-encoded, yields one value per non-empty SinglePageBase page and decodes it with the current ExtractAllocationCandidates provider-keyed shape, including 1.34 mappings and 1.17/1.29 summary fields, while 1.10-1.11 array allocations fail with a JSON type error.
  Tests: `placement/v1/allocationcandidates/native_contracts_test.go:TestNativeAllocationCandidatesQueryAndDecode`, `placement/v1/allocationcandidates/native_contracts_test.go:TestNativeAllocationCandidatesStatusesAndPreflight`.
- sdkgen prefers an unsuffixed page extractor over one whose name only adds a microversion digit suffix, and keeps the field-detail rule for differently named extractors.
  Tests: `internal/cmd/sdkgen/page_extractors_test.go:TestPageExtractorPrefersUnsuffixedOverMicroversionSuffix`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/allocationcandidates call only; Python allocation_candidates are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.
- Responses for microversions 1.10 and 1.11 need native List with ExtractAllocationCandidates110 through RawClient.

### `gophercloud:placement/v1/usages.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/usages.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/usages.WithGetOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/usages.WithGetQuery`

- Get GETs usages with project_id/user_id/consumer_type query values (an empty project_id is omitted because it has no required tag), accepts only 200 and decodes the 1.38+ consumer-type grouped usages with native Extract, so the flat 1.9-1.37 shape is a decode error.
  Tests: `placement/v1/usages/native_contracts_test.go:TestNativeUsagesQueryAndDecode`, `placement/v1/usages/native_contracts_test.go:TestNativeUsagesStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native placement/v1/usages call only; Python usages and the pre-1.38 ExtractPre138 shape are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [resourceproviders](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/placement/v1/resourceproviders/requests.go)
- [resourceclasses](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/placement/v1/resourceclasses/requests.go)
- [traits](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/placement/v1/traits/requests.go)
- [allocations](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/placement/v1/allocations/requests.go)
- [allocationcandidates](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/placement/v1/allocationcandidates/requests.go)
- [usages](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/placement/v1/usages/requests.go)
