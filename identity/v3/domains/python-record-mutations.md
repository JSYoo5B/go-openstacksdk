# Python Record Mutations

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/create_domain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationAttribute`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationLocation`

- Legacy native Create sends POST domains with the {"domain": {...}} envelope and decodes native model fields; Domain.options is omitted from that native model, rather than representing every Python Domain field.
  Tests: `identity/v3/domains/python_parity_test.go:TestPythonDomainCreateGetUpdateRequests`.
- Owned CreateRecord sends only explicit declared raw Domain attributes in a domain POST envelope without requiring name or inventing descriptor defaults. Last input alias wins; request enabled/options values are not cast, while returned Domain Resource projects bool/dict/null/defaults and original response evidence retains unknown fields.
  Tests: `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainCreateRecordNormalizesAliasesWithoutSendingDefaults`, `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainCreateRecordDistinguishesOmittedAndExplicitNull`.
- Options/raw values/headers/version/location are independently owned through callbacks; nil/invalid/source-changing/cancel options fail safely. Rejected404 physical read/Close/cancel failures preserve their causes and close once; clean404 retains actual native status/body proof. None becomes lookup fallback or successful no-op. Source-changing callbacks are separately rejected in preflight before HTTP.
  Tests: `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainRecordMutationsOwnOptionsHeadersVersionAndLocation`, `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainRecordMutationsPreflightAndOptionLifecycle`, `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainRecordMutationsPhysicalFaultsAndRejectionsDoNotFallback`.

Python/Go differences:

- Legacy native Create returns a Gophercloud model pointer rather than the mutable Python Resource; the new sibling CreateRecord returns an independently owned DomainRecord with options and original response evidence.
- Legacy native Create/Update: Python accepts any status below 400; the native Create accepts only 201.
- Legacy native Create/Update: domains.CreateOpts requires Name, so Go rejects before HTTP a Python create_domain without a name.
- Legacy native Create/Update: Domain options have no typed CreateOpts field and are sent with domains.WithCreateField("options", ...).
- New owned mutation APIs leave native Create/Update untouched. They take finite UTF-8 JSON attributes and safe string IDs in a freshly seeded Domain operation, return independent DomainRecord views plus physical Wire/Envelope/header/status, retain Domain.options and explicit null and drop unknown Body attributes. Request values remain raw; descriptor casts apply only to the returned view.
- The fixed Domain collection/member route deliberately rejects raw base_path with ErrUnsupported/ErrInvalidOption rather than reproducing Source arbitrary URI overrides or formatter/runtime objects. Headers, Microversion and Location are explicit Go conveniences; configured location is copied, unknown server project_id/location is not trusted. Class/session/adapter injection and caller custom transport cache bypass are not claimed.
- Ordered attribute helpers use last canonical/wire alias occurrence; bulk Attributes maps use sorted keys as a deterministic Go adaptation. Mutable existing Resource baseline/equality/recommit state is not modeled. SDK-R1/C1/S1, authenticated cloud acceptance and release verification remain separately tracked whole-SDK goals.

### `python:identity/v3/update_domain`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.New`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationAttribute`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/domains.WithRecordMutationLocation`

- Legacy native Update sends PATCH domains/{id} with the {"domain": {...}} envelope and optional explicit WithUpdateField("id", id); the old fixture documents that opt-in native body, not pinned Resource.commit dirty-id behavior.
  Tests: `identity/v3/domains/python_parity_test.go:TestPythonDomainCreateGetUpdateRequests`.
- Owned scalar-ID UpdateRecord sends PATCH with explicit raw declared attributes including nullable description/enabled/options; constructor ID is seeded but never dirty or sent. Existing response overlays seed and retains options plus physical response proof; duplicate attrs id is rejected before HTTP.
  Tests: `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainUpdateRecordPATCHPreservesSeedAndActualResponse`, `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainUpdateRecordExplicitNullIsSentAsDirty`, `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainRecordMutationsPreflightAndOptionLifecycle`.
- Empty or only-unknown scalar-ID updates return a freshly projected ID seed with no HTTP, Wire or fabricated status. This is distinct from mutable Python Resource equality/recommit, and explicit null remains dirty and sends a PATCH.
  Tests: `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainUpdateRecordEmptyOrUnknownAttrsAvoidHTTP`, `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainUpdateRecordExplicitNullIsSentAsDirty`.
- Options/raw values/headers/version/location are independently owned through callbacks; nil/invalid/source-changing/cancel options fail safely. Rejected404 physical read/Close/cancel failures preserve their causes and close once; clean404 retains actual native status/body proof. None becomes lookup fallback or successful no-op. Source-changing callbacks are separately rejected in preflight before HTTP.
  Tests: `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainRecordMutationsOwnOptionsHeadersVersionAndLocation`, `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainRecordMutationsPreflightAndOptionLifecycle`, `identity/v3/domains/python_record_mutations_test.go:TestPythonDomainRecordMutationsPhysicalFaultsAndRejectionsDoNotFallback`.

Python/Go differences:

- Legacy native Create/Update: Python accepts an ID string or a Resource instance and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer.
- Legacy native Create/Update: Python accepts any status below 400; the native Update accepts only 200.
- Legacy native Update permits an explicitly added id body field through WithUpdateField("id", id). The pinned Resource.commit actually discards dirty id before preparing PATCH; the new owned UpdateRecord follows that behavior instead of the older fixture comment.
- New owned mutation APIs leave native Create/Update untouched. They take finite UTF-8 JSON attributes and safe string IDs in a freshly seeded Domain operation, return independent DomainRecord views plus physical Wire/Envelope/header/status, retain Domain.options and explicit null and drop unknown Body attributes. Request values remain raw; descriptor casts apply only to the returned view.
- The fixed Domain collection/member route deliberately rejects raw base_path with ErrUnsupported/ErrInvalidOption rather than reproducing Source arbitrary URI overrides or formatter/runtime objects. Headers, Microversion and Location are explicit Go conveniences; configured location is copied, unknown server project_id/location is not trusted. Class/session/adapter injection and caller custom transport cache bypass are not claimed.
- Ordered attribute helpers use last canonical/wire alias occurrence; bulk Attributes maps use sorted keys as a deterministic Go adaptation. Mutable existing Resource baseline/equality/recommit state is not modeled. SDK-R1/C1/S1, authenticated cloud acceptance and release verification remain separately tracked whole-SDK goals.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--mode", choices=("create", "update"), required=True)
parser.add_argument("--identity", default="")
parser.add_argument("--name", default="example-domain")
args = parser.parse_args()
if args.mode == "update" and not args.identity:
    parser.error("--identity is required for update")
with openstack.connect(cloud=args.cloud) as conn:
    attrs = {"description": None, "is_enabled": False,
             "options": {"immutable": False}}
    if args.mode == "create":
        row = conn.identity.create_domain(name=args.name, **attrs)
    else:
        row = conn.identity.update_domain(args.identity, **attrs)
        unchanged = conn.identity.update_domain(args.identity)
        print(json.dumps(unchanged.to_dict(), indent=2))
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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    mode := flag.String("mode", "", "create or update")
    id := flag.String("identity", "", "Domain ID for update")
    name := flag.String("name", "example-domain", "Domain create name")
    flag.Parse()
    if *mode != "create" && *mode != "update" { log.Fatal("mode must be create or update") }
    if *mode == "update" && *id == "" { log.Fatal("identity is required for update") }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    options := []domains.RecordMutationOption{
        domains.WithRecordMutationAttribute("description", nil),
        domains.WithRecordMutationAttribute("is_enabled", false),
        domains.WithRecordMutationAttribute("options", map[string]any{"immutable": false}),
    }
    var row *domains.DomainRecord
    if *mode == "create" {
        options = append(options, domains.WithRecordMutationAttribute("name", *name))
        row, err = service.Domains.CreateRecord(ctx, options...)
    } else {
        row, err = service.Domains.UpdateRecord(ctx, *id, options...)
    }
    if err != nil { log.Fatal(err) }
    printJSON(row.Resource)
    if *mode == "update" {
        unchanged, err := service.Domains.UpdateRecord(ctx, *id)
        if err != nil { log.Fatal(err) }
        printJSON(unchanged.Resource)
    }
}
```

## References

- [Python Records](python-records.md)
- [Install](../../../docs/install.md)
