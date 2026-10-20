# Defaultstatefulness

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_security_groups_default_statefulness`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithCreateMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithCreateIfMatch`

- POST /security-groups-default-statefulness sends provided attrs under security_group_default_statefulness without If-Match, projects stateful/revision_number casts and unknown tenant_id, and reports 409 with Create context.
  Tests: `network/v2/extensions/security/defaultstatefulness/python_records_test.go:TestPythonStatefulnessCreateAndGet`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- if_match attrs must be a string, string list or null; Python would str() any other value while Go rejects it before HTTP.

### `python:network/v2/delete_security_groups_default_statefulness`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithDeleteMicroversion`

- Bodyless DELETE ignores a clean 404 by default and WithDeleteIgnoreMissing(false) returns it with Delete context.
  Tests: `network/v2/extensions/security/defaultstatefulness/python_records_test.go:TestPythonStatefulnessDeleteListAndFind`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- Python returns None; Go returns a DeleteResult with the actual status, header and body, and IgnoredMissing=true when the default ignore_missing suppressed a clean 404.

### `python:network/v2/find_security_groups_default_statefulness`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithFindMicroversion`

- Find GETs the literal ID, falls back on a clean 404 to a complete list without a name hint and matches name, returning ErrNotFound with WithFindIgnoreMissing(false).
  Tests: `network/v2/extensions/security/defaultstatefulness/python_records_test.go:TestPythonStatefulnessDeleteListAndFind`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- find returns nil, nil for a clean miss by default (Python None) and resource.ErrNotFound/ErrAmbiguous instead of NotFoundException/DuplicateResource.
- When following next links Go keeps the original filters and fails on a link that changes them or repeats a page; Python follows the href as is. Local Body filters compare JSON numbers exactly rather than with Python float rounding.

### `python:network/v2/get_security_groups_default_statefulness`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithGetFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithGetMicroversion`

- GET /security-groups-default-statefulness/{id} sends repeated fields and reports 404 with Get context.
  Tests: `network/v2/extensions/security/defaultstatefulness/python_records_test.go:TestPythonStatefulnessCreateAndGet`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.

### `python:network/v2/security_groups_default_statefulness`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithListPaginated`

- List sends declared query parameters, filters revision_number locally, follows security_groups_default_statefulness_links, sends Accept: application/json and reports 500 with List context.
  Tests: `network/v2/extensions/security/defaultstatefulness/python_records_test.go:TestPythonStatefulnessDeleteListAndFind`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- When following next links Go keeps the original filters and fails on a link that changes them or repeats a page; Python follows the href as is. Local Body filters compare JSON numbers exactly rather than with Python float rounding.

### `python:network/v2/update_security_groups_default_statefulness`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithUpdateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithUpdateMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithUpdateIfMatch`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/defaultstatefulness.WithUpdateRevision`

- PUT sends changed attrs with If-Match: revision_number=N from if_revision overriding if_match, a dirty If-Match alone PUTs an empty envelope, null if_revision with no attrs sends nothing, id and non-integer if_revision fail before HTTP, and 412 keeps Update context.
  Tests: `network/v2/extensions/security/defaultstatefulness/python_records_test.go:TestPythonStatefulnessUpdateRevisionAndNoop`.

Python/Go differences:

- Go takes literal ID strings rather than ID-or-Resource values and escapes each ID as one path segment; blank, dot-segment and control-character IDs fail before HTTP.
- Neutron has no microversion, so Python sends no version header; Go sends OpenStack-API-Version only when the selected client has a Microversion, and the With*Microversion/With*Headers options are Go extensions.
- Go admits 200..399 as success; Python accepts any status below 400, so informational 100..199 responses differ. 400..599 return the native ErrUnexpectedResponseCode wrapped in resource.OperationError.
- The SDK-owned location view is always null (raw value kept in Wire); Python fills it from the Connection.
- if_match attrs must be a string, string list or null; Python would str() any other value while Go rejects it before HTTP.
- if_revision must be a JSON integer (or null to omit); Python formats any value into revision_number=<value>.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
setting = conn.network.create_security_groups_default_statefulness(
    project_id=project.id, stateful=False)
conn.network.update_security_groups_default_statefulness(
    setting, stateful=True, if_revision=setting.revision_number)
for item in conn.network.security_groups_default_statefulness(project_id=project.id):
    print(item.id, item.stateful)
conn.network.delete_security_groups_default_statefulness(setting)
```

```go
api := defaultstatefulness.New(networkClient)
setting, err := api.Create(ctx,
    defaultstatefulness.WithCreateField("project_id", projectID),
    defaultstatefulness.WithCreateField("stateful", false))
if err != nil {
    return err
}
id := setting.Text("id")
var revision int64
_ = json.Unmarshal(setting.Resource.Body["revision_number"], &revision)
if _, err := api.Update(ctx, id,
    defaultstatefulness.WithUpdateField("stateful", true),
    defaultstatefulness.WithUpdateRevision(revision)); err != nil {
    return err
}
for item, err := range api.List(ctx, defaultstatefulness.WithListQuery("project_id", projectID)) {
    if err != nil {
        return err
    }
    fmt.Println(item.Text("id"), string(item.Resource.Body["stateful"]))
}
_, err = api.Delete(ctx, id)
return err
```
