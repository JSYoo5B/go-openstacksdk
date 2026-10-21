# Heat software deployments

Gophercloud v2.15.0 has no binding for `/software_deployments`, so this package
is SDK-owned. It corresponds to the pinned openstacksdk
`create_software_deployment`, `get_software_deployment`,
`update_software_deployment`, `delete_software_deployment` and
`software_deployments`.

| Operation | Route | Success |
|---|---|---|
| `Create` | `POST /software_deployments`, unwrapped body | 200, 201 |
| `Get` | `GET /software_deployments/{id}` | 200 |
| `Update` | `PUT /software_deployments/{id}`, unwrapped body | 200 |
| `Delete` | `DELETE /software_deployments/{id}` | 204 |
| `List`/`All` | `GET /software_deployments` | 200 |

- `CreateOpts`/`UpdateOpts` members (`action`, `config_id`, `input_values`,
  `output_values`, `server_id`, `stack_user_project_id`, `status`,
  `status_reason`) are `request.Optional`: omitted, `Present` (empty strings and
  `{}` are sent) or `Null`. `Update` sends only the members that are set, like
  Python's dirty-attribute commit; an update with no member fails before HTTP
  (Python sends nothing). `id`, `creation_time` and `updated_time` are
  protected from `WithCreateField`/`WithUpdateField`.
- `input_values`/`output_values` keep exact JSON (large integers survive).
- `Delete` ignores a missing deployment by default; use
  `resource.WithMissingError()` to report it.
- `List` paging matches softwareconfigs. `WithListServerID` sends Heat's
  documented `server_id` filter; Python's query mapping lacks it.
- Mutations are sent once, never replayed, and success is an acknowledgement
  only; observe `Status` through `Get` when completion matters.

Differences from Python are the same as in
[softwareconfigs](../softwareconfigs/README.md). Evidence is HTTP fixtures only.
