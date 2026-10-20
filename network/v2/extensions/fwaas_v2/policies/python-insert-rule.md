# Python Insert Rule

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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

## References

- [`insert_rule_into_policy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [`FirewallPolicy._put_request`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/firewall_policy.py)
