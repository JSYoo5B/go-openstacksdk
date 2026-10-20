# Python Record Reads

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:baremetal/v1/nodes`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.WithRecordListDetails`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.WithRecordListQuery`

- Lazy basic/detail collection routes, declared query aliases and CSV fields conversion retain Source descriptor/local-filter behavior and actual unknown wire fields.
  Tests: `baremetal/v1/nodes/python_record_reads_test.go:TestPythonNodeListDetailsCSVAndDeclaredProjection`, `baremetal/v1/nodes/python_record_reads_test.go:TestPythonNodeListExplicitPathAndSourceCSVString`.
- Opaque Proxy generator expressions avoid HTTP until iteration; early break stops pagination and collection failures discard partial results.
  Tests: `baremetal/v1/nodes/python_record_reads_test.go:TestPythonNodeListProxyPreservesOpaqueGeneratorWithoutHTTP`, `baremetal/v1/nodes/python_record_reads_test.go:TestPythonNodeListPaginationStopsAfterBreakAndDropsPartialCollection`.
- Unset versions discover and cap at 1.109 while explicit selections remain uncapped; discovery receipts are independently owned.
  Tests: `baremetal/v1/nodes/python_record_reads_test.go:TestPythonNodeDefaultVersionDiscoveryAndUncappedSelection`.

Python/Go differences:

- Owned immutable JSON views and literal UTF-8 identifiers/routes replace arbitrary Python Resource/session/formatter objects; unknown Node fields remain in actual Wire/Envelope but not the declared view. SDK-R1/C1/S1 and live-cloud/release validation remain separate.
- Explicit client/per-call versions remain uncapped; otherwise endpoint bounds and the pinned Node class maximum 1.109 select the version. Discovery receipts are separate from resource receipts. No SDK HTTP cache or bypass of caller transport caches is claimed.
- Typed CSV fields accept JSON strings/string arrays; Python arbitrary iterable/hashable inputs are outside the JSON profile. Source string fields iterate characters. Typed collection iteration rejects arbitrary scalar JMESPath results; ProxyListRecords preserves the separate opaque-generator behavior.

### `python:baremetal/v1/get_node`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.WithRecordGetFields`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.WithRecordGetOpts`

- GET maps direct Node Body fields including uuid aliases, preserves repeated/unknown selections and passive explicit null identity, and separates view from actual accepted evidence.
  Tests: `baremetal/v1/nodes/python_record_reads_test.go:TestPythonNodeGetMapsFieldsAndKeepsPassiveUUID`.
- Get 404 remains strict; owned options, preflight identity guards and source guards prevent unintended requests.
  Tests: `baremetal/v1/nodes/python_record_reads_test.go:TestPythonNodeFindGETDoesNotRematchAndGet404IsStrict`, `baremetal/v1/nodes/python_record_reads_test.go:TestPythonNodeOptionsOwnDataAndGuardPreflight`, `baremetal/v1/nodes/python_record_reads_test.go:TestPythonNodeHeadersCapturedBeforeCallbacks`.
- Rejected physical read/close/cancellation faults retain their causes without fallback or replay.
  Tests: `baremetal/v1/nodes/python_record_reads_test.go:TestPythonNodePhysicalReadCloseAndCancelNeverFallback`.

Python/Go differences:

- Owned immutable JSON views and literal UTF-8 identifiers/routes replace arbitrary Python Resource/session/formatter objects; unknown Node fields remain in actual Wire/Envelope but not the declared view. SDK-R1/C1/S1 and live-cloud/release validation remain separate.
- Explicit client/per-call versions remain uncapped; otherwise endpoint bounds and the pinned Node class maximum 1.109 select the version. Discovery receipts are separate from resource receipts. No SDK HTTP cache or bypass of caller transport caches is claimed.
- The public Go target is a string and fields are []string, rather than arbitrary Resource/iterable objects. Per-call headers and microversion are explicit Go conveniences.

### `python:baremetal/v1/find_node`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.WithRecordFindDetails`
- `github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes.WithRecordFindIgnoreMissing`

- Clean rejected GET falls back to detailed collection by default or basic collection explicitly, without a name query hint; complete search detects duplicates, late errors and missing/strict policy.
  Tests: `baremetal/v1/nodes/python_record_reads_test.go:TestPythonNodeFindFallbackDetailsDefaultsAndMissing`.
- Accepted GET is returned without rematching the requested name; physical failures are never converted into fallback.
  Tests: `baremetal/v1/nodes/python_record_reads_test.go:TestPythonNodeFindGETDoesNotRematchAndGet404IsStrict`, `baremetal/v1/nodes/python_record_reads_test.go:TestPythonNodePhysicalReadCloseAndCancelNeverFallback`.

Python/Go differences:

- Owned immutable JSON views and literal UTF-8 identifiers/routes replace arbitrary Python Resource/session/formatter objects; unknown Node fields remain in actual Wire/Envelope but not the declared view. SDK-R1/C1/S1 and live-cloud/release validation remain separate.
- Explicit client/per-call versions remain uncapped; otherwise endpoint bounds and the pinned Node class maximum 1.109 select the version. Discovery receipts are separate from resource receipts. No SDK HTTP cache or bypass of caller transport caches is claimed.
- The concrete Go find options match public details/ignore_missing behavior; per-call headers/microversion are owned conveniences, without exposing arbitrary Resource.find runtime kwargs.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import sys
import openstack


def main():
    conn = openstack.connect(cloud='dev', baremetal_api_version='1.109')
    action = sys.argv[1] if len(sys.argv) > 1 else 'list'
    identity = sys.argv[2] if len(sys.argv) > 2 else 'node-a'
    if action == 'list':
        values = conn.baremetal.nodes(details=True, fields=['id', 'name'])
        print(json.dumps([value.to_dict() for value in values], indent=2))
    elif action == 'get':
        value = conn.baremetal.get_node(identity, fields=['id', 'name'])
        print(json.dumps(value.to_dict(), indent=2))
    elif action == 'find':
        value = conn.baremetal.find_node(identity, ignore_missing=True, details=True)
        print(json.dumps(value.to_dict() if value is not None else None, indent=2))
    else:
        raise ValueError('expected list, get or find')


if __name__ == '__main__':
    main()
```

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/baremetal/v1/nodes"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud("dev"))
    if err != nil { log.Fatal(err) }
    service, err := conn.BareMetal(ctx)
    if err != nil { log.Fatal(err) }
    action, identity := "list", "node-a"
    if len(os.Args) > 1 { action = os.Args[1] }
    if len(os.Args) > 2 { identity = os.Args[2] }
    var output any
    switch action {
    case "list":
        values, callErr := service.Nodes.AllRecords(ctx,
            nodes.WithRecordListMicroversion("1.109"),
            nodes.WithRecordListDetails(true),
            nodes.WithRecordListQuery("fields", []string{"id", "name"}))
        err = callErr
        views := make([]any, 0, len(values))
        for _, value := range values { views = append(views, value.Resource.Body) }
        output = views
    case "get":
        value, callErr := service.Nodes.GetRecord(ctx, identity,
            nodes.WithRecordGetMicroversion("1.109"), nodes.WithRecordGetFields("id", "name"))
        err = callErr
        if value != nil && value.Resource != nil { output = value.Resource.Body }
    case "find":
        value, callErr := service.Nodes.FindRecord(ctx, identity,
            nodes.WithRecordFindMicroversion("1.109"), nodes.WithRecordFindDetails(true),
            nodes.WithRecordFindIgnoreMissing(true))
        err = callErr
        if value != nil && value.Resource != nil { output = value.Resource.Body }
    default:
        log.Fatal("expected list, get or find")
    }
    if err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(output, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```
