# Availabilityzones

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/availability_zones`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/availabilityzones.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/availabilityzones.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/availabilityzones.WithProxyQuery`

- Proxy accepts but drops every query kwarg without marshaling, URL filters, Body predicates or JMESPath evaluation.
  Tests: `network/v2/availabilityzones/records_test.go:TestAvailabilityZonesProxyDiscardsAllQueryKwargsWithoutMarshalOrFiltering`.
- Source nullable declarations, unknown/raw extensions and independent actual page metadata are preserved.
  Tests: `network/v2/availabilityzones/records_test.go:TestAvailabilityZonesDeclaredDefaultsUnknownFieldsAndActualReceiptOwnBytes`.
- Lazy iterator stops before next request; eager AllRecords discards partial pages on error.
  Tests: `network/v2/availabilityzones/records_test.go:TestAvailabilityZonesPagingAndAllRecordsDiscardPartialErrors`.
- Selected source/context callbacks are guarded and runtime shapes fail atomically with actual response proof.
  Tests: `network/v2/availabilityzones/records_test.go:TestAvailabilityZonesCallbacksCannotHideSourceChangeOrCancellation`.
- Clean401 reauthentication retains project-independent zone route and latest shared token.
  Tests: `network/v2/availabilityzones/records_test.go:TestAvailabilityZonesNative401KeepsActualOriginalErrorAndCleanReauth`.

Python/Go differences:

- Go returns detached Source Resource/Wire with actual response metadata and uses context plus owned callback guards; optional Python Adapter cache/session and object/dirty lifecycle remain tracked whole-project SDK-R1/C1/S1.
- Common Neutron/REST engine owns strict UTF8, continuation origin and request/header/version policies; no live-cloud support is claimed.

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
