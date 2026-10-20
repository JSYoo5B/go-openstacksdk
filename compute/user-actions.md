# User Actions

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/create_server_remote_console`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/remoteconsoles.API.CreateConsole`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/remoteconsoles.WithConsoleCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/remoteconsoles.WithConsoleCreateProtocol`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/remoteconsoles.WithConsoleCreateProtocolValue`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/remoteconsoles.WithConsoleCreateType`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/remoteconsoles.WithConsoleCreateTypeValue`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/remoteconsoles.WithConsoleCreateURL`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/remoteconsoles.WithConsoleCreateURLValue`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/remoteconsoles.WithConsoleCreateField`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/remoteconsoles.WithConsoleCreateHeader`

- CreateConsole posts directly to the selected ResourceBase servers/{serverID}/remote-consoles route without GET, LIST, query, discovery or a legacy action request. Omitted attributes produce an empty remote_console object. Omitted/null/empty protocol with a truthy type derives all seven pinned protocol mappings, unknown type derives null, and an explicit nonempty protocol is retained even when inconsistent. Nullable Type/URL, URL values and explicit empty values remain concrete inputs; an unset Optional clears an earlier choice.
  Tests: `api/compute_remote_console_test.go:TestComputeRemoteConsoleRequestDefaultsDerivationAndSeedOwnership`.
- The winning type is checked after options using shared source tuple matching: webmks requires selected same-finite-major Compute2.8 and spice-direct2.99, including explicit protocol. Below-boundary, unselected, malformed and global latest selections fail with ErrUnsupported and zero HTTP; numeric2.100 and finite2.latest are accepted. Last-wins, bulk clear and null clear remove an earlier gate. No unrelated2.6 prerequisite or automatic version upgrade is introduced.
  Tests: `api/compute_remote_console_test.go:TestComputeRemoteConsoleWinningTypeSelectedVersionGates`.
- Resource contains the original three declared protocol/type/url seeds, omitted defaults as null and the fixed caller server_id; inferred protocol changes only the transmitted body. Actual known response fields overlay seeds, including explicit null/untyped values, while unknown fields and a response server_id remain only in independently owned Wire. Actual Envelope/Header/StatusCode are retained; nested and flat object responses are supported. Empty, opaque, malformed or invalid-UTF8 accepted bodies retain the seeded Resource and nil Wire rather than manufacturing a response row.
  Tests: `api/compute_remote_console_test.go:TestComputeRemoteConsoleRequestDefaultsDerivationAndSeedOwnership`, `api/compute_remote_console_test.go:TestComputeRemoteConsoleResponseEvidenceSourceAndNativeCompatibility`.
- Owned final200..399 responses are accepted, including actual201/203, bodyless204 and flat300 without Location. Parsed nonobject roots and null/array remote_console envelopes fail with accepted ResponseError and the physical receipt. HTTP403/404 preserve native method/URL/body/header/status evidence; accepted read, nested404 Close and source failures retain receipt/cause with no SDK resend and cannot become ignored missing.
  Tests: `api/compute_remote_console_test.go:TestComputeRemoteConsoleResponseEvidenceSourceAndNativeCompatibility`, `api/compute_remote_console_test.go:TestComputeRemoteConsoleWinningTypeSelectedVersionGates`.
- Concrete bulk/field inputs and callback-added raw fields are snapshotted; bulk replaces declared options while retaining extensions, reusable options execute once per call, and core protocol/type/url/server_id overrides and unsupported query fail before HTTP. The captured client keeps selected token/header/version policy. Option source drift, nil options and custom-cause cancellation stop later callbacks and HTTP while preserving the original option/context causes. Native Create remains a separate required-protocol/type strict200 API.
  Tests: `api/compute_remote_console_test.go:TestComputeRemoteConsoleRequestDefaultsDerivationAndSeedOwnership`, `api/compute_remote_console_test.go:TestComputeRemoteConsoleResponseEvidenceSourceAndNativeCompatibility`.

Python/Go differences:

- Python accepts a server Resource and dynamic **attrs; Go requires an explicit validated single path-segment server ID and nullable string options for protocol/type/url, with protected field/header extensions. Resource is a stateless declared-field raw view, Wire is the precise actual row, and independently owned receipts replace a connected mutable Python Resource. Body descriptors remain untyped on response; omitted known view fields default to null and the caller URI parent is fixed.
- This direct modern API checks selected Nova microversion with shared source tuple matching for webmks/spice-direct, including2.100 and finite2.latest; global latest cannot prove a finite major2 gate. It performs no advertised endpoint discovery or discovered maximum selection. The separate compute.Service.CreateConsole provides advertised/selected checks and modern/legacy dispatch and is reviewed under python:compute/v2/create_console. General all-service Resource/session/discovery scope remains in the SDK goal.
- Accepted JSON-decoding ValueError-like bodies are tolerated with seeded Resource and nil Wire, while parseable nonobject roots/envelopes are explicit Go errors with receipts. Physical read/Close/source/context failures remain errors rather than tolerated JSON failures. Native remoteconsoles.Create retains its required inputs, typed alias and strict200 response policy; the separately reviewed Servers.ConsoleURL legacy action and Service.CreateConsole modern/legacy composition are not part of this direct operation.
- Go treats non-UTF-8 accepted bodies as unparseable and preserves the original envelope with seeded Resource/nil Wire; it does not reproduce Python response charset detection or replacement-character decoding.

### `python:compute/v2/delete_keypair`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.DeleteKeypair`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairDeleteUserID`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairDeleteQuery`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithKeypairDeleteHeader`

- DeleteKeypair treats the supplied name as the logical keypair ID and makes one bodyless DELETE at the selected prefixed member route without GET, LIST or wait. Empty/default owner is omitted, an explicit Unicode/special-character owner and vendor query are URL-escaped, and selected live token/configured headers/both Compute2.1 version headers are retained without adding an owner-version gate or changing the client.
  Tests: `api/compute_keypair_delete_test.go:TestComputeKeypairDeleteOwnedDefaultsOwnerAndNativeCompatibility`.
- IgnoreMissing defaults to true; explicit true ignores ordinary404 and explicit false preserves the native404 cause plus ErrNotFound. Owned final200..399 is success with passive non-JSON response bytes and no extraction or follow-up request; HTTP403 is preserved with actual/expected status, exact DELETE URL, body/header and operation context. Native Delete202/204 and strict404, and Remove's default/strict missing policy, remain separate unchanged APIs.
  Tests: `api/compute_keypair_delete_test.go:TestComputeKeypairDeleteOwnedDefaultsOwnerAndNativeCompatibility`.
- Concrete typed options are snapshotted; last option wins, bulk resets typed owner/missing selections while retaining extensions, and caller query/header aliases cannot alter the prepared request. Nil API/client/context/options, unsafe name, null missing policy, protected owner/ignore_missing queries, unsupported body fields and spoofed authentication headers fail without HTTP. Pre-canceled or option-canceled contexts preserve context.Canceled, custom cause and the original option error; captured-source failures stop later callbacks.
  Tests: `api/compute_keypair_delete_test.go:TestComputeKeypairDeleteOwnedPreflightAndOptionOwnership`.
- Accepted read/Close, nested404, cancellation and source failures remain terminal ResponseErrors with physical202 body/header and original cause, do not become ErrNotFound or ignored-missing success, and do not invoke native retry after acceptance. A native503 retry callback that changes the captured source is stopped before a second DELETE while retaining original HTTP evidence; options are not replayed.
  Tests: `api/compute_keypair_delete_test.go:TestComputeKeypairDeleteAcceptedFailuresAndNativeRetryGuards`.

Python/Go differences:

- Python accepts a keypair Resource or its logical name ID and returns None. Go accepts a validated single-segment/text name and returns error only; nil represents accepted deletion or deliberately ignored ordinary missing, with no completion wait. UserID is a concrete string so empty is omitted, and nullable IgnoreMissing distinguishes unset/default true from false while rejecting JSON null rather than applying Python dynamic truthiness.
- Go retains the caller's selected microversion and sends explicit owner query literally; it does not perform Python session discovery, change the version or add a new local2.10 owner gate. Nova policy determines foreign-owner permissions, with403 preserved. Full connected Resource/session lifecycle remains separate SDK scope.
- Owned DeleteKeypair accepts final200..399 and reads passive body bytes while retaining physical read/Close/source/context evidence. Native Delete remains strict202/204 and404; native Remove retains its existing lookup/missing policy. The separate Cloud ComputeCloudMixin/delete_keypair operation and arbitrary Resource name resolution are not promoted by this direct Proxy mapping.

### `gophercloud:compute/v2/keypairs.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithDeleteOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithDeleteQuery`

- Native Delete issues one bodyless DELETE directly at the selected prefixed os-keypairs/{name} route. Omitted options send no query; the library-owned concrete DeleteOpts builder escapes an explicit owner as user_id and binds an escaped vendor query extension alongside it. Live token, configured source headers and both selected Compute2.55 version headers reach the actual request.
  Tests: `api/compute_keypair_delete_test.go:TestComputeKeypairDeleteOwnedDefaultsOwnerAndNativeCompatibility`.
- Native202/204 succeeds without JSON extraction while200/404 remains strict ErrUnexpectedResponseCode. Failure preserves expected202/204, actual status, DELETE method, exact URL and original passive response body/header. Default/strict Remove404 compatibility remains unchanged and does not alter direct native Delete's missing policy.
  Tests: `api/compute_keypair_delete_test.go:TestComputeKeypairDeleteOwnedDefaultsOwnerAndNativeCompatibility`.

Python/Go differences:

- The upstream DeleteOptsBuilder interface is replaced by concrete DeleteOption/WithDeleteOptions and a library-owned query serializer with WithDeleteQuery extensions. API.Delete returns ExtractErr with operation context; native direct-name route, owner omission, strict202/204 and strict404 behavior are preserved. New DeleteKeypair's default-ignore/broad-status policy is an independent Python operation mapping and is not applied to this native function.

### `gophercloud:compute/v2/remoteconsoles.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/remoteconsoles.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/remoteconsoles.WithCreateField`

- Native Create rejects omitted protocol or type before HTTP, retaining both required concrete inputs. The library-owned builder binds a URL field extension with the exact remote_console body and sends one authenticated POST to the selected prefixed servers/{serverID}/remote-consoles route without query or lookup.
  Tests: `api/compute_remote_console_test.go:TestComputeRemoteConsoleResponseEvidenceSourceAndNativeCompatibility`.
- Native Extract returns the typed protocol/type/url fields and ignores unknown response fields. A nonstring URL preserves the otherwise decoded protocol/type in a partial RemoteConsole with json.UnmarshalTypeError; a null remote_console envelope returns nil,nil. No owned seed or actual Wire model is substituted for native extraction.
  Tests: `api/compute_remote_console_test.go:TestComputeRemoteConsoleResponseEvidenceSourceAndNativeCompatibility`.
- Only native POST200 succeeds; actual203 remains ErrUnexpectedResponseCode with expected200, original POST method, exact member URL and response body/header. The additive CreateConsole200..399 policy does not widen native Create's status contract.
  Tests: `api/compute_remote_console_test.go:TestComputeRemoteConsoleResponseEvidenceSourceAndNativeCompatibility`.

Python/Go differences:

- The upstream CreateOptsBuilder interface is replaced by concrete CreateOpts/CreateOption and a library-owned body serializer with WithCreateField extensions. API.Create returns extracted native RemoteConsole and wraps failures with operation context; native protocol/type requirements, strict200 status and partial/null extraction remain unchanged. Protocol derivation, selected-version gates and seeded Resource/Wire belong to the independent Python CreateConsole mapping.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
console = conn.compute.create_server_remote_console("server-id", type="novnc")
print(console.protocol, console.type, console.url)
conn.compute.delete_keypair("old-login-key")
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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/remoteconsoles"
)

func main() {
    serverID := flag.String("server-id", "", "server ID for the remote console")
    keyName := flag.String("keypair", "", "keypair name to delete")
    ownerID := flag.String("owner-user-id", "", "optional keypair owner")
    strict := flag.Bool("strict-missing", false, "report a missing keypair")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *serverID, *keyName, *ownerID, *strict); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, serverID, keyName, ownerID string, strict bool) error {
    if serverID == "" || keyName == "" {
        return fmt.Errorf("-server-id and -keypair are required")
    }
    conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.10"))
    if err != nil {
        return err
    }
    service, err := conn.ComputeV2(ctx)
    if err != nil {
        return err
    }
    console, err := service.RemoteConsoles.CreateConsole(ctx, serverID,
        remoteconsoles.WithConsoleCreateType("novnc"))
    if console != nil {
        data, printErr := json.MarshalIndent(map[string]any{
            "resource": console.Resource,
            "wire": console.Wire,
            "status_code": console.StatusCode,
            "header": console.Header,
        }, "", "  ")
        if printErr != nil {
            return printErr
        }
        fmt.Println(string(data))
    }
    if err != nil {
        return fmt.Errorf("create remote console: %w", err)
    }
    options := []keypairs.KeypairDeleteOption{
        keypairs.WithKeypairDeleteIgnoreMissing(!strict),
    }
    if ownerID != "" {
        options = append(options, keypairs.WithKeypairDeleteUserID(ownerID))
    }
    if err := service.KeyPairs.DeleteKeypair(ctx, keyName, options...); err != nil {
        return fmt.Errorf("delete keypair: %w", err)
    }
    fmt.Println("keypair deletion accepted or missing")
    return nil
}
```

## References

- [Console Selection](console-selection.md)
- [create_server_remote_console](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L2903-L2920)
- [Server Remote Console](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_remote_console.py#L18-L77)
- [delete_keypair](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L842-L866)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1338-L1404)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
