# Secgroups

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:compute/v2/secgroups.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.WithCreateField`

- Create POSTs {"security_group":{...}} to os-security-groups, requires name and rejects a core-key collision before HTTP, decodes numeric or string group/rule IDs as strings and accepts only 200.
  Tests: `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupRoutesBodiesAndDecode`, `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/secgroups proxy call only; the API was removed at Compute microversion 2.36 and Neutron security groups are reviewed separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/secgroups.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.API.Get`

- Get GETs os-security-groups/{id}, decodes the group with rules, ip_range and peer group and accepts only 200.
  Tests: `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupRoutesBodiesAndDecode`, `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/secgroups proxy call only; the API was removed at Compute microversion 2.36 and Neutron security groups are reviewed separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/secgroups.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.WithUpdateField`

- Update PUTs {"security_group":{...}} to os-security-groups/{id} sending an explicit empty pointer description and accepts only 200.
  Tests: `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupRoutesBodiesAndDecode`, `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/secgroups proxy call only; the API was removed at Compute microversion 2.36 and Neutron security groups are reviewed separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/secgroups.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.API.Delete`

- Delete DELETEs os-security-groups/{id} and accepts only 202/204.
  Tests: `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupRoutesBodiesAndDecode`, `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/secgroups proxy call only; the API was removed at Compute microversion 2.36 and Neutron security groups are reviewed separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/secgroups.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.API.List`

- List GETs os-security-groups as a single page and yields groups with numeric or string IDs as strings, reporting native page status errors.
  Tests: `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupRoutesBodiesAndDecode`, `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/secgroups proxy call only; the API was removed at Compute microversion 2.36 and Neutron security groups are reviewed separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/secgroups.ListByServer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.API.ListByServer`

- ListByServer GETs servers/{server}/os-security-groups as a single page.
  Tests: `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupRoutesBodiesAndDecode`, `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/secgroups proxy call only; the API was removed at Compute microversion 2.36 and Neutron security groups are reviewed separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/secgroups.CreateRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.API.CreateRule`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.WithCreateRuleOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.WithCreateRuleField`

- CreateRule POSTs {"security_group_rule":{...}} to os-security-group-rules always sending from/to ports (zero included), requires parent group and protocol plus at least one of CIDR or group_id before HTTP, decodes numeric or string rule/parent IDs and accepts only 200.
  Tests: `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupRoutesBodiesAndDecode`, `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/secgroups proxy call only; the API was removed at Compute microversion 2.36 and Neutron security groups are reviewed separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/secgroups.DeleteRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.API.DeleteRule`

- DeleteRule DELETEs os-security-group-rules/{id} and accepts only 202/204.
  Tests: `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupRoutesBodiesAndDecode`, `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/secgroups proxy call only; the API was removed at Compute microversion 2.36 and Neutron security groups are reviewed separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/secgroups.AddServer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.API.AddServer`

- AddServer POSTs {"addSecurityGroup":{"name":...}} to servers/{server}/action and accepts only the native POST default 201/202.
  Tests: `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupRoutesBodiesAndDecode`, `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/secgroups proxy call only; the API was removed at Compute microversion 2.36 and Neutron security groups are reviewed separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:compute/v2/secgroups.RemoveServer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/secgroups.API.RemoveServer`

- RemoveServer POSTs {"removeSecurityGroup":{"name":...}} to servers/{server}/action and accepts only the native POST default 201/202.
  Tests: `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupRoutesBodiesAndDecode`, `compute/v2/secgroups/native_contracts_test.go:TestNativeSecurityGroupStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native compute/v2/secgroups proxy call only; the API was removed at Compute microversion 2.36 and Neutron security groups are reviewed separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

## References

- [Requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/secgroups/requests.go)
- [Native Crud](../../../network/v2/native-crud.md)
