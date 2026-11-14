# Stackevents

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stackevents"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { panic(err) }
    service, err := conn.OrchestrationV1(ctx)
    if err != nil { panic(err) }
    events, err := service.StackEvents.InStack(ctx, resource.Name("app"))
    if err != nil { panic(err) }

    for event, err := range events.List(ctx,
        stackevents.WithListOptions(stackevents.ListOpts{Limit: 100}),
        request.Query("nested_depth", "1")) {
        if err != nil { panic(err) }
        fmt.Println(event.ID, event.ResourceName, event.ResourceStatus)
    }

    resourceEvents, err := events.ForResource("server")
    if err != nil { panic(err) }
    values, err := resourceEvents.All(ctx,
        stackevents.WithListResourceEventsOptions(
            stackevents.ListResourceEventsOpts{Limit: 100}))
    if err != nil { panic(err) }
    if len(values) > 0 {
        detail, err := resourceEvents.Get(ctx, values[0].ID)
        if err != nil { panic(err) }
        fmt.Println(detail.ID, detail.ResourceProperties, detail.Header)
    }
}
```

## References

- [V1](https://docs.openstack.org/api-ref/orchestration/v1/#show-event-details)
- [Pinned Python Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/orchestration/v1/_proxy.py)
- [V1](https://docs.openstack.org/api-ref/orchestration/v1/#list-stack-events)
- [Stacks](../stacks/README.md)
- [Gophercloud v2.15.0](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/orchestration/v1/stackevents/requests.go)
- [Python StackEvent](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/orchestration/v1/stack_event.py)

## Native API calls

`API.Find` issues `GET /stacks/{name}/events`, follows Heat's redirect to the canonical stack and returns the `events` array of that single response without following any next link. It is covered by `api/heat_native_stackevents_test.go`.

```go
events, err := stackevents.New(client).Find(ctx, "app")
```

For resource events, `ResourceEventScope.List` serializes every `ListResourceEventsOpts` field (`Marker`, `Limit`, repeated `ResourceActions`, `ResourceStatuses`, `ResourceNames`, `ResourceTypes`, `SortKey`, `SortDir`) and repeats them on later pages, replacing only `marker` with the last event ID. A 204 response ends the listing.
