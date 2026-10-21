# Python openstacksdk mapping for Cyborg (accelerator v2)

This page records how the Go accelerator API relates to the pinned Python
`openstack.accelerator.v2` proxy and the `AcceleratorCloudMixin` helpers. It
describes verified behavior and deliberate differences only. Tests live in
`api/accelerator_*_test.go`, `api/cyborg_cloud_*_test.go` and
`api/py_ha_accel_cyborg_test.go`.

## Shared differences

- Python proxies take `**attrs` / `**query` and mutable `Resource` objects. Go
  uses concrete option types, explicit `resource.Ref` identities and returns
  decoded values with raw JSON and HTTP metadata. There is no dirty-state
  tracking, response-state mutation or cache.
- `Resource.list` sends known query parameters, filters declared body attributes
  locally and drops unknown ones. Go `WithQuery` forwards explicit values
  unchanged, rejects caller `limit`/`marker` and follows only server
  continuation links. `paginated`, `base_path`, `max_items` and JMESPath
  arguments are not exposed.
- The `fields` argument of the getters is a Python no-op with a warning; Go has
  no such argument.
- Deletes: Python `ignore_missing=True` is the default; Go ignores a missing
  resource by default and `resource.WithMissingError()` selects the strict
  policy. Go reports success only on 204 and keeps other statuses as native
  response errors after one attempt. The cloud helpers always return `True` in
  Python; Go returns only an error.
- Context cancellation and deadlines apply to every request. A mutation that
  is uncertain is never replayed, and acknowledgement is not completion.

## Operation notes

- Device `enable`/`disable` use `self.id`, which prefers a numeric body id over
  the UUID for Resource inputs. Go always routes with the supplied UUID and
  requires HTTP 200.
- Device profile create wraps one body in an array; Go builds the same array.
  Python unwraps only `arqs[0]` for accelerator requests; Go returns every
  created request. Attribute create is enveloped by the pinned Resource but the
  controller consumes a flat object, which Go sends.
- Accelerator request delete and patch use the collection endpoint with
  `?arqs=` / a UUID-keyed patch document in Go. The pinned Resource builds a
  member URL. Go restricts patches to typed binding fields.
- `patch_deployable` in Python appends the identifier twice
  (`/deployables/{id}/program/{id}`); Go uses `/deployables/{uuid}/program` and
  accepts only the one-image replace document. The deprecated
  `update_deployable` calls itself recursively and cannot succeed.
- Waiters take a fixed `resource.Ref`, poll through the typed getter and use
  `status` (devices) or `state` (accelerator requests). Python can return a
  cached Resource that already has the target status without fetching.
