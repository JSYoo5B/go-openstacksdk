# Python Parity Core

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/delete_agent`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE agents/{id}, succeeds on 204, ignores 404 by default and returns resource.ErrNotFound with WithMissingError.
  Tests: `network/v2/extensions/agents/python_parity_core_test.go:TestPythonParityCoreAgentUpdateAndDelete`.

Python/Go differences:

- Python accepts an Agent Resource or an ID; Go Remove takes resource.ID.

### `python:network/v2/get_agent`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.GetWithFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.Get`

- GetWithFields sends GET agents/{id} with each Python Agent Body attribute name mapped to its wire name as repeated fields query keys, keeping caller order, repeats, empty and unknown names, and decodes the selected attributes into the typed model.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersSendAliasedRepeatedFields`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- GetWithFields with nil or empty fields sends the plain member GET without a query, which is Python's fields=None request, and a sparse response decodes with zero values for omitted attributes.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersWithoutFieldsSendPlainGet`.
- GetWithFields accepts HTTP 200 through 399 like the Python read, and returns 404 and other failures as the native gophercloud status error wrapped in resource.OperationError with the operation name.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersAcceptPythonStatusesAndWrapFailures`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.
- Get sends a bare GET agents/{id} and decodes the agent, matching get_agent without fields.
  Tests: `network/v2/extensions/agents/python_parity_core_test.go:TestPythonParityCoreAgentDHCPNetworkRoutes`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model.
- Go GetWithFields returns the Gophercloud v2.15.0 typed model, so attributes Neutron returns outside that model are dropped and attributes the server omits are zero values instead of None; Python returns a sparse Resource.
- The SDK field getter accepts HTTP 200..399 and keeps unknown field names in the query like Python; plain API.Get keeps the native status policy.

### `python:network/v2/update_agent`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithUpdateField`

- Update sends PUT agents/{id} with only the given description and admin_state_up inside the agent envelope and decodes the returned agent.
  Tests: `network/v2/extensions/agents/python_parity_core_test.go:TestPythonParityCoreAgentUpdateAndDelete`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model.
- Python maps is_admin_state_up to admin_state_up; Go UpdateOpts uses AdminStateUp and WithUpdateField carries other keys.
- Python sends no request when no attribute changed; Go Update always sends PUT with the envelope.
- Go option structs omit nil and empty core fields and WithUpdateField cannot replace a core key, so an explicit Python None cannot be sent as JSON null for those fields.

### `python:network/v2/add_dhcp_agent_to_network`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.ScheduleDHCPNetwork`

- ScheduleDHCPNetwork sends POST agents/{id}/dhcp-networks with {"network_id": id}.
  Tests: `network/v2/extensions/agents/python_parity_core_test.go:TestPythonParityCoreAgentDHCPNetworkRoutes`.

Python/Go differences:

- Python returns the decoded response body; Go returns only an error.
- Python's Resource action uses Proxy.request without raise_from_response, so an HTTP error returns the error body; Go returns the HTTP error.
- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model.

### `python:network/v2/remove_dhcp_agent_from_network`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.RemoveDHCPNetwork`

- RemoveDHCPNetwork sends DELETE agents/{id}/dhcp-networks/{network_id}.
  Tests: `network/v2/extensions/agents/python_parity_core_test.go:TestPythonParityCoreAgentDHCPNetworkRoutes`.

Python/Go differences:

- Python also sends {"network_id": id} as a JSON body on the DELETE; Go sends no body.
- Python's Resource action uses Proxy.request without raise_from_response, so an HTTP error returns the error body; Go returns the HTTP error.
- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model.

### `python:network/v2/dhcp_agent_hosting_networks`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.ListDHCPNetworkRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithDHCPNetworkRecordsOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithDHCPNetworkRecordsQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithDHCPNetworkRecordsMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithDHCPNetworkRecordsPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.WithDHCPNetworkRecordsHeader`

- ListDHCPNetworkRecords sends GET agents/{id}/dhcp-networks with Network query keys translated to wire names (is_shared as shared=True) and list values repeated, drops unknown keys, follows a /v2.0 relative networks_links next href, then continues with marker=<last id> after a page without links while a limit is present until an empty page, and yields the typed Network, raw Fields without self, AgentID and status.
  Tests: `network/v2/extensions/agents/python_dhcp_networks_test.go:TestPythonDHCPNetworkRecordsQueryAndPaging`.
- Network Body attributes outside the query mapping such as mtu, subnet_ids and dns_domain=None filter rows locally with Python equality, max_items is sent as limit when no limit is given and counts filtered rows before stopping, and paginated=False reads only the first page.
  Tests: `network/v2/extensions/agents/python_dhcp_networks_test.go:TestPythonDHCPNetworkRecordsClientFiltersAndControls`.
- A blank agent ID, a nil option, a call-control query key such as max_items, a blank query key, a negative max_items, an unencodable value and a nil API fail before HTTP with resource.ErrInvalidOption in a ListDHCPNetworkRecords resource.OperationError, and a 404 is returned as the Gophercloud status error wrapped for agents.
  Tests: `network/v2/extensions/agents/python_dhcp_networks_test.go:TestPythonDHCPNetworkRecordsRejectsInputsAndErrorStatuses`.
- ListDHCPNetworks sends a bare GET agents/{id}/dhcp-networks and decodes the networks array, matching the Python call without query arguments.
  Tests: `network/v2/extensions/agents/python_parity_core_test.go:TestPythonParityCoreAgentDHCPNetworkRoutes`.

Python/Go differences:

- Python accepts the agent as an ID or an Agent Resource; Go takes an ID string, and Python base_path and jmespath_filters are not supported.
- Python compares client filters against typed Resource attributes (for example int mtu); Go compares the raw response JSON with Python equality.
- Python yields DHCPAgentHostingNetwork Resources; Go yields DHCPNetworkRecord values with the Gophercloud networks.Network decode and the raw row. Go encodes query keys in sorted order.

### `python:network/v2/create_segment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.WithCreateField`

- Create sends POST segments with the given fields inside the segment envelope; a missing network_type fails before HTTP.
  Tests: `network/v2/extensions/segments/python_parity_core_test.go:TestPythonParityCoreSegmentCreateUpdate`.

Python/Go differences:

- Python sends whatever attributes it is given and leaves validation to Neutron; Go requires network_id and network_type before HTTP.
- Go CreateOpts omits empty strings and a zero segmentation_id, and cannot send JSON null for them; Neutron applies the same defaults when they are omitted.

### `python:network/v2/delete_segment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE segments/{id}, succeeds on 204, ignores 404 by default and returns resource.ErrNotFound with WithMissingError.
  Tests: `network/v2/extensions/segments/python_parity_core_test.go:TestPythonParityCoreSegmentDeleteIgnoreMissing`.

Python/Go differences:

- Python accepts a Segment Resource or an ID; Go Remove takes resource.ID.

### `python:network/v2/get_segment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.API.GetWithFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.API.Get`

- GetWithFields sends GET segments/{id} with each Python Segment Body attribute name mapped to its wire name as repeated fields query keys, keeping caller order, repeats, empty and unknown names, and decodes the selected attributes into the typed model.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersSendAliasedRepeatedFields`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- GetWithFields with nil or empty fields sends the plain member GET without a query, which is Python's fields=None request, and a sparse response decodes with zero values for omitted attributes.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersWithoutFieldsSendPlainGet`.
- GetWithFields accepts HTTP 200 through 399 like the Python read, and returns 404 and other failures as the native gophercloud status error wrapped in resource.OperationError with the operation name.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersAcceptPythonStatusesAndWrapFailures`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.
- Get sends a bare GET segments/{id} and decodes the segment, matching get_segment without fields.
  Tests: `network/v2/extensions/segments/python_parity_core_test.go:TestPythonParityCoreSegmentListAndGet`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model.
- Go GetWithFields returns the Gophercloud v2.15.0 typed model, so attributes Neutron returns outside that model are dropped and attributes the server omits are zero values instead of None; Python returns a sparse Resource.
- The SDK field getter accepts HTTP 200..399 and keeps unknown field names in the query like Python; plain API.Get keeps the native status policy.

### `python:network/v2/update_segment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/segments.WithUpdateField`

- Update sends PUT segments/{id} with name, empty description, segmentation_id 0 and extension keys inside the segment envelope.
  Tests: `network/v2/extensions/segments/python_parity_core_test.go:TestPythonParityCoreSegmentCreateUpdate`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model.
- Python sends no request when no attribute changed; Go Update always sends PUT with the envelope.
- Go option structs omit nil and empty core fields and WithUpdateField cannot replace a core key, so an explicit Python None cannot be sent as JSON null for those fields.

### `python:network/v2/extensions`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordListConflictingAttrs`

- List sends GET extensions and decodes alias, name, description and the raw updated string.
  Tests: `network/v2/extensions/python_parity_core_test.go:TestPythonParityCoreExtensionListAndGetWithoutFallback`.
- Source-shaped extension list implements all six effective Body fields/defaults with direct Resource MRO, actual alias alternate ID by primary presence, canonical/wire updated alias ordering and unknown body/raw timestamps. Nested links type=list/list_type=dict preserves top null, wraps nonlist and converts every nonmapping element including null to an empty dictionary, independently of actual Wire/whole-page/Header/status.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsDefaultMROAliasIDAndNestedLinksCast`.
- Inherited limit/marker are the only server queries. All declared Body attrs including id/alias/name/description/links/updated_at are owned local filters; unsupported fields/wire updated/unknown kwargs are discarded before value encoding. Dictionary subset, bool-number and loaded CPython float equality are shared library behavior. No member fetch/name query hint is invented.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionListRecordsAllBodyFiltersAndQueryControlsAreLibraryOwned`.
- Shared inherited pager handles actual next/Link/plural links/marker with original primary-id presence and alias alternate only when absent. Raw cap counts excluded rows and stops before unconsumed constructor/foreign continuation. Actual primary null/empty never switches to alias; page/late failures preserve whole actual receipt and AllRecords is atomic.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionListAlternateMarkerPrivateProofAndRawCaps`, `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsMalformedEnvelopeAndLateErrorsKeepActualProof`.
- Actual Source Proxy receives an opaque unconsumed generator for deprecated JMESPath: projection null, literal JSON, identity equality, runtime errors and nested generators preserve zero HTTP until consumption. Original expression remains separate from restored unknown JMES; one-pass conflicting attrs and concrete header/version/base-path controls use the shared normalizer. No scalar/object expression is fabricated into a Resource or successful HTTP receipt.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionProxyJMESGeneratorResultsAndSourceConflictControls`.
- Source/option snapshots, initialized callback maps and concurrently reusable factories retain request ownership. Nil/bound duplicate options and malformed constructor/query/base paths fail before HTTP. Physical read/Close/cancel/source faults and native pointer401 retain all causes/actual evidence without invoking native retry/reauth; clean retry/token renewal preserves semantics and forbidden auth mutation cannot turn original503 into success.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsSnapshotsNilMapsConcurrentReuseAndPreflight`, `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsPhysicalFaultsNeverFallbackRetryOrReauth`, `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsCleanRetryTokenRenewalAndProtocolGuard`.
- Source JSON paging honors present root links over plural links, ignores dictionary values, selects first literal lowercase rel next, falls back to top-level next and preserves actual malformed links receipt. HTTP Link interoperability remains explicit.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsSourceLinkPrecedenceAndActualMalformedProof`.

Python/Go differences:

- Python can send the default limit and marker query keys; the Go extension list accepts no query.
- Go reads one page and does not follow extensions_links; Neutron returns the extension list in one response.
- Python filters Body attributes such as name locally; the Go caller filters the returned rows.
- Owned Extension records are sibling to existing native Get/List and preserve full declared/default JSON and actual evidence. Extension derives from Resource directly, with no invented NetworkResource Header/tag/revision declarations.
- Go immutable eager JSON view differs from Source lazy/mutable Resource/Computed/current_location/cache/dirty lifecycle and arbitrary class/session injection. location is SDK-owned null and Wire remains actual. Known runtime/constructor/control replacement is bounded explicitly rather than a claim of full shared Resource parity.
- Default/selected network microversion has no Extension class maximum to negotiate. Concrete safe header maps and fixed source/origin/path/filter/continuation policies bound broad Source adapters/URI percent-formatting. Empty path means default in Go, and HTTP Link interoperability is explicit. Source100..399 acceptance and JSON link precedence are opted in; invalid UTF8 is rejected rather than replaced.
- Primitive scalar/list/dictionary descriptors, nested links dict element null handling and CPython-loaded numeric equality use shared library contracts. Source runtime custom objects/subclasses and configurable integer-string runtime constraints are outside this JSON interface; unknown actual JSON/large integers remain raw and no user decoder/builder is required.
- Neutron27 extension controller alias/list routes have no dedicated ADMIN gate. Catalog/auth middleware and deployment extension visibility remain authoritative; fixture/source/race/main evidence does not assert anonymous access, elevated roles or enabled drivers/backends.
- Typed ListRecords/AllRecords yield resource-only values; ProxyListRecords owns arbitrary JSON or nested opaque generator results. The full Source generator branch is preserved rather than evaluated over collected arrays, and mutable Python iterator/session lifecycles remain separate shared SDK foundations.

### `python:network/v2/find_extension`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.WithRecordFindListBasePath`

- Get sends GET extensions/{name} and returns the 404 without listing extensions.
  Tests: `network/v2/extensions/python_parity_core_test.go:TestPythonParityCoreExtensionListAndGetWithoutFallback`.
- GET-first Source find escapes the literal identity segment, preserves raw member kwargs/constructor seed, consumes formals and projects actual response aliases/defaults without confusing seed with Wire. Actual explicit null id wins over requested seed; accepted empty/nonJSON responses retain seed/default Resource with Wire nil and independent actual HTTP evidence.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionFindMemberSeedExplicitNullAndPrivateActualEvidence`.
- Only clean member400/403/404 trigger fallback. Extension has no name query so no automatic name hint; complete list searches exact alias-derived ID or name and detects late duplicates. Successful full absence is defaultnil,nil; strictErrNotFound, LIST403 and local Body conditions retain distinct phases without hiding permission or late/physical failures.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionFindAliasNameFallbackHasNoServerNameHintAndDetectsLateAmbiguity`, `network/v2/extensions/python_record_contracts_test.go:TestExtensionFindMissingForbiddenAndLocalBodyFiltersPreserveTheirPhases`.
- Owned ListBasePath is a fallback-only route override with cloned pointer; the member stays at default extensions/{literal}. Explicit Find microversion affects only GET and fallback list keeps original selected/default version. Shared safe header policy, named/typed formal values and immutable selected client remain consistent through both phases.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionFindListBasePathAndGetOnlyVersionAreIndependent`.
- Library-owned options/initialized maps and source guards surround callbacks. Bound identity/ignore_missing/class controls reject before application marshaling, unknown response JSON remains visible, and nested links cast uses the common descriptor. Physical/cancel/source failures never become clean404 fallback or missing success, native401 pointer retains actual receipt/handling causes, and clean renewal/retry preserves live tokens with fixed protocol ownership.
  Tests: `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsSnapshotsNilMapsConcurrentReuseAndPreflight`, `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsPhysicalFaultsNeverFallbackRetryOrReauth`, `network/v2/extensions/python_record_contracts_test.go:TestExtensionRecordsCleanRetryTokenRenewalAndProtocolGuard`.
- Extension six-control collection GET retains unknown constructor seed while distinguishing physical Wire/Envelope; alias-only GET response leaves requested seed id, explicit null id wins, fallback alias-derived ID and updated/updated_at literal and links element projections remain resource-specific. Malformed selected envelopes retain actual accepted response proof.
  Tests: `network/v2/extensions/fetch_controls_test.go:TestExtensionFindFetchControlsCollectionKeepsConstructorSeedAndAliasProjection`, `network/v2/extensions/fetch_controls_test.go:TestExtensionFindFetchControlsEnvelopeNullIDAndMalformedProof`.
- Raw headers remain GET mapping-key query values and become fallback HTTP headers with case-insensitive raw-header precedence over typed headers; fallback base path is independent, invalid truthy headers fail only after rejected GET reaches LIST, and no invented name query hint is added for Extension.
  Tests: `network/v2/extensions/fetch_controls_test.go:TestExtensionFindFetchControlsRawHeadersUseGETQueryThenFallbackTransport`, `network/v2/extensions/fetch_controls_test.go:TestExtensionFindFetchControlsRawHeadersValidateOnlyAtFallback`.
- Raw options are owned, repeated calls invoke the stateless reader without cache headers, custom HTTP error causes retain physical proof and clean fallback semantics, while read/Close/cancel faults remain terminal without fallback.
  Tests: `network/v2/extensions/fetch_controls_test.go:TestExtensionFindFetchControlsOwnOptionsWithoutCacheAndKeepHTTPCause`, `network/v2/extensions/fetch_controls_test.go:TestExtensionFindFetchControlsPhysicalFaultsNeverFallback`, `internal/neutronfind/find_fetch_route_test.go:TestFindWithFetchControlsDefaultRouteKeepsCanonicalContinuationAlias`, `internal/neutronfind/find_fetch_route_test.go:TestFindWithFetchControlsExplicitRouteKeepsContinuationInsideSelectedCollection`.

Python/Go differences:

- Owned Extension records are sibling to existing native Get/List and preserve full declared/default JSON and actual evidence. Extension derives from Resource directly, with no invented NetworkResource Header/tag/revision declarations.
- Go immutable eager JSON view differs from Source lazy/mutable Resource/Computed/current_location/cache/dirty lifecycle and arbitrary class/session injection. location is SDK-owned null and Wire remains actual. Known runtime/constructor/control replacement is bounded explicitly rather than a claim of full shared Resource parity.
- Default/selected network microversion has no Extension class maximum to negotiate. Concrete safe header maps and fixed source/origin/path/filter/continuation policies bound broad Source adapters/URI percent-formatting. An empty typed fallback-only ListBasePath retains the selected path, whereas the new raw base_path="" selects the service root. HTTP Link interoperability is explicit. Source100..399 acceptance and JSON link precedence are opted in; invalid UTF8 is rejected rather than replaced.
- Primitive scalar/list/dictionary descriptors, nested links dict element null handling and CPython-loaded numeric equality use shared library contracts. Source runtime custom objects/subclasses and configurable integer-string runtime constraints are outside this JSON interface; unknown actual JSON/large integers remain raw and no user decoder/builder is required.
- Neutron27 extension controller alias/list routes have no dedicated ADMIN gate. Catalog/auth middleware and deployment extension visibility remain authoritative; fixture/source/race/main evidence does not assert anonymous access, elevated roles or enabled drivers/backends.
- Go takes explicit string identity and owned concrete option snapshots; arbitrary Python Resource/Adapter objects, overridden constructors and runtime kwargs/control replacement are separately tracked shared contracts. Returned seeded member ID is the Source constructor view and is not represented as actual Wire identity.
- Owned FindRecord explicitly opts into six raw fetch controls while preserving Extension unknown constructor Body, alias-derived alternate ID and updated/updated_at literal projection. URI formatting is limited to %(id)s and %%, with finite UTF-8 JSON/string identities and string-valued fallback headers in safe service-local routes. Other leaf/native find profiles remain unchanged. skip_cache invokes the stateless reader without an invented cache header, and bypass of caller custom transport cache is not claimed; HTTP messages preserve Go causes rather than Python exception classes.

### `python:network/v2/network_ip_availabilities`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordListConflictingAttrs`

- List sends GET network-ip-availabilities with network_name, ip_version, project_id, sort_key and fields and decodes totals, used counts and subnet rows from one page.
  Tests: `network/v2/extensions/networkipavailabilities/python_parity_core_test.go:TestPythonParityCoreIPAvailabilityListAndGet`.
- Owned nullable Resource descriptors preserve big integer/list/dict/tenant fallback while original Wire/Envelope remain separate; network_id does not synthesize id.
  Tests: `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityRecordsOwnedProjectionAndRawReceipt`.
- Lazy allpage list splits exact QueryParameters from descriptor-local filters, discards unknown before value encoding and follows plural next links.
  Tests: `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityRecordsAllPagesServerQueriesAndLocalFilters`.
- Raw-row caps stop before invalid tail/foreign link, option snapshots and GET-only version remain private; physical errors suppress callbacks and unsafe preflight emits no HTTP.
  Tests: `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityRecordsCapsVersionsAndOwnedOptions`, `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityRecordsPhysicalFailuresAndPreflight`.
- Proxy JMESPath consumes opaque lazy generator: @ and projection require no HTTP, unsupported length fails before request.
  Tests: `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityProxyGeneratorAndJSONProjection`.

Python/Go differences:

- Existing native List/Get/GetFields retain their typed onepage/string-counter behavior; new SDK-owned records supply allpage/localfilter/descriptor evidence without caller builders.
- Pinned Resource._get_one_match compares inherited id/name; unused name_attribute network_name and ordinary network_id are not synthesized into lookup identity. This corrects the older assessment that said fallback matches network_id/network_name.
- Record JSON descriptor/literal path profile deliberately differs from full mutable Python class/session/Requests codec/nonfinite-JSON domain; origin/path safety limits and owned evidence are Go policies.
- Independent Source AST extraction executed10 assertions with stubs for Resource.find/_get_one_match/_get_next_link, not full Proxy/decoder runtime or live Neutron.

### `python:network/v2/get_network_ip_availability`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.GetWithFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.Get`

- GetWithFields sends GET network-ip-availabilities/{network_id} with each Python NetworkIPAvailability Body attribute name mapped to its wire name as repeated fields query keys, keeping caller order, repeats, empty and unknown names, and decodes the selected attributes into the typed model.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersSendAliasedRepeatedFields`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- GetWithFields with nil or empty fields sends the plain member GET without a query, which is Python's fields=None request, and a sparse response decodes with zero values for omitted attributes.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersWithoutFieldsSendPlainGet`.
- GetWithFields accepts HTTP 200 through 399 like the Python read, and returns 404 and other failures as the native gophercloud status error wrapped in resource.OperationError with the operation name.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersAcceptPythonStatusesAndWrapFailures`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.
- A selected response without total_ips or used_ips decodes with those counters left empty instead of failing the native big-integer decoder, while explicit null and invalid counter types still fail.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonNetworkIPAvailabilityFieldsKeepOmittedCountersEmpty`.
- Get sends a bare GET network-ip-availabilities/{network_id} and decodes the availability, matching the call without fields.
  Tests: `network/v2/extensions/networkipavailabilities/python_parity_core_test.go:TestPythonParityCoreIPAvailabilityListAndGet`.

Python/Go differences:

- Go GetWithFields returns the Gophercloud v2.15.0 typed model, so attributes Neutron returns outside that model are dropped and attributes the server omits are zero values instead of None; Python returns a sparse Resource.
- The SDK field getter accepts HTTP 200..399 and keeps unknown field names in the query like Python; plain API.Get keeps the native status policy.
- Go reports total_ips and used_ips as decimal strings and leaves them empty when a fields selection omits them; Python leaves the omitted attributes None.

### `python:network/v2/find_network_ip_availability`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordFindOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordFindHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/networkipavailabilities.WithRecordFindListBasePath`

- GET-first clean400/403/404 fallback compares inherited actual id/name, not network_id/network_name, and adds no unsupported server name hint.
  Tests: `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityFindActualInheritedIdentityAndNoNameHint`.
- Sparse direct response overlays owned constructor seed and preserves actual Wire without seed fields.
  Tests: `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityFindDirectSparseSeedAndNoFallback`.
- Default missing returns nil, strictmissing ErrNotFound, later exact id/name duplicate ErrAmbiguous and late page503 invalidate earlier candidates.
  Tests: `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityFindMissingAmbiguityAndLateFailure`.
- Fallback route override, raw-rowcap, owned options and member-only microversion preserve selected client; physical faults never become cleanfallback and constructor/source/cancel errors stay preflight.
  Tests: `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityRecordsCapsVersionsAndOwnedOptions`, `network/v2/extensions/networkipavailabilities/python_record_contracts_test.go:TestIPAvailabilityRecordsPhysicalFailuresAndPreflight`.

Python/Go differences:

- Existing native List/Get/GetFields retain their typed onepage/string-counter behavior; new SDK-owned records supply allpage/localfilter/descriptor evidence without caller builders.
- Pinned Resource._get_one_match compares inherited id/name; unused name_attribute network_name and ordinary network_id are not synthesized into lookup identity. This corrects the older assessment that said fallback matches network_id/network_name.
- Record JSON descriptor/literal path profile deliberately differs from full mutable Python class/session/Requests codec/nonfinite-JSON domain; origin/path safety limits and owned evidence are Go policies.
- Independent Source AST extraction executed10 assertions with stubs for Resource.find/_get_one_match/_get_next_link, not full Proxy/decoder runtime or live Neutron.

### `python:network/v2/ports`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.ListRecordValues`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordListJMESPath`

- List sends GET ports with network_id, admin_state_up, repeated security_groups and fixed_ips and binding:host_id, and follows ports_links with the next href.
  Tests: `network/v2/ports/python_parity_core_test.go:TestPythonParityCorePortListQueryAndLinks`.
- Default bodyless /ports GET and Accept header, Source binding/DNS/security/QoS/unknown fields, missing defaults and actual page evidence.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsDefaultRequestAndExtensionFields`.
- Canonical and wire server aliases, repeated Requests values, unknown kwargs discarded before JSON encoding, nested local declared Body filters.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsQueryAliasesAndLocalBodyFilters`.
- Plural/HTTP link continuation, short-page limit/marker fallback, paginated false and raw row cap.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsPagingLinksAndLimitMarker`.
- Accepted representations and rejected statuses retain ports operation context; nil option performs no HTTP.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsStatusesAndOperationContext`.
- Cloud search uses dict insertion-order pushdown, exact/glob ID-name matching, nested filters and arbitrary full JMESPath; unknown filter fails after real inventory.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsSelectionAndPushdown`.
- Explicit version and headers apply to every actual list page.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsHeadersAndMicroversionOnEveryPage`.
- Partial field queries do not seed responses; Source constructor alias order/null/boolean/int/list/dict projection and independent actual Wire/Envelope/Header ownership.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsPartialFieldsCoercionAndEvidenceIsolation`.
- Owned runtime controls and malformed UTF-8/JSON fail before HTTP; unknown kwargs discarded without running a JSON Marshaler.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsControlsDiscardAndValidationBeforeEncoding`.
- Option factories/callback maps own input snapshots and nil Headers are initialized for direct custom assignments.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsOptionsOwnMapsAndNilHeaders`.
- Option failure retains simultaneous selected-client mutation or cancellation cause, and no HTTP escapes the boundary.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListAndSearchRecordOptionsGuardCancellationAndSource`.
- Raw cap counts filtered rows, lazy consumer break and single-page stop avoid next page requests.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsCapCountsFilteredRowsAndLazyConsumer`.
- CPython float underflow next is falsey; malformed envelope/row/constructor controls and invalid next retain actual accepted page receipt.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsContinuationTruthinessAndTerminalEvidence`.
- 18 accepted/rejected status plus Read/Close/mixed faults close once, invoke no retry/reauth hook, preserve actual receipt and native401 pointer ErrOriginal/ErrReauth fields.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsPhysicalFailuresPreserveNative401AndStopCallbacks`.
- Clean401 token renewal and clean503 retry retain selected headers/version/bodyless route/query and leave original client configuration intact.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsCleanReauthenticationAndRetryKeepPolicy`.
- Eager explicit expression supports scalar/array/object/null values and preserves all original Inventory on expression failure.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordValuesArbitraryJSONAndInventory`.
- Cloud falsey filters including underflow act as absent; constructor headers snapshot input and unknown dict keys retain actual inventory on local failure.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsFalseyFiltersHeaderSnapshotAndUnknownKeys`.
- Original generator projections/index/type return null, literals/identity equality return values, typed functions/serialization/syntax fail with zero HTTP.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortProxyListRecordsSourceGeneratorExpressionsDoNoHTTP`.
- Source missing/falsey/nonstring controls retain opaque generator; @/not_null/to_array preserve same lazy generator, HTTP starts only on iteration.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortProxyListRecordsPreserveReturnedAndNestedGenerators`.
- Source formal header/version/base_path normalization and ordered single-pass conflicting attrs restoration; restored formal duplicates fail before HTTP and restored JMES cannot replace the original expression.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsNormalizeSourceFormalsAndConflictingAttrs`.
- Source unknown non-Body control names and Header kwargs are discarded before JSON marshaling without introducing new query or transport behavior.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsSourceUnknownControlNamesDiscardBeforeMarshaler`.
- Typed/raw/snapshot base_path selects service-relative collection path for every page; unsafe foreign host/query/fragment/traversal fails before HTTP.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsBasePathOwnsRouteAndLocksContinuation`.
- Source-profile local filters preserve CPython float rounding, arbitrary integer precision, nested boolean/number equality, underflow and overflow comparison.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsLocalEqualityUsesSourceLoadedFloats`.
- Cloud dict filters preserve list_ports polymorphic Proxy result: null/type errors perform no HTTP, @ retains actual inventory and empty literal containers return no fabricated receipt.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsHonorsProxyJMESBeforeCloudLocalSelection`.
- Raw Source formals and dedicated concrete factories use last option consistently for selected path, headers and microversion.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortListRecordsFormalFactoriesAndTypedOptionsUseLastValue`.
- Actual Source canonical cloud dict lookup succeeds, wire Body keys fail after real inventory, raw Connection location is snapshotted and applied before selection without modifying Wire.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsSourceWireFiltersAndConnectionLocationBeforeSelection`.
- Cloud second-phase SelectLoaded preserves Source numeric matches from first-phase list filtering rather than reversing rounded float/integer or nested underflow comparisons.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsSecondFilterKeepsSourceLoadedNumericMatch`.
- Configured CloudLocation is cloned and computed per actual resource before filtering; foreign project clears scope names, keeps original project/zone, and never fabricates Wire location.
  Tests: `network/v2/ports/python_list_records_test.go:TestPythonPortSearchRecordsCloudLocationOwnsForeignProjectAndZone`.

Python/Go differences:

- Python transposes is_admin_state_up, is_port_security_enabled, security_group_ids and tenant_id; Go callers use the wire names through ListOpts or WithListQuery.
- Go ListOpts sends tags filters as one comma-separated value, while Python repeats list values.
- Python filters keyword arguments that are Body attributes but not QueryParameters locally after each page; Go sends WithListQuery keys to the server and the caller filters the returned rows.
- The native ports.Port model has no binding:*, dns_*, extra_dhcp_opts, port_security_enabled, qos_*, resource_request, trunk_details or several other Python Body fields, so those values are not returned and cannot be filtered by the caller.
- Python merges the next link's query into its own query parameters; Go requests the next href exactly as Neutron returned it.
- Concrete Go records expose owned declared Resource view and independent actual Wire/Envelope/Header/StatusCode. Full mutable Python Resource/cache/dirty-state/runtime subclass handling remains tracked shared SDK-R1/C1/S1, not counted as generated transport parity.
- ProxyListRecords reproduces Source generator-based JMES zero-HTTP values/null/type errors/opaque identity and nested generators. ListRecordValues is explicitly eager Go expression functionality, not claimed identical to Source generator input.
- ListRecords keeps a resource-only iterator and directs polymorphic truthy expressions to ProxyListRecords; falsey/nonstring Proxy controls preserve lazy records. Standard Go iter.Seq2 iteration semantics remain the owned SDK abstraction.
- List routes remain within captured service endpoint; continuation preserves selected origin/collection/filter and rejects unsafe route cycles. Go dictionary/RFC link compatibility and strict validation are explicit Source differences.
- Both list local filters and cloud dictionary selection opt into CPython JSON-loaded equality. Full JMESPath retains documented exact-decimal/lexical object traversal/strict grammar boundaries, and JSON-map object functions can differ from Python Port exact-typename function checks.

### `python:network/v2/get_subnet_ports`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.GetSubnetPorts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.SubnetPortID`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.SubnetPortValue`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.SubnetPortRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.SubnetPortNative`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.SubnetPortFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSubnetPortOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSubnetPortHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSubnetPortMicroversion`

- All sends one unfiltered GET ports and the caller keeps the ports whose FixedIPs contain the subnet ID, matching Python's local selection.
  Tests: `network/v2/ports/python_parity_core_test.go:TestPythonParityCorePortSubnetPortsSelection`.
- Source Subnet._get_id accepts literal or owned/native Resource ID; SDK snapshots selection and scans ports without subnet lookup or server filter.
  Tests: `network/v2/ports/python_subnet_ports_test.go:TestPythonSubnetPortsLiteralAndOwnedTargetsScanWithoutSubnetLookup`.
- Unfiltered shared pagination, repeated fixed-IP matches append the identical record pointer without deduplication, and every response is actual remote evidence.
  Tests: `network/v2/ports/python_subnet_ports_test.go:TestPythonSubnetPortsDuplicateMatchesKeepIdenticalPointerAndPagination`.
- CPython loaded JSON bool/int/float and nested equality apply to arbitrary explicit JSON identifiers including null.
  Tests: `network/v2/ports/python_subnet_ports_test.go:TestPythonSubnetPortsRawIdentifierUsesSourceLoadedEquality`.
- Concrete target/option factories own maps/raw ID/header/version values; nil/invalid targets and nil options fail before HTTP.
  Tests: `network/v2/ports/python_subnet_ports_test.go:TestPythonSubnetPortsTargetsAndOptionsAreOwnedAndGuarded`.
- Malformed null fixed_ips, missing subnet_id and wrong fixed-IP shapes preserve actual page receipt plus earlier selected/observed records.
  Tests: `network/v2/ports/python_subnet_ports_test.go:TestPythonSubnetPortsMalformedFixedIPsRetainActualPageAndObservedRows`.
- Physical 401 mixed read/close preserves actual native pointer and causes, one close, no reauthentication/retry selection callbacks.
  Tests: `network/v2/ports/python_subnet_ports_test.go:TestPythonSubnetPortsPhysical401NeverRetriesSelection`.
- Simultaneous callback failure and cancellation or captured source mutation are joined before any HTTP.
  Tests: `network/v2/ports/python_subnet_ports_test.go:TestPythonSubnetPortsCallbackCancellationAndSourceMutationPreserveCauses`.

Python/Go differences:

- Go has no single helper; the caller applies the fixed_ips subnet_id test to the rows from All.
- Python raises KeyError for a fixed IP without subnet_id; Go treats it as an empty ID.
- The native ports.Port model lacks extension fields that Python Port Resources expose.
- Concrete JSON/context/captured-client API; arbitrary Python Resource/session/class/callable substitutions are not accepted. Shared cloud configuration disabled-service semantics and mutable Resource R1 contracts remain separate tracked scope.
- Public Gophercloud testhelper/nativefixture reuse and actual response errors preserve physical read/close causes; opaque or literal local JSON never receives invented resource/HTTP evidence.
- Captured route/header/version protection, Go error returns and retained observed inventory differ intentionally from dynamic Python exception-only return conventions.
- Go SubnetPortTarget is a concrete selector; SubnetPortValue covers non-Resource JSON input and SubnetPortFields deliberately reads only an owned input ID without inventing a response.
- Go returns partial observed Inventory/Ports with an error; Source throws without returning its partial result. No deduplication or subnet-name inference is introduced.
- Source renamed subnet_id keyword compatibility is represented by the explicit SubnetPortID constructor rather than accepting dynamic keyword aliases.

### `python:network/v2/create_ports`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.CreatePorts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.BulkPort`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.BulkPortJSON`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.BulkPortFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithBulkPortAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithBulkPortOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithBulkPortHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithBulkPortMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.BulkPortResult.All`

- Exactly one eager flat POST {ports:[attrs1,attrs2]} owns canonical/wire alias order, raw request descriptor values, unknown Body names and consumes request microversion; collection iterator never resends.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsSingleEagerFlatPostAndRawAliases`.
- All row headers validate, only final row header applies, Source If-Match scalar/list rendering and selected request version win without mutating native client.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsLastRowHeaderAndSelectedVersion`.
- Every malformed/empty input, request constructor collision and earlier invalid row header fails before POST; [empty object] is accepted.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsPreflightRejectsEveryInputBeforePost`.
- Owned input/options/header snapshots and callback maps isolate mutations; public result receipt mutation does not alter retained lazy projection proof.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsInputsAndOptionCallbacksAreOwned`.
- Returned response rows are remote-only with Source defaults/coercions/unknown fields and raw Wire; row count need not match input and HTTP If-Match does not seed logical record.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsRemoteViewsHaveDefaultsAndNoHttpHeaderSeed`.
- 204/non-JSON/missing/null/noniterable ports fail eagerly with actual completed POST receipt; Source empty arrays/objects/strings remain successful empty iterables.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsEagerEnvelopeAndIterableFailuresKeepActualPost`.
- Malformed response rows and cls/self/connection/_synchronized/microversion constructor collisions fail lazily after earlier valid rows, retaining actual POST receipt; nonempty object/string ports also fail lazily.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsResponseRowsFailLazilyAfterEarlierActualRows`.
- Before-POST callback errors join cancellation/source mutation; after-POST captured source mutation rejects lazy consumption with actual completed POST proof.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsCancellationAndSourceMutationRetainCauses`.
- Actual success/rejection physical read/close/mixed failures retain body/headers/status/native401 pointer and causes, one close and zero willing retry/reauth callbacks.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsPhysicalFailuresPreserveNativeEvidenceAndStopCallbacks`.
- Clean 401 renewal and clean 503 retry preserve identical bulk body, route, selected version and conditional header; callbacks cannot expand accepted statuses; no per-row POST.
  Tests: `network/v2/ports/python_bulk_contracts_test.go:TestPythonBulkPortsCleanReauthenticationAndRetryKeepOneBulkBodyPolicy`.

Python/Go differences:

- Concrete JSON/context/captured-client API; arbitrary Python Resource/session/class/callable substitutions are not accepted. Shared cloud configuration disabled-service semantics and mutable Resource R1 contracts remain separate tracked scope.
- Public Gophercloud testhelper/nativefixture reuse and actual response errors preserve physical read/close causes; opaque or literal local JSON never receives invented resource/HTTP evidence.
- Captured route/header/version protection, Go error returns and retained observed inventory differ intentionally from dynamic Python exception-only return conventions.
- Factory/raw JSON keep explicit Source attribute order; public Go Attributes maps use lexical distinct-key order. Alias collision callers can use ordered factories or raw object.
- Go iterator can repeatedly project retained completed response without sending another request; Source response generator is single-use. All discards partial slice on projection error; manual iteration can retain earlier rows.
- Bulk explicitly opts into SDK-owned SourceStatusPolicy accepting valid HTTP100..399, matching Source raise_from_response below400; reviewed older list/find profiles retain200..399. Physical >=400 never becomes logical success; callback OkCodes expansion cannot alter the selected policy.
- Bulk atomicy and per-row/attribute user/admin policy remain server/plugin responsibilities; official Neutron27 Pecan validation/policy and ML2 native bulk transaction source were reviewed, no live deployment/rollback was tested.

### `python:network/v2/add_ip_to_port`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.Update`

- Update with PortID sends PUT floatingips/{id} with {"floatingip": {"port_id": id}} and decodes the associated floating IP.
  Tests: `network/v2/extensions/layer3/floatingips/python_parity_core_test.go:TestPythonParityCoreFloatingIPPortAssociation`.

Python/Go differences:

- Python takes Port and FloatingIP Resources and also commits any other dirty FloatingIP attributes; Go takes the floating IP ID and sends only the given fields.
- Python sends no request when port_id already equals the port ID; Go always sends PUT.

### `python:network/v2/remove_ip_from_port`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.Update`

- Update with an empty PortID sends PUT floatingips/{id} with {"floatingip": {"port_id": null}}.
  Tests: `network/v2/extensions/layer3/floatingips/python_parity_core_test.go:TestPythonParityCoreFloatingIPPortAssociation`.

Python/Go differences:

- Python takes a FloatingIP Resource and also commits its other dirty attributes; Go takes the floating IP ID.
- Python sends no request when port_id is already None; Go always sends PUT.

### `python:network/v2/find_available_ip`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.FindAvailableIPRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithAvailableIPOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithAvailableIPHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithAvailableIPMicroversion`

- List sends GET floatingips without a query, follows floatingips_links lazily and stops fetching when the caller breaks at the first row with an empty PortID; no row yields nil.
  Tests: `network/v2/extensions/layer3/floatingips/python_parity_core_test.go:TestPythonParityCoreFloatingIPFindAvailableStopsAtFirstFreeRow`.
- FindAvailableIPRecord owns unfiltered Source-profile floating IP list iteration, returns the first falsey untyped port_id including missing/null/empty string/false/zero/underflow/empty collections, preserves raw extension/resource views, stops before later malformed row construction and continuation parsing, and returns nil on exhausted inventory.
  Tests: `network/v2/extensions/layer3/floatingips/python_available_test.go:TestPythonAvailableIPFalseyPortIDAndRawExtensionProjection`, `network/v2/extensions/layer3/floatingips/python_available_test.go:TestPythonAvailableIPEarlyStopPagingAndExhaustion`.
- Owned transport-only options do not add server filters, snapshot caller headers, and guard cancellation/selected source mutation before HTTP. Accepted status203 retains actual page evidence; malformed consumed pages/rows and rejected500 yield no fabricated free IP.
  Tests: `network/v2/extensions/layer3/floatingips/python_available_test.go:TestPythonAvailableIPAcceptedStatusAndFailures`, `network/v2/extensions/layer3/floatingips/python_available_test.go:TestPythonAvailableIPOwnedTransportOptionsAndCancellation`.

Python/Go differences:

- FindAvailableIPRecord is the SDK-owned no-query first-falsey-port helper; native typed List remains available as a separate ABI. Returned immutable RawResource views and page receipt do not return a Python mutable Resource object.
- network.FloatingIPs.Available remains a different workflow that filters by network/project and may allocate; this helper only observes existing inventory.
- Explicit Go profile boundaries include transport-only headers/microversion extras absent from Source formals, guarded continuation/route policy, full Requests charset/nonfinite JSON/adapter-session-cache and arbitrary Python object behavior. These are not a remaining absence of the named helper.

### `python:network/v2/add_gateway_network_to_speaker`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.AddGatewayNetwork`

- AddGatewayNetwork sends PUT bgp-speakers/{id}/add_gateway_network with {"network_id": id} and decodes the returned network_id.
  Tests: `network/v2/extensions/bgp/speakers/python_parity_core_test.go:TestPythonParityCoreSpeakerGatewayNetworksAndRoutes`.

Python/Go differences:

- Python returns the response dictionary; Go returns AddGatewayNetworkOpts decoded from it.
- Python's Resource action uses Proxy.request without raise_from_response, so an HTTP error returns the error body; Go returns the HTTP error.
- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model.

### `python:network/v2/remove_gateway_network_from_speaker`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.RemoveGatewayNetwork`

- RemoveGatewayNetwork sends PUT bgp-speakers/{id}/remove_gateway_network with {"network_id": id}.
  Tests: `network/v2/extensions/bgp/speakers/python_parity_core_test.go:TestPythonParityCoreSpeakerGatewayNetworksAndRoutes`.

Python/Go differences:

- Python's Resource action uses Proxy.request without raise_from_response, so an HTTP error returns the error body; Go returns the HTTP error.
- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model.

### `python:network/v2/get_advertised_routes_of_speaker`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.GetAdvertisedRoutes`

- GetAdvertisedRoutes sends GET bgp-speakers/{id}/get_advertised_routes and yields destination and next_hop rows.
  Tests: `network/v2/extensions/bgp/speakers/python_parity_core_test.go:TestPythonParityCoreSpeakerGatewayNetworksAndRoutes`.

Python/Go differences:

- Python returns the whole response dictionary; Go yields the advertised_routes rows as typed values.

### `python:network/v2/get_trunk_subports`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.GetSubports`

- GetSubports sends GET trunks/{id}/get_subports and decodes port_id, segmentation_type and segmentation_id of each sub port.
  Tests: `network/v2/extensions/trunks/python_parity_core_test.go:TestPythonParityCoreTrunkGetSubports`.

Python/Go differences:

- Python returns the response dictionary with sub_ports; Go returns a typed Subport slice.
- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes an ID string and returns the native Gophercloud model.

### `python:network/v2/wait_for_status`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.WaitFor`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.Wait`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFailureStates`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithUnlimitedWait`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithTimeout`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPollInterval`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithProgressCallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithStatusAttribute`

- Ports WaitFor polls GET ports/{id} until a case-insensitive status match, reports progress 0 after nonterminal observations, returns resource.ErrFailedState on ERROR with WithFailureStates and a context deadline error after WithTimeout.
  Tests: `network/v2/ports/python_parity_core_test.go:TestPythonParityCorePortWaiters`.

Python/Go differences:

- Python waits forever with failures=['ERROR'] by default; Go defaults to five minutes and a prefix/suffix failure predicate, which WithUnlimitedWait and WithFailureStates("ERROR") replace.
- Python returns immediately without HTTP when the given Resource already has the target status; Go always sends the first GET.

Unresolved boundaries:

- Python can wait on any Neutron Resource; resources without a Go collection, such as local IPs, segment ranges, port bindings and auto-allocated topologies, have no Go waiter.

### `python:network/v2/wait_for_delete`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.WaitForDeletion`
- `github.com/JSYoo5B/go-openstacksdk/resource.Collection.WaitDeleted`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithTimeout`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithPollInterval`

- Ports WaitForDeletion with WithTimeout(120s) polls GET ports/{id} until 404 and then succeeds.
  Tests: `network/v2/ports/python_parity_core_test.go:TestPythonParityCorePortWaiters`.

Python/Go differences:

- The Neutron Proxy default is wait=120; Go defaults to five minutes and WithTimeout(120*time.Second) matches.
- Python returns the original Resource; Go returns only an error.

Unresolved boundaries:

- Python can wait for any fetchable Neutron Resource to disappear; resources without a Go collection have no deletion waiter.

### `python:network/v2/availability_zones`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/availabilityzones.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/availabilityzones.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/availabilityzones.WithProxyQuery`

- Proxy accepts but drops every query kwarg without marshaling, URL filters, Body predicates or JMESPath evaluation.
  Tests: `network/v2/availabilityzones/records_test.go:TestAvailabilityZonesProxyDiscardsAllQueryKwargsWithoutMarshalOrFiltering`.
- Source nullable declarations, unknown/raw extensions and independent actual page metadata are preserved.
  Tests: `network/v2/availabilityzones/records_test.go:TestAvailabilityZonesDeclaredDefaultsUnknownFieldsAndActualReceiptOwnBytes`.
- Lazy iterator stops before next request; eager AllRecords discards partial pages on error.
  Tests: `network/v2/availabilityzones/records_test.go:TestAvailabilityZonesPagingAndAllRecordsDiscardPartialErrors`.
- Selected source/context callbacks are guarded and runtime shapes fail atomically with actual response proof.
  Tests: `network/v2/availabilityzones/records_test.go:TestAvailabilityZonesCallbacksCannotHideSourceChangeOrCancellation`.
- Clean401 reauthentication retains project-independent zone route and latest shared token.
  Tests: `network/v2/availabilityzones/records_test.go:TestAvailabilityZonesNative401KeepsActualOriginalErrorAndCleanReauth`.

Python/Go differences:

- Go returns detached Source Resource/Wire with actual response metadata and uses context plus owned callback guards; optional Python Adapter cache/session and object/dirty lifecycle remain tracked whole-project SDK-R1/C1/S1.
- Common Neutron/REST engine owns strict UTF8, continuation origin and request/header/version policies; no live-cloud support is claimed.

### `python:network/v2/service_providers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/serviceproviders.WithRecordListConflictingAttrs`

- Library-owned service provider list uses the common neutronfind engine for GET /service-providers and full Source defaults, independent actual Resource/Wire/whole-page/Header/status, raw unknown extensions and sparse rows without inventing provider IDs. Source Resource has no Neutron revision/tag/Header mixin. is_default aliases default, bool coercion follows CPython and omission/null remains null.
  Tests: `network/v2/serviceproviders/python_record_contracts_test.go:TestServiceProviderListOwnedDefaultsAliasesAndNoNetworkMixins`.
- All six canonical query attributes including inherited limit/marker, literal fields, service_type/name and canonical is_default/default presence are server queries; id is a local Body condition and unknown caller kwargs are discarded before marshaling. Canonical null/false precedence and server-only name/default behavior remain explicit.
  Tests: `network/v2/serviceproviders/python_record_contracts_test.go:TestServiceProviderListFullQueriesDefaultPresenceAndLocalID`.
- Actual Proxy JMES evaluation receives an opaque unconsumed generator. Projection null/literal/identity results and runtime failures occur with zero HTTP, nested generator identity remains Value with no fake JSON/evidence, and real source HTTP starts only when the generator is consumed.
  Tests: `network/v2/serviceproviders/python_record_contracts_test.go:TestServiceProviderProxyJMESPreservesOpaqueGeneratorBeforeHTTP`.
- Shared owned Source kwargs consume headers/version/base_path, one-pass conflicting attrs and selected collection continuation, preserving original client facts. Accepted malformed envelope/null-row/constructor failures retain actual status/body/Header. Endpoint route is service-relative and request source/token remain owned.
  Tests: `network/v2/serviceproviders/python_record_contracts_test.go:TestServiceProviderListSourceControlsPagingAndActualErrors`.
- Raw cap counts excluded rows and stops before out-of-range constructor or invalid continuation; option callback maps remain owned through nil reset and retained callback mutation, without requiring user builders or transport classes.
  Tests: `network/v2/serviceproviders/python_record_contracts_test.go:TestServiceProviderListCapsAndOwnedCallbackMaps`.
- Real Gophercloud testhelper HTTP transport response read/Close faults never call RetryFunc or ReauthFunc and preserve one HTTP/one Close plus all physical causes. Accepted faults retain actual ResponseError, rejected native401 pointer keeps ErrOriginal receipt and ErrReauth failures; native503 actual status/body/Header is preserved.
  Tests: `network/v2/serviceproviders/python_record_contracts_test.go:TestServiceProviderListPhysicalProofNeverReauthsOrRetries`.

Python/Go differences:

- Gophercloud2.15 has no native binding for this resource; concrete SDK-owned API/profile/options reuse common reader, projection, paging and JMES engines. Source readonly capabilities are preserved and provider names do not become IDs.
- Immutable eager JSON view differs from mutable Python Resource/lazy descriptors/cache/current_location; location is an owned null view and Wire remains original. None-valued headers, arbitrary adapters/subclasses/runtime non-JSON objects and URI percent-format are outside the concrete JSON option boundary, separately tracked through shared SDK foundations.
- Same-origin selected collection/filter/cursor guards bound broad Source routing. Empty base_path falls back to the default in Go whereas Source empty URI returns without HTTP. Missing stock provider IDs do not become invented marker IDs; unsupported limit/marker continuation fails with actual page evidence. Explicit single-page is available.
- Completed200..399 are supported, informational100..199/replacement UTF8 are explicit differences. Nativephysical/source/cancel causes and clean retry/token renewal retain actual metadata and method ownership rather than fabricating successful Resource data.
- Actual Source Proxy generator JMES results are distinct from cloud filter arrays. Arbitrary JSON or nested opaque generators are returned by ProxyListRecords.Value/JSON; typed ListRecords/AllRecords remain resource iterators and redirect truthy expressions to that API.
- Neutron27 stock get_service_provider is project role:reader without owner project_id because providers are not project resources. SDK does not create drivers, elevate privileges or prove actual deployment backend support; fixture/race/source/main evidence has that explicit scope.

### `python:network/v2/get_auto_allocated_topology`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.API.GetRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.WithProjectID`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.WithOptions`

- Literal or recorded auth project is bound once without name lookup; actual response id may differ from request project.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyGetBindsProjectSeedAndActualNetworkIdentity`.
- Default project requires recorded scoped auth and never guesses from token/endpoint.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyRecordedCurrentProjectAndMissingAuthNeverGuessRoute`.
- Source alias fallback respects explicit null, unknown fields and uncached fresh requests.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyResponseAliasNullPrecedenceAndNoCache`.
- All three topology operations retain physical Read/Close causes and native401 ErrOriginal actual Method/URL/body/header with one HTTP/Close and zero faulty retry/reauth hooks.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyAllThreeOperationsFaultsPreserve401NativeFieldsAndZeroHooks`.
- Pre-canceled contexts retain caller cause before auth lookup or HTTP.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyCancellationRetainsCauseBeforeAnyAuthLookup`.

Python/Go differences:

- Go returns detached Source Resource/Wire with actual response metadata and uses context plus owned callback guards; optional Python Adapter cache/session and object/dirty lifecycle remain tracked whole-project SDK-R1/C1/S1.
- Common Neutron/REST engine owns strict UTF8, continuation origin and request/header/version policies; no live-cloud support is claimed.
- Nil project uses recorded Keystone auth only, without an extra auth refresh; manual/unscoped tokens require explicit literal project instead of inferring it.

### `python:network/v2/delete_auto_allocated_topology`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.API.DeleteRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.WithProjectID`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.WithOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.WithIgnoreMissing`

- Pinned ignore_missing default is false; only explicit true suppresses clean actual404 and retains literal receipt.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyDeleteDefaultStrict404AndExplicitIgnoreOwnActualProof`.
- All three topology operations retain physical Read/Close causes and native401 ErrOriginal actual Method/URL/body/header with one HTTP/Close and zero faulty retry/reauth hooks.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyAllThreeOperationsFaultsPreserve401NativeFieldsAndZeroHooks`.
- Pre-canceled contexts retain caller cause before auth lookup or HTTP.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyCancellationRetainsCauseBeforeAnyAuthLookup`.

Python/Go differences:

- Go returns detached Source Resource/Wire with actual response metadata and uses context plus owned callback guards; optional Python Adapter cache/session and object/dirty lifecycle remain tracked whole-project SDK-R1/C1/S1.
- Common Neutron/REST engine owns strict UTF8, continuation origin and request/header/version policies; no live-cloud support is claimed.
- Nil project uses recorded Keystone auth only, without an extra auth refresh; manual/unscoped tokens require explicit literal project instead of inferring it.
- Go owned DELETE retains the reviewed mutation success policy HTTP200..399; synthetic informational100..199 is an error. Source raise_from_response only rejects400 and above. The new SourceStatusPolicy opt-in governs record reads and bulk responses, not existing mutation acceptance.

### `python:network/v2/validate_auto_allocated_topology`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.API.ValidateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.WithProjectID`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/autoallocatedtopologies.WithOptions`

- Fixed fields=dry-run GET has URI project but no Resource.id seed, extra calls, client-side allocation or fabricated dry_run boolean.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyValidateFixedDryRunHasNoProjectIDSeedOrCreateCalls`.
- Headers/selected version are detached and fixed through Source callback boundaries without changing shared client.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyOptionsKeepHeadersVersionAndCapturedSource`.
- Malformed accepted response retains actual proof and passive receipt without Resource success.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyInvalidOptionsAndAcceptedDecodeFailuresAreAtomic`.
- Clean retry/renewal preserve dry-run query and captured route/version.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyCleanRetryAndReauthKeepDryRunAndDeletionMissingPolicy`.
- All three topology operations retain physical Read/Close causes and native401 ErrOriginal actual Method/URL/body/header with one HTTP/Close and zero faulty retry/reauth hooks.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyAllThreeOperationsFaultsPreserve401NativeFieldsAndZeroHooks`.
- Pre-canceled contexts retain caller cause before auth lookup or HTTP.
  Tests: `network/v2/autoallocatedtopologies/records_test.go:TestAutoTopologyCancellationRetainsCauseBeforeAnyAuthLookup`.

Python/Go differences:

- Go returns detached Source Resource/Wire with actual response metadata and uses context plus owned callback guards; optional Python Adapter cache/session and object/dirty lifecycle remain tracked whole-project SDK-R1/C1/S1.
- Common Neutron/REST engine owns strict UTF8, continuation origin and request/header/version policies; no live-cloud support is claimed.
- Nil project uses recorded Keystone auth only, without an extra auth refresh; manual/unscoped tokens require explicit literal project instead of inferring it.

### `python:network/v2/onboard_network_subnets`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.OnboardNetworkSubnetsRecord`

- SDK-owned one-PUT onboard route sends unwrapped network_id without preliminary GETs, accepts arbitrary JSON under Source status policy, snapshots options and guards physical errors/source identity.
  Tests: `network/v2/extensions/subnetpools/python_onboard_network_test.go:TestPythonOnboardNetworkSubnetsUnwrappedBodyAndArbitraryJSON`, `network/v2/extensions/subnetpools/python_onboard_network_test.go:TestPythonOnboardNetworkSubnetsHTTPAndDecodeFailures`, `network/v2/extensions/subnetpools/python_onboard_network_test.go:TestPythonOnboardNetworkSubnetsOptionsOwnershipAndGuard`, `network/v2/extensions/subnetpools/python_onboard_network_test.go:TestPythonOnboardNetworkSubnetsPhysicalFaultsStopCallbacks`, `network/v2/extensions/subnetpools/python_onboard_network_test.go:TestPythonOnboardNetworkSubnetsPreflight`.

Python/Go differences:

- Go accepts literal string pool/network IDs, escapes the pool route and keeps independent JSON/HTTP proof; arbitrary Python Resource inputs and mutable cache/runtime are not modeled.
- SDK-R1/C1/S1, real-cloud compatibility and release validation remain; local tests and source review do not establish full SDK parity.

## References

- [network v2 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [Agent](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/agent.py)
- [Segment](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/segment.py)
- [Extension](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/extension.py)
- [NetworkIPAvailability](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/network_ip_availability.py)
- [Port](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/port.py)
- [PortBinding](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/port_binding.py)
- [FloatingIP](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/floating_ip.py)
- [BgpSpeaker](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgp_speaker.py)
- [Trunk](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/trunk.py)
- [Python Port Bindings](extensions/portbindings/python-port-bindings.md)
- [Python List Records](extensions/agents/python-list-records.md)
- [Python Available](extensions/layer3/floatingips/python-available.md)
- [Python Records](extensions/networkipavailabilities/python-records.md)
- [Python Router Assignment Dhcp Hosts](extensions/agents/python-router-assignment-dhcp-hosts.md)
- [Python Onboard Network Subnets](extensions/subnetpools/python-onboard-network-subnets.md)
- [Python List Records](extensions/segments/python-list-records.md)
- [Python List Records](networks/python-list-records.md)
- [Python List Records](ports/python-list-records.md)
- [Serviceproviders](serviceproviders/README.md)
- [availability zone](availabilityzones/README.md)
- [auto-allocated topology](autoallocatedtopologies/README.md)
- [Python Records](extensions/python-records.md)
- [Python User Helpers](ports/python-user-helpers.md)
- [Autoallocatedtopologies](autoallocatedtopologies/README.md)
- [`FindAvailableIPRecord`](extensions/layer3/floatingips/python-available.md)
- [network segment range](extensions/networksegmentranges/README.md)
- [local IP](extensions/localips/README.md)
- [local IP association](extensions/localipassociations/README.md)
