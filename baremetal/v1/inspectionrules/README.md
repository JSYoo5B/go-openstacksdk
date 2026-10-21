# Inspection rules

SDK-owned API for `/inspection_rules` (microversion 1.96). See [Python comparison](../python-ironic-b.md) for the
contracts shared with the other Ironic collections (exact statuses, no replay,
patch-based update, listing, find, microversions).

```go
service, _ := conn.BareMetalV1(ctx)
api := service.InspectionRulesAPI()
created, err := api.Create(ctx, inspectionrules.CreateOpts{Conditions: c, Actions: a})
for value, err := range api.List(ctx, inspectionrules.WithListDetail(true)) { _, _ = value, err }
deleted, err := api.Delete(ctx, created.UUID)
```

Operations: Create, Get, List, Update, Patch, Delete.
