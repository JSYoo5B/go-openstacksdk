# Defaultstatefulness

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
setting = conn.network.create_security_groups_default_statefulness(
    project_id=project.id, stateful=False)
conn.network.update_security_groups_default_statefulness(
    setting, stateful=True, if_revision=setting.revision_number)
for item in conn.network.security_groups_default_statefulness(project_id=project.id):
    print(item.id, item.stateful)
conn.network.delete_security_groups_default_statefulness(setting)
```

```go
api := defaultstatefulness.New(networkClient)
setting, err := api.Create(ctx,
    request.Field("project_id", projectID),
    request.Field("stateful", false))
if err != nil {
    return err
}
id := setting.Text("id")
var revision int64
_ = json.Unmarshal(setting.Resource.Body["revision_number"], &revision)
if _, err := api.Update(ctx, id,
    request.Field("stateful", true),
    defaultstatefulness.WithUpdateRevision(revision)); err != nil {
    return err
}
for item, err := range api.List(ctx, request.Query("project_id", projectID)) {
    if err != nil {
        return err
    }
    fmt.Println(item.Text("id"), string(item.Resource.Body["stateful"]))
}
_, err = api.Delete(ctx, id)
return err
```
