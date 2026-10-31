# Floating Ip Available

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "os"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
    "github.com/JSYoo5B/go-openstacksdk/network"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml entry")
    source := flag.String("source", "", "cloud default, neutron, nova or none")
    pool := flag.String("network", "public", "network name or literal Nova pool")
    server := flag.String("server-id", "", "optional server for new Neutron allocation")
    project := flag.String("project", "", "optional Neutron reuse filter only")
    strict := flag.Bool("strict", false, "omit Nova view compatibility aliases")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := run(ctx, *cloud, *source, *pool, *server, *project, *strict); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func run(ctx context.Context, cloud, source, pool, serverID, project string, strict bool) error {
    conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud),
        sdk.WithMicroversion(sdk.Compute, "2.35"))
    if err != nil {
        return err
    }
    input := compute.AvailableFloatingIPRequest{}
    if pool != "" {
        input.Networks = []resource.Ref{resource.Name(pool)}
    }
    if serverID != "" {
        input.Server = resource.ID(serverID)
    }
    options := []compute.AvailableFloatingIPOption{
        compute.WithAvailableIPTimeout(time.Minute),
        compute.WithAvailableIPStrict(strict),
    }
    if source != "" {
        options = append(options, compute.WithAvailableIPSource(compute.FloatingIPSource(source)))
    }
    if project != "" {
        options = append(options, compute.WithAvailableIPNetworkOptions(
            network.WithAvailableProject(project)))
    }
    result, err := conn.AvailableFloatingIP(ctx, input, options...)
    if result != nil {
        summary := map[string]any{
            "backend": result.Backend, "id": result.ID, "address": result.Address,
            "reused": result.Reused, "allocated": result.Allocated,
        }
        if record := result.FloatingIP; record != nil {
            summary["normalization_source"] = record.NormalizationSource
            summary["normalized"] = record.Normalized
            if record.Resource != nil {
                summary["resource"] = record.Resource.Body
            }
            if record.Wire != nil {
                summary["wire"] = record.Wire.Body
            }
        }
        if result.FallbackError != nil {
            summary["fallback_error"] = result.FallbackError.Error()
        }
        encoder := json.NewEncoder(os.Stdout)
        encoder.SetIndent("", "  ")
        if encodeErr := encoder.Encode(summary); encodeErr != nil {
            return errors.Join(err, encodeErr)
        }
    }
    return err
}
```

```python
import openstack

conn = openstack.connect(cloud="dev")
ip = conn.available_floating_ip(network="public")
print(ip["id"], ip["floating_ip_address"])

server = conn.compute.find_server("web-01", ignore_missing=False)
ip = conn.available_floating_ip(network="public", server=server)
```

## References

- [IP helper](server-ip-helpers.md)
- [Ensure·readiness](server-ip-dispatch.md)
- [Floating Ip Available](../network/floating-ip-available.md)
- [Floating Ip Create](floating-ip-create.md)
- [Server Nova Floating Ip](server-nova-floating-ip.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L608-L773)
- [Nova allocation+compat GET](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L951-L975)
- [normalization](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1732-L1830)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Server Ip Dispatch](server-ip-dispatch.md)
- [Delete](floating-ip-delete.md)
- [Floating Ip Queries](floating-ip-queries.md)
- [Connection Floating Ip Available Nova Test](../connectiontest/connection_floating_ip_available_nova_test.go)
- [Connection Floating Ip Available Neutron Test](../connectiontest/connection_floating_ip_available_neutron_test.go)
- [Floating Ip Allocate Test](../network/floating_ip_allocate_test.go)
