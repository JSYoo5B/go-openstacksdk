# Flavors

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_flavor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithCreateMicroversion`

- Create sends POST /v2.0/flavors with {"flavor": attrs}, mapping Python attribute names to wire Body keys and keeping unknown attributes, accepts 200..399, decodes the flavor envelope, rejects 400 and above with the status and operation context, and rejects Resource control names before HTTP.
  Tests: `network/v2/extensions/flavors/api_test.go:TestPythonFlavorCreatePostsAttrsWithWireNames`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python **attrs are passed as WithCreateField options or CreateOpts.Attrs raw JSON; Resource control names such as base_path or connection fail with resource.ErrInvalidOption before HTTP instead of a Python TypeError or constructor effect.

### `python:network/v2/delete_flavor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithDeleteMicroversion`

- Delete sends a bodyless DELETE /v2.0/flavors/{id}, treats a clean 404 as success by default (IgnoredMissing) like ignore_missing=True, returns the 404 error with WithDeleteIgnoreMissing(false) and always reports other 4xx/5xx statuses.
  Tests: `network/v2/extensions/flavors/api_test.go:TestPythonFlavorDeleteIgnoresMissingByDefault`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python returns None; Go returns a DeleteResult receipt with the actual status, headers and an IgnoredMissing flag.
- ignore_missing=False is expressed as WithDeleteIgnoreMissing(false); omitting the option keeps the Python default True.

### `python:network/v2/find_flavor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithFindMicroversion`

- Find sends GET /v2.0/flavors/{id} first and, only after a clean 400, 403 or 404, reads the complete list matched on exact ID or name; it adds name=<name_or_id> to the fallback list because name is a declared query parameter, returns nil for a missing resource by default and resource.ErrNotFound with WithFindIgnoreMissing(false), and does not fall back after a 500.
  Tests: `network/v2/extensions/flavors/api_test.go:TestPythonFlavorFindFallsBackToNameList`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python **query is passed as WithFindQuery options; ignore_missing=False is WithFindIgnoreMissing(false).
- Python emits a RemovedInSDK60Warning when ignore_missing is True; Go emits no warning.

### `python:network/v2/get_flavor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithGetFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithGetMicroversion`

- Get sends GET /v2.0/flavors/{id} with Python field names mapped to wire keys in repeated fields query values, decodes the flavor envelope and returns the 404 as an error with operation context.
  Tests: `network/v2/extensions/flavors/api_test.go:TestPythonFlavorGetMapsFieldsAndReportsMissing`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- A missing resource is a gophercloud ErrUnexpectedResponseCode (404) wrapped in resource.OperationError rather than openstack.exceptions.NotFoundException.

### `python:network/v2/flavors`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithListMicroversion`

- List sends GET /v2.0/flavors with Accept: application/json and the declared server query (is_enabled (sent as enabled), service_type) in Requests encoding, filters id and service_profile_ids locally on the Source view, drops undeclared names, follows flavors_links next links merged over the original query and continues with the last ID as marker after a full page.
  Tests: `network/v2/extensions/flavors/api_test.go:TestPythonFlavorListMapsQueryFiltersLocallyAndFollowsLinks`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python returns a generator of Resources; Go returns a lazy iter.Seq2 that re-sends requests on every iteration (All collects it and discards partial rows on failure).
- Local Body filters compare JSON values exactly after Source descriptor casts instead of Python object equality.

### `python:network/v2/update_flavor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithUpdateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithUpdateMicroversion`

- Update sends PUT /v2.0/flavors/{id} with only the given attributes under flavor, sends nothing and returns the ID-only view when no attribute is given, rejects id and flavor attributes before HTTP and reports 400 and above with operation context.
  Tests: `network/v2/extensions/flavors/api_test.go:TestPythonFlavorUpdatePutsOnlyGivenAttrs`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python **attrs are passed as WithUpdateField options or UpdateOpts.Attrs; id and flavor attributes fail with resource.ErrInvalidOption before HTTP instead of a Python TypeError.

### `python:network/v2/associate_flavor_with_service_profile`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.API.AssociateServiceProfile`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithAssociationOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithAssociationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithAssociationMicroversion`

- AssociateServiceProfile sends POST /v2.0/flavors/{flavor_id}/service_profiles with {"service_profile": {"id": service_profile_id}}, returns the whole response JSON (and service_profile.id) for 200..399, returns 400 and above as an error carrying the status and body with operation context, fails on a non-JSON success body as resp.json() does, and rejects blank IDs before HTTP.
  Tests: `network/v2/extensions/flavors/api_test.go:TestPythonFlavorAssociateServiceProfileReturnsResponseJSON`.

Python/Go differences:

- Python sends the request with raise_exc=False and returns the error response JSON as a dict for 4xx/5xx; Go returns a gophercloud ErrUnexpectedResponseCode (status and body) wrapped in resource.OperationError for 400 and above.
- Python returns resp.json() as a dict; Go returns ServiceProfileAssociation with Body (the raw response JSON), ServiceProfileID, Header and StatusCode.
- Python accepts Resource instances or IDs and sends {"id": null} for a None service profile; Go accepts only nonblank ID strings and escapes the flavor ID as one URL segment.
- Python negotiates the microversion; Go sends the configured client microversion, and Headers/Microversion options are Go extensions.

### `python:network/v2/disassociate_flavor_from_service_profile`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.API.DisassociateServiceProfile`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithAssociationOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithAssociationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors.WithAssociationMicroversion`

- DisassociateServiceProfile sends a bodyless DELETE /v2.0/flavors/{flavor_id}/service_profiles/{service_profile_id} with both IDs escaped, returns nil for 200..399 and returns 404, 409 and 500 as errors with the status and operation context.
  Tests: `network/v2/extensions/flavors/api_test.go:TestPythonFlavorDisassociateServiceProfileDeletesNestedMember`.

Python/Go differences:

- Python sends the request with raise_exc=False, ignores the response and returns None for every status; Go returns nil only for 200..399 and a gophercloud ErrUnexpectedResponseCode wrapped in resource.OperationError for 400 and above, including 404.
- Python accepts Resource instances or IDs and builds a path with an empty last segment for a None service profile; Go accepts only nonblank ID strings and escapes each as one URL segment.
- Python negotiates the microversion; Go sends the configured client microversion, and Headers/Microversion options are Go extensions.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
flavor = conn.network.create_flavor(
    name="gold", service_type="L3_ROUTER_NAT", is_enabled=True)
conn.network.associate_flavor_with_service_profile(flavor, profile_id)
for f in conn.network.flavors(service_type="L3_ROUTER_NAT"):
    print(f.name, f.service_profile_ids)
conn.network.delete_flavor(flavor)
```

```go
api := flavors.New(networkClient)
flavor, err := api.Create(ctx,
    flavors.WithCreateField("name", "gold"),
    flavors.WithCreateField("service_type", "L3_ROUTER_NAT"),
    flavors.WithCreateField("is_enabled", true))
if err != nil { return err }
if _, err := api.AssociateServiceProfile(ctx, flavor.ID, profileID); err != nil { return err }
for f, err := range api.List(ctx, flavors.WithListQuery("service_type", "L3_ROUTER_NAT")) {
    if err != nil { return err }
    fmt.Println(*f.Name, f.ServiceProfileIDs)
}
_, err = api.Delete(ctx, flavor.ID)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L1630-L1794)
- [`flavor.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/flavor.py)
