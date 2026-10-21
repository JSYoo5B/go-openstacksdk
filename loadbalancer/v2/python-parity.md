# Python openstacksdk load balancer (Octavia) parity

This guide compares the Go Octavia packages with the pinned Python `openstack.load_balancer.v2` proxy. Native Gophercloud route, status-code and body behavior is documented in each package README; this guide records only how the Python proxy methods map onto those Go calls and where the two designs deliberately differ. Detailed per-operation contracts and test references are kept in `api/sdk_reviews.json`.

## Mapping

| Python proxy method | Go call |
|---|---|
| `create_/get_/update_/delete_/find_` and list for load balancers, listeners, pools, health monitors, L7 policies, flavors and flavor profiles | `API.Create`, `API.Get`, `API.Update`, `API.Remove` (ignore-missing) or `API.Delete` (strict), `API.Find`, `API.List` / `API.All` in the matching package |
| `get_load_balancer_statistics`, `get_listener_statistics` | `loadbalancers.API.GetStats`, `listeners.API.GetStats` |
| `delete_load_balancer(cascade=True)` | `loadbalancers.API.Delete` with `WithDeleteOptions(DeleteOpts{Cascade: true})`; `false` sends no query, like Python |
| `failover_load_balancer`, `failover_amphora` | `loadbalancers.API.Failover`, `amphorae.API.Failover` (bodyless PUT, 202 only) |
| `configure_amphora` | `amphorae.API.Configure` (bodyless `PUT /octavia/amphorae/{id}/config`, 202 only) |
| `create_member`, `get_member`, `update_member`, `delete_member`, `members`, `find_member` | `pools.API.CreateMember`, `GetMember`, `UpdateMember`, `DeleteMember`, `ListMembers`, or a `pools.API.Members` scope bound to one pool (`FindIdentity` mirrors Python's ID-then-name lookup) |
| `create_l7_rule`, `get_l7_rule`, `update_l7_rule`, `delete_l7_rule`, `l7_rules`, `find_l7_rule` | `l7policies.API.CreateRule`, `GetRule`, `UpdateRule`, `DeleteRule`, `ListRules`, or an `l7policies.API.Rules` scope bound to one policy |
| `wait_for_load_balancer`, `wait_for_status`, `wait_for_delete` | `API.WaitFor` / `API.WaitForDeletion` of the matching package; `resource.WithStatusAttribute`, `WithFailureStates`, `WithTimeout`, `WithUnlimitedWait` and `WithPollInterval` select the Python `attribute`, `failures`, `wait` and `interval` arguments |
| `providers`, `provider_flavor_capabilities` | `providers.API.List`, `providers.API.ListFlavorCapabilities` |
| `quotas`, `get_quota`, `get_quota_default`, `delete_quota` | `quotas.API.ListProjects`, `ProjectQuotaScope.Get`, `API.Defaults`, `ProjectQuotaScope.Reset` |

## Shared differences

- Python methods accept an ID string or a `Resource` and return mutable `Resource` objects with attribute descriptors, dirty-state tracking and an optional response cache. Go takes explicit `resource.Ref` values (`resource.ID` or `resource.Name`) or ID strings and returns the native typed model; the Resource view and the HTTP evidence stay separate.
- Python `**attrs` and `**query` become concrete typed options plus the library-owned `WithCreateField`, `WithUpdateField` and `WithListQuery` extensions. Python-only wire keys, for example `vip_sg_ids` on load balancer create or `vip_qos_policy_id` on list, travel through those extensions.
- Python `find_*` tries a GET by ID, falls back to a list by name on 400/403/404 and defaults to `ignore_missing=True` (with a deprecation warning). Go `Find` is strict by default; add `resource.WithIgnoreMissing()` for the Python default. Duplicate names are an `AmbiguousError` (Python `DuplicateResource`). Amphorae and L7 rules have no name, so only ID lookup applies.
- Python `delete_*` defaults to `ignore_missing=True`. `Remove` ignores a missing object by default; the native `API.Delete` keeps a 404 as a response error.
- Python `update_*` sends a request only when an attribute is dirty. Go always sends the requested PUT, an empty envelope when no field is given.
- Python `wait_for_load_balancer` and `wait_for_status` use the failure list `['ERROR']` with exact comparison; the Go default failure predicate also treats any state starting with `error`, or ending with `fail`/`failed`, and `killed` as failure. `WithFailureStates("ERROR")` reproduces the Python list. `wait_for_delete` has a 120 second default in Python and five minutes in Go; `wait_for_status` has no Python timeout and Go defaults to five minutes unless `WithUnlimitedWait` is given.
- Acknowledgement of any mutation is not completion; wait on `provisioning_status` explicitly. Mutations are never replayed automatically and cancelling a context does not roll back remote work.

## Limitations

- Availability zones and availability zone profiles (`/lbaas/availabilityzones`, `/lbaas/availabilityzoneprofiles`) have no Go call: the pinned Gophercloud release does not bind them and the SDK has not added a native-free implementation. Python proxy methods for them remain unresolved.
- `update_quota`: Python sends the singular `load_balancer` and `health_monitor` keys; the Go quota builder sends the native `loadbalancer` and `healthmonitor` keys and rejects the Python names as alias collisions. Acceptance of the legacy names by the server is not verified here.
- Python resource attributes that Octavia treats as read-only are not offered as typed create or update options.
