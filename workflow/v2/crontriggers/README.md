# Mistral cron triggers

`crontriggers.API` wraps Gophercloud v2.15.0 `workflow/v2/crontriggers` for the `/cron_triggers` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /cron_triggers` | 201, 202 | Flat JSON body; `FirstExecutionTime` is formatted `YYYY-MM-DD HH:MM` (seconds are dropped); one of `WorkflowID` or `WorkflowName` is required |
| `Get` | `GET /cron_triggers/{id}` | 200 | Decodes string-encoded `workflow_input`/`workflow_params`; null times stay nil |
| `Delete` | `DELETE /cron_triggers/{id}` | 202, 204 | Native 404 stays an HTTP response error |
| `List` | `GET /cron_triggers` | 200, 204 | Follows the body `next` URL; `RemainingExecutions`, `Name` and date filters render `op:value` |

`Create` accepts `WithCreateField`, `List` accepts `WithListQuery`. A field extension that names a core input fails with `resource.ErrInvalidOption` before any request; nil options fail the same way. `RemainingExecutions` is omitted from the create body when zero.

`Remove`, `All`, `Find`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`; `Remove` ignores a missing trigger by default. Creating a trigger only schedules it; it does not imply an execution has run. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
