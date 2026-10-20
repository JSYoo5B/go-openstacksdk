# Python Source Mapping

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/create_mapping`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.API.CreateSourceMappingRecord`

- CreateMapping sends PUT OS-FEDERATION/mappings/{id} with id and schema_version in the mapping envelope via WithCreateMappingField, as Python does, but the decoded Mapping drops schema_version and unknown rule keys.
  Tests: `identity/v3/federation/python_parity_test.go:TestPythonMappingRequestsAndDecodeGaps`.
- Owned member PUT preserves arbitrary raw rule keys/schema writes, required wire ID and ignored constructor name and nullable/list/string Resource views; actual response overrides seed and retains independent Wire evidence.
  Tests: `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingCreatePreservesRawRulesAndNullableDescriptorView`, `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingCreateIgnoresNameDefaultsAndStatusPolicy`, `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingOwnershipAndCallbackGuards`, `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingPreflightCollisionsAndInvalidBodies`.

Python/Go differences:

- The additive SourceMappingRecord API preserves the existing native and MappingRecord APIs. It owns finite JSON attributes and safe string route IDs; arbitrary Python objects and mutable Resource dirty-baseline/recommit state are represented by a fresh Go operation instead.
- Raw mutation base_path overrides are explicitly unsupported; headers and microversion options are Go conveniences.
- Python find_mapping deprecation warnings are not emitted. full Requests codec, runtime/session, SDK-R1/C1/S1 and live-cloud/release contracts remain tracked globally.

### `python:identity/v3/find_mapping`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.API.FindSourceMappingRecord`

- GET-first lookup falls back only on clean 400/403/404, scans all pages without a name query, handles missing/strict/duplicate results and preserves physical terminal failures.
  Tests: `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingFindGETFirstFallbackPaginationAndDuplicate`, `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingPhysicalFaultsPreserveProofAndStopFallback`, `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingOwnershipAndCallbackGuards`.

Python/Go differences:

- The additive SourceMappingRecord API preserves the existing native and MappingRecord APIs. It owns finite JSON attributes and safe string route IDs; arbitrary Python objects and mutable Resource dirty-baseline/recommit state are represented by a fresh Go operation instead.
- Raw mutation base_path overrides are explicitly unsupported; headers and microversion options are Go conveniences.
- Python find_mapping deprecation warnings are not emitted. full Requests codec, runtime/session, SDK-R1/C1/S1 and live-cloud/release contracts remain tracked globally.

### `python:identity/v3/update_mapping`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/federation.API.UpdateSourceMappingRecord`

- UpdateMapping sends PATCH OS-FEDERATION/mappings/{id} and always includes rules, as null when empty, so a schema_version-only update differs from Python's body.
  Tests: `identity/v3/federation/python_parity_test.go:TestPythonMappingRequestsAndDecodeGaps`.
- Fresh scalar-ID PATCH includes only supplied dirty raw fields; omitted rules are absent, explicit null is sent, raw name does not change the route, id collision is rejected and empty/unknown-only updates do not send HTTP.
  Tests: `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingUpdateDirtyNoopNullAndIgnoredName`, `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingPreflightCollisionsAndInvalidBodies`, `identity/v3/federation/python_source_mapping_test.go:TestSourceMappingOwnershipAndCallbackGuards`.

Python/Go differences:

- The additive SourceMappingRecord API preserves the existing native and MappingRecord APIs. It owns finite JSON attributes and safe string route IDs; arbitrary Python objects and mutable Resource dirty-baseline/recommit state are represented by a fresh Go operation instead.
- Raw mutation base_path overrides are explicitly unsupported; headers and microversion options are Go conveniences.
- Python find_mapping deprecation warnings are not emitted. full Requests codec, runtime/session, SDK-R1/C1/S1 and live-cloud/release contracts remain tracked globally.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("create", "find", "update"), required=True)
parser.add_argument("--id", default="example-mapping")
args = parser.parse_args()
rules = [{"local": [{"user": {"name": "{0}", "type": "ephemeral"}}],
          "remote": [{"type": "REMOTE_USER", "any_one_of": ["alice"]}]}]
with openstack.connect(cloud=args.cloud) as conn:
    if args.mode == "create":
        row = conn.identity.create_mapping(id=args.id, rules=rules,
                                           schema_version="2.0")
    elif args.mode == "update":
        row = conn.identity.update_mapping(args.id, schema_version="2.0")
    else:
        row = conn.identity.find_mapping(args.id)
    print(json.dumps(None if row is None else row.to_dict(), indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/federation"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    mode := flag.String("mode", "", "create, find or update")
    id := flag.String("id", "example-mapping", "Mapping ID")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    var row *federation.SourceMappingRecord
    switch *mode {
    case "create":
        rules := []any{map[string]any{
            "local": []any{map[string]any{"user": map[string]any{"name": "{0}", "type": "ephemeral"}}},
            "remote": []any{map[string]any{"type": "REMOTE_USER", "any_one_of": []string{"alice"}}},
        }}
        row, err = service.Federation.CreateSourceMappingRecord(ctx,
            federation.WithSourceMappingAttribute("id", *id),
            federation.WithSourceMappingAttribute("rules", rules),
            federation.WithSourceMappingAttribute("schema_version", "2.0"))
    case "update":
        row, err = service.Federation.UpdateSourceMappingRecord(ctx, *id,
            federation.WithSourceMappingAttribute("schema_version", "2.0"))
    case "find":
        row, err = service.Federation.FindSourceMappingRecord(ctx, *id)
    default:
        log.Fatal("mode must be create, find or update")
    }
    if err != nil { log.Fatal(err) }
    var value any
    if row != nil { value = row.Resource }
    encoded, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(encoded))
}
```

## References

- [Install](../../../docs/install.md)
