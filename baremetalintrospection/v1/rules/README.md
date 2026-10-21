# Introspection rules

SDK-owned API for the ironic-inspector `/rules` collection. Shared Ironic
contracts are described in [the baremetal comparison](../../../baremetal/v1/python-ironic-b.md);
reach it with `Service.RulesAPI()`.

Operations: Create (POST /rules, 201), Get (GET /rules/{uuid}, 200), List
(GET /rules, 200) and Delete (DELETE /rules/{uuid}, 204, missing ignored by
default). The inspector has no update route.

## Python create quirk

Python `IntrospectionRule` sets `create_opts = CreateOpts(requires_id=True)`.
With the pinned `Resource.create` this makes create fail locally when no uuid is
given and otherwise address `POST /rules/{uuid}`, while the inspector documents
`POST /rules` with an optional `uuid` member in the body. Go deliberately follows
the documented route and does not mirror the quirk. This is inferred from the
pinned source and the documented API; no live inspector was exercised.

## Other differences

- List sends exactly one GET: no limit, marker or next link. Python with a limit
  keeps paging with a marker the inspector ignores and ends with an endless
  pagination error; Python also filters Body attributes client side, Go does not.
  `WithListMaxItems` is a client-side cap. Rows are the brief form (uuid,
  description, scope, links).
- `get_introspection_rule` has no name lookup in either implementation.
- No retry on 409/503, exact status codes, typed models with raw members.
