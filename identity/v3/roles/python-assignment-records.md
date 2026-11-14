# Python Assignment Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := roles.New(identityClient)
values, err := api.AllAssignmentRecords(ctx,
    request.Query("user_id", userID),
    request.Query("scope", map[string]any{"OS-INHERIT:inherited_to": "projects"}))
```

## References

- [`_proxy.py` role_assignments](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1983-L1997)
- [`role_assignment.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/role_assignment.py)
