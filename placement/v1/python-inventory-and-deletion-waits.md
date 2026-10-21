# Inventory creation and deletion waits (Placement)

`resourceproviders.API.CreateInventory(ctx, providerID, resourceClass, opts)`
sends `POST /resource_providers/{id}/inventories` with
`{"resource_class", "total", ...}` and requires 201. `total` is always sent
(zero included); `reserved`, `min_unit`, `max_unit`, `step_size` and
`allocation_ratio` are omitted when nil. Extension fields cannot replace those
or `resource_class`; reserved headers are rejected before HTTP. No microversion
header is added beyond what the client already carries. A failure is never
replayed; after cancellation the class may exist, so read the inventories first.
It differs from `UpdateInventory` (PUT of an existing class) and
`UpdateInventories` (PUT replacing every class).

`resourceproviders.API.InventoryResources(providerID)` and `traits.API.Resources()`
return collections bound to direct GET/DELETE, so `WaitDeleted` completes on 404
like Python `wait_for_delete`. They have no listing, name lookup or status
attribute: inventory attributes are numeric, and `resource.WithStatusAttribute`
selects only string fields.

Tests: `api/r2_small_placement_test.go`.
