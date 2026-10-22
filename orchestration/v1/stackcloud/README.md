# Heat cloud-layer stack workflows

`stackcloud` ports the stack methods of the pinned openstacksdk
`openstack/cloud/_orchestration.py` (`search_stacks`, `create_stack`,
`update_stack`, `delete_stack`) and the event polling helper they share,
`openstack/orchestration/util/event_utils.py` (`get_events`,
`poll_for_events`). The same workflows are available as `Connection` methods
that select the cached orchestration v1 endpoint.

```go
result, err := conn.CreateStack(ctx, stackcloud.CreateRequest{
    Name: "app",
    StackRequest: stackcloud.StackRequest{
        Template:   templateloader.StackInput{Template: templateloader.Source{File: "stack.yaml"}},
        Parameters: map[string]any{"flavor": "m1.small"},
        Wait:       true,
    },
})
if err != nil && result != nil && result.Accepted != nil {
    // Heat accepted the POST; the failure happened while waiting or reading
    // the stack back. The stack may exist and is not rolled back.
}
if result != nil && result.WaitStatus == "CREATE_FAILED" {
    // Like Python, a failed terminal status is reported, not raised.
}

found, err := conn.SearchStacks(ctx, stackcloud.SearchRequest{NameOrID: "app-*"})
deleted, err := conn.DeleteStack(ctx, stackcloud.DeleteRequest{NameOrID: "app", Wait: true})
```

## search_stacks

`SearchStacks` lists every stack (`GET /stacks` with no query, following
`next` links and Heat's marker fallback) and applies `_utils._filter_list`
locally through the shared cloud filter engine:

- `NameOrID` matches `id` or `name` exactly, otherwise as an `fnmatch` glob.
- `Filters` is nil, a JSON object of exact matches whose nested objects are
  matched as subsets, or a JSON string holding a JMESPath expression. A filter
  key that is not an attribute of the view fails with
  `resource.ErrInvalidOption`, like Python's `AttributeError`.
- Filters see the Python `Stack.to_dict()` view built by `StackView`: attribute
  names (`name`, `status`, `status_reason`, `owner_id`, `parent_id`,
  `is_rollback_disabled`, `created_at`, ...), declared conversions (bool,
  list and dict descriptors), `tags` as `[]` when absent, unknown response
  keys dropped and `location` null.
- `Stacks` holds the matching rows (typed summary plus exact JSON);
  `Inventory` holds every listed row; for a JMESPath expression `Value` is
  the arbitrary result and `Stacks` is empty.

## delete_stack

`DeleteStack` looks the stack up once (`GET /stacks/{name_or_id}?resolve_outputs=False`;
404, `DELETE_COMPLETE` and a name/ID mismatch are absence) and returns
`Deleted=false` without sending anything else when it is missing. Otherwise:

1. With `Wait`, it captures the latest event id
   (`GET /stacks/{name_or_id}/events?limit=1&sort_dir=desc`) as the marker.
2. It sends one `DELETE /stacks/{id}`; a 404 is ignored (`NotFound=true`).
3. With `Wait`, it polls `DELETE` events from the marker. An HTTP error while
   polling ends polling silently (kept in `WaitHTTPError`), as Python ignores
   `HttpException`; other failures are returned.
4. A final lookup that shows `DELETE_FAILED` fails with `ErrDeleteFailed` and
   Python's message `Failed to delete stack {name_or_id}: {reason}`.

## create_stack and update_stack

Both build Python's attributes: `tags` (nil sends `null`),
`disable_rollback = !Rollback` (default rollback true), `timeout_mins` as the
floor division of `TimeoutSeconds` (default 3600) by 60, `parameters` (nil
sends `{}`), and `template`, `files` and `environment` only when
`templateloader.ReadEnvAndTemplates` produced them. Loader errors occur
before any request.

- `CreateStack` sends one `POST /stacks` with `stack_name`, polls `CREATE`
  events without a marker when `Wait` is set, and returns
  `get_stack(name)` (`GET /stacks/{name}`).
- `UpdateStack` captures the latest event marker first when `Wait` is set,
  sends one `PUT /stacks/{name_or_id}` (no `stack_name`), polls `UPDATE`
  events from the marker and returns `get_stack(name_or_id)`.
- An accepted response whose body is not JSON is ignored; valid JSON that is
  not an object, or whose `stack` member is not an object, is an error.
- `Stack` is nil when the final lookup finds no stack (Python `None`).

## Event polling

Polling follows `poll_for_events`: `GET /stacks/{name}/events?marker=...&sort_dir=asc`
each round, the marker advancing to the last event's id (or dropped when that
event has none). Only events whose `resource_name` or `physical_resource_id`
equals the stack reference, and whose `stack` link (or physical id) ends in
the physical id, count; polling stops at exactly `ACTION_COMPLETE` or
`ACTION_FAILED`. After two consecutive empty rounds one
`GET /stacks/{name}?resolve_outputs=False` is made; a terminal status there
also stops polling, and a missing stack keeps polling. A failed terminal
status is returned in `WaitStatus`, never as an error. Malformed events that
make the Python helper raise (non-list `events` or `links`, non-object
entries, an unhashable `rel`, a non-string stack reference) are errors.

## Differences from Python

- There is no overall wait timeout in Python; in Go the context bounds the
  wait. `WithPollInterval` replaces the fixed five-second poll period.
- `TimeoutSeconds` is an integer; Python also accepts float seconds and then
  sends a float `timeout_mins`.
- Results are typed records (`SearchResult`, `MutationResult`, `DeleteResult`)
  that keep every phase's evidence instead of a Stack Resource, a list or a
  bool, and an error after an accepted mutation is returned together with
  that evidence. Mutations are never replayed and a canceled wait does not
  roll back remote work.
- The stack reference is path-escaped as one segment and must be a valid
  identifier; Python inserts it into the path unescaped.
- `location` in the filter view is null; Python computes it from the session
  (shared Resource/session boundary SDK-R1/S1 in
  [docs/sdk-support-ledger.md](../../../docs/sdk-support-ledger.md)).
- Python logs and warns (deprecated JMESPath filters, bad glob patterns);
  Go does not.

Evidence is HTTP fixture tests in `api/heat_cloud_stack_workflows_test.go` and
`connection_orchestration_stacks_test.go`; no deployment acceptance is
claimed.
