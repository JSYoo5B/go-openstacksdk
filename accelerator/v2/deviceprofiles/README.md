# Deviceprofiles

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:accelerator/v2/create_device_profile`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/deviceprofiles.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/deviceprofiles.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/deviceprofiles.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/accelerator/v2/deviceprofiles.WithCreateHeader`

- POST /device_profiles sends a one-element array with original typed groups, original extension values, explicit empty description and omitted nil UUID. Options deeply snapshot nested groups/pointers/fields and remain reusable; 201 response metadata and large group numbers survive.
  Tests: `api/accelerator_profiles_test.go:TestAcceleratorProfileCreateContractAndSnapshots`.
- Invalid profile name, empty/non-object groups, unsafe UUID, unserializable extension, nil option, protected body/header fields and canceled context fail before HTTP; GET decoding rejects malformed group object types.
  Tests: `api/accelerator_profiles_test.go:TestAcceleratorProfileCreateValidationBeforeHTTP`, `api/accelerator_profiles_test.go:TestAcceleratorProfileRejectsMalformedResponseGroups`.

Python/Go differences:

- Pinned DeviceProfile.create_opts disables the singular request envelope and its _prepare_request_body wraps the dirty body in a one-element array; Go explicitly builds that controller array with concrete typed options and extension fields.
- Go validates canonical UUID/name/group shape before HTTP, uses pointers for omitted versus explicit description/UUID and accepts bare/singular/plural-singleton responses; it has no mutable Resource dirty-state.
- Python passes **attrs/**query and mutable Resource objects with dirty-state and response mutation; Go uses concrete options, explicit resource.Ref identities and returns decoded values with raw JSON and HTTP metadata.

### `python:accelerator/v2/delete_device_profile`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`

- An explicit profile name is resolved to UUID before one DELETE. Unsafe UUID/comma/name IDs fail before HTTP; duplicate names or malformed returned UUIDs never trigger deletion.
  Tests: `api/accelerator_profiles_test.go:TestAcceleratorProfileExactNamesAndMetadata`, `api/accelerator_profiles_test.go:TestAcceleratorProfileAmbiguityAndUnsafeIDs`.
- DELETE 404 is ignored by default and becomes ErrNotFound with original HTTP 404 under WithMissingError; 403 keeps actual code/header/body.
  Tests: `api/accelerator_profiles_test.go:TestAcceleratorProfileDeleteWaitAndHTTPFailures`.
- Deletes succeed only on 204, keep 200/403/500 as native errors after one attempt, never replay, and cancellation or deadline sends no extra request.
  Tests: `api/py_ha_accel_cyborg_test.go:TestPyHAAccelDeletesRequire204AndKeepNativeErrors`.

Python/Go differences:

- The controller accepts either a UUID or comma-delimited profile names on DELETE; Go only sends a canonical UUID after optional exact-name resolution, preventing accidental multi-name deletion.
- Go returns an error directly and uses WithMissingError to change its default ignore-missing policy; Python mutates/returns a Resource after inherited delete.
- Python ignore_missing defaults to True; Go ignores a missing resource by default, resource.WithMissingError selects the strict policy, and only 204 is success.
- Python passes **attrs/**query and mutable Resource objects with dirty-state and response mutation; Go uses concrete options, explicit resource.Ref identities and returns decoded values with raw JSON and HTTP metadata.

### `python:accelerator/v2/device_profiles`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.ResolveID`

- Profile iteration preserves created_at, null description/updated_at, large raw unknown numbers and json.Number group values, and breaking stops pagination. Exact UUID-shaped names resolve across two pages without a name query; prefix names do not match and cross-page duplicates prevent deletion.
  Tests: `api/accelerator_profiles_test.go:TestAcceleratorProfileExactNamesAndMetadata`, `api/accelerator_profiles_test.go:TestAcceleratorProfileAmbiguityAndUnsafeIDs`.
- Every Cyborg list sends no query by default, forwards only explicit WithQuery values, returns a non-nil empty slice and rejects caller limit/marker before HTTP.
  Tests: `api/py_ha_accel_cyborg_test.go:TestPyHAAccelListsQueryEmptyAndCallerPaging`.
- Breaking out of the iterator after the first item never requests the next advertised page.
  Tests: `api/py_ha_accel_cyborg_test.go:TestPyHAAccelListIteratorBreakStopsPaging`.

Python/Go differences:

- Python models inherit QueryParameters(limit, marker); Resource.list filters declared Body attributes locally and omits other unknown parameters. Go WithQuery forwards explicit query values to Cyborg, with WithName/WithStatus providing explicit local filters where supported; it rejects caller-supplied limit/marker and follows only server continuation links.
- Go separates canonical UUID IDs from explicit names, resolves names by complete list rather than the controller direct-name endpoint introduced at 2.2, and rejects ambiguity.
- Pinned Resource.list filters declared body attributes locally, drops unknown query keys and supports paginated, base_path, max_items and JMESPath arguments; Go forwards explicit WithQuery values unchanged, rejects caller limit/marker and follows only server continuation links without those arguments.
- Python passes **attrs/**query and mutable Resource objects with dirty-state and response mutation; Go uses concrete options, explicit resource.Ref identities and returns decoded values with raw JSON and HTTP metadata.

### `python:accelerator/v2/get_device_profile`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Get`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Find`

- Find(ID) requests /device_profiles/{requested UUID}, accepts a singular profile with a differing response UUID, preserves empty description/link/unknown false/request-id, and does not reinterpret a UUID-shaped explicit name as an ID.
  Tests: `api/accelerator_profiles_test.go:TestAcceleratorProfileExactNamesAndMetadata`.
- Invalid canonical UUID GET inputs fail before HTTP, and malformed group array/object/null/scalar types are rejected during profile GET decoding.
  Tests: `api/accelerator_profiles_test.go:TestAcceleratorProfileAmbiguityAndUnsafeIDs`, `api/accelerator_profiles_test.go:TestAcceleratorProfileRejectsMalformedResponseGroups`.
- Getters send one GET without body or query, keep native 403/404 errors and send nothing for a canceled context.
  Tests: `api/py_ha_accel_cyborg_test.go:TestPyHAAccelGettersHTTPErrorsAndCancellation`.

Python/Go differences:

- Python deprecated fields only emits a warning and never changes the GET; Go omits that argument and returns its own typed value with raw object JSON and HTTP metadata, without mutable Resource/cache semantics.
- The controller direct UUID-or-name GET uses name fetch starting at 2.2; Go Get is UUID-only and Find(Name) resolves exactly by List. Groups are concrete objects retaining json.Number.
- The Python fields argument is a warned no-op; Go has no such argument. Python accepts a Resource and mutates/caches it; Go takes an explicit identity string.
- Python passes **attrs/**query and mutable Resource objects with dirty-state and response mutation; Go uses concrete options, explicit resource.Ref identities and returns decoded values with raw JSON and HTTP metadata.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
profile = conn.accelerator.create_device_profile(
    name="fpga-profile",
    groups=[{"resources:CUSTOM_FPGA": "1", "trait:CUSTOM_ACCEL": "required"}],
)
conn.accelerator.delete_device_profile(profile.uuid, ignore_missing=False)
```

```go
func createAndDelete(ctx context.Context, client *gophercloud.ServiceClient) error {
    profiles := deviceprofiles.New(client)
    profile, err := profiles.Create(ctx, deviceprofiles.CreateOpts{
        Name: "fpga-profile",
        Groups: []deviceprofiles.Group{{
            "resources:CUSTOM_FPGA": "1",
            "trait:CUSTOM_ACCEL": "required",
        }},
    })
    if err != nil { return err }
    if err := profiles.Delete(ctx, resource.ID(profile.UUID), resource.WithMissingError()); err != nil {
        return err
    }
    return profiles.WaitDeleted(ctx, resource.ID(profile.UUID),
        resource.WithTimeout(2*time.Minute), resource.WithPollInterval(time.Second))
}
```

## References

- [openstacksdk pinned proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/_proxy.py)
- [pinned DeviceProfile resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/device_profile.py)
- [pinned Cyborg controller](https://github.com/openstack/cyborg/blob/dfa0b80ca7060a799db191f73c32b5d9bd577103/cyborg/api/controllers/v2/device_profiles.py)
- [Accelerator](https://docs.openstack.org/api-ref/accelerator/)
