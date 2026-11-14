# Flavor Cloud

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(compute_api_version="2.55")
all_flavors = conn.list_flavors()  # get_extra=False
matched = conn.search_flavors("m1.*", filters={"disk": 20})
found = conn.get_flavor("m1.small")  # get_extra=True, ignore_missing=True
for flavor in matched:
    print(flavor.to_dict())
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

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud name")
    pattern := flag.String("pattern", "*", "flavor name or ID glob")
    identity := flag.String("flavor", "", "optional flavor ID or name to get")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *pattern, *identity); err != nil {
        log.Fatal(err)
    }
}

func recordValue(record *flavors.FlavorRecord) any {
    if record == nil { return nil }
    var enrichment any
    if child := record.Enrichment; child != nil {
        enrichment = map[string]any{
            "resource": child.Resource, "wire": child.Wire,
            "extra_specs": child.ExtraSpecs, "envelope": string(child.Envelope),
            "header": child.Header, "status_code": child.StatusCode,
        }
    }
    return map[string]any{
        "resource": record.Resource, "wire": record.Wire,
        "envelope": string(record.Envelope), "header": record.Header,
        "status_code": record.StatusCode, "enrichment": enrichment,
    }
}

func printQuery(operation string, result *compute.FlavorQueryResult) error {
    if result == nil { return nil }
    inventory := make([]any, len(result.Inventory))
    for i, record := range result.Inventory { inventory[i] = recordValue(record) }
    data, err := json.MarshalIndent(map[string]any{
        "operation": operation, "value": result.Value,
        "matched_count": len(result.Flavors), "inventory_count": len(result.Inventory),
        "inventory": inventory,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    return nil
}

func run(ctx context.Context, cloud, pattern, identity string) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud), sdk.WithMicroversion(sdk.Compute, "2.55"))
    if err != nil { return err }
    service, err := conn.Compute(ctx)
    if err != nil { return err }
    all, err := service.AllFlavors(ctx)
    if printErr := printQuery("all", all); printErr != nil { return errors.Join(err, printErr) }
    if err != nil { return fmt.Errorf("all flavors: %w", err) }
    matched, err := service.SearchFlavors(ctx, pattern)
    if printErr := printQuery("search", matched); printErr != nil { return errors.Join(err, printErr) }
    if err != nil { return fmt.Errorf("search flavors: %w", err) }
    if identity == "" { return nil }
    found, err := service.GetFlavor(ctx, identity)
    if found != nil {
        data, printErr := json.MarshalIndent(map[string]any{"operation": "get", "record": recordValue(found)}, "", "  ")
        if printErr != nil { return errors.Join(err, printErr) }
        fmt.Println(string(data))
    }
    if err != nil { return fmt.Errorf("get flavor: %w", err) }
    if found == nil { fmt.Println("flavor not found") }
    return nil
}
```

## References

- [Flavor Records](flavor-records.md)
- [owned List](flavor-records.md)
- [owned Flavor](flavor-records.md)
- [Cloud list_flavors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L278-L289)
- [search_flavors](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L165-L181)
- [get_flavor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L524-L561)
- [_filter_list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_utils.py#L43-L145)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
