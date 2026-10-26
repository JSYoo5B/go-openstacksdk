# Server Network Interfaces

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
  Tests: `compute/server_networks_test.go:TestServerNICWirePreservesCombinationsOrderAndExplicitEmptyTag`, `connection_server_nics_test.go:TestConnectionServerNICIDsNeedNoNetworkEndpoint`.
- Connection resolves exact network/port names and preserves missing/ambiguous/403 causes without creating a server. Resolver failures, empty/unsafe returned IDs and cancellation stop before Nova.
  Tests: `connection_server_nics_test.go:TestConnectionServerNICsResolveExactNamesAndOwnInputs`, `connection_server_nics_test.go:TestConnectionServerNICNameFailuresPreventNovaCreation`, `compute/server_networks_test.go:TestServerNICResolverFailuresAndCancellationStopBeforeNova`.
- The last valid WithNetworks/NIC/mode selection replaces the previous selection, so discarded names or tag gates have no effect. Options own input slices and Optional tag values and can be reused concurrently.
  Tests: `compute/server_networks_test.go:TestServerNICSelectionUsesLastValidOption`, `compute/server_networks_test.go:TestServerNICOptionsOwnSlicesTagsAndConcurrentReuse`, `connection_server_nics_test.go:TestConnectionServerNICsResolveExactNamesAndOwnInputs`.
- Explicit mode auto/none requires selected Compute 2.37+ without discovery. Every present tag, including empty or null, follows require_microversion(compute, "2.42"): a selected version below 2.42 fails before dependency lookup, and otherwise the advertised range must contain 2.42 (an unselected client then raises only that POST to 2.42) or Create returns ErrUnsupported before the POST. A non-clean discovery failure is returned before the POST. Invalid options/Refs fail before dependency lookup; the shared client version is preserved.
  Tests: `compute/server_networks_test.go:TestServerNICInvalidOptionsFailBeforeDependencyLookup`, `compute/server_networks_test.go:TestServerNICWirePreservesCombinationsOrderAndExplicitEmptyTag`, `compute/server_networks_test.go:TestServerNICTagRequiresAdvertisedRange`.
- Without explicit NICs, mode or a Connection/YAML default, Create follows supports_microversion(compute, "2.37") from openstack/utils.py: the advertised Compute range must contain 2.37 and a selected client version must be at least 2.37. Then networks is auto; an unselected client raises only this POST to 2.37 and leaves the shared client unset. A selection below 2.37 omits networks without discovery; a range below 2.37, a clean 404/405 discovery or a selection beyond the advertised range omits the key. One discovery GET per create; Connection negotiation reuses its recorded advertised bounds without another discovery or Neutron lookup.
  Tests: `compute/server_networks_test.go:TestServerNICDefaultsModesAndVersionBoundaries`, `connection_server_nics_test.go:TestConnectionNegotiatedVersionControlsDefaultNetworkAuto`, `connection_network_role_consumers_test.go:TestConnectionDefaultNICEmptyRolesUseNovaNetworkDefault`.
- NIC choices preserve existing/new volume boot mapping and false delete-on-termination. An ACTIVE wait failure returns the created BUILD server and the error without server/port/volume cleanup.
  Tests: `compute/server_networks_test.go:TestServerNICsPreserveVolumeBootAndCreatedServerOnWaitFailure`.
- Connection owns a default-network Ref and explicit disable policy; the last option wins, invalid Refs remain errors, explicit NICs/modes bypass the default callback and Neutron endpoint lookup, and default IDs work for image/existing-volume/new-volume boots without altering their mappings.
  Tests: `connection_default_network_test.go:TestConnectionDefaultNetworkIDsOptionsAndBootSources`, `connection_default_network_test.go:TestConnectionDefaultNetworkDisableAndExplicitSelections`, `connection_default_network_test.go:TestConnectionDefaultNetworkOptionValidationAndConcurrentReuse`, `compute/default_network_test.go:TestServerDefaultNetworkIsSkippedByExplicitNICsAndPreflight`.
- Typed default names resolve exactly; missing/ambiguous/403 failures, malformed callback Refs and cancellation stop before server creation. Explicit typed default-name resolution remains fresh; configured defaults share successful role discovery and failed discovery is not cached; callbacks and options are request-local, and wait failure preserves the created server without cleanup.
  Tests: `connection_default_network_test.go:TestConnectionDefaultNetworkNameFailuresAndRetry`, `compute/default_network_test.go:TestServerDefaultNetworkDependencyFailureCancellationAndValidation`, `compute/default_network_test.go:TestServerDefaultNetworkDoesNotLeakBetweenRequestsAndPreservesWaitFailure`.
- Cloud/secure/public files are snapshotted once for native auth/endpoint/TLS and SDK metadata. Network lists replace at each precedence layer, [] clears inheritance, null/duplicate defaults/invalid flags are configuration errors, and empty/null metadata profile stops network inheritance while retaining native auth fallback.
  Tests: `connection_cloud_config_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connection_cloud_config_test.go:TestCloudNetworkConfigurationSearchAndReadFailures`, `connection_cloud_config_test.go:TestCloudNetworkConfigurationValidation`.
- YAML default name matches exact network name OR ID across unfiltered complete pagination. Cross-kind collisions and duplicate names are ambiguous; missing/unsafe IDs, a later-page 403 and cancellation preserve errors and prevent Nova POST. Connection override/disable and explicit Create selection bypass the configured lookup.
  Tests: `connection_cloud_network_test.go:TestConnectConfiguredDefaultNetworkMatchesNamesOrIDsAcrossAllPages`, `connection_cloud_network_test.go:TestConnectDefaultNetworkOverridesDisableAndExplicitCreateSelection`.
- The explicit compound workflow shares server input/body preparation with Create, validates server/IP options and external Ref before runtime lookup, and requires a concrete Network service and known/explicit reuse owner before Nova POST. Connection provides the lazy getter; invalid or missing services/scope return errors without server mutation.
  Tests: `compute/server_floating_ip_test.go:TestServerFloatingIPPreflightPrecedesCreationAndLookups`, `connection_server_floating_ip_test.go:TestConnectionServerFloatingIPMissingNetworkStopsBeforeNovaPOST`.
- After one Nova POST the original server ID is retained. Mandatory server waiting checks the actual ID and ACTIVE status even with a custom status attribute before destination port/available-IP requests or floating-IP mutation. Shared role discovery can already have run for the configured default NIC. It then reuses or allocates one IPv4 assignment and always verifies actual IP ACTIVE; prepared policy replacement cannot disable this mandatory wait.
  Tests: `connection_server_floating_ip_test.go:TestConnectionServerFloatingIPOrdersWaitsAndPreservesPartialResults`, `connection_server_floating_ip_test.go:TestConnectionServerFloatingIPPollsServerBeforeStartingIP`.
- One overall five-minute context limit includes service preparation, lookup/creation, both waits and IP work. A parent earlier deadline wins; individual unlimited wait cannot remove it. The overall unlimited option removes the SDK deadline when there is no parent deadline, and a later workflow timeout restores it without restarting a budget per stage. Deadline errors retain the known server and assignment.
  Tests: `connection_server_floating_ip_test.go:TestConnectionServerFloatingIPUsesOneDeadlineAcrossBothWaits`.
- Server wait failures retain the POST server; after actual ACTIVE verification IP selection failures retain the server and late guarded PUT/wait/malformed response failures retain server plus reused/allocated assignment. No fallback allocation, deletion or synthesized Nova addresses is performed.
  Tests: `connection_server_floating_ip_test.go:TestConnectionServerFloatingIPOrdersWaitsAndPreservesPartialResults`, `connection_server_floating_ip_test.go:TestConnectionServerFloatingIPUsesOneDeadlineAcrossBothWaits`.
- Image, existing-volume and new-image-volume boot preserve default private NIC and boot deletion policy independently of the external floating network. Wrapper option slices are snapshotted and concurrent reuse returns independent server/IP objects with the captured metadata/reuse configuration.
  Tests: `connection_server_floating_ip_test.go:TestConnectionServerFloatingIPKeepsBootAndDefaultNetworkPolicies`, `connection_server_floating_ip_test.go:TestConnectionServerFloatingIPOptionsSnapshotConcurrentReuse`.
- PrepareEnsureActive binds the recorded reuse owner to a copy and forces ACTIVE wait while preserving poll/timeout/progress configuration. Changing recorded authentication afterward does not change that bound owner; the original plain prepared policy still resolves the new owner per Ensure call and keeps its original no-wait behavior.
  Tests: `network/floating_ip_prepare_test.go:TestFloatingIPPrepareActiveBindsCopyAndPreservesWaitPolicy`.
- A configured default NIC consumes the same successful network/subnet role snapshot as Connection getters. Caller changes to a returned default cannot alter creation; consecutive creates reuse the successful discovery and Reset refreshes the chosen default ID for the next create.
  Tests: `connection_network_role_consumers_test.go:TestConnectionDefaultNICConsumesRoleCacheAndReset`, `connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Configured default discovery validates the shared snapshot, so subnet403 or another configured role missing stops before Nova POST even when the default itself exists. Failed attempts are not cached; a subsequent success is reused. Later network-page errors, name-or-ID collisions, unsafe IDs and cancellation retain their established pre-create errors.
  Tests: `connection_network_role_consumers_test.go:TestConnectionDefaultNICRoleErrorsStopCreationAndRetry`, `connection_cloud_network_test.go:TestConnectConfiguredDefaultNetworkMatchesNamesOrIDsAcrossAllPages`.
- Both-disabled discovery or a clean pointer/value missing Network endpoint yields no configured default, so Nova auto follows the advertised 2.37 check (including an unselected client raised to 2.37 for that POST). Explicit default ID/disable and explicit NIC/mode bypass configured role discovery; explicit typed default names keep their exact-name resolution lane.
  Tests: `connection_network_role_consumers_test.go:TestConnectionDefaultNICEmptyRolesUseNovaNetworkDefault`, `connection_cloud_network_test.go:TestConnectDefaultNetworkOverridesDisableAndExplicitCreateSelection`, `connection_default_network_test.go:TestConnectionDefaultNetworkNameFailuresAndRetry`.
- Connection CreateWithFloatingIP uses a single successful configured role inventory for its default NIC, floating source and NAT destination, including a configured source without router:external. Role lookup finishes before Nova POST; actual server ACTIVE precedes port and floating-IP requests, and actual floating-IP ACTIVE remains mandatory. A role403 returns no created resource; a later port403 preserves the known ACTIVE Server with nil assignment and the original HTTP cause, without allocation or cleanup.
  Tests: `connection_server_network_roles_test.go:TestConnectionCreateWithFloatingIPSharesDefaultSourceAndNATRoles`.
- Owned high Service CreateNetwork/UpdateNetwork/DeleteNetwork and the existing Networks.Delete adapter invalidate shared roles after accepted mutations. The actual matrix primes a configured default role, observes a new inventory after success or high-delete clean404 after a found lookup, and retains the original inventory after403/409/412, failed no-option lookup, or direct Collection404. A successful no-option Update performs its lookup and reset without PUT; initial missing ID/Name high Delete returns false without mutation or reset.
  Tests: `network/network_mutations_test.go:TestNetworkMutationSuccessFailureAndMissingCacheBoundaries`, `network/network_mutations_test.go:TestDeleteNetworkInitialMissingDoesNotMutateOrReset`.
- The Connection-cached high Network Service resets the same role cache after Create, Update and Delete. A later getter reads a new configured inventory, and Nova default NIC plus floating source/NAT allocation consume that same refreshed inventory without manual Reset. The fixture checks actual Nova and Neutron request fields and exactly two inventory reads.
  Tests: `connection_network_mutations_test.go:TestConnectionNetworkMutationRefreshesGetterDefaultNICAndFloatingRoles`.
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
  Tests: `connection_server_automatic_create_test.go:TestConnectionAutomaticCreateOwnsNamedDependencyPagesAndLifetime`, `internal/nativefind/resolve_name_test.go:TestGuardedCreationNameResolverPagesAndRetainsSource`.
- Configured default NIC selection in the automatic compound consumes one guarded role snapshot before Nova, while an existing-floating-address skip needs no IP candidate or mutation. Nested source registries keep ancestor causes and isolate child-only bindings from the original parent context.
  Tests: `connection_server_automatic_create_test.go:TestConnectionAutomaticCreateSharesConfiguredDefaultRoleSnapshot`, `internal/rest/operation_guard_test.go:TestNestedOperationSourcesRetainAncestorsAndIsolateChildBindings`.
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
  Tests: `connection_server_create_lifecycle_test.go:TestConnectionCreateLifecycleResolvesGroupVolumesAndDeletesOnlyWhenSelected`.
- WithCreateRefresh performs exactly one GET of the created server after an accepted POST and returns the refreshed model; ERROR returns ServerFaultError with Python's fault message and the refreshed Server, while a failed or mismatched GET returns the POST server with the error and never repeats the POST. Combining it with WithWait fails before POST.
  Tests: `compute/server_create_lifecycle_test.go:TestServerCreateRefreshPerformsOneGetAndReportsFault`.
- Explicit block_device_mapping_v2 entries serialize every per-mapping field (null boot_index, delete_on_termination, device name/type, disk bus, guest format, volume size/type, tag, no_device) and are placed before generated local-image and data-volume mappings, as Python appends generated mappings to supplied ones. Legacy block_device_mapping entries and trusted_image_certificates are sent as given. Legacy plus v2, invalid source/destination types and empty lists fail as ErrInvalidOption, and tag (2.42), volume type (2.67) and certificates (2.63) fail as ErrUnsupported before POST.
  Tests: `compute/server_create_mappings_test.go:TestServerCreateExplicitLegacyMappingsAndTrustedCertificates`.
- CreateReservation sends one POST with return_reservation_id true and the requested min/max count and returns only the reservation ID with response metadata. A server-shaped or empty reservation response is an error without a second POST, and WithWait or WithCreateRefresh fail before POST.
  Tests: `compute/server_create_mappings_test.go:TestServerCreateReservationReturnsOnlyReservationID`.
- CreateRefreshedServer performs one POST and one GET, copies the creation adminPass onto the refreshed Server and returns the add_server_interfaces-equivalent ExpandServerInterfaces view. A refreshed ERROR server returns ServerFaultError pointing at the returned Server without expansion, and WithWait fails before POST. The Connection wrapper uses the Connection-owned Compute and Network services.
  Tests: `compute/server_create_mappings_test.go:TestCreateRefreshedServerExpandsAndCopiesAdminPass`, `connection_server_create_refresh_test.go:TestConnectionCreateRefreshedServerPostsOnceAndExpands`.
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
  Tests: `compute/server_networks_test.go:TestServerNICWirePreservesCombinationsOrderAndExplicitEmptyTag`, `connection_server_nics_test.go:TestConnectionServerNICIDsNeedNoNetworkEndpoint`.
- Connection resolves exact network/port names and preserves missing/ambiguous/403 causes without creating a server. Resolver failures, empty/unsafe returned IDs and cancellation stop before Nova.
  Tests: `connection_server_nics_test.go:TestConnectionServerNICsResolveExactNamesAndOwnInputs`, `connection_server_nics_test.go:TestConnectionServerNICNameFailuresPreventNovaCreation`, `compute/server_networks_test.go:TestServerNICResolverFailuresAndCancellationStopBeforeNova`.
- The last valid WithNetworks/NIC/mode selection replaces the previous selection, so discarded names or tag gates have no effect. Options own input slices and Optional tag values and can be reused concurrently.
  Tests: `compute/server_networks_test.go:TestServerNICSelectionUsesLastValidOption`, `compute/server_networks_test.go:TestServerNICOptionsOwnSlicesTagsAndConcurrentReuse`, `connection_server_nics_test.go:TestConnectionServerNICsResolveExactNamesAndOwnInputs`.
- Mode auto/none requires selected Compute 2.37+, and every present tag, including empty or null, requires selected 2.42+. Invalid options/Refs/version gates fail before dependency lookup; the shared client version and Nova headers are preserved.
  Tests: `compute/server_networks_test.go:TestServerNICInvalidOptionsFailBeforeDependencyLookup`, `compute/server_networks_test.go:TestServerNICWirePreservesCombinationsOrderAndExplicitEmptyTag`.
- Without explicit NICs, mode or a Connection/YAML default, Create follows supports_microversion(compute, "2.37") from openstack/utils.py: the advertised Compute range must contain 2.37 and a selected client version must be at least 2.37. Then networks is auto; an unselected client raises only this POST to 2.37 and leaves the shared client unset. A selection below 2.37 omits networks without discovery; a range below 2.37, a clean 404/405 discovery or a selection beyond the advertised range omits the key. One discovery GET per create; Connection negotiation reuses its recorded advertised bounds without another discovery or Neutron lookup.
  Tests: `compute/server_networks_test.go:TestServerNICDefaultsModesAndVersionBoundaries`, `connection_server_nics_test.go:TestConnectionNegotiatedVersionControlsDefaultNetworkAuto`.
- NIC choices preserve existing/new volume boot mapping and false delete-on-termination. An ACTIVE wait failure returns the created BUILD server and the error without server/port/volume cleanup.
  Tests: `compute/server_networks_test.go:TestServerNICsPreserveVolumeBootAndCreatedServerOnWaitFailure`.
- Connection owns a default-network Ref and explicit disable policy; the last option wins, invalid Refs remain errors, explicit NICs/modes bypass the default callback and Neutron endpoint lookup, and default IDs work for image/existing-volume/new-volume boots without altering their mappings.
  Tests: `connection_default_network_test.go:TestConnectionDefaultNetworkIDsOptionsAndBootSources`, `connection_default_network_test.go:TestConnectionDefaultNetworkDisableAndExplicitSelections`, `connection_default_network_test.go:TestConnectionDefaultNetworkOptionValidationAndConcurrentReuse`, `compute/default_network_test.go:TestServerDefaultNetworkIsSkippedByExplicitNICsAndPreflight`.
- Typed default names resolve exactly; missing/ambiguous/403 failures, malformed callback Refs and cancellation stop before server creation. Explicit typed default-name resolution remains fresh; configured defaults share successful role discovery and failed discovery is not cached; callbacks and options are request-local, and wait failure preserves the created server without cleanup.
  Tests: `connection_default_network_test.go:TestConnectionDefaultNetworkNameFailuresAndRetry`, `compute/default_network_test.go:TestServerDefaultNetworkDependencyFailureCancellationAndValidation`, `compute/default_network_test.go:TestServerDefaultNetworkDoesNotLeakBetweenRequestsAndPreservesWaitFailure`.
- Cloud/secure/public files are snapshotted once for native auth/endpoint/TLS and SDK metadata. Network lists replace at each precedence layer, [] clears inheritance, null/duplicate defaults/invalid flags are configuration errors, and empty/null metadata profile stops network inheritance while retaining native auth fallback.
  Tests: `connection_cloud_config_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connection_cloud_config_test.go:TestCloudNetworkConfigurationSearchAndReadFailures`, `connection_cloud_config_test.go:TestCloudNetworkConfigurationValidation`.
- YAML default name matches exact network name OR ID across unfiltered complete pagination. Cross-kind collisions and duplicate names are ambiguous; missing/unsafe IDs, a later-page 403 and cancellation preserve errors and prevent Nova POST. Connection override/disable and explicit Create selection bypass the configured lookup.
  Tests: `connection_cloud_network_test.go:TestConnectConfiguredDefaultNetworkMatchesNamesOrIDsAcrossAllPages`, `connection_cloud_network_test.go:TestConnectDefaultNetworkOverridesDisableAndExplicitCreateSelection`.
- A configured default NIC consumes the same successful network/subnet role snapshot as Connection getters. Caller changes to a returned default cannot alter creation; consecutive creates reuse the successful discovery and Reset refreshes the chosen default ID for the next create.
  Tests: `connection_network_role_consumers_test.go:TestConnectionDefaultNICConsumesRoleCacheAndReset`, `connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Configured default discovery validates the shared snapshot, so subnet403 or another configured role missing stops before Nova POST even when the default itself exists. Failed attempts are not cached; a subsequent success is reused. Later network-page errors, name-or-ID collisions, unsafe IDs and cancellation retain their established pre-create errors.
  Tests: `connection_network_role_consumers_test.go:TestConnectionDefaultNICRoleErrorsStopCreationAndRetry`, `connection_cloud_network_test.go:TestConnectConfiguredDefaultNetworkMatchesNamesOrIDsAcrossAllPages`.
- Both-disabled discovery or a clean pointer/value missing Network endpoint yields no configured default and keeps Nova auto at selected2.37 or omits networks when unset. Explicit default ID/disable and explicit NIC/mode bypass configured role discovery; explicit typed default names keep their exact-name resolution lane.
  Tests: `connection_network_role_consumers_test.go:TestConnectionDefaultNICEmptyRolesUseNovaNetworkDefault`, `connection_cloud_network_test.go:TestConnectDefaultNetworkOverridesDisableAndExplicitCreateSelection`, `connection_default_network_test.go:TestConnectionDefaultNetworkNameFailuresAndRetry`.
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
  Tests: `compute/server_networks_test.go:TestServerNICWirePreservesCombinationsOrderAndExplicitEmptyTag`, `connection_server_nics_test.go:TestConnectionServerNICIDsNeedNoNetworkEndpoint`.
- Connection resolves exact network/port names and preserves missing/ambiguous/403 causes without creating a server. Resolver failures, empty/unsafe returned IDs and cancellation stop before Nova.
  Tests: `connection_server_nics_test.go:TestConnectionServerNICsResolveExactNamesAndOwnInputs`, `connection_server_nics_test.go:TestConnectionServerNICNameFailuresPreventNovaCreation`, `compute/server_networks_test.go:TestServerNICResolverFailuresAndCancellationStopBeforeNova`.
- The last valid WithNetworks/NIC/mode selection replaces the previous selection, so discarded names or tag gates have no effect. Options own input slices and Optional tag values and can be reused concurrently.
  Tests: `compute/server_networks_test.go:TestServerNICSelectionUsesLastValidOption`, `compute/server_networks_test.go:TestServerNICOptionsOwnSlicesTagsAndConcurrentReuse`, `connection_server_nics_test.go:TestConnectionServerNICsResolveExactNamesAndOwnInputs`.
- Mode auto/none requires selected Compute 2.37+, and every present tag, including empty or null, requires selected 2.42+. Invalid options/Refs/version gates fail before dependency lookup; the shared client version and Nova headers are preserved.
  Tests: `compute/server_networks_test.go:TestServerNICInvalidOptionsFailBeforeDependencyLookup`, `compute/server_networks_test.go:TestServerNICWirePreservesCombinationsOrderAndExplicitEmptyTag`.
- Without explicit NICs, mode or a Connection/YAML default, Create follows supports_microversion(compute, "2.37") from openstack/utils.py: the advertised Compute range must contain 2.37 and a selected client version must be at least 2.37. Then networks is auto; an unselected client raises only this POST to 2.37 and leaves the shared client unset. A selection below 2.37 omits networks without discovery; a range below 2.37, a clean 404/405 discovery or a selection beyond the advertised range omits the key. One discovery GET per create; Connection negotiation reuses its recorded advertised bounds without another discovery or Neutron lookup.
  Tests: `compute/server_networks_test.go:TestServerNICDefaultsModesAndVersionBoundaries`, `connection_server_nics_test.go:TestConnectionNegotiatedVersionControlsDefaultNetworkAuto`.
- NIC choices preserve existing/new volume boot mapping and false delete-on-termination. An ACTIVE wait failure returns the created BUILD server and the error without server/port/volume cleanup.
  Tests: `compute/server_networks_test.go:TestServerNICsPreserveVolumeBootAndCreatedServerOnWaitFailure`.
- Connection owns a default-network Ref and explicit disable policy; the last option wins, invalid Refs remain errors, explicit NICs/modes bypass the default callback and Neutron endpoint lookup, and default IDs work for image/existing-volume/new-volume boots without altering their mappings.
  Tests: `connection_default_network_test.go:TestConnectionDefaultNetworkIDsOptionsAndBootSources`, `connection_default_network_test.go:TestConnectionDefaultNetworkDisableAndExplicitSelections`, `connection_default_network_test.go:TestConnectionDefaultNetworkOptionValidationAndConcurrentReuse`, `compute/default_network_test.go:TestServerDefaultNetworkIsSkippedByExplicitNICsAndPreflight`.
- Typed default names resolve exactly; missing/ambiguous/403 failures, malformed callback Refs and cancellation stop before server creation. Explicit typed default-name resolution remains fresh; configured defaults share successful role discovery and failed discovery is not cached; callbacks and options are request-local, and wait failure preserves the created server without cleanup.
  Tests: `connection_default_network_test.go:TestConnectionDefaultNetworkNameFailuresAndRetry`, `compute/default_network_test.go:TestServerDefaultNetworkDependencyFailureCancellationAndValidation`, `compute/default_network_test.go:TestServerDefaultNetworkDoesNotLeakBetweenRequestsAndPreservesWaitFailure`.
- Cloud/secure/public files are snapshotted once for native auth/endpoint/TLS and SDK metadata. Network lists replace at each precedence layer, [] clears inheritance, null/duplicate defaults/invalid flags are configuration errors, and empty/null metadata profile stops network inheritance while retaining native auth fallback.
  Tests: `connection_cloud_config_test.go:TestCloudNetworkConfigurationInheritanceAndFrozenNativeSettings`, `connection_cloud_config_test.go:TestCloudNetworkConfigurationSearchAndReadFailures`, `connection_cloud_config_test.go:TestCloudNetworkConfigurationValidation`.
- YAML default name matches exact network name OR ID across unfiltered complete pagination. Cross-kind collisions and duplicate names are ambiguous; missing/unsafe IDs, a later-page 403 and cancellation preserve errors and prevent Nova POST. Connection override/disable and explicit Create selection bypass the configured lookup.
  Tests: `connection_cloud_network_test.go:TestConnectConfiguredDefaultNetworkMatchesNamesOrIDsAcrossAllPages`, `connection_cloud_network_test.go:TestConnectDefaultNetworkOverridesDisableAndExplicitCreateSelection`.
- A configured default NIC consumes the same successful network/subnet role snapshot as Connection getters. Caller changes to a returned default cannot alter creation; consecutive creates reuse the successful discovery and Reset refreshes the chosen default ID for the next create.
  Tests: `connection_network_role_consumers_test.go:TestConnectionDefaultNICConsumesRoleCacheAndReset`, `connection_network_roles_test.go:TestConnectNetworkRoleConfigurationAndTypedOverrideSupplyDefaultNIC`.
- Configured default discovery validates the shared snapshot, so subnet403 or another configured role missing stops before Nova POST even when the default itself exists. Failed attempts are not cached; a subsequent success is reused. Later network-page errors, name-or-ID collisions, unsafe IDs and cancellation retain their established pre-create errors.
  Tests: `connection_network_role_consumers_test.go:TestConnectionDefaultNICRoleErrorsStopCreationAndRetry`, `connection_cloud_network_test.go:TestConnectConfiguredDefaultNetworkMatchesNamesOrIDsAcrossAllPages`.
- Both-disabled discovery or a clean pointer/value missing Network endpoint yields no configured default and keeps Nova auto at selected2.37 or omits networks when unset. Explicit default ID/disable and explicit NIC/mode bypass configured role discovery; explicit typed default names keep their exact-name resolution lane.
  Tests: `connection_network_role_consumers_test.go:TestConnectionDefaultNICEmptyRolesUseNovaNetworkDefault`, `connection_cloud_network_test.go:TestConnectDefaultNetworkOverridesDisableAndExplicitCreateSelection`, `connection_default_network_test.go:TestConnectionDefaultNetworkNameFailuresAndRetry`.
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

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(compute_api_version="2.42")
port = conn.network.find_port("app-port", ignore_missing=False)
server = conn.create_server(
    name="web-01", image="ubuntu", flavor="small",
    nics=[
        {"net-name": "private", "fixed_ip": "10.0.0.10", "tag": "app"},
        {"port-id": port.id, "tag": "existing-port"},
    ],
    auto_ip=False, wait=False,
)
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
	"github.com/JSYoo5B/go-openstacksdk/request"
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
	conn, err := sdk.Connect(ctx, sdk.WithMicroversion(sdk.Compute, "2.42"))
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	server, err := service.Servers.Create(ctx, compute.CreateServerRequest{
		Name: "web-01", Image: resource.Name("ubuntu"), Flavor: resource.Name("small"),
	}, compute.WithNetworkInterfaces(
		compute.ServerNetworkInterface{Network: resource.Name("private"), FixedIP: "10.0.0.10", Tag: request.Present("app")},
		compute.ServerNetworkInterface{Port: resource.Name("app-port"), Tag: request.Present("existing-port")},
	))
	if err != nil {
		return err
	}
	fmt.Println(server.ID)
	return nil
}
```

## References

- [Reference](README.md)
- [Server Default Network](server-default-network.md)
- [API Microversion History](https://docs.openstack.org/nova/2026.1/reference/api-microversion-history.html)
- [WithMicroversionRange/WithLatestMicroversion](../docs/microversions.md)
- [ Compute](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L1074-L1154)
- [Utils](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L242-L312)
- [Network Roles](../network/network-roles.md)
- [Create With Floating Ip](create-with-floating-ip.md)
- [Network Mutations](../network/network-mutations.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
