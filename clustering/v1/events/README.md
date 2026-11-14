# Events

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
event = conn.clustering.get_event(event_id)
events = conn.clustering.events(obj_id=node_id, limit=20)
```

```go
api := events.New(client)
event, err := api.Get(ctx, eventID)
if err != nil {
    return err
}
fmt.Println(event.ObjectID, event.GeneratedAt, event.Level)

records, err := api.All(ctx, events.WithListOptions(events.ListOpts{
    Limit:       20,
    ObjectIDs:   []string{nodeID, otherNodeID},
    ObjectTypes: []string{"NODE", "CLUSTER"},
    Sort:        "timestamp:desc",
}))
```

```go
listingAPI := events.New(client)
values, err := listingAPI.All(ctx,
    events.WithListOptions(events.ListOpts{Limit: 20}),
    request.MaxItems(50))
if err != nil { return err }
fmt.Println(len(values))
for value, err := range listingAPI.List(ctx, request.Paginated(false)) {
    if err != nil { return err }
    fmt.Println(value.ID)
}
```

```go
package examples

import (
	"context"
	"encoding/json"
	"fmt"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/clustering/v1/events"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func FilterEvents(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ClusteringV1(ctx)
	if err != nil {
		return err
	}
	for event, err := range service.Events.List(ctx,
		events.WithListFilter("status", "ERROR"),
		events.WithListFilter("meta_data", json.RawMessage(`{"count":9007199254740993}`)),
		request.MaxItems(50), request.Paginated(false)) {
		if err != nil {
			return err
		}
		fmt.Println(event.ID, event.Level, event.Status)
	}
	return nil
}
```

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#events-events)
- [Listing](../listing/README.md)
- [Clustering Typed List Controls Test](../../../api/clustering_typed_list_controls_test.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)
