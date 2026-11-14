# Stackresources

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

See `Example_inStack` in [example_test.go](example_test.go), compiled by `go test`.

See `Example_forStack` in [example_test.go](example_test.go), compiled by `go test`.

See `Example_forResource` in [example_test.go](example_test.go), compiled by `go test`.

## References

- [stack resolver](../stacks/README.md)
- [..](../README.md)
- [Heat Stackresources Contracts Test](../../../api/heat_stackresources_contracts_test.go)
- [SDK Support Ledger](../../../docs/sdk-support-ledger.md)
- [Heat 2026.1 ResourceController](https://docs.openstack.org/heat/2026.1/_modules/heat/api/openstack/v1/resources.html)
- [Heat resource formatter](https://docs.openstack.org/heat/2026.1/_modules/heat/engine/api.html#format_stack_resource)
- [Heat resource API reference](https://docs.openstack.org/api-ref/orchestration/v1/#stack-resources)
- [Python Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/orchestration/v1/resource.py)
- [Proxy.resources](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/orchestration/v1/_proxy.py#L370-L403)

## Native API calls

The generated `API` methods address Heat paths directly and are covered by `api/heat_native_stackresources_test.go`.

See `ExampleAPI` in [example_test.go](example_test.go), compiled by `go test`.

Limitations: `ListTypes` is one non-paged request that preserves the server order. `MarkUnhealthy` accepts HTTP 200/202/204, sends `mark_unhealthy: false` when unset and omits an empty reason. Extensions cannot replace the typed `mark_unhealthy` field. Unlike the scoped `StackScope.MarkUnhealthy`, the native call performs no GET preflight or owner check.
