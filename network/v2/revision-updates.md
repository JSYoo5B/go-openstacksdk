# Revision Updates

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:networking/v2/networks.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithUpdateField`

- For this API among all nine native facades, actual HTTP Update retains PUT, effective ResourceBase route, native JSON envelope, description and extension body values, auth and ordinary source headers. Typed RevisionNumber nil/zero/positive/signed emits no header or exact revision_number=N respectively, never a JSON revision field, and returns the native typed response ID without source-client mutation.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- For all nine APIs, final WithUpdateOptions replacement chooses body fields and revision condition together while retaining permitted extension fields and input scalar values. Existing query/header extension and nil-option errors remain preflight OperationError/ErrInvalidOption without HTTP; no new header capability is introduced.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- All nine retain native200 success and their exact201 policy: networks/ports/subnets accept201, the other six reject it. Native412 keeps status/body under OperationError with no SDK fallback or additional request.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`.
- For all nine, native401 reauth,429 backoff and503 retry replay the prepared body and serialized revision through four physical requests with refreshed live auth and permitted native header addition. Caller revision mutation during reauth does not reselect the condition; the shared client remains intact.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Representative Network API exercises the shared native escape path: service-wide If-Match override, RetryFunc explicit condition override and OmitHeaders removal remain effective. This is one Network fixture, not separate per-API raw-override coverage for all nine.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsRawNativeHeaderEscapePolicy`.
- Representative Network API exercises native307 redirect refusal or following with retained PUT/body/live auth and callback-overridden If-Match. This is one shared-path fixture, not per-API redirect coverage for the other eight.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeRedirectPolicy`.
- Generator checks all nine current native source declarations and regenerated adapter declarations against stored facades, retaining RevisionNumber *int json:-/h:If-Match from final validated cfg.Options and existing Update/WithUpdateOptions/WithUpdateField without new helpers. Source/field/import drift mutations use Network as the representative case; owning-import/lexical alias-shadow and optional-builder/secondary-input tests are generic fixtures, not all-nine drift-case coverage.
  Tests: `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersActualNineNativeAdapters`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersRejectNativeSourceAndRevisionDrift`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersOwningImportsAndLexicalScope`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersPreserveOptionalBuildersAndSecondaryInputs`.

Python/Go differences:

- The facade accepts native concrete UpdateOpts with library-owned private header/body adapter rather than requiring a caller UpdateOptsBuilder. Final WithUpdateOptions selects the original header-tagged concrete fields and body together, while WithUpdateField adds permitted body fields. Public signatures, aliases and extension capabilities are retained.
- Native execution still formats If-Match from nil/non-nil signed revision pointers, preserves request/body/success/result behavior and raw MoreHeaders/OmitHeaders/native callbacks, and wraps errors in inspectable OperationError. The bridge preserves the header through adaptation; it does not protect it from native raw overrides or add unconditional conflict fallback.

### `python:network/v2/update_network`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.UpdateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithIfMatch`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRevision`

- The concrete Go update for this operation emits exact revision_number=N for typed zero/positive/signed values, omits a nil generated condition and preserves the native route/body/response envelope. This all-nine Go matrix is evidence of Go behavior; Python conditional similarity or difference is established from the pinned class/helper sources.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- Final typed replacement selects body and condition together, preserves allowed body extensions, and rejects unsupported query/header/nil options before HTTP. These concrete option semantics do not emulate Python mutable Resource/attrs state.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- Native accepted-code and412 policies remain inspectable with no SDK unconditional fallback; native401/429/503 replay retains the prepared condition/body and live auth despite caller pointer mutation. Python adapter/session retry/error semantics remain a documented difference.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Representative Network API exercises the shared native escape path: service-wide If-Match override, RetryFunc explicit condition override and OmitHeaders removal remain effective. This is one Network fixture, not separate per-API raw-override coverage for all nine.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsRawNativeHeaderEscapePolicy`.
- Representative Network API exercises native307 redirect refusal or following with retained PUT/body/live auth and callback-overridden If-Match. This is one shared-path fixture, not per-API redirect coverage for the other eight.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeRedirectPolicy`.
- Literal-ID update with no Body/Header options performs no HTTP and returns a new ID-only native model. Supplied attrs send one PUT networks/{id}, with no lookup. Raw If-Match/WithRevision alone sends an empty network envelope. WithRevision(0) sends revision_number=0, overrides raw if_match regardless option order, discards invalid raw Header values before validation, and later revisions win.
  Tests: `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkMutationBodiesDefaultsAndAliases`, `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkRevisionHeaderOnlyPriorityAndClientIsolation`.
- WithAttribute snapshots JSON at constructor time; reusable options preserve false/zero/null/empty values, later wire-key values win, and aliases use the pinned generated Body MRO table: is_admin_state_up=>admin_state_up; is_router_external=>router:external; is_shared=>shared; is_port_security_enabled=>port_security_enabled; subnet_ids=>subnets; provider_network_type=>provider:network_type; provider_segmentation_id=>provider:segmentation_id; ipv4_address_scope_id=>ipv4_address_scope. Unknown and query-only attrs stay in Body. Invalid option/serialization/capability and Resource control attrs fail before HTTP.
  Tests: `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkMutationBodiesDefaultsAndAliases`, `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkOptionSnapshotsHeaderRoutingAndPreflight`.
- Inherited if_match Header is separate from Body. String values, string lists joined by comma-space, explicit empty list and null->None are supported, with constructor snapshots and configured client Header isolation. Create accepts raw If-Match; update accepts revision/If-Match, and explicitly selected conditions replace case-insensitive configured If-Match on a cloned client.
  Tests: `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkOptionSnapshotsHeaderRoutingAndPreflight`, `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkRevisionHeaderOnlyPriorityAndClientIsolation`.
- SDK HTTP success codes are 200..399 with unchanged native typed extraction. Network native Extract returns an ID-less zero model pointer and nil error on 204. Results are not seeded with request attrs/ID. OperationError retains native decode errors, HTTP status/body/header and pre-canceled/option-callback/post-response context.Canceled plus custom context.Cause.
  Tests: `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkMutationStatusesDecodeAndContextCauses`, `network/v2/networks/python_mutations_contracts_test.go:TestPythonCoreNetworkRevisionHeaderOnlyPriorityAndClientIsolation`.

Python/Go differences:

- A functional callable JSON mutation profile uses native Go models and a literal update ID, while existing native Create/Update ABI, required validations and result decoders are unchanged.
- The explicit SDK success profile is 200..399 and strict native decode remains. Network native Extract returns an ID-less zero model pointer and nil error on 204. Source can retain a mutable request Resource on empty/invalid JSON success; native Go results do not store response unknown fields, merge request attrs, or apply descriptor access/coercion.
- WithRevision accepts int64 (including zero, negative and exact max-int64), rather than arbitrary-precision Python values. Raw If-Match is a JSON string/string-list/null profile; arbitrary Python object str() coercion is unavailable. A selected Header containing CR/LF is rejected before HTTP. Raw Header values discarded by revision are not validated.
- Body controls base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location are rejected. Update id and if_revision must use the literal ID parameter and WithRevision rather than Body extensions. Configured Gophercloud session/client policy is retained.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Full Python Resource/Munch instance input, returning the same mutable instance, comparison against existing dirty/cache state and retaining request attrs or unknown response attrs remain separate from this callable Go profile.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Complete Proxy/Resource control kwargs, computed location/metadata, dynamic microversion, Python descriptor access/conversion and adapter/session/cache/exception policy remain separate.

### `gophercloud:networking/v2/ports.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithUpdateField`

- For this API among all nine native facades, actual HTTP Update retains PUT, effective ResourceBase route, native JSON envelope, description and extension body values, auth and ordinary source headers. Typed RevisionNumber nil/zero/positive/signed emits no header or exact revision_number=N respectively, never a JSON revision field, and returns the native typed response ID without source-client mutation.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- For all nine APIs, final WithUpdateOptions replacement chooses body fields and revision condition together while retaining permitted extension fields and input scalar values. Existing query/header extension and nil-option errors remain preflight OperationError/ErrInvalidOption without HTTP; no new header capability is introduced.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- All nine retain native200 success and their exact201 policy: networks/ports/subnets accept201, the other six reject it. Native412 keeps status/body under OperationError with no SDK fallback or additional request.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`.
- For all nine, native401 reauth,429 backoff and503 retry replay the prepared body and serialized revision through four physical requests with refreshed live auth and permitted native header addition. Caller revision mutation during reauth does not reselect the condition; the shared client remains intact.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Generator checks all nine current native source declarations and regenerated adapter declarations against stored facades, retaining RevisionNumber *int json:-/h:If-Match from final validated cfg.Options and existing Update/WithUpdateOptions/WithUpdateField without new helpers. Source/field/import drift mutations use Network as the representative case; owning-import/lexical alias-shadow and optional-builder/secondary-input tests are generic fixtures, not all-nine drift-case coverage.
  Tests: `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersActualNineNativeAdapters`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersRejectNativeSourceAndRevisionDrift`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersOwningImportsAndLexicalScope`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersPreserveOptionalBuildersAndSecondaryInputs`.

Python/Go differences:

- The facade accepts native concrete UpdateOpts with library-owned private header/body adapter rather than requiring a caller UpdateOptsBuilder. Final WithUpdateOptions selects the original header-tagged concrete fields and body together, while WithUpdateField adds permitted body fields. Public signatures, aliases and extension capabilities are retained.
- Native execution still formats If-Match from nil/non-nil signed revision pointers, preserves request/body/success/result behavior and raw MoreHeaders/OmitHeaders/native callbacks, and wraps errors in inspectable OperationError. The bridge preserves the header through adaptation; it does not protect it from native raw overrides or add unconditional conflict fallback.

### `python:network/v2/update_port`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.UpdateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithIfMatch`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRevision`

- The concrete Go update for this operation emits exact revision_number=N for typed zero/positive/signed values, omits a nil generated condition and preserves the native route/body/response envelope. This all-nine Go matrix is evidence of Go behavior; Python conditional similarity or difference is established from the pinned class/helper sources.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- Final typed replacement selects body and condition together, preserves allowed body extensions, and rejects unsupported query/header/nil options before HTTP. These concrete option semantics do not emulate Python mutable Resource/attrs state.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- Native accepted-code and412 policies remain inspectable with no SDK unconditional fallback; native401/429/503 replay retains the prepared condition/body and live auth despite caller pointer mutation. Python adapter/session retry/error semantics remain a documented difference.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Literal-ID update with no Body/Header options performs no HTTP and returns an ID-only native model. Supplied attrs send a single PUT ports/{id} without lookup. A raw If-Match or WithRevision alone is dirty and sends {port:{}}. WithRevision(0) emits revision_number=0; it overrides raw if_match regardless option order, including invalid discarded raw values, and the last revision wins.
  Tests: `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortMutationBodiesDefaultsAndAliases`, `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortRevisionHeaderOnlyPriorityAndClientIsolation`.
- WithAttribute constructor snapshots supplied JSON and Header values; reusable options preserve zero/false/null/empty values, later wire-key values win, canonical Body aliases map through the existing pinned MRO table, unknown extension and query-only attrs stay in Body. binding_host_id=>binding:host_id; binding_profile=>binding:profile; is_admin_state_up=>admin_state_up; is_port_security_enabled=>port_security_enabled; security_group_ids=>security_groups. Invalid options/serialization, non-If-Match headers, query mechanisms, and Resource/Proxy control attrs fail before HTTP.
  Tests: `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortMutationBodiesDefaultsAndAliases`, `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortOptionSnapshotsHeaderRoutingAndPreflight`.
- Inherited if_match Header is separate from Body. String values, string lists joined by comma-space, explicit empty list and null->None are supported, with constructor snapshots and configured client Header isolation. Create accepts raw If-Match; update accepts revision/If-Match, and explicitly selected conditions replace case-insensitive configured If-Match on a cloned client.
  Tests: `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortOptionSnapshotsHeaderRoutingAndPreflight`, `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortRevisionHeaderOnlyPriorityAndClientIsolation`.
- SDK HTTP success codes are 200..399 and native typed result extraction is retained. Port native Extract returns an ID-less zero model pointer and nil error on 204. No response model is seeded with request ID/attrs. HTTP status/body/header, native JSON decode errors, and pre-canceled/option-callback/post-response context.Canceled plus custom context.Cause remain observable through OperationError.
  Tests: `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortMutationStatusesDecodeAndContextCauses`, `network/v2/ports/python_mutations_contracts_test.go:TestPythonCorePortRevisionHeaderOnlyPriorityAndClientIsolation`.

Python/Go differences:

- A functional callable JSON mutation profile uses native Go models and a literal update ID, while existing native Create/Update ABI, required validations and result decoders are unchanged.
- HTTP 200..399 is the explicit SDK profile; strict native decode behavior and returned known fields are retained. Port native Extract returns an ID-less zero model pointer and nil error on 204. Source retains a mutable Resource request view on empty or invalid JSON success bodies; this Go API does not synthesize request attributes, returned unknown fields or partial-response fallback.
- WithRevision accepts int64 (including zero, negative and exact max-int64), rather than arbitrary-precision Python values. Raw If-Match is a JSON string/string-list/null profile; arbitrary Python object str() coercion is unavailable. A selected Header containing CR/LF is rejected before HTTP. Raw Header values discarded by revision are not validated.
- Body controls base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location are rejected. Update id and if_revision must use the literal ID parameter and WithRevision rather than Body extensions. Configured Gophercloud session/client policy is retained.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Full Python Resource/Munch instance input, returning the same mutable instance, comparison against existing dirty/cache state and retaining request attrs or unknown response attrs remain separate from this callable Go profile.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Complete Proxy/Resource control kwargs, computed location/metadata, dynamic microversion, Python descriptor access/conversion and adapter/session/cache/exception policy remain separate.

### `gophercloud:networking/v2/subnets.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithUpdateField`

- For this API among all nine native facades, actual HTTP Update retains PUT, effective ResourceBase route, native JSON envelope, description and extension body values, auth and ordinary source headers. Typed RevisionNumber nil/zero/positive/signed emits no header or exact revision_number=N respectively, never a JSON revision field, and returns the native typed response ID without source-client mutation.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- For all nine APIs, final WithUpdateOptions replacement chooses body fields and revision condition together while retaining permitted extension fields and input scalar values. Existing query/header extension and nil-option errors remain preflight OperationError/ErrInvalidOption without HTTP; no new header capability is introduced.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- All nine retain native200 success and their exact201 policy: networks/ports/subnets accept201, the other six reject it. Native412 keeps status/body under OperationError with no SDK fallback or additional request.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`.
- For all nine, native401 reauth,429 backoff and503 retry replay the prepared body and serialized revision through four physical requests with refreshed live auth and permitted native header addition. Caller revision mutation during reauth does not reselect the condition; the shared client remains intact.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Generator checks all nine current native source declarations and regenerated adapter declarations against stored facades, retaining RevisionNumber *int json:-/h:If-Match from final validated cfg.Options and existing Update/WithUpdateOptions/WithUpdateField without new helpers. Source/field/import drift mutations use Network as the representative case; owning-import/lexical alias-shadow and optional-builder/secondary-input tests are generic fixtures, not all-nine drift-case coverage.
  Tests: `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersActualNineNativeAdapters`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersRejectNativeSourceAndRevisionDrift`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersOwningImportsAndLexicalScope`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersPreserveOptionalBuildersAndSecondaryInputs`.

Python/Go differences:

- The facade accepts native concrete UpdateOpts with library-owned private header/body adapter rather than requiring a caller UpdateOptsBuilder. Final WithUpdateOptions selects the original header-tagged concrete fields and body together, while WithUpdateField adds permitted body fields. Public signatures, aliases and extension capabilities are retained.
- Native execution still formats If-Match from nil/non-nil signed revision pointers, preserves request/body/success/result behavior and raw MoreHeaders/OmitHeaders/native callbacks, and wraps errors in inspectable OperationError. The bridge preserves the header through adaptation; it does not protect it from native raw overrides or add unconditional conflict fallback.

### `python:network/v2/update_subnet`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.UpdateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithIfMatch`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRevision`

- The concrete Go update for this operation emits exact revision_number=N for typed zero/positive/signed values, omits a nil generated condition and preserves the native route/body/response envelope. This all-nine Go matrix is evidence of Go behavior; Python conditional similarity or difference is established from the pinned class/helper sources.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- Final typed replacement selects body and condition together, preserves allowed body extensions, and rejects unsupported query/header/nil options before HTTP. These concrete option semantics do not emulate Python mutable Resource/attrs state.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- Native accepted-code and412 policies remain inspectable with no SDK unconditional fallback; native401/429/503 replay retains the prepared condition/body and live auth despite caller pointer mutation. Python adapter/session retry/error semantics remain a documented difference.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Literal-ID update with no Body/Header options performs no HTTP and returns an ID-only native model. Supplied attrs send a single PUT subnets/{id} without lookup. A raw If-Match or WithRevision alone is dirty and sends {subnet:{}}. WithRevision(0) emits revision_number=0; it overrides raw if_match regardless option order, including invalid discarded raw values, and the last revision wins.
  Tests: `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetMutationBodiesDefaultsAndAliases`, `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetRevisionHeaderOnlyPriorityAndClientIsolation`.
- WithAttribute constructor snapshots supplied JSON and Header values; reusable options preserve zero/false/null/empty values, later wire-key values win, canonical Body aliases map through the existing pinned MRO table, unknown extension and query-only attrs stay in Body. is_dhcp_enabled=>enable_dhcp; prefix_length=>prefixlen; subnet_pool_id=>subnetpool_id; use_default_subnet_pool=>use_default_subnetpool. Invalid options/serialization, non-If-Match headers, query mechanisms, and Resource/Proxy control attrs fail before HTTP.
  Tests: `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetMutationBodiesDefaultsAndAliases`, `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetOptionSnapshotsHeaderRoutingAndPreflight`.
- Inherited if_match Header is separate from Body. String values, string lists joined by comma-space, explicit empty list and null->None are supported, with constructor snapshots and configured client Header isolation. Create accepts raw If-Match; update accepts revision/If-Match, and explicitly selected conditions replace case-insensitive configured If-Match on a cloned client.
  Tests: `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetOptionSnapshotsHeaderRoutingAndPreflight`, `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetRevisionHeaderOnlyPriorityAndClientIsolation`.
- SDK HTTP success codes are 200..399 and native typed result extraction is retained. Subnet native Extract returns nil model and nil error on 204. No response model is seeded with request ID/attrs. HTTP status/body/header, native JSON decode errors, and pre-canceled/option-callback/post-response context.Canceled plus custom context.Cause remain observable through OperationError.
  Tests: `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetMutationStatusesDecodeAndContextCauses`, `network/v2/subnets/python_mutations_contracts_test.go:TestPythonCoreSubnetRevisionHeaderOnlyPriorityAndClientIsolation`.

Python/Go differences:

- A functional callable JSON mutation profile uses native Go models and a literal update ID, while existing native Create/Update ABI, required validations and result decoders are unchanged.
- HTTP 200..399 is the explicit SDK profile; strict native decode behavior and returned known fields are retained. Subnet native Extract returns nil model and nil error on 204. Source retains a mutable Resource request view on empty or invalid JSON success bodies; this Go API does not synthesize request attributes, returned unknown fields or partial-response fallback.
- WithRevision accepts int64 (including zero, negative and exact max-int64), rather than arbitrary-precision Python values. Raw If-Match is a JSON string/string-list/null profile; arbitrary Python object str() coercion is unavailable. A selected Header containing CR/LF is rejected before HTTP. Raw Header values discarded by revision are not validated.
- Body controls base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location are rejected. Update id and if_revision must use the literal ID parameter and WithRevision rather than Body extensions. Configured Gophercloud session/client policy is retained.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Full Python Resource/Munch instance input, returning the same mutable instance, comparison against existing dirty/cache state and retaining request attrs or unknown response attrs remain separate from this callable Go profile.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Complete Proxy/Resource control kwargs, computed location/metadata, dynamic microversion, Python descriptor access/conversion and adapter/session/cache/exception policy remain separate.

### `gophercloud:networking/v2/extensions/trunks.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithUpdateField`

- For this API among all nine native facades, actual HTTP Update retains PUT, effective ResourceBase route, native JSON envelope, description and extension body values, auth and ordinary source headers. Typed RevisionNumber nil/zero/positive/signed emits no header or exact revision_number=N respectively, never a JSON revision field, and returns the native typed response ID without source-client mutation.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- For all nine APIs, final WithUpdateOptions replacement chooses body fields and revision condition together while retaining permitted extension fields and input scalar values. Existing query/header extension and nil-option errors remain preflight OperationError/ErrInvalidOption without HTTP; no new header capability is introduced.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- All nine retain native200 success and their exact201 policy: networks/ports/subnets accept201, the other six reject it. Native412 keeps status/body under OperationError with no SDK fallback or additional request.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`.
- For all nine, native401 reauth,429 backoff and503 retry replay the prepared body and serialized revision through four physical requests with refreshed live auth and permitted native header addition. Caller revision mutation during reauth does not reselect the condition; the shared client remains intact.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Generator checks all nine current native source declarations and regenerated adapter declarations against stored facades, retaining RevisionNumber *int json:-/h:If-Match from final validated cfg.Options and existing Update/WithUpdateOptions/WithUpdateField without new helpers. Source/field/import drift mutations use Network as the representative case; owning-import/lexical alias-shadow and optional-builder/secondary-input tests are generic fixtures, not all-nine drift-case coverage.
  Tests: `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersActualNineNativeAdapters`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersRejectNativeSourceAndRevisionDrift`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersOwningImportsAndLexicalScope`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersPreserveOptionalBuildersAndSecondaryInputs`.

Python/Go differences:

- The facade accepts native concrete UpdateOpts with library-owned private header/body adapter rather than requiring a caller UpdateOptsBuilder. Final WithUpdateOptions selects the original header-tagged concrete fields and body together, while WithUpdateField adds permitted body fields. Public signatures, aliases and extension capabilities are retained.
- Native execution still formats If-Match from nil/non-nil signed revision pointers, preserves request/body/success/result behavior and raw MoreHeaders/OmitHeaders/native callbacks, and wraps errors in inspectable OperationError. The bridge preserves the header through adaptation; it does not protect it from native raw overrides or add unconditional conflict fallback.

### `python:network/v2/update_trunk`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithUpdateField`

- The concrete Go update for this operation emits exact revision_number=N for typed zero/positive/signed values, omits a nil generated condition and preserves the native route/body/response envelope. This all-nine Go matrix is evidence of Go behavior; Python conditional similarity or difference is established from the pinned class/helper sources.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- Final typed replacement selects body and condition together, preserves allowed body extensions, and rejects unsupported query/header/nil options before HTTP. These concrete option semantics do not emulate Python mutable Resource/attrs state.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- Native accepted-code and412 policies remain inspectable with no SDK unconditional fallback; native401/429/503 replay retains the prepared condition/body and live auth despite caller pointer mutation. Python adapter/session retry/error semantics remain a documented difference.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- update_trunk(trunk, name=..., description="", is_admin_state_up=False, x_vendor=1) maps to Update with typed Name/Description/AdminStateUp pointers and WithUpdateField("x_vendor", 1), sending PUT /trunks/<id> {"trunk":{...}} with only those keys and no If-Match header, and decodes the updated trunk.
  Tests: `network/v2/extensions/trunks/python_rejudge_test.go:TestPythonRejudgeTrunkUpdate`.

Python/Go differences:

- Go updates a concrete ID with typed UpdateOpts and permitted body fields and returns a native typed value/error. Python accepts ID or existing mutable Resource, applies attrs/descriptor state and commits that Resource.
- Typed Go RevisionNumber nil/zero/signed semantics use the existing native conditional-header capability without a new public helper. This pinned Python resource does not inherit NetworkResource, so forwarded if_revision binds the helper but does not generate If-Match; the Go conditional capability is a native extension difference.
- Go preserves raw native MoreHeaders/OmitHeaders/callback override and retry behavior and reports native412 without SDK unconditional fallback; generic Python adapter/session and mutable/dirty header behavior remain separate.
- Python skips the PUT when no attribute is dirty; Go Update with empty UpdateOpts still sends PUT {"trunk":{}}.
- Python attribute names (is_admin_state_up, project_id with tenant_id alias) map to the typed AdminStateUp field or WithUpdateField wire keys; Go cannot send JSON null for name, description or admin_state_up, which Neutron rejects.
- Go accepts only 200 for the update; Python accepts any non-error status.

### `gophercloud:networking/v2/extensions/subnetpools.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithUpdateField`

- For this API among all nine native facades, actual HTTP Update retains PUT, effective ResourceBase route, native JSON envelope, description and extension body values, auth and ordinary source headers. Typed RevisionNumber nil/zero/positive/signed emits no header or exact revision_number=N respectively, never a JSON revision field, and returns the native typed response ID without source-client mutation.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- For all nine APIs, final WithUpdateOptions replacement chooses body fields and revision condition together while retaining permitted extension fields and input scalar values. Existing query/header extension and nil-option errors remain preflight OperationError/ErrInvalidOption without HTTP; no new header capability is introduced.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- All nine retain native200 success and their exact201 policy: networks/ports/subnets accept201, the other six reject it. Native412 keeps status/body under OperationError with no SDK fallback or additional request.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`.
- For all nine, native401 reauth,429 backoff and503 retry replay the prepared body and serialized revision through four physical requests with refreshed live auth and permitted native header addition. Caller revision mutation during reauth does not reselect the condition; the shared client remains intact.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Generator checks all nine current native source declarations and regenerated adapter declarations against stored facades, retaining RevisionNumber *int json:-/h:If-Match from final validated cfg.Options and existing Update/WithUpdateOptions/WithUpdateField without new helpers. Source/field/import drift mutations use Network as the representative case; owning-import/lexical alias-shadow and optional-builder/secondary-input tests are generic fixtures, not all-nine drift-case coverage.
  Tests: `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersActualNineNativeAdapters`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersRejectNativeSourceAndRevisionDrift`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersOwningImportsAndLexicalScope`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersPreserveOptionalBuildersAndSecondaryInputs`.

Python/Go differences:

- The facade accepts native concrete UpdateOpts with library-owned private header/body adapter rather than requiring a caller UpdateOptsBuilder. Final WithUpdateOptions selects the original header-tagged concrete fields and body together, while WithUpdateField adds permitted body fields. Public signatures, aliases and extension capabilities are retained.
- Native execution still formats If-Match from nil/non-nil signed revision pointers, preserves request/body/success/result behavior and raw MoreHeaders/OmitHeaders/native callbacks, and wraps errors in inspectable OperationError. The bridge preserves the header through adaptation; it does not protect it from native raw overrides or add unconditional conflict fallback.

### `python:network/v2/update_subnet_pool`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordMutationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRecordMutationMicroversion`

- The concrete Go update for this operation emits exact revision_number=N for typed zero/positive/signed values, omits a nil generated condition and preserves the native route/body/response envelope. This all-nine Go matrix is evidence of Go behavior; Python conditional similarity or difference is established from the pinned class/helper sources.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- Final typed replacement selects body and condition together, preserves allowed body extensions, and rejects unsupported query/header/nil options before HTTP. These concrete option semantics do not emulate Python mutable Resource/attrs state.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- Native accepted-code and412 policies remain inspectable with no SDK unconditional fallback; native401/429/503 replay retains the prepared condition/body and live auth despite caller pointer mutation. Python adapter/session retry/error semantics remain a documented difference.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Update sends only raw supplied attrs in subnetpool envelope, resolves canonical/wire aliases in option order, preserves null/false/zero/unknown attrs and sparse member seed while retaining independent literal Wire/Envelope/Header/status. No attrs is a no-HTTP commit, tags default to [] and missing Body fields become null; this Resource has no NetworkResource conditional Header descriptors.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsUpdateAliasesNoopAndSeedWireProjection`, `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsMemberDescriptorsSparseAndMalformedEvidence`.
- Member kwargs remain original query plus constructor seed; fallback/direct lists transpose only Source declared query aliases, canonical keys override wire aliases even when null, known nonserver canonical Body attrs filter descriptor views, unknown list attrs are discarded, and all_projects is consumed without granting access. fields list values remain literal, tags and repeated query values preserve Requests JSON-domain representation. LIST Accept defaults override source Accept while explicit Headers prevail.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsQueryClassificationAliasesAndLocalDescriptors`.
- Lazy owned full pagination supports body/header continuation, marker fallback on short pages, raw-row caps before local filters and single-page choice. Find checks all candidate rows/pages before returning one exact ID/name; late duplicates/rejected/malformed pages remain errors. Cycles and conflicting/foreign continuation are terminal; standard dictionary next links are a declared Go improvement.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsPaginationFullSearchAndRawCaps`.
- Only clean native400/403/404 member rejection falls back; default missing requires a completely successful search and strict missing retains no suppressed native rejection cause. Nonobject row/envelope and constructor cls/connection/microversion/_synchronized collisions retain actual page proof. Source response descriptors retain integer/bool/list/null/tag/tenant aliases; malformed accepted member syntax follows seed while valid wrong shapes fail.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsCleanMissingAndConstructorFailures`, `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsMemberDescriptorsSparseAndMalformedEvidence`.
- Accepted/rejected Read+Close/cancel/source-changed-and-restored-on-Close failures preserve every cause and physical body/header/status, close once, stop before fallback/missing and never resend. Member result receipts for accepted physical failures hold actual metadata without a fabricated descriptor acknowledgement.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsPhysicalFailuresPreserveAcceptedAndRejectedProof`.
- Native retry cannot change/delete/alias/omit selected network version, inject version when absent, add foreign-service/auth/framing headers or mutate JSON body/Content-Type. One503 proof plus native callback causes survives; ordinary retry headers, exact method/URL/JSON, live token and reauthentication are retained.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsRetriesKeepProtocolBodyAndNativeCauses`.
- Source is captured before callbacks; concrete options snapshot raw JSON/maps/pointers and retained callbacks cannot mutate requests/results. Invalid controls/headers/IDs/descriptors/context/options fail before HTTP and preserve causes. One immutable option supports concurrent requests with independent records/evidence.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsOptionsSourceContextAndConcurrentOwnership`.
- Source Resource.find consumes the explicit microversion for member fetch only; fallback LIST and all its pages keep the captured original selected/default microversion. Ordinary Header options apply to both phases. Fallback retry cannot inject the member-only override when selected version differs or is absent; one original LIST503 body/header/status survives. Direct List/Update per-operation overrides remain effective and never mutate source.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsFindMicroversionIsMemberPhaseOnly`.
- Library-owned PagePolicy.NextTruthy reuses SubnetPool response binary64 truthiness: top-level next1e-400/-1e-400/0.0 are falsey and do not request another page, while truthy numeric1e400/1.0 fail URL decoding with exact whole actual page body/header/status. Normal bool/null next remain falsey; independently yielded row receipts retain their own page snapshots.
  Tests: `network/v2/extensions/subnetpools/python_records_contracts_test.go:TestPythonSubnetPoolRecordsNextUsesBinary64TruthinessAndWholePageProof`.

Python/Go differences:

- Go updates a concrete ID with typed UpdateOpts and permitted body fields and returns a native typed value/error. Python accepts ID or existing mutable Resource, applies attrs/descriptor state and commits that Resource.
- Typed Go RevisionNumber nil/zero/signed semantics use the existing native conditional-header capability without a new public helper. This pinned Python resource does not inherit NetworkResource, so forwarded if_revision binds the helper but does not generate If-Match; the Go conditional capability is a native extension difference.
- Go preserves raw native MoreHeaders/OmitHeaders/callback override and retry behavior and reports native412 without SDK unconditional fallback; generic Python adapter/session and mutable/dirty header behavior remain separate.
- Bounded named mappings cover literal identity/name and JSON-domain attrs, not arbitrary Python mutable Resource instances, Connection/session/cache side effects, resource location calculation or deprecation warnings. location is null; whole Source Resource/Connection parity remains separately tracked.
- Go rejects SDK constructor/control attr collisions rather than allowing caller retargeting, and supplies per-call owned Headers/Microversion and typed list-only pagination conveniences. Raw unknown Body/Find kwargs remain data; unsupported list kwargs are discarded.
- Bulk Go maps are sorted when conflicting canonical/wire alias insertion order matters; individual mutation options preserve order. Local map filter evaluation is sorted. Shared JSON query float repr/Unicode runtime and CPython int-string-limit differences remain explicit common representation bounds.
- Owned pager additionally follows standard links:{next:URL}, which pinned Source dict-to-item loop ignores, and rejects ambiguous/foreign/cyclic links with physical proof. Raw caps, marker-on-short-page and complete find search match Source supported behavior.
- Fixed Neutron27.0.0 policy permits admin/owning-project reader or shared-pool reads and admin/owning-project member updates; is_default update is separately admin-only. Examples omit that admin attr and all implicit defaults. No explicit service-only branch exists. neutron-lib3.21.1 rejects shared/ip_version/tenant_id/id PUT fields; raw SDK attrs stay unpoliced and backend rejection authoritative.

### `gophercloud:networking/v2/extensions/layer3/routers.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithUpdateField`

- For this API among all nine native facades, actual HTTP Update retains PUT, effective ResourceBase route, native JSON envelope, description and extension body values, auth and ordinary source headers. Typed RevisionNumber nil/zero/positive/signed emits no header or exact revision_number=N respectively, never a JSON revision field, and returns the native typed response ID without source-client mutation.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- For all nine APIs, final WithUpdateOptions replacement chooses body fields and revision condition together while retaining permitted extension fields and input scalar values. Existing query/header extension and nil-option errors remain preflight OperationError/ErrInvalidOption without HTTP; no new header capability is introduced.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- All nine retain native200 success and their exact201 policy: networks/ports/subnets accept201, the other six reject it. Native412 keeps status/body under OperationError with no SDK fallback or additional request.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`.
- For all nine, native401 reauth,429 backoff and503 retry replay the prepared body and serialized revision through four physical requests with refreshed live auth and permitted native header addition. Caller revision mutation during reauth does not reselect the condition; the shared client remains intact.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Generator checks all nine current native source declarations and regenerated adapter declarations against stored facades, retaining RevisionNumber *int json:-/h:If-Match from final validated cfg.Options and existing Update/WithUpdateOptions/WithUpdateField without new helpers. Source/field/import drift mutations use Network as the representative case; owning-import/lexical alias-shadow and optional-builder/secondary-input tests are generic fixtures, not all-nine drift-case coverage.
  Tests: `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersActualNineNativeAdapters`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersRejectNativeSourceAndRevisionDrift`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersOwningImportsAndLexicalScope`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersPreserveOptionalBuildersAndSecondaryInputs`.

Python/Go differences:

- The facade accepts native concrete UpdateOpts with library-owned private header/body adapter rather than requiring a caller UpdateOptsBuilder. Final WithUpdateOptions selects the original header-tagged concrete fields and body together, while WithUpdateField adds permitted body fields. Public signatures, aliases and extension capabilities are retained.
- Native execution still formats If-Match from nil/non-nil signed revision pointers, preserves request/body/success/result behavior and raw MoreHeaders/OmitHeaders/native callbacks, and wraps errors in inspectable OperationError. The bridge preserves the header through adaptation; it does not protect it from native raw overrides or add unconditional conflict fallback.

### `python:network/v2/update_router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.UpdateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithIfMatch`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRevision`

- The concrete Go update for this operation emits exact revision_number=N for typed zero/positive/signed values, omits a nil generated condition and preserves the native route/body/response envelope. This all-nine Go matrix is evidence of Go behavior; Python conditional similarity or difference is established from the pinned class/helper sources.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- Final typed replacement selects body and condition together, preserves allowed body extensions, and rejects unsupported query/header/nil options before HTTP. These concrete option semantics do not emulate Python mutable Resource/attrs state.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- Native accepted-code and412 policies remain inspectable with no SDK unconditional fallback; native401/429/503 replay retains the prepared condition/body and live auth despite caller pointer mutation. Python adapter/session retry/error semantics remain a documented difference.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Literal-ID update with no Body/Header options performs no HTTP and returns an ID-only native model. Supplied attrs send a single PUT routers/{id} without lookup. A raw If-Match or WithRevision alone is dirty and sends {router:{}}. WithRevision(0) emits revision_number=0; it overrides raw if_match regardless option order, including invalid discarded raw values, and the last revision wins.
  Tests: `network/v2/extensions/layer3/routers/python_mutations_contracts_test.go:TestPythonCoreRouterMutationBodiesDefaultsAndAliases`, `network/v2/extensions/layer3/routers/python_mutations_contracts_test.go:TestPythonCoreRouterRevisionHeaderOnlyPriorityAndClientIsolation`.
- WithAttribute constructor snapshots supplied JSON and Header values; reusable options preserve zero/false/null/empty values, later wire-key values win, canonical Body aliases map through the existing pinned MRO table, unknown extension and query-only attrs stay in Body. is_admin_state_up=>admin_state_up; is_distributed=>distributed; is_ha=>ha; revision_number=>revision (pinned Source override). Invalid options/serialization, non-If-Match headers, query mechanisms, and Resource/Proxy control attrs fail before HTTP.
  Tests: `network/v2/extensions/layer3/routers/python_mutations_contracts_test.go:TestPythonCoreRouterMutationBodiesDefaultsAndAliases`, `network/v2/extensions/layer3/routers/python_mutations_contracts_test.go:TestPythonCoreRouterOptionSnapshotsHeaderRoutingAndPreflight`.
- Inherited if_match Header is separate from Body. String values, string lists joined by comma-space, explicit empty list and null->None are supported, with constructor snapshots and configured client Header isolation. Create accepts raw If-Match; update accepts revision/If-Match, and explicitly selected conditions replace case-insensitive configured If-Match on a cloned client.
  Tests: `network/v2/extensions/layer3/routers/python_mutations_contracts_test.go:TestPythonCoreRouterOptionSnapshotsHeaderRoutingAndPreflight`, `network/v2/extensions/layer3/routers/python_mutations_contracts_test.go:TestPythonCoreRouterRevisionHeaderOnlyPriorityAndClientIsolation`.
- SDK HTTP success codes are 200..399 and native typed result extraction is retained. Router native Extract returns nil model and nil error on 204. No response model is seeded with request ID/attrs. HTTP status/body/header, native JSON decode errors, and pre-canceled/option-callback/post-response context.Canceled plus custom context.Cause remain observable through OperationError.
  Tests: `network/v2/extensions/layer3/routers/python_mutations_contracts_test.go:TestPythonCoreRouterMutationStatusesDecodeAndContextCauses`, `network/v2/extensions/layer3/routers/python_mutations_contracts_test.go:TestPythonCoreRouterRevisionHeaderOnlyPriorityAndClientIsolation`.

Python/Go differences:

- A functional callable JSON mutation profile uses native Go models and a literal update ID, while existing native Create/Update ABI, required validations and result decoders are unchanged.
- HTTP 200..399 is the explicit SDK profile; strict native decode behavior and returned known fields are retained. Router native Extract returns nil model and nil error on 204. Source retains a mutable Resource request view on empty or invalid JSON success bodies; this Go API does not synthesize request attributes, returned unknown fields or partial-response fallback.
- WithRevision accepts int64 (including zero, negative and exact max-int64), rather than arbitrary-precision Python values. Raw If-Match is a JSON string/string-list/null profile; arbitrary Python object str() coercion is unavailable. A selected Header containing CR/LF is rejected before HTTP. Raw Header values discarded by revision are not validated.
- Body controls base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location are rejected. Update id and if_revision must use the literal ID parameter and WithRevision rather than Body extensions. Configured Gophercloud session/client policy is retained.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Full Python Resource/Munch instance input, returning the same mutable instance, comparison against existing dirty/cache state and retaining request attrs or unknown response attrs remain separate from this callable Go profile.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Complete Proxy/Resource control kwargs, computed location/metadata, dynamic microversion, Python descriptor access/conversion and adapter/session/cache/exception policy remain separate.

### `gophercloud:networking/v2/extensions/layer3/floatingips.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithUpdateField`

- For this API among all nine native facades, actual HTTP Update retains PUT, effective ResourceBase route, native JSON envelope, description and extension body values, auth and ordinary source headers. Typed RevisionNumber nil/zero/positive/signed emits no header or exact revision_number=N respectively, never a JSON revision field, and returns the native typed response ID without source-client mutation.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- For all nine APIs, final WithUpdateOptions replacement chooses body fields and revision condition together while retaining permitted extension fields and input scalar values. Existing query/header extension and nil-option errors remain preflight OperationError/ErrInvalidOption without HTTP; no new header capability is introduced.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- All nine retain native200 success and their exact201 policy: networks/ports/subnets accept201, the other six reject it. Native412 keeps status/body under OperationError with no SDK fallback or additional request.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`.
- For all nine, native401 reauth,429 backoff and503 retry replay the prepared body and serialized revision through four physical requests with refreshed live auth and permitted native header addition. Caller revision mutation during reauth does not reselect the condition; the shared client remains intact.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Generator checks all nine current native source declarations and regenerated adapter declarations against stored facades, retaining RevisionNumber *int json:-/h:If-Match from final validated cfg.Options and existing Update/WithUpdateOptions/WithUpdateField without new helpers. Source/field/import drift mutations use Network as the representative case; owning-import/lexical alias-shadow and optional-builder/secondary-input tests are generic fixtures, not all-nine drift-case coverage.
  Tests: `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersActualNineNativeAdapters`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersRejectNativeSourceAndRevisionDrift`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersOwningImportsAndLexicalScope`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersPreserveOptionalBuildersAndSecondaryInputs`.

Python/Go differences:

- The facade accepts native concrete UpdateOpts with library-owned private header/body adapter rather than requiring a caller UpdateOptsBuilder. Final WithUpdateOptions selects the original header-tagged concrete fields and body together, while WithUpdateField adds permitted body fields. Public signatures, aliases and extension capabilities are retained.
- Native execution still formats If-Match from nil/non-nil signed revision pointers, preserves request/body/success/result behavior and raw MoreHeaders/OmitHeaders/native callbacks, and wraps errors in inspectable OperationError. The bridge preserves the header through adaptation; it does not protect it from native raw overrides or add unconditional conflict fallback.

### `python:network/v2/update_ip`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.UpdateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithIfMatch`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithRevision`

- The concrete Go update for this operation emits exact revision_number=N for typed zero/positive/signed values, omits a nil generated condition and preserves the native route/body/response envelope. This all-nine Go matrix is evidence of Go behavior; Python conditional similarity or difference is established from the pinned class/helper sources.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- Final typed replacement selects body and condition together, preserves allowed body extensions, and rejects unsupported query/header/nil options before HTTP. These concrete option semantics do not emulate Python mutable Resource/attrs state.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- Native accepted-code and412 policies remain inspectable with no SDK unconditional fallback; native401/429/503 replay retains the prepared condition/body and live auth despite caller pointer mutation. Python adapter/session retry/error semantics remain a documented difference.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Literal-ID update with no Body/Header options performs no HTTP and returns an ID-only native model. Supplied attrs send a single PUT floatingips/{id} without lookup. A raw If-Match or WithRevision alone is dirty and sends {floatingip:{}}. WithRevision(0) emits revision_number=0; it overrides raw if_match regardless option order, including invalid discarded raw values, and the last revision wins.
  Tests: `network/v2/extensions/layer3/floatingips/python_mutations_contracts_test.go:TestPythonCoreFloatingIPMutationBodiesDefaultsAndAliases`, `network/v2/extensions/layer3/floatingips/python_mutations_contracts_test.go:TestPythonCoreFloatingIPRevisionHeaderOnlyPriorityAndClientIsolation`.
- WithAttribute constructor snapshots supplied JSON and Header values; reusable options preserve zero/false/null/empty values, later wire-key values win, canonical Body aliases map through the existing pinned MRO table, unknown extension and query-only attrs stay in Body. Canonical Body names from the generated FloatingIP/NetworkResource/Resource MRO are reused; floating_network_id, port_id and fixed_ip_address remain literal wire names. Invalid options/serialization, non-If-Match headers, query mechanisms, and Resource/Proxy control attrs fail before HTTP.
  Tests: `network/v2/extensions/layer3/floatingips/python_mutations_contracts_test.go:TestPythonCoreFloatingIPMutationBodiesDefaultsAndAliases`, `network/v2/extensions/layer3/floatingips/python_mutations_contracts_test.go:TestPythonCoreFloatingIPOptionSnapshotsHeaderRoutingAndPreflight`.
- Inherited if_match Header is separate from Body. String values, string lists joined by comma-space, explicit empty list and null->None are supported, with constructor snapshots and configured client Header isolation. Create accepts raw If-Match; update accepts revision/If-Match, and explicitly selected conditions replace case-insensitive configured If-Match on a cloned client.
  Tests: `network/v2/extensions/layer3/floatingips/python_mutations_contracts_test.go:TestPythonCoreFloatingIPOptionSnapshotsHeaderRoutingAndPreflight`, `network/v2/extensions/layer3/floatingips/python_mutations_contracts_test.go:TestPythonCoreFloatingIPRevisionHeaderOnlyPriorityAndClientIsolation`.
- SDK HTTP success codes are 200..399 and native typed result extraction is retained. FloatingIP native Extract returns an ID-less zero model pointer and nil error on 204. No response model is seeded with request ID/attrs. HTTP status/body/header, native JSON decode errors, and pre-canceled/option-callback/post-response context.Canceled plus custom context.Cause remain observable through OperationError.
  Tests: `network/v2/extensions/layer3/floatingips/python_mutations_contracts_test.go:TestPythonCoreFloatingIPMutationStatusesDecodeAndContextCauses`, `network/v2/extensions/layer3/floatingips/python_mutations_contracts_test.go:TestPythonCoreFloatingIPRevisionHeaderOnlyPriorityAndClientIsolation`.

Python/Go differences:

- A functional callable JSON mutation profile uses native Go models and a literal update ID, while existing native Create/Update ABI, required validations and result decoders are unchanged.
- HTTP 200..399 is the explicit SDK profile; strict native decode behavior and returned known fields are retained. FloatingIP native Extract returns an ID-less zero model pointer and nil error on 204. Source retains a mutable Resource request view on empty or invalid JSON success bodies; this Go API does not synthesize request attributes, returned unknown fields or partial-response fallback.
- WithRevision accepts int64 (including zero, negative and exact max-int64), rather than arbitrary-precision Python values. Raw If-Match is a JSON string/string-list/null profile; arbitrary Python object str() coercion is unavailable. A selected Header containing CR/LF is rejected before HTTP. Raw Header values discarded by revision are not validated.
- Body controls base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location are rejected. Update id and if_revision must use the literal ID parameter and WithRevision rather than Body extensions. Configured Gophercloud session/client policy is retained.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Full Python Resource/Munch instance input, returning the same mutable instance, comparison against existing dirty/cache state and retaining request attrs or unknown response attrs remain separate from this callable Go profile.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Complete Proxy/Resource control kwargs, computed location/metadata, dynamic microversion, Python descriptor access/conversion and adapter/session/cache/exception policy remain separate.

### `gophercloud:networking/v2/extensions/qos/policies.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithUpdateField`

- For this API among all nine native facades, actual HTTP Update retains PUT, effective ResourceBase route, native JSON envelope, description and extension body values, auth and ordinary source headers. Typed RevisionNumber nil/zero/positive/signed emits no header or exact revision_number=N respectively, never a JSON revision field, and returns the native typed response ID without source-client mutation.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- For all nine APIs, final WithUpdateOptions replacement chooses body fields and revision condition together while retaining permitted extension fields and input scalar values. Existing query/header extension and nil-option errors remain preflight OperationError/ErrInvalidOption without HTTP; no new header capability is introduced.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- All nine retain native200 success and their exact201 policy: networks/ports/subnets accept201, the other six reject it. Native412 keeps status/body under OperationError with no SDK fallback or additional request.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`.
- For all nine, native401 reauth,429 backoff and503 retry replay the prepared body and serialized revision through four physical requests with refreshed live auth and permitted native header addition. Caller revision mutation during reauth does not reselect the condition; the shared client remains intact.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Generator checks all nine current native source declarations and regenerated adapter declarations against stored facades, retaining RevisionNumber *int json:-/h:If-Match from final validated cfg.Options and existing Update/WithUpdateOptions/WithUpdateField without new helpers. Source/field/import drift mutations use Network as the representative case; owning-import/lexical alias-shadow and optional-builder/secondary-input tests are generic fixtures, not all-nine drift-case coverage.
  Tests: `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersActualNineNativeAdapters`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersRejectNativeSourceAndRevisionDrift`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersOwningImportsAndLexicalScope`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersPreserveOptionalBuildersAndSecondaryInputs`.

Python/Go differences:

- The facade accepts native concrete UpdateOpts with library-owned private header/body adapter rather than requiring a caller UpdateOptsBuilder. Final WithUpdateOptions selects the original header-tagged concrete fields and body together, while WithUpdateField adds permitted body fields. Public signatures, aliases and extension capabilities are retained.
- Native execution still formats If-Match from nil/non-nil signed revision pointers, preserves request/body/success/result behavior and raw MoreHeaders/OmitHeaders/native callbacks, and wraps errors in inspectable OperationError. The bridge preserves the header through adaptation; it does not protect it from native raw overrides or add unconditional conflict fallback.

### `python:network/v2/update_qos_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.UpdateCloudRecord`

- Retained native Update API: The concrete Go update for this operation emits exact revision_number=N for typed zero/positive/signed values, omits a nil generated condition and preserves the native route/body/response envelope. This all-nine Go matrix is evidence of Go behavior; Python conditional similarity or difference is established from the pinned class/helper sources.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- Retained native Update API: Final typed replacement selects body and condition together, preserves allowed body extensions, and rejects unsupported query/header/nil options before HTTP. These concrete option semantics do not emulate Python mutable Resource/attrs state.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- Retained native Update API: Native accepted-code and412 policies remain inspectable with no SDK unconditional fallback; native401/429/503 replay retains the prepared condition/body and live auth despite caller pointer mutation. Python adapter/session retry/error semantics remain a documented difference.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Fresh scalar-ID commit writes raw canonical/wire aliases, explicit null and unknown extensions, with boolean/list descriptor casts applied only to the returned view. Partial response raw getter precedence and passive ID evidence are preserved.
  Tests: `network/v2/extensions/qos/policies/python_update_record_test.go:TestQoSPolicyUpdateRecordRawWireAliasesAndDescriptorView`, `network/v2/extensions/qos/policies/python_update_record_test.go:TestQoSPolicyUpdateRecordUnknownAndNullRemainDirtyWithoutConditionalHeader`, `network/v2/extensions/qos/policies/python_update_record_test.go:TestQoSPolicyUpdateRecordResponseWireGetterPrecedenceAndEvidence`.
- Empty and helper-only if_revision commits send no HTTP. if_revision does not become If-Match for this Resource/TagMixin class; bound constructor collisions fail before transport.
  Tests: `network/v2/extensions/qos/policies/python_update_record_test.go:TestQoSPolicyUpdateRecordEmptyAndIgnoredRevisionDoNotSend`, `network/v2/extensions/qos/policies/python_update_record_test.go:TestQoSPolicyUpdateRecordConstructorCollisionsAndSourceGuardsPreflight`.
- Options snapshot raw bytes and transport policy; actual rejection/read/Close/cancel errors are terminal without fallback or replay. Returned records bind the existing private borrowed cloud no-op ledger.
  Tests: `network/v2/extensions/qos/policies/python_update_record_test.go:TestQoSPolicyUpdateRecordOptionsOwnRawBytesAndTransport`, `network/v2/extensions/qos/policies/python_update_record_test.go:TestQoSPolicyUpdateRecordActualRejectionsAndPhysicalFaultsNeverReplay`, `network/v2/extensions/qos/policies/python_update_record_test.go:TestQoSPolicyUpdateRecordBindsExistingBorrowedNoopLane`.

Python/Go differences:

- UpdateRecord owns a fresh scalar string-ID JSON mutation and typed transport controls. Raw base_path is explicitly unsupported; arbitrary Python Resource/object inputs and shared session/cache/recommit contracts remain separate. Existing UpdateCloudRecord provides a restricted trusted borrowed-resource equality/no-op lane, not arbitrary mutable Resource identity.
- Finite UTF-8 JSON and shared REST decoding do not reproduce arbitrary Requests codec/sniffing behavior. Live-cloud and release validation remain open; no supported/full-parity claim is made.
- The native Update API retains its explicit revision header/callback/retry capabilities. The new Source-shaped profile consumes if_revision without creating a conditional header for QoSPolicy.

### `gophercloud:networking/v2/extensions/security/groups.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithUpdateField`

- For this API among all nine native facades, actual HTTP Update retains PUT, effective ResourceBase route, native JSON envelope, description and extension body values, auth and ordinary source headers. Typed RevisionNumber nil/zero/positive/signed emits no header or exact revision_number=N respectively, never a JSON revision field, and returns the native typed response ID without source-client mutation.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- For all nine APIs, final WithUpdateOptions replacement chooses body fields and revision condition together while retaining permitted extension fields and input scalar values. Existing query/header extension and nil-option errors remain preflight OperationError/ErrInvalidOption without HTTP; no new header capability is introduced.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- All nine retain native200 success and their exact201 policy: networks/ports/subnets accept201, the other six reject it. Native412 keeps status/body under OperationError with no SDK fallback or additional request.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`.
- For all nine, native401 reauth,429 backoff and503 retry replay the prepared body and serialized revision through four physical requests with refreshed live auth and permitted native header addition. Caller revision mutation during reauth does not reselect the condition; the shared client remains intact.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Generator checks all nine current native source declarations and regenerated adapter declarations against stored facades, retaining RevisionNumber *int json:-/h:If-Match from final validated cfg.Options and existing Update/WithUpdateOptions/WithUpdateField without new helpers. Source/field/import drift mutations use Network as the representative case; owning-import/lexical alias-shadow and optional-builder/secondary-input tests are generic fixtures, not all-nine drift-case coverage.
  Tests: `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersActualNineNativeAdapters`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersRejectNativeSourceAndRevisionDrift`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersOwningImportsAndLexicalScope`, `internal/cmd/sdkgen/reflected_headers_test.go:TestReflectedHeadersPreserveOptionalBuildersAndSecondaryInputs`.

Python/Go differences:

- The facade accepts native concrete UpdateOpts with library-owned private header/body adapter rather than requiring a caller UpdateOptsBuilder. Final WithUpdateOptions selects the original header-tagged concrete fields and body together, while WithUpdateField adds permitted body fields. Public signatures, aliases and extension capabilities are retained.
- Native execution still formats If-Match from nil/non-nil signed revision pointers, preserves request/body/success/result behavior and raw MoreHeaders/OmitHeaders/native callbacks, and wraps errors in inspectable OperationError. The bridge preserves the header through adaptation; it does not protect it from native raw overrides or add unconditional conflict fallback.

### `python:network/v2/update_security_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.UpdateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithIfMatch`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRevision`

- The concrete Go update for this operation emits exact revision_number=N for typed zero/positive/signed values, omits a nil generated condition and preserves the native route/body/response envelope. This all-nine Go matrix is evidence of Go behavior; Python conditional similarity or difference is established from the pinned class/helper sources.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsAllNativeUpdates`.
- Final typed replacement selects body and condition together, preserves allowed body extensions, and rejects unsupported query/header/nil options before HTTP. These concrete option semantics do not emulate Python mutable Resource/attrs state.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsReplacementChoosesBodyAndCondition`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities`.
- Native accepted-code and412 policies remain inspectable with no SDK unconditional fallback; native401/429/503 replay retains the prepared condition/body and live auth despite caller pointer mutation. Python adapter/session retry/error semantics remain a documented difference.
  Tests: `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeStatusesAndConflict`, `api/neutron_revision_contracts_test.go:TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition`.
- Literal-ID update with no Body/Header options performs no HTTP and returns a new ID-only native model. Supplied attrs send one PUT security-groups/{id}, with no lookup. Raw If-Match/WithRevision alone sends an empty security_group envelope. WithRevision(0) sends revision_number=0, overrides raw if_match regardless option order, discards invalid raw Header values before validation, and later revisions win.
  Tests: `network/v2/extensions/security/groups/python_mutations_contracts_test.go:TestPythonCoreSecGroupMutationBodiesDefaultsAndAliases`, `network/v2/extensions/security/groups/python_mutations_contracts_test.go:TestPythonCoreSecGroupRevisionHeaderOnlyPriorityAndClientIsolation`.
- WithAttribute snapshots JSON at constructor time; reusable options preserve false/zero/null/empty values, later wire-key values win, and aliases use the pinned generated Body MRO table: is_shared=>shared; name/description/stateful/security_group_rules/project_id/tenant_id retain Source Body wire names. Unknown and query-only attrs stay in Body. Invalid option/serialization/capability and Resource control attrs fail before HTTP.
  Tests: `network/v2/extensions/security/groups/python_mutations_contracts_test.go:TestPythonCoreSecGroupMutationBodiesDefaultsAndAliases`, `network/v2/extensions/security/groups/python_mutations_contracts_test.go:TestPythonCoreSecGroupOptionSnapshotsHeaderRoutingAndPreflight`.
- Inherited if_match Header is separate from Body. String values, string lists joined by comma-space, explicit empty list and null->None are supported, with constructor snapshots and configured client Header isolation. Create accepts raw If-Match; update accepts revision/If-Match, and explicitly selected conditions replace case-insensitive configured If-Match on a cloned client.
  Tests: `network/v2/extensions/security/groups/python_mutations_contracts_test.go:TestPythonCoreSecGroupOptionSnapshotsHeaderRoutingAndPreflight`, `network/v2/extensions/security/groups/python_mutations_contracts_test.go:TestPythonCoreSecGroupRevisionHeaderOnlyPriorityAndClientIsolation`.
- SDK HTTP success codes are 200..399 with unchanged native typed extraction. SecGroup native Extract returns nil model and nil error on 204. Results are not seeded with request attrs/ID. OperationError retains native decode errors, HTTP status/body/header and pre-canceled/option-callback/post-response context.Canceled plus custom context.Cause.
  Tests: `network/v2/extensions/security/groups/python_mutations_contracts_test.go:TestPythonCoreSecGroupMutationStatusesDecodeAndContextCauses`, `network/v2/extensions/security/groups/python_mutations_contracts_test.go:TestPythonCoreSecGroupRevisionHeaderOnlyPriorityAndClientIsolation`.

Python/Go differences:

- A functional callable JSON mutation profile uses native Go models and a literal update ID, while existing native Create/Update ABI, required validations and result decoders are unchanged.
- The explicit SDK success profile is 200..399 and strict native decode remains. SecGroup native Extract returns nil model and nil error on 204. Source can retain a mutable request Resource on empty/invalid JSON success; native Go results do not store response unknown fields, merge request attrs, or apply descriptor access/coercion.
- WithRevision accepts int64 (including zero, negative and exact max-int64), rather than arbitrary-precision Python values. Raw If-Match is a JSON string/string-list/null profile; arbitrary Python object str() coercion is unavailable. A selected Header containing CR/LF is rejected before HTTP. Raw Header values discarded by revision are not validated.
- Body controls base_path, microversion, connection, _synchronized, resource_type, __conflicting_attrs and location are rejected. Update id and if_revision must use the literal ID parameter and WithRevision rather than Body extensions. Configured Gophercloud session/client policy is retained.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Full Python Resource/Munch instance input, returning the same mutable instance, comparison against existing dirty/cache state and retaining request attrs or unknown response attrs remain separate from this callable Go profile.
- Separately tracked SDK-R1/C1/S1 or deployment boundary: Complete Proxy/Resource control kwargs, computed location/metadata, dynamic microversion, Python descriptor access/conversion and adapter/session/cache/exception policy remain separate.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
network = conn.network.update_network(network_id, if_revision=0, name="renamed")
```

```go
package main

import (
    "context"
    "errors"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
    "github.com/JSYoo5B/go-openstacksdk/network/v2/networks"
)

func renameNetwork(ctx context.Context, service *networkv2.Service, id, name string, revision int) (*networks.Network, error) {
    options := networks.UpdateOpts{
        Name: &name,
        RevisionNumber: &revision,
    }
    updated, err := service.Networks.Update(ctx, id, networks.UpdateOpts{},
        networks.WithUpdateOptions(options))
    if err != nil {
        var response gophercloud.ErrUnexpectedResponseCode
        if errors.As(err, &response) {
            fmt.Printf("HTTP%d body=%q headers=%v\n",
                response.Actual, response.Body, response.ResponseHeader)
        }
    }
    return updated, err
}

func main() {}
```

## References

- [networks](networks/api_generated.go)
- [ports](ports/api_generated.go)
- [subnets](subnets/api_generated.go)
- [routers](extensions/layer3/routers/api_generated.go)
- [floatingips](extensions/layer3/floatingips/api_generated.go)
- [groups](extensions/security/groups/api_generated.go)
- [trunks](extensions/trunks/api_generated.go)
- [subnetpools](extensions/subnetpools/api_generated.go)
- [policies](extensions/qos/policies/api_generated.go)
- [pinned Network Proxy._update](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L213-L229)
- [NetworkResource.if_match](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_base.py#L23-L29)
- [Trunk](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/trunk.py#L22)
- [SubnetPool](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet_pool.py#L23)
- [QoSPolicy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/qos_policy.py#L21)
- [dirty/cache·no-op commit·session](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881-L1993)
- [Python Update Record](extensions/qos/policies/python-update-record.md)
