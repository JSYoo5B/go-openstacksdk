# Cloud Router Interface Remove Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/remove_router_interface`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.RemoveInterfaceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfaceSubnet`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfacePort`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.RemoveRouterInterface`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceRemoveOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceRemoveSubnetID`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceRemovePortID`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceRemoveHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.RemoveCloudInterfaceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfaceOpts`

- RemoveInterfaceRecord sends PUT /v2.0/routers/{id}/remove_router_interface with subnet_id, or port_id when a port is given, and with neither it still sends subnet_id null.
  Tests: `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBRemoveRouterInterfaceBodies`.
- Concrete SDK options own absent/null/raw JSON and cached SDK Resource references. Both subnet_id and port_id falsey rejects before network service/router ID validation or HTTP. Python-loaded float-underflow/empty values are falsey; nonnil Resources remain truthy even with null/false/zero/empty IDs. Plain JSON dictionaries remain body values rather than being treated as Resources.
  Tests: `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveBothFalseyFirstAndHTTPZero`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemovePortFirstRawAndCachedResourceReferences`.
- Source performs no get_router/find/name resolution: strings are literal router IDs. A truthy port wins and causes only port_id to be sent, otherwise subnet_id. SDK-owned raw/PortRecord/SubnetRecord/RawResource conversion avoids application builders/per-record adapters and owns copied options, selected service/API/client/header state and immediately stopping callbacks after detected source changes.
  Tests: `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemovePortFirstRawAndCachedResourceReferences`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveOwnsOptionRefsHeadersAndEscapedState`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveCancellationNilOptionsAndStickySourceStopCallbacks`.
- Router.remove_interface evaluates actual response.json before cloud discards it and returns None. Successful Value is JSON null for arbitrary valid object/list/string/number/bool/null, while ResponseJSON/optional actual Wire/whole Envelope/header/status retain independent physical observations. Accepted empty204 or malformed JSON remains decode error; no synthetic router Resource/default or convergence claim is created.
  Tests: `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveSourceNoneAfterArbitraryJSONAndActualCopies`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveAcceptedDecodeAndCleanHTTPErrors`.
- A closed new helper reuses existing interface reference/body/header/JSON and shared REST contracts, opting into Source100..399 and StopFaultRetryCallbacks. Existing RemoveInterfaceRecord default-subnet-null/200..399 and native status/required behavior remain unchanged; 199 lane isolation is verified without promoting another Source API.
  Tests: `network/v2/extensions/layer3/routers/python_cloud_interface_remove_test.go:TestPythonCloudRouterInterfaceRemoveClosedSourceStatusAndLegacyIsolation`.
- Clean403/404/409 remains actual native error, not absence/success. Accepted and rejected physical read/Close/cancel/source faults preserve actual status/body/header and original causes with exactly one physical PUT/body Close and zero retry/backoff/reauth callbacks. Native401 ErrOriginal is explicitly inspected despite absent Unwrap, and ErrReauth preserves read/Close causes. Clean401 renewal and503 retry keep original request body/headers and actual final receipt.
  Tests: `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveAcceptedDecodeAndCleanHTTPErrors`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemovePhysicalFaultsKeepActualProofAndPreventReplay`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveCleanRetryAndTokenRenewalKeepCapturedBody`, `network/v2/extensions/layer3/routers/python_cloud_interface_remove_test.go:TestPythonCloudRouterInterfaceRemovePhysicalOptInNeverCallsHooks`.
- Only RemoveCloudInterfaceRecord normalizes explicit ISO8859-1/latin1, UTF16/32 with BOM or explicit LE/BE through the fixed shared responsejson.Normalize ABI. Parsed Value and optional object Wire.Body use normalized JSON; Envelope/Header/status/ResponseError.Body and Wire HTTP metadata preserve actual wire evidence. Unknown/malformed charset and bare UTF16 without BOM keep Unsupported cause; normalized malformed JSON retains SyntaxError and original bytes. Existing RemoveInterfaceRecord and generic routerJSONFromResponse charset/status/body defaults stay unchanged.
  Tests: `network/v2/extensions/layer3/routers/python_cloud_interface_remove_test.go:TestPythonCloudRouterInterfaceRemoveExplicitCharsetsKeepRawProofAndLegacy`, `network/v2/extensions/layer3/routers/python_cloud_interface_remove_test.go:TestPythonCloudRouterInterfaceRemoveCharsetFailuresKeepOriginalResponse`.
- The existing cloud Service exposes successful None as Value null after charset-aware parsing while diagnostic ResponseJSON and optional normalized object Wire preserve parsed fields; actual raw Envelope/Header/status remain independent. Unsupported charset does not expose successful Value/ResponseJSON and retains original accepted ResponseError proof.
  Tests: `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveExplicitCharsetReceiptAndNone`.

Python/Go differences:

- Existing RemoveInterfaceRecord keeps its original parsed-JSON return. The separate Service.RemoveRouterInterface returns Source None as Value null with an SDK-owned diagnostic ResponseJSON and physical response receipt.
- This implements the named cloud remove_router_interface flow but remains unresolved due to the remaining fullRequests decoder/permissiveJSON gaps with SDK-owned validation/options/port-first selection/None return and actual response evidence, without a new Connection forwarding method. conn.Network(ctx) service construction may authenticate/discover before argument validation; direct selected/nil Service missing-ID validation remains HTTP0. New leaf helper is only this fixed dependency and no extra Source API is promoted; full SDK supported delta remains0.
- Go router input is a validated nonblank UTF8 string ID, escaping slash/percent/query as one segment and rejecting controls/traversal. Source arbitrary Router Resource/dict/null/nonstring dynamic dispatch and URI/class/member/session/cache behavior remain shared SDK-R1/C1/S1 foundation scope. No name lookup is invented, and broad Python runtime objects are not claimed from raw JSON preservation.
- Go returns an owned diagnostic result with Value:null for Source None plus actual parsed ResponseJSON/Wire/Envelope/Header/StatusCode rather than returning a bare nil. Failed observations never expose successful Value/ResponseJSON or a synthetic Router Resource. Valid JSON numeric tokens preserve Go raw precision; Python floating/runtime representation and wire key order remain explicit Go policies.
- The closed Source100..399/physical-fault callback suppression opt-in leaves existing native and legacy leaf policies unchanged. Headers are a concrete Go transport convenience with immutable selected source/client/captured framing and live token renewal. Stock Neutron27 project ADMIN_OR_PROJECT_MEMBER gate is documented without inferring parent-owner reader/shared-subnet authorization or live ACL acceptance.
- Evidence is actual CPython3.14.8 pinned cloud/Proxy/Resource/Router code with Adapter.put/_get_connection fixtures (75 observations), plus independently audited bounded router13 observations. Source/AST, Go/race, main compilation and live backend authorization are separate claims; no live backend was used.
- Cycle12 changes only the fixed cloud dependency to reuse internal/responsejson.Normalize for bounded explicit charsets. The generic/legacy decoder and common ABI/codec set remain unchanged. BOM is required for bare UTF16/32; explicit LE/BE works without it. Actual raw proof stays separate from normalized diagnostic JSON. Reused Source75 and independent13 observations are not rerun or combined into a new count. Worker ran gofmt only; root owns Go/race/main builds; no mapping promotion is claimed.

Unresolved boundaries:

- Full Requests Response.json codec registry, unknown explicit charsets, charset-free BOM/guess_json_utf/text fallback and bare UTF16/UTF32 without BOM native-endian decoding remain unsupported. The closed cloud helper now supports ISO8859-1/latin1 and UTF16/32 with BOM or explicit LE/BE. Unsupported/malformed declared charset retains ErrUnsupported plus original response evidence, while unmarked nonUTF8 preserves the original decode failure. This narrower reachable decoder gap still prevents complete named mapping.
- Full permissive Python JSON loading and runtime numeric domain remain unmatched. Source Response.json may accept NaN/Infinity-style values before cloud discards the result and returns None, while the reused Go JSON carrier rejects nonstandard JSON tokens. Finite JSON raw numbers retain Go literal precision rather than claiming all Python loaded-value semantics.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--router", required=True)
parser.add_argument("--subnet-json", default="null")
parser.add_argument("--port-json", default="null")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    result = conn.remove_router_interface(
        args.router,
        subnet_id=json.loads(args.subnet_json),
        port_id=json.loads(args.port_json),
    )
    print(json.dumps(result))
```

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
 "github.com/JSYoo5B/go-openstacksdk/network"
)

func run(ctx context.Context, cloud, router, subnetJSON, portJSON string) error {
 conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
 if err != nil { return err }
 service, err := conn.Network(ctx)
 if err != nil { return err }
 result, err := service.RemoveRouterInterface(ctx, router,
  network.WithRouterInterfaceRemoveSubnetID(json.RawMessage(subnetJSON)),
  network.WithRouterInterfaceRemovePortID(json.RawMessage(portJSON)))
 if result != nil {
  fmt.Fprintf(os.Stderr, "PUT status=%d request-id=%q\n",
   result.StatusCode, result.Header.Get("X-Request-Id"))
 }
 if err != nil { return err }
 _, err = fmt.Fprintln(os.Stdout, string(result.Value))
 return err
}

func main() {
 cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
 router := flag.String("router", "", "literal router ID, not a name lookup")
 subnet := flag.String("subnet-json", "null", "subnet_id as JSON")
 port := flag.String("port-json", "null", "port_id as JSON; a truthy port wins")
 flag.Parse()
 if *router == "" {
  fmt.Fprintln(os.Stderr, "router is required")
  os.Exit(2)
 }
 ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
 defer cancel()
 if err := run(ctx, *cloud, *router, *subnet, *port); err != nil {
  fmt.Fprintln(os.Stderr, err)
  os.Exit(1)
 }
}
```

## References

- [Install](../docs/install.md)
- [Router](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/router.py#L384-L398)
