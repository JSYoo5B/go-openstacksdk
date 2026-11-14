# Defaultstatefulness

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

```python
setting = conn.network.create_security_groups_default_statefulness(
    project_id=project.id, stateful=False)
conn.network.update_security_groups_default_statefulness(
    setting, stateful=True, if_revision=setting.revision_number)
for item in conn.network.security_groups_default_statefulness(project_id=project.id):
    print(item.id, item.stateful)
conn.network.delete_security_groups_default_statefulness(setting)
```

See `Example` in [example_guide_test.go](example_guide_test.go), compiled by `go test`.
