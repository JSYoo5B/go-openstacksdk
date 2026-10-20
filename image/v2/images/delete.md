# Delete

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:image/v2/images.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/image/v2/images.API.RawClient`

- Generated API.Delete retains the exact RawClient pointer, DELETE with nil request body/query options and native ServiceURL(ResourceBase or Endpoint, images, raw id). Native empty IDs, slashes and query delimiters are forwarded without local validation or path escaping. Native default Accept, microversion, service MoreHeaders precedence and live auth are preserved. Accepted202/204 bodies are drained opaquely even for invalid UTF-8/JSON; no owned ImageRecord or ACK is fabricated.
  Tests: `image/v2/images/delete_contracts_test.go:TestNativeImageDeleteRoutesHeadersAndOpaqueAcceptedBodies`.
- Default accepted statuses are exactly202/204. Rejected200/201/203/301/404/409/503 preserve native ErrUnexpectedResponseCode actual/expected/method/URL/raw body/headers and never ignore404. DeleteResult embeds native ErrResult and ExtractErr returns the exact native error; request.Wrap adds inspectable Delete/images OperationError without translating its cause.
  Tests: `image/v2/images/delete_contracts_test.go:TestNativeImageDeleteStrictStatusEvidenceAndErrResult`, `image/delete_contracts_test.go:TestDeleteImageNativeDeleteCompatibility`.
- Caller cancellation and native context values, rejected-response RetryFunc, live token refresh,429 RetryBackoffFunc and terminal hook replacement remain upstream policies. Default OkCodes is not assigned to RequestOpts.OkCodes, which remains nil for native Delete. Native hooks are neither disabled nor replaced by owned guards.
  Tests: `image/v2/images/delete_contracts_test.go:TestNativeImageDeleteContextRetryAndBackoff`.
- Native accepted body read errors and transport causes remain inspectable through OperationError; accepted drain read errors do not invoke RetryFunc. Pinned native disposal ignores accepted Close errors and rejected read/Close errors; native rejected partial body/status/header proof remains. No owned ResponseError/ACK or automatic accepted JSON decode is added.
  Tests: `image/v2/images/delete_contracts_test.go:TestNativeImageDeleteTransportAndBodyErrorsRemainNative`.

Python/Go differences:

- This declaration maps Gophercloud native Delete only; it is not Python delete_image parity or owned DeleteImageRecord metadata proof.
- No store selector, automatic ignore-missing, SDK-produced private state, descriptor/current-location projection, escaped IDs, snapshots, source guards or accepted response receipt ownership is introduced by the native facade.
- Native caller client/hook configuration can change retries, headers, redirect/auth and accepted policies; nil/invalid client safety, deployed authorization and cloud outcomes are not claimed.
- Native read/Close behavior is intentionally documented rather than promoted to owned acknowledgement guarantees.
- Generated request.Wrap adds inspectable Delete/images OperationError context around the exact upstream ExtractErr cause. This is a documented Go wrapper addition, so the native declaration is go_mapping rather than a claim of identical error objects or Python SDK parity.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api, err := conn.ImageV2(ctx)
if err != nil { return err }
```

```go
err := service.API.Images.Delete(ctx, imageID)
if err != nil {
    var native gophercloud.ErrUnexpectedResponseCode
    if errors.As(err, &native) {
        fmt.Printf("HTTP %d, expected=%v, response bytes=%d\n",
            native.Actual, native.Expected, len(native.Body))
    }
    return err
}
```

## References

- [Image Record Delete](../../image-record-delete.md)
- [Install](../../../docs/install.md)
- [upstream Delete](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/images/requests.go#L222-L226)
- [OkCodes](https://github.com/gophercloud/gophercloud/blob/v2.15.0/provider_client.go#L582-L597)
- [native accepted drain](https://github.com/gophercloud/gophercloud/blob/v2.15.0/provider_client.go#L570-L580)
- [Provider Client](https://github.com/gophercloud/gophercloud/blob/v2.15.0/provider_client.go#L458-L535)
- [Delete Contracts Test](delete_contracts_test.go)
