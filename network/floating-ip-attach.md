# Floating Ip Attach

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network_common/NetworkCommonCloudMixin/add_ip_list`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.Attach`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.PrepareAttach`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPs.AttachPrepared`
- `github.com/JSYoo5B/go-openstacksdk/network.FloatingIPPlanner.PrepareAttach`
- `github.com/JSYoo5B/go-openstacksdk/network.PrepareAttachFloatingIPOptions`
- `github.com/JSYoo5B/go-openstacksdk/network.WithAttachFloatingIPPolicy`
- `github.com/JSYoo5B/go-openstacksdk/network.WithAttachDestinationPolicy`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPPool`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithFloatingIPAddresses`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.PlanServerFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.EnsureServerFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.GetActiveServer`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.WaitForServer`
- `github.com/JSYoo5B/go-openstacksdk/compute.Service.CreateWithAutomaticFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk.Connection.PlanServerFloatingIP`
- `github.com/JSYoo5B/go-openstacksdk.Connection.EnsureServerFloatingIP`
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

- Owner-free exact existing-IP preparation/execution allows a foreign recorded owner and stable other-port movement with no allocation or free/project/network filters. Preparation does not mutate, Selection is value-only, original revision0 remains If-Match0 after recheck revision7, and absent/null omit the condition. PUT body is exactly the selected port_id and fixed_ip_address.
  Tests: `network/floating_ip_attach_test.go:TestFloatingIPAttachMovesExactExistingIPWithoutOwnerOrAllocation`.
- Address query selection consumes all pages including empty JSON/HTTP links and locally checks exact addresses. Duplicate exact rows are ambiguous, missing/terminal204 are NotFound, and late403,malformed JSON or changed continuation filter fail before destination/mutation with native or accepted-response proof. Invalid selected ID/network,negative revision and contradictory owner aliases retain actual200 body/header proof on both ID and address paths.
  Tests: `network/floating_ip_attach_test.go:TestFloatingIPAttachAddressConsumesEveryPageBeforeSelection`, `network/floating_ip_attach_boundaries_test.go:TestFloatingIPAttachInvalidSelectedModelsRetainHTTPProof`.
- Execution rejects changed IP ID,address,external network,recorded owner or unrelated association and changed port server/fixed address before PUT.412 retains the known original candidate and native conflict evidence without retry/fallback/allocation.
  Tests: `network/floating_ip_attach_test.go:TestFloatingIPAttachRechecksPinnedIPAndPortBeforeMutation`.
- Accepted PUT200 complete matching models survive read/Close/source/cancel errors with original cause,status/header/body and exactly one mutation; malformed or wrong-ID/address/owner/fixed responses retain the prior valid candidate plus joined validation/decode/process causes. Fatal errors do not enter readiness wait or native RetryFunc.
  Tests: `network/floating_ip_attach_test.go:TestFloatingIPAttachAcceptedPUTRetainsOnlyExactValidatedModel`.
- Prepared Ensure destination/wait policy is reused without its allocation/project/reuse rules; NoWait preserves prepared wait policy for a later Active override. Same-tuple attachment skips PUT and still polls actual ACTIVE; alternate completion field cannot bypass actual Status, changed wait address and actual ERROR fail on first wait GET, and timeout retains the known IP.
  Tests: `network/floating_ip_attach_test.go:TestFloatingIPAttachPreparedPolicyAndResultAreOwned`, `network/floating_ip_attach_test.go:TestFloatingIPAttachWaitRequiresActualStatusAndPinnedAddress`.
- Nil/canceled/invalid input/options/zero or failed plans stop before HTTP, foreign-service plans reject, and caller mutation of initial-failure result ID,address or Tags does not change the same executable plan on reuse; tags null/empty array are preserved. Source changes remain sticky after restoration, accepted selection Close source errors preserve200 proof, and revision-header/source/outer-guard retry changes prevent another physical mutation while retaining original503 evidence.
  Tests: `network/floating_ip_attach_boundaries_test.go:TestFloatingIPAttachPreflightRejectsInvalidInputAndPlans`, `network/floating_ip_attach_boundaries_test.go:TestFloatingIPAttachPlanSnapshotSurvivesCallerMutationAndRejectsForeignService`, `network/floating_ip_attach_boundaries_test.go:TestFloatingIPAttachGuardsSourceCloseAndPhysicalRetries`.
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
- The explicit Ensure fixture propagates one identical HTTP context deadline shorter than the parent across its items, then parent cancellation during the second raw observation retains both known assignments and stops the third. Missing mandatory raw observation capability fails before any lookup or mutation. Plan/Ensure snapshot supplied server ID/status before application options mutate them, and outer selector, address, Ensure and resource-wait option factories each execute once across a two-item operation. This deadline evidence checks context identity and cancellation; it does not wait for actual timer expiry.
  Tests: `compute/server_ip_dispatch_workflow_test.go:TestServerIPDispatchOneDeadlineBoundsAllItemsAndPreflightsObservation`, `compute/server_ip_dispatch_workflow_test.go:TestServerIPDispatchPinsSuppliedServerBeforeOptionsAndPreparesPolicyOnce`.
- An accepted203 second raw observation whose Close fails retains the newly decoded matching raw Server, both started attempts and current second assignment beside the original Close cause and actual status/header proof. The first attempt remains observed, the second and aggregate do not claim observation completion, and no third request starts.
  Tests: `compute/server_ip_dispatch_workflow_test.go:TestServerIPDispatchAcceptedRawCloseKeepsCurrentServerAndAllKnownIPs`.
- Connection GetActive async and standalone Wait delegate the same ordered explicit-address policy. The root fixture attaches two exact addresses from unscoped queries with revision0: async locates no Compute endpoint, sends no raw observation and performs one singular IP recheck per item; Wait lazily locates Compute once, performs one raw readiness plus two target observations and one additional IP readiness GET per item. Both attempts complete in input order with no allocation; Observed is false in async and true per sequential Wait item, not simultaneous final presence.
  Tests: `connection_server_ip_dispatch_test.go:TestConnectionExplicitIPDispatchSharesLazyAsyncAndOrderedWaitPolicies`.
- In Connection standalone Wait, changing the cached Compute.API during the accepted second association body Close retains the first completed/observed attempt, latest first-target Server and decoded second bound IP beside ErrInvalidOption and actual200 header proof. The second attempt remains incomplete/unobserved, no third lookup or second-item readiness/observation starts, and raw count remains readiness plus the first observation. This fixture guards a late source failure, not legacy backend fallback or full Resource policy.
  Tests: `connection_server_ip_dispatch_test.go:TestConnectionOrderedIPSourceFailureKeepsFirstObservationAndAcceptedSecondIP`.
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
- Connection explicit IP assignment and shared role discovery recognize only direct native missing-endpoint pointer/value or a single wrapped absence as clean Network absence. The positive AddIPList cases execute the Nova action; an error joined with a catalog sentinel, wrapped joined error or context.Canceled preserves that cause instead of selecting Nova. Negative cases make exactly one Network locator call with no Compute lookup, fixture HTTP/token read, attempt or assignment; when a partial result exists its backend remains Neutron. This is the tested catalog-error boundary, not full source has_service/discovery/session parity.
  Tests: `connection_server_ip_nova_test.go:TestConnectionNovaIPMixedCatalogFailureCannotSelectBackend`.
- A disabled has_network flag is the has_service('network') half of _use_neutron_floating: with the Neutron source AddAutoIP, AddIPList and AddIPsToServer use the Nova lane (one addFloatingIp action) and never contact the network endpoint.
  Tests: `connection_network_service_disable_test.go:TestFloatingIPHelpersUseNovaWhenNetworkServiceDisabled`.

Python/Go differences:

- Network Attach remains one-existing-IP Neutron foundation with safe IDs/IPv4 names and stable association movement. Typed ordered consumers and independent positional AddIPList now execute Neutron or bounded legacy Nova lists. Positional order/duplicates/empty no-op and async whole60second defaults remain; results contain structured Server/Attempts plus separate Neutron Assignment or NovaAssignment rather than mutable Python Resource or dynamic str/list coercion.
- The Neutron Attach path consumes all address pages, locally matches exact addresses and pins opaque destination/revision/port invariants. The Nova consumer instead reads unfiltered legacy inventory, validates only the selected candidate, rechecks its ID/address/pool/association and sends addFloatingIp; it has no Neutron port/revision. Both reject ambiguity and late inventory failures before execution; current fixtures do not prove every pagination shape.
- Network Attach remains async with an optional common five-minute/two-second Neutron ACTIVE waiter. Independent AddIPList defaults to async/60second whole context/five-second raw poll; WithServerIPWait observes each exact raw IPv4 without requiring server/IP ACTIVE. Existing Ensure/sync ready/create consumers keep stronger ACTIVE checks. Each Observed attempt is historical, not simultaneous final presence of all targets. Source list passes timeout60 independently to each attach; Go uses one shared operation context and preserves partial evidence without cleanup.
- Bounded Nova execution retains actual allocation/pre-action fields and accepted200 allocation or202 empty-action evidence in additive NovaAssignment, without synthesizing IP ACTIVE or overwriting instance_id after action acceptance. Strong workflows still require actual raw server ACTIVE/target convergence; standalone wait observes the exact target under any status. Version<2.36 and no Neutron-only port/NAT/project overrides are deliberate capability restrictions. Full source normalization, configured-Neutron fallback branch semantics and async attached-IP refresh remain distinct.

Unresolved boundaries:

- Compare full dynamic single-string/list/Server Resource inputs and returned mutable Resource/interface/location/normalization against the new typed positional AddIPList. Independent wait=false/timeout60 and required empty-list semantics now have bounded fixtures, but Go common validation, one whole context and structured attempt results remain deliberate differences.
- Compare all fixed/NAT/port, raw-poll/progress, parent/unlimited/default and repeated-input combinations beyond the current standalone and stronger typed consumer fixtures. Ordinary Servers.Create and full cloud lifecycle/returned-interface behavior remain separate. Sequential Observed evidence does not prove all prior IPs remain simultaneously in the final Server.
- Complete source Nova attachment/backend/fallback/error ordering beyond the bounded executable legacy lane and precise missing-Network/direct-source capability cases. has_network configuration-disable now selects the Nova lane; preserve has_service discovery/version/session policy, source conditional async attached-IP refresh, Cloud versus Proxy fixed-address presence semantics and model normalization; full public floating-IP CRUD/detach/remove and authenticated interoperability remain outside these assignment fixtures.
- Complete inherited Resource/descriptor/configuration/location/normalization/cache and authenticated Python/cloud interoperability contracts beyond local HTTP fixtures and example compilation.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
server = conn.compute.find_server("web-01", ignore_missing=False)
server = conn.add_ip_list(
    server, ips=["198.51.100.10"],
    fixed_address="10.0.0.10", wait=True, timeout=60,
)
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
	"github.com/JSYoo5B/go-openstacksdk/network"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
	cloud := flag.String("cloud", "dev", "clouds.yaml entry")
	server := flag.String("server-id", "", "existing Nova server ID")
	ipID := flag.String("ip-id", "", "existing Floating IP ID")
	address := flag.String("address", "", "existing floating IPv4 address")
	fixed := flag.String("fixed", "", "exact destination fixed IPv4")
	execute := flag.Bool("execute", false, "execute the prepared attachment")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := run(ctx, *cloud, *server, *ipID, *address, *fixed, *execute); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, cloud, server, ipID, address, fixed string, execute bool) error {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return err
	}
	service, err := conn.Network(ctx)
	if err != nil {
		return err
	}
	ip := resource.Name(address)
	if ipID != "" {
		ip = resource.ID(ipID)
	}
	options := []network.AttachFloatingIPOption{
		network.WithAttachWait(resource.WithTimeout(time.Minute)),
	}
	if fixed != "" {
		options = append(options, network.WithAttachFixedAddress(fixed))
	}
	plan, err := service.FloatingIPs.PrepareAttach(ctx,
		network.AttachFloatingIPRequest{Server: resource.ID(server), IP: ip}, options...)
	if err != nil {
		return err
	}
	fmt.Printf("selection: %+v\n", plan.Selection())
	if !execute {
		return nil
	}
	result, err := service.FloatingIPs.AttachPrepared(ctx, plan)
	if result != nil && result.FloatingIP != nil {
		fmt.Printf("IP: %s address=%s port=%s reused=%t allocated=%t\n",
			result.FloatingIP.ID, result.FloatingIP.FloatingIP, result.FloatingIP.PortID,
			result.Reused, result.Allocated)
	}
	return err
}
```

## References

- [AddIPsToServer·AddIPList](../compute/server-ip-helpers.md)
- [ Network Common](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1342-L1391)
- [Server Ip Dispatch](../compute/server-ip-dispatch.md)
- [Compute legacy Nova backend](../compute/server-nova-floating-ip.md)
- [Floating Ip Ensure](floating-ip-ensure.md)
- [Implementation Plan](../docs/implementation-plan.md)
