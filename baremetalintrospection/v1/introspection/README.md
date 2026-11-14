# Introspection

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_introspectionWaitUntilFinished` in [example_introspection_fragment1_test.go](example_introspection_fragment1_test.go), compiled by `go test`.

See `Example_introspectionStartIntrospection` in [example_introspection_fragment2_test.go](example_introspection_fragment2_test.go), compiled by `go test`.

## References

- [Introspection.wait](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/baremetal_introspection/v1/introspection.py)
- [Gophercloud StartIntrospection](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/baremetalintrospection/v1/introspection/requests.go)
- [private helper](start.go)
- [Audited Requests](../../../internal/cmd/sdkgen/audited_requests.go)
- [Gophercloud Inventory](../../../api/gophercloud_inventory.json)
- [Resources Generated](resources_generated.go)
- [waiter](wait.go)
- [start helper](start.go)
- [Introspection Contracts Test](../../../api/introspection_contracts_test.go)
- [Introspection Start Contracts Test](../../../api/introspection_start_contracts_test.go)
- [Start Test](start_test.go)
- [Audited Requests Test](../../../internal/cmd/sdkgen/audited_requests_test.go)
- [..](../README.md)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)
