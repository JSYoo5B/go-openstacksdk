# Cloud Router Update Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/update_router`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateRouterRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterUpdateOpts`

- SDK-owned six cloud formals distinguish truthy name/gateway network/fixed IPs from nonnull admin state/SNAT/routes, reuse library gateway assembly without creation admin defaults or merging existing gateway state, skip all HTTP when no effective args exist, and commit only raw dirty fields after member/name lookup. No-op and native mutation result remain distinct from lookup evidence.
  Tests: `network/cloud_router_update_test.go:TestCloudRouterUpdateEffectiveValuesAndDirtySelection`, `network/cloud_router_update_test.go:TestCloudRouterUpdateIneffectiveBeforeServiceLookup`, `network/cloud_router_update_test.go:TestCloudRouterUpdateNameFallbackAndMissing`.
- Nonnull routes including [] and false perform fresh complete extraroute discovery before lookup; missing extension drops routes only and can return lookup-free no-op, while other attrs continue. HTTP/late malformed alias/continuation failures never become unsupported-route fallback. Owned options/preflight and canceled-context guards prevent unwanted HTTP; native conflict preserves lookup and no success model.
  Tests: `network/cloud_router_update_test.go:TestCloudRouterUpdateRoutesGateAndDrop`, `network/cloud_router_update_test.go:TestCloudRouterUpdateExtensionFailureCannotBecomeDroppedRoutes`, `network/cloud_router_update_test.go:TestCloudRouterUpdateOwnedOptionsAndPreflight`, `network/cloud_router_update_test.go:TestCloudRouterUpdateOptionCancellationCannotBecomeNoOp`, `network/cloud_router_update_test.go:TestCloudRouterUpdateConflictKeepsLookupAndSingleAttempt`.

Python/Go differences:

- UpdateRouterRecord owns formal defaults and optional extraroute handling. RoutesDropped and Extensions expose omission/discovery without adding Source logger behavior. Lookup is an owned record while Router is native mutation output; neither is the same mutable Python Resource. Client capture follows effective-option selection and retains configured native provider hooks.

Unresolved boundaries:

- Native PUT result projection omits unknown response fields and does not merge partial/empty/malformed success bodies into the existing Python Resource; no original mutation receipt is exposed.
- Member GET/LIST canonical-only/canonical-wire collision mapping, arbitrary Python kwargs/object/numeric types, full Requests encoding/nonfinite JSON and SDK-R1/C1/S1 Resource/adapter/session/cache lifecycle remain separate.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--identity", required=True)
parser.add_argument("--attrs", default="{}")
args = parser.parse_args()
attrs = json.loads(args.attrs)
if not isinstance(attrs, dict):
    parser.error("attrs must be a JSON object")
with openstack.connect(cloud=args.cloud) as conn:
    result = conn.update_router(args.identity, **attrs)
    print(json.dumps(None if result is None else result.to_dict(), indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/network"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    identity := flag.String("identity", "", "router ID or exact name")
    attrs := flag.String("attrs", "{}", "cloud update kwargs as JSON")
    flag.Parse()
    if *identity == "" { log.Fatal("identity is required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    result, err := service.UpdateRouterRecord(ctx, *identity,
        network.WithRouterUpdateJSON(json.RawMessage(*attrs)))
    if err != nil { log.Fatal(err) }
    var value any = result.Router
    if result.NoOp {
        value = nil
        if result.Lookup != nil { value = result.Lookup.Resource }
    }
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Install](../docs/install.md)
