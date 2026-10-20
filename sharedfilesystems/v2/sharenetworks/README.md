# Manila share networks

`sharenetworks.API` wraps Gophercloud v2.15.0 `sharedfilesystems/v2/sharenetworks` for the `/share-networks` routes. Python openstacksdk Resource and workflow equivalence is tracked separately. The service client's `Microversion` is sent as `X-OpenStack-Manila-API-Version` for the native Manila client types; this package does not negotiate it. Newer Manila releases use share network subnets for network settings, and this binding exposes only the pinned Gophercloud fields.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST /share-networks` | 200, 202 | Body wrapped in `share_network`; `name` and `description` are always sent (empty strings included), empty network IDs are omitted |
| `Get` | `GET /share-networks/{id}` | 200 | Timestamps use the Manila no-zone format; null `updated_at` decodes to the zero time |
| `Update` | `PUT /share-networks/{id}` | 200 only | `Name`, `Description` are pointers: nil omits, `""` is sent; empty network IDs are omitted |
| `Delete` | `DELETE /share-networks/{id}` | 202, 204 | Native 404 stays an HTTP response error |
| `ListDetail` | `GET /share-networks/detail` | 200 | `ListOpts` filters; false/zero filters are omitted |
| `AddSecurityService`, `RemoveSecurityService` | `POST /share-networks/{id}/action` | 200 only | `add_security_service` / `remove_security_service` envelope with `security_service_id` (always sent) |

`ListDetail` pages by offset. With `ListOpts.Limit` set, the next request uses `offset = offset + limit` and stops at the first empty page. Without a limit the pager still issues one more request at the maximum integer offset and expects an empty page. Paging follows no `next` links.

Extension inputs use `WithCreateField`, `WithUpdateField`, `WithAddSecurityServiceField`, `WithRemoveSecurityServiceField` and `WithListDetailQuery`. A field extension that names a core input field fails with `resource.ErrInvalidOption` before any request; nil options fail the same way.

`Find`, `All`, `Remove`, `WaitFor` and `WaitForDeletion` use the common `resource.Collection`: exact name lookup, duplicate-name errors, and ignore-missing deletion by default. `API.Get` and `API.Delete` do not add `resource.ErrNotFound` or ignore a missing share network.

Deleting or updating a share network that is in use is rejected by the server and surfaces as an HTTP response error. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
