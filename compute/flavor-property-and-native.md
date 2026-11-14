# Flavor Property And Native

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
flavor = conn.get_flavor_by_id("7", get_extra=False)
value = conn.compute.get_flavor_extra_specs_property(flavor, "hw:cpu_policy")
print(flavor.id, value)
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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
)

func main() {
    id := flag.String("flavor-id", "", "flavor ID")
    property := flag.String("property", "hw:cpu_policy", "single extra-spec property")
    extra := flag.Bool("extra", false, "conditionally fetch missing inline flavor specs")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *id, *property, *extra); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, id, property string, extra bool) error {
    if id == "" {
        return fmt.Errorf("-flavor-id is required")
    }
    conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.55"))
    if err != nil {
        return err
    }
    service, err := conn.Compute(ctx)
    if err != nil {
        return err
    }
    cloudFlavor, err := service.GetFlavorByID(ctx, id,
        func(c *request.Config[compute.FlavorByIDOpts]) error { c.Options.GetExtraSpecs = extra; return nil })
    if err != nil {
        return fmt.Errorf("cloud flavor by ID: %w", err)
    }
    api := service.API.Flavors
    record, propertyErr := api.GetExtraSpecsProperty(ctx,
        flavors.FlavorExtraSpecsRequest{ID: id}, property)
    if record != nil {
        data, printErr := json.MarshalIndent(map[string]any{
            "value": record.Value,
            "present": record.Present,
            "wire": record.Wire,
            "response_body": string(record.Envelope),
            "header": record.Header,
            "status_code": record.StatusCode,
        }, "", "  ")
        if printErr != nil {
            return errors.Join(propertyErr, printErr)
        }
        fmt.Println(string(data))
    }
    if propertyErr != nil {
        return fmt.Errorf("single property: %w", propertyErr)
    }
    nativeFlavor, err := api.Get(ctx, id)
    if err != nil {
        return fmt.Errorf("native flavor: %w", err)
    }
    nativeProperty, err := api.GetExtraSpec(ctx, id, property)
    if err != nil {
        return fmt.Errorf("native property string map: %w", err)
    }
    listed := make([]*flavors.Flavor, 0, 20)
    for flavor, listErr := range api.ListDetail(ctx,
        flavors.WithListDetailOptions(flavors.ListOpts{Limit: 20})) {
        if listErr != nil {
            return fmt.Errorf("native detailed list: %w", listErr)
        }
        listed = append(listed, flavor)
        if len(listed) == 20 {
            break
        }
    }
    data, err := json.MarshalIndent(map[string]any{
        "cloud_flavor": cloudFlavor,
        "native_flavor": nativeFlavor,
        "native_property_map": nativeProperty,
        "native_detailed_flavors": listed,
    }, "", "  ")
    if err != nil {
        return err
    }
    fmt.Println(string(data))
    return nil
}
```

## References

- [Policy](https://docs.openstack.org/nova/latest/configuration/policy.html)
- [Flavor Extra Specs](flavor-extra-specs.md)
- [User Read APIs](user-read-apis.md)
- [Finding Identities](../docs/finding-identities.md)
