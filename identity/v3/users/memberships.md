# Memberships

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import openstack

conn = openstack.connect()
user_id = "user-id"

for project in conn.identity.user_projects(user_id):
    print(project.id, project.name, project.domain_id)

for group in conn.identity.user_groups(user_id):
    print(group.id, group.name, group.domain_id)
```

See `Example_memberships1` in [example_memberships1_test.go](example_memberships1_test.go), compiled by `go test`.

```python
import openstack

conn = openstack.connect()
for project in conn.identity.user_projects(
    "user-id", is_enabled=True, limit=20, max_items=50
):
    print(project.id, project.name, project.domain_id)
```

See `Example_memberships2` in [example_memberships2_test.go](example_memberships2_test.go), compiled by `go test`.

## References

- [user_projects](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1112-L1131)
- [user_groups](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1414-L1429)
- [UserProject manifest](../../../api/openstacksdk/resources/identity/v3/user_project.json)
