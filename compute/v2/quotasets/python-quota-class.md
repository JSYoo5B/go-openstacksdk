# Python Quota Class

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/get_quota_class_set`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.API.GetQuotaClassRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaClassGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaClassName`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaClassGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaClassGetMicroversion`

- GET default/custom class names use the selected literal route, preserve declared untyped id and15 integer/default/null fields independently from actual unknown Wire and response proof. Invalid accepted integer conversion retains original evidence.
  Tests: `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassGetDefaultAndCustomNamesKeepLiteralRoutes`, `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassInvalidIntegerRetainsAcceptedResponse`.
- Empty version discovers Nova endpoint and selects ceiling2.56 while explicit2.96 remains uncapped; independent Discovery receipt snapshots and option/header ownership survive reusable calls.
  Tests: `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassDefaultDiscoveryCapsVersionAndOwnsProof`, `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassOptionsOwnAttributesHeadersAndExplicitVersion`.

Python/Go differences:

- Go takes a scalar string class name in a fixed /os-quota-class-sets/{className} route rather than mutable Python Resource input. Only omitted GET name defaults to default; explicit empty names are rejected and other names are sent for server validation. Safe escaped routing, finite UTF-8 JSON and independent declared Resource/Wire/Envelope/header/status snapshots replace arbitrary runtime objects and URI formatter overrides.
- Raw update attributes are not coerced on the wire; only the declared returned Resource view applies Source integer/default/null behavior, including bool remaining bool and non-digit strings becoming0. Unknown Body attributes are dropped from the view while actual Wire retains unknown response JSON. Descriptor projection errors are reported without claiming a successful wire operation or Python runtime object identity.
- New scalar-name update seeds id but never commits dirty id; duplicate attrs id is rejected, and no recognized changes produces no HTTP/Discovery or fabricated receipt. Existing mutable Resource equality/recommit, optional response cache and generic adapter/session/runtime SDK-R1/C1/S1 remain whole-SDK goals. Header/version options are explicit Go conveniences and caller custom transport cache bypass is not claimed.
- Nova explicitly selected client/per-call versions are not class-maximum capped. Only an empty selected version triggers owned endpoint discovery/minimum-maximum selection with ceiling2.56 and independent Discovery receipts; this bounded JSON profile does not emulate arbitrary Python adapters/session caches.

### `python:compute/v2/update_quota_class_set`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.API.UpdateQuotaClassRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaClassUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaClassUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaClassUpdateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaClassUpdateMicroversion`

- PUT sends explicit raw quota_class_set attrs including nullable values, then projects declared integer view on response overlay without sending defaults or dirty id. Empty/unknown-only updates perform neither HTTP nor discovery and duplicate attrs id is rejected.
  Tests: `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassUpdatePUTUsesRawAttributesAndProjectedResponse`, `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassUpdateNoopSkipsHTTPAndDiscoveryAndIDCollisions`.
- Raw update options/header/version and Discovery/response channels remain independently owned; preflight/source/cancel guards prevent invalid requests and physical404 failures never become successful no-op or replay.
  Tests: `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassOptionsOwnAttributesHeadersAndExplicitVersion`, `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassPreflightAndCallbackGuardsPreventRequests`, `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassPhysicalErrorsAnd404NeverReplay`.

Python/Go differences:

- Go takes a scalar string class name in a fixed /os-quota-class-sets/{className} route rather than mutable Python Resource input. Only omitted GET name defaults to default; explicit empty names are rejected and other names are sent for server validation. Safe escaped routing, finite UTF-8 JSON and independent declared Resource/Wire/Envelope/header/status snapshots replace arbitrary runtime objects and URI formatter overrides.
- Raw update attributes are not coerced on the wire; only the declared returned Resource view applies Source integer/default/null behavior, including bool remaining bool and non-digit strings becoming0. Unknown Body attributes are dropped from the view while actual Wire retains unknown response JSON. Descriptor projection errors are reported without claiming a successful wire operation or Python runtime object identity.
- New scalar-name update seeds id but never commits dirty id; duplicate attrs id is rejected, and no recognized changes produces no HTTP/Discovery or fabricated receipt. Existing mutable Resource equality/recommit, optional response cache and generic adapter/session/runtime SDK-R1/C1/S1 remain whole-SDK goals. Header/version options are explicit Go conveniences and caller custom transport cache bypass is not claimed.
- Nova explicitly selected client/per-call versions are not class-maximum capped. Only an empty selected version triggers owned endpoint discovery/minimum-maximum selection with ceiling2.56 and independent Discovery receipts; this bounded JSON profile does not emulate arbitrary Python adapters/session caches.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--action", choices=("get", "update"), required=True)
parser.add_argument("--name", default="default")
parser.add_argument("--attrs", default='{"cores":0}')
parser.add_argument("--microversion", default="2.56")
args = parser.parse_args()
config = {"cloud": args.cloud}
if args.microversion:
    config["compute_api_version"] = args.microversion
with openstack.connect(**config) as conn:
    if args.action == "get":
        row = conn.compute.get_quota_class_set(args.name)
    else:
        row = conn.compute.update_quota_class_set(args.name, **json.loads(args.attrs))
    print(json.dumps(row.to_dict(), indent=2))
```

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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    action := flag.String("action", "", "get or update")
    name := flag.String("name", "default", "quota class name")
    attrs := flag.String("attrs", `{"cores":0}`, "update JSON attributes")
    version := flag.String("microversion", "2.56", "explicit service version, empty uses configured/default policy")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    options := []sdk.ConnectionOption{sdk.WithCloud(*cloud)}
    if *version != "" { options = append(options, sdk.WithMicroversion(sdk.Compute, *version)) }
    conn, err := sdk.Connect(ctx, options...)
    if err != nil { log.Fatal(err) }
    service, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    api := quotasets.New(service.RawClient())
    var row *quotasets.QuotaClassRecord
    switch *action {
    case "get":
        row, err = api.GetQuotaClassRecord(ctx, quotasets.WithQuotaClassName(*name))
    case "update":
        var fields map[string]json.RawMessage
        if err := json.Unmarshal([]byte(*attrs), &fields); err != nil { log.Fatal(err) }
        if fields == nil { log.Fatal("attrs must be a JSON object") }
        row, err = api.UpdateQuotaClassRecord(ctx, *name,
            quotasets.WithQuotaClassUpdateOpts(quotasets.QuotaClassUpdateOpts{Attributes: fields}))
    default:
        log.Fatal("action must be get or update")
    }
    if err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(row.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Install](../../../docs/install.md)
