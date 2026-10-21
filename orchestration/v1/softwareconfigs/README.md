# Heat software configurations

Gophercloud v2.15.0 has no binding for `/software_configs`, so this package is
SDK-owned. It corresponds to the pinned openstacksdk `create_software_config`,
`get_software_config`, `delete_software_config` and `software_configs`.

```go
service, _ := conn.OrchestrationV1(ctx)
api := service.SoftwareConfigs() // or softwareconfigs.New(client)
config, err := api.Create(ctx, softwareconfigs.CreateOpts{Name: request.Present("boot")},
    softwareconfigs.WithCreateGroup("script"),
    softwareconfigs.WithCreateConfig("#!/bin/sh\necho hi\n"))
got, err := api.Get(ctx, config.ID)
for item, err := range api.List(ctx, softwareconfigs.WithListLimit(50)) { ... }
err = api.Delete(ctx, config.ID)
```

| Operation | Route | Success |
|---|---|---|
| `Create` | `POST /software_configs`, unwrapped body | 200, 201 |
| `Get` | `GET /software_configs/{id}` | 200 |
| `Delete` | `DELETE /software_configs/{id}` | 204 |
| `List`/`All` | `GET /software_configs` | 200 |

- `CreateOpts` members (`name`, `group`, `config`, `inputs`, `outputs`,
  `options`) are `request.Optional`: omitted, `Present` (empty and false-like
  values are sent) or `Null`. `config`, `inputs`, `outputs` and `options` hold
  exact JSON. `WithCreateField` adds extra members; declared members, `id` and
  `creation_time` are protected, and SDK-owned headers cannot be overridden.
  An empty create sends `{}`; Heat decides whether that is valid.
- Responses keep the raw members (`Metadata.Body`), headers and status.
- `Delete` ignores a missing configuration by default (Python
  `ignore_missing=True`); pass `resource.WithMissingError()` to report it.
- `List` reads one page without a limit; with `WithListLimit` it follows the
  marker (last ID) until an empty page, like Python's Resource pagination.
  `limit` must be positive and `marker` non-empty (Python sends them as given).
- Mutations are sent once and never replayed; a canceled context sends nothing.
  Acknowledgement is not completion tracking: Heat configurations are
  immutable and there is no update.

Differences from Python: typed options instead of `**attrs` (unknown attributes
are not silently dropped; extensions are explicit), no Resource objects, no
per-call `base_path`/microversion, only the documented success codes, and IDs
must be single path segments. Evidence is HTTP fixtures only.
