# Labels

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_metering_label`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithCreateMicroversion`

- Create sends POST /v2.0/metering/metering-labels with {"metering_label": attrs}, mapping Python attribute names to wire Body keys and keeping unknown attributes, accepts 200..399, decodes the metering_label envelope, rejects 400 and above with the status and operation context, and rejects Resource control names before HTTP.
  Tests: `network/v2/extensions/metering/labels/api_test.go:TestPythonMeteringLabelCreatePostsAttrsWithWireNames`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python **attrs are passed as WithCreateField options or CreateOpts.Attrs raw JSON; Resource control names such as base_path or connection fail with resource.ErrInvalidOption before HTTP instead of a Python TypeError or constructor effect.

### `python:network/v2/delete_metering_label`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithDeleteMicroversion`

- Delete sends a bodyless DELETE /v2.0/metering/metering-labels/{id}, treats a clean 404 as success by default (IgnoredMissing) like ignore_missing=True, returns the 404 error with WithDeleteIgnoreMissing(false) and always reports other 4xx/5xx statuses.
  Tests: `network/v2/extensions/metering/labels/api_test.go:TestPythonMeteringLabelDeleteIgnoresMissingByDefault`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python returns None; Go returns a DeleteResult receipt with the actual status, headers and an IgnoredMissing flag.
- ignore_missing=False is expressed as WithDeleteIgnoreMissing(false); omitting the option keeps the Python default True.

### `python:network/v2/find_metering_label`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithFindMicroversion`

- Find sends GET /v2.0/metering/metering-labels/{id} first and, only after a clean 400, 403 or 404, reads the complete list matched on exact ID or name; it adds name=<name_or_id> to the fallback list because name is a declared query parameter, returns nil for a missing resource by default and resource.ErrNotFound with WithFindIgnoreMissing(false), and does not fall back after a 500.
  Tests: `network/v2/extensions/metering/labels/api_test.go:TestPythonMeteringLabelFindFallsBackToNameList`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python **query is passed as WithFindQuery options; ignore_missing=False is WithFindIgnoreMissing(false).
- Python emits a RemovedInSDK60Warning when ignore_missing is True; Go emits no warning.

### `python:network/v2/get_metering_label`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithGetFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithGetMicroversion`

- Get sends GET /v2.0/metering/metering-labels/{id} with Python field names mapped to wire keys in repeated fields query values, decodes the metering_label envelope and returns the 404 as an error with operation context.
  Tests: `network/v2/extensions/metering/labels/api_test.go:TestPythonMeteringLabelGetMapsFieldsAndReportsMissing`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- A missing resource is a gophercloud ErrUnexpectedResponseCode (404) wrapped in resource.OperationError rather than openstack.exceptions.NotFoundException.

### `python:network/v2/metering_labels`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithListMicroversion`

- List sends GET /v2.0/metering/metering-labels with Accept: application/json and the declared server query (name, project_id, sort_key and is_shared (sent as shared)) in Requests encoding, filters id and tenant_id locally on the Source view, drops undeclared names, follows metering_labels_links next links merged over the original query and continues with the last ID as marker after a full page.
  Tests: `network/v2/extensions/metering/labels/api_test.go:TestPythonMeteringLabelListMapsQueryAndFiltersBodyLocally`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python returns a generator of Resources; Go returns a lazy iter.Seq2 that re-sends requests on every iteration (All collects it and discards partial rows on failure).
- Local Body filters compare JSON values exactly after Source descriptor casts instead of Python object equality.

### `python:network/v2/update_metering_label`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithUpdateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/labels.WithUpdateMicroversion`

- Update sends PUT /v2.0/metering/metering-labels/{id} with only the given attributes under metering_label, sends nothing and returns the ID-only view when no attribute is given, rejects id and metering_label attributes before HTTP and reports 400 and above with operation context.
  Tests: `network/v2/extensions/metering/labels/api_test.go:TestPythonMeteringLabelUpdatePutsOnlyGivenAttrs`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python **attrs are passed as WithUpdateField options or UpdateOpts.Attrs; id and metering_label attributes fail with resource.ErrInvalidOption before HTTP instead of a Python TypeError.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
label = conn.network.create_metering_label(name="web", is_shared=False)
for item in conn.network.metering_labels(project_id=project_id):
    print(item.name, item.is_shared)
conn.network.delete_metering_label(label)
```

```go
api := labels.New(networkClient)
label, err := api.Create(ctx,
    labels.WithCreateField("name", "web"),
    labels.WithCreateField("is_shared", false))
if err != nil { return err }
for item, err := range api.List(ctx, labels.WithListQuery("project_id", projectID)) {
    if err != nil { return err }
    fmt.Println(*item.Name, item.IsShared != nil && *item.IsShared)
}
_, err = api.Delete(ctx, label.ID)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L2861-L3002)
- [`metering_label.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/metering_label.py)
- [`rules`](../rules/README.md)
