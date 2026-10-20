# Heat resource types

`resourcetypes.API` wraps Gophercloud v2.15.0 `orchestration/v1/resourcetypes`.

| Operation | Route | Notes |
|---|---|---|
| `List` | `GET /resource_types` | `NameRegex`, `SupportStatus` and `WithDescription` map to `name`, `support_status` and `with_description`; zero values are omitted; accepts both the plain-string and the object response shape |
| `GetSchema` | `GET /resource_types/{type}` | Typed `ResourceSchema` with attributes, properties, constraints and nested support status |
| `GenerateTemplate` | `GET /resource_types/{type}/template` | `template_type` defaults to `hot`; `TemplateTypeCFn` selects `cfn`; returns the template as a map |

All operations accept only the default 200 status. `List` accepts `WithListOptions` and `WithListQuery`; `GenerateTemplate` accepts `WithGenerateTemplateOptions` and `WithGenerateTemplateQuery`. Nil options fail with `resource.ErrInvalidOption` before any request. A 404 stays a `gophercloud.ErrUnexpectedResponseCode`. Results are native typed models without raw-body retention. Python openstacksdk equivalence is tracked separately.
