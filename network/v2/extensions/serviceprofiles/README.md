# Serviceprofiles

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_service_profile`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithCreateMicroversion`

- Create sends POST /v2.0/service_profiles with {"service_profile": attrs}, mapping Python attribute names to wire Body keys and keeping unknown attributes, accepts 200..399, decodes the service_profile envelope, rejects 400 and above with the status and operation context, and rejects Resource control names before HTTP.
  Tests: `network/v2/extensions/serviceprofiles/api_test.go:TestPythonServiceProfileCreatePostsAttrsWithWireNames`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python **attrs are passed as WithCreateField options or CreateOpts.Attrs raw JSON; Resource control names such as base_path or connection fail with resource.ErrInvalidOption before HTTP instead of a Python TypeError or constructor effect.

### `python:network/v2/delete_service_profile`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithDeleteMicroversion`

- Delete sends a bodyless DELETE /v2.0/service_profiles/{id}, treats a clean 404 as success by default (IgnoredMissing) like ignore_missing=True, returns the 404 error with WithDeleteIgnoreMissing(false) and always reports other 4xx/5xx statuses.
  Tests: `network/v2/extensions/serviceprofiles/api_test.go:TestPythonServiceProfileDeleteIgnoresMissingByDefault`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python returns None; Go returns a DeleteResult receipt with the actual status, headers and an IgnoredMissing flag.
- ignore_missing=False is expressed as WithDeleteIgnoreMissing(false); omitting the option keeps the Python default True.

### `python:network/v2/find_service_profile`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithFindMicroversion`

- Find sends GET /v2.0/service_profiles/{id} first and, only after a clean 400, 403 or 404, reads the complete list matched on exact ID or name; it lists without a name query because name is not a declared query parameter, returns nil for a missing resource by default and resource.ErrNotFound with WithFindIgnoreMissing(false), and does not fall back after a 500.
  Tests: `network/v2/extensions/serviceprofiles/api_test.go:TestPythonServiceProfileFindListsWithoutNameHint`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python **query is passed as WithFindQuery options; ignore_missing=False is WithFindIgnoreMissing(false).
- Python emits a RemovedInSDK60Warning when ignore_missing is True; Go emits no warning.

### `python:network/v2/get_service_profile`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithGetFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithGetMicroversion`

- Get sends GET /v2.0/service_profiles/{id} with Python field names mapped to wire keys in repeated fields query values, decodes the service_profile envelope and returns the 404 as an error with operation context.
  Tests: `network/v2/extensions/serviceprofiles/api_test.go:TestPythonServiceProfileGetMapsFieldsAndReportsMissing`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- A missing resource is a gophercloud ErrUnexpectedResponseCode (404) wrapped in resource.OperationError rather than openstack.exceptions.NotFoundException.

### `python:network/v2/service_profiles`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithListMicroversion`

- List sends GET /v2.0/service_profiles with Accept: application/json and the declared server query (driver, project_id and is_enabled (sent as enabled)) in Requests encoding, filters id, name, meta_info and tenant_id locally on the Source view, drops undeclared names, follows service_profiles_links next links merged over the original query and continues with the last ID as marker after a full page.
  Tests: `network/v2/extensions/serviceprofiles/api_test.go:TestPythonServiceProfileListMapsQueryAndFiltersBodyLocally`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python returns a generator of Resources; Go returns a lazy iter.Seq2 that re-sends requests on every iteration (All collects it and discards partial rows on failure).
- Local Body filters compare JSON values exactly after Source descriptor casts instead of Python object equality.

### `python:network/v2/update_service_profile`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithUpdateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/serviceprofiles.WithUpdateMicroversion`

- Update sends PUT /v2.0/service_profiles/{id} with only the given attributes under service_profile, sends nothing and returns the ID-only view when no attribute is given, rejects id and service_profile attributes before HTTP and reports 400 and above with operation context.
  Tests: `network/v2/extensions/serviceprofiles/api_test.go:TestPythonServiceProfileUpdatePutsOnlyGivenAttrs`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python **attrs are passed as WithUpdateField options or UpdateOpts.Attrs; id and service_profile attributes fail with resource.ErrInvalidOption before HTTP instead of a Python TypeError.

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7290-L7431)
- [`service_profile.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/service_profile.py)
- [`flavors`](../flavors/README.md)
