# Workflow v2: openstacksdk Python comparison

This page records how the pinned openstacksdk `workflow/v2` proxy methods map to the Go workflows, executions and crontriggers APIs. Go behavior is verified by `api/py_msg_wf_mistral_test.go` and the Mistral tests cited from the review ledger.

## Update

`workflows.API.Update(ctx, workflows.UpdateOpts{Scope, Namespace, Definition})` sends one `PUT /workflows` with the definition as `text/plain`, no JSON `Accept` header, the optional scope/namespace/extension query, and requires 200. Mistral selects workflows by the names in the definition, so Python's `workflow` argument does not appear in the URL and Go takes no reference. Differences: Python sends `?scope=None` when no scope was set while Go omits an empty scope; Python ignores the response body while Go decodes the returned workflow list; Go rejects a missing definition before HTTP. The call is never replayed and the one-shot definition reader is consumed.

## Find

`FindIdentity` on each API follows `find_workflow`, `find_execution` and `find_cron_trigger`: GET with the literal identity first, then a complete list only after a clean 400/403/404, matching the exact ID or name on every page and failing on duplicates. Missing results are `(nil, nil)` by default and an error with `resource.WithIdentityFindIgnoreMissing(false)`. Executions have no name, so only IDs match. The cron-trigger list fallback carries `name=<identity>`; the workflow and execution fallbacks send no name filter, as in Python. `crontriggers.API.FindIdentityAllProjects` is Python's `all_projects=True`: the list fallback adds `all_projects=1` (administrator only). Python's default sends `all_projects=False` on the list fallback; Go omits the key. The typed `resource.WithIdentityFindAllProjects` option and a raw `all_projects` key are not accepted by these bindings. Python's `**query` for cron triggers is not supported: the SDK rejects `resource.WithIdentityFindQuery` for these bindings with `ErrUnsupported` before HTTP because the native GET takes no query. The existing `Find(resource.Name(...))` remains a name-filtered list lookup.

## Delete and list

`Remove(ctx, resource.ID(x))` is `delete_*` with Python's default `ignore_missing=True`; `resource.WithMissingError()` selects strict deletion. `API.Delete` is the strict native call. Python accepts any non-error delete status while Go's native Delete accepts 202 and 204. Lists follow the body `next` link as Python does; Go exposes typed filters plus raw query extensions for keys such as `fields`, `sort_keys` and `sort_dirs`. Python `all_projects=True` is sent as `True`; Go sends `1` or the raw value passed.

## Waiting

Only executions carry a status in Python (the body field `state`). Go `executions.API.WaitFor` needs `resource.WithStatusAttribute("state")`; pair it with `resource.WithFailureStates("ERROR")` for Python's default failure list and `resource.WithUnlimitedWait()` for Python's default of no timeout. Comparison is case-insensitive and the progress callback receives 0 after each nonterminal read. Workflows and cron triggers have no status, so `WaitFor` returns `ErrUnsupported`. `WaitForDeletion` polls GET until a 404; the shared default timeout is five minutes (Python's `wait_for_delete` default is 120 seconds).
