# Attributes

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:accelerator/v2/attributes`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.All`

- All forwards deployable_id=17 and key=same, follows an explicit second-page query and keeps both distinct attribute UUIDs sharing the same key. Large database id/raw unknown values, timestamp/nulls, header and status are preserved; key/name/status capabilities fail without HTTP.
  Tests: `api/accelerator_attributes_test.go:TestAcceleratorAttributeReadAndKeyIsNotName`.

Python/Go differences:

- Python models inherit QueryParameters(limit, marker); Resource.list filters declared Body attributes locally and omits other unknown parameters. Go WithQuery forwards explicit query values to Cyborg, with WithName/WithStatus providing explicit local filters where supported; it rejects caller-supplied limit/marker and follows only server continuation links.
- Go uses canonical UUID identity, retains numeric database id independently and never treats a potentially repeated key as a name; controller deployable_id is a numeric database ID, not deployable UUID.

Unresolved boundaries:

- Audit and document the whole inherited Python Proxy._list/Resource.list contract: body-attribute local equality filtering, unknown query omission, paginated/base_path/max_items/jmespath arguments, limit/marker fallback, Link-header continuation, URI rewriting, lazy iteration and Adapter error/microversion behavior. Current fixtures cover the stated Go subset and do not establish those inherited contracts.
- Add attribute list iterator-break, empty-page/cycle/foreign continuation and explicit remaining query/error observations; the successful two-page All fixture covers only the stated filtering and metadata subset.

### `python:accelerator/v2/create_attribute`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/attributes.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/attributes.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/attributes.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/attributes.WithCreateHeader`

- POST /attributes sends a flat object with explicit numeric deployable_id=0 and empty key/value, not an attribute envelope. Typed/extension options snapshot caller values and remain reusable; 201 response metadata and unknown null survive.
  Tests: `api/accelerator_attributes_test.go:TestAcceleratorAttributeCreateFlatBodyAndSnapshots`.
- Bare/singular/plural-singleton accepted responses decode one attribute; null/scalar/array root, missing UUID, malformed/ambiguous/empty/multiple envelopes and wrong deployable numeric types return nil plus an error rather than choosing a batch member.
  Tests: `api/accelerator_attributes_test.go:TestAcceleratorAttributeSingletonResponseShapes`.
- Protected body/header fields, unserializable extension, invalid UUID/comma IDs and canceled context fail before HTTP; actual POST 403 retains status/header/body.
  Tests: `api/accelerator_attributes_test.go:TestAcceleratorAttributeValidationAndHTTPFailures`.

Python/Go differences:

- Pinned Attribute inherits Resource.create_opts and singular resource_key=attribute, so Resource._prepare_request_body envelopes the POST as {attribute: ...}; the pinned controller consumes a flat JSON object. Go sends the controller flat body and preserves explicit zero/empty fields.
- The controller create returns a flat Attribute but the official API example shows an attributes array. Go accepts bare/singular/exactly-one plural representations, never silently selects a member, and rejects malformed accepted responses with nil plus an error.

Unresolved boundaries:

- Audit Python Proxy._create, Resource initialization/dirty and unknown attributes, create_opts, _transform_create_request, _translate_response and Adapter error/retry/microversion behavior before treating the entire Python create operation as mapped. Concrete body/response tests do not validate that inherited pipeline.
- Document/verify the Python default-envelope correction explicitly, and audit Resource body filtering/dirty-state/response conversion versus the typed flat-body policy. The API documentation/controller envelope disagreement is observed compatibility handling, not proof of a single canonical server response.
- Add accepted in-flight attribute response cancellation/malformed-body retry-boundary tests if claiming that mutation boundary; current tests assert nil plus decoder error for malformed accepted singleton data rather than ARQ-style retained created batches.

### `python:accelerator/v2/delete_attribute`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`

- Single attribute DELETE uses its stable UUID member URL. 404 is ignored by default and WithMissingError exposes ErrNotFound plus original HTTP404; key/name inputs are unsupported and comma IDs fail before HTTP.
  Tests: `api/accelerator_attributes_test.go:TestAcceleratorAttributeWaitDeletedStableIDAndContext`, `api/accelerator_attributes_test.go:TestAcceleratorAttributeReadAndKeyIsNotName`, `api/accelerator_attributes_test.go:TestAcceleratorAttributeValidationAndHTTPFailures`.

Python/Go differences:

- Go returns error directly and uses explicit UUID Ref/WithMissingError; attribute key is not a name. Python inherited delete returns/mutates its Resource and catches NotFound only when ignore_missing=True.

Unresolved boundaries:

- Audit Python Proxy._delete/_get_resource and Resource._raw_delete/_translate_response/Adapter behavior for Resource/dict inputs, alternate identity, response-state mutation, cache, microversion and exception classification. UUID DELETE and ignore-missing fixtures cover only the stated Go policy.
- Add successful strict 204, forbidden/other-code and canceled attribute DELETE assertions; current delete fixture exercises only 404 while POST/GET/waiter errors do not prove DELETE-specific observations.

### `python:accelerator/v2/get_attribute`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Get`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.ResolveID`

- Find(ID) GETs the requested canonical UUID, preserves a differing returned UUID, request-id and unknown false, and ResolveID(ID) keeps the requested ID. Key/name/status references are unsupported before HTTP.
  Tests: `api/accelerator_attributes_test.go:TestAcceleratorAttributeReadAndKeyIsNotName`.
- Invalid non-UUID getter input fails before HTTP; an in-flight attribute GET whose handler waits on request cancellation exposes context.DeadlineExceeded and retains that cause.
  Tests: `api/accelerator_attributes_test.go:TestAcceleratorAttributeValidationAndHTTPFailures`, `api/accelerator_attributes_test.go:TestAcceleratorAttributeWaitDeletedStableIDAndContext`.

Python/Go differences:

- Python deprecated fields only emits a warning and never changes the GET; Go omits that argument and returns its own typed value with raw object JSON and HTTP metadata, without mutable Resource/cache semantics.
- Go Get is canonical-UUID-only and keeps numeric database id independently; key is not a lookup name and singleton cardinality is explicit.

Unresolved boundaries:

- Audit Python Proxy._get/_get_resource, Resource.fetch/_translate_response and Adapter behavior for string, existing Resource and dict inputs, response conversion/cache, alternate identity, default microversion and exception classification. The HTTP fixture establishes the Go request/result policy, not complete inherited Python equivalence.
- Add getter-specific 404/403 and singular/plural metadata/cardinality observations; create singleton and POST-forbidden fixtures validate different operations even though helpers are shared.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
attribute = conn.accelerator.create_attribute(
    deployable_id=17, key="trait:CUSTOM_ACCEL", value="required",
)
for item in conn.accelerator.attributes(deployable_id=17, key="trait:CUSTOM_ACCEL"):
    print(item.uuid, item.value)
```

```go
func createAndList(ctx context.Context, client *gophercloud.ServiceClient) error {
    api := attributes.New(client)
    _, err := api.Create(ctx, attributes.CreateOpts{
        DeployableID: 17, Key: "trait:CUSTOM_ACCEL", Value: "required",
    })
    if err != nil { return err }
    for item, err := range api.List(ctx,
        resource.WithQuery("deployable_id", "17"),
        resource.WithQuery("key", "trait:CUSTOM_ACCEL")) {
        if err != nil { return err }
        fmt.Println(item.UUID, item.Value)
    }
    return nil
}
```

## References

- [pinned openstacksdk proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/_proxy.py)
- [pinned Attribute resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/attribute.py)
- [pinned Cyborg controller](https://github.com/openstack/cyborg/blob/dfa0b80ca7060a799db191f73c32b5d9bd577103/cyborg/api/controllers/v2/attributes.py)
- [Accelerator](https://docs.openstack.org/api-ref/accelerator/)
