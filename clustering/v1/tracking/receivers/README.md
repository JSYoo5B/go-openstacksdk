# Receivers

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
receiver = conn.clustering.get_receiver("RECEIVER_ID")
receiver.name = "scale_in"
receiver.action = "CLUSTER_SCALE_IN"
receiver.params = {"team": "platform"}
receiver = conn.clustering.update_receiver(receiver)
```

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
    "github.com/JSYoo5B/go-openstacksdk/request"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func RenameTrackedReceiver(ctx context.Context, conn *sdk.Connection) (*receivers.Receiver, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    tracked, err := service.Receivers.Load(ctx, resource.Name("scale_out"))
    if err != nil { return nil, err }
    if err := tracked.Edit(receivers.UpdateOpts{},
        request.Custom(func(c *request.Config[receivers.UpdateOpts]) error { c.Options.Name = request.Present("scale_in"); return nil }),
        request.Custom(func(c *request.Config[receivers.UpdateOpts]) error { c.Options.Action = request.Present("CLUSTER_SCALE_IN"); return nil }),
        receivers.WithUpdateParams(map[string]any{"team": "platform"})); err != nil {
        return nil, err
    }
    return tracked.Commit(ctx, request.Header("X-Audit-Tag", "rename"))
}
```

```go
package example

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/request"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func UpdateReceiverAt(ctx context.Context, conn *sdk.Connection, collectionPath, receiverID string) (*receivers.Receiver, error) {
    service, err := conn.Clustering(ctx)
    if err != nil { return nil, err }
    scope, err := service.Receivers.AtBasePath(collectionPath)
    if err != nil { return nil, err }
    tracked, err := scope.Load(ctx, resource.ID(receiverID))
    if err != nil { return nil, err }
    if err := tracked.Edit(receivers.UpdateOpts{Action: request.Null[string]()}); err != nil {
        return nil, err
    }
    return tracked.Commit(ctx)
}
```

## References

- [update_receiver](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L1114)
- [Proxy._update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L716)
- [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881)
- [TrackedReceiver](../../receivers/lifecycle.go)
- [UpdateScope](../../receivers/scope.go)
- [Tracked](../../../../internal/senlin/tracked.go)
- [Collection Path](../../../../internal/senlin/collection_path.go)
- [Clustering Receivers Lifecycle Test](../../../../api/clustering_receivers_lifecycle_test.go)
- [Tracked Test](../../../../internal/senlin/tracked_test.go)
- [Receivers](../../receivers/README.md)
