# Python Parity L3 QoS Fw

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_firewall_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.WithCreateField`

- Create sends POST /fwaas/firewall_groups with the {"firewall_group": {...}} envelope holding the supplied attributes and extension keys, and decodes the created firewall_group.
  Tests: `network/v2/extensions/fwaas_v2/groups/python_parity_test.go:TestPythonFirewallGroupProxyParity`.

Python/Go differences:

- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.
- Python sends every supplied attribute, including empty strings and unknown keys; Go CreateOpts drops empty strings through omitempty and WithCreateField cannot rewrite a declared key, so an explicit empty or null value for a declared field cannot be sent on create.

### `python:network/v2/update_firewall_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.RemoveIngressPolicy`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.RemoveEgressPolicy`

- Update sends PUT /fwaas/firewall_groups/{id} with only the set pointer fields in the firewall_group envelope, including explicit empty strings, false and empty lists. RemoveIngressPolicy and RemoveEgressPolicy send the policy ID as JSON null, matching update_firewall_group(..., <policy>_id=None).
  Tests: `network/v2/extensions/fwaas_v2/groups/python_parity_test.go:TestPythonFirewallGroupProxyParity`.

Python/Go differences:

- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.
- Python skips the request when no attribute changed; Go Update always sends PUT, with an empty envelope when UpdateOpts is empty.
- Gophercloud accepts only 200 for the PUT; Python accepts any non-error status.
- A null ingress or egress policy ID is sent by the separate RemoveIngressPolicy or RemoveEgressPolicy call rather than inside one combined update.

### `python:network/v2/delete_firewall_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove(resource.ID(id)) sends DELETE /fwaas/firewall_groups/{id}, succeeds on 204, ignores 404 by default like ignore_missing=True, returns resource.ErrNotFound with WithMissingError and keeps 409 as an error.
  Tests: `network/v2/extensions/fwaas_v2/groups/python_parity_test.go:TestPythonFirewallGroupProxyParity`.

Python/Go differences:

- Python accepts an ID or a Resource; Go takes resource.ID(id).
- Go uses resource.WithMissingError() for ignore_missing=False and returns resource.ErrNotFound instead of NotFoundException.

### `python:network/v2/get_firewall_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.GetWithFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.Get`

- GetWithFields sends GET fwaas/firewall_groups/{id} with each Python FirewallGroup Body attribute name mapped to its wire name as repeated fields query keys, keeping caller order, repeats, empty and unknown names, and decodes the selected attributes into the typed model.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersSendAliasedRepeatedFields`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- GetWithFields with nil or empty fields sends the plain member GET without a query, which is Python's fields=None request, and a sparse response decodes with zero values for omitted attributes.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersWithoutFieldsSendPlainGet`.
- GetWithFields accepts HTTP 200 through 399 like the Python read, and returns 404 and other failures as the native gophercloud status error wrapped in resource.OperationError with the operation name.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersAcceptPythonStatusesAndWrapFailures`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.
- Get(id) sends GET /fwaas/firewall_groups/{id} with no query like the fields=None default, decodes the firewall_group, and returns a 404 error where Python raises NotFoundException.
  Tests: `network/v2/extensions/fwaas_v2/groups/python_parity_test.go:TestPythonFirewallGroupProxyParity`.

Python/Go differences:

- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.
- Go GetWithFields returns the Gophercloud v2.15.0 typed model, so attributes Neutron returns outside that model are dropped and attributes the server omits are zero values instead of None; Python returns a sparse Resource.
- The SDK field getter accepts HTTP 200..399 and keeps unknown field names in the query like Python; plain API.Get keeps the native status policy.

### `python:network/v2/find_firewall_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Find(resource.ID(x), WithIgnoreMissing) sends GET /fwaas/firewall_groups/{x}; after a 404, or a 400 or 403 the caller continues past, Find(resource.Name(x)) sends GET /fwaas/firewall_groups?name=x and returns the unique match, nil with WithIgnoreMissing, resource.ErrNotFound when strict and resource.ErrAmbiguous for duplicate names.
  Tests: `network/v2/extensions/fwaas_v2/groups/python_parity_test.go:TestPythonFirewallGroupFindIDThenName`.

Python/Go differences:

- Go needs two Find calls; Python Resource.find does the ID GET and the name list in one call.
- Python falls back to the list after GET 400, 403 or 404; Go Find with WithIgnoreMissing returns nil only for 404, so the caller continues to the name lookup after a 400 or 403 error.
- The Python list match compares both ID and name; the Go name lookup compares only the name. Duplicates raise DuplicateResource in Python and resource.ErrAmbiguous in Go.
- Python forwards **query to both the ID GET and the name list; Go Find takes no extra query.
- Python defaults ignore_missing=True (deprecated toward False); Go Find defaults to the missing error and needs WithIgnoreMissing.
- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.

### `python:network/v2/firewall_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.All`

- List with WithListOptions(ListOpts{Name, Status, ProjectID, Ports}) and WithListQuery for shared and fields sends the Python query mapping keys, repeats ports per value, follows firewall_groups_links, and All sends no query for firewall_groups().
  Tests: `network/v2/extensions/fwaas_v2/groups/python_parity_test.go:TestPythonFirewallGroupProxyParity`.

Python/Go differences:

- Python filters Body attributes that are not in its query mapping (such as admin_state_up) on the returned rows; Go ListOpts sends such fields to the server and leaves other local filtering to the caller.
- Python encodes booleans as True/False; typed Go ListOpts booleans send true/false, so WithListQuery(key, "True") reproduces the exact string.
- WithListQuery sends one value per key, so Python's repeated fields=[a, b] query cannot be sent; typed slice fields such as ListOpts.Ports do repeat.
- Go follows the server _links next href; Python can also continue by marker after a page that fills limit.
- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.

### `python:network/v2/create_firewall_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithCreateField`

- Create sends POST /fwaas/firewall_policies with the {"firewall_policy": {...}} envelope holding the supplied attributes and extension keys, and decodes the created firewall_policy.
  Tests: `network/v2/extensions/fwaas_v2/policies/python_parity_test.go:TestPythonFirewallPolicyProxyParity`.

Python/Go differences:

- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.
- Python sends every supplied attribute, including empty strings and unknown keys; Go CreateOpts drops empty strings through omitempty and WithCreateField cannot rewrite a declared key, so an explicit empty or null value for a declared field cannot be sent on create.

### `python:network/v2/update_firewall_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithUpdateField`

- Update sends PUT /fwaas/firewall_policies/{id} with only the set pointer fields in the firewall_policy envelope, including explicit empty strings, false and empty lists.
  Tests: `network/v2/extensions/fwaas_v2/policies/python_parity_test.go:TestPythonFirewallPolicyProxyParity`.

Python/Go differences:

- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.
- Python skips the request when no attribute changed; Go Update always sends PUT, with an empty envelope when UpdateOpts is empty.
- Gophercloud accepts only 200 for the PUT; Python accepts any non-error status.

### `python:network/v2/delete_firewall_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove(resource.ID(id)) sends DELETE /fwaas/firewall_policies/{id}, succeeds on 204, ignores 404 by default like ignore_missing=True, returns resource.ErrNotFound with WithMissingError and keeps 409 as an error.
  Tests: `network/v2/extensions/fwaas_v2/policies/python_parity_test.go:TestPythonFirewallPolicyProxyParity`.

Python/Go differences:

- Python accepts an ID or a Resource; Go takes resource.ID(id).
- Go uses resource.WithMissingError() for ignore_missing=False and returns resource.ErrNotFound instead of NotFoundException.

### `python:network/v2/get_firewall_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.GetWithFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.Get`

- GetWithFields sends GET fwaas/firewall_policies/{id} with each Python FirewallPolicy Body attribute name mapped to its wire name as repeated fields query keys, keeping caller order, repeats, empty and unknown names, and decodes the selected attributes into the typed model.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersSendAliasedRepeatedFields`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- GetWithFields with nil or empty fields sends the plain member GET without a query, which is Python's fields=None request, and a sparse response decodes with zero values for omitted attributes.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersWithoutFieldsSendPlainGet`.
- GetWithFields accepts HTTP 200 through 399 like the Python read, and returns 404 and other failures as the native gophercloud status error wrapped in resource.OperationError with the operation name.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersAcceptPythonStatusesAndWrapFailures`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.
- Get(id) sends GET /fwaas/firewall_policies/{id} with no query like the fields=None default, decodes the firewall_policy, and returns a 404 error where Python raises NotFoundException.
  Tests: `network/v2/extensions/fwaas_v2/policies/python_parity_test.go:TestPythonFirewallPolicyProxyParity`.

Python/Go differences:

- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.
- Go GetWithFields returns the Gophercloud v2.15.0 typed model, so attributes Neutron returns outside that model are dropped and attributes the server omits are zero values instead of None; Python returns a sparse Resource.
- The SDK field getter accepts HTTP 200..399 and keeps unknown field names in the query like Python; plain API.Get keeps the native status policy.

### `python:network/v2/find_firewall_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Find(resource.ID(x), WithIgnoreMissing) sends GET /fwaas/firewall_policies/{x}; after a 404, or a 400 or 403 the caller continues past, Find(resource.Name(x)) sends GET /fwaas/firewall_policies?name=x and returns the unique match, nil with WithIgnoreMissing, resource.ErrNotFound when strict and resource.ErrAmbiguous for duplicate names.
  Tests: `network/v2/extensions/fwaas_v2/policies/python_parity_test.go:TestPythonFirewallPolicyFindIDThenName`.

Python/Go differences:

- Go needs two Find calls; Python Resource.find does the ID GET and the name list in one call.
- Python falls back to the list after GET 400, 403 or 404; Go Find with WithIgnoreMissing returns nil only for 404, so the caller continues to the name lookup after a 400 or 403 error.
- The Python list match compares both ID and name; the Go name lookup compares only the name. Duplicates raise DuplicateResource in Python and resource.ErrAmbiguous in Go.
- Python forwards **query to both the ID GET and the name list; Go Find takes no extra query.
- Python defaults ignore_missing=True (deprecated toward False); Go Find defaults to the missing error and needs WithIgnoreMissing.
- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.

### `python:network/v2/firewall_policies`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.All`

- List with WithListOptions(ListOpts{Name, ProjectID}) and WithListQuery for shared, firewall_rules and fields sends the Python query mapping keys, and All sends no query for firewall_policies().
  Tests: `network/v2/extensions/fwaas_v2/policies/python_parity_test.go:TestPythonFirewallPolicyProxyParity`.

Python/Go differences:

- Python filters Body attributes that are not in its query mapping (such as admin_state_up) on the returned rows; Go ListOpts sends such fields to the server and leaves other local filtering to the caller.
- Python encodes booleans as True/False; typed Go ListOpts booleans send true/false, so WithListQuery(key, "True") reproduces the exact string.
- WithListQuery sends one value per key, so Python's repeated fields=[a, b] query cannot be sent; typed slice fields such as ListOpts.Ports do repeat.
- Go follows the server _links next href; Python can also continue by marker after a page that fills limit.
- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.

### `python:network/v2/create_firewall_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithCreateField`

- Create sends POST /fwaas/firewall_rules with the {"firewall_rule": {...}} envelope holding the supplied attributes and extension keys, and decodes the created firewall_rule. Omitting protocol or action fails before HTTP, while ProtocolAny with ActionDeny sends protocol null and action deny.
  Tests: `network/v2/extensions/fwaas_v2/rules/python_parity_test.go:TestPythonFirewallRuleProxyParity`.

Python/Go differences:

- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.
- Python sends every supplied attribute, including empty strings and unknown keys; Go CreateOpts drops empty strings through omitempty and WithCreateField cannot rewrite a declared key, so an explicit empty or null value for a declared field cannot be sent on create.
- Python may omit protocol and action and let Neutron default them to any (null) and deny; Go CreateOpts requires both before HTTP, so the caller passes ProtocolAny (sent as null) and ActionDeny to get the same rule.

### `python:network/v2/update_firewall_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithUpdateField`

- Update sends PUT /fwaas/firewall_rules/{id} with only the set pointer fields in the firewall_rule envelope, including explicit empty strings, false and empty lists.
  Tests: `network/v2/extensions/fwaas_v2/rules/python_parity_test.go:TestPythonFirewallRuleProxyParity`.

Python/Go differences:

- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.
- Python skips the request when no attribute changed; Go Update always sends PUT, with an empty envelope when UpdateOpts is empty.
- Gophercloud accepts only 200 for the PUT; Python accepts any non-error status.

### `python:network/v2/delete_firewall_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove(resource.ID(id)) sends DELETE /fwaas/firewall_rules/{id}, succeeds on 204, ignores 404 by default like ignore_missing=True, returns resource.ErrNotFound with WithMissingError and keeps 409 as an error.
  Tests: `network/v2/extensions/fwaas_v2/rules/python_parity_test.go:TestPythonFirewallRuleProxyParity`.

Python/Go differences:

- Python accepts an ID or a Resource; Go takes resource.ID(id).
- Go uses resource.WithMissingError() for ignore_missing=False and returns resource.ErrNotFound instead of NotFoundException.

### `python:network/v2/get_firewall_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.GetWithFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.Get`

- GetWithFields sends GET fwaas/firewall_rules/{id} with each Python FirewallRule Body attribute name mapped to its wire name as repeated fields query keys, keeping caller order, repeats, empty and unknown names, and decodes the selected attributes into the typed model.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersSendAliasedRepeatedFields`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- GetWithFields with nil or empty fields sends the plain member GET without a query, which is Python's fields=None request, and a sparse response decodes with zero values for omitted attributes.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersWithoutFieldsSendPlainGet`.
- GetWithFields accepts HTTP 200 through 399 like the Python read, and returns 404 and other failures as the native gophercloud status error wrapped in resource.OperationError with the operation name.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersAcceptPythonStatusesAndWrapFailures`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.
- Get(id) sends GET /fwaas/firewall_rules/{id} with no query like the fields=None default, decodes the firewall_rule, and returns a 404 error where Python raises NotFoundException.
  Tests: `network/v2/extensions/fwaas_v2/rules/python_parity_test.go:TestPythonFirewallRuleProxyParity`.

Python/Go differences:

- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.
- Go GetWithFields returns the Gophercloud v2.15.0 typed model, so attributes Neutron returns outside that model are dropped and attributes the server omits are zero values instead of None; Python returns a sparse Resource.
- The SDK field getter accepts HTTP 200..399 and keeps unknown field names in the query like Python; plain API.Get keeps the native status policy.

### `python:network/v2/find_firewall_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Find(resource.ID(x), WithIgnoreMissing) sends GET /fwaas/firewall_rules/{x}; after a 404, or a 400 or 403 the caller continues past, Find(resource.Name(x)) sends GET /fwaas/firewall_rules?name=x and returns the unique match, nil with WithIgnoreMissing, resource.ErrNotFound when strict and resource.ErrAmbiguous for duplicate names.
  Tests: `network/v2/extensions/fwaas_v2/rules/python_parity_test.go:TestPythonFirewallRuleFindIDThenName`.

Python/Go differences:

- Go needs two Find calls; Python Resource.find does the ID GET and the name list in one call.
- Python falls back to the list after GET 400, 403 or 404; Go Find with WithIgnoreMissing returns nil only for 404, so the caller continues to the name lookup after a 400 or 403 error.
- The Python list match compares both ID and name; the Go name lookup compares only the name. Duplicates raise DuplicateResource in Python and resource.ErrAmbiguous in Go.
- Python forwards **query to both the ID GET and the name list; Go Find takes no extra query.
- Python defaults ignore_missing=True (deprecated toward False); Go Find defaults to the missing error and needs WithIgnoreMissing.
- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.

### `python:network/v2/firewall_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.All`

- List with WithListOptions(ListOpts{Protocol, Action, IPVersion, FirewallPolicyID}) and WithListQuery for enabled sends the Python query mapping keys, and All sends no query for firewall_rules().
  Tests: `network/v2/extensions/fwaas_v2/rules/python_parity_test.go:TestPythonFirewallRuleProxyParity`.

Python/Go differences:

- Python filters Body attributes that are not in its query mapping (such as admin_state_up) on the returned rows; Go ListOpts sends such fields to the server and leaves other local filtering to the caller.
- Python encodes booleans as True/False; typed Go ListOpts booleans send true/false, so WithListQuery(key, "True") reproduces the exact string.
- WithListQuery sends one value per key, so Python's repeated fields=[a, b] query cannot be sent; typed slice fields such as ListOpts.Ports do repeat.
- Go follows the server _links next href; Python can also continue by marker after a page that fills limit.
- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.

### `python:network/v2/insert_rule_into_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.InsertRuleRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithInsertRuleRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithInsertRuleRecordAfter`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithInsertRuleRecordBefore`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithInsertRuleRecordHeader`

- InsertRuleRecord sends PUT fwaas/firewall_policies/{id}/insert_rule with an unwrapped body that always carries firewall_rule_id, insert_after and insert_before, sending null for an omitted position and an empty string when given, snapshots InsertRuleRecordOpts when the option is built, accepts 200 and 202, and decodes the unwrapped policy into the typed Policy and raw Fields.
  Tests: `network/v2/extensions/fwaas_v2/policies/python_insert_rule_test.go:TestPythonFirewallPolicyInsertRuleRecordSendsNullPositions`.
- A blank policy ID, a nil option or a nil API fail before HTTP with resource.ErrInvalidOption in an InsertRuleRecord resource.OperationError, a 404 is returned as the Gophercloud status error wrapped for policies, and a non-JSON 200 body fails with resource.ResponseError like Python resp.json().
  Tests: `network/v2/extensions/fwaas_v2/policies/python_insert_rule_test.go:TestPythonFirewallPolicyInsertRuleRecordRejectsInputsAndErrorStatuses`.

Python/Go differences:

- Python accepts the policy and the rule as an ID or a Resource; Go takes ID strings and rejects a blank policy ID before HTTP.
- Python updates and returns the FirewallPolicy Resource; Go returns PolicyRecord with the typed Policy, the raw response object fields, headers and status.
- Python raises HttpException with the NeutronError message for statuses of 400 and above; Go returns the Gophercloud status error wrapped in resource.OperationError.

### `python:network/v2/remove_rule_from_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.RemoveRule`

- RemoveRule(id, ruleID) sends PUT /fwaas/firewall_policies/{id}/remove_rule with {"firewall_rule_id": ruleID}, decodes the unwrapped policy response and returns HTTP errors such as 400.
  Tests: `network/v2/extensions/fwaas_v2/policies/python_parity_test.go:TestPythonFirewallPolicyRuleActions`.

Python/Go differences:

- Python FirewallPolicy._put_request treats any status below 400 as success; Gophercloud accepts only 200.
- Python takes firewall_rule as an ID or Resource and resolves its ID; Go takes the rule ID string.
- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.

### `python:network/v2/create_qos_bandwidth_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateBandwidthLimitRule`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithCreateBandwidthLimitRuleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithCreateBandwidthLimitRuleField`

- CreateBandwidthLimitRule(policyID, opts) sends POST /qos/policies/{policy_id}/bandwidth_limit_rules with the bandwidth_limit_rule envelope, keeps the policy ID out of the body, carries extension keys and decodes the created rule.
  Tests: `network/v2/extensions/qos/rules/python_parity_test.go:TestPythonQoSRuleParityCreateUpdate`.

Python/Go differences:

- Go always sends max_kbps, even as 0, where Python omits a missing attribute; Neutron requires max_kbps, so only failing Python requests differ.
- Declared Go fields with omitempty (such as direction) cannot be sent as empty or null, and WithCreate...Field cannot rewrite a declared key.
- Gophercloud accepts only 201 for the POST; Python accepts any non-error status.
- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.

### `python:network/v2/update_qos_bandwidth_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.UpdateBandwidthLimitRule`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithUpdateBandwidthLimitRuleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithUpdateBandwidthLimitRuleField`

- UpdateBandwidthLimitRule(policyID, ruleID, opts) sends PUT /qos/policies/{policy_id}/bandwidth_limit_rules/{rule_id} with only the set fields, including an explicit 0, and returns HTTP errors such as 409.
  Tests: `network/v2/extensions/qos/rules/python_parity_test.go:TestPythonQoSRuleParityCreateUpdate`.

Python/Go differences:

- Python skips the request when no attribute changed; Go always sends PUT, with an empty envelope when the options are empty.
- Go takes ID strings in (policyID, ruleID) order while Python takes (qos_rule, qos_policy) as IDs or Resources.
- Gophercloud accepts only 200 for the PUT.
- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.

### `python:network/v2/delete_qos_bandwidth_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteBandwidthLimitRule`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- resource.IgnoreMissing(DeleteBandwidthLimitRule(policyID, ruleID)) sends DELETE /qos/policies/{policy_id}/bandwidth_limit_rules/{rule_id}, succeeds on 204, turns only a plain 404 into nil like ignore_missing=True and keeps 409; the unwrapped call returns the 404 like ignore_missing=False.
  Tests: `network/v2/extensions/qos/rules/python_parity_test.go:TestPythonQoSRuleParityDeleteIgnoreMissing`.

Python/Go differences:

- Python defaults ignore_missing=True inside delete; Go wraps the direct delete with resource.IgnoreMissing for that default and leaves it unwrapped for ignore_missing=False.
- Python accepts IDs or Resources for rule and policy; Go takes ID strings in (policyID, ruleID) order.

### `python:network/v2/create_qos_dscp_marking_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateDSCPMarkingRule`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithCreateDSCPMarkingRuleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithCreateDSCPMarkingRuleField`

- CreateDSCPMarkingRule(policyID, opts) sends POST /qos/policies/{policy_id}/dscp_marking_rules with the dscp_marking_rule envelope, keeps the policy ID out of the body, carries extension keys and decodes the created rule.
  Tests: `network/v2/extensions/qos/rules/python_parity_test.go:TestPythonQoSRuleParityCreateUpdate`.

Python/Go differences:

- Go always sends dscp_mark, even as 0, where Python omits a missing attribute; Neutron requires dscp_mark, so only failing Python requests differ.
- Declared Go fields with omitempty (such as direction) cannot be sent as empty or null, and WithCreate...Field cannot rewrite a declared key.
- Gophercloud accepts only 201 for the POST; Python accepts any non-error status.
- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.

### `python:network/v2/update_qos_dscp_marking_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.UpdateDSCPMarkingRule`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithUpdateDSCPMarkingRuleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithUpdateDSCPMarkingRuleField`

- UpdateDSCPMarkingRule(policyID, ruleID, opts) sends PUT /qos/policies/{policy_id}/dscp_marking_rules/{rule_id} with only the set fields, including an explicit 0, and returns HTTP errors such as 409.
  Tests: `network/v2/extensions/qos/rules/python_parity_test.go:TestPythonQoSRuleParityCreateUpdate`.

Python/Go differences:

- Python skips the request when no attribute changed; Go always sends PUT, with an empty envelope when the options are empty.
- Go takes ID strings in (policyID, ruleID) order while Python takes (qos_rule, qos_policy) as IDs or Resources.
- Gophercloud accepts only 200 for the PUT.
- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.

### `python:network/v2/delete_qos_dscp_marking_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteDSCPMarkingRule`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- resource.IgnoreMissing(DeleteDSCPMarkingRule(policyID, ruleID)) sends DELETE /qos/policies/{policy_id}/dscp_marking_rules/{rule_id}, succeeds on 204, turns only a plain 404 into nil like ignore_missing=True and keeps 409; the unwrapped call returns the 404 like ignore_missing=False.
  Tests: `network/v2/extensions/qos/rules/python_parity_test.go:TestPythonQoSRuleParityDeleteIgnoreMissing`.

Python/Go differences:

- Python defaults ignore_missing=True inside delete; Go wraps the direct delete with resource.IgnoreMissing for that default and leaves it unwrapped for ignore_missing=False.
- Python accepts IDs or Resources for rule and policy; Go takes ID strings in (policyID, ruleID) order.

### `python:network/v2/create_qos_minimum_bandwidth_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateMinimumBandwidthRule`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithCreateMinimumBandwidthRuleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithCreateMinimumBandwidthRuleField`

- CreateMinimumBandwidthRule(policyID, opts) sends POST /qos/policies/{policy_id}/minimum_bandwidth_rules with the minimum_bandwidth_rule envelope, keeps the policy ID out of the body, carries extension keys and decodes the created rule.
  Tests: `network/v2/extensions/qos/rules/python_parity_test.go:TestPythonQoSRuleParityCreateUpdate`.

Python/Go differences:

- Go always sends min_kbps, even as 0, where Python omits a missing attribute; Neutron requires min_kbps, so only failing Python requests differ.
- Declared Go fields with omitempty (such as direction) cannot be sent as empty or null, and WithCreate...Field cannot rewrite a declared key.
- Gophercloud accepts only 201 for the POST; Python accepts any non-error status.
- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.

### `python:network/v2/update_qos_minimum_bandwidth_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.UpdateMinimumBandwidthRule`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithUpdateMinimumBandwidthRuleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithUpdateMinimumBandwidthRuleField`

- UpdateMinimumBandwidthRule(policyID, ruleID, opts) sends PUT /qos/policies/{policy_id}/minimum_bandwidth_rules/{rule_id} with only the set fields, including an explicit 0, and returns HTTP errors such as 409.
  Tests: `network/v2/extensions/qos/rules/python_parity_test.go:TestPythonQoSRuleParityCreateUpdate`.

Python/Go differences:

- Python skips the request when no attribute changed; Go always sends PUT, with an empty envelope when the options are empty.
- Go takes ID strings in (policyID, ruleID) order while Python takes (qos_rule, qos_policy) as IDs or Resources.
- Gophercloud accepts only 200 for the PUT.
- Python accepts an ID or a Resource and returns a mutable Resource; Go takes an ID string and returns the Gophercloud v2.15.0 typed model, so response fields outside that model are dropped.

### `python:network/v2/delete_qos_minimum_bandwidth_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteMinimumBandwidthRule`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- resource.IgnoreMissing(DeleteMinimumBandwidthRule(policyID, ruleID)) sends DELETE /qos/policies/{policy_id}/minimum_bandwidth_rules/{rule_id}, succeeds on 204, turns only a plain 404 into nil like ignore_missing=True and keeps 409; the unwrapped call returns the 404 like ignore_missing=False.
  Tests: `network/v2/extensions/qos/rules/python_parity_test.go:TestPythonQoSRuleParityDeleteIgnoreMissing`.

Python/Go differences:

- Python defaults ignore_missing=True inside delete; Go wraps the direct delete with resource.IgnoreMissing for that default and leaves it unwrapped for ignore_missing=False.
- Python accepts IDs or Resources for rule and policy; Go takes ID strings in (policyID, ruleID) order.

### `python:network/v2/add_router_to_agent`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.ScheduleL3Router`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithScheduleL3RouterOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithScheduleL3RouterField`

- ScheduleL3Router(agentID, {RouterID}) sends POST /agents/{agent_id}/l3-routers with {"router_id": id}, WithScheduleL3RouterField adds ha_chassis_priority only when given, and a 404 is returned as an error.
  Tests: `network/v2/extensions/agents/python_parity_test.go:TestPythonL3AgentRouterScheduling`.

Python/Go differences:

- Python returns the decoded response JSON as a dict and never raises because the proxy requests with raise_exc=False; Go accepts only 201, returns an error otherwise and discards the response body.
- Python accepts agent and router as IDs or Resources; Go takes ID strings.

### `python:network/v2/remove_router_from_agent`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.RemoveL3Router`

- RemoveL3Router(agentID, routerID) sends DELETE /agents/{agent_id}/l3-routers/{router_id}, succeeds on 204 and returns 404 or 409 as errors.
  Tests: `network/v2/extensions/agents/python_parity_test.go:TestPythonL3AgentRouterScheduling`.

Python/Go differences:

- Python also sends {"router_id": id} as a DELETE body; Go sends no body.
- Python never raises and returns None for any status; Go reports non-2xx statuses, so a caller that wants Python's behavior discards the error.
- Python accepts agent and router as IDs or Resources; Go takes ID strings.

### `python:network/v2/agent_hosted_routers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.ListL3Routers`

- ListL3Routers(agentID) sends GET /agents/{agent_id}/l3-routers with no query, decodes the routers array into routers.Router values and returns HTTP errors.
  Tests: `network/v2/extensions/agents/python_parity_test.go:TestPythonL3AgentRouterScheduling`.

Python/Go differences:

- Python forwards **query (the inherited Router query mapping) to the server; Go ListL3Routers takes no query, so the caller filters the returned slice.
- Go reads one response and does not follow routers_links; it returns a slice instead of a lazy generator.
- routers.Router lacks fields such as ha and flavor_id that Python L3AgentRouter exposes.
- Python accepts the agent as an ID or Resource; Go takes an ID string.

### `python:network/v2/routers_hosting_l3_agents`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.ListL3Agents`

- ListL3Agents(routerID) sends GET /routers/{router_id}/l3-agents with no query, yields each agent with host, ha_state, alive, admin_state_up, agent_type, binary, availability_zone and configurations, and yields HTTP errors.
  Tests: `network/v2/extensions/layer3/routers/python_parity_test.go:TestPythonRouterHostingL3Agents`.

Python/Go differences:

- Python forwards **query (the inherited Agent query mapping such as host or agent_type) to the server; Go ListL3Agents takes no query, so the caller filters the yielded agents.
- Go reads a single page and does not follow agents_links.
- routers.L3Agent lacks ha_chassis_priority, and its timestamps are parsed time values rather than strings.
- Python accepts the router as an ID or Resource; Go takes an ID string.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
net, err := conn.NetworkV2(ctx)
if err != nil {
	return err
}
err = resource.IgnoreMissing(net.QoSRules.DeleteBandwidthLimitRule(ctx, policyID, ruleID))
```

```go
found, err := net.FirewallGroups.Find(ctx, resource.ID(v), resource.WithIgnoreMissing())
if err == nil && found == nil {
	found, err = net.FirewallGroups.Find(ctx, resource.Name(v), resource.WithIgnoreMissing())
}
```

## References

- [`network/v2/_proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [`extensions/fwaas_v2/groups/python_parity_test.go`](extensions/fwaas_v2/groups/python_parity_test.go)
- [`extensions/fwaas_v2/policies/python_parity_test.go`](extensions/fwaas_v2/policies/python_parity_test.go)
- [`extensions/fwaas_v2/rules/python_parity_test.go`](extensions/fwaas_v2/rules/python_parity_test.go)
- [`extensions/qos/rules/python_parity_test.go`](extensions/qos/rules/python_parity_test.go)
- [`extensions/agents/python_parity_test.go`](extensions/agents/python_parity_test.go)
- [`extensions/layer3/routers/python_parity_test.go`](extensions/layer3/routers/python_parity_test.go)
- [Python Router Assignment Dhcp Hosts](extensions/agents/python-router-assignment-dhcp-hosts.md)
- [`agent.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/agent.py)
- [qos-bandwidth-limit-records-python.md](qos-bandwidth-limit-records-python.md)
- [conntrack helper](extensions/layer3/conntrackhelpers/README.md)
- [NDP proxy](extensions/layer3/ndpproxies/README.md)
- [Defaultrules](extensions/security/defaultrules/README.md)
- [Defaultstatefulness](extensions/security/defaultstatefulness/README.md)
- [Packet Rate Mutations Python](extensions/qos/rules/packet-rate-mutations-python.md)
- [`openstack/network/v2/_proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [`firewall_group.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/firewall_group.py)
- [`firewall_policy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/firewall_policy.py)
- [`firewall_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/firewall_rule.py)
- [`router.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/router.py)
- [`qos_bandwidth_limit_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_bandwidth_limit_rule.py)
- [`qos_dscp_marking_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_dscp_marking_rule.py)
- [`qos_minimum_bandwidth_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_minimum_bandwidth_rule.py)
- [`qos_minimum_packet_rate_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_minimum_packet_rate_rule.py)
- [`qos_packet_rate_limit_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_packet_rate_limit_rule.py)
- [`l3_conntrack_helper.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/l3_conntrack_helper.py)
- [`ndp_proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/ndp_proxy.py)
- [`default_security_group_rule.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/default_security_group_rule.py)
- [`security_groups_default_statefulness.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/security_groups_default_statefulness.py)
- [`proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py)
- [`resource.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
