# openstacksdk container_infrastructure_management comparison

This page records how the pinned openstacksdk `container_infrastructure_management/v1` proxy and the `CoeCloudMixin` cloud helpers map onto the Go API. Native operation contracts stay beside the packages ([clusters](clusters/README.md), [clustertemplates](clustertemplates/README.md), [certificates](certificates/README.md)); this page covers what is added or different.

## Summary

| openstacksdk | Go |
|---|---|
| `create_cluster`, `create_coe_cluster` | `clusters.API.Create` (returns the new UUID) |
| `get_cluster`, `find_cluster`, `delete_cluster`, `update_cluster` | `clusters.API.Get`, `Find`, `Remove`, `Update` |
| `create_cluster_template`, `get_cluster_template`, `find_cluster_template`, `delete_cluster_template`, `update_cluster_template` | `clustertemplates.API.Create`, `Get`, `Find`, `Remove`, `Update` |
| `create_cluster_certificate`, `sign_coe_cluster_certificate`, `get_cluster_certificate`, `get_coe_cluster_certificate` | `certificates.API.Create`, `Get` |
| `services`, `list_magnum_services` | `Service.ListMagnumServices`, `Service.AllMagnumServices` |
| `list_coe_clusters`, `list_cluster_templates` | `clusters.API.List`, `clustertemplates.API.List` (one unfiltered iteration each) |
| `delete_coe_cluster`, `update_coe_cluster` | `Service.DeleteCOECluster`, `Service.UpdateCOECluster` |
| `delete_cluster_template`, `update_cluster_template` (cloud) | `Service.DeleteClusterTemplate`, `Service.UpdateClusterTemplate` |
| `wait_for_status`, `wait_for_delete` | `clusters.API.WaitFor`, `WaitForDeletion` and the template equivalents |

## Behavior notes

- Python passes `**attrs` through a Resource; Go uses concrete `CreateOpts` plus `With...Field` extension fields. The native Magnum builder always serializes `create_timeout` (null when omitted) and `name` (empty when omitted) for clusters, while Python sends only the attributes it was given. Explicit zero and false values are preserved.
- Python accepts every status below 400; Go accepts the documented success codes only (cluster create 202, template create 201, update 200 or 202, delete 202 or 204) and reports others as the native unexpected-status error. Mutations are sent once and never replayed. A 2xx acknowledgement is not completion; use `WaitFor`.
- Python returns Resource objects; Go returns typed Gophercloud models (cluster create and update return only the UUID acknowledged by Magnum).
- Python `update_*` builds a JSON patch from the difference between the Resource's original and current body. Go takes the explicit RFC 6902 operations (`UpdateOpts{Op, Path, Value}`) and sends them as the PATCH body.
- Python `find_*` first tries a GET with the string as an ID and then falls back to a list match on ID or name. Go separates `resource.ID` (member GET only) and `resource.Name` (exact name match over the detail list for clusters and the list for templates; duplicates are `resource.ErrAmbiguous`). Missing resources are `resource.ErrNotFound` unless `resource.WithIgnoreMissing()` is passed.
- `delete_*` defaults to ignoring a missing resource in Python; `Remove` has the same default and `resource.WithMissingError()` selects strict behavior.

## Waiting

`WaitFor` resolves the reference once, then polls the same ID every two seconds (`resource.WithPollInterval` changes it). The target status is compared case-insensitively. Differences from `wait_for_status`:

- The default Go timeout is five minutes; `resource.WithTimeout` changes it and `resource.WithUnlimitedWait()` removes it (Python defaults to waiting forever). Parent context deadlines always apply.
- Python treats only the exact status `ERROR` as failure. The Go default also treats any status that starts with `error`, ends with `fail` or `failed`, or equals `killed` as a failure (for example `CREATE_FAILED`). `resource.WithFailureStates("ERROR")` reproduces the Python list; an empty list disables failure detection.
- The first observation is always a GET; Python returns a supplied Resource without fetching when its status already matches.
- A progress callback receives `0` after each nonterminal observation, as Python does when the resource has no `progress` attribute.

`WaitForDeletion` completes on a 404, on a status equal to `deleted`, or when the resource is already missing; other errors (for example 403) stop the wait. It never sends a DELETE. Python's `wait_for_delete` defaults to a 120-second bound in the proxy; Go uses the same five-minute default as `WaitFor`.

## Magnum services

`Service.ListMagnumServices(ctx)` issues one `GET /mservices` and yields `MagnumService` records (`ID`, `Binary`, `Host`, `State`, `ReportCount`, `DisabledReason`, `CreatedAt`, `UpdatedAt`). Nullable fields are pointers so null and empty stay distinct. Only 200 is accepted, no query options exist, a continuation link is not followed and unknown response fields are not retained. The route is not part of Gophercloud v2.15.0, so this model is SDK-owned.

```go
service, err := conn.ContainerInfraV1(ctx)
if err != nil { return err }
for item, err := range service.ListMagnumServices(ctx) {
    if err != nil { return err }
    fmt.Println(item.Binary, item.Host, item.State)
}
```

## Cloud helpers

The cloud helpers resolve one string and then act on the selected UUID:

```go
deleted, err := service.DeleteCOECluster(ctx, "my-cluster") // false, nil when nothing matches
id, err := service.UpdateCOECluster(ctx, "my-cluster", []clusters.UpdateOpts{
    {Op: clusters.ReplaceOp, Path: "/node_count", Value: 3},
})
deleted, err = service.DeleteClusterTemplate(ctx, "my-template")
template, err := service.UpdateClusterTemplate(ctx, "my-template", []clustertemplates.UpdateOpts{
    {Op: clustertemplates.ReplaceOp, Path: "/public", Value: false},
})
```

- Cluster helpers list every cluster and select by UUID or name, including fnmatch-style globs. More than one match fails with `resource.ErrAmbiguous` and nothing is sent. A missing cluster is `false, nil` for delete and `resource.ErrNotFound` for update.
- Template helpers try a GET by ID first, fall back to an exact ID or name match when that GET returns 400, 403 or 404 (other errors are returned), and do not expand globs.
- Empty references fail with `resource.ErrInvalidOption` before any request. A 404 on the DELETE that follows a successful selection still reports `true`, and one DELETE or PATCH is sent at most once.
- The `filters` argument of `search_*` and `get_*` (dictionary or JMESPath) and the deprecated `get_cluster_template(filters=...)` are not offered; filter the typed results instead. Those four cloud helpers are therefore not marked as fully mapped.
