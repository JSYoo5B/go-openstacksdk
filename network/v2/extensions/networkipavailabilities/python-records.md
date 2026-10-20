# Python Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/network_ip_availabilities`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListConflictingAttrs`

- List sends GET network-ip-availabilities with network_name, ip_version, project_id, sort_key and fields and decodes totals, used counts and subnet rows from one page.
  Tests: `network/v2/extensions/networkipavailabilities/python_parity_core_test.go:TestPythonParityCoreIPAvailabilityListAndGet`.
- Owned nullable Resource descriptors preserve big integer/list/dict/tenant fallback while original Wire/Envelope remain separate; network_id does not synthesize id.
  Tests: `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityRecordsOwnedProjectionAndRawReceipt`.
- Lazy allpage list splits exact QueryParameters from descriptor-local filters, discards unknown before value encoding and follows plural next links.
  Tests: `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityRecordsAllPagesServerQueriesAndLocalFilters`.
- Raw-row caps stop before invalid tail/foreign link, option snapshots and GET-only version remain private; physical errors suppress callbacks and unsafe preflight emits no HTTP.
  Tests: `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityRecordsCapsVersionsAndOwnedOptions`, `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityRecordsPhysicalFailuresAndPreflight`.
- Proxy JMESPath consumes opaque lazy generator: @ and projection require no HTTP, unsupported length fails before request.
  Tests: `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityProxyGeneratorAndJSONProjection`.

Python/Go differences:

- Existing native List/Get/GetFields retain their typed onepage/string-counter behavior; new SDK-owned records supply allpage/localfilter/descriptor evidence without caller builders.
- Pinned Resource._get_one_match compares inherited id/name; unused name_attribute network_name and ordinary network_id are not synthesized into lookup identity. This corrects the older assessment that said fallback matches network_id/network_name.
- Record JSON descriptor/literal path profile deliberately differs from full mutable Python class/session/Requests codec/nonfinite-JSON domain; origin/path safety limits and owned evidence are Go policies.
- Independent Source AST extraction executed10 assertions with stubs for Resource.find/_get_one_match/_get_next_link, not full Proxy/decoder runtime or live Neutron.

### `python:network/v2/find_network_ip_availability`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordFindListBasePath`

- GET-first clean400/403/404 fallback compares inherited actual id/name, not network_id/network_name, and adds no unsupported server name hint.
  Tests: `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityFindActualInheritedIdentityAndNoNameHint`.
- Sparse direct response overlays owned constructor seed and preserves actual Wire without seed fields.
  Tests: `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityFindDirectSparseSeedAndNoFallback`.
- Default missing returns nil, strictmissing ErrNotFound, later exact id/name duplicate ErrAmbiguous and late page503 invalidate earlier candidates.
  Tests: `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityFindMissingAmbiguityAndLateFailure`.
- Fallback route override, raw-rowcap, owned options and member-only microversion preserve selected client; physical faults never become cleanfallback and constructor/source/cancel errors stay preflight.
  Tests: `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityRecordsCapsVersionsAndOwnedOptions`, `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityRecordsPhysicalFailuresAndPreflight`.

Python/Go differences:

- Existing native List/Get/GetFields retain their typed onepage/string-counter behavior; new SDK-owned records supply allpage/localfilter/descriptor evidence without caller builders.
- Pinned Resource._get_one_match compares inherited id/name; unused name_attribute network_name and ordinary network_id are not synthesized into lookup identity. This corrects the older assessment that said fallback matches network_id/network_name.
- Record JSON descriptor/literal path profile deliberately differs from full mutable Python class/session/Requests codec/nonfinite-JSON domain; origin/path safety limits and owned evidence are Go policies.
- Independent Source AST extraction executed10 assertions with stubs for Resource.find/_get_one_match/_get_next_link, not full Proxy/decoder runtime or live Neutron.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--find", default="")
    parser.add_argument("--network-name", default="")
    args = parser.parse_args()
    with openstack.connect(cloud=args.cloud) as conn:
        if args.find:
            result = conn.network.find_network_ip_availability(args.find, ignore_missing=False)
            print(json.dumps(result.to_dict(), ensure_ascii=False))
        else:
            query = {"network_name": args.network_name} if args.network_name else {}
            for record in conn.network.network_ip_availabilities(**query):
                print(json.dumps(record.to_dict(), ensure_ascii=False))


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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities"
)

func run(ctx context.Context, cloud, identity, name string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.NetworkV2(ctx)
    if err != nil { return err }
    api := service.NetworkIPAvailabilities
    if identity != "" {
        record, err := api.FindRecord(ctx, identity, networkipavailabilities.WithRecordFindIgnoreMissing(false))
        if err != nil { return err }
        return json.NewEncoder(os.Stdout).Encode(record.Resource.Body)
    }
    var options []networkipavailabilities.RecordListOption
    if name != "" { options = append(options, networkipavailabilities.WithRecordListQuery("network_name", name)) }
    for record, err := range api.ListRecords(ctx, options...) {
        if err != nil { return err }
        if err := json.NewEncoder(os.Stdout).Encode(record.Resource.Body); err != nil { return err }
    }
    return nil
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    identity := flag.String("find", "", "literal member ID or Source fallback id/name")
    name := flag.String("network-name", "", "server-side network name filter")
    timeout := flag.Duration("timeout", time.Minute, "whole operation deadline")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud, *identity, *name); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

```sh
python main.py --cloud devstack --network-name public
go run main.go --cloud devstack --network-name public
python main.py --cloud devstack --find n1
go run main.go --cloud devstack --find n1
```
