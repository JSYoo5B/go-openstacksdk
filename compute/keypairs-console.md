# Keypairs Console

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/create_keypair`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.CreateKeypair`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairCreateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairCreateName`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairCreatePublicKey`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairCreateType`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairCreateUserID`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairCreateField`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairCreateHeader`

- CreateKeypair performs one fixed selected-ResourceBase collection POST at os-keypairs with keypair envelope and no query, preliminary GET/LIST, completion wait or cleanup. No attributes produce an empty keypair object. All eight declared body fields plus id/is_deleted aliases accept explicit null, empty, scalar, array or object JSON values; name is body data without member-path ID validation. Unknown semantic attributes are discarded before serialization. Selected live token, source headers and both Compute2.1 version headers are retained without a new local type or owner gate.
  Tests: `api/compute_keypair_create_test.go:TestComputeKeypairCreateDirtyBodyAliasesAndNativeCompatibility`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateOwnedOptionsSourceAndAcceptedFailures`.
- Original dirty attributes form the Resource seed; no type=ssh field is synthesized into the POST. Omitted final type projects as ssh only in the view, while null and empty remain literal. Input and response numeric id become name only when canonical name is absent; present name including null wins. Resource id independently mirrors name. Raw deleted values are transmitted unchanged and shared BodyFieldBoolean projects is_deleted only in the view; response null stays null, nonempty false-string is true and empty array/zero is false. Bulk and response deleted win over is_deleted; individual semantic options use the last canonical value.
  Tests: `api/compute_keypair_create_test.go:TestComputeKeypairCreateDirtyBodyAliasesAndNativeCompatibility`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateResponseSeedWireAndPassiveErrors`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateOwnedOptionsSourceAndAcceptedFailures`.
- Actual wrapped keypair or flat object fields overlay recognized seed fields, with response omission preserving input and present null replacing it. The independently owned nine-field Resource excludes unknown attributes; independently owned Wire preserves only actual response fields, literal id/deleted aliases, unknown/vendor fields and large JSON numbers. View defaults and input are not inserted into Wire. Envelope, record/resource/wire headers and status preserve the physical response, and mutations do not cross these records.
  Tests: `api/compute_keypair_create_test.go:TestComputeKeypairCreateResponseSeedWireAndPassiveErrors`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateDirtyBodyAliasesAndNativeCompatibility`.
- Accepted final200..399 includes actual200/201/202/203, bodyless204 and flat300. Empty, opaque and malformed JSON leave the seeded Resource and nil Wire; parseable nonobject roots or null/array keypair envelopes fail with accepted ResponseError and body/header/status receipt. HTTP403/404 retain the native actual/expected status, POST method, exact collection URL and original body/header; create404 is not ignored. Physical read/Close nested404 and post-response source failure retain accepted201 evidence and original cause rather than tolerated JSON failure, with zero SDK resend.
  Tests: `api/compute_keypair_create_test.go:TestComputeKeypairCreateResponseSeedWireAndPassiveErrors`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateOwnedOptionsSourceAndAcceptedFailures`.
- Bulk/individual semantic inputs, vendor values and callback-added attrs/raw fields are snapshotted before later mutations. Bulk replaces semantic choices while retaining extensions; callbacks execute once per call. Core name/id/type/user_id/deleted/is_deleted extensions and unsupported query fail before HTTP. Captured API/source changes and callback cancellation stop later callbacks and HTTP, preserving the original callback error, context.Canceled and custom cancellation cause. A native503 retry hook that changes the source cannot issue a second allocation and retains the original HTTP evidence; allowed token refresh reaches the fixed request.
  Tests: `api/compute_keypair_create_test.go:TestComputeKeypairCreateOwnedOptionsSourceAndAcceptedFailures`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateDirtyBodyAliasesAndNativeCompatibility`.

Python/Go differences:

- Python returns a connected mutable Keypair and accepts dynamic attrs. Go returns a finite stateless source-shaped Resource view plus independently owned actual Wire and response receipt; all nine declared view attributes and eight body fields are represented by raw JSON. String convenience options coexist with concrete map/bulk attributes for nullable or untyped values. Unknown semantic attrs are discarded and explicit vendor fields use protected library-owned extensions. Full Resource/session/location/discovery remains separate shared SDK scope, without blocking this finite named create contract.
- Canonical name over id, including present null, matches Keypair._consume_attrs for input and response. For deleted/is_deleted collisions Python generic mapping follows input or response insertion order; Go deterministically favors canonical deleted in bulk and response maps. Individual Go options apply sequentially with the last canonical choice winning. These documented map/option choices are not insertion-order emulation.
- Go uses the caller-selected client microversion unchanged rather than performing Python discovery and selecting the discovered maximum capped at Keypair2.10 when no session default exists. Name/type/owner semantics and permissions are left to Nova; an owner fixture is not a real cloud authorization result. The separate Cloud ComputeCloudMixin/create_keypair wrapper is not promoted by this direct Proxy operation.
- Go tolerates invalid-UTF8 accepted bodies as unparseable with seeded Resource and nil Wire, preserving the envelope; Python response charset detection and replacement-character decoding are not reproduced. Valid nonobject JSON and physical read/Close/source/context failures remain explicit receipt errors. Native Create retains its required string Name, typed model and strict200/201 policy as a separately reviewed source declaration.

### `gophercloud:compute/v2/keypairs.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithCreateField`

- Native Create preserves required Name preflight and sends one authenticated POST to the selected prefixed os-keypairs collection without query or lookup. Concrete CreateOpts binds Name, UserID, Type and PublicKey together with a vendor object extension in the actual keypair body. A protected user_id extension and omitted required Name fail before HTTP; the application need not implement CreateOptsBuilder.
  Tests: `api/compute_keypair_create_test.go:TestComputeKeypairCreateDirtyBodyAliasesAndNativeCompatibility`.
- Native Extract preserves all six typed string fields Name/Fingerprint/PublicKey/PrivateKey/UserID/Type and ignores unknown vendor fields. A nonstring name preserves the decoded fingerprint in a partial KeyPair and json.UnmarshalTypeError; a null keypair envelope returns nil,nil. SDK-owned seed/default/alias/RawResource handling is not substituted for native extraction.
  Tests: `api/compute_keypair_create_test.go:TestComputeKeypairCreateDirtyBodyAliasesAndNativeCompatibility`.
- Only native POST200/201 is accepted, with actual fixtures for both. Actual202 remains ErrUnexpectedResponseCode with expected200/201, original POST method, exact collection URL, original response body and header. The additive CreateKeypair200..399 policy does not widen native Create.
  Tests: `api/compute_keypair_create_test.go:TestComputeKeypairCreateDirtyBodyAliasesAndNativeCompatibility`.

Python/Go differences:

- The upstream CreateOptsBuilder interface is replaced by concrete CreateOpts/CreateOption and a library-owned serializer with WithCreateOptions/WithCreateField. API.Create returns the extracted native KeyPair and wraps failures with operation context. Native required Name, optional string fields, strict200/201 and partial/null extraction remain intact; source-shaped nullable attributes, view-only ssh and Resource/Wire belong to the independent Python CreateKeypair mapping.

### `python:compute/v2/get_server_console_url`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ConsoleURL`

- ConsoleURL preserves mandatory exact console type and all six legacy action mappings: novnc/xvpvnc to os-getVNCConsole, spice-html5/spice-direct to os-getSPICEConsole, rdp-html5 to os-getRDPConsole and serial to os-getSerialConsole. It performs one fixed selected-ResourceBase POST at servers/{serverID}/action with the exact type body, no query and empty Accept, retaining live token/source headers and selected version. Selected2.100, absent version and spice-direct2.5 remain legacy without discovery, modern fallback or a new local gate. Empty/webmks/unknown/case or whitespace variants fail before HTTP.
  Tests: `api/compute_console_url_test.go:TestComputeConsoleURLLegacyActionsSelectedSourceAndUnsupportedTypes`.
- The actual root object must contain the exact console key. Its object/empty object/null/array/string/false/zero value is returned as independently copied json.RawMessage without a type/url model, input seed, coercion or unknown/large-number loss. Actual response type may differ from the requested type; literal Unicode, escaped controls and URL text remain raw. Owned final200..399 accepts actual200/201/203 and300 without Location.
  Tests: `api/compute_console_url_test.go:TestComputeConsoleURLRawValueAndResponseEvidence`, `api/compute_console_url_test.go:TestComputeConsoleURLLegacyActionsSelectedSourceAndUnsupportedTypes`.
- Missing console or wrong-case key, null/array/string root, empty204, opaque/malformed accepted JSON and invalid UTF8 remain receipt errors rather than nil console success. Accepted errors retain original body/header/status and do not invoke native retry; this legacy getter does not borrow creation JSON tolerance. HTTP400/403/404 preserve native actual/expected200..399 status, POST method, exact URL and original body/header, with no lookup or modern fallback.
  Tests: `api/compute_console_url_test.go:TestComputeConsoleURLRawValueAndResponseEvidence`.
- Nil API/client/context, unsafe server ID, pre-canceled custom-cause context and an outer operation guard fail with zero HTTP. Accepted read, nested404 Close, source drift and cancellation retain physical203 receipt and original causes; nested404 is not ErrNotFound. Accepted processing failures cause no SDK rePOST and the shared fault wrappers confirm one physical request/read/Close where applicable.
  Tests: `api/compute_console_url_test.go:TestComputeConsoleURLSourceContextAndAcceptedFailures`.

Python/Go differences:

- Python accepts a Server Resource or ID and returns the actual dynamically shaped console value. Go requires a validated explicit single-path-segment/text server ID and mandatory type, returning an independently owned json.RawMessage; Python None is represented by raw JSON null. Missing console is an operation-scoped ResponseError rather than Python KeyError, while null/scalar/array values remain successful exact values.
- This legacy getter itself preserves the selected client microversion without advertised discovery or Server2.100 maximum selection. The separate compute.Service.CreateConsole provides advertised/selected modern-versus-legacy dispatch and is reviewed under python:compute/v2/create_console. validate_console_auth_token, cloud console wrappers and general Resource/session lifecycle are not promoted by this getter row.
- Go rejects invalid-UTF8 response bodies with a receipt and does not reproduce Python response charset detection or replacement-character decoding. JSON shape/selection/read/Close/source/context failures preserve actual evidence through Go errors; accepted processing errors are not retried. This strict getter policy is independent from the tolerant accepted-nonJSON policy of CreateConsole/CreateKeypair.

### `python:cloud/_compute/ComputeCloudMixin/create_keypair`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.CreateKeypair`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithCloudKeypairPublicKey`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithCloudKeypairCreateHeader`

- Cloud CreateKeypair always supplies name as literal body data, including an empty or path-like body name, and supplies public_key only for a nonempty string. Empty public key is omitted rather than forwarded as the leaf's explicit empty attribute. The library composes the existing one fixed keypair-enveloped collection POST without lookup, wait or cleanup.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairCreateAndDeleteOutcomes`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateDirtyBodyAliasesAndNativeCompatibility`.
- The existing owned creation result preserves original input seed/defaults and actual response overlay, including absent type=ssh only in Resource, present null authority, logical name/id aliases, independently owned Wire and physical receipt. HTTP and accepted read/Close/source errors remain original causes beside actual receipt instead of a second Cloud creation or synthetic successful result.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairCreateAndDeleteOutcomes`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateResponseSeedWireAndPassiveErrors`, `api/compute_keypair_create_test.go:TestComputeKeypairCreateOwnedOptionsSourceAndAcceptedFailures`.
- Creation captures configured Cloud location once before its option callback and keeps that snapshot in Resource when the callback changes the caller cloud pointer. Location is omitted from the transmitted body, while the actual response location remains unchanged in Wire and Envelope. A service without the adapter projects null location.
  Tests: `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairOptionsOwnershipAndTerminalErrors`, `api/compute_keypair_cloud_test.go:TestComputeCloudKeypairCreateAndDeleteOutcomes`.

Python/Go differences:

- Python name/public_key string or None inputs become a required Go name argument and optional concrete PublicKey string option; empty represents the source falsey/None omission. Full nullable/type/user_id/vendor body control remains available through the separate versioned CreateKeypair, not invented arguments of the Cloud convenience.
- Go returns the owned KeypairRecord and adds finite current Connection location to Resource only; standalone location is null. It does not return a connected mutable Python object or populate view defaults/current location in actual Wire/Envelope.
- The versioned create engine retains selected microversion rather than adding Python session discovery for this mutation. Its documented final200..399/accepted opaque-body tolerance/strict parsed-object/physical receipt policies and deterministic alias choices are reused. Nova still validates names/types/owner permissions; mutable session/discovery declarations remain independent SDK scope.
- This row closes the two-input Cloud creation transform only and does not incidentally promote native/Proxy create, server creation, keypair deletion or unrelated service workflow declarations.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
created = conn.compute.create_keypair(name="workstation", public_key=public_key)
console = conn.compute.get_server_console_url(server_id, "novnc")
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
)

func main() {
    name := flag.String("keypair", "", "keypair name to create")
    publicKeyPath := flag.String("public-key-file", "", "SSH public key to import")
    owner := flag.String("owner-user-id", "", "optional owner")
    serverID := flag.String("server-id", "", "optional server ID for a legacy console")
    consoleType := flag.String("console-type", "novnc", "legacy console type")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *name, *publicKeyPath, *owner, *serverID, *consoleType); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, name, publicKeyPath, owner, serverID, consoleType string) error {
    if name == "" || publicKeyPath == "" {
        return fmt.Errorf("-keypair and -public-key-file are required")
    }
    publicKey, err := os.ReadFile(publicKeyPath)
    if err != nil { return err }
    conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.10"))
    if err != nil { return err }
    service, err := conn.ComputeV2(ctx)
    if err != nil { return err }
    options := []keypairs.KeypairCreateOption{
        keypairs.WithKeypairCreateName(name),
        keypairs.WithKeypairCreatePublicKey(string(publicKey)),
    }
    if owner != "" {
        options = append(options, keypairs.WithKeypairCreateUserID(owner))
    }
    created, err := service.KeyPairs.CreateKeypair(ctx, options...)
    if err != nil { return fmt.Errorf("create keypair: %w", err) }
    data, err := json.MarshalIndent(map[string]any{
        "resource": created.Resource, "wire": created.Wire,
        "status_code": created.StatusCode, "header": created.Header,
    }, "", "  ")
    if err != nil { return err }
    fmt.Println(string(data))
    if serverID != "" {
        console, err := service.Servers.ConsoleURL(ctx, serverID, consoleType)
        if err != nil { return fmt.Errorf("get legacy console: %w", err) }
        fmt.Println(string(console))
    }
    return nil
}
```

## References

- [User Actions](user-actions.md)
