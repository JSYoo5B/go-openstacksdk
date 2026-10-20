# Python Quota Class

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:block_storage/v3/get_quota_class_set`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.API.GetQuotaClassRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaClassGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaClassName`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaClassGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaClassGetMicroversion`

- GET default/custom class names project the seven declared quota integer fields with missing/null distinction and preserve actual unknown Wire/body/header/status separately. Selected version remains owned and no version discovery is added.
  Tests: `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassGetSevenDescriptorsAndActualWire`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassGetMissingNullAndCopiedProof`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassOwnedOptionsAndNoVersionDiscovery`.
- Owned options/callback/source/context guards preserve selected routing and do not allow malformed parent or bound controls. Physical read/Close/cancel errors retain actual404 evidence without replay or logical missing suppression.
  Tests: `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassOwnedOptionsAndNoVersionDiscovery`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassPreflightIDsAndBoundConstructorControls`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassCallbacksSourceAndCancellationStayTerminal`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassPhysicalFaultKeeps404AndAllCauses`.

Python/Go differences:

- Go takes a scalar string class name in a fixed /os-quota-class-sets/{className} route rather than mutable Python Resource input. Only omitted GET name defaults to default; explicit empty names are rejected and other names are sent for server validation. Safe escaped routing, finite UTF-8 JSON and independent declared Resource/Wire/Envelope/header/status snapshots replace arbitrary runtime objects and URI formatter overrides.
- Raw update attributes are not coerced on the wire; only the declared returned Resource view applies Source integer/default/null behavior, including bool remaining bool and non-digit strings becoming0. Unknown Body attributes are dropped from the view while actual Wire retains unknown response JSON. Descriptor projection errors are reported without claiming a successful wire operation or Python runtime object identity.
- New scalar-name update seeds id but never commits dirty id; duplicate attrs id is rejected, and no recognized changes produces no HTTP/Discovery or fabricated receipt. Existing mutable Resource equality/recommit, optional response cache and generic adapter/session/runtime SDK-R1/C1/S1 remain whole-SDK goals. Header/version options are explicit Go conveniences and caller custom transport cache bypass is not claimed.
- Cinder QuotaClassSet has no class maximum; the API retains selected version without adding discovery. Nova class ceiling2.56 is not applied to Cinder, and no automatic Connection location is synthesized.

### `python:block_storage/v3/update_quota_class_set`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.API.UpdateQuotaClassRecord`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaClassUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaClassUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaClassUpdateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets.WithQuotaClassUpdateMicroversion`

- PUT transmits raw dirty quota_class_set attrs and explicit null, overlays actual response on class id seed and casts only the returned declared view. No attrs or unknown-only fields yields no HTTP, while duplicate id/constructor controls fail before requests.
  Tests: `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassUpdateRawDirtyAttrsAndSeedProjection`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassUpdateNoopUnknownAndExplicitNull`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassPreflightIDsAndBoundConstructorControls`.
- Owned options/callback/source/context guards preserve selected routing and do not allow malformed parent or bound controls. Physical read/Close/cancel errors retain actual404 evidence without replay or logical missing suppression.
  Tests: `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassOwnedOptionsAndNoVersionDiscovery`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassPreflightIDsAndBoundConstructorControls`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassCallbacksSourceAndCancellationStayTerminal`, `blockstorage/v3/quotasets/python_quota_class_test.go:TestPythonCinderQuotaClassPhysicalFaultKeeps404AndAllCauses`.

Python/Go differences:

- Go takes a scalar string class name in a fixed /os-quota-class-sets/{className} route rather than mutable Python Resource input. Only omitted GET name defaults to default; explicit empty names are rejected and other names are sent for server validation. Safe escaped routing, finite UTF-8 JSON and independent declared Resource/Wire/Envelope/header/status snapshots replace arbitrary runtime objects and URI formatter overrides.
- Raw update attributes are not coerced on the wire; only the declared returned Resource view applies Source integer/default/null behavior, including bool remaining bool and non-digit strings becoming0. Unknown Body attributes are dropped from the view while actual Wire retains unknown response JSON. Descriptor projection errors are reported without claiming a successful wire operation or Python runtime object identity.
- New scalar-name update seeds id but never commits dirty id; duplicate attrs id is rejected, and no recognized changes produces no HTTP/Discovery or fabricated receipt. Existing mutable Resource equality/recommit, optional response cache and generic adapter/session/runtime SDK-R1/C1/S1 remain whole-SDK goals. Header/version options are explicit Go conveniences and caller custom transport cache bypass is not claimed.
- Cinder QuotaClassSet has no class maximum; the API retains selected version without adding discovery. Nova class ceiling2.56 is not applied to Cinder, and no automatic Connection location is synthesized.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--action", choices=("get", "update"), required=True)
parser.add_argument("--name", default="default")
parser.add_argument("--attrs", default='{"volumes":0}')
parser.add_argument("--microversion", default="")
args = parser.parse_args()
config = {"cloud": args.cloud}
if args.microversion:
    config["block_storage_api_version"] = args.microversion
with openstack.connect(**config) as conn:
    if args.action == "get":
        row = conn.block_storage.get_quota_class_set(args.name)
    else:
        row = conn.block_storage.update_quota_class_set(args.name, **json.loads(args.attrs))
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
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/quotasets"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    action := flag.String("action", "", "get or update")
    name := flag.String("name", "default", "quota class name")
    attrs := flag.String("attrs", `{"volumes":0}`, "update JSON attributes")
    version := flag.String("microversion", "", "explicit service version, empty uses configured/default policy")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    options := []sdk.ConnectionOption{sdk.WithCloud(*cloud)}
    if *version != "" { options = append(options, sdk.WithMicroversion(sdk.BlockStorage, *version)) }
    conn, err := sdk.Connect(ctx, options...)
    if err != nil { log.Fatal(err) }
    service, err := conn.BlockStorageV3(ctx)
    if err != nil { log.Fatal(err) }
    api := quotasets.New(service.RawClient())
    var row *quotasets.QuotaClassRecord
    switch *action {
    case "get":
        row, err = api.GetQuotaClassRecord(ctx, quotasets.WithQuotaClassName(*name))
    case "update":
        var fields map[string]json.RawMessage
        if err := json.Unmarshal([]byte(*attrs), &fields); err != nil { log.Fatal(err) }
        if fields == nil { log.Fatal("attrs must be a JSON object") }
        row, err = api.UpdateQuotaClassRecord(ctx, *name,
            quotasets.WithQuotaClassUpdateOpts(quotasets.QuotaClassUpdateOpts{Attributes: fields}))
    default:
        log.Fatal("action must be get or update")
    }
    if err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(row.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Install](../../../docs/install.md)
