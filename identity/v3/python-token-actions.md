# Python Token Actions

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:identity/v3/validate_token`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens.API.ValidateToken`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens.WithValidateTokenOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens.WithValidateTokenNoCatalog`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens.WithValidateTokenAllowExpired`

- GET auth/tokens sends an independent X-Subject-Token and only truthy nocatalog=True/allow_expired=True parameters while retaining provider authentication. The owned Token Resource, Wire, Envelope and Header channels are independent; thirteen Body descriptors, raw date values and response-only subject token follow pinned Source defaults and conversion.
  Tests: `identity/v3/tokens/token_actions_test.go:TestPythonTokenValidateQueriesHeadersAndOwnedRecord`, `identity/v3/tokens/token_actions_test.go:TestPythonTokenValidateDescriptorConversionsAndToleratedBodies`.
- HTTP200..399 is accepted. Empty or syntactically invalid JSON becomes an empty declared view; valid non-object envelope or token values fail with actual response evidence. Rejected HTTP responses retain native status/body/header causes; accepted representation/read/close/context failures retain the partial owned receipt.
  Tests: `identity/v3/tokens/token_actions_test.go:TestPythonTokenValidateRepresentationAndHTTPFailures`, `identity/v3/tokens/token_actions_test.go:TestPythonTokenTransportCancellationAndAcceptedBodyErrors`.
- Concrete options replace/apply in order and remain owned after callbacks. Nil/failed options, cancellation and service-source changes stop before dispatch; simultaneous callback and guard failures are joined. Subject headers cannot be changed by configured aliases or native retry hooks.
  Tests: `identity/v3/tokens/token_actions_test.go:TestPythonTokenOptionsPreflightAndSourceCapture`, `identity/v3/tokens/token_actions_test.go:TestPythonTokenSelectedHeaderCannotChangeDuringRetry`.

Python/Go differences:

- Python returns a mutable Token Resource; Go returns an owned TokenRecord with separate declared/raw/envelope/HTTP channels, no automatic Resource mutation or session/cache subclass behavior.
- Token headers and JSON use valid UTF-8 Go strings. The fixed request/source guard forbids retargeting through retry or redirect hooks; native Get is unchanged.

### `python:identity/v3/check_token`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens.API.CheckToken`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens.WithCheckTokenOpts`
- `github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens.WithCheckTokenAllowExpired`

- HEAD auth/tokens returns true only for actual HTTP200; other statuses, including204/403/404/500, return false without status error. True allow_expired is encoded as allow_expired=True and false is omitted; the subject header remains separate from authentication.
  Tests: `identity/v3/tokens/token_actions_test.go:TestPythonTokenCheckAllHTTPStatusesAndExpiredQuery`.
- Concrete options, source capture and fixed subject header ownership are shared with token validation. Transport/read/close/cancellation failures remain errors; callback and guard causes are joined and no subsequent callback or HTTP runs after preflight failure.
  Tests: `identity/v3/tokens/token_actions_test.go:TestPythonTokenOptionsPreflightAndSourceCapture`, `identity/v3/tokens/token_actions_test.go:TestPythonTokenTransportCancellationAndAcceptedBodyErrors`, `identity/v3/tokens/token_actions_test.go:TestPythonTokenSelectedHeaderCannotChangeDuringRetry`.

Python/Go differences:

- Python arguments become a string subject token and concrete CheckTokenOption values; Go returns (bool,error) and preserves non-HTTP handling failures.
- The SDK captures service/provider/endpoint/microversion once and fixes target and subject header across native attempts; arbitrary session/adapter/cache policies are separate.

### `python:identity/v3/revoke_token`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens.API.RevokeToken`

- DELETE auth/tokens sends an independent subject token and an empty body; actual HTTP200..399 is accepted without JSON decoding, while400+ keeps native status/body/header evidence.
  Tests: `identity/v3/tokens/token_actions_test.go:TestPythonTokenRevokeAcceptedCodesAndOpaqueBody`.
- Invalid headers, cancelled contexts or invalid sources stop before HTTP. Transport/read/close/context failures retain their causes and response evidence, while selected subject headers and service capture cannot be changed by retry hooks.
  Tests: `identity/v3/tokens/token_actions_test.go:TestPythonTokenOptionsPreflightAndSourceCapture`, `identity/v3/tokens/token_actions_test.go:TestPythonTokenTransportCancellationAndAcceptedBodyErrors`, `identity/v3/tokens/token_actions_test.go:TestPythonTokenSelectedHeaderCannotChangeDuringRetry`.

Python/Go differences:

- Python positional token becomes a valid UTF-8 Go string; Go returns error under the existing OperationError cause chain and leaves native Revoke unchanged.
- The SDK-owned fixed-request and source guard reject retry/redirect retargeting and preserve physical body failures rather than ignoring them.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
validated = conn.identity.validate_token(token, nocatalog=True)
print(validated.expires_at, validated.subject_token)
valid = conn.identity.check_token(token, allow_expired=True)
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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/tokens"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    token := flag.String("token", "", "subject token")
    allowExpired := flag.Bool("allow-expired", false, "allow expired subject token")
    revoke := flag.Bool("revoke", false, "revoke after validation and check")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    record, err := service.Tokens.ValidateToken(ctx, *token,
        tokens.WithValidateTokenNoCatalog(true),
        tokens.WithValidateTokenAllowExpired(*allowExpired))
    if err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(record.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
    valid, err := service.Tokens.CheckToken(ctx, *token,
        tokens.WithCheckTokenAllowExpired(*allowExpired))
    if err != nil { log.Fatal(err) }
    fmt.Printf("valid: %t\n", valid)
    if *revoke {
        if err := service.Tokens.RevokeToken(ctx, *token); err != nil { log.Fatal(err) }
    }
}
```

## References

- [Install](../../docs/install.md)
- [Token Actions Test](tokens/token_actions_test.go)
