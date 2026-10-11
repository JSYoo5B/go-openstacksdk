# Secret consumer associations: pinned Python and Go

The three named proxy operations below are available through
`conn.KeyManagerV1(ctx).SecretConsumers`. The SDK owns attribute selection,
request bodies, defaults, descriptor views, local filtering, and pagination;
applications need no custom builder or resource implementation.

| Pinned openstacksdk operation | Go mapping |
| --- | --- |
| `create_secret_consumer(secret_id, **attrs)` | `CreateRecord(ctx, secretID, WithCreateRecordAttributes(attrs))` |
| `delete_secret_consumer(secret_id, ignore_missing=True, **attrs)` | `DeleteRecord(ctx, secretID, WithDeleteRecordAttributes(attrs))` |
| `secret_consumers(secret_id, **query)` | `ListRecords` / `AllRecords(ctx, secretID, WithListRecordQuery(query))` |

`InSecretRecord(ctx, secretID)` fixes a literal secret ID once and returns the
existing `SecretScope`, whose `CreateRecord`, `DeleteRecord`, `ListRecords`, and
`AllRecords` omit the repeated parent argument. It performs no name lookup.
The original concrete `InSecret`/`Create`/`Delete`/`List` APIs still provide
nonempty string inputs and the documented HTTP200/offset lane.

## Runnable Python

Install the pinned openstacksdk in an environment with a `dev` entry in
`clouds.yaml`. Save this as `consumer.py`, then run
`python consumer.py list --secret SECRET_ID`. The create/delete commands change
the association identified by the three supplied fields.

```python
import argparse
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("action", choices=["create", "delete", "list"])
parser.add_argument("--cloud", default="dev")
parser.add_argument("--secret", required=True)
parser.add_argument("--service", default="image")
parser.add_argument("--resource-type", default="image")
parser.add_argument("--resource-id", default="example-image-id")
parser.add_argument("--strict", action="store_true")
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
attrs = dict(service=args.service, resource_type=args.resource_type,
             resource_id=args.resource_id)
if args.action == "create":
    consumer = conn.key_manager.create_secret_consumer(args.secret, **attrs)
    print(consumer.to_dict())
elif args.action == "delete":
    conn.key_manager.delete_secret_consumer(
        args.secret, ignore_missing=not args.strict, **attrs)
else:
    for consumer in conn.key_manager.secret_consumers(
            args.secret, service=args.service):
        print(consumer.to_dict())
```

## Runnable Go main

Prepare a module using the [installation guide](../../docs/install.md), save
this as `main.go`, and run `go run . -action list -secret SECRET_ID`. The
create/delete flags mirror the Python example. `-strict` disables the default
missing-association suppression.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secretconsumers"
)

func main() {
    action := flag.String("action", "list", "create, delete, or list")
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    secret := flag.String("secret", "", "literal secret ID")
    service := flag.String("service", "image", "consumer service")
    kind := flag.String("resource-type", "image", "consumer resource type")
    id := flag.String("resource-id", "example-image-id", "consumer resource ID")
    strict := flag.Bool("strict", false, "report missing deletion")
    flag.Parse()
    if *secret == "" { log.Fatal("-secret is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    keyManager, err := conn.KeyManagerV1(ctx)
    if err != nil { log.Fatal(err) }
    attrs := map[string]any{
        "service": *service, "resource_type": *kind, "resource_id": *id,
    }
    switch *action {
    case "create":
        result, err := keyManager.SecretConsumers.CreateRecord(ctx, *secret,
            secretconsumers.WithCreateRecordAttributes(attrs))
        if err != nil { log.Fatal(err) }
        view, err := json.MarshalIndent(result.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        log.Printf("POST status=%d resource=%s actual_response=%s",
            result.StatusCode, view, result.Envelope)
    case "delete":
        result, err := keyManager.SecretConsumers.DeleteRecord(ctx, *secret,
            secretconsumers.WithDeleteRecordAttributes(attrs),
            secretconsumers.WithDeleteRecordIgnoreMissing(!*strict))
        if err != nil { log.Fatal(err) }
        if result == nil { log.Print("association absent"); return }
        log.Printf("DELETE status=%d actual_response=%s",
            result.StatusCode, result.Envelope)
    case "list":
        for consumer, err := range keyManager.SecretConsumers.ListRecords(ctx,
                *secret, secretconsumers.WithListRecordQuery(
                    map[string]any{"service": *service})) {
            if err != nil { log.Fatal(err) }
            view, err := json.MarshalIndent(consumer.Resource, "", "  ")
            if err != nil { log.Fatal(err) }
            log.Printf("secret=%s resource=%s", consumer.SecretID, view)
        }
    default:
        log.Fatal("-action must be create, delete, or list")
    }
}
```

## Attributes, results, and defaults

Only the exact declared Body names `service`, `resource_type`, `resource_id`,
`id`, and `name` enter Source-shaped mutation bodies. Unknown semantic
attributes, including status, timestamps, secret references, or differently
cased names, are discarded before serialization. The SDK accepts omitted,
null, false, zero, empty, object, and array values as the pinned untyped
`Body` descriptors do. The server validates association schema requirements.
The body is flat JSON for both POST and DELETE; no consumer member ID is
appended. The concrete `resource_type` option needs no Python
`__conflicting_attrs` workaround.

`WithCreateRecordAttribute` / `WithDeleteRecordAttribute` select individual
values; attributes/options helpers snapshot selected JSON when constructed.
Each callback receives owned maps and its result is copied and validated before
another callback or HTTP can use it. Raw Config body/query/argument carriers
are rejected; descriptor inputs belong in the Attributes or Query map.
Ordinary per-call headers have explicit helpers, while authentication,
content/version headers and list Accept remain library-owned.

Create returns a `ConsumerRecord`:

- `Resource` is the Source descriptor view: request values remain cached until
  a declared response field replaces them; omitted descriptors and computed
  location are null. It contains exactly five Body fields plus location.
- `Wire` retains actual response object fields, including unknown data and large
  numbers; it never receives request fields. Invalid/empty response JSON leaves
  Wire nil and preserves the request-seeded Resource, matching the Source's
  JSON ValueError handling. Valid nonobject JSON fails with its accepted receipt.
- `Envelope`, `Header`, and `StatusCode` retain the physical whole response.
  `SecretID` retains the fixed URI parent, regardless of response attributes.

Delete returns the actual receipt as `DeletedConsumer`; its body remains bytes
and is never parsed. Python returns None and calls `_translate_response` with
`has_body=False`, so HTTP204, malformed JSON, and otherwise accepted nonobject
bodies are successful. By default a clean native HTTP404 returns nil receipt
and nil error. `WithDeleteRecordIgnoreMissing(false)` preserves that status.
Transport, Read, Close, retry, source-change, context cancellation and custom
cancellation causes cannot be suppressed as missing.

Both Source mutation methods and list status handling accept final HTTP100–399,
matching the pinned `raise_from_response` boundary. The HTTP client normally
handles interim 1xx responses; a transport exposing a final 1xx retains that
Source boundary. All HTTP400–599 rejection bodies receive the same Read/Close,
source and cancellation observation before native retry policy may continue.
The primary
[Barbican consumer reference](https://docs.openstack.org/barbican/latest/api/reference/secret_consumers.html)
documents normal success as HTTP200 with a full Secret root for POST/DELETE;
accepting the wider Source boundary does not assert extra server endpoints.

## Listing and pagination

`WithListRecordQuery` sends only inherited `limit` and `marker` as wire query
parameters. All five Body descriptors become local filters; unknown names such
as status and offset are discarded. Null matches absent descriptor values.
Dictionary filters recursively match subsets of truthy actual dictionaries;
other JSON values compare recursively using Python boolean/number equality
(True equals 1, False equals 0). Numbers stay exact in the Go JSON domain.
The Resource projection does not promote unknown wire values to descriptors.

`WithListRecordMaxItems` counts raw rows before local filtering and supplies
an initial wire limit when the requested limit is absent or falsey.
`WithListRecordPaginated(false)` reads one page. Iterators are lazy and reusable;
a break, cap or empty page prevents additional HTTP. AllRecords returns nil
rows on a terminal failure. Invalid rows after a cap are not decoded.

The SDK follows Source link precedence: body `links` or `consumers_links`, then
truthy top-level next, then HTTP Link. It admits the fixed collection's `/v1`
alias behind service prefixes and supports advertised forward offset pages
without falsely making offset a Source keyword query. With a truthy initial
limit and no advertised continuation, it uses the last raw row's ID as marker,
including rows rejected by local filters. A missing marker or repeated link
fails before HTTP replay. Each continuation stays on the captured origin and
parent path; unknown filters, backwards offsets and established-limit changes
remain errors with the original page receipt. Neither secret_ref nor a row's
secret_id can retarget requests.

## User scope and parity boundary

These are project operations. Current Barbican
[policy documentation](https://docs.openstack.org/barbican/latest/configuration/policy.html)
permits each route through project secret admin, qualifying project member
ownership/nonprivate access, or secret read ACL paths under the new defaults.
Deployment policy decides the actual authorization; the SDK preserves native
401/403 errors and never substitutes administrative credentials. The policy
page also documents Barbican's runtime opt-in for the new defaults, so the Go
API does not infer permissions from roles or alter ACLs.

This mapping covers the three named operations and their practical Source
contracts. It does not claim a complete mutable Python Resource/session system:
Python Secret instance overloads, arbitrary per-call base_path/session swaps,
deprecated JMESPath or __conflicting_attrs controls, automatic connection
location construction, and dirty-cache reuse across later Resource methods
remain outside this fixed-scope API. Go provides raw literal IDs escaped once,
rejects empty/control/dot identities, retains owned raw wire proof, stops at a
raw cap without an extra page, and constrains continuations more strictly.
Invalid UTF-8 and valid nonobject Resource representations remain errors.
JSON equality uses exact numbers rather than Python float rounding.

Source pin: openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`,
`key_manager/v1/_proxy.py:544–610`, `secret_consumer.py:22–67`, and
`resource.py` Resource.new/create/delete/list and QueryParameters. Meaningful
fixture proofs are in [python_record_contracts_test.go](secretconsumers/python_record_contracts_test.go),
using the shared `internal/testcloud` fixture backed by public Gophercloud
`testhelper.SetupHTTP`. No live-cloud authorization or universal service
compatibility is claimed.
