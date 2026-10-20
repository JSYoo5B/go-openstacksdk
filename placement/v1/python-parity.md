# Python Parity

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:placement/v1/allocation_candidates`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocationcandidates.API.List`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocationcandidates.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocationcandidates.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocationcandidates.WithListQueryValues`
- `github.com/JSYoo5B/go-openstacksdk/request.WithQueryValues`

- At placement 1.34 List sends GET allocation_candidates with repeated required and member_of, single resources, limit and group_policy, and suffixed resources1, required1, member_of1 and in_tree1, and decodes every allocation request with its mappings and the provider summaries.
  Tests: `placement/v1/allocationcandidates/python_parity_test.go:TestPythonAllocationCandidatesQueryAndCandidates`.
- WithListQueryValues sends a key once per value in order, such as member_of1=agg-4&member_of1=agg-5 and same_subtree, replacing the ListOpts serialization of that key, while a later WithListQuery for the same key wins.
  Tests: `placement/v1/allocationcandidates/python_list_query_test.go:TestPythonAllocationCandidatesRepeatedQueryValues`.
- WithListQueryValues with a blank key fails before HTTP as resource.ErrInvalidOption wrapped in a List resource.OperationError, and a 400 response to a repeated query is returned as the Gophercloud status error.
  Tests: `placement/v1/allocationcandidates/python_list_query_test.go:TestPythonAllocationCandidatesRepeatedQueryRejections`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.34.
- Python yields one AllocationCandidate per allocation request with provider_summaries reduced to that candidate's providers; Go yields the whole response once and the caller selects ProviderSummaries by the AllocationRequests[i].Allocations keys.
- Python accepts every status below 400 and also decodes the 1.10-1.11 list response shape; Go accepts 200, 204 and 300 through Gophercloud paging and decodes only the 1.12+ shape.
- Python consumes paginated, base_path, jmespath_filters, allow_unknown_params and microversion as call controls rather than query keys; Go has no equivalents and takes the microversion from the client.

### `python:placement/v1/get_allocation`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations.API.Get`

- At placement 1.38 Get sends GET allocations/{consumer} and returns per-provider generation and resources, consumer_generation, consumer_type, project_id and user_id.
  Tests: `placement/v1/allocations/python_parity_test.go:TestPythonAllocationProxyCalls`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.38.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.
- Go returns *Allocations with nil pointers for absent consumer fields instead of an Allocation Resource; HTTP errors stay native gophercloud errors instead of openstack.exceptions.

### `python:placement/v1/update_allocation`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations.WithUpdateField`

- At placement 1.38 Update sends PUT allocations/{consumer} with allocations, project_id, user_id, consumer_generation as null for a new consumer, and consumer_type.
  Tests: `placement/v1/allocations/python_parity_test.go:TestPythonAllocationProxyCalls`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.38.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.
- Python sends only the Body attributes passed in attrs; Go always sends allocations, project_id, user_id and consumer_generation and omits only an empty consumer_type. Placement 1.38 requires all five keys, so valid calls produce the same body.
- Python returns the Allocation built from attrs; Go returns only an error because the 204 response has no body.

### `python:placement/v1/create_allocations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations.WithCreateRecordHeader`

- At placement 1.38 CreateRecord posts the caller's per-consumer objects unchanged to POST allocations, so mappings, null consumer_generation and an empty allocations map stay inside each consumer, sends {} for a nil or empty map and null for a nil consumer, and accepts 200 and 204.
  Tests: `placement/v1/allocations/python_create_record_test.go:TestPythonAllocationCreateRecordSendsPerConsumerKeys`.
- A 409 response is returned as the Gophercloud status error wrapped in resource.OperationError, and values encoding/json cannot encode, a nil option or a nil API fail before HTTP.
  Tests: `placement/v1/allocations/python_create_record_test.go:TestPythonAllocationCreateRecordRejections`.
- At placement 1.38 Manage sends POST allocations with each consumer's allocations, project_id, user_id, consumer_generation and consumer_type, and an empty non-nil allocations map clears a consumer.
  Tests: `placement/v1/allocations/python_parity_test.go:TestPythonAllocationProxyCalls`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.38.
- Python returns None; Go returns CreateRecordResult with the response headers and status code.
- Python sends no body when allocations is None; Go sends {} for a nil consumer map.

### `python:placement/v1/delete_allocation`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocations.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- At placement 1.38 Delete sends DELETE allocations/{consumer} and succeeds on 204.
  Tests: `placement/v1/allocations/python_parity_test.go:TestPythonAllocationProxyCalls`.
- Delete reports HTTP 404 as a native gophercloud.ErrUnexpectedResponseCode.
  Tests: `placement/v1/allocations/python_parity_test.go:TestPythonAllocationDeleteMissing`.
- Wrapping the direct delete in resource.IgnoreMissing turns a plain HTTP 404 into success like Python's default ignore_missing=True, keeps other failures such as 409 as errors, and leaves the DELETE request unchanged.
  Tests: `resource/missing_test.go:TestIgnoreMissingDropsOnlyPlain404`, `placement/v1/allocations/python_ignore_missing_test.go:TestPythonDeleteIgnoreMissingAllocation`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.38.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.
- Python ignore_missing=True is the default; Go selects it by wrapping the call in resource.IgnoreMissing, and the bare call returns the 404 like ignore_missing=False.

### `python:placement/v1/create_resource_class`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.API.Create`

- At placement 1.2 Create sends POST resource_classes with {"name": ...} and accepts the bodyless 201.
  Tests: `placement/v1/resourceclasses/python_parity_test.go:TestPythonResourceClassProxyCalls`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.2.
- Python returns a ResourceClass built from the attrs; Go returns only an error, and the 201 response carries no body.
- Go requires a non-empty Name before HTTP; Python sends whatever Body attributes it was given.

### `python:placement/v1/delete_resource_class`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove ignores HTTP 404 by default like Python ignore_missing=True, resource.WithMissingError reports resource.ErrNotFound like ignore_missing=False, and both send one DELETE to the same path and microversion. The route is DELETE resource_classes/{name} at 1.2.
  Tests: `placement/v1/resourceclasses/python_parity_test.go:TestPythonResourceClassProxyCalls`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.2.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.
- API.Delete without the collection keeps the 404 as an error.

### `python:placement/v1/update_resource_class`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.WithUpdateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.WithUpdateRecordName`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.WithUpdateRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/request.Present`
- `github.com/JSYoo5B/go-openstacksdk/request.Null`

- UpdateRecord sends PUT resource_classes/{name} at placement 1.2 even when the client selects 1.39, with {"name": new}, {} without a new name or {"name": null}, accepts 200, 201 and 204, decodes the 200 body into the typed ResourceClass and raw fields while an empty body leaves both nil, and leaves the client microversion unchanged.
  Tests: `placement/v1/resourceclasses/python_update_record_test.go:TestPythonResourceClassUpdateRecordRenames`.
- A 409 response is returned as the Gophercloud status error wrapped in resource.OperationError, a non-object JSON body is returned with resource.ResponseError, and invalid names, a conflicting version header, nil options and a nil API fail before HTTP.
  Tests: `placement/v1/resourceclasses/python_update_record_test.go:TestPythonResourceClassUpdateRecordRejections`.

Python/Go differences:

- Python sends min(1.2, server maximum) or the session default microversion; Go always sends 1.2 for this call because 1.7 and later turn the PUT into a bodyless ensure-exists call.
- Python accepts a ResourceClass or a name string; Go takes the current name string.
- Python returns the Resource merged from request attributes and the response, so its id keeps the old name; Go returns ResourceClassRecord with the native typed decode, the raw response object fields, headers and status.

### `python:placement/v1/get_resource_class`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.API.Get`

- At placement 1.2 Get sends GET resource_classes/{name} and returns the name; 404 stays a native error.
  Tests: `placement/v1/resourceclasses/python_parity_test.go:TestPythonResourceClassProxyCalls`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.2.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.
- Go also returns links, which the Python Resource does not declare.

### `python:placement/v1/resource_classes`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.API.List`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.API.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithName`

- At placement 1.2 List and All send GET resource_classes without a query, and All with WithName filters the name locally after the same request.
  Tests: `placement/v1/resourceclasses/python_parity_test.go:TestPythonResourceClassProxyCalls`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.2.
- Python drops unknown query keys, forwards limit and marker, and filters Body attributes such as name locally; Go List accepts no query and the collection rejects other query keys, but placement ignores limit and marker on this route.

### `python:placement/v1/create_resource_provider`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithCreateField`

- At placement 1.20 Create sends POST resource_providers with name, uuid and parent_provider_uuid and decodes the 200 provider body.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderProxyCRUD`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.20.
- Python id and parent_provider_id map to Go CreateOpts.UUID and ParentProviderUUID; Go always sends name, other Body keys need WithCreateField.
- Go accepts only 200, matching the 1.20 Python negotiation; below 1.20 the bodyless 201 is an error.

### `python:placement/v1/delete_resource_provider`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove ignores HTTP 404 by default like Python ignore_missing=True, resource.WithMissingError reports resource.ErrNotFound like ignore_missing=False, and both send one DELETE to the same path and microversion. The route is DELETE resource_providers/{id} at 1.20.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderProxyCRUD`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.20.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.
- API.Delete without the collection keeps the 404 as an error.

### `python:placement/v1/update_resource_provider`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithUpdateField`

- At placement 1.20 Update sends PUT resource_providers/{id} with name, and an empty ParentProviderUUID pointer sends parent_provider_uuid null like parent_provider_id=None.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderProxyCRUD`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.20.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.
- Python skips the request when nothing is dirty; Go sends {} for empty UpdateOpts.

### `python:placement/v1/get_resource_provider`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.Get`

- At placement 1.20 Get sends GET resource_providers/{id} and returns uuid, name, generation, parent, root and links; 404 stays a native error.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderProxyCRUD`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.20.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.

### `python:placement/v1/find_resource_provider`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`

- Find(ID) sends GET resource_providers/{value}; after a 404 with WithIgnoreMissing, or a 400 or 403 error, Find(Name) sends GET resource_providers?name={value} and returns the match or nil, reproducing Python's ID-then-name lookup with ignore_missing=True.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderFindIDThenName`.
- Without WithIgnoreMissing both Find(ID) and Find(Name) return resource.ErrNotFound like ignore_missing=False.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderFindIDThenName`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.20.
- Python performs the GET-then-list fallback in one call and also falls back on 400 and 403; Go needs two explicit Find calls and the caller decides to continue after 400 or 403.
- Python defaults ignore_missing=True (deprecated toward False); Go Find defaults to the missing error and needs WithIgnoreMissing.
- Go rejects ID references containing /, ?, #, % or whitespace before HTTP; such values go straight to the name lookup.

### `python:placement/v1/resource_providers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.List`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.All`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithName`

- At placement 1.20 List with ListOpts sends name, uuid, member_of, resources, in_tree and required as one value each, and no query when unset.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderProxyCRUD`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.20.
- Python maps id to uuid, drops unknown keys and filters other Body attributes locally; Go forwards extra keys with WithListQuery and leaves Body filtering to the caller except exact name via resource.WithName.
- Repeated member_of or required keys (1.24+/1.39) cannot be sent from Go; Python can send them only after raising the session microversion above its 1.20 default.

### `python:placement/v1/fetch_resource_provider_aggregates`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.GetAggregates`

- At placement 1.20 GetAggregates sends GET resource_providers/{id}/aggregates and returns the aggregates with the generation.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderAggregatesUsagesAllocationsTraits`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.20.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.
- Python returns a ResourceProvider with only aggregates and generation filled; Go returns ResourceProviderAggregates with a generation pointer.

### `python:placement/v1/get_resource_provider_aggregates`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.GetAggregates`

- At placement 1.20 GetAggregates sends GET resource_providers/{id}/aggregates and returns the aggregates with the generation.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderAggregatesUsagesAllocationsTraits`.

Python/Go differences:

- Python emits RemovedInSDK50Warning and delegates to fetch_resource_provider_aggregates; Go has a single non-deprecated GetAggregates.
- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.20.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.

### `python:placement/v1/set_resource_provider_aggregates`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.UpdateAggregates`

- At placement 1.20 UpdateAggregates sends {"aggregates": [...], "resource_provider_generation": N}, an empty non-nil slice sends [], and the returned generation and aggregates are decoded.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderAggregatesUsagesAllocationsTraits`.
- Below 1.19 UpdateAggregates sends the bare aggregate list and decodes no generation.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderAggregatesUsagesAllocationsTraits`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.20.
- Python reads the generation from the passed ResourceProvider and sends null for a bare ID; Go takes it in UpdateAggregatesOpts and omits a nil pointer. Placement rejects both.
- Python aggregates default to []; a nil Go slice is sent as null.
- Python stores the PUT response generation under resource_provider_generation, leaving generation unchanged; Go returns it in ResourceProviderGeneration.

### `python:placement/v1/create_resource_provider_inventory`

Status: `unresolved`.

Python/Go differences:

- Python sends POST resource_providers/{id}/inventories without a microversion header and returns the 201 inventory body.

Unresolved boundaries:

- Go has no call for POST resource_providers/{id}/inventories; UpdateInventory only replaces an existing class and UpdateInventories replaces every class, which are different requests.

### `python:placement/v1/delete_resource_provider_inventory`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.DeleteInventory`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- DeleteInventory sends DELETE resource_providers/{id}/inventories/{class} without a microversion header, succeeds on 204 and reports 404 as a native error.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderInventories`.
- Wrapping the direct delete in resource.IgnoreMissing turns a plain HTTP 404 into success like Python's default ignore_missing=True, keeps other failures such as 409 as errors, and leaves the DELETE request unchanged.
  Tests: `resource/missing_test.go:TestIgnoreMissingDropsOnlyPlain404`, `placement/v1/resourceproviders/python_ignore_missing_test.go:TestPythonDeleteIgnoreMissingResourceProvider`.

Python/Go differences:

- Python accepts an inventory Resource carrying resource_provider_id or the class name plus provider; Go takes both strings.
- Python ignore_missing=True is the default; Go selects it by wrapping the call in resource.IgnoreMissing, and the bare call returns the 404 like ignore_missing=False.

### `python:placement/v1/update_resource_provider_inventory`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.UpdateInventory`

- UpdateInventory sends PUT resource_providers/{id}/inventories/{class} without a microversion header, with the generation and all six inventory fields, and decodes the new generation.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderInventories`.

Python/Go differences:

- Python sends only the attrs it was given plus resource_provider_generation and placement fills defaults; Go always sends all six fields including zeros, so the caller supplies placement's defaults for fields Python would omit.
- Python sends resource_provider_generation null when not given; Go always sends the int. Go allocation_ratio is float32.

### `python:placement/v1/get_resource_provider_inventory`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.GetInventory`

- GetInventory sends GET resource_providers/{id}/inventories/{class} without a microversion header and decodes the generation and inventory fields.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderInventories`.

Python/Go differences:

- Python keeps the class name in id; Go returns only the fields in the response.
- Go allocation_ratio is float32.

### `python:placement/v1/resource_provider_inventories`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.GetInventories`

- GetInventories sends GET resource_providers/{id}/inventories without a microversion header and returns every class with the provider generation.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderInventories`.

Python/Go differences:

- Python yields one ResourceProviderInventory per class and filters Body attribute query arguments locally; Go returns one map keyed by class and the caller filters.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.

### `python:placement/v1/set_resource_provider_inventories`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.UpdateInventories`

- At placement 1.20 UpdateInventories sends PUT resource_providers/{id}/inventories with the generation and class map, an empty non-nil map clears all inventories, and the new generation is decoded.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderInventories`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python uses ResourceProvider's 1.20.
- Python inventory dicts may omit fields that placement defaults; Go Inventory always sends all six fields, so the caller supplies the defaults.
- Python returns the ResourceProvider with the new generation; Go returns ResourceProviderInventories.

### `python:placement/v1/delete_resource_provider_inventories`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.DeleteInventories`

- At placement 1.20 DeleteInventories sends DELETE resource_providers/{id}/inventories and succeeds on 204.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderInventories`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python uses ResourceProvider's 1.20.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.

### `python:placement/v1/fetch_resource_provider_usages`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.GetUsages`

- At placement 1.20 GetUsages sends GET resource_providers/{id}/usages and returns the usage counts.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderAggregatesUsagesAllocationsTraits`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.20.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.
- Python returns a ResourceProvider with usages filled; Go returns ResourceProviderUsage including the generation.

### `python:placement/v1/resource_provider_allocations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.GetAllocations`

- GetAllocations sends GET resource_providers/{id}/allocations without a microversion header and returns every consumer's resources with the generation.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderAggregatesUsagesAllocationsTraits`.

Python/Go differences:

- Python yields one ResourceProviderAllocation per consumer and filters Body attribute query arguments locally; Go returns one map keyed by consumer UUID.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.

### `python:placement/v1/create_trait`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/traits.API.Create`

- At placement 1.6 Create sends PUT traits/{name} and accepts 201.
  Tests: `placement/v1/traits/python_parity_test.go:TestPythonTraitProxyCalls`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.6.
- Python sends an empty JSON object body; Go sends no body, which placement ignores.
- Python returns a Trait with the name; Go returns only an error.

### `python:placement/v1/delete_trait`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/traits.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- At placement 1.6 Delete sends DELETE traits/{name} and succeeds on 204.
  Tests: `placement/v1/traits/python_parity_test.go:TestPythonTraitProxyCalls`.
- Delete reports HTTP 404 as a native gophercloud.ErrUnexpectedResponseCode.
  Tests: `placement/v1/traits/python_parity_test.go:TestPythonTraitMissing`.
- Wrapping the direct delete in resource.IgnoreMissing turns a plain HTTP 404 into success like Python's default ignore_missing=True, keeps other failures such as 409 as errors, and leaves the DELETE request unchanged.
  Tests: `resource/missing_test.go:TestIgnoreMissingDropsOnlyPlain404`, `placement/v1/traits/python_ignore_missing_test.go:TestPythonDeleteIgnoreMissingTrait`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.6.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.
- Python ignore_missing=True is the default; Go selects it by wrapping the call in resource.IgnoreMissing, and the bare call returns the 404 like ignore_missing=False.

### `python:placement/v1/get_trait`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/traits.API.Get`

- At placement 1.6 Get sends GET traits/{name} and treats the bodyless 204 as success.
  Tests: `placement/v1/traits/python_parity_test.go:TestPythonTraitProxyCalls`.
- Get reports HTTP 404 as a native error like Python's NotFoundException.
  Tests: `placement/v1/traits/python_parity_test.go:TestPythonTraitMissing`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.6.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.
- Python returns a Trait holding the requested name; Go returns only an error.

### `python:placement/v1/traits`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/traits.API.List`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/traits.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/traits.WithListQuery`

- At placement 1.6 List sends GET traits with name and associated and yields each trait name string.
  Tests: `placement/v1/traits/python_parity_test.go:TestPythonTraitProxyCalls`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.6.
- Python sends only name and associated and passes values unchanged, so a Python bool is sent as True; Go encodes Associated as true or false and forwards extra keys via WithListQuery.
- Python yields Trait Resources; Go yields *string names.

### `python:placement/v1/get_resource_provider_trait`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.GetTraits`

- At placement 1.6 GetTraits sends GET resource_providers/{id}/traits and returns the generation and trait list.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderAggregatesUsagesAllocationsTraits`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.6.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.

### `python:placement/v1/set_resource_provider_trait`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.UpdateTraits`

- At placement 1.6 UpdateTraits sends PUT resource_providers/{id}/traits with the generation and traits, an empty non-nil slice sends [], and the new generation is decoded.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderAggregatesUsagesAllocationsTraits`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.6.
- Python requires a ResourceProviderTrait Resource carrying resource_provider_id and the generation; Go takes the provider ID and UpdateTraitsOpts.
- A nil Go Traits slice is sent as null.

### `python:placement/v1/delete_resource_provider_trait`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.DeleteTraits`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- At placement 1.6 DeleteTraits sends DELETE resource_providers/{id}/traits and succeeds on 204.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderAggregatesUsagesAllocationsTraits`.
- DeleteTraits reports HTTP 404 as a native gophercloud.ErrUnexpectedResponseCode.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderAggregatesUsagesAllocationsTraits`.
- Wrapping the direct delete in resource.IgnoreMissing turns a plain HTTP 404 into success like Python's default ignore_missing=True, keeps other failures such as 409 as errors, and leaves the DELETE request unchanged.
  Tests: `resource/missing_test.go:TestIgnoreMissingDropsOnlyPlain404`, `placement/v1/resourceproviders/python_ignore_missing_test.go:TestPythonDeleteIgnoreMissingResourceProvider`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.6.
- Python accepts a Resource instance or an ID string; Go takes the ID or name string.
- Python ignore_missing=True is the default; Go selects it by wrapping the call in resource.IgnoreMissing, and the bare call returns the 404 like ignore_missing=False.

### `python:placement/v1/usages`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/usages.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/usages.WithGetOptions`

- At placement 1.38 Get sends GET usages with project_id and the optional user_id and consumer_type, omitting unset filters, and decodes usage per consumer type including consumer_count.
  Tests: `placement/v1/usages/python_parity_test.go:TestPythonUsagesProxyCall`.
- Get does not decode the flat pre-1.38 usages shape.
  Tests: `placement/v1/usages/python_parity_test.go:TestPythonUsagesPre138ShapeIsNotDecoded`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.38.
- Python yields one Usage per consumer type with consumer_count split out; Go returns one map whose ConsumerTypeUsage keeps consumer_count beside the resource classes.
- Python also decodes the pre-1.38 flat shape when the server maximum is lower; Go Get fails there and RawClient with native ExtractPre138 is needed.
- Go sends no project_id when ProjectID is empty; Python requires the argument.

### `python:placement/v1/wait_for_status`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Wait`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithStatusAttribute`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFailureStates`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPollInterval`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithTimeout`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithUnlimitedWait`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithProgressCallback`

- ResourceProviders.WaitFor with WithStatusAttribute(name) and WithFailureStates(ERROR) polls GET resource_providers/{id} case-insensitively until the target, returns resource.ErrFailedState on a failure value and context.DeadlineExceeded after WithTimeout.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderWaiters`.

Python/Go differences:

- Python accepts any Resource and attribute and returns immediately without HTTP when it already matches; Go waits by ID, always fetches first and returns the final model.
- Python defaults to failures=['ERROR'], interval 2s and wait=None (unbounded); Go defaults to no failure states for placement, 2s and 5 minutes, adjustable with options.

Unresolved boundaries:

- Go waiters exist only for the ResourceProviders and ResourceClasses collections; Python can wait on any fetchable placement Resource such as inventories, traits, provider traits and allocations, which have no Go waiter.

### `python:placement/v1/wait_for_delete`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceproviders.API.WaitForDeletion`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses.API.WaitForDeletion`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.WaitDeleted`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPollInterval`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithTimeout`

- ResourceProviders.WaitForDeletion polls GET resource_providers/{id} until 404 and returns context.DeadlineExceeded after WithTimeout.
  Tests: `placement/v1/resourceproviders/python_parity_test.go:TestPythonResourceProviderWaiters`.
- ResourceClasses.WaitForDeletion polls GET resource_classes/{name} until 404.
  Tests: `placement/v1/resourceclasses/python_parity_test.go:TestPythonResourceClassWaitForDelete`.

Python/Go differences:

- Python returns the original Resource; Go returns only an error.
- Python defaults to interval 2s and wait 120s; Go defaults to 2s and 5 minutes, so WithTimeout(120*time.Second) matches.

Unresolved boundaries:

- Go deletion waiters exist only for the ResourceProviders and ResourceClasses collections; Python can wait for inventories, traits and other fetchable placement Resources to return 404, which Go cannot.

## References

- [`placement/v1/_proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/_proxy.py)
- [native-calls.md](native-calls.md)
- [`ResourceProvider`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/resource_provider.py)
- [`ResourceClass`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/resource_class.py)
- [`Trait`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/trait.py)
- [`ResourceProviderTrait`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/resource_provider_trait.py)
- [`AllocationCandidate`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/allocation_candidate.py)
- [`Allocation`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/allocation.py)
- [`Usage`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/usage.py)
- [`ResourceProviderInventory`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/resource_provider_inventory.py)
- [`ResourceProviderAllocation`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/resource_provider_allocation.py)
