# Flavor Extra Specs

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import sys
import openstack
from openstack.compute.v2.flavor import Flavor

conn = openstack.connect()
existing = Flavor.existing(id=sys.argv[1], extra_specs={"inline": "old"})
updated = conn.compute.fetch_flavor_extra_specs(existing)
print(updated.extra_specs)
print(updated.to_dict())
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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors"
)

func main() {
    id := flag.String("flavor-id", "", "flavor ID to fetch extra specs for")
    version := flag.String("microversion", "", "optional operation version; explicit empty is versionless")
    flag.Parse()
    versionSet := false
    flag.Visit(func(f *flag.Flag) {
        if f.Name == "microversion" {
            versionSet = true
        }
    })
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *id, *version, versionSet); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, id, version string, versionSet bool) error {
    if id == "" {
        return fmt.Errorf("-flavor-id is required")
    }
    conn, err := sdk.Connect(ctx)
    if err != nil {
        return err
    }
    service, err := conn.Compute(ctx)
    if err != nil {
        return err
    }
    options := []flavors.FlavorExtraSpecsOption{}
    if versionSet {
        options = append(options, request.Microversion(version))
    }
    record, fetchErr := service.API.Flavors.FetchExtraSpecs(ctx,
        flavors.FlavorExtraSpecsRequest{ID: id}, options...)
    if record != nil {
        data, printErr := json.MarshalIndent(map[string]any{
            "resource": record.Resource,
            "wire": record.Wire,
            "extra_specs": record.ExtraSpecs,
            "status_code": record.StatusCode,
            "header": record.Header,
        }, "", "  ")
        if printErr != nil {
            return errors.Join(fetchErr, printErr)
        }
        fmt.Println(string(data))
        fmt.Printf("response body: %q\n", record.Envelope)
    }
    return fetchErr
}
```

## References

- [Console Selection](console-selection.md)
- [User Read APIs](user-read-apis.md)
- [`fetch_flavor_extra_specs`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L334)
- [`Flavor.fetch_extra_specs`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/flavor.py#L186)
- [`fields._convert_type`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L86)
- [`_BaseComponent.__get__`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/fields.py#L208)
- [`ListExtraSpecs`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/flavors/requests.go#L277)
- [`Extract`](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/flavors/results.go#L220)
- [Compute Flavor Extra Specs Fetch Test](../api/compute_flavor_extra_specs_fetch_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Implementation Plan](../docs/implementation-plan.md)
