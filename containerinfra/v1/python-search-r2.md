# COE search and get helpers

`Service.SearchCOEClusters`, `GetCOECluster`, `SearchClusterTemplates` and `GetClusterTemplate` are the Go counterparts of the openstacksdk cloud helpers `search_coe_clusters`, `get_coe_cluster`, `search_cluster_templates` and `get_cluster_template`.

```go
found, err := service.SearchCOEClusters(ctx, "prod-*",
    v1.WithSearchFilters(json.RawMessage(`{"status":"CREATE_COMPLETE"}`)))
cluster, err := service.GetCOECluster(ctx, "prod-a")      // nil, nil when nothing matches
template, err := service.GetClusterTemplate(ctx, "k8s")   // GET by ID, then exact ID/name match
```

## Contract

- Search lists every page of `GET /clusters` (or `GET /clustertemplates`) once, then applies the Python identifier selection (exact or fnmatch-style glob on UUID or name; an empty reference selects everything) and the optional local filters. A failed page returns the error and no partial selection.
- Filters are a JSON object of exact, nested matches or a JSON string holding a JMESPath expression. A string filter yields an arbitrary JSON value: `Expression` is true, `Value` holds it and no records are returned.
- Filter keys are the Python Resource attribute names, for example `is_floating_ip_enabled`, `is_master_lb_enabled`, `status_reason`, `is_public`, plus `id` (the UUID) and `uuid`. Timestamps are RFC 3339 strings. The computed Python `location` key is not provided.
- `Get` returns nil when nothing matches and `resource.AmbiguousError` for several matches. `GetCOECluster` always searches. `GetClusterTemplate` without filters is the proxy find (GET by ID, falling back to an exact list match on 400, 403 or 404); with filters it searches. A JMESPath result that is not a record list is `resource.ErrInvalidOption`.
- All calls are reads; context cancellation stops listing and wraps `context.Canceled`.

## Differences from Python

- No `use_direct_get` shortcut and no object/dict short-circuit: Go callers pass a string.
- `detail` of the template helpers is ignored in Python and has no Go counterpart.
- The deprecation warnings of the Python `filters` arguments have no Go equivalent.
- Rows are a typed projection of the Gophercloud model, not Resource dictionaries.
