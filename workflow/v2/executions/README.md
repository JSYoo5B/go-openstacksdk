# Mistral executions

`executions.API` wraps Gophercloud v2.15.0 `workflow/v2/executions` for the `/executions` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /executions` | 201, 202 | Flat JSON body (no envelope); `input` and `params` are JSON objects; one of `WorkflowID` or `WorkflowName` is required |
| `Get` | `GET /executions/{id}` | 200 | Decodes the string-encoded `input`, `output` and `params`; null `state_info` stays nil |
| `Delete` | `DELETE /executions/{id}` | 202, 204 | Native 404 stays an HTTP response error |
| `List` | `GET /executions` | 200, 204 | Follows the body `next` URL; `Input`, `Output`, `Params` are sent as JSON; `State` uses `op:value` filters |

`Create` accepts `WithCreateField`, `List` accepts `WithListQuery`. A field extension that names a core input fails with `resource.ErrInvalidOption` before any request; nil options fail the same way.

Pinned Gophercloud sets `include_output=1` for `IncludeOutput` and also emits a stray `-=true` query parameter.

Creating an execution only starts it; the 201/202 response reports state `RUNNING` and does not mean the workflow finished. Poll `Get` for `State`. `Remove`, `All`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection` (no name lookup; `Remove` ignores a missing execution by default). Cancelling a context stops the local call and does not stop or roll back a remote execution; a create is never replayed automatically.
