# Receivers

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
receiver = conn.clustering.create_receiver(
    name="scale_out", type="webhook", cluster_id="CLUSTER_ID",
    action="CLUSTER_SCALE_OUT", params={},
)
message = conn.clustering.create_receiver(name="events", type="message")
receiver = conn.clustering.get_receiver(receiver.id)
```

```go
api := receivers.New(client)
receiver, err := api.Create(ctx, receivers.CreateOpts{Name: "scale_out", Type: "webhook"},
    request.Custom(func(c *request.Config[receivers.CreateOpts]) error { c.Options.ClusterID = request.Present("CLUSTER_ID"); return nil }), request.Custom(func(c *request.Config[receivers.CreateOpts]) error { c.Options.Action = request.Present("CLUSTER_SCALE_OUT"); return nil }),
    receivers.WithCreateParams(map[string]any{}))
if err != nil { return err }
message, err := api.Create(ctx, receivers.CreateOpts{Name: "events", Type: "message"})
if err != nil { return err }
receiver, err = api.Get(ctx, receiver.ID)
if err != nil { return err }
fmt.Println(receiver.ID, receiver.Action, receiver.Channel, message.ID)
```

```python
receiver = conn.clustering.update_receiver(
    "scale_out", name="scale_in", action="CLUSTER_SCALE_IN", params={},
)
```

```go
api := receivers.New(client)
receiver, err := api.Update(ctx, resource.Name("scale_out"), receivers.UpdateOpts{},
    request.Custom(func(c *request.Config[receivers.UpdateOpts]) error { c.Options.Name = request.Present("scale_in"); return nil }), request.Custom(func(c *request.Config[receivers.UpdateOpts]) error { c.Options.Action = request.Present("CLUSTER_SCALE_IN"); return nil }),
    receivers.WithUpdateParams(map[string]any{}))
if err != nil { return err }
fmt.Println(receiver.ID, receiver.Action)
```

```python
receivers = conn.clustering.receivers(type="message", user_id="USER_ID", limit=20)
```

```go
api := receivers.New(client)
for receiver, err := range api.List(ctx,
    receivers.WithListOptions(receivers.ListOpts{Type: "message", Limit: 20}),
    request.Custom(func(c *request.Config[receivers.ListOpts]) error { c.Options.UserID = "USER_ID"; return nil }), request.Custom(func(c *request.Config[receivers.ListOpts]) error { v := false; c.Options.GlobalProject = &v; return nil }),
    receivers.WithListFilter("params", map[string]any{"team": "infra"}),
) {
    if err != nil { return err }
    fmt.Println(receiver.ID, receiver.Params)
}
```

```python
receiver = conn.clustering.find_receiver("scale_out", ignore_missing=True)
conn.clustering.delete_receiver("scale_out", ignore_missing=True)
```

```go
api := receivers.New(client)
receiver, err := api.Find(ctx, resource.Name("scale_out"))
if err != nil { return err }
if receiver != nil { fmt.Println(receiver.ID) }
if err := api.Delete(ctx, resource.Name("scale_out")); err != nil { return err }
```

```go
listingAPI := receivers.New(client)
values, err := listingAPI.All(ctx,
    receivers.WithListOptions(receivers.ListOpts{Limit: 20}),
    request.MaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, request.Paginated(false)) {
    if err != nil { return err }
    fmt.Println(value.ID)
}
```

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func FindReceiverStrict(ctx context.Context, conn *sdk.Connection) (*receivers.Receiver, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    return service.Receivers.FindIdentity(ctx, "scale_out",
        request.IgnoreMissing(false))
}
```

## References

- [Receivers](../tracking/receivers/README.md)
- [Finding](../finding/README.md)
- [Clustering](https://docs.openstack.org/api-ref/clustering/#receivers-receivers)
- [Listing](../listing/README.md)
- [Clustering Typed List Controls Test](../../../api/clustering_typed_list_controls_test.go)
