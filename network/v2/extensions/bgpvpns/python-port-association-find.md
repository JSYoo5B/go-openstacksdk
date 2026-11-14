# Python Port Association Find

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

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
        request.IgnoreMissing(false),
        request.Query("port_id", *port))
    if err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(row.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Proxy.find_bgpvpn_port_association](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [Bgpvpn Port Association](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgpvpn_port_association.py)
