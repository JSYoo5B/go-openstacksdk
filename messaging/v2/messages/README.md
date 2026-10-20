# Zaqar messages

`messages.API` wraps Gophercloud v2.15.0 `messaging/v2/messages` for queue-scoped `/v2/queues/{queue}/messages` routes. Python openstacksdk Resource and workflow equivalence is tracked separately. Every method takes the queue name explicitly; there is no common `resource.Collection`.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Create` | `POST .../messages` | 201 only | Body `{"messages":[...]}`; each item sends `ttl`/`delay` only when non-zero and always sends `body` (an empty map stays `{}`) |
| `Get` | `GET .../messages/{id}` | 200 | Typed `Message` |
| `GetMessages` | `GET .../messages?ids=...` | 200 | `GetMessagesOpts.IDs` are sent as repeated `ids` parameters; with no options no query is sent |
| `List` | `GET .../messages` | 200, 204 | Follows `links` next entries; 204 ends iteration; `Echo`, `IncludeClaimed`, `IncludeDelayed` send `true` only |
| `Delete` | `DELETE .../messages/{id}` | 204 only | `DeleteOpts.ClaimID` sends `claim_id`; native 404 stays an HTTP response error |
| `DeleteMessages` | `DELETE .../messages?ids=...` | 200, 204 | With no options the request carries no query |
| `PopMessages` | `DELETE .../messages?pop=N` | 200, 204 | 204 yields no messages; results include `ClaimID` and `ClaimCount` |
| `DeleteMessageURL` | none | none | Returns the absolute single-message URL without a request |

`Create` accepts `WithCreateField`; the `messages` envelope key is protected. `Delete`, `DeleteMessages`, `GetMessages`, `List` and `PopMessages` accept their `With...Query` extensions. Collisions and nil options fail with `resource.ErrInvalidOption` before any request. Message bodies are `map[string]any`; non-object bodies are not supported by the native model.

The native wrappers do not add a `Client-ID` header. Deployments that require one must receive it from the caller's client configuration.

`DeleteMessages` without `IDs` and `PopMessages` without `Pop` send an unfiltered `DELETE` on the messages collection; callers must supply the filter they intend. Acknowledgement does not imply that a pop or delete is observable by a later read. Cancelling a context stops the local call and does not roll back remote work; mutations are never replayed automatically.
