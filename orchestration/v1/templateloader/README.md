# Heat template and environment loader

`templateloader` reimplements the pinned openstacksdk
`orchestration/util/template_utils.py` flow behind `Proxy.get_template_contents`
and `Proxy.read_env_and_templates`. The Python helpers derive from
python-heatclient (Apache License 2.0, Copyright OpenStack Foundation); this Go
code is an independent reimplementation of their observable behavior.

```go
loader, _ := templateloader.New()
attrs, err := loader.ReadEnvAndTemplates(ctx, templateloader.StackInput{
    Template: templateloader.Source{File: "stack.yaml"},
    EnvironmentFiles: []string{"base-env.yaml", "site-env.yaml"},
})
// attrs.Template, attrs.Environment, attrs.Files feed a stack create/update.
```

Behavior kept from Python:

- A bare path becomes an absolute `file://` URL; any value with a URL scheme is
  used as is. Template text starting with `{` is JSON, otherwise YAML (YAML
  timestamps keep their text). It must be a mapping with `heat_template_version`,
  `HeatTemplateFormatVersion` or `AWSTemplateFormatVersion`.
- `get_file` values, and `type` values ending in `.yaml` or `.template`, are
  resolved against the template's directory URL, replaced by the absolute URL
  and loaded into `Files` (children first, shared files read once). A file that
  parses as a template is loaded recursively and stored as JSON.
  UTF-8 text is stored as is; other bytes use base64 `encodebytes` layout.
- Environment files are YAML mappings limited to the six Heat sections, merged
  in order with deep mapping updates. `resource_registry` entries (except
  `base_url`, `hooks`, `restricted_actions`, mappings and values containing
  `::`) are loaded like template files, honoring `base_url` and per-resource
  `base_url` under `resources`.
- `ReadEnvAndTemplates` sets `Environment` only for non-empty environment file
  lists and `Template` only for a template source. `Files` is set only with a
  template source and at least one file; environment files replace template
  files with the same URL. As in Python, registry files from environment files
  are dropped when no template source is given.
- `GetTemplateContents` wraps every error as `Error in processing template
  files: ...`; `ReadEnvAndTemplates` does not. Python's template object has no
  reader in the proxy, so `Source{Object: ...}` fails with "Could not fetch
  template" unless `WithObjectReader` is supplied.

Documented differences: exactly one `Source` field is allowed (Python prefers
file over URL over object); input `Files` is copied, not mutated; only `file`,
`http` and `https` URLs are read (no `ftp`/`data`), with an optional
`WithMaxFileBytes`; HTTP statuses outside 2xx are errors; a template that
references itself fails instead of recursing forever; nested templates are
serialized as compact JSON with sorted keys instead of insertion order;
`resource_registry` entries of unsupported types and non-mapping registries are
errors; `urljoin` is replaced by RFC 3986 resolution. The loader sends no
OpenStack token. Evidence is local file and HTTP fixtures only.
