# Barbican metadata lists: pinned Python and Go

`Containers.ListRecords`, `Orders.ListRecords`, and `Secrets.ListRecords` provide
owned Resource views for the three project user lists. They reuse the existing
Container/Order metadata projector and Secret Find projector. This guide compares
openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe` and Gophercloud v2.15.0.

| Python | Go |
| --- | --- |
| `conn.key_manager.containers(**query)` | `service.Containers.ListRecords(ctx, options...)` |
| `conn.key_manager.orders(**query)` | `service.Orders.ListRecords(ctx, options...)` |
| `conn.key_manager.secrets(**query)` | `service.Secrets.ListRecords(ctx, options...)` |
| `list(iterator)` | Each leaf's `AllRecords(ctx, options...)` |
| keyword query or Body condition | Each leaf's `WithRecordListQuery(key, value)` |
| `max_items=n`, `paginated=False` | `WithRecordListMaxItems(n)`, `WithRecordListPaginated(false)` |
| `headers={...}`, `microversion=...` | `WithRecordListHeaders(...)`, `WithRecordListMicroversion(...)` |
| `item.to_dict()` | `json.MarshalIndent(item.Resource, "", "  ")` |
| original response fields | `item.Wire`, `Envelope`, `Header`, `StatusCode` |

The existing typed `List` and `Resources.List/All` retain their native models,
query encoding, whole-page decoding and pagination contracts. `ListRecords`
consumes raw rows without imposing those typed decoders. It is a separate
concrete API and requires no application builders or matching predicates.

## Runnable Python

Install the pinned SDK and configure a `dev` cloud in `clouds.yaml`. Save this
as `list_metadata.py`; run `python list_metadata.py --max-items 20`.
A supplied `--name` is a local Container/Order condition and a server Secret
condition. Orders normally have a name inside `meta`, not a top-level name.

```python
import argparse
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--name")
parser.add_argument("--max-items", type=int, default=0)
parser.add_argument("--first-page", action="store_true")
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
query = {"paginated": not args.first_page}
if args.max_items:
    query["max_items"] = args.max_items
if args.name is not None:
    query["name"] = args.name
for label, iterator in (
    ("containers", conn.key_manager.containers(**query)),
    ("orders", conn.key_manager.orders(**query)),
    ("secrets", conn.key_manager.secrets(**query)),
):
    for item in iterator:
        print(label, item.to_dict())
```

## Runnable Go main

Prepare a module using the [installation guide](../../docs/install.md), save
this as `main.go`, and run `go run . -max-items 20`. The code uses the actual
Connection service and the same named operations as the Python example.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/containers"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/orders"
    "github.com/JSYoo5B/go-openstacksdk/keymanager/v1/secrets"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    name := flag.String("name", "", "optional exact top-level name condition")
    maximum := flag.Int("max-items", 0, "raw row cap; zero is unbounded")
    firstPage := flag.Bool("first-page", false, "consume only the first page")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.KeyManagerV1(ctx)
    if err != nil { log.Fatal(err) }

    containerOptions := []containers.RecordListOption{
        containers.WithRecordListMaxItems(*maximum),
        containers.WithRecordListPaginated(!*firstPage),
    }
    orderOptions := []orders.RecordListOption{
        orders.WithRecordListMaxItems(*maximum),
        orders.WithRecordListPaginated(!*firstPage),
    }
    secretOptions := []secrets.RecordListOption{
        secrets.WithRecordListMaxItems(*maximum),
        secrets.WithRecordListPaginated(!*firstPage),
    }
    if *name != "" {
        containerOptions = append(containerOptions, containers.WithRecordListQuery("name", *name))
        orderOptions = append(orderOptions, orders.WithRecordListQuery("name", *name))
        secretOptions = append(secretOptions, secrets.WithRecordListQuery("name", *name))
    }
    for value, err := range service.Containers.ListRecords(ctx, containerOptions...) {
        if err != nil { log.Fatal(err) }
        encoded, err := json.MarshalIndent(value.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Printf("containers %s\n", encoded)
    }
    for value, err := range service.Orders.ListRecords(ctx, orderOptions...) {
        if err != nil { log.Fatal(err) }
        encoded, err := json.MarshalIndent(value.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Printf("orders %s\n", encoded)
    }
    for value, err := range service.Secrets.ListRecords(ctx, secretOptions...) {
        if err != nil { log.Fatal(err) }
        encoded, err := json.MarshalIndent(value.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Printf("secrets %s\n", encoded)
    }
}
```

## Defaults and complete returned views

Iteration is lazy. `ListRecords` captures the selected service and applies each
option once when consumed. Options own nested JSON, header maps and pointer
values. `Paginated` defaults to true, zero `MaxItems` is unbounded, and nil
`Microversion` keeps the selected client default. All three Source models inherit
`_max_microversion=None`, so an absent default causes no discovery or negotiation.
An explicit microversion applies only to this iteration and does not mutate the
client. Source default `Accept: application/json` overrides a configured client
Accept; an explicit per-operation header can replace it. Auth and protocol
headers remain library-owned and are validated before HTTP.

The same existing model types are used: `FetchedContainer`, `FetchedOrder`, and
`FoundSecretRecord`. Lists leave `RequestID` empty because no member GET occurred;
the Container/Order `Ref()` helper consequently has no executable request ID.
Returned full reference values are passive metadata. Secret lists return
`Payload=nil` and make no payload request, even if `content_types.default` exists.

`Resource` contains all finite Source attributes: 11 Container, 15 Order, and
17 Secret fields including inherited ID/name and computed null location.
Missing descriptors are null. IDs, dates, names and status remain untyped JSON.
Absent literal `id` derives from the full container/order/secret reference;
explicit null/false/zero/nonstring ID wins. The separate `container_id`,
`order_id`, and `secret_id` format only the reference's final path component.
`secret_refs` and `consumers` wrap nonnull nonlists in a one-element list; `meta`
and `content_types` convert nonnull nonobjects to `{}`. Null remains null.
The existing projector uses deterministic canonical-wire priority when both
wire and attribute aliases occur; Python input-order behavior can differ.
Malformed HREF descriptors fail eagerly on consumed rows, including when no
filter selects that field. Existing Go URL parsing guards remain different from
some Python `urlsplit` authority/leading-control edge cases.

`Wire` retains the entire actual row including unknown fields. `Envelope` is
its original whole page, and header/status belong to that physical response.
These channels and `Resource` own independent bytes/maps. `self` remains only
in Wire. Exact row keys `connection`, `microversion`, `_synchronized`, and `cls`
collide with Source `existing`/initializer arguments even when null or false;
the consumed row fails with the actual accepted page receipt. A cap or consumer
break can stop before a trailing collision or invalid descriptor.

## Server queries, local filters and pagination

| List | Declared server query mapping | Examples of local Body conditions |
| --- | --- | --- |
| Containers | inherited `limit`, `marker` | `name`, `type`, `consumers`, `secret_refs`, `container_id` |
| Orders | inherited `limit`, `marker` | top-level `name`, `meta`, `order_id`, `secret_id`, `status` |
| Secrets | `limit`, `marker`, `name`, `mode`, `bits`, `secret_type`, `acl_only`, `created`, `updated`, `expiration`, `sort`, `algorithm→alg` | `bit_length`, `created_at`, `updated_at`, `expires_at`, `content_types`, `status` |

The query mapping's client key wins its wire alias, including null. Null query
values are omitted, bools use `True`/`False`, flat arrays repeat nonnull values,
and object query values iterate their keys as Requests does. Declared Body
attributes absent from the server mapping's client names filter the projected
Resource locally. Unknown kwargs are discarded; Source Container/Order names
and Source offset are not promoted to server queries. Source does not declare
an initial offset even though stock Barbican and existing native `List` support it.

Local objects recursively match subsets of nonempty actual objects. Empty
actual objects do not match object conditions; nonobject actual values return a
nonmatch without Python attribute-access errors. Ordinary bool/number equality
is shared, but `EqualPythonJSON` uses exact decimals. Source JSON binary64 can
compare integer `9007199254740992` equal to `9007199254740993.0`; this Go iterator
compares them differently. The difference is explicit and tested through the
underlying `content_types` local condition. Unknown raw JSON values in
`RecordListOpts.Query` are discarded without validation; selected values must
be complete UTF-8 JSON. `WithRecordListQuery` itself accepts JSON-marshalable Go
values and reports encoding failures. Raw inherited control names are rejected;
use the concrete control/header/microversion options. Free `base_path`, deprecated
JMESPath, conflicting-attrs overload, and mutable/cache/session replacement remain
outside this finite API.

`MaxItems` counts raw consumed rows before local filtering and provides a wire
limit hint when limit is absent or zero. A local mismatch does not refill the
cap. `Paginated(false)` and consumer break stop before continuation. `AllRecords`
discards partial results on terminal error. The shared pager accepts body links,
RFC Link headers, stock advertised forward offsets, the `/v1/<collection>` alias,
and explicit-limit short-page marker fallback. Marker uses the retained literal
ID or full reference, independently of public Resource mutation. JSON scalar
IDs follow Requests spelling; array/object IDs use a single Python-repr marker
rather than iterable Requests query expansion. Dictionary-link compatibility is
an explicit Go extension.

Each request uses one cursor family. Switching marker/offset drops the previous
cursor; an offset cannot fall below an earlier offset even after marker pages.
Origin/path/fixed filters/limit are guarded, negative cursors and cycles fail,
and new cursor families must move forward. These are explicit Go guards beyond
Source's permissive adapter link merging. References never select HTTP routes.

All valid final HTTP 100–399 representations follow the Source success policy.
Empty/malformed accepted JSON, schema/descriptor errors, physical Read/Close
failures and source/cancellation faults retain the actual page as
`resource.ResponseError`. Rejected HTTP 400–599 preserves native
`gophercloud.ErrUnexpectedResponseCode` and its method/URL/body/header/status,
joined with physical causes. A mixed rejection never becomes an empty result
or an observation retry, even if the native retry callback returns nil.
Tests reuse the public Gophercloud testhelper-backed `internal/testcloud`, existing
native transports/response helpers and the existing Secret Find fault body.
Live-cloud acceptance is separate from these focused fixtures.

The three generic Source `update_container/order/secret` declarations are not
closed by these lists. Stock Container mutations use child secret associations;
stock Secret PUT uploads raw payload and rejects JSON Content-Type. Those
operations need their own reviewed contracts. See the official
[Container API](https://docs.openstack.org/barbican/latest/api/reference/containers.html),
[Order API](https://docs.openstack.org/barbican/latest/api/reference/orders.html), and
[Secret API](https://docs.openstack.org/barbican/latest/api/reference/secrets.html)
for current server behavior; those pages do not pin a deployed server version.
