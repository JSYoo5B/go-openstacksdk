# openstacksdk database comparison

This page records how the pinned openstacksdk `database/v1` proxy maps onto the Go Trove API. Native contracts stay beside the packages ([instances](instances/README.md), [databases](databases/README.md), [users](users/README.md), [flavors](flavors/README.md)).

## Summary

| openstacksdk | Go |
|---|---|
| `create_instance`, `get_instance`, `find_instance`, `delete_instance` | `instances.API.Create`, `Get`, `Find`, `Remove` |
| `find_flavor`, `get_flavor` | `flavors.API.Find`, `Get` |
| `create_database`, `delete_database`, `get_database`, `find_database` | `databases.DatabaseScope.Create`, `Delete`, `Get`, `Find` (scope from `databases.API.InInstance`) |
| `create_user`, `delete_user`, `get_user`, `find_user` | `users.UserScope.Create`, `Delete`, `Get`, `Find` (scope from `users.API.InInstance`) |
| `wait_for_status`, `wait_for_delete` | `instances.API.WaitFor`, `WaitForDeletion` |
| `instances()`, `flavors()`, `databases()`, `users()` | `List`, `All` and scope `List`; Python client-side Body-attribute filters are not offered |
| `update_instance` | only `AttachConfigurationGroup` and `DetachConfigurationGroup` are provided |

## Behavior notes

- Python sends `**attrs`; Go uses concrete options. The Trove body shapes differ where the Python source builds them differently: Python wraps one database as `{"database": {...}}` and one user as `{"users": {...}}`, while Go sends Trove's documented arrays (`{"databases": [...]}`, `{"users": [...]}`) and supports batches. The Python request shapes were read from the pinned source and not exercised against a server.
- Python accepts every status below 400. Go accepts the documented codes only: instance create 200, database and user create 201 or 202, delete 202 or 204. Mutations are never replayed and acknowledgement is not completion.
- Python returns Resource objects built from the response. Database and user creation return no body in Go (errors only); instance creation returns the BUILD-time `Instance` model, not a completed instance.
- `get_database`, `find_database`, `get_user` and `find_user` call `Resource.fetch` on resources whose class does not allow fetching, so the pinned Python entry points raise `MethodNotSupported`. Go provides the intended lookup instead: scopes search the fixed instance's complete list for an exact name (users may also be scoped by host with `users.WithHost`). Missing is `resource.ErrNotFound`, duplicates are `resource.ErrAmbiguous`.
- `delete_user` and `delete_database` take an optional instance argument in Python; when the instance is omitted the URL cannot be formed. Go requires the parent when the scope is created.
- Python `find_instance` and `find_flavor` try a GET with the string as an ID, then fall back to a list match on ID or name. Go separates `resource.ID` from `resource.Name`; names are exact and duplicates are ambiguous.
- `delete_*` ignores a missing resource by default in Python and in `Remove`; `resource.WithMissingError()` makes it strict.

## Waiting

`instances.API.WaitFor` resolves the reference once and polls the same ID every two seconds, comparing the target case-insensitively. Differences from `wait_for_status`: the default timeout is five minutes (`resource.WithUnlimitedWait()` removes it; Python waits forever), the Go default failure rule also treats statuses such as `FAILED` as failure (Python: only `ERROR`; `resource.WithFailureStates("ERROR")` reproduces it), the first observation is always a GET, and progress callbacks receive `0` after each nonterminal observation.

`WaitForDeletion` completes on a 404 or a status equal to `deleted`, stops on other errors such as 403, and never sends a DELETE. Python's proxy default bound for `wait_for_delete` is 120 seconds; Go uses five minutes.

## Not mapped

- Python list generators accept body attributes as client-side filters (for example `databases(instance, character_set=...)`); Go offers exact-name and status filtering through Collections and scopes only.
- `update_instance` commits arbitrary changed attributes with `PUT /instances/{id}`; Go has no general instance update.
