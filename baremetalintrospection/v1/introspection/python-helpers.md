# Introspection helpers

These additions follow the pinned openstacksdk baremetal introspection proxy.

- `API.StartAndGet` posts the start request and then reads the status once, like `start_introspection`. Neither request is replayed. If only the read fails the error is `*StartedError`, because the introspection may be running. The node identity is used in the URL as given; the inspector decides whether a name is accepted. The Go start request carries no body; the Python resource carries its identity as a body attribute that the inspector ignores.
- `API.AbortIntrospectionWithOptions` treats HTTP 404 as success by default (`WithAbortIgnoreMissing(false)` makes it an error). Python skips the request when it was given an already finished Resource; Go has no cached Resource, so it always sends the request. Only HTTP 202 is accepted and 409/503 are not retried.
- `API.GetIntrospectionDataRaw(ctx, id, processed)` returns the exact JSON document. The unprocessed route sends inspector API version 1.17 whatever the client version is. `GetIntrospectionData` keeps the typed, processed-only model.
- `API.WaitUntilFinishedIgnoringError` is `wait_for_introspection(ignore_error=True)`: the error state is an ordinary final result once finished. `WaitUntilFinished` keeps failing on a service error, and it also fails when the `error` field is non-empty even if the state is not `error`.
- Introspection rules (`create_introspection_rule`, `get_introspection_rule`, `delete_introspection_rule`, `introspection_rules`) have no Go API: Gophercloud v2.15.0 has none.
- Generic `wait_for_status` and `wait_for_delete` on arbitrary Resources have no introspection counterpart; `Resources.WaitFor` returns `resource.ErrUnsupported`.

Tests: `api/py_cloud_ironic_introspection_test.go`.
