# Metadata Delete

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:key_manager/v1/delete_container`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Pinned _proxy.py:58-78 maps an ID or Container Resource to one common proxy delete and returns None. Existing Go Remove and the default Resources.Delete binding map this to error-only success for an explicit executable ID. Both lanes directly execute one escaped DELETE containers/{id} with no query/body/list/GET/wait, payload access or SDK cascade; opaque response identity data cannot reroute the request.
  Tests: `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeleteFixedTargetsAndPassiveBodies`.
- Collection default ignores a clean native DELETE404, while WithMissingError preserves ErrNotFound, the requested identity and nested native DELETE cause. Actual403/409 remain errors with original status, method/URL, body/header and the Remove→collection delete→owned Delete cause chain. All three routes use existing leaf/cached Connection fixtures and perform one DELETE under the fixture's provider policy.
  Tests: `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeleteMissingAndNativeFailures`.
- The owned default binding accepts actual final HTTP200..399, matching pinned exceptions.raise_from_response status<400 and Resource.delete's has_body=False. The route table directly checks200/202/204/299/399 with a malformed opaque/binary body: no JSON, Resource/HREF formatter, metadata/payload or model schema is used to establish success and no follow-up GET or wait occurs.
  Tests: `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeleteFixedTargetsAndPassiveBodies`.
- Leaf and cached Connection lanes preserve the authenticated shared Provider, selected reverse ResourceBase, source headers, selected microversion and tokens rotated after API construction. The reused source/operation guards capture selected facts before Remove lookup-policy preparation and keep the fixed DELETE method/URL. Representative shared-engine preflight rejects invalid refs, nil context/API/collection/provider, wrong service/base, nil options and cancellation before HTTP; custom context cause is retained.
  Tests: `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeleteFixedTargetsAndPassiveBodies`, `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeletePreflightAndNativeRetrySources`.
- Representative shared-engine Container/Order lanes prove that accepted202 read/Close failures, including a nested native404 cause, and accepted source/custom-cancellation failures keep actual ResponseError body/header/status and original causes. Default missing never hides these errors or reclassifies them as ErrNotFound; the body is read and its Close is called once, native retry callback is not invoked and no SDK composition resend occurs. Native retry source drift stops at the original503 attempt; a hook-expanded404 policy preserves terminal native404 evidence after two physical attempts instead of becoming missing success.
  Tests: `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeleteAcceptedFailuresStayTerminal`, `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeletePreflightAndNativeRetrySources`.

Python/Go differences:

- Python public None success maps to nil Go error. A default-missing success can mean the object was already absent; no deleted Resource model, physical-delete count or eventual disappearance observation is synthesized.
- Python ID-string or mutable Container Resource input maps to explicit resource.Ref. The owned fetched value's Ref() provides its original executable RequestID rather than its response/derived identity. Safe single-path-segment validation replaces arbitrary dynamic/alternate-HREF Resource input.
- Containers.Remove(Name) retains the existing Go exact-name lookup extension and can make a list request before deletion. It is documented and protected separately from this source-matching explicit-ID row; no full name/Find/List support promotion is implied.
- Owned default collection DELETE accepts final200..399 and passively consumes body bytes, while generated native Delete(ctx,string) retains its existing strict202/204 policy, result aliases and independent review. Native signature/alias compatibility and existing Name extension regression groups are preservation checks, not additional named operations classified as complete.
- Go adds concrete context and owned source/fixed-route guards, retaining configured Provider authentication/reauthentication/native retry. Accepted read/Close/source/context failures return actual evidence and remain terminal even with nested404; successful body is not decoded. Public Resources replacement remains a caller extension, and this owned contract describes the SDK's default collection binding.
- Ordinary route, status and missing assertions run for all three services and both leaf/cached facades. Common accepted/preflight/retry faults reuse representative Container/Order lanes of the same engine; this does not claim every fault combination or separate transport lifecycle was re-executed for Secret.
- Completion is restricted to python:key_manager/v1/delete_container. Supplied mutable Resource state/dirty headers, generic session/cache negotiation and independent native/List/Find/CRUD/wait declarations remain separate goal scope rather than blanket blockers for this error-only named delete. Offline Go HTTP tests and example build do not imply authenticated OpenStack or Python execution.

### `python:key_manager/v1/delete_order`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Pinned _proxy.py:169-185 maps an ID or Order Resource to one common proxy delete and returns None. Existing Go Remove and the default Resources.Delete binding map this to error-only success for an explicit executable ID. Both lanes directly execute one escaped DELETE orders/{id} with no query/body/list/GET/wait, payload access or SDK cascade; opaque response identity data cannot reroute the request.
  Tests: `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeleteFixedTargetsAndPassiveBodies`.
- Collection default ignores a clean native DELETE404, while WithMissingError preserves ErrNotFound, the requested identity and nested native DELETE cause. Actual403/409 remain errors with original status, method/URL, body/header and the Remove→collection delete→owned Delete cause chain. All three routes use existing leaf/cached Connection fixtures and perform one DELETE under the fixture's provider policy.
  Tests: `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeleteMissingAndNativeFailures`.
- The owned default binding accepts actual final HTTP200..399, matching pinned exceptions.raise_from_response status<400 and Resource.delete's has_body=False. The route table directly checks200/202/204/299/399 with a malformed opaque/binary body: no JSON, Resource/HREF formatter, metadata/payload or model schema is used to establish success and no follow-up GET or wait occurs.
  Tests: `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeleteFixedTargetsAndPassiveBodies`.
- Leaf and cached Connection lanes preserve the authenticated shared Provider, selected reverse ResourceBase, source headers, selected microversion and tokens rotated after API construction. The reused source/operation guards capture selected facts before Remove lookup-policy preparation and keep the fixed DELETE method/URL. Representative shared-engine preflight rejects invalid refs, nil context/API/collection/provider, wrong service/base, nil options and cancellation before HTTP; custom context cause is retained.
  Tests: `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeleteFixedTargetsAndPassiveBodies`, `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeletePreflightAndNativeRetrySources`.
- Representative shared-engine Container/Order lanes prove that accepted202 read/Close failures, including a nested native404 cause, and accepted source/custom-cancellation failures keep actual ResponseError body/header/status and original causes. Default missing never hides these errors or reclassifies them as ErrNotFound; the body is read and its Close is called once, native retry callback is not invoked and no SDK composition resend occurs. Native retry source drift stops at the original503 attempt; a hook-expanded404 policy preserves terminal native404 evidence after two physical attempts instead of becoming missing success.
  Tests: `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeleteAcceptedFailuresStayTerminal`, `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeletePreflightAndNativeRetrySources`.

Python/Go differences:

- Python public None success maps to nil Go error. A default-missing success can mean the object was already absent; no deleted Resource model, physical-delete count or eventual disappearance observation is synthesized.
- Python ID-string or mutable Order Resource input maps to explicit resource.Ref. The owned fetched value's Ref() provides its original executable RequestID rather than its response/derived identity. Safe single-path-segment validation replaces arbitrary dynamic/alternate-HREF Resource input.
- Orders.Remove(Name) retains the existing unsupported behavior. This row maps only the named Python explicit-ID delete; it does not promote a name lookup operation.
- Owned default collection DELETE accepts final200..399 and passively consumes body bytes, while generated native Delete(ctx,string) retains its existing strict202/204 policy, result aliases and independent review. Native signature/alias compatibility and existing Name extension regression groups are preservation checks, not additional named operations classified as complete.
- Go adds concrete context and owned source/fixed-route guards, retaining configured Provider authentication/reauthentication/native retry. Accepted read/Close/source/context failures return actual evidence and remain terminal even with nested404; successful body is not decoded. Public Resources replacement remains a caller extension, and this owned contract describes the SDK's default collection binding.
- Ordinary route, status and missing assertions run for all three services and both leaf/cached facades. Common accepted/preflight/retry faults reuse representative Container/Order lanes of the same engine; this does not claim every fault combination or separate transport lifecycle was re-executed for Secret.
- Completion is restricted to python:key_manager/v1/delete_order. Supplied mutable Resource state/dirty headers, generic session/cache negotiation and independent native/List/Find/CRUD/wait declarations remain separate goal scope rather than blanket blockers for this error-only named delete. Offline Go HTTP tests and example build do not imply authenticated OpenStack or Python execution.

### `python:key_manager/v1/delete_secret`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.New`
- `github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Pinned _proxy.py:270-286 maps an ID or Secret Resource to one common proxy delete and returns None. Existing Go Remove and the default Resources.Delete binding map this to error-only success for an explicit executable ID. Both lanes directly execute one escaped DELETE secrets/{id} with no query/body/list/GET/wait, payload access or SDK cascade; opaque response identity data cannot reroute the request.
  Tests: `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeleteFixedTargetsAndPassiveBodies`.
- Collection default ignores a clean native DELETE404, while WithMissingError preserves ErrNotFound, the requested identity and nested native DELETE cause. Actual403/409 remain errors with original status, method/URL, body/header and the Remove→collection delete→owned Delete cause chain. All three routes use existing leaf/cached Connection fixtures and perform one DELETE under the fixture's provider policy.
  Tests: `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeleteMissingAndNativeFailures`.
- The owned default binding accepts actual final HTTP200..399, matching pinned exceptions.raise_from_response status<400 and Resource.delete's has_body=False. The route table directly checks200/202/204/299/399 with a malformed opaque/binary body: no JSON, Resource/HREF formatter, metadata/payload or model schema is used to establish success and no follow-up GET or wait occurs.
  Tests: `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeleteFixedTargetsAndPassiveBodies`.
- Leaf and cached Connection lanes preserve the authenticated shared Provider, selected reverse ResourceBase, source headers, selected microversion and tokens rotated after API construction. The reused source/operation guards capture selected facts before Remove lookup-policy preparation and keep the fixed DELETE method/URL. Representative shared-engine preflight rejects invalid refs, nil context/API/collection/provider, wrong service/base, nil options and cancellation before HTTP; custom context cause is retained.
  Tests: `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeleteFixedTargetsAndPassiveBodies`, `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeletePreflightAndNativeRetrySources`.
- Representative shared-engine Container/Order lanes prove that accepted202 read/Close failures, including a nested native404 cause, and accepted source/custom-cancellation failures keep actual ResponseError body/header/status and original causes. Default missing never hides these errors or reclassifies them as ErrNotFound; the body is read and its Close is called once, native retry callback is not invoked and no SDK composition resend occurs. Native retry source drift stops at the original503 attempt; a hook-expanded404 policy preserves terminal native404 evidence after two physical attempts instead of becoming missing success.
  Tests: `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeleteAcceptedFailuresStayTerminal`, `api/keymanager_metadata_delete_test.go:TestKeyManagerOwnedMetadataDeletePreflightAndNativeRetrySources`.

Python/Go differences:

- Python public None success maps to nil Go error. A default-missing success can mean the object was already absent; no deleted Resource model, physical-delete count or eventual disappearance observation is synthesized.
- Python ID-string or mutable Secret Resource input maps to explicit resource.Ref. Existing owned Secret Fetch exposes its fixed request SecretID for resource.ID(value.SecretID); no new Ref() method or payload retrieval is introduced. Safe single-path-segment validation replaces arbitrary dynamic/alternate-HREF Resource input.
- Secrets.Remove(Name) retains the existing Go exact-name lookup extension and can make a list request before deletion. It is documented and protected separately from this source-matching explicit-ID row; no full name/Find/List support promotion is implied.
- Owned default collection DELETE accepts final200..399 and passively consumes body bytes, while generated native Delete(ctx,string) retains its existing strict202/204 policy, result aliases and independent review. Native signature/alias compatibility and existing Name extension regression groups are preservation checks, not additional named operations classified as complete.
- Go adds concrete context and owned source/fixed-route guards, retaining configured Provider authentication/reauthentication/native retry. Accepted read/Close/source/context failures return actual evidence and remain terminal even with nested404; successful body is not decoded. Public Resources replacement remains a caller extension, and this owned contract describes the SDK's default collection binding.
- Ordinary route, status and missing assertions run for all three services and both leaf/cached facades. Common accepted/preflight/retry faults reuse representative Container/Order lanes of the same engine; this does not claim every fault combination or separate transport lifecycle was re-executed for Secret.
- Completion is restricted to python:key_manager/v1/delete_secret. Supplied mutable Resource state/dirty headers, generic session/cache negotiation and independent native/List/Find/CRUD/wait declarations remain separate goal scope rather than blanket blockers for this error-only named delete. Offline Go HTTP tests and example build do not imply authenticated OpenStack or Python execution.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    kind := flag.String("kind", "container", "container, order or secret")
    id := flag.String("id", "", "explicit resource ID")
    strict := flag.Bool("strict", false, "return an error when the ID is missing")
    timeout := flag.Duration("timeout", time.Minute, "authentication and deletion timeout")
    flag.Parse()
    if err := run(*cloud, *kind, *id, *strict, *timeout); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(cloud, kind, id string, strict bool, timeout time.Duration) error {
    if timeout <= 0 || (kind != "container" && kind != "order" && kind != "secret") {
        return fmt.Errorf("positive timeout and kind container/order/secret are required")
    }
    ref := resource.ID(id)
    if err := ref.Validate(); err != nil {
        return err
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
    var options []resource.LookupOption
    if strict {
        options = append(options, resource.WithMissingError())
    }
    switch kind {
    case "container":
        err = service.Containers.Remove(ctx, ref, options...)
    case "order":
        err = service.Orders.Remove(ctx, ref, options...)
    case "secret":
        err = service.Secrets.Remove(ctx, ref, options...)
    }
    if err != nil {
        return err
    }
    encoder := json.NewEncoder(os.Stdout)
    encoder.SetIndent("", "  ")
    return encoder.Encode(map[string]any{
        "kind": kind,
        "request_id": id,
        "ignore_missing": !strict,
        "completed": true,
    })
}
```

## References

- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L58)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L169)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L270)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L669)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2094)
- [SDK Support Ledger](../../docs/sdk-support-ledger.md)
