# Python Cloud Parity A

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/get_network`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithRecordFindQuery`

- FindRecord with ignore-missing true and a filter query sends GET /{collection}/{name_or_id}?filters first, falls back to GET /{collection}?name=...&filters, returns the single match, and returns nil without error when nothing matches, for networks, routers, subnets and ports.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudNetworkGetHelpersFindByIDThenNameWithFilters`.

Python/Go differences:

- Python returns a Network Resource or None; Go returns a record with Resource/Wire attribute views and the raw response, or nil.
- The cloud filters dict maps to repeated WithRecordFindQuery options; the docstring's jmespath string form does not work in Python either because filters are expanded with **.
- Python's ignore_missing=True is explicit here; Go callers pass WithRecordFindIgnoreMissing(true), which is also the FindRecord default.
- Python negotiates the Neutron microversion per resource; Go sends the configured client version (Neutron v2.0 has no microversions in practice).

### `python:cloud/_network/NetworkCloudMixin/get_router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordFindQuery`

- FindRecord with ignore-missing true and a filter query sends GET /{collection}/{name_or_id}?filters first, falls back to GET /{collection}?name=...&filters, returns the single match, and returns nil without error when nothing matches, for networks, routers, subnets and ports.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudNetworkGetHelpersFindByIDThenNameWithFilters`.

Python/Go differences:

- Python returns a Router Resource or None; Go returns a record with Resource/Wire attribute views and the raw response, or nil.
- The cloud filters dict maps to repeated WithRecordFindQuery options; the docstring's jmespath string form does not work in Python either because filters are expanded with **.
- Python's ignore_missing=True is explicit here; Go callers pass WithRecordFindIgnoreMissing(true), which is also the FindRecord default.
- Python negotiates the Neutron microversion per resource; Go sends the configured client version (Neutron v2.0 has no microversions in practice).

### `python:cloud/_network/NetworkCloudMixin/get_subnet`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordFindQuery`

- FindRecord with ignore-missing true and a filter query sends GET /{collection}/{name_or_id}?filters first, falls back to GET /{collection}?name=...&filters, returns the single match, and returns nil without error when nothing matches, for networks, routers, subnets and ports.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudNetworkGetHelpersFindByIDThenNameWithFilters`.

Python/Go differences:

- Python returns a Subnet Resource or None; Go returns a record with Resource/Wire attribute views and the raw response, or nil.
- The cloud filters dict maps to repeated WithRecordFindQuery options; the docstring's jmespath string form does not work in Python either because filters are expanded with **.
- Python's ignore_missing=True is explicit here; Go callers pass WithRecordFindIgnoreMissing(true), which is also the FindRecord default.
- Python negotiates the Neutron microversion per resource; Go sends the configured client version (Neutron v2.0 has no microversions in practice).

### `python:cloud/_network/NetworkCloudMixin/get_port`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordFindQuery`

- FindRecord with ignore-missing true and a filter query sends GET /{collection}/{name_or_id}?filters first, falls back to GET /{collection}?name=...&filters, returns the single match, and returns nil without error when nothing matches, for networks, routers, subnets and ports.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudNetworkGetHelpersFindByIDThenNameWithFilters`.

Python/Go differences:

- Python returns a Port Resource or None; Go returns a record with Resource/Wire attribute views and the raw response, or nil.
- The cloud filters dict maps to repeated WithRecordFindQuery options; the docstring's jmespath string form does not work in Python either because filters are expanded with **.
- Python's ignore_missing=True is explicit here; Go callers pass WithRecordFindIgnoreMissing(true), which is also the FindRecord default.
- Python negotiates the Neutron microversion per resource; Go sends the configured client version (Neutron v2.0 has no microversions in practice).

### `python:cloud/_network/NetworkCloudMixin/get_network_by_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.Get`

- networks.API.Get sends GET /networks/{id}, decodes the network, and returns the 404 as an error just like the Python proxy get_network.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudNetworkGetByIDIsPlainGet`.

Python/Go differences:

- Python docstring says None when missing but the proxy raises NotFoundException; Go returns the Gophercloud 404 error.
- Go returns the native Gophercloud Network struct, which omits some Python Network extension attributes (for example provider and DNS fields).

### `python:cloud/_network/NetworkCloudMixin/get_port_by_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.Get`

- ports.API.Get sends GET /ports/{id}, decodes the port, and returns the 404 as an error just like the Python proxy get_port.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudNetworkGetByIDIsPlainGet`.

Python/Go differences:

- Python docstring says None when missing but the proxy raises NotFoundException; Go returns the Gophercloud 404 error.
- Go returns the native Gophercloud Port struct rather than the Python Port Resource with every extension attribute.

### `python:cloud/_network/NetworkCloudMixin/get_network_extensions`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`

- extensions.API.List sends one GET /extensions and yields every extension whose Alias values form the same set Python returns.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudNetworkExtensionsListAliases`.

Python/Go differences:

- Python returns set[str] of aliases; Go yields Extension values through an iterator and the caller collects Alias.

### `python:cloud/_network/NetworkCloudMixin/add_router_interface`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.AddInterfaceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfaceSubnet`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfacePort`

- AddInterfaceRecord sends PUT /routers/{id}/add_router_interface with port_id when a truthy port is given, otherwise subnet_id (null when neither is given), and returns the whole response JSON.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudNetworkAddRouterInterfaceBodies`.

Python/Go differences:

- Python accepts a Router object or ID; Go takes the router ID string.
- Python returns the response dict; Go returns RouterJSONResult with the raw JSON Value, headers and status.
- The docstring says passing both IDs is an error, but Python and Go both send only port_id in that case.

### `python:cloud/_network/NetworkCloudMixin/create_port`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.CreateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithAttribute`

- CreateWithAttributes with network_id plus forwarded attributes sends POST /ports with exactly those keys, including wire names such as binding:vnic_type and explicit false values.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudNetworkCreatePortForwardsKwargs`.

Python/Go differences:

- Python rejects kwargs outside its valid_kwargs list; Go WithAttribute forwards any key.
- Python returns a Port Resource; Go returns the native Gophercloud Port struct.
- Python negotiates the Neutron microversion per resource; Go sends the configured client version (Neutron v2.0 has no microversions in practice).

### `python:cloud/_network/NetworkCloudMixin/delete_port`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.DeleteWithOptions`

- FindRecord with ignore-missing returns nil without any DELETE for an absent name; otherwise DeleteWithOptions on the found ID sends DELETE without a body, ignores a racing 404, and returns a 409 as an error, for ports, routers and subnets.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudNetworkDeleteHelpersFindThenDelete`.

Python/Go differences:

- Python returns True/False; Go callers derive the bool from whether FindRecord returned a record and pass its id to DeleteWithOptions.
- Python passes the found Resource to the proxy delete; Go passes the found ID string.
- Python logs a debug message when nothing is found; Go does not log.

### `python:cloud/_network/NetworkCloudMixin/delete_router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.DeleteWithOptions`

- FindRecord with ignore-missing returns nil without any DELETE for an absent name; otherwise DeleteWithOptions on the found ID sends DELETE without a body, ignores a racing 404, and returns a 409 as an error, for ports, routers and subnets.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudNetworkDeleteHelpersFindThenDelete`.

Python/Go differences:

- Python returns True/False; Go callers derive the bool from whether FindRecord returned a record and pass its id to DeleteWithOptions.
- Python passes the found Resource to the proxy delete; Go passes the found ID string.
- Python logs a debug message when nothing is found; Go does not log.

### `python:cloud/_network/NetworkCloudMixin/delete_subnet`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.DeleteWithOptions`

- FindRecord with ignore-missing returns nil without any DELETE for an absent name; otherwise DeleteWithOptions on the found ID sends DELETE without a body, ignores a racing 404, and returns a 409 as an error, for ports, routers and subnets.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudNetworkDeleteHelpersFindThenDelete`.

Python/Go differences:

- Python returns True/False; Go callers derive the bool from whether FindRecord returned a record and pass its id to DeleteWithOptions.
- Python passes the found Resource to the proxy delete; Go passes the found ID string.
- Python logs a debug message when nothing is found; Go does not log.

### `python:cloud/_network/NetworkCloudMixin/create_firewall_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.Create`

- Each firewall rule name or ID is resolved with Find(ID, ignore missing) then Find(Name), sending GET /fwaas/firewall_rules/{x} then GET /fwaas/firewall_rules?name=x; the IDs go into one POST /fwaas/firewall_policies, and an unknown rule returns ErrNotFound before any POST.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudFirewallPolicyCreateResolvesRuleNames`.

Python/Go differences:

- Python Resource.find falls back to the name list after a 400 or 403 member GET and matches rows by ID or name; Go Find(resource.ID) treats only 404 as absent and Find(resource.Name) uses the server-side name filter.
- Gophercloud CreateOpts uses omitempty, so an empty name or an empty firewall_rules list is omitted rather than sent; WithCreateField cannot override these core fields.
- Python returns a FirewallPolicy Resource; Go returns the native Gophercloud Policy struct.

### `python:cloud/_network/NetworkCloudMixin/create_firewall_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/policies.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.Name`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/groups.API.Create`

- Ingress/egress policy names resolve through Find(ID) then Find(Name), each port resolves through ports FindRecord with ignore-missing false (ErrNotFound before POST when absent), and one POST /fwaas/firewall_groups carries ingress_firewall_policy_id, egress_firewall_policy_id and the port IDs.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudFirewallGroupCreateResolvesPoliciesAndPorts`.

Python/Go differences:

- Python Resource.find falls back to the name list after a 400 or 403 member GET and matches rows by ID or name; Go Find(resource.ID) treats only 404 as absent and Find(resource.Name) uses the server-side name filter.
- Python maps ingress_firewall_policy/egress_firewall_policy=None to an explicit *_policy_id null and sends empty ports lists; Gophercloud CreateOpts omits empty strings and lists, so Go omits those keys.
- Python returns a FirewallGroup Resource; Go returns the native Gophercloud Group struct.

### `python:cloud/_network/NetworkCloudMixin/create_firewall_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithCreateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithCreateRecordField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/fwaas_v2/rules.WithCreateRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/request.Present`
- `github.com/JSYoo5B/go-openstacksdk/request.Null`

- CreateRecord sends POST fwaas/firewall_rules with an empty firewall_rule object for no arguments and otherwise only the supplied attributes, including protocol null, enabled false, an empty description, extension keys and a declared key with another JSON type through WithCreateRecordField when the typed field is omitted, accepts 200, 201 and 202, and decodes enveloped, flat and empty responses.
  Tests: `network/v2/extensions/fwaas_v2/rules/python_create_record_test.go:TestPythonFirewallRuleCreateRecordSendsOnlyGivenAttributes`.
- A blank extension key, a nil option, an extension that repeats a supplied typed key and a nil API fail before HTTP with resource.ErrInvalidOption, and a 400 response is returned as the Gophercloud status error wrapped in a rules resource.OperationError.
  Tests: `network/v2/extensions/fwaas_v2/rules/python_create_record_test.go:TestPythonFirewallRuleCreateRecordRejectsInputsAndErrorStatuses`.

Python/Go differences:

- Python returns the FirewallRule Resource merged from request attributes and the response; Go returns RuleRecord with the typed Rule, the raw response object fields, headers and status.
- Go rejects the same key supplied both in CreateRecordOpts and WithCreateRecordField, which Python keyword arguments cannot express.
- A response field that does not fit the Gophercloud Rule model, such as a string firewall_policy_id, fails the Go typed decode while Python keeps it.

### `python:cloud/_network/NetworkCloudMixin/create_router`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.CreateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.CreateRouterRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.CreateCloudRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateName`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateAdminStateUp`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateExternalGatewayNetworkID`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateEnableSNAT`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateExternalFixedIPs`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateProjectID`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateAvailabilityZoneHints`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterCreateCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk.Connection.CreateRouterRecord`

- routers.API.CreateWithAttributes sends only explicit attributes: no arguments produce {"router":{}}, and admin_state_up plus a caller-built external_gateway_info are sent as given.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudNetworkCreateRouterSubnetSendOnlyExplicitAttributes`.
- Cloud default admin_state_up=True; raw null/false/nonstring values stay in request. Name/net/fixed-IPs truthiness differs from project/enable_snat nonNone presence; zero/empty project survives, enable_snat:false alone builds gateway. Known7kwargs only; options/JSON/Opts/CurrentLocation are SDK-owned snapshots, no app builder required.
  Tests: `network/cloud_router_create_test.go:TestCloudRouterCreateDefaultAndSourceJSONPresenceTruthiness`, `network/cloud_router_create_test.go:TestCloudRouterCreateGatewayAssemblyAndOwnedConcreteOptions`, `network/cloud_router_create_test.go:TestCloudRouterCreatePreflightCancellationAndSelectedSourceGuards`.
- Omitted/null AZ hints send no discovery; nonnull must be list before any HTTP. Empty list and nonstring list elements remain valid Source input. Every requested hint call fully consumes fresh router_availability_zone inventory including later aliases/pages; failure has actual extension proof but no fabricated POST receipt.
  Tests: `network/cloud_router_create_test.go:TestCloudRouterCreateAZListValidationThenFreshCompleteGate`, `network/cloud_router_create_test.go:TestCloudRouterCreateAZMissingLateAliasAndPageFailuresPreserveOnlyGates`, `network/cloud_router_create_test.go:TestCloudRouterCreateAZPhysicalFailureHasExtensionProofAndNoPostReceipt`.
- One collection POST contains only translated cloud fields. Canonical admin seed is separate from actual wire admin_state_up; sparse response keeps input, explicit null overrides. Existing Router descriptors/MRO/list/dict/int/bool/default/header/tenant aliases are reused. Wire and whole receipt remain physical response only.
  Tests: `network/v2/extensions/layer3/routers/python_cloud_create_test.go:TestPythonCloudRouterCreateTranslatedBodySeedCanonicalAndSourceStatus`, `network/cloud_router_create_test.go:TestCloudRouterCreateResponseSourceDefaultsAliasesAndOwnedLocation`, `network/cloud_router_create_test.go:TestCloudRouterCreateRejectedStatusesAndInvalidResponsesKeepActualReceipt`.
- Source100..399 acceptance opts in only new leaf; existing native/shared typed baseline is unchanged. Selected microversion and owned null/current location replace response members only in Resource; current foreign project metadata is computed without extra API requests. Response mutation cannot alias independent Wire evidence.
  Tests: `network/v2/extensions/layer3/routers/python_cloud_create_test.go:TestPythonCloudRouterCreateTranslatedBodySeedCanonicalAndSourceStatus`, `network/cloud_router_create_test.go:TestCloudRouterCreateResponseSourceDefaultsAliasesAndOwnedLocation`, `network/cloud_router_create_test.go:TestCloudRouterCreateCleanRetryTokenRenewalAndHeaderVersionCapture`.
- Actual rejected400/403/404/409/503 retain native POST error/status/body/header. Invalid response shapes error with actual accepted receipt; nonJSON/empty response retains Source seed without fabricated Wire. Nil/invalid cloud attrs/context/cancel/selected API/source fail before HTTP.
  Tests: `network/cloud_router_create_test.go:TestCloudRouterCreateRejectedStatusesAndInvalidResponsesKeepActualReceipt`, `network/v2/extensions/layer3/routers/python_cloud_create_test.go:TestPythonCloudRouterCreatePreflightAndResponseShapeErrors`, `network/v2/extensions/layer3/routers/python_cloud_create_test.go:TestPythonCloudRouterCreateNativeRejectionsAndStickyRestoredSourceProof`, `network/cloud_router_create_test.go:TestCloudRouterCreatePreflightCancellationAndSelectedSourceGuards`.
- Physical accepted/rejected200/401/404/503 read/Close faults, custom cancel and source change preserve actual receipts/all causes with zero retry/reauth hooks. Native401 value ErrOriginal and ErrReauth remain available. Restored source cannot erase observed mismatch; clean503 retry and401 token renewal preserve owned route/body/headers. Changed route aborts before resend with original503 receipt.
  Tests: `network/cloud_router_create_test.go:TestCloudRouterCreatePhysicalReadCloseCancellationSourceAndNative401NoHooks`, `network/cloud_router_create_test.go:TestCloudRouterCreateAZPhysicalFailureHasExtensionProofAndNoPostReceipt`, `network/v2/extensions/layer3/routers/python_cloud_create_test.go:TestPythonCloudRouterCreateNativeRejectionsAndStickyRestoredSourceProof`, `network/cloud_router_create_test.go:TestCloudRouterCreateCleanRetryTokenRenewalAndHeaderVersionCapture`.
- Connection supplies cached Neutron and an owned default current location without caller builders; explicit location remains selectable and invalid/cancelled calls do not discover a service.
  Tests: `connectiontest/connection_record_service_cache_test.go:TestConnectionRouterCreateOwnsDefaultLocationAndCachedService`, `connectiontest/connection_record_service_cache_test.go:TestConnectionCycle7PreflightPreservesCancellationBeforeDiscovery`.

Python/Go differences:

- Concrete Go RouterCreateResult exposes owned Router/Wire/Envelope/Header/StatusCode and independently observed Extensions, unlike only mutable Python Router. Cloud7kwargs are complete UTF8 JSON; application builders are not required. Failed extension GET cannot fabricate POST evidence.
- Source float serialization/runtime domain, custom Python runtime objects, mutable Resource/class/adapter/session/API cache/current_location and dynamic class dispatch remain shared SDK-R1/SDK-C1/SDK-S1 foundation differences. Their supported increment remains0; this one cloud mapping does not promote general Proxy raw attrs create, update/delete or full SDK parity.
- Go preserves selected source/API/header/version snapshots and context cancellation. JSON object wire order is not guaranteed. Safe literal service-relative URI/protocol policy remains shared; router POST has no caller ID path segment to normalize. Invalid values fail preflight rather than silently inventing IDs/defaults.
- Standalone location defaults null; explicit owned CurrentLocation can be injected from Connection.CurrentLocation without HTTP, and never enters POST. Existing Network/Subnet role cache and mutation notifications are preserved; source-wide URL cache invalidation is not claimed closed.
- Neutron27 stock create_router and gateway/network_id use project ADMIN_OR_PROJECT_MEMBER; enable_snat/external_fixed_ips field specification requires ADMIN, including explicit false. Deprecated historic rules differ. SDK/test/runtime evidence does not assert real deployment authorization, role, ownership, driver or server acceptance.
- Actual imported pinned cloud create_router/_build_external_gateway_info + Neutron Proxy._create + Resource.new/create/translate/descriptors ran on Python3.14.8 for90cases. Only Adapter.post and extension iterator were fixture substitutions. stock27 router policy SHA2569c7436013df00879c69d0821f6fd4b127186efa7dc3774b6a0e2bdfa3d29aefa matched official tagged primary source. This is not live OpenStack acceptance.

### `python:cloud/_network/NetworkCloudMixin/create_subnet`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.CreateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk.Connection.CreateSubnetRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.CreateSubnetRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.CreateCloudRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithCloudCreateOrder`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.FindSubnetSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.FindSubnetSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateCIDR`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateIPVersion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateDHCP`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateName`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateTenantID`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateGatewayIP`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateDisableGateway`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateAllocationPools`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateDNSNameservers`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateHostRoutes`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateIPv6RAMode`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateIPv6AddressMode`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreatePrefixLength`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateDefaultSubnetPool`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetCreateSubnetPool`

- subnets.API.CreateWithAttributes sends only explicit attributes, so ip_version, enable_dhcp and a null gateway_ip appear only when the caller sets them.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudNetworkCreateRouterSubnetSendOnlyExplicitAttributes`.
- Library-owned concrete options accept Source formal keywords and arbitrary extension kwargs, snapshot JSON/maps/ordered aliases/location/client before callbacks, own defaults ip_version=4 and enable_dhcp=False, and require no application builder/interface. Positional network/receiver duplicates, nil options and malformed complete UTF8 JSON fail before HTTP. JSON/individual options preserve input order; map-only order is deterministic lexical.
  Tests: `network/cloud_subnet_create_test.go:TestCloudSubnetCreateDefaultsOwnedKeywordsRawBodyAndIndependentEvidence`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreatePreflightBoundArgumentsCancellationAndImmutableService`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateOwnedLocationUsesActualProjectsWithoutForgedWire`.
- Default network lookup occurs before gateway/CIDR/pool/IP semantic validation. A tenant_id that is not None/null remains a raw member query including false/zero/empty string. New fixed Source helpers opt into valid100..399, Source root-links precedence, loaded JSON policy and physical-fault callback suppression without altering baseline FindRecord/native/list bindings. Only clean400/403/404 fall back to all-page exact ID/name find; duplicate/late-page errors never POST or accept an early match.
  Tests: `network/cloud_subnet_create_test.go:TestCloudSubnetCreateNetworkLookupPrecedesAllSemanticValidation`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateFalseyFormalsTenantPresenceAndGatewayNull`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateDefaultLookupsConsumeWholePagesAndKeepPartialFailure`.
- Gateway truthy conflict is checked before CIDR/pool requirements and mutually-exclusive default/named pool selection. Named pool lookup precedes string ip_version int conversion; missing pool masks later invalid IP as in Source. Unicode decimal digits/signs/spaces/underscores use the library PythonIntegerString, while nonstring None/bool/list/float retains raw dirty JSON. Required network/pool data is taken from the returned Source getter rather than caller label.
  Tests: `network/cloud_subnet_create_test.go:TestCloudSubnetCreateSubnetPoolLookupThenPythonStringIPConversion`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateNetworkLookupPrecedesAllSemanticValidation`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateFalseyFormalsTenantPresenceAndGatewayNull`.
- Body starts from network_id/ip_version/DHCP defaults, applies arbitrary kwargs, then truthy formal optional fields in Source order. Falsey formal fields omit; disable gateway forces explicit null; default pool forces true and named pool uses resolved ID. Raw bool/int/list dirty values remain uncoerced in request; only getter descriptors cast. Ordered canonical/wire aliases and response aliases use the final Source value, with partial/empty responses retaining seed only in Resource while Wire/Envelope stay actual.
  Tests: `network/cloud_subnet_create_test.go:TestCloudSubnetCreateOrderedAliasesConflictRestorationAndRawDescriptorValues`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateResponseCanonicalAliasesDoNotLoseActualHeaderOrWire`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateDefaultsOwnedKeywordsRawBodyAndIndependentEvidence`, `network/v2/subnets/python_cloud_create_test.go:TestPythonCloudSubnetCreateOwnedTranslatedBodySeedAndSourceStatuses`.
- Proxy base_path is consumed before one-level truthy __conflicting_attrs restoration; falsey conflicting attrs remain ordinary unknown Body, and restored base_path remains Body while the fixed class getter is /subnets. Source constructor collisions are rejected after network lookup; microversion constructor does not override selected create version. Header aliases select the ordered winner before formatting/validation, including shadowed invalid lists, null=>None and string-list comma-space. Actual HTTP If-Match overrides request header seed; absent response header retains seed.
  Tests: `network/cloud_subnet_create_test.go:TestCloudSubnetCreateSourceProxyControlsHeadersAndOriginalBasePath`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateResponseCanonicalAliasesDoNotLoseActualHeaderOrWire`, `network/v2/subnets/python_cloud_create_test.go:TestPythonCloudSubnetCreateOnlyWinningHeaderAliasIsValidated`, `network/v2/subnets/python_cloud_create_test.go:TestPythonCloudSubnetCreateHeaderSeedAndActualResponsePrecedence`, `network/v2/subnets/python_cloud_create_test.go:TestPythonCloudSubnetCreateBasePathAndRestoredUnknownClassGetter`.
- Lookup observations and subnet POST expose independently owned Resource/Wire/Envelope/Header/Status. Actual or foreign project IDs compute SDK-owned location without overwriting actual Wire location/microversion. Public Connection supplies current scope; standalone nil location is null. Malformed remote/body/header projection or late selected-source/cancellation errors retain actual accepted receipt as ResponseError and return no successful Resource, never a fabricated server acknowledgement.
  Tests: `network/cloud_subnet_create_test.go:TestCloudSubnetCreateOwnedLocationUsesActualProjectsWithoutForgedWire`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateResponseCanonicalAliasesDoNotLoseActualHeaderOrWire`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreatePhysicalFaultsKeepActualLookupOrMutationAndStopHooks`, `network/v2/subnets/python_cloud_create_test.go:TestPythonCloudSubnetCreateHeaderSeedAndActualResponsePrecedence`.
- Actual GET/POST200/401/404/503 read+Close/cancel/source failures preserve response bytes/header/status and original causes with zero retry/reauth callbacks. Pointer native401 retains ErrOriginal value ErrUnexpectedResponseCode and ErrReauth despite absent Unwrap. Clean503 retry and401 token renewal remain supported; route changes stop before resend with original rejection receipt. Selected headers remain captured when callback mutates the original valid header map.
  Tests: `network/cloud_subnet_create_test.go:TestCloudSubnetCreatePhysicalFaultsKeepActualLookupOrMutationAndStopHooks`, `network/cloud_subnet_create_test.go:TestCloudSubnetCreateCleanRetryRenewalSource199AndFixedHeaderCapture`, `network/v2/subnets/python_cloud_create_test.go:TestPythonCloudSubnetCreateConstructorGetterFailuresAndNativeRejectedReceipts`.
- An actual accepted subnet POST100..399 invalidates the original private successful network role/subnet-gateway topology cache even when response decode/Read/Close/cancel/source handling subsequently fails. Rejected POST and pre-POST lookup/validation errors do not invalidate success cache. Replacing public Service.Roles does not redirect the private mutation cache target.
  Tests: `network/cloud_subnet_create_test.go:TestCloudSubnetCreateAcceptedMutationInvalidatesOnlyPrivateSuccessfulRoleCache`.
- Connection supplies cached Neutron and an owned default location for lookup/create while the service invalidates only actual accepted topology mutations.
  Tests: `connectiontest/connection_record_service_cache_test.go:TestConnectionSubnetCreateSharesNetworkCacheAndLocation`, `connectiontest/connection_record_service_cache_test.go:TestConnectionCycle7PreflightPreservesCancellationBeforeDiscovery`.

Python/Go differences:

- This maps one fixed cloud create_subnet declaration with library-owned orchestration and returned owned Resource/Wire/actual receipt plus independent Network/SubnetPool observations. It does not promote native Create/CreateWithAttributes, old general FindRecord, arbitrary raw Proxy.Find/constructor/class/adapter/session/cache behavior or any additional Source declaration. Full SDK supported delta remains 0; shared SDK-R1/SDK-C1/SDK-S1 foundation remaining persists separately.
- Go network/named pool references are validated literal string IDs or exact names. Truthy nonstring/mutable Resource reference branches remain the shared Resource/class foundation; falsey pool values still skip lookup as in the fixed cloud flow. Go escapes slash/percent as a single route segment and rejects blank/control/traversal references. base_path accepts only literal service-relative routes, including empty endpoint-root route; Source arbitrary percent-template/query/external path formatting is not reproduced.
- Go exposes concrete With options, safe immutable snapshots, captured service/header/protocol guards, independent actual transport receipts and accepted-mutation topology cache invalidation instead of Python mutable Resource/Connection side effects. CurrentLocation is SDK-owned configured context; unknown raw location remains request/Wire data while the getter is null or computed scope. General dynamic class/member dispatch and arbitrary computed properties are not claimed by unknown JSON extension Body preservation.
- Ordered JSON and individual keyword options preserve Source insertion order; map-only remaining keys use Go lexical order, and JSON wire key order is not guaranteed. Raw JSON numeric tokens retain precision instead of first converting to float64; Python configurable int-string digit limit and generic JSON float/runtime-domain serialization remain explicit Go/foundation differences. No new semantic CIDR/IP/range/driver validation beyond the Source cloud rules is claimed.
- Stock Neutron27 create_subnet is project ADMIN_OR_NET_OWNER_MEMBER, with independent admin segment_id/service_types and separate tags policy. Readable/shared network or library lookup success does not prove write authorization, other-tenant/pool access or server/driver acceptance. Live authenticated OpenStack policy is not part of fixture/runtime/main/race proof.
- Actual imported pinned cloud get_network/get_subnetpool/create_subnet plus Neutron Proxy.find/create and Resource.find/new/create ran under CPython3.14.8 for 127 cases. Only HTTP adapter methods were substituted. This is not live OpenStack acceptance.

### `python:cloud/_network/NetworkCloudMixin/get_qos_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindQuery`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.GetQoSPolicyRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindListBasePath`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Every cloud call completes fresh uncached qos extension discovery before options/member/deletion and preserves completed extension evidence. Unavailable extension has the shared sentinel and no policy HTTP. Context/source/selected private API guards remain active through dependent leaf requests and caller option snapshots.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsSourceStatusPolicyAndGateFailures`, `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyGetFalseyFiltersRawControlsAndSourceSnapshot`.
- New Source action profile clones class Body/Query and opts Source100..399, link precedence, CPython-loaded equality and physical-stop policies in without changing prior typed API baselines. Actual accepted/rejected response receipts and native401 pointer ErrOriginal/value ErrUnexpectedResponseCode plus ErrReauth causes survive read/Close/cancel/source failures with zero retry/reauth and no missing/success fallback.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsPhysicalFaultsNeverFallbackMissingOrRetry`, `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsSourceStatusPolicyAndGateFailures`.
- Clean503 retry and401 live token renewal are retained for Get/DELETE while changed selected route before retry preserves the original503 receipt. Source status199 opt-in is separately exercised over the public testhelper HTTP fixture with a response observer; no resource/response is fabricated.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsCleanRetryRenewalAndRouteGuard`, `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsSourceStatusPolicyAndGateFailures`.
- Cloud Get default missing is a nil Policy and nil error. Falsey JSON filters become {} after the gate; truthy nonmapping filters fail as kwargs rather than being interpreted from docstring as JMES. Ordinary raw member params seed constructor fields and actual body overrides them, keeping explicit null/defaults/unknown values/private Wire/Header/Envelope evidence independent.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyGetDefaultMissingAndMemberEvidence`, `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyGetFalseyFiltersRawControlsAndSourceSnapshot`.
- Clean400/403/404 member failures alone trigger complete exact ID/name list fallback with declared name hint and server/local Body classification. Raw microversion is GET-only; list_base_path is fallback-only and an unused foreign fallback route is ignored on successful member GET. CPython-loaded local equality and late duplicate ambiguity are handled by the owned library.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyGetFallbackListPathVersionLocalFiltersAndLateAmbiguity`.
- After a complete fresh gate, cloud-bound name_or_id/ignore_missing kwargs collide by presence including null, while the six raw fetch controls are now consumed by the dedicated FindSourceRecord lane; ordinary legacy FindRecord retains its reviewed classifier.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyGetFalseyFiltersRawControlsAndSourceSnapshot`.
- Each six-control cloud lookup fully repeats extension discovery before a collection/member policy GET. Policy descriptor/tenant alias/rules/defaults, unknown constructor controls and computed Resource.location=null are preserved in the Resource view; actual server location JSON remains in physical Wire/Envelope/status/header evidence.
  Tests: `network/cloud_qos_policy_fetch_controls_test.go:TestCloudQoSPolicyFetchControlsFreshFullGateAndPolicyProfile`.
- Cloud filters raw headers are member GET mapping-key query values and fallback HTTP headers, with independent raw base_path/list_base_path and GET-only microversion. Invalid truthy headers fail only at fallback, after completed fresh gate evidence.
  Tests: `network/cloud_qos_policy_fetch_controls_test.go:TestCloudQoSPolicyFetchControlsRawHeadersHaveSeparatePhases`, `network/cloud_qos_policy_fetch_controls_test.go:TestCloudQoSPolicyFetchControlsInvalidHeadersWaitForFallback`.
- Copied cloud filter/header inputs, falsey response-key fallback and fresh uncached repeated reads retain independent results. Custom HTTP messages preserve native rejected status/body/header causes or clean404 missing semantics. Physical accepted/rejected response faults never replay, fall back or issue deletion.
  Tests: `network/cloud_qos_policy_fetch_controls_test.go:TestCloudQoSPolicyFetchControlsOwnedOptionsAndFalseyEnvelope`, `network/cloud_qos_policy_fetch_controls_test.go:TestCloudQoSPolicyFetchControlsCustomErrorKeepsNativeProof`, `network/cloud_qos_policy_fetch_controls_test.go:TestCloudQoSPolicyFetchControlsPhysicalFaultNeverReplaysOrDeletes`.

Python/Go differences:

- The concrete Go carrier preserves completed Extensions, the owned PolicyRecord and independent deletion receipt rather than returning only Python Resource/None/bool. Failed operations retain observed evidence and contextual causes; successful bodyless deletion does not invent Resource/Wire.
- Go opts source100..399/loaded equality/link/fault contracts into a new private action lane while preserving legacy typed Find/List/Delete. Options and records are independently owned JSON snapshots; Python mutable Resource/cache/custom class/adapter/current_location state remains the shared SDK-R1/SDK-C1/SDK-S1 foundation boundary, not a named operation support count.
- Literal IDs/continuations remain in the selected service origin/path/query and are escaped as one member segment. Source URL joining/URI percent formatting can rewrite arbitrary paths; concrete Go source/header/protocol guard and strict UTF8/JSON policies remain explicit reviewed differences.
- For actual returned null ID, Source raises InvalidRequest and Go rejects before DELETE. Source urljoin maps nonnull false/0/underflow/empty ID to a collection route; Go rejects such IDs as a member-route policy instead of issuing collection DELETE, returning false or fabricating success. Nonempty numeric/true JSON IDs use the shared PythonString conversion and preserve large integer precision.
- Stock Neutron27 get_policy permits owner project reader or shared policy (and admin); delete_policy requires ADMIN_OR_PROJECT_MANAGER in project scope with historic deprecated ADMIN_ONLY. Mock/source/race/main validation does not assert authenticated backend success or elevated permissions.
- Cloud filters remain kwargs expanded after fresh full extension discovery and are not a local Cloud _filter_list/JMES predicate. Only the dedicated FindSourceRecord opts into six fetch controls, leaving ordinary QoS FindRecord unchanged. The existing source profile computes Policy Resource.location as null and does not synthesize Connection facts; actual server location JSON remains only in physical Wire/Envelope evidence.
- The fetch-control lane uses finite UTF-8 JSON/string identities and service-local safe routes with only %(id)s and %% URI formatting and string-valued fallback headers. skip_cache is consumed without an invented cache header; each lookup invokes the stateless reader, without claiming bypass of caller custom transport caches. error_message decorates Go HTTP causes and leaves transport/accepted decode/final missing errors distinct. SDK-R1/C1/S1 remain whole-SDK goals.

### `python:cloud/_network/NetworkCloudMixin/create_qos_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.CreateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.CreateQoSPolicyRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.CreateCloudRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyMutationJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyName`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyDescription`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyShared`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyDefault`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyProjectID`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Cloud valid_kwargs name/description/shared/default/project_id are validated before any extension HTTP. Complete JSON null/presence and concrete options are library-owned, nil/unknown cloud keys including is_shared/is_default are rejected with HTTP0 and later options overwrite prior keys from owned snapshots.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationValidKwargsBeforeGateAndOwnedOptions`.
- Every valid call consumes a fresh full qos extension discovery; every nonnull default including false/0/[]/{} triggers an independent fresh full qos-default gate. Supported default maps raw to is_default; unavailable default is removed without hiding gate errors, with completed Extensions evidence and DefaultDropped separately retained.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyCreateDefaultNonnullFreshGatesAndEmptyPOST`, `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyUpdateEmptyAfterDefaultSkipsLookupAndMissingRaises`.
- New cloud Source mutation lane uses selected original service/API/source and header snapshots captured before option callbacks. Source100..399 completion preserves status199. Clean503 retry and401 token renewal remain possible; changed route aborts renewal/retry with original rejected evidence. Prior typed/native profiles are unchanged.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationSourceStatusPolicyCleanRetryAndImmutableHeaders`, `network/v2/extensions/qos/policies/python_cloud_mutations_test.go:TestPolicyCloudMutationSelectedVersionAndEqualUpdateReset`.
- POST/PUT rejected400/403/404/409/503 retain actual status/body/header and native value ErrUnexpectedResponseCode rather than fabricating Resource/Wire or swallowing404. Accepted/rejected200/401/404/503 read+Close/cancel/source fault matrix retains actual physical evidence and causes with zero retry/reauth callbacks. Native401 retains pointer ErrUnableToReauthenticate.ErrOriginal (value native response proof) and ErrReauth despite native absence of Unwrap.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationRejectedStatusPreservesActualEvidence`, `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationPhysicalFaultsPreserveActualEvidenceWithoutHooks`.
- Create always POSTs the translated cloud attrs even empty {}, without requiring name or sending synthesized Body defaults. Explicit null and raw default/is_default values remain raw in the request; actual partial response updates the Source seed while Wire remains actual response only.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyCreateDefaultNonnullFreshGatesAndEmptyPOST`, `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationValidKwargsBeforeGateAndOwnedOptions`.
- Actual unknown response microversion cannot shadow selected Source create version: Resource view exposes selected version while Wire keeps the original actual member. Cloud helper uses library-owned descriptor/default materialization and preserves selected header/version protocol.
  Tests: `network/v2/extensions/qos/policies/python_cloud_mutations_test.go:TestPolicyCloudMutationSelectedVersionAndEqualUpdateReset`.

Python/Go differences:

- Concrete Go result preserves Policy, Lookup, Extensions, NoOp and DefaultDropped rather than Python Resource/None only. Wire/Envelope/Header/StatusCode remain actual observed response evidence; a prepared seed/no-op is not represented as a server PUT response.
- The new Source mutation lane preserves legacy typed/native defaults and policies. Complete UTF8 JSON/concrete owned options cover the named kwargs domain; custom runtime objects and mutable Python Resource/class/adapter/session/cache/current_location remain SDK-R1/SDK-C1/SDK-S1 shared foundation boundaries, with supported increment 0 and no generic full-SDK claim.
- Go source-relative URI guards escape returned member identity as one segment. Null/false/0/underflow/empty dirty member ID is rejected before PUT; Python null raises InvalidRequest but falsey nonnull URL joining can target a collection path. Go does not issue collection PUT from such a member seed. Equal updates do not validate or require a nonnull ID before returning no-op.
- JSON object wire order is not guaranteed. Source constructor conversion reads raw row insertion order, which is retained for declared/wire alias ledger conversion; dirty payload contains only changed keys and Python-loaded numeric equality preserves bool/int equality and CPython floating-point behavior without converting arbitrary integers through float64.
- Stock Neutron27 create_policy/update_policy are project ADMIN_OR_PROJECT_MANAGER, distinct from deprecated ADMIN_ONLY. Project_id field/cross-project privileges remain independent server validation (fixed neutron-lib3.21.0 reference); no mock/race/Source/main result proves authenticated live deployment/backend/driver acceptance.

### `python:cloud/_network/NetworkCloudMixin/delete_qos_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.DeleteWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.DeleteQoSPolicyRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.DeleteSourceRecord`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Every cloud call completes fresh uncached qos extension discovery before options/member/deletion and preserves completed extension evidence. Unavailable extension has the shared sentinel and no policy HTTP. Context/source/selected private API guards remain active through dependent leaf requests and caller option snapshots.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsSourceStatusPolicyAndGateFailures`, `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyGetFalseyFiltersRawControlsAndSourceSnapshot`.
- New Source action profile clones class Body/Query and opts Source100..399, link precedence, CPython-loaded equality and physical-stop policies in without changing prior typed API baselines. Actual accepted/rejected response receipts and native401 pointer ErrOriginal/value ErrUnexpectedResponseCode plus ErrReauth causes survive read/Close/cancel/source failures with zero retry/reauth and no missing/success fallback.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsPhysicalFaultsNeverFallbackMissingOrRetry`, `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsSourceStatusPolicyAndGateFailures`.
- Clean503 retry and401 live token renewal are retained for Get/DELETE while changed selected route before retry preserves the original503 receipt. Source status199 opt-in is separately exercised over the public testhelper HTTP fixture with a response observer; no resource/response is fabricated.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsCleanRetryRenewalAndRouteGuard`, `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyActionsSourceStatusPolicyAndGateFailures`.
- After one fresh qos gate, default Source Find ignore_missing=True runs once. Missing lookup returns Deleted=false without DELETE. A found lookup uses returned Resource.id, sends one bodyless literal member DELETE and returns true after success or clean404 according to Proxy default ignore_missing=True. Lookup and deletion keep separate actual receipts, including clean404 status/body/header/IgnoredMissing.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyDeleteMissingFoundAndIgnored404HaveDifferentResults`.
- Deletion consumes the returned JSON ID without a second lookup or caller-name substitution; large integer and true IDs use PythonString without float64 loss. Empty/null/false/zero/underflow IDs fail before member deletion under the explicit Go path policy. Header/body edits in the deletion carrier cannot alter the lookup receipt.
  Tests: `network/cloud_qos_policy_actions_test.go:TestCloudQoSPolicyDeleteReturnedJSONIDAndIndependentActualReceipts`.

Python/Go differences:

- The concrete Go carrier preserves completed Extensions, the owned PolicyRecord and independent deletion receipt rather than returning only Python Resource/None/bool. Failed operations retain observed evidence and contextual causes; successful bodyless deletion does not invent Resource/Wire.
- Go opts source100..399/loaded equality/link/fault contracts into a new private action lane while preserving legacy typed Find/List/Delete. Options and records are independently owned JSON snapshots; Python mutable Resource/cache/custom class/adapter/current_location state remains the shared SDK-R1/SDK-C1/SDK-S1 foundation boundary, not a named operation support count.
- Literal IDs/continuations remain in the selected service origin/path/query and are escaped as one member segment. Source URL joining/URI percent formatting can rewrite arbitrary paths; concrete Go source/header/protocol guard and strict UTF8/JSON policies remain explicit reviewed differences.
- For actual returned null ID, Source raises InvalidRequest and Go rejects before DELETE. Source urljoin maps nonnull false/0/underflow/empty ID to a collection route; Go rejects such IDs as a member-route policy instead of issuing collection DELETE, returning false or fabricating success. Nonempty numeric/true JSON IDs use the shared PythonString conversion and preserve large integer precision.
- Stock Neutron27 get_policy permits owner project reader or shared policy (and admin); delete_policy requires ADMIN_OR_PROJECT_MANAGER in project scope with historic deprecated ADMIN_ONLY. Mock/source/race/main validation does not assert authenticated backend success or elevated permissions.

### `python:cloud/_network/NetworkCloudMixin/get_qos_rule_type_details`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.GetQoSRuleTypeDetails`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.API.GetRecord`

- Two ordered fresh extension listings (qos, then qos-rule-type-details) precede GET qos/rule-types/{name}; every extension row is retained with the Resource/Wire rule type record.
  Tests: `api/r2_cloud_net_firewall_test.go:TestR2CloudNetQoSRuleTypeDetailsGatesThenGet`.
- A missing qos-rule-type-details alias returns UnavailableExtensionError before any rule type request; a 404 and an empty name are errors.
  Tests: `api/r2_cloud_net_firewall_test.go:TestR2CloudNetQoSRuleTypeDetailsMissingExtensionAndErrors`.

Python/Go differences:

- The Python filters argument is ignored by the pinned source and has no Go parameter.
- Neutron restricts the details read to administrators by default; Go reports the server denial as the HTTP error and performs no client-side role check.
- Typed Go options (ruletypes record get options) replace Resource get kwargs; the result is an SDK record, not a Resource.

### `python:cloud/_network/NetworkCloudMixin/get_qos_bandwidth_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetBandwidthLimitRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.GetQoSBandwidthLimitRule`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitMicroversion`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Literal rule getter gates and resolves policy once, returns raw untyped record/default location on success, and preserves native404 without rule Find fallback or ignored missing.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthGetResolvesPolicyOnceAndKeeps404`.
- Extension gate consumes every alias including later unhashable alias or failed page after an early QoS match; absence has typed unavailable extension error and no policy/callback/rule request.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthGateConsumesAllExtensionsAndStopsBeforePolicy`.
- Policy missing and ambiguity prevent rules/callback/filter expansion, preserve real extension inventory and typed missing/ambiguity without inventing policy None HTTP evidence.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthPolicyMissingAndAmbiguityRetainActualGate`.
- CPython falsey filters, truthy nonobject pre-rule errors, initialized owned callback maps, joined option error/cancellation or original captured-source mutation and real prerequisite receipts.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthFilterOwnershipCallbackGuardAndFalseyCases`.
- Actual truncated401 rule response keeps native reauthentication pointer/UnexpectedResponseCode/body/header/EOF, stops willing retry/reauth callbacks; late failed page preserves only earlier actual Inventory and completed parent/extensions.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthPhysical401RulePageAndLateFailurePreserveStageProof`.

Python/Go differences:

- Concrete JSON/context/captured-client Go APIs and protected route/header policy replace dynamic Python session/Resource class/constructor/callable substitutions; arbitrary Python-only objects are not accepted.
- SourceProxy lane accepts100..399 with Source loaded local equality and computed location:null. Existing typed list/Get record status200..399 and typed raw base_path rejection remain compatibility APIs; concrete typed BasePath is additive.
- JMESPath literal object key iteration uses Go lexical order vs Python insertion order; existing strict syntax/exact JSON numeric evaluator/custom Resource function type policies remain explicit differences.
- Literal and nested generator values never acquire fake BandwidthLimitRecord or HTTP receipts. Actual extension/policy/rule proofs and observed partial inventory are retained with Go errors.
- Standalone Service computed location is Source null, including ignoring raw location in logical view. Full Connection computed location and mutable Resource/session model are tracked shared SDK-C1/R1 scope.
- Source policy and extension/user authorization are checked by the real Neutron deployment; these are read-only named helpers and no creation/update/delete or server-side policy emulation is added.

### `python:cloud/_network/NetworkCloudMixin/create_qos_bandwidth_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateBandwidthLimitRule`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.CreateQoSBandwidthLimitRuleRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateCloudBandwidthLimitRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateDirection`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateMaxBurstKBPS`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Required max_kbps/dscp_mark/min_kbps is positional JSON-domain data and retained by presence: zero/null/false/string/large integer are not omitted or coerced. Concrete optional kwargs/options are owned snapshots; invalid/duplicate required or bound policy/unknown keys fail before extension HTTP, with no application builder/interface responsibility.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateValidKwargsOwnedOptionsAndRequiredRawPresence`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateRequiredPresencePreflightAndNativeRejections`.
- Each call fully consumes a fresh uncached qos extension discovery before policy/rule stages. Initial alias discovery does not hide late malformed alias or later-page HTTP failures. Completed extension/policy evidence remains separate from any actual rule mutation receipt; failed gate causes no policy/rule HTTP.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCompleteGatesAndOptionalFailuresKeepStages`.
- Source-only policy lookup opts existing FindSourceRecord into Source status/equality/link/fault policy without modifying legacy state.Policy or native APIs. Exact member GET follows clean400/403/404 all-page list fallback, raises missing/late ambiguity before POST, and uses the returned policy ID rather than caller identity.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateFindMissingAmbiguityAndListFallback`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`.
- Translated cloud attrs alone enter collection POST. Source Body seed/defaults merge partial response with explicit-null override; missing response required fields remain only in Resource seed while Wire/Envelope retain actual response. Bound URI parent does not enter request body; foreign response qos_policy_id remains actual Wire while Resource reads the original raw JSON URI parent. Null/false/zero URI formatting uses None/False/0 and arbitrary integer precision survives.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateReturnedParentJSONURIAndActualViews`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateSourceSeedDefaultsAndWireReceipts`.
- Source selected create microversion and owned null location override same-named response members only in Resource view; actual Wire is preserved. Source100..399 handles lookup/POST199 separately from legacy baseline. Concrete input/protocol/API/source/header snapshots are captured before caller callbacks and guard every dependent stage.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateReturnedParentJSONURIAndActualViews`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePreflightCancellationAndSelectedAPIGuards`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateSourceSeedDefaultsAndWireReceipts`.
- Actual rejected400/403/404/409/503 POST preserves native value ErrUnexpectedResponseCode and actual status/body/header rather than ignored missing/success. Accepted/rejected200/401/404/503 read+Close/cancel/source faults retain actual receipt/causes with zero retry/reauth hooks and explicit bounded unexpected hook fences. Native401 retains pointer ErrUnableToReauthenticate.ErrOriginal value response proof and ErrReauth even without native Unwrap.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePhysicalFailuresPreserveActualProofWithoutHooks`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateRequiredPresencePreflightAndNativeRejections`.
- Clean503 retry and401 live token renewal remain supported. Changed selected route aborts before a resend with the original503 receipt, while callback MoreHeaders mutation does not affect the captured initial/gate/policy/POST headers. Fault/cancel/source failures are not normalized to missing or successful creation.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePreflightCancellationAndSelectedAPIGuards`.
- BW accepts optional max_burst_kbps/direction only and resolves policy with ignore_missing=True before the direction gate. A nonnull direction (false/zero/empty string/array/object included) triggers a second fresh complete qos-bw-limit-direction discovery. Unavailable extension removes only direction and marks DirectionDropped; null retains direction:null with no extra gate, and HTTP/physical gate failure is never treated as unsupported direction.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateBandwidthDirectionNonnullGateAfterPolicy`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCompleteGatesAndOptionalFailuresKeepStages`.

Python/Go differences:

- The concrete Go carrier exposes Rule/Wire/Envelope/Header/StatusCode, independent Policy/Extensions and DirectionDropped instead of only Python rule Resource. Policy is the independently owned actual lookup view/receipt, not Python mutable borrowed object side effects. Request seed/no-op/class state is not fabricated as server response evidence.
- Required values accept complete UTF8 JSON/Go primitives with library-owned concrete With options; custom Python runtime objects, mutable Resource/class/adapter/cache/session/current_location and dynamic class attribute dispatch remain shared SDK-R1/SDK-C1/SDK-S1 foundation differences. Their supported increment remains 0. These three cloud mappings do not promote general Proxy.create raw attrs, update/delete/details or full SDK parity.
- Go guards selected source/header/protocol and service-relative literal parent segments: slash/percent are escaped as one segment; blank/control/traversal identities fail preflight. Source URI %s formatting can introduce arbitrary path structure. Null/false/0 parents are explicitly preserved as None/False/0 paths with original raw URI view rather than incorrectly applying falsey policy member DELETE joining.
- JSON object wire key order is not guaranteed. Raw request values are forwarded without adding native constructor requirements or type/default coercions; actual server value/range/driver validation may reject null/false/string/rates even though the pinned Source forwards them. Generic JSON float serialization/runtime-domain differences remain explicit shared foundation policy.
- Fixed Neutron27 child create policies require project ADMIN_OR_PARENT_OWNER_MANAGER, distinct from historic deprecated ADMIN_ONLY. Ability to read a shared policy does not confer child creation permission. Source/runtime/mock/race/main checks do not assert authenticated backend, role, driver or deployment policy acceptance.
- Actual pinned Source cloud decorator/methods + Neutron Proxy.find/create + Resource.find/new/create were imported from ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe and executed under Python3.14.8 in a bounded62-case reference. Only Adapter HTTP methods and extension generator were substituted. An additional 45-case comparison covers response, URI, and version behavior. This is not live OpenStack acceptance.

### `python:cloud/_network/NetworkCloudMixin/delete_qos_bandwidth_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteBandwidthLimitRule`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.DeleteQoSBandwidthLimitRule`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteBandwidthLimitRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteMicroversion`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Each cloud kind checks all QoS extension evidence, resolves the parent once and sends one bodyless literal rule DELETE; valid Source accepted ACK returns Completed/Deleted true, only clean rule404 returns Completed=true/Deleted=false, other400/403/408/409/500 preserve error/actual receipt.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteSourceTrueFalseAndOtherStatuses`.
- Absent extension, policy missing/ambiguity and minimum strict parent failure are errors without DELETE or false completion; actual Extensions and parent proof are retained without fabricating missing policy response.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteGateAndPolicyErrorsNeverBecomeFalse`.
- Early QoS match does not stop extension iteration; a later failed extension page keeps observed Extensions and original rejection, with no policy/DELETE/false result.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteConsumesEntireExtensionGateBeforeMutation`.
- Owned safe header/version options and initialized callback maps apply only to strict DELETE; blank/control/dot IDs fail after prerequisites with no DELETE receipt or missing bool.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteOwnedHeadersVersionAndSafeIDs`.
- Before-DELETE callback failure joins cancellation or original selected-client mutation while actual extension/policy evidence survives; already-cancelled call sends no HTTP.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteCallbackSourceAndCancellationFailuresKeepParents`.
- Actual truncated404/401 DELETE bytes/header/status/UnexpectedEOF retain physical evidence, zero willing retry/reauth callbacks and cannot become false; real parent/extension evidence survives.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteFaulty404And401KeepRealReceiptNotFalse`.
- A native hook expanding accepted404 or returning a callback404 after actual503 leaves operation incomplete and keeps actual DELETE status; neither can establish rule absence.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteExpanded404AndCallback404NeverCompleteFalse`.
- New Source-only deletion profile opts into199/204/302/399 ACK acceptance while preserving bodyless fixed family route and opaque actual bytes/header/status with no Resource/Wire.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteSourceOpaqueACKAndStatusOptIn`.
- Direct leaf defaults ignore_missing true only for clean404; explicit false remains strict. Native400/403/404/409/429/500 returns owned actual DELETE receipt and operation context without decode.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteDefaultMissingAndStrictRejectionsKeepActualReceipt`.
- Source delete IDs use safe UTF8 literal path segments including slash/space/literal percent2F; blank/control/dot parent/rule and nil option fail before HTTP with no invented receipt.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteSafeLiteralIdentityAndPreflightNoReceipt`.
- Concrete header/version option and callback results own snapshots, nil maps are initialized; simultaneous option error/cancellation or captured client mutation preserves all causes before HTTP.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteOwnedHeadersVersionCallbacksAndGuardCauses`.
- Accepted204 and rejected404/401/500 physical read/close/mixed faults keep actual receipt/native401.ErrOriginal/value/status/causes, close once and stop willing callbacks; default missing policy cannot suppress physical404.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeletePhysicalFaultsCannotBecomeMissing`.
- Accepted-code404 and callback404 after real503 remain terminal, preserve real receipt and do not set IgnoredMissing even under default direct-leaf missing policy.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteAccepted404AndCallback404NeverEstablishMissing`.
- Clean503 retry and401 renewal keep one logical bodyless DELETE route/conditional-header/selected-version policy; renewed tokens work and actual final ACK proof is retained.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteCleanRetryAndAuthenticationKeepFixedBodylessPolicy`.

Python/Go differences:

- Public Go returns an owned partial QoSRuleDeleteResult with Completed/Deleted plus actual Extensions/Policy/Receipt instead of Python bool or exception only. Deleted=false is Source missing only when Completed=true and err=nil.
- New receipt-only Source leaf methods default ignore_missing true like Proxy; each cloud helper always requests strict false, then uses shared resource.IgnoreMissing terminal policy only on clean child404. Error paths never fabricate a policy None or DELETE Resource/Wire.
- Bandwidth/DSCP Source policy lookup ignore_missing=True plus explicit None NotFound differs from minimum-bandwidth Source ignore_missing=False. SDK reuses shared Policy vs PolicyStrict and preserves that difference rather than making all parents default tolerant.
- Source mutation status100..399 opt-in applies to these new profiles through root Engine.Mutate e.codes; old false/generated/typed policies remain unchanged. Opaque ACK content is not forced into JSON.
- Source dynamic Resource/class/session/adapter/custom Python IDs and mutable state remain SDK-R1/C1/S1 foundations. Go accepts safe concrete UTF8 strings/options/context, rejects blank/control/dot IDs and dangerous route/header changes; whitespace-only Source identifiers can reach HTTP while Go rejects them explicitly. No empty ID silently becomes missing false.
- Safe Header/Microversion options and callback ownership are Go transport conveniences; original cloud helper has no such kwargs. Library owns prerequisite/gate/strict child/missing behavior without requiring caller builders/interfaces.
- Fixed Neutron27 project-scoped delete_policy_bandwidth_limit_rule/delete_policy_dscp_marking_rule/delete_policy_minimum_bandwidth_rule use ADMIN_OR_PARENT_OWNER_MANAGER; manager must own parent policy, admin alternative, deprecated ADMIN_ONLY distinguished. Shared visibility/read does not imply delete rights, and deployment policy overrides/QoS driver/real deletion were not tested.

### `python:cloud/_network/NetworkCloudMixin/get_qos_dscp_marking_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetDSCPMarkingRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.GetQoSDSCPMarkingRule`
- `github.com/JSYoo5B/go-openstacksdk/network.WithDSCPMarkingCloudOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithDSCPMarkingCloudCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetSourceDSCPMarkingRecord`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Every call consumes the full fresh qos extension inventory before GET-first exact policy Find; early qos match does not suppress later alias/page faults. Missing or ambiguous policy stops rule I/O with actual already observed gate/policy evidence, no fabricated policy response.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingGateConsumesAllExtensionsAndStopsBeforePolicy`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPolicyMissingAndAmbiguityRetainActualGate`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingFreshCompleteGatePolicyNameLookupAndSourceLinkPrecedence`.
- Selected original network service/client/source is captured before owned option callbacks; cancellation causes and source mutation remain terminal. Nil context/service fails before discovery or invented response. Location and raw JSON options are cloned and callbacks run once.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingFilterOwnershipCallbackGuardAndFalseyCases`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPreflightAndInvalidLocationHaveHonestEvidence`.
- Actual HTTP physical rule faults and late pages retain partial Inventory plus native response bytes/header/status in error, with no successful Value/Rules. Real Content-Length faults at 200/203/401/404/429/503 preserve pointer401 ErrOriginal VALUE response and ErrReauth EOF; retry/backoff/reauth fault hooks remain zero.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPhysical401RulePageAndLateFailurePreserveStageProof`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPhysicalLatePagesKeepActualHeaderStatusAndNoFaultHooks`.
- Named member getter resolves policy with ignore_missing=True for DSCP marking. Then one literal escaped member GET uses Source status/projection, no rule Find/list fallback or suppressed rule404. Nonstring policy identity fails with actual policy response proof.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingGetResolvesPolicyOnceAndKeeps404`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingGetterPolicyStrictnessAndMalformedPolicyReceipt`.
- Rule Resource default/unknown Body, raw Wire, envelope/header/status and URI parent remain distinct. Standalone location null and owned connected current_location are applied before filtering; undeclared project_id/availability_zone cannot change the configured project/zone. Selected Rules, Values, Inventory, LastRule and JSON Value do not alias exported rule metadata/bytes.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingListOwnsGateParentPagingAndRawFields`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPreflightAndInvalidLocationHaveHonestEvidence`.

Python/Go differences:

- Go returns an owned result carrier with real Extensions/Policy/Inventory/LastRule evidence, correlated Rules/Rule and independent Values/Value; literal/opaque output never fabricates response Resource/Wire/Header/status. Failed page receipt remains in the actual error rather than a fictional failed resource.
- Source kwargs are restricted to JSON-domain data, policy/rule identities to safe nonblank Go strings, HTTP headers to string-valued mappings, microversion/base_path to nullable strings and truthy max_items to nonnegative Go integers or true. Go validates these typed/ownership boundaries and service-relative URI safety before some Python dynamic errors; unsafe origin/query/fragment/traversal and non-string numeric %-route formatting are explicit errors.
- Dictionary-key iterable results and key-based JMESPath functions use reviewed Go lexical order instead of Python insertion order. JMES deprecation warnings are omitted. Opaque generator repr/to_string involving Python process addresses and non-JSON dynamic object/session/mutable Resource behavior remain SDK-R1/C1/S1 scope; no equivalent process address string is fabricated.
- Explicit CurrentLocation is an owned configured snapshot; the rule classes have no project_id/availability_zone descriptors, so unknown remote fields cannot override it. Root Connection wrappers may supply this library-owned option automatically. Existing paging bounds/cycle checks/HTTP Link repair and fixed request guards remain documented Go improvements. Authenticated live Neutron acceptance is separate from testhelper HTTP/Source AST proof.

### `python:cloud/_network/NetworkCloudMixin/create_qos_dscp_marking_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateDSCPMarkingRule`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.CreateQoSDSCPMarkingRuleRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateCloudDSCPMarkingRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateJSON`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Required max_kbps/dscp_mark/min_kbps is positional JSON-domain data and retained by presence: zero/null/false/string/large integer are not omitted or coerced. Concrete optional kwargs/options are owned snapshots; invalid/duplicate required or bound policy/unknown keys fail before extension HTTP, with no application builder/interface responsibility.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateValidKwargsOwnedOptionsAndRequiredRawPresence`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateRequiredPresencePreflightAndNativeRejections`.
- Each call fully consumes a fresh uncached qos extension discovery before policy/rule stages. Initial alias discovery does not hide late malformed alias or later-page HTTP failures. Completed extension/policy evidence remains separate from any actual rule mutation receipt; failed gate causes no policy/rule HTTP.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCompleteGatesAndOptionalFailuresKeepStages`.
- Source-only policy lookup opts existing FindSourceRecord into Source status/equality/link/fault policy without modifying legacy state.Policy or native APIs. Exact member GET follows clean400/403/404 all-page list fallback, raises missing/late ambiguity before POST, and uses the returned policy ID rather than caller identity.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateFindMissingAmbiguityAndListFallback`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`.
- Translated cloud attrs alone enter collection POST. Source Body seed/defaults merge partial response with explicit-null override; missing response required fields remain only in Resource seed while Wire/Envelope retain actual response. Bound URI parent does not enter request body; foreign response qos_policy_id remains actual Wire while Resource reads the original raw JSON URI parent. Null/false/zero URI formatting uses None/False/0 and arbitrary integer precision survives.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateReturnedParentJSONURIAndActualViews`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateSourceSeedDefaultsAndWireReceipts`.
- Source selected create microversion and owned null location override same-named response members only in Resource view; actual Wire is preserved. Source100..399 handles lookup/POST199 separately from legacy baseline. Concrete input/protocol/API/source/header snapshots are captured before caller callbacks and guard every dependent stage.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateReturnedParentJSONURIAndActualViews`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePreflightCancellationAndSelectedAPIGuards`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateSourceSeedDefaultsAndWireReceipts`.
- Actual rejected400/403/404/409/503 POST preserves native value ErrUnexpectedResponseCode and actual status/body/header rather than ignored missing/success. Accepted/rejected200/401/404/503 read+Close/cancel/source faults retain actual receipt/causes with zero retry/reauth hooks and explicit bounded unexpected hook fences. Native401 retains pointer ErrUnableToReauthenticate.ErrOriginal value response proof and ErrReauth even without native Unwrap.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePhysicalFailuresPreserveActualProofWithoutHooks`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateRequiredPresencePreflightAndNativeRejections`.
- Clean503 retry and401 live token renewal remain supported. Changed selected route aborts before a resend with the original503 receipt, while callback MoreHeaders mutation does not affect the captured initial/gate/policy/POST headers. Fault/cancel/source failures are not normalized to missing or successful creation.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePreflightCancellationAndSelectedAPIGuards`.
- DSCP cloud method has no optional kwargs in pinned Python; concrete Go variadic options provide consistent SDK call syntax but any actual optional field is rejected before HTTP. Policy lookup defaults ignore_missing=True and absence becomes a cloud NotFound error before rule POST. No direction gate/default attrs are introduced.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateValidKwargsOwnedOptionsAndRequiredRawPresence`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateFindMissingAmbiguityAndListFallback`.

Python/Go differences:

- The concrete Go carrier exposes Rule/Wire/Envelope/Header/StatusCode, independent Policy/Extensions and DirectionDropped instead of only Python rule Resource. Policy is the independently owned actual lookup view/receipt, not Python mutable borrowed object side effects. Request seed/no-op/class state is not fabricated as server response evidence.
- Required values accept complete UTF8 JSON/Go primitives with library-owned concrete With options; custom Python runtime objects, mutable Resource/class/adapter/cache/session/current_location and dynamic class attribute dispatch remain shared SDK-R1/SDK-C1/SDK-S1 foundation differences. Their supported increment remains 0. These three cloud mappings do not promote general Proxy.create raw attrs, update/delete/details or full SDK parity.
- Go guards selected source/header/protocol and service-relative literal parent segments: slash/percent are escaped as one segment; blank/control/traversal identities fail preflight. Source URI %s formatting can introduce arbitrary path structure. Null/false/0 parents are explicitly preserved as None/False/0 paths with original raw URI view rather than incorrectly applying falsey policy member DELETE joining.
- JSON object wire key order is not guaranteed. Raw request values are forwarded without adding native constructor requirements or type/default coercions; actual server value/range/driver validation may reject null/false/string/rates even though the pinned Source forwards them. Generic JSON float serialization/runtime-domain differences remain explicit shared foundation policy.
- Fixed Neutron27 child create policies require project ADMIN_OR_PARENT_OWNER_MANAGER, distinct from historic deprecated ADMIN_ONLY. Ability to read a shared policy does not confer child creation permission. Source/runtime/mock/race/main checks do not assert authenticated backend, role, driver or deployment policy acceptance.
- Actual pinned Source cloud decorator/methods + Neutron Proxy.find/create + Resource.find/new/create were imported from ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe and executed under Python3.14.8 in a bounded62-case reference. Only Adapter HTTP methods and extension generator were substituted. An additional 45-case comparison covers response, URI, and version behavior. This is not live OpenStack acceptance.

### `python:cloud/_network/NetworkCloudMixin/delete_qos_dscp_marking_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteDSCPMarkingRule`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.DeleteQoSDSCPMarkingRule`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteDSCPMarkingRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteMicroversion`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Each cloud kind checks all QoS extension evidence, resolves the parent once and sends one bodyless literal rule DELETE; valid Source accepted ACK returns Completed/Deleted true, only clean rule404 returns Completed=true/Deleted=false, other400/403/408/409/500 preserve error/actual receipt.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteSourceTrueFalseAndOtherStatuses`.
- Absent extension, policy missing/ambiguity and minimum strict parent failure are errors without DELETE or false completion; actual Extensions and parent proof are retained without fabricating missing policy response.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteGateAndPolicyErrorsNeverBecomeFalse`.
- Early QoS match does not stop extension iteration; a later failed extension page keeps observed Extensions and original rejection, with no policy/DELETE/false result.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteConsumesEntireExtensionGateBeforeMutation`.
- Owned safe header/version options and initialized callback maps apply only to strict DELETE; blank/control/dot IDs fail after prerequisites with no DELETE receipt or missing bool.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteOwnedHeadersVersionAndSafeIDs`.
- Before-DELETE callback failure joins cancellation or original selected-client mutation while actual extension/policy evidence survives; already-cancelled call sends no HTTP.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteCallbackSourceAndCancellationFailuresKeepParents`.
- Actual truncated404/401 DELETE bytes/header/status/UnexpectedEOF retain physical evidence, zero willing retry/reauth callbacks and cannot become false; real parent/extension evidence survives.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteFaulty404And401KeepRealReceiptNotFalse`.
- A native hook expanding accepted404 or returning a callback404 after actual503 leaves operation incomplete and keeps actual DELETE status; neither can establish rule absence.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteExpanded404AndCallback404NeverCompleteFalse`.
- New Source-only deletion profile opts into199/204/302/399 ACK acceptance while preserving bodyless fixed family route and opaque actual bytes/header/status with no Resource/Wire.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteSourceOpaqueACKAndStatusOptIn`.
- Direct leaf defaults ignore_missing true only for clean404; explicit false remains strict. Native400/403/404/409/429/500 returns owned actual DELETE receipt and operation context without decode.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteDefaultMissingAndStrictRejectionsKeepActualReceipt`.
- Source delete IDs use safe UTF8 literal path segments including slash/space/literal percent2F; blank/control/dot parent/rule and nil option fail before HTTP with no invented receipt.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteSafeLiteralIdentityAndPreflightNoReceipt`.
- Concrete header/version option and callback results own snapshots, nil maps are initialized; simultaneous option error/cancellation or captured client mutation preserves all causes before HTTP.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteOwnedHeadersVersionCallbacksAndGuardCauses`.
- Accepted204 and rejected404/401/500 physical read/close/mixed faults keep actual receipt/native401.ErrOriginal/value/status/causes, close once and stop willing callbacks; default missing policy cannot suppress physical404.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeletePhysicalFaultsCannotBecomeMissing`.
- Accepted-code404 and callback404 after real503 remain terminal, preserve real receipt and do not set IgnoredMissing even under default direct-leaf missing policy.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteAccepted404AndCallback404NeverEstablishMissing`.
- Clean503 retry and401 renewal keep one logical bodyless DELETE route/conditional-header/selected-version policy; renewed tokens work and actual final ACK proof is retained.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteCleanRetryAndAuthenticationKeepFixedBodylessPolicy`.

Python/Go differences:

- Public Go returns an owned partial QoSRuleDeleteResult with Completed/Deleted plus actual Extensions/Policy/Receipt instead of Python bool or exception only. Deleted=false is Source missing only when Completed=true and err=nil.
- New receipt-only Source leaf methods default ignore_missing true like Proxy; each cloud helper always requests strict false, then uses shared resource.IgnoreMissing terminal policy only on clean child404. Error paths never fabricate a policy None or DELETE Resource/Wire.
- Bandwidth/DSCP Source policy lookup ignore_missing=True plus explicit None NotFound differs from minimum-bandwidth Source ignore_missing=False. SDK reuses shared Policy vs PolicyStrict and preserves that difference rather than making all parents default tolerant.
- Source mutation status100..399 opt-in applies to these new profiles through root Engine.Mutate e.codes; old false/generated/typed policies remain unchanged. Opaque ACK content is not forced into JSON.
- Source dynamic Resource/class/session/adapter/custom Python IDs and mutable state remain SDK-R1/C1/S1 foundations. Go accepts safe concrete UTF8 strings/options/context, rejects blank/control/dot IDs and dangerous route/header changes; whitespace-only Source identifiers can reach HTTP while Go rejects them explicitly. No empty ID silently becomes missing false.
- Safe Header/Microversion options and callback ownership are Go transport conveniences; original cloud helper has no such kwargs. Library owns prerequisite/gate/strict child/missing behavior without requiring caller builders/interfaces.
- Fixed Neutron27 project-scoped delete_policy_bandwidth_limit_rule/delete_policy_dscp_marking_rule/delete_policy_minimum_bandwidth_rule use ADMIN_OR_PARENT_OWNER_MANAGER; manager must own parent policy, admin alternative, deprecated ADMIN_ONLY distinguished. Shared visibility/read does not imply delete rights, and deployment policy overrides/QoS driver/real deletion were not tested.

### `python:cloud/_network/NetworkCloudMixin/get_qos_minimum_bandwidth_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetMinimumBandwidthRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.GetQoSMinimumBandwidthRule`
- `github.com/JSYoo5B/go-openstacksdk/network.WithMinimumBandwidthCloudOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithMinimumBandwidthCloudCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetSourceMinimumBandwidthRecord`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Every call consumes the full fresh qos extension inventory before GET-first exact policy Find; early qos match does not suppress later alias/page faults. Missing or ambiguous policy stops rule I/O with actual already observed gate/policy evidence, no fabricated policy response.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthGateConsumesAllExtensionsAndStopsBeforePolicy`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPolicyMissingAndAmbiguityRetainActualGate`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFreshCompleteGatePolicyNameLookupAndSourceLinkPrecedence`.
- Selected original network service/client/source is captured before owned option callbacks; cancellation causes and source mutation remain terminal. Nil context/service fails before discovery or invented response. Location and raw JSON options are cloned and callbacks run once.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFilterOwnershipCallbackGuardAndFalseyCases`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPreflightAndInvalidLocationHaveHonestEvidence`.
- Actual HTTP physical rule faults and late pages retain partial Inventory plus native response bytes/header/status in error, with no successful Value/Rules. Real Content-Length faults at 200/203/401/404/429/503 preserve pointer401 ErrOriginal VALUE response and ErrReauth EOF; retry/backoff/reauth fault hooks remain zero.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPhysical401RulePageAndLateFailurePreserveStageProof`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPhysicalLatePagesKeepActualHeaderStatusAndNoFaultHooks`.
- Named member getter resolves policy with ignore_missing=False for minimum bandwidth. Then one literal escaped member GET uses Source status/projection, no rule Find/list fallback or suppressed rule404. Nonstring policy identity fails with actual policy response proof.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthGetResolvesPolicyOnceAndKeeps404`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthGetterPolicyStrictnessAndMalformedPolicyReceipt`.
- Rule Resource default/unknown Body, raw Wire, envelope/header/status and URI parent remain distinct. Standalone location null and owned connected current_location are applied before filtering; undeclared project_id/availability_zone cannot change the configured project/zone. Selected Rules, Values, Inventory, LastRule and JSON Value do not alias exported rule metadata/bytes.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthListOwnsGateParentPagingAndRawFields`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPreflightAndInvalidLocationHaveHonestEvidence`.

Python/Go differences:

- Go returns an owned result carrier with real Extensions/Policy/Inventory/LastRule evidence, correlated Rules/Rule and independent Values/Value; literal/opaque output never fabricates response Resource/Wire/Header/status. Failed page receipt remains in the actual error rather than a fictional failed resource.
- Source kwargs are restricted to JSON-domain data, policy/rule identities to safe nonblank Go strings, HTTP headers to string-valued mappings, microversion/base_path to nullable strings and truthy max_items to nonnegative Go integers or true. Go validates these typed/ownership boundaries and service-relative URI safety before some Python dynamic errors; unsafe origin/query/fragment/traversal and non-string numeric %-route formatting are explicit errors.
- Dictionary-key iterable results and key-based JMESPath functions use reviewed Go lexical order instead of Python insertion order. JMES deprecation warnings are omitted. Opaque generator repr/to_string involving Python process addresses and non-JSON dynamic object/session/mutable Resource behavior remain SDK-R1/C1/S1 scope; no equivalent process address string is fabricated.
- Explicit CurrentLocation is an owned configured snapshot; the rule classes have no project_id/availability_zone descriptors, so unknown remote fields cannot override it. Root Connection wrappers may supply this library-owned option automatically. Existing paging bounds/cycle checks/HTTP Link repair and fixed request guards remain documented Go improvements. Authenticated live Neutron acceptance is separate from testhelper HTTP/Source AST proof.

### `python:cloud/_network/NetworkCloudMixin/create_qos_minimum_bandwidth_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateMinimumBandwidthRule`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.CreateQoSMinimumBandwidthRuleRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.CreateCloudMinimumBandwidthRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleCreateDirection`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Required max_kbps/dscp_mark/min_kbps is positional JSON-domain data and retained by presence: zero/null/false/string/large integer are not omitted or coerced. Concrete optional kwargs/options are owned snapshots; invalid/duplicate required or bound policy/unknown keys fail before extension HTTP, with no application builder/interface responsibility.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateValidKwargsOwnedOptionsAndRequiredRawPresence`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateRequiredPresencePreflightAndNativeRejections`.
- Each call fully consumes a fresh uncached qos extension discovery before policy/rule stages. Initial alias discovery does not hide late malformed alias or later-page HTTP failures. Completed extension/policy evidence remains separate from any actual rule mutation receipt; failed gate causes no policy/rule HTTP.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCompleteGatesAndOptionalFailuresKeepStages`.
- Source-only policy lookup opts existing FindSourceRecord into Source status/equality/link/fault policy without modifying legacy state.Policy or native APIs. Exact member GET follows clean400/403/404 all-page list fallback, raises missing/late ambiguity before POST, and uses the returned policy ID rather than caller identity.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateFindMissingAmbiguityAndListFallback`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`.
- Translated cloud attrs alone enter collection POST. Source Body seed/defaults merge partial response with explicit-null override; missing response required fields remain only in Resource seed while Wire/Envelope retain actual response. Bound URI parent does not enter request body; foreign response qos_policy_id remains actual Wire while Resource reads the original raw JSON URI parent. Null/false/zero URI formatting uses None/False/0 and arbitrary integer precision survives.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateReturnedParentJSONURIAndActualViews`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateSourceSeedDefaultsAndWireReceipts`.
- Source selected create microversion and owned null location override same-named response members only in Resource view; actual Wire is preserved. Source100..399 handles lookup/POST199 separately from legacy baseline. Concrete input/protocol/API/source/header snapshots are captured before caller callbacks and guard every dependent stage.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateReturnedParentJSONURIAndActualViews`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePreflightCancellationAndSelectedAPIGuards`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateSourceSeedDefaultsAndWireReceipts`.
- Actual rejected400/403/404/409/503 POST preserves native value ErrUnexpectedResponseCode and actual status/body/header rather than ignored missing/success. Accepted/rejected200/401/404/503 read+Close/cancel/source faults retain actual receipt/causes with zero retry/reauth hooks and explicit bounded unexpected hook fences. Native401 retains pointer ErrUnableToReauthenticate.ErrOriginal value response proof and ErrReauth even without native Unwrap.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePhysicalFailuresPreserveActualProofWithoutHooks`, `network/v2/extensions/qos/rules/python_cloud_creates_test.go:TestQoSCloudRuleCreateRequiredPresencePreflightAndNativeRejections`.
- Clean503 retry and401 live token renewal remain supported. Changed selected route aborts before a resend with the original503 receipt, while callback MoreHeaders mutation does not affect the captured initial/gate/policy/POST headers. Fault/cancel/source failures are not normalized to missing or successful creation.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateCleanRetryRenewalSource199AndHeaderCapture`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreatePreflightCancellationAndSelectedAPIGuards`.
- Minimum accepts optional direction only and resolves policy with ignore_missing=False. Missing policy raises strict find error before POST. Unlike BW, direction never triggers qos-bw-limit-direction discovery or gets dropped; raw required min_kbps and raw direction are forwarded as supplied.
  Tests: `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateBandwidthDirectionNonnullGateAfterPolicy`, `network/cloud_qos_rule_creates_test.go:TestCloudQoSRuleCreateFindMissingAmbiguityAndListFallback`.

Python/Go differences:

- The concrete Go carrier exposes Rule/Wire/Envelope/Header/StatusCode, independent Policy/Extensions and DirectionDropped instead of only Python rule Resource. Policy is the independently owned actual lookup view/receipt, not Python mutable borrowed object side effects. Request seed/no-op/class state is not fabricated as server response evidence.
- Required values accept complete UTF8 JSON/Go primitives with library-owned concrete With options; custom Python runtime objects, mutable Resource/class/adapter/cache/session/current_location and dynamic class attribute dispatch remain shared SDK-R1/SDK-C1/SDK-S1 foundation differences. Their supported increment remains 0. These three cloud mappings do not promote general Proxy.create raw attrs, update/delete/details or full SDK parity.
- Go guards selected source/header/protocol and service-relative literal parent segments: slash/percent are escaped as one segment; blank/control/traversal identities fail preflight. Source URI %s formatting can introduce arbitrary path structure. Null/false/0 parents are explicitly preserved as None/False/0 paths with original raw URI view rather than incorrectly applying falsey policy member DELETE joining.
- JSON object wire key order is not guaranteed. Raw request values are forwarded without adding native constructor requirements or type/default coercions; actual server value/range/driver validation may reject null/false/string/rates even though the pinned Source forwards them. Generic JSON float serialization/runtime-domain differences remain explicit shared foundation policy.
- Fixed Neutron27 child create policies require project ADMIN_OR_PARENT_OWNER_MANAGER, distinct from historic deprecated ADMIN_ONLY. Ability to read a shared policy does not confer child creation permission. Source/runtime/mock/race/main checks do not assert authenticated backend, role, driver or deployment policy acceptance.
- Actual pinned Source cloud decorator/methods + Neutron Proxy.find/create + Resource.find/new/create were imported from ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe and executed under Python3.14.8 in a bounded62-case reference. Only Adapter HTTP methods and extension generator were substituted. An additional 45-case comparison covers response, URI, and version behavior. This is not live OpenStack acceptance.

### `python:cloud/_network/NetworkCloudMixin/delete_qos_minimum_bandwidth_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteMinimumBandwidthRule`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.DeleteQoSMinimumBandwidthRule`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleDeleteMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.DeleteMinimumBandwidthRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithQoSRuleDeleteMicroversion`

- After the cloud extension gate, Go sends the same GET /extensions, QoS policy GET-then-name-list find, rule GET/POST/DELETE, rule-type GET, policy POST and policy DELETE requests, while a rule DELETE 404 stays an error.
  Tests: `network/python_cloud_parity_a_test.go:TestPythonCloudQoSRequestsAfterExtensionGate`.
- Each cloud kind checks all QoS extension evidence, resolves the parent once and sends one bodyless literal rule DELETE; valid Source accepted ACK returns Completed/Deleted true, only clean rule404 returns Completed=true/Deleted=false, other400/403/408/409/500 preserve error/actual receipt.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteSourceTrueFalseAndOtherStatuses`.
- Absent extension, policy missing/ambiguity and minimum strict parent failure are errors without DELETE or false completion; actual Extensions and parent proof are retained without fabricating missing policy response.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteGateAndPolicyErrorsNeverBecomeFalse`.
- Early QoS match does not stop extension iteration; a later failed extension page keeps observed Extensions and original rejection, with no policy/DELETE/false result.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteConsumesEntireExtensionGateBeforeMutation`.
- Owned safe header/version options and initialized callback maps apply only to strict DELETE; blank/control/dot IDs fail after prerequisites with no DELETE receipt or missing bool.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteOwnedHeadersVersionAndSafeIDs`.
- Before-DELETE callback failure joins cancellation or original selected-client mutation while actual extension/policy evidence survives; already-cancelled call sends no HTTP.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteCallbackSourceAndCancellationFailuresKeepParents`.
- Actual truncated404/401 DELETE bytes/header/status/UnexpectedEOF retain physical evidence, zero willing retry/reauth callbacks and cannot become false; real parent/extension evidence survives.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteFaulty404And401KeepRealReceiptNotFalse`.
- A native hook expanding accepted404 or returning a callback404 after actual503 leaves operation incomplete and keeps actual DELETE status; neither can establish rule absence.
  Tests: `network/cloud_qos_rule_deletes_test.go:TestCloudQoSRuleDeleteExpanded404AndCallback404NeverCompleteFalse`.
- New Source-only deletion profile opts into199/204/302/399 ACK acceptance while preserving bodyless fixed family route and opaque actual bytes/header/status with no Resource/Wire.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteSourceOpaqueACKAndStatusOptIn`.
- Direct leaf defaults ignore_missing true only for clean404; explicit false remains strict. Native400/403/404/409/429/500 returns owned actual DELETE receipt and operation context without decode.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteDefaultMissingAndStrictRejectionsKeepActualReceipt`.
- Source delete IDs use safe UTF8 literal path segments including slash/space/literal percent2F; blank/control/dot parent/rule and nil option fail before HTTP with no invented receipt.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteSafeLiteralIdentityAndPreflightNoReceipt`.
- Concrete header/version option and callback results own snapshots, nil maps are initialized; simultaneous option error/cancellation or captured client mutation preserves all causes before HTTP.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteOwnedHeadersVersionCallbacksAndGuardCauses`.
- Accepted204 and rejected404/401/500 physical read/close/mixed faults keep actual receipt/native401.ErrOriginal/value/status/causes, close once and stop willing callbacks; default missing policy cannot suppress physical404.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeletePhysicalFaultsCannotBecomeMissing`.
- Accepted-code404 and callback404 after real503 remain terminal, preserve real receipt and do not set IgnoredMissing even under default direct-leaf missing policy.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteAccepted404AndCallback404NeverEstablishMissing`.
- Clean503 retry and401 renewal keep one logical bodyless DELETE route/conditional-header/selected-version policy; renewed tokens work and actual final ACK proof is retained.
  Tests: `network/v2/extensions/qos/rules/python_cloud_deletes_test.go:TestQoSRuleDeleteCleanRetryAndAuthenticationKeepFixedBodylessPolicy`.

Python/Go differences:

- Public Go returns an owned partial QoSRuleDeleteResult with Completed/Deleted plus actual Extensions/Policy/Receipt instead of Python bool or exception only. Deleted=false is Source missing only when Completed=true and err=nil.
- New receipt-only Source leaf methods default ignore_missing true like Proxy; each cloud helper always requests strict false, then uses shared resource.IgnoreMissing terminal policy only on clean child404. Error paths never fabricate a policy None or DELETE Resource/Wire.
- Bandwidth/DSCP Source policy lookup ignore_missing=True plus explicit None NotFound differs from minimum-bandwidth Source ignore_missing=False. SDK reuses shared Policy vs PolicyStrict and preserves that difference rather than making all parents default tolerant.
- Source mutation status100..399 opt-in applies to these new profiles through root Engine.Mutate e.codes; old false/generated/typed policies remain unchanged. Opaque ACK content is not forced into JSON.
- Source dynamic Resource/class/session/adapter/custom Python IDs and mutable state remain SDK-R1/C1/S1 foundations. Go accepts safe concrete UTF8 strings/options/context, rejects blank/control/dot IDs and dangerous route/header changes; whitespace-only Source identifiers can reach HTTP while Go rejects them explicitly. No empty ID silently becomes missing false.
- Safe Header/Microversion options and callback ownership are Go transport conveniences; original cloud helper has no such kwargs. Library owns prerequisite/gate/strict child/missing behavior without requiring caller builders/interfaces.
- Fixed Neutron27 project-scoped delete_policy_bandwidth_limit_rule/delete_policy_dscp_marking_rule/delete_policy_minimum_bandwidth_rule use ADMIN_OR_PARENT_OWNER_MANAGER; manager must own parent policy, admin alternative, deprecated ADMIN_ONLY distinguished. Shared visibility/read does not imply delete rights, and deployment policy overrides/QoS driver/real deletion were not tested.

## References

- [`openstack/cloud/_network.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py)
- [`get_network`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L351)
- [`get_network_by_id`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L377)
- [`get_router`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L386)
- [`get_subnet`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L413)
- [`get_port`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L444)
- [`get_port_by_id`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L470)
- [`get_network_extensions`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L734)
- [`add_router_interface`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2151)
- [`create_port`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2690)
- [`delete_port`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2838)
- [`delete_router`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2379)
- [`delete_subnet`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2569)
- [`create_firewall_policy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L984)
- [`create_firewall_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1269)
- [`create_firewall_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L775)
- [`get_firewall_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L846)
- [`get_firewall_policy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1051)
- [`get_firewall_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1331)
- [`delete_firewall_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L804)
- [Cloud Firewall Deletes Python](cloud-firewall-deletes-python.md)
- [`delete_firewall_policy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1010)
- [`delete_firewall_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1299)
- [`create_router`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2250)
- [Cloud Router Create Python](cloud-router-create-python.md)
- [`create_subnet`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2401)
- [Cloud Subnet Create Python](cloud-subnet-create-python.md)
- [`get_qos_policy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L221)
- [Cloud QoS Policy Actions Python](cloud-qos-policy-actions-python.md)
- [`create_qos_policy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1435)
- [Cloud QoS Policy Mutations Python](cloud-qos-policy-mutations-python.md)
- [`delete_qos_policy`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1512)
- [`get_qos_rule_type_details`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L300)
- [`get_qos_bandwidth_limit_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1594)
- [Cloud QoS Bandwidth Python](cloud-qos-bandwidth-python.md)
- [`create_qos_bandwidth_limit_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1621)
- [Cloud QoS Rule Creates Python](cloud-qos-rule-creates-python.md)
- [`delete_qos_bandwidth_limit_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1723)
- [Cloud QoS Rule Deletes Python](cloud-qos-rule-deletes-python.md)
- [`get_qos_dscp_marking_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1819)
- [Cloud QoS Dscp Python](cloud-qos-dscp-python.md)
- [`create_qos_dscp_marking_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1844)
- [`delete_qos_dscp_marking_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1919)
- [`get_qos_minimum_bandwidth_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2018)
- [Cloud QoS Minimum Bandwidth Python](cloud-qos-minimum-bandwidth-python.md)
- [`create_qos_minimum_bandwidth_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2041)
- [`delete_qos_minimum_bandwidth_rule`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2115)
- [`python_cloud_parity_a_test.go`](python_cloud_parity_a_test.go)
- [Cloud QoS Policies Python](cloud-qos-policies-python.md)
- [rule types](cloud-qos-rule-types-python.md)
- [bandwidth](cloud-qos-bandwidth-python.md)
- [DSCP](cloud-qos-dscp-python.md)
- [minimum bandwidth](cloud-qos-minimum-bandwidth-python.md)
