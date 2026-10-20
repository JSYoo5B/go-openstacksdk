# User Read APIs

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/get_flavor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindExtraSpecs`

- The documented explicit-ID get_flavor mapping combines FindFallbackNever with IgnoreMissing(false). Native Flavor GET200 returns the typed ID/name/RAM/disk/VCPU/swap/bandwidth/public/ephemeral/description fields; GET400/403/404 remain errors without a flavor list or extra-specs request under the Never/strict policy.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsGETAndExtraSpecsPresence`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsMissingPoliciesAndUnsafeNames`.
- Omitted or false ExtraSpecs makes no extra request. True fetches the returned canonical ID once when inline specs are missing/null/empty, retains inline nonempty specs without another GET, preserves unrelated Flavor fields and forwards no caller query to enrichment. Missing/null extra_specs retain native nil-map behavior while an explicit object remains nonnil; the selected Compute version is not upgraded to2.61.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsGETAndExtraSpecsPresence`.
- Extra-spec HTTP403/404/203/204, typed decode, read/transport and cancellation failures are terminal and cannot trigger another lookup or become ignored missing. Native GET203/204, invalid typed Flavor fields, null result or unsafe returned ID are rejected before enrichment or fallback.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsEnrichmentFailuresAreTerminal`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsPreflightAndNativeGETFailures`.
- Nil/canceled contexts, nil/unsupported/invalid options and an unsafe direct route under FindFallbackNever fail without HTTP. The shared direct/enrichment engine preserves caller context, current provider token, configured middleware/headers, endpoint/resource-base and selected microversion rather than adding a new service or discovery policy.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsPreflightAndNativeGETFailures`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsKeepsLiveProviderVersionAndContext`.

Python/Go differences:

- Python get_flavor accepts an ID or Flavor Resource; this Go mapping requires an explicit safe ID string and the documented Never/strict options. Ordinary FindIdentity retains its independent list-fallback defaults. This row neither closes find_flavor nor treats native flavors_links paging as the whole Python inherited continuation contract.
- The result is native Flavor, with native typed field/swap extraction and a required safe nonblank returned ID, rather than a seeded mutable Python Flavor Resource. Native missing/null ExtraSpecs are nil maps; missing IsPublic is false rather than the source descriptor default true. Python ID/name/original-input fallbacks, nullable/unknown/location descriptors and connected mutable state are deliberate DTO differences.
- Go retains the selected client microversion and native strict200 GET/specs policy, without automatic source Resource discovery/version selection. Full generic Resource/session/cache APIs remain in the SDK goal, independently of this explicit-ID GET and optional enrichment. Caller raw query and Find controls are Go engine extensions, not extra required arguments of Python get_flavor.

### `python:compute/v2/fetch_server_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Metadata`

- The server-ID metadata read makes one GET at the selected ResourceBase/reverse-prefix servers/{id}/metadata route, with no query/body or detail/fallback request. Current token, configured header and both selected Compute2.55 version headers reach the request; the selected client remains unchanged.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeServerMetadataReadNativeContract`.
- Native metadata extraction preserves Unicode and empty string values, nonnil empty object maps and missing/null nil maps. A null map member becomes the native empty string. Wrong metadata shape or nonstring values preserve the typed decode cause and native partial-map ABI; malformed JSON returns its original error.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeServerMetadataReadNativeContract`.
- Only native GET200 is accepted. Actual403/404/unexpected203 remain errors with native actual/expected status, GET method, exact URL, body and response header, and no unrelated request or missing-success conversion.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeServerMetadataReadNativeContract`.

Python/Go differences:

- Pinned fetch_server_metadata mutates and returns a seeded Server Resource. Go accepts an explicit server ID and returns a stateless map[string]string projection, with no supplied Server mutation or detail GET. Native missing/null wrapper is nil and an explicit empty object is nonnil; null members become empty strings while nonstring values may return a partial map with a type error. Python arbitrary dictionary/descriptor/seed behavior is not claimed.
- The selected native client policy and strict200 status are preserved rather than adapter-wide Python successful statuses or automatic discovery. Native success results do not add RawResource/Wire/header fields; HTTP errors retain original evidence. Full mutable Resource/session/cache state remains a separate SDK declaration scope.

### `python:compute/v2/get_server_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Metadata`

- The pinned get_server_metadata alias delegates to fetch_server_metadata; both names map to the same server-ID Metadata GET. The actual SDK request preserves the selected reverse ResourceBase, live token, configured header and both Compute2.55 version headers, with no query/body or preliminary server-detail request.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeServerMetadataReadNativeContract`.
- The shared native string-map result preserves Unicode/empty values, nonnil empty maps, nil missing/null wrappers and null-member empty strings. Wrong shapes/nonstring fields preserve the typed cause and native partial result; malformed JSON remains an error.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeServerMetadataReadNativeContract`.
- Native strict GET200 is retained. Actual403/404/unexpected203 expose original actual/expected status, method, URL, body/header and make no fallback or unrelated request; selected service configuration remains unchanged.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeServerMetadataReadNativeContract`.

Python/Go differences:

- Pinned get_server_metadata is the deprecated4.14 alias of fetch_server_metadata and emits a Python warning. Go exposes one Metadata method and does not reproduce the language-specific alias warning; distinct source IDs/fingerprints retain the two named declarations with the same actual HTTP evidence.
- Go accepts an explicit server ID and returns a stateless map[string]string instead of mutating/returning a seeded Server Resource. Native nullable wrapper/member, typed partial error and strict200 behavior are documented Go alternatives; no Python arbitrary mapping/descriptor/adapter-wide status or mutable Resource/session/cache behavior is promoted by this alias.

### `python:compute/v2/get_keypair`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithGetOptions`

- Keypair Get directly uses the requested name at the selected reverse-prefix os-keypairs/{name} route and makes one bodyless GET with no LIST/fallback. Omitted or explicit empty owner sends no query; nonempty owner is the escaped user_id query, independent of response numeric id/name.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeKeypairGetNativeContract`.
- Native Name/Fingerprint/PublicKey/PrivateKey/UserID/Type fields are extracted. Missing/null Type stays empty without an ssh default; missing Name is not synthesized from the request. Missing/null keypair envelope returns nil,nil. Malformed JSON remains an error, while a known-field type error preserves the native partial model and original UnmarshalTypeError.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeKeypairGetNativeContract`.
- The selected Compute2.55 headers, configured source header and live token reach every keypair GET without changing the client/provider/resource-base. Only native200 succeeds;403/404/unexpected203 retain actual/expected status, method, exact owner-qualified URL, body and response header instead of a missing success or unrelated request.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeKeypairGetNativeContract`.

Python/Go differences:

- Python accepts a name/Keypair Resource, uses name as Resource ID and can retain a seeded name; Go requires the explicit name string and returns actual native Name/Fingerprint/PublicKey/PrivateKey/UserID/Type fields. Numeric wire id, source ssh type default, created/deleted/location/unknown descriptors and mutable Resource methods are not synthesized. Native missing/null wrapper may be nil,nil and known-field decode errors may retain a partial model.
- Optional owner is concrete GetOpts.UserID; empty string omits it like the source truthy check. Go preserves the selected client version rather than Keypair Resource discovery/max2.10 selection or an automatic owner version change. A foreign-owner403 remains an HTTP error; local fixtures do not grant or verify real-cloud admin permissions.
- The documented native strict200 response and typed projection are Go alternatives to Python adapter-wide response/seed/session policy. No list fallback, keypair creation/deletion or whole Resource lifecycle support is promoted through this named getter.

### `python:compute/v2/get_server_console_output`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ConsoleOutput`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithConsoleOutputLength`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithConsoleOutputOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithConsoleOutputField`

- The owned ConsoleOutput action uses the selected reverse/resource-base servers/{id}/action route with no preliminary GET or query. Omitted/nil length sends an empty action object; explicit0, positive and negative integers are transmitted literally. Default Accept is empty like the source action, while current authentication, configured source header and selected Compute2.87 are preserved. Reused helper options apply once per call and do not replace the native service/provider.
  Tests: `api/compute_console_output_test.go:TestComputeConsoleOutputLiteralLengthOwnedOptionsAndNativeCompatibility`.
- Typed length pointers and JSON extension input are owned snapshots, including custom callback values between options. Last individual length wins; bulk zero options restore omission. Core length extension, empty/non-JSON fields, nil option and unsupported per-call query/header/arguments fail before HTTP. Existing ShowConsoleOutput retains its signature and native0 omission while owned0 is explicit.
  Tests: `api/compute_console_output_test.go:TestComputeConsoleOutputLiteralLengthOwnedOptionsAndNativeCompatibility`.
- Owned final200..399 action statuses include actual200/201/203/300-without-Location. The native output-string projection preserves Unicode/control characters and native case-insensitive field matching; missing/null/empty output yields empty string. Nonstring output, null/array root, malformed/invalid-UTF8 JSON and empty204 fail with accepted status/body/header proof and no accepted-processing retry. Actual400/403/404 and native ShowConsoleOutput203 rejection keep original method/URL/body/header and native expected status policy.
  Tests: `api/compute_console_output_test.go:TestComputeConsoleOutputLiteralLengthOwnedOptionsAndNativeCompatibility`, `api/compute_console_output_test.go:TestComputeConsoleOutputStringProjectionAndHTTPResponseEvidence`.
- Nil API/context, invalid explicit ID, pre-cancelled custom cause and option cancellation/source/API replacement stop before later callbacks or HTTP while preserving original causes. Accepted read/Close-with-nested404 and source/context failures keep actual200 body/header/status, original cause and one physical action rather than missing success or SDK resend. Native401 reauthentication retains the literal body and selected version with two physical requests, one reauth and one custom option application.
  Tests: `api/compute_console_output_test.go:TestComputeConsoleOutputSourceContextAndAcceptedBodyBoundaries`.

Python/Go differences:

- Python accepts server ID/Server Resource and returns the entire decoded response dict. Go requires a validated single URL path-segment ID with valid text/no controls and returns the native output string projection. Missing/null output becomes empty string, native case-insensitive matching is retained, nonstring output and nonnull-UTF8-object violations are explicit errors. Unknown response fields are not exposed through this string result.
- Concrete nullable integer Length preserves the source omitted-versus-explicit0 distinction and forwards negative values literally. Go excludes arbitrary dynamic input coercion and supplies library-owned pointer/JSON snapshots, extension/core-field protection and preflight/source/context guards. Per-call session/query/header/argument overrides are not part of this named API; selected client policy is preserved.
- Owned ConsoleOutput accepts final200..399 like the source action and adds strict object/string decoding with accepted response evidence and no SDK processing-error replay. Native ShowConsoleOutput remains strict200 with its int omitempty and signature unchanged. Provider native retry/reauthentication remains distinct from SDK composition and the general mutable Resource/session/cache goal. No remote-console creation, server lifecycle or unrelated native operation is promoted by this row.

### `gophercloud:compute/v2/keypairs.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs.WithGetOptions`

- The direct keypair-name GET uses the selected ResourceBase and makes no LIST or unrelated request. Default and explicit-empty UserID omit the owner query; a supplied owner is escaped as user_id. The library-owned concrete GetOpts builder retains live provider authentication, configured headers and both selected Compute2.55 version headers without changing the selected client.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeKeypairGetNativeContract`.
- Native Extract preserves the six string fields Name/Fingerprint/PublicKey/PrivateKey/UserID/Type and ignores unknown wire fields. Missing/null type remains empty, missing name is not seeded from the requested name, and missing/null keypair wrappers return nil,nil. Malformed JSON remains an error; a nonstring known field preserves the native partial KeyPair and json.UnmarshalTypeError.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeKeypairGetNativeContract`.
- Only native GET200 is accepted. Actual403/404/unexpected203 preserve ErrUnexpectedResponseCode with actual/expected status, method, exact owner-bearing URL, response body and header, and do not trigger a fallback request or convert missing to success.
  Tests: `api/compute_metadata_keypair_reads_test.go:TestComputeKeypairGetNativeContract`.

Python/Go differences:

- The upstream optional GetOptsBuilder interface is replaced by concrete GetOption/WithGetOptions and a library-owned serializer; API.Get returns the extracted native KeyPair and wraps failures with operation context. Native aliases, owner omission, strict200 status and partial-result extraction are preserved. The documented Python get_keypair mapping is a separate catalog operation.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
flavor = conn.compute.get_flavor("flavor-id", get_extra_specs=True)
server = conn.compute.fetch_server_metadata("server-id")
keypair = conn.compute.get_keypair("login-key")
console = conn.compute.get_server_console_output("server-id", length=50)
print(flavor.extra_specs, server.metadata, keypair.fingerprint, console)
```

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/keypairs"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := run(ctx); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context) error {
    if len(os.Args) != 4 && len(os.Args) != 5 {
        return fmt.Errorf("usage: %s SERVER_ID FLAVOR_ID KEYPAIR_NAME [OWNER_USER_ID]", os.Args[0])
    }
    conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.10"))
    if err != nil {
        return err
    }
    service, err := conn.ComputeV2(ctx)
    if err != nil {
        return err
    }
    flavor, err := service.Flavors.FindIdentity(ctx, os.Args[2],
        resource.WithIdentityFindFallback(resource.FindFallbackNever),
        resource.WithIdentityFindIgnoreMissing(false),
        resource.WithIdentityFindExtraSpecs(true),
    )
    if err != nil {
        return fmt.Errorf("get flavor: %w", err)
    }
    metadata, err := service.Servers.Metadata(ctx, os.Args[1])
    if err != nil {
        return fmt.Errorf("get server metadata: %w", err)
    }
    var keyOptions []keypairs.GetOption
    if len(os.Args) == 5 {
        keyOptions = append(keyOptions,
            keypairs.WithGetOptions(keypairs.GetOpts{UserID: os.Args[4]}))
    }
    pair, err := service.KeyPairs.Get(ctx, os.Args[3], keyOptions...)
    if err != nil {
        return fmt.Errorf("get keypair: %w", err)
    }
    output, err := service.Servers.ConsoleOutput(ctx, os.Args[1],
        servers.WithConsoleOutputLength(50))
    if err != nil {
        return fmt.Errorf("get console output: %w", err)
    }
    data, err := json.MarshalIndent(map[string]any{
        "flavor": flavor,
        "metadata": metadata,
        "keypair": pair,
        "console_output": output,
    }, "", "  ")
    if err != nil {
        return err
    }
    fmt.Println(string(data))
    return nil
}
```

## References

- [Finding Identities](../docs/finding-identities.md)
- [get_flavor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L246-L266)
- [ Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L1876-L1906)
- [MetadataMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/metadata.py#L33-L46)
- [get_keypair](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L868-L890)
- [Server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server.py#L938-L952)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
