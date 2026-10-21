# Senlin list, wait and update comparison with openstacksdk

This guide records how the Go Senlin APIs compare with the pinned Python
openstacksdk `clustering/v1` proxy for the inherited `Resource.list`,
`wait_for_status`, `wait_for_delete` and `Resource.commit` behavior. It
complements the per-package READMEs. Evidence is HTTP fixtures; no live Senlin
service was exercised.

## List operations

The pinned Python proxy methods `actions`, `events`, `clusters`, `nodes`,
`profiles`, `policies`, `receivers`, `policy_types`, `profile_types`, `services`
and `cluster_policies` delegate to `Proxy._list` and `Resource.list`. Observed
behavior and the Go decision:

| Python behavior | Go behavior |
|---|---|
| `Proxy._list` accepts a per-call `base_path` and the deprecated `jmespath_filters` (it warns `RemovedInSDK60Warning`) among the keyword arguments. | Routes belong to the SDK and neither option exists. |
| `Resource.list` validates keywords with `allow_unknown_params=True`: a keyword that is neither a declared query key nor a Body attribute is silently discarded, and a Body attribute that is not a query key becomes a client-side filter. | Declared server filters are typed list options. `WithListQuery` sends an explicit extension query to the server; reserved names and names owned by a typed option are rejected. Local Body filters use `WithListFilter` with source-validated names and fail before HTTP when unknown. |
| Query aliases map attribute names to wire names (for example `target_id` to `target`, `obj_id` to `oid`, `obj_name` to `oname`, `obj_type` to `otype`, `user_id` to `user`, `is_enabled` to `enabled`). | Go uses wire names and applies no Python alias. |
| `max_items` counts raw rows, including rows removed by client filters; with `max_items` and no `limit` it sets `limit=max_items`. | `MaxItems` counts raw wire rows before local filters and supplies a limit hint only where the binding allows it. |
| With an explicit `limit` and no next link the next request uses the last row's `Resource.id` as the marker, including short pages, until an empty page. | The explicit-limit fallback uses the retained wire ID, ends on an empty page, and rejects a repeated marker. Catalog endpoints (policy types, profile types, services) declare no paging protocol, so Go has no marker fallback for them. |
| Rows are rebuilt with `Resource.existing`, which applies descriptor coercion and defaults. | Rows are typed snapshots that retain the raw JSON, so filters compare exact raw JSON values. |
| `collect_cluster_attrs` passes `cluster_id=cluster` and `path` to `_list` and drops its own `**query`. | The attribute scope resolves the cluster once and List/All take only SDK options. Python formats the `cluster_id` value into the URL as given, which is only correct for an ID string. |

## Waits

`wait_for_status` returns the passed resource at once, with no request, when its
cached attribute already equals the target. Otherwise it fetches with
`skip_cache=True` until the target; default failures are `ERROR`, the default
interval is 2 seconds, `wait=None` waits forever and the callback receives the
`progress` attribute or 0. `wait_for_delete` defaults to a 120 second wait,
returns the original resource when a fetch is missing or reports `deleted`, and
passes the same progress value to its callback.

The Go waiters always begin with a request, so a stale cached object cannot
satisfy a wait. They take a reference rather than a Resource, return a typed
snapshot (or only an error for deletion), accept a nonempty target and typed
string attributes and report a progress of 0 for Senlin models.

## Action update

`update_action` calls `Proxy._update`, which commits changed attributes with
PATCH `/actions/{id}`. A call with no attributes sends no request and returns the
Resource holding the submitted values as if they were applied. Go sends the
published `{"action": {"status": "CANCELLED"}}` request (numeric microversion 1.12
or later, optional `force` query), requires a status, and returns the accepted
response evidence without fabricating an updated Action.
