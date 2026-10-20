# Python Assignment Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/role_assignments`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.ListAssignmentRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.API.AllAssignmentRecords`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithAssignmentRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithAssignmentRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithAssignmentRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithAssignmentRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithAssignmentRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/roles.WithAssignmentRecordListOpts`

- ListAssignmentRecords sends GET role_assignments with Accept application/json, maps role_id, user_id, group_id, scope_project_id, scope_domain_id, scope_system, inherited_to (or their dotted server names, client name winning), effective, include_names and include_subtree to Requests-encoded server query, ignores unknown names, and keeps raw role, scope (including OS-INHERIT:inherited_to), user, group and links per row.
  Tests: `identity/v3/roles/python_assignment_records_test.go:TestPythonRoleAssignmentRecordsKeepScopeLinksAndQuery`.
- role, scope, user, group, links, id and name filter rows locally with nested dictionary subset matching; with a limit the first page rows are yielded and the id-less marker continuation fails like Python's endless pagination guard; HTTP errors keep their status.
  Tests: `identity/v3/roles/python_assignment_records_test.go:TestPythonRoleAssignmentRecordsNestedLocalFiltersAndPaging`.

Python/Go differences:

- Python returns RoleAssignment Resources with dict-typed role, scope, user and group; Go yields owned records with those objects as raw JSON.
- With a limit, Python raises SDKException for the repeated None marker after the first page; Go stops at the same point with resource.ErrInvalidOption wrapped in resource.OperationError and sends no second request.
- Go follows a string Keystone links.next URL as a compatibility extension; the pinned Python paginator ignores that dictionary shape and relies on limit/marker continuation.
- Local Body filters compare JSON values, so Go distinguishes bool from number and does not apply Python type coercion; nested dictionary filters match subsets like Resource.list _dict_filter.
- Per-request header options are a Go extension; authentication and content headers stay SDK-owned, and the Resource location attribute and deprecation warnings are not reproduced.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := roles.New(identityClient)
values, err := api.AllAssignmentRecords(ctx,
    roles.WithAssignmentRecordListQuery("user_id", userID),
    roles.WithAssignmentRecordListQuery("scope", map[string]any{"OS-INHERIT:inherited_to": "projects"}))
```

## References

- [`_proxy.py` role_assignments](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1983-L1997)
- [`role_assignment.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/role_assignment.py)
