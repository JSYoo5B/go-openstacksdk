# Python Port Bindings

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/activate_port_binding`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/portbindings.API.Activate`

- Two actual parent GETs precede first matching Binding selection; activation stops iteration and applies flat JSON Body updates without unwrapping or registering new unknown fields; rejected/invalid physical responses preserve proof.
  Tests: `network/v2/extensions/portbindings/python_records_test.go:TestPortBindingActivateRepeatsParentAndStopsAtFirstMatch`, `network/v2/extensions/portbindings/python_records_test.go:TestPortBindingActivationMissingAndCapturedOptions`.

Python/Go differences:

- The SDK owns concrete finite-JSON options, safe string port/host IDs and explicit Go receipt/result structures instead of accepting Python Resource objects or returning None. Parent full Port descriptors and common REST/Resource engines are reused.
- Go List defers its parent GET until iteration. Polymorphic string JMESPath list returns, raw mutation base_path/Proxy constructor restoration, arbitrary Python objects and nonstring activation pair keys are explicitly outside this concrete typed API. Headers and microversion are Go conveniences.
- Requests codecs, mutable Resource/cache lifecycle, SDK-R1/C1/S1 and live-cloud/release contracts remain tracked globally; API mapping does not assert full SDK parity.

### `python:network/v2/create_port_binding`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/portbindings.API.Create`

- Full Port GET resolves the parent ID before collection POST; only explicit raw attributes including unknown extensions/location are sent, while nullable/dict views and original physical evidence remain separate.
  Tests: `network/v2/extensions/portbindings/python_records_test.go:TestPortBindingCreateResolvesParentAndPreservesRawAttrs`, `network/v2/extensions/portbindings/python_records_test.go:TestPortBindingPreflightAndCreateCollisions`, `network/v2/extensions/portbindings/python_records_test.go:TestPortBindingActivationMissingAndCapturedOptions`.

Python/Go differences:

- The SDK owns concrete finite-JSON options, safe string port/host IDs and explicit Go receipt/result structures instead of accepting Python Resource objects or returning None. Parent full Port descriptors and common REST/Resource engines are reused.
- Go List defers its parent GET until iteration. Polymorphic string JMESPath list returns, raw mutation base_path/Proxy constructor restoration, arbitrary Python objects and nonstring activation pair keys are explicitly outside this concrete typed API. Headers and microversion are Go conveniences.
- Requests codecs, mutable Resource/cache lifecycle, SDK-R1/C1/S1 and live-cloud/release contracts remain tracked globally; API mapping does not assert full SDK parity.

### `python:network/v2/delete_port_binding`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/portbindings.API.Delete`

- Two parent GETs and complete matching-list materialization precede one DELETE per matching row; default ignore_missing applies only to initial port404/empty selection, and late list or physical failures stop mutation.
  Tests: `network/v2/extensions/portbindings/python_records_test.go:TestPortBindingDeleteMaterializesAllMatchesBeforeMutations`, `network/v2/extensions/portbindings/python_records_test.go:TestPortBindingDeleteIgnoreMissingAppliesOnlyToSourceBranches`, `network/v2/extensions/portbindings/python_records_test.go:TestPortBindingPhysicalFaultNeverBecomesIgnoredDeletion`.

Python/Go differences:

- The SDK owns concrete finite-JSON options, safe string port/host IDs and explicit Go receipt/result structures instead of accepting Python Resource objects or returning None. Parent full Port descriptors and common REST/Resource engines are reused.
- Go List defers its parent GET until iteration. Polymorphic string JMESPath list returns, raw mutation base_path/Proxy constructor restoration, arbitrary Python objects and nonstring activation pair keys are explicitly outside this concrete typed API. Headers and microversion are Go conveniences.
- Requests codecs, mutable Resource/cache lifecycle, SDK-R1/C1/S1 and live-cloud/release contracts remain tracked globally; API mapping does not assert full SDK parity.

### `python:network/v2/port_bindings`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/portbindings.API.List`

- Owned lazy iteration resolves the parent, walks every page, applies declared host/body filters locally without server-name queries and preserves separate Port and Binding response evidence.
  Tests: `network/v2/extensions/portbindings/python_records_test.go:TestPortBindingListFiltersLocallyAndOwnsParentEvidence`, `network/v2/extensions/portbindings/python_records_test.go:TestPortBindingDeleteMaterializesAllMatchesBeforeMutations`.

Python/Go differences:

- The SDK owns concrete finite-JSON options, safe string port/host IDs and explicit Go receipt/result structures instead of accepting Python Resource objects or returning None. Parent full Port descriptors and common REST/Resource engines are reused.
- Go List defers its parent GET until iteration. Polymorphic string JMESPath list returns, raw mutation base_path/Proxy constructor restoration, arbitrary Python objects and nonstring activation pair keys are explicitly outside this concrete typed API. Headers and microversion are Go conveniences.
- Requests codecs, mutable Resource/cache lifecycle, SDK-R1/C1/S1 and live-cloud/release contracts remain tracked globally; API mapping does not assert full SDK parity.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--port", required=True)
parser.add_argument("--host", required=True)
parser.add_argument("--mode", choices=("create", "list", "activate", "delete"), required=True)
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "create":
        row = conn.network.create_port_binding(
            args.port, host=args.host, profile={"capabilities": ["switchdev"]})
        value = row.to_dict()
    elif args.mode == "list":
        value = [row.to_dict() for row in conn.network.port_bindings(
            args.port, host=args.host)]
    elif args.mode == "activate":
        row = conn.network.activate_port_binding(args.port, args.host)
        value = None if row is None else row.to_dict()
    else:
        conn.network.delete_port_binding(args.port, args.host)
        value = None
    print(json.dumps(value, indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/portbindings"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    port := flag.String("port", "", "Port ID")
    host := flag.String("host", "", "Binding host")
    mode := flag.String("mode", "", "create, list, activate or delete")
    flag.Parse()
    if *port == "" || *host == "" { log.Fatal("port and host are required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    network, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := network.PortBindings
    var value any
    switch *mode {
    case "create":
        value, err = api.Create(ctx, *port,
            portbindings.WithAttribute("host", *host),
            portbindings.WithAttribute("profile", map[string]any{"capabilities": []string{"switchdev"}}))
    case "list":
        value, err = api.All(ctx, *port, portbindings.WithQuery("host", *host))
    case "activate":
        value, err = api.Activate(ctx, *port, *host)
    case "delete":
        value, err = api.Delete(ctx, *port, *host)
    default:
        log.Fatal("mode must be create, list, activate or delete")
    }
    if err != nil { log.Fatal(err) }
    encoded, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(encoded))
}
```

## References

- [Install](../../../../docs/install.md)
