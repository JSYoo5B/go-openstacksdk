# Introspection

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:baremetalintrospection/v1/introspection.StartIntrospection`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetalintrospection/v1/introspection.API.StartIntrospection`
- `github.com/JSYoo5B/go-openstacksdk/baremetalintrospection/v1/introspection.WithStartIntrospectionQuery`

- Omitted, false and true manage_boot and escaped extension query values reach the POST URL.
  Tests: `api/introspection_start_contracts_test.go:TestIntrospectionStartPreservesBootQueryAndEscapedExtensions`.
- Invalid options fail before HTTP; only 202 succeeds and HTTP status, body, headers, URL and cancellation remain observable.
  Tests: `api/introspection_start_contracts_test.go:TestIntrospectionStartRejectsQueryAndCapabilityErrorsBeforeHTTP`, `api/introspection_start_contracts_test.go:TestIntrospectionStartRequires202AndPreservesHTTPCause`, `api/introspection_start_contracts_test.go:TestIntrospectionStartPreservesCanceledContext`.

Python/Go differences:

- The SDK fixes the pinned native function that discards its serialized query; a private helper preserves that query and the original 202/error policy.

### `python:baremetal_introspection/v1/start_introspection`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetalintrospection/v1/introspection.API.StartIntrospection`

- Omitted, false and true manage_boot and escaped extension query values reach the POST URL.
  Tests: `api/introspection_start_contracts_test.go:TestIntrospectionStartPreservesBootQueryAndEscapedExtensions`.
- Invalid options fail before HTTP; only 202 succeeds and HTTP status, body, headers, URL and cancellation remain observable.
  Tests: `api/introspection_start_contracts_test.go:TestIntrospectionStartRejectsQueryAndCapabilityErrorsBeforeHTTP`, `api/introspection_start_contracts_test.go:TestIntrospectionStartRequires202AndPreservesHTTPCause`, `api/introspection_start_contracts_test.go:TestIntrospectionStartPreservesCanceledContext`.

Python/Go differences:

- Python returns an Introspection Resource after accepting the request; the Go API returns an error and exposes Resources for later lookup.

Unresolved boundaries:

- Audit the Python node-object/name input and Introspection create/response state against a consistent Go node-reference operation.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
service, err := conn.BareMetalIntrospectionV1(ctx)
if err != nil { return err }
value, err := service.Introspection.WaitUntilFinished(ctx, resource.ID("node-uuid"),
    resource.WithTimeout(10*time.Minute),
    resource.WithPollInterval(3*time.Second))
if err != nil { return err }
fmt.Println(value.UUID, value.Finished, value.State)
```

```go
manageBoot := false
err = service.Introspection.StartIntrospection(ctx, "node-uuid",
    introspection.StartOpts{ManageBoot: &manageBoot},
    introspection.WithStartIntrospectionQuery("vendor", "a&b"))
if err != nil { return err }
```

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
