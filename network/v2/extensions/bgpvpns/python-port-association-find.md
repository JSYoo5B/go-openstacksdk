# Python Port Association Find

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/find_bgpvpn_port_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.GetPortAssociation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.ListPortAssociations`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.FindPortAssociationRecord`

- GetPortAssociation returns the 404 error for a missing ID and ListPortAssociations sends the unfiltered association list, the two requests Python's find sends.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNPortAssociationFindHasNoHelper`.
- Owned parent-scoped GET-first find reuses Source fetch controls, declared Body defaults/casts, local fallback filters, complete pagination and late duplicate detection; raw HTTP proof and terminal faults remain distinct.
  Tests: `network/v2/extensions/bgpvpns/python_port_association_find_test.go:TestPythonPortAssociationFindMemberProjectionAndOwnership`, `network/v2/extensions/bgpvpns/python_port_association_find_test.go:TestPythonPortAssociationFindFallbackLocalFiltersAndLateDuplicates`, `network/v2/extensions/bgpvpns/python_port_association_find_test.go:TestPythonPortAssociationFindMissingAndStrictJSONIdentity`, `network/v2/extensions/bgpvpns/python_port_association_find_test.go:TestPythonPortAssociationFindRawFetchControlsAndHeadersPhase`, `network/v2/extensions/bgpvpns/python_port_association_find_test.go:TestPythonPortAssociationFindOptionsGuardAndPreflight`, `network/v2/extensions/bgpvpns/python_port_association_find_test.go:TestPythonPortAssociationFindPhysicalFaultNeverFallback`.

Python/Go differences:

- Go takes fixed parent and lookup string IDs plus finite UTF-8 JSON options; arbitrary Python Resource/class/session/connection/location runtime and shared cache are not modeled. Raw URI formatter supports only common literal/%(id)s/%% paths, not dynamic %(bgpvpn_id)s; the parent argument supplies that URI value.
- SDK-R1/C1/S1, real-cloud compatibility and release validation remain; local tests and source review do not establish full SDK parity.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--vpn", required=True)
parser.add_argument("--association", required=True)
parser.add_argument("--port", required=True)
args = parser.parse_args()
with openstack.connect() as conn:
    row = conn.network.find_bgpvpn_port_association(
        args.association, args.vpn, ignore_missing=False, port_id=args.port)
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
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns"
)

func main() {
    vpn := flag.String("vpn", "", "parent BGPVPN ID")
    association := flag.String("association", "", "association ID or name")
    port := flag.String("port", "", "port ID filter")
    flag.Parse()
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.Network(ctx)
    if err != nil { log.Fatal(err) }
    row, err := service.API.BGPVPNs.FindPortAssociationRecord(ctx, *vpn, *association,
        bgpvpns.WithPortAssociationFindIgnoreMissing(false),
        bgpvpns.WithPortAssociationFindQuery("port_id", *port))
    if err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(row.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Proxy.find_bgpvpn_port_association](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [Bgpvpn Port Association](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgpvpn_port_association.py)
