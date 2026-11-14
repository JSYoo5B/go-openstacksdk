# Python Record Reads

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
import json
import os
import openstack

conn = openstack.connect(cloud=os.environ["OS_CLOUD"])
domain = os.environ["OS_PROJECT_DOMAIN_ID"]

for project in conn.identity.projects(domain_id=domain, limit=2, max_items=2):
    print(json.dumps(project.to_dict(), indent=2))

project = conn.identity.find_project(
    "demo", domain_id=domain, skip_cache=True,
    error_message="Project lookup failed",
)
if project is not None:
    print(json.dumps(project.to_dict(), indent=2))

listed = conn.list_projects(domain_id=domain, filters={"is_enabled": True})
print(json.dumps([project.to_dict() for project in listed], indent=2))
searched = conn.search_projects("demo*", domain_id=domain,
                                filters={"is_enabled": True})
print(json.dumps([project.to_dict() for project in searched], indent=2))
```

See `Example_recordReads` in [example_record_reads_test.go](example_record_reads_test.go), compiled by `go test`.

## References

- [openstacksdk ef55d7d](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/project.py)
- [Python Record Find](../users/python-record-find.md)
- [Source list_projects](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L82)
- [Project](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/project.py#L157-L174)
- [Base](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/base.py)
