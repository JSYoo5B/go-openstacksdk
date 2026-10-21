# Python-style Body filters and instance updates (Trove)

## Client-side Body filters

The pinned Python `flavors()`, `instances()`, `databases()` and `users()`
generators treat keyword arguments that name a declared Body attribute, and that
are not server query parameters, as client-side filters. The Go equivalents are
`flavors.API.FilteredResources`, `instances.API.FilteredResources`,
`databases.DatabaseScope.FilteredResources` and `users.UserScope.FilteredResources`.
Each returns a `resource.Collection` whose `List`/`All` accept
`resource.WithBodyFilter` and `resource.WithBodyFilters`.

- Filters run on the original response rows after the native pager, so omitted
  and null fields both match a `nil` filter and numbers keep their spelling.
- An object filter matches a subset of a nonempty object; other values,
  including arrays, compare exactly (no boolean/number equivalence as in Python).
- Accepted names are the Python attributes: flavors `id name ram links`;
  instances `id name status flavor volume datastore links region hostname
  created_at updated_at` (wire names `created`/`updated` also work); databases
  `id name character_set collate`; users `id name databases password`. Unknown
  names fail with `resource.ErrInvalidOption` before HTTP.
- Server query extensions are rejected (`resource.ErrUnsupported`); the native
  pager follows advertised links only and sends no `limit`/`marker`.
- `users.UserScope.FilteredResources` lists all hosts, like Python, and ignores
  the scope's `WithHost` narrowing.

## `update_instance`

`instances.API.UpdateAttributes` sends `PUT /instances/{id}` with
`{"instance": {...}}` containing only the supplied `request.Optional`
attributes (Python's declared Body attributes except `id`). Null, empty and
false values are preserved. 200, 202 and 204 are accepted; the result is an
`UpdateReceipt` with status, headers and the undecoded body because Trove
acknowledges asynchronously, so read the instance again to observe state.
A request without attributes is rejected before HTTP. The call is never
replayed. Trove owns the update schema; the SDK does not check which attributes
a deployment accepts. The existing `AttachConfigurationGroup` and
`DetachConfigurationGroup` remain the typed configuration updates.

Tests: `api/r2_small_filters_test.go`.
