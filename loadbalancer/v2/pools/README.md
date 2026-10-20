# Octavia pools and members

`pools.API` wraps Gophercloud v2.15.0 `loadbalancer/v2/pools` for `/lbaas/pools` and `/lbaas/pools/{pool}/members`. Python openstacksdk Resource and workflow equivalence is tracked separately.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /lbaas/pools` | 201, 202 | Body wrapped in `pool`; nested `Members` and `Persistence` are serialized by the native builders |
| `Get` | `GET .../{id}` | 200 | Typed model; extension JSON is not retained |
| `Update` | `PUT .../{id}` | 200 only | Pointer fields distinguish omitted from empty; an empty `SessionPersistence{}`, empty `ALPNProtocols` and empty `TLSVersions` are sent as JSON `null` |
| `Delete` | `DELETE .../{id}` | 202, 204 | No options; native 404 stays an HTTP response error |
| `List` | `GET /lbaas/pools` | 200 | Follows `pools_links` next links; `AdminStateUp` is a pointer so `false` is sent |
| `CreateMember` | `POST .../{pool}/members` | 201, 202 | Body wrapped in `member`; `Weight`, `AdminStateUp`, `Backup` and `MonitorPort` pointers send explicit `0`/`false` |
| `GetMember` | `GET .../{pool}/members/{id}` | 200 | Typed `Member` |
| `UpdateMember` | `PUT .../{pool}/members/{id}` | 200, 201, 202 | Pointer fields send explicit empty values |
| `DeleteMember` | `DELETE .../{pool}/members/{id}` | 202, 204 | Native 404 stays an HTTP response error |
| `ListMembers` | `GET .../{pool}/members` | 200 | Follows `members_links` next links |
| `BatchUpdateMembers` | `PUT .../{pool}/members` | 202 only | Replaces the member set with `{"members":[...]}`; a nil or empty slice sends `{"members":[]}`; an empty `SubnetID` is sent as `null` |

Extension inputs use `WithCreateField`, `WithUpdateField`, `WithListQuery`, `WithCreateMemberField`, `WithUpdateMemberField` and `WithListMembersQuery`. A field extension that names a core input field fails with `resource.ErrInvalidOption` before any request; nil options fail the same way. `Delete`, `Get`, `GetMember`, `DeleteMember` and `BatchUpdateMembers` take no extensions.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection` for pools. `API.Members(ctx, parent)` returns a `MemberScope` bound to one pool: an explicit `resource.ID` parent needs no lookup request, and the scope offers `Create`, `Update`, `FindIdentity` and the common collection operations (ignore-missing `Delete`). The native methods (`API.Delete`, `API.DeleteMember`) do not ignore a missing object.

Limitations:

- `UpdateMemberOpts.Tags` and `BatchUpdateMemberOpts.Tags` are plain slices with `omitempty`; an empty slice is omitted and cannot clear tags. `CreateOpts` non-pointer fields (`TLSEnabled`, `Name`) cannot send an explicit zero value on create.
- `WithStatus` is applied locally to `provisioning_status`; it is not sent as a query parameter.
- The nested `Monitor` (health monitor) create input follows the native builder; the SDK adds no validation to it.
- `BatchUpdateMembers` replaces the whole member set and has no per-member result; callers must list members to observe the outcome.

Acknowledgement of any mutation does not imply completion; wait on `provisioning_status` explicitly. Cancelling a context stops the local call and does not roll back remote work; mutations (including `BatchUpdateMembers`) are never replayed automatically.
