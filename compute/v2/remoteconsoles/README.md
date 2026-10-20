# Remoteconsoles

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

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

## References

- [User Actions](../../user-actions.md)
