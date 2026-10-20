# Octavia L7 policies and rules

`l7policies.API` wraps Gophercloud v2.15.0 `loadbalancer/v2/l7policies` for `/lbaas/l7policies` and the nested `/lbaas/l7policies/{policy}/rules` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /lbaas/l7policies` | 201, 202 | Body wrapped in `l7policy`; may carry nested `Rules`; `AdminStateUp` is a pointer so `false` is sent |
| `Get` | `GET .../{id}` | 200 | Typed model; extension JSON is not retained |
| `Update` | `PUT .../{id}` | 200 only | `Name`, `Description`, `Tags`, `AdminStateUp` and the redirect strings are pointers. An empty `RedirectPoolID`, `RedirectURL` or `RedirectPrefix` is sent as JSON `null`; zero `Position` and `RedirectHttpCode` are omitted |
| `Delete` | `DELETE .../{id}` | 202, 204 | Native 404 stays an HTTP response error |
| `List` | `GET /lbaas/l7policies` | 200 | Follows `l7policies_links`; `ListOpts.AdminStateUp` is a plain bool, so `false` is omitted from the typed query; send `false` with `WithListQuery("admin_state_up", "false")` |
| `CreateRule` | `POST .../{policy}/rules` | 201, 202 | Body wrapped in `rule`; `Invert` is a plain bool (omitted when false), `AdminStateUp` is a pointer |
| `GetRule` | `GET .../{policy}/rules/{rule}` | 200 | Typed rule model |
| `UpdateRule` | `PUT .../{policy}/rules/{rule}` | 200, 201, 202 | `Key`, `Invert`, `AdminStateUp` and `Tags` are pointers; an empty `Key` is sent as `null` |
| `DeleteRule` | `DELETE .../{policy}/rules/{rule}` | 202, 204 | Native 404 stays an HTTP response error |
| `ListRules` | `GET .../{policy}/rules` | 200 | Follows `rules_links`; `ListRulesOpts.Invert` and `AdminStateUp` are plain bools, so `false` needs `WithListRulesQuery` |

Extension inputs use `WithCreateField`, `WithUpdateField`, `WithListQuery`, `WithCreateRuleField`, `WithUpdateRuleField` and `WithListRulesQuery`. A field extension that names a core input field fails with `resource.ErrInvalidOption` before any request; nil options fail the same way. Missing required inputs (`Action`, rule `RuleType`/`CompareType`/`Value`) fail locally.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` on `API` use the common `resource.Collection` for policies (exact name lookup, ignore-missing removal by default). `API.Delete` itself does not ignore a missing policy.

`API.Rules(ctx, parent)` returns a `RuleScope` fixed to one policy. An explicit `resource.ID` parent needs no lookup request; a `resource.Name` parent is resolved once. The scope offers `Create`, `Update` (by rule reference), and the common `Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` over rules by ID. Rules have no name lookup. `All` with `resource.WithStatus` filters client-side on `provisioning_status`; `status` is never sent as a query for rules.

Acknowledgement of a mutation does not imply completion; wait on `provisioning_status` explicitly. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
