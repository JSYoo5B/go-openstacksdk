# Availabilityzones

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
zones = list(conn.network.availability_zones(name="az-a"))
for zone in zones:
    print(zone.name, zone.resource, zone.state)
```

```go
package main

import (
    "context"
    "fmt"
    "log"

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/availabilityzones"
)

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.NetworkV2(ctx)
    if err != nil { log.Fatal(err) }
    api := availabilityzones.New(service.RawClient())
    zones, err := api.AllRecords(ctx, availabilityzones.WithProxyQuery("name", "az-a"))
    if err != nil { log.Fatal(err) }
    for _, zone := range zones {
        fmt.Println(string(zone.Resource.Body["name"]), string(zone.Resource.Body["resource"]), string(zone.Resource.Body["state"]))
    }
}
```

## References

- [availability_zones proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [Availability Zone](https://github.com/openstack/neutron/blob/27.0.0/neutron/conf/policies/availability_zone.py)
- [Records Test](records_test.go)
