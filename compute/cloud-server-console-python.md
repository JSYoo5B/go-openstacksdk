# Cloud Server Console Python

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_compute/ComputeCloudMixin/get_server_console`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ConsoleOutput`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetServerConsole`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerConsoleOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerConsoleLength`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerConsoleLengthJSON`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithoutServerConsoleLength`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerConsoleServerJSON`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerConsoleServerRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithoutServerConsoleServer`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ConsoleOutputRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithCloudConsoleOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithCloudConsoleLength`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithCloudConsoleLengthJSON`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithoutCloudConsoleLength`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetServerConsole`

- Existing typed FindRecord and ConsoleOutput preserve their original lookup/action and 400 error contract. The separate GetServerConsole helper below owns the clean-action400 empty-string transformation.
  Tests: `compute/v2/python_parity_test.go:TestPythonCloudServerDeleteAndConsolePartialPaths`.
- Source bare=True lookup uses existing FindRecord defaults(details=True/all_projects=False), member first then detailed local exact fallback, no network/image/flavor expansion; dictionary and Source record seed bypass lookup. Missing/duplicate/lookup physical errors precede the action-only400 catch.
  Tests: `compute/cloud_server_console_test.go:TestCloudServerConsoleBareLookupAndSeedBypass`, `compute/cloud_server_console_test.go:TestCloudServerConsoleNameFallbackUsesDetailsLocalScopeAndAmbiguity`, `compute/cloud_server_console_test.go:TestCloudServerConsoleConnectionLocationSnapshotAndGuard`.
- Nil/explicit null length omits JSON field; zero,false,negative,string,array,dictionary,large integer lengths remain raw. Factories and callback application own snapshots; resets select default lookup/length; nil/cancel/client bindings and invalidUTF8 supplied dictionaries fail before workflowHTTP. Malformed lengthJSON and effective dot/control IDs fail before leaf actionHTTP; a cloud string lookup may already have performed HTTP, preserving lookup-before-action phase order.
  Tests: `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordLiteralLengthCaseAndRawOutput`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordOptionsAreOwnedAndReset`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordPreflightAndSourceCallback`, `compute/cloud_server_console_test.go:TestCloudServerConsoleLengthServerSnapshotsRecordAndReset`, `compute/cloud_server_console_test.go:TestCloudServerConsoleNilCancelledAndSourceBindingsPreflight`.
- Raw output membership is case-sensitive and cast is not coercion: null/false/0/list/dict values preserved, missing key is Value null/HasOutput false, explicit null is true; empty root lists/quiet strings return null, output-containing nonmapping/rootnull/scalar or malformed/204 JSON retains actual response error.
  Tests: `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordMembershipPresenceAndShape`, `compute/cloud_server_console_test.go:TestCloudServerConsoleCleanAction400OnlyAndRawPresence`.
- Falsey dictionary id maps to Source /servers//action; a dictionary-valued id follows the selected Proxy inner-ID extraction once; absent outerID fails before action, absent innerID uses the falsey route. Arbitrary Server constructor attrs/descriptors remain SDK-R1 foundation gaps. Effective ID trims outer slashes, then bounded dot/control validation/literal escaping; Seed does not fabricate lookup response.
  Tests: `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordFalseyAndLiteralIDRoutes`, `compute/cloud_server_console_test.go:TestCloudServerConsoleDictionaryFalseyIDsAndMissingKeys`.
- Action onlyclean400 becomes Value JSON emptystring/Unsupported true with actual400 receipt. Dirty400 read/Close/cancel/sticky restoredsource/nativecallback/expandedOkCodes never suppressed; accepted actionread failure preserves independent lookup/action receipts.
  Tests: `compute/cloud_server_console_test.go:TestCloudServerConsoleDirty400NeverBecomesUnsupported`, `compute/cloud_server_console_test.go:TestCloudServerConsoleCleanRetryStopsAndAcceptedReadKeepsSeparateLookup`.
- The remaining unmarked/unknown-codec Requests JSON-decoding and nonfinite numeric branches stay unsupported with actual response proof. Explicit supported charset normalization is covered by the added contract below. Existing Source68 actualCloud/Proxy/Server/Requests2.34.2/urllib32.8.0/CPython3.14.8 observations are reused, not extended; fixedfind adapter and memoryHTTP post remain distinct from live lookup/auth/retry proof.
  Tests: `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordRequestsDecodingAndNonFiniteGapStayExplicit`.
- Connection exposes this cloud read through its cached service and SDK-owned location defaults. Cancellation is checked before catalog discovery. Only Neutron list helpers turn clean missing catalog endpoints into [] before evaluating options; searches retain catalog errors. Caller-provided options retain their explicit values.
  Tests: `connectiontest/connection_cycle8_reads_test.go:TestConnectionCycle8CancelledReadsNeverDiscover`, `connectiontest/connection_cycle8_reads_test.go:TestConnectionCycle8ConsoleUsesCachedComputeAndSuppliedDictionary`.
- Accepted100..399 remains original source policy; selectedversion retained, undiscoveredversion uses Server2.100 ceiling. FixedemptyAccept/version/request JSON/origin and copiedsourceheaders surviveclean retry/reauth; faults at200/400/401/403/404/429/500/777 keep physicalheaders/body/status and block new action hooks. ErrUnable original401 receipt is recovered explicitly without dropping ErrReauth. Cloud options pre-capture private readerheaders for both lookup/action while original selected bindings remain guarded. Console discovery explicitly copies NovaProfile with StopFaultRetryCallbacks=true; physicaldiscovery503 read+Close faults preserve actual native proof without a POST or hooks; global profile baseline remains unchanged.
  Tests: `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordOriginalStatusAndInformationalPolicy`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordDiscoveryUsesServerCeilingAndSelectedShortcut`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordPhysicalFaultsRetainActualProofAndStopActionRetry`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordRetryOwnershipAndNativeReauthentication`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordOwnedSourceHeadersAndIndependentReceipt`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordFailedReauthReceiptKeepsOriginalAndCause`, `compute/cloud_server_console_test.go:TestCloudServerConsoleCapturesHeadersBeforeOptionsAndAcrossLookupAction`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordDiscoveryFaultOptInStopsCallbacksBeforeAction`.
- Shared responsejson.Normalize decodes explicit UTF16(BOM/LE/BE), UTF32(BOM/LE/BE), ISO8859-1/latin1 before output parsing. Representative Latin1/UTF16-BOM/UTF32BE accepted202 output values are UTF8 JSON while Envelope/Header remain independent actual wire proof. Unknown/malformed charset and supported-charset malformed JSON preserve original203 ResponseError.Body/Header with distinct Unsupported/InvalidOption causes; shared decoder matrix is root owned.
  Tests: `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordExplicitCharsetsKeepWireReceipt`, `compute/v2/servers/python_cloud_console_test.go:TestPythonCloudConsoleRecordCharsetFailuresPreserveActualResponse`.

Python/Go differences:

- Gophercloudv2.15.0 and openstacksdk ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe are pinned. Closed JSON options are SDK-owned concrete factories; callers need no interface builder. Value rawJSON plus HasOutput/Unsupported and separate lookup/seed/action evidence replaces dynamic str|None return.
- SDK-R1/C1/S1 Resource class/constructor/session/cache foundations remain separate shared gaps; general Proxy.get_server_console_output and mutable Server/attrs are not newly mapped. No full cloud get_server expansion is claimed.
- Nil/cancel and invalid unsafe route controls are bounded preflight; effective IDs are escaped literal segments inside the captured Nova origin. Falsey and one-level nested dictionary ID Source lanes remain supported.
- Go integer/rawJSON options retain finite literal values instead of prematurely casting Source annotations. Go raw numbers preserve their original decimal spelling/large integers rather than Python float conversion; nonfinite Python values remain an explicit named gap.
- Synthetic final informational199 status tests establish policy only, because actual net/http may process1xx internally. SDK action ownership guards block retry/auth hooks after physical/context/source faults; reused FindRecord existing lookup hook semantics remain unchanged.
- Source oracle substitutes only fixedfind return/errors and post memoryHTTPResponse; no liveNova/auth/ACL/retry evidence. Python mainASTPASS; root owns Go gates and standalone mainbuilds.
- Cycle11 adds only bounded explicit response charset normalization in the console leaf, reused by existing Service/Connection GetServerConsole. Actual Envelope/Header/ResponseError.Body keep wirebytes while Value comes from normalized JSON. Existing Source68 receipts are unchanged; no new Source matrix, codec registry, guess_json_utf or nonfinite support is claimed. Worker ran gofmt only; root owns Go/race and standalone main validation.
- Text is decoded only from UTF-8, a byte-order mark or a supported declared charset (UTF-16/UTF-32 with a BOM or explicit endianness, ISO-8859-1); undeclared non-UTF-8 bytes and other codecs return ErrUnsupported with the original bytes instead of Requests' statistical detection or the full CPython codec registry; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Only standard finite JSON numbers are accepted: NaN, Infinity and overflowing literals such as 1e400 return ErrUnsupported with the original response, where Python's json loads them as floats, and a latest microversion must be written as "latest"; see Deliberate runtime differences in docs/sdk-support-ledger.md.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--server", default="vm")
    parser.add_argument("--server-json", default="")
    parser.add_argument("--length-json", default="")
    args = parser.parse_args()
    server = json.loads(args.server_json) if args.server_json else args.server
    length = json.loads(args.length_json) if args.length_json else None
    with openstack.connect(cloud=args.cloud) as conn:
        output = conn.get_server_console(server, length=length)
        print(json.dumps(output, ensure_ascii=False))


if __name__ == "__main__":
    main()
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
    "github.com/JSYoo5B/go-openstacksdk/compute"
)

func run(ctx context.Context, cloud, server, suppliedServer, length string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    var options []compute.ServerConsoleOption
    if suppliedServer != "" {
        options = append(options, compute.WithServerConsoleServerJSON(json.RawMessage(suppliedServer)))
    }
    if length != "" {
        options = append(options, compute.WithServerConsoleLengthJSON(json.RawMessage(length)))
    }
    result, err := conn.GetServerConsole(ctx, server, options...)
    if err != nil { return err }
    if result == nil { return fmt.Errorf("console result is missing") }
    return json.NewEncoder(os.Stdout).Encode(struct {
        Output json.RawMessage `json:"output"`
        HasOutput bool `json:"has_output"`
        Unsupported bool `json:"unsupported"`
        StatusCode int `json:"status_code"`
    }{result.Value, result.HasOutput, result.Unsupported, result.Action.StatusCode})
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    server := flag.String("server", "vm", "literal name or ID; ignored when server-json is supplied")
    suppliedServer := flag.String("server-json", "", "owned server dictionary, for example {\"id\":\"s1\"}")
    length := flag.String("length-json", "", "optional literal JSON length; empty flag omits it")
    timeout := flag.Duration("timeout", time.Minute, "whole lookup and action deadline")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud, *server, *suppliedServer, *length); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

```sh
python main.py --cloud devstack --server vm --length-json 50
go run main.go --cloud devstack --server vm --length-json 50
python main.py --cloud devstack --server-json '{"id":"s1"}' --length-json 0
go run main.go --cloud devstack --server-json '{"id":"s1"}' --length-json 0
```

```go
service, err := conn.Compute(ctx)
if err != nil { return err }
result, err := service.GetServerConsole(ctx, "vm", compute.WithServerConsoleLength(50))
```
