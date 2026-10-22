# Python `update_quota` wire spelling

Pinned Python `openstack/load_balancer/v2/quota.py` maps its attributes to the body keys `load_balancer` and `health_monitor` (and `listener`, `pool`, `member`) and defines no L7 limits. The Octavia API reference documents the request keys `loadbalancer`, `healthmonitor`, `listener`, `member`, `pool`, `l7policy` and `l7rule` for `PUT /lbaas/quotas/{project_id}` (202).

Octavia's request type `QuotaBase` (`octavia/api/v2/types/quotas.py`, reviewed at upstream octavia `12.0.0.0rc1-74-g12d8e0de`) declares both spellings. The underscore names are marked "Misspelled version, deprecated in Rocky". `QuotaBase.to_dict` stores `loadbalancer` under `load_balancer` and `healthmonitor` under `health_monitor`, so either spelling updates the same quota column; when both are sent, the documented key wins. Responses carry both spellings.

`ProjectQuotaScope.Update` sends the documented keys by default. `WithLegacyQuotaKeys(true)` sends `load_balancer` and `health_monitor` instead, which matches the Python wire body. Only those two key names change. Values, explicit `null` inheritance from `WithDefaultLimit` and the other limits are serialized identically, and the last `WithLegacyQuotaKeys` option wins. `WithUpdateField` still rejects `load_balancer`, `health_monitor`, `loadbalancer` and `healthmonitor` in every mode, so a request never carries both spellings.

Server acceptance of the deprecated keys comes from the source review above and from fixtures that pin the exact request body (`api/py_lb_quota_spelling_test.go`). It has not been verified against a deployment, and a future Octavia release may remove the deprecated names.

Other Python differences:

- Python `Quota` has no `l7policy`/`l7rule` attributes and does not allow unknown body attributes, so `update_quota(project, l7policy=5)` silently drops the key. Go sends `L7Policy`/`L7Rule` as typed fields.
- Python commits only dirty attributes and sends nothing when no attribute changed. Go always sends exactly one PUT, with an empty `quota` object when no limit is set, and never replays it.
- Python `Quota._prepare_request` removes `id` from the body. Go takes the project only from the scope and rejects `id`/`project_id` extensions before HTTP.

```python
conn.load_balancer.update_quota("project-id", load_balancers=10, health_monitors=-1)
```

```go
func updateLikePython(ctx context.Context, scope *quotas.ProjectQuotaScope) error {
    ten, unlimited := 10, -1
    _, err := scope.Update(ctx, quotas.UpdateOpts{Loadbalancer: &ten, Healthmonitor: &unlimited},
        quotas.WithLegacyQuotaKeys(true)) // {"quota":{"load_balancer":10,"health_monitor":-1}}
    return err
}
```
