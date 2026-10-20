# Barbican native ACLs

`acls.New(client)` exposes the pinned Gophercloud `v2.15.0`
[ACL requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/keymanager/v1/acls/requests.go)
unchanged. Each secret and container has an ACL at `secrets/{id}/acl` or
`containers/{id}/acl`; IDs are joined into the path without escaping.

```go
api := acls.New(client)
users := []string{userID}
access := false
ref, err := api.SetSecretACL(ctx, secretID, acls.SetOpts{{Type: "read", Users: &users, ProjectAccess: &access}})
if err != nil { return err }
fmt.Println(*ref)
```

| Method (secret / container) | Request | Result |
|---|---|---|
| `GetSecretACL` / `GetContainerACL` | `GET .../acl` | `ACL`, a map from type to `ACLDetails` |
| `SetSecretACL` / `SetContainerACL` | `PUT .../acl`, replacing the ACL | `ACLRef` from `acl_ref` |
| `UpdateSecretACL` / `UpdateContainerACL` | `PATCH .../acl`, updating the given types | `ACLRef` |
| `DeleteSecretACL` / `DeleteContainerACL` | `DELETE .../acl` | error only |

Every call accepts only status 200. `SetOpts` is a list of `SetOpt{Type, Users,
ProjectAccess}`; `Type` is required before HTTP and becomes the body key, as in
`{"read": {"users": [...], "project-access": false}}`. Nil `Users` and
`ProjectAccess` are omitted, so `{"read": {}}` is valid. `ACLDetails` decodes
`created`/`updated` with the native RFC3339-without-zone parser.

The `With...ACLField` options add extension JSON. With a single ACL type the
body is one object envelope, so the shared merge places the field inside that
type's object; with several types it is added at the top level. A key already
present at that level is rejected before HTTP. Other statuses keep the native
`gophercloud.ErrUnexpectedResponseCode`; the SDK adds only
`resource.OperationError{Resource: "acls"}` context.
