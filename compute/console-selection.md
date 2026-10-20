# Console Selection

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/create_console`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.CreateConsole`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithConsoleProtocol`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithConsoleProtocolValue`

- CreateConsole first reads the captured endpoint version path without body/query/microversion headers, preserving the selected live token, source header and reverse-proxy prefix. Nova row eligibility requires a permitted status and usable self-link; missing ID/status, unknown or experimental status (even with malformed ID), or no self-link is excluded without following the link. Catalog v2.1 selects an exact eligible v2.1 API row rather than v2.0; flat/version/versions/versions.values documents use the common engine. Clean404/405 allows only the finite root attempt and valid accepted201/300 discovery is parsed.
  Tests: `api/compute_console_composition_test.go:TestComputeConsoleCompositionAdvertisedAndSelectedBranches`.
- Modern dispatch requires both advertised min/max and2.6 inside that range, plus same-major selected support when present. Unselected old max, a minimum above2.6, missing bounds, selected2.5 and global latest use legacy. Explicit selected2.100 above advertised2.90 remains unchanged; finite2.latest supports modern. Absent selected version is privately capped at modern2.99 or legacy2.100; original client binding, endpoint, ResourceBase, version and headers are preserved and only one winning console POST occurs.
  Tests: `api/compute_console_composition_test.go:TestComputeConsoleCompositionAdvertisedAndSelectedBranches`.
- Protocol defaults to explicit null, explicit nonempty/empty values are preserved as response seeds, and a cleared Optional restores the null facade default. Modern known-type protocol inference affects only the transmitted body; unknown modern type sends null for server validation. Webmks2.8 and spice-direct2.99 each require both advertised and selected support before POST, including explicit selected values above cloud max. Legacy ignores protocol and imposes no modern spice-direct gate.
  Tests: `api/compute_console_composition_test.go:TestComputeConsoleCompositionProtocolTypeAndLocationProjection`.
- Modern result has exactly protocol/type/url/id/name/computed location; URI server_id and unknown/response location fields are excluded. Actual known untyped/null/large-number id values overlay seeds, missing id/name become null, and configured scoped location is independently captured once before POST with zone null. Direct no-dependency location is null. Legacy console scalar/array/null and vendor/large-number object values remain precise independently owned JSON.
  Tests: `api/compute_console_composition_test.go:TestComputeConsoleCompositionProtocolTypeAndLocationProjection`, `api/compute_console_composition_test.go:TestComputeConsoleCompositionAdvertisedAndSelectedBranches`.
- Option source/context failures and replacement of API, Servers or RemoteConsoles binding stop later callbacks and discovery; location errors/source drift after successful discovery stop console POST. Original option error and custom cancellation causes are preserved. Accepted discovery read/nested404 Close/source/cancellation/empty204 faults remain terminal with only physical discovery receipt and no root/action retry; shared fault wrappers check actual request/read/Close counts.
  Tests: `api/compute_console_composition_test.go:TestComputeConsoleCompositionGuardsDiscoveryFailuresAndNoErrorFallback`.
- Discovery HTTP403 preserves actual GET URL/body/header/status and prevents root/action fallback. Modern403/404 never invokes legacy and legacy404 never invokes modern. Modern accepted opaque202 retains seeded dictionary, while legacy opaque202 is an accepted response processing error; no failed console action is resent through the other executor.
  Tests: `api/compute_console_composition_test.go:TestComputeConsoleCompositionGuardsDiscoveryFailuresAndNoErrorFallback`.

Python/Go differences:

- Python accepts a Server Resource or ID and dynamic console protocol; Go takes a validated explicit single path-segment text server ID, mandatory string type and optional nullable string protocol. The independently owned json.RawMessage is the modern six-field source dictionary or the exact legacy console value, rather than a mutable connected Resource. The example supplies its own novnc type default.
- Go reuses guarded finite operation-owned endpoint/version-root discovery and tuple support/selection from the Cinder engine. Discovery is versionless and accepted final200..399 is parsed; actual native clean404/405 or a valid unmatched API row alone allows the root attempt. Advertised links do not grant routing/authentication authority, and source/context/physical faults are terminal. This operation does not reproduce the complete cached Keystoneauth session/discovery lifecycle or response charset detection; invalid UTF8/JSON is a receipt error. The fixed-route Go reader uses the first matching eligible API row instead of Python version-data sorting and normalization of all unrelated rows. Go uses the captured authenticated token rather than Python anonymous discovery and401 authenticated fallback; Go terminal errors and clean404/405-only unavailable fallback are deliberate operation-scoped error semantics, while Keystoneauth can try later URLs after broader discovery/HTTP/connection failures.
- An explicit selected microversion is preserved even above advertised max, with finite2.latest matching and global latest failing finite-major support. Absent selection uses a private action client capped at modern2.99 or legacy2.100; Connection client/version policy remains owned and unchanged. Direct compute.New with no CloudLocation dependency returns location null, while conn.Compute supplies the configured scoped computed location. Discovery may run once per operation rather than through Python global discovery cache.
- Branch-specific decoding policy is inherited from the owned executors: modern accepted opaque/empty JSON bodies retain original type/protocol seed with null missing fields, while legacy requires an actual console key and returns its raw JSON value. Go physical read/Close/source/context failures retain actual evidence and do not become tolerated JSON failure. This single composed declaration does not add a new HTTP endpoint or count existing executors again.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
console = conn.compute.create_console("server-id", "novnc")
print(console)
```

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
)

func main() {
    serverID := flag.String("server-id", "", "server ID")
    kind := flag.String("console-type", "novnc", "console type")
    protocol := flag.String("protocol", "", "optional console protocol")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *serverID, *kind, *protocol); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, serverID, kind, protocol string) error {
    if serverID == "" {
        return fmt.Errorf("-server-id is required")
    }
    conn, err := sdk.Connect(ctx)
    if err != nil {
        return err
    }
    service, err := conn.Compute(ctx)
    if err != nil {
        return err
    }
    var options []compute.ConsoleOption
    flag.Visit(func(f *flag.Flag) {
        if f.Name == "protocol" {
            options = append(options, compute.WithConsoleProtocol(protocol))
        }
    })
    console, err := service.CreateConsole(ctx, serverID, kind, options...)
    if err != nil {
        return err
    }
    fmt.Println(string(console))
    return nil
}
```

## References

- [User Actions](user-actions.md)
- [Keypairs Console](keypairs-console.md)
