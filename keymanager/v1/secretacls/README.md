# Barbican secret ACLs: Python and Go

`service.SecretACLs.InSecret(ctx, ref)` fixes one secret and provides the
pinned openstacksdk `get_secret_acl`, `set_secret_acl`, `update_secret_acl`
and `delete_secret_acl` proxy operations. For the raw Gophercloud ACL calls,
including container ACLs and the PATCH update, use [`acls`](../acls/README.md).

| Python `conn.key_manager` | Go `SecretScope` | Request |
|---|---|---|
| `get_secret_acl(secret)` | `Get(ctx, options...)` | `GET secrets/{id}/acl` |
| `set_secret_acl(secret, **attrs)` | `Set(ctx, ACLInput{...}, options...)` | `PUT secrets/{id}/acl` |
| `update_secret_acl(secret, **attrs)` | `Update(ctx, ACLInput{...}, options...)` | `PUT secrets/{id}/acl` |
| `delete_secret_acl(secret, ignore_missing=True)` | `Delete(ctx, options...)` | `DELETE secrets/{id}/acl` |

```python
acl = conn.key_manager.get_secret_acl(secret_id)
conn.key_manager.set_secret_acl(secret_id, read={"users": [user_id], "project-access": False})
conn.key_manager.delete_secret_acl(secret_id)
```

```go
scope, err := service.SecretACLs.InSecret(ctx, resource.ID(secretID))
if err != nil { return err }
acl, err := scope.Get(ctx)
if err != nil { return err }
fmt.Println(string(acl.Read))
_, err = scope.Set(ctx, secretacls.ACLInput{Read: json.RawMessage(`{"users":["` + userID + `"],"project-access":false}`)})
if err != nil { return err }
_, err = scope.Delete(ctx)
```

The Python string argument is an ID; `InSecret` also accepts a `resource.Name`
and resolves it once through the Secrets collection as a Go convenience.
Whitespace, control characters and empty IDs fail before HTTP.

## Pinned Python behavior

The `SecretACL` Resource declares only `read` (a dict) and `acl_ref`; other
keyword attributes are dropped. Like `Resource.fetch` and `commit`, every
actual status below 400 is accepted. A JSON object response overlays the
declared fields, an empty or invalid body keeps the seeded values, and a valid
non-object body (or a non-object `read`) fails with the actual response as a
`resource.ResponseError`. `Read` and `ACLRef` are JSON `null` when absent.

`set_secret_acl` and `update_secret_acl` both call `_update`, which commits
with PUT; the pinned SDK never sends PATCH here. When `ACLInput` has neither
`Read` nor `ACLRef`, nothing is dirty and the call returns the seeded ACL
without HTTP, as Python does. `Read` must be a JSON object. `Delete` ignores a
clean 404 by default (`nil, nil`); `WithDeleteIgnoreMissing(false)` keeps it.
Headers from `WithHeader`/`WithDeleteHeader` are a Go extension; the SDK-owned
authentication and content headers are rejected. The mutable Python Resource,
its session and cache are not reproduced.
