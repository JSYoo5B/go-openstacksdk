# Cloud network helpers (round 2)

Composed Neutron helpers matching the pinned `NetworkCloudMixin` methods. All methods are on `network.Service`.

## `ListFirewallGroups`, `ListFirewallPolicies`, `ListFirewallRules`

`list_firewall_groups`, `list_firewall_policies` and `list_firewall_rules` pass a filters dictionary to `Resource.list`. Go takes the same dictionary as a JSON object through `WithFirewallListFilters`.

- Nil and JSON falsey filters (`null`, `false`, `0`, `""`, `[]`, `{}`) mean no filters. A truthy non-object is rejected before any request.
- Keys in the resource query mapping, including `limit`, `marker` and `fields`, are sent to Neutron. Lists become repeated query values (`ports=a&ports=b`); booleans use Python spelling (`shared=True`); `null` and empty lists are omitted.
- Body-only attributes are compared locally with Python equality: `admin_state_up` and `id` for groups, `audited` and `id` for policies, `id` for rules. A number equals a boolean only as 1/0. A nonempty object filter against a truthy scalar fails, as the Python helper raises there; an empty object matches truthy values.
- Any other key is discarded without an error. The Proxy/Resource controls `paginated`, `base_path`, `allow_unknown_params`, `microversion`, `headers`, `max_items` and `jmespath_filters` are rejected rather than interpreted.
- No extension check is made, matching the pinned helpers. Rows use the native Gophercloud models, so unknown body attributes are dropped and an absent field compares as its zero value. Paging follows native `next` links; a server that pages only through `limit` and `marker` without links is not continued.

## `GetQoSRuleTypeDetails`

Reads `qos/rule-types/{name}` after two ordered, uncached extension listings: first `qos`, then `qos-rule-type-details`. A missing alias returns `*UnavailableExtensionError` (matching `ErrUnavailableExtension`) and sends no rule type request. The Python `filters` argument is ignored by the pinned source and has no Go counterpart. A 404 is an error. Neutron's default policy restricts the detail read to administrators; the denial is the server's HTTP error. The result keeps every extension row, and the rule type separates the Resource view, actual wire row, envelope, headers and status.

Behavior is covered by `api/r2_cloud_net_firewall_test.go`.
