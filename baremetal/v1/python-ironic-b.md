# Deploy templates, inspection rules and runbooks versus Python

The pinned Gophercloud release has no binding for these three Ironic
collections, so the SDK owns [deploytemplates](deploytemplates/README.md),
[inspectionrules](inspectionrules/README.md) and [runbooks](runbooks/README.md).
They are reached through `Service.DeployTemplatesAPI()`,
`Service.InspectionRulesAPI()` and `Service.RunbooksAPI()` on the same
authenticated client as the native resources. The introspection `/rules` API is
described in [rules](../../baremetalintrospection/v1/rules/README.md).

## Shared contracts

- One request per call. A failed or uncertain mutation is never replayed;
  Python retries HTTP 409 and 503 on these calls.
- Exact success statuses: create 201, get/list/patch 200, delete and trait
  routes 204. Python accepts any status below 400.
- The client microversion is used as configured and never raised. Python
  negotiates min(server maximum, class cap): deploy templates 1.55, runbooks
  1.112 (routes 1.92, description and traits 1.112), inspection rules 1.96.
  Configure `sdk.WithMicroversion(sdk.BareMetal, ...)` accordingly.
- Python `**attrs` become concrete `CreateOpts`/`UpdateOpts`; `With...Field`
  adds members the package does not model but never replaces a typed member.
  Python silently drops unknown attributes.
- Python `update_*` builds a JSON patch from the dirty attributes: one `add`
  operation per attribute, sorted by path. Go `Update` sends the same
  operations. An update that sets nothing sends `PATCH []` in Python
  (the uuid stays dirty); Go rejects it before HTTP. `Patch` takes explicit
  operations with server member paths. Python rewrites SDK attribute names to
  server names and does not validate the operation; Go validates op, path and
  value presence and does not rewrite names.
- `Delete` ignores a missing resource by default and returns `deleted=false`,
  like `ignore_missing=True`; `WithDeleteIgnoreMissing(false)` returns
  `resource.NotFoundError` wrapping the native 404. Python returns the
  resource instance; Go returns a boolean.
- `Get` accepts `WithGetFields(server member names...)`, sent as a CSV `fields`
  query. Python also maps SDK attribute names (for example `id`) to server names.
- List: `WithListDetail(true)` sends `detail=true` (false sends nothing, as
  Python). `WithListLimit` is a page size; without a published `next` link the
  walk continues with `marker=<last uuid>` until an empty page, like Python.
  A next link is used only for its query and only if it continues the same
  collection route; the link's host is never contacted. A repeated marker
  returns `resource.ErrPaginationCycle`. `WithListMaxItems` caps rows and, without
  a limit, is sent as the limit; Go stops at the cap without an extra request.
  Python's client-side filtering of Body attributes is not reproduced.
- `inspection_rules(details=True)` in Python writes `details`, which is not a
  query parameter of the resource, so it is silently dropped. Go sends the
  intended `detail=true`.
- `find_deploy_template` in Python passes `details=True` into the direct fetch
  as an unrecognised query and discards it in the fallback list. Go reads the
  reference directly, falls back on 400/403/404 to one complete list matched by
  exact uuid or name, and applies `WithFindDetail` (default true) as `detail=true`
  on that list. Two matches return `resource.AmbiguousError`.
- Runbook trait routes escape the trait as one path segment; Python joins it
  unescaped. `SetTraits(nil)` sends `{"traits":[]}`. Python also updates the local
  object's traits, which Go does not keep.
- Results are typed models with `Metadata` (raw members, headers, status), not
  Resource objects.

## Unverified schema detail

Step, condition and action member names (`order` for runbook steps, `priority`
for deploy template steps, `op/args/loop/multiple` for inspection rules) follow
the Ironic API reference as understood from the Python resource definitions,
which pass lists through unchanged. Unmodelled members can be sent through the
`Extra` maps. No test runs against a live Ironic.
