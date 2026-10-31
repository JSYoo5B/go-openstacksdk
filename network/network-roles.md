# Network Roles

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationSearchAndReadFailures`, `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationValidation`.
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
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationSearchAndReadFailures`, `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationValidation`.
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
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationSearchAndReadFailures`, `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationValidation`.
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

### `python:cloud/_network_common/NetworkCommonCloudMixin/get_default_network`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connect`
- `github.com/JSYoo5B/go-openstacksdk.WithDefaultNetwork`
- `github.com/JSYoo5B/go-openstacksdk.WithoutDefaultNetwork`
- `github.com/JSYoo5B/go-openstacksdk/compute.Servers.Create`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetDefaultNetwork`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ResetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.WithNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Discover`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Reset`
- `github.com/JSYoo5B/go-openstacksdk/network.PrepareNetworkRoleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network.WithConfiguredNetworks`
- `github.com/JSYoo5B/go-openstacksdk/network.WithExternalNetworkDiscovery`
- `github.com/JSYoo5B/go-openstacksdk/network.WithInternalNetworkDiscovery`
- `github.com/JSYoo5B/go-openstacksdk.WithServiceLookupFailureAsUnavailable`

- Connection owns a default-network Ref and explicit disable policy; the last option wins, invalid Refs remain errors, explicit NICs/modes bypass the default callback and Neutron endpoint lookup, and default IDs work for image/existing-volume/new-volume boots without altering their mappings.
  Tests: `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkIDsOptionsAndBootSources`, `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkDisableAndExplicitSelections`, `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkOptionValidationAndConcurrentReuse`, `compute/default_network_test.go:TestServerDefaultNetworkIsSkippedByExplicitNICsAndPreflight`.
- Typed default names resolve exactly; missing/ambiguous/403 failures, malformed callback Refs and cancellation stop before server creation. Explicit typed default-name resolution remains fresh; configured defaults share successful role discovery and failed discovery is not cached; callbacks and options are request-local, and wait failure preserves the created server without cleanup.
  Tests: `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkNameFailuresAndRetry`, `compute/default_network_test.go:TestServerDefaultNetworkDependencyFailureCancellationAndValidation`, `compute/default_network_test.go:TestServerDefaultNetworkDoesNotLeakBetweenRequestsAndPreservesWaitFailure`.
- Cloud/secure/public files are snapshotted once for native auth/endpoint/TLS and SDK metadata. Network lists replace at each precedence layer, [] clears inheritance, null/duplicate defaults/invalid flags are configuration errors, and empty/null metadata profile stops network inheritance while retaining native auth fallback.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationSearchAndReadFailures`, `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationValidation`.
- YAML default name matches exact network name OR ID across unfiltered complete pagination. Cross-kind collisions and duplicate names are ambiguous; missing/unsafe IDs, a later-page 403 and cancellation preserve errors and prevent Nova POST. Connection override/disable and explicit Create selection bypass the configured lookup.
  Tests: `connectiontest/connection_cloud_network_test.go:TestConnectConfiguredDefaultNetworkMatchesNamesOrIDsAcrossAllPages`, `connectiontest/connection_cloud_network_test.go:TestConnectDefaultNetworkOverridesDisableAndExplicitCreateSelection`.
- Shared discovery classifies IPv4 with router:external or provider:physical_network and IPv6 with router:external only, unless configured name-or-ID family overrides apply. Floating IPv4 candidates are router-external networks or the first configured NAT source; configured source/destination/default roles remain distinct from ordinary family roles.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- An explicit NAT destination skips subnet discovery. Without one, all subnet pages are consumed and the last network in network response order with a nonempty gateway subnet becomes the NAT destination, independent of subnet ip_version. Configured NAT/default name-or-ID selection rejects missing, duplicate or cross-kind ambiguous matches; the first configured NAT source is used.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- Unfiltered network discovery and conditional subnet discovery consume linked empty and later pages. Unsafe network IDs, invalid router extension decoding, later network403, subnet403 and repeated continuation links return errors without a partial snapshot; failed attempts remain retryable. Native204 returns an empty successful discovery.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- All ten Connection role getters share one successful discovery and return independent owned native-plus-extension models. Aggregate getters concatenate IPv4 then IPv6 with duplicate entries retained. Caller changes do not modify cached roles, and tags/subnets/availability-zone-hints retain explicit empty versus null native array presence.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`.
- Concurrent uncached discoveries share one in-flight request and return independent models. A waiting follower can reach its own context deadline without canceling the leader. Reset advances the cache generation so a successful already-running discovery cannot repopulate the cache, while the existing caller still receives its result; subsequent concurrent cached calls issue no new request.
  Tests: `network/roles_test.go:TestNetworkRoleConcurrentDiscoveriesShareFlightAndOwnResults`, `network/roles_test.go:TestNetworkRoleCacheFlightCancellationAndReset`.
- Typed use-external/use-internal flags default enabled and preserve ordered replacement. Both disabled returns empty roles before client/catalog/HTTP access; disabling only external still discovers both families when internal is enabled. Connection nil/canceled context fails before lookup, clean pointer or value ErrEndpointNotFound yields empty roles, and other catalog errors remain inspectable.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- Role selectors are frozen with the same cloud/secure/public snapshot as native authentication. YAML family flags inherit routes_externally unless overridden; a typed WithNetworkRoles policy replaces the YAML role configuration and its default selector, and an empty typed replacement clears configured roles/default. Configured default NIC selection now consumes the same successful role snapshot; explicit typed default names retain their separate exact-name lookup.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Owned configured-network slices can be reused after caller mutation. Nil options, empty names and multiple default/NAT-destination declarations fail before discovery; a later replacement does not hide an earlier invalid option. Explicit Reset invalidates a completed Connection role cache without making a mutation request.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- YAML boolean false and case-insensitive string false disable discovery before network HTTP. An explicit typed policy replaces those flags and restores discovery; a numeric YAML flag is rejected as ErrInvalidOption. Direct role discovery rejects nil or providerless clients before native HTTP.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleBooleanFlagsAndExplicitPolicyReplacement`, `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`.
- A configured default NIC consumes the same successful network/subnet role snapshot as Connection getters. Caller changes to a returned default cannot alter creation; consecutive creates reuse the successful discovery and Reset refreshes the chosen default ID for the next create.
  Tests: `connectiontest/connection_network_role_consumers_test.go:TestConnectionDefaultNICConsumesRoleCacheAndReset`, `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Configured default discovery validates the shared snapshot, so subnet403 or another configured role missing stops before Nova POST even when the default itself exists. Failed attempts are not cached; a subsequent success is reused. Later network-page errors, name-or-ID collisions, unsafe IDs and cancellation retain their established pre-create errors.
  Tests: `connectiontest/connection_network_role_consumers_test.go:TestConnectionDefaultNICRoleErrorsStopCreationAndRetry`, `connectiontest/connection_cloud_network_test.go:TestConnectConfiguredDefaultNetworkMatchesNamesOrIDsAcrossAllPages`.
- Both-disabled discovery or a clean pointer/value missing Network endpoint yields no configured default and keeps Nova auto at selected2.37 or omits networks when unset. Explicit default ID/disable and explicit NIC/mode bypass configured role discovery; explicit typed default names keep their exact-name resolution lane.
  Tests: `connectiontest/connection_network_role_consumers_test.go:TestConnectionDefaultNICEmptyRolesUseNovaNetworkDefault`, `connectiontest/connection_cloud_network_test.go:TestConnectDefaultNetworkOverridesDisableAndExplicitCreateSelection`, `connectiontest/connection_default_network_test.go:TestConnectionDefaultNetworkNameFailuresAndRetry`.
- Connection CreateWithFloatingIP uses a single successful configured role inventory for its default NIC, floating source and NAT destination, including a configured source without router:external. Role lookup finishes before Nova POST; actual server ACTIVE precedes port and floating-IP requests, and actual floating-IP ACTIVE remains mandatory. A role403 returns no created resource; a later port403 preserves the known ACTIVE Server with nil assignment and the original HTTP cause, without allocation or cleanup.
  Tests: `connectiontest/connection_server_network_roles_test.go:TestConnectionCreateWithFloatingIPSharesDefaultSourceAndNATRoles`.
- Owned high Service CreateNetwork/UpdateNetwork/DeleteNetwork and the existing Networks.Delete adapter invalidate shared roles after accepted mutations. The actual matrix primes a configured default role, observes a new inventory after success or high-delete clean404 after a found lookup, and retains the original inventory after403/409/412, failed no-option lookup, or direct Collection404. A successful no-option Update performs its lookup and reset without PUT; initial missing ID/Name high Delete returns false without mutation or reset.
  Tests: `network/network_mutations_test.go:TestNetworkMutationSuccessFailureAndMissingCacheBoundaries`, `network/network_mutations_test.go:TestDeleteNetworkInitialMissingDoesNotMutateOrReset`.
- Accepted mutation read/Close/context/source or decode/ID errors preserve the actual response evidence and invalidate the originally owned role cache without another mutation caused by that error. Create/Update malformed/null/missing or wrong-shaped network bodies return an inspectable ResponseError; decoded unsafe or mismatched Update IDs retain the partial model. Delete keeps true plus an accepted response error. Public role/collection replacements during accepted Close are errors but cannot redirect the cache reset to a replacement.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedDecodeAndIdentityErrorsResetWithoutResend`, `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedIOCancellationAndSourceErrorsKeepEvidenceAndOwnedReset`.
- A representative accepted Create resets while an earlier role inventory is in flight: that earlier caller still receives its original result, but the old generation cannot repopulate cache. The next configured default lookup reads a new inventory and repeated lookup uses that new cache. This is the actual Create/Reset fixture, not separate mutation-race coverage for every CRUD method or every getter.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationResetCannotPublishEarlierInflightDiscovery`.
- The Connection-cached high Network Service resets the same role cache after Create, Update and Delete. A later getter reads a new configured inventory, and Nova default NIC plus floating source/NAT allocation consume that same refreshed inventory without manual Reset. The fixture checks actual Nova and Neutron request fields and exactly two inventory reads.
  Tests: `connectiontest/connection_network_mutations_test.go:TestConnectionNetworkMutationRefreshesGetterDefaultNICAndFloatingRoles`.
- A has_network flag from WithServiceEnabled or clouds.yaml mirrors the has_service('network') check of _find_interesting_networks: disabled roles are empty without a network list even with configured roles, a typed enable overrides a YAML disable, and an enabled flag keeps discovery and configured NAT/default selection.
  Tests: `connectiontest/connection_network_service_disable_test.go:TestNetworkRolesHonourHasNetworkConfiguration`.
- The typed YAML use_external_network/use_internal_network policy agrees with Python raw-value truthiness for booleans, null, the empty string and any casing of 'true'; other strings disable discovery in Go though truthy in Python, and numbers or collections are configuration errors in Go while Python's normalize_keys makes numbers (even 0) truthy strings, as a table test records.
  Tests: `connectiontest/connection_network_role_flags_test.go:TestNetworkRoleUseFlagsCompareYAMLValuesWithPythonTruthiness`.
- The selected clouds.yaml regions entry overlays its non-null values on the merged cloud settings before role, default-network and has_<service> configuration is read: whole networks lists, use_* and has_network flags change per region. Without WithRegion the first entry (or the one named by region_name) is used; an unknown explicit region and malformed entries are configuration errors.
  Tests: `connectiontest/connection_cloud_region_config_test.go:TestCloudRegionValuesOverlayNetworkRoleConfiguration`.
- A profile absent from clouds-public.yaml (or with no such file) resolves to the embedded, unmodified pinned vendor profile: rackspace disables has_network and sets secgroup_source under the cloud's own settings, limestonenetworks supplies per-region networks and default interface, and native authentication receives the profile's auth settings. A clouds-public.yaml entry wins; shutdown and URL profiles are configuration errors, deprecated profiles load without warnings, and embedded files match their documented hashes.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudBuiltinVendorProfilesFeedRoleConfiguration`, `internal/cloudvendors/vendors_test.go:TestProfilesResolveStatusNestedAndRemoteNames`, `internal/cloudvendors/vendors_test.go:TestEmbeddedFilesMatchREADMEHashes`.
- Without OS_CLOUD, Python's environment-variable cloud is mapped for role settings: OS_INTERNAL_NETWORK/OS_EXTERNAL_NETWORK legacy roles, OS_USE_* flags (typed string policy) and an OS_PROFILE built-in vendor profile shape discovery, string OS_HAS_* values and shutdown profiles are configuration errors, and a typed WithNetworkRoles policy still replaces the environment roles.
  Tests: `connectiontest/connection_cloud_env_config_test.go:TestEnvironmentCloudSuppliesNetworkRoleConfiguration`.
- WithServiceLookupFailureAsUnavailable maps has_service returning False when the Network endpoint lookup raises: roles are empty and the lookup is retried on the next call, while the default returns the lookup error and cancellation is returned under either policy.
  Tests: `connectiontest/connection_service_lookup_test.go:TestServiceLookupFailureAsUnavailableMapsHasServiceFalse`.

Python/Go differences:

- The SDK exposes Connection.GetDefaultNetwork as an owned native-plus-extension nullable role model. Explicit typed defaults remain a separate Ref-based creation policy: IDs bypass lookup and names resolve exactly. Configured YAML/role defaults match name OR ID through the shared successful discovery.
- Explicit typed default-name resolution remains fresh; configured server defaults now consume the shared successful role snapshot. Network/subnet/catalog failures remain visible and retryable rather than Python sticky empty cache results; disabled discovery or clean missing endpoint yields no configured default and the existing Nova version fallback.
- YAML network names are nonblank strings; Python truthy numeric names are stringified and whitespace names can be preserved. List/flag shape failures return ErrInvalidOption instead of Python incidental TypeError/AttributeError. Metadata profile presence follows Python while native zero-profile authentication fallback remains unchanged.
- The SDK exposes context-aware Connection getters returning owned native-plus-router/provider models and typed nullable pointers; Python returns mutable Network Resources and cached list/object identities. Ordinary generated Networks model behavior is unchanged. This bounded role model does not emulate all nullable descriptors, unknown properties, location or session state.
- Go configured-role validation accepts a matching name OR ID consistently. Pinned classification accepts either, but its subsequent external/internal family validation checks only selected names and can reject ID-only selectors. Go also rejects unsafe IDs and preserves missing/ambiguous and decoding errors rather than incidental dynamic-input behavior.
- Go caches only successful discovery, coalesces in-flight work, lets followers use their own contexts, and uses Reset generations to prevent a pre-reset flight from restoring the cache. Python serializes shared discovery with a lock and can swallow SDKException into sticky empty results. Owned returns and observable/retryable errors are deliberate Go policies.
- Aggregate results retain the source IPv4-then-IPv6 order and duplicates; family classification reflects routing configuration and network extensions, not discovered subnet address families. The NAT destination inference preserves the source last-network-with-gateway rule.
- Go flags are strict typed booleans. The YAML parser agrees with Python truthiness of the raw value for booleans, null, the empty string and any casing of 'true'; other strings are false in Go but truthy in Python, and numeric or collection values are rejected instead of being truthiness-tested (Python's normalize_keys turns numbers, even 0, into truthy strings). Pinned cloud use_* accessors return the raw configured value; Go returns a bool. Both-disabled skips discovery; a single disabled flag does not remove that family after discovery.
- The Connection maps clean native missing-endpoint detection and the has_network configuration flag with official-service defaults (ServiceEnabled); Python's False for other lookup failures needs WithServiceLookupFailureAsUnavailable, and no version discovery applies to role discovery. Configured default NIC and automatic Ensure floating source use the shared successful role snapshot. Automatic NAT applies only without explicit port/fixed/NAT options and with multiple server-owned ports; strict unique IPv4-pair selection remains. These consumers do not implement private/floating-source flags or automatic IP need/skip decisions.
- Region values follow get_one's overlay for SDK-owned settings only; native authentication and endpoint parsing do not see them. Without WithRegion, an entry named by region_name is preferred over the first entry so that the values match the native endpoint region, whereas Python ignores region_name when regions is present. Typed Connect options (WithNetworkRoles, WithServiceEnabled) replace the configured policy for their concern instead of overlaying single keys like get_one keyword arguments.
- has_service('network') is mapped by ServiceEnabled (has_network flag, official-service default) plus the endpoint lookup. Python turns every lookup exception into False; Go returns lookup failures other than a missing endpoint unless WithServiceLookupFailureAsUnavailable is set, and always returns cancellation. Role discovery passes no version, so no version-range discovery applies, and Go emits no Python debug log.
- Configuration sources: clouds.yaml with secure.yaml, clouds-public.yaml or embedded built-in vendor profiles, region values, and the OS_* environment cloud when OS_CLOUD is unset. The environment cloud feeds only SDK-owned role, default-network and service settings; native authentication still uses Gophercloud's environment parser, and Python argparse/keyword arguments map to typed Connect options.
- Results are immutable owned snapshots rather than Python's mutable Resource with adapter, session, cache and dirty state (SDK-R1/C1/S1); see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Inputs are typed Go values; arbitrary Python objects, iterables and numeric types accepted by dynamic kwargs have no equivalent; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Network roles are cached per Connection and reset by the SDK's own CreateNetwork, UpdateNetwork and DeleteNetwork, as Python resets its role cache only inside cloud create_network, update_network and delete_network; changes made by other clients or through the raw NetworkV2 client need an explicit Reset, which Python does not see either.
- A clouds.yaml profile named by URL, including the nested URLs of the built-in ovh and vexxhost profiles, is rejected as a configuration error instead of being fetched from the vendor's well-known document at connect time; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- The consumers of the role snapshot (create_server, add_ips_to_server, get_active_server, wait_for_server) are reviewed under those operations; this operation covers discovery and the returned role.

### `python:cloud/_network_common/NetworkCommonCloudMixin/available_floating_ip`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.Ensure`
- `github.com/JSYoo5B/go-openstacksdk/network.PrepareEnsureFloatingIPOptions`
- `github.com/JSYoo5B/go-openstacksdk/network.WithEnsureFloatingIPPolicy`
- `github.com/JSYoo5B/go-openstacksdk/network.WithEnsureReuse`
- `github.com/JSYoo5B/go-openstacksdk/network.WithEnsureProject`
- `github.com/JSYoo5B/go-openstacksdk/network.WithEnsurePort`
- `github.com/JSYoo5B/go-openstacksdk/network.WithEnsureNATDestination`
- `github.com/JSYoo5B/go-openstacksdk/network.WithEnsureFixedAddress`
- `github.com/JSYoo5B/go-openstacksdk/network.WithEnsureWait`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.PrepareEnsureActive`
- `github.com/JSYoo5B/go-openstacksdk/compute.Servers.CreateWithFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.Create`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Discover`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Reset`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.PrepareEnsure`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.EnsurePrepared`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPPlan.Selection`
- `github.com/JSYoo5B/go-openstacksdk.Connection.PlanServerFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk.Connection.EnsureServerFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.PlanServerFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.EnsureServerFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.NewPlanner`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPPlanner.NetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPPlanner.PrepareEnsure`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPPlanner.Check`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPPool`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPAddresses`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.AddIPsToServer`
- `github.com/JSYoo5B/go-openstacksdk.Connection.AddIPsToServer`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerIPAutomaticOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerIPWait`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPSource`
- `github.com/JSYoo5B/go-openstacksdk.WithServerAddressPolicy`
- `github.com/JSYoo5B/go-openstacksdk.WithMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetActiveServer`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.WaitForServer`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.CreateWithAutomaticFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetActiveServer`
- `github.com/JSYoo5B/go-openstacksdk.Connection.WaitForServer`
- `github.com/JSYoo5B/go-openstacksdk.Connection.CreateWithAutomaticFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.Available`
- `github.com/JSYoo5B/go-openstacksdk/network.PrepareAvailableFloatingIPOptions`
- `github.com/JSYoo5B/go-openstacksdk/network.WithAvailableFloatingIPPolicy`
- `github.com/JSYoo5B/go-openstacksdk/network.WithAvailableProject`
- `github.com/JSYoo5B/go-openstacksdk/network.WithAvailableFixedAddress`
- `github.com/JSYoo5B/go-openstacksdk/network.WithAvailableNATDestination`
- `github.com/JSYoo5B/go-openstacksdk/network.WithAvailableTimeout`
- `github.com/JSYoo5B/go-openstacksdk/network.WithUnlimitedAvailableTimeout`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.AvailableFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAvailableIPSource`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAvailableIPNetworkOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAvailableIPTimeout`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithUnlimitedAvailableIPTimeout`
- `github.com/JSYoo5B/go-openstacksdk.Connection.AvailableFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAvailableIPLocation`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAvailableIPStrict`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.PrepareAvailability`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPPlanner.PrepareAvailability`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPAvailabilityPlan.NetworkID`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPAvailabilityPlan.ProjectID`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPAvailabilityPlan.Check`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPAvailabilityPlan.Allocate`

- Reuse defaults to true. A reusable prepared policy reads the current recorded Keystone owner and live token separately on each call, resolves a server name through Connection, and fails before HTTP when reuse has no recorded/explicit owner or options are invalid. Disabling reuse skips available-IP listing and may omit project_id so Neutron selects the owner.
  Tests: `connectiontest/connection_floating_ip_ensure_test.go:TestConnectionFloatingIPEnsureReadsCurrentAuthAndResolvesServer`, `connectiontest/connection_floating_ip_ensure_test.go:TestConnectionFloatingIPEnsureWithoutReuseLetsNeutronChooseProject`, `network/floating_ip_ensure_test.go:TestFloatingIPEnsureValidationAndUnknownProjectPrecedeHTTP`.
- Reuse lists owner/network-constrained candidates and rechecks project/legacy tenant consistency, external network, IPv4, ERROR status and attachment destination locally over both pages. It chooses an already-associated candidate before the first free candidate in response order. Revision absent/null omits If-Match, while present zero/positive values preserve revision_number=N for the association PUT.
  Tests: `network/floating_ip_ensure_test.go:TestFloatingIPEnsureFiltersAllPagesAndPreservesRevisionPresence`, `network/floating_ip_ensure_test.go:TestFloatingIPEnsurePrefersAlreadyAttachedAndAllocatesOnlyWhenNeeded`.
- A zero external-network Ref consumes the first shared ExternalIPv4Floating role candidate, then an enabled router gateway only when the successfully discovered role list is empty. Selection consumes later/linked-empty pages before mutation. Network/subnet/router errors, invalid/no gateways and cancellation stop before port/IP mutation; port and available-IP selection also follow linked empty pages.
  Tests: `network/floating_ip_selection_test.go:TestFloatingIPEnsureAutomaticNetworkConsumesEmptyAndLaterPages`.
- Selection preserves native response URL and rejects repeated next links, including empty pages, before another GET or allocation/association.
  Tests: `network/floating_ip_selection_test.go:TestFloatingIPEnsureRejectsEmptyPageCyclesBeforeRefetchOrMutation`.
- No reusable candidate or explicit reuse=false makes one Neutron POST with selected external network, uniquely selected server port/fixed IPv4 and the explicit/resolved owner when present. An already attached candidate makes neither PUT nor POST. Allocation responses use the native floatingip envelope and preserve Allocated/Reused phase flags.
  Tests: `network/floating_ip_ensure_test.go:TestFloatingIPEnsurePrefersAlreadyAttachedAndAllocatesOnlyWhenNeeded`, `connectiontest/connection_floating_ip_ensure_test.go:TestConnectionFloatingIPEnsureWithoutReuseLetsNeutronChooseProject`.
- List/continuation HTTP failures prevent mutation. Association 409/412 never triggers a new allocation; the selected reused IP remains observable. Response ID/project/destination mismatches and wait ERROR/custom-status mismatches return errors with the selected/allocated assignment rather than silently succeeding or deleting it.
  Tests: `network/floating_ip_ensure_test.go:TestFloatingIPEnsureErrorsPreserveSelectionAndNeverFallback`.
- ACTIVE waiting is optional and verifies the actual returned identity/project/destination/Status. Wait timeout or cancellation retains the allocated assignment and performs no cleanup or fallback allocation.
  Tests: `network/floating_ip_selection_test.go:TestFloatingIPEnsureWaitCancellationAndTimeoutPreserveAssignment`, `network/floating_ip_ensure_test.go:TestFloatingIPEnsureErrorsPreserveSelectionAndNeverFallback`.
- PrepareEnsureActive binds the recorded reuse owner to a copy and forces ACTIVE wait while preserving poll/timeout/progress configuration. Changing recorded authentication afterward does not change that bound owner; the original plain prepared policy still resolves the new owner per Ensure call and keeps its original no-wait behavior.
  Tests: `network/floating_ip_prepare_test.go:TestFloatingIPPrepareActiveBindsCopyAndPreservesWaitPolicy`.
- Server wait failures retain the POST server; after actual ACTIVE verification IP selection failures retain the server and late guarded PUT/wait/malformed response failures retain server plus reused/allocated assignment. No fallback allocation, deletion or synthesized Nova addresses is performed.
  Tests: `connectiontest/connection_server_floating_ip_test.go:TestConnectionServerFloatingIPOrdersWaitsAndPreservesPartialResults`, `connectiontest/connection_server_floating_ip_test.go:TestConnectionServerFloatingIPUsesOneDeadlineAcrossBothWaits`.
- Ensure zero external Ref consumes the shared floating-role candidates rather than the old router-external filtered collection. Configured NAT source may select a non-router-external network and ordinary physical-provider external IPv4 is not automatically a floating source. Consecutive calls reuse the owned successful role snapshot, still allocate per call, and Reset refreshes the chosen external ID; PrepareEnsureActive itself does not perform role discovery.
  Tests: `network/floating_ip_roles_test.go:TestFloatingIPEnsureSharedSourceCacheAndReset`.
- Failed shared network/subnet discovery, including later-page403 or a configured floating source missing, prevents port/IP mutation and router fallback and remains retryable. Only a successful empty floating-role list proceeds to enabled-router gateway fallback; linked empty/later router pages and their errors are consumed before mutation. Both family flags disabled also produce an empty role snapshot and allow router fallback without a role inventory request, despite a configured source missing.
  Tests: `network/floating_ip_roles_test.go:TestFloatingIPEnsureRoleErrorsStopRouterFallbackAndRetry`, `network/floating_ip_selection_test.go:TestFloatingIPEnsureAutomaticNetworkConsumesEmptyAndLaterPages`, `network/floating_ip_roles_test.go:TestFloatingIPEnsureDisabledRolesStillUseRouterFallback`.
- High-level FloatingIPs.Create and Ensure consume shared configured or inferred NAT destination only without explicit port/fixed/NAT selection and with multiple ports owned by the requested server. All port pages are consumed first and foreign ports are excluded. Successful NAT discovery is reused but ports are listed on each call; Reset changes the chosen destination while preserving the actual allocation body.
  Tests: `network/floating_ip_roles_test.go:TestFloatingIPAutomaticNATUsesSharedDestinationAndReset`.
- Owned port count precedes IPv4 filtering, so an IPv6-only second port requires a NAT role. No NAT role returns ErrAmbiguous with sorted owned port IDs; a selected network absent from ports returns ErrNotFound; multiple selected IPv4 pairs remain ErrAmbiguous. Role403, later port403 and subnet cancellation prevent allocation without fallback or cleanup.
  Tests: `network/floating_ip_roles_test.go:TestFloatingIPAutomaticNATFailuresStopAllocation`.
- A single owned port or explicit fixed address, port or NAT destination bypasses automatic NAT role discovery despite an invalid configured policy elsewhere. Explicit port performs its GET rather than port listing; explicit NAT retains network_id filtering. An external Name retains the router:external/exact-name lane, while an explicit external ID needs no network-role inventory.
  Tests: `network/floating_ip_roles_test.go:TestFloatingIPExplicitAndSinglePortSelectionBypassRoles`.
- Connection CreateWithFloatingIP uses a single successful configured role inventory for its default NIC, floating source and NAT destination, including a configured source without router:external. Role lookup finishes before Nova POST; actual server ACTIVE precedes port and floating-IP requests, and actual floating-IP ACTIVE remains mandatory. A role403 returns no created resource; a later port403 preserves the known ACTIVE Server with nil assignment and the original HTTP cause, without allocation or cleanup.
  Tests: `connectiontest/connection_server_network_roles_test.go:TestConnectionCreateWithFloatingIPSharesDefaultSourceAndNATRoles`.
- Owned high Service CreateNetwork/UpdateNetwork/DeleteNetwork and the existing Networks.Delete adapter invalidate shared roles after accepted mutations. The actual matrix primes a configured default role, observes a new inventory after success or high-delete clean404 after a found lookup, and retains the original inventory after403/409/412, failed no-option lookup, or direct Collection404. A successful no-option Update performs its lookup and reset without PUT; initial missing ID/Name high Delete returns false without mutation or reset.
  Tests: `network/network_mutations_test.go:TestNetworkMutationSuccessFailureAndMissingCacheBoundaries`, `network/network_mutations_test.go:TestDeleteNetworkInitialMissingDoesNotMutateOrReset`.
- The Connection-cached high Network Service resets the same role cache after Create, Update and Delete. A later getter reads a new configured inventory, and Nova default NIC plus floating source/NAT allocation consume that same refreshed inventory without manual Reset. The fixture checks actual Nova and Neutron request fields and exactly two inventory reads.
  Tests: `connectiontest/connection_network_mutations_test.go:TestConnectionNetworkMutationRefreshesGetterDefaultNICAndFloatingRoles`.
- The separate Neutron PrepareEnsure lane selects an owned concrete external-network/server-port/fixed-IPv4 plan without recorded-project lookup or IP mutation. Selection returns a copy. EnsurePrepared binds reuse owner only at execution, refuses zero/foreign plans and rechecks the exact port ID/server/network/fixed address before IP listing or mutation, preserving accepted GET evidence on drift instead of reselecting. Ordinary Ensure keeps its earlier owner preflight.
  Tests: `network/floating_ip_plan_test.go:TestFloatingIPPlanDefersOwnerAndPreservesConcreteSelection`, `network/floating_ip_plan_test.go:TestFloatingIPPlanRechecksPortWithoutReselectionOrMutation`, `network/floating_ip_plan_test.go:TestFloatingIPPlanSourceGuardStopsMutationRetryAndRejectsForeignPlans`.
- Only completed automatic no-external/no-server-port or unconstrained explicit-fixed search absence receives a typed plan absence. IPv6-only nonempty ports, wrong NAT, ambiguous destinations, explicit port404 and later403 remain errors. A single preparation holds one shared role snapshot across Reset; a later preparation sees new inventory. Initial public dependency replacement and changes during cold owned role response Close stop the plan, retaining response evidence and making no follow-up request.
  Tests: `network/floating_ip_plan_test.go:TestFloatingIPPlanTypedAbsenceDoesNotHideSelectionOrHTTPErrors`, `network/floating_ip_plan_boundaries_test.go:TestFloatingIPPlanUsesOneRoleSnapshotAcrossReset`, `network/floating_ip_plan_boundaries_test.go:TestFloatingIPPlanRejectsInitialPublicDependencyReplacement`, `network/floating_ip_plan_boundaries_test.go:TestFloatingIPPlanColdRoleReadGuardsAcceptedCloseBeforeNextHTTP`.
- Prepared assignment consumes empty/HTTP-Link candidate pages before selecting an attached IP over a free one, preserves revision zero for guarded PUT, and waits for actual ACTIVE with target invariants. Source-changing native mutation retry stops before resending and remains rejected after restoration. The owned REST lane also prevents retry removal/change/conflicting-case rewrite of SDK If-Match while existing unrelated caller headers/authentication retain their policy. Wait callback and preparation cancellation preserve the custom context cause and known assignment. Actual prepared PUT fixtures reject caller retry deletion/replacement or conflicting source revision headers, preserve original503/412 evidence and the original free candidate, and preserve accepted200 body/header/status on ID or fixed-address mismatch, without replacement allocation or cleanup.
  Tests: `network/floating_ip_plan_test.go:TestFloatingIPPlanReusesAfterAllPagesAndWaitsForActualACTIVE`, `network/floating_ip_plan_test.go:TestFloatingIPPlanSourceGuardStopsMutationRetryAndRejectsForeignPlans`, `internal/rest/request_headers_test.go:TestJSONRetryPreservesSDKOwnedHeaders`, `network/floating_ip_plan_boundaries_test.go:TestFloatingIPPlanWaitAndPreflightPreserveCancellationCause`, `network/floating_ip_plan_boundaries_test.go:TestFloatingIPPlanRevisionHeadersRejectRetryChangesAndSourceOverride`, `network/floating_ip_plan_boundaries_test.go:TestFloatingIPPlanPUTFailurePreservesOriginalCandidateAndHTTPProof`.
- The separate Compute automatic existing-server consumer now uses one guarded planner for owned address-role classification and concrete Neutron selection, skips known policy/address cases without owner work, preflights required raw observation capability before mutation, executes the selected tuple and observes the assigned floating IPv4 in actual ACTIVE Nova response rows. It preserves Decision/Assignment/known Server and context/HTTP evidence; this consumer is separate from the standalone availability getter and does not make ordinary Create/Wait equivalent.
  Tests: `compute/server_automatic_ip_test.go:TestAutomaticIPKnownSkipsAvoidAllServiceAndOwnerWork`, `network/floating_ip_planner_test.go:TestFloatingIPPlannerSharesOwnedClassificationAndSelectionSnapshot`, `compute/server_automatic_ip_boundaries_test.go:TestAutomaticIPObservationCapabilityAndReuseOwnerPreflight`, `compute/server_automatic_ip_test.go:TestAutomaticIPExecutesFixedTargetAndObservesSecondRawFloatingIPv4`, `compute/server_automatic_ip_boundaries_test.go:TestAutomaticIPAcceptedRawCloseFailuresRetainEvidenceWithoutRetry`.
- The SDK-owned prepared Neutron plan retains a decoded accepted201/202 allocation after complete-body read, Close, source or custom-cancellation errors with actual status/header/body and original cause; no retry, ACTIVE wait, observer or cleanup follows. Decoded wrong-owner allocation remains evidence alongside joined invariant/Close failures. Truncated202 read and Close causes remain joined with the actual json.SyntaxError and Allocated=true/nil model. Reused PUT200 adopts a complete matching body beside Close failure, but wrong-ID/fixed responses preserve the original candidate and both processing and validation causes; revision zero remains protected. This is owned workflow evidence, not a change to generated native transport.
  Tests: `network/floating_ip_accepted_model_test.go:TestFloatingIPAcceptedAllocationKeepsDecodedModelAndProcessingCauses`, `network/floating_ip_accepted_model_test.go:TestFloatingIPAcceptedAssociationAdoptsOnlyMatchingModelOnCloseError`, `network/floating_ip_accepted_model_test.go:TestFloatingIPAcceptedReadAndDecodeErrorsRemainJoined`.
- In the tested Ensure consumer a selected pool wins over malformed explicit address strings regardless of selector option order, even with auto disabled, private cloud and supplied public/floating metadata. Pool reuse=true uses a free owned candidate and no POST; reuse=false sends one attached allocation with selected network/port/fixed/project fields. Pool zero clears a prior invalid pool, empty address options clear a prior malformed list, and only the final winning selector is validated before HTTP. This is the existing Ensure pool policy, not source optional-server unattached availability.
  Tests: `compute/server_ip_dispatch_test.go:TestServerIPDispatchPoolWinsIndependentOfOptionOrderAndIgnoresIPSelectors`, `compute/server_ip_dispatch_test.go:TestServerIPDispatchClearsSelectorsAndValidatesOnlyWinningSelector`.
- The standalone Connection Ensure automatic lane retains its lazily resolved Compute source registry across raw observation. A progress callback that clears cached Compute API, Servers or Flavors returns ErrInvalidOption with the known matching raw Server and allocated assignment, Observed=false, one allocation, one raw fetch and one callback; no subsequent fetch is allowed. This is a concrete source-ownership regression, not complete source configuration/session parity.
  Tests: `connectiontest/connection_server_automatic_sources_test.go:TestConnectionStandaloneAutomaticIPRetainsLazyComputeSourceGuard`.
- Standalone AddIPsToServer uses the same winning pool priority with ignored malformed address syntax even for supplied BUILD/nil metadata and auto disabled. The async reuse=true fixture binds a free candidate with zero allocation POST; reuse=false sends one attached pool allocation. It returns one completed DOWN assignment/Observed=false and no raw observation. A separate already-associated pool fixture preserves supplied BUILD/nil metadata and returns Reused=true/Allocated=false without Compute resolution, raw fetch, association PUT, allocation POST or singular selected-IP GET. These bounded consumers do not implement source unattached-first optional-server availability or its nonreuse refresh/timing path.
  Tests: `compute/server_ip_helpers_test.go:TestStandaloneServerIPPoolUsesSamePriorityAndAsyncAllocationPolicy`, `compute/server_ip_helpers_test.go:TestStandaloneServerIPAlreadyAttachedAddressesAndPoolKeepComputeLazy`.
- The bounded Nova pool consumer uses the literal selected pool ahead of ignored malformed address selectors. Reuse selects the first matching raw instance_id=null candidate without allocation; reuse=false sends POST200 with only the pool name, one compatibility GET and a further selected-IP recheck before action202. An empty-string instance_id is not a free null candidate. Automatic mode without a pool chooses the first returned pool and uses the same assignment foundation; automatic disabled/private/floating/empty skips send no fixture HTTP and create no NovaAssignment. These are assignment consumers, not the source optional-server availability getter.
  Tests: `compute/server_ip_nova_test.go:TestNovaServerIPPoolReuseAndFreshAllocation`, `compute/server_ip_nova_test.go:TestNovaServerIPListAbsenceAndAssociationMetadataCannotAllocate`, `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPAutomaticDefaultPoolAndKnownSkips`.
- Nova matching is local and unfiltered: an unrelated null address in another pool does not poison a valid exact-address or free-pool selection, and present canonical null prevents the legacy address alias from becoming a second match. Fractional numeric IDs, selected missing association metadata and duplicate matching addresses fail without action/allocation; duplicate yields ErrAmbiguous. List204 is an error rather than absence, and an empty200 default-pool response is ErrNotFound. When a first IP inventory page already has a match but its advertised next page returns403, both address and pool consumers retain the late native status/header/body, return no NovaAssignment and send no selected-IP GET/action/allocation.
  Tests: `compute/server_ip_nova_test.go:TestNovaServerIPLocalSelectionIgnoresUnrelatedNullAddresses`, `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPSelectedDecoderAndPoolResponseBoundaries`, `compute/server_ip_nova_test.go:TestNovaServerIPListAbsenceAndAssociationMetadataCannotAllocate`, `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPLateInventoryPageFailureCannotAttachOrAllocate`.
- A Nova allocation accepted200 retains Allocated=true, its decoded IP and AllocationResponse proof when a later compatibility GET is denied or contradicts the selected address/pool. Accepted allocation or compatibility Close failure preserves the known model beside the original Close cause and the actual failed response proof; no association action starts. A decoded wrong-pool allocation remains partial evidence and is not used as a new target.
  Tests: `compute/server_ip_nova_test.go:TestNovaServerIPAllocationAndCompatibilityFailureKeepPartial`, `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPAcceptedAllocationAndCompatibilityClosePreserveProof`.
- The separate Network Available getter selects the first caller-ordered matching shared floating-network role and lists floatingips without pushed-down filters. It skips attached, empty-string-port and foreign-project rows, then returns the first free candidate without server-name resolution, allocation, association or readiness waiting. The fixture keeps an IPv6-addressed ERROR candidate and its raw nullable port/vendor integer metadata; Connection Neutron reuse resolves only Network and keeps Compute lazy.
  Tests: `network/floating_ip_available_test.go:TestFloatingIPAvailableFreeFirstNeverUsesServer`, `connectiontest/connection_floating_ip_available_test.go:TestConnectionAvailableIPNeutronFreeUsesNoCompute`.
- Available compares raw nullable project ownership locally. Without a recorded project the tested null project_id or tenant_id candidate is reusable, missing port_id is treated as null, an explicit null project_id wins a conflicting tenant_id, and an empty-string project is excluded. The native model and owned raw Metadata remain distinct.
  Tests: `network/floating_ip_available_test.go:TestFloatingIPAvailableNullableCurrentProjectAndPort`.
- Available consumes the linked candidate inventory before returning its first free result or allocating. A successful second empty page permits that first result; later 403, malformed 200 or204 returns an error with no result and no allocation despite a free row on the first page.
  Tests: `network/floating_ip_available_test.go:TestFloatingIPAvailableConsumesEveryPageBeforeReturningOrAllocating`.
- When no free candidate exists, optional Server controls only new Neutron allocation. Absent Server skips port listing; no ports or an explicit fixed IPv4 absent from the server ports produce an unattached allocation. With multiple ports and a configured NAT destination, the tested newer port is selected and its first IPv4 after an IPv6 row is used. POST 201/202 sends only floating_network_id plus selected port_id/fixed_ip_address; WithAvailableProject is a reuse filter and never overrides the allocation project. The actual authenticated owner and DOWN status are returned without a wait.
  Tests: `network/floating_ip_available_test.go:TestFloatingIPAvailableAllocationUsesAuthenticatedProjectAndOptionalPort`.
- The getter refuses allocation when multiple owned ports lack a NAT destination or a nonempty selected inventory has no IPv4. It retains a decoded accepted 201 allocation, raw wrong-network field and AllocationResponse/ResponseError status/header evidence beside an invariant error rather than deleting or retrying the known allocation.
  Tests: `network/floating_ip_available_boundaries_test.go:TestFloatingIPAvailableMultiplePortsRequireNATAndIPv4`, `network/floating_ip_available_boundaries_test.go:TestFloatingIPAvailableAllocationMismatchKeepsKnownModel`.
- Connection AvailableFloatingIP selects legacy Nova for explicit Nova or None source, ignores the optional Server even when its Ref is invalid, and returns a free unattached candidate without Network discovery, action or raw-server observation. The tested raw integer ID 9007199254740993 is preserved as a decimal string with the actual HTTP200 model and common identity/address.
  Tests: `connectiontest/connection_floating_ip_available_test.go:TestConnectionAvailableIPNovaAndNoneReuseWithoutAttachment`.
- The Nova getter uses the literal pool or first discovered default pool, consumes the free inventory, and when empty performs POST 200 plus the compatibility GET for the accepted IP. Its result remains unattached, with Allocated/Reused phases and the original allocation proof; no addFloatingIp action, association recheck, readiness wait or server observation is part of this getter.
  Tests: `connectiontest/connection_floating_ip_available_test.go:TestConnectionAvailableIPNovaAllocationOnlyAndDefaultPool`.
- The Connection getter distinguishes public-list inner fallback from outer availability fallback. Completed external-network absence or a pure singly wrapped missing catalog endpoint can enter outer Nova availability. Clean unfiltered Neutron-list404 first produces configured-Neutron-normalized Nova public inventory and applies the Neutron selected-network/project/null-port filter; no match allocates on Neutron before a pure pre-accept helper NotFound may enter outer Nova. HTTP403/204, malformed JSON, mixed catalog causes, accepted processing errors and terminal SDK failure never create another allocation. Caller retry still retains actual HTTP failure evidence and terminal SDK policy.
  Tests: `connectiontest/connection_floating_ip_available_test.go:TestConnectionAvailableIPNotFoundFallbackAndOtherFailuresStayDistinct`, `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronInnerListFallbackAllocatesOnSelectedNetwork`, `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronOuterFallbackRelistsOriginalPoolUnderOneBudget`, `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronListFailuresNeverAllocateMatchingPrefix`.
- Accepted Nova 200 or Neutron 201 allocation followed by Close failure, source replacement or custom cancellation retains Backend/Allocated, the decoded known IP and actual allocation ResponseError body/header/status. The Close/cancellation cause or source ErrInvalidOption remains accessible; no later compatibility GET, association or observation follows the failed allocation response.
  Tests: `connectiontest/connection_floating_ip_available_test.go:TestConnectionAvailableIPAcceptedFailuresKeepBackendAndAllocation`.
- The getter adds no SDK deadline by default. A 17 second override is shared by all tested Nova list/POST/compatibility requests, a shorter parent deadline wins, and explicit unlimited removes the SDK cap while preserving a 3 minute parent. Caller network selection is copied before option callbacks, and outer plus nested prepared option factories each run once. The assertions inspect request deadlines and do not simulate an actual timer expiry.
  Tests: `connectiontest/connection_floating_ip_available_test.go:TestConnectionAvailableIPBudgetAndPreparedOptionsOwnInputs`.
- Captured source ownership survives option application: Network endpoint/provider/version replacement or invalid fixed IPv4 stops before HTTP; replacing or clearing Compute Servers/API/endpoint, including through a nested network option, returns ErrInvalidOption before a request. Legacy Nova version 2.36, multiple literal pools, nil context and pre-cancelled context also stop before HTTP, with Unsupported/version and cancellation cause checks and no service discovery for the non-version preflight cases.
  Tests: `network/floating_ip_available_boundaries_test.go:TestFloatingIPAvailableOptionsCannotReplaceCapturedClient`, `connectiontest/connection_floating_ip_available_test.go:TestAvailableIPServiceOptionsCannotReplaceCapturedSource`, `connectiontest/connection_floating_ip_available_test.go:TestConnectionAvailableIPVersionMultiplePoolsAndContextPreflight`.
- On rows accepted by the existing typed availability lane, the additive FloatingIP record projects the final observed row through the shared cloud view engine without another query, allocation, source capture or deadline. Direct Nova and None sources return a normalized Resource with source-synthetic ACTIVE, canonical-null alias priority, properties and optional strict compatibility aliases; Wire, raw Nova metadata, decimal integer identity, common ID/Address and actual allocation phases remain unchanged. This view status is not an ACTIVE wait or association proof.
  Tests: `connectiontest/connection_floating_ip_available_view_test.go:TestAvailableIPNovaViewPreservesWireAndStrictAliases`, `compute/floating_ip_query_normalize_test.go:TestFloatingIPQueryNormalizationCanonicalNullPropertiesAndStrictAliases`.
- Resource projection captures configured Neutron mode after successful service discovery rather than inferring normalization from final Backend. Inner public-list Nova fallback and semantic outer Nova fallback can return actual Nova rows with configured-Neutron UNKNOWN status and port-based attachment when status is absent; a missing Network endpoint instead uses Nova ACTIVE and instance-based attachment. View status is not readiness. The inner helper reuses shared Query/Create engines and the distinct inner/outer evidence records the implemented fallback order.
  Tests: `connectiontest/connection_floating_ip_available_view_test.go:TestAvailableIPViewUsesConfiguredSourceAfterFallback`, `compute/floating_ip_query_normalize_test.go:TestFloatingIPQueryNormalizationUsesConfiguredNeutronAfterNovaFallback`, `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronInnerNovaReuseIgnoresInstanceAndServer`, `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronOuterFallbackRelistsOriginalPoolUnderOneBudget`.
- The public Neutron availability path constructs shared Resource projections for inventory before its local free filter, including name/address and project/tenant aliases, known Body defaults, revision, tags, self removal and owned location. Strict does not remove Neutron Resource fields. The existing IPv6 ERROR candidate remains reusable without Compute/server lookup; the primary returned record owns Resource/Wire independently of Inventory and legacy Metadata/header/model. No readiness wait is introduced.
  Tests: `connectiontest/connection_floating_ip_available_view_test.go:TestAvailableIPNeutronViewAndLocationAreOwned`, `connectiontest/connection_floating_ip_queries_test.go:TestFloatingIPQueryNeutronProjectionAndWireOwnership`, `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronListFailuresNeverAllocateMatchingPrefix`, `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronInnerNovaReuseIgnoresInstanceAndServer`.
- Available snapshots the outer callback slice and input selection before callbacks, clones source/nested-options/location between application steps and guards each outer and nested callback before and after execution. WithAvailableIPLocation owns factory inputs and overrides the recorded/configured default. The tested caller changes cannot replace the captured strict option, network selector, nested callback or location bytes; factories run once and view construction retains the same tested17second request budget. Source or custom-cancellation failure prevents a later outer callback and HTTP.
  Tests: `connectiontest/connection_floating_ip_available_view_test.go:TestAvailableIPViewOptionsSnapshotAndBudget`, `connectiontest/connection_floating_ip_available_test.go:TestConnectionAvailableIPBudgetAndPreparedOptionsOwnInputs`, `connectiontest/connection_floating_ip_available_test.go:TestAvailableIPServiceOptionsCannotReplaceCapturedSource`.
- The view adapter stores an owned Wire-only record before location/projection work. A final Nova location getter error, source replacement or custom cancellation retains the accepted allocation, ID, raw model and receipt with nil Resource, joins the original cause and preserves the actual compatibility GET envelope/header/status rather than constructing a response from serialized row fields. An explicit location override skips the dependency getter and returns a successful view. No later allocation, action or observation follows a view error.
  Tests: `connectiontest/connection_floating_ip_available_view_test.go:TestAvailableIPViewLocationErrorKeepsKnownAllocation`.
- After accepted Nova200 or Neutron201 allocation followed by Close, source or cancellation failure, the additive record preserves known raw Wire alongside Backend/Allocated/model, actual allocation envelope receipt and cause; no compatibility GET or replacement allocation follows accepted processing failure. Source/context failure does not create a successful Resource view or allow another request. Neutron allocation-view errors preserve the exact accepted receipt and known raw allocation with nil Resource. A public Neutron inventory-view error instead retains Inventory.Failure, its actual200 page Envelope/Header and ResponseError, with no returned free candidate, legacy Neutron result, Creation or allocation; it does not invent an HTTP envelope from free-row metadata.
  Tests: `connectiontest/connection_floating_ip_available_view_test.go:TestAvailableIPViewAcceptedFailuresPreserveRawReceipts`, `connectiontest/connection_floating_ip_available_test.go:TestConnectionAvailableIPAcceptedFailuresKeepBackendAndAllocation`, `connectiontest/connection_floating_ip_available_view_test.go:TestAvailableIPNeutronViewErrorsKeepPhysicalReceipts`, `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronListFailuresNeverAllocateMatchingPrefix`.
- Direct/outer Nova availability uses the captured raw Query collection and an insertion-ordered instance_id=null then literal-pool filter before normalization. Excluded attached/empty-instance/other-pool rows are not subject to mutation ID/IPv4 checks. Missing visited keys preserve actual page proof; passive null IDs/non-IPv4 values remain in the owned FloatingIPRecord, with optional legacy typed projection. Explicit-pool reuse adds no pool/POST/action/server/port/readiness/cleanup request.
  Tests: `connectiontest/connection_floating_ip_available_nova_test.go:TestAvailableIPNovaRawFilterBeforeNormalization`.
- All matching Nova rows are normalized before returning the first; a late matching normalization error prevents successful first-row return and keeps original page evidence. Location is evaluated once per matching row and excluded rows are not normalized. The returned first Resource/Wire do not alias Inventory. Semantic outer Neutron NotFound retains configured-Neutron port/status normalization on actual Nova rows.
  Tests: `connectiontest/connection_floating_ip_available_nova_test.go:TestAvailableIPNovaNormalizesEveryMatchBeforeFirst`, `connectiontest/connection_floating_ip_available_view_test.go:TestAvailableIPViewUsesConfiguredSourceAfterFallback`.
- A clean raw Nova-list404 becomes empty inventory and continues to fresh POST plus mandatory compatibility GET; forbidden/malformed/accepted terminal failures remain errors. Default pool lookup precedes inventory and a selected non-null pool is reused without a second lookup. Inventory retains physical pages, suppressed NotFound and failure evidence, while availability discards failed inventory prefixes before selection.
  Tests: `connectiontest/connection_floating_ip_available_nova_test.go:TestAvailableIPNovaCleanListNotFoundAllocates`, `connectiontest/connection_floating_ip_available_test.go:TestConnectionAvailableIPNovaAllocationOnlyAndDefaultPool`.
- Nova availability reuses standalone Create's prepared-pool raw POST/compatibility engine. Independent AllocationResponse, Creation.Allocation/Compatibility/Failure and last known FloatingIP survive accepted and later failures. Compatibility identity/address/pool are passive rather than constrained by association equality/IPv4 verification. All phases share the original source registry/context and deadline; no accepted failure triggers reallocation or cleanup.
  Tests: `connectiontest/connection_floating_ip_available_nova_test.go:TestAvailableIPNovaFreshKeepsCreateProofAndBudget`, `connectiontest/connection_floating_ip_available_view_test.go:TestAvailableIPViewAcceptedFailuresPreserveRawReceipts`, `connectiontest/connection_floating_ip_available_view_test.go:TestAvailableIPViewLocationErrorKeepsKnownAllocation`.
- The Neutron cloud availability helper selects its external role network and nullable recorded/explicit project filter before consuming unfiltered public query inventory. It applies the source-ordered port_id=null, selected floating_network_id and project_id filter to Resource views, with canonical project_id winning tenant_id and null distinct from an empty normalized string. Reuse keeps optional Server resolution, port lookup and attachment lazy; mutating the returned record Wire does not change Inventory Wire.
  Tests: `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronInnerNovaReuseIgnoresInstanceAndServer`, `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronInnerProjectNullAndCanonicalAlias`.
- The helper consumes the shared public List engine before choosing a free result. Real Neutron Resource descriptor/view errors are processed before continuation; a matching prefix cannot survive an early descriptor failure or later HTTP failure. Internal Nova rows are all normalized before the Neutron local filter, so an unrelated later row with a missing required normalization ID is still terminal. Physical Inventory Pages/Failure and original HTTP error evidence survive, with no chosen candidate, allocation or outer fallback on these failures. Close, source replacement and custom-cancellation failures also stop before mutation.
  Tests: `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronListFailuresNeverAllocateMatchingPrefix`.
- An unfiltered Neutron-list404 first invokes public-list Nova fallback. Those rows use configured-Neutron normalization and then the same already selected Neutron network/project/port filter. A raw Nova row with non-null instance_id may be reused when its normalized port is null; actual Backend stays Nova while NormalizationSource is Neutron and synthetic status is UNKNOWN. Strict mode does not repair missing compatibility port_id from canonical port or Wire, and its local filter error retains inventory without mutation. Nova-list404 becomes empty public inventory and still proceeds to selected-network Neutron allocation.
  Tests: `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronInnerNovaReuseIgnoresInstanceAndServer`, `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronInnerStrictMissingAliasIsTerminal`, `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronInnerNovaNotFoundStillAllocatesNeutron`.
- No matching direct or internally-fallback inventory allocates fresh on the selected Neutron network through the shared raw allocation/Create-adoption engine. The explicit project is only a reuse filter and is not a POST override. A pure pre-accept Neutron allocation404 permits outer Nova availability, which performs a second raw inventory phase using the original literal pool, and may reuse or POST plus compatibility GET. Top-level Inventory/Creation preserve the earlier Neutron helper history independently of later Nova.Inventory/Creation and preserve the distinct inner-list and outer-allocation NotFound proofs. Outer and nested option factories run once, and all observed outer fallback requests retain one operation deadline.
  Tests: `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronInnerListFallbackAllocatesOnSelectedNetwork`, `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronInnerNovaNotFoundStillAllocatesNeutron`, `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronOuterFallbackRelistsOriginalPoolUnderOneBudget`.
- Optional Server/destination is consumed only for fresh allocation after inventory. The shared Create no-wait completion checks only port_id when the selected destination contains a port. A missing response port is seeded into Resource without inventing that field in Wire; explicit null or a wrong port is an accepted partial error. With no selected port, raw null identity/address, changed network, IPv6 fixed address, passive port and ERROR status remain successful passive results. Accepted203 allocation receipt, Selection, Wire and known result survive wrong-port, Close, source and custom-cancellation failures; no Nova fallback, ACTIVE wait, server action, observation or cleanup follows. Tested helper requests retain a single deadline.
  Tests: `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronFreshPortSeedsAndAcceptedFailures`, `connectiontest/connection_floating_ip_available_neutron_test.go:TestAvailableIPNeutronFreshWithoutPortKeepsPassiveResponse`.
- The concrete availability plan retains the selected external network through fresh allocation without another ordinary external-network lookup. Typed NATDestination preserves resource.Ref identity: a UUID-shaped Name uses exact-name lookup and excludes a nonmatching returned row, a non-UUID ID bypasses NAT network lookup, and a missing typed Name returns NotFound plus InvalidOption before POST. The selected destination body and accepted202 Wire/header receipt are inspected by the actual lower HTTP fixture. Application callers use the SDK-owned plan and resolver rather than implement a builder.
  Tests: `network/floating_ip_allocate_test.go:TestFloatingIPAvailabilityPlanPreservesTypedNATReferences`.

Python/Go differences:

- Pinned available_floating_ip accepts an optional existing Server and returns a free floating-IP Resource/Munch, allocating only when none is free. Compute/Connection provides the source-shaped cloud getter with an owned FloatingIPRecord and permissive raw null/extended fields; independent optional legacy backend models and string identity/address are convenience projections. Network.Available remains the separate typed Neutron-only lower receiver. Ensure verifies a server destination and prefers an already attached candidate, so its concrete Assignment remains a separate receiver contract.
- Network.Available uses unfiltered native inventory and raw-null local filters. Compute/Connection consumes shared public Query inventory with eager Neutron Resource defaults, aliases, descriptors and location before its ordered null-port/selected-network/nullable-project filter. Internal Nova-list fallback is fully normalized with configured-Neutron rules before that same local filter. Both reject prefix success after inventory errors; actual Wire and Pages/Failure remain distinct from Resource and the owned primary result. Ensure IPv4/ERROR exclusion, attached-first preference and mutation identity eligibility are not added to this cloud getter.
- The getter uses explicit ID/Name Refs and a caller-ordered network list against shared floating roles; zero selection uses the existing default-role/router policy. High network Create/Ensure keep their strict destination policy. Cloud Available fresh allocation shares Cloud Create raw destination selection: newest eligible NAT port and first IPv4, absent Server/no ports or unmatched explicit fixed address may allocate unattached, and a nonempty selected inventory without IPv4 is an error. Available allocation completion checks only an actually selected port, and passive response fields remain source-shaped. The added fixed-address option validates IPv4 before HTTP and empty clears it; Python private matching accepts literal fixed addresses more permissively. Typed NAT Name/ID semantics are maintained lazily through ResolveID rather than Python string nameORID coercion.
- Network.Available remains Neutron-only. Compute/Connection selects Nova directly for Nova or None; automatic None instead disables need. Unfiltered Neutron-list404 first yields configured-Neutron-normalized Nova public inventory for the Neutron filter, which may reuse an actual Nova row or still allocate freshly on selected Neutron. Only a pure pre-accept helper NotFound permits the separate outer Nova raw availability flow. Accepted response/source/decode/cancellation/multiple-cause or terminal SDK failure never triggers another allocation. Actual Backend, NormalizationSource, raw models and Wire remain distinct; synthetic Resource ACTIVE/UNKNOWN does not prove readiness.
- The getter resolves nullable recorded current-project ownership per call, and WithAvailableProject changes only its free-candidate filter; new Neutron allocation omits project_id and leaves ownership to authentication. Plain prepared Ensure resolves its reuse owner per call and service-bound PrepareEnsureActive binds that owner before Nova creation. These are different receiver policies, not a Python Resource/Adapter/session/cache or server-interface-refresh emulation.
- Direct/outer Nova uses raw instance_id-null then literal-pool filtering before all-selected-row normalization and the owned Query/Create engines. Neutron uses public unfiltered inventory and ordered Resource filters, including internal Nova fallback and selected-network fresh allocation. Top-level Inventory/Creation retain Neutron helper history, while Nova.Inventory/PoolQuery/Creation retain direct or outer Nova phases. Private helpers are dependencies without separate catalog IDs. Local source/HTTP evidence does not assert authenticated Python/OpenStack execution.
- PrepareEnsure/EnsurePrepared remain a separate Go assignment foundation with immutable selection, late reuse-owner binding, tuple revalidation and actual ACTIVE policy. Available returns a free or allocated backend model plus an owned cloud view, never associates a reused candidate and performs no IP readiness or raw Nova convergence wait. An automatic consumer does not need to call this getter to complete the standalone available_floating_ip operation.
- Stronger Ensure/ready/create keep their existing readiness/observation policy and AddIPsToServer keeps its independent async/whole60second policy. Available adds no SDK deadline or wait by default and accepts optional whole-operation timeout/unlimited controls. The additive view uses the original guarded operation context; accepted model/HTTP evidence and custom context causes survive projection failure without cleanup or replacement allocation. Context/receipt semantics and owned snapshots are explicit Go improvements.
- The getter accepts one typed literal Nova pool, ignores Server on Nova and sends no action or association recheck. Primary FloatingIPRecord preserves permissive null/raw fields and configured-source normalization, location and strict aliases. Nova.FloatingIP and common string ID/Address are optional best-effort independent projections; an unrepresentable passive read is not rejected. Raw inventory uses source key order, all selected rows are normalized, clean raw-list404 allocates, and a default null pool follows Create default lookup. Typed Ref/multiple-pool preflight, recorded scope and owned receipt/time/error policies remain deliberate Go mappings. Dynamic object coercion and full inherited session remain separate SDK scope.

### `gophercloud:networking/v2/extensions/layer3/floatingips.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.Create`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.Ensure`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.PrepareEnsure`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.EnsurePrepared`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPPlan.Selection`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithCreateField`

- The high-level fresh-allocation paths invoke the native concrete Create adapter and emit the floatingip JSON envelope with selected floating_network_id, port_id/fixed_ip_address and project_id when selected; 201 responses return the native typed FloatingIP. Reuse=false can omit project_id rather than guessing the owner.
  Tests: `network/floating_ip_ensure_test.go:TestFloatingIPEnsurePrefersAlreadyAttachedAndAllocatesOnlyWhenNeeded`, `connectiontest/connection_floating_ip_ensure_test.go:TestConnectionFloatingIPEnsureWithoutReuseLetsNeutronChooseProject`.
- The existing fresh Create workflow can allocate without port association or network-ID lookup, sends a requested floating IPv4 address, and snapshots an extension body containing explicit false across option reuse.
  Tests: `network/floating_ip_test.go:TestFloatingIPAllocatesWithoutAssociationAndSnapshotsExtensions`.
- The fresh Create workflow preserves original 403/404/409 causes without alternate allocation or API fallback. Ensure retains an allocated assignment on destination-verification, wait timeout or cancellation failure and does not issue implicit cleanup.
  Tests: `network/floating_ip_test.go:TestFloatingIPHTTPFailureDoesNotFallbackOrReuse`, `network/floating_ip_ensure_test.go:TestFloatingIPEnsureErrorsPreserveSelectionAndNeverFallback`, `network/floating_ip_selection_test.go:TestFloatingIPEnsureWaitCancellationAndTimeoutPreserveAssignment`.
- High-level FloatingIPs.Create and Ensure consume shared configured or inferred NAT destination only without explicit port/fixed/NAT selection and with multiple ports owned by the requested server. All port pages are consumed first and foreign ports are excluded. Successful NAT discovery is reused but ports are listed on each call; Reset changes the chosen destination while preserving the actual allocation body.
  Tests: `network/floating_ip_roles_test.go:TestFloatingIPAutomaticNATUsesSharedDestinationAndReset`.
- Owned port count precedes IPv4 filtering, so an IPv6-only second port requires a NAT role. No NAT role returns ErrAmbiguous with sorted owned port IDs; a selected network absent from ports returns ErrNotFound; multiple selected IPv4 pairs remain ErrAmbiguous. Role403, later port403 and subnet cancellation prevent allocation without fallback or cleanup.
  Tests: `network/floating_ip_roles_test.go:TestFloatingIPAutomaticNATFailuresStopAllocation`.
- A single owned port or explicit fixed address, port or NAT destination bypasses automatic NAT role discovery despite an invalid configured policy elsewhere. Explicit port performs its GET rather than port listing; explicit NAT retains network_id filtering. An external Name retains the router:external/exact-name lane, while an explicit external ID needs no network-role inventory.
  Tests: `network/floating_ip_roles_test.go:TestFloatingIPExplicitAndSinglePortSelectionBypassRoles`.
- Connection CreateWithFloatingIP uses a single successful configured role inventory for its default NIC, floating source and NAT destination, including a configured source without router:external. Role lookup finishes before Nova POST; actual server ACTIVE precedes port and floating-IP requests, and actual floating-IP ACTIVE remains mandatory. A role403 returns no created resource; a later port403 preserves the known ACTIVE Server with nil assignment and the original HTTP cause, without allocation or cleanup.
  Tests: `connectiontest/connection_server_network_roles_test.go:TestConnectionCreateWithFloatingIPSharesDefaultSourceAndNATRoles`.
- The independent prepared-plan allocation lane accepts native POST201/202 and retains decoded allocated resources on owner invariant failure. Malformed bodies retain Allocated=true/nil model and actual response proof; complete accepted201 Close/source/cancellation failures now retain the decoded known IP with the original processing cause. These fatal accepted errors never trigger retry, further wait or deletion. This owned REST lane does not change generated native API.Create transport.
  Tests: `network/floating_ip_plan_boundaries_test.go:TestFloatingIPPlanPOSTCodesAndInvariantFailuresKeepAcceptedResources`, `network/floating_ip_plan_test.go:TestFloatingIPPlanAcceptedAllocationFailuresKeepEvidenceAndNoRetry`, `network/floating_ip_plan_test.go:TestFloatingIPPlanDefersOwnerAndPreservesConcreteSelection`.
- The SDK-owned prepared Neutron plan retains a decoded accepted201/202 allocation after complete-body read, Close, source or custom-cancellation errors with actual status/header/body and original cause; no retry, ACTIVE wait, observer or cleanup follows. Decoded wrong-owner allocation remains evidence alongside joined invariant/Close failures. Truncated202 read and Close causes remain joined with the actual json.SyntaxError and Allocated=true/nil model. Reused PUT200 adopts a complete matching body beside Close failure, but wrong-ID/fixed responses preserve the original candidate and both processing and validation causes; revision zero remains protected. This is owned workflow evidence, not a change to generated native transport.
  Tests: `network/floating_ip_accepted_model_test.go:TestFloatingIPAcceptedAllocationKeepsDecodedModelAndProcessingCauses`, `network/floating_ip_accepted_model_test.go:TestFloatingIPAcceptedAssociationAdoptsOnlyMatchingModelOnCloseError`, `network/floating_ip_accepted_model_test.go:TestFloatingIPAcceptedReadAndDecodeErrorsRemainJoined`.
- The generated native API.Create POSTs {"floatingip":{...}} to floatingips requiring floating_network_id, omitting empty Description/FloatingIP/PortID/FixedIP/SubnetID/TenantID/ProjectID, letting WithCreateOptions replace positional options, placing extensions inside the envelope and rejecting core-key collisions or nil options before HTTP; it accepts only 201/202, decodes both timestamp formats, null port/fixed/router as empty strings and ignores unknown fields, and wraps status, invalid JSON and non-object envelope errors with operation context.
  Tests: `network/v2/extensions/layer3/floatingips/native_contracts_test.go:TestNativeFloatingIPRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/floatingips/native_contracts_test.go:TestNativeFloatingIPStrictStatusesAndPreflight`, `network/v2/extensions/layer3/floatingips/native_contracts_test.go:TestNativeFloatingIPCreateFullOptionsReplacementAndDecode`.

Python/Go differences:

- The generated facade owns concrete options/builders and extracts a native typed value. Higher-level Create/Ensure add safe Ref/name selection, unique IPv4 destination policy, optional wait and partial allocation results around that native call; these workflow differences do not characterize all native CreateOptsBuilder inputs.
- Ensure adds reuse and presence-aware native Update revision handling without changing the separately reviewed native Update declaration. No unconditional conflict fallback, new allocation after association failure, or automatic cleanup is introduced.
- Selected high-level Create/Ensure allocation fixtures now use the shared NAT role for the bounded automatic multi-owned-port case and preserve explicit/single-port lanes. The generated native API.Create still accepts supplied concrete options; these role-consuming fixtures do not establish its complete input/body/accepted-status/decoder/session contract.
- The separate prepared-plan workflow uses owned guarded REST for exact selected targets and accepted-response evidence, including201/202 allocation proof and partial decoded models. Existing fresh Create/Ensure and the generated native API.Create keep their distinct lanes; no whole native transport/decoder parity is inferred from plan fixtures.
- The native API.Create contract is now audited directly; live auth, retry and redirect behavior remain provider-owned and unchanged by the facade.

### `python:cloud/_network_common/NetworkCommonCloudMixin/get_nat_destination`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetNATDestination`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ResetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.WithNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Discover`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Reset`
- `github.com/JSYoo5B/go-openstacksdk/network.PrepareNetworkRoleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network.WithConfiguredNetworks`
- `github.com/JSYoo5B/go-openstacksdk/network.WithExternalNetworkDiscovery`
- `github.com/JSYoo5B/go-openstacksdk/network.WithInternalNetworkDiscovery`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.AttachByID`
- `github.com/JSYoo5B/go-openstacksdk.WithServiceLookupFailureAsUnavailable`

- Shared discovery classifies IPv4 with router:external or provider:physical_network and IPv6 with router:external only, unless configured name-or-ID family overrides apply. Floating IPv4 candidates are router-external networks or the first configured NAT source; configured source/destination/default roles remain distinct from ordinary family roles.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- An explicit NAT destination skips subnet discovery. Without one, all subnet pages are consumed and the last network in network response order with a nonempty gateway subnet becomes the NAT destination, independent of subnet ip_version. Configured NAT/default name-or-ID selection rejects missing, duplicate or cross-kind ambiguous matches; the first configured NAT source is used.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- Unfiltered network discovery and conditional subnet discovery consume linked empty and later pages. Unsafe network IDs, invalid router extension decoding, later network403, subnet403 and repeated continuation links return errors without a partial snapshot; failed attempts remain retryable. Native204 returns an empty successful discovery.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- All ten Connection role getters share one successful discovery and return independent owned native-plus-extension models. Aggregate getters concatenate IPv4 then IPv6 with duplicate entries retained. Caller changes do not modify cached roles, and tags/subnets/availability-zone-hints retain explicit empty versus null native array presence.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`.
- Concurrent uncached discoveries share one in-flight request and return independent models. A waiting follower can reach its own context deadline without canceling the leader. Reset advances the cache generation so a successful already-running discovery cannot repopulate the cache, while the existing caller still receives its result; subsequent concurrent cached calls issue no new request.
  Tests: `network/roles_test.go:TestNetworkRoleConcurrentDiscoveriesShareFlightAndOwnResults`, `network/roles_test.go:TestNetworkRoleCacheFlightCancellationAndReset`.
- Typed use-external/use-internal flags default enabled and preserve ordered replacement. Both disabled returns empty roles before client/catalog/HTTP access; disabling only external still discovers both families when internal is enabled. Connection nil/canceled context fails before lookup, clean pointer or value ErrEndpointNotFound yields empty roles, and other catalog errors remain inspectable.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- Role selectors are frozen with the same cloud/secure/public snapshot as native authentication. YAML family flags inherit routes_externally unless overridden; a typed WithNetworkRoles policy replaces the YAML role configuration and its default selector, and an empty typed replacement clears configured roles/default. Configured default NIC selection now consumes the same successful role snapshot; explicit typed default names retain their separate exact-name lookup.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Owned configured-network slices can be reused after caller mutation. Nil options, empty names and multiple default/NAT-destination declarations fail before discovery; a later replacement does not hide an earlier invalid option. Explicit Reset invalidates a completed Connection role cache without making a mutation request.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- YAML boolean false and case-insensitive string false disable discovery before network HTTP. An explicit typed policy replaces those flags and restores discovery; a numeric YAML flag is rejected as ErrInvalidOption. Direct role discovery rejects nil or providerless clients before native HTTP.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleBooleanFlagsAndExplicitPolicyReplacement`, `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`.
- High-level FloatingIPs.Create and Ensure consume shared configured or inferred NAT destination only without explicit port/fixed/NAT selection and with multiple ports owned by the requested server. All port pages are consumed first and foreign ports are excluded. Successful NAT discovery is reused but ports are listed on each call; Reset changes the chosen destination while preserving the actual allocation body.
  Tests: `network/floating_ip_roles_test.go:TestFloatingIPAutomaticNATUsesSharedDestinationAndReset`.
- Owned port count precedes IPv4 filtering, so an IPv6-only second port requires a NAT role. No NAT role returns ErrAmbiguous with sorted owned port IDs; a selected network absent from ports returns ErrNotFound; multiple selected IPv4 pairs remain ErrAmbiguous. Role403, later port403 and subnet cancellation prevent allocation without fallback or cleanup.
  Tests: `network/floating_ip_roles_test.go:TestFloatingIPAutomaticNATFailuresStopAllocation`.
- A single owned port or explicit fixed address, port or NAT destination bypasses automatic NAT role discovery despite an invalid configured policy elsewhere. Explicit port performs its GET rather than port listing; explicit NAT retains network_id filtering. An external Name retains the router:external/exact-name lane, while an explicit external ID needs no network-role inventory.
  Tests: `network/floating_ip_roles_test.go:TestFloatingIPExplicitAndSinglePortSelectionBypassRoles`.
- Connection CreateWithFloatingIP uses a single successful configured role inventory for its default NIC, floating source and NAT destination, including a configured source without router:external. Role lookup finishes before Nova POST; actual server ACTIVE precedes port and floating-IP requests, and actual floating-IP ACTIVE remains mandatory. A role403 returns no created resource; a later port403 preserves the known ACTIVE Server with nil assignment and the original HTTP cause, without allocation or cleanup.
  Tests: `connectiontest/connection_server_network_roles_test.go:TestConnectionCreateWithFloatingIPSharesDefaultSourceAndNATRoles`.
- Owned high Service CreateNetwork/UpdateNetwork/DeleteNetwork and the existing Networks.Delete adapter invalidate shared roles after accepted mutations. The actual matrix primes a configured default role, observes a new inventory after success or high-delete clean404 after a found lookup, and retains the original inventory after403/409/412, failed no-option lookup, or direct Collection404. A successful no-option Update performs its lookup and reset without PUT; initial missing ID/Name high Delete returns false without mutation or reset.
  Tests: `network/network_mutations_test.go:TestNetworkMutationSuccessFailureAndMissingCacheBoundaries`, `network/network_mutations_test.go:TestDeleteNetworkInitialMissingDoesNotMutateOrReset`.
- Accepted mutation read/Close/context/source or decode/ID errors preserve the actual response evidence and invalidate the originally owned role cache without another mutation caused by that error. Create/Update malformed/null/missing or wrong-shaped network bodies return an inspectable ResponseError; decoded unsafe or mismatched Update IDs retain the partial model. Delete keeps true plus an accepted response error. Public role/collection replacements during accepted Close are errors but cannot redirect the cache reset to a replacement.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedDecodeAndIdentityErrorsResetWithoutResend`, `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedIOCancellationAndSourceErrorsKeepEvidenceAndOwnedReset`.
- A representative accepted Create resets while an earlier role inventory is in flight: that earlier caller still receives its original result, but the old generation cannot repopulate cache. The next configured default lookup reads a new inventory and repeated lookup uses that new cache. This is the actual Create/Reset fixture, not separate mutation-race coverage for every CRUD method or every getter.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationResetCannotPublishEarlierInflightDiscovery`.
- The Connection-cached high Network Service resets the same role cache after Create, Update and Delete. A later getter reads a new configured inventory, and Nova default NIC plus floating source/NAT allocation consume that same refreshed inventory without manual Reset. The fixture checks actual Nova and Neutron request fields and exactly two inventory reads.
  Tests: `connectiontest/connection_network_mutations_test.go:TestConnectionNetworkMutationRefreshesGetterDefaultNICAndFloatingRoles`.
- A has_network flag from WithServiceEnabled or clouds.yaml mirrors the has_service('network') check of _find_interesting_networks: disabled roles are empty without a network list even with configured roles, a typed enable overrides a YAML disable, and an enabled flag keeps discovery and configured NAT/default selection.
  Tests: `connectiontest/connection_network_service_disable_test.go:TestNetworkRolesHonourHasNetworkConfiguration`.
- The Neutron attach of a Nova fallback row narrows several server ports to the discovered NAT destination network, selects the newest port by created_at and its first IPv4 fixed address (skipping IPv6), and fails before mutation when several ports have no NAT destination.
  Tests: `connectiontest/connection_cloud_net_auto_ip_fallback_test.go:TestCloudNetAddAutoIPReuseNovaFallbackRowUsesNeutronAttachAndIgnoresNotFound`, `connectiontest/connection_cloud_net_auto_ip_fallback_test.go:TestCloudNetAddAutoIPReuseNovaFallbackRowAttachOutcomes`.
- The typed YAML use_external_network/use_internal_network policy agrees with Python raw-value truthiness for booleans, null, the empty string and any casing of 'true'; other strings disable discovery in Go though truthy in Python, and numbers or collections are configuration errors in Go while Python's normalize_keys makes numbers (even 0) truthy strings, as a table test records.
  Tests: `connectiontest/connection_network_role_flags_test.go:TestNetworkRoleUseFlagsCompareYAMLValuesWithPythonTruthiness`.
- The selected clouds.yaml regions entry overlays its non-null values on the merged cloud settings before role, default-network and has_<service> configuration is read: whole networks lists, use_* and has_network flags change per region. Without WithRegion the first entry (or the one named by region_name) is used; an unknown explicit region and malformed entries are configuration errors.
  Tests: `connectiontest/connection_cloud_region_config_test.go:TestCloudRegionValuesOverlayNetworkRoleConfiguration`.
- A profile absent from clouds-public.yaml (or with no such file) resolves to the embedded, unmodified pinned vendor profile: rackspace disables has_network and sets secgroup_source under the cloud's own settings, limestonenetworks supplies per-region networks and default interface, and native authentication receives the profile's auth settings. A clouds-public.yaml entry wins; shutdown and URL profiles are configuration errors, deprecated profiles load without warnings, and embedded files match their documented hashes.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudBuiltinVendorProfilesFeedRoleConfiguration`, `internal/cloudvendors/vendors_test.go:TestProfilesResolveStatusNestedAndRemoteNames`, `internal/cloudvendors/vendors_test.go:TestEmbeddedFilesMatchREADMEHashes`.
- Without OS_CLOUD, Python's environment-variable cloud is mapped for role settings: OS_INTERNAL_NETWORK/OS_EXTERNAL_NETWORK legacy roles, OS_USE_* flags (typed string policy) and an OS_PROFILE built-in vendor profile shape discovery, string OS_HAS_* values and shutdown profiles are configuration errors, and a typed WithNetworkRoles policy still replaces the environment roles.
  Tests: `connectiontest/connection_cloud_env_config_test.go:TestEnvironmentCloudSuppliesNetworkRoleConfiguration`.
- WithServiceLookupFailureAsUnavailable maps has_service returning False when the Network endpoint lookup raises: roles are empty and the lookup is retried on the next call, while the default returns the lookup error and cancellation is returned under either policy.
  Tests: `connectiontest/connection_service_lookup_test.go:TestServiceLookupFailureAsUnavailableMapsHasServiceFalse`.

Python/Go differences:

- The SDK exposes context-aware Connection getters returning owned native-plus-router/provider models and typed nullable pointers; Python returns mutable Network Resources and cached list/object identities. Ordinary generated Networks model behavior is unchanged. This bounded role model does not emulate all nullable descriptors, unknown properties, location or session state.
- Go configured-role validation accepts a matching name OR ID consistently. Pinned classification accepts either, but its subsequent external/internal family validation checks only selected names and can reject ID-only selectors. Go also rejects unsafe IDs and preserves missing/ambiguous and decoding errors rather than incidental dynamic-input behavior.
- Go caches only successful discovery, coalesces in-flight work, lets followers use their own contexts, and uses Reset generations to prevent a pre-reset flight from restoring the cache. Python serializes shared discovery with a lock and can swallow SDKException into sticky empty results. Owned returns and observable/retryable errors are deliberate Go policies.
- Aggregate results retain the source IPv4-then-IPv6 order and duplicates; family classification reflects routing configuration and network extensions, not discovered subnet address families. The NAT destination inference preserves the source last-network-with-gateway rule.
- Go flags are strict typed booleans. The YAML parser agrees with Python truthiness of the raw value for booleans, null, the empty string and any casing of 'true'; other strings are false in Go but truthy in Python, and numeric or collection values are rejected instead of being truthiness-tested (Python's normalize_keys turns numbers, even 0, into truthy strings). Pinned cloud use_* accessors return the raw configured value; Go returns a bool. Both-disabled skips discovery; a single disabled flag does not remove that family after discovery.
- The Connection maps clean native missing-endpoint detection and the has_network configuration flag with official-service defaults (ServiceEnabled); Python's False for other lookup failures needs WithServiceLookupFailureAsUnavailable, and no version discovery applies to role discovery. Configured default NIC and automatic Ensure floating source use the shared successful role snapshot. Automatic NAT applies only without explicit port/fixed/NAT options and with multiple server-owned ports; strict unique IPv4-pair selection remains. These consumers do not implement private/floating-source flags or automatic IP need/skip decisions.
- Region values follow get_one's overlay for SDK-owned settings only; native authentication and endpoint parsing do not see them. Without WithRegion, an entry named by region_name is preferred over the first entry so that the values match the native endpoint region, whereas Python ignores region_name when regions is present. Typed Connect options (WithNetworkRoles, WithServiceEnabled) replace the configured policy for their concern instead of overlaying single keys like get_one keyword arguments.
- has_service('network') is mapped by ServiceEnabled (has_network flag, official-service default) plus the endpoint lookup. Python turns every lookup exception into False; Go returns lookup failures other than a missing endpoint unless WithServiceLookupFailureAsUnavailable is set, and always returns cancellation. Role discovery passes no version, so no version-range discovery applies, and Go emits no Python debug log.
- Configuration sources: clouds.yaml with secure.yaml, clouds-public.yaml or embedded built-in vendor profiles, region values, and the OS_* environment cloud when OS_CLOUD is unset. The environment cloud feeds only SDK-owned role, default-network and service settings; native authentication still uses Gophercloud's environment parser, and Python argparse/keyword arguments map to typed Connect options.
- Results are immutable owned snapshots rather than Python's mutable Resource with adapter, session, cache and dirty state (SDK-R1/C1/S1); see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Inputs are typed Go values; arbitrary Python objects, iterables and numeric types accepted by dynamic kwargs have no equivalent; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Network roles are cached per Connection and reset by the SDK's own CreateNetwork, UpdateNetwork and DeleteNetwork, as Python resets its role cache only inside cloud create_network, update_network and delete_network; changes made by other clients or through the raw NetworkV2 client need an explicit Reset, which Python does not see either.
- A clouds.yaml profile named by URL, including the nested URLs of the built-in ovh and vexxhost profiles, is rejected as a configuration error instead of being fetched from the vendor's well-known document at connect time; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- The consumers of the role snapshot (create_server, add_ips_to_server, get_active_server, wait_for_server) are reviewed under those operations; this operation covers discovery and the returned role.

### `python:cloud/_network_common/NetworkCommonCloudMixin/get_nat_source`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetNATSource`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ResetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.WithNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Discover`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Reset`
- `github.com/JSYoo5B/go-openstacksdk/network.PrepareNetworkRoleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network.WithConfiguredNetworks`
- `github.com/JSYoo5B/go-openstacksdk/network.WithExternalNetworkDiscovery`
- `github.com/JSYoo5B/go-openstacksdk/network.WithInternalNetworkDiscovery`
- `github.com/JSYoo5B/go-openstacksdk.WithServiceLookupFailureAsUnavailable`

- Shared discovery classifies IPv4 with router:external or provider:physical_network and IPv6 with router:external only, unless configured name-or-ID family overrides apply. Floating IPv4 candidates are router-external networks or the first configured NAT source; configured source/destination/default roles remain distinct from ordinary family roles.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- An explicit NAT destination skips subnet discovery. Without one, all subnet pages are consumed and the last network in network response order with a nonempty gateway subnet becomes the NAT destination, independent of subnet ip_version. Configured NAT/default name-or-ID selection rejects missing, duplicate or cross-kind ambiguous matches; the first configured NAT source is used.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- Unfiltered network discovery and conditional subnet discovery consume linked empty and later pages. Unsafe network IDs, invalid router extension decoding, later network403, subnet403 and repeated continuation links return errors without a partial snapshot; failed attempts remain retryable. Native204 returns an empty successful discovery.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- All ten Connection role getters share one successful discovery and return independent owned native-plus-extension models. Aggregate getters concatenate IPv4 then IPv6 with duplicate entries retained. Caller changes do not modify cached roles, and tags/subnets/availability-zone-hints retain explicit empty versus null native array presence.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`.
- Concurrent uncached discoveries share one in-flight request and return independent models. A waiting follower can reach its own context deadline without canceling the leader. Reset advances the cache generation so a successful already-running discovery cannot repopulate the cache, while the existing caller still receives its result; subsequent concurrent cached calls issue no new request.
  Tests: `network/roles_test.go:TestNetworkRoleConcurrentDiscoveriesShareFlightAndOwnResults`, `network/roles_test.go:TestNetworkRoleCacheFlightCancellationAndReset`.
- Typed use-external/use-internal flags default enabled and preserve ordered replacement. Both disabled returns empty roles before client/catalog/HTTP access; disabling only external still discovers both families when internal is enabled. Connection nil/canceled context fails before lookup, clean pointer or value ErrEndpointNotFound yields empty roles, and other catalog errors remain inspectable.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- Role selectors are frozen with the same cloud/secure/public snapshot as native authentication. YAML family flags inherit routes_externally unless overridden; a typed WithNetworkRoles policy replaces the YAML role configuration and its default selector, and an empty typed replacement clears configured roles/default. Configured default NIC selection now consumes the same successful role snapshot; explicit typed default names retain their separate exact-name lookup.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Owned configured-network slices can be reused after caller mutation. Nil options, empty names and multiple default/NAT-destination declarations fail before discovery; a later replacement does not hide an earlier invalid option. Explicit Reset invalidates a completed Connection role cache without making a mutation request.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- YAML boolean false and case-insensitive string false disable discovery before network HTTP. An explicit typed policy replaces those flags and restores discovery; a numeric YAML flag is rejected as ErrInvalidOption. Direct role discovery rejects nil or providerless clients before native HTTP.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleBooleanFlagsAndExplicitPolicyReplacement`, `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`.
- Owned high Service CreateNetwork/UpdateNetwork/DeleteNetwork and the existing Networks.Delete adapter invalidate shared roles after accepted mutations. The actual matrix primes a configured default role, observes a new inventory after success or high-delete clean404 after a found lookup, and retains the original inventory after403/409/412, failed no-option lookup, or direct Collection404. A successful no-option Update performs its lookup and reset without PUT; initial missing ID/Name high Delete returns false without mutation or reset.
  Tests: `network/network_mutations_test.go:TestNetworkMutationSuccessFailureAndMissingCacheBoundaries`, `network/network_mutations_test.go:TestDeleteNetworkInitialMissingDoesNotMutateOrReset`.
- Accepted mutation read/Close/context/source or decode/ID errors preserve the actual response evidence and invalidate the originally owned role cache without another mutation caused by that error. Create/Update malformed/null/missing or wrong-shaped network bodies return an inspectable ResponseError; decoded unsafe or mismatched Update IDs retain the partial model. Delete keeps true plus an accepted response error. Public role/collection replacements during accepted Close are errors but cannot redirect the cache reset to a replacement.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedDecodeAndIdentityErrorsResetWithoutResend`, `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedIOCancellationAndSourceErrorsKeepEvidenceAndOwnedReset`.
- A representative accepted Create resets while an earlier role inventory is in flight: that earlier caller still receives its original result, but the old generation cannot repopulate cache. The next configured default lookup reads a new inventory and repeated lookup uses that new cache. This is the actual Create/Reset fixture, not separate mutation-race coverage for every CRUD method or every getter.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationResetCannotPublishEarlierInflightDiscovery`.
- The Connection-cached high Network Service resets the same role cache after Create, Update and Delete. A later getter reads a new configured inventory, and Nova default NIC plus floating source/NAT allocation consume that same refreshed inventory without manual Reset. The fixture checks actual Nova and Neutron request fields and exactly two inventory reads.
  Tests: `connectiontest/connection_network_mutations_test.go:TestConnectionNetworkMutationRefreshesGetterDefaultNICAndFloatingRoles`.
- A has_network flag from WithServiceEnabled or clouds.yaml mirrors the has_service('network') check of _find_interesting_networks: disabled roles are empty without a network list even with configured roles, a typed enable overrides a YAML disable, and an enabled flag keeps discovery and configured NAT/default selection.
  Tests: `connectiontest/connection_network_service_disable_test.go:TestNetworkRolesHonourHasNetworkConfiguration`.
- The typed YAML use_external_network/use_internal_network policy agrees with Python raw-value truthiness for booleans, null, the empty string and any casing of 'true'; other strings disable discovery in Go though truthy in Python, and numbers or collections are configuration errors in Go while Python's normalize_keys makes numbers (even 0) truthy strings, as a table test records.
  Tests: `connectiontest/connection_network_role_flags_test.go:TestNetworkRoleUseFlagsCompareYAMLValuesWithPythonTruthiness`.
- The selected clouds.yaml regions entry overlays its non-null values on the merged cloud settings before role, default-network and has_<service> configuration is read: whole networks lists, use_* and has_network flags change per region. Without WithRegion the first entry (or the one named by region_name) is used; an unknown explicit region and malformed entries are configuration errors.
  Tests: `connectiontest/connection_cloud_region_config_test.go:TestCloudRegionValuesOverlayNetworkRoleConfiguration`.
- A profile absent from clouds-public.yaml (or with no such file) resolves to the embedded, unmodified pinned vendor profile: rackspace disables has_network and sets secgroup_source under the cloud's own settings, limestonenetworks supplies per-region networks and default interface, and native authentication receives the profile's auth settings. A clouds-public.yaml entry wins; shutdown and URL profiles are configuration errors, deprecated profiles load without warnings, and embedded files match their documented hashes.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudBuiltinVendorProfilesFeedRoleConfiguration`, `internal/cloudvendors/vendors_test.go:TestProfilesResolveStatusNestedAndRemoteNames`, `internal/cloudvendors/vendors_test.go:TestEmbeddedFilesMatchREADMEHashes`.
- Without OS_CLOUD, Python's environment-variable cloud is mapped for role settings: OS_INTERNAL_NETWORK/OS_EXTERNAL_NETWORK legacy roles, OS_USE_* flags (typed string policy) and an OS_PROFILE built-in vendor profile shape discovery, string OS_HAS_* values and shutdown profiles are configuration errors, and a typed WithNetworkRoles policy still replaces the environment roles.
  Tests: `connectiontest/connection_cloud_env_config_test.go:TestEnvironmentCloudSuppliesNetworkRoleConfiguration`.
- WithServiceLookupFailureAsUnavailable maps has_service returning False when the Network endpoint lookup raises: roles are empty and the lookup is retried on the next call, while the default returns the lookup error and cancellation is returned under either policy.
  Tests: `connectiontest/connection_service_lookup_test.go:TestServiceLookupFailureAsUnavailableMapsHasServiceFalse`.

Python/Go differences:

- The SDK exposes context-aware Connection getters returning owned native-plus-router/provider models and typed nullable pointers; Python returns mutable Network Resources and cached list/object identities. Ordinary generated Networks model behavior is unchanged. This bounded role model does not emulate all nullable descriptors, unknown properties, location or session state.
- Go configured-role validation accepts a matching name OR ID consistently. Pinned classification accepts either, but its subsequent external/internal family validation checks only selected names and can reject ID-only selectors. Go also rejects unsafe IDs and preserves missing/ambiguous and decoding errors rather than incidental dynamic-input behavior.
- Go caches only successful discovery, coalesces in-flight work, lets followers use their own contexts, and uses Reset generations to prevent a pre-reset flight from restoring the cache. Python serializes shared discovery with a lock and can swallow SDKException into sticky empty results. Owned returns and observable/retryable errors are deliberate Go policies.
- Aggregate results retain the source IPv4-then-IPv6 order and duplicates; family classification reflects routing configuration and network extensions, not discovered subnet address families. The NAT destination inference preserves the source last-network-with-gateway rule.
- Go flags are strict typed booleans. The YAML parser agrees with Python truthiness of the raw value for booleans, null, the empty string and any casing of 'true'; other strings are false in Go but truthy in Python, and numeric or collection values are rejected instead of being truthiness-tested (Python's normalize_keys turns numbers, even 0, into truthy strings). Pinned cloud use_* accessors return the raw configured value; Go returns a bool. Both-disabled skips discovery; a single disabled flag does not remove that family after discovery.
- The Connection maps clean native missing-endpoint detection and the has_network configuration flag with official-service defaults (ServiceEnabled); Python's False for other lookup failures needs WithServiceLookupFailureAsUnavailable, and no version discovery applies to role discovery. Configured default NIC and automatic Ensure floating source use the shared successful role snapshot. Automatic NAT applies only without explicit port/fixed/NAT options and with multiple server-owned ports; strict unique IPv4-pair selection remains. These consumers do not implement private/floating-source flags or automatic IP need/skip decisions.
- Region values follow get_one's overlay for SDK-owned settings only; native authentication and endpoint parsing do not see them. Without WithRegion, an entry named by region_name is preferred over the first entry so that the values match the native endpoint region, whereas Python ignores region_name when regions is present. Typed Connect options (WithNetworkRoles, WithServiceEnabled) replace the configured policy for their concern instead of overlaying single keys like get_one keyword arguments.
- has_service('network') is mapped by ServiceEnabled (has_network flag, official-service default) plus the endpoint lookup. Python turns every lookup exception into False; Go returns lookup failures other than a missing endpoint unless WithServiceLookupFailureAsUnavailable is set, and always returns cancellation. Role discovery passes no version, so no version-range discovery applies, and Go emits no Python debug log.
- Configuration sources: clouds.yaml with secure.yaml, clouds-public.yaml or embedded built-in vendor profiles, region values, and the OS_* environment cloud when OS_CLOUD is unset. The environment cloud feeds only SDK-owned role, default-network and service settings; native authentication still uses Gophercloud's environment parser, and Python argparse/keyword arguments map to typed Connect options.
- Results are immutable owned snapshots rather than Python's mutable Resource with adapter, session, cache and dirty state (SDK-R1/C1/S1); see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Inputs are typed Go values; arbitrary Python objects, iterables and numeric types accepted by dynamic kwargs have no equivalent; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- Network roles are cached per Connection and reset by the SDK's own CreateNetwork, UpdateNetwork and DeleteNetwork, as Python resets its role cache only inside cloud create_network, update_network and delete_network; changes made by other clients or through the raw NetworkV2 client need an explicit Reset, which Python does not see either.
- A clouds.yaml profile named by URL, including the nested URLs of the built-in ovh and vexxhost profiles, is rejected as a configuration error instead of being fetched from the vendor's well-known document at connect time; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- The consumers of the role snapshot (create_server, add_ips_to_server, get_active_server, wait_for_server) are reviewed under those operations; this operation covers discovery and the returned role.

### `python:cloud/_network_common/NetworkCommonCloudMixin/get_external_networks`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetExternalNetworks`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ResetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.WithNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Discover`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Reset`
- `github.com/JSYoo5B/go-openstacksdk/network.PrepareNetworkRoleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network.WithConfiguredNetworks`
- `github.com/JSYoo5B/go-openstacksdk/network.WithExternalNetworkDiscovery`
- `github.com/JSYoo5B/go-openstacksdk/network.WithInternalNetworkDiscovery`

- Shared discovery classifies IPv4 with router:external or provider:physical_network and IPv6 with router:external only, unless configured name-or-ID family overrides apply. Floating IPv4 candidates are router-external networks or the first configured NAT source; configured source/destination/default roles remain distinct from ordinary family roles.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- An explicit NAT destination skips subnet discovery. Without one, all subnet pages are consumed and the last network in network response order with a nonempty gateway subnet becomes the NAT destination, independent of subnet ip_version. Configured NAT/default name-or-ID selection rejects missing, duplicate or cross-kind ambiguous matches; the first configured NAT source is used.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- Unfiltered network discovery and conditional subnet discovery consume linked empty and later pages. Unsafe network IDs, invalid router extension decoding, later network403, subnet403 and repeated continuation links return errors without a partial snapshot; failed attempts remain retryable. Native204 returns an empty successful discovery.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- All ten Connection role getters share one successful discovery and return independent owned native-plus-extension models. Aggregate getters concatenate IPv4 then IPv6 with duplicate entries retained. Caller changes do not modify cached roles, and tags/subnets/availability-zone-hints retain explicit empty versus null native array presence.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`.
- Concurrent uncached discoveries share one in-flight request and return independent models. A waiting follower can reach its own context deadline without canceling the leader. Reset advances the cache generation so a successful already-running discovery cannot repopulate the cache, while the existing caller still receives its result; subsequent concurrent cached calls issue no new request.
  Tests: `network/roles_test.go:TestNetworkRoleConcurrentDiscoveriesShareFlightAndOwnResults`, `network/roles_test.go:TestNetworkRoleCacheFlightCancellationAndReset`.
- Typed use-external/use-internal flags default enabled and preserve ordered replacement. Both disabled returns empty roles before client/catalog/HTTP access; disabling only external still discovers both families when internal is enabled. Connection nil/canceled context fails before lookup, clean pointer or value ErrEndpointNotFound yields empty roles, and other catalog errors remain inspectable.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- Role selectors are frozen with the same cloud/secure/public snapshot as native authentication. YAML family flags inherit routes_externally unless overridden; a typed WithNetworkRoles policy replaces the YAML role configuration and its default selector, and an empty typed replacement clears configured roles/default. Configured default NIC selection now consumes the same successful role snapshot; explicit typed default names retain their separate exact-name lookup.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Owned configured-network slices can be reused after caller mutation. Nil options, empty names and multiple default/NAT-destination declarations fail before discovery; a later replacement does not hide an earlier invalid option. Explicit Reset invalidates a completed Connection role cache without making a mutation request.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- YAML boolean false and case-insensitive string false disable discovery before network HTTP. An explicit typed policy replaces those flags and restores discovery; a numeric YAML flag is rejected as ErrInvalidOption. Direct role discovery rejects nil or providerless clients before native HTTP.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleBooleanFlagsAndExplicitPolicyReplacement`, `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`.
- Owned high Service CreateNetwork/UpdateNetwork/DeleteNetwork and the existing Networks.Delete adapter invalidate shared roles after accepted mutations. The actual matrix primes a configured default role, observes a new inventory after success or high-delete clean404 after a found lookup, and retains the original inventory after403/409/412, failed no-option lookup, or direct Collection404. A successful no-option Update performs its lookup and reset without PUT; initial missing ID/Name high Delete returns false without mutation or reset.
  Tests: `network/network_mutations_test.go:TestNetworkMutationSuccessFailureAndMissingCacheBoundaries`, `network/network_mutations_test.go:TestDeleteNetworkInitialMissingDoesNotMutateOrReset`.
- Accepted mutation read/Close/context/source or decode/ID errors preserve the actual response evidence and invalidate the originally owned role cache without another mutation caused by that error. Create/Update malformed/null/missing or wrong-shaped network bodies return an inspectable ResponseError; decoded unsafe or mismatched Update IDs retain the partial model. Delete keeps true plus an accepted response error. Public role/collection replacements during accepted Close are errors but cannot redirect the cache reset to a replacement.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedDecodeAndIdentityErrorsResetWithoutResend`, `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedIOCancellationAndSourceErrorsKeepEvidenceAndOwnedReset`.
- A representative accepted Create resets while an earlier role inventory is in flight: that earlier caller still receives its original result, but the old generation cannot repopulate cache. The next configured default lookup reads a new inventory and repeated lookup uses that new cache. This is the actual Create/Reset fixture, not separate mutation-race coverage for every CRUD method or every getter.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationResetCannotPublishEarlierInflightDiscovery`.
- The Connection-cached high Network Service resets the same role cache after Create, Update and Delete. A later getter reads a new configured inventory, and Nova default NIC plus floating source/NAT allocation consume that same refreshed inventory without manual Reset. The fixture checks actual Nova and Neutron request fields and exactly two inventory reads.
  Tests: `connectiontest/connection_network_mutations_test.go:TestConnectionNetworkMutationRefreshesGetterDefaultNICAndFloatingRoles`.

Python/Go differences:

- The SDK exposes context-aware Connection getters returning owned native-plus-router/provider models and typed nullable pointers; Python returns mutable Network Resources and cached list/object identities. Ordinary generated Networks model behavior is unchanged. This bounded role model does not emulate all nullable descriptors, unknown properties, location or session state.
- Go configured-role validation accepts a matching name OR ID consistently. Pinned classification accepts either, but its subsequent external/internal family validation checks only selected names and can reject ID-only selectors. Go also rejects unsafe IDs and preserves missing/ambiguous and decoding errors rather than incidental dynamic-input behavior.
- Go caches only successful discovery, coalesces in-flight work, lets followers use their own contexts, and uses Reset generations to prevent a pre-reset flight from restoring the cache. Python serializes shared discovery with a lock and can swallow SDKException into sticky empty results. Owned returns and observable/retryable errors are deliberate Go policies.
- Aggregate results retain the source IPv4-then-IPv6 order and duplicates; family classification reflects routing configuration and network extensions, not discovered subnet address families. The NAT destination inference preserves the source last-network-with-gateway rule.
- Go flags are strict typed booleans; the YAML parser accepts boolean/string values via its explicit conversion policy. Pinned cloud use_* accessors store and return raw configured values, whose Python truthiness can differ. Both-disabled skips discovery; a single disabled flag does not remove that family after discovery.
- The Connection maps clean native missing-endpoint detection and the has_network configuration flag with official-service defaults (ServiceEnabled); Python's False for other lookup failures needs WithServiceLookupFailureAsUnavailable, and no version discovery applies to role discovery. Configured default NIC and automatic Ensure floating source use the shared successful role snapshot. Automatic NAT applies only without explicit port/fixed/NAT options and with multiple server-owned ports; strict unique IPv4-pair selection remains. These consumers do not implement private/floating-source flags or automatic IP need/skip decisions.
- For this no-argument named role-list getter, the concrete configured classification/default/order contract and owned RoleNetwork DTO result are the Go mapping. RoleNetwork uses native fields plus router/provider role extensions, not RawResource Body/Wire or full nullable/vendor/location/mutable Network descriptors. Discovery keeps its documented native200/204, linked-empty-page, success-only cache and explicit Reset policies.
- The original consumer-only difference describes the configured-default-NIC and Ensure role layer; separate later automatic-IP, private/source/address, pool/explicit-IP and Nova entries now exist and are compared in their own guides. Full Python has_service, CloudRegion loader overlays and Resource/session/cache lifecycle remain independent declared scopes rather than required inputs of this argument-free getter. This row does not promote scalar role getters, automatic-IP consumers or Network Proxy/Resource operations.

### `python:cloud/_network_common/NetworkCommonCloudMixin/get_internal_networks`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetInternalNetworks`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ResetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.WithNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Discover`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Reset`
- `github.com/JSYoo5B/go-openstacksdk/network.PrepareNetworkRoleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network.WithConfiguredNetworks`
- `github.com/JSYoo5B/go-openstacksdk/network.WithExternalNetworkDiscovery`
- `github.com/JSYoo5B/go-openstacksdk/network.WithInternalNetworkDiscovery`

- Shared discovery classifies IPv4 with router:external or provider:physical_network and IPv6 with router:external only, unless configured name-or-ID family overrides apply. Floating IPv4 candidates are router-external networks or the first configured NAT source; configured source/destination/default roles remain distinct from ordinary family roles.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- An explicit NAT destination skips subnet discovery. Without one, all subnet pages are consumed and the last network in network response order with a nonempty gateway subnet becomes the NAT destination, independent of subnet ip_version. Configured NAT/default name-or-ID selection rejects missing, duplicate or cross-kind ambiguous matches; the first configured NAT source is used.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- Unfiltered network discovery and conditional subnet discovery consume linked empty and later pages. Unsafe network IDs, invalid router extension decoding, later network403, subnet403 and repeated continuation links return errors without a partial snapshot; failed attempts remain retryable. Native204 returns an empty successful discovery.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- All ten Connection role getters share one successful discovery and return independent owned native-plus-extension models. Aggregate getters concatenate IPv4 then IPv6 with duplicate entries retained. Caller changes do not modify cached roles, and tags/subnets/availability-zone-hints retain explicit empty versus null native array presence.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`.
- Concurrent uncached discoveries share one in-flight request and return independent models. A waiting follower can reach its own context deadline without canceling the leader. Reset advances the cache generation so a successful already-running discovery cannot repopulate the cache, while the existing caller still receives its result; subsequent concurrent cached calls issue no new request.
  Tests: `network/roles_test.go:TestNetworkRoleConcurrentDiscoveriesShareFlightAndOwnResults`, `network/roles_test.go:TestNetworkRoleCacheFlightCancellationAndReset`.
- Typed use-external/use-internal flags default enabled and preserve ordered replacement. Both disabled returns empty roles before client/catalog/HTTP access; disabling only external still discovers both families when internal is enabled. Connection nil/canceled context fails before lookup, clean pointer or value ErrEndpointNotFound yields empty roles, and other catalog errors remain inspectable.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- Role selectors are frozen with the same cloud/secure/public snapshot as native authentication. YAML family flags inherit routes_externally unless overridden; a typed WithNetworkRoles policy replaces the YAML role configuration and its default selector, and an empty typed replacement clears configured roles/default. Configured default NIC selection now consumes the same successful role snapshot; explicit typed default names retain their separate exact-name lookup.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Owned configured-network slices can be reused after caller mutation. Nil options, empty names and multiple default/NAT-destination declarations fail before discovery; a later replacement does not hide an earlier invalid option. Explicit Reset invalidates a completed Connection role cache without making a mutation request.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- YAML boolean false and case-insensitive string false disable discovery before network HTTP. An explicit typed policy replaces those flags and restores discovery; a numeric YAML flag is rejected as ErrInvalidOption. Direct role discovery rejects nil or providerless clients before native HTTP.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleBooleanFlagsAndExplicitPolicyReplacement`, `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`.
- Owned high Service CreateNetwork/UpdateNetwork/DeleteNetwork and the existing Networks.Delete adapter invalidate shared roles after accepted mutations. The actual matrix primes a configured default role, observes a new inventory after success or high-delete clean404 after a found lookup, and retains the original inventory after403/409/412, failed no-option lookup, or direct Collection404. A successful no-option Update performs its lookup and reset without PUT; initial missing ID/Name high Delete returns false without mutation or reset.
  Tests: `network/network_mutations_test.go:TestNetworkMutationSuccessFailureAndMissingCacheBoundaries`, `network/network_mutations_test.go:TestDeleteNetworkInitialMissingDoesNotMutateOrReset`.
- Accepted mutation read/Close/context/source or decode/ID errors preserve the actual response evidence and invalidate the originally owned role cache without another mutation caused by that error. Create/Update malformed/null/missing or wrong-shaped network bodies return an inspectable ResponseError; decoded unsafe or mismatched Update IDs retain the partial model. Delete keeps true plus an accepted response error. Public role/collection replacements during accepted Close are errors but cannot redirect the cache reset to a replacement.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedDecodeAndIdentityErrorsResetWithoutResend`, `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedIOCancellationAndSourceErrorsKeepEvidenceAndOwnedReset`.
- A representative accepted Create resets while an earlier role inventory is in flight: that earlier caller still receives its original result, but the old generation cannot repopulate cache. The next configured default lookup reads a new inventory and repeated lookup uses that new cache. This is the actual Create/Reset fixture, not separate mutation-race coverage for every CRUD method or every getter.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationResetCannotPublishEarlierInflightDiscovery`.
- The Connection-cached high Network Service resets the same role cache after Create, Update and Delete. A later getter reads a new configured inventory, and Nova default NIC plus floating source/NAT allocation consume that same refreshed inventory without manual Reset. The fixture checks actual Nova and Neutron request fields and exactly two inventory reads.
  Tests: `connectiontest/connection_network_mutations_test.go:TestConnectionNetworkMutationRefreshesGetterDefaultNICAndFloatingRoles`.

Python/Go differences:

- The SDK exposes context-aware Connection getters returning owned native-plus-router/provider models and typed nullable pointers; Python returns mutable Network Resources and cached list/object identities. Ordinary generated Networks model behavior is unchanged. This bounded role model does not emulate all nullable descriptors, unknown properties, location or session state.
- Go configured-role validation accepts a matching name OR ID consistently. Pinned classification accepts either, but its subsequent external/internal family validation checks only selected names and can reject ID-only selectors. Go also rejects unsafe IDs and preserves missing/ambiguous and decoding errors rather than incidental dynamic-input behavior.
- Go caches only successful discovery, coalesces in-flight work, lets followers use their own contexts, and uses Reset generations to prevent a pre-reset flight from restoring the cache. Python serializes shared discovery with a lock and can swallow SDKException into sticky empty results. Owned returns and observable/retryable errors are deliberate Go policies.
- Aggregate results retain the source IPv4-then-IPv6 order and duplicates; family classification reflects routing configuration and network extensions, not discovered subnet address families. The NAT destination inference preserves the source last-network-with-gateway rule.
- Go flags are strict typed booleans; the YAML parser accepts boolean/string values via its explicit conversion policy. Pinned cloud use_* accessors store and return raw configured values, whose Python truthiness can differ. Both-disabled skips discovery; a single disabled flag does not remove that family after discovery.
- The Connection maps clean native missing-endpoint detection and the has_network configuration flag with official-service defaults (ServiceEnabled); Python's False for other lookup failures needs WithServiceLookupFailureAsUnavailable, and no version discovery applies to role discovery. Configured default NIC and automatic Ensure floating source use the shared successful role snapshot. Automatic NAT applies only without explicit port/fixed/NAT options and with multiple server-owned ports; strict unique IPv4-pair selection remains. These consumers do not implement private/floating-source flags or automatic IP need/skip decisions.
- For this no-argument named role-list getter, the concrete configured classification/default/order contract and owned RoleNetwork DTO result are the Go mapping. RoleNetwork uses native fields plus router/provider role extensions, not RawResource Body/Wire or full nullable/vendor/location/mutable Network descriptors. Discovery keeps its documented native200/204, linked-empty-page, success-only cache and explicit Reset policies.
- The original consumer-only difference describes the configured-default-NIC and Ensure role layer; separate later automatic-IP, private/source/address, pool/explicit-IP and Nova entries now exist and are compared in their own guides. Full Python has_service, CloudRegion loader overlays and Resource/session/cache lifecycle remain independent declared scopes rather than required inputs of this argument-free getter. This row does not promote scalar role getters, automatic-IP consumers or Network Proxy/Resource operations.

### `python:cloud/_network_common/NetworkCommonCloudMixin/get_external_ipv4_networks`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetExternalIPv4Networks`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ResetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.WithNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Discover`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Reset`
- `github.com/JSYoo5B/go-openstacksdk/network.PrepareNetworkRoleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network.WithConfiguredNetworks`
- `github.com/JSYoo5B/go-openstacksdk/network.WithExternalNetworkDiscovery`
- `github.com/JSYoo5B/go-openstacksdk/network.WithInternalNetworkDiscovery`

- Shared discovery classifies IPv4 with router:external or provider:physical_network and IPv6 with router:external only, unless configured name-or-ID family overrides apply. Floating IPv4 candidates are router-external networks or the first configured NAT source; configured source/destination/default roles remain distinct from ordinary family roles.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- An explicit NAT destination skips subnet discovery. Without one, all subnet pages are consumed and the last network in network response order with a nonempty gateway subnet becomes the NAT destination, independent of subnet ip_version. Configured NAT/default name-or-ID selection rejects missing, duplicate or cross-kind ambiguous matches; the first configured NAT source is used.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- Unfiltered network discovery and conditional subnet discovery consume linked empty and later pages. Unsafe network IDs, invalid router extension decoding, later network403, subnet403 and repeated continuation links return errors without a partial snapshot; failed attempts remain retryable. Native204 returns an empty successful discovery.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- All ten Connection role getters share one successful discovery and return independent owned native-plus-extension models. Aggregate getters concatenate IPv4 then IPv6 with duplicate entries retained. Caller changes do not modify cached roles, and tags/subnets/availability-zone-hints retain explicit empty versus null native array presence.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`.
- Concurrent uncached discoveries share one in-flight request and return independent models. A waiting follower can reach its own context deadline without canceling the leader. Reset advances the cache generation so a successful already-running discovery cannot repopulate the cache, while the existing caller still receives its result; subsequent concurrent cached calls issue no new request.
  Tests: `network/roles_test.go:TestNetworkRoleConcurrentDiscoveriesShareFlightAndOwnResults`, `network/roles_test.go:TestNetworkRoleCacheFlightCancellationAndReset`.
- Typed use-external/use-internal flags default enabled and preserve ordered replacement. Both disabled returns empty roles before client/catalog/HTTP access; disabling only external still discovers both families when internal is enabled. Connection nil/canceled context fails before lookup, clean pointer or value ErrEndpointNotFound yields empty roles, and other catalog errors remain inspectable.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- Role selectors are frozen with the same cloud/secure/public snapshot as native authentication. YAML family flags inherit routes_externally unless overridden; a typed WithNetworkRoles policy replaces the YAML role configuration and its default selector, and an empty typed replacement clears configured roles/default. Configured default NIC selection now consumes the same successful role snapshot; explicit typed default names retain their separate exact-name lookup.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Owned configured-network slices can be reused after caller mutation. Nil options, empty names and multiple default/NAT-destination declarations fail before discovery; a later replacement does not hide an earlier invalid option. Explicit Reset invalidates a completed Connection role cache without making a mutation request.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- YAML boolean false and case-insensitive string false disable discovery before network HTTP. An explicit typed policy replaces those flags and restores discovery; a numeric YAML flag is rejected as ErrInvalidOption. Direct role discovery rejects nil or providerless clients before native HTTP.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleBooleanFlagsAndExplicitPolicyReplacement`, `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`.
- Owned high Service CreateNetwork/UpdateNetwork/DeleteNetwork and the existing Networks.Delete adapter invalidate shared roles after accepted mutations. The actual matrix primes a configured default role, observes a new inventory after success or high-delete clean404 after a found lookup, and retains the original inventory after403/409/412, failed no-option lookup, or direct Collection404. A successful no-option Update performs its lookup and reset without PUT; initial missing ID/Name high Delete returns false without mutation or reset.
  Tests: `network/network_mutations_test.go:TestNetworkMutationSuccessFailureAndMissingCacheBoundaries`, `network/network_mutations_test.go:TestDeleteNetworkInitialMissingDoesNotMutateOrReset`.
- Accepted mutation read/Close/context/source or decode/ID errors preserve the actual response evidence and invalidate the originally owned role cache without another mutation caused by that error. Create/Update malformed/null/missing or wrong-shaped network bodies return an inspectable ResponseError; decoded unsafe or mismatched Update IDs retain the partial model. Delete keeps true plus an accepted response error. Public role/collection replacements during accepted Close are errors but cannot redirect the cache reset to a replacement.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedDecodeAndIdentityErrorsResetWithoutResend`, `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedIOCancellationAndSourceErrorsKeepEvidenceAndOwnedReset`.
- A representative accepted Create resets while an earlier role inventory is in flight: that earlier caller still receives its original result, but the old generation cannot repopulate cache. The next configured default lookup reads a new inventory and repeated lookup uses that new cache. This is the actual Create/Reset fixture, not separate mutation-race coverage for every CRUD method or every getter.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationResetCannotPublishEarlierInflightDiscovery`.
- The Connection-cached high Network Service resets the same role cache after Create, Update and Delete. A later getter reads a new configured inventory, and Nova default NIC plus floating source/NAT allocation consume that same refreshed inventory without manual Reset. The fixture checks actual Nova and Neutron request fields and exactly two inventory reads.
  Tests: `connectiontest/connection_network_mutations_test.go:TestConnectionNetworkMutationRefreshesGetterDefaultNICAndFloatingRoles`.

Python/Go differences:

- The SDK exposes context-aware Connection getters returning owned native-plus-router/provider models and typed nullable pointers; Python returns mutable Network Resources and cached list/object identities. Ordinary generated Networks model behavior is unchanged. This bounded role model does not emulate all nullable descriptors, unknown properties, location or session state.
- Go configured-role validation accepts a matching name OR ID consistently. Pinned classification accepts either, but its subsequent external/internal family validation checks only selected names and can reject ID-only selectors. Go also rejects unsafe IDs and preserves missing/ambiguous and decoding errors rather than incidental dynamic-input behavior.
- Go caches only successful discovery, coalesces in-flight work, lets followers use their own contexts, and uses Reset generations to prevent a pre-reset flight from restoring the cache. Python serializes shared discovery with a lock and can swallow SDKException into sticky empty results. Owned returns and observable/retryable errors are deliberate Go policies.
- Aggregate results retain the source IPv4-then-IPv6 order and duplicates; family classification reflects routing configuration and network extensions, not discovered subnet address families. The NAT destination inference preserves the source last-network-with-gateway rule.
- Go flags are strict typed booleans; the YAML parser accepts boolean/string values via its explicit conversion policy. Pinned cloud use_* accessors store and return raw configured values, whose Python truthiness can differ. Both-disabled skips discovery; a single disabled flag does not remove that family after discovery.
- The Connection maps clean native missing-endpoint detection and the has_network configuration flag with official-service defaults (ServiceEnabled); Python's False for other lookup failures needs WithServiceLookupFailureAsUnavailable, and no version discovery applies to role discovery. Configured default NIC and automatic Ensure floating source use the shared successful role snapshot. Automatic NAT applies only without explicit port/fixed/NAT options and with multiple server-owned ports; strict unique IPv4-pair selection remains. These consumers do not implement private/floating-source flags or automatic IP need/skip decisions.
- For this no-argument named role-list getter, the concrete configured classification/default/order contract and owned RoleNetwork DTO result are the Go mapping. RoleNetwork uses native fields plus router/provider role extensions, not RawResource Body/Wire or full nullable/vendor/location/mutable Network descriptors. Discovery keeps its documented native200/204, linked-empty-page, success-only cache and explicit Reset policies.
- The original consumer-only difference describes the configured-default-NIC and Ensure role layer; separate later automatic-IP, private/source/address, pool/explicit-IP and Nova entries now exist and are compared in their own guides. Full Python has_service, CloudRegion loader overlays and Resource/session/cache lifecycle remain independent declared scopes rather than required inputs of this argument-free getter. This row does not promote scalar role getters, automatic-IP consumers or Network Proxy/Resource operations.

### `python:cloud/_network_common/NetworkCommonCloudMixin/get_external_ipv4_floating_networks`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetExternalIPv4FloatingNetworks`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ResetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.WithNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Discover`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Reset`
- `github.com/JSYoo5B/go-openstacksdk/network.PrepareNetworkRoleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network.WithConfiguredNetworks`
- `github.com/JSYoo5B/go-openstacksdk/network.WithExternalNetworkDiscovery`
- `github.com/JSYoo5B/go-openstacksdk/network.WithInternalNetworkDiscovery`

- Shared discovery classifies IPv4 with router:external or provider:physical_network and IPv6 with router:external only, unless configured name-or-ID family overrides apply. Floating IPv4 candidates are router-external networks or the first configured NAT source; configured source/destination/default roles remain distinct from ordinary family roles.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- An explicit NAT destination skips subnet discovery. Without one, all subnet pages are consumed and the last network in network response order with a nonempty gateway subnet becomes the NAT destination, independent of subnet ip_version. Configured NAT/default name-or-ID selection rejects missing, duplicate or cross-kind ambiguous matches; the first configured NAT source is used.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- Unfiltered network discovery and conditional subnet discovery consume linked empty and later pages. Unsafe network IDs, invalid router extension decoding, later network403, subnet403 and repeated continuation links return errors without a partial snapshot; failed attempts remain retryable. Native204 returns an empty successful discovery.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- All ten Connection role getters share one successful discovery and return independent owned native-plus-extension models. Aggregate getters concatenate IPv4 then IPv6 with duplicate entries retained. Caller changes do not modify cached roles, and tags/subnets/availability-zone-hints retain explicit empty versus null native array presence.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`.
- Concurrent uncached discoveries share one in-flight request and return independent models. A waiting follower can reach its own context deadline without canceling the leader. Reset advances the cache generation so a successful already-running discovery cannot repopulate the cache, while the existing caller still receives its result; subsequent concurrent cached calls issue no new request.
  Tests: `network/roles_test.go:TestNetworkRoleConcurrentDiscoveriesShareFlightAndOwnResults`, `network/roles_test.go:TestNetworkRoleCacheFlightCancellationAndReset`.
- Typed use-external/use-internal flags default enabled and preserve ordered replacement. Both disabled returns empty roles before client/catalog/HTTP access; disabling only external still discovers both families when internal is enabled. Connection nil/canceled context fails before lookup, clean pointer or value ErrEndpointNotFound yields empty roles, and other catalog errors remain inspectable.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- Role selectors are frozen with the same cloud/secure/public snapshot as native authentication. YAML family flags inherit routes_externally unless overridden; a typed WithNetworkRoles policy replaces the YAML role configuration and its default selector, and an empty typed replacement clears configured roles/default. Configured default NIC selection now consumes the same successful role snapshot; explicit typed default names retain their separate exact-name lookup.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Owned configured-network slices can be reused after caller mutation. Nil options, empty names and multiple default/NAT-destination declarations fail before discovery; a later replacement does not hide an earlier invalid option. Explicit Reset invalidates a completed Connection role cache without making a mutation request.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- YAML boolean false and case-insensitive string false disable discovery before network HTTP. An explicit typed policy replaces those flags and restores discovery; a numeric YAML flag is rejected as ErrInvalidOption. Direct role discovery rejects nil or providerless clients before native HTTP.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleBooleanFlagsAndExplicitPolicyReplacement`, `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`.
- Ensure zero external Ref consumes the shared floating-role candidates rather than the old router-external filtered collection. Configured NAT source may select a non-router-external network and ordinary physical-provider external IPv4 is not automatically a floating source. Consecutive calls reuse the owned successful role snapshot, still allocate per call, and Reset refreshes the chosen external ID; PrepareEnsureActive itself does not perform role discovery.
  Tests: `network/floating_ip_roles_test.go:TestFloatingIPEnsureSharedSourceCacheAndReset`.
- Failed shared network/subnet discovery, including later-page403 or a configured floating source missing, prevents port/IP mutation and router fallback and remains retryable. Only a successful empty floating-role list proceeds to enabled-router gateway fallback; linked empty/later router pages and their errors are consumed before mutation. Both family flags disabled also produce an empty role snapshot and allow router fallback without a role inventory request, despite a configured source missing.
  Tests: `network/floating_ip_roles_test.go:TestFloatingIPEnsureRoleErrorsStopRouterFallbackAndRetry`, `network/floating_ip_selection_test.go:TestFloatingIPEnsureAutomaticNetworkConsumesEmptyAndLaterPages`, `network/floating_ip_roles_test.go:TestFloatingIPEnsureDisabledRolesStillUseRouterFallback`.
- Connection CreateWithFloatingIP uses a single successful configured role inventory for its default NIC, floating source and NAT destination, including a configured source without router:external. Role lookup finishes before Nova POST; actual server ACTIVE precedes port and floating-IP requests, and actual floating-IP ACTIVE remains mandatory. A role403 returns no created resource; a later port403 preserves the known ACTIVE Server with nil assignment and the original HTTP cause, without allocation or cleanup.
  Tests: `connectiontest/connection_server_network_roles_test.go:TestConnectionCreateWithFloatingIPSharesDefaultSourceAndNATRoles`.
- Owned high Service CreateNetwork/UpdateNetwork/DeleteNetwork and the existing Networks.Delete adapter invalidate shared roles after accepted mutations. The actual matrix primes a configured default role, observes a new inventory after success or high-delete clean404 after a found lookup, and retains the original inventory after403/409/412, failed no-option lookup, or direct Collection404. A successful no-option Update performs its lookup and reset without PUT; initial missing ID/Name high Delete returns false without mutation or reset.
  Tests: `network/network_mutations_test.go:TestNetworkMutationSuccessFailureAndMissingCacheBoundaries`, `network/network_mutations_test.go:TestDeleteNetworkInitialMissingDoesNotMutateOrReset`.
- Accepted mutation read/Close/context/source or decode/ID errors preserve the actual response evidence and invalidate the originally owned role cache without another mutation caused by that error. Create/Update malformed/null/missing or wrong-shaped network bodies return an inspectable ResponseError; decoded unsafe or mismatched Update IDs retain the partial model. Delete keeps true plus an accepted response error. Public role/collection replacements during accepted Close are errors but cannot redirect the cache reset to a replacement.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedDecodeAndIdentityErrorsResetWithoutResend`, `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedIOCancellationAndSourceErrorsKeepEvidenceAndOwnedReset`.
- A representative accepted Create resets while an earlier role inventory is in flight: that earlier caller still receives its original result, but the old generation cannot repopulate cache. The next configured default lookup reads a new inventory and repeated lookup uses that new cache. This is the actual Create/Reset fixture, not separate mutation-race coverage for every CRUD method or every getter.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationResetCannotPublishEarlierInflightDiscovery`.
- The Connection-cached high Network Service resets the same role cache after Create, Update and Delete. A later getter reads a new configured inventory, and Nova default NIC plus floating source/NAT allocation consume that same refreshed inventory without manual Reset. The fixture checks actual Nova and Neutron request fields and exactly two inventory reads.
  Tests: `connectiontest/connection_network_mutations_test.go:TestConnectionNetworkMutationRefreshesGetterDefaultNICAndFloatingRoles`.

Python/Go differences:

- The SDK exposes context-aware Connection getters returning owned native-plus-router/provider models and typed nullable pointers; Python returns mutable Network Resources and cached list/object identities. Ordinary generated Networks model behavior is unchanged. This bounded role model does not emulate all nullable descriptors, unknown properties, location or session state.
- Go configured-role validation accepts a matching name OR ID consistently. Pinned classification accepts either, but its subsequent external/internal family validation checks only selected names and can reject ID-only selectors. Go also rejects unsafe IDs and preserves missing/ambiguous and decoding errors rather than incidental dynamic-input behavior.
- Go caches only successful discovery, coalesces in-flight work, lets followers use their own contexts, and uses Reset generations to prevent a pre-reset flight from restoring the cache. Python serializes shared discovery with a lock and can swallow SDKException into sticky empty results. Owned returns and observable/retryable errors are deliberate Go policies.
- Aggregate results retain the source IPv4-then-IPv6 order and duplicates; family classification reflects routing configuration and network extensions, not discovered subnet address families. The NAT destination inference preserves the source last-network-with-gateway rule.
- Go flags are strict typed booleans; the YAML parser accepts boolean/string values via its explicit conversion policy. Pinned cloud use_* accessors store and return raw configured values, whose Python truthiness can differ. Both-disabled skips discovery; a single disabled flag does not remove that family after discovery.
- The Connection maps clean native missing-endpoint detection and the has_network configuration flag with official-service defaults (ServiceEnabled); Python's False for other lookup failures needs WithServiceLookupFailureAsUnavailable, and no version discovery applies to role discovery. Configured default NIC and automatic Ensure floating source use the shared successful role snapshot. Automatic NAT applies only without explicit port/fixed/NAT options and with multiple server-owned ports; strict unique IPv4-pair selection remains. These consumers do not implement private/floating-source flags or automatic IP need/skip decisions.
- For this no-argument named role-list getter, the concrete configured classification/default/order contract and owned RoleNetwork DTO result are the Go mapping. RoleNetwork uses native fields plus router/provider role extensions, not RawResource Body/Wire or full nullable/vendor/location/mutable Network descriptors. Discovery keeps its documented native200/204, linked-empty-page, success-only cache and explicit Reset policies.
- The original consumer-only difference describes the configured-default-NIC and Ensure role layer; separate later automatic-IP, private/source/address, pool/explicit-IP and Nova entries now exist and are compared in their own guides. Full Python has_service, CloudRegion loader overlays and Resource/session/cache lifecycle remain independent declared scopes rather than required inputs of this argument-free getter. This row does not promote scalar role getters, automatic-IP consumers or Network Proxy/Resource operations.

### `python:cloud/_network_common/NetworkCommonCloudMixin/get_internal_ipv4_networks`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetInternalIPv4Networks`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ResetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.WithNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Discover`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Reset`
- `github.com/JSYoo5B/go-openstacksdk/network.PrepareNetworkRoleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network.WithConfiguredNetworks`
- `github.com/JSYoo5B/go-openstacksdk/network.WithExternalNetworkDiscovery`
- `github.com/JSYoo5B/go-openstacksdk/network.WithInternalNetworkDiscovery`

- Shared discovery classifies IPv4 with router:external or provider:physical_network and IPv6 with router:external only, unless configured name-or-ID family overrides apply. Floating IPv4 candidates are router-external networks or the first configured NAT source; configured source/destination/default roles remain distinct from ordinary family roles.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- An explicit NAT destination skips subnet discovery. Without one, all subnet pages are consumed and the last network in network response order with a nonempty gateway subnet becomes the NAT destination, independent of subnet ip_version. Configured NAT/default name-or-ID selection rejects missing, duplicate or cross-kind ambiguous matches; the first configured NAT source is used.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- Unfiltered network discovery and conditional subnet discovery consume linked empty and later pages. Unsafe network IDs, invalid router extension decoding, later network403, subnet403 and repeated continuation links return errors without a partial snapshot; failed attempts remain retryable. Native204 returns an empty successful discovery.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- All ten Connection role getters share one successful discovery and return independent owned native-plus-extension models. Aggregate getters concatenate IPv4 then IPv6 with duplicate entries retained. Caller changes do not modify cached roles, and tags/subnets/availability-zone-hints retain explicit empty versus null native array presence.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`.
- Concurrent uncached discoveries share one in-flight request and return independent models. A waiting follower can reach its own context deadline without canceling the leader. Reset advances the cache generation so a successful already-running discovery cannot repopulate the cache, while the existing caller still receives its result; subsequent concurrent cached calls issue no new request.
  Tests: `network/roles_test.go:TestNetworkRoleConcurrentDiscoveriesShareFlightAndOwnResults`, `network/roles_test.go:TestNetworkRoleCacheFlightCancellationAndReset`.
- Typed use-external/use-internal flags default enabled and preserve ordered replacement. Both disabled returns empty roles before client/catalog/HTTP access; disabling only external still discovers both families when internal is enabled. Connection nil/canceled context fails before lookup, clean pointer or value ErrEndpointNotFound yields empty roles, and other catalog errors remain inspectable.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- Role selectors are frozen with the same cloud/secure/public snapshot as native authentication. YAML family flags inherit routes_externally unless overridden; a typed WithNetworkRoles policy replaces the YAML role configuration and its default selector, and an empty typed replacement clears configured roles/default. Configured default NIC selection now consumes the same successful role snapshot; explicit typed default names retain their separate exact-name lookup.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Owned configured-network slices can be reused after caller mutation. Nil options, empty names and multiple default/NAT-destination declarations fail before discovery; a later replacement does not hide an earlier invalid option. Explicit Reset invalidates a completed Connection role cache without making a mutation request.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- YAML boolean false and case-insensitive string false disable discovery before network HTTP. An explicit typed policy replaces those flags and restores discovery; a numeric YAML flag is rejected as ErrInvalidOption. Direct role discovery rejects nil or providerless clients before native HTTP.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleBooleanFlagsAndExplicitPolicyReplacement`, `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`.
- Owned high Service CreateNetwork/UpdateNetwork/DeleteNetwork and the existing Networks.Delete adapter invalidate shared roles after accepted mutations. The actual matrix primes a configured default role, observes a new inventory after success or high-delete clean404 after a found lookup, and retains the original inventory after403/409/412, failed no-option lookup, or direct Collection404. A successful no-option Update performs its lookup and reset without PUT; initial missing ID/Name high Delete returns false without mutation or reset.
  Tests: `network/network_mutations_test.go:TestNetworkMutationSuccessFailureAndMissingCacheBoundaries`, `network/network_mutations_test.go:TestDeleteNetworkInitialMissingDoesNotMutateOrReset`.
- Accepted mutation read/Close/context/source or decode/ID errors preserve the actual response evidence and invalidate the originally owned role cache without another mutation caused by that error. Create/Update malformed/null/missing or wrong-shaped network bodies return an inspectable ResponseError; decoded unsafe or mismatched Update IDs retain the partial model. Delete keeps true plus an accepted response error. Public role/collection replacements during accepted Close are errors but cannot redirect the cache reset to a replacement.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedDecodeAndIdentityErrorsResetWithoutResend`, `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedIOCancellationAndSourceErrorsKeepEvidenceAndOwnedReset`.
- A representative accepted Create resets while an earlier role inventory is in flight: that earlier caller still receives its original result, but the old generation cannot repopulate cache. The next configured default lookup reads a new inventory and repeated lookup uses that new cache. This is the actual Create/Reset fixture, not separate mutation-race coverage for every CRUD method or every getter.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationResetCannotPublishEarlierInflightDiscovery`.
- The Connection-cached high Network Service resets the same role cache after Create, Update and Delete. A later getter reads a new configured inventory, and Nova default NIC plus floating source/NAT allocation consume that same refreshed inventory without manual Reset. The fixture checks actual Nova and Neutron request fields and exactly two inventory reads.
  Tests: `connectiontest/connection_network_mutations_test.go:TestConnectionNetworkMutationRefreshesGetterDefaultNICAndFloatingRoles`.

Python/Go differences:

- The SDK exposes context-aware Connection getters returning owned native-plus-router/provider models and typed nullable pointers; Python returns mutable Network Resources and cached list/object identities. Ordinary generated Networks model behavior is unchanged. This bounded role model does not emulate all nullable descriptors, unknown properties, location or session state.
- Go configured-role validation accepts a matching name OR ID consistently. Pinned classification accepts either, but its subsequent external/internal family validation checks only selected names and can reject ID-only selectors. Go also rejects unsafe IDs and preserves missing/ambiguous and decoding errors rather than incidental dynamic-input behavior.
- Go caches only successful discovery, coalesces in-flight work, lets followers use their own contexts, and uses Reset generations to prevent a pre-reset flight from restoring the cache. Python serializes shared discovery with a lock and can swallow SDKException into sticky empty results. Owned returns and observable/retryable errors are deliberate Go policies.
- Aggregate results retain the source IPv4-then-IPv6 order and duplicates; family classification reflects routing configuration and network extensions, not discovered subnet address families. The NAT destination inference preserves the source last-network-with-gateway rule.
- Go flags are strict typed booleans; the YAML parser accepts boolean/string values via its explicit conversion policy. Pinned cloud use_* accessors store and return raw configured values, whose Python truthiness can differ. Both-disabled skips discovery; a single disabled flag does not remove that family after discovery.
- The Connection maps clean native missing-endpoint detection and the has_network configuration flag with official-service defaults (ServiceEnabled); Python's False for other lookup failures needs WithServiceLookupFailureAsUnavailable, and no version discovery applies to role discovery. Configured default NIC and automatic Ensure floating source use the shared successful role snapshot. Automatic NAT applies only without explicit port/fixed/NAT options and with multiple server-owned ports; strict unique IPv4-pair selection remains. These consumers do not implement private/floating-source flags or automatic IP need/skip decisions.
- For this no-argument named role-list getter, the concrete configured classification/default/order contract and owned RoleNetwork DTO result are the Go mapping. RoleNetwork uses native fields plus router/provider role extensions, not RawResource Body/Wire or full nullable/vendor/location/mutable Network descriptors. Discovery keeps its documented native200/204, linked-empty-page, success-only cache and explicit Reset policies.
- The original consumer-only difference describes the configured-default-NIC and Ensure role layer; separate later automatic-IP, private/source/address, pool/explicit-IP and Nova entries now exist and are compared in their own guides. Full Python has_service, CloudRegion loader overlays and Resource/session/cache lifecycle remain independent declared scopes rather than required inputs of this argument-free getter. This row does not promote scalar role getters, automatic-IP consumers or Network Proxy/Resource operations.

### `python:cloud/_network_common/NetworkCommonCloudMixin/get_external_ipv6_networks`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetExternalIPv6Networks`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ResetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.WithNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Discover`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Reset`
- `github.com/JSYoo5B/go-openstacksdk/network.PrepareNetworkRoleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network.WithConfiguredNetworks`
- `github.com/JSYoo5B/go-openstacksdk/network.WithExternalNetworkDiscovery`
- `github.com/JSYoo5B/go-openstacksdk/network.WithInternalNetworkDiscovery`

- Shared discovery classifies IPv4 with router:external or provider:physical_network and IPv6 with router:external only, unless configured name-or-ID family overrides apply. Floating IPv4 candidates are router-external networks or the first configured NAT source; configured source/destination/default roles remain distinct from ordinary family roles.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- An explicit NAT destination skips subnet discovery. Without one, all subnet pages are consumed and the last network in network response order with a nonempty gateway subnet becomes the NAT destination, independent of subnet ip_version. Configured NAT/default name-or-ID selection rejects missing, duplicate or cross-kind ambiguous matches; the first configured NAT source is used.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- Unfiltered network discovery and conditional subnet discovery consume linked empty and later pages. Unsafe network IDs, invalid router extension decoding, later network403, subnet403 and repeated continuation links return errors without a partial snapshot; failed attempts remain retryable. Native204 returns an empty successful discovery.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- All ten Connection role getters share one successful discovery and return independent owned native-plus-extension models. Aggregate getters concatenate IPv4 then IPv6 with duplicate entries retained. Caller changes do not modify cached roles, and tags/subnets/availability-zone-hints retain explicit empty versus null native array presence.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`.
- Concurrent uncached discoveries share one in-flight request and return independent models. A waiting follower can reach its own context deadline without canceling the leader. Reset advances the cache generation so a successful already-running discovery cannot repopulate the cache, while the existing caller still receives its result; subsequent concurrent cached calls issue no new request.
  Tests: `network/roles_test.go:TestNetworkRoleConcurrentDiscoveriesShareFlightAndOwnResults`, `network/roles_test.go:TestNetworkRoleCacheFlightCancellationAndReset`.
- Typed use-external/use-internal flags default enabled and preserve ordered replacement. Both disabled returns empty roles before client/catalog/HTTP access; disabling only external still discovers both families when internal is enabled. Connection nil/canceled context fails before lookup, clean pointer or value ErrEndpointNotFound yields empty roles, and other catalog errors remain inspectable.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- Role selectors are frozen with the same cloud/secure/public snapshot as native authentication. YAML family flags inherit routes_externally unless overridden; a typed WithNetworkRoles policy replaces the YAML role configuration and its default selector, and an empty typed replacement clears configured roles/default. Configured default NIC selection now consumes the same successful role snapshot; explicit typed default names retain their separate exact-name lookup.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Owned configured-network slices can be reused after caller mutation. Nil options, empty names and multiple default/NAT-destination declarations fail before discovery; a later replacement does not hide an earlier invalid option. Explicit Reset invalidates a completed Connection role cache without making a mutation request.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- YAML boolean false and case-insensitive string false disable discovery before network HTTP. An explicit typed policy replaces those flags and restores discovery; a numeric YAML flag is rejected as ErrInvalidOption. Direct role discovery rejects nil or providerless clients before native HTTP.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleBooleanFlagsAndExplicitPolicyReplacement`, `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`.
- Owned high Service CreateNetwork/UpdateNetwork/DeleteNetwork and the existing Networks.Delete adapter invalidate shared roles after accepted mutations. The actual matrix primes a configured default role, observes a new inventory after success or high-delete clean404 after a found lookup, and retains the original inventory after403/409/412, failed no-option lookup, or direct Collection404. A successful no-option Update performs its lookup and reset without PUT; initial missing ID/Name high Delete returns false without mutation or reset.
  Tests: `network/network_mutations_test.go:TestNetworkMutationSuccessFailureAndMissingCacheBoundaries`, `network/network_mutations_test.go:TestDeleteNetworkInitialMissingDoesNotMutateOrReset`.
- Accepted mutation read/Close/context/source or decode/ID errors preserve the actual response evidence and invalidate the originally owned role cache without another mutation caused by that error. Create/Update malformed/null/missing or wrong-shaped network bodies return an inspectable ResponseError; decoded unsafe or mismatched Update IDs retain the partial model. Delete keeps true plus an accepted response error. Public role/collection replacements during accepted Close are errors but cannot redirect the cache reset to a replacement.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedDecodeAndIdentityErrorsResetWithoutResend`, `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedIOCancellationAndSourceErrorsKeepEvidenceAndOwnedReset`.
- A representative accepted Create resets while an earlier role inventory is in flight: that earlier caller still receives its original result, but the old generation cannot repopulate cache. The next configured default lookup reads a new inventory and repeated lookup uses that new cache. This is the actual Create/Reset fixture, not separate mutation-race coverage for every CRUD method or every getter.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationResetCannotPublishEarlierInflightDiscovery`.
- The Connection-cached high Network Service resets the same role cache after Create, Update and Delete. A later getter reads a new configured inventory, and Nova default NIC plus floating source/NAT allocation consume that same refreshed inventory without manual Reset. The fixture checks actual Nova and Neutron request fields and exactly two inventory reads.
  Tests: `connectiontest/connection_network_mutations_test.go:TestConnectionNetworkMutationRefreshesGetterDefaultNICAndFloatingRoles`.

Python/Go differences:

- The SDK exposes context-aware Connection getters returning owned native-plus-router/provider models and typed nullable pointers; Python returns mutable Network Resources and cached list/object identities. Ordinary generated Networks model behavior is unchanged. This bounded role model does not emulate all nullable descriptors, unknown properties, location or session state.
- Go configured-role validation accepts a matching name OR ID consistently. Pinned classification accepts either, but its subsequent external/internal family validation checks only selected names and can reject ID-only selectors. Go also rejects unsafe IDs and preserves missing/ambiguous and decoding errors rather than incidental dynamic-input behavior.
- Go caches only successful discovery, coalesces in-flight work, lets followers use their own contexts, and uses Reset generations to prevent a pre-reset flight from restoring the cache. Python serializes shared discovery with a lock and can swallow SDKException into sticky empty results. Owned returns and observable/retryable errors are deliberate Go policies.
- Aggregate results retain the source IPv4-then-IPv6 order and duplicates; family classification reflects routing configuration and network extensions, not discovered subnet address families. The NAT destination inference preserves the source last-network-with-gateway rule.
- Go flags are strict typed booleans; the YAML parser accepts boolean/string values via its explicit conversion policy. Pinned cloud use_* accessors store and return raw configured values, whose Python truthiness can differ. Both-disabled skips discovery; a single disabled flag does not remove that family after discovery.
- The Connection maps clean native missing-endpoint detection and the has_network configuration flag with official-service defaults (ServiceEnabled); Python's False for other lookup failures needs WithServiceLookupFailureAsUnavailable, and no version discovery applies to role discovery. Configured default NIC and automatic Ensure floating source use the shared successful role snapshot. Automatic NAT applies only without explicit port/fixed/NAT options and with multiple server-owned ports; strict unique IPv4-pair selection remains. These consumers do not implement private/floating-source flags or automatic IP need/skip decisions.
- For this no-argument named role-list getter, the concrete configured classification/default/order contract and owned RoleNetwork DTO result are the Go mapping. RoleNetwork uses native fields plus router/provider role extensions, not RawResource Body/Wire or full nullable/vendor/location/mutable Network descriptors. Discovery keeps its documented native200/204, linked-empty-page, success-only cache and explicit Reset policies.
- The original consumer-only difference describes the configured-default-NIC and Ensure role layer; separate later automatic-IP, private/source/address, pool/explicit-IP and Nova entries now exist and are compared in their own guides. Full Python has_service, CloudRegion loader overlays and Resource/session/cache lifecycle remain independent declared scopes rather than required inputs of this argument-free getter. This row does not promote scalar role getters, automatic-IP consumers or Network Proxy/Resource operations.

### `python:cloud/_network_common/NetworkCommonCloudMixin/get_internal_ipv6_networks`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetInternalIPv6Networks`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ResetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.WithNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Discover`
- `github.com/JSYoo5B/go-openstacksdk/network.NetworkRoles.Reset`
- `github.com/JSYoo5B/go-openstacksdk/network.PrepareNetworkRoleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network.WithConfiguredNetworks`
- `github.com/JSYoo5B/go-openstacksdk/network.WithExternalNetworkDiscovery`
- `github.com/JSYoo5B/go-openstacksdk/network.WithInternalNetworkDiscovery`

- Shared discovery classifies IPv4 with router:external or provider:physical_network and IPv6 with router:external only, unless configured name-or-ID family overrides apply. Floating IPv4 candidates are router-external networks or the first configured NAT source; configured source/destination/default roles remain distinct from ordinary family roles.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- An explicit NAT destination skips subnet discovery. Without one, all subnet pages are consumed and the last network in network response order with a nonempty gateway subnet becomes the NAT destination, independent of subnet ip_version. Configured NAT/default name-or-ID selection rejects missing, duplicate or cross-kind ambiguous matches; the first configured NAT source is used.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- Unfiltered network discovery and conditional subnet discovery consume linked empty and later pages. Unsafe network IDs, invalid router extension decoding, later network403, subnet403 and repeated continuation links return errors without a partial snapshot; failed attempts remain retryable. Native204 returns an empty successful discovery.
  Tests: `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`, `network/roles_test.go:TestNetworkRoleDiscoveryValidatesConfiguredSelectionsAndDoesNotCacheErrors`.
- All ten Connection role getters share one successful discovery and return independent owned native-plus-extension models. Aggregate getters concatenate IPv4 then IPv6 with duplicate entries retained. Caller changes do not modify cached roles, and tags/subnets/availability-zone-hints retain explicit empty versus null native array presence.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `network/roles_test.go:TestNetworkRoleDiscoveryClassifiesFamiliesAndNATAcrossEmptyPages`.
- Concurrent uncached discoveries share one in-flight request and return independent models. A waiting follower can reach its own context deadline without canceling the leader. Reset advances the cache generation so a successful already-running discovery cannot repopulate the cache, while the existing caller still receives its result; subsequent concurrent cached calls issue no new request.
  Tests: `network/roles_test.go:TestNetworkRoleConcurrentDiscoveriesShareFlightAndOwnResults`, `network/roles_test.go:TestNetworkRoleCacheFlightCancellationAndReset`.
- Typed use-external/use-internal flags default enabled and preserve ordered replacement. Both disabled returns empty roles before client/catalog/HTTP access; disabling only external still discovers both families when internal is enabled. Connection nil/canceled context fails before lookup, clean pointer or value ErrEndpointNotFound yields empty roles, and other catalog errors remain inspectable.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- Role selectors are frozen with the same cloud/secure/public snapshot as native authentication. YAML family flags inherit routes_externally unless overridden; a typed WithNetworkRoles policy replaces the YAML role configuration and its default selector, and an empty typed replacement clears configured roles/default. Configured default NIC selection now consumes the same successful role snapshot; explicit typed default names retain their separate exact-name lookup.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Owned configured-network slices can be reused after caller mutation. Nil options, empty names and multiple default/NAT-destination declarations fail before discovery; a later replacement does not hide an earlier invalid option. Explicit Reset invalidates a completed Connection role cache without making a mutation request.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- YAML boolean false and case-insensitive string false disable discovery before network HTTP. An explicit typed policy replaces those flags and restores discovery; a numeric YAML flag is rejected as ErrInvalidOption. Direct role discovery rejects nil or providerless clients before native HTTP.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleBooleanFlagsAndExplicitPolicyReplacement`, `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`.
- Owned high Service CreateNetwork/UpdateNetwork/DeleteNetwork and the existing Networks.Delete adapter invalidate shared roles after accepted mutations. The actual matrix primes a configured default role, observes a new inventory after success or high-delete clean404 after a found lookup, and retains the original inventory after403/409/412, failed no-option lookup, or direct Collection404. A successful no-option Update performs its lookup and reset without PUT; initial missing ID/Name high Delete returns false without mutation or reset.
  Tests: `network/network_mutations_test.go:TestNetworkMutationSuccessFailureAndMissingCacheBoundaries`, `network/network_mutations_test.go:TestDeleteNetworkInitialMissingDoesNotMutateOrReset`.
- Accepted mutation read/Close/context/source or decode/ID errors preserve the actual response evidence and invalidate the originally owned role cache without another mutation caused by that error. Create/Update malformed/null/missing or wrong-shaped network bodies return an inspectable ResponseError; decoded unsafe or mismatched Update IDs retain the partial model. Delete keeps true plus an accepted response error. Public role/collection replacements during accepted Close are errors but cannot redirect the cache reset to a replacement.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedDecodeAndIdentityErrorsResetWithoutResend`, `network/network_mutation_boundaries_test.go:TestNetworkMutationAcceptedIOCancellationAndSourceErrorsKeepEvidenceAndOwnedReset`.
- A representative accepted Create resets while an earlier role inventory is in flight: that earlier caller still receives its original result, but the old generation cannot repopulate cache. The next configured default lookup reads a new inventory and repeated lookup uses that new cache. This is the actual Create/Reset fixture, not separate mutation-race coverage for every CRUD method or every getter.
  Tests: `network/network_mutation_boundaries_test.go:TestNetworkMutationResetCannotPublishEarlierInflightDiscovery`.
- The Connection-cached high Network Service resets the same role cache after Create, Update and Delete. A later getter reads a new configured inventory, and Nova default NIC plus floating source/NAT allocation consume that same refreshed inventory without manual Reset. The fixture checks actual Nova and Neutron request fields and exactly two inventory reads.
  Tests: `connectiontest/connection_network_mutations_test.go:TestConnectionNetworkMutationRefreshesGetterDefaultNICAndFloatingRoles`.

Python/Go differences:

- The SDK exposes context-aware Connection getters returning owned native-plus-router/provider models and typed nullable pointers; Python returns mutable Network Resources and cached list/object identities. Ordinary generated Networks model behavior is unchanged. This bounded role model does not emulate all nullable descriptors, unknown properties, location or session state.
- Go configured-role validation accepts a matching name OR ID consistently. Pinned classification accepts either, but its subsequent external/internal family validation checks only selected names and can reject ID-only selectors. Go also rejects unsafe IDs and preserves missing/ambiguous and decoding errors rather than incidental dynamic-input behavior.
- Go caches only successful discovery, coalesces in-flight work, lets followers use their own contexts, and uses Reset generations to prevent a pre-reset flight from restoring the cache. Python serializes shared discovery with a lock and can swallow SDKException into sticky empty results. Owned returns and observable/retryable errors are deliberate Go policies.
- Aggregate results retain the source IPv4-then-IPv6 order and duplicates; family classification reflects routing configuration and network extensions, not discovered subnet address families. The NAT destination inference preserves the source last-network-with-gateway rule.
- Go flags are strict typed booleans; the YAML parser accepts boolean/string values via its explicit conversion policy. Pinned cloud use_* accessors store and return raw configured values, whose Python truthiness can differ. Both-disabled skips discovery; a single disabled flag does not remove that family after discovery.
- The Connection maps clean native missing-endpoint detection and the has_network configuration flag with official-service defaults (ServiceEnabled); Python's False for other lookup failures needs WithServiceLookupFailureAsUnavailable, and no version discovery applies to role discovery. Configured default NIC and automatic Ensure floating source use the shared successful role snapshot. Automatic NAT applies only without explicit port/fixed/NAT options and with multiple server-owned ports; strict unique IPv4-pair selection remains. These consumers do not implement private/floating-source flags or automatic IP need/skip decisions.
- For this no-argument named role-list getter, the concrete configured classification/default/order contract and owned RoleNetwork DTO result are the Go mapping. RoleNetwork uses native fields plus router/provider role extensions, not RawResource Body/Wire or full nullable/vendor/location/mutable Network descriptors. Discovery keeps its documented native200/204, linked-empty-page, success-only cache and explicit Reset policies.
- The original consumer-only difference describes the configured-default-NIC and Ensure role layer; separate later automatic-IP, private/source/address, pool/explicit-IP and Nova entries now exist and are compared in their own guides. Full Python has_service, CloudRegion loader overlays and Resource/session/cache lifecycle remain independent declared scopes rather than required inputs of this argument-free getter. This row does not promote scalar role getters, automatic-IP consumers or Network Proxy/Resource operations.

### `python:cloud/_network_common/NetworkCommonCloudMixin/use_external_network`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.UseExternalNetwork`
- `github.com/JSYoo5B/go-openstacksdk.WithNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk/network.PrepareNetworkRoleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network.WithExternalNetworkDiscovery`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.WithServiceLookupFailureAsUnavailable`

- Typed use-external/use-internal flags default enabled and preserve ordered replacement. Both disabled returns empty roles before client/catalog/HTTP access; disabling only external still discovers both families when internal is enabled. Connection nil/canceled context fails before lookup, clean pointer or value ErrEndpointNotFound yields empty roles, and other catalog errors remain inspectable.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- Owned configured-network slices can be reused after caller mutation. Nil options, empty names and multiple default/NAT-destination declarations fail before discovery; a later replacement does not hide an earlier invalid option. Explicit Reset invalidates a completed Connection role cache without making a mutation request.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- Role selectors are frozen with the same cloud/secure/public snapshot as native authentication. YAML family flags inherit routes_externally unless overridden; a typed WithNetworkRoles policy replaces the YAML role configuration and its default selector, and an empty typed replacement clears configured roles/default. Configured default NIC selection now consumes the same successful role snapshot; explicit typed default names retain their separate exact-name lookup.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- YAML boolean false and case-insensitive string false disable discovery before network HTTP. An explicit typed policy replaces those flags and restores discovery; a numeric YAML flag is rejected as ErrInvalidOption. Direct role discovery rejects nil or providerless clients before native HTTP.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleBooleanFlagsAndExplicitPolicyReplacement`, `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`.
- A has_network flag from WithServiceEnabled or clouds.yaml mirrors the has_service('network') check of _find_interesting_networks: disabled roles are empty without a network list even with configured roles, a typed enable overrides a YAML disable, and an enabled flag keeps discovery and configured NAT/default selection.
  Tests: `connectiontest/connection_network_service_disable_test.go:TestNetworkRolesHonourHasNetworkConfiguration`.
- The typed YAML use_external_network/use_internal_network policy agrees with Python raw-value truthiness for booleans, null, the empty string and any casing of 'true'; other strings disable discovery in Go though truthy in Python, and numbers or collections are configuration errors in Go while Python's normalize_keys makes numbers (even 0) truthy strings, as a table test records.
  Tests: `connectiontest/connection_network_role_flags_test.go:TestNetworkRoleUseFlagsCompareYAMLValuesWithPythonTruthiness`.
- The selected clouds.yaml regions entry overlays its non-null values on the merged cloud settings before role, default-network and has_<service> configuration is read: whole networks lists, use_* and has_network flags change per region. Without WithRegion the first entry (or the one named by region_name) is used; an unknown explicit region and malformed entries are configuration errors.
  Tests: `connectiontest/connection_cloud_region_config_test.go:TestCloudRegionValuesOverlayNetworkRoleConfiguration`.
- A profile absent from clouds-public.yaml (or with no such file) resolves to the embedded, unmodified pinned vendor profile: rackspace disables has_network and sets secgroup_source under the cloud's own settings, limestonenetworks supplies per-region networks and default interface, and native authentication receives the profile's auth settings. A clouds-public.yaml entry wins; shutdown and URL profiles are configuration errors, deprecated profiles load without warnings, and embedded files match their documented hashes.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudBuiltinVendorProfilesFeedRoleConfiguration`, `internal/cloudvendors/vendors_test.go:TestProfilesResolveStatusNestedAndRemoteNames`, `internal/cloudvendors/vendors_test.go:TestEmbeddedFilesMatchREADMEHashes`.
- Without OS_CLOUD, Python's environment-variable cloud is mapped for role settings: OS_INTERNAL_NETWORK/OS_EXTERNAL_NETWORK legacy roles, OS_USE_* flags (typed string policy) and an OS_PROFILE built-in vendor profile shape discovery, string OS_HAS_* values and shutdown profiles are configuration errors, and a typed WithNetworkRoles policy still replaces the environment roles.
  Tests: `connectiontest/connection_cloud_env_config_test.go:TestEnvironmentCloudSuppliesNetworkRoleConfiguration`.
- WithServiceLookupFailureAsUnavailable maps has_service returning False when the Network endpoint lookup raises: roles are empty and the lookup is retried on the next call, while the default returns the lookup error and cancellation is returned under either policy.
  Tests: `connectiontest/connection_service_lookup_test.go:TestServiceLookupFailureAsUnavailableMapsHasServiceFalse`.

Python/Go differences:

- Go flags are strict typed booleans. The YAML parser agrees with Python truthiness of the raw value for booleans, null, the empty string and any casing of 'true'; other strings are false in Go but truthy in Python, and numeric or collection values are rejected instead of being truthiness-tested (Python's normalize_keys turns numbers, even 0, into truthy strings). Pinned cloud use_* accessors return the raw configured value; Go returns a bool. Both-disabled skips discovery; a single disabled flag does not remove that family after discovery.
- Go caches only successful discovery, coalesces in-flight work, lets followers use their own contexts, and uses Reset generations to prevent a pre-reset flight from restoring the cache. Python serializes shared discovery with a lock and can swallow SDKException into sticky empty results. Owned returns and observable/retryable errors are deliberate Go policies.
- The Connection maps clean native missing-endpoint detection and the has_network configuration flag with official-service defaults (ServiceEnabled); Python's False for other lookup failures needs WithServiceLookupFailureAsUnavailable, and no version discovery applies to role discovery. Configured default NIC and automatic Ensure floating source use the shared successful role snapshot. Automatic NAT applies only without explicit port/fixed/NAT options and with multiple server-owned ports; strict unique IPv4-pair selection remains. These consumers do not implement private/floating-source flags or automatic IP need/skip decisions.
- Region values follow get_one's overlay for SDK-owned settings only; native authentication and endpoint parsing do not see them. Without WithRegion, an entry named by region_name is preferred over the first entry so that the values match the native endpoint region, whereas Python ignores region_name when regions is present. Typed Connect options (WithNetworkRoles, WithServiceEnabled) replace the configured policy for their concern instead of overlaying single keys like get_one keyword arguments.
- has_service('network') is mapped by ServiceEnabled (has_network flag, official-service default) plus the endpoint lookup. Python turns every lookup exception into False; Go returns lookup failures other than a missing endpoint unless WithServiceLookupFailureAsUnavailable is set, and always returns cancellation. Role discovery passes no version, so no version-range discovery applies, and Go emits no Python debug log.
- Configuration sources: clouds.yaml with secure.yaml, clouds-public.yaml or embedded built-in vendor profiles, region values, and the OS_* environment cloud when OS_CLOUD is unset. The environment cloud feeds only SDK-owned role, default-network and service settings; native authentication still uses Gophercloud's environment parser, and Python argparse/keyword arguments map to typed Connect options.
- A clouds.yaml profile named by URL, including the nested URLs of the built-in ovh and vexxhost profiles, is rejected as a configuration error instead of being fetched from the vendor's well-known document at connect time; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- The consumers of the role snapshot (create_server, add_ips_to_server, get_active_server, wait_for_server) are reviewed under those operations; this operation covers discovery and the returned role.

### `python:cloud/_network_common/NetworkCommonCloudMixin/use_internal_network`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.UseInternalNetwork`
- `github.com/JSYoo5B/go-openstacksdk.WithNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk/network.PrepareNetworkRoleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network.WithInternalNetworkDiscovery`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetNetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk.WithServiceLookupFailureAsUnavailable`

- Typed use-external/use-internal flags default enabled and preserve ordered replacement. Both disabled returns empty roles before client/catalog/HTTP access; disabling only external still discovers both families when internal is enabled. Connection nil/canceled context fails before lookup, clean pointer or value ErrEndpointNotFound yields empty roles, and other catalog errors remain inspectable.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- Owned configured-network slices can be reused after caller mutation. Nil options, empty names and multiple default/NAT-destination declarations fail before discovery; a later replacement does not hide an earlier invalid option. Explicit Reset invalidates a completed Connection role cache without making a mutation request.
  Tests: `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRoleGettersShareOwnedSnapshotAndReset`, `connectiontest/connection_network_roles_test.go:TestConnectionNetworkRolesFlagsMissingEndpointAndPreflight`.
- Role selectors are frozen with the same cloud/secure/public snapshot as native authentication. YAML family flags inherit routes_externally unless overridden; a typed WithNetworkRoles policy replaces the YAML role configuration and its default selector, and an empty typed replacement clears configured roles/default. Configured default NIC selection now consumes the same successful role snapshot; explicit typed default names retain their separate exact-name lookup.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- YAML boolean false and case-insensitive string false disable discovery before network HTTP. An explicit typed policy replaces those flags and restores discovery; a numeric YAML flag is rejected as ErrInvalidOption. Direct role discovery rejects nil or providerless clients before native HTTP.
  Tests: `connectiontest/connection_network_roles_test.go:TestConnectNetworkRoleBooleanFlagsAndExplicitPolicyReplacement`, `network/roles_test.go:TestNetworkRoleOptionsValidateBeforeDiscoveryAndSnapshotSlices`.
- A has_network flag from WithServiceEnabled or clouds.yaml mirrors the has_service('network') check of _find_interesting_networks: disabled roles are empty without a network list even with configured roles, a typed enable overrides a YAML disable, and an enabled flag keeps discovery and configured NAT/default selection.
  Tests: `connectiontest/connection_network_service_disable_test.go:TestNetworkRolesHonourHasNetworkConfiguration`.
- The typed YAML use_external_network/use_internal_network policy agrees with Python raw-value truthiness for booleans, null, the empty string and any casing of 'true'; other strings disable discovery in Go though truthy in Python, and numbers or collections are configuration errors in Go while Python's normalize_keys makes numbers (even 0) truthy strings, as a table test records.
  Tests: `connectiontest/connection_network_role_flags_test.go:TestNetworkRoleUseFlagsCompareYAMLValuesWithPythonTruthiness`.
- The selected clouds.yaml regions entry overlays its non-null values on the merged cloud settings before role, default-network and has_<service> configuration is read: whole networks lists, use_* and has_network flags change per region. Without WithRegion the first entry (or the one named by region_name) is used; an unknown explicit region and malformed entries are configuration errors.
  Tests: `connectiontest/connection_cloud_region_config_test.go:TestCloudRegionValuesOverlayNetworkRoleConfiguration`.
- A profile absent from clouds-public.yaml (or with no such file) resolves to the embedded, unmodified pinned vendor profile: rackspace disables has_network and sets secgroup_source under the cloud's own settings, limestonenetworks supplies per-region networks and default interface, and native authentication receives the profile's auth settings. A clouds-public.yaml entry wins; shutdown and URL profiles are configuration errors, deprecated profiles load without warnings, and embedded files match their documented hashes.
  Tests: `internal/cloudconfig/cloudconfig_test.go:TestCloudBuiltinVendorProfilesFeedRoleConfiguration`, `internal/cloudvendors/vendors_test.go:TestProfilesResolveStatusNestedAndRemoteNames`, `internal/cloudvendors/vendors_test.go:TestEmbeddedFilesMatchREADMEHashes`.
- Without OS_CLOUD, Python's environment-variable cloud is mapped for role settings: OS_INTERNAL_NETWORK/OS_EXTERNAL_NETWORK legacy roles, OS_USE_* flags (typed string policy) and an OS_PROFILE built-in vendor profile shape discovery, string OS_HAS_* values and shutdown profiles are configuration errors, and a typed WithNetworkRoles policy still replaces the environment roles.
  Tests: `connectiontest/connection_cloud_env_config_test.go:TestEnvironmentCloudSuppliesNetworkRoleConfiguration`.
- WithServiceLookupFailureAsUnavailable maps has_service returning False when the Network endpoint lookup raises: roles are empty and the lookup is retried on the next call, while the default returns the lookup error and cancellation is returned under either policy.
  Tests: `connectiontest/connection_service_lookup_test.go:TestServiceLookupFailureAsUnavailableMapsHasServiceFalse`.

Python/Go differences:

- Go flags are strict typed booleans. The YAML parser agrees with Python truthiness of the raw value for booleans, null, the empty string and any casing of 'true'; other strings are false in Go but truthy in Python, and numeric or collection values are rejected instead of being truthiness-tested (Python's normalize_keys turns numbers, even 0, into truthy strings). Pinned cloud use_* accessors return the raw configured value; Go returns a bool. Both-disabled skips discovery; a single disabled flag does not remove that family after discovery.
- Go caches only successful discovery, coalesces in-flight work, lets followers use their own contexts, and uses Reset generations to prevent a pre-reset flight from restoring the cache. Python serializes shared discovery with a lock and can swallow SDKException into sticky empty results. Owned returns and observable/retryable errors are deliberate Go policies.
- The Connection maps clean native missing-endpoint detection and the has_network configuration flag with official-service defaults (ServiceEnabled); Python's False for other lookup failures needs WithServiceLookupFailureAsUnavailable, and no version discovery applies to role discovery. Configured default NIC and automatic Ensure floating source use the shared successful role snapshot. Automatic NAT applies only without explicit port/fixed/NAT options and with multiple server-owned ports; strict unique IPv4-pair selection remains. These consumers do not implement private/floating-source flags or automatic IP need/skip decisions.
- Region values follow get_one's overlay for SDK-owned settings only; native authentication and endpoint parsing do not see them. Without WithRegion, an entry named by region_name is preferred over the first entry so that the values match the native endpoint region, whereas Python ignores region_name when regions is present. Typed Connect options (WithNetworkRoles, WithServiceEnabled) replace the configured policy for their concern instead of overlaying single keys like get_one keyword arguments.
- has_service('network') is mapped by ServiceEnabled (has_network flag, official-service default) plus the endpoint lookup. Python turns every lookup exception into False; Go returns lookup failures other than a missing endpoint unless WithServiceLookupFailureAsUnavailable is set, and always returns cancellation. Role discovery passes no version, so no version-range discovery applies, and Go emits no Python debug log.
- Configuration sources: clouds.yaml with secure.yaml, clouds-public.yaml or embedded built-in vendor profiles, region values, and the OS_* environment cloud when OS_CLOUD is unset. The environment cloud feeds only SDK-owned role, default-network and service settings; native authentication still uses Gophercloud's environment parser, and Python argparse/keyword arguments map to typed Connect options.
- A clouds.yaml profile named by URL, including the nested URLs of the built-in ovh and vexxhost profiles, is rejected as a configuration error instead of being fetched from the vendor's well-known document at connect time; see Deliberate runtime differences in docs/sdk-support-ledger.md.
- The consumers of the role snapshot (create_server, add_ips_to_server, get_active_server, wait_for_server) are reviewed under those operations; this operation covers discovery and the returned role.

## Configuration sources

`Connect` reads role configuration from the selected clouds.yaml cloud, merged with secure.yaml and its profile. Without `OS_CLOUD` (and without explicit authentication options) it reads Python's environment-variable cloud instead.

- Profiles: a `profile` (or legacy `cloud`) found in clouds-public.yaml is used; otherwise the embedded, unmodified built-in vendor profiles of the pinned Python SDK are consulted, with the cloud's own settings taking precedence. Native authentication then receives that built-in profile too. A `shutdown` profile and a profile given by URL (including the nested URLs of `ovh` and `vexxhost`, which Python fetches) are configuration errors; deprecated profiles load without a warning.
- Environment cloud: every `OS_*` variable except the `OS_TEST`, `OS_STD` and `OS_LOG` families becomes a lower-case string setting, as in `_get_os_environ`, so `OS_EXTERNAL_NETWORK`, `OS_INTERNAL_NETWORK`, `OS_USE_EXTERNAL_NETWORK`, `OS_USE_INTERNAL_NETWORK`, `OS_SECGROUP_SOURCE` and `OS_PROFILE` apply. An empty variable is a present setting. String `OS_HAS_*` values are rejected because the flag must be a boolean. Authentication still comes from Gophercloud's environment parser.

- Regions: when the cloud (or, without `regions` and `region_name`, its profile) has a `regions` list, the selected entry's non-null `values` replace the merged settings before roles are read, as `OpenStackConfig.get_one` does. A `networks` list in the values replaces the whole list. `WithRegion` must name an entry; without it the entry named by `region_name` is used when present, otherwise the first entry. Region values only reach SDK-owned settings, not native authentication or endpoint parsing.
- `has_network`: `WithServiceEnabled("network", false)` or `has_network: false` makes every role getter return empty results without a request, and makes the floating IP helpers use Nova even with the Neutron source, mirroring `has_service('network')`. Unset, the official `network` type is enabled.
- `use_external_network`/`use_internal_network`: YAML booleans, null, the empty string and any casing of `"true"` behave like Python's truthiness of the raw value. Other strings are false in Go but truthy in Python; numbers and lists are rejected by `Connect` (Python turns numbers, even 0, into truthy strings).
- Lookup failures: a missing Network endpoint always means no Neutron. Other endpoint lookup failures are returned unless `WithServiceLookupFailureAsUnavailable(true)` is set, which maps Python's `has_service` returning False: roles are empty (retried on the next call) and the floating IP helpers use Nova. Cancellation is always returned.
- Typed `WithNetworkRoles` and `WithServiceEnabled` options replace the configured policy for their concern.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
for net in conn.get_external_ipv4_networks():
    print(net.id, net.name)
print(conn.get_nat_source())
print(conn.get_nat_destination())
print(conn.get_default_network())
```

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/network"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := run(ctx); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context) error {
    conn, err := sdk.Connect(ctx,
        sdk.WithCloud("dev"),
        sdk.WithNetworkRoles(
            network.WithConfiguredNetworks(
                network.ConfiguredNetwork{
                    Name: "public",
                    RoutesIPv4Externally: true,
                    RoutesIPv6Externally: true,
                    NATSource: true,
                },
                network.ConfiguredNetwork{
                    Name: "private",
                    NATDestination: true,
                    DefaultInterface: true,
                },
            ),
        ),
    )
    if err != nil {
        return err
    }
    fmt.Printf("external=%t internal=%t\n",
        conn.UseExternalNetwork(), conn.UseInternalNetwork())

    roles, err := conn.GetNetworkRoles(ctx)
    if err != nil {
        return err
    }
    for _, net := range roles.ExternalIPv4 {
        fmt.Printf("external IPv4: %s %s router_external=%t provider=%q\n",
            net.ID, net.Name, net.RouterExternal, net.ProviderPhysicalNetwork)
    }
    printNetwork("NAT source", roles.NATSource)
    printNetwork("NAT destination", roles.NATDestination)

    defaultNetwork, err := conn.GetDefaultNetwork(ctx)
    if err != nil {
        return err
    }
    printNetwork("default", defaultNetwork)

    conn.ResetNetworkRoles()
    fresh, err := conn.GetNetworkRoles(ctx)
    if err != nil {
        return err
    }
    fmt.Printf("fresh external IPv4 networks: %d\n", len(fresh.ExternalIPv4))
    return nil
}

func printNetwork(label string, net *network.RoleNetwork) {
    if net == nil {
        fmt.Printf("%s: none\n", label)
        return
    }
    fmt.Printf("%s: %s %s\n", label, net.ID, net.Name)
}
```

```go
service, err := conn.Network(ctx)
if err != nil {
    return err
}
roles, err := service.Roles.Discover(ctx)
if err != nil {
    return err
}
_ = roles
service.Roles.Reset()
```

```python
import openstack

conn = openstack.connect(cloud="dev")
print(conn.get_default_network())
server = conn.create_server(
    name="web", image="ubuntu", flavor="c2", wait=True, auto_ip=True,
)
print(server.id)
```

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    if err := run(ctx); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud("dev"))
    if err != nil {
        return err
    }
    roles, err := conn.GetNetworkRoles(ctx)
    if err != nil {
        return err
    }
    if roles.DefaultNetwork != nil {
        fmt.Printf("default NIC: %s\n", roles.DefaultNetwork.ID)
    }
    service, err := conn.Compute(ctx)
    if err != nil {
        return err
    }
    result, err := service.Servers.CreateWithFloatingIP(ctx,
        compute.CreateServerWithFloatingIPRequest{
            Server: compute.CreateServerRequest{
                Name: "web",
                Image: resource.Name("ubuntu"),
                Flavor: resource.Name("c2"),
            },
        },
    )
    if err != nil {
        if result != nil && result.Server != nil {
            fmt.Printf("known server: %s\n", result.Server.ID)
        }
        if result != nil && result.Assignment != nil && result.Assignment.FloatingIP != nil {
            fmt.Printf("known floating IP: %s\n", result.Assignment.FloatingIP.ID)
        }
        return err
    }
    fmt.Printf("server=%s floating IPv4=%s\n",
        result.Server.ID, result.Assignment.FloatingIP.FloatingIP)
    return nil
}
```

## References

- [Server Automatic Ip](../compute/server-automatic-ip.md)
- [Create With Automatic Floating Ip](../compute/create-with-automatic-floating-ip.md)
- [Server Addresses](../compute/server-addresses.md)
- [Connection Network Role Consumers Test](../connectiontest/connection_network_role_consumers_test.go)
- [Floating Ip Roles Test](floating_ip_roles_test.go)
- [Connection Server Network Roles Test](../connectiontest/connection_server_network_roles_test.go)
- [CreateWithAutomaticFloatingIP](../compute/create-with-automatic-floating-ip.md)
- [Network Mutations](network-mutations.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L95-L432)
- [Cloud Region](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/config/cloud_region.py#L1427-L1495)
- [Config Loader](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/config/loader.py#L733-L790)
- [Server Ip Dispatch](../compute/server-ip-dispatch.md)
- [Nova backend](../compute/server-nova-floating-ip.md)
- [ready/wait](../compute/server-ready.md)
- [Roles Test](roles_test.go)
- [Connection Network Roles Test](../connectiontest/connection_network_roles_test.go)
