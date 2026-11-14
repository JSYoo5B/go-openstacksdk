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

See `Example_eventsFragment2` in [example_events_fragment2_test.go](example_events_fragment2_test.go), compiled by `go test`.

See `example_events1_test.go` in [example_events1_test.go](example_events1_test.go), compiled by `go test`.

## References

- [Clustering](https://docs.openstack.org/api-ref/clustering/#events-events)
- [Listing](../listing/README.md)
- [Clustering Typed List Controls Test](../../../api/clustering_typed_list_controls_test.go)
- [Clustering Body Filters Test](../../../api/clustering_body_filters_test.go)
