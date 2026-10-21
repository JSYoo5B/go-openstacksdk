# Messaging v2: openstacksdk Python comparison

This page records how the pinned openstacksdk `message/v2` proxy methods map to the Go queues, messages, claims and subscriptions APIs. Behavior is compared against the pinned Python source; Go behavior is verified by `api/py_msg_wf_zaqar_test.go` and the Zaqar tests it cites from the review ledger.

## Identity headers

Python `MessageResource` sends `Client-ID` (a fresh UUID when none is given) and `X-PROJECT-ID` (the session project) on every Zaqar request. The native Go queues, messages and claims APIs add no identity headers; supply both through the service client configuration (`MoreHeaders`), which the Go tests verify is sent on every call. The subscriptions scope also accepts per-call identity options. Go never invents a random Client-ID per request.

## Delete defaults

Python `delete_queue`, `delete_message`, `delete_claim` and `delete_subscription` default to `ignore_missing=True`. The native Go `Delete` methods stay strict and keep the native 404 response error. To get the Python default, wrap the call: `err = resource.IgnoreMissing(api.Delete(ctx, ...))`. `IgnoreMissing` suppresses only an actual HTTP 404 (not transport, decode, cancellation or joined errors) and nothing is retried. Subscriptions expose the same default through `WithDeleteIgnoreMissing`.

Python `delete_message(claim=...)` appends `claim_id` only for a non-empty claim; use `messages.WithDeleteOptions(messages.DeleteOpts{ClaimID: id})`.

## Claims

Python `create_claim` returns a `Claim` whose ID comes from the `Location` response header (the text after `claims/`), and returns early on 204 when nothing could be claimed. `claims.API.CreateClaim` returns `ClaimedMessages{ID, Location, Messages}` with the same rules, so the ID can feed `Get`, `Update`, `Delete` and `WaitForDeletion`. `Create` remains available and returns only the claimed messages. Python `update_claim` ignores the PATCH response status; Go `Update` requires 204 and reports other statuses as errors. A claim is a lease; local cancellation does not release it remotely and creation is never replayed.

## Waiting

No Zaqar resource has a `status` attribute, so `wait_for_status` has no Go counterpart (Python raises `AttributeError` for the default attribute). `WaitForDeletion` on `queues.API`, `messages.API`, `claims.API` and the subscriptions `QueueScope` polls GET until a native 404, like `wait_for_delete`; other errors, cancellation and the shared wait timeout fail it. The shared Go default timeout is five minutes (Python defaults to 120 seconds for `wait_for_delete`); pass `resource.WithTimeout` for another bound.

## Known differences

- Python `create_queue` sends `name` in the request body; Go sends the name only in the path.
- Python lists use their own marker/limit pagination for queues, messages and subscriptions; Go queues and messages follow the server `links.next` entries and treat 204 as the end. The subscriptions scope implements the Python marker algorithm.
- Go results are typed Gophercloud models; Python returns mutable Resource objects.
