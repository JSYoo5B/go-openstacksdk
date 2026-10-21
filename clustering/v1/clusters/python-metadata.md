# Python metadata routes (Senlin clusters)

`API.PythonMetadata()` opts into the routes used by the pinned Python
`set_cluster_metadata` and `delete_cluster_metadata`:

| Go | Request |
|---|---|
| `SetMetadata(ctx, id, values)` | `POST /clusters/{id}/metadata` with `{"metadata": values}`; nil sends `{}` |
| `ReplaceMetadata(ctx, id, values)` | `PUT /clusters/{id}/metadata` |
| `DeleteMetadata(ctx, id, nil)` | `PUT /clusters/{id}/metadata` with `{"metadata": {}}` |
| `DeleteMetadata(ctx, id, keys)` | one `DELETE /clusters/{id}/metadata/{key}` per key, in order |

Senlin's router registers none of these routes, so a stock deployment answers
404; that error is kept and never replaced by the GET plus PATCH of
`API.SetMetadata`/`API.DeleteMetadata`. Creating the profile is an opt-in, not a
capability claim. The cluster ID must be a Senlin identifier and is not resolved
by name. Any 2xx is accepted and returned as an opaque receipt; no metadata is
synthesized from the input. Key deletion stops at the first failure and returns
the keys acknowledged so far; a nonnil empty key slice performs no HTTP.
Nothing is replayed.

Tests: `api/r2_small_clustering_test.go`.
