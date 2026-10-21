# Zone imports

SDK-owned binding for Designate import tasks. Reached with `service.ZoneImports()`.

| Method | Wire | Success |
|---|---|---|
| `Create(ctx, zoneFile)` | `POST /zones/tasks/imports`, raw body, `Content-Type: text/dns` | 202 |
| `Get`, `Delete`, `Remove`, `List`, `All`, `WaitFor`, `WaitForDeletion` | as for [zone exports](../zoneexports/README.md) on `/zones/tasks/imports` | |

Create sends the zone file bytes as given (a snapshot, so native re-authentication replays the same bytes). An empty file is sent as an empty body and rejected by the server. A 202 acknowledges the task; wait for `COMPLETE`.

Python difference: the pinned Python `create_zone_import` clears the request body and only sets the `text/dns` content type, so it cannot transmit a zone file; Go sends the documented body. Only `text/dns` is implemented, not the JSON form.
