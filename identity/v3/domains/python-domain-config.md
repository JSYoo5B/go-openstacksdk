# Python Domain Config

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/create_domain_config`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.CreateConfigRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigAttribute`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigIgnoreMissing`

- CreateConfigRecord PUTs the selected Domain singleton with config envelope, raw identity/ldap payloads including unknown nested settings and nullable values, then overlays the actual response and projects declared child Resource views without fabricating physical fields.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigCreateUsesSingletonPUTAndRawChildPayload`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigGetProjectsChildrenAndPreservesActualEvidence`.
- Options and selected source/route/header/version remain owned through callbacks; invalid parent/constructor/attribute usage and read-versus-mutation separation are rejected before HTTP. Physical faults and strict read rejections retain actual causes and never trigger fallback.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigOptionsSnapshotsAndRequestPolicy`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPreflightGuardsAndReadMutationSeparation`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPhysicalFaultsAndStrictReadErrorsNeverFallback`.

Python/Go differences:

- The SDK directly owns the DomainConfig singleton binding absent from Gophercloud v2.15.0. It takes a string parent Domain ID in the fixed /domains/{domain_id}/config path, never appending a config member ID or performing parent name lookup. Raw base_path overrides/formatters, Python Domain/Resource object inputs and constructor/runtime injection are deliberate exclusions of this Go profile.
- Mutable Python DomainConfigDriver/DomainConfigLDAP child Resources are represented as independent declared JSON snapshots with child inherited defaults; mapping unknowns are dropped from the Resource view but actual Wire/Envelope preserves them, nonmapping nonnull becomes {}, and null remains null. Parent location is SDK-owned null with no automatic Connection inference. The Go view is eagerly projected; malformed child input can fail before HTTP rather than when a Source lazy getter is later read.
- Only explicit raw JSON attributes are written; descriptor casts apply to the returned view, not transmitted nested payloads. Options/headers/version are copied, bulk attribute maps use deterministic sorted keys and typed header/microversion options are Go conveniences. SDK-R1/C1/S1 mutable Resource/cache/adapter/session goals and authenticated cloud/release verification remain separately tracked; caller custom transport cache bypass is not claimed.

### `python:identity/v3/delete_domain_config`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.DeleteConfigRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigAttribute`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigIgnoreMissing`

- DeleteConfigRecord performs bodyless singleton DELETE, defaults to ignoring only clean404 and offers strict missing behavior. Accepted deletion and ignored404 retain independent actual response receipt; Python None becomes a Go Deleted/Response carrier.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigDeleteMissingDefaultAndStrictPolicy`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigDeleteAcceptedResponseIsIndependentReceipt`.
- Options and selected source/route/header/version remain owned through callbacks; invalid parent/constructor/attribute usage and read-versus-mutation separation are rejected before HTTP. Physical faults and strict read rejections retain actual causes and never trigger fallback.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigOptionsSnapshotsAndRequestPolicy`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPreflightGuardsAndReadMutationSeparation`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPhysicalFaultsAndStrictReadErrorsNeverFallback`.

Python/Go differences:

- The SDK directly owns the DomainConfig singleton binding absent from Gophercloud v2.15.0. It takes a string parent Domain ID in the fixed /domains/{domain_id}/config path, never appending a config member ID or performing parent name lookup. Raw base_path overrides/formatters, Python Domain/Resource object inputs and constructor/runtime injection are deliberate exclusions of this Go profile.
- Mutable Python DomainConfigDriver/DomainConfigLDAP child Resources are represented as independent declared JSON snapshots with child inherited defaults; mapping unknowns are dropped from the Resource view but actual Wire/Envelope preserves them, nonmapping nonnull becomes {}, and null remains null. Parent location is SDK-owned null with no automatic Connection inference. The Go view is eagerly projected; malformed child input can fail before HTTP rather than when a Source lazy getter is later read.
- Only explicit raw JSON attributes are written; descriptor casts apply to the returned view, not transmitted nested payloads. Options/headers/version are copied, bulk attribute maps use deterministic sorted keys and typed header/microversion options are Go conveniences. SDK-R1/C1/S1 mutable Resource/cache/adapter/session goals and authenticated cloud/release verification remain separately tracked; caller custom transport cache bypass is not claimed.
- Python delete returns None; Go Deleted is true only on observed accepted deletion and false on ignored404. Deleted=false with an error does not prove nonexecution. Physical read/Close/cancel/source failures are never suppressed as logical absence.

### `python:identity/v3/get_domain_config`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.GetConfigRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigAttribute`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigIgnoreMissing`

- GetConfigRecord GETs the singleton without an extra resource ID, preserves absent/null/passive id and actual envelope/header/status, and projects declared child mapping/defaults independently from raw Wire. Null, nonmapping scalar and empty-object child values remain distinct.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigGetProjectsChildrenAndPreservesActualEvidence`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigGetNullScalarAndEmptyChildrenHaveDistinctViews`.
- Malformed nested child projection after an accepted response retains actual ResponseError proof; strict GET404 is not ignored or converted into fallback/missing success.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigMalformedNestedChildKeepsAcceptedResponse`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPhysicalFaultsAndStrictReadErrorsNeverFallback`.
- Options and selected source/route/header/version remain owned through callbacks; invalid parent/constructor/attribute usage and read-versus-mutation separation are rejected before HTTP. Physical faults and strict read rejections retain actual causes and never trigger fallback.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigOptionsSnapshotsAndRequestPolicy`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPreflightGuardsAndReadMutationSeparation`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPhysicalFaultsAndStrictReadErrorsNeverFallback`.

Python/Go differences:

- The SDK directly owns the DomainConfig singleton binding absent from Gophercloud v2.15.0. It takes a string parent Domain ID in the fixed /domains/{domain_id}/config path, never appending a config member ID or performing parent name lookup. Raw base_path overrides/formatters, Python Domain/Resource object inputs and constructor/runtime injection are deliberate exclusions of this Go profile.
- Mutable Python DomainConfigDriver/DomainConfigLDAP child Resources are represented as independent declared JSON snapshots with child inherited defaults; mapping unknowns are dropped from the Resource view but actual Wire/Envelope preserves them, nonmapping nonnull becomes {}, and null remains null. Parent location is SDK-owned null with no automatic Connection inference. The Go view is eagerly projected; malformed child input can fail before HTTP rather than when a Source lazy getter is later read.
- Only explicit raw JSON attributes are written; descriptor casts apply to the returned view, not transmitted nested payloads. Options/headers/version are copied, bulk attribute maps use deterministic sorted keys and typed header/microversion options are Go conveniences. SDK-R1/C1/S1 mutable Resource/cache/adapter/session goals and authenticated cloud/release verification remain separately tracked; caller custom transport cache bypass is not claimed.

### `python:identity/v3/update_domain_config`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.UpdateConfigRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigAttribute`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithDomainConfigIgnoreMissing`

- UpdateConfigRecord PATCHes raw explicit identity/ldap values and null in a config envelope, overlays actual response on seed and discards dirty id from wire while retaining the explicit seed id view. No recognized dirty fields yields no HTTP and no invented receipt.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigUpdatePATCHNullAndSeedOverlay`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigUpdateEmptyAndUnknownAttributesAvoidHTTP`.
- Options and selected source/route/header/version remain owned through callbacks; invalid parent/constructor/attribute usage and read-versus-mutation separation are rejected before HTTP. Physical faults and strict read rejections retain actual causes and never trigger fallback.
  Tests: `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigOptionsSnapshotsAndRequestPolicy`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPreflightGuardsAndReadMutationSeparation`, `identity/v3/domains/python_domain_config_test.go:TestPythonDomainConfigPhysicalFaultsAndStrictReadErrorsNeverFallback`.

Python/Go differences:

- The SDK directly owns the DomainConfig singleton binding absent from Gophercloud v2.15.0. It takes a string parent Domain ID in the fixed /domains/{domain_id}/config path, never appending a config member ID or performing parent name lookup. Raw base_path overrides/formatters, Python Domain/Resource object inputs and constructor/runtime injection are deliberate exclusions of this Go profile.
- Mutable Python DomainConfigDriver/DomainConfigLDAP child Resources are represented as independent declared JSON snapshots with child inherited defaults; mapping unknowns are dropped from the Resource view but actual Wire/Envelope preserves them, nonmapping nonnull becomes {}, and null remains null. Parent location is SDK-owned null with no automatic Connection inference. The Go view is eagerly projected; malformed child input can fail before HTTP rather than when a Source lazy getter is later read.
- Only explicit raw JSON attributes are written; descriptor casts apply to the returned view, not transmitted nested payloads. Options/headers/version are copied, bulk attribute maps use deterministic sorted keys and typed header/microversion options are Go conveniences. SDK-R1/C1/S1 mutable Resource/cache/adapter/session goals and authenticated cloud/release verification remain separately tracked; caller custom transport cache bypass is not claimed.
- Update is a newly seeded singleton operation, not mutation/equality/recommit of an existing Python Resource. An optional attrs id may seed the view but is removed from commit body; the parent domain_id URI remains bound independently.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--domain", required=True)
parser.add_argument("--action", choices=("create", "get", "update", "delete"), required=True)
parser.add_argument("--attrs", default='{"identity":{"driver":"ldap"},"ldap":{"url":"ldap://localhost"}}')
parser.add_argument("--strict-delete", action="store_true")
args = parser.parse_args()
with openstack.connect(cloud=args.cloud) as conn:
    if args.action == "create":
        value = conn.identity.create_domain_config(args.domain, **json.loads(args.attrs))
    elif args.action == "get":
        value = conn.identity.get_domain_config(args.domain)
    elif args.action == "update":
        value = conn.identity.update_domain_config(args.domain, **json.loads(args.attrs))
    else:
        value = conn.identity.delete_domain_config(args.domain, ignore_missing=not args.strict_delete)
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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    domain := flag.String("domain", "", "parent Domain ID")
    action := flag.String("action", "", "create, get, update or delete")
    attrs := flag.String("attrs", `{"identity":{"driver":"ldap"},"ldap":{"url":"ldap://localhost"}}`, "create/update JSON attributes")
    strict := flag.Bool("strict-delete", false, "do not ignore missing config")
    flag.Parse()
    if *domain == "" { log.Fatal("domain is required") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    var row *domains.DomainConfigRecord
    switch *action {
    case "create", "update":
        var fields map[string]json.RawMessage
        if err := json.Unmarshal([]byte(*attrs), &fields); err != nil { log.Fatal(err) }
        if fields == nil { log.Fatal("attrs must be a JSON object") }
        option := domains.WithDomainConfigOpts(domains.DomainConfigOpts{Attributes: fields})
        if *action == "create" {
            row, err = service.Domains.CreateConfigRecord(ctx, *domain, option)
        } else {
            row, err = service.Domains.UpdateConfigRecord(ctx, *domain, option)
        }
    case "get":
        row, err = service.Domains.GetConfigRecord(ctx, *domain)
    case "delete":
        result, err := service.Domains.DeleteConfigRecord(ctx, *domain, domains.WithDomainConfigIgnoreMissing(!*strict))
        if err != nil { log.Fatal(err) }
        printJSON(result.Deleted)
        return
    default:
        log.Fatal("action must be create, get, update or delete")
    }
    if err != nil { log.Fatal(err) }
    printJSON(row.Resource)
}
```

## References

- [Domain scalar-ID update](python-record-mutations.md)
- [Install](../../../docs/install.md)
