# Attributes

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:accelerator/v2/attributes`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.All`

- All forwards deployable_id=17 and key=same, follows an explicit second-page query and keeps both distinct attribute UUIDs sharing the same key. Large database id/raw unknown values, timestamp/nulls, header and status are preserved; key/name/status capabilities fail without HTTP.
  Tests: `api/accelerator_attributes_test.go:TestAcceleratorAttributeReadAndKeyIsNotName`.
- Every Cyborg list sends no query by default, forwards only explicit WithQuery values, returns a non-nil empty slice and rejects caller limit/marker before HTTP.
  Tests: `api/py_ha_accel_cyborg_test.go:TestPyHAAccelListsQueryEmptyAndCallerPaging`.
- Breaking out of the iterator after the first item never requests the next advertised page.
  Tests: `api/py_ha_accel_cyborg_test.go:TestPyHAAccelListIteratorBreakStopsPaging`.

Python/Go differences:

- Python models inherit QueryParameters(limit, marker); Resource.list filters declared Body attributes locally and omits other unknown parameters. Go WithQuery forwards explicit query values to Cyborg, with WithName/WithStatus providing explicit local filters where supported; it rejects caller-supplied limit/marker and follows only server continuation links.
- Go uses canonical UUID identity, retains numeric database id independently and never treats a potentially repeated key as a name; controller deployable_id is a numeric database ID, not deployable UUID.
- Pinned Resource.list filters declared body attributes locally, drops unknown query keys and supports paginated, base_path, max_items and JMESPath arguments; Go forwards explicit WithQuery values unchanged, rejects caller limit/marker and follows only server continuation links without those arguments.
- Python passes **attrs/**query and mutable Resource objects with dirty-state and response mutation; Go uses concrete options, explicit resource.Ref identities and returns decoded values with raw JSON and HTTP metadata.

### `python:accelerator/v2/create_attribute`

Status: `go_mapping`.

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
- Python passes **attrs/**query and mutable Resource objects with dirty-state and response mutation; Go uses concrete options, explicit resource.Ref identities and returns decoded values with raw JSON and HTTP metadata.

### `python:accelerator/v2/delete_attribute`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`

- Single attribute DELETE uses its stable UUID member URL. 404 is ignored by default and WithMissingError exposes ErrNotFound plus original HTTP404; key/name inputs are unsupported and comma IDs fail before HTTP.
  Tests: `api/accelerator_attributes_test.go:TestAcceleratorAttributeWaitDeletedStableIDAndContext`, `api/accelerator_attributes_test.go:TestAcceleratorAttributeReadAndKeyIsNotName`, `api/accelerator_attributes_test.go:TestAcceleratorAttributeValidationAndHTTPFailures`.
- Deletes succeed only on 204, keep 200/403/500 as native errors after one attempt, never replay, and cancellation or deadline sends no extra request.
  Tests: `api/py_ha_accel_cyborg_test.go:TestPyHAAccelDeletesRequire204AndKeepNativeErrors`.

Python/Go differences:

- Go returns error directly and uses explicit UUID Ref/WithMissingError; attribute key is not a name. Python inherited delete returns/mutates its Resource and catches NotFound only when ignore_missing=True.
- Python ignore_missing defaults to True; Go ignores a missing resource by default, resource.WithMissingError selects the strict policy, and only 204 is success.
- Python passes **attrs/**query and mutable Resource objects with dirty-state and response mutation; Go uses concrete options, explicit resource.Ref identities and returns decoded values with raw JSON and HTTP metadata.

### `python:accelerator/v2/get_attribute`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Get`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.ResolveID`

- Find(ID) GETs the requested canonical UUID, preserves a differing returned UUID, request-id and unknown false, and ResolveID(ID) keeps the requested ID. Key/name/status references are unsupported before HTTP.
  Tests: `api/accelerator_attributes_test.go:TestAcceleratorAttributeReadAndKeyIsNotName`.
- Invalid non-UUID getter input fails before HTTP; an in-flight attribute GET whose handler waits on request cancellation exposes context.DeadlineExceeded and retains that cause.
  Tests: `api/accelerator_attributes_test.go:TestAcceleratorAttributeValidationAndHTTPFailures`, `api/accelerator_attributes_test.go:TestAcceleratorAttributeWaitDeletedStableIDAndContext`.
- Getters send one GET without body or query, keep native 403/404 errors and send nothing for a canceled context.
  Tests: `api/py_ha_accel_cyborg_test.go:TestPyHAAccelGettersHTTPErrorsAndCancellation`.

Python/Go differences:

- Python deprecated fields only emits a warning and never changes the GET; Go omits that argument and returns its own typed value with raw object JSON and HTTP metadata, without mutable Resource/cache semantics.
- Go Get is canonical-UUID-only and keeps numeric database id independently; key is not a lookup name and singleton cardinality is explicit.
- The Python fields argument is a warned no-op; Go has no such argument. Python accepts a Resource and mutates/caches it; Go takes an explicit identity string.
- Python passes **attrs/**query and mutable Resource objects with dirty-state and response mutation; Go uses concrete options, explicit resource.Ref identities and returns decoded values with raw JSON and HTTP metadata.

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
