# Python `update_quota` wire spelling

Pinned Python `openstack/load_balancer/v2/quota.py` maps its attributes to the singular body keys `load_balancer` and `health_monitor` (and `listener`, `pool`, `member`) and defines no L7 limits. The Octavia API reference documents the request keys `loadbalancer`, `healthmonitor`, `listener`, `member`, `pool`, `l7policy` and `l7rule` for `PUT /lbaas/quotas/{project_id}` (202) and does not mention `load_balancer` or `health_monitor` for updates, although responses may carry both spellings.

`ProjectQuotaScope.Update` sends the documented keys. It rejects `load_balancer` and `health_monitor` supplied through `WithUpdateField` because they alias core fields. No option emits the Python spelling: whether a server accepts those legacy keys on update has not been verified against a deployment or a fixture, so a spelling option would be unsupported behavior. The Python `update_quota` therefore remains unresolved until server acceptance is established.
