# Instanceactions

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_instanceactionsFragment1` in [example_instanceactions_fragment1_test.go](example_instanceactions_fragment1_test.go), compiled by `go test`.

See `example_instanceactions1_test.go` in [example_instanceactions1_test.go](example_instanceactions1_test.go), compiled by `go test`.

## References

- [Compute](https://docs.openstack.org/api-ref/compute/#list-actions-for-server)
- [Nova API version history](https://docs.openstack.org/nova/latest/reference/api-microversion-history.html)
- [action detail API](https://docs.openstack.org/api-ref/compute/#show-server-action-details)
- [pinned ServerAction](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_action.py)
- [Resources](resources.go)
- [Instance Actions Scope Test](../../../api/instance_actions_scope_test.go)
- [Listing](../../../docs/listing.md)
- [Native Scope List Controls Test](../../../api/native_scope_list_controls_test.go)
- [Request](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/instanceactions/request.go)
