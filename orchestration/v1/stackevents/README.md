# Stackevents

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example` in [example_test.go](example_test.go), compiled by `go test`.

## References

- [V1](https://docs.openstack.org/api-ref/orchestration/v1/#show-event-details)
- [Pinned Python Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/orchestration/v1/_proxy.py)
- [V1](https://docs.openstack.org/api-ref/orchestration/v1/#list-stack-events)
- [Stacks](../stacks/README.md)
- [Gophercloud v2.15.0](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/orchestration/v1/stackevents/requests.go)
- [Python StackEvent](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/orchestration/v1/stack_event.py)

## Native API calls

`API.Find` issues `GET /stacks/{name}/events`, follows Heat's redirect to the canonical stack and returns the `events` array of that single response without following any next link. It is covered by `api/heat_native_stackevents_test.go`.

See `ExampleAPI_Find` in [example_test.go](example_test.go), compiled by `go test`.

For resource events, `ResourceEventScope.List` serializes every `ListResourceEventsOpts` field (`Marker`, `Limit`, repeated `ResourceActions`, `ResourceStatuses`, `ResourceNames`, `ResourceTypes`, `SortKey`, `SortDir`) and repeats them on later pages, replacing only `marker` with the last event ID. A 204 response ends the listing.
