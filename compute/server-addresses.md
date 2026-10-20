# Server Addresses

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_compute/ComputeCloudMixin/get_server_public_ip`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetServerPublicIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetServerPublicIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.ExpandServerInterfaces`

- External use gate precedes AccessIPv4, configured external role NAME, any floating IPv4, literal public label and final CPython3.13 not-private IPv4 classification. Floating rows precede fixed rows in a selected step and empty selected candidates preserve priority.
  Tests: `compute/server_address_selection_test.go:TestServerPublicAddressOrderingRolesFloatingAndAccessShortCircuit`.
- Final classification rejects documentation, loopback, linklocal, benchmark and reserved IPv4 blocks; CPython3.13 exceptions .9/.10, shared100.64/10 and multicast are accepted, malformed and IPv4-mappedIPv6 strings are skipped. This final fallback follows parsed actual family rather than declared wire version.
  Tests: `compute/server_address_selection_test.go:TestServerPublicAddressFinalFallbackUsesPython313Classification`.
- Concrete address options are validated before lazy roles/services. Disabled internal/external getters skip malformed addresses; public AccessIPv4 skips parsing and role discovery. Connection metadata getters need no Compute endpoint.
  Tests: `compute/server_address_selection_test.go:TestServerAddressUseFlagsSkipMalformedAddressesAndNetwork`, `compute/server_address_selection_test.go:TestServerAddressValidationPrecedesDependencyAndPropagatesRoleFailure`, `connection_server_addresses_test.go:TestConnectionServerAddressShortcutsAndInvalidInputsSelectNoEndpoint`.
- An owned typed address view retains nil map versus empty map and null versus empty network lists, unknown JSON fields, MAC presence, family and wire row order; view mutation does not change the native server. Default interface tries IPv6 before IPv4 only under localIPv6/nonforced policy; an empty selected default stops family iteration but uses final interface fallback. Private/public/access fields and no-privateIPv6 semantics are independently scoped.
  Tests: `compute/server_address_expansion_test.go:TestServerAddressExpansionDefaultPrivateIPv6AndOwnedSnapshot`, `compute/server_address_expansion_test.go:TestServerAddressExpansionEmptyDefaultStopsFamilyBeforeInterfaceFallback`, `compute/server_address_expansion_test.go:TestServerAddressEmptyCandidateDoesNotFallThroughRoleOrFloatingPriority`.
- TCP reachability probes multiple public-cloud public candidates, skips that probing direction on a private cloud, falls back to the first candidate after all failures, and stops on the parent deadline. Candidate port/budget are concrete options; local TCP fixtures exercise the actual dial behavior.
  Tests: `compute/server_address_selection_test.go:TestServerAddressReachabilitySelectionDirectionFallbackAndContext`.
- Independent expansion supplements exactly ACTIVE servers without a nonIPv6 floating row. Neutron ports and per-port floating IPs traverse empty linked pages, locally filter server/port scope, map duplicate fixed addresses to the final network label, retain DOWN IPs, synthesize port MAC fields and mark Supplemental provenance without modifying input. Source none skips; source Nova uses unfiltered per-port /os-floating-ips, local port matching, canonical-null/legacy normalization and clean404 empty.
  Tests: `compute/server_address_expansion_test.go:TestServerAddressSupplementGatesSkipNetworkAndOnlyNonIPv6Floating`, `compute/server_address_expansion_test.go:TestServerAddressSupplementPagesScopeMACAndDuplicateFixedNetwork`, `compute/server_address_expansion_test.go:TestServerAddressSupplementNovaSourceLocalFilterAliasesAnd404`, `compute/server_address_expansion_test.go:TestServerAddressSupplementPortMACMissingNullAndEmptyAreDistinct`.
- Supplemental clean HTTP errors preserve earlier synthesized addresses with a diagnostic; explicitly accepted204 is terminal empty. Accepted malformed/Close/context/network source and Compute collection replacement failures preserve a partial view with an error and never mutate the server or allocate/delete resources.
  Tests: `compute/server_address_expansion_test.go:TestServerAddressSupplementPartialCleanHTTPFailureAndTerminal204`, `compute/server_address_expansion_test.go:TestServerAddressSupplementAcceptedFailuresKeepPartialAndGuardComputeOwner`.
- Connection and its cached Compute share the same network role cache, while supplemental reads remain fresh each call. Missing Neutron endpoints yield empty roles without dropping supplied Nova private/IPv6 addresses. File profiles/base/secure snapshots, normalized booleans, global environment/client preference and percloud force/prefer priority, source-specific HTTP routing and typed whole-policy replacement are exercised.
  Tests: `connection_server_addresses_test.go:TestConnectionServerAddressFacadesShareRoleCacheAndFreshSupplementation`, `connection_server_addresses_test.go:TestConnectionServerAddressMissingNeutronRetainsNovaAndIPv6`, `connection_server_addresses_test.go:TestConnectServerAddressYAMLSourceAndTypedReplacement`, `connection_server_address_config_test.go:TestCloudServerAddressConfigurationPriorityNormalizedBooleansAndValidation`, `connection_server_address_config_test.go:TestCloudServerAddressConfigurationUsesFrozenProfileBaseSecureLayers`.
- Cancellation during address JSON snapshot, role getter and network getter preserves both context.Canceled and its explicit cancellation cause, plus a distinct dependency failure. Supplemental fixed null/missing stays distinct from a literal empty fixed address; canonical null never falls through to legacy. A selected null floating value fails the typed address schema with original accepted response and partial view.
  Tests: `compute/server_address_expansion_test.go:TestServerAddressCalculationPreservesCancellationCauseAtEachDependencyBoundary`, `compute/server_address_expansion_test.go:TestServerAddressSupplementNullableFixedAndFloatingFieldsDoNotBecomeEmptyAddresses`.

Python/Go differences:

- The getter returns (string,error), with empty string for no usable result, and accepts the native Nova Server snapshot. Independent ExpandServerInterfaces returns owned typed fields and address/provenance snapshots; it does not mutate the supplied Server or implicitly expand Get/Create/Wait.
- Native address map loses Python dictionary insertion order. SDK uses sorted network keys, preserves row order and offers WithAddressNetworkOrder including an empty key. Wire version must be present and addr must be a string; null addr is rejected instead of Python dynamic None behavior.
- Concrete private/force/prefer booleans normalize bool/string rather than percloud raw Python truthiness. WithServerAddressPolicy replaces file/environment policy after loading; FromProvider reads no cloud/env policy. Unknown sources are rejected. CPython3.13 IPv4 classification is stable while the pinned SDK permits multiple Python runtimes.
- TCP selection uses context-bound Go DNS/dial fallback, candidate budget and typed port options; single candidates are unprobed and all failed candidates retain first-candidate fallback. Parent cancellation, callback/source and accepted IO/decode failures remain observable. Ordinary supplemental clean HTTP/DNS/connect failures are diagnostics; source may swallow broader SDKException.
- The independently callable Compute automatic consumer now reuses bounded owned address/role evidence for lazy need/skip and conditional Neutron assignment, then obtains actual raw Nova observation separately. This supplied-native-model getter remains a pure address calculation and is not a Create/Get/Wait lifecycle or full cloud model mapping.

Unresolved boundaries:

- Complete the connected Python Server Resource/normalization, nullable/dynamic field behavior, all inherited descriptor/adapter/session/location/error/retry semantics beyond this supplied-native-model getter and the independently tested owned view. This row is not full Compute Proxy/Resource or cloud SDK parity.
- Close full cloud client/profile/environment-only raw-string loading, service/session availability and runtime-dependent address ordering/probing/error behaviors beyond the typed file/environment policy and tested lazy Connection scope.
- Integrate address expansion and the independently callable bounded automatic Neutron decision/assignment/raw-observation lane into the complete cloud Get/Create/get_active_server/Wait lifecycle. Complete pool/explicit-IP priority, Nova mutation/fallback and full service/source/model/session/cleanup contracts beyond these separate APIs. Supplemental metadata remains need/skip evidence rather than raw Nova observation, and existing explicit CreateWithFloatingIP has not become the full automatic cloud workflow.

### `python:cloud/_compute/ComputeCloudMixin/get_server_private_ip`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetServerPrivateIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetServerPrivateIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.ExpandServerInterfaces`

- Internal use gate precedes first floatingIPv4 MAC selection, all configured internal fixed passes, tag-unrestricted passes, literal private fixed/MAC and final private fallback without MAC. Missing/null and explicit-empty MAC remain distinct; comparison is case-sensitive and floatingIPv6 MAC does not constrain private IPv4.
  Tests: `compute/server_address_selection_test.go:TestServerPrivateAddressRoleMACAndLegacyFallback`.
- Concrete address options are validated before lazy roles/services. Disabled internal/external getters skip malformed addresses; public AccessIPv4 skips parsing and role discovery. Connection metadata getters need no Compute endpoint.
  Tests: `compute/server_address_selection_test.go:TestServerAddressUseFlagsSkipMalformedAddressesAndNetwork`, `compute/server_address_selection_test.go:TestServerAddressValidationPrecedesDependencyAndPropagatesRoleFailure`, `connection_server_addresses_test.go:TestConnectionServerAddressShortcutsAndInvalidInputsSelectNoEndpoint`.
- An owned typed address view retains nil map versus empty map and null versus empty network lists, unknown JSON fields, MAC presence, family and wire row order; view mutation does not change the native server. Default interface tries IPv6 before IPv4 only under localIPv6/nonforced policy; an empty selected default stops family iteration but uses final interface fallback. Private/public/access fields and no-privateIPv6 semantics are independently scoped.
  Tests: `compute/server_address_expansion_test.go:TestServerAddressExpansionDefaultPrivateIPv6AndOwnedSnapshot`, `compute/server_address_expansion_test.go:TestServerAddressExpansionEmptyDefaultStopsFamilyBeforeInterfaceFallback`, `compute/server_address_expansion_test.go:TestServerAddressEmptyCandidateDoesNotFallThroughRoleOrFloatingPriority`.
- TCP reachability probes multiple public-cloud public candidates, skips that probing direction on a private cloud, falls back to the first candidate after all failures, and stops on the parent deadline. Candidate port/budget are concrete options; local TCP fixtures exercise the actual dial behavior.
  Tests: `compute/server_address_selection_test.go:TestServerAddressReachabilitySelectionDirectionFallbackAndContext`.
- Independent expansion supplements exactly ACTIVE servers without a nonIPv6 floating row. Neutron ports and per-port floating IPs traverse empty linked pages, locally filter server/port scope, map duplicate fixed addresses to the final network label, retain DOWN IPs, synthesize port MAC fields and mark Supplemental provenance without modifying input. Source none skips; source Nova uses unfiltered per-port /os-floating-ips, local port matching, canonical-null/legacy normalization and clean404 empty.
  Tests: `compute/server_address_expansion_test.go:TestServerAddressSupplementGatesSkipNetworkAndOnlyNonIPv6Floating`, `compute/server_address_expansion_test.go:TestServerAddressSupplementPagesScopeMACAndDuplicateFixedNetwork`, `compute/server_address_expansion_test.go:TestServerAddressSupplementNovaSourceLocalFilterAliasesAnd404`, `compute/server_address_expansion_test.go:TestServerAddressSupplementPortMACMissingNullAndEmptyAreDistinct`.
- Supplemental clean HTTP errors preserve earlier synthesized addresses with a diagnostic; explicitly accepted204 is terminal empty. Accepted malformed/Close/context/network source and Compute collection replacement failures preserve a partial view with an error and never mutate the server or allocate/delete resources.
  Tests: `compute/server_address_expansion_test.go:TestServerAddressSupplementPartialCleanHTTPFailureAndTerminal204`, `compute/server_address_expansion_test.go:TestServerAddressSupplementAcceptedFailuresKeepPartialAndGuardComputeOwner`.
- Connection and its cached Compute share the same network role cache, while supplemental reads remain fresh each call. Missing Neutron endpoints yield empty roles without dropping supplied Nova private/IPv6 addresses. File profiles/base/secure snapshots, normalized booleans, global environment/client preference and percloud force/prefer priority, source-specific HTTP routing and typed whole-policy replacement are exercised.
  Tests: `connection_server_addresses_test.go:TestConnectionServerAddressFacadesShareRoleCacheAndFreshSupplementation`, `connection_server_addresses_test.go:TestConnectionServerAddressMissingNeutronRetainsNovaAndIPv6`, `connection_server_addresses_test.go:TestConnectServerAddressYAMLSourceAndTypedReplacement`, `connection_server_address_config_test.go:TestCloudServerAddressConfigurationPriorityNormalizedBooleansAndValidation`, `connection_server_address_config_test.go:TestCloudServerAddressConfigurationUsesFrozenProfileBaseSecureLayers`.
- Cancellation during address JSON snapshot, role getter and network getter preserves both context.Canceled and its explicit cancellation cause, plus a distinct dependency failure. Supplemental fixed null/missing stays distinct from a literal empty fixed address; canonical null never falls through to legacy. A selected null floating value fails the typed address schema with original accepted response and partial view.
  Tests: `compute/server_address_expansion_test.go:TestServerAddressCalculationPreservesCancellationCauseAtEachDependencyBoundary`, `compute/server_address_expansion_test.go:TestServerAddressSupplementNullableFixedAndFloatingFieldsDoNotBecomeEmptyAddresses`.

Python/Go differences:

- The getter returns (string,error), with empty string for no usable result, and accepts the native Nova Server snapshot. Independent ExpandServerInterfaces returns owned typed fields and address/provenance snapshots; it does not mutate the supplied Server or implicitly expand Get/Create/Wait.
- Native address map loses Python dictionary insertion order. SDK uses sorted network keys, preserves row order and offers WithAddressNetworkOrder including an empty key. Wire version must be present and addr must be a string; null addr is rejected instead of Python dynamic None behavior.
- Concrete private/force/prefer booleans normalize bool/string rather than percloud raw Python truthiness. WithServerAddressPolicy replaces file/environment policy after loading; FromProvider reads no cloud/env policy. Unknown sources are rejected. CPython3.13 IPv4 classification is stable while the pinned SDK permits multiple Python runtimes.
- TCP selection uses context-bound Go DNS/dial fallback, candidate budget and typed port options; single candidates are unprobed and all failed candidates retain first-candidate fallback. Parent cancellation, callback/source and accepted IO/decode failures remain observable. Ordinary supplemental clean HTTP/DNS/connect failures are diagnostics; source may swallow broader SDKException.
- The independently callable Compute automatic consumer now reuses bounded owned address/role evidence for lazy need/skip and conditional Neutron assignment, then obtains actual raw Nova observation separately. This supplied-native-model getter remains a pure address calculation and is not a Create/Get/Wait lifecycle or full cloud model mapping.

Unresolved boundaries:

- Complete the connected Python Server Resource/normalization, nullable/dynamic field behavior, all inherited descriptor/adapter/session/location/error/retry semantics beyond this supplied-native-model getter and the independently tested owned view. This row is not full Compute Proxy/Resource or cloud SDK parity.
- Close full cloud client/profile/environment-only raw-string loading, service/session availability and runtime-dependent address ordering/probing/error behaviors beyond the typed file/environment policy and tested lazy Connection scope.
- Integrate address expansion and the independently callable bounded automatic Neutron decision/assignment/raw-observation lane into the complete cloud Get/Create/get_active_server/Wait lifecycle. Complete pool/explicit-IP priority, Nova mutation/fallback and full service/source/model/session/cleanup contracts beyond these separate APIs. Supplemental metadata remains need/skip evidence rather than raw Nova observation, and existing explicit CreateWithFloatingIP has not become the full automatic cloud workflow.

### `python:cloud/_network_common/NetworkCommonCloudMixin/add_ips_to_server`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.PlanServerFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk.Connection.EnsureServerFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.PlanServerFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.EnsureServerFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAutomaticIPEnabled`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAutomaticAddressOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAutomaticEnsureOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAutomaticIPTimeout`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithUnlimitedAutomaticIPTimeout`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAutomaticIPPollInterval`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAutomaticIPProgress`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.NewPlanner`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPPlanner.NetworkRoles`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPPlanner.PrepareEnsure`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPPlanner.Check`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPPool`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPAddresses`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetActiveServer`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.WaitForServer`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.CreateWithAutomaticFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk.Connection.GetActiveServer`
- `github.com/JSYoo5B/go-openstacksdk.Connection.WaitForServer`
- `github.com/JSYoo5B/go-openstacksdk.Connection.CreateWithAutomaticFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.AddIPsToServer`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.AddIPList`
- `github.com/JSYoo5B/go-openstacksdk.Connection.AddIPsToServer`
- `github.com/JSYoo5B/go-openstacksdk.Connection.AddIPList`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerIPAutomaticOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithServerIPWait`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPSource`
- `github.com/JSYoo5B/go-openstacksdk.WithServerAddressPolicy`
- `github.com/JSYoo5B/go-openstacksdk.WithMicroversion`

- The bounded existing-server automatic lane keeps policy skips separate from an undetermined/error decision. Disabled/source-none, private policy, enabled external AccessIPv4, any-family floating tags including an empty address, and explicit empty/null-row inventories return a clean reason without Compute/Network/role/owner work or assignment. Supplied address presence is tested separately from nil refresh.
  Tests: `compute/server_automatic_ip_test.go:TestAutomaticIPKnownSkipsAvoidAllServiceAndOwnerWork`.
- Nil addresses cause one owned raw Nova refresh; explicit empty addresses do not. Actual200/203 responses are accepted. Refreshed AccessIPv4 or floating evidence can skip; still-nil addresses return no-fixed. Missing404, malformed address rows, wrong server identity and ERROR fail before Network/IP work, retaining accepted response or known fault evidence instead of becoming a skip.
  Tests: `compute/server_automatic_ip_test.go:TestAutomaticIPNilOnlyRefreshPreservesActualEvidence`.
- A missing Network backend or requested Nova source produces Needed=true/Backend=nova. Strong Ensure rejects a supplied BUILD server before assignment; missing authenticated Compute capability returns ErrUnsupported. Completed absent external network or owned ports can skip; nonempty IPv6-only destination and denied port inventory remain undetermined errors. A semantic absence joined with accepted-response cancellation remains a failure with context cause and HTTP proof.
  Tests: `compute/server_automatic_ip_test.go:TestAutomaticIPBackendAndCompletedAbsenceStayDistinct`, `compute/server_automatic_ip_boundaries_test.go:TestAutomaticIPSelectionAbsenceWithAcceptedCancellationIsFailure`, `connection_server_automatic_ip_test.go:TestConnectionAutomaticIPMissingNetworkRequiresComputeForNovaExecution`.
- Existing Neutron association supplementation may add an owned Supplemental floating row and avoid allocation, but does not set Observed or make a Nova GET. Supplement403 remains an error with SupplementalError rather than a successful skip. Calculated public IPv4 can skip; an untagged private IPv4 can require assignment, and disabled external/internal role-use policies do not turn a supplied AccessIPv4 into external evidence.
  Tests: `compute/server_automatic_ip_test.go:TestAutomaticIPSupplementedExistingAddressSkipsAndFailuresStayVisible`, `compute/server_automatic_ip_boundaries_test.go:TestAutomaticIPComputedAddressAndRoleUseFlagsPreserveNeedPolicy`.
- One lazy FloatingIPPlanner supplies owned classification roles and the subsequent external/NAT/port selection from one role snapshot. Construction performs no HTTP or owner binding. Returned role edits and Reset do not alter that planner snapshot; a new planner reads the next inventory. Cold-role retries check the installed outer Compute guard and retain original503 while stopping before another physical role request.
  Tests: `network/floating_ip_planner_test.go:TestFloatingIPPlannerSharesOwnedClassificationAndSelectionSnapshot`, `network/floating_ip_planner_test.go:TestFloatingIPPlannerOuterGuardStopsColdRoleRetry`, `compute/server_automatic_ip_test.go:TestAutomaticIPColdRoleRetryChecksOuterComputeSource`.
- A needed Neutron execution preflights required raw Compute observation capability and URL before mutation. Missing Compute, invalid Compute URL and unknown reuse owner return the determined Decision without allocation or raw observation; unsupported capability/scope remains ErrUnsupported. Existing ACTIVE server is required for the assignment lane, and the prepared Neutron target is executed without reselection.
  Tests: `compute/server_automatic_ip_boundaries_test.go:TestAutomaticIPObservationCapabilityAndReuseOwnerPreflight`, `compute/server_automatic_ip_test.go:TestAutomaticIPExecutesFixedTargetAndObservesSecondRawFloatingIPv4`.
- Successful assignment uses the selected external network, server-owned port and fixed IPv4, requires actual Neutron IP ACTIVE, then observes the exact assigned version4/type=floating address in any raw Nova network row with an ACTIVE matching server identity. The target may be the second floating row. Reset during observation and sequential caller Server-ID edits do not move the fixed target; the progress callback receives a detached model so its ID/name/address edits do not alter the known result or original input address data.
  Tests: `compute/server_automatic_ip_test.go:TestAutomaticIPExecutesFixedTargetAndObservesSecondRawFloatingIPv4`.
- Fixed rows, version6 floating rows, another floating address, AccessIPv4 alone, BUILD status and empty raw addresses are not observation success. Cancellation preserves its custom cause, the allocated Assignment and last raw Server/fault evidence. Raw ERROR,404 and malformed rows fail instead of being retried by a blanket cloud exception policy; callback edits do not replace the retained Server.
  Tests: `compute/server_automatic_ip_test.go:TestAutomaticIPRawNonEvidenceCancelRetainsAssignmentAndLastServer`.
- Accepted malformed allocation retains Allocated=true with nil IP and actual201 proof. A false description-based status cannot bypass actual Neutron ACTIVE, and failure returns the known Server/Assignment before raw Nova observation. Accepted raw200 Close failure, cancellation or source change retains original known Server/allocated Assignment and actual response status/header/body with one raw request. Raw503 retry that replaces the Compute collection stops before resending and preserves original HTTP error and Assignment.
  Tests: `compute/server_automatic_ip_test.go:TestAutomaticIPAcceptedAllocationAndActualIPReadinessKeepPartialResources`, `compute/server_automatic_ip_boundaries_test.go:TestAutomaticIPAcceptedRawCloseFailuresRetainEvidenceWithoutRetry`, `compute/server_automatic_ip_boundaries_test.go:TestAutomaticIPRawRetrySourceChangePreservesAssignmentAndOriginalHTTPError`.
- Nil options, nonpositive timeout/poll interval, nil progress callback and invalid nested waiter options fail before selection or mutation. A parent deadline still bounds raw observation when the SDK whole-operation timeout is disabled, retaining the allocated Assignment. Planner nil/canceled preflight preserves custom context cause.
  Tests: `compute/server_automatic_ip_test.go:TestAutomaticIPStaticValidationAndDeadlineDoNotPermitMutation`, `network/floating_ip_planner_test.go:TestFloatingIPPlannerPreflightPreservesNilAndCancellation`.
- Connection applies installed owned address policy and per-call overlays lazily. Private/source-none/IPv6-floating/per-call-private skips make zero endpoint locator calls. The missing-Network execution fixture resolves Network then Compute, two locator calls in that order; absent Compute capability returns Needed/Backend=Nova and ErrUnsupported with neither Neutron nor Nova assignment. This is capability failure, not a blanket Nova execution prohibition.
  Tests: `connection_server_automatic_ip_test.go:TestConnectionAutomaticIPKnownSkipsUseFrozenPolicyWithoutEndpoints`, `connection_server_automatic_ip_test.go:TestConnectionAutomaticIPMissingNetworkRequiresComputeForNovaExecution`.
- The SDK-owned prepared Neutron plan retains a decoded accepted201/202 allocation after complete-body read, Close, source or custom-cancellation errors with actual status/header/body and original cause; no retry, ACTIVE wait, observer or cleanup follows. Decoded wrong-owner allocation remains evidence alongside joined invariant/Close failures. Truncated202 read and Close causes remain joined with the actual json.SyntaxError and Allocated=true/nil model. Reused PUT200 adopts a complete matching body beside Close failure, but wrong-ID/fixed responses preserve the original candidate and both processing and validation causes; revision zero remains protected. This is owned workflow evidence, not a change to generated native transport.
  Tests: `network/floating_ip_accepted_model_test.go:TestFloatingIPAcceptedAllocationKeepsDecodedModelAndProcessingCauses`, `network/floating_ip_accepted_model_test.go:TestFloatingIPAcceptedAssociationAdoptsOnlyMatchingModelOnCloseError`, `network/floating_ip_accepted_model_test.go:TestFloatingIPAcceptedReadAndDecodeErrorsRemainJoined`.
- In the bounded automatic Neutron GetActiveServer, standalone WaitForServer, automatic creation and EnsureServerFloatingIP workflows, an accepted allocation body plus Close failure retains known Server and decoded DOWN Assignment with Allocated=true, Observed=false and actual201 response/header proof. Get/Ensure do not start raw observation; Wait/Create retain the one completed raw readiness fetch. Each has one allocation and no follow-up IP/observer/fallback/cleanup request. These fixtures do not complete explicit IP/pool or Nova dispatch.
  Tests: `compute/server_accepted_assignment_test.go:TestServerIPWorkflowsKeepAcceptedAssignmentModelOnProcessingError`.
- The existing stronger EnsureServerFloatingIP explicit-address lane owns the supplied address slice and preserves order and duplicate entries. Each item completes attachment, actual Neutron ACTIVE and exact raw Nova target observation before the next item starts; a duplicate creates another attempt but skips an already-matching association PUT. Explicit addresses bypass disabled/private automatic policy. Attempts retain index, requested address, assignment, Completed and Observed. Aggregate Observed records successful sequential observations, not simultaneous survival of every prior address in the final Server: the final fixture raw response contains only the last target.
  Tests: `compute/server_ip_dispatch_test.go:TestServerIPDispatchPreservesOrderedDuplicateInputAndOneCompletedStepAtATime`.
- When the second explicit item fails with missing/ambiguous/403 lookup,412 association, wrong accepted address, accepted PUT Close/source error, IP cancellation or raw cancellation/source error, the first completed/observed assignment survives and the third lookup never starts. Attempts retain the failing item error and current partial assignment when known; the aggregate Assignment remains the latest known assignment and Server the latest accepted matching raw model. Native status or accepted200 response/header evidence and original validation/processing/custom-cause errors remain; no IP allocation, backend fallback or cleanup is attempted by this explicit list lane.
  Tests: `compute/server_ip_dispatch_test.go:TestServerIPDispatchSecondFailureRetainsFirstCompletionAndCurrentPartial`.
- In the tested Ensure consumer a selected pool wins over malformed explicit address strings regardless of selector option order, even with auto disabled, private cloud and supplied public/floating metadata. Pool reuse=true uses a free owned candidate and no POST; reuse=false sends one attached allocation with selected network/port/fixed/project fields. Pool zero clears a prior invalid pool, empty address options clear a prior malformed list, and only the final winning selector is validated before HTTP. This is the existing Ensure pool policy, not source optional-server unattached availability.
  Tests: `compute/server_ip_dispatch_test.go:TestServerIPDispatchPoolWinsIndependentOfOptionOrderAndIgnoresIPSelectors`, `compute/server_ip_dispatch_test.go:TestServerIPDispatchClearsSelectorsAndValidatesOnlyWinningSelector`.
- Diagnostic Plan returns ordered explicit Neutron attachment selections without mutation or raw observation, retaining earlier selections when a later address is missing. Explicit pool/list requests with Nova/None source, or configured Neutron with absent Network, can remain Needed with Nova diagnostic backend and nil Plan error. In the old fixture Compute is absent, so execution is ErrUnsupported before attempts or assignment; capable Nova execution is covered separately. Explicit requests are not converted to disabled/private automatic skips, and the fixture separately asserts Network dependency use.
  Tests: `compute/server_ip_dispatch_test.go:TestServerIPDispatchReadOnlyPlanKeepsOrderedSelectionsAndNeverClaimsMutation`, `compute/server_ip_dispatch_test.go:TestServerIPDispatchBackendAndSkippedAutomaticConditionsStayDistinct`.
- The same ordered explicit-address policy is consumed by typed GetActive async/sync, standalone Wait and automatic creation. Default GetActive async keeps Compute discovery lazy, returns both completed DOWN assignments with no IP readiness/raw observation; sync GetActive observes two targets, while Wait/Create perform one initial raw readiness fetch plus two observations and force actual Neutron ACTIVE despite EnsureNoWait. Creation preserves its original AdminPass separately from later Server models. The new success fixture uses image/flavor creation and explicit lists; it does not test all pool or volume/NIC combinations. GetActive BUILD returns nil, ERROR retains a failed-state result, and ACTIVE nil metadata returns the existing typed address error before dispatch and with no HTTP.
  Tests: `compute/server_ip_dispatch_workflow_test.go:TestServerIPDispatchAllWorkflowConsumersUseTheSameOrderedPolicy`, `compute/server_ip_dispatch_workflow_test.go:TestServerIPDispatchExplicitSelectorsPreserveGetReadinessBoundaries`.
- Typed automatic creation rejects selected malformed address/unsafe pool before POST. The historical known-unsupported legacy fixture includes a Neutron-only project override, so capability preflight rejects it before server POST or dependency HTTP; capable Nova backend is not rejected categorically. In the tested Neutron second-address failure, original Creation.AdminPass, latest raw Server, completed/observed first assignment and failing second attempt remain with ErrNotFound, no third address or allocation. Source wait=false creation remains separate; new bounded Nova creation has its own evidence.
  Tests: `compute/server_ip_dispatch_workflow_test.go:TestServerIPDispatchPureInvalidAndKnownUnsupportedCreateStopBeforePOST`, `compute/server_ip_dispatch_workflow_test.go:TestServerIPDispatchCreateRetainsCreationFirstIPAndSecondFailure`.
- The explicit Ensure fixture propagates one identical HTTP context deadline shorter than the parent across its items, then parent cancellation during the second raw observation retains both known assignments and stops the third. Missing mandatory raw observation capability fails before any lookup or mutation. Plan/Ensure snapshot supplied server ID/status before application options mutate them, and outer selector, address, Ensure and resource-wait option factories each execute once across a two-item operation. This deadline evidence checks context identity and cancellation; it does not wait for actual timer expiry.
  Tests: `compute/server_ip_dispatch_workflow_test.go:TestServerIPDispatchOneDeadlineBoundsAllItemsAndPreflightsObservation`, `compute/server_ip_dispatch_workflow_test.go:TestServerIPDispatchPinsSuppliedServerBeforeOptionsAndPreparesPolicyOnce`.
- An accepted203 second raw observation whose Close fails retains the newly decoded matching raw Server, both started attempts and current second assignment beside the original Close cause and actual status/header proof. The first attempt remains observed, the second and aggregate do not claim observation completion, and no third request starts.
  Tests: `compute/server_ip_dispatch_workflow_test.go:TestServerIPDispatchAcceptedRawCloseKeepsCurrentServerAndAllKnownIPs`.
- Connection GetActive async and standalone Wait delegate the same ordered explicit-address policy. The root fixture attaches two exact addresses from unscoped queries with revision0: async locates no Compute endpoint, sends no raw observation and performs one singular IP recheck per item; Wait lazily locates Compute once, performs one raw readiness plus two target observations and one additional IP readiness GET per item. Both attempts complete in input order with no allocation; Observed is false in async and true per sequential Wait item, not simultaneous final presence.
  Tests: `connection_server_ip_dispatch_test.go:TestConnectionExplicitIPDispatchSharesLazyAsyncAndOrderedWaitPolicies`.
- In Connection standalone Wait, changing the cached Compute.API during the accepted second association body Close retains the first completed/observed attempt, latest first-target Server and decoded second bound IP beside ErrInvalidOption and actual200 header proof. The second attempt remains incomplete/unobserved, no third lookup or second-item readiness/observation starts, and raw count remains readiness plus the first observation. This fixture guards a late source failure, not legacy backend fallback or full Resource policy.
  Tests: `connection_server_ip_dispatch_test.go:TestConnectionOrderedIPSourceFailureKeepsFirstObservationAndAcceptedSecondIP`.
- The standalone Connection Ensure automatic lane retains its lazily resolved Compute source registry across raw observation. A progress callback that clears cached Compute API, Servers or Flavors returns ErrInvalidOption with the known matching raw Server and allocated assignment, Observed=false, one allocation, one raw fetch and one callback; no subsequent fetch is allowed. This is a concrete source-ownership regression, not complete source configuration/session parity.
  Tests: `connection_server_automatic_sources_test.go:TestConnectionStandaloneAutomaticIPRetainsLazyComputeSourceGuard`.
- Standalone Service AddIPsToServer and positional AddIPList attach two exact existing IPv4 addresses in caller order from a supplied BUILD server with nil metadata. The default async lane keeps Compute discovery and raw observation absent, returns completed DOWN assignments with Observed=false, performs one singular selected-IP recheck per item, and starts the next lookup after accepted association. No allocation or server/IP readiness is required. Existing stronger Ensure/ready/create consumers retain their own ACTIVE contracts.
  Tests: `compute/server_ip_helpers_test.go:TestStandaloneServerIPAsyncPreservesOrderedAttachmentsWithoutReadiness`.
- Standalone WithServerIPWait(true) waits only for each exact version4 floating-tagged target in the matching raw server ID. In the direct-list fixture supplied SHUTOFF and raw BUILD or ERROR do not block successful observation; Neutron assignments remain DOWN with only the single association recheck, not an IP ACTIVE waiter. All attempts complete/observe in sequence. This is the independent helper policy, not the actual server/IP ACTIVE policy of Ensure, synchronous GetActive, Wait or creation.
  Tests: `compute/server_ip_helpers_test.go:TestStandaloneServerIPWaitObservesExactAddressWithoutServerOrIPActive`.
- The direct-list wire fixture observes one identical HTTP deadline across two items under the shared standalone helper budget: default near60seconds,17second override, earlier30second parent, and explicit unlimited retaining a three-minute parent deadline equal to every request deadline and later than60seconds. The unlimited assertion demonstrates removal of the default60second cap. This validates deadlines rather than waiting60seconds. A separate actual one-second observation expiry preserves the first assignment/incomplete attempt with DeadlineExceeded and stops before the second address lookup; no allocation/fallback/cleanup follows.
  Tests: `compute/server_ip_helpers_test.go:TestStandaloneServerIPDefaultsUseOneSixtySecondHTTPBudget`, `compute/server_ip_helpers_test.go:TestStandaloneServerIPExpiredObservationKeepsAssignmentAndStopsNextItem`.
- Positional AddIPList snapshots its server and owned address list before callbacks and remains authoritative over malformed optional list/pool selectors; source Neutron/None/Nova plus nil positional list is a clean explicit empty-list result with no dependencies/mutation/observation. The optional-list AddIPs dispatcher instead clears an empty selector and can reach the disabled automatic reason. Outer helper, nested automatic, destination and wait factories execute once even after the caller mutates original slices/server ID/status. Common nil/canceled/unsafe/IPv6/invalid-option inputs fail before HTTP, and missing mandatory raw capability fails before lookup/mutation.
  Tests: `compute/server_ip_helpers_test.go:TestStandaloneServerIPPositionalEmptyListNeverFallsThroughToAutomatic`, `compute/server_ip_helpers_test.go:TestStandaloneServerIPPositionalSelectionAndOptionsArePreparedOnce`, `compute/server_ip_helpers_test.go:TestStandaloneServerIPPreflightStopsBeforeAnyRequest`.
- In independent async and raw-wait list calls, a missing second IP or accepted second PUT Close failure retains the completed first item with its historical Observed state, latest known assignment and the failing second attempt; no third lookup starts. Missing retains the first assignment, while a complete accepted second body retains that bound IP beside the original Close cause and actual200 accepted-b header proof. Async raw count stays0 and wait mode retains only the first completed raw observation.
  Tests: `compute/server_ip_helpers_test.go:TestStandaloneServerIPSecondFailurePreservesAsyncAndObservedHistory`.
- Standalone AddIPsToServer uses the same winning pool priority with ignored malformed address syntax even for supplied BUILD/nil metadata and auto disabled. The async reuse=true fixture binds a free candidate with zero allocation POST; reuse=false sends one attached pool allocation. It returns one completed DOWN assignment/Observed=false and no raw observation. A separate already-associated pool fixture preserves supplied BUILD/nil metadata and returns Reused=true/Allocated=false without Compute resolution, raw fetch, association PUT, allocation POST or singular selected-IP GET. These bounded consumers do not implement source unattached-first optional-server availability or its nonreuse refresh/timing path.
  Tests: `compute/server_ip_helpers_test.go:TestStandaloneServerIPPoolUsesSamePriorityAndAsyncAllocationPolicy`, `compute/server_ip_helpers_test.go:TestStandaloneServerIPAlreadyAttachedAddressesAndPoolKeepComputeLazy`.
- Standalone automatic AddIPsToServer can allocate one DOWN assignment asynchronously from supplied BUILD/fixed metadata with no raw observation. When nil metadata requires refresh and an accepted203 matching BUILD body closes with an error, it retains the decoded refreshed Server and original response/Close evidence, sends one raw fetch and no allocation, and does not treat the processing error as a clean skip.
  Tests: `compute/server_ip_helpers_test.go:TestStandaloneServerIPAutomaticAsyncAndAcceptedRefreshModels`.
- Connection AddIPsToServer and AddIPList share ordered async/raw-wait behavior with supplied BUILD/nil metadata: async locates no Compute and performs no raw GET; raw wait resolves Compute once and observes two targets without IP readiness GETs, returning completed DOWN assignments. Empty positional AddIPList remains a clean no-op with no service discovery, assignment or attempts even with wait=true and ignored optional selectors. The empty fixture constructs FromProvider without endpoint overrides and installs an EndpointLocator that would fail for any service; its locator count remains0.
  Tests: `connection_server_ip_helpers_test.go:TestConnectionStandaloneIPHelpersShareLazyAsyncAndRawWait`, `connection_server_ip_helpers_test.go:TestConnectionStandaloneEmptyIPListAvoidsEveryServiceLookup`.
- In Connection standalone list raw-wait, replacing cached Compute.API during the accepted second association Close retains the first completed/observed attempt, last first-target Server and decoded second bound IP with ErrInvalidOption and actual200 accepted-b proof. The second remains incomplete, no third lookup or further raw fetch starts, and the original single first-target observation remains recorded.
  Tests: `connection_server_ip_helpers_test.go:TestConnectionStandaloneIPSourceFailureRetainsFirstObservationAndSecondIP`.
- Standalone async direct-list and pool calls reuse an already-associated IP with the exact selected port/fixed tuple from supplied BUILD/nil metadata without resolving Compute, raw observation, allocation POST or association PUT. Each returns one completed Reused=true/Allocated=false assignment with Observed=false and preserves nil Server.Addresses. Direct-list performs one singular selected-IP GET; the already-attached pool path performs none. This is bounded reuse behavior, not source preattached Compute refresh or full availability parity.
  Tests: `compute/server_ip_helpers_test.go:TestStandaloneServerIPAlreadyAttachedAddressesAndPoolKeepComputeLazy`.
- The bounded legacy Nova standalone list lane consumes an unfiltered IP inventory, locally matches exact IPv4 and preserves ordered duplicates. The fixture attaches a,b,a from supplied SHUTOFF/nil metadata using integer IDs including9007199254740993 and retained unknown JSON-number digits. Each item performs list, selected-ID GET and POST addFloatingIp with the requested fixed IPv4; async performs no raw GET and wait observes the exact tagged target under raw BUILD or ERROR. Accepted202 empty action proof is distinct from raw observation. Neutron Assignment stays nil; separate NovaAssignment and per-item NovaSelections preserve the actual pre-action instance metadata rather than synthesizing association/status.
  Tests: `compute/server_ip_nova_test.go:TestNovaServerIPOrderedAsyncAndRawOnlyObservation`.
- Nova matching is local and unfiltered: an unrelated null address in another pool does not poison a valid exact-address or free-pool selection, and present canonical null prevents the legacy address alias from becoming a second match. Fractional numeric IDs, selected missing association metadata and duplicate matching addresses fail without action/allocation; duplicate yields ErrAmbiguous. List204 is an error rather than absence, and an empty200 default-pool response is ErrNotFound. When a first IP inventory page already has a match but its advertised next page returns403, both address and pool consumers retain the late native status/header/body, return no NovaAssignment and send no selected-IP GET/action/allocation.
  Tests: `compute/server_ip_nova_test.go:TestNovaServerIPLocalSelectionIgnoresUnrelatedNullAddresses`, `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPSelectedDecoderAndPoolResponseBoundaries`, `compute/server_ip_nova_test.go:TestNovaServerIPListAbsenceAndAssociationMetadataCannotAllocate`, `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPLateInventoryPageFailureCannotAttachOrAllocate`.
- Missing second lookup or denied second Nova action preserves the completed first attempt and its historical Observed value, latest known Nova model and failing second Error; no third lookup starts. A complete accepted second action202 followed by Close, custom cancellation or service-source failure retains empty-body action proof, ActionAccepted=true, its pre-action IP model and the last first-target BUILD Server. The second remains incomplete/unobserved and no later observer or target work starts.
  Tests: `compute/server_ip_nova_test.go:TestNovaServerIPSecondFailureKeepsHistoryAndCurrentModel`, `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPAcceptedActionFailurePreservesPerItemHistory`.
- A Nova IP already associated to the fixed server and requested fixed IPv4 is rechecked by ID and returned with AlreadyAttached=true, Reused=true and the actual associated raw model/header. ActionAccepted remains false and ActionResponse nil because no action is sent. Async returns without raw observation; the explicit wait variant adds only one raw target observation and completes. No synthetic IP ACTIVE or rewritten instance_id is supplied.
  Tests: `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPAlreadyAttachedKeepsRawModelAndSkipsAction`.
- The Nova direct-list fixture keeps all six HTTP deadlines identical under one standalone budget: near60seconds,17second override, an earlier30second parent and unlimited equal to a three-minute parent with the default60second cap removed. Server ID/address slices are snapshotted before mutating option callbacks and nested factories execute once. A distinct actual one-second blocked raw observation expiry retains the first accepted action/incomplete attempt with DeadlineExceeded and stops before the second item. The60second assertion observes deadlines rather than waiting for expiry.
  Tests: `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPOneHTTPBudgetAndOptionSnapshot`, `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPRealDeadlineStopsAfterAcceptedFirstAction`.
- For denied Nova action403, a native RetryFunc that changes the owned JSON body or service source is called once but cannot resend the action or start the next item. ErrInvalidOption remains joined with the original native403 header/body and the selected known IP, without accepted-action proof. This verifies these two retry mutation boundaries, not every possible transport/session retry contract.
  Tests: `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPRetryCannotChangeOwnedActionOrSource`.
- Explicit read-only Plan for configured Nova, source None and configured Neutron with absent Network returns Needed=true/Backend=Nova and no Nova selections or fixture HTTP. The later independent standalone execution resolves the actual legacy Compute backend and completes; Plan did not hold an executable plan. Connection AddIPList resolves Compute for direct Nova/None, or Network then Compute for precise Network absence, sends no Nova query filters, preserves configured2.35 headers and supports async/raw-only wait from supplied SHUTOFF/nil metadata. Missing Compute capability remains separately unsupported.
  Tests: `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPReadOnlyPlanAndExplicitBackendChoice`, `connection_server_ip_nova_test.go:TestConnectionNovaIPBackendChoiceAndAsyncRawWait`.
- The Connection Nova list consumer reads the live shared token for every HTTP request: after the first accepted action the second item uses the new token without rebuilding the backend. Empty positional lists under Nova/None with wait=true and an ignored unsafe pool make no endpoint locator calls, HTTP events or token reads. Replacing cached Compute.API during accepted second action Close retains first completed/observed history, decoded pre-action second IP and action202 empty-body proof with ErrInvalidOption; no third item or second raw observation starts.
  Tests: `connection_server_ip_nova_test.go:TestConnectionNovaIPReadsLiveTokenAndEmptyListStaysLazy`, `connection_server_ip_nova_test.go:TestConnectionNovaIPAcceptedCachedSourceFailureRetainsHistory`.
- The same bounded Nova ordered-address foundation is consumed by strong Ensure, GetActive async/sync, standalone Wait and automatic creation. Both items complete with separate NovaAssignment and no Neutron Assignment. GetActive async performs zero raw observations; Ensure/Get sync observe each item, while Wait/Create perform one initial actual ACTIVE readiness GET plus two target observations. Wait ignores supplied ERROR in favor of raw state. The image/flavor creation body is checked and original Creation.AdminPass stays separate from the last raw Server. This new upper success fixture is an explicit list, not every pool/auto/volume/NIC combination.
  Tests: `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPReadinessAndCreationConsumeLegacyBackend`.
- For strong Ensure, synchronous GetActive, Wait and creation, an exact assigned floating target under late raw BUILD does not complete: one progress failure retains the accepted action and current Server. Raw ERROR returns typed FailedState without progress. Only the first attempt starts; the next IP is not selected. Creation preserves the initial AdminPass. Nova has no IP ACTIVE field to poll or synthesize; these checks prove actual server readiness and raw target policy, while standalone Add keeps its any-status observation contract.
  Tests: `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPUpperObservationRequiresActualActiveServer`.
- The Nova direct-list lane rejects selected Compute2.36/2.44 and Neutron-only port/NAT-network/project overrides with ErrUnsupported before HTTP or attempts; a nonnumeric latest microversion is an invalid option. Known legacy capability preflight for automatic creation separately rejects2.36, explicit port and project overrides before any server POST or other fixture HTTP. These capability limits are distinct from a blanket Nova backend prohibition.
  Tests: `compute/server_ip_nova_test.go:TestNovaServerIPVersionAndNeutronSelectorsFailBeforeHTTP`, `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPKnownCreateCapabilityStopsBeforeServerPOST`.
- The bounded Nova pool consumer uses the literal selected pool ahead of ignored malformed address selectors. Reuse selects the first matching raw instance_id=null candidate without allocation; reuse=false sends POST200 with only the pool name, one compatibility GET and a further selected-IP recheck before action202. An empty-string instance_id is not a free null candidate. Automatic mode without a pool chooses the first returned pool and uses the same assignment foundation; automatic disabled/private/floating/empty skips send no fixture HTTP and create no NovaAssignment. These are assignment consumers, not the source optional-server availability getter.
  Tests: `compute/server_ip_nova_test.go:TestNovaServerIPPoolReuseAndFreshAllocation`, `compute/server_ip_nova_test.go:TestNovaServerIPListAbsenceAndAssociationMetadataCannotAllocate`, `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPAutomaticDefaultPoolAndKnownSkips`.
- A Nova allocation accepted200 retains Allocated=true, its decoded IP and AllocationResponse proof when a later compatibility GET is denied or contradicts the selected address/pool. Accepted allocation or compatibility Close failure preserves the known model beside the original Close cause and the actual failed response proof; no association action starts. A decoded wrong-pool allocation remains partial evidence and is not used as a new target.
  Tests: `compute/server_ip_nova_test.go:TestNovaServerIPAllocationAndCompatibilityFailureKeepPartial`, `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPAcceptedAllocationAndCompatibilityClosePreserveProof`.
- With configured Nova source and an available Neutron topology, automatic classification consumes shared networks/owned ports and configured NAT selection, but the input Nova pool name is not resolved as a Neutron network identity. The fixture returns a Nova action assignment from supplied BUILD in the standalone async lane and records both topology inventory reads. This does not complete the source has_service or every configured topology/normalization combination.
  Tests: `compute/server_ip_nova_boundaries_test.go:TestNovaServerIPAutomaticNeutronTopologyDoesNotConsumePoolAsNetwork`.
- Connection explicit IP assignment and shared role discovery recognize only direct native missing-endpoint pointer/value or a single wrapped absence as clean Network absence. The positive AddIPList cases execute the Nova action; an error joined with a catalog sentinel, wrapped joined error or context.Canceled preserves that cause instead of selecting Nova. Negative cases make exactly one Network locator call with no Compute lookup, fixture HTTP/token read, attempt or assignment; when a partial result exists its backend remains Neutron. This is the tested catalog-error boundary, not full source has_service/discovery/session parity.
  Tests: `connection_server_ip_nova_test.go:TestConnectionNovaIPMixedCatalogFailureCannotSelectBackend`.

Python/Go differences:

- The pinned public add_ips_to_server returns mutable Server Resource and dispatches pool > truthy ips > auto/needs. Independent typed Service/Connection AddIPsToServer now provides bounded Neutron and legacy Nova selected modes with async/whole60second defaults and structured Decision/Attempts/Server/Observed plus separate Neutron Assignment or NovaAssignment. Existing Plan/Ensure/ready/create keep their distinct default/readiness policies; Nova Plan is diagnostic and independent execution replans.
- Disabled/private/AccessIPv4/floating/empty skips apply to automatic mode only. Pool/list requests bypass automatic needs, and only the final winning selector is validated; pool does not inspect ignored address strings. Empty list/zero pool clears selection. Concrete IPv4/Ref inputs, owned snapshots and option preparation replace Python dynamic coercion; full Resource/session/normalization and has_service configuration/version behavior remain partial.
- Unlike the source needs helper swallowing some floating-network SDKException as false, only a clean completed semantic plan absence becomes a skip. HTTP/decode/read/Close/source/cancellation/ambiguous or other explicit-selector failures remain errors. Strict destination tuple selection, revision protection and re-GET behavior follow the separately reviewed Go Neutron plan.
- Existing stronger Ensure, synchronous GetActive, Wait and creation require matching actual ACTIVE server and exact assigned version4 floating-tagged raw row. Independent WithServerIPWait instead observes that exact row under any server status and does not wait for Neutron ACTIVE; supplemental/AccessIPv4/fixed/IPv6 metadata cannot prove raw observation in either family. Both retain known resources without automatic cleanup/backend fallback; per-item observations do not prove simultaneous final presence.
- Independent AddIPsToServer defaults to one60-second overall context, async and five-second raw poll, with explicit helper wait/timeout overrides. Existing Ensure/automatic creation retain five-minute/two-second synchronous defaults; GetActive/Wait retain their180second/five-second and readiness policies. Go also bounds selection/HTTP work and shares one budget across items, whereas source forwards timeout60 into wait branches and repeated list attachments. Source async pre-attached-IP Compute refresh remains a separate difference.
- The selected Neutron pool lane reuses Ensure and may prefer already-attached candidates or allocate attached directly. Bounded Nova pool mode instead selects raw instance_id=null in the literal pool or allocates by POST200, compatibility GET and separate action202; it retains actual pre-action association fields. Source optional-server availability/full normalization and nonreuse refresh/elapsed subtraction remain distinct. Sequential Observed records item-time targets, not simultaneous final presence.
- Direct Nova/None explicit or precise absent-Network selected execution requires actual legacy Compute capability. Missing Compute, selected microversion>=2.36 and unsupported Neutron port/NAT/project overrides remain typed capability failures; source/backend/config failures do not trigger automatic Neutron-error fallback. Nova has no IP status and receives no synthetic ACTIVE. Token sharing and accepted action proof are preserved separately from observation.

Unresolved boundaries:

- Complete source aliases/coercions, dynamic Server/Resource inputs, fixed/NAT forwarding and returned Resource/interface/location normalization beyond independent typed AddIPsToServer, whose async/60second default and selected Neutron and bounded Nova modes now have bounded fixtures. Compare uncovered selector/default/wait/progress and lifetime combinations; named entry/default alignment does not complete the public source contract.
- Compare full source backend/fallback/error ordering, dynamic pool/source aliases, normalization/location/Resource and service/config/API-version/discovery/session/catalog policy beyond bounded Nova list/pool/allocate/action and missing-Network/direct-source fixtures. Complete public availability/CRUD/detach/remove contracts separately; capable Nova mutation now exists but unsupported version/selectors/missing endpoint remain receiver-scoped limitations.
- Complete source pool unattached-first availability, nonreuse post-create refresh/timing, broad lookup Exception/None retry, integer remaining budget, fault/ACTIVE-no-address/delete/cleanup and ordinary Create/Get lifecycle semantics beyond the typed upper workflows. Preserve deliberate whole-context/noDELETE/exact-target and sequential-Observed policies without claiming full cloud parity.
- Complete configuration/vendor-profile/raw-string/nullable, inherited Resource/descriptor/adapter/session/cache and out-of-band invalidation contracts. Local HTTP fixtures and example compilation do not establish authenticated cloud or Python runtime interoperability.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack
from openstack.cloud import meta

conn = openstack.connect(cloud="dev")
server = conn.compute.find_server("web-01", ignore_missing=False)
print(conn.get_server_public_ip(server))
print(conn.get_server_private_ip(server))

expanded = meta.add_server_interfaces(conn, server)
print(expanded["public_v4"], expanded["public_v6"])
print(expanded["private_v4"], expanded["interface_ip"])
```

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	server := flag.String("server", "web-01", "exact server name, or ID with -server-id")
	serverID := flag.Bool("server-id", false, "treat -server as an ID")
	private := flag.Bool("private", false, "prefer internal IPv4 for server access")
	forceIPv4 := flag.Bool("force-ipv4", false, "omit IPv6 from calculated access fields")
	probe := flag.Bool("probe", false, "try TCP reachability among multiple candidates")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := run(ctx, *cloud, *server, *serverID, *private, *forceIPv4, *probe); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cloud, serverNameOrID string, byID, private, forceIPv4, probe bool) error {
	conn, err := sdk.Connect(ctx,
		sdk.WithCloud(cloud),
		sdk.WithServerAddressPolicy(
			compute.WithPrivateCloud(private),
			compute.WithForceIPv4(forceIPv4),
			compute.WithFloatingIPSource(compute.FloatingIPNeutron),
			compute.WithAddressReachability(probe),
		),
	)
	if err != nil {
		return err
	}
	service, err := conn.Compute(ctx)
	if err != nil {
		return err
	}
	ref := resource.Name(serverNameOrID)
	if byID {
		ref = resource.ID(serverNameOrID)
	}
	server, err := service.Servers.Find(ctx, ref)
	if err != nil {
		return err
	}
	view, expandErr := service.ExpandServerInterfaces(ctx, server)
	if view != nil {
		fmt.Printf("server=%s public4=%q public6=%q private4=%q interface=%q access4=%q access6=%q\n",
			view.ServerID, view.PublicIPv4, view.PublicIPv6, view.PrivateIPv4,
			view.InterfaceIP, view.AccessIPv4, view.AccessIPv6)
		if view.SupplementalError != nil {
			log.Printf("supplemental address lookup: %v", view.SupplementalError)
		}
	}
	if expandErr != nil {
		return fmt.Errorf("expand server addresses: %w", expandErr)
	}

	publicIP, err := service.GetServerPublicIP(ctx, server)
	if err != nil {
		return fmt.Errorf("public address: %w", err)
	}
	privateIP, err := service.GetServerPrivateIP(ctx, server)
	if err != nil {
		return fmt.Errorf("private address: %w", err)
	}
	fmt.Printf("snapshot public4=%q private4=%q\n", publicIP, privateIP)
	return nil
}
```

## References

- [ Compute](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py#L859-L873)
- [add_server_interfaces](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L528-L563)
- [Meta](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L215-L269)
- [Meta](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L147-L190)
- [configured default interface](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L348-L405)
- [Ipaddress](https://docs.python.org/3.13/library/ipaddress.html#ipaddress.IPv4Address.is_private)
- [Meta](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L272-L327)
- [Network Roles](../network/network-roles.md)
- [Meta](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/meta.py#L474-L525)
- [CreateWithFloatingIP](create-with-floating-ip.md)
- [server_address_selection_test.go](server_address_selection_test.go)
- [Server Address Expansion Test](server_address_expansion_test.go)
- [Connection Server Addresses Test](../connection_server_addresses_test.go)
- [Connection Server Address Config Test](../connection_server_address_config_test.go)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Server Automatic Ip](server-automatic-ip.md)
