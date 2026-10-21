# Runbooks

SDK-owned API for `/runbooks` and runbook traits (microversions 1.92 and 1.112). See [Python comparison](../python-ironic-b.md) for the
contracts shared with the other Ironic collections (exact statuses, no replay,
patch-based update, listing, find, microversions).

```go
service, _ := conn.BareMetalV1(ctx)
api := service.RunbooksAPI()
created, err := api.Create(ctx, runbooks.CreateOpts{Name: "CUSTOM_X", Steps: steps})
for value, err := range api.List(ctx, runbooks.WithListDetail(true)) { _, _ = value, err }
deleted, err := api.Delete(ctx, created.UUID)
```

Operations: Create, Get, List, Update, Patch, Delete, AddTrait, RemoveTrait, SetTraits.
