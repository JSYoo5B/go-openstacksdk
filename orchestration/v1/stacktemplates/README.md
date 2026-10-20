# Heat stack templates

`stacktemplates.API` wraps Gophercloud v2.15.0 `orchestration/v1/stacktemplates`.

| Operation | Route | Success codes | Notes |
|---|---|---|---|
| `Get` | `GET /stacks/{name}/{id}/template` | 200 | Returns the body re-encoded as two-space-indented JSON bytes; the original formatting is not preserved |
| `Validate` | `POST /validate` | 200 only | Body sends both `template` and `template_url`; the unused one is an empty string; one of them is required |

`Validate` accepts `WithValidateField`; the `template` and `template_url` keys are protected and collisions, nil options or a missing template fail with an error before any request. `ValidatedTemplate` exposes `Description`, `Parameters` and `ParameterGroups` as decoded JSON. A 404 from `Get` stays a `gophercloud.ErrUnexpectedResponseCode`. Cancelling the context before a call sends no request. Python openstacksdk equivalence is tracked separately.
