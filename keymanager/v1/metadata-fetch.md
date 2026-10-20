# Metadata Fetch

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:key_manager/v1/get_container`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.API.Fetch`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.WithFetchHeader`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.FetchedContainer.Ref`

- Pinned _proxy.py:114-127 dispatches one Container metadata fetch. The concrete Fetch lane executes one fixed explicit-ID GET containers/{RequestID} with no body/query/list/name lookup under the fixture's provider policy. Foreign response refs are passive, and changing or nil-ing public Resources does not redirect or redefine this owned request; Name is rejected before HTTP.
  Tests: `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchFixedTargetsAndOwnedReceipts`, `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchPreparedOptionsAndSourceBoundaries`.
- The declared Container view preserves untyped name/status/type and raw created/updated values. secret_refs and consumers preserve arbitrary JSON elements, wrap non-null scalar/object values in a singleton list, and distinguish missing/null from empty arrays. Known client aliases are accepted only when the canonical key is absent; canonical null wins. Wire remains the original object with unknown/self/case-variant values and exact large integers.
  Tests: `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchFixedTargetsAndOwnedReceipts`, `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchNullableDescriptorsAndAliases`.
- RequestID and Ref() keep the original executable ID separately from Resource.Body.id and nullable ContainerID. A missing response id preserves the request seed and a present id/null replaces it without rerouting. The shared passive HREF formatter takes the last raw path component, preserves percent spelling and a trailing empty component, excludes query/fragment, and returns null for missing/null; invalid or nonstring refs retain the actual response proof.
  Tests: `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchFixedTargetsAndOwnedReceipts`, `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchNullableDescriptorsAndAliases`, `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchIDSeedsAndReferenceFormatting`.
- Leaf and cached Connection fixtures share the authenticated provider and selected reverse ResourceBase. Tokens rotated after API creation, source headers, operation headers and selected microversion remain effective. Resource/Wire/Envelope/Header are owned snapshots. Option slices and intermediate header maps are cloned and callbacks run once; captured route/provider/version or cancellation changes stop before later options/HTTP, and a native retry route change stops before a second request while retaining the original503 cause.
  Tests: `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchFixedTargetsAndOwnedReceipts`, `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchPreparedOptionsAndSourceBoundaries`.
- Actual403/404 remain native HTTP errors with original body/header and Fetch operation/resource cause under the default fixture policy. Successful203/302 preserve actual status; accepted malformed/empty/non-object/non-UTF8 bodies and HREF formatter errors retain actual response evidence. The shared engine's complete known-object read/Close failure and post-response source/custom-cancellation failure return receipt and possible Wire with Resource=nil, preserve original/custom causes, close the tracked body once, invoke no native retry and perform no SDK composition resend. Pre-cancel and nil/invalid source/ref/options fail before HTTP.
  Tests: `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchHTTPAndStrictAcceptedBodies`, `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchIDSeedsAndReferenceFormatting`, `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchAcceptedFailuresRetainRawEvidence`, `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchPreparedOptionsAndSourceBoundaries`.

Python/Go differences:

- Concrete resource.Ref and the fetched value's Ref() replace Python's ID-string or mutable Resource input. Only an explicit, valid single URL-path segment is executable; Name and full-HREF refs fail before HTTP. RequestID remains the route even when response id or passive refs differ. Arbitrary Resource/subclass/seeded-instance overloads remain separate lifecycle scope rather than a missing declared getter input.
- The owned Resource view projects declared Body attributes and coercions while Wire and Envelope preserve the actual object, including unknown/self/case-variant fields, null/presence and exact large JSON numbers. This independently owned result replaces the mutable Python Resource and its dynamic descriptor/cache/instance state; generated native Get retains its previous alias and return type.
- Canonical wire keys, including present null, deterministically precede client attribute aliases in Go. Pinned Resource._consume_attrs follows input insertion order for conflicting aliases. The reused Go HREF parser is stricter than urllib.urlsplit for some authorities; the formatter still treats refs as passive strings and preserves percent spelling and trailing empty path components.
- Resource.Body.location is null. Pinned Proxy._get_resource at proxy.py:583-596 passes connection=conn to a new ID Resource, and Resource initialization at resource.py:838-839 may seed computed location from conn.current_location even without project_id/availability_zone. This concrete authenticated-ServiceClient getter does not synthesize that Connection location.
- Go accepts final HTTP200..399 object responses and preserves actual status, including the tested203 and302 without Location. Pinned exceptions.raise_from_response allows statuses below400 and generic Resource._translate_response can suppress JSON ValueError. Go deliberately returns an accepted ResponseError plus receipt for empty/malformed/non-object/non-UTF8 responses or formatter failures instead of returning a seed-only Resource.
- Context and WithFetchHeader are concrete Go inputs; no SDK timeout is added, and body/query/argument extensions are unsupported. Captured source guards and fixed requests retain the configured shared provider/live token and native retry/reauthentication policy. SDK composition errors do not introduce name fallback, deletion or resend. Generic session/cache negotiation and inherited lifecycle are tracked separately.
- Completion is restricted to python:key_manager/v1/get_container. Native Container Get/List/CRUD, Proxy list/Find/wait and Resource lifecycle/session declarations are not promoted by this named getter mapping. The six offline Go HTTP groups use existing common fixtures/testhelper/body fault adapters; they are not authenticated OpenStack or Python execution evidence.

### `python:key_manager/v1/get_order`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.API.Fetch`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.WithFetchHeader`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.FetchedOrder.Ref`

- Pinned _proxy.py:220-231 dispatches one Order metadata fetch. The concrete Fetch lane executes one fixed explicit-ID GET orders/{RequestID} with no body/query/list/name lookup under the fixture's provider policy. Foreign response refs are passive, and changing or nil-ing public Resources does not redirect or redefine this owned request; Name is rejected before HTTP.
  Tests: `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchFixedTargetsAndOwnedReceipts`, `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchPreparedOptionsAndSourceBoundaries`.
- The declared Order view preserves inherited name, untyped creator/status/substatus/type and raw created/updated values. meta retains arbitrary nested dictionaries and exact large integers; non-null non-dict values become {}, while missing/null remain null. Known client aliases are accepted only when the canonical key is absent; canonical null wins. Wire remains the original object with unknown/self/case-variant fields.
  Tests: `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchFixedTargetsAndOwnedReceipts`, `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchNullableDescriptorsAndAliases`.
- RequestID and Ref() keep the original executable ID separately from Resource.Body.id and nullable OrderID/SecretID. A missing response id preserves the request seed and a present id/null replaces it without rerouting. Both Order and Secret refs are passive nullable last-path-component accessors. The shared formatter's percent/trailing-component rules and the Order second-formatter failure retain actual response proof.
  Tests: `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchFixedTargetsAndOwnedReceipts`, `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchNullableDescriptorsAndAliases`, `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchIDSeedsAndReferenceFormatting`.
- Leaf and cached Connection fixtures share the authenticated provider and selected reverse ResourceBase. Tokens rotated after API creation, source headers, operation headers and selected microversion remain effective. Resource/Wire/Envelope/Header are owned snapshots. Option slices and intermediate header maps are cloned and callbacks run once; captured route/provider/version or cancellation changes stop before later options/HTTP, and a native retry route change stops before a second request while retaining the original503 cause.
  Tests: `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchFixedTargetsAndOwnedReceipts`, `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchPreparedOptionsAndSourceBoundaries`.
- Actual403/404 remain native HTTP errors with original body/header and Fetch operation/resource cause under the default fixture policy. Successful203/302 preserve actual status; accepted malformed/empty/non-object/non-UTF8 bodies and HREF formatter errors retain actual response evidence. The shared engine's complete known-object read/Close failure and post-response source/custom-cancellation failure return receipt and possible Wire with Resource=nil, preserve original/custom causes, close the tracked body once, invoke no native retry and perform no SDK composition resend. Pre-cancel and nil/invalid source/ref/options fail before HTTP.
  Tests: `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchHTTPAndStrictAcceptedBodies`, `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchIDSeedsAndReferenceFormatting`, `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchAcceptedFailuresRetainRawEvidence`, `api/keymanager_metadata_fetch_test.go:TestKeyManagerOwnedMetadataFetchPreparedOptionsAndSourceBoundaries`.

Python/Go differences:

- Concrete resource.Ref and the fetched value's Ref() replace Python's ID-string or mutable Resource input. Only an explicit, valid single URL-path segment is executable; Name and full-HREF refs fail before HTTP. RequestID remains the route even when response id or passive refs differ. Arbitrary Resource/subclass/seeded-instance overloads remain separate lifecycle scope rather than a missing declared getter input.
- The owned Resource view projects declared Body attributes and coercions while Wire and Envelope preserve the actual object, including unknown/self/case-variant fields, null/presence and exact large JSON numbers. This independently owned result replaces the mutable Python Resource and its dynamic descriptor/cache/instance state; generated native Get retains its previous alias and return type.
- Canonical wire keys, including present null, deterministically precede client attribute aliases in Go. Pinned Resource._consume_attrs follows input insertion order for conflicting aliases. The reused Go HREF parser is stricter than urllib.urlsplit for some authorities; the formatter still treats refs as passive strings and preserves percent spelling and trailing empty path components.
- Resource.Body.location is null. Pinned Proxy._get_resource at proxy.py:583-596 passes connection=conn to a new ID Resource, and Resource initialization at resource.py:838-839 may seed computed location from conn.current_location even without project_id/availability_zone. This concrete authenticated-ServiceClient getter does not synthesize that Connection location.
- Go accepts final HTTP200..399 object responses and preserves actual status, including the tested203 and302 without Location. Pinned exceptions.raise_from_response allows statuses below400 and generic Resource._translate_response can suppress JSON ValueError. Go deliberately returns an accepted ResponseError plus receipt for empty/malformed/non-object/non-UTF8 responses or formatter failures instead of returning a seed-only Resource.
- Context and WithFetchHeader are concrete Go inputs; no SDK timeout is added, and body/query/argument extensions are unsupported. Captured source guards and fixed requests retain the configured shared provider/live token and native retry/reauthentication policy. SDK composition errors do not introduce name fallback, deletion or resend. Generic session/cache negotiation and inherited lifecycle are tracked separately.
- Completion is restricted to python:key_manager/v1/get_order. Native Order Get/List/CRUD, Proxy list/Find/wait and Resource lifecycle/session declarations are not promoted by this named getter mapping. The six offline Go HTTP groups use existing common fixtures/testhelper/body fault adapters; they are not authenticated OpenStack or Python execution evidence.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    kind := flag.String("kind", "container", "container or order")
    id := flag.String("id", "", "explicit resource ID")
    timeout := flag.Duration("timeout", time.Minute, "authentication and metadata query timeout")
    flag.Parse()
    if err := run(*cloud, *kind, *id, *timeout); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(cloud, kind, id string, timeout time.Duration) error {
    if timeout <= 0 || id == "" || (kind != "container" && kind != "order") {
        return fmt.Errorf("positive timeout, explicit id and kind container/order are required")
    }
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil {
        return err
    }
    service, err := conn.KeyManagerV1(ctx)
    if err != nil {
        return err
    }
    var output map[string]any
    switch kind {
    case "container":
        value, fetchErr := service.Containers.Fetch(ctx, resource.ID(id))
        err = fetchErr
        if value != nil {
            output = map[string]any{
                "request_id": value.RequestID,
                "container_id": value.ContainerID,
                "resource": value.Resource,
                "wire": value.Wire,
                "envelope": string(value.Envelope),
                "header": value.Header,
                "status_code": value.StatusCode,
            }
        }
    case "order":
        value, fetchErr := service.Orders.Fetch(ctx, resource.ID(id))
        err = fetchErr
        if value != nil {
            output = map[string]any{
                "request_id": value.RequestID,
                "order_id": value.OrderID,
                "secret_id": value.SecretID,
                "resource": value.Resource,
                "wire": value.Wire,
                "envelope": string(value.Envelope),
                "header": value.Header,
                "status_code": value.StatusCode,
            }
        }
    }
    if output != nil {
        encoder := json.NewEncoder(os.Stdout)
        encoder.SetIndent("", "  ")
        err = errors.Join(err, encoder.Encode(output))
    }
    return err
}
```

## References

- [Container getter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L114)
- [Order getter](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L220)
- [Container fields](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/container.py#L17)
- [Order fields](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/order.py#L17)
- [SDK Support Ledger](../../docs/sdk-support-ledger.md)
