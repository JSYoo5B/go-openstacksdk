# Compute

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/find_flavor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindFallback`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.FindFlavor`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.FindFlavor`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindLimit`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindMarker`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindPaginated`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindHeader`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindFilter`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorFindFilters`

- Pinned direct find_flavor accepts a string, ignore_missing=True, keyword get_extra_specs=False and query. Safe identities use native GET200 with the flavor envelope and native model/Swap decoding; compatible GET400/403/404 may fall back, with explicit strict, 404-only and GET-only Go policies.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsGETAndExtraSpecsPresence`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsMissingPoliciesAndUnsafeNames`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsPreflightAndNativeGETFailures`.
- Fallback uses native /flavors/detail and all FlavorPage continuations without an automatic name hint. SDK list-only is_public=None is added only for an absent caller key; direct GET remains the frozen caller query. Repeated and explicit nil/empty query keys retain precedence and raw wire extensions remain observable.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsListDefaultsAndCallerQuery`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsLookupClosesBeforeEnrichment`.
- Extra specs enrichment defaults to false. Explicit true enriches only a validated direct GET or a fully consumed unique ID-or-name match whose inline ExtraSpecs is nil/empty. It uses the returned canonical ID for native GET200 /os-extra_specs without caller query, clones the Flavor/model map and replaces fetched specs. Nonempty inline specs, missing lookup, ambiguity and late failures produce no enrichment request.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsGETAndExtraSpecsPresence`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsLookupClosesBeforeEnrichment`, `internal/nativefind/flavor_specs_test.go:TestNativeFlavorExtraSpecsUsesResolvedIDAndOwnsReturnedModel`, `internal/nativefind/flavor_specs_test.go:TestNativeFlavorExtraSpecsInlineAndNativeNilEmptyDefaults`, `connectiontest/connection_find_identity_flavor_test.go:TestConnectionFlavorFindIdentityExtraSpecs`.
- Extra-specs GET403/404, undeclared status, typed decode, read/transport errors and cancellation remain terminal without another lookup or IgnoreMissing suppression. Native missing/null extra_specs wrappers remain nil maps; the separate GET is not incorrectly gated at inline-field introduction microversion2.61.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsEnrichmentFailuresAreTerminal`, `internal/nativefind/flavor_specs_test.go:TestNativeFlavorExtraSpecsNativeFailuresRemainTerminal`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsKeepsLiveProviderVersionAndContext`.
- Bulk/custom options, bool pointers and query maps are frozen across callback phases and concurrent reuse. Generated leaf and handwritten Flavor collections retain the original provider, live token/headers/microversion/HTTP middleware/context and reverse ResourceBase. Extra-spec maps are independent; invalid controls, unsupported capabilities and unsafe GET-only names fail before HTTP.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsSnapshotsConcurrentReuseAndLastWins`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsKeepsLiveProviderVersionAndContext`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsPreflightAndNativeGETFailures`, `internal/nativefind/flavor_specs_test.go:TestNativeFlavorExtraSpecsSharesLiveSourceAndFreezesInput`, `internal/nativefind/flavor_specs_test.go:TestNativeFlavorExtraSpecsRejectsInvalidInputAndCancellationBeforeHTTP`, `internal/nativefind/flavor_specs_test.go:TestNativeFlavorExtraSpecsConcurrentCallsReturnIndependentMaps`.
- SDK-owned list default maps, bool options and enrichment callback are frozen before user/native callbacks; caller nil/empty keys retain precedence. Extra-specs capability and reserved controls fail before HTTP, callback errors/results remain terminal, and ordinary lookup plus secondary-search defaults remain isolated.
  Tests: `resource/find_identity_extra_specs_test.go:TestCollectionFindIdentityExtraSpecsDefaultsOnlyAffectIdentityLists`, `resource/find_identity_extra_specs_test.go:TestCollectionFindIdentityExtraSpecsCapabilityAndQueryPreflight`, `resource/find_identity_extra_specs_test.go:TestCollectionFindIdentityExtraSpecsEnrichesValidatedUniqueSuccess`, `resource/find_identity_extra_specs_test.go:TestCollectionFindIdentityExtraSpecsListFailuresAndAbsenceNeverEnrich`, `resource/find_identity_extra_specs_test.go:TestCollectionFindIdentityExtraSpecsErrorsAndInvalidResultsAreTerminal`, `resource/find_identity_extra_specs_test.go:TestCollectionFindIdentityExtraSpecsSnapshotsMetadataOptionsAndCallback`, `resource/find_identity_extra_specs_test.go:TestCollectionFindIdentityExtraSpecsConcurrentOwnedOptionReuse`, `resource/find_identity_extra_specs_test.go:TestCollectionFindIdentityExtraSpecsDefaultsPreserveOrdinaryAndSecondarySearches`.
- The generated Flavor Resources bridge corrects ordinary raw status forwarding as well as identity fallback. Ordinary All and explicit Name Find consume their native pages without implicit is_public=None, identity enrichment or extra member GET.
  Tests: `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsOrdinaryCollectionsRetainQueryWithoutIdentityPolicies`.
- Explicit Go raw status is preserved as a frozen wire query in both GET and fallback LIST even for statusless native models; nil/empty/repeated values and caller name precedence are retained. Raw status alone does not activate local response filtering, and server query rejection remains a terminal LIST error rather than missing suppression.
  Tests: `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusSurvivesBothPhases`, `api/find_identity_wire_status_test.go:TestNativeFindIdentityWireStatusServerRejectionIsTerminal`.
- The primary owned FindFlavor always attempts one escaped literal member GET, including names with spaces or reserved URL characters, without a name hint or list-only is_public default. Semantic attributes retain their original member query spellings and vendor parameters while known attributes seed the Resource view; semantic id conflicts fail before discovery or member HTTP.
  Tests: `api/compute_flavor_find_test.go:TestComputeFindFlavorMemberSeedAliasesAndLiteralRoutes`, `resource/filters_test.go:TestSemanticFilterSelectionKeepsMemberSpellingSeedAndDeferredUnknownErrors`.
- Accepted member 200..399 retains its whole HTTP envelope, headers and status. A flavor envelope selects only the actual member object, otherwise the flat root is used; known response attributes overlay the input seed in parsed JSON alias order. Omission preserves seed, explicit null overrides it, and raw id/name/original_name truthiness plus response descriptors produce the owned view independently of Wire. Accepted empty, opaque or malformed JSON returns the seed with nil Wire; parsed null or array representations return a receipt error.
  Tests: `api/compute_flavor_find_test.go:TestComputeFindFlavorMemberSeedAliasesAndLiteralRoutes`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordDeclaredDefaultsDescriptorsAndRawIdentity`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordResponseAliasOrderAndIndependentReceipts`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordDescriptorErrorsAndPassiveStringIdentity`.
- Only clean member 400/403/404 uses the prepared detailed fallback with transposed server queries, list-only is_public=None and local Body filters. Passive JSON string ID or name is compared exactly; numeric values are not stringified. A unique match requires successful consumption of the selected list, a second matching row immediately returns Ambiguous, a late list failure remains terminal, and zero matches returns nil by default or NotFound with IgnoreMissing(false).
  Tests: `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsQueryProjectionAndOwnedViews`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsPagingRawCapsAndLazyOwnership`.
- GetExtraSpecs defaults false. Owned true conditionally enriches only the successfully resolved unique result whose projected extra_specs is falsey; inline nonempty dictionaries, absence, duplicates and late list failures do not fetch. The separate read uses the logical string identity and the same prepared version without caller query, exposes independent Enrichment receipts, and preserves the original row or member Wire.
  Tests: `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.
- The owned find prepares options, source and effective version once across member, fallback and enrichment. Nil version retains the selected client or discovers at most 2.61, explicit empty stays versionless, and the original client remains unchanged. Service.FindFlavor captures Connection location before options and adds it only to the view. Live tokens continue across reads; accepted read/Close or source errors preserve partial receipts and never fall back, become missing, or invoke an SDK response resend.
  Tests: `api/compute_flavor_find_test.go:TestComputeFindFlavorMemberSeedAliasesAndLiteralRoutes`, `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.

Python/Go differences:

- Retained native-shaped API.FindIdentity path: Go exposes SDK-owned typed GetExtraSpecs options and explicit raw wire query. Python list transposes min_ram/min_disk aliases and automatically classifies declared Body fields; Go uses minRam/minDisk and preserves explicit wire extensions instead. Raw get_extra_specs is an SDK control, not a service query.
- Retained native-shaped API.FindIdentity path: Go explicit nil/empty query slices retain key presence, suppress list defaults and omit the wire value; pinned Flavor.list instead converts missing or Python None is_public to the string None. Native missing/null fetched extra_specs remains a nil map, whereas Python fetch_extra_specs defaults a missing wrapper to an empty dict.
- Retained native-shaped API.FindIdentity path: The lookup keeps strict safe nonblank result-ID validation, native Flavor/Swap extraction and native full-page pagination. Details/AllProjects options are unsupported on Flavor; its actual list override has no details parameter, and its default base_path is /flavors/detail. Ordinary Get/List/explicit Ref lookup do not gain identity list defaults or enrichment. The generated Flavor Resources list bridge also fixes whole-map forwarding: ordinary List/All now preserve raw status instead of dropping it.
- The primary Python named-operation mapping is Service.FindFlavor or API.FindFlavor returning owned Resource/Wire/receipts; the retained native-shaped FindIdentity contracts above continue to document the independent typed ABI and its explicit fallback/enrichment policies. The owned leaf view has location=null; the Service wrapper adds one captured Connection CloudLocation to Resource and Enrichment.Resource without changing actual response evidence.
- Owned semantic filters transpose only on list fallback; original spellings and vendor attributes reach member GET. The common Go member query encoder accepts scalar, null and scalar-array values, so object-valued or nested-array semantic inputs fail before HTTP, including local Body predicates that Python may encode as arbitrary iterables. Source controls such as arbitrary session/path/resource_type, deprecated JMESPath and cache-bypass controls remain outside this fixed-route typed API and the broader SDK lifecycle goal.
- Go query booleans use lowercase true/false. The shared local matcher distinguishes JSON booleans from numbers and treats nested object-subset matching against a scalar as false; it does not reproduce Python empty-dict truthiness or nested scalar attribute errors. Source input dictionary order is unavailable in a Go map, so canonical seed spellings win over body wire aliases; actual selected response alias order is preserved.
- Executable input identities and markers must be nonblank UTF-8 literal strings without routing controls; passive numeric/container response identities are retained as JSON values and are never converted into string comparisons or specs request paths. The shared extra-specs target validator retains its stricter safe-ID policy.
- Owned reads accept 200..399 and validate nonnull UTF-8 object representations; member unavailable JSON including invalid UTF-8 is tolerated as a seed-only fetch, while physical read/Close and source/context failures remain terminal. Python charset replacement and mutable Resource/session/conditional-cache behavior are separate lifecycle scope. Raw and descriptor views own their bytes; nonfinite float/overflow cannot be represented in a finite JSON view, and the audited numeric conversions use explicit IEEE-754 and Unicode 16 rules.
- Dictionary next links are a deliberate Go compatibility extension. Initial positive limits enable logical last-raw-row marker fallback; the first advertised server limit may be accepted and fixed without enabling an absent initial fallback. Explicit row caps and first-page controls scope the uniqueness observation to the selected inventory.

### `python:cloud/_compute/ComputeCloudMixin/create_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Servers.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithNetworkInterfaces`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithNetworkMode`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithNetworks`
- `github.com/JSYoo5B/go-openstacksdk.WithDefaultNetwork`
- `github.com/JSYoo5B/go-openstacksdk.WithoutDefaultNetwork`
- `github.com/JSYoo5B/go-openstacksdk/compute.Servers.CreateWithFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithWorkflowTimeout`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithUnlimitedWorkflowTimeout`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.PrepareEnsureActive`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.CreateWithAutomaticFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk.Connection.CreateWithAutomaticFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPPool`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPAddresses`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPSource`
- `github.com/JSYoo5B/go-openstacksdk.WithServerAddressPolicy`
- `github.com/JSYoo5B/go-openstacksdk.WithMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithSchedulerHints`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroup`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerCount`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithDescription`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAdminPass`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithDiskConfig`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAccessIPv4`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAccessIPv6`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithTags`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithHostname`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithHypervisorHostname`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithPersonality`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithDataVolumes`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithUserData`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerWithoutAddressesPolicy`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithCreateRefresh`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithBlockDeviceMappings`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithLegacyBlockDeviceMappings`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithTrustedImageCertificates`
- `github.com/JSYoo5B/go-openstacksdk/compute.Servers.CreateReservation`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.CreateRefreshedServer`
- `github.com/JSYoo5B/go-openstacksdk.Connection.CreateRefreshedServer`

- Ordered NICs preserve network UUID and port combinations, literal fixed IP, fixed-IP-only/tag-only/empty objects, and omitted versus explicit empty or null tag values in the HTTP body. IDs bypass Neutron lookup and need no Network endpoint.
  Tests: `compute/server_networks_test.go:TestServerNICWirePreservesCombinationsOrderAndExplicitEmptyTag`, `connectiontest/connection_server_nics_test.go:TestConnectionServerNICIDsNeedNoNetworkEndpoint`.
- Connection resolves exact network/port names and preserves missing/ambiguous/403 causes without creating a server. Resolver failures, empty/unsafe returned IDs and cancellation stop before Nova.
  Tests: `connectiontest/connection_server_nics_test.go:TestConnectionServerNICsResolveExactNamesAndOwnInputs`, `connectiontest/connection_server_nics_test.go:TestConnectionServerNICNameFailuresPreventNovaCreation`, `compute/server_networks_test.go:TestServerNICResolverFailuresAndCancellationStopBeforeNova`.
- The last valid WithNetworks/NIC/mode selection replaces the previous selection, so discarded names or tag gates have no effect. Options own input slices and Optional tag values and can be reused concurrently.
  Tests: `compute/server_networks_test.go:TestServerNICSelectionUsesLastValidOption`, `compute/server_networks_test.go:TestServerNICOptionsOwnSlicesTagsAndConcurrentReuse`, `connectiontest/connection_server_nics_test.go:TestConnectionServerNICsResolveExactNamesAndOwnInputs`.
- Explicit mode auto/none requires selected Compute 2.37+ without discovery. Every present tag, including empty or null, follows require_microversion(compute, "2.42"): a selected version below 2.42 fails before dependency lookup, and otherwise the advertised range must contain 2.42 (an unselected client then raises only that POST to 2.42) or Create returns ErrUnsupported before the POST. A non-clean discovery failure is returned before the POST. Invalid options/Refs fail before dependency lookup; the shared client version is preserved.
  Tests: `compute/server_networks_test.go:TestServerNICInvalidOptionsFailBeforeDependencyLookup`, `compute/server_networks_test.go:TestServerNICWirePreservesCombinationsOrderAndExplicitEmptyTag`, `compute/server_networks_test.go:TestServerNICTagRequiresAdvertisedRange`.
- Without explicit NICs, mode or a Connection/YAML default, Create follows supports_microversion(compute, "2.37") from openstack/utils.py: the advertised Compute range must contain 2.37 and a selected client version must be at least 2.37. Then networks is auto; an unselected client raises only this POST to 2.37 and leaves the shared client unset. A selection below 2.37 omits networks without discovery; a range below 2.37, a clean 404/405 discovery or a selection beyond the advertised range omits the key. One discovery GET per create; Connection negotiation reuses its recorded advertised bounds without another discovery or Neutron lookup.
  Tests: `compute/server_networks_test.go:TestServerNICDefaultsModesAndVersionBoundaries`, `connectiontest/connection_server_nics_test.go:TestConnectionNegotiatedVersionControlsDefaultNetworkAuto`, `connectiontest/connection_network_role_consumers_test.go:TestConnectionDefaultNICEmptyRolesUseNovaNetworkDefault`.
- NIC choices preserve existing/new volume boot mapping and false delete-on-termination. An ACTIVE wait failure returns the created BUILD server and the error without server/port/volume cleanup.
  Tests: `compute/server_networks_test.go:TestServerNICsPreserveVolumeBootAndCreatedServerOnWaitFailure`.
- Connection owns a default-network Ref and explicit disable policy; the last option wins, invalid Refs remain errors, explicit NICs/modes bypass the default callback and Neutron endpoint lookup, and default IDs work for image/existing-volume/new-volume boots without altering their mappings.
  Tests: `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkIDsOptionsAndBootSources`, `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkDisableAndExplicitSelections`, `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkOptionValidationAndConcurrentReuse`, `compute/default_network_test.go:TestServerDefaultNetworkIsSkippedByExplicitNICsAndPreflight`.
- Typed default names resolve exactly; missing/ambiguous/403 failures, malformed callback Refs and cancellation stop before server creation. Explicit typed default-name resolution remains fresh; configured defaults share successful role discovery and failed discovery is not cached; callbacks and options are request-local, and wait failure preserves the created server without cleanup.
  Tests: `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkNameFailuresAndRetry`, `compute/default_network_test.go:TestServerDefaultNetworkDependencyFailureCancellationAndValidation`, `compute/default_network_test.go:TestServerDefaultNetworkDoesNotLeakBetweenRequestsAndPreservesWaitFailure`.
- Cloud/secure/public files are snapshotted once for native auth/endpoint/TLS and SDK metadata. Network lists replace at each precedence layer, [] clears inheritance, null/duplicate defaults/invalid flags are configuration errors, and empty/null metadata profile stops network inheritance while retaining native auth fallback.
  Tests: `connection_cloud_config_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connection_cloud_config_test.go:TestCloudNetworkConfigurationSearchAndReadFailures`, `connection_cloud_config_test.go:TestCloudNetworkConfigurationValidation`.
- YAML default name matches exact network name OR ID across unfiltered complete pagination. Cross-kind collisions and duplicate names are ambiguous; missing/unsafe IDs, a later-page 403 and cancellation preserve errors and prevent Nova POST. Connection override/disable and explicit Create selection bypass the configured lookup.
  Tests: `connectiontest/connection_cloud_network_test.go:TestConnectConfiguredDefaultNetworkMatchesNamesOrIDsAcrossAllPages`, `connectiontest/connection_cloud_network_test.go:TestConnectDefaultNetworkOverridesDisableAndExplicitCreateSelection`.
- The explicit compound workflow shares server input/body preparation with Create, validates server/IP options and external Ref before runtime lookup, and requires a concrete Network service and known/explicit reuse owner before Nova POST. Connection provides the lazy getter; invalid or missing services/scope return errors without server mutation.
  Tests: `compute/server_floating_ip_test.go:TestServerFloatingIPPreflightPrecedesCreationAndLookups`, `connectiontest/connection_server_floating_ip_test.go:TestConnectionServerFloatingIPMissingNetworkStopsBeforeNovaPOST`.
- After one Nova POST the original server ID is retained. Mandatory server waiting checks the actual ID and ACTIVE status even with a custom status attribute before destination port/available-IP requests or floating-IP mutation. Shared role discovery can already have run for the configured default NIC. It then reuses or allocates one IPv4 assignment and always verifies actual IP ACTIVE; prepared policy replacement cannot disable this mandatory wait.
  Tests: `connectiontest/connection_server_floating_ip_test.go:TestConnectionServerFloatingIPOrdersWaitsAndPreservesPartialResults`, `connectiontest/connection_server_floating_ip_test.go:TestConnectionServerFloatingIPPollsServerBeforeStartingIP`.
- One overall five-minute context limit includes service preparation, lookup/creation, both waits and IP work. A parent earlier deadline wins; individual unlimited wait cannot remove it. The overall unlimited option removes the SDK deadline when there is no parent deadline, and a later workflow timeout restores it without restarting a budget per stage. Deadline errors retain the known server and assignment.
  Tests: `connectiontest/connection_server_floating_ip_test.go:TestConnectionServerFloatingIPUsesOneDeadlineAcrossBothWaits`.
- Server wait failures retain the POST server; after actual ACTIVE verification IP selection failures retain the server and late guarded PUT/wait/malformed response failures retain server plus reused/allocated assignment. No fallback allocation, deletion or synthesized Nova addresses is performed.
  Tests: `connectiontest/connection_server_floating_ip_test.go:TestConnectionServerFloatingIPOrdersWaitsAndPreservesPartialResults`, `connectiontest/connection_server_floating_ip_test.go:TestConnectionServerFloatingIPUsesOneDeadlineAcrossBothWaits`.
- Image, existing-volume and new-image-volume boot preserve default private NIC and boot deletion policy independently of the external floating network. Wrapper option slices are snapshotted and concurrent reuse returns independent server/IP objects with the captured metadata/reuse configuration.
  Tests: `connectiontest/connection_server_floating_ip_test.go:TestConnectionServerFloatingIPKeepsBootAndDefaultNetworkPolicies`, `connectiontest/connection_server_floating_ip_test.go:TestConnectionServerFloatingIPOptionsSnapshotConcurrentReuse`.
- PrepareEnsureActive binds the recorded reuse owner to a copy and forces ACTIVE wait while preserving poll/timeout/progress configuration. Changing recorded authentication afterward does not change that bound owner; the original plain prepared policy still resolves the new owner per Ensure call and keeps its original no-wait behavior.
  Tests: `network/floating_ip_prepare_test.go:TestFloatingIPPrepareActiveBindsCopyAndPreservesWaitPolicy`.
- A configured default NIC consumes the same successful network/subnet role snapshot as Connection getters. Caller changes to a returned default cannot alter creation; consecutive creates reuse the successful discovery and Reset refreshes the chosen default ID for the next create.
  Tests: `connectiontest/connection_network_role_consumers_test.go:TestConnectionDefaultNICConsumesRoleCacheAndReset`, `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Configured default discovery validates the shared snapshot, so subnet403 or another configured role missing stops before Nova POST even when the default itself exists. Failed attempts are not cached; a subsequent success is reused. Later network-page errors, name-or-ID collisions, unsafe IDs and cancellation retain their established pre-create errors.
  Tests: `connectiontest/connection_network_role_consumers_test.go:TestConnectionDefaultNICRoleErrorsStopCreationAndRetry`, `connectiontest/connection_cloud_network_test.go:TestConnectConfiguredDefaultNetworkMatchesNamesOrIDsAcrossAllPages`.
- Both-disabled discovery or a clean pointer/value missing Network endpoint yields no configured default, so Nova auto follows the advertised 2.37 check (including an unselected client raised to 2.37 for that POST). Explicit default ID/disable and explicit NIC/mode bypass configured role discovery; explicit typed default names keep their exact-name resolution lane.
  Tests: `connectiontest/connection_network_role_consumers_test.go:TestConnectionDefaultNICEmptyRolesUseNovaNetworkDefault`, `connectiontest/connection_cloud_network_test.go:TestConnectDefaultNetworkOverridesDisableAndExplicitCreateSelection`, `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkNameFailuresAndRetry`.
- Connection CreateWithFloatingIP uses a single successful configured role inventory for its default NIC, floating source and NAT destination, including a configured source without router:external. Role lookup finishes before Nova POST; actual server ACTIVE precedes port and floating-IP requests, and actual floating-IP ACTIVE remains mandatory. A role403 returns no created resource; a later port403 preserves the known ACTIVE Server with nil assignment and the original HTTP cause, without allocation or cleanup.
  Tests: `connectiontest/connection_server_network_roles_test.go:TestConnectionCreateWithFloatingIPSharesDefaultSourceAndNATRoles`.
- Owned high Service CreateNetwork/UpdateNetwork/DeleteNetwork and the existing Networks.Delete adapter invalidate shared roles after accepted mutations. The actual matrix primes a configured default role, observes a new inventory after success or high-delete clean404 after a found lookup, and retains the original inventory after403/409/412, failed no-option lookup, or direct Collection404. A successful no-option Update performs its lookup and reset without PUT; initial missing ID/Name high Delete returns false without mutation or reset.
  Tests: `network/network_mutations_test.go:TestNetworkMutationSuccessFailureAndMissingCacheBoundaries`, `network/network_mutations_test.go:TestDeleteNetworkInitialMissingDoesNotMutateOrReset`.
- The Connection-cached high Network Service resets the same role cache after Create, Update and Delete. A later getter reads a new configured inventory, and Nova default NIC plus floating source/NAT allocation consume that same refreshed inventory without manual Reset. The fixture checks actual Nova and Neutron request fields and exactly two inventory reads.
  Tests: `connectiontest/connection_network_mutations_test.go:TestConnectionNetworkMutationRefreshesGetterDefaultNICAndFloatingRoles`.
- The owned compound creation POST accepts200/202 and rejects201. Accepted200/202 null/missing/malformed server envelopes and unsafe decoded IDs preserve actual response errors without invoking decode retry, raw wait, IP allocation or cleanup. Unexpected HTTP201 still reaches the configured native HTTP retry hook; it is not an accepted-create proof. The normal image path emits selected image/flavor/NIC fields, preserves original Creation.AdminPass separately, and completes actual Neutron readiness plus raw Nova floating-address observation when needed.
  Tests: `compute/server_automatic_create_test.go:TestAutomaticCreateAcceptedResponsePolicyAndDecodeDoNotRetry`, `compute/server_automatic_create_test.go:TestAutomaticCreateWaitsForRawMetadataAndObservesAssignment`.
- The new bounded creation lane waits on matching raw Nova200/203 responses for actual ACTIVE and at least one address row. BUILD and ACTIVE/null metadata remain progress states; explicit nonempty-map/empty-row inventory returns ServerAddressesUnavailableError rather than silently proceeding. Raw ERROR remains failed even when user failure states are disabled, and wrong identity/HTTP/progress-source/cancellation failures retain the original Creation and last matching Server without automatic deletion or IP mutation.
  Tests: `compute/server_automatic_create_test.go:TestAutomaticCreateWaitsForRawMetadataAndObservesAssignment`, `compute/server_automatic_create_test.go:TestAutomaticCreateWaitErrorsKeepLastMatchingServer`.
- After mandatory creation readiness, disabled/private/source-none or already floating/public cases return a clean automatic decision without loading IP dependencies. These fixtures use an explicit NIC ID, and still perform one NovaPOST plus one matching ACTIVE GET. This skip does not waive image/flavor/volume/NIC/default-network resolution needed by server creation.
  Tests: `compute/server_automatic_create_test.go:TestAutomaticCreateKnownSkipsDoNotLoadIPDependencies`.
- Creation uses concrete server/automatic options prepared before NovaPOST. A custom server status attribute, nil server/automatic option, unsafe floating network and image/existing-volume conflict fail before POST or role/port/raw requests. Actual-status readiness cannot be replaced with another model attribute.
  Tests: `compute/server_automatic_create_test.go:TestAutomaticCreateRejectsInvalidPolicyBeforePOST`.
- The compound preserves image, existing-volume and image-backed-new-volume native boot mappings, empty-string imageRef for volume boot, false delete-on-termination, volume type, userdata base64, security groups, metadata, owned extension fields and combined NIC network/port/fixed-IP body. Explicit IDs require no Cinder or Neutron creation lookup.
  Tests: `compute/server_automatic_create_boundaries_test.go:TestAutomaticCreatePreservesBootAndNativeBodyOptions`.
- Accepted NovaPOST200/202 and matching rawGET200 preserve decoded Creation or last Server alongside actual status/header/body evidence after Close errors, custom cancellation, Compute source replacement or an ancestor workflow guard failure. Accepted-response processing and decode failures do not invoke a native RetryFunc or issue follow-up mutation/cleanup.
  Tests: `compute/server_automatic_create_boundaries_test.go:TestAutomaticCreateAcceptedPOSTAndGETCloseKeepResources`.
- The same overall deadline bounds NovaPOST, raw readiness, Neutron role/selection/allocation/readiness and Nova convergence. Cancellation in convergence preserves its custom cause, the original Creation, last Server and allocated Assignment without fallback or deletion. A named resolver consumes the same overall timeout.
  Tests: `compute/server_automatic_create_boundaries_test.go:TestAutomaticCreateSharesDeadlineAndKeepsAssignmentOnConvergenceCancel`, `compute/server_automatic_create_preflight_test.go:TestAutomaticCreatePreparesClosuresOnceAndCapturesSourceBeforeOptions`.
- SDK-owned server/automatic/address closures are prepared once before POST. Source fingerprints precede automatic options; raw endpoint changes, server-client replacement and custom resolver Compute-source mutation stop before Nova. Stored wait policy reuses application WaitOption preparation and checks a post-progress invariant before another fetch.
  Tests: `compute/server_automatic_create_preflight_test.go:TestAutomaticCreatePreparesClosuresOnceAndCapturesSourceBeforeOptions`, `resource/prepared_wait_policy_test.go:TestPreparedWaitPolicyDoesNotReapplyApplicationOptions`, `resource/prepared_wait_policy_test.go:TestWaitGuardStopsAfterProgressBeforeAnotherFetch`.
- Connection exact-name creation references use owned paginated image/network/port/volume/flavor reads while preserving route/name-query differences. A resolved dependency remains source-bound through later stages; a subsequent dependency API replacement, initial nil API, or retry callback changing Compute stops before POST. Completed late403 and duplicate-name resolver pages remain errors; accepted204 is clean not-found and resolver source mutation stays visible after resolution.
  Tests: `connectiontest/connection_server_automatic_create_test.go:TestConnectionAutomaticCreateOwnsNamedDependencyPagesAndLifetime`, `internal/nativefind/resolve_name_test.go:TestGuardedCreationNameResolverPagesAndRetainsSource`.
- Configured default NIC selection in the automatic compound consumes one guarded role snapshot before Nova, while an existing-floating-address skip needs no IP candidate or mutation. Nested source registries keep ancestor causes and isolate child-only bindings from the original parent context.
  Tests: `connectiontest/connection_server_automatic_create_test.go:TestConnectionAutomaticCreateSharesConfiguredDefaultRoleSnapshot`, `internal/rest/operation_guard_test.go:TestNestedOperationSourcesRetainAncestorsAndIsolateChildBindings`.
- The SDK-owned prepared Neutron plan retains a decoded accepted201/202 allocation after complete-body read, Close, source or custom-cancellation errors with actual status/header/body and original cause; no retry, ACTIVE wait, observer or cleanup follows. Decoded wrong-owner allocation remains evidence alongside joined invariant/Close failures. Truncated202 read and Close causes remain joined with the actual json.SyntaxError and Allocated=true/nil model. Reused PUT200 adopts a complete matching body beside Close failure, but wrong-ID/fixed responses preserve the original candidate and both processing and validation causes; revision zero remains protected. This is owned workflow evidence, not a change to generated native transport.
  Tests: `network/floating_ip_accepted_model_test.go:TestFloatingIPAcceptedAllocationKeepsDecodedModelAndProcessingCauses`, `network/floating_ip_accepted_model_test.go:TestFloatingIPAcceptedAssociationAdoptsOnlyMatchingModelOnCloseError`, `network/floating_ip_accepted_model_test.go:TestFloatingIPAcceptedReadAndDecodeErrorsRemainJoined`.
- In the bounded automatic Neutron GetActiveServer, standalone WaitForServer, automatic creation and EnsureServerFloatingIP workflows, an accepted allocation body plus Close failure retains known Server and decoded DOWN Assignment with Allocated=true, Observed=false and actual201 response/header proof. Get/Ensure do not start raw observation; Wait/Create retain the one completed raw readiness fetch. Each has one allocation and no follow-up IP/observer/fallback/cleanup request. These fixtures do not complete explicit IP/pool or Nova dispatch.
  Tests: `compute/server_accepted_assignment_test.go:TestServerIPWorkflowsKeepAcceptedAssignmentModelOnProcessingError`.
- The same ordered explicit-address policy is consumed by typed GetActive async/sync, standalone Wait and automatic creation. Default GetActive async keeps Compute discovery lazy, returns both completed DOWN assignments with no IP readiness/raw observation; sync GetActive observes two targets, while Wait/Create perform one initial raw readiness fetch plus two observations and force actual Neutron ACTIVE despite EnsureNoWait. Creation preserves its original AdminPass separately from later Server models. The new success fixture uses image/flavor creation and explicit lists; it does not test all pool or volume/NIC combinations. GetActive BUILD returns nil, ERROR retains a failed-state result, and ACTIVE nil metadata returns the existing typed address error before dispatch and with no HTTP.
  Tests: `compute/server_ip_dispatch_workflow_test.go:TestServerIPDispatchAllWorkflowConsumersUseTheSameOrderedPolicy`, `compute/server_ip_dispatch_workflow_test.go:TestServerIPDispatchExplicitSelectorsPreserveGetReadinessBoundaries`.
- Typed automatic creation rejects selected malformed address/unsafe pool before POST. The historical known-unsupported legacy fixture includes a Neutron-only project override, so capability preflight rejects it before server POST or dependency HTTP; capable Nova backend is not rejected categorically. In the tested Neutron second-address failure, original Creation.AdminPass, latest raw Server, completed/observed first assignment and failing second attempt remain with ErrNotFound, no third address or allocation. Source wait=false creation remains separate; new bounded Nova creation has its own evidence.
  Tests: `compute/server_ip_dispatch_workflow_test.go:TestServerIPDispatchPureInvalidAndKnownUnsupportedCreateStopBeforePOST`, `compute/server_ip_dispatch_workflow_test.go:TestServerIPDispatchCreateRetainsCreationFirstIPAndSecondFailure`.
- The same bounded Nova ordered-address foundation is consumed by strong Ensure, GetActive async/sync, standalone Wait and automatic creation. Both items complete with separate NovaAssignment and no Neutron Assignment. GetActive async performs zero raw observations; Ensure/Get sync observe each item, while Wait/Create perform one initial actual ACTIVE readiness GET plus two target observations. Wait ignores supplied ERROR in favor of raw state. The image/flavor creation body is checked and original Creation.AdminPass stays separate from the last raw Server. This new upper success fixture is an explicit list, not every pool/auto/volume/NIC combination.
  Tests: `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPReadinessAndCreationConsumeLegacyBackend`.
- For strong Ensure, synchronous GetActive, Wait and creation, an exact assigned floating target under late raw BUILD does not complete: one progress failure retains the accepted action and current Server. Raw ERROR returns typed FailedState without progress. Only the first attempt starts; the next IP is not selected. Creation preserves the initial AdminPass. Nova has no IP ACTIVE field to poll or synthesize; these checks prove actual server readiness and raw target policy, while standalone Add keeps its any-status observation contract.
  Tests: `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPUpperObservationRequiresActualActiveServer`.
- The Nova direct-list lane rejects selected Compute2.36/2.44 and Neutron-only port/NAT-network/project overrides with ErrUnsupported before HTTP or attempts; a nonnumeric latest microversion is an invalid option. Known legacy capability preflight for automatic creation separately rejects2.36, explicit port and project overrides before any server POST or other fixture HTTP. These capability limits are distinct from a blanket Nova backend prohibition.
  Tests: `compute/server_ip_nova_test.go:TestNovaServerIPVersionAndNeutronSelectorsFailBeforeHTTP`, `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPKnownCreateCapabilityStopsBeforeServerPOST`.
- Create sends Python's top-level OS-SCH-HNT:scheduler_hints object (never os:scheduler_hints) with JSON-snapshotted values; WithServerGroup resolves a name through one os-server-groups list (IDs skip lookup) and wins over a group hint. min/max count, explicit empty description, adminPass, OS-DCF:diskConfig, access IPs, tags, hostname, hypervisor hostname and base64 personality are serialized, count keys are omitted without WithServerCount, and userdata is always base64-encoded even when it already decodes.
  Tests: `compute/server_create_lifecycle_test.go:TestServerCreatePropertiesSchedulerHintsGroupAndCount`.
- WithDataVolumes appends boot_index -1/delete_on_termination false volume mappings after the root mapping, prepending a local image mapping (boot_index 0, delete_on_termination true) only for image boot, in Python's order. Existing and new boot volumes keep imageRef empty; names resolve through Block Storage before POST.
  Tests: `compute/server_create_lifecycle_test.go:TestServerCreateDataVolumeMappingsFollowPythonOrder`.
- Invalid counts, disk configs, access IP families, unserializable or empty-key hints, empty tags/admin password/volume lists/hostnames/personality and zero refs fail as ErrInvalidOption before any request. Description (2.19), tags (2.52), hypervisor hostname (2.74), hostname (2.90) and personality (removed at 2.57) are checked against the client microversion as ErrUnsupported before HTTP. Missing or duplicate server group names fail after the lookup and before POST.
  Tests: `compute/server_create_lifecycle_test.go:TestServerCreatePropertiesRejectBeforeRequests`.
- A raw ERROR server returns ServerFaultError (unwrapping to FailedStateError) using Python's key-presence test: a non-null fault object with a message key gives 'Compute service reports fault: <message>' (empty string kept, null formatted as None); a missing, null or message-less fault gives 'no further information available'. Server carries the observed model like extra_data. A supplied typed Server uses a non-empty Fault.Message.
  Tests: `compute/server_create_lifecycle_test.go:TestServerFaultErrorFollowsPythonKeyPresence`.
- ACTIVE without address rows keeps the server by default. DeleteServerWithoutAddresses sends exactly one DELETE and still returns ServerAddressesUnavailableError with Deletion evidence: 204 without a wait in async GetActiveServer, 404 recorded as NotFound without an error, 409 joined as 'AND then could not be deleted' without retry, and WaitForServer/CreateWithAutomaticFloatingIP polling GET until 404. Unknown policies fail before POST.
  Tests: `compute/server_create_lifecycle_test.go:TestServerWithoutAddressesDeletePolicyIsExplicitAndSingleShot`.
- Connection.CreateWithAutomaticFloatingIP resolves server group and data volume names through Connection-owned Compute and Block Storage lookups, sends hints and mappings in one POST, and applies the default keep or explicit delete-and-wait policy to an ACTIVE server without addresses while retaining Creation adminPass.
  Tests: `connectiontest/connection_server_create_lifecycle_test.go:TestConnectionCreateLifecycleResolvesGroupVolumesAndDeletesOnlyWhenSelected`.
- WithCreateRefresh performs exactly one GET of the created server after an accepted POST and returns the refreshed model; ERROR returns ServerFaultError with Python's fault message and the refreshed Server, while a failed or mismatched GET returns the POST server with the error and never repeats the POST. Combining it with WithWait fails before POST.
  Tests: `compute/server_create_lifecycle_test.go:TestServerCreateRefreshPerformsOneGetAndReportsFault`.
- Explicit block_device_mapping_v2 entries serialize every per-mapping field (null boot_index, delete_on_termination, device name/type, disk bus, guest format, volume size/type, tag, no_device) and are placed before generated local-image and data-volume mappings, as Python appends generated mappings to supplied ones. Legacy block_device_mapping entries and trusted_image_certificates are sent as given. Legacy plus v2, invalid source/destination types and empty lists fail as ErrInvalidOption, and tag (2.42), volume type (2.67) and certificates (2.63) fail as ErrUnsupported before POST.
  Tests: `compute/server_create_mappings_test.go:TestServerCreateExplicitLegacyMappingsAndTrustedCertificates`.
- CreateReservation sends one POST with return_reservation_id true and the requested min/max count and returns only the reservation ID with response metadata. A server-shaped or empty reservation response is an error without a second POST, and WithWait or WithCreateRefresh fail before POST.
  Tests: `compute/server_create_mappings_test.go:TestServerCreateReservationReturnsOnlyReservationID`.
- CreateRefreshedServer performs one POST and one GET, copies the creation adminPass onto the refreshed Server and returns the add_server_interfaces-equivalent ExpandServerInterfaces view. A refreshed ERROR server returns ServerFaultError pointing at the returned Server without expansion, and WithWait fails before POST. The Connection wrapper uses the Connection-owned Compute and Network services.
  Tests: `compute/server_create_mappings_test.go:TestCreateRefreshedServerExpandsAndCopiesAdminPass`, `connectiontest/connection_server_create_refresh_test.go:TestConnectionCreateRefreshedServerPostsOnceAndExpands`.
- With DeleteServerWithoutAddresses, only Python's falsy address maps (null or empty) are deleted. A non-empty map whose rows are all empty or null returns ServerAddressesUnavailableError with EmptyRows and no DELETE in both GetActiveServer and WaitForServer.
  Tests: `compute/server_create_mappings_test.go:TestServerWithoutAddressesDeleteSkipsTruthyEmptyRowMaps`.
- Through CreateWithAutomaticFloatingIP the boot source, data volumes and NIC selection combine in one POST: image boot with explicit NIC and data volume (local image mapping then data mapping), existing boot volume with fixed-IP and port NICs (empty imageRef), new image-backed boot volume with a default network and data volume, a new boot volume without NICs at advertised 2.37 (networks auto, request 2.37, one discovery) and image boot without NICs below advertised 2.37 (no networks key, no version header). Explicit NICs and default networks need no discovery.
  Tests: `compute/server_create_combinations_test.go:TestAutomaticCreateCombinesBootVolumesAndNetworkSelections`.
- After creation, add_ips_to_server precedence holds: a Neutron pool wins over explicit addresses and auto (allocated in the pool, no address lookup), explicit addresses win over enabled auto (reuse, no allocation), auto by default allocates on the external network after a networks=auto POST, and source none disables auto. The legacy Nova source at a pinned 2.35 client takes the default pool for auto, the selected pool over addresses, or the explicit address, each with one accepted addFloatingIp action and raw observation, while the POST omits networks without discovery.
  Tests: `compute/server_create_combinations_test.go:TestAutomaticCreateIPLanesFollowPythonPrecedence`.

Python/Go differences:

- The SDK owns concrete NIC inputs and an Optional tag field; request.Present/Null distinguish omitted, empty and null without a caller builder. The owned network body overlay avoids native empty-string tag omission.
- Network and port are explicit safe Ref IDs/names. Connection adds port name resolution, while Python cloud port/port-id values are literal IDs. The SDK copies inputs instead of mutating Python NIC dictionaries.
- The final selection option wins. Empty network/NIC selections are invalid options rather than Python empty-list fallback.
- Version gates follow supports_microversion/require_microversion with the client's selected microversion as the adapter default. Python sends an unselected request at min(Server._max_microversion 2.100, advertised maximum); Go keeps the base version and raises only the create POST to the gate it needs (2.37 for auto, 2.42 for tags), so other body fields keep their base-version validation. A selection below the gate decides without discovery. Python's discovery failure yields unsupported; Go treats only a clean 404/405 or missing bound as unsupported and returns other discovery failures before the POST rather than silently changing the networks body. Explicit WithNetworkMode keeps its selected-2.37 check without discovery, which Python's cloud layer does not expose.
- Connection supplies configured defaults without an application builder/resolver. Explicit Ref IDs bypass Neutron and Ref names use exact-name lookup. Configured YAML/role defaults match name OR ID through shared successful role discovery, while explicit typed default Name remains fresh. Discovery errors preserve their causes and are not cached as empty success.
- CreateWithFloatingIP explicitly requests one Neutron IPv4 assignment and always waits actual server/IP ACTIVE under a five-minute budget including pre-creation runtime work. Python cloud wait timeout defaults to180s after creation and refreshes server interface addresses; generic Create remains asynchronous and the source auto_ip need/skip policy is not inferred from this explicit workflow.
- The explicit Service/Connection CreateWithAutomaticFloatingIP compound connects owned Nova creation, raw actual ACTIVE/address readiness and selected Neutron or bounded legacy Nova dispatch with separate Creation/last Server/Automatic results. It always waits, rejects alternate status selectors and preserves original creation-only AdminPass. Source create_server defaults wait=false and only forwards IP dispatch through its wait=true path; ordinary Servers.Create and the older explicit workflow retain their separate contracts.
- The new guarded named-creation lane retains native typed models and200/204/300 list codes but owns page continuation and accepted Read/Close/source errors. Ordinary native lookup remains separate; flavor-name creation retains its existing no-name-query/no-is_public-default behavior rather than completing the Python flavor-list defaults.
- Create property options are typed and validated locally: version-gated fields fail against the client microversion before HTTP rather than relying on Nova 400s, falsy values such as config_drive false are sent instead of dropped, empty admin passwords/tags are rejected, and min_count/max_count are omitted unless WithServerCount is used (Python's cloud layer always sends 1/1). Boot indexes are JSON integers where Python sends some as strings, and group/data-volume lookups run after image/flavor/network resolution.
- ERROR servers return ServerFaultError with Python's two message forms and the observed Server instead of SDKException extra_data. Raw reads apply the exact fault/message key-presence test; a supplied typed Server cannot record key presence, so a non-empty Fault.Message counts as present. Non-string fault messages fail the typed decode.
- ACTIVE-without-addresses cleanup is an explicit policy: the default keeps the server, and DeleteServerWithoutAddresses performs Python's single DELETE (404 tolerated) and deletion wait inside the remaining workflow budget, recording Deletion evidence. A failed DELETE or wait is joined to ServerAddressesUnavailableError and never retried.
- config_drive is a typed boolean; Python's documented volume-ID value is not modeled because Nova's create schema accepts only booleans. Servers.Create with a count returns Nova's single server response; CreateReservation returns the reservation ID instead. Python's Server resource has no return_reservation_id attribute, and its reservation_id kwarg is a server body attribute rather than this response shape. Explicit mapping UUIDs are sent literally without name resolution.
- The creation adminPass is copied onto a detached returned Server when the latest GET has none, as Python sets admin_password; Creation and the Automatic evidence keep the raw models.
- A non-empty address map with only empty or null rows returns ServerAddressesUnavailableError with EmptyRows and is never deleted. Python treats that map as truthy and proceeds to add_ips_to_server; Go does not attempt IP assignment on a server with no address rows.
- Results are immutable owned snapshots rather than Python's mutable Resource with adapter, session, cache and dirty state (SDK-R1/C1/S1); see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Inputs are typed Go values; arbitrary Python objects, iterables and numeric types accepted by dynamic kwargs have no equivalent; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Network roles are cached per Connection and reset by the SDK's own CreateNetwork, UpdateNetwork and DeleteNetwork, as Python resets its role cache only inside cloud create_network, update_network and delete_network; changes made by other clients or through the raw NetworkV2 client need an explicit Reset, which Python does not see either.
- Evidence is local HTTP fixtures and pinned-source comparison; authenticated deployment acceptance is a separate evidence class shared by every operation (see docs/sdk-support-ledger.md).

### `python:compute/v2/create_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Servers.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithNetworkInterfaces`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithNetworkMode`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithNetworks`
- `github.com/JSYoo5B/go-openstacksdk.WithDefaultNetwork`
- `github.com/JSYoo5B/go-openstacksdk.WithoutDefaultNetwork`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithSchedulerHints`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerGroup`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerCount`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithDescription`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAdminPass`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithDiskConfig`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAccessIPv4`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAccessIPv6`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithTags`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithHostname`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithHypervisorHostname`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithPersonality`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithDataVolumes`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithUserData`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithBlockDeviceMappings`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithLegacyBlockDeviceMappings`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithTrustedImageCertificates`
- `github.com/JSYoo5B/go-openstacksdk/compute.Servers.CreateReservation`

- Ordered NICs preserve network UUID and port combinations, literal fixed IP, fixed-IP-only/tag-only/empty objects, and omitted versus explicit empty or null tag values in the HTTP body. IDs bypass Neutron lookup and need no Network endpoint.
  Tests: `compute/server_networks_test.go:TestServerNICWirePreservesCombinationsOrderAndExplicitEmptyTag`, `connectiontest/connection_server_nics_test.go:TestConnectionServerNICIDsNeedNoNetworkEndpoint`.
- Connection resolves exact network/port names and preserves missing/ambiguous/403 causes without creating a server. Resolver failures, empty/unsafe returned IDs and cancellation stop before Nova.
  Tests: `connectiontest/connection_server_nics_test.go:TestConnectionServerNICsResolveExactNamesAndOwnInputs`, `connectiontest/connection_server_nics_test.go:TestConnectionServerNICNameFailuresPreventNovaCreation`, `compute/server_networks_test.go:TestServerNICResolverFailuresAndCancellationStopBeforeNova`.
- The last valid WithNetworks/NIC/mode selection replaces the previous selection, so discarded names or tag gates have no effect. Options own input slices and Optional tag values and can be reused concurrently.
  Tests: `compute/server_networks_test.go:TestServerNICSelectionUsesLastValidOption`, `compute/server_networks_test.go:TestServerNICOptionsOwnSlicesTagsAndConcurrentReuse`, `connectiontest/connection_server_nics_test.go:TestConnectionServerNICsResolveExactNamesAndOwnInputs`.
- Mode auto/none requires selected Compute 2.37+, and every present tag, including empty or null, requires selected 2.42+. Invalid options/Refs/version gates fail before dependency lookup; the shared client version and Nova headers are preserved.
  Tests: `compute/server_networks_test.go:TestServerNICInvalidOptionsFailBeforeDependencyLookup`, `compute/server_networks_test.go:TestServerNICWirePreservesCombinationsOrderAndExplicitEmptyTag`.
- Without explicit NICs, mode or a Connection/YAML default, Create follows supports_microversion(compute, "2.37") from openstack/utils.py: the advertised Compute range must contain 2.37 and a selected client version must be at least 2.37. Then networks is auto; an unselected client raises only this POST to 2.37 and leaves the shared client unset. A selection below 2.37 omits networks without discovery; a range below 2.37, a clean 404/405 discovery or a selection beyond the advertised range omits the key. One discovery GET per create; Connection negotiation reuses its recorded advertised bounds without another discovery or Neutron lookup.
  Tests: `compute/server_networks_test.go:TestServerNICDefaultsModesAndVersionBoundaries`, `connectiontest/connection_server_nics_test.go:TestConnectionNegotiatedVersionControlsDefaultNetworkAuto`.
- NIC choices preserve existing/new volume boot mapping and false delete-on-termination. An ACTIVE wait failure returns the created BUILD server and the error without server/port/volume cleanup.
  Tests: `compute/server_networks_test.go:TestServerNICsPreserveVolumeBootAndCreatedServerOnWaitFailure`.
- Connection owns a default-network Ref and explicit disable policy; the last option wins, invalid Refs remain errors, explicit NICs/modes bypass the default callback and Neutron endpoint lookup, and default IDs work for image/existing-volume/new-volume boots without altering their mappings.
  Tests: `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkIDsOptionsAndBootSources`, `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkDisableAndExplicitSelections`, `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkOptionValidationAndConcurrentReuse`, `compute/default_network_test.go:TestServerDefaultNetworkIsSkippedByExplicitNICsAndPreflight`.
- Typed default names resolve exactly; missing/ambiguous/403 failures, malformed callback Refs and cancellation stop before server creation. Explicit typed default-name resolution remains fresh; configured defaults share successful role discovery and failed discovery is not cached; callbacks and options are request-local, and wait failure preserves the created server without cleanup.
  Tests: `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkNameFailuresAndRetry`, `compute/default_network_test.go:TestServerDefaultNetworkDependencyFailureCancellationAndValidation`, `compute/default_network_test.go:TestServerDefaultNetworkDoesNotLeakBetweenRequestsAndPreservesWaitFailure`.
- Cloud/secure/public files are snapshotted once for native auth/endpoint/TLS and SDK metadata. Network lists replace at each precedence layer, [] clears inheritance, null/duplicate defaults/invalid flags are configuration errors, and empty/null metadata profile stops network inheritance while retaining native auth fallback.
  Tests: `connection_cloud_config_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connection_cloud_config_test.go:TestCloudNetworkConfigurationSearchAndReadFailures`, `connection_cloud_config_test.go:TestCloudNetworkConfigurationValidation`.
- YAML default name matches exact network name OR ID across unfiltered complete pagination. Cross-kind collisions and duplicate names are ambiguous; missing/unsafe IDs, a later-page 403 and cancellation preserve errors and prevent Nova POST. Connection override/disable and explicit Create selection bypass the configured lookup.
  Tests: `connectiontest/connection_cloud_network_test.go:TestConnectConfiguredDefaultNetworkMatchesNamesOrIDsAcrossAllPages`, `connectiontest/connection_cloud_network_test.go:TestConnectDefaultNetworkOverridesDisableAndExplicitCreateSelection`.
- A configured default NIC consumes the same successful network/subnet role snapshot as Connection getters. Caller changes to a returned default cannot alter creation; consecutive creates reuse the successful discovery and Reset refreshes the chosen default ID for the next create.
  Tests: `connectiontest/connection_network_role_consumers_test.go:TestConnectionDefaultNICConsumesRoleCacheAndReset`, `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Configured default discovery validates the shared snapshot, so subnet403 or another configured role missing stops before Nova POST even when the default itself exists. Failed attempts are not cached; a subsequent success is reused. Later network-page errors, name-or-ID collisions, unsafe IDs and cancellation retain their established pre-create errors.
  Tests: `connectiontest/connection_network_role_consumers_test.go:TestConnectionDefaultNICRoleErrorsStopCreationAndRetry`, `connectiontest/connection_cloud_network_test.go:TestConnectConfiguredDefaultNetworkMatchesNamesOrIDsAcrossAllPages`.
- Both-disabled discovery or a clean pointer/value missing Network endpoint yields no configured default and keeps Nova auto at selected2.37 or omits networks when unset. Explicit default ID/disable and explicit NIC/mode bypass configured role discovery; explicit typed default names keep their exact-name resolution lane.
  Tests: `connectiontest/connection_network_role_consumers_test.go:TestConnectionDefaultNICEmptyRolesUseNovaNetworkDefault`, `connectiontest/connection_cloud_network_test.go:TestConnectDefaultNetworkOverridesDisableAndExplicitCreateSelection`, `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkNameFailuresAndRetry`.
- Create sends Python's top-level OS-SCH-HNT:scheduler_hints object (never os:scheduler_hints) with JSON-snapshotted values; WithServerGroup resolves a name through one os-server-groups list (IDs skip lookup) and wins over a group hint. min/max count, explicit empty description, adminPass, OS-DCF:diskConfig, access IPs, tags, hostname, hypervisor hostname and base64 personality are serialized, count keys are omitted without WithServerCount, and userdata is always base64-encoded even when it already decodes.
  Tests: `compute/server_create_lifecycle_test.go:TestServerCreatePropertiesSchedulerHintsGroupAndCount`.
- WithDataVolumes appends boot_index -1/delete_on_termination false volume mappings after the root mapping, prepending a local image mapping (boot_index 0, delete_on_termination true) only for image boot, in Python's order. Existing and new boot volumes keep imageRef empty; names resolve through Block Storage before POST.
  Tests: `compute/server_create_lifecycle_test.go:TestServerCreateDataVolumeMappingsFollowPythonOrder`.
- Invalid counts, disk configs, access IP families, unserializable or empty-key hints, empty tags/admin password/volume lists/hostnames/personality and zero refs fail as ErrInvalidOption before any request. Description (2.19), tags (2.52), hypervisor hostname (2.74), hostname (2.90) and personality (removed at 2.57) are checked against the client microversion as ErrUnsupported before HTTP. Missing or duplicate server group names fail after the lookup and before POST.
  Tests: `compute/server_create_lifecycle_test.go:TestServerCreatePropertiesRejectBeforeRequests`.
- Explicit block_device_mapping_v2 entries serialize every per-mapping field (null boot_index, delete_on_termination, device name/type, disk bus, guest format, volume size/type, tag, no_device) and are placed before generated local-image and data-volume mappings, as Python appends generated mappings to supplied ones. Legacy block_device_mapping entries and trusted_image_certificates are sent as given. Legacy plus v2, invalid source/destination types and empty lists fail as ErrInvalidOption, and tag (2.42), volume type (2.67) and certificates (2.63) fail as ErrUnsupported before POST.
  Tests: `compute/server_create_mappings_test.go:TestServerCreateExplicitLegacyMappingsAndTrustedCertificates`.
- CreateReservation sends one POST with return_reservation_id true and the requested min/max count and returns only the reservation ID with response metadata. A server-shaped or empty reservation response is an error without a second POST, and WithWait or WithCreateRefresh fail before POST.
  Tests: `compute/server_create_mappings_test.go:TestServerCreateReservationReturnsOnlyReservationID`.

Python/Go differences:

- The SDK owns concrete NIC inputs and an Optional tag field; request.Present/Null distinguish omitted, empty and null without a caller builder. The owned network body overlay avoids native empty-string tag omission.
- Network and port are explicit safe Ref IDs/names. Connection adds port name resolution, while Python cloud port/port-id values are literal IDs. The SDK copies inputs instead of mutating Python NIC dictionaries.
- The final selection option wins. Empty network/NIC selections are invalid options rather than Python empty-list fallback.
- Connection supplies configured defaults without an application builder/resolver. Explicit Ref IDs bypass Neutron and Ref names use exact-name lookup. Configured YAML/role defaults match name OR ID through shared successful role discovery, while explicit typed default Name remains fresh. Discovery errors preserve their causes and are not cached as empty success.
- Create property options are typed and validated locally: version-gated fields fail against the client microversion before HTTP rather than relying on Nova 400s, falsy values such as config_drive false are sent instead of dropped, empty admin passwords/tags are rejected, and min_count/max_count are omitted unless WithServerCount is used (Python's cloud layer always sends 1/1). Boot indexes are JSON integers where Python sends some as strings, and group/data-volume lookups run after image/flavor/network resolution.
- config_drive is a typed boolean; Python's documented volume-ID value is not modeled because Nova's create schema accepts only booleans. Servers.Create with a count returns Nova's single server response; CreateReservation returns the reservation ID instead. Python's Server resource has no return_reservation_id attribute, and its reservation_id kwarg is a server body attribute rather than this response shape. Explicit mapping UUIDs are sent literally without name resolution.
- Results are immutable owned snapshots rather than Python's mutable Resource with adapter, session, cache and dirty state (SDK-R1/C1/S1); see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Inputs are typed Go values; arbitrary Python objects, iterables and numeric types accepted by dynamic kwargs have no equivalent; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Version gates follow supports_microversion/require_microversion with the client's selected microversion as the adapter default. Python sends an unselected request at min(Server._max_microversion 2.100, advertised maximum); Go keeps the base version and raises only the create POST to the gate it needs (2.37 for auto, 2.42 for tags), so other body fields keep their base-version validation. A selection below the gate decides without discovery. Python's discovery failure yields unsupported; Go treats only a clean 404/405 or missing bound as unsupported and returns other discovery failures before the POST rather than silently changing the networks body. Explicit WithNetworkMode keeps its selected-2.37 check without discovery, which Python's cloud layer does not expose.

### `gophercloud:compute/v2/servers.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Servers.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithNetworkInterfaces`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithNetworkMode`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithNetworks`
- `github.com/JSYoo5B/go-openstacksdk.WithDefaultNetwork`
- `github.com/JSYoo5B/go-openstacksdk.WithoutDefaultNetwork`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithCreateField`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithCreateHintOpts`

- Ordered NICs preserve network UUID and port combinations, literal fixed IP, fixed-IP-only/tag-only/empty objects, and omitted versus explicit empty or null tag values in the HTTP body. IDs bypass Neutron lookup and need no Network endpoint.
  Tests: `compute/server_networks_test.go:TestServerNICWirePreservesCombinationsOrderAndExplicitEmptyTag`, `connectiontest/connection_server_nics_test.go:TestConnectionServerNICIDsNeedNoNetworkEndpoint`.
- Connection resolves exact network/port names and preserves missing/ambiguous/403 causes without creating a server. Resolver failures, empty/unsafe returned IDs and cancellation stop before Nova.
  Tests: `connectiontest/connection_server_nics_test.go:TestConnectionServerNICsResolveExactNamesAndOwnInputs`, `connectiontest/connection_server_nics_test.go:TestConnectionServerNICNameFailuresPreventNovaCreation`, `compute/server_networks_test.go:TestServerNICResolverFailuresAndCancellationStopBeforeNova`.
- The last valid WithNetworks/NIC/mode selection replaces the previous selection, so discarded names or tag gates have no effect. Options own input slices and Optional tag values and can be reused concurrently.
  Tests: `compute/server_networks_test.go:TestServerNICSelectionUsesLastValidOption`, `compute/server_networks_test.go:TestServerNICOptionsOwnSlicesTagsAndConcurrentReuse`, `connectiontest/connection_server_nics_test.go:TestConnectionServerNICsResolveExactNamesAndOwnInputs`.
- Mode auto/none requires selected Compute 2.37+, and every present tag, including empty or null, requires selected 2.42+. Invalid options/Refs/version gates fail before dependency lookup; the shared client version and Nova headers are preserved.
  Tests: `compute/server_networks_test.go:TestServerNICInvalidOptionsFailBeforeDependencyLookup`, `compute/server_networks_test.go:TestServerNICWirePreservesCombinationsOrderAndExplicitEmptyTag`.
- Without explicit NICs, mode or a Connection/YAML default, Create follows supports_microversion(compute, "2.37") from openstack/utils.py: the advertised Compute range must contain 2.37 and a selected client version must be at least 2.37. Then networks is auto; an unselected client raises only this POST to 2.37 and leaves the shared client unset. A selection below 2.37 omits networks without discovery; a range below 2.37, a clean 404/405 discovery or a selection beyond the advertised range omits the key. One discovery GET per create; Connection negotiation reuses its recorded advertised bounds without another discovery or Neutron lookup.
  Tests: `compute/server_networks_test.go:TestServerNICDefaultsModesAndVersionBoundaries`, `connectiontest/connection_server_nics_test.go:TestConnectionNegotiatedVersionControlsDefaultNetworkAuto`.
- NIC choices preserve existing/new volume boot mapping and false delete-on-termination. An ACTIVE wait failure returns the created BUILD server and the error without server/port/volume cleanup.
  Tests: `compute/server_networks_test.go:TestServerNICsPreserveVolumeBootAndCreatedServerOnWaitFailure`.
- Connection owns a default-network Ref and explicit disable policy; the last option wins, invalid Refs remain errors, explicit NICs/modes bypass the default callback and Neutron endpoint lookup, and default IDs work for image/existing-volume/new-volume boots without altering their mappings.
  Tests: `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkIDsOptionsAndBootSources`, `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkDisableAndExplicitSelections`, `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkOptionValidationAndConcurrentReuse`, `compute/default_network_test.go:TestServerDefaultNetworkIsSkippedByExplicitNICsAndPreflight`.
- Typed default names resolve exactly; missing/ambiguous/403 failures, malformed callback Refs and cancellation stop before server creation. Explicit typed default-name resolution remains fresh; configured defaults share successful role discovery and failed discovery is not cached; callbacks and options are request-local, and wait failure preserves the created server without cleanup.
  Tests: `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkNameFailuresAndRetry`, `compute/default_network_test.go:TestServerDefaultNetworkDependencyFailureCancellationAndValidation`, `compute/default_network_test.go:TestServerDefaultNetworkDoesNotLeakBetweenRequestsAndPreservesWaitFailure`.
- Cloud/secure/public files are snapshotted once for native auth/endpoint/TLS and SDK metadata. Network lists replace at each precedence layer, [] clears inheritance, null/duplicate defaults/invalid flags are configuration errors, and empty/null metadata profile stops network inheritance while retaining native auth fallback.
  Tests: `connection_cloud_config_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connection_cloud_config_test.go:TestCloudNetworkConfigurationSearchAndReadFailures`, `connection_cloud_config_test.go:TestCloudNetworkConfigurationValidation`.
- YAML default name matches exact network name OR ID across unfiltered complete pagination. Cross-kind collisions and duplicate names are ambiguous; missing/unsafe IDs, a later-page 403 and cancellation preserve errors and prevent Nova POST. Connection override/disable and explicit Create selection bypass the configured lookup.
  Tests: `connectiontest/connection_cloud_network_test.go:TestConnectConfiguredDefaultNetworkMatchesNamesOrIDsAcrossAllPages`, `connectiontest/connection_cloud_network_test.go:TestConnectDefaultNetworkOverridesDisableAndExplicitCreateSelection`.
- A configured default NIC consumes the same successful network/subnet role snapshot as Connection getters. Caller changes to a returned default cannot alter creation; consecutive creates reuse the successful discovery and Reset refreshes the chosen default ID for the next create.
  Tests: `connectiontest/connection_network_role_consumers_test.go:TestConnectionDefaultNICConsumesRoleCacheAndReset`, `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Configured default discovery validates the shared snapshot, so subnet403 or another configured role missing stops before Nova POST even when the default itself exists. Failed attempts are not cached; a subsequent success is reused. Later network-page errors, name-or-ID collisions, unsafe IDs and cancellation retain their established pre-create errors.
  Tests: `connectiontest/connection_network_role_consumers_test.go:TestConnectionDefaultNICRoleErrorsStopCreationAndRetry`, `connectiontest/connection_cloud_network_test.go:TestConnectConfiguredDefaultNetworkMatchesNamesOrIDsAcrossAllPages`.
- Both-disabled discovery or a clean pointer/value missing Network endpoint yields no configured default and keeps Nova auto at selected2.37 or omits networks when unset. Explicit default ID/disable and explicit NIC/mode bypass configured role discovery; explicit typed default names keep their exact-name resolution lane.
  Tests: `connectiontest/connection_network_role_consumers_test.go:TestConnectionDefaultNICEmptyRolesUseNovaNetworkDefault`, `connectiontest/connection_cloud_network_test.go:TestConnectDefaultNetworkOverridesDisableAndExplicitCreateSelection`, `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkNameFailuresAndRetry`.
- Create POSTs {"server": ...} with required name, base64 user_data, named security groups and auto/none or structured networks, places extension fields inside the single server envelope and scheduler hints at top-level os:scheduler_hints, rejects invalid networks, invalid hints, collisions and nil options before HTTP, and accepts 200 and 202.
  Tests: `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCRUDRoutesBodiesAndDecode`, `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCRUDStrictStatusesAndPreflight`.
- The remaining CreateOpts fields keep their pinned JSON names and omitempty rules: availability zone, metadata, base64 personality files, admin password, access IPs, tags, hostname, block_device_mapping_v2 with boot_index/delete_on_termination always sent, OS-DCF:diskConfig, hypervisor_hostname and structured or none networks; the selected client microversion header reaches the request.
  Tests: `compute/v2/servers/native_crud_contracts_test.go:TestNativeServerCreateFullOptionsAndMicroversion`.

Python/Go differences:

- The SDK owns concrete NIC inputs and an Optional tag field; request.Present/Null distinguish omitted, empty and null without a caller builder. The owned network body overlay avoids native empty-string tag omission.
- Network and port are explicit safe Ref IDs/names. Connection adds port name resolution, while Python cloud port/port-id values are literal IDs. The SDK copies inputs instead of mutating Python NIC dictionaries.
- The final selection option wins. Empty network/NIC selections are invalid options rather than Python empty-list fallback.
- Connection supplies configured defaults without an application builder/resolver. Explicit Ref IDs bypass Neutron and Ref names use exact-name lookup. Configured YAML/role defaults match name OR ID through shared successful role discovery, while explicit typed default Name remains fresh. Discovery errors preserve their causes and are not cached as empty success.
- Version gates follow supports_microversion/require_microversion with the client's selected microversion as the adapter default. Python sends an unselected request at min(Server._max_microversion 2.100, advertised maximum); Go keeps the base version and raises only the create POST to the gate it needs (2.37 for auto, 2.42 for tags), so other body fields keep their base-version validation. A selection below the gate decides without discovery. Python's discovery failure yields unsupported; Go treats only a clean 404/405 or missing bound as unsupported and returns other discovery failures before the POST rather than silently changing the networks body. Explicit WithNetworkMode keeps its selected-2.37 check without discovery, which Python's cloud layer does not expose.

### `python:compute/v2/create_console`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.CreateConsole`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithConsoleProtocol`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithConsoleProtocolValue`

- CreateConsole first reads the captured endpoint version path without body/query/microversion headers, preserving the selected live token, source header and reverse-proxy prefix. Nova row eligibility requires a permitted status and usable self-link; missing ID/status, unknown or experimental status (even with malformed ID), or no self-link is excluded without following the link. Catalog v2.1 selects an exact eligible v2.1 API row rather than v2.0; flat/version/versions/versions.values documents use the common engine. Clean404/405 allows only the finite root attempt and valid accepted201/300 discovery is parsed.
  Tests: `api/compute_console_composition_test.go:TestComputeConsoleCompositionAdvertisedAndSelectedBranches`.
- Modern dispatch requires both advertised min/max and2.6 inside that range, plus same-major selected support when present. Unselected old max, a minimum above2.6, missing bounds, selected2.5 and global latest use legacy. Explicit selected2.100 above advertised2.90 remains unchanged; finite2.latest supports modern. Absent selected version is privately capped at modern2.99 or legacy2.100; original client binding, endpoint, ResourceBase, version and headers are preserved and only one winning console POST occurs.
  Tests: `api/compute_console_composition_test.go:TestComputeConsoleCompositionAdvertisedAndSelectedBranches`.
- Protocol defaults to explicit null, explicit nonempty/empty values are preserved as response seeds, and a cleared Optional restores the null facade default. Modern known-type protocol inference affects only the transmitted body; unknown modern type sends null for server validation. Webmks2.8 and spice-direct2.99 each require both advertised and selected support before POST, including explicit selected values above cloud max. Legacy ignores protocol and imposes no modern spice-direct gate.
  Tests: `api/compute_console_composition_test.go:TestComputeConsoleCompositionProtocolTypeAndLocationProjection`.
- Modern result has exactly protocol/type/url/id/name/computed location; URI server_id and unknown/response location fields are excluded. Actual known untyped/null/large-number id values overlay seeds, missing id/name become null, and configured scoped location is independently captured once before POST with zone null. Direct no-dependency location is null. Legacy console scalar/array/null and vendor/large-number object values remain precise independently owned JSON.
  Tests: `api/compute_console_composition_test.go:TestComputeConsoleCompositionProtocolTypeAndLocationProjection`, `api/compute_console_composition_test.go:TestComputeConsoleCompositionAdvertisedAndSelectedBranches`.
- Option source/context failures and replacement of API, Servers or RemoteConsoles binding stop later callbacks and discovery; location errors/source drift after successful discovery stop console POST. Original option error and custom cancellation causes are preserved. Accepted discovery read/nested404 Close/source/cancellation/empty204 faults remain terminal with only physical discovery receipt and no root/action retry; shared fault wrappers check actual request/read/Close counts.
  Tests: `api/compute_console_composition_test.go:TestComputeConsoleCompositionGuardsDiscoveryFailuresAndNoErrorFallback`.
- Discovery HTTP403 preserves actual GET URL/body/header/status and prevents root/action fallback. Modern403/404 never invokes legacy and legacy404 never invokes modern. Modern accepted opaque202 retains seeded dictionary, while legacy opaque202 is an accepted response processing error; no failed console action is resent through the other executor.
  Tests: `api/compute_console_composition_test.go:TestComputeConsoleCompositionGuardsDiscoveryFailuresAndNoErrorFallback`.

Python/Go differences:

- Python accepts a Server Resource or ID and dynamic console protocol; Go takes a validated explicit single path-segment text server ID, mandatory string type and optional nullable string protocol. The independently owned json.RawMessage is the modern six-field source dictionary or the exact legacy console value, rather than a mutable connected Resource. The example supplies its own novnc type default.
- Go reuses guarded finite operation-owned endpoint/version-root discovery and tuple support/selection from the Cinder engine. Discovery is versionless and accepted final200..399 is parsed; actual native clean404/405 or a valid unmatched API row alone allows the root attempt. Advertised links do not grant routing/authentication authority, and source/context/physical faults are terminal. This operation does not reproduce the complete cached Keystoneauth session/discovery lifecycle or response charset detection; invalid UTF8/JSON is a receipt error. The fixed-route Go reader uses the first matching eligible API row instead of Python version-data sorting and normalization of all unrelated rows. Go uses the captured authenticated token rather than Python anonymous discovery and401 authenticated fallback; Go terminal errors and clean404/405-only unavailable fallback are deliberate operation-scoped error semantics, while Keystoneauth can try later URLs after broader discovery/HTTP/connection failures.
- An explicit selected microversion is preserved even above advertised max, with finite2.latest matching and global latest failing finite-major support. Absent selection uses a private action client capped at modern2.99 or legacy2.100; Connection client/version policy remains owned and unchanged. Direct compute.New with no CloudLocation dependency returns location null, while conn.Compute supplies the configured scoped computed location. Discovery may run once per operation rather than through Python global discovery cache.
- Branch-specific decoding policy is inherited from the owned executors: modern accepted opaque/empty JSON bodies retain original type/protocol seed with null missing fields, while legacy requires an actual console key and returns its raw JSON value. Go physical read/Close/source/context failures retain actual evidence and do not become tolerated JSON failure. This single composed declaration does not add a new HTTP endpoint or count existing executors again.

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

### `python:compute/v2/get_flavor_extra_specs_property`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.GetExtraSpecsProperty`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorExtraSpecsOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorExtraSpecsMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorExtraSpecsHeader`

- GetExtraSpecsProperty reads only the selected flavor/property member route with an empty body and no query, flavor GET, list or whole-specs GET. The concrete ID/native Flavor/RawResource input and selected source are captured before option callbacks; inline specs never skip this read, explicit ID determines the route, and the supplied existing extra_specs remain unchanged.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecsPropertyRawValuesLiteralRouteAndVersionBinding`, `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorFetchExtraSpecsAlwaysGETRawValuesAndOwnedExistingInputs`.
- The selected Value remains exact owned JSON rather than a string/dict conversion: string, precise large number, array, object and explicit null are retained. Missing properties return Value null and Present false, while explicit null is Present true. Wire contains only the actual flat response, and Value/Envelope/Header do not alias Wire. Actual final200/201/202/203/300 payloads retain their body/header/status receipt.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecsPropertyRawValuesLiteralRouteAndVersionBinding`.
- The property uses the same concrete read-options/source/version engine as FetchExtraSpecs: selected2.100 is retained, an unselected client performs one finite discovery and chooses2.61, and explicit empty is versionless without changing the caller client. Options execute once before discovery/member GET; input identity is already snapshotted when a callback changes caller models. Empty/dot/control property selections fail before callbacks and HTTP.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecsPropertyRawValuesLiteralRouteAndVersionBinding`, `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecsPropertyPreflightAndBindingTerminalReceipts`, `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorFetchExtraSpecsPreflightAndAcceptedTerminalReceipts`.
- Actual HTTP404 stays a strict native failure with original method/URL/body/header and no missing normalization. Accepted read/Close causes, source drift and nonobject response roots retain the actual partial receipt with nil Value/Wire and a ResponseError; nested404 cannot become ignored missing. Accepted processing errors make no SDK resend or native RetryFunc callback, while native provider handling of an actual HTTP failure is preserved.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecsPropertyPreflightAndBindingTerminalReceipts`, `api/compute_flavor_extra_specs_fetch_test.go:TestComputeFlavorFetchExtraSpecsPreflightAndAcceptedTerminalReceipts`.

Python/Go differences:

- Python accepts an ID or connected mutable Flavor; Go uses the concrete FlavorExtraSpecsRequest input and independent record, snapshots existing values, and exposes Present to distinguish missing from explicit null. The scalar selection does not update extra_specs or synthesize a Flavor Resource view.
- Python's cast(str | None, response.json().get(prop)) performs no runtime conversion. Go likewise retains arbitrary actual member JSON in Value, including exact large integers, rather than using native GetExtraSpec's string map.
- Go validates a nonempty valid property without controls/dot segments and escapes one literal path component, including slash/percent/space/reserved characters. Python utils.urljoin strips slash components; Go also requires a safe string route identity instead of allowing Python's broader dynamic identity values.
- Owned member reads accept final200..399 and require a valid UTF-8 nonnull JSON object; accepted failures include physical receipt and cause. WithFlavorExtraSpecsOptions/Microversion/Header are Go extensions; source's public named getter has no query/filter kwargs. Selected or finite2.61/versionless policy is shared with the separately reviewed FetchExtraSpecs.
- Full generic mutable Resource/session/cache and other flavor list/find declarations remain separate SDK goal work rather than additional required branches of this fixed single-property getter.

### `gophercloud:compute/v2/flavors.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.Get`

- Get binds one bodyless queryless native GET to the selected ResourceBase flavors/{id} member using the live token/source headers and both selected Nova version headers. Native Flavor fields, unknown-field omission, numeric/string/blank swap conversion, missing public/specs zero defaults and nullable fields are preserved. Missing/null flavor wrappers return nil,nil; known-field and swap decode failures retain the native allocated/partially assigned Flavor together with their typed/numeric cause.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetNativeBindingDTOAndOriginalHTTPErrors`.
- Only native200 succeeds. Actual203/204/403 remain native failures with nil result, expected200, original method/URL/body/header and selected client unchanged; no list or specs lookup is introduced.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetNativeBindingDTOAndOriginalHTTPErrors`.

Python/Go differences:

- Go owns the exported concrete facade and wraps upstream errors while preserving native Get/Extract behavior and partial values. This native method does not add Cloud GetFlavorByID safe-result validation, ignored missing, Find fallback or optional specs enrichment.
- The returned Flavor is the native typed DTO; it does not synthesize Python descriptor defaults, unknown/location fields or connected mutable Resource state.

### `gophercloud:compute/v2/flavors.GetExtraSpec`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.GetExtraSpec`

- GetExtraSpec makes one bodyless queryless native GET to flavors/{id}/os-extra_specs/{key}, retaining the selected token and both Nova version headers. Its result is the complete flat map[string]string, including unrelated string keys and a null member as an empty string; a missing requested property does not reduce the returned map. Null root retains nil map and empty object retains nonnil empty map.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecNativeBindingReturnsWholeStringMap`.
- Numeric members retain the native partial map plus UnmarshalTypeError and an array root returns the native type failure. Only200 succeeds; actual203/204/404 preserve nil result and the original expected-code/method/URL/body/header proof without flavor lookup or ignored missing.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorGetExtraSpecNativeBindingReturnsWholeStringMap`.

Python/Go differences:

- The library-owned facade preserves native route/result/status ABI and wraps errors; this whole string map is deliberately distinct from the owned raw scalar GetExtraSpecsProperty API.
- Native null string values become empty strings and nonstring JSON values produce native partial decode errors. The caller must check error even when a map is returned.

### `gophercloud:compute/v2/flavors.ListDetail`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.ListDetail`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithListDetailOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithListDetailQuery`

- ListDetail owns a concrete request.Options builder while preserving the fixed flavors/detail native route. Default typed options omit is_public and all other query values. All eight native ListOpts fields reach their original wire spelling, raw limit/is_public extensions override typed values, and vendor query is retained. Options execute once at sequence construction while HTTP remains lazy; bodyless requests use the selected token and both version headers.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorListDetailNativeBindingEagerOptionsLazyLinkedPages`.
- Native Flavor/Swap/inline-specs DTOs and flavors_links continuation are preserved across two pages. Native200 empty and204 return no rows,300 page is accepted, and consumer break never fetches the next page. An empty native page stops even when it advertises a next link; no Python summary route, is_public default or item enrichment is invented.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorListDetailNativeBindingEagerOptionsLazyLinkedPages`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsOrdinaryCollectionsRetainQueryWithoutIdentityPolicies`.
- Late403 retains already yielded rows and yields one terminal error with original method/URL/body/header and expected200/204/300. First203, malformed JSON and typed malformed Flavor rows yield one terminal native error. A failing option produces no HTTP and retains the original callback cause.
  Tests: `api/compute_flavor_property_native_test.go:TestComputeFlavorListDetailNativeBindingEagerOptionsLazyLinkedPages`.

Python/Go differences:

- Go exposes iter.Seq2[*Flavor,error] over the upstream native pager rather than returning a Pager and requiring caller extraction. Native FlavorPage/status/query behavior is preserved.
- Native fixed detailed-list paging does not add Python flavors(details=False), per-item get_extra_specs, semantic alias/Body filters, HTTP Link or marker fallback. Those separate public Python list/find declarations retain their own completion work.
- PrivateAccess/AllAccess behavior remains server-policy dependent; this mapping does not imply those requests are authorized for every core-user deployment.

### `python:cloud/_compute/ComputeCloudMixin/get_flavor_by_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetFlavorByID`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorByIDOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorByIDExtraSpecs`

- GetFlavorByID owns Cloud's default-false get_extra flag and strict GET-only behavior: one native flavor GET, no query/body/list/automatic discovery, and400/403/404 stay original HTTP errors rather than name fallback or nil success. It returns canonical native flavor fields and keeps the selected client2.55/live token.
  Tests: `api/compute_flavor_by_id_test.go:TestComputeCloudGetFlavorByIDOwnsStrictDefaultsAndNativeEnrichment`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsGETAndExtraSpecsPresence`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsMissingPoliciesAndUnsafeNames`.
- Omitted/false enrichment makes no specs request. True fetches canonical returned ID once for missing/null/empty inline specs and keeps a nonempty inline map without another GET. Unrelated typed flavor fields are retained, caller query never leaks to specs, and actual specs404 is terminal. The existing native enrichment engine also preserves its missing/null-map and typed/transport/read/cancel error policy.
  Tests: `api/compute_flavor_by_id_test.go:TestComputeCloudGetFlavorByIDOwnsStrictDefaultsAndNativeEnrichment`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsGETAndExtraSpecsPresence`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsEnrichmentFailuresAreTerminal`.
- Typed FlavorByID options are reusable captured values and execute once per call. Nil service/context/option, unsafe ID and unsupported header/query/field/argument configurations fail before HTTP; source drift or option/cancellation error stops later callbacks and retains the original custom option/context causes.
  Tests: `api/compute_flavor_by_id_test.go:TestComputeCloudGetFlavorByIDTypedOptionsAndPreflight`.
- The Cloud entry delegates to the already reviewed strict FindFallbackNever/IgnoreMissing(false) proxy-get mapping and checks the captured service/client/component binding around options and completion. The shared GET/enrichment implementation preserves live provider/version/context and rejects unsafe or null successful identity models before returning a Cloud result.
  Tests: `api/compute_flavor_by_id_test.go:TestComputeCloudGetFlavorByIDOwnsStrictDefaultsAndNativeEnrichment`, `api/compute_flavor_by_id_test.go:TestComputeCloudGetFlavorByIDTypedOptionsAndPreflight`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsKeepsLiveProviderVersionAndContext`, `api/find_identity_flavors_test.go:TestNativeFindIdentityFlavorsPreflightAndNativeGETFailures`.

Python/Go differences:

- The pinned Cloud declaration delegates get_extra=False to compute.get_flavor and introduces no independent query/filter/list behavior. Go's concrete bool options own the strict Never/IgnoreMissing(false) combination instead of requiring callers to assemble it.
- Go returns the native Flavor DTO and requires a safe explicit input/returned ID. Native missing/null ExtraSpecs stays nil and missing IsPublic stays false, with no seeded mutable Python Resource identity/location/default reconstruction.
- The existing native GET/specs calls retain the configured microversion and native strict200 policy without automatic source Resource version discovery. Cloud binding checks do not replace the native provider's physical retry/reauth machine with owned MemberGet; the single-property API has its separate owned transport.
- Other Cloud/proxy flavor list/find/search and generic Resource/session/cache APIs remain in the original SDK goal independently of this ID-only alias.

### `python:compute/v2/flavors`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.ListFlavors`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListDetails`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListLimit`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListMarker`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListHeader`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListQuery`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListFilter`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithFlavorListFilters`

- The owned flavor iterator defaults to /flavors/detail with Details=true and GetExtraSpecs=false; Details(false) selects /flavors. Semantic classification supports seven server-query attributes, min_disk/min_ram aliases, twelve local Body attributes and ignored unknown list attributes. Absent or semantic-null is_public sends literal None, while empty string and false remain distinct; canonical query spelling wins aliases in one bulk map, and conflicting semantic versus typed/raw query channels fail before HTTP.
  Tests: `api/compute_flavor_records_test.go:TestComputeFlavorRecordsQueryProjectionAndOwnedViews`, `resource/filters_test.go:TestSemanticFilterSelectionKeepsMemberSpellingSeedAndDeferredUnknownErrors`, `resource/filters_test.go:TestSemanticFiltersCanonicalPrecedencePresenceAndExactQueryEncoding`, `resource/filters_test.go:TestSemanticFiltersNamespaceCollisionsAndIndependentClear`.
- Flat list rows project the thirteen declared Body values plus location into an owned view: raw id/name/original_name truthy fallback, omission-only name alias, absent defaults, explicit null, integer/boolean/float/dictionary response conversions and parsed response alias order. Nested flavor and vendor values remain actual Wire fields. Wire, row envelope, page headers and status stay independent; Service.ListFlavors captures current location once before options and adds it only to view copies.
  Tests: `api/compute_flavor_records_test.go:TestComputeFlavorRecordsQueryProjectionAndOwnedViews`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordDeclaredDefaultsDescriptorsAndRawIdentity`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordResponseAliasOrderAndIndependentReceipts`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordDescriptorErrorsAndPassiveStringIdentity`, `internal/jsonfilter/descriptor_float_test.go:TestDescriptorFloatJSONSourceConversionsAndOwnership`, `internal/jsonfilter/descriptor_float_test.go:TestDescriptorFloatJSONRejectsInvalidStringsAndNonfiniteWithCauses`, `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`.
- Each iterator consumption prepares options lazily once and follows rel/href links, flavors_links, top-level next and HTTP Link. Initial effective positive limits allow last-consumed-raw-row logical markers even for filtered rows and caller-mutated views; the first advertised server limit is fixed without creating an initial-limit fallback. Raw row caps precede filters and avoid decoding unconsumed bad rows; first-page, empty-page and break stop later reads, while consumed malformed rows and late HTTP errors remain observable with actual receipts.
  Tests: `api/compute_flavor_records_test.go:TestComputeFlavorRecordsPagingRawCapsAndLazyOwnership`, `internal/rest/list_test.go:TestListFirstServerLimitIsOptInAndFixedAcrossContinuations`, `internal/rest/list_test.go:TestListHTTPLinksAreOptInAndConflictingNextLinksFail`, `internal/rest/list_test.go:TestListContinuationCannotWidenCollectionFiltersOrMarkers`.
- GetExtraSpecs(true) enriches only yielded, locally matched rows whose projected dictionary is falsey. Nonempty inline dictionaries skip GET; missing, null, empty or normalized nonobject dictionaries fetch through the existing prepared extra-specs reader. Filtering, cap and break precede enrichment. The child has independent actual response evidence and Resource copies, uses the logical string identity without list query, and actual HTTP404 or accepted Close404 remains terminal with the original row retained.
  Tests: `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.
- Owned list/version policy preserves a selected client version, discovers at most 2.61 only when unselected, and supports explicit empty versionless override. One prepared policy and option application is shared with per-row enrichment; the original client is unchanged and live provider tokens remain in use. Model conversion, accepted-body failures, source/context failures and invalid controls retain their causes rather than fabricating successful rows or SDK resends.
  Tests: `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsQueryProjectionAndOwnedViews`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsPagingRawCapsAndLazyOwnership`, `compute/v2/flavors/flavor_record_test.go:TestFlavorRecordDescriptorErrorsAndPassiveStringIdentity`, `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`.

Python/Go differences:

- Service.ListFlavors supplies Connection location; direct leaf API.ListRecords has location=null. FlavorRecord separates the computed owned response view from actual Wire, row envelope, page metadata and optional child receipt rather than exposing a mutable Python Resource/session.
- Go query booleans use lowercase true/false and the shared query encoder supports scalar/null/scalar-array values. The common local matcher preserves JSON type distinctions and scalar-versus-object subset policy instead of Python bool-number equivalence, empty-dict truthiness and nested scalar attribute failures.
- Typed Limit/MaxItems zero means omission/unlimited and negatives are rejected; an effective wire limit is one positive integer and a marker is one nonblank string. Semantic and typed/raw wire query collisions are errors even for equal values; raw wire query can replace the matching typed limit or marker independently. Arbitrary session/path/resource_type, cache-bypass and deprecated Resource/JMESPath controls remain separate SDK goal scope.
- Owned list reads accept 200..399 but require complete nonnull UTF-8 JSON objects and flavors arrays or singleton objects. Accepted empty204 is an error rather than native empty-page compatibility. Invalid representations and descriptors retain actual response receipts; nonfinite float/overflow fails the finite JSON view, with audited IEEE-754/Unicode 16 conversions and independent actual raw values.
- Dictionary next links are an explicit Go compatibility extension; safe continuation checks prevent origin/route/filter widening and cycles while preserving reverse proxy prefixes. Logical markers and executable specs targets require string identities; numeric/container identities remain raw descriptor values and are never stringified into request paths.
- The named flavor list/conditional-enrichment workflow is mapped; global mutable Resource, session/cache/discovery lifecycle and unrelated native/Cloud flavor operations are not claimed complete by this row.

### `python:cloud/_compute/ComputeCloudMixin/list_flavors`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.AllFlavors`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryHeader`

- AllFlavors eagerly consumes the existing detailed owned Flavor list in source page order, retaining duplicates and producing a completed JSON array, including [] for an empty inventory. Its actual source get_extra default is false; an explicit true enriches falsey inline specs while false makes no child request. It sends the fixed detail list query, never Cloud filters or a name selector, and retains the chosen version and live token through all pages and child reads.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsQueryProjectionAndOwnedViews`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.
- The eager collection reuses the owned reader's guarded paging and conditional enrichment. A late page403 retains prior Inventory but cannot publish completed Value or Flavors. Requested child failure follows the same shared collection path: the Cloud Search composition proves the failing known row is retained, actual child404 is terminal, and an accepted child Close404 keeps the original physical cause plus separate row/child receipt without a completed collection.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsPagingRawCapsAndLazyOwnership`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.
- The shared Cloud read preparation applies captured typed/raw/header options once, checks the original service/client binding and preserves cancellation causes. AllFlavors rejects any present Filters pointer, including null and {}, before HTTP because its source declaration has no filter argument; nil explicitly clears an earlier filter. The reused inventory stage captures Connection location before callbacks and independently adds it to each projected Resource and child Resource while leaving actual Wire and receipts unchanged.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence`, `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorSelectionOwnedOptionsAndLocation`, `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorGetFindBindingAndFilterPreflight`.

Python/Go differences:

- The pinned source signature and body use get_extra=False and read no clouds.yaml override, despite the list_flavors docstring saying default True. Go follows the executed signature and exposes a concrete optional bool override.
- Python returns an eager list of connected mutable Flavor Resources. Go returns completed raw JSON Value/Flavors plus partial Inventory on failure; records retain an owned projected Resource, actual Wire/row receipt and separate child enrichment receipt. The owned response view can be modified by its caller; it is not an immutable Python Resource replacement.
- This Cloud composition reuses the separately mapped owned reader's documented JSON-domain descriptors, finite selected/discovered or explicit-versionless policy, route/control/continuation validation and Connection location snapshot. Generic mutable Resource/session/cache facilities and other native/Proxy declarations remain separate parts of the original SDK goal.
- AllFlavors names the eager Cloud operation without replacing Service.ListFlavors, which remains the lazy Proxy-list mapping. The shared options carrier does not add a filters parameter to this source operation: present filters are a Go preflight error and nil clears them.

### `python:cloud/_compute/ComputeCloudMixin/search_flavors`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.SearchFlavors`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryFilters`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryExpression`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryHeader`

- SearchFlavors defaults requested extra specs to true and completes the entire detailed inventory and conditional child reads before applying any identifier or Cloud filter. Explicit false skips enrichment. Even an ultimately unmatched row is enriched when required, and a dictionary can select newly fetched specs. No name or Cloud dictionary is forwarded into the early Proxy query/Body matcher. Empty successful inventory/selection is [] rather than an inventory error.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.
- After successful inventory completion, the existing Cloud selector applies per-row Python-shaped id/name exact-or-glob matching, preserving source order and duplicate matches, then ordered dictionary subset or JMESPath selection. Numeric logical IDs participate in Cloud string matching. Reached unknown predicates fail, predicates after an earlier mismatch remain unconsumed, and an empty identifier selection does not consume unknown dictionary keys. Expressions retain arbitrary scalar/object/array/false JSON Value and do not invent associated Flavor records.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorSelectionOwnedOptionsAndLocation`, `internal/cloudfilter/identifier_test.go:TestCloudFilterIdentifierPreservesPerRowExactGlobMatchesDuplicatesAndSourceOrder`, `internal/cloudfilter/identifier_test.go:TestCloudFilterIdentifiersUsePythonJSONDomainStringAndContainerRepresentation`, `internal/cloudfilter/mapping_test.go:TestCloudFilterOrderedMappingConsumesOnlyReachedKeysAndKeepsDuplicateFirstPosition`, `internal/cloudfilter/mapping_test.go:TestCloudFilterNestedEmptyObjectsUseActualTruthinessBeforeShapeOrMissingKeys`, `internal/cloudfilter/mapping_test.go:TestCloudFilterLeafEqualityKeepsExactArrayShapeAndPythonBooleanNumberPrecision`, `internal/cloudfilter/results_test.go:TestCloudFilterExpressionBridgeReturnsArbitraryOwnedJSONWithoutInventedRowIndices`.
- Any late inventory, requested child or local filter failure retains consumed Inventory without completed Value/Flavors. Accepted child failures preserve independent raw row/child receipts and their original read/Close causes. Reusable bulk pointer/raw filters and callback header carriers are owned, callbacks execute once, captured location is applied independently before local selection, and actual row Wire/Envelope is unchanged. Source mutation or callback cancellation stops later callbacks/HTTP and retains the custom cancellation cause; selection publication remains guarded.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorEagerInventoryDefaultsAndPartialEvidence`, `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorSelectionOwnedOptionsAndLocation`, `internal/cloudfilter/results_test.go:TestCloudFilterSelectOwnsViewsFiltersIndicesAndResultsBeforeGuardCallbacks`, `internal/cloudfilter/results_test.go:TestCloudFilterGuardInterruptsTraversalAndRetainsCancellationCauseWithoutPartialSelection`.

Python/Go differences:

- Python annotates a list result, but the executed string-filter branch can return arbitrary JMESPath values. Go exposes that raw JSON in Value and leaves Flavors unset for expression results; ordinary selector indices connect Flavors to the retained Inventory without synthesizing expression records.
- Dictionary filters are local Cloud predicates after complete enriched inventory, not early owned ListRecords query/Body filters. The shared JSON-domain Cloud selector represents Python truthiness, repr/glob, ordered predicate consumption and equality through finite documented Go rules rather than arbitrary Python object execution.
- Search uses the separately mapped reader's owned projected Resource plus captured Connection location, and keeps actual Wire/Envelope/Header/Status and child receipt separate. Partial Inventory is an additive diagnostic result on error. Selected/finite-discovered/explicit-empty version and typed shared-control differences remain documented in the owned guide; generic mutable Resource/session/cache lifecycle remains an independent SDK goal.

### `python:cloud/_compute/ComputeCloudMixin/get_flavor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetFlavor`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryExtraSpecs`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryFilters`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFlavorQueryHeader`

- GetFlavor delegates directly to the owned FindFlavor reader with IgnoreMissing(true) and requested extra specs true by default; it does not call Cloud Search or First. Absent, explicit null and every source-falsey filter become an unfiltered Find. Truthy dictionaries retain original member query spelling and known seed attributes. Direct success keeps inline nonempty specs, or enriches the canonical returned ID when requested specs are falsey; explicit false skips the child and an explicit empty microversion remains versionless for both reads without discovery.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorGetFindBindingAndFilterPreflight`, `api/compute_flavor_find_test.go:TestComputeFindFlavorMemberSeedAliasesAndLiteralRoutes`.
- The direct Find delegation keeps its source-shaped clean400/403/404 detail-list fallback. Missing returns nil,nil by default, a second match immediately returns ambiguity without child enrichment, and only a unique fallback result is enriched. Captured location is applied to separate result/child Resources, and member and child status/header/envelope remain distinct. The existing reader preserves late fallback and accepted child failures as terminal partial evidence rather than inventing a successful Get.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorGetFindBindingAndFilterPreflight`, `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`, `api/compute_flavor_records_test.go:TestComputeFlavorRecordsConditionalExtraSpecsEnrichment`.
- Truthy nonobject filters fail before HTTP instead of becoming Cloud expressions. Dictionary keys name_or_id, ignore_missing and get_extra_specs collide with explicit find arguments and are rejected before HTTP; the reused reader protects its explicit id argument and scalar original-query encoding. The shared captured read configuration and final binding check preserve reusable options, selected/live client state and guarded context/location preparation without a second application of Cloud callbacks.
  Tests: `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorGetFindBindingAndFilterPreflight`, `api/compute_flavor_cloud_test.go:TestComputeCloudFlavorSelectionOwnedOptionsAndLocation`, `api/compute_flavor_find_test.go:TestComputeFindFlavorMemberSeedAliasesAndLiteralRoutes`, `api/compute_flavor_find_test.go:TestComputeFindFlavorFallbackTerminalErrorsAndConnectedLocation`.

Python/Go differences:

- The source's deprecated filter docstring mentions strings/JMESPath, but its executed body normalizes falsey filters and expands truthy filters as Find kwargs. Go follows that direct delegation: truthy nonobjects and duplicate explicit kwargs are typed HTTP0 errors; it does not turn them into Cloud search. Python's deprecation warning is documented rather than emitted as a Go warning.
- Original member kwargs reuse the separately mapped Find encoder's scalar/null/scalar-array range. Dictionary-valued and nested-array values that Python may stringify or iterate are explicit Go input errors, distinct from SearchFlavors' nested dictionary predicates. The owned Find reader also retains its documented literal route validation, identity/filter/projection, version and continuation differences.
- Go returns an owned FlavorRecord and can retain a known partial record/child receipt with error, rather than returning a connected mutable Python Resource or silently losing physical evidence. Computed current location changes only Resource views, with Wire/Envelope/Header/Status kept actual. Generic mutable Resource/session/cache facilities and other native/Proxy/Cloud declarations remain separate portions of the original SDK goal.

### `python:cloud/_compute/ComputeCloudMixin/list_availability_zone_names`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.ListAvailabilityZoneNames`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAvailabilityZoneNamesOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithUnavailableZones`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAvailabilityZoneNamesMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAvailabilityZoneNamesHeader`

- The Cloud name method always consumes the ordinary os-availability-zone inventory. Unavailable defaults false; true includes unavailable rows without requesting the admin detail route. Name values remain passive JSON, including numeric/bool/object/array/null and omitted-as-null, preserving source order and duplicates; a normal empty inventory completes Names and Value as an empty array.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNamesTruthinessSuppressionAndPartialInventory`.
- The state available key is read before the unavailable option is considered. JSON-domain Python truthiness controls ordinary inclusion; missing/null/scalar/array state and a missing available key remain errors even with unavailable true, returning known Inventory without fabricated completed Names or Value.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNamesTruthinessSuppressionAndPartialInventory`.
- A clean actual HTTP 403/500, late actual404, or isolated repeated pagination marker cycle produces the source-compatible completed empty array, discards the collected name prefix, retains Inventory diagnostics, and exposes the original status/body/header/URL or cycle through SuppressedError. Duplicate re-return of the same original retry error does not invent another independent terminal cause.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNamesTruthinessSuppressionAndPartialInventory`.
- Accepted malformed JSON and accepted Close failures with a nested native404 remain visible response errors with actual200 receipt and no accepted replay. A different retry callback cause beside actual HTTP500 retains both causes, source failure beside HTTP500 and cancellation with custom cause beside HTTP403 remain visible, and an option callback returning a native404 does not become a suppressed fetched SDK failure or completed names.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNamesTruthinessSuppressionAndPartialInventory`.
- Concrete typed option/version/header inputs are owned, full replacement and individual helpers apply at most once, and current location is captured before option callbacks once per logical call. Header/version overrides reach the fixed GET while the original selected native client and actual wire location remain unchanged; response Names/Value/Resource/Envelope own independent bytes.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNamesTruthinessSuppressionAndPartialInventory`, `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneRecordsOrdinaryViewsLazyPagingAndOwnership`.

Python/Go differences:

- Pinned _compute.py:255-276 catches openstack.exceptions.SDKException, not arbitrary Python failures. Go explicitly classifies a clean actual native HTTP4xx/5xx rejection and an isolated pagination cycle; accepted IO/decode/source/context/URL-transport/local safety and different joined error causes remain visible. SuppressedError and retained Inventory are additive diagnostic results; Source returns only [].
- The Python list[str] annotation does not coerce the untyped zone name descriptor. Names uses []json.RawMessage and Value uses an owned JSON array to preserve the actual JSON values instead of synthesizing strings. Malformed state/key errors use Go errors with a known partial result rather than Python TypeError/KeyError classes.
- Public Source takes only unavailable=False. Go adds concrete header and microversion options; blank selected source performs no version discovery, nil preserves selection, explicit empty suppresses version headers for this call, and no detail/query/body controls are introduced.
- Service supplies an owned snapshot of existing Connection location to declared Resource only; standalone service without that dependency uses null. Wire/Envelope remain the actual row and header/status evidence is additive. This finite name-list mapping does not implement a general mutable Resource/cache/Session surface.
- The shared guarded pager accepts compatible advertised continuation but requires safe fixed collection scope and consistent continuation candidates. Python chooses a precedence channel instead; dictionary links.next is additional Go compatibility. Completing this Cloud row does not close Proxy details=True or native ListDetail.

### `gophercloud:compute/v2/availabilityzones.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/availabilityzones.API.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.SinglePageStream`

- Native ordinary List remains lazy and uses the original fixed os-availability-zone GET with selected client/version/token and native AvailabilityZone DTO. One physical page decodes the complete availabilityZoneInfo array before any row is published; embedded next links do not cause another request or a detail lookup.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNativeListTypedSinglePageABI`.
- Actual200 and300 object envelopes preserve ZoneName string, ZoneState.Available bool, Hosts and nested ServiceState booleans plus the native millisecond no-zone timestamp decoder. Missing/null DTO fields retain native zero values, and null or missing plural data yields no rows.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNativeListTypedSinglePageABI`.
- Native success codes remain200/204/300: empty application/json204 exposes EOF, unexpected201 and403 retain original method/URL/body/header/status evidence, and syntax or a wrong known field type fails without publishing an earlier row from the same page.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNativeListTypedSinglePageABI`.

Python/Go differences:

- Pinned v2.15.0 AvailabilityZonePage inherits array-only SinglePageBase.IsEmpty, so native EachPage cannot handle the normal object envelope. The audited SDK generator selects SinglePageStream only for ordinary AvailabilityZones.List; it uses native Pager.AllPages, whose SinglePageBase shortcut bypasses that emptiness check, then the unchanged native extractor. ListDetail keeps its original generated path and is not promoted.
- The SDK exposes context-aware iter.Seq2 rather than returning pagination.Pager. It retains original AllPages single-page semantics and complete native decode, with local cancellation and consumer break ending publication; native DTO fields are separate from the new untyped owned Resource/Wire view.
- Native pagination accepts200/204/300 but does not define a successful empty JSON representation for204. Native PageResultFrom/ExtractInto/null/default behavior and configured prebody provider policies remain native; the SDK does not retrofit owned accepted-response IO evidence or Python descriptor values into this native row.

### `python:compute/v2/availability_zones`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/availabilityzones.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/availabilityzones.API.ListDetailRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.ListAvailabilityZones`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/availabilityzones.WithAvailabilityZoneListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/availabilityzones.WithAvailabilityZoneMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/availabilityzones.WithAvailabilityZoneHeader`

- The details=False branch is available through ListRecords and Service.ListAvailabilityZones at the ordinary fixed path. id/name/state/hosts are passive raw descriptors with missing or null represented by JSON null; zoneName/name and zoneState/state aliases follow original dictionary order, unknown fields remain Wire-only, and a name never fabricates inherited id.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneRecordsOrdinaryViewsLazyPagingAndOwnership`, `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNamesTruthinessSuppressionAndPartialInventory`, `compute/v2/availabilityzones/python_detail_records_test.go:TestPythonZoneDetailRecordsLeaveOrdinaryRoute`.
- A logical iteration owns options lazily, captures source and Service binding before callbacks, snapshots current location once before the callbacks, and keeps location confined to independent Resource views. Actual row Envelope/Wire and header/status remain independent. Selected version is retained, explicit empty is versionless, and source mutation stops later callbacks before HTTP.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneRecordsOrdinaryViewsLazyPagingAndOwnership`, `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneNamesTruthinessSuppressionAndPartialInventory`.
- Owned ordinary rows accept the Source-facing 200..399 success range through the shared bodyless guarded reader, preserving reverse endpoint prefixes and actual metadata. Advertised linked pages retain duplicate records, an empty raw page stops before its next link, consumer break stops continuation, and late HTTP403 is visible with its original receipt after prior rows.
  Tests: `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneRecordsOrdinaryViewsLazyPagingAndOwnership`.
- The actual collection implementation composes the existing optional page reader, singleton-object representation and decode-no-content policies, plus an opt-in first advertised server limit that remains fixed without enabling initial-limit marker fallback. This shared behavior is separate from native SinglePageBase DTO listing.
  Tests: `internal/rest/list_test.go:TestListOptionalReaderAndRepresentationsKeepDefaultPolicy`, `internal/rest/list_test.go:TestListFirstServerLimitIsOptInAndFixedAcrossContinuations`, `api/compute_availability_zones_test.go:TestComputeAvailabilityZoneRecordsOrdinaryViewsLazyPagingAndOwnership`.
- The details=True branch is ListDetailRecords: it sends a bodyless, queryless GET os-availability-zone/detail with the client microversion in X-OpenStack-Nova-API-Version and OpenStack-API-Version, no version headers for a blank client or an explicit empty WithAvailabilityZoneMicroversion, and caller headers from WithAvailabilityZoneHeader.
  Tests: `compute/v2/availabilityzones/python_detail_records_test.go:TestPythonZoneDetailRecordsUseDetailRouteAndKeepHosts`.
- Detail rows use the same AvailabilityZoneRecord projection as ListRecords: zoneName/zoneState become name/state, hosts with per-service active/available/updated_at state stay raw JSON, id and location are null, unknown vendor fields remain only in Wire, and Envelope/Header/StatusCode keep the actual row and page receipt.
  Tests: `compute/v2/availabilityzones/python_detail_records_test.go:TestPythonZoneDetailRecordsUseDetailRouteAndKeepHosts`.
- ListDetailRecords follows availabilityZoneInfo_links continuations on the detail path and rejects a continuation that leaves os-availability-zone/detail before HTTP, accepts 200..399 including a singleton object page, and reports 403/404 rejections as resource.OperationError{ListDetailRecords, availabilityzones} wrapping the native status error and body.
  Tests: `compute/v2/availabilityzones/python_detail_records_test.go:TestPythonZoneDetailRecordsFollowDetailLinks`, `compute/v2/availabilityzones/python_detail_records_test.go:TestPythonZoneDetailRecordsStatuses`.

Python/Go differences:

- Python selects the branch with availability_zones(details=False|True); Go exposes ListRecords for details=False and ListDetailRecords for details=True.
- The pinned public signature is availability_zones(details=False), with no public **query. Go offers only concrete headers/microversion options for both branches; it does not invent reachable Source body filters, limit/marker/max-items arguments, arbitrary base_path, or supplied Resource parameters.
- AvailabilityZone declares id/name/state/hosts without typed conversion, with default None and no alternate ID. Resource is the declared owned view; Wire is actual JSON including unknown/vendor fields, Envelope is the actual row, and physical Header/StatusCode are additive Go evidence. Python returns hosts as a dict while Go keeps raw JSON. Service maps existing Connection current_location while standalone leaf location is null.
- Blank source version causes no discovery because the Source class has no _max_microversion. Explicit per-call empty is a Go versionless override; the original native client and live authentication remain unchanged under the existing guarded read policy.
- Go requires valid UTF8 object/page representations, supports one object in the plural envelope, and decodes an accepted204 representation rather than treating an empty204 as a successful empty list. Shared reader errors remain visible; Python JSON/shape exception class spelling is not reproduced.
- Python selects body links, then top-level next, then HTTP Link according to source precedence. Go validates all advertised candidates consistently, restricts origin/prefix/route/query changes (detail continuations must stay on os-availability-zone/detail), and supports dictionary links.next as extra compatibility. An advertised first positive limit may be adopted but does not create the absent initial limit or enable marker fallback.
- Owned records do not provide generic mutable Resource/session/cache behavior; the generated native ListDetail remains a separate typed Gophercloud listing.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
server = conn.compute.find_server("web-01", ignore_missing=False)
for server in conn.compute.servers(status="ACTIVE"):
    print(server.id, server.name)
```

```go
service, err := conn.Compute(ctx)
if err != nil { return err }
server, err := service.Servers.Find(ctx, resource.Name("web-01"))
if err != nil { return err }
fmt.Println(server.ID)

for server, err := range service.Servers.List(ctx, resource.WithStatus("ACTIVE")) {
    if err != nil { return err }
    fmt.Println(server.ID, server.Name)
}
```

```go
server, err := service.Servers.Create(ctx, compute.CreateServerRequest{
    Name: "web-01",
    Image: resource.Name("ubuntu"),
    Flavor: resource.Name("small"),
}, compute.WithNetworks(resource.Name("private")),
   compute.WithKeyName("my-key"),
   compute.WithConfigDrive(false),
   compute.WithWait(resource.WithTimeout(5*time.Minute)))
if err != nil {
    if server != nil { fmt.Println("created:", server.ID) }
    return err
}
```

```python
server = conn.create_server(
    name="volume-server",
    flavor="small",
    boot_volume="root-volume",
    terminate_volume=False,
    network="private",
    auto_ip=False,
    wait=True,
)
```

```go
server, err := service.Servers.Create(ctx, compute.CreateServerRequest{
    Name: "volume-server",
    Flavor: resource.Name("small"),
}, compute.WithBootVolume(resource.Name("root-volume")),
   compute.WithNetworks(resource.Name("private")),
   compute.WithWait(resource.WithTimeout(5*time.Minute)))
if err != nil {
    if server != nil { fmt.Println("created:", server.ID) }
    return err
}
```

```python
server = conn.create_server(
    name="new-volume-server",
    image="ubuntu",
    flavor="small",
    boot_from_volume=True,
    volume_size=50,
    terminate_volume=False,
    auto_ip=False,
    wait=True,
)
```

```go
server, err := service.Servers.Create(ctx, compute.CreateServerRequest{
    Name: "new-volume-server",
    Image: resource.Name("ubuntu"),
    Flavor: resource.Name("small"),
}, compute.WithBootVolumeSize(50),
   compute.WithWait(resource.WithTimeout(5*time.Minute)))
if err != nil {
    if server != nil { fmt.Println("created:", server.ID) }
    return err
}
```

```python
server = conn.create_server(
    name="grouped",
    image="ubuntu",
    flavor="small",
    group="anti-affinity",
    scheduler_hints={"different_host": ["other-id"]},
    volumes=["data"],
    userdata="#cloud-config\n",
    meta={"role": "web"},
)
```

```go
server, err := service.Servers.Create(ctx, compute.CreateServerRequest{
    Name: "grouped",
    Image: resource.Name("ubuntu"),
    Flavor: resource.Name("small"),
}, compute.WithServerGroup(resource.Name("anti-affinity")),
   compute.WithSchedulerHints(map[string]any{"different_host": []string{"other-id"}}),
   compute.WithDataVolumes(resource.Name("data")),
   compute.WithUserData([]byte("#cloud-config\n")),
   compute.WithMetadata(map[string]string{"role": "web"}))
```

Create property rules:

- Scheduler hints are sent in the top-level `OS-SCH-HNT:scheduler_hints`
  object, as Python's Server resource does. `WithServerGroup` always sets the
  `group` hint, replacing a `group` key from `WithSchedulerHints`. Group names
  are resolved by listing `os-server-groups` before the POST; IDs skip lookup.
- `WithDataVolumes` adds `block_device_mapping_v2` entries with
  `boot_index` -1 and `delete_on_termination` false. When booting from an
  image, a local image mapping with `boot_index` 0 and
  `delete_on_termination` true is added first. Python sends some boot indexes
  as strings; Go sends integers.
- `WithUserData` always base64-encodes the supplied bytes, even when they
  already decode as base64. `WithServerCount` sends `min_count`/`max_count`;
  without it both keys are omitted, whereas Python's cloud layer sends 1/1.
- `WithDescription` (2.19), `WithTags` (2.52), `WithHypervisorHostname` (2.74)
  and `WithHostname` (2.90) are checked against the client microversion before
  any request. `WithPersonality` is rejected locally at 2.57 and later.
- `WithBlockDeviceMappings` sends explicit `block_device_mapping_v2` entries
  with every per-mapping field (`BootIndex` may be `request.Null[int]()`);
  generated boot and data-volume mappings follow them. Tag needs 2.42 and
  volume type 2.67. `WithLegacyBlockDeviceMappings` sends the legacy
  `block_device_mapping` list and cannot be combined with any v2 mapping.
  `WithTrustedImageCertificates` needs 2.63.
- `Servers.CreateReservation` sends `return_reservation_id` and returns only
  the reservation ID; combine it with `WithServerCount` for several servers.
  `Servers.Create` with a count returns the single server Nova reports.
- `config_drive` is a boolean; volume-ID values are not accepted by Nova.
- Falsy values are not silently dropped: `WithConfigDrive(false)` sends
  `false`, and empty admin passwords, tags and hint keys are rejected.

Readiness errors and cleanup:

- `WithCreateRefresh` matches Python's `wait=False` path: after the POST it
  performs one GET and returns the refreshed server, or `ServerFaultError`
  when it is in ERROR. A failed GET returns the POST response with the error;
  the POST is never repeated. It cannot be combined with `WithWait`.
- `Service.CreateRefreshedServer` (and `Connection.CreateRefreshedServer`)
  adds Python's `add_server_interfaces` step: it returns the POST response,
  the refreshed server with the creation `adminPass` copied onto it, and the
  `ExpandServerInterfaces` address view. `CreateWithAutomaticFloatingIP` also
  copies the creation `adminPass` onto its returned `Server`.
- Only Python's falsy address maps (null or `{}`) are deleted under
  `DeleteServerWithoutAddresses`. A map with only empty rows sets `EmptyRows`
  and is kept; Python would instead try to add IPs to it.

- A server in ERROR returns `*compute.ServerFaultError`, which unwraps to
  `resource.FailedStateError`. Its message follows Python's two forms
  ("Compute service reports fault: ..." or "no further information
  available"), based on whether the raw `fault` object has a `message` key.
  `Server` is the observed server, like Python's `extra_data`.
- An ACTIVE server with no address rows returns
  `ServerAddressesUnavailableError`. By default no DELETE is sent. Selecting
  `compute.DeleteServerWithoutAddresses` (`WithServerWithoutAddressesPolicy`
  or `AutomaticServerCreateOptions.ServerWithoutAddresses`) sends exactly one
  DELETE, records the result in `Deletion`, and waits for 404 when the
  workflow waits. A failed DELETE is joined to the error and never retried.

```go
ready, err := service.Servers.Wait(ctx, resource.ID(server.ID), "ACTIVE",
    resource.WithTimeout(5*time.Minute), resource.WithPollInterval(time.Second))
if err != nil { return err }
fmt.Println(ready.Status)

if err := service.Servers.Delete(ctx, resource.ID(server.ID)); err != nil {
    return err
}
```

## References

- [Console Auth Token](console-auth-token.md)
- [Console Selection](console-selection.md)
- [..](../README.md)
- [User Read APIs](user-read-apis.md)
- [User Actions](user-actions.md)
- [Keypairs List Find](keypairs-list-find.md)
- [Keypairs Cloud](keypairs-cloud.md)
- [Keypairs Console](keypairs-console.md)
- [Flavor Records](flavor-records.md)
- [Flavor Cloud](flavor-cloud.md)
- [Availability Zones](availability-zones.md)
- [Python Detail Records](v2/availabilityzones/python-detail-records.md)
- [Flavor Extra Specs](flavor-extra-specs.md)
- [Cloud keypair](keypairs-cloud.md)
- [AddIPsToServer](server-ip-helpers.md)
- [AddIPList](server-ip-helpers.md)
- [Server Addresses](server-addresses.md)
- [CreateWithAutomaticFloatingIP](create-with-automatic-floating-ip.md)
- [GetActiveServer](server-ready.md)
- [Service·Connection WaitForServer](server-ready.md)
- [pool·IP dispatch](server-ip-dispatch.md)
- [Compute](https://docs.openstack.org/openstacksdk/latest/user/proxies/compute.html)
- [Finding Identities](../docs/finding-identities.md)
- [Flavor Property And Native](flavor-property-and-native.md)
- [Server Default Network](server-default-network.md)
- [Server Network Interfaces](server-network-interfaces.md)
- [Python Cloud Parity](python-cloud-parity.md)
- [V2](v2/README.md)
- [compute_test.go](compute_test.go)
- [boot_volume_test.go](boot_volume_test.go)
- [new_boot_volume_test.go](new_boot_volume_test.go)
- [server_create_test.go](../connectiontest/server_create_test.go)
- [collections_test.go](../connectiontest/collections_test.go)
- [Service Waits](../docs/service-waits.md)
- [Server Ready](server-ready.md)
- [CreateWithFloatingIP](create-with-floating-ip.md)
- [Server Ip Dispatch](server-ip-dispatch.md)
- [Network Roles](../network/network-roles.md)
- [Create With Automatic Floating Ip](create-with-automatic-floating-ip.md)
- [Server Automatic Ip](server-automatic-ip.md)
- [Legacy Nova floating IP](server-nova-floating-ip.md)
- [Floating Ip Available](floating-ip-available.md)
- [Floating Ip Queries](floating-ip-queries.md)
- [Floating Ip Delete](floating-ip-delete.md)
- [Floating Ip Create](floating-ip-create.md)
- [Floating Ip Unattached Delete](floating-ip-unattached-delete.md)
