# Python List Query

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:placement/v1/allocation_candidates`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocationcandidates.API.List`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocationcandidates.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocationcandidates.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/placement/v1/allocationcandidates.WithListQueryValues`
- `github.com/JSYoo5B/go-openstacksdk/request.WithQueryValues`

- At placement 1.34 List sends GET allocation_candidates with repeated required and member_of, single resources, limit and group_policy, and suffixed resources1, required1, member_of1 and in_tree1, and decodes every allocation request with its mappings and the provider summaries.
  Tests: `placement/v1/allocationcandidates/python_parity_test.go:TestPythonAllocationCandidatesQueryAndCandidates`.
- WithListQueryValues sends a key once per value in order, such as member_of1=agg-4&member_of1=agg-5 and same_subtree, replacing the ListOpts serialization of that key, while a later WithListQuery for the same key wins.
  Tests: `placement/v1/allocationcandidates/python_list_query_test.go:TestPythonAllocationCandidatesRepeatedQueryValues`.
- WithListQueryValues with a blank key fails before HTTP as resource.ErrInvalidOption wrapped in a List resource.OperationError, and a 400 response to a repeated query is returned as the Gophercloud status error.
  Tests: `placement/v1/allocationcandidates/python_list_query_test.go:TestPythonAllocationCandidatesRepeatedQueryRejections`.

Python/Go differences:

- Python negotiates min(Resource._max_microversion, server maximum) per call unless the session sets a default; Go sends the service client's Microversion string unchanged, so the caller builds a client with the matching value. Python pins 1.34.
- Python yields one AllocationCandidate per allocation request with provider_summaries reduced to that candidate's providers; Go yields the whole response once and the caller selects ProviderSummaries by the AllocationRequests[i].Allocations keys.
- Python accepts every status below 400 and also decodes the 1.10-1.11 list response shape; Go accepts 200, 204 and 300 through Gophercloud paging and decodes only the 1.12+ shape.
- Python consumes paginated, base_path, jmespath_filters, allow_unknown_params and microversion as call controls rather than query keys; Go has no equivalents and takes the microversion from the client.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
for page, err := range api.List(ctx,
	allocationcandidates.WithListOptions(allocationcandidates.ListOpts{
		ResourceGroups: map[string]allocationcandidates.ResourceGroup{"1": {Resources: "VCPU:1"}},
	}),
	allocationcandidates.WithListQueryValues("member_of1", "agg-4", "agg-5"),
) {
	...
}
```

## References

- [`allocation_candidates(**query)`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/_proxy.py)
- [`AllocationCandidate.list`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/placement/v1/allocation_candidate.py)
