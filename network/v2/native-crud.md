# Native Crud

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

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

### `gophercloud:networking/v2/networks.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithCreateField`

- Create POSTs {"network":{...}} to networks with extensions inside the envelope, rejects a core-key extension collision and nil option before HTTP, decodes the response and accepts only 201/202.
  Tests: `network/v2/networks/native_contracts_test.go:TestNativeNetworkRoutesBodiesPagingAndDecode`, `network/v2/networks/native_contracts_test.go:TestNativeNetworkStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/networks call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/networks.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.Get`

- Get GETs networks/{raw id}, decodes the network envelope and accepts only 200.
  Tests: `network/v2/networks/native_contracts_test.go:TestNativeNetworkRoutesBodiesPagingAndDecode`, `network/v2/networks/native_contracts_test.go:TestNativeNetworkStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/networks call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/networks.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithListQuery`

- List GETs networks with ListOpts and extension query, follows networks_links next as received, yields nothing for an empty array, surfaces io.EOF for a bodyless 204, rejects nil options before HTTP and returns native pager status errors (200/204/300) without operation context.
  Tests: `network/v2/networks/native_contracts_test.go:TestNativeNetworkRoutesBodiesPagingAndDecode`, `network/v2/networks/native_contracts_test.go:TestNativeNetworkStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/networks call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/networks.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.Delete`

- Delete DELETEs networks/{raw id} and accepts only 202/204.
  Tests: `network/v2/networks/native_contracts_test.go:TestNativeNetworkRoutesBodiesPagingAndDecode`, `network/v2/networks/native_contracts_test.go:TestNativeNetworkStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/networks call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/subnets.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithCreateField`

- Create POSTs {"subnet":{...}} to subnets with extensions inside the envelope, requires network_id and sends an empty GatewayIP pointer as gateway_ip null, rejects a core-key extension collision and nil option before HTTP, decodes the response and accepts only 201/202.
  Tests: `network/v2/subnets/native_contracts_test.go:TestNativeSubnetRoutesBodiesPagingAndDecode`, `network/v2/subnets/native_contracts_test.go:TestNativeSubnetStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/subnets call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/subnets.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.Get`

- Get GETs subnets/{raw id}, decodes the subnet envelope and accepts only 200.
  Tests: `network/v2/subnets/native_contracts_test.go:TestNativeSubnetRoutesBodiesPagingAndDecode`, `network/v2/subnets/native_contracts_test.go:TestNativeSubnetStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/subnets call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/subnets.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithListQuery`

- List GETs subnets with ListOpts and extension query, follows subnets_links next as received, yields nothing for an empty array, surfaces io.EOF for a bodyless 204, rejects nil options before HTTP and returns native pager status errors (200/204/300) without operation context.
  Tests: `network/v2/subnets/native_contracts_test.go:TestNativeSubnetRoutesBodiesPagingAndDecode`, `network/v2/subnets/native_contracts_test.go:TestNativeSubnetStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/subnets call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/subnets.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.Delete`

- Delete DELETEs subnets/{raw id} and accepts only 202/204.
  Tests: `network/v2/subnets/native_contracts_test.go:TestNativeSubnetRoutesBodiesPagingAndDecode`, `network/v2/subnets/native_contracts_test.go:TestNativeSubnetStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/subnets call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/ports.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithCreateField`

- Create POSTs {"port":{...}} to ports with extensions inside the envelope, requires network_id, flattens value_specs into the port while rejecting banned shared/tenant_id and overwriting keys, rejects a core-key extension collision and nil option before HTTP, decodes the response and accepts only 201/202.
  Tests: `network/v2/ports/native_contracts_test.go:TestNativePortRoutesBodiesPagingAndDecode`, `network/v2/ports/native_contracts_test.go:TestNativePortStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/ports call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/ports.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.Get`

- Get GETs ports/{raw id}, decodes the port envelope and accepts only 200.
  Tests: `network/v2/ports/native_contracts_test.go:TestNativePortRoutesBodiesPagingAndDecode`, `network/v2/ports/native_contracts_test.go:TestNativePortStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/ports call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/ports.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithListQuery`

- List GETs ports with ListOpts and extension query, repeating fixed_ips conditions and security_groups keys, follows ports_links next as received, yields nothing for an empty array, surfaces io.EOF for a bodyless 204, rejects nil options before HTTP and returns native pager status errors (200/204/300) without operation context.
  Tests: `network/v2/ports/native_contracts_test.go:TestNativePortRoutesBodiesPagingAndDecode`, `network/v2/ports/native_contracts_test.go:TestNativePortStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/ports call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/ports.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.Delete`

- Delete DELETEs ports/{raw id} and accepts only 202/204.
  Tests: `network/v2/ports/native_contracts_test.go:TestNativePortRoutesBodiesPagingAndDecode`, `network/v2/ports/native_contracts_test.go:TestNativePortStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/ports call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/security/groups.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithCreateField`

- Create POSTs {"security_group":{...}} to security-groups with extensions inside the envelope, requires name and sends an explicit false stateful pointer, rejects a core-key extension collision and nil option before HTTP, decodes the response and accepts only 201/202.
  Tests: `network/v2/extensions/security/groups/native_contracts_test.go:TestNativeSecGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/security/groups/native_contracts_test.go:TestNativeSecGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/security/groups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/security/groups.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.Get`

- Get GETs security-groups/{raw id}, decodes the security_group envelope and accepts only 200.
  Tests: `network/v2/extensions/security/groups/native_contracts_test.go:TestNativeSecGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/security/groups/native_contracts_test.go:TestNativeSecGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/security/groups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/security/groups.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithListOptions`

- List GETs security-groups with concrete ListOpts (no extension query builder), follows security-groups_links next as received, yields nothing for an empty array, surfaces io.EOF for a bodyless 204, rejects nil options before HTTP and returns native pager status errors (200/204/300) without operation context.
  Tests: `network/v2/extensions/security/groups/native_contracts_test.go:TestNativeSecGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/security/groups/native_contracts_test.go:TestNativeSecGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/security/groups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/security/groups.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.Delete`

- Delete DELETEs security-groups/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/security/groups/native_contracts_test.go:TestNativeSecGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/security/groups/native_contracts_test.go:TestNativeSecGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/security/groups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/security/rules.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithCreateField`

- Create POSTs {"security_group_rule":{...}} to security-group-rules with extensions inside the envelope, requires direction, ethertype and security_group_id and omits zero port ranges, rejects a core-key extension collision and nil option before HTTP, decodes the response and accepts only 201/202.
  Tests: `network/v2/extensions/security/rules/native_contracts_test.go:TestNativeRuleRoutesBodiesPagingAndDecode`, `network/v2/extensions/security/rules/native_contracts_test.go:TestNativeRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/security/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/security/rules.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.API.Get`

- Get GETs security-group-rules/{raw id}, decodes the security_group_rule envelope and accepts only 200.
  Tests: `network/v2/extensions/security/rules/native_contracts_test.go:TestNativeRuleRoutesBodiesPagingAndDecode`, `network/v2/extensions/security/rules/native_contracts_test.go:TestNativeRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/security/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/security/rules.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithListQuery`

- List GETs security-group-rules with ListOpts and extension query, follows security-group-rules_links next as received, yields nothing for an empty array, surfaces io.EOF for a bodyless 204, rejects nil options before HTTP and returns native pager status errors (200/204/300) without operation context.
  Tests: `network/v2/extensions/security/rules/native_contracts_test.go:TestNativeRuleRoutesBodiesPagingAndDecode`, `network/v2/extensions/security/rules/native_contracts_test.go:TestNativeRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/security/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/security/rules.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.API.Delete`

- Delete DELETEs security-group-rules/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/security/rules/native_contracts_test.go:TestNativeRuleRoutesBodiesPagingAndDecode`, `network/v2/extensions/security/rules/native_contracts_test.go:TestNativeRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/security/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/floatingips.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.Get`

- Get GETs floatingips/{raw id}, decodes the floatingip envelope and accepts only 200.
  Tests: `network/v2/extensions/layer3/floatingips/native_contracts_test.go:TestNativeFloatingIPRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/floatingips/native_contracts_test.go:TestNativeFloatingIPStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/floatingips call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/floatingips.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.WithListQuery`

- List GETs floatingips with ListOpts and extension query, follows floatingips_links next as received, yields nothing for an empty array, surfaces io.EOF for a bodyless 204, rejects nil options before HTTP and returns native pager status errors (200/204/300) without operation context.
  Tests: `network/v2/extensions/layer3/floatingips/native_contracts_test.go:TestNativeFloatingIPRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/floatingips/native_contracts_test.go:TestNativeFloatingIPStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/floatingips call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/floatingips.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/floatingips.API.Delete`

- Delete DELETEs floatingips/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/layer3/floatingips/native_contracts_test.go:TestNativeFloatingIPRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/floatingips/native_contracts_test.go:TestNativeFloatingIPStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/floatingips call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/security/rules.CreateBulk`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.API.CreateBulk`

- CreateBulk POSTs {"security_group_rules":[...]} to security-group-rules, validates every element's required direction/ethertype/security_group_id before HTTP, decodes the response array and accepts only 201/202 with operation context on errors.
  Tests: `network/v2/extensions/security/rules/native_contracts_test.go:TestNativeSecGroupRuleCreateBulk`.

Python/Go differences:

- This declaration maps the Gophercloud native bulk rule call only; it has no extension option builder.
- The SDK adds only request.Wrap operation context; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/routers.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithCreateField`

- Create POSTs {"router":{...}} to routers with extensions inside the envelope, omitting empty fields while keeping explicit false pointers, rejects a core-key extension collision and nil option before HTTP, decodes the response and accepts only 201/202.
  Tests: `network/v2/extensions/layer3/routers/native_contracts_test.go:TestNativeRouterRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/routers/native_contracts_test.go:TestNativeRouterStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/routers call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/routers.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.Get`

- Get GETs routers/{raw id}, decodes the router envelope and accepts only 200.
  Tests: `network/v2/extensions/layer3/routers/native_contracts_test.go:TestNativeRouterRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/routers/native_contracts_test.go:TestNativeRouterStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/routers call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/routers.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithListQuery`

- List GETs routers with ListOpts and extension query, follows routers_links next as received, yields nothing for an empty array, surfaces io.EOF for a bodyless 204, rejects nil options before HTTP and returns native pager status errors (200/204/300) without operation context.
  Tests: `network/v2/extensions/layer3/routers/native_contracts_test.go:TestNativeRouterRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/routers/native_contracts_test.go:TestNativeRouterStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/routers call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/routers.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.Delete`

- Delete DELETEs routers/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/layer3/routers/native_contracts_test.go:TestNativeRouterRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/routers/native_contracts_test.go:TestNativeRouterStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/routers call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/routers.AddInterface`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.AddInterface`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithAddInterfaceOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithAddInterfaceField`

- AddInterface PUTs routers/{id}/add_router_interface with an envelope-less subnet_id or port_id body (extensions beside it), requires exactly one of them before HTTP, decodes InterfaceInfo and accepts only 200.
  Tests: `network/v2/extensions/layer3/routers/native_contracts_test.go:TestNativeRouterInterfacesAndExternalGateways`.

Python/Go differences:

- This declaration maps the Gophercloud native router action call only; Python router helpers are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/routers.RemoveInterface`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.RemoveInterface`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRemoveInterfaceOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRemoveInterfaceField`

- RemoveInterface PUTs routers/{id}/remove_router_interface with subnet_id and/or port_id, requires at least one before HTTP, decodes InterfaceInfo and accepts only 200.
  Tests: `network/v2/extensions/layer3/routers/native_contracts_test.go:TestNativeRouterInterfacesAndExternalGateways`.

Python/Go differences:

- This declaration maps the Gophercloud native router action call only; Python router helpers are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/routers.AddExternalGateways`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.AddExternalGateways`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithAddExternalGatewaysOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithAddExternalGatewaysField`

- AddExternalGateways PUTs {"router":{"external_gateways":[...]}} to routers/{id}/add_external_gateways with extensions inside the envelope, rejects nil gateways, a core-key collision or nil option before HTTP, decodes the router and accepts only 200.
  Tests: `network/v2/extensions/layer3/routers/native_contracts_test.go:TestNativeRouterInterfacesAndExternalGateways`.

Python/Go differences:

- This declaration maps the Gophercloud native router action call only; Python router helpers are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/routers.UpdateExternalGateways`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.UpdateExternalGateways`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithUpdateExternalGatewaysOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithUpdateExternalGatewaysField`

- UpdateExternalGateways PUTs {"router":{"external_gateways":[...]}} to routers/{id}/update_external_gateways, rejects nil gateways, a core-key collision or nil option before HTTP, decodes the router and accepts only 200.
  Tests: `network/v2/extensions/layer3/routers/native_contracts_test.go:TestNativeRouterInterfacesAndExternalGateways`.

Python/Go differences:

- This declaration maps the Gophercloud native router action call only; Python router helpers are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/routers.RemoveExternalGateways`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.RemoveExternalGateways`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRemoveExternalGatewaysOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRemoveExternalGatewaysField`

- RemoveExternalGateways PUTs {"router":{"external_gateways":[...]}} to routers/{id}/remove_external_gateways sending an empty slice as [], rejects nil gateways before HTTP, decodes the router and accepts only 200.
  Tests: `network/v2/extensions/layer3/routers/native_contracts_test.go:TestNativeRouterInterfacesAndExternalGateways`.

Python/Go differences:

- This declaration maps the Gophercloud native router action call only; Python router helpers are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/extraroutes.Add`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/extraroutes.API.Add`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/extraroutes.WithAddOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/extraroutes.WithAddField`

- Add PUTs {"router":{"routes":[...]}} to routers/{id}/add_extraroutes (nil routes as an empty router object) with extensions inside the envelope, rejects a routes collision or nil option before HTTP, decodes the router and accepts only 200.
  Tests: `network/v2/extensions/layer3/extraroutes/native_contracts_test.go:TestNativeExtraRoutesAddRemoveBodiesAndStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native router action call only; Python router helpers are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/extraroutes.Remove`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/extraroutes.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/extraroutes.WithRemoveOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/extraroutes.WithRemoveField`

- Remove PUTs {"router":{"routes":[...]}} to routers/{id}/remove_extraroutes, rejects nil options before HTTP, decodes the router and accepts only 200.
  Tests: `network/v2/extensions/layer3/extraroutes/native_contracts_test.go:TestNativeExtraRoutesAddRemoveBodiesAndStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native router action call only; Python router helpers are tracked separately.
- The SDK adds only request.Wrap operation context and the documented extension merge; native URL construction, OkCodes and decoding are unchanged.

### `gophercloud:networking/v2/extensions/subnetpools.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithCreateField`

- Create POSTs {"subnetpool":{...}} to subnetpools with extensions inside the envelope, always sending name and prefixes (nil prefixes as null) while omitting other empty fields, rejects a core-key extension collision and nil option before HTTP, decodes numeric or numeric-string prefix lengths and accepts only 201.
  Tests: `network/v2/extensions/subnetpools/native_contracts_test.go:TestNativeSubnetPoolRoutesBodiesPagingAndDecode`, `network/v2/extensions/subnetpools/native_contracts_test.go:TestNativeSubnetPoolStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/subnetpools call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/subnetpools.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.Get`

- Get GETs subnetpools/{raw id}, decodes the subnetpool envelope with numeric or numeric-string prefix lengths and both timestamp forms, returns the partially decoded value beside a wrapped error when a prefix length is missing, null or non-numeric, and accepts only 200.
  Tests: `network/v2/extensions/subnetpools/native_contracts_test.go:TestNativeSubnetPoolRoutesBodiesPagingAndDecode`, `network/v2/extensions/subnetpools/native_contracts_test.go:TestNativeSubnetPoolStrictStatusesAndPreflight`, `network/v2/extensions/subnetpools/native_contracts_test.go:TestNativeSubnetPoolPrefixLengthDecodeFailures`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/subnetpools call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/subnetpools.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithListQuery`

- List GETs subnetpools with ListOpts and extension query values, follows subnetpools_links next hrefs as received, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/subnetpools/native_contracts_test.go:TestNativeSubnetPoolRoutesBodiesPagingAndDecode`, `network/v2/extensions/subnetpools/native_contracts_test.go:TestNativeSubnetPoolStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/subnetpools call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/subnetpools.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.Delete`

- Delete DELETEs subnetpools/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/subnetpools/native_contracts_test.go:TestNativeSubnetPoolRoutesBodiesPagingAndDecode`, `network/v2/extensions/subnetpools/native_contracts_test.go:TestNativeSubnetPoolStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/subnetpools call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/subnetpools.AddPrefixes`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.AddPrefixes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithAddPrefixesOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithAddPrefixesField`

- AddPrefixes PUTs an envelope-free {"prefixes":[...]} body (nil prefixes as {}) to subnetpools/{id}/add_prefixes with extensions at top level, rejects a prefixes extension collision before HTTP, returns the top-level prefixes list and accepts only 200.
  Tests: `network/v2/extensions/subnetpools/native_contracts_test.go:TestNativeSubnetPoolPrefixOperations`, `network/v2/extensions/subnetpools/native_contracts_test.go:TestNativeSubnetPoolStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/subnetpools call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/subnetpools.RemovePrefixes`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.RemovePrefixes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRemovePrefixesOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.WithRemovePrefixesField`

- RemovePrefixes PUTs an envelope-free {"prefixes":[...]} body to subnetpools/{id}/remove_prefixes, rejects a nil option before HTTP, returns the top-level prefixes list and accepts only 200.
  Tests: `network/v2/extensions/subnetpools/native_contracts_test.go:TestNativeSubnetPoolPrefixOperations`, `network/v2/extensions/subnetpools/native_contracts_test.go:TestNativeSubnetPoolStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/subnetpools call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/addressscopes.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.WithCreateField`

- Create POSTs {"address_scope":{...}} to address-scopes with extensions inside the envelope, always sending name and ip_version (zero values included), rejects a core-key extension collision before HTTP, decodes the response and accepts only 201.
  Tests: `network/v2/extensions/layer3/addressscopes/native_contracts_test.go:TestNativeAddressScopeRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/addressscopes/native_contracts_test.go:TestNativeAddressScopeStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/addressscopes call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/addressscopes.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.API.Get`

- Get GETs address-scopes/{raw id}, decodes the address_scope envelope ignoring unknown fields, wraps malformed or non-object envelope decode errors and accepts only 200.
  Tests: `network/v2/extensions/layer3/addressscopes/native_contracts_test.go:TestNativeAddressScopeRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/addressscopes/native_contracts_test.go:TestNativeAddressScopeStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/addressscopes call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/addressscopes.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.WithListQuery`

- List GETs address-scopes with ListOpts and extension query values, follows address_scopes_links next hrefs as received, decodes null name/shared as zero values, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/layer3/addressscopes/native_contracts_test.go:TestNativeAddressScopeRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/addressscopes/native_contracts_test.go:TestNativeAddressScopeStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/addressscopes call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/addressscopes.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.WithUpdateField`

- Update PUTs {"address_scope":{...}} to address-scopes/{id} sending explicit pointer name/shared values including false and an empty envelope when both are nil, keeps extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `network/v2/extensions/layer3/addressscopes/native_contracts_test.go:TestNativeAddressScopeRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/addressscopes/native_contracts_test.go:TestNativeAddressScopeStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/addressscopes call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/addressscopes.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/addressscopes.API.Delete`

- Delete DELETEs address-scopes/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/layer3/addressscopes/native_contracts_test.go:TestNativeAddressScopeRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/addressscopes/native_contracts_test.go:TestNativeAddressScopeStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/addressscopes call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/trunks.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithCreateField`

- Create POSTs {"trunk":{...}} to trunks with extensions inside the envelope, sends nil sub_ports as [] and explicit false admin_state_up, rejects a missing parent port_id, subport missing segmentation_type, core-key extension collision and nil option before HTTP, decodes RFC3339 timestamps and accepts only 201/202.
  Tests: `network/v2/extensions/trunks/native_contracts_test.go:TestNativeTrunkRoutesBodiesSinglePageAndDecode`, `network/v2/extensions/trunks/native_contracts_test.go:TestNativeTrunkStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/trunks call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/trunks.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.Get`

- Get GETs trunks/{raw id}, decodes the trunk envelope with subports, wraps a non-RFC3339 timestamp decode error and accepts only 200.
  Tests: `network/v2/extensions/trunks/native_contracts_test.go:TestNativeTrunkRoutesBodiesSinglePageAndDecode`, `network/v2/extensions/trunks/native_contracts_test.go:TestNativeTrunkStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/trunks call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/trunks.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithListQuery`

- List GETs trunks with ListOpts (string revision_number including "0") and extension query values, ignores the Neutron trunks_links array and follows only a links.next string because TrunkPage defines no NextPageURL, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/trunks/native_contracts_test.go:TestNativeTrunkRoutesBodiesSinglePageAndDecode`, `network/v2/extensions/trunks/native_contracts_test.go:TestNativeTrunkStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/trunks call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/trunks.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.Delete`

- Delete DELETEs trunks/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/trunks/native_contracts_test.go:TestNativeTrunkRoutesBodiesSinglePageAndDecode`, `network/v2/extensions/trunks/native_contracts_test.go:TestNativeTrunkStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/trunks call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/trunks.GetSubports`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.GetSubports`

- GetSubports GETs trunks/{id}/get_subports, returns the top-level sub_ports list including segmentation_id 0 and accepts only 200.
  Tests: `network/v2/extensions/trunks/native_contracts_test.go:TestNativeTrunkSubportOperations`, `network/v2/extensions/trunks/native_contracts_test.go:TestNativeTrunkStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/trunks call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/trunks.AddSubports`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.AddSubports`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithAddSubportsOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithAddSubportsField`

- AddSubports PUTs an envelope-free {"sub_ports":[...]} body with top-level extensions to trunks/{id}/add_subports, sends segmentation_id 0 and an empty list as [], rejects nil subports, a subport missing segmentation_type and a sub_ports extension before HTTP, decodes the envelope-free trunk response and accepts only 200.
  Tests: `network/v2/extensions/trunks/native_contracts_test.go:TestNativeTrunkSubportOperations`, `network/v2/extensions/trunks/native_contracts_test.go:TestNativeTrunkStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/trunks call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/trunks.RemoveSubports`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.API.RemoveSubports`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithRemoveSubportsOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks.WithRemoveSubportsField`

- RemoveSubports PUTs an envelope-free {"sub_ports":[{"port_id":...}]} body (nil as null) to trunks/{id}/remove_subports, rejects an empty subport port_id and nil option before HTTP, decodes the envelope-free trunk response and accepts only 200.
  Tests: `network/v2/extensions/trunks/native_contracts_test.go:TestNativeTrunkSubportOperations`, `network/v2/extensions/trunks/native_contracts_test.go:TestNativeTrunkStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/trunks call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/portforwarding.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithCreateField`

- Create POSTs {"port_forwarding":{...}} to floatingips/{fip}/port_forwardings, always sending internal_port_id, internal_ip_address and protocol while omitting empty or zero port fields, keeps extensions inside the envelope, rejects a core-key extension collision before HTTP and accepts only 201/202.
  Tests: `network/v2/extensions/layer3/portforwarding/native_contracts_test.go:TestNativePortForwardingRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/portforwarding/native_contracts_test.go:TestNativePortForwardingStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/portforwarding call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/portforwarding.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.API.Get`

- Get GETs floatingips/{fip}/port_forwardings/{id}, decodes the port_forwarding key returning a zero value for {} or null, wraps errors for a missing key beside other data, non-object envelope or malformed JSON, and accepts only 200.
  Tests: `network/v2/extensions/layer3/portforwarding/native_contracts_test.go:TestNativePortForwardingRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/portforwarding/native_contracts_test.go:TestNativePortForwardingStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/portforwarding call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/portforwarding.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithListQuery`

- List GETs floatingips/{fip}/port_forwardings with ListOpts and extension query values, follows only singular port_forwarding_links next hrefs as received (not plural port_forwardings_links), yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/layer3/portforwarding/native_contracts_test.go:TestNativePortForwardingRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/portforwarding/native_contracts_test.go:TestNativePortForwardingStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/portforwarding call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/portforwarding.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.WithUpdateField`

- Update PUTs {"port_forwarding":{...}} to floatingips/{fip}/port_forwardings/{id} sending an explicit empty description pointer, omitting other empty values (an empty envelope when all are empty), keeps extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `network/v2/extensions/layer3/portforwarding/native_contracts_test.go:TestNativePortForwardingRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/portforwarding/native_contracts_test.go:TestNativePortForwardingStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/portforwarding call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/layer3/portforwarding.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/portforwarding.API.Delete`

- Delete DELETEs floatingips/{fip}/port_forwardings/{id} and accepts only 202/204.
  Tests: `network/v2/extensions/layer3/portforwarding/native_contracts_test.go:TestNativePortForwardingRoutesBodiesPagingAndDecode`, `network/v2/extensions/layer3/portforwarding/native_contracts_test.go:TestNativePortForwardingStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/layer3/portforwarding call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/policies.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithCreateField`

- Create POSTs {"policy":{...}} to qos/policies with extensions inside the envelope, always sending name and omitting false bool flags, rejects a core-key extension collision and nil option before HTTP, decodes rules as raw maps and accepts only 201.
  Tests: `network/v2/extensions/qos/policies/native_contracts_test.go:TestNativeQoSPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/qos/policies/native_contracts_test.go:TestNativeQoSPolicyStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/policies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/policies.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.Get`

- Get GETs qos/policies/{raw id}, decodes the policy envelope with tags and raw rule maps, wraps a non-RFC3339 timestamp decode error and accepts only 200.
  Tests: `network/v2/extensions/qos/policies/native_contracts_test.go:TestNativeQoSPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/qos/policies/native_contracts_test.go:TestNativeQoSPolicyStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/policies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/policies.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithListQuery`

- List GETs qos/policies with ListOpts (pointer revision_number including 0) and extension query values, follows policies_links next hrefs as received, decodes null rules/description as zero values, yields nothing for an empty page, reports a non-array policies decode error, io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/qos/policies/native_contracts_test.go:TestNativeQoSPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/qos/policies/native_contracts_test.go:TestNativeQoSPolicyStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/policies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/policies.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.Delete`

- Delete DELETEs qos/policies/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/qos/policies/native_contracts_test.go:TestNativeQoSPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/qos/policies/native_contracts_test.go:TestNativeQoSPolicyStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/policies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/rules.CreateBandwidthLimitRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateBandwidthLimitRule`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithCreateBandwidthLimitRuleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithCreateBandwidthLimitRuleField`

- CreateBandwidthLimitRule POSTs {"bandwidth_limit_rule":{...}} to qos/policies/{policy}/bandwidth_limit_rules, always sending max_kbps including 0 and omitting empty optional fields, keeps extensions inside the envelope, rejects a max_kbps extension collision before HTTP, decodes the rule and accepts only 201.
  Tests: `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleRoutesBodiesAndDecode`, `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/rules.GetBandwidthLimitRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetBandwidthLimitRule`

- GetBandwidthLimitRule GETs qos/policies/{policy}/bandwidth_limit_rules/{id}, decodes the bandwidth_limit_rule envelope and accepts only 200.
  Tests: `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleRoutesBodiesAndDecode`, `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/rules.UpdateBandwidthLimitRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.UpdateBandwidthLimitRule`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithUpdateBandwidthLimitRuleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithUpdateBandwidthLimitRuleField`

- UpdateBandwidthLimitRule PUTs {"bandwidth_limit_rule":{...}} to qos/policies/{policy}/bandwidth_limit_rules/{id}, sending pointer MaxKBps/MaxBurstKBps values including 0 and an empty envelope when nothing is set, keeps extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleRoutesBodiesAndDecode`, `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/rules.DeleteBandwidthLimitRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteBandwidthLimitRule`

- DeleteBandwidthLimitRule DELETEs qos/policies/{policy}/bandwidth_limit_rules/{id} and accepts only 202/204.
  Tests: `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleRoutesBodiesAndDecode`, `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/rules.ListBandwidthLimitRules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ListBandwidthLimitRules`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithListBandwidthLimitRulesOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithListBandwidthLimitRulesQuery`

- ListBandwidthLimitRules GETs qos/policies/{policy}/bandwidth_limit_rules with list options and extension query values, ignores the Neutron bandwidth_limit_rules_links array and follows only a links.next string because the page defines no NextPageURL, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleRoutesBodiesAndDecode`, `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/rules.CreateDSCPMarkingRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateDSCPMarkingRule`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithCreateDSCPMarkingRuleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithCreateDSCPMarkingRuleField`

- CreateDSCPMarkingRule POSTs {"dscp_marking_rule":{...}} to qos/policies/{policy}/dscp_marking_rules, always sending dscp_mark including 0 and omitting empty optional fields, keeps extensions inside the envelope, rejects a dscp_mark extension collision before HTTP, decodes the rule and accepts only 201.
  Tests: `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleRoutesBodiesAndDecode`, `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/rules.GetDSCPMarkingRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetDSCPMarkingRule`

- GetDSCPMarkingRule GETs qos/policies/{policy}/dscp_marking_rules/{id}, decodes the dscp_marking_rule envelope and accepts only 200.
  Tests: `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleRoutesBodiesAndDecode`, `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/rules.UpdateDSCPMarkingRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.UpdateDSCPMarkingRule`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithUpdateDSCPMarkingRuleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithUpdateDSCPMarkingRuleField`

- UpdateDSCPMarkingRule PUTs {"dscp_marking_rule":{...}} to qos/policies/{policy}/dscp_marking_rules/{id}, sending pointer DSCPMark values including 0 and an empty envelope when nothing is set, keeps extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleRoutesBodiesAndDecode`, `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/rules.DeleteDSCPMarkingRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteDSCPMarkingRule`

- DeleteDSCPMarkingRule DELETEs qos/policies/{policy}/dscp_marking_rules/{id} and accepts only 202/204.
  Tests: `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleRoutesBodiesAndDecode`, `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/rules.ListDSCPMarkingRules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ListDSCPMarkingRules`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithListDSCPMarkingRulesOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithListDSCPMarkingRulesQuery`

- ListDSCPMarkingRules GETs qos/policies/{policy}/dscp_marking_rules with list options and extension query values, ignores the Neutron dscp_marking_rules_links array and follows only a links.next string because the page defines no NextPageURL, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleRoutesBodiesAndDecode`, `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/rules.CreateMinimumBandwidthRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateMinimumBandwidthRule`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithCreateMinimumBandwidthRuleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithCreateMinimumBandwidthRuleField`

- CreateMinimumBandwidthRule POSTs {"minimum_bandwidth_rule":{...}} to qos/policies/{policy}/minimum_bandwidth_rules, always sending min_kbps including 0 and omitting empty optional fields, keeps extensions inside the envelope, rejects a min_kbps extension collision before HTTP, decodes the rule and accepts only 201.
  Tests: `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleRoutesBodiesAndDecode`, `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/rules.GetMinimumBandwidthRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetMinimumBandwidthRule`

- GetMinimumBandwidthRule GETs qos/policies/{policy}/minimum_bandwidth_rules/{id}, decodes the minimum_bandwidth_rule envelope and accepts only 200.
  Tests: `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleRoutesBodiesAndDecode`, `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/rules.UpdateMinimumBandwidthRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.UpdateMinimumBandwidthRule`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithUpdateMinimumBandwidthRuleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithUpdateMinimumBandwidthRuleField`

- UpdateMinimumBandwidthRule PUTs {"minimum_bandwidth_rule":{...}} to qos/policies/{policy}/minimum_bandwidth_rules/{id}, sending pointer MinKBps values including 0 and an empty envelope when nothing is set, keeps extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleRoutesBodiesAndDecode`, `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/rules.DeleteMinimumBandwidthRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteMinimumBandwidthRule`

- DeleteMinimumBandwidthRule DELETEs qos/policies/{policy}/minimum_bandwidth_rules/{id} and accepts only 202/204.
  Tests: `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleRoutesBodiesAndDecode`, `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/rules.ListMinimumBandwidthRules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ListMinimumBandwidthRules`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithListMinimumBandwidthRulesOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithListMinimumBandwidthRulesQuery`

- ListMinimumBandwidthRules GETs qos/policies/{policy}/minimum_bandwidth_rules with list options and extension query values, ignores the Neutron minimum_bandwidth_rules_links array and follows only a links.next string because the page defines no NextPageURL, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleRoutesBodiesAndDecode`, `network/v2/extensions/qos/rules/native_contracts_test.go:TestNativeQoSRuleStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/ruletypes.GetRuleType`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.API.GetRuleType`

- GetRuleType GETs qos/rule-types/{name}, decodes the rule_type envelope with drivers and untyped parameter_values, wraps malformed or non-object envelope decode errors and accepts only 200.
  Tests: `network/v2/extensions/qos/ruletypes/native_contracts_test.go:TestNativeQoSRuleTypeGetAndList`, `network/v2/extensions/qos/ruletypes/native_contracts_test.go:TestNativeQoSRuleTypeStatusesAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/ruletypes call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/qos/ruletypes.ListRuleTypes`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.API.ListRuleTypes`

- ListRuleTypes GETs qos/rule-types as a single page without options or link following, decodes null drivers, yields nothing for an empty or missing rule_types list, and reports io.EOF for a bodyless 204 and native page status errors without operation context.
  Tests: `network/v2/extensions/qos/ruletypes/native_contracts_test.go:TestNativeQoSRuleTypeGetAndList`, `network/v2/extensions/qos/ruletypes/native_contracts_test.go:TestNativeQoSRuleTypeStatusesAndDecode`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/qos/ruletypes call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/security/addressgroups.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithCreateField`

- Create POSTs {"address_group":{...}} to address-groups with extensions inside the envelope, sending an optional caller ID and an empty address list as [], rejects nil addresses before HTTP, decodes null description as empty and accepts only 201/202.
  Tests: `network/v2/extensions/security/addressgroups/native_contracts_test.go:TestNativeAddressGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/security/addressgroups/native_contracts_test.go:TestNativeAddressGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/security/addressgroups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/security/addressgroups.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.Get`

- Get GETs address-groups/{raw id}, decodes the address_group envelope and accepts only 200.
  Tests: `network/v2/extensions/security/addressgroups/native_contracts_test.go:TestNativeAddressGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/security/addressgroups/native_contracts_test.go:TestNativeAddressGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/security/addressgroups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/security/addressgroups.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithListQuery`

- List GETs address-groups with ListOpts (repeated addresses query per value) and extension query values, follows address_groups_links next hrefs as received, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/security/addressgroups/native_contracts_test.go:TestNativeAddressGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/security/addressgroups/native_contracts_test.go:TestNativeAddressGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/security/addressgroups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/security/addressgroups.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithUpdateField`

- Update PUTs {"address_group":{...}} to address-groups/{id} sending explicit empty pointer name/description values with extensions inside the envelope, rejects a core-key extension collision before HTTP and accepts only 200.
  Tests: `network/v2/extensions/security/addressgroups/native_contracts_test.go:TestNativeAddressGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/security/addressgroups/native_contracts_test.go:TestNativeAddressGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/security/addressgroups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/security/addressgroups.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.Delete`

- Delete DELETEs address-groups/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/security/addressgroups/native_contracts_test.go:TestNativeAddressGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/security/addressgroups/native_contracts_test.go:TestNativeAddressGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/security/addressgroups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/security/addressgroups.AddAddresses`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.AddAddresses`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithAddAddressesOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithAddAddressesField`

- AddAddresses PUTs an envelope-free {"addresses":[...]} body with top-level extensions to address-groups/{id}/add_addresses, rejects nil addresses and an addresses extension before HTTP, decodes the address_group envelope and accepts only 200.
  Tests: `network/v2/extensions/security/addressgroups/native_contracts_test.go:TestNativeAddressGroupAddressOperations`, `network/v2/extensions/security/addressgroups/native_contracts_test.go:TestNativeAddressGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/security/addressgroups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/security/addressgroups.RemoveAddresses`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.API.RemoveAddresses`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithRemoveAddressesOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups.WithRemoveAddressesField`

- RemoveAddresses PUTs an envelope-free {"addresses":[...]} body (empty list as []) to address-groups/{id}/remove_addresses, rejects nil addresses before HTTP, decodes the address_group envelope and accepts only 200.
  Tests: `network/v2/extensions/security/addressgroups/native_contracts_test.go:TestNativeAddressGroupAddressOperations`, `network/v2/extensions/security/addressgroups/native_contracts_test.go:TestNativeAddressGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/security/addressgroups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/rbacpolicies.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithCreateField`

- Create POSTs {"rbac_policy":{...}} to rbac-policies with extensions inside the envelope, rejects missing action/object_type/object_id/target_tenant and a core-key extension collision before HTTP, decodes the rbac_policy key and accepts only 201/202.
  Tests: `network/v2/extensions/rbacpolicies/native_contracts_test.go:TestNativeRBACPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/rbacpolicies/native_contracts_test.go:TestNativeRBACPolicyStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/rbacpolicies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/rbacpolicies.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.API.Get`

- Get GETs rbac-policies/{raw id}, decodes the rbac_policy key returning a zero value for {} or null, wraps errors for a missing key beside other data or a non-object envelope, and accepts only 200.
  Tests: `network/v2/extensions/rbacpolicies/native_contracts_test.go:TestNativeRBACPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/rbacpolicies/native_contracts_test.go:TestNativeRBACPolicyStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/rbacpolicies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/rbacpolicies.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithListQuery`

- List GETs rbac-policies with ListOpts and extension query values, ignores the Neutron rbac_policies_links array and follows only a links.next string because the page defines no NextPageURL, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/rbacpolicies/native_contracts_test.go:TestNativeRBACPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/rbacpolicies/native_contracts_test.go:TestNativeRBACPolicyStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/rbacpolicies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/rbacpolicies.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.WithUpdateField`

- Update PUTs {"rbac_policy":{"target_tenant":...}} to rbac-policies/{id} with extensions inside the envelope, rejects an empty target_tenant and nil option before HTTP, and accepts only 200/201.
  Tests: `network/v2/extensions/rbacpolicies/native_contracts_test.go:TestNativeRBACPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/rbacpolicies/native_contracts_test.go:TestNativeRBACPolicyStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/rbacpolicies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/rbacpolicies.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/rbacpolicies.API.Delete`

- Delete DELETEs rbac-policies/{id} and accepts only 202/204.
  Tests: `network/v2/extensions/rbacpolicies/native_contracts_test.go:TestNativeRBACPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/rbacpolicies/native_contracts_test.go:TestNativeRBACPolicyStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/rbacpolicies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/attributestags.ReplaceAll`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.API.ReplaceAll`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.WithReplaceAllOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.WithReplaceAllField`

- ReplaceAll PUTs an envelope-free {"tags":[...]} body (empty list as []) with top-level extensions to {type}/{id}/tags, rejects nil tags, a tags extension and nil option before HTTP, returns the tags list and accepts only 200.
  Tests: `network/v2/extensions/attributestags/native_contracts_test.go:TestNativeAttributeTagRoutesAndBodies`, `network/v2/extensions/attributestags/native_contracts_test.go:TestNativeAttributeTagStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/attributestags call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/attributestags.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.API.List`

- List GETs {type}/{id}/tags, returns the tags list and accepts only 200.
  Tests: `network/v2/extensions/attributestags/native_contracts_test.go:TestNativeAttributeTagRoutesAndBodies`, `network/v2/extensions/attributestags/native_contracts_test.go:TestNativeAttributeTagStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/attributestags call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/attributestags.Add`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.API.Add`

- Add PUTs a bodyless request to {type}/{id}/tags/{tag} with the resource type and tag unescaped and accepts only 201.
  Tests: `network/v2/extensions/attributestags/native_contracts_test.go:TestNativeAttributeTagRoutesAndBodies`, `network/v2/extensions/attributestags/native_contracts_test.go:TestNativeAttributeTagStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/attributestags call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/attributestags.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.API.Delete`

- Delete DELETEs {type}/{id}/tags/{tag} and accepts only 204, rejecting 202.
  Tests: `network/v2/extensions/attributestags/native_contracts_test.go:TestNativeAttributeTagRoutesAndBodies`, `network/v2/extensions/attributestags/native_contracts_test.go:TestNativeAttributeTagStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/attributestags call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/attributestags.DeleteAll`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.API.DeleteAll`

- DeleteAll DELETEs {type}/{id}/tags and accepts only 204, rejecting 202.
  Tests: `network/v2/extensions/attributestags/native_contracts_test.go:TestNativeAttributeTagRoutesAndBodies`, `network/v2/extensions/attributestags/native_contracts_test.go:TestNativeAttributeTagStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/attributestags call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/attributestags.Confirm`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/attributestags.API.Confirm`

- Confirm GETs {type}/{id}/tags/{tag}, returns true for 204, false without error for 404, and false with the wrapped native status error otherwise.
  Tests: `network/v2/extensions/attributestags/native_contracts_test.go:TestNativeAttributeTagRoutesAndBodies`, `network/v2/extensions/attributestags/native_contracts_test.go:TestNativeAttributeTagStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/attributestags call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/apiversions.ListVersions`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/apiversions.API.ListVersions`

- ListVersions GETs the client Endpoint root with the version segment and query removed (ignoring ResourceBase), reads one page without link following, and reports io.EOF for a bodyless 204 and native page status errors.
  Tests: `network/v2/apiversions/native_contracts_test.go:TestNativeNetworkAPIVersionsUseTheUnversionedRoot`, `network/v2/apiversions/native_contracts_test.go:TestNativeNetworkAPIVersionsStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/apiversions call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/apiversions.ListVersionResources`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/apiversions.API.ListVersionResources`

- ListVersionResources GETs {endpoint root}/{version}/ with a trailing version slash trimmed, decodes name/collection resources from one page, and reports io.EOF for a bodyless 204 and native page status errors.
  Tests: `network/v2/apiversions/native_contracts_test.go:TestNativeNetworkAPIVersionsUseTheUnversionedRoot`, `network/v2/apiversions/native_contracts_test.go:TestNativeNetworkAPIVersionsStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/apiversions call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`

- List GETs extensions from ResourceBase as a single page without link following, yields nothing for an empty list, and reports io.EOF for a bodyless 204 and native page status errors.
  Tests: `network/v2/extensions/native_contracts_test.go:TestNativeNetworkExtensionsRoutesAndDecode`, `network/v2/extensions/native_contracts_test.go:TestNativeNetworkExtensionsStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.Get`

- Get GETs extensions/{alias}, decodes the extension envelope keeping updated as the raw string and accepts only 200.
  Tests: `network/v2/extensions/native_contracts_test.go:TestNativeNetworkExtensionsRoutesAndDecode`, `network/v2/extensions/native_contracts_test.go:TestNativeNetworkExtensionsStatuses`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/groups.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.WithCreateField`

- Create POSTs {"firewall_group":{...}} to fwaas/firewall_groups with extensions inside the envelope, sending explicit false pointer flags and decoding null policy IDs as empty, rejects a core-key extension collision before HTTP and accepts only 201/202.
  Tests: `network/v2/extensions/fwaas_v2/groups/native_contracts_test.go:TestNativeFirewallGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/fwaas_v2/groups/native_contracts_test.go:TestNativeFirewallGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/groups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/groups.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.Get`

- Get GETs fwaas/firewall_groups/{raw id}, decodes the firewall_group envelope and accepts only 200.
  Tests: `network/v2/extensions/fwaas_v2/groups/native_contracts_test.go:TestNativeFirewallGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/fwaas_v2/groups/native_contracts_test.go:TestNativeFirewallGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/groups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/groups.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.WithListQuery`

- List GETs fwaas/firewall_groups with ListOpts (repeated ports query per value) and extension query values, follows firewall_groups_links next hrefs as received, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/fwaas_v2/groups/native_contracts_test.go:TestNativeFirewallGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/fwaas_v2/groups/native_contracts_test.go:TestNativeFirewallGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/groups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/groups.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.WithUpdateField`

- Update PUTs {"firewall_group":{...}} to fwaas/firewall_groups/{id} sending explicit empty pointer strings and an empty ports list as [] with extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `network/v2/extensions/fwaas_v2/groups/native_contracts_test.go:TestNativeFirewallGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/fwaas_v2/groups/native_contracts_test.go:TestNativeFirewallGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/groups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/groups.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.Delete`

- Delete DELETEs fwaas/firewall_groups/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/fwaas_v2/groups/native_contracts_test.go:TestNativeFirewallGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/fwaas_v2/groups/native_contracts_test.go:TestNativeFirewallGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/groups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/groups.RemoveIngressPolicy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.RemoveIngressPolicy`

- RemoveIngressPolicy PUTs {"firewall_group":{"ingress_firewall_policy_id":null}} to fwaas/firewall_groups/{id}, decodes the group and accepts only 200.
  Tests: `network/v2/extensions/fwaas_v2/groups/native_contracts_test.go:TestNativeFirewallGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/fwaas_v2/groups/native_contracts_test.go:TestNativeFirewallGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/groups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/groups.RemoveEgressPolicy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.RemoveEgressPolicy`

- RemoveEgressPolicy PUTs {"firewall_group":{"egress_firewall_policy_id":null}} to fwaas/firewall_groups/{id}, decodes the group and accepts only 200.
  Tests: `network/v2/extensions/fwaas_v2/groups/native_contracts_test.go:TestNativeFirewallGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/fwaas_v2/groups/native_contracts_test.go:TestNativeFirewallGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/groups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/policies.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithCreateField`

- Create POSTs {"firewall_policy":{...}} to fwaas/firewall_policies with extensions inside the envelope, sending explicit false audited/shared pointers and the rule list, rejects a core-key extension collision before HTTP and accepts only 201/202.
  Tests: `network/v2/extensions/fwaas_v2/policies/native_contracts_test.go:TestNativeFirewallPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/fwaas_v2/policies/native_contracts_test.go:TestNativeFirewallPolicyStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/policies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/policies.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.Get`

- Get GETs fwaas/firewall_policies/{raw id}, decodes the firewall_policy envelope and accepts only 200.
  Tests: `network/v2/extensions/fwaas_v2/policies/native_contracts_test.go:TestNativeFirewallPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/fwaas_v2/policies/native_contracts_test.go:TestNativeFirewallPolicyStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/policies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/policies.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithListQuery`

- List GETs fwaas/firewall_policies with ListOpts and extension query values, follows firewall_policies_links next hrefs as received, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/fwaas_v2/policies/native_contracts_test.go:TestNativeFirewallPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/fwaas_v2/policies/native_contracts_test.go:TestNativeFirewallPolicyStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/policies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/policies.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithUpdateField`

- Update PUTs {"firewall_policy":{...}} to fwaas/firewall_policies/{id} sending an empty firewall_rules list as [] and explicit false pointers with extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `network/v2/extensions/fwaas_v2/policies/native_contracts_test.go:TestNativeFirewallPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/fwaas_v2/policies/native_contracts_test.go:TestNativeFirewallPolicyStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/policies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/policies.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.Delete`

- Delete DELETEs fwaas/firewall_policies/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/fwaas_v2/policies/native_contracts_test.go:TestNativeFirewallPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/fwaas_v2/policies/native_contracts_test.go:TestNativeFirewallPolicyStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/policies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/policies.InsertRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.InsertRule`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithInsertRuleOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.WithInsertRuleField`

- InsertRule PUTs an envelope-free {"firewall_rule_id":...,"insert_before"|"insert_after":...} body with top-level extensions to fwaas/firewall_policies/{id}/insert_rule, requires the rule ID and exactly one position before HTTP, rejects a firewall_rule_id extension, decodes the envelope-free policy and accepts only 200.
  Tests: `network/v2/extensions/fwaas_v2/policies/native_contracts_test.go:TestNativeFirewallPolicyRuleOperations`, `network/v2/extensions/fwaas_v2/policies/native_contracts_test.go:TestNativeFirewallPolicyStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/policies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/policies.RemoveRule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.RemoveRule`

- RemoveRule PUTs {"firewall_rule_id":...} (including an empty ID) to fwaas/firewall_policies/{id}/remove_rule, decodes the envelope-free policy and accepts only 200.
  Tests: `network/v2/extensions/fwaas_v2/policies/native_contracts_test.go:TestNativeFirewallPolicyRuleOperations`, `network/v2/extensions/fwaas_v2/policies/native_contracts_test.go:TestNativeFirewallPolicyStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/policies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/rules.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithCreateField`

- Create POSTs {"firewall_rule":{...}} to fwaas/firewall_rules with extensions inside the envelope, requiring protocol and action, sending protocol any as null and explicit false enabled, decoding firewall_policy_id only as a list, rejects a core-key extension collision before HTTP and accepts only 201/202.
  Tests: `network/v2/extensions/fwaas_v2/rules/native_contracts_test.go:TestNativeFirewallRuleRoutesBodiesPagingAndDecode`, `network/v2/extensions/fwaas_v2/rules/native_contracts_test.go:TestNativeFirewallRuleStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/rules.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.Get`

- Get GETs fwaas/firewall_rules/{raw id}, decodes the firewall_rule envelope and accepts only 200.
  Tests: `network/v2/extensions/fwaas_v2/rules/native_contracts_test.go:TestNativeFirewallRuleRoutesBodiesPagingAndDecode`, `network/v2/extensions/fwaas_v2/rules/native_contracts_test.go:TestNativeFirewallRuleStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/rules.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithListQuery`

- List GETs fwaas/firewall_rules with ListOpts and extension query values, follows firewall_rules_links next hrefs as received, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/fwaas_v2/rules/native_contracts_test.go:TestNativeFirewallRuleRoutesBodiesPagingAndDecode`, `network/v2/extensions/fwaas_v2/rules/native_contracts_test.go:TestNativeFirewallRuleStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/rules.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithUpdateField`

- Update PUTs {"firewall_rule":{...}} to fwaas/firewall_rules/{id} sending explicit empty pointer strings with extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `network/v2/extensions/fwaas_v2/rules/native_contracts_test.go:TestNativeFirewallRuleRoutesBodiesPagingAndDecode`, `network/v2/extensions/fwaas_v2/rules/native_contracts_test.go:TestNativeFirewallRuleStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/fwaas_v2/rules.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.Delete`

- Delete DELETEs fwaas/firewall_rules/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/fwaas_v2/rules/native_contracts_test.go:TestNativeFirewallRuleRoutesBodiesPagingAndDecode`, `network/v2/extensions/fwaas_v2/rules/native_contracts_test.go:TestNativeFirewallRuleStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/fwaas_v2/rules call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/endpointgroups.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.WithCreateField`

- Create POSTs {"endpoint_group":{...}} to vpn/endpoint-groups with extensions inside the envelope, sending nil endpoints as null, rejects a core-key extension collision before HTTP and accepts only 201/202.
  Tests: `network/v2/extensions/vpnaas/endpointgroups/native_contracts_test.go:TestNativeVPNEndpointGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/endpointgroups/native_contracts_test.go:TestNativeVPNEndpointGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/endpointgroups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/endpointgroups.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.API.Get`

- Get GETs vpn/endpoint-groups/{raw id}, decodes the endpoint_group envelope and accepts only 200.
  Tests: `network/v2/extensions/vpnaas/endpointgroups/native_contracts_test.go:TestNativeVPNEndpointGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/endpointgroups/native_contracts_test.go:TestNativeVPNEndpointGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/endpointgroups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/endpointgroups.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.WithListQuery`

- List GETs vpn/endpoint-groups with ListOpts and extension query values, follows endpoint_groups_links next hrefs as received, decodes null names as empty, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/vpnaas/endpointgroups/native_contracts_test.go:TestNativeVPNEndpointGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/endpointgroups/native_contracts_test.go:TestNativeVPNEndpointGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/endpointgroups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/endpointgroups.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.WithUpdateField`

- Update PUTs {"endpoint_group":{...}} to vpn/endpoint-groups/{id} sending explicit empty pointer name/description with extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `network/v2/extensions/vpnaas/endpointgroups/native_contracts_test.go:TestNativeVPNEndpointGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/endpointgroups/native_contracts_test.go:TestNativeVPNEndpointGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/endpointgroups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/endpointgroups.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.API.Delete`

- Delete DELETEs vpn/endpoint-groups/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/vpnaas/endpointgroups/native_contracts_test.go:TestNativeVPNEndpointGroupRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/endpointgroups/native_contracts_test.go:TestNativeVPNEndpointGroupStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/endpointgroups call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/ikepolicies.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithCreateField`

- Create POSTs {"ikepolicy":{...}} to vpn/ikepolicies with extensions inside the envelope, sending the phase1_negotiation_mode key and a nested lifetime object (empty when pointed but unset), rejects a core-key extension collision before HTTP and accepts only 201/202.
  Tests: `network/v2/extensions/vpnaas/ikepolicies/native_contracts_test.go:TestNativeVPNPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/ikepolicies/native_contracts_test.go:TestNativeVPNPolicyStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/ikepolicies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/ikepolicies.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.API.Get`

- Get GETs vpn/ikepolicies/{raw id}, decodes the ikepolicy envelope and accepts only 200.
  Tests: `network/v2/extensions/vpnaas/ikepolicies/native_contracts_test.go:TestNativeVPNPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/ikepolicies/native_contracts_test.go:TestNativeVPNPolicyStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/ikepolicies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/ikepolicies.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithListQuery`

- List GETs vpn/ikepolicies with ListOpts (phase_1_negotiation_mode filter key) and extension query values, follows ikepolicies_links next hrefs as received, decodes null names as empty, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/vpnaas/ikepolicies/native_contracts_test.go:TestNativeVPNPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/ikepolicies/native_contracts_test.go:TestNativeVPNPolicyStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/ikepolicies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/ikepolicies.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithUpdateField`

- Update PUTs {"ikepolicy":{...}} to vpn/ikepolicies/{id} sending the differently spelled phase_1_negotiation_mode key and explicit empty pointer name with extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `network/v2/extensions/vpnaas/ikepolicies/native_contracts_test.go:TestNativeVPNPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/ikepolicies/native_contracts_test.go:TestNativeVPNPolicyStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/ikepolicies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/ikepolicies.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.API.Delete`

- Delete DELETEs vpn/ikepolicies/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/vpnaas/ikepolicies/native_contracts_test.go:TestNativeVPNPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/ikepolicies/native_contracts_test.go:TestNativeVPNPolicyStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/ikepolicies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/ipsecpolicies.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithCreateField`

- Create POSTs {"ipsecpolicy":{...}} to vpn/ipsecpolicies with extensions inside the envelope, omitting every empty optional field and nesting lifetime, rejects a core-key extension collision before HTTP and accepts only 201/202.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/native_contracts_test.go:TestNativeVPNPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/ipsecpolicies/native_contracts_test.go:TestNativeVPNPolicyStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/ipsecpolicies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/ipsecpolicies.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.Get`

- Get GETs vpn/ipsecpolicies/{raw id}, decodes the ipsecpolicy envelope and accepts only 200.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/native_contracts_test.go:TestNativeVPNPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/ipsecpolicies/native_contracts_test.go:TestNativeVPNPolicyStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/ipsecpolicies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/ipsecpolicies.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithListQuery`

- List GETs vpn/ipsecpolicies with ListOpts and extension query values, follows ipsecpolicies_links next hrefs as received, decodes null names as empty, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/native_contracts_test.go:TestNativeVPNPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/ipsecpolicies/native_contracts_test.go:TestNativeVPNPolicyStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/ipsecpolicies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/ipsecpolicies.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithUpdateField`

- Update PUTs {"ipsecpolicy":{...}} to vpn/ipsecpolicies/{id} sending explicit empty pointer description and a partial lifetime with extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/native_contracts_test.go:TestNativeVPNPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/ipsecpolicies/native_contracts_test.go:TestNativeVPNPolicyStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/ipsecpolicies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/ipsecpolicies.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.Delete`

- Delete DELETEs vpn/ipsecpolicies/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/native_contracts_test.go:TestNativeVPNPolicyRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/ipsecpolicies/native_contracts_test.go:TestNativeVPNPolicyStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/ipsecpolicies call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/services.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.WithCreateField`

- Create POSTs {"vpnservice":{...}} to vpn/vpnservices with extensions inside the envelope, requiring router_id before HTTP, sending explicit false and nil admin_state_up as null, decoding null subnet_id as empty, rejects a core-key extension collision before HTTP and accepts only 201/202.
  Tests: `network/v2/extensions/vpnaas/services/native_contracts_test.go:TestNativeVPNServiceRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/services/native_contracts_test.go:TestNativeVPNServiceStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/services call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/services.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.Get`

- Get GETs vpn/vpnservices/{raw id}, decodes the vpnservice envelope and accepts only 200.
  Tests: `network/v2/extensions/vpnaas/services/native_contracts_test.go:TestNativeVPNServiceRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/services/native_contracts_test.go:TestNativeVPNServiceStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/services call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/services.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.WithListQuery`

- List GETs vpn/vpnservices with ListOpts and extension query values, follows vpnservices_links next hrefs as received, decodes null names as empty, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/vpnaas/services/native_contracts_test.go:TestNativeVPNServiceRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/services/native_contracts_test.go:TestNativeVPNServiceStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/services call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/services.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.WithUpdateField`

- Update PUTs {"vpnservice":{...}} to vpn/vpnservices/{id} sending explicit empty pointer name and admin_state_up with extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `network/v2/extensions/vpnaas/services/native_contracts_test.go:TestNativeVPNServiceRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/services/native_contracts_test.go:TestNativeVPNServiceStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/services call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/services.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.Delete`

- Delete DELETEs vpn/vpnservices/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/vpnaas/services/native_contracts_test.go:TestNativeVPNServiceRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/services/native_contracts_test.go:TestNativeVPNServiceStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/services call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/siteconnections.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithCreateField`

- Create POSTs {"ipsec_site_connection":{...}} to vpn/ipsec-site-connections with extensions inside the envelope, always sending policy, service, peer and psk fields even when empty, nesting dpd and decoding psk/dpd/route_mode, rejects a core-key extension collision before HTTP and accepts only 201/202.
  Tests: `network/v2/extensions/vpnaas/siteconnections/native_contracts_test.go:TestNativeVPNConnectionRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/siteconnections/native_contracts_test.go:TestNativeVPNConnectionStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/siteconnections call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/siteconnections.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.API.Get`

- Get GETs vpn/ipsec-site-connections/{raw id}, decodes the ipsec_site_connection envelope and accepts only 200.
  Tests: `network/v2/extensions/vpnaas/siteconnections/native_contracts_test.go:TestNativeVPNConnectionRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/siteconnections/native_contracts_test.go:TestNativeVPNConnectionStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/siteconnections call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/siteconnections.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithListQuery`

- List GETs vpn/ipsec-site-connections with ListOpts (psk filter sent in the query string) and extension query values, follows ipsec_site_connections_links next hrefs as received, decodes null names as empty, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/vpnaas/siteconnections/native_contracts_test.go:TestNativeVPNConnectionRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/siteconnections/native_contracts_test.go:TestNativeVPNConnectionStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/siteconnections call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/siteconnections.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithUpdateField`

- Update PUTs {"ipsec_site_connection":{...}} to vpn/ipsec-site-connections/{id} sending explicit empty pointer name, false admin_state_up and mtu with extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `network/v2/extensions/vpnaas/siteconnections/native_contracts_test.go:TestNativeVPNConnectionRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/siteconnections/native_contracts_test.go:TestNativeVPNConnectionStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/siteconnections call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/vpnaas/siteconnections.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.API.Delete`

- Delete DELETEs vpn/ipsec-site-connections/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/vpnaas/siteconnections/native_contracts_test.go:TestNativeVPNConnectionRoutesBodiesPagingAndDecode`, `network/v2/extensions/vpnaas/siteconnections/native_contracts_test.go:TestNativeVPNConnectionStrictStatusesAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/vpnaas/siteconnections call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/taas/tapmirrors.Create`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.WithCreateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.WithCreateField`

- Create POSTs {"tap_mirror":{...}} to taas/tap_mirrors with extensions inside the envelope, always sending name, port_id, mirror_type, remote_ip and directions (zero as {}) with string-quoted IN/OUT tunnel IDs, rejects a core-key extension collision before HTTP and accepts only 201/202.
  Tests: `network/v2/extensions/taas/tapmirrors/native_contracts_test.go:TestNativeTapMirrorRoutesBodiesPagingAndDecode`, `network/v2/extensions/taas/tapmirrors/native_contracts_test.go:TestNativeTapMirrorStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/taas/tapmirrors call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/taas/tapmirrors.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.API.Get`

- Get GETs taas/tap_mirrors/{raw id}, decodes string-quoted direction IDs, wraps a decode error for unquoted numeric IDs and accepts only 200.
  Tests: `network/v2/extensions/taas/tapmirrors/native_contracts_test.go:TestNativeTapMirrorRoutesBodiesPagingAndDecode`, `network/v2/extensions/taas/tapmirrors/native_contracts_test.go:TestNativeTapMirrorStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/taas/tapmirrors call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/taas/tapmirrors.List`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.WithListQuery`

- List GETs taas/tap_mirrors with ListOpts and extension query values, follows tap_mirrors_links next hrefs as received, yields nothing for an empty page, reports io.EOF for a bodyless 204 and native page status errors without operation context, and rejects a nil option before HTTP.
  Tests: `network/v2/extensions/taas/tapmirrors/native_contracts_test.go:TestNativeTapMirrorRoutesBodiesPagingAndDecode`, `network/v2/extensions/taas/tapmirrors/native_contracts_test.go:TestNativeTapMirrorStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/taas/tapmirrors call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/taas/tapmirrors.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.WithUpdateOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.WithUpdateField`

- Update PUTs {"tap_mirror":{...}} to taas/tap_mirrors/{id} sending explicit empty pointer name/description with extensions inside the envelope, rejects a nil option before HTTP and accepts only 200.
  Tests: `network/v2/extensions/taas/tapmirrors/native_contracts_test.go:TestNativeTapMirrorRoutesBodiesPagingAndDecode`, `network/v2/extensions/taas/tapmirrors/native_contracts_test.go:TestNativeTapMirrorStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/taas/tapmirrors call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

### `gophercloud:networking/v2/extensions/taas/tapmirrors.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors.API.Delete`

- Delete DELETEs taas/tap_mirrors/{raw id} and accepts only 202/204.
  Tests: `network/v2/extensions/taas/tapmirrors/native_contracts_test.go:TestNativeTapMirrorRoutesBodiesPagingAndDecode`, `network/v2/extensions/taas/tapmirrors/native_contracts_test.go:TestNativeTapMirrorStrictStatusesDecodeAndPreflight`.

Python/Go differences:

- This declaration maps the Gophercloud native networking/v2/extensions/taas/tapmirrors call only; Python resource models, finders and waits are tracked separately.
- The SDK adds only request.Wrap operation context (not on list streams) and the documented extension merge; native URL construction, OkCodes, paging and decoding are unchanged.

## References

- [networks](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/networks/requests.go)
- [subnets](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/subnets/requests.go)
- [ports](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/ports/requests.go)
- [Revision Updates](revision-updates.md)
- [Floating Ip Ensure](../floating-ip-ensure.md)
- [scoped resource](../../docs/scoped-resources.md)
