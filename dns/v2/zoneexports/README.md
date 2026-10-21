# Zone exports

SDK-owned binding for Designate export tasks. Reached with `service.ZoneExports()`.

| Method | Wire | Success |
|---|---|---|
| `Create(ctx, zoneID)` | `POST /zones/{zone}/tasks/export`, no body | 202 |
| `Get(ctx, id)` | `GET /zones/tasks/exports/{id}` | 200 |
| `GetText(ctx, id)` | `GET /zones/tasks/exports/{id}/export`, `Accept: text/dns` | 200 |
| `Delete(ctx, id)` / `Remove` | `DELETE /zones/tasks/exports/{id}` | 204 |
| `List(ctx, ...)` / `All` | `GET /zones/tasks/exports?zone_id&message&status` | 200 |
| `WaitFor`, `WaitForDeletion` | poll GET; `ERROR` fails, `DELETED` or 404 completes | |

A 202 acknowledges a task; it does not mean the export is `COMPLETE`. List options accept `AllProjects` (`X-Auth-All-Projects`, explicit false is sent) and `SudoProject` (`X-Auth-Sudo-Project-ID`); Get/Delete send no scope headers.

Python differences (documented, deliberate):

- The pinned Python resource uses `/zones/tasks/export` (singular) for list/get/delete; the API reference documents `/zones/tasks/exports`, which Go uses.
- Python `get_zone_export_text` returns a ZoneExport and discards the non-JSON body; Go returns the zone file text with its headers.
