# Flavor Extra Specs

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/fetch_flavor_extra_specs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.FetchExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorExtraSpecsOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorExtraSpecsMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorExtraSpecsHeader`

- FetchExtraSpecs always performs one logical bodyless GET at the captured prefixed flavors/{id}/os-extra_specs route, with no flavor lookup or inline extra_specs shortcut. ID-only input seeds the requested ID; existing native or raw flavor fields are copied before options, keep their original fields, and only the Resource extra_specs value is replaced. Native Swap survives its json-excluded native field and native is_public/ephemeral wire aliases become source view names. An explicit route ID does not rewrite a supplied passive Resource ID; absent route ID uses the existing truthy string id/name/original_name identity.
  Tests: `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorFetchExtraSpecsAlwaysGETRawValuesAndOwnedExistingInputs`.
- A missing response extra_specs becomes an empty object. Record.ExtraSpecs and independently owned actual whole-root Wire preserve present object, null, array, scalar, string, false and large precise JSON values. Resource.extra_specs reproduces Flavor's dict descriptor: objects and null remain, while nonnull nonobjects project to an empty object. Other actual response fields do not replace the existing flavor. Resource retains owned input metadata while record/Wire headers and status plus Envelope preserve the fresh physical response; input, Resource, Wire and raw result bytes do not alias.
  Tests: `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorFetchExtraSpecsAlwaysGETRawValuesAndOwnedExistingInputs`.
- A selected microversion is retained without an inline-specs or 2.61 gate. With no selection, finite shared Nova discovery selects the advertised maximum capped at 2.61 or a versionless request for unavailable bounds; an explicit empty override bypasses discovery and issues a versionless private GET. Source token and headers remain live/owned, options apply once and mutable typed pointers/header inputs are snapshotted before later option mutations, while the original service client's version and route remain unchanged.
  Tests: `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorFetchExtraSpecsAlwaysGETRawValuesAndOwnedExistingInputs`.
- Owned reads admit final200..399 but require a nonnull UTF-8 JSON object root; malformed/empty accepted JSON and parsed nonobject roots preserve the accepted receipt rather than being tolerated. HTTP403 retains original native status, expected codes, method, fixed URL and response body/header. Accepted read/Close nested404 and post-fetch source failures remain terminal ResponseErrors with the actual receipt and original cause, never a missing result, follow-up request or accepted-response replay. Invalid concrete input, nil options, callback cancellation, source drift, outer guard and spoofed auth header stop HTTP.
  Tests: `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorFetchExtraSpecsPreflightAndAcceptedTerminalReceipts`.

Python/Go differences:

- Python accepts a string or connected Flavor and mutates/returns the same resource. Go accepts a concrete ID/native Flavor/raw Resource carrier, returns an independently owned updated input view plus a raw specs value and physical receipt, and never mutates the supplied input. Raw existing fields and metadata are preserved rather than recreating every Flavor descriptor default or injecting computed Connection location.
- Explicit ID with an existing input is an additional route-only Go selection. With no explicit ID, only a validated safe UTF-8 string id/name/original_name can become the executable path; Python's arbitrary dynamic truthy identity and custom base_path/session controls are not exposed. Passive original IDs remain in the input view, even when a different validated string selects the request route.
- The Resource extra_specs object/null/empty projection matches the pinned type=dict descriptor; the additive ExtraSpecs field deliberately exposes the original nonobject value that the Python property converts to {}. Actual Wire and Envelope never contain invented defaults or input fields. ID-only Resource is a finite id plus extra_specs view, not a fully connected default-hydrated Python Flavor.
- Nil operation microversion uses a captured selected version or the existing finite guarded Nova discovery capped at Flavor2.61. Explicit empty is an additional versionless Go override. Full Keystoneauth discovery ordering/cache/session facilities retain independent SDK scope; discovery physical/source/JSON failures are explicit Go errors rather than blanket source DiscoveryFailure swallowing.
- Final200..399 and strict UTF-8 object parsing are the mapped Go transport boundary. Python accepts status<400 through its session and response.json text decoder; informational/session redirect/charset combinations are not asserted as identical. Physical read/Close/source/context faults retain typed receipt evidence. Native ListExtraSpecs remains a separate strict200 string-map API.
- This review maps the directly named unconditional fetch operation only. find_flavor/get_flavor/flavors optional inline enrichment, create/update/delete individual extra-specs, mutable Resource lifecycle, computed location and other service declarations keep independent tracking; none is incidentally promoted by this read.

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
        options = append(options, flavors.WithFlavorExtraSpecsMicroversion(version))
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
