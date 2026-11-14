# Availability Zones

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
for zone in conn.compute.availability_zones():
    print(zone.name, zone.state, zone.hosts)

print(conn.list_availability_zone_names())
print(conn.list_availability_zone_names(unavailable=True))
```

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "time"

    "github.com/JSYoo5B/go-openstacksdk/request"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
)

func main() {
    cloudName := flag.String("cloud", "dev", "clouds.yaml cloud name")
    unavailable := flag.Bool("unavailable", false, "include unavailable zone names")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloudName, *unavailable); err != nil {
        log.Fatal(err)
    }
}

func printJSON(value any) error {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloudName string, unavailable bool) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloudName))
    if err != nil { return err }
    service, err := conn.Compute(ctx)
    if err != nil { return err }

    records := make([]map[string]any, 0)
    for record, readErr := range service.ListAvailabilityZones(ctx) {
        if record != nil {
            records = append(records, map[string]any{
                "resource": record.Resource, "wire": record.Wire,
                "envelope": string(record.Envelope),
                "header": record.Header, "status_code": record.StatusCode,
            })
        }
        if readErr != nil {
            return errors.Join(readErr, printJSON(map[string]any{
                "operation": "zones", "partial_records": records,
            }))
        }
    }
    if err := printJSON(map[string]any{"operation": "zones", "records": records}); err != nil {
        return err
    }

    options := make([]compute.AvailabilityZoneNamesOption, 0)
    if unavailable { options = append(options, request.Custom(func(c *request.Config[compute.AvailabilityZoneNamesOpts]) error { c.Options.Unavailable = true; return nil })) }
    result, readErr := service.ListAvailabilityZoneNames(ctx, options...)
    if result != nil {
        suppressed := ""
        if result.SuppressedError != nil { suppressed = result.SuppressedError.Error() }
        printErr := printJSON(map[string]any{
            "operation": "names", "names": result.Names, "value": result.Value,
            "inventory_count": len(result.Inventory), "suppressed_error": suppressed,
        })
        if printErr != nil { return errors.Join(readErr, printErr) }
    }
    return readErr
}
```

## References

- [Policy](https://docs.openstack.org/nova/2024.1/configuration/policy.html)
- [Implementation Plan](../docs/implementation-plan.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Proxy availability_zones](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L1854-L1874)
- [AvailabilityZone descriptors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/availability_zone.py#L16-L36)
- [Cloud list_availability_zone_names](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L255-L276)
