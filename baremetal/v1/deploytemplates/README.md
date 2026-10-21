# Deploy templates

SDK-owned API for `/deploy_templates` (microversion 1.55). See [Python comparison](../python-ironic-b.md) for the
contracts shared with the other Ironic collections (exact statuses, no replay,
patch-based update, listing, find, microversions).

```go
service, _ := conn.BareMetalV1(ctx)
api := service.DeployTemplatesAPI()
created, err := api.Create(ctx, deploytemplates.CreateOpts{Name: "CUSTOM_X", Steps: steps})
for value, err := range api.List(ctx, deploytemplates.WithListDetail(true)) { _, _ = value, err }
deleted, err := api.Delete(ctx, created.UUID)
```

Operations: Create, Get, List, Update, Patch, Delete, Find.
