# Stackresources

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:orchestration/v1/stackresources.Find`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.API.Find`

Unresolved boundaries:

- Native Find requests /stacks/{stackName}/resources and returns the stack-wide resource list; StackScope.Find performs a child exact-name/ID lookup under an already fixed name+ID and is not the same operation.
- Audit and test the generated stack-name Find route, redirect/errors, response decoding, headers and cancellation before assigning a mapping status.

### `gophercloud:orchestration/v1/stackresources.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.API.ForStack`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.API.InStack`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.API.ForResource`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.StackScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.ResourceScope.Get`

- Resource routing uses resource_name independently from logical/physical IDs, retains native fields/timestamps and fixes the complete stack identity.
  Tests: `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesKnownPairAndParentResolution`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesSinglePageModelsAndLocalFilters`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesExactFindRoutingAndMissing`.
- A direct GET checks the requested resource_name and known owner, rejects conflicting identity links and invalid models, and retains original HTTP/decode causes.
  Tests: `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesModelAndHTTPErrors`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesExactFindRoutingAndMissing`.
- Status waits poll the fixed owning stack/resource_name, detect *_FAILED states and preserve timeout causes.
  Tests: `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesWaitUsesFixedResourceNameAndHeatFailureStates`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesListAndWaitErrorCauses`.
- Raw resource Body retains unknown fields, explicit null and the exact integer 9007199254740993; response StatusCode and cloned headers are preserved without aliasing between list models or later GETs.
  Tests: `api/heat_stackresources_responses_test.go:TestHeatStackResourcesRetainRawFieldsAndIndependentHeaders`.

Python/Go differences:

- Go returns a ResourceView with owner/detail metadata instead of GetResult; scope identity checks and malformed response rejection are stricter than native extraction.
- The scope directly performs the same resource GET with the pinned native success policy of HTTP 200 so it can retain StatusCode, copied Header and raw Body; it does not change accepted success codes.

### `gophercloud:orchestration/v1/stackresources.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.API.ForStack`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.API.InStack`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.StackScope.List`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.StackScope.All`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.WithNestedDepth`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.WithDetails`

- Resource routing uses resource_name independently from logical/physical IDs, retains native fields/timestamps and fixes the complete stack identity.
  Tests: `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesKnownPairAndParentResolution`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesSinglePageModelsAndLocalFilters`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesExactFindRoutingAndMissing`.
- Resource lists follow Heat single-page behavior, preserve nested owners, apply exact local name/status filters, ignore top-level next links, stop on break, and reject unsupported pagination/invalid queries before HTTP.
  Tests: `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesSinglePageModelsAndLocalFilters`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesValidationBeforeHTTP`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesListAndWaitErrorCauses`.
- Raw resource Body retains unknown fields, explicit null and the exact integer 9007199254740993; response StatusCode and cloned headers are preserved without aliasing between list models or later GETs.
  Tests: `api/heat_stackresources_responses_test.go:TestHeatStackResourcesRetainRawFieldsAndIndependentHeaders`.
- Missing/null/non-array list envelopes and null entries fail rather than being misclassified as an empty resource list.
  Tests: `api/heat_stackresources_responses_test.go:TestHeatStackResourcesRejectMalformedListEnvelopes`.

Python/Go differences:

- Go lazily yields ResourceView values with actual owner identities and optional detailed results; it follows the server SinglePage contract and rejects unsupported marker/limit before HTTP.
- The private single-page resource parser requires a resources JSON array and rejects malformed/missing/null envelopes instead of treating them as empty; item response metadata and raw fields remain observable.

### `gophercloud:orchestration/v1/stackresources.ListTypes`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.API.ListTypes`

Unresolved boundaries:

- ListTypes is a global resource-type operation (GET /resource_types), not a child stack resource scope; a generated facade alone does not establish SDK support.
- Audit native request/result serialization and meaningful model/status/header/context/error contracts, add HTTP tests and direct usage documentation.

### `gophercloud:orchestration/v1/stackresources.MarkUnhealthy`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.StackScope.MarkUnhealthy`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.ResourceScope.MarkUnhealthy`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.WithHealthReason`

- A health mutation preserves false and its optional reason, preflights resource_name/owner, blocks unknown or different nested owners, and requires an explicit owning ResourceIdentity to target a child.
  Tests: `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesNestedOwnerCannotMutateRoot`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesUnknownOwnerAndPhysicalFallbackAreBlocked`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesValidationBeforeHTTP`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesHealthHTTPErrorsAndMetadataDecodeCause`.

Python/Go differences:

- The scope takes the required bool and optional SDK-owned reason rather than MarkUnhealthyOptsBuilder, and GET-preflights logical/resource_name targeting before PATCH.

Unresolved boundaries:

- The native/server health endpoint also permits physical-ID targeting; the high-level scope deliberately restricts routing to resource_name and the generated physical-ID API branch has not received a meaningful dedicated HTTP contract test.
- Audit generated WithMarkUnhealthyOptions/WithMarkUnhealthyField and native builder extension/error behavior separately before judging the entire direct native operation complete.

### `gophercloud:orchestration/v1/stackresources.Metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.StackScope.Metadata`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.ResourceScope.Metadata`

- GET metadata within the fixed owner returns arbitrary JSON object/array/number/bool/null values with inherited auth/MoreHeaders and preserves original HTTP/decode/context error causes.
  Tests: `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesMetadataPreservesJSONAndHTTPCauses`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesHealthHTTPErrorsAndMetadataDecodeCause`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesCancellationReachesHTTP`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesValidationBeforeHTTP`.

Python/Go differences:

- Go scope expands the native map[string]string result to map[string]any according to the Heat metadata object contract; numbers use the default float64 decoder.
- The parent scope and exact Name/ID policy own routing; callers provide no custom result extractors.

### `gophercloud:orchestration/v1/stackresources.Schema`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.API.Schema`

Unresolved boundaries:

- Schema is a global resource-type operation (GET /resource_types/{typeName}), not a child stack resource scope; a generated facade alone does not establish SDK support.
- Audit native request/result serialization and meaningful model/status/header/context/error contracts, add HTTP tests and direct usage documentation.

### `gophercloud:orchestration/v1/stackresources.Template`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.API.Template`

Unresolved boundaries:

- Template is a global resource-type operation (GET /resource_types/{typeName}/template), not a child stack resource scope; a generated facade alone does not establish SDK support.
- Audit native request/result serialization and meaningful model/status/header/context/error contracts, add HTTP tests and direct usage documentation.

### `python:orchestration/v1/resources`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.API.ForStack`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.API.InStack`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.StackScope.List`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.StackScope.All`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.StackScope.Find`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.ResourceView.Identity`
- `github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackresources.API.ForResource`

- Resource routing uses resource_name independently from logical/physical IDs, retains native fields/timestamps and fixes the complete stack identity.
  Tests: `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesKnownPairAndParentResolution`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesSinglePageModelsAndLocalFilters`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesExactFindRoutingAndMissing`.
- Resource lists follow Heat single-page behavior, preserve nested owners, apply exact local name/status filters, ignore top-level next links, stop on break, and reject unsupported pagination/invalid queries before HTTP.
  Tests: `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesSinglePageModelsAndLocalFilters`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesValidationBeforeHTTP`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesListAndWaitErrorCauses`.
- A health mutation preserves false and its optional reason, preflights resource_name/owner, blocks unknown or different nested owners, and requires an explicit owning ResourceIdentity to target a child.
  Tests: `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesNestedOwnerCannotMutateRoot`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesUnknownOwnerAndPhysicalFallbackAreBlocked`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesValidationBeforeHTTP`, `api/heat_stackresources_contracts_test.go:TestHeatStackResourcesHealthHTTPErrorsAndMetadataDecodeCause`.
- Raw resource Body retains unknown fields, explicit null and the exact integer 9007199254740993; response StatusCode and cloned headers are preserved without aliasing between list models or later GETs.
  Tests: `api/heat_stackresources_responses_test.go:TestHeatStackResourcesRetainRawFieldsAndIndependentHeaders`.
- Malformed resources envelopes are rejected explicitly.
  Tests: `api/heat_stackresources_responses_test.go:TestHeatStackResourcesRejectMalformedListEnvelopes`.

Python/Go differences:

- Go explicitly fixes stack name+canonical ID, uses resource_name as routing identity, and preserves nested actual owners rather than treating every result as root-owned.
- Go rejects marker/limit and unknown server query keys before HTTP and maps results to native fields plus owner/detail/raw-response metadata rather than Python Resource objects.

Unresolved boundaries:

- Audit inherited Python Stack.find/_find name-or-ID and object inputs, deleted/adopted-stack state policy, and full Resource.list/MRO/descriptor/cache/adapter behavior before whole-operation parity.
- Audit Python client-side query/filter behavior and inherited pagination against the Heat server single-page restrictions, including unsupported marker/limit and all accepted filter forms.
- Audit Python Resource unknown Body/URI state, alternate logical-resource ID, descriptor/date coercion and response/error policies separately; tested Go raw fields/header retention does not account for the complete inherited Resource surface.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
scope, err := service.StackResources.InStack(ctx, resource.Name("app"))
if err != nil { return err }
item, err := scope.Find(ctx, resource.Name("web"))
if err != nil { return err }
fmt.Println(item.Name, item.LogicalID, item.PhysicalID, item.Owner)

id, err := scope.ResolveID(ctx, resource.Name("web"))
if err != nil { return err }
detail, err := scope.Get(ctx, id)
if err != nil { return err }
fmt.Println(detail.Attributes, detail.Detailed)
```

```go
scope, err := service.StackResources.ForStack(stacks.StackIdentity{
    Name: "app", ID: "stack-uuid",
})
if err != nil { return err }
items, err := scope.All(ctx, stackresources.WithNestedDepth(1),
    stackresources.WithDetails(true))
if err != nil { return err }
for _, item := range items {
    identity, err := item.Identity()
    if err != nil { return err }
    child, err := service.StackResources.ForResource(identity)
    if err != nil { return err }
    metadata, err := child.Metadata(ctx)
    if err != nil { return err }
    fmt.Println(identity.Stack.Name, identity.Stack.ID, identity.Name, metadata)
}
```

```go
node, err := service.StackResources.ForResource(stackresources.ResourceIdentity{
    Stack: stacks.StackIdentity{Name: "app", ID: "stack-uuid"},
    Name: "web",
})
if err != nil { return err }
if err := node.MarkUnhealthy(ctx, false,
    stackresources.WithHealthReason("recovered")); err != nil { return err }
if _, err := node.Wait(ctx, "CHECK_COMPLETE"); err != nil { return err }
```

## References

- [stack resolver](../stacks/README.md)
- [..](../README.md)
- [Heat Stackresources Contracts Test](../../../api/heat_stackresources_contracts_test.go)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)
- [Heat 2026.1 ResourceController](https://docs.openstack.org/heat/2026.1/_modules/heat/api/openstack/v1/resources.html)
- [Heat resource formatter](https://docs.openstack.org/heat/2026.1/_modules/heat/engine/api.html#format_stack_resource)
- [Heat resource API reference](https://docs.openstack.org/api-ref/orchestration/v1/#stack-resources)
- [Python Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/orchestration/v1/resource.py)
- [Proxy.resources](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/orchestration/v1/_proxy.py#L370-L403)

## Native API calls

The generated `API` methods address Heat paths directly and are covered by `api/heat_native_stackresources_test.go`.

```go
api := stackresources.New(client)

// GET /stacks/{name}/resources; the redirect to the canonical stack is followed.
resources, err := api.Find(ctx, "app")

// Global resource-type operations (not tied to a stack).
for name, err := range api.ListTypes(ctx) { if err != nil { panic(err) }; fmt.Println(*name) }
schema, err := api.Schema(ctx, "OS::Nova::Server")
template, err := api.Template(ctx, "OS::Nova::Server") // indented JSON bytes

// The path segment is sent unchanged, so a physical resource ID can be targeted directly.
err = api.MarkUnhealthy(ctx, "app", id, physicalID, stackresources.MarkUnhealthyOpts{MarkUnhealthy: true, ResourceStatusReason: "bad disk"})
```

Limitations: `ListTypes` is one non-paged request that preserves the server order. `MarkUnhealthy` accepts HTTP 200/202/204, sends `mark_unhealthy: false` when unset and omits an empty reason. Extensions cannot replace the typed `mark_unhealthy` field. Unlike the scoped `StackScope.MarkUnhealthy`, the native call performs no GET preflight or owner check.
