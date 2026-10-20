# Console Auth Token

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/validate_console_auth_token`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.ValidateConsoleAuthToken`

- ValidateConsoleAuthToken performs a fixed captured-ResourceBase bodyless GET at os-console-auth-tokens/{literal token}, with no query/server lookup/console creation. Opaque Unicode and reserved token characters are escaped as one member and remain distinct from the live provider X-Auth-Token. Selected2.100/latest/2.latest bypass discovery and remain literal. Unselected advertised2.110 uses private2.99, old2.5 has no local type/minimum gate, missing max or min2.100>ceiling continues without version headers. Original client binding, version, endpoints and source headers are unchanged.
  Tests: `api/compute_console_auth_token_test.go:TestComputeConsoleAuthTokenFixedLookupAndOwnedVersionSelection`.
- Resource has exactly instance_uuid/host/port/tls_port/internal_access_path/id/name/location. Token id is a seed; missing id preserves it while actual null/large-number id overrides it. All declared fields remain passive JSON with missing values null, including structured instance/host and string/numeric/null ports. Wrapped console or flat actual response is selected as Wire; unknown vendor/precise numbers/response location remain only there. Resource, Wire, envelope and headers have independent storage and actual status metadata.
  Tests: `api/compute_console_auth_token_test.go:TestComputeConsoleAuthTokenFixedLookupAndOwnedVersionSelection`, `api/compute_console_auth_token_test.go:TestComputeConsoleAuthTokenSeededResourceActualWireAndResponseEvidence`.
- Configured computed cloud/project/region location is independently recorded once before unselected discovery and member GET, with zone null. Configuration bytes mutated during discovery and response-provided location do not replace it. Direct no-dependency location is null. Accepted empty204/opaque202/malformed201/invalidUTF8 bodies keep seeded Resource and nil Wire, while valid root null/array or console null/scalar remain processing errors with actual physical receipt and no Resource.
  Tests: `api/compute_console_auth_token_test.go:TestComputeConsoleAuthTokenSeededResourceActualWireAndResponseEvidence`.
- Ordinary403/404 retain original native GET URL/body/header/actual status and bounded200..399 expected codes, rather than false validation or ignored missing. Accepted lookup read/nested404 Close/source faults preserve actual203 partial record with nil Resource/Wire; an accepted discovery Close404 preserves only that discovery300 receipt with no member lookup. Original nested method/URL/body/header/cause survives and cannot become ErrNotFound or accepted-processing retry.
  Tests: `api/compute_console_auth_token_test.go:TestComputeConsoleAuthTokenSeededResourceActualWireAndResponseEvidence`, `api/compute_console_auth_token_test.go:TestComputeConsoleAuthTokenPreflightAcceptedFaultsAndLiveNativePolicy`.
- Nil facade/context/source, invalid/control/invalidUTF8 token, pre-canceled custom-cause context, conflicting blank source version headers and outer guards fail before authenticated I/O and before the location callback where applicable. Location source/API replacement or cancellation stops discovery/lookup and retains callback/context causes. Native source/version/body retry mutations after original503 prevent another lookup with the original HTTP proof; live401 reauthentication refreshes the provider token while retaining the exact console-token member route.
  Tests: `api/compute_console_auth_token_test.go:TestComputeConsoleAuthTokenPreflightAcceptedFaultsAndLiveNativePolicy`.

Python/Go differences:

- Python returns a connected mutable ConsoleAuthToken Resource despite the Dictionary docstring. Go requires a validated explicit single path-segment UTF8 string token and returns an independently owned ConsoleAuthTokenRecord separating the seeded source view, actual Wire and physical HTTP receipt. No option/builder is needed because the pinned operation has one mandatory input and no kwargs. The console token is distinct from provider authentication; response IDs/host/ports grant no request authority.
- The seven untyped Body fields plus computed location remain passive raw JSON, including null, precise numbers and arbitrary structured values. Direct compute.New with no CloudLocation dependency returns location null; Connection.Compute supplies a scoped configured location captured before discovery. Unknown response fields remain in Wire only. General Python Resource/session mutation and cache lifecycle are not promoted by this stateless named getter.
- Go uses the shared finite catalog-bound Nova reader with exact API-row status/self-link eligibility, first matching row, captured authenticated token and per-operation snapshot. It does not reproduce complete Keystoneauth sorting/self-URL routing/anonymous401 authentication switch/session discovery cache. Clean native404/405 or a valid unmatched row alone permits the root discovery attempt. Go discovery parse/physical/source/context errors are terminal; Python maximum_supported_microversion can continue versionless after some DiscoveryFailure cases.
- Accepted opaque/empty/malformed or invalidUTF8 response JSON retains the input-seeded Resource and nil Wire; valid nonobject roots/envelopes and physical read/Close/source/context failures retain actual receipt errors. InvalidUTF8 is a Go parsing-unavailable policy rather than Python response charset/replacement-character decoding. Explicit selected microversions remain literal, while unselected2.99 maximum selection uses a private client with bodyless/version-owned guarded GET; missing maximum or minimum above ceiling remains a versionless lookup, not a local Unsupported check.
- This token connection-information GET is classified as a core admin API: the official Nova policy documented on 2026-10-08 sets os_compute_api:os-console-auth-tokens to rule:context_is_admin. The earlier work-plan user classification is corrected. Server deployment policy overrides remain authoritative; this SDK adds no local role gate. End-user console creation is a separate policy and operation.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import sys
import openstack

conn = openstack.connect()
console = conn.compute.validate_console_auth_token(sys.argv[1])
print(console.to_dict())
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
)

func main() {
    token := flag.String("console-token", "", "console token to look up")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *token); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, token string) error {
    if token == "" {
        return fmt.Errorf("-console-token is required")
    }
    conn, err := sdk.Connect(ctx)
    if err != nil {
        return err
    }
    service, err := conn.Compute(ctx)
    if err != nil {
        return err
    }
    record, err := service.ValidateConsoleAuthToken(ctx, token)
    if record != nil {
        data, marshalErr := json.MarshalIndent(map[string]any{
            "resource": record.Resource,
            "wire": record.Wire,
            "status_code": record.StatusCode,
            "header": record.Header,
        }, "", "  ")
        if marshalErr != nil {
            return marshalErr
        }
        fmt.Println(string(data))
    }
    return err
}
```

## References

- [Policy](https://docs.openstack.org/nova/latest/configuration/policy.html)
- [Console Selection](console-selection.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Implementation Plan](../docs/implementation-plan.md)
