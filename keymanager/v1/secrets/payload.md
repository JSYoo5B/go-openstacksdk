# Secret payload bytes

`Secrets.GetPayload(ctx, secretID, options...)` retrieves only the payload and
returns `[]byte, error`. It reads the complete native response and closes its
body before returning. The bytes preserve binary data, NUL, and invalid UTF-8;
`text/plain` does not cause text decoding.

```go
package example

import (
    "context"

    "github.com/JSYoo5B/gophercloudsdk"
    "github.com/JSYoo5B/gophercloudsdk/keymanager/v1/secrets"
)

func secretPayload(ctx context.Context, conn *gophercloudsdk.Connection, secretID string) ([]byte, error) {
    service, err := conn.KeyManagerV1(ctx)
    if err != nil {
        return nil, err
    }
    return service.Secrets.GetPayload(ctx, secretID,
        secrets.WithGetPayloadOptions(secrets.GetPayloadOpts{
            PayloadContentType: "application/octet-stream",
        }),
        secrets.WithGetPayloadHeader("X-Trace", "payload"))
}
```

With no options, the request sends `Accept: text/plain`.
`WithGetPayloadOptions` selects the native concrete `GetPayloadOpts`; its
nonempty `PayloadContentType` changes `Accept`. An empty value keeps the native
default. `WithGetPayloadHeader` adds an ordinary extra header; it cannot replace
`Accept`, which is a declared typed input. Body, query, and argument extensions
are unsupported.

The native request is `GET /secrets/{secretID}/payload` at the configured service
base and accepts HTTP 200. It performs no metadata lookup, name resolution, or
conditional content-type selection. Configured authentication, source headers,
transport, reauthentication, retries, and redirects retain their native policy.
Raw service `MoreHeaders` and native hooks can override request headers;
`OmitHeaders` retains its native precedence. The SDK does not make `Accept`
immutable across those escape paths.

Successful bytes need no `Close` and can be read again with `bytes.NewReader`.
The native extractor buffers the entire response without an SDK size limit.
A read failure returns nil bytes and the original error, discarding a partial
read while still closing the response. A response-body `Close` error is ignored,
as in the native extractor. Native HTTP, transport, and context errors are
wrapped as `OperationError` for `GetPayload`/`secrets`; native causes remain
inspectable. The SDK does not retry extraction failures or add a metadata
fallback. The returned bytes do not carry response headers or status.

## Choosing a secret workflow

| Go API | Result and requests |
| --- | --- |
| `Secrets.Get` | Native secret metadata model from a metadata GET. |
| `Secrets.GetPayload` | Raw payload bytes from one logical payload GET; native retry policy may make additional physical requests. |
| `Secrets.Fetch` | Owned metadata, then an optional payload selected from metadata or explicit options; separate response evidence and partial metadata on later failure. |
| `Secrets.FindIdentity` | Direct ID/name lookup composition with documented metadata-only list fallback. |
| `Secrets.Resources.List/All` | Metadata collection with query and local filters; it does not fetch payloads. |

[Fetch comparison](README.md) explains automatic content-type selection and
exact UTF-8 text handling. [Identity lookup](finding/README.md) and
[metadata listing](listing/README.md) describe those separate workflows.

## OpenStackSDK comparison

At OpenStackSDK `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, the actual
[`get_secret`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L322-L334)
returns a mutable `Secret` after its custom
[`fetch`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/secret.py#L91-L139)
reads metadata and, when a content type is selected, reads payload.

```python
secret = conn.key_manager.get_secret(secret_id)
payload = secret.payload
```

A preexisting `Secret.payload_content_type` wins over the new metadata's
`content_types.default`. A selected exact `text/plain` produces a UTF-8 `str`;
other selected types produce `bytes`. With no selected content type, no payload
request follows. Response `Content-Type` does not change that selection.
The example function named
[`get_secret_payload`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/doc/source/user/examples/key_manager/get.py#L20-L26)
calls `get_secret`; it is not a separate Proxy operation.

`Secrets.Fetch` provides the corresponding Go composition under its documented
response, identity, and error policies. `GetPayload` preserves the separate
[Gophercloud payload extractor](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/keymanager/v1/secrets/results.go#L107-L129)
contract. This distinction does not claim equivalence with Python's mutable
Resource, descriptor, cache, or session behavior.

## Return-type migration

The former facade returned `*request.Download[[]byte]`. Its `Header` held the
bytes extracted from the response, while its `Body` had already been consumed
and closed by that extraction. The corrected return is those bytes directly.

Use the returned value wherever a caller previously used `download.Header`.
Remove subsequent `Read`, `io.ReadAll`, and `Close` calls on that old Download.
Code declaring the old return type must change to `[]byte`. Existing `Fetch`
results and streaming download APIs in other services retain their contracts.
