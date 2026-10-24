# Python Cloud Parity B

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:cloud/_network/NetworkCloudMixin/get_subnet_by_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.GetWithFields`

- get_subnet_by_id sends one GET /v2.0/subnets/{id}, decodes the subnet fields and keeps a 404 as an error like Python NotFoundException.
  Tests: `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBGetSubnetByID`.

Python/Go differences:

- Python returns a mutable Subnet Resource; Go returns the native Gophercloud Subnet model, so unknown response fields are not kept.
- The Python docstring promises None when missing, but the code raises NotFoundException; Go returns the 404 error.

### `python:cloud/_network/NetworkCloudMixin/get_subnetpool`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/subnetpools.API.FindIdentity`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIdentityFindIgnoreMissing`

- get_subnetpool sends GET /v2.0/subnetpools/{name_or_id}, falls back to GET /v2.0/subnetpools?name= after a 404 and returns nil without error when nothing matches, as find_subnet_pool(ignore_missing=True) does.
  Tests: `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBGetSubnetpool`.

Python/Go differences:

- Go returns the native SubnetPool model, which requires the three prefix length fields; a projected response without them fails instead of yielding a partial Resource.
- Absence is nil, nil in Go instead of Python None; FindIdentity ignores missing by default so no option is needed.

### `python:cloud/_network/NetworkCloudMixin/list_routers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilters`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilter`

- list_routers sends GET /v2.0/routers with the filters as server query parameters (none by default) and follows routers_links to collect every router.
  Tests: `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBListAndSearchRouters`.

Python/Go differences:

- Python returns an empty list when has_service('network') is false; Go fails when the network service client cannot be built.
- Python query booleans are encoded by Requests as True/False while Go semantic filters encode lowercase true/false.
- Go returns native Router models instead of mutable Router Resources.

### `python:cloud/_network/NetworkCloudMixin/search_routers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilter`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilters`

- search_routers sends name_or_id only as the server name query merged with the filters and follows every page, which Go reproduces with WithFilter("name", x) and WithFilters.
  Tests: `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBListAndSearchRouters`.

Python/Go differences:

- Like Python, a router ID passed as name_or_id is only sent as a name query; Go does not add a local ID match.
- Python query booleans are encoded by Requests as True/False while Go semantic filters encode lowercase true/false.
- Go returns native Router models instead of mutable Router Resources.

### `python:cloud/_network_common/NetworkCommonCloudMixin/create_security_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.CreateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithAttribute`

- create_security_group posts security_group with name and description, adding stateful and tenant_id only when supplied, and returns the decoded group with its rules.
  Tests: `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBCreateSecurityGroup`.

Python/Go differences:

- Only the default secgroup_source=neutron path with a network service is mapped; the Nova /os-security-groups branch with its normalization and the OpenStackCloudUnavailableFeature raised when secgroup_source is unset are not reproduced.
- Python maps project_id to the tenant_id body key itself; Go callers pass WithAttribute("tenant_id", id) and WithAttribute("stateful", v) only when those arguments are given.
- Go returns the native SecGroup model instead of a SecurityGroup Resource.

### `python:cloud/_network_common/NetworkCommonCloudMixin/get_security_group_by_id`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.GetWithFields`

- get_security_group_by_id sends GET /v2.0/security-groups/{id}, decodes the group and keeps a 404 as an error.
  Tests: `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBSecurityGroupReads`.

Python/Go differences:

- Only the default secgroup_source=neutron path with a network service is mapped; the Nova /os-security-groups branch with its normalization and the OpenStackCloudUnavailableFeature raised when secgroup_source is unset are not reproduced.
- Go returns the native SecGroup model instead of a SecurityGroup Resource.

### `python:cloud/_network_common/NetworkCommonCloudMixin/list_security_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithFilters`

- list_security_groups sends GET /v2.0/security-groups with the filters as server query parameters and follows security_groups_links to collect every group.
  Tests: `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBSecurityGroupReads`.

Python/Go differences:

- Only the default secgroup_source=neutron path with a network service is mapped; the Nova /os-security-groups branch with its normalization and the OpenStackCloudUnavailableFeature raised when secgroup_source is unset are not reproduced.
- Python query booleans are encoded by Requests as True/False while Go semantic filters encode lowercase true/false.
- Go returns native SecGroup models instead of SecurityGroup Resources.

### `python:cloud/_network_common/NetworkCommonCloudMixin/delete_security_group_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.API.DeleteWithOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/rules.WithDeleteIgnoreMissing`

- delete_security_group_rule sends DELETE /v2.0/security-group-rules/{id} with no body; with WithDeleteIgnoreMissing(false) a 204 succeeds while 404 and 409 remain errors, matching ignore_missing=False.
  Tests: `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBDeleteSecurityGroupRule`.

Python/Go differences:

- Only the default secgroup_source=neutron path with a network service is mapped; the Nova /os-security-groups branch with its normalization and the OpenStackCloudUnavailableFeature raised when secgroup_source is unset are not reproduced. The Nova branch returns False on 404, which Go does not provide.
- Python returns True on success; Go returns a nil error. DeleteWithOptions ignores 404 by default, so WithDeleteIgnoreMissing(false) is required.

### `python:cloud/_network/NetworkCloudMixin/list_networks`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.ListNetworkRecords`
- `github.com/JSYoo5B/go-openstacksdk/network.WithNetworkCloudListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithNetworkCloudFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithNetworkCloudHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithNetworkCloudMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithNetworkCloudCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network.WithNetworkCloudServiceAvailable`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.ProxyCloudListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithCloudRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ListNetworkRecords`

- Source list service absence returns owned nonnil [] even on a nil service before filters/JMES/HTTP. Truthy list filters require a JSON mapping, falsey filters become empty mapping; configured service errors are not silently mapped to absence.
  Tests: `network/cloud_network_lists_test.go:TestCloudNetworkListAbsentServicePrecedesFiltersAndSearchHasNoAbsenceGate`, `network/cloud_network_lists_test.go:TestCloudNetworkListFalseyFiltersAndSearchOrderedNameUpdate`.
- New connected Source lane uses cloned SourceStatusPolicy/SourceLinkPolicy and the complete existing Network descriptor/query profile. Original row/full page/header/status proof remains separate from projected defaults, large integers, Python-loaded local bool/number equality and owned connected location. Unknown availability_zone is read as the Source runtime attribute, while tenant_id is not inferred as Network.project_id.
  Tests: `network/cloud_network_lists_test.go:TestCloudNetworkListRawControlsLocalLoadedEqualityPagingAndReceipts`, `network/cloud_network_lists_test.go:TestCloudNetworkListRawMicroversionTransportAndUnknownLocationZone`, `network/v2/networks/python_cloud_lists_test.go:TestPythonNetworkCloudListSourceStatusDefaultsAndConnectedLocation`.
- Original bound self/resource_type fail during Proxy call; one-pass ordered conflicting attrs then reject restored base_path/paginated/cls/session collisions but ignore restored self/resource_type and restored JMES spelling. Raw headers/base_path/microversion are evaluated only when the Resource generator is consumed. Explicit empty base path returns empty with no HTTP but still checks iteration-start cancellation/source guard.
  Tests: `network/v2/networks/python_cloud_lists_test.go:TestPythonNetworkCloudListRawFormalsRemainLazyAcrossProxyPhases`, `network/v2/networks/python_cloud_lists_test.go:TestPythonNetworkCloudListOriginalAndRestoredBindingHaveDifferentPhases`, `network/v2/networks/python_cloud_lists_test.go:TestPythonNetworkCloudListEmptyPathSkipsHTTPAndKeepsIterationGuards`, `network/cloud_network_lists_test.go:TestCloudNetworkListOpaqueJMESListConversionAndDeferredFormals`.
- Opaque Proxy JMES values retain Source generator identity and HTTP0 timing; @ is consumed by cloud list(), array/string/object literals become iterable values, projection null/noniterable results and type errors fail without HTTP. Nested generator values retain laziness/location/source and no fabricated JSON/HTTP receipts. Go object keys iterate lexical order rather than Source insertion order, asserted explicitly.
  Tests: `network/cloud_network_lists_test.go:TestCloudNetworkListOpaqueJMESListConversionAndDeferredFormals`, `network/v2/networks/python_cloud_lists_test.go:TestPythonNetworkCloudListOpaqueProxyStaysLazyWithOwnedLocation`.
- Raw JSON microversion falsey nonnull values suppress headers, null uses selected version, numeric/bool and iterable tuple values follow the separately audited Keystoneauth JSON-domain normalization. Concrete Headers/Microversion/Location options and raw filters snapshot caller inputs before guarded callback/response use; original selected source and live token are preserved.
  Tests: `network/cloud_network_lists_test.go:TestCloudNetworkListRawMicroversionTransportAndUnknownLocationZone`, `network/cloud_network_lists_test.go:TestCloudNetworkListLateFailureIsAtomicAndOptionsOwnSource`.
- Cloud collection discards partial public outputs on late failure while actual page rejection/physical proof is retained. Accepted200 and rejected401/404/503 read+Close/cancel/source lanes close physical bodies once, make no retry/reauth calls and preserve actual status/body/header. Native401 independent ErrOriginal value and ErrReauth causes remain available; clean retry/token renewal are guarded and bounded in fixtures.
  Tests: `network/cloud_network_lists_test.go:TestCloudNetworkListLateFailureIsAtomicAndOptionsOwnSource`, `network/cloud_network_lists_test.go:TestCloudNetworkListPhysicalFaultsKeepActualReceiptAndNoNativeHooks`, `network/cloud_network_lists_test.go:TestCloudNetworkListCleanRetryTokenRenewalAndGuardedRetry`.
- Connection exposes this cloud read through its cached service and SDK-owned location defaults. Cancellation is checked before catalog discovery. Only Neutron list helpers turn clean missing catalog endpoints into [] before evaluating options; searches retain catalog errors. Caller-provided options retain their explicit values.
  Tests: `connection_cycle8_reads_test.go:TestConnectionCycle8CancelledReadsNeverDiscover`, `connection_cycle8_reads_test.go:TestConnectionCycle8MissingNeutronOnlyListsSkipOptions`, `connection_cycle8_reads_test.go:TestConnectionCycle8NetworkReadsCacheServiceAndForwardNameQuery`.
- Cloud filters reach the Source-shaped ProxyCloudListRecords lane: Integer max_items equal to one whole page is compared before the next raw row, so the marker GET is performed and its failure is observed; raw true/negative/fractional caps follow the same before-row comparison. Raw limit values reach the wire with Requests spelling (False, -1, 1.5, x, repeated values) and Resource.list keeps the original limit for its loop: only a Python-truthy limit performs the marker GET after a nonempty page without a next link, so false/0/0.0/""/null send one GET. base_path="" never enters Resource.list's while loop: no HTTP and no KSA microversion normalization, so an invalid raw microversion is unobserved and the result is empty; a nonempty route still rejects it before HTTP.
  Tests: `network/cloud_network_list_limits_test.go:TestCloudNetworkSubnetListsSourceLimitsAndCaps`, `network/v2/networks/python_raw_max_items_test.go:TestPythonNetworkRawMaxItemsJSONDomainAndDeferredComparison`, `network/v2/networks/python_raw_max_items_test.go:TestPythonNetworkRawMaxItemsExactPageBoundaryObservesNextResponse`.

Python/Go differences:

- SDK-R1: Go returns owned RawResource/receipt views instead of a mutable Python Resource and cache/lifecycle object. Declared descriptors are projected before row return and may fail earlier for an unaccessed malformed property. Actual Wire/Envelope/header/status are never forged from normalized values; runtime Resource.microversion attribute/object identity remain generic Resource parity scope.
- SDK-C1: JSON bytes retain exact integers and raw proof; local comparisons use audited CPython-loaded float/bool/int policy. Non-JSON custom objects, nonfinite JSON runtime values and exact unrestricted Python representation/Unicode/marshal behavior remain independent generic contracts. Go wraps structured errors while preserving actual response and physical/context causes.
- SDK-S1: source origin, service-relative path/filter/continuation guards and concrete transport options retain the original selected client. Arbitrary adapter/class injection, URI percent-mapping retargeting, foreign/query/fragment/traversal base paths are restricted; existing typed/legacy list APIs retain their baseline rather than receiving hidden Source flags.
- New ProxyCloudListRecords raw WithCloudRecordListQuery preserves Resource formal validation timing and original/restored binding behavior. It is the Source-connected lane; older WithRecordListQuery/ProxyListRecords keep the prior eager concrete control contract. Informational final100..199 and Source continuation-link policy apply only to the new cloned profile.
- Source cloud list() turns dict results into insertion-ordered keys; Go uses lexical keys and has an explicit z,a→a,z assertion. Opaque nested generators remain in Values with JSON=nil; ordinary outputs have Networks/Values/JSON. Typed-only callers can select Networks while arbitrary JMES callers inspect Values.
- Neutron27 stock get_network project-reader/service/shared/external/advsvc paths do not guarantee ADMIN-only provider/segments attributes or all-project visibility. SDK injects current-location facts but does not elevate access. Actual pinned Source/AST/fixture/race/build receipts and live authenticated deployment evidence remain distinct.
- Raw limit spelling and Python truthiness, raw max_items caps and the cap-equals-page marker GET are implemented on the Source-shaped Proxy lanes (shared opt-in Source list controls); typed ListRecords/AllRecords and the semantic Collection lanes keep strict Go integer limit/max_items preflight.

Unresolved boundaries:

- Raw JSON microversion exponent overflow remains a precise named runtime boundary: actual CPython json.loads + runtime KSA normalization accepts positive-overflow 1e400/[1e400] as network latest, while the current Go finite-number conversion rejects it on a nonempty route. KSA-only12 header-normalizer evidence is separately recorded; it is not claimed as pinned Source physical HTTP proof.

### `python:cloud/_network/NetworkCloudMixin/search_networks`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.SearchNetworkRecords`
- `github.com/JSYoo5B/go-openstacksdk/network.WithNetworkCloudListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithNetworkCloudFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithNetworkCloudHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithNetworkCloudMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithNetworkCloudCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network.WithNetworkCloudServiceAvailable`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.API.ProxyCloudListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/networks.WithCloudRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SearchNetworkRecords`

- Source search has no has_service absence gate: literal nonempty identity seeds server name query, filters ordered dict.update replaces that name and JSON iterable-pairs are accepted. UUID/glob inputs remain literal name; no local ID/name/_filter_list is invented.
  Tests: `network/cloud_network_lists_test.go:TestCloudNetworkListAbsentServicePrecedesFiltersAndSearchHasNoAbsenceGate`, `network/cloud_network_lists_test.go:TestCloudNetworkListFalseyFiltersAndSearchOrderedNameUpdate`.
- New connected Source lane uses cloned SourceStatusPolicy/SourceLinkPolicy and the complete existing Network descriptor/query profile. Original row/full page/header/status proof remains separate from projected defaults, large integers, Python-loaded local bool/number equality and owned connected location. Unknown availability_zone is read as the Source runtime attribute, while tenant_id is not inferred as Network.project_id.
  Tests: `network/cloud_network_lists_test.go:TestCloudNetworkListRawControlsLocalLoadedEqualityPagingAndReceipts`, `network/cloud_network_lists_test.go:TestCloudNetworkListRawMicroversionTransportAndUnknownLocationZone`, `network/v2/networks/python_cloud_lists_test.go:TestPythonNetworkCloudListSourceStatusDefaultsAndConnectedLocation`.
- Original bound self/resource_type fail during Proxy call; one-pass ordered conflicting attrs then reject restored base_path/paginated/cls/session collisions but ignore restored self/resource_type and restored JMES spelling. Raw headers/base_path/microversion are evaluated only when the Resource generator is consumed. Explicit empty base path returns empty with no HTTP but still checks iteration-start cancellation/source guard.
  Tests: `network/v2/networks/python_cloud_lists_test.go:TestPythonNetworkCloudListRawFormalsRemainLazyAcrossProxyPhases`, `network/v2/networks/python_cloud_lists_test.go:TestPythonNetworkCloudListOriginalAndRestoredBindingHaveDifferentPhases`, `network/v2/networks/python_cloud_lists_test.go:TestPythonNetworkCloudListEmptyPathSkipsHTTPAndKeepsIterationGuards`, `network/cloud_network_lists_test.go:TestCloudNetworkListOpaqueJMESListConversionAndDeferredFormals`.
- Opaque Proxy JMES values retain Source generator identity and HTTP0 timing; @ is consumed by cloud list(), array/string/object literals become iterable values, projection null/noniterable results and type errors fail without HTTP. Nested generator values retain laziness/location/source and no fabricated JSON/HTTP receipts. Go object keys iterate lexical order rather than Source insertion order, asserted explicitly.
  Tests: `network/cloud_network_lists_test.go:TestCloudNetworkListOpaqueJMESListConversionAndDeferredFormals`, `network/v2/networks/python_cloud_lists_test.go:TestPythonNetworkCloudListOpaqueProxyStaysLazyWithOwnedLocation`.
- Raw JSON microversion falsey nonnull values suppress headers, null uses selected version, numeric/bool and iterable tuple values follow the separately audited Keystoneauth JSON-domain normalization. Concrete Headers/Microversion/Location options and raw filters snapshot caller inputs before guarded callback/response use; original selected source and live token are preserved.
  Tests: `network/cloud_network_lists_test.go:TestCloudNetworkListRawMicroversionTransportAndUnknownLocationZone`, `network/cloud_network_lists_test.go:TestCloudNetworkListLateFailureIsAtomicAndOptionsOwnSource`.
- Cloud collection discards partial public outputs on late failure while actual page rejection/physical proof is retained. Accepted200 and rejected401/404/503 read+Close/cancel/source lanes close physical bodies once, make no retry/reauth calls and preserve actual status/body/header. Native401 independent ErrOriginal value and ErrReauth causes remain available; clean retry/token renewal are guarded and bounded in fixtures.
  Tests: `network/cloud_network_lists_test.go:TestCloudNetworkListLateFailureIsAtomicAndOptionsOwnSource`, `network/cloud_network_lists_test.go:TestCloudNetworkListPhysicalFaultsKeepActualReceiptAndNoNativeHooks`, `network/cloud_network_lists_test.go:TestCloudNetworkListCleanRetryTokenRenewalAndGuardedRetry`.
- Connection exposes this cloud read through its cached service and SDK-owned location defaults. Cancellation is checked before catalog discovery. Only Neutron list helpers turn clean missing catalog endpoints into [] before evaluating options; searches retain catalog errors. Caller-provided options retain their explicit values.
  Tests: `connection_cycle8_reads_test.go:TestConnectionCycle8CancelledReadsNeverDiscover`, `connection_cycle8_reads_test.go:TestConnectionCycle8MissingNeutronOnlyListsSkipOptions`, `connection_cycle8_reads_test.go:TestConnectionCycle8NetworkReadsCacheServiceAndForwardNameQuery`.
- Cloud filters reach the Source-shaped ProxyCloudListRecords lane: Integer max_items equal to one whole page is compared before the next raw row, so the marker GET is performed and its failure is observed; raw true/negative/fractional caps follow the same before-row comparison. Raw limit values reach the wire with Requests spelling (False, -1, 1.5, x, repeated values) and Resource.list keeps the original limit for its loop: only a Python-truthy limit performs the marker GET after a nonempty page without a next link, so false/0/0.0/""/null send one GET. base_path="" never enters Resource.list's while loop: no HTTP and no KSA microversion normalization, so an invalid raw microversion is unobserved and the result is empty; a nonempty route still rejects it before HTTP.
  Tests: `network/cloud_network_list_limits_test.go:TestCloudNetworkSubnetListsSourceLimitsAndCaps`, `network/v2/networks/python_raw_max_items_test.go:TestPythonNetworkRawMaxItemsJSONDomainAndDeferredComparison`, `network/v2/networks/python_raw_max_items_test.go:TestPythonNetworkRawMaxItemsExactPageBoundaryObservesNextResponse`.

Python/Go differences:

- SDK-R1: Go returns owned RawResource/receipt views instead of a mutable Python Resource and cache/lifecycle object. Declared descriptors are projected before row return and may fail earlier for an unaccessed malformed property. Actual Wire/Envelope/header/status are never forged from normalized values; runtime Resource.microversion attribute/object identity remain generic Resource parity scope.
- SDK-C1: JSON bytes retain exact integers and raw proof; local comparisons use audited CPython-loaded float/bool/int policy. Non-JSON custom objects, nonfinite JSON runtime values and exact unrestricted Python representation/Unicode/marshal behavior remain independent generic contracts. Go wraps structured errors while preserving actual response and physical/context causes.
- SDK-S1: source origin, service-relative path/filter/continuation guards and concrete transport options retain the original selected client. Arbitrary adapter/class injection, URI percent-mapping retargeting, foreign/query/fragment/traversal base paths are restricted; existing typed/legacy list APIs retain their baseline rather than receiving hidden Source flags.
- New ProxyCloudListRecords raw WithCloudRecordListQuery preserves Resource formal validation timing and original/restored binding behavior. It is the Source-connected lane; older WithRecordListQuery/ProxyListRecords keep the prior eager concrete control contract. Informational final100..199 and Source continuation-link policy apply only to the new cloned profile.
- Source cloud list() turns dict results into insertion-ordered keys; Go uses lexical keys and has an explicit z,a→a,z assertion. Opaque nested generators remain in Values with JSON=nil; ordinary outputs have Networks/Values/JSON. Typed-only callers can select Networks while arbitrary JMES callers inspect Values.
- Neutron27 stock get_network project-reader/service/shared/external/advsvc paths do not guarantee ADMIN-only provider/segments attributes or all-project visibility. SDK injects current-location facts but does not elevate access. Actual pinned Source/AST/fixture/race/build receipts and live authenticated deployment evidence remain distinct.
- Raw limit spelling and Python truthiness, raw max_items caps and the cap-equals-page marker GET are implemented on the Source-shaped Proxy lanes (shared opt-in Source list controls); typed ListRecords/AllRecords and the semantic Collection lanes keep strict Go integer limit/max_items preflight.

Unresolved boundaries:

- Raw JSON microversion exponent overflow remains a precise named runtime boundary: actual CPython json.loads + runtime KSA normalization accepts positive-overflow 1e400/[1e400] as network latest, while the current Go finite-number conversion rejects it on a nonempty route. KSA-only12 header-normalizer evidence is separately recorded; it is not claimed as pinned Source physical HTTP proof.

### `python:cloud/_network/NetworkCloudMixin/list_subnets`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.RecordListGenerator.JMESPathIdentity`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.RecordListGenerator.MarshalJSON`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListConflictingAttrs`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetReadFiltersJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetReadBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetReadCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetNetworkAvailable`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ListSubnetRecords`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.ListSubnetRecords`

- New opt-in source profile clones pinned Subnet/NetworkResource descriptor and query declarations; missing defaults, bool/int/list casting, ordered canonical/wire response aliases, project/tenant fallback, unknown Body and large integer tokens are owned. Row if_match is a constructor Header and actual page Header remains separate. Baseline native/Find/Resources policies stay unchanged.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsOwnedSparseDefaultsAndOrderedAliases`, `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsDescriptorsRowHeadersAndIndependentActualEvidence`.
- Canonical/wire Query mapping is pushed down without an invented name hint; non-query declared Body attributes filter loaded descriptors locally with Python loaded JSON equality. Unknown kwargs are discarded before application marshaling. Source max_items/limit raw behavior remains incomplete and is explicitly excluded below.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsPushdownLocalLoadedFiltersAndUnknownMarshalDiscard`.
- Proxy._list evaluates JMESPath against the actual opaque Resource.list generator, before Resource control normalization or HTTP. Index/projection/type expressions produce null without HTTP; literals preserve arbitrary JSON; @ returns the same generator, nested to_array keeps it unconsumed, and length errors before HTTP. ListRecords/AllRecords are ordinary resource-only convenience APIs.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsProxyGeneratorJMESAndLazyResourceControls`.
- Original bound self/resource_type/cls/session collisions and restored cls/session/base_path/paginated conflicts are eager. Truthy __conflicting_attrs restores ordered kwargs once; restored self/resource_type are discarded as unknown Resource attrs and restored JMES does not replace original expression. Resource headers/base_path/microversion are deferred until actual generator consumption, so an opaque empty literal avoids invalid controls entirely.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsOneLevelRestorationBindingAndImmutableOptions`, `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsProxyGeneratorJMESAndLazyResourceControls`.
- Only explicit empty string base_path produces an empty HTTP0 generator; null selects default, false/zero/list/object fails at consumption. Context/source checks still run before an empty generator returns. Literal same-origin service-relative path policy remains narrower than arbitrary Python formatting.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsEmptyPathOnlyNullDefaultAndDelayedCancellation`.
- Per-call microversion None keeps captured default and every other falsey JSON value disables it. Ordinary JSON numeric/string/list/object-key iterable normalization follows KSA conventions including true/1 -> 1.0, [2,4] -> 2.4, v2.4 -> 2.4, latest and major.latest rejection. Python overflow infinity/latest conversion remains unsupported.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsMicroversionJSONDomainAndDefaultDisable`.
- Source status100..399, plural singleton objects, required accepted JSON parsing, row constructor/descriptor faults, root links precedence, ignored dictionary next, HTTP/limit/raw-marker continuation use the shared guarded pager. Actual late-page body/read/Close failures preserve original native/ResponseError evidence and callbacks are suppressed only for physical faults; clean token renewal retains captured headers. Dynamic cap exact-boundary semantics remain named gaps.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsSourceLinkPrecedenceAndRawMarkerMaxItems`, `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsMalformedAcceptedPagesKeepPhysicalProof`, `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsLatePhysicalFaultsNative401AndNoFaultReplay`, `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsCleanReauthenticationAndCapturedHeaders`.
- Concrete Source cloud read options snapshot raw filters, transport map/pointers, selected service/API/provider/source and current location. List alone checks network availability before filter/JMES evaluation and can return [] for an owned clean catalog absence; Search never suppresses unavailable network and directly sends a truthy name before dict.update(filters). UUID/glob strings remain literal name queries. Truthy list filters require a mapping, while search accepts JSON iterable key/value pairs and name override.
  Tests: `network/cloud_subnet_lists_test.go:TestCloudSubnetListsDirectQuerySearchOverrideAndFalseyFilters`, `network/cloud_subnet_lists_test.go:TestCloudSubnetListsOnlyListMissingServiceSkipsInvalidFilters`, `network/cloud_subnet_lists_test.go:TestCloudSubnetListsSnapshotsHeaderNilResetAndSourceMutation`.
- Cloud list() consumes actual Resource generators fully or converts arbitrary JSON lists/Unicode strings/map keys. It rejects scalar/null, preserves falsey literal members, and never invents Resource/Wire/header/status from a JSON literal. Nested opaque generators stay unconsumed with JSON nil and retain SDK location/source/context when subsequently consumed. Go dictionary key iteration is explicitly lexical.
  Tests: `network/cloud_subnet_lists_test.go:TestCloudSubnetListsLiteralPolymorphismNoInventedResourcesOrReceipts`, `network/cloud_subnet_lists_test.go:TestCloudSubnetListsNestedGeneratorCarriesLocationAndDelayedGuards`.
- Current location is SDK owned: loaded falsey/current project retains current scope and truthy foreign project clears names; unknown availability_zone keeps raw null/false/zero/list/object as zone. Response forged location remains actual Wire only. Resource, Wire, Inventory and LastResponse metadata are independently copied.
  Tests: `network/cloud_subnet_lists_test.go:TestCloudSubnetListsSDKLocationRawZoneProjectsAndOwnedEvidence`, `network/cloud_subnet_lists_test.go:TestCloudSubnetListsNestedGeneratorCarriesLocationAndDelayedGuards`.
- Only a complete conversion commits Subnets/Value/JSON; partial actual resources remain in Inventory on late errors. LastResponse is the last observed yielded resource/error receipt, not a guarantee of last empty/filtered page. Post-row nested location/guard failures use immutable actual ResponseError; pre-consumption cancellation has HTTP0/no receipt. Context causes and actual rejected page/descriptor status/body/header stay observable.
  Tests: `network/cloud_subnet_lists_test.go:TestCloudSubnetListsAllPagesCommitOnlyCompleteRowsAndActualFailure`, `network/cloud_subnet_lists_test.go:TestCloudSubnetListsPreflightCancellationNilOptionsAndTerminalDescriptor`, `network/cloud_subnet_lists_test.go:TestCloudSubnetListsNestedGeneratorCarriesLocationAndDelayedGuards`.
- Connection exposes this cloud read through its cached service and SDK-owned location defaults. Cancellation is checked before catalog discovery. Only Neutron list helpers turn clean missing catalog endpoints into [] before evaluating options; searches retain catalog errors. Caller-provided options retain their explicit values.
  Tests: `connection_cycle8_reads_test.go:TestConnectionCycle8CancelledReadsNeverDiscover`, `connection_cycle8_reads_test.go:TestConnectionCycle8MissingNeutronOnlyListsSkipOptions`, `connection_cycle8_reads_test.go:TestConnectionCycle8NetworkReadsCacheServiceAndForwardNameQuery`.
- Cloud filters reach the Source-shaped subnet list lane: Integer max_items equal to one whole page is compared before the next raw row, so the marker GET is performed and its failure is observed; raw true/negative/fractional caps follow the same before-row comparison. Raw limit values reach the wire with Requests spelling (False, -1, 1.5, x, repeated values) and Resource.list keeps the original limit for its loop: only a Python-truthy limit performs the marker GET after a nonempty page without a next link, so false/0/0.0/""/null send one GET. base_path="" never enters Resource.list's while loop: no HTTP and no KSA microversion normalization, so an invalid raw microversion is unobserved and the result is empty; a nonempty route still rejects it before HTTP.
  Tests: `network/cloud_network_list_limits_test.go:TestCloudNetworkSubnetListsSourceLimitsAndCaps`, `network/v2/subnets/python_raw_max_items_test.go:TestPythonSubnetRawMaxItemsJSONDomainAndDeferredComparison`, `network/v2/subnets/python_raw_max_items_test.go:TestPythonSubnetRawMaxItemsExactPageBoundaryObservesNextResponse`.

Python/Go differences:

- This adds owned Source-shaped lazy Subnet records and ordinary cloud list/search orchestration to three exact declarations; status stays unresolved and full SDK supported delta is 0 because the nonfinite-version and shared runtime gaps remain. Existing native List/Resources/Find bindings retain their baseline policies.
- SDK APIs are concrete functional options rather than application builders/interfaces. JSON/individual options preserve source order; unspecified Go map order and cloud literal dictionary key results use deterministic lexical order. UTF8 JSON ownership, cancellation/source validation, typed positive controls and same-origin literal relative paths are explicit Go policies.
- Actual HTTP data is separated from Resource defaults/computed location and JMES literals. LastResponse identifies the last observed yielded resource/error receipt and is not a final-page receipt for empty or fully filtered pages. Native401 preserves ErrOriginal and ErrReauth despite absent Unwrap; physical faults are never presented as clean retry.
- Pinned Source evidence uses CPython3.14.8 actual cloud/Proxy/Resource/KSA code with identified adapter/config/HTTP fixtures, not a live backend or runtime authorization. Independent six fixture sets total696 observations including348 Subnet observations; KSA overflow-edge is header normalization only. Neutron27 stock get_subnet project-read policy and admin-only segment visibility do not guarantee live deployment permissions.

Unresolved boundaries:

- Named Source nonfinite microversion gap: KSA normalizes JSON float overflow 1e400 and [1e400] to latest, but this finite Go lane rejects nonfinite conversion on a nonempty route. Ordinary finite numeric/string/iterable and falsey/default cases are implemented, not the full Python runtime domain.
- Shared SDK-R1 mutable Resource/Container/Body/URI/Header descriptors, dirty/cache/serialization lifecycle and arbitrary computed/class members; SDK-C1 Connection/auth/session/discovery configuration and generic request-cache/microversion negotiation; SDK-S1 arbitrary service/Proxy/session/runtime-object behavior remain unresolved. Verified immutable literal same-origin route/header policies are Go constraints rather than claims of arbitrary Python class/URI formatting support.

### `python:cloud/_network/NetworkCloudMixin/search_subnets`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.RecordListGenerator.JMESPathIdentity`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.RecordListGenerator.MarshalJSON`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListConflictingAttrs`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetReadFiltersJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetReadBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetReadCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetNetworkAvailable`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SearchSubnetRecords`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.SearchSubnetRecords`

- New opt-in source profile clones pinned Subnet/NetworkResource descriptor and query declarations; missing defaults, bool/int/list casting, ordered canonical/wire response aliases, project/tenant fallback, unknown Body and large integer tokens are owned. Row if_match is a constructor Header and actual page Header remains separate. Baseline native/Find/Resources policies stay unchanged.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsOwnedSparseDefaultsAndOrderedAliases`, `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsDescriptorsRowHeadersAndIndependentActualEvidence`.
- Canonical/wire Query mapping is pushed down without an invented name hint; non-query declared Body attributes filter loaded descriptors locally with Python loaded JSON equality. Unknown kwargs are discarded before application marshaling. Source max_items/limit raw behavior remains incomplete and is explicitly excluded below.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsPushdownLocalLoadedFiltersAndUnknownMarshalDiscard`.
- Proxy._list evaluates JMESPath against the actual opaque Resource.list generator, before Resource control normalization or HTTP. Index/projection/type expressions produce null without HTTP; literals preserve arbitrary JSON; @ returns the same generator, nested to_array keeps it unconsumed, and length errors before HTTP. ListRecords/AllRecords are ordinary resource-only convenience APIs.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsProxyGeneratorJMESAndLazyResourceControls`.
- Original bound self/resource_type/cls/session collisions and restored cls/session/base_path/paginated conflicts are eager. Truthy __conflicting_attrs restores ordered kwargs once; restored self/resource_type are discarded as unknown Resource attrs and restored JMES does not replace original expression. Resource headers/base_path/microversion are deferred until actual generator consumption, so an opaque empty literal avoids invalid controls entirely.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsOneLevelRestorationBindingAndImmutableOptions`, `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsProxyGeneratorJMESAndLazyResourceControls`.
- Only explicit empty string base_path produces an empty HTTP0 generator; null selects default, false/zero/list/object fails at consumption. Context/source checks still run before an empty generator returns. Literal same-origin service-relative path policy remains narrower than arbitrary Python formatting.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsEmptyPathOnlyNullDefaultAndDelayedCancellation`.
- Per-call microversion None keeps captured default and every other falsey JSON value disables it. Ordinary JSON numeric/string/list/object-key iterable normalization follows KSA conventions including true/1 -> 1.0, [2,4] -> 2.4, v2.4 -> 2.4, latest and major.latest rejection. Python overflow infinity/latest conversion remains unsupported.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsMicroversionJSONDomainAndDefaultDisable`.
- Source status100..399, plural singleton objects, required accepted JSON parsing, row constructor/descriptor faults, root links precedence, ignored dictionary next, HTTP/limit/raw-marker continuation use the shared guarded pager. Actual late-page body/read/Close failures preserve original native/ResponseError evidence and callbacks are suppressed only for physical faults; clean token renewal retains captured headers. Dynamic cap exact-boundary semantics remain named gaps.
  Tests: `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsSourceLinkPrecedenceAndRawMarkerMaxItems`, `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsMalformedAcceptedPagesKeepPhysicalProof`, `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsLatePhysicalFaultsNative401AndNoFaultReplay`, `network/v2/subnets/python_cloud_lists_test.go:TestPythonSubnetListsCleanReauthenticationAndCapturedHeaders`.
- Concrete Source cloud read options snapshot raw filters, transport map/pointers, selected service/API/provider/source and current location. List alone checks network availability before filter/JMES evaluation and can return [] for an owned clean catalog absence; Search never suppresses unavailable network and directly sends a truthy name before dict.update(filters). UUID/glob strings remain literal name queries. Truthy list filters require a mapping, while search accepts JSON iterable key/value pairs and name override.
  Tests: `network/cloud_subnet_lists_test.go:TestCloudSubnetListsDirectQuerySearchOverrideAndFalseyFilters`, `network/cloud_subnet_lists_test.go:TestCloudSubnetListsOnlyListMissingServiceSkipsInvalidFilters`, `network/cloud_subnet_lists_test.go:TestCloudSubnetListsSnapshotsHeaderNilResetAndSourceMutation`.
- Cloud list() consumes actual Resource generators fully or converts arbitrary JSON lists/Unicode strings/map keys. It rejects scalar/null, preserves falsey literal members, and never invents Resource/Wire/header/status from a JSON literal. Nested opaque generators stay unconsumed with JSON nil and retain SDK location/source/context when subsequently consumed. Go dictionary key iteration is explicitly lexical.
  Tests: `network/cloud_subnet_lists_test.go:TestCloudSubnetListsLiteralPolymorphismNoInventedResourcesOrReceipts`, `network/cloud_subnet_lists_test.go:TestCloudSubnetListsNestedGeneratorCarriesLocationAndDelayedGuards`.
- Current location is SDK owned: loaded falsey/current project retains current scope and truthy foreign project clears names; unknown availability_zone keeps raw null/false/zero/list/object as zone. Response forged location remains actual Wire only. Resource, Wire, Inventory and LastResponse metadata are independently copied.
  Tests: `network/cloud_subnet_lists_test.go:TestCloudSubnetListsSDKLocationRawZoneProjectsAndOwnedEvidence`, `network/cloud_subnet_lists_test.go:TestCloudSubnetListsNestedGeneratorCarriesLocationAndDelayedGuards`.
- Only a complete conversion commits Subnets/Value/JSON; partial actual resources remain in Inventory on late errors. LastResponse is the last observed yielded resource/error receipt, not a guarantee of last empty/filtered page. Post-row nested location/guard failures use immutable actual ResponseError; pre-consumption cancellation has HTTP0/no receipt. Context causes and actual rejected page/descriptor status/body/header stay observable.
  Tests: `network/cloud_subnet_lists_test.go:TestCloudSubnetListsAllPagesCommitOnlyCompleteRowsAndActualFailure`, `network/cloud_subnet_lists_test.go:TestCloudSubnetListsPreflightCancellationNilOptionsAndTerminalDescriptor`, `network/cloud_subnet_lists_test.go:TestCloudSubnetListsNestedGeneratorCarriesLocationAndDelayedGuards`.
- Connection exposes this cloud read through its cached service and SDK-owned location defaults. Cancellation is checked before catalog discovery. Only Neutron list helpers turn clean missing catalog endpoints into [] before evaluating options; searches retain catalog errors. Caller-provided options retain their explicit values.
  Tests: `connection_cycle8_reads_test.go:TestConnectionCycle8CancelledReadsNeverDiscover`, `connection_cycle8_reads_test.go:TestConnectionCycle8MissingNeutronOnlyListsSkipOptions`, `connection_cycle8_reads_test.go:TestConnectionCycle8NetworkReadsCacheServiceAndForwardNameQuery`.
- Cloud filters reach the Source-shaped subnet list lane: Integer max_items equal to one whole page is compared before the next raw row, so the marker GET is performed and its failure is observed; raw true/negative/fractional caps follow the same before-row comparison. Raw limit values reach the wire with Requests spelling (False, -1, 1.5, x, repeated values) and Resource.list keeps the original limit for its loop: only a Python-truthy limit performs the marker GET after a nonempty page without a next link, so false/0/0.0/""/null send one GET. base_path="" never enters Resource.list's while loop: no HTTP and no KSA microversion normalization, so an invalid raw microversion is unobserved and the result is empty; a nonempty route still rejects it before HTTP.
  Tests: `network/cloud_network_list_limits_test.go:TestCloudNetworkSubnetListsSourceLimitsAndCaps`, `network/v2/subnets/python_raw_max_items_test.go:TestPythonSubnetRawMaxItemsJSONDomainAndDeferredComparison`, `network/v2/subnets/python_raw_max_items_test.go:TestPythonSubnetRawMaxItemsExactPageBoundaryObservesNextResponse`.

Python/Go differences:

- This adds owned Source-shaped lazy Subnet records and ordinary cloud list/search orchestration to three exact declarations; status stays unresolved and full SDK supported delta is 0 because the nonfinite-version and shared runtime gaps remain. Existing native List/Resources/Find bindings retain their baseline policies.
- SDK APIs are concrete functional options rather than application builders/interfaces. JSON/individual options preserve source order; unspecified Go map order and cloud literal dictionary key results use deterministic lexical order. UTF8 JSON ownership, cancellation/source validation, typed positive controls and same-origin literal relative paths are explicit Go policies.
- Actual HTTP data is separated from Resource defaults/computed location and JMES literals. LastResponse identifies the last observed yielded resource/error receipt and is not a final-page receipt for empty or fully filtered pages. Native401 preserves ErrOriginal and ErrReauth despite absent Unwrap; physical faults are never presented as clean retry.
- Pinned Source evidence uses CPython3.14.8 actual cloud/Proxy/Resource/KSA code with identified adapter/config/HTTP fixtures, not a live backend or runtime authorization. Independent six fixture sets total696 observations including348 Subnet observations; KSA overflow-edge is header normalization only. Neutron27 stock get_subnet project-read policy and admin-only segment visibility do not guarantee live deployment permissions.

Unresolved boundaries:

- Named Source nonfinite microversion gap: KSA normalizes JSON float overflow 1e400 and [1e400] to latest, but this finite Go lane rejects nonfinite conversion on a nonempty route. Ordinary finite numeric/string/iterable and falsey/default cases are implemented, not the full Python runtime domain.
- Shared SDK-R1 mutable Resource/Container/Body/URI/Header descriptors, dirty/cache/serialization lifecycle and arbitrary computed/class members; SDK-C1 Connection/auth/session/discovery configuration and generic request-cache/microversion negotiation; SDK-S1 arbitrary service/Proxy/session/runtime-object behavior remain unresolved. Verified immutable literal same-origin route/header policies are Go constraints rather than claims of arbitrary Python class/URI formatting support.

### `python:cloud/_network/NetworkCloudMixin/list_ports`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ListPorts`
- `github.com/JSYoo5B/go-openstacksdk.Connection.ListPortValues`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.ListSourceValues`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSourceCloudListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSourceCloudListFilters`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSourceCloudListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSourceCloudListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithSourceCloudListLocation`

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
- Connection uses cached Neutron service; owned per-row CurrentLocation is applied before local selection without changing actual Wire, and callbacks execute once.
  Tests: `connection_network_record_lists_test.go:TestConnectionPortRecordsCachedServiceAndAutomaticPerRowLocation`.
- Clean absent catalog returns nonnil empty list before list options; search still evaluates loaded local filters and expressions on [], including scalar/null and invalid expressions, with zero HTTP. Mixed catalog errors and cancellation remain failures.
  Tests: `connection_network_record_lists_test.go:TestConnectionPortRecordsCatalogAbsencePrecedesListOptionsButSearchStillEvaluatesEmpty`, `connection_network_record_lists_test.go:TestConnectionPortRecordsMixedCatalogFailureAndCancellationDoNotClaimEmpty`.
- Invalid Connection/context fails before discovery; late-page failure preserves actual response error evidence and prevents a fabricated successful selection.
  Tests: `connection_network_record_lists_test.go:TestConnectionPortRecordsPreflightAndLatePageFailureRetainActualEvidence`.
- Source list() on literal arrays, objects, Unicode strings and empty iterables preserves JSON values without fake PortRecord or receipt; JSON object key order is explicitly Go lexical.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListLiteralIterablesDoNotFabricateResources`.
- Null/scalar noniterable and opaque-generator JMES function/syntax failures return before HTTP; result JSON is never fabricated.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListNonIterablesAndGeneratorErrorsDoNoHTTP`.
- Direct returned Source generator is eagerly collected into actual records with unknown/default fields, query/header pushdown, remote-only seed, actual page proof and computed per-record cloud location.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListReturnedGeneratorUsesActualRowsAndNoSeed`.
- Nested generator pointer remains opaque and lazy; serialization fails, explicit later Records iteration performs actual transport.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListNestedGeneratorRemainsOpaqueAndLazy`.
- Missing and CPython falsey JSON filters become empty kwargs; truthy nonobject filters fail before HTTP instead of being treated as cloud search expressions.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListFiltersFalseyAndTruthyNonMappings`.
- Raw filters/dedicated headers/typed callback snapshots own input state and nil callback maps; dedicated headers last-win without empty default overriding Source raw headers.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListHeadersInputAndCallbackSnapshots`.
- A later page physical fault preserves only observed actual Inventory; successful cloud Values/Ports/JSON are absent on failure.
  Tests: `network/v2/ports/python_list_records_values_test.go:TestSourceCloudPortListLateFaultRetainsOnlyActualInventory`.
- Clean absent network service returns nonnil [] before filter expansion or option callbacks; endpoint errors and context remain errors.
  Tests: `connection_port_values_test.go:TestConnectionListPortValuesAbsentServiceSkipsFilterExpansion`.
- Whole Connection shares cached network client and injects current location for actual records, preserves literal values without HTTP/fake records, and retains existing typed ListPorts compatibility.
  Tests: `connection_port_values_test.go:TestConnectionListPortValuesCachedClientLocationAndTypedListCompatibility`.

Python/Go differences:

- Concrete Go records expose owned declared Resource view and independent actual Wire/Envelope/Header/StatusCode. Full mutable Python Resource/cache/dirty-state/runtime subclass handling remains tracked shared SDK-R1/C1/S1, not counted as generated transport parity.
- ProxyListRecords reproduces Source generator-based JMES zero-HTTP values/null/type errors/opaque identity and nested generators. ListRecordValues is explicitly eager Go expression functionality, not claimed identical to Source generator input.
- ListRecords keeps a resource-only iterator and directs polymorphic truthy expressions to ProxyListRecords; falsey/nonstring Proxy controls preserve lazy records. Standard Go iter.Seq2 iteration semantics remain the owned SDK abstraction.
- List routes remain within captured service endpoint; continuation preserves selected origin/collection/filter and rejects unsafe route cycles. Go dictionary/RFC link compatibility and strict validation are explicit Source differences.
- Both list local filters and cloud dictionary selection opt into CPython JSON-loaded equality. Full JMESPath retains documented exact-decimal/lexical object traversal/strict grammar boundaries, and JSON-map object functions can differ from Python Port exact-typename function checks.
- Connection models clean service-catalog absence and explicit endpoint overrides; the broad Python has_service configuration-disabled service policy remains a shared SDK-C1 contract. Typed Connection.ListPorts intentionally retains record-only results; the named list_ports row remains unresolved for polymorphic iterable conversion.
- Concrete JSON/context/captured-client API; arbitrary Python Resource/session/class/callable substitutions are not accepted. Shared cloud configuration disabled-service semantics and mutable Resource R1 contracts remain separate tracked scope.
- Public Gophercloud testhelper/nativefixture reuse and actual response errors preserve physical read/close causes; opaque or literal local JSON never receives invented resource/HTTP evidence.
- Captured route/header/version protection, Go error returns and retained observed inventory differ intentionally from dynamic Python exception-only return conventions.
- Python literal object list() uses insertion order; Go JMES maps use lexical key order. Existing strict JMES syntax/exact numeric evaluator/custom Resource function type differences remain documented.
- Existing typed Connection.ListPorts remains resource-only compatibility API; the complete Source polymorphic list path is explicitly Connection.ListPortValues/API.ListSourceValues.
- Computed current cloud location is injected by Connection and optional for direct leaf; shared Source service configuration disable controls are not inferred from absent endpoint tests.

### `python:cloud/_network/NetworkCloudMixin/search_ports`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.SearchRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordSearchFilters`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordSearchExpression`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordSearchHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordSearchMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordSearchOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordSearchLocation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordSearchCloudLocation`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SearchPorts`

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
- Connection uses cached Neutron service; owned per-row CurrentLocation is applied before local selection without changing actual Wire, and callbacks execute once.
  Tests: `connection_network_record_lists_test.go:TestConnectionPortRecordsCachedServiceAndAutomaticPerRowLocation`.
- Clean absent catalog returns nonnil empty list before list options; search still evaluates loaded local filters and expressions on [], including scalar/null and invalid expressions, with zero HTTP. Mixed catalog errors and cancellation remain failures.
  Tests: `connection_network_record_lists_test.go:TestConnectionPortRecordsCatalogAbsencePrecedesListOptionsButSearchStillEvaluatesEmpty`, `connection_network_record_lists_test.go:TestConnectionPortRecordsMixedCatalogFailureAndCancellationDoNotClaimEmpty`.
- Invalid Connection/context fails before discovery; late-page failure preserves actual response error evidence and prevents a fabricated successful selection.
  Tests: `connection_network_record_lists_test.go:TestConnectionPortRecordsPreflightAndLatePageFailureRetainActualEvidence`.

Python/Go differences:

- Concrete Go records expose owned declared Resource view and independent actual Wire/Envelope/Header/StatusCode. Full mutable Python Resource/cache/dirty-state/runtime subclass handling remains tracked shared SDK-R1/C1/S1, not counted as generated transport parity.
- ProxyListRecords reproduces Source generator-based JMES zero-HTTP values/null/type errors/opaque identity and nested generators. ListRecordValues is explicitly eager Go expression functionality, not claimed identical to Source generator input.
- ListRecords keeps a resource-only iterator and directs polymorphic truthy expressions to ProxyListRecords; falsey/nonstring Proxy controls preserve lazy records. Standard Go iter.Seq2 iteration semantics remain the owned SDK abstraction.
- List routes remain within captured service endpoint; continuation preserves selected origin/collection/filter and rejects unsafe route cycles. Go dictionary/RFC link compatibility and strict validation are explicit Source differences.
- Both list local filters and cloud dictionary selection opt into CPython JSON-loaded equality. Full JMESPath retains documented exact-decimal/lexical object traversal/strict grammar boundaries, and JSON-map object functions can differ from Python Port exact-typename function checks.
- Connection models clean service-catalog absence and explicit endpoint overrides; the broad Python has_service configuration-disabled service policy remains a shared SDK-C1 contract. Typed Connection.ListPorts intentionally retains record-only results; the named list_ports row remains unresolved for polymorphic iterable conversion.

### `python:cloud/_network/NetworkCloudMixin/list_router_interfaces`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.ListRouterInterfaces`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.ListRouterInterfaces`
- `github.com/JSYoo5B/go-openstacksdk/network.RouterInterfaceRouterID`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceType`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceTypeJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceReadCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.ListRecords`

- Library reads literal router dictionary id and sends only the declared device_id query without router member fetch/name lookup/response seeds. Null/bool/number/list/dict ID query behavior, missing/nonmapping input errors and real untyped owner values follow Source JSON-domain semantics.
  Tests: `network/cloud_router_interfaces_test.go:TestCloudRouterInterfacesRouterDictionaryReadDoesNotResolveOrSeed`.
- Every port page is collected before the Source discriminator. None/false/zero/underflow/empty values select both buckets; exact internal/external select their owner families; other truthy values yield empty success after I/O. Internal owners precede gateway owners with stable within-bucket order and no duplicate removal.
  Tests: `network/cloud_router_interfaces_test.go:TestCloudRouterInterfacesCollectsEveryPageBeforeStableOwnerBuckets`, `network/cloud_router_interfaces_test.go:TestCloudRouterInterfacesFalseyExactUnknownAndNonstringTypes`.
- Actual normalized Port Resource/raw Wire/envelope/header/status, selected Ports and partial Inventory/LastPort/JSON Value are independently owned. CurrentLocation is injected by Connection with loaded project semantics and no forged wire zone. Headers/microversion/JSON pointers are snapshots and source/API/context guards span callbacks, collection and selection.
  Tests: `network/cloud_router_interfaces_test.go:TestCloudRouterInterfacesOwnedInputsLocationHeadersAndRawReceipt`, `network/cloud_router_interfaces_test.go:TestCloudRouterInterfacesSourceGuardCancellationAndPreflight`, `connection_router_interface_records_test.go:TestConnectionRouterInterfacesClassifyCompletePortsWithConfiguredLocation`.
- Late physical pages at 200/203/401/404/429/503 preserve real response body/header/status and native pointer401 ErrOriginal VALUE receipt/ErrReauth, stop fault retry/backoff/reauth callbacks and prevent successful partial Ports/Value even for unknown interface_type.
  Tests: `network/cloud_router_interfaces_test.go:TestCloudRouterInterfacesLatePagesPreventPartialSuccessAndKeepPhysicalFaults`.

Python/Go differences:

- Go returns a raw JSON-domain router dictionary and concrete discriminator/header/version/location options plus a literal ID dictionary convenience; no user builder or extra router lookup is introduced. Malformed JSON and nil/unsafe sources fail as explicit Go errors.
- Go separates selected Ports/Value from actual partial Inventory/LastPort receipts with independent raw bytes/metadata and automatically configured Connection location. Unknown/nonstring owners are skipped instead of guessed; no fake interface Resource is synthesized.
- The existing frozen Port list transport retains its reviewed 200..399 status policy and service-relative request/paging/Link/cycle ownership. Python mutable Resource/dirty/session/cache and descriptor-laziness contracts remain SDK-R1/C1/S1 generic scope; stock controller read/field policies and authenticated live cloud acceptance remain separate.

### `python:cloud/_network/NetworkCloudMixin/remove_router_interface`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.RemoveInterfaceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfaceSubnet`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfacePort`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.RemoveRouterInterface`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceRemoveOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceRemoveSubnetID`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceRemovePortID`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterInterfaceRemoveHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.API.RemoveCloudInterfaceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers.WithRouterInterfaceOpts`

- RemoveInterfaceRecord sends PUT /v2.0/routers/{id}/remove_router_interface with subnet_id, or port_id when a port is given, and with neither it still sends subnet_id null.
  Tests: `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBRemoveRouterInterfaceBodies`.
- Concrete SDK options own absent/null/raw JSON and cached SDK Resource references. Both subnet_id and port_id falsey rejects before network service/router ID validation or HTTP. Python-loaded float-underflow/empty values are falsey; nonnil Resources remain truthy even with null/false/zero/empty IDs. Plain JSON dictionaries remain body values rather than being treated as Resources.
  Tests: `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveBothFalseyFirstAndHTTPZero`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemovePortFirstRawAndCachedResourceReferences`.
- Source performs no get_router/find/name resolution: strings are literal router IDs. A truthy port wins and causes only port_id to be sent, otherwise subnet_id. SDK-owned raw/PortRecord/SubnetRecord/RawResource conversion avoids application builders/per-record adapters and owns copied options, selected service/API/client/header state and immediately stopping callbacks after detected source changes.
  Tests: `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemovePortFirstRawAndCachedResourceReferences`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveOwnsOptionRefsHeadersAndEscapedState`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveCancellationNilOptionsAndStickySourceStopCallbacks`.
- Router.remove_interface evaluates actual response.json before cloud discards it and returns None. Successful Value is JSON null for arbitrary valid object/list/string/number/bool/null, while ResponseJSON/optional actual Wire/whole Envelope/header/status retain independent physical observations. Accepted empty204 or malformed JSON remains decode error; no synthetic router Resource/default or convergence claim is created.
  Tests: `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveSourceNoneAfterArbitraryJSONAndActualCopies`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveAcceptedDecodeAndCleanHTTPErrors`.
- A closed new helper reuses existing interface reference/body/header/JSON and shared REST contracts, opting into Source100..399 and StopFaultRetryCallbacks. Existing RemoveInterfaceRecord default-subnet-null/200..399 and native status/required behavior remain unchanged; 199 lane isolation is verified without promoting another Source API.
  Tests: `network/v2/extensions/layer3/routers/python_cloud_interface_remove_test.go:TestPythonCloudRouterInterfaceRemoveClosedSourceStatusAndLegacyIsolation`.
- Clean403/404/409 remains actual native error, not absence/success. Accepted and rejected physical read/Close/cancel/source faults preserve actual status/body/header and original causes with exactly one physical PUT/body Close and zero retry/backoff/reauth callbacks. Native401 ErrOriginal is explicitly inspected despite absent Unwrap, and ErrReauth preserves read/Close causes. Clean401 renewal and503 retry keep original request body/headers and actual final receipt.
  Tests: `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveAcceptedDecodeAndCleanHTTPErrors`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemovePhysicalFaultsKeepActualProofAndPreventReplay`, `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveCleanRetryAndTokenRenewalKeepCapturedBody`, `network/v2/extensions/layer3/routers/python_cloud_interface_remove_test.go:TestPythonCloudRouterInterfaceRemovePhysicalOptInNeverCallsHooks`.
- Only RemoveCloudInterfaceRecord normalizes explicit ISO8859-1/latin1, UTF16/32 with BOM or explicit LE/BE through the fixed shared responsejson.Normalize ABI. Parsed Value and optional object Wire.Body use normalized JSON; Envelope/Header/status/ResponseError.Body and Wire HTTP metadata preserve actual wire evidence. Unknown/malformed charset and bare UTF16 without BOM keep Unsupported cause; normalized malformed JSON retains SyntaxError and original bytes. Existing RemoveInterfaceRecord and generic routerJSONFromResponse charset/status/body defaults stay unchanged.
  Tests: `network/v2/extensions/layer3/routers/python_cloud_interface_remove_test.go:TestPythonCloudRouterInterfaceRemoveExplicitCharsetsKeepRawProofAndLegacy`, `network/v2/extensions/layer3/routers/python_cloud_interface_remove_test.go:TestPythonCloudRouterInterfaceRemoveCharsetFailuresKeepOriginalResponse`.
- The existing cloud Service exposes successful None as Value null after charset-aware parsing while diagnostic ResponseJSON and optional normalized object Wire preserve parsed fields; actual raw Envelope/Header/status remain independent. Unsupported charset does not expose successful Value/ResponseJSON and retains original accepted ResponseError proof.
  Tests: `network/cloud_router_interface_remove_test.go:TestCloudRouterInterfaceRemoveExplicitCharsetReceiptAndNone`.

Python/Go differences:

- Existing RemoveInterfaceRecord keeps its original parsed-JSON return. The separate Service.RemoveRouterInterface returns Source None as Value null with an SDK-owned diagnostic ResponseJSON and physical response receipt.
- This implements the named cloud remove_router_interface flow but remains unresolved due to the remaining fullRequests decoder/permissiveJSON gaps with SDK-owned validation/options/port-first selection/None return and actual response evidence, without a new Connection forwarding method. conn.Network(ctx) service construction may authenticate/discover before argument validation; direct selected/nil Service missing-ID validation remains HTTP0. New leaf helper is only this fixed dependency and no extra Source API is promoted; full SDK supported delta remains0.
- Go router input is a validated nonblank UTF8 string ID, escaping slash/percent/query as one segment and rejecting controls/traversal. Source arbitrary Router Resource/dict/null/nonstring dynamic dispatch and URI/class/member/session/cache behavior remain shared SDK-R1/C1/S1 foundation scope. No name lookup is invented, and broad Python runtime objects are not claimed from raw JSON preservation.
- Go returns an owned diagnostic result with Value:null for Source None plus actual parsed ResponseJSON/Wire/Envelope/Header/StatusCode rather than returning a bare nil. Failed observations never expose successful Value/ResponseJSON or a synthetic Router Resource. Valid JSON numeric tokens preserve Go raw precision; Python floating/runtime representation and wire key order remain explicit Go policies.
- The closed Source100..399/physical-fault callback suppression opt-in leaves existing native and legacy leaf policies unchanged. Headers are a concrete Go transport convenience with immutable selected source/client/captured framing and live token renewal. Stock Neutron27 project ADMIN_OR_PROJECT_MEMBER gate is documented without inferring parent-owner reader/shared-subnet authorization or live ACL acceptance.
- Evidence is actual CPython3.14.8 pinned cloud/Proxy/Resource/Router code with Adapter.put/_get_connection fixtures (75 observations), plus independently audited bounded router13 observations. Source/AST, Go/race, main compilation and live backend authorization are separate claims; no live backend was used.
- Cycle12 changes only the fixed cloud dependency to reuse internal/responsejson.Normalize for bounded explicit charsets. The generic/legacy decoder and common ABI/codec set remain unchanged. BOM is required for bare UTF16/32; explicit LE/BE works without it. Actual raw proof stays separate from normalized diagnostic JSON. Reused Source75 and independent13 observations are not rerun or combined into a new count. Worker ran gofmt only; root owns Go/race/main builds; no mapping promotion is claimed.

Unresolved boundaries:

- Full Requests Response.json codec registry, unknown explicit charsets, charset-free BOM/guess_json_utf/text fallback and bare UTF16/UTF32 without BOM native-endian decoding remain unsupported. The closed cloud helper now supports ISO8859-1/latin1 and UTF16/32 with BOM or explicit LE/BE. Unsupported/malformed declared charset retains ErrUnsupported plus original response evidence, while unmarked nonUTF8 preserves the original decode failure. This narrower reachable decoder gap still prevents complete named mapping.
- Full permissive Python JSON loading and runtime numeric domain remain unmatched. Source Response.json may accept NaN/Infinity-style values before cloud discards the result and returns None, while the reused Go JSON carrier rejects nonstandard JSON tokens. Finite JSON raw numbers retain Go literal precision rather than claiming all Python loaded-value semantics.

### `python:cloud/_network/NetworkCloudMixin/update_router`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateRouterRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithRouterUpdateOpts`

- SDK-owned six cloud formals distinguish truthy name/gateway network/fixed IPs from nonnull admin state/SNAT/routes, reuse library gateway assembly without creation admin defaults or merging existing gateway state, skip all HTTP when no effective args exist, and commit only raw dirty fields after member/name lookup. No-op and native mutation result remain distinct from lookup evidence.
  Tests: `network/cloud_router_update_test.go:TestCloudRouterUpdateEffectiveValuesAndDirtySelection`, `network/cloud_router_update_test.go:TestCloudRouterUpdateIneffectiveBeforeServiceLookup`, `network/cloud_router_update_test.go:TestCloudRouterUpdateNameFallbackAndMissing`.
- Nonnull routes including [] and false perform fresh complete extraroute discovery before lookup; missing extension drops routes only and can return lookup-free no-op, while other attrs continue. HTTP/late malformed alias/continuation failures never become unsupported-route fallback. Owned options/preflight and canceled-context guards prevent unwanted HTTP; native conflict preserves lookup and no success model.
  Tests: `network/cloud_router_update_test.go:TestCloudRouterUpdateRoutesGateAndDrop`, `network/cloud_router_update_test.go:TestCloudRouterUpdateExtensionFailureCannotBecomeDroppedRoutes`, `network/cloud_router_update_test.go:TestCloudRouterUpdateOwnedOptionsAndPreflight`, `network/cloud_router_update_test.go:TestCloudRouterUpdateOptionCancellationCannotBecomeNoOp`, `network/cloud_router_update_test.go:TestCloudRouterUpdateConflictKeepsLookupAndSingleAttempt`.
- An accepted PUT (any status 200-399, as Python raise_from_response) returns Response, an owned UpdateReceipt with the actual body bytes, headers and status. Body follows Python Resource._translate_response: the resource-key member when present, otherwise the whole top-level object, without self and with unknown members preserved. Merged is the lookup wire row overlaid with every effective attribute and then Body, so a partial body keeps lookup fields; an empty, 204, truncated or otherwise unparsable body sets BodyIgnored and keeps that pre-commit state without error. Non-object JSON or a non-object resource-key member fails after remote success with the receipt and no Merged/native model. A rejected PUT has no receipt, and nothing is replayed. The native model is the projection of Merged.
  Tests: `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptMergesPythonTranslateResponse`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptNonObjectBodiesFailAfterRemoteSuccess`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptRejectedPutHasNoReceipt`.
- Lookup.SourceBody is the pinned Python Router body-attribute state: an attribute-named member such as is_admin_state_up is stored under admin_state_up, a wire/canonical collision keeps CPython dict order (first position, last value) and Router's own revision_number declaration maps a Neutron revision_number member to the revision wire key. Supplied values compare against that state, so a canonical-only equal value is a no-op without PUT, and the accepted response is merged through the same mapping while Body keeps the actual member names.
  Tests: `network/cloud_update_source_state_test.go:TestCloudUpdateSourceStateCanonicalLookupDirtyComparison`, `network/cloud_update_source_state_test.go:TestCloudUpdateSourceStateRouterRevisionAndSubnetAlias`, `internal/neutronfind/source_state_test.go:TestSourceStateConsumeAttrsMapping`.

Python/Go differences:

- UpdateRouterRecord owns formal defaults and optional extraroute handling. RoutesDropped and Extensions expose omission/discovery without adding Source logger behavior. Lookup is an owned record, Response is the owned PUT receipt and Router is the native projection of its merged view; none is the same mutable Python Resource. Client capture follows effective-option selection and retains configured native provider hooks.
- Lookup dirty comparison and receipt merge use the pinned Python body-attribute state (SourceBody/TranslateSourceBody): Resource._consume_attrs over CPython dict order maps attribute-named members to their wire keys, with translate rules for member GET and PUT responses and constructor rules for fallback LIST rows. The native model omits Merged members its schema cannot hold and reports them in UpdateReceipt.ProjectionError without failing the accepted update; Merged keeps the raw values Python would keep. NaN/Infinity bodies are invalid JSON to Go and take the BodyIgnored branch where Python would merge them (shared nonfinite JSON boundary in docs/sdk-support-ledger.md).

Unresolved boundaries:

- Arbitrary Python kwargs/object/numeric types, full Requests encoding/nonfinite JSON and SDK-R1/C1/S1 Resource/adapter/session/cache lifecycle remain separate runtime boundaries.

### `python:cloud/_network/NetworkCloudMixin/update_subnet`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.API.UpdateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/subnets.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateSubnetRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSubnetUpdateOpts`

- With attributes, FindRecord with ignore_missing false followed by UpdateWithAttributes sends the lookup GET and a PUT whose body carries name, enable_dhcp and gateway_ip null for disable_gateway_ip.
  Tests: `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBUpdatePortAndSubnetChains`.
- SDK-owned seven formal kwargs use Source nonnull enable_dhcp versus truthiness for the remaining inputs, rename subnet_name to name, turn a truthy disable_gateway_ip into gateway_ip null, reject truthy gateway conflict before lookup, and skip service/identity validation and all HTTP when the effective body is empty. Go cancellation cannot be hidden by that no-op.
  Tests: `network/cloud_subnet_update_test.go:TestCloudSubnetUpdateEffectiveValuesAndDirtySelection`, `network/cloud_subnet_update_test.go:TestCloudSubnetUpdateIneffectiveAndConflictBeforeServiceLookup`, `network/cloud_subnet_update_test.go:TestCloudSubnetUpdateOptionCancellationCannotBecomeNoOp`.
- Effective attrs resolve member/name lookup and compare ordinary raw wire values with Python equality, skip equal-value PUT and commit only changed fields to the resolved ID. Options snapshot JSON, reject unsupported kwargs before HTTP, retain lookup on native conflict without fabricated success and preserve configured provider hooks.
  Tests: `network/cloud_subnet_update_test.go:TestCloudSubnetUpdateNameFallbackAndMissing`, `network/cloud_subnet_update_test.go:TestCloudSubnetUpdateOwnedOptionsAndPreflight`, `network/cloud_subnet_update_test.go:TestCloudSubnetUpdateConflictKeepsLookupAndSingleAttempt`.
- An accepted PUT (any status 200-399, as Python raise_from_response) returns Response, an owned UpdateReceipt with the actual body bytes, headers and status. Body follows Python Resource._translate_response: the resource-key member when present, otherwise the whole top-level object, without self and with unknown members preserved. Merged is the lookup wire row overlaid with every effective attribute and then Body, so a partial body keeps lookup fields; an empty, 204, truncated or otherwise unparsable body sets BodyIgnored and keeps that pre-commit state without error. Non-object JSON or a non-object resource-key member fails after remote success with the receipt and no Merged/native model. A rejected PUT has no receipt, and nothing is replayed. The native model is the projection of Merged.
  Tests: `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptMergesPythonTranslateResponse`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptNonObjectBodiesFailAfterRemoteSuccess`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptRejectedPutHasNoReceipt`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptProjectionKeepsMergedAuthority`.
- Lookup.SourceBody is the pinned Python Subnet body-attribute state: a canonical-only is_dhcp_enabled member is stored under enable_dhcp, so an equal enable_dhcp update is a no-op without PUT; collisions keep CPython dict order and the accepted response is merged through the same mapping while Body keeps the actual member names.
  Tests: `network/cloud_update_source_state_test.go:TestCloudUpdateSourceStateRouterRevisionAndSubnetAlias`, `internal/neutronfind/source_state_test.go:TestSourceStateConsumeAttrsMapping`.

Python/Go differences:

- Go returns the native Subnet projection plus the owned UpdateReceipt instead of a Subnet Resource.
- UpdateSubnetRecord owns argument defaults and effective-body selection. NoOp with nil Lookup corresponds to Source None; NoOp with Lookup corresponds to equal raw fields after lookup. Native Subnet is only the mutation result and is not a shared mutable Python Resource. Client capture begins after effective-argument selection; direct option-time client-internal mutation is outside the before-options guard contract.
- Lookup dirty comparison and receipt merge use the pinned Python body-attribute state (SourceBody/TranslateSourceBody): Resource._consume_attrs over CPython dict order maps attribute-named members to their wire keys, with translate rules for member GET and PUT responses and constructor rules for fallback LIST rows. The native model omits Merged members its schema cannot hold and reports them in UpdateReceipt.ProjectionError without failing the accepted update; Merged keeps the raw values Python would keep. NaN/Infinity bodies are invalid JSON to Go and take the BodyIgnored branch where Python would merge them (shared nonfinite JSON boundary in docs/sdk-support-ledger.md).

Unresolved boundaries:

- Arbitrary Python kwargs/object/numeric type comparison, full Requests encoding/nonfinite JSON and SDK-R1/C1/S1 Resource/adapter/session/cache lifecycle remain separate runtime boundaries.

### `python:cloud/_network/NetworkCloudMixin/update_port`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithRecordFindIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.API.UpdateWithAttributes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/ports.WithAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdatePortRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithPortUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithPortUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithPortUpdateOpts`

- With attributes, FindRecord with ignore_missing false (member GET, then name list) followed by UpdateWithAttributes sends Python's lookup and PUT body, and a missing port fails before any PUT.
  Tests: `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBUpdatePortAndSubnetChains`.
- SDK-owned cloud helper snapshots strict thirteen-key kwargs, validates before lookup, always resolves even empty kwargs, skips PUT for no attributes or Python-equal raw wire values, and PUTs only changed fields to the resolved response ID. Explicit null/false/empty collections remain distinct from omission; raw string boolean values are not replaced by descriptor getter views.
  Tests: `network/cloud_port_update_test.go:TestCloudPortUpdateLookupOnlyAndDirtySelection`, `network/cloud_port_update_test.go:TestCloudPortUpdateNameFallbackAndMissing`.
- Options own caller JSON, unsupported canonical aliases/revision/extensions and malformed inputs fail before HTTP; canceled context or selected-service mutation cannot send lookup/PUT. A default-client conflict retains a separate lookup record and makes one PUT attempt without fabricated native success. No new mutation retry policy is added; configured provider retry/reauthentication hooks remain in force.
  Tests: `network/cloud_port_update_test.go:TestCloudPortUpdateOwnedOptionsAndPreflight`, `network/cloud_port_update_test.go:TestCloudPortUpdateConflictKeepsLookupAndSingleAttempt`.
- An accepted PUT (any status 200-399, as Python raise_from_response) returns Response, an owned UpdateReceipt with the actual body bytes, headers and status. Body follows Python Resource._translate_response: the resource-key member when present, otherwise the whole top-level object, without self and with unknown members preserved. Merged is the lookup wire row overlaid with every effective attribute and then Body, so a partial body keeps lookup fields; an empty, 204, truncated or otherwise unparsable body sets BodyIgnored and keeps that pre-commit state without error. Non-object JSON or a non-object resource-key member fails after remote success with the receipt and no Merged/native model. A rejected PUT has no receipt, and nothing is replayed. The native model is the projection of Merged.
  Tests: `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptMergesPythonTranslateResponse`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptNonObjectBodiesFailAfterRemoteSuccess`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptRejectedPutHasNoReceipt`.
- A fallback LIST row named by attribute names (is_port_security_enabled, is_admin_state_up, binding_host_id) is stored under its wire keys by the Resource.existing constructor rules, without self and with unknown members kept. Only values that differ from that state are sent, and the accepted response's attribute-named members are merged under their wire keys (Merged) while Body keeps the actual names.
  Tests: `network/cloud_update_source_state_test.go:TestCloudUpdateSourceStateListRowAndResponseMapping`, `internal/neutronfind/source_state_test.go:TestSourceStateConsumeAttrsMapping`.

Python/Go differences:

- Python raises SDKException for a missing port; Go FindRecord returns its own not-found error.
- Go returns the native Port projection, which omits extension attributes; Response.Merged keeps them with every other wire member.
- Cloud UpdatePortRecord returns a separate Lookup PortRecord, the PUT receipt and its native Port projection; NoOp has Lookup and no fabricated Port or receipt. This callable JSON profile owns defaults/options and does not return the same Python mutable Resource instance.
- Lookup dirty comparison and receipt merge use the pinned Python body-attribute state (SourceBody/TranslateSourceBody): Resource._consume_attrs over CPython dict order maps attribute-named members to their wire keys, with translate rules for member GET and PUT responses and constructor rules for fallback LIST rows. The native model omits Merged members its schema cannot hold and reports them in UpdateReceipt.ProjectionError without failing the accepted update; Merged keeps the raw values Python would keep. NaN/Infinity bodies are invalid JSON to Go and take the BodyIgnored branch where Python would merge them (shared nonfinite JSON boundary in docs/sdk-support-ledger.md).

Unresolved boundaries:

- Original arbitrary Python kwargs numeric/object types remain outside this bounded JSON profile. Full Resource lifecycle, adapter/session/cache policy and SDK-R1/C1/S1 remain separate. Native configured retry/auth hooks keep existing behavior; this helper does not establish universal mutation no-replay guarantees.

### `python:cloud/_network/NetworkCloudMixin/list_qos_policies`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.ListQoSPolicyRecords`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithOperationGuard`

- Every call collects the complete uncached Source extension alias inventory before policy reads, including later pages after an early qos match. Missing qos returns ErrUnavailableExtension with completed extension evidence; late gate permission/malformed failures remain terminal rather than cached/early-success.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyListAndSearchGateEveryCallWithoutEarlyExit`.
- Cloud filters remain owned JSON snapshots and all loaded-Python falsey values become an empty kwargs mapping after the extension gate. Truthy list nonmapping filters fail after the gate before policy HTTP. No _filter_list, glob, local ID/name search or post-server name predicate is invented.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyFiltersFalseyAndMappingErrorsHappenAfterGate`, `network/cloud_qos_policies_test.go:TestCloudQoSPolicySearchLiteralNameAndOrderedUpdateAreNotLocalSearch`.
- Existing QoSPolicy declared server aliases and tenant_id/rules local filters are owned by the library. Source Proxy lane consumes raw headers/microversion/service-relative base_path, ordered conflicting attrs and original JMES separately; raw cap precedes local filtering and malformed/bound controls retain pre-policy failures.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyRawFormalsAliasesAndLocalJSONAreOwned`, `network/v2/extensions/qos/policies/python_records_proxy_contracts_test.go:TestQoSPolicyProxyListRawFormalsGeneratorAndOpaqueJSON`.
- Actual Source cloud list(Proxy) consumes an opaque generator or iterates arbitrary array/string/dictionary result, with null/number/type errors occurring after extension discovery without policy HTTP. Nested generators preserve identity/zero policy HTTP and JSON remains absent. Dictionary keys have the explicitly reviewed Go lexical order; [z,a] Source is [a,z] Go, never claimed to match Source insertion order.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyProxyIterableAndOpaqueResultsPreserveHTTPPhases`, `network/v2/extensions/qos/policies/python_records_proxy_contracts_test.go:TestQoSPolicyProxyListRawFormalsGeneratorAndOpaqueJSON`.
- New Source Proxy profile opts into root-link precedence, CPython-loaded equality and physical fault stop without changing typed List/Get baseline. BasePath pointers/source/options are cloned and cloud operation guard remains active through leaf callbacks/retry/physical handling. Concurrent reusable options, late failures and physical read/Close/native401 faults preserve completed extension/actual policy evidence and discard partial Policies/Value/JSON; clean503 retries remain allowed and changed selected route retains original rejected503.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyReadOptionsSnapshotsCallbacksAndLateFailures`, `network/cloud_qos_policies_test.go:TestCloudQoSPolicyPhysicalFaultsAndCleanRetryKeepActualReceipt`, `network/v2/extensions/qos/policies/python_records_proxy_contracts_test.go:TestQoSPolicyProxySourceLinksLoadedEqualityAndGuardAreOptIn`, `network/v2/extensions/qos/policies/python_records_proxy_contracts_test.go:TestQoSPolicyProxyOptsSnapshotsFormalErrorsAndTypedBasePath`.

Python/Go differences:

- The concrete Service helper returns a QoSPolicyRecords carrier: ordinary PolicyRecord Resource views plus independent actual Wire/Envelope/Header/status, completed Extensions evidence and arbitrary list Values/JSON. A failed operation keeps completed extension evidence and no partial policy values instead of returning a Python exception with no result.
- Go uses a concrete string search identity and JSON-domain filters/options. Mutable Python Resource/class/session/adapter/cache/Connection-derived location and custom objects remain shared SDK foundations; the current owned location is null. Options are library-owned snapshots, and no caller filter builder is needed.
- Source cloud list(JMES dictionary) iterates insertion-order keys; the existing shared Go JMES map result is iterated in deterministic lexical order. The explicit z,a literal yields z,a in Source and a,z in Go. Arrays, Unicode strings, opaque generators and noniterable/type failures retain their actual phases.
- Safe string Header maps, same-origin service-relative fixed paths/query/cycle guards, nonnegative Go raw caps and strict complete UTF8/JSON are explicit Go policies. Some lazy Source descriptor/Resource.list formal errors are validated eagerly in owned options. Unknown actual body and large integers remain raw.
- Only the new Proxy/cloud lane uses Source link precedence/loaded equality/physical-fault policy. Existing typed Policies List/Get query reservations, dictionary/HTTP Link interoperability and numeric equality are preserved. Source Proxy receives the unconsumed generator, never an invented materialized resource array.
- Fixed Neutron27 get_policy permits project reader for owned policies or shared policy (and admin) in project scope. Source/fake HTTP/race/main checks do not assert authenticated deployment success, enabled QoS drivers or elevated write/admin permissions.

### `python:cloud/_network/NetworkCloudMixin/search_qos_policies`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.SearchQoSPolicyRecords`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.WithOperationGuard`

- Every call collects the complete uncached Source extension alias inventory before policy reads, including later pages after an early qos match. Missing qos returns ErrUnavailableExtension with completed extension evidence; late gate permission/malformed failures remain terminal rather than cached/early-success.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyListAndSearchGateEveryCallWithoutEarlyExit`.
- Cloud filters remain owned JSON snapshots and all loaded-Python falsey values become an empty kwargs mapping after the extension gate. Truthy list nonmapping filters fail after the gate before policy HTTP. No _filter_list, glob, local ID/name search or post-server name predicate is invented.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyFiltersFalseyAndMappingErrorsHappenAfterGate`, `network/cloud_qos_policies_test.go:TestCloudQoSPolicySearchLiteralNameAndOrderedUpdateAreNotLocalSearch`.
- Search seeds a truthy literal identity as server name then applies Source dict.update in insertion order. Mapping values replace the original slot, including name null/override; JSON iterable pairs include Unicode string and dictionary-key pairs, final kwargs keys must be strings and invalid pair shapes fail after completed gate before policy HTTP.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicySearchLiteralNameAndOrderedUpdateAreNotLocalSearch`.
- Existing QoSPolicy declared server aliases and tenant_id/rules local filters are owned by the library. Source Proxy lane consumes raw headers/microversion/service-relative base_path, ordered conflicting attrs and original JMES separately; raw cap precedes local filtering and malformed/bound controls retain pre-policy failures.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyRawFormalsAliasesAndLocalJSONAreOwned`, `network/v2/extensions/qos/policies/python_records_proxy_contracts_test.go:TestQoSPolicyProxyListRawFormalsGeneratorAndOpaqueJSON`.
- Actual Source cloud list(Proxy) consumes an opaque generator or iterates arbitrary array/string/dictionary result, with null/number/type errors occurring after extension discovery without policy HTTP. Nested generators preserve identity/zero policy HTTP and JSON remains absent. Dictionary keys have the explicitly reviewed Go lexical order; [z,a] Source is [a,z] Go, never claimed to match Source insertion order.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyProxyIterableAndOpaqueResultsPreserveHTTPPhases`, `network/v2/extensions/qos/policies/python_records_proxy_contracts_test.go:TestQoSPolicyProxyListRawFormalsGeneratorAndOpaqueJSON`.
- New Source Proxy profile opts into root-link precedence, CPython-loaded equality and physical fault stop without changing typed List/Get baseline. BasePath pointers/source/options are cloned and cloud operation guard remains active through leaf callbacks/retry/physical handling. Concurrent reusable options, late failures and physical read/Close/native401 faults preserve completed extension/actual policy evidence and discard partial Policies/Value/JSON; clean503 retries remain allowed and changed selected route retains original rejected503.
  Tests: `network/cloud_qos_policies_test.go:TestCloudQoSPolicyReadOptionsSnapshotsCallbacksAndLateFailures`, `network/cloud_qos_policies_test.go:TestCloudQoSPolicyPhysicalFaultsAndCleanRetryKeepActualReceipt`, `network/v2/extensions/qos/policies/python_records_proxy_contracts_test.go:TestQoSPolicyProxySourceLinksLoadedEqualityAndGuardAreOptIn`, `network/v2/extensions/qos/policies/python_records_proxy_contracts_test.go:TestQoSPolicyProxyOptsSnapshotsFormalErrorsAndTypedBasePath`.

Python/Go differences:

- The concrete Service helper returns a QoSPolicyRecords carrier: ordinary PolicyRecord Resource views plus independent actual Wire/Envelope/Header/status, completed Extensions evidence and arbitrary list Values/JSON. A failed operation keeps completed extension evidence and no partial policy values instead of returning a Python exception with no result.
- Go uses a concrete string search identity and JSON-domain filters/options. Mutable Python Resource/class/session/adapter/cache/Connection-derived location and custom objects remain shared SDK foundations; the current owned location is null. Options are library-owned snapshots, and no caller filter builder is needed.
- Source cloud list(JMES dictionary) iterates insertion-order keys; the existing shared Go JMES map result is iterated in deterministic lexical order. The explicit z,a literal yields z,a in Source and a,z in Go. Arrays, Unicode strings, opaque generators and noniterable/type failures retain their actual phases.
- Safe string Header maps, same-origin service-relative fixed paths/query/cycle guards, nonnegative Go raw caps and strict complete UTF8/JSON are explicit Go policies. Some lazy Source descriptor/Resource.list formal errors are validated eagerly in owned options. Unknown actual body and large integers remain raw.
- Only the new Proxy/cloud lane uses Source link precedence/loaded equality/physical-fault policy. Existing typed Policies List/Get query reservations, dictionary/HTTP Link interoperability and numeric equality are preserved. Source Proxy receives the unconsumed generator, never an invented materialized resource array.
- Fixed Neutron27 get_policy permits project reader for owned policies or shared policy (and admin) in project scope. Source/fake HTTP/race/main checks do not assert authenticated deployment success, enabled QoS drivers or elevated write/admin permissions.

### `python:cloud/_network/NetworkCloudMixin/list_qos_rule_types`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.ListQoSRuleTypeValues`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleTypeListFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleTypeListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithProxyListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes.WithRecordListBasePath`

- Every call reads the complete fresh qos alias inventory before kwargs/callbacks; missing, late HTTP failure or unhashable aliases stop the rule list and preserve actual observed Extension records. Clean HTTP403 preserves the native ErrUnexpectedResponseCode value with actual Method/URL/Body/ResponseHeader; accepted200 alias-projection faults retain ResponseError.
  Tests: `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesFreshCompleteGateAndMissingShortCircuit`, `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesGateReadsPastFirstMatchAndKeepsActualFailure`.
- Source falsey JSON and binary64 underflow normalize to empty kwargs; truthy nonobjects fail after the extension gate. Original ordered JSON kwargs, header/version/path formals and one conflicting-attrs pass are library-owned.
  Tests: `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesFalseyFiltersAndTruthyShapeAfterGate`, `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesOriginalKwargsOwnedHeadersVersionPathAndConflictMerge`, `network/v2/extensions/qos/ruletypes/python_proxy_list_test.go:TestQoSRuleTypeProxyOriginalExpressionWinsAfterOneConflictPass`.
- Generator JMESPath precedes list() conversion; literal arrays/objects/strings and nested generators remain polymorphic without fabricated RuleType or HTTP metadata; unused transport formals remain lazy.
  Tests: `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesGeneratorJMESPathAndPolymorphicListConversion`, `network/v2/extensions/qos/ruletypes/python_proxy_list_test.go:TestQoSRuleTypeProxySourceFormalsStayLazyBehindConstantExpressions`, `network/v2/extensions/qos/ruletypes/python_proxy_list_test.go:TestQoSRuleTypeProxyGuardsObserveEveryOptionAndOpaqueIdentity`.
- Successful empty lists are nonnil; nullable aliases, drivers descriptors and unknown extensions retain separate owned Resource/Wire and actual page proof; late errors retain Inventory without success Values/RuleTypes/JSON. Late clean403 remains a native value error with the actual rejected page status/header/body/URL.
  Tests: `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesEmptyDefaultsAliasesAndWireOwnership`, `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesLatePageFailureRetainsInventoryWithoutSuccessValues`.
- Every callback sees a clone and a before/after source check; cancellation, source changes, native401 physical Read/Close faults retain actual native original proof and stop faulty retry/reauth hooks.
  Tests: `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesOptionGuardsStopRestoreAndCancellationCallbacks`, `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesPhysical401PreservesActualNativeFieldsAndNoHooks`, `network/v2/extensions/qos/ruletypes/python_proxy_list_test.go:TestQoSRuleTypeProxyUnknownValuesAndCallbackMapsAreOwned`.
- Existing typed raw reserved-key rejection and 200..399 admission stay unchanged; only the explicit typed BasePath and separate Proxy lanes broaden options.
  Tests: `network/cloud_qos_rule_types_test.go:TestCloudQoSRuleTypesOldStatusAndTypedRawRejectionRemainUnchanged`, `network/v2/extensions/qos/ruletypes/python_proxy_list_test.go:TestQoSRuleTypeProxyOwnsRawFormalsAndTypedBasePathKeepsOldRawRejection`.

Python/Go differences:

- Go sorts object keys lexically when list() consumes a JMESPath object, while Python preserves insertion order; arrays and Unicode character ordering are preserved.
- Existing rule type leaf keeps 200..399 versus Source synthetic100..199, nonnegative Go-integer max_items, exact-decimal id/name local equality versus Python loaded float rounding/underflow, and explicit safe origin/path/header/UTF8/continuation policy. These are documented Go policies; they are not full runtime parity.
- Concrete owned JSON/client/context/options do not provide arbitrary Python class/session/operator replacement or mutable Resource/cloud configuration/cache lifecycle. Whole-project SDK-R1 remains separate; no live deployment support or whole SDK supported promotion is claimed.

### `python:cloud/_network/NetworkCloudMixin/update_qos_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateQoSPolicyRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindCloudMutationRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.UpdateCloudRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.PolicyRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyMutationJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyMutationOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyName`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyDescription`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyShared`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyDefault`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSPolicyProjectID`

- Cloud valid_kwargs name/description/shared/default/project_id are validated before any extension HTTP. Complete JSON null/presence and concrete options are library-owned, nil/unknown cloud keys including is_shared/is_default are rejected with HTTP0 and later options overwrite prior keys from owned snapshots.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationValidKwargsBeforeGateAndOwnedOptions`.
- Every valid call consumes a fresh full qos extension discovery; every nonnull default including false/0/[]/{} triggers an independent fresh full qos-default gate. Supported default maps raw to is_default; unavailable default is removed without hiding gate errors, with completed Extensions evidence and DefaultDropped separately retained.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyCreateDefaultNonnullFreshGatesAndEmptyPOST`, `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyUpdateEmptyAfterDefaultSkipsLookupAndMissingRaises`.
- New cloud Source mutation lane uses selected original service/API/source and header snapshots captured before option callbacks. Source100..399 completion preserves status199. Clean503 retry and401 token renewal remain possible; changed route aborts renewal/retry with original rejected evidence. Prior typed/native profiles are unchanged.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationSourceStatusPolicyCleanRetryAndImmutableHeaders`, `network/v2/extensions/qos/policies/python_cloud_mutations_test.go:TestPolicyCloudMutationSelectedVersionAndEqualUpdateReset`.
- POST/PUT rejected400/403/404/409/503 retain actual status/body/header and native value ErrUnexpectedResponseCode rather than fabricating Resource/Wire or swallowing404. Accepted/rejected200/401/404/503 read+Close/cancel/source fault matrix retains actual physical evidence and causes with zero retry/reauth callbacks. Native401 retains pointer ErrUnableToReauthenticate.ErrOriginal (value native response proof) and ErrReauth despite native absence of Unwrap.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationRejectedStatusPreservesActualEvidence`, `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyMutationPhysicalFaultsPreserveActualEvidenceWithoutHooks`.
- After default handling, empty kwargs returns Source None without Find/PUT (including default:null and unavailable nonnull default). Nonempty attrs use default ignore_missing=True Source Find; absent policy raises SDK not-found with no PUT, retaining completed extension/lookup stage evidence.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyUpdateEmptyAfterDefaultSkipsLookupAndMissingRaises`.
- Dirty comparison uses private raw Source Body presence rather than coerced/default/alias getters: missing name→null is dirty; tenant_id-only→same project_id is dirty; shared:"false"→true is dirty; shared:true→1 is equal. Only changed keys enter PUT, partial response seed retains old/new raw state, and equal update returns original private receipt without invented PUT proof.
  Tests: `network/cloud_qos_policy_mutations_test.go:TestCloudQoSPolicyUpdateRawDirtyMissingAliasAndOnlyChangedPayload`, `network/v2/extensions/qos/policies/python_cloud_mutations_test.go:TestPolicyCloudMutationListConstructorRawLedgerAndPartialResponseSeed`.
- GET response raw Body and LIST constructor canonical/wire consumption have separate private ledgers. Borrowed seed is bound to original API/client/source guard and cloned private evidence; caller mutations to exported Resource/Wire/Header/Envelope or Clone cannot create false dirtiness or forge transport provenance. Another API or changed original route rejects before PUT.
  Tests: `network/v2/extensions/qos/policies/python_cloud_mutations_test.go:TestPolicyCloudMutationBorrowedLedgerIgnoresExportedTamperingAndBindsOwner`, `network/v2/extensions/qos/policies/python_cloud_mutations_test.go:TestPolicyCloudMutationListConstructorRawLedgerAndPartialResponseSeed`.
- Source _update resets microversion to None when no dirty PUT occurs; successful dirty PUT restores selected commit version. Actual unknown response microversion stays in Wire and cannot shadow Resource state.
  Tests: `network/v2/extensions/qos/policies/python_cloud_mutations_test.go:TestPolicyCloudMutationSelectedVersionAndEqualUpdateReset`.

Python/Go differences:

- Concrete Go result preserves Policy, Lookup, Extensions, NoOp and DefaultDropped rather than Python Resource/None only. Wire/Envelope/Header/StatusCode remain actual observed response evidence; a prepared seed/no-op is not represented as a server PUT response.
- The new Source mutation lane preserves legacy typed/native defaults and policies. Complete UTF8 JSON/concrete owned options cover the named kwargs domain; custom runtime objects and mutable Python Resource/class/adapter/session/cache/current_location remain SDK-R1/SDK-C1/SDK-S1 shared foundation boundaries, with supported increment 0 and no generic full-SDK claim.
- Go source-relative URI guards escape returned member identity as one segment. Null/false/0/underflow/empty dirty member ID is rejected before PUT; Python null raises InvalidRequest but falsey nonnull URL joining can target a collection path. Go does not issue collection PUT from such a member seed. Equal updates do not validate or require a nonnull ID before returning no-op.
- JSON object wire order is not guaranteed. Source constructor conversion reads raw row insertion order, which is retained for declared/wire alias ledger conversion; dirty payload contains only changed keys and Python-loaded numeric equality preserves bool/int equality and CPython floating-point behavior without converting arbitrary integers through float64.
- Stock Neutron27 create_policy/update_policy are project ADMIN_OR_PROJECT_MANAGER, distinct from deprecated ADMIN_ONLY. Project_id field/cross-project privileges remain independent server validation (fixed neutron-lib3.21.0 reference); no mock/race/Source/main result proves authenticated live deployment/backend/driver acceptance.

### `python:cloud/_network/NetworkCloudMixin/list_qos_bandwidth_limit_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.ListQoSBandwidthLimitRules`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ProxyBandwidthLimitRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitProxyQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitListBasePath`

- Complete QoS extension gate then exactly one parent policy resolution, owned untyped defaults/unknown fields/computed-null location, bound URI parent and local Body filtering with shared actual paging.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthListOwnsGateParentPagingAndRawFields`.
- Extension gate consumes every alias including later unhashable alias or failed page after an early QoS match; absence has typed unavailable extension error and no policy/callback/rule request.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthGateConsumesAllExtensionsAndStopsBeforePolicy`.
- Policy missing and ambiguity prevent rules/callback/filter expansion, preserve real extension inventory and typed missing/ambiguity without inventing policy None HTTP evidence.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthPolicyMissingAndAmbiguityRetainActualGate`.
- Source proxy generator JMESPath literals/Unicode/object keys/nested generator/null/type errors distinguish rule HTTP from completed prerequisite HTTP; owned base_path succeeds while existing typed raw base_path rejection stays unchanged.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthSourceProxyLiteralsBasePathAndOldTypedBoundary`.
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

### `python:cloud/_network/NetworkCloudMixin/search_qos_bandwidth_limit_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.SearchQoSBandwidthLimitRules`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSBandwidthLimitMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ProxyBandwidthLimitRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitProxyQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithBandwidthLimitListBasePath`

- Complete QoS extension gate then exactly one parent policy resolution, owned untyped defaults/unknown fields/computed-null location, bound URI parent and local Body filtering with shared actual paging.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthListOwnsGateParentPagingAndRawFields`.
- Search delegates exactly one complete list workflow, preserves identifier/glob selection and CPython loaded bool/integer local equality, applies original ordered dictionary filter after list and retains actual Inventory/proof on unknown-key failure.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthSearchKeepsRuleIDPhaseAndFilterPresence`.
- Extension gate consumes every alias including later unhashable alias or failed page after an early QoS match; absence has typed unavailable extension error and no policy/callback/rule request.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthGateConsumesAllExtensionsAndStopsBeforePolicy`.
- Policy missing and ambiguity prevent rules/callback/filter expansion, preserve real extension inventory and typed missing/ambiguity without inventing policy None HTTP evidence.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthPolicyMissingAndAmbiguityRetainActualGate`.
- Source proxy generator JMESPath literals/Unicode/object keys/nested generator/null/type errors distinguish rule HTTP from completed prerequisite HTTP; owned base_path succeeds while existing typed raw base_path rejection stays unchanged.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthSourceProxyLiteralsBasePathAndOldTypedBoundary`.
- CPython falsey filters, truthy nonobject pre-rule errors, initialized owned callback maps, joined option error/cancellation or original captured-source mutation and real prerequisite receipts.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthFilterOwnershipCallbackGuardAndFalseyCases`.
- Actual truncated401 rule response keeps native reauthentication pointer/UnexpectedResponseCode/body/header/EOF, stops willing retry/reauth callbacks; late failed page preserves only earlier actual Inventory and completed parent/extensions.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthPhysical401RulePageAndLateFailurePreserveStageProof`.
- Source search without identifier skips falsey literal values; nested generator membership consumes actual rule records before unknown-key failure, while identifier phase rejects opaque generator without rule HTTP.
  Tests: `network/cloud_qos_bandwidth_test.go:TestCloudQoSBandwidthSearchSourceFalseyLiteralsAndNestedMembership`.

Python/Go differences:

- Concrete JSON/context/captured-client Go APIs and protected route/header policy replace dynamic Python session/Resource class/constructor/callable substitutions; arbitrary Python-only objects are not accepted.
- SourceProxy lane accepts100..399 with Source loaded local equality and computed location:null. Existing typed list/Get record status200..399 and typed raw base_path rejection remain compatibility APIs; concrete typed BasePath is additive.
- JMESPath literal object key iteration uses Go lexical order vs Python insertion order; existing strict syntax/exact JSON numeric evaluator/custom Resource function type policies remain explicit differences.
- Literal and nested generator values never acquire fake BandwidthLimitRecord or HTTP receipts. Actual extension/policy/rule proofs and observed partial inventory are retained with Go errors.
- Standalone Service computed location is Source null, including ignoring raw location in logical view. Full Connection computed location and mutable Resource/session model are tracked shared SDK-C1/R1 scope.
- Source policy and extension/user authorization are checked by the real Neutron deployment; these are read-only named helpers and no creation/update/delete or server-side policy emulation is added.

### `python:cloud/_network/NetworkCloudMixin/update_qos_bandwidth_limit_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateQoSBandwidthLimitRuleRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetCloudBandwidthLimitUpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.UpdateCloudBandwidthLimitRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.QoSRuleUpdateRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateDirection`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateMaxKBPS`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateMaxBurstKBPS`

- All named cloud kwargs are library-validated before HTTP, with concrete cloned JSON/map/primitive With options. Unknown/bound identity keys/nil or invalid JSON fail before qos gate. No caller builder/interface is needed; explicit null/zero remain present and later individual options replace prior bulk keys.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateValidKwargsBeforeHTTPAndOwnedOptions`.
- Each valid call fully consumes fresh uncached qos extension discovery and Source-only policy Find. Policy missing raises even for empty kwargs. BW/DSCP default ignore_missing=True then cloud NotFound, Minimum ignore_missing=False. Empty kwargs only after policy/optional gate returns None/no current GET/no PUT and does not validate unused child identity.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateEmptyAfterPolicyAndDirectionGateSkipsChild`.
- Nonempty attrs trigger exactly one strict current-rule GET with no child find/list fallback/ignored404. Dirty PUT uses actual returned rule identity and separate current/policy/rule evidence. Rejected400/403/404/409/503 retain native value status/body/header and are not normalized to successful update or None.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateStrictChildGETAndFailedPUTKeepIndependentReceipts`.
- A private borrowed seed owns original raw Body presence, URI parent, actual receipt, family/API/client/source guard. Clone deep-copies raw state and private proof; exported Resource/Wire/QoSPolicyID/Envelope/Header mutations do not create false dirtiness, forge identity or change no-PUT proof. Foreign API/family or changed selected source rejects before mutation.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateBorrowedLedgerCloneOwnerAndNullURI`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateInvalidBorrowingAndDirtyIDNeverPUT`.
- Raw Source dirty equality uses presence and CPython-loaded values rather than projected defaults/getters: missing→null is dirty, true/1 and false/0 are equal, float precision/underflow equality matches Source. PUT contains changed keys only; partial response seeds untouched/new fields only in Resource view while Wire remains actual. No-dirty returns cloned private current receipt without PUT evidence.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateRawDeltaEqualAndPartialResponse`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateRawDirtyPresenceSelectedVersionAndActualPartialWire`.
- Equal null-ID updates return without ID validation or PUT; dirty null/missing raises pre-PUT. Dirty falsey nonnull0/false/emptystring/0.0/underflow/emptyarray/object follows actual Source collection trailing-slash PUT through explicit new copied SourceCollectionPUT profile. Root shared seam requires SourceFindPolicy+SourceCollectionPUT+PUT empty identity and preserves legacy false/default/other-method behavior; selected fixed profile.Path plus slash is the only collection target.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateInvalidBorrowingAndDirtyIDNeverPUT`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateFalseyNonnullIDCollectionPUTAndNullParentView`.
- Private nil raw parent is normalized to JSON null and path formatting follows Python %s without losing raw URI getter state. Foreign response qos_policy_id/microversion/location stay in actual Wire while Resource reads original raw URI/selected version/owned null location. Source _update resets version to null on no-dirty; a dirty commit restores selected version.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateBorrowedLedgerCloneOwnerAndNullURI`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateRawDirtyPresenceSelectedVersionAndActualPartialWire`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateFalseyNonnullIDCollectionPUTAndNullParentView`.
- Actual strict GET/PUT accepted/rejected200/401/404/503 read+Close/cancel/source faults preserve stage receipt and exact causes with zero retry/reauth hooks and explicit unexpected bounded sentinels. Native401 retains pointer ErrUnableToReauthenticate.ErrOriginal native value response and ErrReauth despite no native Unwrap. Faults never hide as missing/None/no-op.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdatePhysicalGETPUTFaultsKeepNative401AndNoHooks`.
- New Source profiles opt100..399 in while existing typed/native profiles stay unchanged. Source PUT199, clean503 retry/401 live token renewal and sticky selected route guards retain actual proof. Source/header capture precedes option callbacks, so callback MoreHeaders mutation cannot leak into ongoing requests; changed route before retry aborts with original503.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateCleanRetryRenewal199AndHeaderCapture`.
- BW update allows max_kbps/max_burst_kbps/direction. After policy lookup, every nonnull direction (false/0/emptystr/array/object included) triggers an independent fresh qos-bw-limit-direction inventory; unavailable drops direction only, potentially making kwargs empty before child GET. Null keeps its field with no optional gate; optional HTTP/physical failure is not treated as unsupported direction.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateEmptyAfterPolicyAndDirectionGateSkipsChild`.

Python/Go differences:

- Go carrier separates Rule/Current/Policy/Extensions and NoOp/DirectionDropped rather than Python Resource/None only. Actual GET and PUT receipt are independent owned snapshots; no-dirty returns prior receipt without fabricated PUT proof. Mutable Python borrowed object side effects are not exposed as server observations.
- Complete UTF8 JSON/concrete With defaults and owned snapshot boundaries remain explicit Go policy. Custom runtime objects, mutable Resource/class/adapter/session/cache/current_location and dynamic class dispatch remain SDK-R1/SDK-C1/SDK-S1 shared foundation remaining with supported increment0. These three cloud mappings do not promote general Proxy raw update kwargs or full SDK parity.
- Go literal service-relative URI segments are escaped and blank/control/traversal identities rejected. Source slash/percent URI formatting may create arbitrary path structure. Actual falsey nonnull collection PUT is implemented through Source-only opt-in rather than dismissed as an API convenience difference; null dirty ID still errors and equal null ID does not require a route.
- JSON object wire order is not guaranteed; raw keyword data is not coerced/clamped to native rate/direction types/defaults. Server schema/range/driver validation remains authoritative. Actual Source74-case execution under CPython3.14.8 imported pinned cloud decorator/Proxy find-get-update/Resource find-fetch-_update-commit, with HTTP Adapter and extension generator fixture only.
- Stock Neutron27 BW/DSCP/minimum update policy is project ADMIN_OR_PARENT_OWNER_MANAGER, distinct from historic deprecated ADMIN_ONLY. Readable shared policy does not confer update permission. Runtime/reference/mock/race/main checks do not prove authenticated backend/driver/deployment acceptance.

### `python:cloud/_network/NetworkCloudMixin/list_qos_dscp_marking_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.ListQoSDSCPMarkingRules`
- `github.com/JSYoo5B/go-openstacksdk/network.WithDSCPMarkingCloudOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithDSCPMarkingCloudCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network.WithDSCPMarkingCloudFiltersJSON`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ProxyDSCPMarkingRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingProxyQuery`

- Every call consumes the full fresh qos extension inventory before GET-first exact policy Find; early qos match does not suppress later alias/page faults. Missing or ambiguous policy stops rule I/O with actual already observed gate/policy evidence, no fabricated policy response.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingGateConsumesAllExtensionsAndStopsBeforePolicy`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPolicyMissingAndAmbiguityRetainActualGate`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingFreshCompleteGatePolicyNameLookupAndSourceLinkPrecedence`.
- Selected original network service/client/source is captured before owned option callbacks; cancellation causes and source mutation remain terminal. Nil context/service fails before discovery or invented response. Location and raw JSON options are cloned and callbacks run once.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingFilterOwnershipCallbackGuardAndFalseyCases`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPreflightAndInvalidLocationHaveHonestEvidence`.
- Actual HTTP physical rule faults and late pages retain partial Inventory plus native response bytes/header/status in error, with no successful Value/Rules. Real Content-Length faults at 200/203/401/404/429/503 preserve pointer401 ErrOriginal VALUE response and ErrReauth EOF; retry/backoff/reauth fault hooks remain zero.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPhysical401RulePageAndLateFailurePreserveStageProof`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPhysicalLatePagesKeepActualHeaderStatusAndNoFaultHooks`.
- Falsey cloud filters become empty kwargs only after extension/policy lookup; truthy nonmappings fail Source **kwargs expansion before rule HTTP. Source Body filters are local while server queries are only declared limit/marker. Full paging and CPython JSON-loaded equality are owned by the concrete Source family profile.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingListOwnsGateParentPagingAndRawFields`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingFilterOwnershipCallbackGuardAndFalseyCases`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingSearchKeepsRuleIDPhaseAndFilterPresence`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingFalseyMaxItemsAndTemplateOnlyJSONArguments`.
- Reachable Proxy/Resource formals headers, microversion, paginated/max_items, nullable/empty BasePath and JSON-domain %(key)s route binding are handled by an owned family Source lane; one-pass conflicting attrs restores the URI parent and preserves original JMES expression. Formal duplicates fail before rule I/O. Legacy typed list policy remains unchanged.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingSourceFormalFiltersAndRestoredURIBeforeJMES`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingSourceProxyLiteralsBasePathAndOldTypedBoundary`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingFalseyMaxItemsAndTemplateOnlyJSONArguments`.
- Full JMESPath evaluates the opaque generator before rule collection. Identity/default returns actual rules; array/string/dict literals return arbitrary list() values with zero rule HTTP and no invented records. Nested generators remain lazy, guarded, location-bound and represented in Values without fake JSON Value.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingSourceProxyLiteralsBasePathAndOldTypedBoundary`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPolymorphicFalseySearchAndNestedGeneratorMembership`.
- Rule Resource default/unknown Body, raw Wire, envelope/header/status and URI parent remain distinct. Standalone location null and owned connected current_location are applied before filtering; undeclared project_id/availability_zone cannot change the configured project/zone. Selected Rules, Values, Inventory, LastRule and JSON Value do not alias exported rule metadata/bytes.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingListOwnsGateParentPagingAndRawFields`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPreflightAndInvalidLocationHaveHonestEvidence`.

Python/Go differences:

- Go returns an owned result carrier with real Extensions/Policy/Inventory/LastRule evidence, correlated Rules/Rule and independent Values/Value; literal/opaque output never fabricates response Resource/Wire/Header/status. Failed page receipt remains in the actual error rather than a fictional failed resource.
- Source kwargs are restricted to JSON-domain data, policy/rule identities to safe nonblank Go strings, HTTP headers to string-valued mappings, microversion/base_path to nullable strings and truthy max_items to nonnegative Go integers or true. Go validates these typed/ownership boundaries and service-relative URI safety before some Python dynamic errors; unsafe origin/query/fragment/traversal and non-string numeric %-route formatting are explicit errors.
- Dictionary-key iterable results and key-based JMESPath functions use reviewed Go lexical order instead of Python insertion order. JMES deprecation warnings are omitted. Opaque generator repr/to_string involving Python process addresses and non-JSON dynamic object/session/mutable Resource behavior remain SDK-R1/C1/S1 scope; no equivalent process address string is fabricated.
- Explicit CurrentLocation is an owned configured snapshot; the rule classes have no project_id/availability_zone descriptors, so unknown remote fields cannot override it. Root Connection wrappers may supply this library-owned option automatically. Existing paging bounds/cycle checks/HTTP Link repair and fixed request guards remain documented Go improvements. Authenticated live Neutron acceptance is separate from testhelper HTTP/Source AST proof.

### `python:cloud/_network/NetworkCloudMixin/search_qos_dscp_marking_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.SearchQoSDSCPMarkingRules`
- `github.com/JSYoo5B/go-openstacksdk/network.WithDSCPMarkingCloudOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithDSCPMarkingCloudCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network.WithDSCPMarkingCloudFiltersJSON`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ProxyDSCPMarkingRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithDSCPMarkingProxyQuery`

- Every call consumes the full fresh qos extension inventory before GET-first exact policy Find; early qos match does not suppress later alias/page faults. Missing or ambiguous policy stops rule I/O with actual already observed gate/policy evidence, no fabricated policy response.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingGateConsumesAllExtensionsAndStopsBeforePolicy`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPolicyMissingAndAmbiguityRetainActualGate`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingFreshCompleteGatePolicyNameLookupAndSourceLinkPrecedence`.
- Selected original network service/client/source is captured before owned option callbacks; cancellation causes and source mutation remain terminal. Nil context/service fails before discovery or invented response. Location and raw JSON options are cloned and callbacks run once.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingFilterOwnershipCallbackGuardAndFalseyCases`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPreflightAndInvalidLocationHaveHonestEvidence`.
- Actual HTTP physical rule faults and late pages retain partial Inventory plus native response bytes/header/status in error, with no successful Value/Rules. Real Content-Length faults at 200/203/401/404/429/503 preserve pointer401 ErrOriginal VALUE response and ErrReauth EOF; retry/backoff/reauth fault hooks remain zero.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPhysical401RulePageAndLateFailurePreserveStageProof`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPhysicalLatePagesKeepActualHeaderStatusAndNoFaultHooks`.
- Falsey cloud filters become empty kwargs only after extension/policy lookup; truthy nonmappings fail Source **kwargs expansion before rule HTTP. Source Body filters are local while server queries are only declared limit/marker. Full paging and CPython JSON-loaded equality are owned by the concrete Source family profile.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingListOwnsGateParentPagingAndRawFields`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingFilterOwnershipCallbackGuardAndFalseyCases`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingSearchKeepsRuleIDPhaseAndFilterPresence`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingFalseyMaxItemsAndTemplateOnlyJSONArguments`.
- Reachable Proxy/Resource formals headers, microversion, paginated/max_items, nullable/empty BasePath and JSON-domain %(key)s route binding are handled by an owned family Source lane; one-pass conflicting attrs restores the URI parent and preserves original JMES expression. Formal duplicates fail before rule I/O. Legacy typed list policy remains unchanged.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingSourceFormalFiltersAndRestoredURIBeforeJMES`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingSourceProxyLiteralsBasePathAndOldTypedBoundary`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingFalseyMaxItemsAndTemplateOnlyJSONArguments`.
- Full JMESPath evaluates the opaque generator before rule collection. Identity/default returns actual rules; array/string/dict literals return arbitrary list() values with zero rule HTTP and no invented records. Nested generators remain lazy, guarded, location-bound and represented in Values without fake JSON Value.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingSourceProxyLiteralsBasePathAndOldTypedBoundary`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPolymorphicFalseySearchAndNestedGeneratorMembership`.
- Cloud search first finishes list, then uses exact ID/name or fnmatch glob and ordered/nested Source getter dictionary filtering, including loaded float precision. Unknown attributes fail lazily with actual final successful receipt; an earlier mismatch skips later invalid keys. Top-level truthy string filters remain unreachable JMES because list **kwargs fails first.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingSearchKeepsRuleIDPhaseAndFilterPresence`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingOrderedNestedCloudFiltersAndLoadedFloatEquality`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingFilterOwnershipCallbackGuardAndFalseyCases`.
- Without identifier selection, Source skips falsey JSON literal values before dictionary lookup. Identifier get() errors happen before that skip. Unknown-key membership on a nested opaque generator consumes its actual rules before failure and retains those actual Inventory/receipts.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPolymorphicFalseySearchAndNestedGeneratorMembership`.
- Rule Resource default/unknown Body, raw Wire, envelope/header/status and URI parent remain distinct. Standalone location null and owned connected current_location are applied before filtering; undeclared project_id/availability_zone cannot change the configured project/zone. Selected Rules, Values, Inventory, LastRule and JSON Value do not alias exported rule metadata/bytes.
  Tests: `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingListOwnsGateParentPagingAndRawFields`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_dscp_test.go:TestCloudQoSDSCPMarkingPreflightAndInvalidLocationHaveHonestEvidence`.

Python/Go differences:

- Go returns an owned result carrier with real Extensions/Policy/Inventory/LastRule evidence, correlated Rules/Rule and independent Values/Value; literal/opaque output never fabricates response Resource/Wire/Header/status. Failed page receipt remains in the actual error rather than a fictional failed resource.
- Source kwargs are restricted to JSON-domain data, policy/rule identities to safe nonblank Go strings, HTTP headers to string-valued mappings, microversion/base_path to nullable strings and truthy max_items to nonnegative Go integers or true. Go validates these typed/ownership boundaries and service-relative URI safety before some Python dynamic errors; unsafe origin/query/fragment/traversal and non-string numeric %-route formatting are explicit errors.
- Dictionary-key iterable results and key-based JMESPath functions use reviewed Go lexical order instead of Python insertion order. JMES deprecation warnings are omitted. Opaque generator repr/to_string involving Python process addresses and non-JSON dynamic object/session/mutable Resource behavior remain SDK-R1/C1/S1 scope; no equivalent process address string is fabricated.
- Explicit CurrentLocation is an owned configured snapshot; the rule classes have no project_id/availability_zone descriptors, so unknown remote fields cannot override it. Root Connection wrappers may supply this library-owned option automatically. Existing paging bounds/cycle checks/HTTP Link repair and fixed request guards remain documented Go improvements. Authenticated live Neutron acceptance is separate from testhelper HTTP/Source AST proof.

### `python:cloud/_network/NetworkCloudMixin/update_qos_dscp_marking_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateQoSDSCPMarkingRuleRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetCloudDSCPMarkingUpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.UpdateCloudDSCPMarkingRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.QoSRuleUpdateRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateDSCPMark`

- All named cloud kwargs are library-validated before HTTP, with concrete cloned JSON/map/primitive With options. Unknown/bound identity keys/nil or invalid JSON fail before qos gate. No caller builder/interface is needed; explicit null/zero remain present and later individual options replace prior bulk keys.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateValidKwargsBeforeHTTPAndOwnedOptions`.
- Each valid call fully consumes fresh uncached qos extension discovery and Source-only policy Find. Policy missing raises even for empty kwargs. BW/DSCP default ignore_missing=True then cloud NotFound, Minimum ignore_missing=False. Empty kwargs only after policy/optional gate returns None/no current GET/no PUT and does not validate unused child identity.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateEmptyAfterPolicyAndDirectionGateSkipsChild`.
- Nonempty attrs trigger exactly one strict current-rule GET with no child find/list fallback/ignored404. Dirty PUT uses actual returned rule identity and separate current/policy/rule evidence. Rejected400/403/404/409/503 retain native value status/body/header and are not normalized to successful update or None.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateStrictChildGETAndFailedPUTKeepIndependentReceipts`.
- A private borrowed seed owns original raw Body presence, URI parent, actual receipt, family/API/client/source guard. Clone deep-copies raw state and private proof; exported Resource/Wire/QoSPolicyID/Envelope/Header mutations do not create false dirtiness, forge identity or change no-PUT proof. Foreign API/family or changed selected source rejects before mutation.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateBorrowedLedgerCloneOwnerAndNullURI`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateInvalidBorrowingAndDirtyIDNeverPUT`.
- Raw Source dirty equality uses presence and CPython-loaded values rather than projected defaults/getters: missing→null is dirty, true/1 and false/0 are equal, float precision/underflow equality matches Source. PUT contains changed keys only; partial response seeds untouched/new fields only in Resource view while Wire remains actual. No-dirty returns cloned private current receipt without PUT evidence.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateRawDeltaEqualAndPartialResponse`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateRawDirtyPresenceSelectedVersionAndActualPartialWire`.
- Equal null-ID updates return without ID validation or PUT; dirty null/missing raises pre-PUT. Dirty falsey nonnull0/false/emptystring/0.0/underflow/emptyarray/object follows actual Source collection trailing-slash PUT through explicit new copied SourceCollectionPUT profile. Root shared seam requires SourceFindPolicy+SourceCollectionPUT+PUT empty identity and preserves legacy false/default/other-method behavior; selected fixed profile.Path plus slash is the only collection target.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateInvalidBorrowingAndDirtyIDNeverPUT`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateFalseyNonnullIDCollectionPUTAndNullParentView`.
- Private nil raw parent is normalized to JSON null and path formatting follows Python %s without losing raw URI getter state. Foreign response qos_policy_id/microversion/location stay in actual Wire while Resource reads original raw URI/selected version/owned null location. Source _update resets version to null on no-dirty; a dirty commit restores selected version.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateBorrowedLedgerCloneOwnerAndNullURI`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateRawDirtyPresenceSelectedVersionAndActualPartialWire`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateFalseyNonnullIDCollectionPUTAndNullParentView`.
- Actual strict GET/PUT accepted/rejected200/401/404/503 read+Close/cancel/source faults preserve stage receipt and exact causes with zero retry/reauth hooks and explicit unexpected bounded sentinels. Native401 retains pointer ErrUnableToReauthenticate.ErrOriginal native value response and ErrReauth despite no native Unwrap. Faults never hide as missing/None/no-op.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdatePhysicalGETPUTFaultsKeepNative401AndNoHooks`.
- New Source profiles opt100..399 in while existing typed/native profiles stay unchanged. Source PUT199, clean503 retry/401 live token renewal and sticky selected route guards retain actual proof. Source/header capture precedes option callbacks, so callback MoreHeaders mutation cannot leak into ongoing requests; changed route before retry aborts with original503.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateCleanRetryRenewal199AndHeaderCapture`.
- DSCP accepts dscp_mark only and default ignore_missing=True policy lookup with cloud NotFound on nil. Empty attrs return None after qos/policy but before current GET; nonempty attrs use strict current GET and raw-only delta commit.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateEmptyAfterPolicyAndDirectionGateSkipsChild`, `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateRawDeltaEqualAndPartialResponse`.

Python/Go differences:

- Go carrier separates Rule/Current/Policy/Extensions and NoOp/DirectionDropped rather than Python Resource/None only. Actual GET and PUT receipt are independent owned snapshots; no-dirty returns prior receipt without fabricated PUT proof. Mutable Python borrowed object side effects are not exposed as server observations.
- Complete UTF8 JSON/concrete With defaults and owned snapshot boundaries remain explicit Go policy. Custom runtime objects, mutable Resource/class/adapter/session/cache/current_location and dynamic class dispatch remain SDK-R1/SDK-C1/SDK-S1 shared foundation remaining with supported increment0. These three cloud mappings do not promote general Proxy raw update kwargs or full SDK parity.
- Go literal service-relative URI segments are escaped and blank/control/traversal identities rejected. Source slash/percent URI formatting may create arbitrary path structure. Actual falsey nonnull collection PUT is implemented through Source-only opt-in rather than dismissed as an API convenience difference; null dirty ID still errors and equal null ID does not require a route.
- JSON object wire order is not guaranteed; raw keyword data is not coerced/clamped to native rate/direction types/defaults. Server schema/range/driver validation remains authoritative. Actual Source74-case execution under CPython3.14.8 imported pinned cloud decorator/Proxy find-get-update/Resource find-fetch-_update-commit, with HTTP Adapter and extension generator fixture only.
- Stock Neutron27 BW/DSCP/minimum update policy is project ADMIN_OR_PARENT_OWNER_MANAGER, distinct from historic deprecated ADMIN_ONLY. Readable shared policy does not confer update permission. Runtime/reference/mock/race/main checks do not prove authenticated backend/driver/deployment acceptance.

### `python:cloud/_network/NetworkCloudMixin/list_qos_minimum_bandwidth_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.ListQoSMinimumBandwidthRules`
- `github.com/JSYoo5B/go-openstacksdk/network.WithMinimumBandwidthCloudOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithMinimumBandwidthCloudCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network.WithMinimumBandwidthCloudFiltersJSON`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ProxyMinimumBandwidthRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthProxyQuery`

- Every call consumes the full fresh qos extension inventory before GET-first exact policy Find; early qos match does not suppress later alias/page faults. Missing or ambiguous policy stops rule I/O with actual already observed gate/policy evidence, no fabricated policy response.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthGateConsumesAllExtensionsAndStopsBeforePolicy`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPolicyMissingAndAmbiguityRetainActualGate`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFreshCompleteGatePolicyNameLookupAndSourceLinkPrecedence`.
- Selected original network service/client/source is captured before owned option callbacks; cancellation causes and source mutation remain terminal. Nil context/service fails before discovery or invented response. Location and raw JSON options are cloned and callbacks run once.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFilterOwnershipCallbackGuardAndFalseyCases`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPreflightAndInvalidLocationHaveHonestEvidence`.
- Actual HTTP physical rule faults and late pages retain partial Inventory plus native response bytes/header/status in error, with no successful Value/Rules. Real Content-Length faults at 200/203/401/404/429/503 preserve pointer401 ErrOriginal VALUE response and ErrReauth EOF; retry/backoff/reauth fault hooks remain zero.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPhysical401RulePageAndLateFailurePreserveStageProof`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPhysicalLatePagesKeepActualHeaderStatusAndNoFaultHooks`.
- Falsey cloud filters become empty kwargs only after extension/policy lookup; truthy nonmappings fail Source **kwargs expansion before rule HTTP. Source Body filters are local while server queries are only declared limit/marker. Full paging and CPython JSON-loaded equality are owned by the concrete Source family profile.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthListOwnsGateParentPagingAndRawFields`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFilterOwnershipCallbackGuardAndFalseyCases`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSearchKeepsRuleIDPhaseAndFilterPresence`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFalseyMaxItemsAndTemplateOnlyJSONArguments`.
- Reachable Proxy/Resource formals headers, microversion, paginated/max_items, nullable/empty BasePath and JSON-domain %(key)s route binding are handled by an owned family Source lane; one-pass conflicting attrs restores the URI parent and preserves original JMES expression. Formal duplicates fail before rule I/O. Legacy typed list policy remains unchanged.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSourceFormalFiltersAndRestoredURIBeforeJMES`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSourceProxyLiteralsBasePathAndOldTypedBoundary`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFalseyMaxItemsAndTemplateOnlyJSONArguments`.
- Full JMESPath evaluates the opaque generator before rule collection. Identity/default returns actual rules; array/string/dict literals return arbitrary list() values with zero rule HTTP and no invented records. Nested generators remain lazy, guarded, location-bound and represented in Values without fake JSON Value.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSourceProxyLiteralsBasePathAndOldTypedBoundary`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPolymorphicFalseySearchAndNestedGeneratorMembership`.
- Rule Resource default/unknown Body, raw Wire, envelope/header/status and URI parent remain distinct. Standalone location null and owned connected current_location are applied before filtering; undeclared project_id/availability_zone cannot change the configured project/zone. Selected Rules, Values, Inventory, LastRule and JSON Value do not alias exported rule metadata/bytes.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthListOwnsGateParentPagingAndRawFields`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPreflightAndInvalidLocationHaveHonestEvidence`.

Python/Go differences:

- Go returns an owned result carrier with real Extensions/Policy/Inventory/LastRule evidence, correlated Rules/Rule and independent Values/Value; literal/opaque output never fabricates response Resource/Wire/Header/status. Failed page receipt remains in the actual error rather than a fictional failed resource.
- Source kwargs are restricted to JSON-domain data, policy/rule identities to safe nonblank Go strings, HTTP headers to string-valued mappings, microversion/base_path to nullable strings and truthy max_items to nonnegative Go integers or true. Go validates these typed/ownership boundaries and service-relative URI safety before some Python dynamic errors; unsafe origin/query/fragment/traversal and non-string numeric %-route formatting are explicit errors.
- Dictionary-key iterable results and key-based JMESPath functions use reviewed Go lexical order instead of Python insertion order. JMES deprecation warnings are omitted. Opaque generator repr/to_string involving Python process addresses and non-JSON dynamic object/session/mutable Resource behavior remain SDK-R1/C1/S1 scope; no equivalent process address string is fabricated.
- Explicit CurrentLocation is an owned configured snapshot; the rule classes have no project_id/availability_zone descriptors, so unknown remote fields cannot override it. Root Connection wrappers may supply this library-owned option automatically. Existing paging bounds/cycle checks/HTTP Link repair and fixed request guards remain documented Go improvements. Authenticated live Neutron acceptance is separate from testhelper HTTP/Source AST proof.

### `python:cloud/_network/NetworkCloudMixin/search_qos_minimum_bandwidth_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.SearchQoSMinimumBandwidthRules`
- `github.com/JSYoo5B/go-openstacksdk/network.WithMinimumBandwidthCloudOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithMinimumBandwidthCloudCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network.WithMinimumBandwidthCloudFiltersJSON`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.ProxyMinimumBandwidthRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.WithMinimumBandwidthProxyQuery`

- Every call consumes the full fresh qos extension inventory before GET-first exact policy Find; early qos match does not suppress later alias/page faults. Missing or ambiguous policy stops rule I/O with actual already observed gate/policy evidence, no fabricated policy response.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthGateConsumesAllExtensionsAndStopsBeforePolicy`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPolicyMissingAndAmbiguityRetainActualGate`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFreshCompleteGatePolicyNameLookupAndSourceLinkPrecedence`.
- Selected original network service/client/source is captured before owned option callbacks; cancellation causes and source mutation remain terminal. Nil context/service fails before discovery or invented response. Location and raw JSON options are cloned and callbacks run once.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFilterOwnershipCallbackGuardAndFalseyCases`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPreflightAndInvalidLocationHaveHonestEvidence`.
- Actual HTTP physical rule faults and late pages retain partial Inventory plus native response bytes/header/status in error, with no successful Value/Rules. Real Content-Length faults at 200/203/401/404/429/503 preserve pointer401 ErrOriginal VALUE response and ErrReauth EOF; retry/backoff/reauth fault hooks remain zero.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPhysical401RulePageAndLateFailurePreserveStageProof`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPhysicalLatePagesKeepActualHeaderStatusAndNoFaultHooks`.
- Falsey cloud filters become empty kwargs only after extension/policy lookup; truthy nonmappings fail Source **kwargs expansion before rule HTTP. Source Body filters are local while server queries are only declared limit/marker. Full paging and CPython JSON-loaded equality are owned by the concrete Source family profile.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthListOwnsGateParentPagingAndRawFields`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFilterOwnershipCallbackGuardAndFalseyCases`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSearchKeepsRuleIDPhaseAndFilterPresence`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFalseyMaxItemsAndTemplateOnlyJSONArguments`.
- Reachable Proxy/Resource formals headers, microversion, paginated/max_items, nullable/empty BasePath and JSON-domain %(key)s route binding are handled by an owned family Source lane; one-pass conflicting attrs restores the URI parent and preserves original JMES expression. Formal duplicates fail before rule I/O. Legacy typed list policy remains unchanged.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSourceFormalFiltersAndRestoredURIBeforeJMES`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSourceProxyLiteralsBasePathAndOldTypedBoundary`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFalseyMaxItemsAndTemplateOnlyJSONArguments`.
- Full JMESPath evaluates the opaque generator before rule collection. Identity/default returns actual rules; array/string/dict literals return arbitrary list() values with zero rule HTTP and no invented records. Nested generators remain lazy, guarded, location-bound and represented in Values without fake JSON Value.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSourceProxyLiteralsBasePathAndOldTypedBoundary`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPolymorphicFalseySearchAndNestedGeneratorMembership`.
- Cloud search first finishes list, then uses exact ID/name or fnmatch glob and ordered/nested Source getter dictionary filtering, including loaded float precision. Unknown attributes fail lazily with actual final successful receipt; an earlier mismatch skips later invalid keys. Top-level truthy string filters remain unreachable JMES because list **kwargs fails first.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthSearchKeepsRuleIDPhaseAndFilterPresence`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthOrderedNestedCloudFiltersAndLoadedFloatEquality`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthFilterOwnershipCallbackGuardAndFalseyCases`.
- Without identifier selection, Source skips falsey JSON literal values before dictionary lookup. Identifier get() errors happen before that skip. Unknown-key membership on a nested opaque generator consumes its actual rules before failure and retains those actual Inventory/receipts.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPolymorphicFalseySearchAndNestedGeneratorMembership`.
- Rule Resource default/unknown Body, raw Wire, envelope/header/status and URI parent remain distinct. Standalone location null and owned connected current_location are applied before filtering; undeclared project_id/availability_zone cannot change the configured project/zone. Selected Rules, Values, Inventory, LastRule and JSON Value do not alias exported rule metadata/bytes.
  Tests: `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthListOwnsGateParentPagingAndRawFields`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthRawFilterLocationAndResultCarriersOwnAllData`, `network/cloud_qos_minimum_bandwidth_test.go:TestCloudQoSMinimumBandwidthPreflightAndInvalidLocationHaveHonestEvidence`.

Python/Go differences:

- Go returns an owned result carrier with real Extensions/Policy/Inventory/LastRule evidence, correlated Rules/Rule and independent Values/Value; literal/opaque output never fabricates response Resource/Wire/Header/status. Failed page receipt remains in the actual error rather than a fictional failed resource.
- Source kwargs are restricted to JSON-domain data, policy/rule identities to safe nonblank Go strings, HTTP headers to string-valued mappings, microversion/base_path to nullable strings and truthy max_items to nonnegative Go integers or true. Go validates these typed/ownership boundaries and service-relative URI safety before some Python dynamic errors; unsafe origin/query/fragment/traversal and non-string numeric %-route formatting are explicit errors.
- Dictionary-key iterable results and key-based JMESPath functions use reviewed Go lexical order instead of Python insertion order. JMES deprecation warnings are omitted. Opaque generator repr/to_string involving Python process addresses and non-JSON dynamic object/session/mutable Resource behavior remain SDK-R1/C1/S1 scope; no equivalent process address string is fabricated.
- Explicit CurrentLocation is an owned configured snapshot; the rule classes have no project_id/availability_zone descriptors, so unknown remote fields cannot override it. Root Connection wrappers may supply this library-owned option automatically. Existing paging bounds/cycle checks/HTTP Link repair and fixed request guards remain documented Go improvements. Authenticated live Neutron acceptance is separate from testhelper HTTP/Source AST proof.

### `python:cloud/_network/NetworkCloudMixin/update_qos_minimum_bandwidth_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateQoSMinimumBandwidthRuleRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.GetCloudMinimumBandwidthUpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.API.UpdateCloudMinimumBandwidthRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules.QoSRuleUpdateRecord.Clone`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies.API.FindSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateDirection`
- `github.com/JSYoo5B/go-openstacksdk/network.WithQoSRuleUpdateMinKBPS`

- All named cloud kwargs are library-validated before HTTP, with concrete cloned JSON/map/primitive With options. Unknown/bound identity keys/nil or invalid JSON fail before qos gate. No caller builder/interface is needed; explicit null/zero remain present and later individual options replace prior bulk keys.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateValidKwargsBeforeHTTPAndOwnedOptions`.
- Each valid call fully consumes fresh uncached qos extension discovery and Source-only policy Find. Policy missing raises even for empty kwargs. BW/DSCP default ignore_missing=True then cloud NotFound, Minimum ignore_missing=False. Empty kwargs only after policy/optional gate returns None/no current GET/no PUT and does not validate unused child identity.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateEmptyAfterPolicyAndDirectionGateSkipsChild`.
- Nonempty attrs trigger exactly one strict current-rule GET with no child find/list fallback/ignored404. Dirty PUT uses actual returned rule identity and separate current/policy/rule evidence. Rejected400/403/404/409/503 retain native value status/body/header and are not normalized to successful update or None.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateStrictChildGETAndFailedPUTKeepIndependentReceipts`.
- A private borrowed seed owns original raw Body presence, URI parent, actual receipt, family/API/client/source guard. Clone deep-copies raw state and private proof; exported Resource/Wire/QoSPolicyID/Envelope/Header mutations do not create false dirtiness, forge identity or change no-PUT proof. Foreign API/family or changed selected source rejects before mutation.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateBorrowedLedgerCloneOwnerAndNullURI`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateInvalidBorrowingAndDirtyIDNeverPUT`.
- Raw Source dirty equality uses presence and CPython-loaded values rather than projected defaults/getters: missing→null is dirty, true/1 and false/0 are equal, float precision/underflow equality matches Source. PUT contains changed keys only; partial response seeds untouched/new fields only in Resource view while Wire remains actual. No-dirty returns cloned private current receipt without PUT evidence.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateRawDeltaEqualAndPartialResponse`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateRawDirtyPresenceSelectedVersionAndActualPartialWire`.
- Equal null-ID updates return without ID validation or PUT; dirty null/missing raises pre-PUT. Dirty falsey nonnull0/false/emptystring/0.0/underflow/emptyarray/object follows actual Source collection trailing-slash PUT through explicit new copied SourceCollectionPUT profile. Root shared seam requires SourceFindPolicy+SourceCollectionPUT+PUT empty identity and preserves legacy false/default/other-method behavior; selected fixed profile.Path plus slash is the only collection target.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateInvalidBorrowingAndDirtyIDNeverPUT`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateFalseyNonnullIDCollectionPUTAndNullParentView`.
- Private nil raw parent is normalized to JSON null and path formatting follows Python %s without losing raw URI getter state. Foreign response qos_policy_id/microversion/location stay in actual Wire while Resource reads original raw URI/selected version/owned null location. Source _update resets version to null on no-dirty; a dirty commit restores selected version.
  Tests: `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateBorrowedLedgerCloneOwnerAndNullURI`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateRawDirtyPresenceSelectedVersionAndActualPartialWire`, `network/v2/extensions/qos/rules/python_cloud_updates_test.go:TestQoSCloudRuleUpdateFalseyNonnullIDCollectionPUTAndNullParentView`.
- Actual strict GET/PUT accepted/rejected200/401/404/503 read+Close/cancel/source faults preserve stage receipt and exact causes with zero retry/reauth hooks and explicit unexpected bounded sentinels. Native401 retains pointer ErrUnableToReauthenticate.ErrOriginal native value response and ErrReauth despite no native Unwrap. Faults never hide as missing/None/no-op.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdatePhysicalGETPUTFaultsKeepNative401AndNoHooks`.
- New Source profiles opt100..399 in while existing typed/native profiles stay unchanged. Source PUT199, clean503 retry/401 live token renewal and sticky selected route guards retain actual proof. Source/header capture precedes option callbacks, so callback MoreHeaders mutation cannot leak into ongoing requests; changed route before retry aborts with original503.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateCleanRetryRenewal199AndHeaderCapture`.
- Minimum accepts min_kbps/direction and uses strict ignore_missing=False policy Find; direction never triggers BW optional gate. Empty attrs still require policy lookup, and nonempty attrs strict current GET precedes dirty-only commit.
  Tests: `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateEmptyAfterPolicyAndDirectionGateSkipsChild`, `network/cloud_qos_rule_updates_test.go:TestCloudQoSRuleUpdateRawDeltaEqualAndPartialResponse`.

Python/Go differences:

- Go carrier separates Rule/Current/Policy/Extensions and NoOp/DirectionDropped rather than Python Resource/None only. Actual GET and PUT receipt are independent owned snapshots; no-dirty returns prior receipt without fabricated PUT proof. Mutable Python borrowed object side effects are not exposed as server observations.
- Complete UTF8 JSON/concrete With defaults and owned snapshot boundaries remain explicit Go policy. Custom runtime objects, mutable Resource/class/adapter/session/cache/current_location and dynamic class dispatch remain SDK-R1/SDK-C1/SDK-S1 shared foundation remaining with supported increment0. These three cloud mappings do not promote general Proxy raw update kwargs or full SDK parity.
- Go literal service-relative URI segments are escaped and blank/control/traversal identities rejected. Source slash/percent URI formatting may create arbitrary path structure. Actual falsey nonnull collection PUT is implemented through Source-only opt-in rather than dismissed as an API convenience difference; null dirty ID still errors and equal null ID does not require a route.
- JSON object wire order is not guaranteed; raw keyword data is not coerced/clamped to native rate/direction types/defaults. Server schema/range/driver validation remains authoritative. Actual Source74-case execution under CPython3.14.8 imported pinned cloud decorator/Proxy find-get-update/Resource find-fetch-_update-commit, with HTTP Adapter and extension generator fixture only.
- Stock Neutron27 BW/DSCP/minimum update policy is project ADMIN_OR_PARENT_OWNER_MANAGER, distinct from historic deprecated ADMIN_ONLY. Readable shared policy does not confer update permission. Runtime/reference/mock/race/main checks do not prove authenticated backend/driver/deployment acceptance.

### `python:cloud/_network/NetworkCloudMixin/list_firewall_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.ListFirewallGroups`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallListFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallListOpts`

- Default and falsey filters send an unfiltered GET fwaas list and follow next links.
  Tests: `api/r2_cloud_net_firewall_test.go:TestR2CloudNetFirewallGroupListQueryAndLocalFilters`, `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBFirewallDefaultLists`.
- Declared query keys are sent to the server (lists repeat, booleans use Python spelling), unknown keys are discarded and Body-only keys are filtered locally with Python equality.
  Tests: `api/r2_cloud_net_firewall_test.go:TestR2CloudNetFirewallGroupListQueryAndLocalFilters`, `api/r2_cloud_net_firewall_test.go:TestR2CloudNetFirewallPoliciesAndRulesLocalFiltersAndPaging`.
- Invalid truthy filters and list controls fail before HTTP; server errors return no rows; a canceled context sends nothing.
  Tests: `api/r2_cloud_net_firewall_test.go:TestR2CloudNetFirewallListRejectsBadFiltersBeforeHTTP`, `api/r2_cloud_net_firewall_test.go:TestR2CloudNetFirewallListCancelledAndServerError`.

Python/Go differences:

- Python keeps unknown body attributes; the native Go model drops them and an absent field compares as its zero value.
- Go takes the filters dictionary as a JSON object option (WithFirewallListFilters) instead of **filters, rejects Proxy/Resource control keys (paginated, base_path, microversion, headers, max_items, jmespath_filters, allow_unknown_params) and returns typed rows instead of Resource objects.
- Paging follows native next links; a server that continues only through limit/marker without links is not continued.

### `python:cloud/_network/NetworkCloudMixin/list_firewall_policies`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.ListFirewallPolicies`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallListFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallListOpts`

- Default and falsey filters send an unfiltered GET fwaas list and follow next links.
  Tests: `api/r2_cloud_net_firewall_test.go:TestR2CloudNetFirewallGroupListQueryAndLocalFilters`, `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBFirewallDefaultLists`.
- Declared query keys are sent to the server (lists repeat, booleans use Python spelling), unknown keys are discarded and Body-only keys are filtered locally with Python equality.
  Tests: `api/r2_cloud_net_firewall_test.go:TestR2CloudNetFirewallGroupListQueryAndLocalFilters`, `api/r2_cloud_net_firewall_test.go:TestR2CloudNetFirewallPoliciesAndRulesLocalFiltersAndPaging`.
- Invalid truthy filters and list controls fail before HTTP; server errors return no rows; a canceled context sends nothing.
  Tests: `api/r2_cloud_net_firewall_test.go:TestR2CloudNetFirewallListRejectsBadFiltersBeforeHTTP`, `api/r2_cloud_net_firewall_test.go:TestR2CloudNetFirewallListCancelledAndServerError`.

Python/Go differences:

- Python keeps unknown body attributes; the native Go model drops them and an absent field compares as its zero value.
- Go takes the filters dictionary as a JSON object option (WithFirewallListFilters) instead of **filters, rejects Proxy/Resource control keys (paginated, base_path, microversion, headers, max_items, jmespath_filters, allow_unknown_params) and returns typed rows instead of Resource objects.
- Paging follows native next links; a server that continues only through limit/marker without links is not continued.

### `python:cloud/_network/NetworkCloudMixin/list_firewall_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.ListFirewallRules`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallListFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallListOpts`

- Default and falsey filters send an unfiltered GET fwaas list and follow next links.
  Tests: `api/r2_cloud_net_firewall_test.go:TestR2CloudNetFirewallGroupListQueryAndLocalFilters`, `network/python_parity_cloud_b_test.go:TestPythonCloudNetworkBFirewallDefaultLists`.
- Declared query keys are sent to the server (lists repeat, booleans use Python spelling), unknown keys are discarded and Body-only keys are filtered locally with Python equality.
  Tests: `api/r2_cloud_net_firewall_test.go:TestR2CloudNetFirewallGroupListQueryAndLocalFilters`, `api/r2_cloud_net_firewall_test.go:TestR2CloudNetFirewallPoliciesAndRulesLocalFiltersAndPaging`.
- Invalid truthy filters and list controls fail before HTTP; server errors return no rows; a canceled context sends nothing.
  Tests: `api/r2_cloud_net_firewall_test.go:TestR2CloudNetFirewallListRejectsBadFiltersBeforeHTTP`, `api/r2_cloud_net_firewall_test.go:TestR2CloudNetFirewallListCancelledAndServerError`.

Python/Go differences:

- Python keeps unknown body attributes; the native Go model drops them and an absent field compares as its zero value.
- Go takes the filters dictionary as a JSON object option (WithFirewallListFilters) instead of **filters, rejects Proxy/Resource control keys (paginated, base_path, microversion, headers, max_items, jmespath_filters, allow_unknown_params) and returns typed rows instead of Resource objects.
- Paging follows native next links; a server that continues only through limit/marker without links is not continued.

### `python:cloud/_network/NetworkCloudMixin/update_firewall_group`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateFirewallGroupRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallGroupUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallGroupUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallGroupUpdateFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallGroupUpdatePorts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallGroupUpdateOpts`

- SDK-owned exact eight cloud kwargs validate before strict target group Find, rejecting direct egress/ingress_firewall_policy_id despite the Source docstring claim. Empty attrs still look up; raw equal values skip PUT and dirty attrs preserve null/false/empty via a private SDK builder with separate lookup/native mutation output.
  Tests: `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateLookupOnlyAndDirtySelection`, `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateNameFallbackAndMissing`, `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateOwnedOptionsAndPreflight`, `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateConflictKeepsLookupAndSingleAttempt`, `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateFiltersStrictBindingAndFalseyDefaults`.
- After target Find, nonnull policy aliases resolve strict egress then ingress regardless input key order, null aliases set wire ID null without policy lookup, then ports resolve in input order with duplicates and without target-filter propagation. [] clears with no port lookup; equal resolved references still resolve before no-PUT. Midphase failures and null ports retain prior target/policy/port evidence and prevent further lookup/PUT.
  Tests: `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateTargetPolicyPortsOrderAndEmptyClear`, `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateNullPoliciesAndEmptyPorts`, `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateReferenceFailuresPreservePhaseOrder`, `network/cloud_firewall_group_update_test.go:TestCloudFirewallGroupUpdateResolvedEqualReferencesStillLookUp`.
- An accepted PUT (any status 200-399, as Python raise_from_response) returns Response, an owned UpdateReceipt with the actual body bytes, headers and status. Body follows Python Resource._translate_response: the resource-key member when present, otherwise the whole top-level object, without self and with unknown members preserved. Merged is the lookup wire row overlaid with every effective attribute and then Body, so a partial body keeps lookup fields; an empty, 204, truncated or otherwise unparsable body sets BodyIgnored and keeps that pre-commit state without error. Non-object JSON or a non-object resource-key member fails after remote success with the receipt and no Merged/native model. A rejected PUT has no receipt, and nothing is replayed. The native model is the projection of Merged.
  Tests: `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptMergesPythonTranslateResponse`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptNonObjectBodiesFailAfterRemoteSuccess`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptRejectedPutHasNoReceipt`.
- Cloud filters reach Resource.find through the shared Source fetch-control profile: base_path routes the initial GET (and fallback LIST), resource_response_key selects its response member, skip_cache has no Go cache to bypass, and these kwargs seed the found Resource as unknown constructor attributes while the PUT keeps the class route and dirty body; list_base_path changes only the fallback LIST route.
  Tests: `network/cloud_firewall_find_kwargs_test.go:TestCloudFirewallUpdateFindFetchControls`.
- Firewall lookup state is the actual member set without self (no attribute aliases are declared), so only differing values are sent and the accepted response, including case-variant unknown members, is merged by literal key.
  Tests: `network/cloud_update_source_state_test.go:TestCloudUpdateSourceStateFirewallIdentityMapping`.

Python/Go differences:

- UpdateFirewallGroupRecord owns exact decorator kwargs, target-first filters and ordered policy/port resolution. Lookup/PolicyLookups/PortLookups are separate read evidence; Response is the owned PUT receipt and Group the native projection of its merged view. The Source docstring direct-ID suggestion conflicts with its decorator; this helper follows actual rejection.
- The firewall Body declarations have no attribute aliases (every attribute name equals its wire key), so lookup dirty comparison and receipt merge use the actual members without self through the same Resource._consume_attrs state; a case-variant member stays an unknown member. The native model omits Merged members its schema cannot hold and reports them in UpdateReceipt.ProjectionError without failing the accepted update; Merged keeps the raw values Python would keep. NaN/Infinity bodies are invalid JSON to Go and take the BodyIgnored branch where Python would merge them (shared nonfinite JSON boundary in docs/sdk-support-ledger.md).

Unresolved boundaries:

- Policy references are nonempty JSON strings or null; ports are JSON arrays of nonempty strings. Arbitrary Python iterable/object/nonstring/empty response-ID/runtime-exception behavior and full same-instance Resource lifecycle remain outside this bounded profile.
- Full Requests encoding/nonfinite JSON, arbitrary object/numeric equality and SDK-R1/C1/S1 Resource/adapter/session/cache contracts remain separate. Referenced ports retain the existing ports.FindRecord status/encoding/Resource profile; firewall-engine coverage does not close those port-profile gaps.

### `python:cloud/_network/NetworkCloudMixin/update_firewall_policy`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateFirewallPolicyRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyUpdateFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyUpdateRules`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyUpdateOpts`

- SDK-owned six cloud kwargs validate before strict target policy Find, including empty attrs; raw equal attrs skip PUT and dirty attrs use a private upstream builder preserving explicit null/false/empty JSON. Lookup evidence and native Policy mutation result are distinct, with nil mutation result on failure.
  Tests: `network/cloud_firewall_policy_update_test.go:TestCloudFirewallPolicyUpdateLookupOnlyAndDirtySelection`, `network/cloud_firewall_policy_update_test.go:TestCloudFirewallPolicyUpdateNameFallbackAndMissing`, `network/cloud_firewall_policy_update_test.go:TestCloudFirewallPolicyUpdateOwnedOptionsAndPreflight`, `network/cloud_firewall_policy_update_test.go:TestCloudFirewallPolicyUpdateConflictKeepsLookupAndSingleAttempt`.
- A present firewall_rules key is translated only after target lookup: ordered strict rule lookup repeats duplicate references without forwarding policy filters, [] clears with zero rule lookup, and equal resolved IDs skip PUT only after lookups. A later missing reference retains target/prior rule evidence and stops further work; unsupported JSON reference shapes fail after target lookup. Falsey filters/defaults and bound argument collisions preserve the bounded strict Find contract.
  Tests: `network/cloud_firewall_policy_update_test.go:TestCloudFirewallPolicyUpdateOrderedRulesTargetFirstAndEmptyClear`, `network/cloud_firewall_policy_update_test.go:TestCloudFirewallPolicyUpdateRuleFailureKeepsPriorEvidence`, `network/cloud_firewall_policy_update_test.go:TestCloudFirewallPolicyUpdateUnsupportedRuleShapesAfterTargetLookup`, `network/cloud_firewall_policy_update_test.go:TestCloudFirewallPolicyUpdateFiltersStrictBindingAndFalseyDefaults`.
- An accepted PUT (any status 200-399, as Python raise_from_response) returns Response, an owned UpdateReceipt with the actual body bytes, headers and status. Body follows Python Resource._translate_response: the resource-key member when present, otherwise the whole top-level object, without self and with unknown members preserved. Merged is the lookup wire row overlaid with every effective attribute and then Body, so a partial body keeps lookup fields; an empty, 204, truncated or otherwise unparsable body sets BodyIgnored and keeps that pre-commit state without error. Non-object JSON or a non-object resource-key member fails after remote success with the receipt and no Merged/native model. A rejected PUT has no receipt, and nothing is replayed. The native model is the projection of Merged.
  Tests: `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptMergesPythonTranslateResponse`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptNonObjectBodiesFailAfterRemoteSuccess`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptRejectedPutHasNoReceipt`.
- Cloud filters reach Resource.find through the shared Source fetch-control profile: base_path routes the initial GET (and fallback LIST), resource_response_key selects its response member, skip_cache has no Go cache to bypass, and these kwargs seed the found Resource as unknown constructor attributes while the PUT keeps the class route and dirty body; list_base_path changes only the fallback LIST route.
  Tests: `network/cloud_firewall_find_kwargs_test.go:TestCloudFirewallUpdateFindFetchControls`.
- Firewall lookup state is the actual member set without self (no attribute aliases are declared), so only differing values are sent and the accepted response, including case-variant unknown members, is merged by literal key.
  Tests: `network/cloud_update_source_state_test.go:TestCloudUpdateSourceStateFirewallIdentityMapping`.

Python/Go differences:

- UpdateFirewallPolicyRecord owns kwargs/filters, target-first strict lookup and ordered string-array rule translation. Lookup and RuleLookups retain separate read evidence; Response is the owned PUT receipt and Policy the native projection of its merged view. Empty and equal-update branches return a carrier rather than the same Python mutable Resource.
- The firewall Body declarations have no attribute aliases (every attribute name equals its wire key), so lookup dirty comparison and receipt merge use the actual members without self through the same Resource._consume_attrs state; a case-variant member stays an unknown member. The native model omits Merged members its schema cannot hold and reports them in UpdateReceipt.ProjectionError without failing the accepted update; Merged keeps the raw values Python would keep. NaN/Infinity bodies are invalid JSON to Go and take the BodyIgnored branch where Python would merge them (shared nonfinite JSON boundary in docs/sdk-support-ledger.md).

Unresolved boundaries:

- Rule references use JSON arrays of nonempty strings; arbitrary Python iterables, object/nonstring IDs, Source runtime exception classes and full same-instance Resource lifecycle are outside this bounded profile.
- Requests encoding/nonfinite JSON and SDK-R1/C1/S1 Resource/adapter/session/cache contracts remain separate runtime boundaries.

### `python:cloud/_network/NetworkCloudMixin/update_firewall_rule`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.UpdateFirewallRuleRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallRuleUpdateJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallRuleUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallRuleUpdateFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallRuleUpdateOpts`

- SDK-owned fourteen-key cloud attrs are validated before strict member/name Find; empty attrs still look up, equal raw kwargs skip PUT, and only dirty fields reach an SDK-private upstream builder. Explicit null/false/empty and raw ANY protocol are preserved without requiring application builders; native mutation output remains separate from lookup receipt and is nil on failure.
  Tests: `network/cloud_firewall_rule_update_test.go:TestCloudFirewallRuleUpdateLookupOnlyAndDirtySelection`, `network/cloud_firewall_rule_update_test.go:TestCloudFirewallRuleUpdateNameFallbackAndMissing`, `network/cloud_firewall_rule_update_test.go:TestCloudFirewallRuleUpdateConflictKeepsLookupAndSingleAttempt`.
- Owned JSON filters normalize falsey JSON to default strict Find kwargs and push ordinary filter fields through member/fallback lookup. Truthy nonobjects and bound name_or_id/ignore_missing collisions fail without HTTP; raw headers/microversion and fetch controls follow the Source Find phases instead of being rejected. Option snapshots, unsupported cloud keys, cancellation and selected-source mutation are checked before lookup.
  Tests: `network/cloud_firewall_rule_update_test.go:TestCloudFirewallRuleUpdateFiltersStrictBindingAndFalseyDefaults`, `network/cloud_firewall_rule_update_test.go:TestCloudFirewallRuleUpdateOwnedOptionsAndPreflight`.
- An accepted PUT (any status 200-399, as Python raise_from_response) returns Response, an owned UpdateReceipt with the actual body bytes, headers and status. Body follows Python Resource._translate_response: the resource-key member when present, otherwise the whole top-level object, without self and with unknown members preserved. Merged is the lookup wire row overlaid with every effective attribute and then Body, so a partial body keeps lookup fields; an empty, 204, truncated or otherwise unparsable body sets BodyIgnored and keeps that pre-commit state without error. Non-object JSON or a non-object resource-key member fails after remote success with the receipt and no Merged/native model. A rejected PUT has no receipt, and nothing is replayed. The native model is the projection of Merged.
  Tests: `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptMergesPythonTranslateResponse`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptNonObjectBodiesFailAfterRemoteSuccess`, `network/cloud_update_receipt_test.go:TestCloudUpdateReceiptRejectedPutHasNoReceipt`.
- Cloud filters reach Resource.find through the shared Source fetch-control profile: base_path routes the initial GET (and fallback LIST), resource_response_key selects its response member, skip_cache has no Go cache to bypass, and these kwargs seed the found Resource as unknown constructor attributes while the PUT keeps the class route and dirty body; list_base_path changes only the fallback LIST route.
  Tests: `network/cloud_firewall_find_kwargs_test.go:TestCloudFirewallUpdateFindFetchControls`.
- Firewall lookup state is the actual member set without self (no attribute aliases are declared), so only differing values are sent and the accepted response, including case-variant unknown members, is merged by literal key.
  Tests: `network/cloud_update_source_state_test.go:TestCloudUpdateSourceStateFirewallIdentityMapping`.

Python/Go differences:

- UpdateFirewallRuleRecord owns cloud options and filters, strict lookup and raw dirty selection. The dirty fourteen-key subset is sent as one owned PUT without exposing builder-interface responsibility to callers. Lookup is a separate record; Response is the owned PUT receipt and Rule the native projection of its merged view.
- The firewall Body declarations have no attribute aliases (every attribute name equals its wire key), so lookup dirty comparison and receipt merge use the actual members without self through the same Resource._consume_attrs state; a case-variant member stays an unknown member. The native model omits Merged members its schema cannot hold and reports them in UpdateReceipt.ProjectionError without failing the accepted update; Merged keeps the raw values Python would keep. NaN/Infinity bodies are invalid JSON to Go and take the BodyIgnored branch where Python would merge them (shared nonfinite JSON boundary in docs/sdk-support-ledger.md).

Unresolved boundaries:

- Full Requests encoding/nonfinite JSON, arbitrary Python object/numeric types and SDK-R1/C1/S1 Resource/adapter/session/cache lifecycle remain separate runtime boundaries.

### `python:cloud/_network/NetworkCloudMixin/insert_rule_into_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.InsertFirewallRuleIntoPolicyRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyInsertOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyInsertFilters`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyInsertAfter`
- `github.com/JSYoo5B/go-openstacksdk/network.WithFirewallPolicyInsertBefore`

- SDK-owned cloud insertion resolves strict policy with filters then strict rule without filters, uses the policy list descriptor view for membership and returns lookup-only no-op when already present before parsing positions. Missing/null membership fails only after both lookups. Position parsing then resolves after before before and keeps prior receipts on later failure.
  Tests: `network/cloud_firewall_policy_insert_rule_test.go:TestCloudFirewallPolicyInsertAlreadyPresentSkipsInvalidPositions`, `network/cloud_firewall_policy_insert_rule_test.go:TestCloudFirewallPolicyInsertMissingMembershipFailsAfterRuleLookup`, `network/cloud_firewall_policy_insert_rule_test.go:TestCloudFirewallPolicyInsertLaterPositionFailureKeepsReceipts`.
- Both or neither positions are accepted, omitted positions send explicit JSON null, and the existing InsertRuleRecord action preserves unwrapped object fields with separate header/status and native policy output. Options snapshot JSON and strict bound-filter/pre-canceled context errors prevent HTTP.
  Tests: `network/cloud_firewall_policy_insert_rule_test.go:TestCloudFirewallPolicyInsertOrderBothPositionsAndDefaultNulls`, `network/cloud_firewall_policy_insert_rule_test.go:TestCloudFirewallPolicyInsertOwnedOptionsAndPreflight`.

Python/Go differences:

- The SDK owns strict lookup, membership and delayed concrete position options; Lookup/Rule/Positions are independent read receipts and Action is the existing leaf record, not the same Python Resource instance.
- Explicit Go profile boundary: Existing action accepts HTTP200..399, not all Source100..399; strict typed response, full Requests encoding/nonfinite JSON and original action Envelope remain outside this profile.
- Explicit Go profile boundary: References and response IDs use nonempty strings with existing action path validation; arbitrary Python objects/nonstring/empty IDs/dynamic membership/runtime exception types, full raw Find control namespace and SDK-R1/C1/S1 Resource/adapter/session/cache lifecycle remain separate.

### `python:cloud/_network/NetworkCloudMixin/remove_rule_from_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.Service.RemoveRuleFromPolicyRecord`

- Cloud policy removal performs strict policy Find with owned filters before default ignore-missing rule Find without policy filters; missing rule returns the looked-up policy without inspecting null/missing membership or sending an action.
  Tests: `network/cloud_firewall_policy_remove_rule_test.go:TestCloudFirewallPolicyRemoveRuleMissingTargetAndRule`, `network/cloud_firewall_policy_remove_rule_test.go:TestCloudFirewallPolicyRemoveRuleOwnedFiltersAndFallback`.
- Source List descriptor projection and Python loaded JSON membership decide absent-rule no-op versus SDK-owned native PUT remove_rule with resolved IDs; lookup receipts remain distinct from successful native action Policy.
  Tests: `network/cloud_firewall_policy_remove_rule_test.go:TestCloudFirewallPolicyRemoveRuleMembershipAndAction`, `network/cloud_firewall_policy_remove_rule_test.go:TestCloudFirewallPolicyRemoveRuleFailureKeepsLookupProof`.
- Owned options isolate filters from callers; falsey filters default to empty kwargs, duplicate Find bindings/nil options/service replacement/cancellation fail without later HTTP.
  Tests: `network/cloud_firewall_policy_remove_rule_test.go:TestCloudFirewallPolicyRemoveRulePreflightAndFalseyFilters`, `network/cloud_firewall_policy_remove_rule_test.go:TestCloudFirewallPolicyRemoveRuleOwnedFiltersAndFallback`.

Python/Go differences:

- The cloud helper now owns policy-first and optional-rule lookups and association no-op; its action deliberately uses the existing native Gophercloud RemoveRule path, preserving a separate typed Policy rather than claiming Source Resource merge parity.
- Action HTTP status policy remains native 200-only versus Source _put_request resp.ok (<400); 202 is explicitly regression-tested as a documented Go profile difference.
- Action return remains a flat native typed Policy rather than Source applying the whole response JSON mapping, including unknown/nested fields, to a new ID-seeded Resource (the cloud action passes the lookup policy ID to Proxy; only the no-op returns the lookup Resource); action raw receipt/error metadata is not exposed.
- Action Requests response charset guessing/codecs and permissive JSON parsing remain outside the native decoder domain.
- Find/action IDs are nonempty strings; arbitrary nonstring/empty Source JSON IDs and Python path conversion/equality are not reproduced. Source location projection beyond the reused Find profile is not added by this helper.

### `python:cloud/_network_common/NetworkCommonCloudMixin/search_security_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.SearchSecurityGroupRecords`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.SearchSecurityGroups`
- `github.com/JSYoo5B/go-openstacksdk/network.SecurityGroupCloud.SearchSecurityGroups`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SecurityGroupCloud`
- `github.com/JSYoo5B/go-openstacksdk/network.NewSecurityGroupCloud`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadFiltersJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadExpression`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordProxyQuery`

- SDK-owned concrete Neutron/Nova backend selection follows Source source and network availability. Connection only falls back to Nova on clean absent Neutron catalog endpoint, never HTTP404/cancellation/joined discovery failure. Explicit Source empty/unrecognized feature errors perform no HTTP. Nova source casing and deployment/API/microversion restrictions are explicit.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaSourceSelectionAndNeutron404DoesNotFallback`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaAbsenceFallbackNormalizationAndStrictCompatibility`, `connection_security_group_records_test.go:TestConnectionSecurityGroupRecordsCleanCatalogAbsenceSelectsNova`, `connection_security_group_records_test.go:TestConnectionSecurityGroupRecordsDoNotFallbackOnJoinedFailureOrCancellation`.
- Neutron dictionary filters alone push down concrete Proxy kwargs, then the complete paged inventory receives exact ID/name/fnmatch and original cloud mapping. Top-level string/scalar/list filters request the full list without name/filter query, then JMESPath or Source lazy mapping. Falsey/underflow, ordered/nested keys and loaded numeric equality are preserved.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupSearchDictionaryPushdownThenWholeLocalSelection`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupSearchStringJMESHasNoServerQueryAndArbitraryJSON`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupOrderedNestedAndLoadedFilters`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupFalseyAndNonmappingFiltersAreLocalAfterWholeList`.
- Reachable Proxy headers/version/BasePath/paginated/max_items and conflicting-attrs formals use an opt-in Source family lane. Opaque generator JMESPath is evaluated before HTTP; falsey literal search skip, earlier identifier get failure and nested generator membership consumption retain actual receipts without fabricated Resource/Wire.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupProxyLiteralsAndNestedMembershipDoNotInventRows`, `network/v2/extensions/security/groups/python_list_proxy_test.go:TestPythonSecurityGroupSourceProxyOpaqueJMESAndLegacyTypedBoundary`, `network/v2/extensions/security/groups/python_list_proxy_test.go:TestPythonSecurityGroupSourceProxyFormalsBasePathAndSnapshot`, `network/v2/extensions/security/groups/python_list_proxy_test.go:TestPythonSecurityGroupSourceProxyDuplicateControlsAndLazyContext`.
- One Nova legacy GET is normalized by library-owned Source contracts: raw numeric/string IDs, eager pop/default precedence, artifact removal, rules aliases, properties, strict compatibility, loaded current/foreign project location, Python minimum int conversion and unchanged Source maximum-port string bug. Malformed/partial normalization preserves actual NovaResponse/envelope/status/header.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaAbsenceFallbackNormalizationAndStrictCompatibility`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaDirectGetAndEagerRuleDefaultsPreserveSourceBug`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaMalformedRowsKeepActualPartialProof`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaFalseyContainersAndQueryShapes`.
- Input options/filters/location, normalized Resource/raw Wire, selected records and partial Inventory/LastGroup own independent bytes and headers. Selected original API/client/route is captured before callbacks. Context cancellation/source mutation and real late 200/203/401/404/429/503 body faults preserve original native pointer401 causes and response receipt while stopping retry/backoff/reauth fault hooks and successful partial selection.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupReadOwnsOptionsLocationAndExportedCarriers`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupReadSourceGuardCancellationAndPreflight`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupReadLatePhysicalPagesKeepNativeReceiptsAndStopHooks`, `connection_security_group_records_test.go:TestConnectionSecurityGroupRecordsNeutronDefaultsAndLocationBeforeFilter`.

Python/Go differences:

- Go returns SecurityGroupReadResult with actual backend/Inventory/LastGroup/NovaResponse evidence and independent Groups/Group/Values/Value. JMES arbitrary JSON has Expression=true and no invented record correlation; Nova logical normalized Munch view is distinguished from its actual Nova Wire, not claimed to be a Neutron class instance.
- Go exposes concrete SecurityGroupReadDependencies and typed transport/current-location options, with owned resets and source invariants instead of developer interfaces/builders. It restricts transport headers/version/BasePath/route formatting/max_items and finite JSON input to safe Go representations; explicit Source policy flags are factory data while Connection supplies default backend/location without expanding global configuration scope.
- Go preserves the Source minimum conversion and maximum-port string normalization bug, eager pop/default behavior, source casing and strict compatibility. Nova proxy availability is conditional on actual deployment/API/microversion; modern Nova 2.36 proxy404 is not represented as universally supported fallback.
- Go lexical dict-key/JMESPath key-function order, omitted deprecation/log warnings, escaped service-relative routes, fixed request guards and stronger paging/cycle/Link safety are documented differences. Python arbitrary object/attribute/repr addresses, mutable Resource/dirty/session/cache and lazy materialization behavior remain SDK-R1/C1/S1 generic scope, separate from authenticated live cloud acceptance.

### `python:cloud/_network_common/NetworkCommonCloudMixin/get_security_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.GetSecurityGroupRecord`
- `github.com/JSYoo5B/go-openstacksdk/network.Service.GetSecurityGroup`
- `github.com/JSYoo5B/go-openstacksdk/network.SecurityGroupCloud.GetSecurityGroup`
- `github.com/JSYoo5B/go-openstacksdk/network.ExistingSecurityGroupRead`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.GetSourceRecord`
- `github.com/JSYoo5B/go-openstacksdk.Connection.SecurityGroupCloud`
- `github.com/JSYoo5B/go-openstacksdk/network.NewSecurityGroupCloud`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadOpts`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadFiltersJSON`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadExpression`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network.WithSecurityGroupReadCurrentLocation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/groups.WithRecordProxyQuery`

- SDK-owned concrete Neutron/Nova backend selection follows Source source and network availability. Connection only falls back to Nova on clean absent Neutron catalog endpoint, never HTTP404/cancellation/joined discovery failure. Explicit Source empty/unrecognized feature errors perform no HTTP. Nova source casing and deployment/API/microversion restrictions are explicit.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaSourceSelectionAndNeutron404DoesNotFallback`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaAbsenceFallbackNormalizationAndStrictCompatibility`, `connection_security_group_records_test.go:TestConnectionSecurityGroupRecordsCleanCatalogAbsenceSelectsNova`, `connection_security_group_records_test.go:TestConnectionSecurityGroupRecordsDoNotFallbackOnJoinedFailureOrCancellation`.
- Neutron dictionary filters alone push down concrete Proxy kwargs, then the complete paged inventory receives exact ID/name/fnmatch and original cloud mapping. Top-level string/scalar/list filters request the full list without name/filter query, then JMESPath or Source lazy mapping. Falsey/underflow, ordered/nested keys and loaded numeric equality are preserved.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupSearchDictionaryPushdownThenWholeLocalSelection`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupSearchStringJMESHasNoServerQueryAndArbitraryJSON`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupOrderedNestedAndLoadedFilters`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupFalseyAndNonmappingFiltersAreLocalAfterWholeList`.
- Reachable Proxy headers/version/BasePath/paginated/max_items and conflicting-attrs formals use an opt-in Source family lane. Opaque generator JMESPath is evaluated before HTTP; falsey literal search skip, earlier identifier get failure and nested generator membership consumption retain actual receipts without fabricated Resource/Wire.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupProxyLiteralsAndNestedMembershipDoNotInventRows`, `network/v2/extensions/security/groups/python_list_proxy_test.go:TestPythonSecurityGroupSourceProxyOpaqueJMESAndLegacyTypedBoundary`, `network/v2/extensions/security/groups/python_list_proxy_test.go:TestPythonSecurityGroupSourceProxyFormalsBasePathAndSnapshot`, `network/v2/extensions/security/groups/python_list_proxy_test.go:TestPythonSecurityGroupSourceProxyDuplicateControlsAndLazyContext`.
- One Nova legacy GET is normalized by library-owned Source contracts: raw numeric/string IDs, eager pop/default precedence, artifact removal, rules aliases, properties, strict compatibility, loaded current/foreign project location, Python minimum int conversion and unchanged Source maximum-port string bug. Malformed/partial normalization preserves actual NovaResponse/envelope/status/header.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaAbsenceFallbackNormalizationAndStrictCompatibility`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaDirectGetAndEagerRuleDefaultsPreserveSourceBug`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaMalformedRowsKeepActualPartialProof`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaFalseyContainersAndQueryShapes`.
- Input options/filters/location, normalized Resource/raw Wire, selected records and partial Inventory/LastGroup own independent bytes and headers. Selected original API/client/route is captured before callbacks. Context cancellation/source mutation and real late 200/203/401/404/429/503 body faults preserve original native pointer401 causes and response receipt while stopping retry/backoff/reauth fault hooks and successful partial selection.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupReadOwnsOptionsLocationAndExportedCarriers`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupReadSourceGuardCancellationAndPreflight`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupReadLatePhysicalPagesKeepNativeReceiptsAndStopHooks`, `connection_security_group_records_test.go:TestConnectionSecurityGroupRecordsNeutronDefaultsAndLocationBeforeFilter`.
- Default getter searches the complete list and applies Source truthiness/len/index0: empty/falsey means None, one ordinary record means Group, duplicates are typed ambiguity, and expression false/0/null/string/object/scalar cases preserve Source distinction. Explicit UUID-like direct getter ignores filters and keeps member404; it never creates implicit GET-first Find behavior.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupGetDefaultListsAllAndReturnsNoneOneOrAmbiguity`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupGetExpressionFirstScalarAndFalseySemantics`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupDirectUUIDGetterIgnoresSearchFiltersAndKeeps404`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupNovaDirectGetAndEagerRuleDefaultsPreserveSourceBug`.
- Already fetched SecurityGroupRecord, RawResource or JSON dictionary with id present (including null/false/zero) is cloned through public pure ExistingSecurityGroupRead before backend discovery or user callbacks; Connection uses the same helper and does not invent transport proof.
  Tests: `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupExistingReferenceBypassesFeatureDiscoveryAndCallbacks`, `connection_security_group_records_test.go:TestConnectionSecurityGroupExistingReferenceBypassesDiscoveryAndOptions`.

Python/Go differences:

- Go returns SecurityGroupReadResult with actual backend/Inventory/LastGroup/NovaResponse evidence and independent Groups/Group/Values/Value. JMES arbitrary JSON has Expression=true and no invented record correlation; Nova logical normalized Munch view is distinguished from its actual Nova Wire, not claimed to be a Neutron class instance.
- Go exposes concrete SecurityGroupReadDependencies and typed transport/current-location options, with owned resets and source invariants instead of developer interfaces/builders. It restricts transport headers/version/BasePath/route formatting/max_items and finite JSON input to safe Go representations; explicit Source policy flags are factory data while Connection supplies default backend/location without expanding global configuration scope.
- Go preserves the Source minimum conversion and maximum-port string normalization bug, eager pop/default behavior, source casing and strict compatibility. Nova proxy availability is conditional on actual deployment/API/microversion; modern Nova 2.36 proxy404 is not represented as universally supported fallback.
- Go lexical dict-key/JMESPath key-function order, omitted deprecation/log warnings, escaped service-relative routes, fixed request guards and stronger paging/cycle/Link safety are documented differences. Python arbitrary object/attribute/repr addresses, mutable Resource/dirty/session/cache and lazy materialization behavior remain SDK-R1/C1/S1 generic scope, separate from authenticated live cloud acceptance.

### `python:cloud/_network_common/NetworkCommonCloudMixin/update_security_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.SecurityGroupCloud.UpdateSecurityGroup`
- `github.com/JSYoo5B/go-openstacksdk/network.SecurityGroupCloud.GetSecurityGroup`

- The cloud lookup lists every group without a query and selects locally; duplicate matches fail before any mutation and a disabled security group feature sends no request.
  Tests: `api/py_cloud_net_security_group_writes_test.go:TestPyCloudNetSecurityGroupDeleteStrictAfterLookupAndAmbiguous`, `api/py_cloud_net_security_group_writes_test.go:TestPyCloudNetSecurityGroupDeleteFeatureDisabledSendsNothing`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupGetDefaultListsAllAndReturnsNoneOneOrAmbiguity`.
- Neutron sends only supplied name/description/stateful, including empty string and false, sends no request when nothing is supplied, and a missing group is resource.ErrNotFound before any mutation.
  Tests: `api/py_cloud_net_security_group_writes_test.go:TestPyCloudNetSecurityGroupUpdateNeutronSendsOnlySuppliedIncludingFalseAndEmpty`.
- Nova fills a missing name and description from the found group before PUT /os-security-groups/{id}.
  Tests: `api/py_cloud_net_security_group_writes_test.go:TestPyCloudNetSecurityGroupUpdateNovaFillsNameAndDescription`.

Python/Go differences:

- Go exposes concrete options (SecurityGroupUpdate, SecurityGroupRuleSpec) instead of Python **kwargs; unknown keys cannot be passed.
- Nova responses are returned as raw security_group/security_group_rule JSON (NovaBody) without the Python munch normalization; Neutron results use native SecGroup/SecGroupRule models instead of Resource objects.
- Python selects the backend from the cloud secgroup_source and network availability; Go takes the same inputs from SecurityGroupReadDependencies (the Connection supplies them).

### `python:cloud/_network_common/NetworkCommonCloudMixin/delete_security_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.SecurityGroupCloud.DeleteSecurityGroup`
- `github.com/JSYoo5B/go-openstacksdk/network.SecurityGroupCloud.GetSecurityGroup`

- The cloud lookup lists every group without a query and selects locally; duplicate matches fail before any mutation and a disabled security group feature sends no request.
  Tests: `api/py_cloud_net_security_group_writes_test.go:TestPyCloudNetSecurityGroupDeleteStrictAfterLookupAndAmbiguous`, `api/py_cloud_net_security_group_writes_test.go:TestPyCloudNetSecurityGroupDeleteFeatureDisabledSendsNothing`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupGetDefaultListsAllAndReturnsNoneOneOrAmbiguity`.
- Neutron lists then deletes once and returns false without a mutation when nothing matches; a 404 on the delete after the lookup is an error (ignore_missing=False).
  Tests: `api/py_cloud_net_security_group_writes_test.go:TestPyCloudNetSecurityGroupDeleteNeutronListsThenDeletesOnceAndMissingIsFalse`, `api/py_cloud_net_security_group_writes_test.go:TestPyCloudNetSecurityGroupDeleteStrictAfterLookupAndAmbiguous`.
- The Nova backend deletes through DELETE /os-security-groups/{id}.
  Tests: `api/py_cloud_net_security_group_writes_test.go:TestPyCloudNetSecurityGroupDeleteNovaUsesOSSecurityGroups`.

Python/Go differences:

- Go exposes concrete options (SecurityGroupUpdate, SecurityGroupRuleSpec) instead of Python **kwargs; unknown keys cannot be passed.
- Nova responses are returned as raw security_group/security_group_rule JSON (NovaBody) without the Python munch normalization; Neutron results use native SecGroup/SecGroupRule models instead of Resource objects.
- Python selects the backend from the cloud secgroup_source and network availability; Go takes the same inputs from SecurityGroupReadDependencies (the Connection supplies them).
- Python returns True/False; Go returns the same boolean plus an error value.

### `python:cloud/_network_common/NetworkCommonCloudMixin/create_security_group_rule`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network.SecurityGroupCloud.CreateSecurityGroupRule`
- `github.com/JSYoo5B/go-openstacksdk/network.SecurityGroupCloud.GetSecurityGroup`

- The cloud lookup lists every group without a query and selects locally; duplicate matches fail before any mutation and a disabled security group feature sends no request.
  Tests: `api/py_cloud_net_security_group_writes_test.go:TestPyCloudNetSecurityGroupDeleteStrictAfterLookupAndAmbiguous`, `api/py_cloud_net_security_group_writes_test.go:TestPyCloudNetSecurityGroupDeleteFeatureDisabledSendsNothing`, `network/cloud_security_group_reads_test.go:TestCloudSecurityGroupGetDefaultListsAllAndReturnsNoneOneOrAmbiguity`.
- Neutron maps port -1 to null, keeps null body keys, defaults direction/ethertype and adds tenant_id and description only when supplied.
  Tests: `api/py_cloud_net_security_group_writes_test.go:TestPyCloudNetSecurityGroupRuleNeutronBodyKeepsNullsAndMapsMinusOne`, `api/py_cloud_net_security_group_writes_test.go:TestPyCloudNetSecurityGroupRuleNeutronProjectAndDescriptionOnlyWhenGiven`.
- Nova requires a protocol, rejects egress before mutation, and fills ICMP -1 and TCP/UDP 1-65535 port defaults.
  Tests: `api/py_cloud_net_security_group_writes_test.go:TestPyCloudNetSecurityGroupRuleNovaRulesAndValidation`.

Python/Go differences:

- Go exposes concrete options (SecurityGroupUpdate, SecurityGroupRuleSpec) instead of Python **kwargs; unknown keys cannot be passed.
- Nova responses are returned as raw security_group/security_group_rule JSON (NovaBody) without the Python munch normalization; Neutron results use native SecGroup/SecGroupRule models instead of Resource objects.
- Python selects the backend from the cloud secgroup_source and network availability; Go takes the same inputs from SecurityGroupReadDependencies (the Connection supplies them).
- Python project_id maps to the tenant_id body key; Go ProjectID does the same, and a pointer distinguishes unset from an empty string.

### `python:cloud/_network_common/NetworkCommonCloudMixin/add_auto_ip`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.AddAutoIP`
- `github.com/JSYoo5B/go-openstacksdk.Connection.AddAutoIP`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAddAutoIPWait`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAddAutoIPReuse`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAddAutoIPTimeout`
- `github.com/JSYoo5B/go-openstacksdk/compute.WithAddAutoIPCleanup`

- Nova reuse: available IP selection, exact-address attach, address wait and interface IP; the caller server is not mutated.
  Tests: `connection_r2_cloud_net_ip_test.go:TestR2CloudNetAddAutoIPNovaReuseAttachesAndWaits`.
- A server already showing the floating address is not attached again.
  Tests: `connection_r2_cloud_net_ip_test.go:TestR2CloudNetAddAutoIPAlreadyAttachedSkipsAttach`.
- Neutron allocation (reuse=false) skips the attach, waits for the address and does not delete on success.
  Tests: `connection_r2_cloud_net_ip_test.go:TestR2CloudNetAddAutoIPNeutronCreateSkipsAttachAndWaits`.
- A wait timeout is a deadline error; a Neutron IP created by the call is deleted once unless cleanup is disabled, and Nova IPs are never deleted.
  Tests: `connection_r2_cloud_net_ip_test.go:TestR2CloudNetAddAutoIPNeutronCreateTimeoutDeletesOnceUnlessDisabled`, `connection_r2_cloud_net_ip_test.go:TestR2CloudNetAddAutoIPWaitTimeoutOnNovaReportsDeadlineWithoutCleanup`.
- Invalid server/options and canceled contexts send nothing.
  Tests: `connection_r2_cloud_net_ip_test.go:TestR2CloudNetAddAutoIPValidation`.
- With the Neutron floating source, an allocation that falls back to Nova after a Neutron 404 (Backend nova, NeutronFloating true) is not attached by either backend, as Python sets skip_attach and honours it in its Neutron-source branch. On a wait timeout that IP is cleaned up through the configured Neutron delete exactly once (a Neutron 404 there is a not-deleted result, not a Nova delete) unless cleanup is disabled.
  Tests: `connection_cloud_net_ip_fallback_test.go:TestCloudNetAddAutoIPNeutronSourceNovaFallbackAllocationSkipsAttach`, `connection_cloud_net_ip_fallback_test.go:TestCloudNetAddAutoIPNeutronSourceNovaFallbackTimeoutCleansUp`.

Python/Go differences:

- Attach goes through the explicit-address workflow (extra address lookup) instead of a direct attach; results are an AddAutoIPResult with evidence instead of only the interface IP string.
- Cleanup after a timeout is an explicit option (default on, as Python) and its failure joins the timeout error instead of replacing it.

Unresolved boundaries:

- Dynamic Server/Resource inputs, configuration/vendor aliases, source-version and discovery policy are not compared; the reuse path that receives a Nova-fallback row under the Neutron source still attaches through the explicit-address workflow instead of Python's Neutron attach that swallows NotFound; no authenticated cloud interoperability is claimed.

### `python:cloud/_network_common/NetworkCommonCloudMixin/detach_ip_from_server`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute.Service.DetachIPFromServer`
- `github.com/JSYoo5B/go-openstacksdk.Connection.DetachIPFromServer`

- Neutron: lookup, then exactly one PUT floatingips/{id} with port_id null; the result records Detached and the accepted response.
  Tests: `connection_r2_cloud_net_ip_test.go:TestR2CloudNetDetachNeutronSendsOneNullPortUpdate`.
- Neutron: an absent row or a null/empty port_id returns not_attached with no mutation.
  Tests: `connection_r2_cloud_net_ip_test.go:TestR2CloudNetDetachNeutronReturnsFalseWithoutMutation`.
- Neutron update failures (404/409/500) are wrapped, retain the response and never fall back to Nova or replay the PUT.
  Tests: `connection_r2_cloud_net_ip_test.go:TestR2CloudNetDetachNeutronUpdateFailureNeverFallsBackToNova`.
- Nova: one removeFloatingIp action with the row address; a missing IP fails before the action; a rejected action is reported once.
  Tests: `connection_r2_cloud_net_ip_test.go:TestR2CloudNetDetachNovaSendsRemoveFloatingIPAction`, `connection_r2_cloud_net_ip_test.go:TestR2CloudNetDetachNovaMissingIPFailsBeforeAction`, `connection_r2_cloud_net_ip_test.go:TestR2CloudNetDetachNovaActionRejectionIsReportedOnce`.
- Invalid IDs, a nil connection and pre-canceled contexts send nothing; cancellation during the PUT does not replay it.
  Tests: `connection_r2_cloud_net_ip_test.go:TestR2CloudNetDetachValidatesInputsAndContextBeforeHTTP`, `connection_r2_cloud_net_ip_test.go:TestR2CloudNetDetachCancellationDuringUpdateDoesNotReplay`.
- With the Neutron floating source the Neutron branch runs even when the lookup fell back to normalized Nova rows: a falsy port_id returns not_attached without HTTP mutation, a truthy port_id sends the one Neutron PUT whose failure is a detach error, and a strict-mode row without port_id fails before mutation. A clean Neutron lookup 404 records NeutronLookup/NeutronNotFound and runs the Nova branch, which repeats the lookup.
  Tests: `connection_cloud_net_ip_fallback_test.go:TestCloudNetDetachNeutronSourceRunsNeutronBranchOnNovaFallbackRow`, `connection_cloud_net_ip_fallback_test.go:TestCloudNetDetachNeutronSourceTruthyFallbackPortUsesNeutronUpdate`, `connection_cloud_net_ip_fallback_test.go:TestCloudNetDetachNeutronSourceStrictFallbackRowHasNoPortID`, `connection_cloud_net_ip_fallback_test.go:TestCloudNetDetachNeutronLookupNotFoundRunsNovaBranch`.

Python/Go differences:

- The result is a DetachFloatingIPResult with an outcome enum; Python returns True/False/None. Nova acknowledgement is not proof of detachment.
- Server and floating IP IDs are validated up front; Go options replace the cloud kwargs.
- Python Munch/Resource attribute access is modeled on the owned normalized JSON row: port_id uses Python JSON truthiness, and a strict-mode Nova row without port_id is an ErrInvalidOption instead of AttributeError. Non-string Python ID objects and Requests/session behavior are the shared arbitrary-object and Requests boundaries in docs/sdk-support-ledger.md; the Neutron-versus-Nova branch follows the configured floating IP source and network service availability.

## References

- [`openstack/cloud/_network.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py)
- [`openstack/cloud/_network_common.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py)
- [`python_parity_cloud_b_test.go`](python_parity_cloud_b_test.go)
- [get_subnet_by_id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L435)
- [get_subnetpool](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L478)
- [list_routers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L168)
- [search_routers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L77)
- [create_security_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1931)
- [get_security_group_by_id](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1906)
- [list_security_groups](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1845)
- [delete_security_group_rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L2219)
- [list_networks](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L151)
- [Cloud Network Lists Python](cloud-network-lists-python.md)
- [search_networks](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L53)
- [list_subnets](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L185)
- [Cloud Subnet Lists Python](cloud-subnet-lists-python.md)
- [search_subnets](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L101)
- [list_ports](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L202)
- [Python User Helpers](v2/ports/python-user-helpers.md)
- [search_ports](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L125)
- [Python List Records](v2/ports/python-list-records.md)
- [list_router_interfaces](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2205)
- [Cloud Router Interfaces Python](cloud-router-interfaces-python.md)
- [remove_router_interface](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2174)
- [Cloud Router Interface Remove Python](cloud-router-interface-remove-python.md)
- [update_router](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2309)
- [update_subnet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2591)
- [update_port](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2776)
- [list_qos_policies](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L333)
- [Cloud QoS Policies Python](cloud-qos-policies-python.md)
- [search_qos_policies](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L253)
- [update_qos_policy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1468)
- [Cloud QoS Policy Mutations Python](cloud-qos-policy-mutations-python.md)
- [list_qos_rule_types](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L281)
- [Cloud QoS Rule Types Python](cloud-qos-rule-types-python.md)
- [list_qos_bandwidth_limit_rules](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1557)
- [Cloud QoS Bandwidth Python](cloud-qos-bandwidth-python.md)
- [search_qos_bandwidth_limit_rules](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1535)
- [update_qos_bandwidth_limit_rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1667)
- [Cloud QoS Rule Updates Python](cloud-qos-rule-updates-python.md)
- [list_qos_dscp_marking_rules](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1786)
- [Cloud QoS Dscp Python](cloud-qos-dscp-python.md)
- [search_qos_dscp_marking_rules](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1764)
- [update_qos_dscp_marking_rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1877)
- [list_qos_minimum_bandwidth_rules](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1983)
- [Cloud QoS Minimum Bandwidth Python](cloud-qos-minimum-bandwidth-python.md)
- [search_qos_minimum_bandwidth_rules](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1959)
- [update_qos_minimum_bandwidth_rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L2073)
- [list_firewall_groups](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1350)
- [list_firewall_policies](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1079)
- [list_firewall_rules](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L874)
- [update_firewall_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1372)
- [Cloud Firewall Group Update Python](cloud-firewall-group-update-python.md)
- [update_firewall_policy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1110)
- [Cloud Firewall Policy Update Python](cloud-firewall-policy-update-python.md)
- [update_firewall_rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L913)
- [Cloud Firewall Rule Update Python](cloud-firewall-rule-update-python.md)
- [insert_rule_into_policy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1150)
- [Cloud Firewall Policy Insert Rule Python](cloud-firewall-policy-insert-rule-python.md)
- [remove_rule_from_policy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network.py#L1211)
- [Cloud Firewall Policy Remove Rule Python](cloud-firewall-policy-remove-rule-python.md)
- [search_security_groups](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1834)
- [Cloud Security Group Reads Python](cloud-security-group-reads-python.md)
- [get_security_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1878)
- [update_security_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L2025)
- [delete_security_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1983)
- [create_security_group_rule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L2067)
- [add_auto_ip](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1394)
- [detach_ip_from_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_network_common.py#L1214)
- [rule types](cloud-qos-rule-types-python.md)
- [bandwidth](cloud-qos-bandwidth-python.md)
- [DSCP](cloud-qos-dscp-python.md)
- [minimum bandwidth](cloud-qos-minimum-bandwidth-python.md)
