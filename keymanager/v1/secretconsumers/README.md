# Secret consumers

This SDK-owned package implements the three declared Barbican SecretConsumer proxy operations. These associations use a different endpoint and body from the native Container consumers API.

| Python operation | Go API on `SecretScope` | HTTP contract |
| --- | --- | --- |
| `create_secret_consumer(secret, **attrs)` | `Create(ctx, ConsumerOpts, ...CreateOption)` | POST `secrets/{fixed-secret-id}/consumers`, flat body, 200 root Secret object |
| `delete_secret_consumer(secret, ignore_missing=True, **attrs)` | `Delete(ctx, ConsumerOpts, ...DeleteOption)` | DELETE the same collection with a flat body, 200 root Secret object |
| `secret_consumers(secret, **query)` | `List` / `All` | GET200, `consumers` array with optional `total`, `next`, `previous` |

The primary [Secret consumers API reference](https://docs.openstack.org/barbican/latest/api/reference/secret_consumers.html) documents all three success codes as **200**. A pinned Python unit test mocks DELETE204 while checking only `_raw_delete`; it does not establish a server204 contract. Go rejects unexpected201/204 rather than treating them as the documented root response.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/keymanager/v1/secretconsumers"
    "gophercloudsdk/resource"
)

func Associate(ctx context.Context, client *gophercloud.ServiceClient, secretID string) error {
    scope, err := secretconsumers.New(client).InSecret(ctx, resource.ID(secretID))
    if err != nil {
        return err
    }
    response, err := scope.Create(ctx, secretconsumers.ConsumerOpts{
        Service: "image", ResourceType: "image", ResourceID: "image-id",
    })
    if err != nil {
        return err
    }
    _ = response.SecretRef // HTTP evidence; never automatically followed.
    return nil
}
```

`InSecret(resource.ID(...))` fixes a safe single unescaped path segment without HTTP. `InSecret(resource.Name(...))` performs an exact name lookup through the existing [Secrets collection](../secrets/api_generated.go), checks ambiguity and captures its returned identifier once. This explicit Name convenience is a Go extension: the Python consumer methods call `Resource._get_id`, treating a string as an ID without a name search. Go does not accept a mutable Python Secret Resource overload. `SecretID()` and `RawClient()` expose the selected parent and shared configured client; response fields cannot retarget the scope.

`ConsumerOpts` requires nonempty `Service`, `ResourceType`, and `ResourceID`. These are body strings, so service-specific resource type paths and resource IDs are not restricted to UUIDs or URL path segments. `WithCreateOptions`, `WithDeleteOptions`, body-field and header helpers provide owned operation options. Extensions cannot replace core association fields, the secret parent, or known response fields, including case variants. Authentication, routing, content and version headers remain SDK-owned; ordinary configured project headers and the original provider/HTTP client are retained. Custom options are validated before HTTP and the source is rechecked afterward.

The SDK serializes `ConsumerOpts.ResourceType` directly as `resource_type` in both operations. Python create has to route this attribute through `__conflicting_attrs`, and delete bypasses the generic proxy to retain the body. Go callers need neither workaround nor a custom builder.

`SecretResponse` contains **actual HTTP response fields** (`Name`, `Status`, `SecretRef`, `Consumers`, original timestamp strings), complete original `Body`, cloned `Header`, and `StatusCode`. It does not fill missing consumer fields from request inputs or fetch the returned Secret reference. Python `create_secret_consumer` returns a seeded `SecretConsumer`: the server's full Secret root response is merged into that Resource's declared descriptors while requested association fields remain cached. Go makes this representation difference explicit. Python delete returns `None`; Go additionally exposes the accepted Secret response. An ignored missing association returns `(nil, nil)`. Default ignoreMissing suppresses only actual HTTP404; transport errors wrapping404, accepted response errors, cancellation and other statuses remain errors.

All typed model fields use exact canonical wire keys. Case-variant extensions remain in `Body` and cannot overwrite canonical values or cause a typed alias failure. Raw JSON preserves unknown fields, large numbers, null and omission. Typed strings narrow Python's untyped Body descriptors; timestamps are optional strings, not a fabricated format conversion. Consumer rows have no synthesized ID. A returned reference is passive evidence and may be a foreign URL or other string. Nested consumer rows preserve their own raw object; their header/status are not presented as a separate HTTP response.

## Listing

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/keymanager/v1/secretconsumers"
    "gophercloudsdk/resource"
)

func Associations(ctx context.Context, client *gophercloud.ServiceClient, secretID string) ([]*secretconsumers.Consumer, error) {
    scope, err := secretconsumers.New(client).InSecret(ctx, resource.ID(secretID))
    if err != nil {
        return nil, err
    }
    return scope.All(ctx,
        secretconsumers.WithListOptions(secretconsumers.ListOpts{Limit: 10, Offset: 0}),
        secretconsumers.WithListMaxItems(25),
    )
}
```

`Limit` and `Offset` request server paging. Zero omits the field and preserves server defaults of limit10/offset0; Go does not guess or enforce the server's maximum limit100. `WithListQuery` supplies explicit wire extensions, repeated values through an owned custom Config, and explicit empty values. Positive concrete limit/offset plus the same raw key is rejected even if the values agree. The server must validate extension queries; Go does not claim that arbitrary wire fields are supported by Barbican.

`List` is lazy and reusable. `WithListOptions` owns pointer values at creation and each application; the iterator owns the options slice. `MaxItems` is a local raw-row cap and emits no limit hint. `WithListPaginated(false)` reads one page. A consumer break, cap or empty page stops before continuation parsing or further HTTP. `All` discards partial rows on terminal errors.

Only advertised pagination is followed; there is no consumer UUID/ID marker or synthetic marker fallback. An advertised offset must increase on the same service origin and exact fixed-parent collection path. An omitted initial offset means0. The first continuation may introduce the server's positive limit when the caller omitted it; afterward that limit and other query values remain fixed. Changed parents/filters, backward offsets and cycles fail. These are explicit guarded Go policies. Accepted decode/read failures retain original whole-response body/header/status and never trigger a resend.

The pinned Python `SecretConsumer` inherits the generic query mapping (`limit`, `marker`), whereas the official REST API uses `offset`. Python also classifies `service`, `resource_type`, and `resource_id` kwargs as local Body equality filters and discards unknown query keys. Go's concrete offset paging and explicit wire extensions do not implement that automatic classification, alias/descriptor equality, JMESPath, generic per-call base-path/session/version controls, or complete mutable Resource cache/dirty lifecycle. Its advertised-offset lane is not a claim that every inherited Python pagination behavior is equivalent. No `Resources` CRUD/Find/Wait surface is supplied because a consumer has no independently addressable resource ID.

Source pin: openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`, `_proxy.py:544–610`, `secret_consumer.py:22–67`, and `resource.py:1231–1249,1338–1394,2217–2245,2275–2436`. Mock HTTP proofs are in [keymanager_secretconsumers_test.go](../../../api/keymanager_secretconsumers_test.go); no live deployment is required or claimed.
