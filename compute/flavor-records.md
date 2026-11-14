# Flavor Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(compute_api_version="2.55")
for flavor in conn.compute.flavors(min_ram=1024, disk=20):
    print(flavor.to_dict())

found = conn.compute.find_flavor("m1.small", ignore_missing=True,
                                 get_extra_specs=True)
if found is not None:
    print(found.to_dict())
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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    identity := flag.String("flavor", "", "optional flavor ID or name to find after listing")
    extra := flag.Bool("extra", false, "conditionally fetch missing or empty extra specs")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *identity, *extra); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, cloud, identity string, extra bool) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud), sdk.WithMicroversion(sdk.Compute, "2.55"))
    if err != nil { return err }
    service, err := conn.Compute(ctx)
    if err != nil { return err }
    printRecord := func(operation string, record *flavors.FlavorRecord) error {
        if record == nil { return nil }
        var enrichment any
        if child := record.Enrichment; child != nil {
            enrichment = map[string]any{
                "resource": child.Resource, "wire": child.Wire,
                "extra_specs": child.ExtraSpecs, "envelope": string(child.Envelope),
                "header": child.Header, "status_code": child.StatusCode,
            }
        }
        data, err := json.MarshalIndent(map[string]any{
            "operation": operation, "resource": record.Resource, "wire": record.Wire,
            "envelope": string(record.Envelope), "header": record.Header,
            "status_code": record.StatusCode, "enrichment": enrichment,
        }, "", "  ")
        if err != nil { return err }
        fmt.Println(string(data))
        return nil
    }
    for record, err := range service.ListFlavors(ctx,
        request.MaxItems(20),
        request.Custom(func(c *request.Config[flavors.FlavorListOpts]) error { c.Options.GetExtraSpecs = extra; return nil })) {
        if printErr := printRecord("list", record); printErr != nil { return errors.Join(err, printErr) }
        if err != nil { return fmt.Errorf("list flavors: %w", err) }
    }
    if identity == "" { return nil }
    record, err := service.FindFlavor(ctx, identity, request.Custom(func(c *request.Config[flavors.FlavorFindOpts]) error { c.Options.GetExtraSpecs = extra; return nil }))
    if printErr := printRecord("find", record); printErr != nil { return errors.Join(err, printErr) }
    if err != nil { return fmt.Errorf("find flavor: %w", err) }
    if record == nil { fmt.Println("flavor not found") }
    return nil
}
```

## References

- [FetchExtraSpecs](flavor-extra-specs.md)
- [Console Selection](console-selection.md)
- [Flavor Property And Native](flavor-property-and-native.md)
- [Proxy flavors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L269-L296)
- [Proxy find_flavor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L165-L199)
- [Flavor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/flavor.py#L43-L138)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
