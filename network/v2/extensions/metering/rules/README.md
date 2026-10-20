# Rules

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_metering_label_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithCreateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithCreateMicroversion`

- Create sends POST /v2.0/metering/metering-label-rules with {"metering_label_rule": attrs}, mapping Python attribute names to wire Body keys and keeping unknown attributes, accepts 200..399, decodes the metering_label_rule envelope, rejects 400 and above with the status and operation context, and rejects Resource control names before HTTP.
  Tests: `network/v2/extensions/metering/rules/api_test.go:TestPythonMeteringLabelRuleCreatePostsAttrsWithWireNames`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python **attrs are passed as WithCreateField options or CreateOpts.Attrs raw JSON; Resource control names such as base_path or connection fail with resource.ErrInvalidOption before HTTP instead of a Python TypeError or constructor effect.

### `python:network/v2/delete_metering_label_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithDeleteMicroversion`

- Delete sends a bodyless DELETE /v2.0/metering/metering-label-rules/{id}, treats a clean 404 as success by default (IgnoredMissing) like ignore_missing=True, returns the 404 error with WithDeleteIgnoreMissing(false) and always reports other 4xx/5xx statuses.
  Tests: `network/v2/extensions/metering/rules/api_test.go:TestPythonMeteringLabelRuleDeleteIgnoresMissingByDefault`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python returns None; Go returns a DeleteResult receipt with the actual status, headers and an IgnoredMissing flag.
- ignore_missing=False is expressed as WithDeleteIgnoreMissing(false); omitting the option keeps the Python default True.

### `python:network/v2/find_metering_label_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithFindMicroversion`

- Find sends GET /v2.0/metering/metering-label-rules/{id} first and, only after a clean 400, 403 or 404, reads the complete list matched on exact ID or name; it lists without a name query because name is not a declared query parameter, returns nil for a missing resource by default and resource.ErrNotFound with WithFindIgnoreMissing(false), and does not fall back after a 500.
  Tests: `network/v2/extensions/metering/rules/api_test.go:TestPythonMeteringLabelRuleFindListsWithoutNameHint`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python **query is passed as WithFindQuery options; ignore_missing=False is WithFindIgnoreMissing(false).
- Python emits a RemovedInSDK60Warning when ignore_missing is True; Go emits no warning.

### `python:network/v2/get_metering_label_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithGetFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithGetMicroversion`

- Get sends GET /v2.0/metering/metering-label-rules/{id} with Python field names mapped to wire keys in repeated fields query values, decodes the metering_label_rule envelope and returns the 404 as an error with operation context.
  Tests: `network/v2/extensions/metering/rules/api_test.go:TestPythonMeteringLabelRuleGetMapsFieldsAndReportsMissing`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- A missing resource is a gophercloud ErrUnexpectedResponseCode (404) wrapped in resource.OperationError rather than openstack.exceptions.NotFoundException.

### `python:network/v2/metering_label_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithListMicroversion`

- List sends GET /v2.0/metering/metering-label-rules with Accept: application/json and the declared server query (metering_label_id, direction and remote_ip_prefix) in Requests encoding, filters id, name, is_excluded and tenant_id locally on the Source view, drops undeclared names, follows metering_label_rules_links next links merged over the original query and continues with the last ID as marker after a full page.
  Tests: `network/v2/extensions/metering/rules/api_test.go:TestPythonMeteringLabelRuleListMapsQueryAndFiltersBodyLocally`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python returns a generator of Resources; Go returns a lazy iter.Seq2 that re-sends requests on every iteration (All collects it and discards partial rows on failure).
- Local Body filters compare JSON values exactly after Source descriptor casts instead of Python object equality.

### `python:network/v2/update_metering_label_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithUpdateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/metering/rules.WithUpdateMicroversion`

- Update sends PUT /v2.0/metering/metering-label-rules/{id} with only the given attributes under metering_label_rule, sends nothing and returns the ID-only view when no attribute is given, rejects id and metering_label_rule attributes before HTTP and reports 400 and above with operation context.
  Tests: `network/v2/extensions/metering/rules/api_test.go:TestPythonMeteringLabelRuleUpdatePutsOnlyGivenAttrs`.

Python/Go differences:

- Python accepts a Resource instance or an ID; Go accepts only an ID string, escapes it as one URL segment and rejects blank, '.', '..' or control-character IDs before HTTP, while Python joins the raw ID into the path.
- Python negotiates the microversion; Go sends the configured client microversion (none by default for Neutron), and the Headers/Microversion options are Go extensions.
- Python returns a Resource object; Go returns a typed struct whose typed fields are set only when the JSON type matches, plus Resource (Source view by Python name), Wire, Envelope, Header and StatusCode.
- Python **attrs are passed as WithUpdateField options or UpdateOpts.Attrs; id and metering_label_rule attributes fail with resource.ErrInvalidOption before HTTP instead of a Python TypeError.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
rule = conn.network.create_metering_label_rule(
    metering_label_id=label.id, direction="egress",
    destination_ip_prefix="0.0.0.0/0", is_excluded=False)
for item in conn.network.metering_label_rules(metering_label_id=label.id):
    print(item.direction, item.destination_ip_prefix)
conn.network.delete_metering_label_rule(rule)
```

```go
api := rules.New(networkClient)
rule, err := api.Create(ctx,
    rules.WithCreateField("metering_label_id", labelID),
    rules.WithCreateField("direction", "egress"),
    rules.WithCreateField("destination_ip_prefix", "0.0.0.0/0"),
    rules.WithCreateField("is_excluded", false))
if err != nil { return err }
for item, err := range api.List(ctx, rules.WithListQuery("metering_label_id", labelID)) {
    if err != nil { return err }
    fmt.Println(*item.Direction, *item.DestinationIPPrefix)
}
_, err = api.Delete(ctx, rule.ID)
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3005-L3157)
- [`metering_label_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/metering_label_rule.py)
- [`labels`](../labels/README.md)
