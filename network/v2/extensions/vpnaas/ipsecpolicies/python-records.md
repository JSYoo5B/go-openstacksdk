# Python Records

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/vpn_ipsec_policies`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListConflictingAttrs`

- List sends GET vpn/ipsecpolicies with WithListOptions and WithListQuery values as query and follows the ipsecpolicies_links next href.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/python_parity_test.go:TestPythonIPsecPolicyListQueryAndLinks`.
- Owned nullable Source policy descriptors keep lifetime nested values raw, units separate and value integer; unknown/tenant/tag/revision fields do not leak into Resource while actual Wire/Envelope remain unmodified.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/python_record_contracts_test.go:TestVPNPolicyRecordsDescriptorDefaultsAndActualProof`.
- Exact server query including phase1_negotiation_mode is separated from nested lifetime/top-level units/value local filters; allpages follow next and unknown values are discarded before encoding.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/python_record_contracts_test.go:TestVPNPolicyRecordsServerQueryAndLocalBodyFiltersAllPages`.
- No-link limit marker continuation and paginated=false are SDK-owned; max_items counts raw rows before filtering and stops before invalid tail/foreign next.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/python_record_contracts_test.go:TestVPNPolicyRecordsMarkerContinuationRawCapAndPaginationFlag`.
- Header/version/basepath/rawquery options are snapshotted and apply only to private operation copies.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/python_record_contracts_test.go:TestVPNPolicyRecordsSnapshotControlsAndVersion`.
- Descriptor/constructor/shape errors retain accepted actual proof, physical body faults suppress retry callbacks, unsafe options/source changes fail before HTTP; Proxy generator JMESPath has correct noHTTP behavior.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/python_record_contracts_test.go:TestVPNPolicyRecordsResponseErrorsAndPhysicalFaults`, `network/v2/extensions/vpnaas/ipsecpolicies/python_record_contracts_test.go:TestVPNPolicyRecordsPreflightAndProxyGenerator`.

Python/Go differences:

- Existing native typed List retains its native query/model behavior; new record APIs use pinned Source QueryParameters and apply non-query Body kwargs locally across all pages.
- Source policy lifetime is a nullable dict; comment defaults seconds/3600 are not descriptor defaults. Top-level units/value remain separate. Go JSONprofile preserves received bytes and owned descriptor projection, with literal path/origin safety and full Requests/NaN/Infinity/runtime-type differences.
- Source declaration/dict-conversion/nested-filter audit used22 AST-extracted assertions with stubs; this is not full Proxy/HTTP/decoder runtime or live Neutron evidence.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cloud", default="devstack")
    parser.add_argument("--name", default="")
    parser.add_argument("--units", default="")
    args = parser.parse_args()
    query = {}
    if args.name:
        query["name"] = args.name
    if args.units:
        query["units"] = args.units
    with openstack.connect(cloud=args.cloud) as conn:
        for row in conn.network.vpn_ipsec_policies(**query):
            print(json.dumps(row.to_dict(), ensure_ascii=False))


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
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies"
)

func run(ctx context.Context, cloud, name, units string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
    if err != nil { return err }
    service, err := conn.NetworkV2(ctx)
    if err != nil { return err }
    var options []ipsecpolicies.RecordListOption
    if name != "" { options = append(options, ipsecpolicies.WithRecordListQuery("name", name)) }
    if units != "" { options = append(options, ipsecpolicies.WithRecordListQuery("units", units)) }
    for record, err := range service.VPNIPsecPolicies.ListRecords(ctx, options...) {
        if err != nil { return err }
        if err := json.NewEncoder(os.Stdout).Encode(record.Resource.Body); err != nil { return err }
    }
    return nil
}

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    name := flag.String("name", "", "server-side policy name filter")
    units := flag.String("units", "", "local top-level units Body filter")
    timeout := flag.Duration("timeout", time.Minute, "whole operation deadline")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := run(ctx, *cloud, *name, *units); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
}
```

```sh
python main.py --cloud devstack --name policy
go run main.go --cloud devstack --name policy
```
