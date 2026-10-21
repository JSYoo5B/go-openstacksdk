# Blacklists

SDK-owned binding for Designate `/v2/blacklists` (Gophercloud v2.15.0 has none). Reached with `service.Blacklists()` or `blacklists.New(client)`.

| Method | Wire | Success |
|---|---|---|
| `Create(ctx, CreateOpts, ...CreateOption)` | `POST /blacklists`, flat `{pattern, description}` | 201 |
| `Get(ctx, id)` | `GET /blacklists/{id}` | 200 |
| `Update(ctx, id, UpdateOpts, ...UpdateOption)` | `PATCH /blacklists/{id}`, only supplied fields | 200 |
| `Delete(ctx, id)` | `DELETE /blacklists/{id}` | 204 |
| `Remove(ctx, ref, ...)` | delete by reference, missing ignored by default | 204 |
| `List(ctx, ...ListOption)` / `All` | `GET /blacklists?pattern=...` | 200 |
| `Find(ctx, ref, ...)` | ID only (blacklists have no name) | 200 |

Rules:

- Bodies are flat. `request.Optional` distinguishes omitted, `Present` (including `""`) and `Null`. `WithCreateField` / `WithUpdateField` add extension fields; a name that collides with a core field fails before HTTP.
- Results embed the exact decoded object (`Body`), `Header` and `StatusCode`; numbers and null stay inspectable.
- Success codes are explicit and never broadened. A 404 on Get/Update/Delete wraps `resource.ErrNotFound`.
- Lists follow `links.next` (same origin only), stop at `metadata.total_count`, and with an explicit positive `Limit` continue with `marker=<last id>` until an empty page; repeated pages and repeated markers fail.
- Mutations are never replayed by the SDK. A 204 is the final answer for delete; a cancelled context makes no request and does not roll back remote work.

Python mapping: `blacklists`, `create_blacklist`, `get_blacklist`, `update_blacklist`, `delete_blacklist`; see [the R2 mapping](../r2-python-mapping.md).
