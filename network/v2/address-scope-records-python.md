# Address scope reads: pinned Python and Go

The concrete `conn.NetworkV2(ctx).AddressScopes` API provides the three named
Python operations `get_address_scope`, `find_address_scope`, and
`address_scopes`. Applications use SDK-owned functional options rather than
implementing a resource builder. This comparison pins openstacksdk
`ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe` and Gophercloud v2.15.0.

| Python | Go |
| --- | --- |
| `network.get_address_scope(id, fields=["is_shared", "name"])` | `AddressScopes.GetRecord(ctx, id, addressscopes.WithRecordReadFields("is_shared", "name"))` |
| `network.find_address_scope(name_or_id)` | `AddressScopes.FindRecord(ctx, nameOrID)` |
| `ignore_missing=False` | `addressscopes.WithRecordFindIgnoreMissing(false)` |
| `network.address_scopes(is_shared=False, tenant_id="project")` | `AddressScopes.ListRecords(ctx, addressscopes.WithRecordListQuery("is_shared", false), addressscopes.WithRecordListQuery("tenant_id", "project"))` |
| `paginated=False`, `max_items=10` | `WithRecordListQuery("paginated", false)`, `WithRecordListQuery("max_items", 10)` |
| `scope.to_dict()` | `json.MarshalIndent(scope.Resource, "", "  ")` |
| Received response fields | `scope.Wire`; whole body/header/status in `Envelope`, `Header`, `StatusCode` |

Existing native `Get`, `List`, `Resources` and mutation APIs remain separately
callable with their established native result and HTTP policies. These three
Record APIs add a finite declared-field read view; they do not change those
existing operations or count create/update/delete again.

## Runnable Python

Configure the `dev` cloud in `clouds.yaml` and install the pinned SDK. Save as
`address_scopes.py`; run `python address_scopes.py find NAME_OR_ID`,
`python address_scopes.py get ID`, or `python address_scopes.py list`.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("mode", choices=["get", "find", "list"])
parser.add_argument("identity", nargs="?")
parser.add_argument("--cloud", default="dev")
parser.add_argument("--strict", action="store_true")
args = parser.parse_args()
conn = openstack.connect(cloud=args.cloud)
if args.mode in {"get", "find"} and not args.identity:
    parser.error("identity is required for get/find")
if args.mode == "get":
    rows = [conn.network.get_address_scope(
        args.identity, fields=["id", "name", "is_shared", "ip_version"])]
elif args.mode == "find":
    found = conn.network.find_address_scope(
        args.identity, ignore_missing=not args.strict)
    rows = [] if found is None else [found]
else:
    rows = conn.network.address_scopes(is_shared=False, max_items=10)
for row in rows:
    print(json.dumps(row.to_dict(), indent=2))
```

## Runnable Go main

Use the [installation guide](../../docs/install.md), save as `main.go`, and
run `go run . -mode=find -identity=NAME_OR_ID`,
`go run . -mode=get -identity=ID`, or `go run . -mode=list`.
The example calls the actual Connection binding with no manual client builder.

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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes"
)

func main() {
    mode := flag.String("mode", "find", "get, find or list")
    identity := flag.String("identity", "", "literal ID or exact name")
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    strict := flag.Bool("strict", false, "report logical absence from find")
    flag.Parse()
    if (*mode == "get" || *mode == "find") && *identity == "" {
        log.Fatal("-identity is required for get/find")
    }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    printRecord := func(row *addressscopes.AddressScopeRecord) {
        if row == nil { fmt.Println("address scope absent"); return }
        data, err := json.MarshalIndent(row.Resource, "", "  ")
        if err != nil { log.Fatal(err) }
        fmt.Println(string(data))
    }
    switch *mode {
    case "get":
        row, err := network.AddressScopes.GetRecord(ctx, *identity,
            addressscopes.WithRecordReadFields("id", "name", "is_shared", "ip_version"))
        if err != nil { log.Fatal(err) }
        printRecord(row)
    case "find":
        row, err := network.AddressScopes.FindRecord(ctx, *identity,
            addressscopes.WithRecordFindIgnoreMissing(!*strict))
        if err != nil { log.Fatal(err) }
        printRecord(row)
    case "list":
        for row, err := range network.AddressScopes.ListRecords(ctx,
            addressscopes.WithRecordListQuery("is_shared", false),
            addressscopes.WithRecordListQuery("max_items", 10)) {
            if err != nil { log.Fatal(err) }
            printRecord(row)
        }
    default:
        log.Fatal("-mode must be get, find or list")
    }
}
```

## Resource and request behavior

AddressScope directly inherits `Resource`, with six Body attributes:
`id`, `name`, `project_id`, deprecated `tenant_id`, integer `ip_version`, and
boolean `is_shared` mapped from `shared`. Missing values are JSON null.
`project_id` uses `tenant_id` only when the former is absent; explicit null
prevents fallback. This class has no inherited tag, revision-number or
if-match Header descriptor. Actual HTTP If-Match remains in the receipt;
unknown literal Body attributes retain their values instead of becoming
NetworkResource defaults. `self` is removed only from the declared view.

Boolean casting follows Python truthiness, so the nonempty string `"False"`
is true. Fraction/exponent JSON tokens use binary64 for boolean casting and
paging truthiness: `1e-400` is false, and `1e400` is true. Integer descriptors
use the shared audited Source conversion: digit strings convert, non-digit
strings become zero, decimal floats truncate after binary64 conversion,
null stays null and bool remains bool. Nondecimal Unicode digit strings
such as `"²"` fail. The helper pins Unicode16 digit tables rather than the
host Python runtime and does not adopt its configurable integer-string cap.
Raw JSON, large integers and unknown extensions remain exact in `Wire` and
`Envelope`. Local JSON equality deliberately compares exact decimal values;
Python float rounding and arbitrary Python object semantics are outside that
JSON equality policy.

GET fixes and escapes the caller's literal ID once. Omitted or empty Fields
sends no `fields` query; canonical Body names map to wire names, preserving
order, duplicates and unknown field strings. Fetch initializes the ID and
optional immutable `WithRecordReadSeed(map[string]json.RawMessage)` state.
Response fields overwrite the seed in HTTP member order, including null.
Simultaneous canonical/wire values in an unordered Go seed use canonical
priority; response aliases retain last-value-wins. Invalid constructor
integer values fail before HTTP even if the server could overwrite them.
Malformed JSON/bodyless accepted fetch responses retain the constructor
view and actual receipt with `Wire=nil`, matching Source's ignored JSON
ValueError. A valid nonobject representation remains an error.

Find sends raw JSON kwargs to its literal GET (its `fields` query does not
perform the dedicated getter's canonical field translation). It separates
`all_projects` from GET and discards it for this Neutron list. Only clean
native400/403/404 starts fallback. The fallback adds the name hint only when
`name` is absent; `name=null` disables the hint. It matches passive ID OR name
exactly over the complete list. Null/nonstring returned IDs never become a
successful request-ID match. Default logical absence returns nil; explicit
false returns `resource.ErrNotFound`. Duplicate matches and late list errors
invalidate earlier candidates. Direct GET and explicit List retain HTTP404
errors; list failures are never ignored as logical absence.

List is lazy and reusable. Options run once per iteration after Source
capture and are copied before and after each callback. Each iteration owns
an independent request state. Server query declarations are, in Source
order, `limit`, `marker`, `fields`, `name`, `ip_version`, `project_id`,
`sort_key`, `sort_dir`, and `is_shared→shared`. Canonical query names win
simultaneous wire aliases. Lists preserve repeated field/query values,
omit null values, discard unknown kwargs, and apply declared Body attributes
outside that server mapping (`id`, `tenant_id`) as local filters in option
order. Dictionary filters recurse with Source truthiness and short-circuit
behavior. Arbitrary extension kwargs are not invented local filters.

`max_items` counts raw rows before local filtering, supplies the first-page
limit hint when absent, and stops at the cap without an extra request.
`paginated` uses Python JSON truthiness; omitted means true. Top-level `next`
uses the same binary64 fraction/exponent truthiness: `1e-400` is falsey and
ends pagination without losing the received page or a Find candidate. Its
original JSON bytes remain in `Envelope`; truthy nonstring next values fail
with the actual whole-page receipt. Singleton rows,
body/HTTP links, versioned collection links, marker fallback on nonempty
short pages with a limit, empty-page termination and cycle checks use shared
paging. Markers come from retained wire IDs, independent of mutations to the
returned Resource. Marker IDs must encode one nonempty query value.
Continuation URLs remain within the captured origin and fixed collection
and retain filters/caps; unrestricted Python adapter link behavior is not
asserted. A list constructor rejects response keys `connection`,
`microversion`, `_synchronized` by presence, retaining the actual whole page
as error proof. Fetch response updates can preserve those unknown Body keys
without causing duplicate constructor arguments.

GET retains captured Accept; LIST defaults to `application/json` even if the
selected source configured another value. Explicit operation Headers override
that list default. `WithRecordReadMicroversion` overrides the dedicated GET;
`WithRecordListMicroversion` overrides every standalone LIST page.
`WithRecordFindMicroversion` overrides only the initial Find GET, matching
pinned `Resource.find`: fallback LIST pages use the original selected service
version, including absence. An explicit empty Find override removes the GET
version header while preserving the selected version on fallback LIST.
Matching configured source version headers are replaced only on an operation
copy; the selected client and its header map remain unchanged. Options are not
called again for fallback. Specified and absent protocol version headers
are both protected against retry callback replacement. Authentication tokens
remain live through the shared provider; auth, route and protocol controls
cannot be injected as arbitrary Headers/kwargs.

All three operations accept transport-exposed final100..399 representations,
following pinned `raise_from_response`; ordinary HTTP clients consume interim
1xx themselves. All rejected400..599 responses receive physical Read/Close
and sticky Source checks. Mixed Read/Close/source replacement/cancellation
errors preserve every cause, actual body/header/status and native rejection
proof and cannot become fallback, missing success or a replay even when a
retry callback is willing. Clean provider retries preserve the fixed request;
HTTP/transport/retry failures retain their existing causes. Optional accepted
projection-error records retain actual receipts and are never a success.

Neutron27.0.0 stock `get_address_scope` policy permits project admin,
project reader, or shared-scope access for collection and member routes.
Deployment policy and resource ownership are enforced by Neutron. These are
core user reads; sharing mutations are outside this three-operation change.
This finite Record lane does not implement mutable Python instance/cache/
dirty-state/session replacement, Python object subclasses, arbitrary
`base_path`/JMESPath overrides or automatically computed Connection location.
Location defaults to null; a literal returned/seed extension is retained.
Python ordinary control/computed attributes can shadow colliding unknown
Body keys, while Go's finite JSON view preserves the received literal.
