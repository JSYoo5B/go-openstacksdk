# Python Create Record

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/create_firewall_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithCreateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithCreateRecordField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithCreateRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/request.Present`
- `github.com/JSYoo5B/go-openstacksdk/request.Null`

- CreateRecord sends POST fwaas/firewall_rules with an empty firewall_rule object for no arguments and otherwise only the supplied attributes, including protocol null, enabled false, an empty description, extension keys and a declared key with another JSON type through WithCreateRecordField when the typed field is omitted, accepts 200, 201 and 202, and decodes enveloped, flat and empty responses.
  Tests: `network/v2/extensions/fwaas_v2/rules/python_create_record_test.go:TestPythonFirewallRuleCreateRecordSendsOnlyGivenAttributes`.
- A blank extension key, a nil option, an extension that repeats a supplied typed key and a nil API fail before HTTP with resource.ErrInvalidOption, and a 400 response is returned as the Gophercloud status error wrapped in a rules resource.OperationError.
  Tests: `network/v2/extensions/fwaas_v2/rules/python_create_record_test.go:TestPythonFirewallRuleCreateRecordRejectsInputsAndErrorStatuses`.

Python/Go differences:

- Python returns the FirewallRule Resource merged from request attributes and the response; Go returns RuleRecord with the typed Rule, the raw response object fields, headers and status.
- Go rejects the same key supplied both in CreateRecordOpts and WithCreateRecordField, which Python keyword arguments cannot express.
- A response field that does not fit the Gophercloud Rule model, such as a string firewall_policy_id, fails the Go typed decode while Python keeps it.

## References

- [`create_firewall_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [`create_firewall_rule(**kwargs)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py)
- [`FirewallRule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/firewall_rule.py)
