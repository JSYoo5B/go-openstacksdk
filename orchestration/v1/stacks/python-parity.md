# Heat stack workflows and openstacksdk comparison

This guide complements [README.md](README.md). It describes the SDK-owned
Heat stack operations that are not part of Gophercloud v2.15.0, and records how
the Go APIs compare with the pinned Python openstacksdk `orchestration/v1`
proxy. Evidence is HTTP fixtures only; no live Heat deployment was exercised.

## Stack actions

`StackScope.Check`, `StackScope.Suspend` and `StackScope.Resume` POST to
`/stacks/{name}/{id}/actions` with the fixed canonical identity:

| Method | Request body |
|---|---|
| `Check` | `{"check": ""}` |
| `Suspend` | `{"suspend": null}` |
| `Resume` | `{"resume": null}` |

The bodies match the pinned Python `Stack.check/suspend/resume`. HTTP 200, 201,
202 and 204 are accepted; any other status is returned as the native
`gophercloud.ErrUnexpectedResponseCode`. The result,
`ActionAcknowledgement`, is an acknowledgement only. Observe the outcome with
`StackScope.Wait`. A request is sent once and is never replayed after an
uncertain failure; a canceled context sends nothing.

Python addresses `/stacks/{id}/actions` and, for a non-ID string, the name. Go
always uses the resolved canonical name and ID. Go does not return the raw
response object that Python `_action` returns.

## Stack documents

`StackScope.Export`, `Environment`, `Files` and `Template` GET
`.../export`, `.../environment`, `.../files` and `.../template` for the fixed
identity and return a `resource.RawResource`. The complete JSON object is kept
in `Body` (unknown fields, explicit `null`, `false` and integers such as
`9007199254740993` are preserved), together with the response headers and
status. Only HTTP 200 succeeds and the body must be a JSON object.

Differences from Python:

- Python `get_stack_files` returns `resp.json()` without checking the status, so
  an error body is returned as if it were a file map. Go returns the HTTP error.
- Python wraps environment and template documents in declared Resource fields.
  Go keeps the raw object and does not coerce `parameters`, `resources` or
  other members.
- `stacktemplates.API.Get` still re-encodes the template with two-space
  indentation. `StackScope.Template` returns the original object instead.

## Name-or-ID lookup

`API.FindNameOrID(ctx, nameOrID, options...)` performs one
`GET /stacks/{nameOrID}`; Heat redirects it to the canonical location and the
redirect is followed. It makes no list request and no duplicate scan. The result
is a `StackDetail`: the typed native model plus the complete raw stack object, so
fields the native model does not declare (`deletion_time`, `stack_owner`,
`parent`, `stack_user_project_id`, `files_container`, `environment_files`,
`files`, `template`, `environment` and extension fields) stay available.

- `WithFindResolveOutputs(false)` appends `resolve_outputs=False` once.
- A 404, `DELETE_COMPLETE` or `ADOPT_COMPLETE` is reported as
  `resource.ErrNotFound` (the same two states Python `Stack.fetch` treats as
  missing). `WithFindIgnoreMissing()` returns `nil, nil` instead. `DELETE_FAILED`
  stays visible, and 403 or server errors are never ignored.
- The response must contain a `stack` object with a complete canonical name and
  ID. Python falls back to the whole body when the `stack` key is absent; Go
  rejects it.

`StackResources.Find` with a name continues to use an exact local scan of the
list, which also detects duplicates, and `StackResources.Get` requires an ID.

## Template validation

`stacktemplates.API.ValidateTemplate` sends the request shape of Python
`Template.validate`: `{"template": template-or-null}` plus optional
`environment` and `template_url`, and `?ignore_errors=` when the string is not
blank. The template may be a JSON object or string. A request with neither a
template nor a template URL fails before HTTP. The response is returned as a
`resource.RawResource` because the native typed `ValidatedTemplate` declares
`ParameterGroups` as a map, while Heat templates describe parameter groups as a
list. Only HTTP 200 succeeds and a validation failure keeps its native error.

## Python operations without a Go equivalent

- `read_env_and_templates` and `get_template_contents` read template and
  environment files, fetch URLs and resolve `get_file` and child-template
  references on the client. Go has no such template utility; callers pass
  the final template and `files` to `CreateOpts`/`UpdateOpts`, whose native
  builders fetch template and environment URLs and merge referenced child
  templates into `files`.
- Heat software configurations and deployments have no Gophercloud binding and
  no Go API yet.
- The update-preview endpoint is not exposed. The pinned Python
  `update_stack(preview=True)` is wired incorrectly: the value reaches
  `Stack.commit` as the positional `prepend_key`, which that method ignores, so
  it performs an ordinary PUT.

## Cloud-layer stack workflows

The Python cloud methods `search_stacks`, `create_stack`, `update_stack` and
`delete_stack` (with the event polling of `event_utils.poll_for_events`) are
ported by the `orchestration/v1/stackcloud` package and the matching
`Connection` methods; see [stackcloud/README.md](../stackcloud/README.md).
`StackResources.AllRows` is the list form those workflows use: it pages like
`All` and keeps each row's original JSON object.
