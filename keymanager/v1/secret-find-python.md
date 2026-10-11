# Find a Barbican secret: pinned Python and Go

`conn.KeyManagerV1(ctx).Secrets.FindRecord(ctx, nameOrID, options...)` owns the
named `key_manager.find_secret` lookup. Applications supply a literal string
and optional concrete functions; they need no resource builder or interface.
This guide compares openstacksdk commit
`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe` with Gophercloud v2.15.0.

| Python | Go |
| --- | --- |
| `find_secret(name_or_id)` | `Secrets.FindRecord(ctx, nameOrID)` |
| `ignore_missing=False` | `secrets.WithFindRecordIgnoreMissing(false)` |
| Generic Resource catch policy | SDK default clean HTTP400/403/404 fallback |
| Explicit Go fallback policy | `secrets.WithFindRecordFallback(resource.FindFallbackNotFoundOnly)` or `FindFallbackNever` |
| `secret.to_dict()` | `json.MarshalIndent(result.Resource, "", "  ")` |
| `secret.payload` text/bytes | `result.Payload.Text` / `result.Payload.Body` for a fetched payload |

`FindRecord` is the declared-field mapping. The existing `FindIdentity` API
returns the narrower `FetchedSecret` projection and retains its existing
HTTP200 policy; `Get`, `GetPayload`, `Fetch`, and explicit `Resources.Find`
remain separate callable operations.

## Runnable Python

Install the pinned SDK and configure a `dev` cloud in `clouds.yaml`. Save this
as `find_secret.py` and run `python find_secret.py SECRET_NAME_OR_ID`.
Printing metadata avoids exposing the secret payload by default.

```python
import argparse
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("identity")
parser.add_argument("--cloud", default="dev")
parser.add_argument("--strict", action="store_true")
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
secret = conn.key_manager.find_secret(
    args.identity, ignore_missing=not args.strict)
if secret is None:
    print("secret absent")
else:
    print({"id": secret.id, "name": secret.name, "status": secret.status})
```

## Runnable Go main

Prepare a module using the [installation guide](../../docs/install.md), save
this as `main.go`, and run `go run . -identity SECRET_NAME_OR_ID`.
`-strict` reports logical absence after a successful complete search.
This example uses the actual Connection service, with no dependency builder.

```go
package main

import (
    "context"
    "flag"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
)

func main() {
    identity := flag.String("identity", "", "literal secret name or ID")
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    strict := flag.Bool("strict", false, "report logical absence")
    flag.Parse()
    if *identity == "" { log.Fatal("-identity is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    keyManager, err := conn.KeyManagerV1(ctx)
    if err != nil { log.Fatal(err) }
    secret, err := keyManager.Secrets.FindRecord(ctx, *identity,
        secrets.WithFindRecordIgnoreMissing(!*strict))
    if err != nil {
        // An optional returned receipt has Resource=nil on failure.
        // Inspect errors.As for native/accepted HTTP evidence as needed.
        log.Fatal(err)
    }
    if secret == nil { log.Print("secret absent"); return }
    log.Printf("id=%s name=%s status=%s",
        secret.Resource.Body["id"], secret.Resource.Body["name"],
        secret.Resource.Body["status"])
}
```

## Lookup and payload behavior

A safe, unescaped single-segment identity first reads
`secrets/{original-identity}`. The SDK snapshots the source before any option
callback and fixes metadata and payload routes before HTTP. A returned `id`,
`secret_id`, or `secret_ref` is passive data and never redirects a request.
Names containing spaces, slashes, escapes, or URL syntax use only a literal
`name` query on the collection. Blank, invalid UTF-8, and control-containing
identities fail before HTTP. `FindFallbackNever` rejects names that require
list lookup.

The direct branch starts with only the request ID, matching Source
`Secret.existing(id=name_or_id)`. It fetches a payload only when raw metadata
contains a non-null `content_types.default`. A response's
`payload_content_type` does not select the payload: Python merges response
attributes after its conditional payload request. Missing `content_types`
or a null default skips it; a null/nonobject `content_types`, missing default,
or nonstring default fails. An empty string default produces an explicit
empty Accept header. The exact `text/plain` value decodes UTF-8; media type
parameters and every other value retain bytes.

Only a clean native HTTP400/403/404 from either direct step can use the default
compatible fallback. Under that default policy, a successful list response
is required before logical absence can be ignored. Default
`ignore_missing=true` returns nil for logical absence; explicit false returns
`resource.ErrNotFound`. The explicit Go `FindFallbackNever` policy instead
suppresses a clean direct HTTP404 when ignore_missing is true, without a list
request; explicit false retains that direct404. List HTTP failures always
remain errors. Read/Close/transport/cancellation/source/retry errors attached to a
rejection prevent fallback or missing suppression, including a physical404.

Fallback supplies the original string as the server name hint and compares
exact string ID OR name across every returned row. It accepts a singleton
object as well as an array, retains untyped descriptors, and never performs a
final secret/payload GET. A literal lowercase `id` key takes precedence,
including null/false/zero/nonstring values. When absent, direct metadata uses
the original request ID; list rows use the full `secret_ref`. The separate
`secret_id` formatter exposes only its last HREF path component. Repeated
matches, including the same ID twice, are ambiguous. A late page/row/continuation
failure defeats an earlier candidate. A list row containing `connection`,
`microversion`, or `_synchronized` fails even when its value is null or false:
Source `Resource.list` passes these constructor controls to `existing` or its
initializer, so matching row keys duplicate explicit arguments. The error
retains the actual whole page as `resource.ResponseError` and an optional
`Resource=nil` receipt. Direct metadata accepts these same keys as unknown
Body data; they remain only in Wire and do not change the declared view.

The shared iterator follows advertised body/RFC Link continuations, including
Barbican offsets and the relative `/v1/secrets` alias. It preserves origin,
collection, literal name query, forward offsets, and an introduced limit.
Cycles, changed filters, foreign URLs, and backward offsets fail before the
next HTTP. Empty pages stop; no limit/marker/offset is invented for this proxy.

## Source view and physical evidence

`FoundSecretRecord` owns independent values:

- `Resource` contains declared Body attributes and their null defaults,
  canonical timestamp aliases, the ID seed/alternate rule, `secret_id`, and
  null computed `location`. Untyped false/null/zero/objects/arrays and large
  JSON numbers remain exact. Nonobject list `content_types` projects `{}` as
  Python's dict descriptor does.
- `Wire` contains actual metadata object/row fields, including unknown data.
  It never receives a request ID seed or downloaded payload.
- `Envelope`, `Header`, and `StatusCode` retain actual metadata response or
  whole list page. `RequestID` is set only for the direct read.
- `Payload` independently retains its original bytes, selected Accept, header,
  status, and optional UTF-8 text. Text payloads replace `Resource.payload` as
  a JSON string. Binary payloads become JSON base64 in that view, while exact
  bytes remain in `Payload.Body`; JSON cannot contain Python bytes objects.

A failure, including strict logical absence, may return a partial receipt with
`Resource=nil`. This retains
completed metadata and an observed payload without presenting a found secret.
Accepted parse/UTF-8 failures include `resource.ResponseError`; rejected errors
retain native HTTP body/header/status and every additional physical cause.
All HTTP400..599 rejections receive shared Read/Close/source observation.
Clean provider retries remain available. Mixed physical failures and changed
operation body/Accept cannot replay, even if a retry hook requests another
attempt. Configuration is captured around callbacks, each physical attempt,
Read/Close, and completion; the original provider keeps live token/transport
policy. Configure service fields before concurrent calls.

## Intentional differences and scope

The Go default checks every final HTTP status: 100..399 may be accepted when
surfaced by a custom transport, and 400..599 are rejected. Ordinary net/http
clients process interim1xx themselves. Pinned `Secret.fetch` calls
`session.get(...).json()` and reads payload bytes without `_translate_response`;
`Proxy.request` defaults `raise_exc=False`. A parseable error metadata response
or error payload can therefore appear as data in that Python configuration.
Source `Resource.find` catches raised NotFound/BadRequest/Forbidden exceptions,
which does not prove that its default custom Secret.fetch actually raised
from a corresponding status. Go's clean fallback and strict rejection are
explicit SDK policy; there is no unchecked Source compatibility mode.

Descriptor views are eagerly validated. A malformed HREF formatter can fail
before returning a view, while Python can defer that failure until a descriptor
is accessed. Canonical wire keys win over simultaneous attribute aliases in Go;
Python attribute merge order can depend on input order. Origin/path/filter
continuation guards and safe-route list-only handling are stronger Go policies.
Computed connection location, mutable-resource dirty caches, arbitrary inherited
`Resource.find/list` kwargs, adapter/cache replacement, and Python session
lifecycle are outside this named operation mapping. `find_secret` itself
accepts only string plus ignore_missing. It does not promise whole Secret or
whole Resource/session equivalence.

Both examples use the configured ordinary key-manager client and caller token.
They perform no secret creation, ACL change, consumer association mutation,
admin credential substitution, or automatic permission escalation. Barbican
continues to enforce the deployed secret/payload visibility policy.
