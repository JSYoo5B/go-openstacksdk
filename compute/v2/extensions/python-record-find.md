# Python Record Find

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/find_extension`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/extensions.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/extensions.API.FindRecord`

- Get sends GET extensions/{alias} and returns the 404 for a name without listing extensions.
  Tests: `compute/v2/extensions/python_parity_test.go:TestPythonExtensionListAndGetHaveNoFindFallback`.
- Concrete Source-owned compute-role projection, library-owned callback snapshots and selected/versionless policy retain original client and actual response evidence. Source-aware lookup/list parsing shares guarded REST, descriptor MRO, Python-loaded equality and paging rather than new transport wrappers.
  Tests: `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindMemberOwnsSourceMROWithoutDiscovery`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindCleanFallbackAliasNameMissingAndAmbiguity`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindStrictMissingKeepsExistingNativeAPIs`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindOwnedKwargsAndGetOnlyVersion`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindLaterPageAndSourcePrimaryPresence`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindOptionSourceGuardAndCancellation`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindPhysicalRejectionsNeverFallbackOrRunHooks`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindCleanAuthenticationRenewalKeepsMember`.
- GET-first literal extensions path suppresses only clean400403404; complete fallback pages match exact alias-derived id or name, detect duplicate and default missingnil/strictErrNotFound. Compute Extension unknownBody=false includes namespace, links=list_type=dict, updated alias and untyped fields; maxNone yields no default discovery and explicit override is GET-only. Physical401 pointer native public receipt and all causes survive without fallback/retry/auth/backoff.
  Tests: `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindMemberOwnsSourceMROWithoutDiscovery`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindCleanFallbackAliasNameMissingAndAmbiguity`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindStrictMissingKeepsExistingNativeAPIs`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindOwnedKwargsAndGetOnlyVersion`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindLaterPageAndSourcePrimaryPresence`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindOptionSourceGuardAndCancellation`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindPhysicalRejectionsNeverFallbackOrRunHooks`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindCleanAuthenticationRenewalKeepsMember`.

Python/Go differences:

- Go concrete Source records separate Resource default/projection from complete actual Wire/page/header/status; SDK-owned location is null and optional HTTP cache/Connection/dirty-state remain generic SDK-R1/C1. Actual cloud permissions/acceptance are not established by HTTP fixtures.
- Direct Source find_extension signature declares only identity/ignore_missing; Go raw Query/Header/Microversion/ListBasePath options are explicit convenience corresponding to generic Resource.find, not implicit extra kwargs on the Python named method.
- Official Nova31.0.0 extension list/show policy is project-scope RULE_ANY, not admin-only; deployment policies can differ. Role-specific version headers, safe literal IDs, source/context guards and owned callback maps improve Go safety.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="devstack")
parser.add_argument("--extension", required=True)
parser.add_argument("--strict", action="store_true")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    value = conn.compute.find_extension(args.extension, ignore_missing=not args.strict)
    print(json.dumps(None if value is None else value.to_dict(), indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/extensions"
)

func main() {
    cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
    identity := flag.String("extension", "", "literal alias or exact name")
    strict := flag.Bool("strict", false, "return missing errors")
    flag.Parse()
    if *identity == "" { log.Fatal("-extension is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    compute, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    value, err := compute.Extensions.FindRecord(ctx, *identity,
        extensions.WithRecordFindIgnoreMissing(!*strict))
    if err != nil { log.Fatal(err) }
    var output any
    if value != nil { output = value.Resource }
    data, err := json.MarshalIndent(output, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [find_extension](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L108)
- [Extension](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/extension.py)
- [Install](../../../docs/install.md)
- [extension policy](https://github.com/openstack/nova/blob/31.0.0/nova/policies/extensions.py)
