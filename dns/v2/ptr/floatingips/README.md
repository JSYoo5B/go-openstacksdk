# Floating IP PTR records

SDK-owned binding for Designate reverse DNS. Reached with `service.FloatingIPPTR()`. Identifiers are `<region>:<floatingip-id>`.

| Method | Wire | Success |
|---|---|---|
| `List(ctx, ...)` / `All` | `GET /reverse/floatingips` (paging, `AllProjects`, `SudoProject`) | 200 |
| `Get(ctx, id)` | `GET /reverse/floatingips/{id}` | 200 |
| `Set(ctx, id, SetOpts, ...)` | `PATCH` with only supplied `ptrdname`, `description`, `ttl` | 200, 202 |
| `Unset(ctx, id)` | `PATCH {"ptrdname": null}` | 200, 202 |
| `WaitFor` | polls GET until the status matches; `ERROR` fails | |

`request.Present` keeps `0`, `""`; `request.Null` sends null. The body never carries the identifier (Python sends its dirty attributes, which may include `id`). A 202 is an acknowledgement, not an `ACTIVE` record. There is no delete route; `Unset` is Python `unset_floating_ip`.
