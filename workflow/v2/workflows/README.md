# Mistral workflows

`workflows.API` wraps Gophercloud v2.15.0 `workflow/v2/workflows` for the `/workflows` routes. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /workflows?scope=&namespace=` | 201, 202 | Body is the raw definition text (`Content-Type: text/plain`, no JSON `Accept`); returns every created workflow |
| `Get` | `GET /workflows/{id}` | 200 | `UpdatedAt` is nil when null |
| `Delete` | `DELETE /workflows/{id}` | 202, 204 | Native 404 stays an HTTP response error |
| `List` | `GET /workflows` | 200, 204 | Follows the body `next` URL; `Tags` are comma-joined; `ListFilter` and `ListDateFilter` render `op:value` (dates as `YYYY-MM-DD HH:MM:SS`) |

Create has no JSON body, so it accepts `WithCreateOptions` and `WithCreateQuery` only; List accepts `WithListOptions` and `WithListQuery`. Nil options fail with `resource.ErrInvalidOption` before any request.

`Remove`, `All`, `Find`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`. `Remove` ignores a missing workflow by default and returns `resource.ErrNotFound` with `resource.WithMissingError()`; `API.Delete` itself does not ignore 404.

Create and Delete acknowledgements do not prove later visibility. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
