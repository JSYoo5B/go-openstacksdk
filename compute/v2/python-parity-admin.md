# Python Parity Admin

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/aggregates`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.All`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.List`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithName`

- List and All send GET os-aggregates without a query, and All with resource.WithName filters the decoded rows locally like Python's Body attribute filter.
  Tests: `compute/v2/aggregates/python_parity_test.go:TestPythonAggregateCreateGetListDeleteMatchProxyRequests`.

Python/Go differences:

- Python can send limit and marker query keys, which Nova's aggregate list ignores; the Go aggregate list accepts no wire query.
- Go reads one page and does not follow aggregates_links.
- Python sends min(Aggregate _max_microversion 2.81, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/get_aggregate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.Get`

- Get sends GET os-aggregates/{id} and decodes id, uuid, name, availability_zone, hosts and metadata.
  Tests: `compute/v2/aggregates/python_parity_test.go:TestPythonAggregateCreateGetListDeleteMatchProxyRequests`.

Python/Go differences:

- Python accepts an Aggregate or an ID string; Go generated calls take the integer aggregate ID.
- Go decodes metadata values as strings and the timestamps only in the zone-less Nova format.
- Python sends min(Aggregate _max_microversion 2.81, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/create_aggregate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.Create`

- Create sends POST os-aggregates with {"aggregate": {name, availability_zone}} and returns the decoded aggregate.
  Tests: `compute/v2/aggregates/python_parity_test.go:TestPythonAggregateCreateGetListDeleteMatchProxyRequests`.

Python/Go differences:

- Python sends only passed attributes and may send availability_zone null; Go omits an empty AvailabilityZone, which Nova treats the same as null on create.
- Go rejects an empty Name before HTTP.
- Python sends min(Aggregate _max_microversion 2.81, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/delete_aggregate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE os-aggregates/{id}, ignores 404 by default and reports NotFound with WithMissingError.
  Tests: `compute/v2/aggregates/python_parity_test.go:TestPythonAggregateCreateGetListDeleteMatchProxyRequests`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- Remove parses the reference as an integer and rejects a non-numeric ID before HTTP.
- Python sends min(Aggregate _max_microversion 2.81, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/add_host_to_aggregate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.AddHost`

- AddHost posts {"add_host": {"host": h}} to os-aggregates/{id}/action and returns the response aggregate.
  Tests: `compute/v2/aggregates/python_parity_test.go:TestPythonAggregateHostAndMetadataActionsReturnAggregate`.

Python/Go differences:

- Python accepts an Aggregate or an ID string; Go generated calls take the integer aggregate ID.
- Go rejects an empty host before HTTP; Nova rejects it as well.
- Python sends no microversion for this request unless the session has a default; Go always sends the Connection client microversion.

### `python:compute/v2/remove_host_from_aggregate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.RemoveHost`

- RemoveHost posts {"remove_host": {"host": h}} to os-aggregates/{id}/action and returns the response aggregate.
  Tests: `compute/v2/aggregates/python_parity_test.go:TestPythonAggregateHostAndMetadataActionsReturnAggregate`.

Python/Go differences:

- Python accepts an Aggregate or an ID string; Go generated calls take the integer aggregate ID.
- Go rejects an empty host before HTTP; Nova rejects it as well.
- Python sends no microversion for this request unless the session has a default; Go always sends the Connection client microversion.

### `python:compute/v2/set_aggregate_metadata`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.SetMetadata`

- SetMetadata posts {"set_metadata": {"metadata": {...}}}, sends nil values as null and an empty map as {} like Python's call without metadata.
  Tests: `compute/v2/aggregates/python_parity_test.go:TestPythonAggregateHostAndMetadataActionsReturnAggregate`.

Python/Go differences:

- Python accepts an Aggregate or an ID string; Go generated calls take the integer aggregate ID.
- Python's call without metadata sends {}; Go needs an explicit empty map because a nil Metadata map is rejected before HTTP.
- Python sends no microversion for this request unless the session has a default; Go always sends the Connection client microversion.

### `python:compute/v2/update_aggregate`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithUpdateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithUpdateRecordName`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithUpdateRecordAvailabilityZone`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithUpdateRecordField`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates.WithUpdateRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/request.Null`
- `github.com/JSYoo5B/go-openstacksdk/request.Present`

- UpdateRecord sends PUT os-aggregates/{id} with only the supplied attributes in the aggregate envelope, including availability_zone null, an empty name and extension attributes, accepts any status below 400, decodes enveloped and flat responses, and sends nothing when no attribute is supplied.
  Tests: `compute/v2/aggregates/python_update_record_test.go:TestPythonAggregateUpdateRecordSendsNullAndEmptyValues`.
- Extension fields that collide with typed keys or the id identity and nil options fail before HTTP, and a 409 response is returned as the Gophercloud status error wrapped in resource.OperationError.
  Tests: `compute/v2/aggregates/python_update_record_test.go:TestPythonAggregateUpdateRecordRejectsInputsAndErrorStatuses`.

Python/Go differences:

- Python accepts an Aggregate or an ID string; Go takes the integer aggregate ID.
- Python sends min(Aggregate _max_microversion 2.81, server maximum) discovered per request; Go sends the one microversion configured on the client.
- Python drops undeclared keyword attributes; WithUpdateRecordField sends any extension key except the typed keys and id.
- Python returns the Resource merged from request attributes and the response; Go returns AggregateRecord with the native typed decode, the raw response object fields, headers and status, and Sent false when no request was needed.

### `python:compute/v2/create_flavor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.WithCreateField`

- Create sends POST flavors with name, ram, vcpus, disk, id, os-flavor-access:is_public, OS-FLV-EXT-DATA:ephemeral and description, carries other keys through WithCreateField and decodes the flavor.
  Tests: `compute/v2/flavors/python_parity_test.go:TestPythonFlavorCreateDeleteMatchProxyRequests`.

Python/Go differences:

- Python sends only passed attributes; Go always sends name, ram, vcpus and disk, which Nova requires anyway.
- Go omits an empty description and a zero rxtx_factor; Nova rejects a zero rxtx_factor and stores no description for both.
- flavors.Flavor has no OS-FLV-DISABLED:disabled or original_name field.
- Python sends min(Flavor _max_microversion 2.61, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/delete_flavor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE flavors/{id}, accepts 202, ignores 404 by default and reports NotFound with WithMissingError.
  Tests: `compute/v2/flavors/python_parity_test.go:TestPythonFlavorCreateDeleteMatchProxyRequests`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- Python sends min(Flavor _max_microversion 2.61, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/flavor_add_tenant_access`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.AddAccess`

- AddAccess posts {"addTenantAccess": {"tenant": t}} to flavors/{id}/action.
  Tests: `compute/v2/flavors/python_parity_test.go:TestPythonFlavorTenantAccessMatchesProxyRequests`.

Python/Go differences:

- Python returns None and sends an empty Accept header; Go returns the response flavor_access rows with the default Accept header.
- Python sends no microversion for this request unless the session has a default; Go always sends the Connection client microversion.

### `python:compute/v2/flavor_remove_tenant_access`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.RemoveAccess`

- RemoveAccess posts {"removeTenantAccess": {"tenant": t}} to flavors/{id}/action.
  Tests: `compute/v2/flavors/python_parity_test.go:TestPythonFlavorTenantAccessMatchesProxyRequests`.

Python/Go differences:

- Python returns None and sends an empty Accept header; Go returns the response flavor_access rows with the default Accept header.
- Python sends no microversion for this request unless the session has a default; Go always sends the Connection client microversion.

### `python:compute/v2/get_flavor_access`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.ListAccesses`

- ListAccesses sends GET flavors/{id}/os-flavor-access and yields flavor_id and tenant_id rows.
  Tests: `compute/v2/flavors/python_parity_test.go:TestPythonFlavorTenantAccessMatchesProxyRequests`.

Python/Go differences:

- Python returns the raw flavor_access dicts as a list; Go yields typed FlavorAccess rows one by one.
- Python sends no microversion for this request unless the session has a default; Go always sends the Connection client microversion.

### `python:compute/v2/create_flavor_extra_specs`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.CreateExtraSpecs`

- CreateExtraSpecs posts {"extra_specs": {...}} to flavors/{id}/os-extra_specs and returns the merged response specs.
  Tests: `compute/v2/flavors/python_parity_test.go:TestPythonFlavorExtraSpecsWriteRequests`.

Python/Go differences:

- Python returns a Flavor holding the response extra_specs; Go returns the map.
- Go ExtraSpecsOpts holds string values, so numeric values are passed in string form, which Nova stores identically.
- Python sends min(Flavor _max_microversion 2.61, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/update_flavor_extra_specs_property`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.UpdateExtraSpec`

- UpdateExtraSpec sends PUT flavors/{id}/os-extra_specs/{key} with {key: value} and returns the response map.
  Tests: `compute/v2/flavors/python_parity_test.go:TestPythonFlavorExtraSpecsWriteRequests`.

Python/Go differences:

- Python returns the value under the key; Go returns the response map.
- Go rejects an ExtraSpecsOpts without exactly one pair before HTTP.
- Python sends min(Flavor _max_microversion 2.61, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/delete_flavor_extra_specs_property`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/flavors.API.DeleteExtraSpec`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- DeleteExtraSpec sends DELETE flavors/{id}/os-extra_specs/{key} and returns the 404 as an error.
  Tests: `compute/v2/flavors/python_parity_test.go:TestPythonFlavorExtraSpecsWriteRequests`.
- Wrapping the direct delete in resource.IgnoreMissing turns a plain HTTP 404 into success like Python's default ignore_missing=True, keeps other failures such as 409 as errors, and leaves the DELETE request unchanged.
  Tests: `resource/missing_test.go:TestIgnoreMissingDropsOnlyPlain404`, `compute/v2/flavors/python_ignore_missing_test.go:TestPythonDeleteIgnoreMissingFlavorExtraSpec`.

Python/Go differences:

- Python sends min(Flavor _max_microversion 2.61, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python ignore_missing=True is the default; Go selects it by wrapping the call in resource.IgnoreMissing, and the bare call returns the 404 like ignore_missing=False.

### `python:compute/v2/extensions`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/extensions.API.List`

- List sends GET extensions and decodes alias, name, namespace, updated and description.
  Tests: `compute/v2/extensions/python_parity_test.go:TestPythonExtensionListAndGetHaveNoFindFallback`.

Python/Go differences:

- Python sends no microversion for this request unless the session has a default; Go always sends the Connection client microversion.

### `python:compute/v2/find_extension`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/extensions.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/extensions.API.FindRecord`

- Get sends GET extensions/{alias} and returns the 404 for a name without listing extensions.
  Tests: `compute/v2/extensions/python_parity_test.go:TestPythonExtensionListAndGetHaveNoFindFallback`.
- Concrete Source-owned compute-role projection, library-owned callback snapshots and selected/versionless policy retain original client and actual response evidence. Source-aware lookup/list parsing shares guarded REST, descriptor MRO, Python-loaded equality and paging rather than new transport wrappers.
  Tests: `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindMemberOwnsSourceMROWithoutDiscovery`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindCleanFallbackAliasNameMissingAndAmbiguity`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindStrictMissingKeepsExistingNativeAPIs`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindOwnedKwargsAndGetOnlyVersion`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindLaterPageAndSourcePrimaryPresence`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindOptionSourceGuardAndCancellation`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindPhysicalRejectionsNeverFallbackOrRunHooks`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindCleanAuthenticationRenewalKeepsMember`.
- GET-first literal extensions path suppresses only clean400403404; complete fallback pages match exact alias-derived id or name, detect duplicate and default missingnil/strictErrNotFound. Compute Extension unknownBody=false includes namespace, links=list_type=dict, updated alias and untyped fields; maxNone yields no default discovery and explicit override is GET-only. Physical401 pointer native public receipt and all causes survive without fallback/retry/auth/backoff.
  Tests: `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindMemberOwnsSourceMROWithoutDiscovery`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindCleanFallbackAliasNameMissingAndAmbiguity`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindStrictMissingKeepsExistingNativeAPIs`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindOwnedKwargsAndGetOnlyVersion`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindLaterPageAndSourcePrimaryPresence`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindOptionSourceGuardAndCancellation`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindPhysicalRejectionsNeverFallbackOrRunHooks`, `compute/v2/extensions/python_record_find_contracts_test.go:TestComputeExtensionFindCleanAuthenticationRenewalKeepsMember`.

Python/Go differences:

- Go concrete Source records separate Resource default/projection from complete actual Wire/page/header/status; SDK-owned location is null and optional HTTP cache/Connection/dirty-state remain generic SDK-R1/C1. Actual cloud permissions/acceptance are not established by HTTP fixtures.
- Direct Source find_extension signature declares only identity/ignore_missing; Go raw Query/Header/Microversion/ListBasePath options are explicit convenience corresponding to generic Resource.find, not implicit extra kwargs on the Python named method.
- Official Nova31.0.0 extension list/show policy is project-scope RULE_ANY, not admin-only; deployment policies can differ. Role-specific version headers, safe literal IDs, source/context guards and owned callback maps improve Go safety.

### `python:compute/v2/get_hypervisor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.API.Get`

- Get sends GET os-hypervisors/{id} and decodes id, hypervisor_hostname, type, version, host_ip, state, status and service.
  Tests: `compute/v2/hypervisors/python_parity_test.go:TestPythonHypervisorGetAndUptimeMatchProxyRequests`.

Python/Go differences:

- hypervisors.Hypervisor has no uptime field, which Nova returns in the show response from 2.88.
- Python sends min(Hypervisor _max_microversion 2.88, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/get_hypervisor_uptime`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.API.GetUptime`

- GetUptime sends GET os-hypervisors/{id}/uptime below 2.88 and decodes id, hypervisor_hostname, state, status and uptime, and returns Nova's 404 at 2.88.
  Tests: `compute/v2/hypervisors/python_parity_test.go:TestPythonHypervisorGetAndUptimeMatchProxyRequests`.

Python/Go differences:

- Python raises SDKException without a request when the server supports 2.88; Go sends the request and returns Nova's 404.
- Python returns a Hypervisor with uptime filled; Go returns the Uptime value.
- Python sends min(Hypervisor _max_microversion 2.88, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/hypervisors`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.API.List`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.WithRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.API.ProxyListRecords`

- List always sends GET os-hypervisors/detail and carries hypervisor_hostname_pattern and with_servers as query keys.
  Tests: `compute/v2/hypervisors/python_parity_test.go:TestPythonHypervisorListAlwaysUsesDetailRoute`.
- Summary is the default; details is explicit, descriptor aliases and dictionary/list projection preserve raw evidence; pre-2.53 pattern uses the escaped search route after advertised-version gating.
  Tests: `compute/v2/hypervisors/python_records_test.go:TestHypervisorRecordsSummaryDetailAndDeclaredProjection`, `compute/v2/hypervisors/python_records_test.go:TestHypervisorRecordsLegacyHostnameSearchOwnsPatternAndVersion`.
- Unselected version discovers and caps at Hypervisor 2.88; header options are owned.
  Tests: `compute/v2/hypervisors/python_records_test.go:TestHypervisorRecordsDefaultDiscoveryAndOptionSnapshots`.
- Proxy JMESPath uses the shared interpreter against an opaque unconsumed generator: projection returns null, length fails, identity and containers preserve the generator without HTTP until consumption. Invalid typed headers fail before discovery.
  Tests: `compute/v2/hypervisors/python_records_test.go:TestHypervisorProxyRecordsEvaluateSharedJMESAgainstLazyGenerator`, `compute/v2/hypervisors/python_records_test.go:TestHypervisorRecordHeadersRejectBeforeUnversionedDiscovery`.

Python/Go differences:

- Go exposes immutable ReadRecord and lazy typed iterators plus the separate ProxyListRecords result for polymorphic Source JMESPath values. Inputs are explicit scalar identities and finite JSON options; native List remains unchanged.
- Preparation/discovery failure returns an error without a discovery-only ReadRecord receipt; successful records retain observed discovery evidence. This does not establish shared session or live-cloud parity.

### `python:compute/v2/find_hypervisor`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.API.Find`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/hypervisors.WithRecordOptions`

- Find(resource.ID) sends one GET os-hypervisors/{id} without list fallback, and Find(resource.Name) returns ErrUnsupported before HTTP.
  Tests: `compute/v2/hypervisors/python_parity_test.go:TestPythonHypervisorFindHasNoNameFallback`.
- Find GETs first, then configured detail/summary fallback compares literal string IDs or hostname while preserving passive numeric response IDs, checks complete pages for duplicates and preserves missing/late failures.
  Tests: `compute/v2/hypervisors/python_records_test.go:TestHypervisorFindRecordsGETThenConfiguredFallbackPreservesPassiveID`, `compute/v2/hypervisors/python_records_test.go:TestHypervisorFindRecordsStrictMissingDuplicatesAndLateError`.
- Cancelled calls and physical read/Close faults remain terminal rather than triggering list fallback.
  Tests: `compute/v2/hypervisors/python_records_test.go:TestHypervisorRecordPreflightAndPhysicalFailureNeverFallback`, `compute/v2/hypervisors/python_records_test.go:TestHypervisorFindPhysicalReadAndCloseFaultsNeverFallback`.

Python/Go differences:

- Go takes an explicit string identity rather than a mutable Resource instance. Typed transport options and safe service-relative route rules are Go-owned.
- Preparation/discovery failure returns an error without a discovery-only ReadRecord receipt; successful records retain observed discovery evidence. This does not establish shared session or live-cloud parity.

### `python:compute/v2/services`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.API.List`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.WithListQuery`

- List sends GET os-services without a query by default and maps Python's name/binary and host to binary and host query keys.
  Tests: `compute/v2/services/python_parity_test.go:TestPythonComputeServiceListMapsNameToBinary`.

Python/Go differences:

- Python filters non-query Body attributes such as status locally; Go Services.List has no local filter, so the caller filters the rows.
- Python sends min(Service _max_microversion 2.69, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/enable_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.API.EnableRecord`

- From client microversion 2.53 EnableRecord sends PUT os-services/{id} with the unwrapped body {"status": "enabled"} for a given ID, ignoring Host and Binary, at the client microversion, accepts every status below 400 and decodes the typed Service and raw Fields.
  Tests: `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsFromMicroversion253`.
- From 2.53 without an ID EnableRecord lists GET os-services?binary=&host=, requires exactly one row (resource.ErrNotFound for none, resource.ErrAmbiguous for several, no PUT), sends only attributes whose listed value differs, and returns the listed row with Sent false and the list header/status when nothing changed.
  Tests: `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsFromMicroversion253`, `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsLookupFailures`.
- Below 2.53, including an unset client microversion, EnableRecord sends PUT os-services/enable with {host, binary} at the client microversion or without version headers, keeps Nova's id-less service response in Fields with a nil typed Service, and wraps a 404 in resource.OperationError{EnableRecord, services} with the Gophercloud status error.
  Tests: `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsBelowMicroversion253`.
- A target without an ID or both Host and Binary from 2.53, a target without both Host and Binary below 2.53 (even with an ID), an unparsable client microversion and a nil API fail with resource.ErrInvalidOption before HTTP, as Python _get_service raises ValueError.
  Tests: `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsLookupFailures`, `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsBelowMicroversion253`.

Python/Go differences:

- Python decides the 2.53 branch with supports_microversion against the server range and default microversion and sends min(Service _max_microversion 2.69, server maximum); Go decides and sends with the one microversion configured or negotiated on the Connection client.
- Python accepts a Service or an ID whose id, host and binary it reuses; Go takes ServiceActionTarget values and treats empty strings as absent, so an empty host or binary that Python would send is rejected before HTTP.
- Python returns the Resource merged from seeded attributes and the response, including the seeded id below 2.53; Go returns ServiceRecord with the response object in Fields, a typed Service from 2.53, and no typed Service for the id-less legacy response.
- Python's host and binary lookup would follow a next link from services_links, links, next or a Link header; Go reads only the single os-services response that Nova returns.

### `python:compute/v2/disable_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.API.DisableRecord`

- From client microversion 2.53 DisableRecord sends PUT os-services/{id} with {"status": "disabled"} plus disabled_reason only when the reason is non-empty, at the client microversion, and accepts every status below 400.
  Tests: `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsFromMicroversion253`.
- From 2.53 without an ID DisableRecord lists services by host and binary, requires exactly one row (resource.ErrAmbiguous wrapped in resource.OperationError{DisableRecord, services} for several), sends only dirty attributes ({"disabled_reason": ...} when only the reason changes), and sends no PUT when status and reason already match.
  Tests: `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsFromMicroversion253`, `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsLookupFailures`.
- Below 2.53 DisableRecord sends PUT os-services/disable with {host, binary}, or os-services/disable-log-reason with disabled_reason added, at the client microversion or without version headers, and keeps Nova's id-less response in Fields with a nil typed Service.
  Tests: `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsBelowMicroversion253`.
- An unparsable client microversion and a target without the required ID or host and binary fail with resource.ErrInvalidOption before HTTP.
  Tests: `compute/v2/services/python_service_actions_test.go:TestPythonComputeServiceStatusRecordsLookupFailures`.

Python/Go differences:

- Python decides the 2.53 branch with supports_microversion against the server range and default microversion and sends min(Service _max_microversion 2.69, server maximum); Go decides and sends with the one microversion configured or negotiated on the Connection client.
- Python accepts a Service or an ID whose id, host and binary it reuses; Go takes ServiceActionTarget values and treats empty strings, including an empty reason, as absent; Python also treats an empty reason as no reason.
- Python returns the Resource merged from seeded attributes and the response, including the seeded id below 2.53; Go returns ServiceRecord with the response object in Fields, a typed Service from 2.53, and no typed Service for the id-less legacy response.
- Python's host and binary lookup would follow a next link from services_links, links, next or a Link header; Go reads only the single os-services response that Nova returns.

### `python:compute/v2/find_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.API.List`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.API.FindRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.WithRecordOptions`

- List sends GET os-services and decodes id, binary and host, which a caller must match by hand.
  Tests: `compute/v2/services/python_parity_test.go:TestPythonComputeServiceListMapsNameToBinary`.
- Find lists only, compares Python-style ID strings or name/binary descriptor, applies host locally and discards other query kwargs; it rejects duplicates and distinguishes ignored from strict missing.
  Tests: `compute/v2/services/python_records_test.go:TestComputeServiceFindRecordsListOnlyMatchesIDBinaryAndHost`, `compute/v2/services/python_records_test.go:TestComputeServiceFindRecordsMissingDuplicateAndLateErrors`.
- Owned headers and list route are applied without leaking query kwargs; preflight/cancellation/accepted-row failures retain their distinction.
  Tests: `compute/v2/services/python_records_test.go:TestComputeServiceFindRecordOwnedOptionsAndFallbackPath`, `compute/v2/services/python_records_test.go:TestComputeServiceFindRecordGuardAndAcceptedProjectionFailures`.

Python/Go differences:

- Go accepts scalar string identities and finite JSON equality rather than Resource/object inputs. Typed transport headers/microversion are Go conveniences; Source query kwargs remain ignored except host and the explicit list route.
- Preparation/discovery failure returns an error without a discovery-only ReadRecord receipt; successful records retain observed discovery evidence. This does not establish shared session or live-cloud parity.

### `python:compute/v2/delete_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/services.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- Delete sends DELETE os-services/{id}, succeeds on 204 and returns the 404 as an error.
  Tests: `compute/v2/services/python_parity_test.go:TestPythonComputeServiceDeleteReturnsMissingError`.
- Wrapping the direct delete in resource.IgnoreMissing turns a plain HTTP 404 into success like Python's default ignore_missing=True, keeps other failures such as 409 as errors, and leaves the DELETE request unchanged.
  Tests: `resource/missing_test.go:TestIgnoreMissingDropsOnlyPlain404`, `compute/v2/services/python_ignore_missing_test.go:TestPythonDeleteIgnoreMissingService`.

Python/Go differences:

- Python sends min(Service _max_microversion 2.69, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.
- Python ignore_missing=True is the default; Go selects it by wrapping the call in resource.IgnoreMissing, and the bare call returns the 404 like ignore_missing=False.

### `python:compute/v2/migrations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/migrations.API.List`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/migrations.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/migrations.WithListQuery`

- List sends GET os-migrations, maps every Python query key including changes-since, changes-before and instance_uuid, and decodes the migration fields from one page.
  Tests: `compute/v2/migrations/python_parity_test.go:TestPythonMigrationListMapsQueryAndReadsOnePage`.

Python/Go differences:

- Go reads one page and does not follow migrations_links; the caller continues with Marker.
- Python sends the caller's changes_since/changes_before strings; Go formats time.Time values as RFC3339.
- Python filters non-query Body attributes locally; Go WithListQuery sends such keys to the server.
- Python sends min(Migration _max_microversion 2.80, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/server_migrations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ListServerMigrationRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.AllServerMigrationRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationIgnoreMissing`

- Lazy server-scoped migration iteration uses the bound parent collection, untyped nullable descriptor/default projection, absent-id uuid alternate identity and actual physical page evidence; unknown Body/server_id values are not allowed to alter routing or Resource attributes. Complete advertised pages are consumed while early consumer break avoids further requests.
  Tests: `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationListIsLazyAndKeepsUntypedProjection`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationListFollowsCompletePagesAndHonorsEarlyBreak`.
- Malformed accepted list payloads preserve actual response proof and selected source/version/header snapshots and preflight remain owned.
  Tests: `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationListMalformedAcceptedResponseKeepsActualProof`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationOptionsAreOwnedAcrossReusableCalls`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationPreflightAndOptionGuardsStopRequests`.

Python/Go differences:

- Go takes explicit parent server/migration string IDs in safe escaped segments instead of Python Resource objects with inferred URI parents. Owned immutable record snapshots and actual response channels replace mutable cached Resource/session state. Runtime object/formatter injection, caller custom transport cache bypass and full SDK-R1/C1/S1 or live-cloud/release parity are not claimed.
- For migration operations, explicit selected client/per-call versions take precedence without class-maximum capping. Only an empty selection triggers owned Nova discovery and endpoint minimum/maximum selection with ceiling2.80, retaining actual Discovery receipts. This bounded JSON discovery profile is distinct from Python arbitrary adapter/session cache state; class maximum is not an action-local feature minimum.

### `python:compute/v2/get_server_migration`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.GetServerMigrationRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationIgnoreMissing`

- Member GET escapes parent and migration IDs independently, overlays passive actual response id on constructor seed and returns declared view plus physical response evidence. Source ignore_missing is a constructor-only discarded attr, so GET404 remains strict rather than missing suppression.
  Tests: `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationGetOwnsParentURIAndPassiveResponseID`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationGet404IsStrictForBothMissingFlags`.
- Copied options and selected source/header/version guards surround callbacks; terminal physical faults preserve causes instead of GET fallback or replay.
  Tests: `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationOptionsAreOwnedAcrossReusableCalls`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationPreflightAndOptionGuardsStopRequests`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationPhysicalFaultsAreTerminalAndNeverReplay`.

Python/Go differences:

- Go takes explicit parent server/migration string IDs in safe escaped segments instead of Python Resource objects with inferred URI parents. Owned immutable record snapshots and actual response channels replace mutable cached Resource/session state. Runtime object/formatter injection, caller custom transport cache bypass and full SDK-R1/C1/S1 or live-cloud/release parity are not claimed.
- For migration operations, explicit selected client/per-call versions take precedence without class-maximum capping. Only an empty selection triggers owned Nova discovery and endpoint minimum/maximum selection with ceiling2.80, retaining actual Discovery receipts. This bounded JSON discovery profile is distinct from Python arbitrary adapter/session cache state; class maximum is not an action-local feature minimum.

### `python:compute/v2/abort_server_migration`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.AbortServerMigration`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationIgnoreMissing`

- Abort sends bodyless DELETE to the selected migration member, defaults to ignoring clean404 and permits strict missing. Completed/IgnoredMissing and actual Response receipt distinguish accepted completion from ignored absence without fabricating a Resource.
  Tests: `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationAbortMissingDefaultAndStrictPolicies`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationActionsOwnLiteralForceCompleteAndReceipts`.
- Physical read/Close/cancel failures remain terminal with actual errors and are not ignored as clean absence or replayed.
  Tests: `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationPhysicalFaultsAreTerminalAndNeverReplay`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationPreflightAndOptionGuardsStopRequests`.

Python/Go differences:

- Go takes explicit parent server/migration string IDs in safe escaped segments instead of Python Resource objects with inferred URI parents. Owned immutable record snapshots and actual response channels replace mutable cached Resource/session state. Runtime object/formatter injection, caller custom transport cache bypass and full SDK-R1/C1/S1 or live-cloud/release parity are not claimed.
- For migration operations, explicit selected client/per-call versions take precedence without class-maximum capping. Only an empty selection triggers owned Nova discovery and endpoint minimum/maximum selection with ceiling2.80, retaining actual Discovery receipts. This bounded JSON discovery profile is distinct from Python arbitrary adapter/session cache state; class maximum is not an action-local feature minimum.
- Python abort returns None; Go returns an action carrier with actual receipt. Completed=false with an error does not prove the server did not execute the request.

### `python:compute/v2/force_complete_server_migration`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ForceCompleteServerMigration`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationIgnoreMissing`

- Force-complete posts exactly {"force_complete":null} to the selected migration member action without resource-body decoding; non-JSON successful response remains actual receipt and the API does not wait for migration completion.
  Tests: `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationActionsOwnLiteralForceCompleteAndReceipts`.
- Physical action failures do not replay; copied options/source guards reject invalid route/context/callback state before requests.
  Tests: `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationPhysicalFaultsAreTerminalAndNeverReplay`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationOptionsAreOwnedAcrossReusableCalls`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationPreflightAndOptionGuardsStopRequests`.

Python/Go differences:

- Go takes explicit parent server/migration string IDs in safe escaped segments instead of Python Resource objects with inferred URI parents. Owned immutable record snapshots and actual response channels replace mutable cached Resource/session state. Runtime object/formatter injection, caller custom transport cache bypass and full SDK-R1/C1/S1 or live-cloud/release parity are not claimed.
- For migration operations, explicit selected client/per-call versions take precedence without class-maximum capping. Only an empty selection triggers owned Nova discovery and endpoint minimum/maximum selection with ceiling2.80, retaining actual Discovery receipts. This bounded JSON discovery profile is distinct from Python arbitrary adapter/session cache state; class maximum is not an action-local feature minimum.
- Python force_complete returns None; Go exposes a Completed/Response carrier for accepted request evidence. This is not migration lifecycle completion or automatic cleanup.

### `python:compute/v2/reset_server_state`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ResetState`

- ResetState posts {"os-resetState": {"state": s}} to servers/{id}/action for active and error.
  Tests: `compute/v2/servers/python_parity_admin_test.go:TestPythonAdminServerResetStateMatchesProxyBody`.

Python/Go differences:

- Python takes a string; Go takes servers.ServerState, so the caller converts the string.
- Python sends an empty Accept header and accepts any status below 400; Go uses the default Accept header and accepts 201 or 202.
- Python sends min(Server _max_microversion 2.100, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/trigger_server_crash_dump`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.TriggerServerCrashDump`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithUserActionHeader`

- TriggerServerCrashDump posts exactly {"trigger_crash_dump":null} to the escaped server action with owned empty Accept and selected version, leaving the shared client unchanged. Like the pinned Source method it adds no action-local2.17 rejection gate.
  Tests: `compute/v2/servers/python_crash_dump_test.go:TestPythonTriggerServerCrashDumpOwnsPayloadRouteAndSelectedVersion`.
- Options/source/cancellation guards precede HTTP, actual rejected statuses preserve native operation proof and physical read/Close/cancel failures never replay. Success is request acceptance without guest dump/reboot waiting.
  Tests: `compute/v2/servers/python_crash_dump_test.go:TestPythonTriggerServerCrashDumpOptionsAndSourceGuardsPrecedeHTTP`, `compute/v2/servers/python_crash_dump_test.go:TestPythonTriggerServerCrashDumpStatusesKeepNativeOperationProof`, `compute/v2/servers/python_crash_dump_test.go:TestPythonTriggerServerCrashDumpPhysicalFaultNeverReplays`.

Python/Go differences:

- Go takes explicit parent server/migration string IDs in safe escaped segments instead of Python Resource objects with inferred URI parents. Owned immutable record snapshots and actual response channels replace mutable cached Resource/session state. Runtime object/formatter injection, caller custom transport cache bypass and full SDK-R1/C1/S1 or live-cloud/release parity are not claimed.
- The Go API uses the selected client/per-call Nova version rather than Source default-first/endpoint-discovered class maximum negotiation. ServerMigration class maximum2.80 and Server maximum2.100 are not action-local API minimum gates. A selected Source session default can exceed the class maximum and is not capped; Go does not claim automatic negotiation.
- Nova API minimum2.17 is distinct from pinned trigger_crash_dump local code, which only delegates _action. The existing Go version validator checks selected Nova-version syntax/range, leaves action availability to the server and exposes error-only success rather than a fabricated receipt.

### `python:compute/v2/get_server_diagnostics`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/diagnostics.API.Get`

- Get sends GET servers/{id}/diagnostics and returns the 2.48 diagnostics keys as a map.
  Tests: `compute/v2/diagnostics/python_parity_test.go:TestPythonServerDiagnosticsGetMatchesProxyRequest`.

Python/Go differences:

- Python returns a ServerDiagnostics resource; Go returns map[string]any with float64 numbers.
- Python sends min(ServerDiagnostics _max_microversion 2.48, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/usages`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/usage.API.AllTenants`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/usage.WithAllTenantsQuery`

- AllTenants sends GET os-simple-tenant-usage without a query by default and with start, end and detailed=1 when set.
  Tests: `compute/v2/usage/python_parity_test.go:TestPythonUsageAllTenantsMatchesProxyQuery`.

Python/Go differences:

- Python sends start.isoformat() including any offset; Go sends the wall-clock time without the offset, so callers pass UTC values.
- Go Detailed true sends detailed=1; other Python query values go through WithAllTenantsQuery.
- Python sends min(Usage _max_microversion 2.75, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/get_usage`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/usage.API.SingleTenant`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/usage.WithSingleTenantQuery`

- SingleTenant sends GET os-simple-tenant-usage/{project} with start and end, and stopping after the first value keeps it to Python's single fetch.
  Tests: `compute/v2/usage/python_parity_test.go:TestPythonUsageSingleProjectFirstPageMatchesFetch`.

Python/Go differences:

- Python fetches one response; Go SingleTenant is a stream that follows tenant_usage_links unless the caller stops after the first value.
- Python sends start.isoformat() including any offset; Go sends the wall-clock time without the offset, so callers pass UTC values.
- Python sends min(Usage _max_microversion 2.75, server maximum) discovered per request; Go sends the one microversion configured or negotiated on the Connection client.

### `python:compute/v2/get_quota_class_set`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.API.GetQuotaClassRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaClassGetOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaClassName`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaClassGetHeaders`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaClassGetMicroversion`

- GET default/custom class names use the selected literal route, preserve declared untyped id and15 integer/default/null fields independently from actual unknown Wire and response proof. Invalid accepted integer conversion retains original evidence.
  Tests: `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassGetDefaultAndCustomNamesKeepLiteralRoutes`, `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassInvalidIntegerRetainsAcceptedResponse`.
- Empty version discovers Nova endpoint and selects ceiling2.56 while explicit2.96 remains uncapped; independent Discovery receipt snapshots and option/header ownership survive reusable calls.
  Tests: `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassDefaultDiscoveryCapsVersionAndOwnsProof`, `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassOptionsOwnAttributesHeadersAndExplicitVersion`.

Python/Go differences:

- Go takes a scalar string class name in a fixed /os-quota-class-sets/{className} route rather than mutable Python Resource input. Only omitted GET name defaults to default; explicit empty names are rejected and other names are sent for server validation. Safe escaped routing, finite UTF-8 JSON and independent declared Resource/Wire/Envelope/header/status snapshots replace arbitrary runtime objects and URI formatter overrides.
- Raw update attributes are not coerced on the wire; only the declared returned Resource view applies Source integer/default/null behavior, including bool remaining bool and non-digit strings becoming0. Unknown Body attributes are dropped from the view while actual Wire retains unknown response JSON. Descriptor projection errors are reported without claiming a successful wire operation or Python runtime object identity.
- New scalar-name update seeds id but never commits dirty id; duplicate attrs id is rejected, and no recognized changes produces no HTTP/Discovery or fabricated receipt. Existing mutable Resource equality/recommit, optional response cache and generic adapter/session/runtime SDK-R1/C1/S1 remain whole-SDK goals. Header/version options are explicit Go conveniences and caller custom transport cache bypass is not claimed.
- Nova explicitly selected client/per-call versions are not class-maximum capped. Only an empty selected version triggers owned endpoint discovery/minimum-maximum selection with ceiling2.56 and independent Discovery receipts; this bounded JSON profile does not emulate arbitrary Python adapters/session caches.

### `python:compute/v2/update_quota_class_set`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.API.UpdateQuotaClassRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaClassUpdateOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaClassUpdateAttribute`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaClassUpdateHeaders`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithQuotaClassUpdateMicroversion`

- PUT sends explicit raw quota_class_set attrs including nullable values, then projects declared integer view on response overlay without sending defaults or dirty id. Empty/unknown-only updates perform neither HTTP nor discovery and duplicate attrs id is rejected.
  Tests: `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassUpdatePUTUsesRawAttributesAndProjectedResponse`, `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassUpdateNoopSkipsHTTPAndDiscoveryAndIDCollisions`.
- Raw update options/header/version and Discovery/response channels remain independently owned; preflight/source/cancel guards prevent invalid requests and physical404 failures never become successful no-op or replay.
  Tests: `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassOptionsOwnAttributesHeadersAndExplicitVersion`, `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassPreflightAndCallbackGuardsPreventRequests`, `compute/v2/quotasets/python_quota_class_test.go:TestNovaQuotaClassPhysicalErrorsAnd404NeverReplay`.

Python/Go differences:

- Go takes a scalar string class name in a fixed /os-quota-class-sets/{className} route rather than mutable Python Resource input. Only omitted GET name defaults to default; explicit empty names are rejected and other names are sent for server validation. Safe escaped routing, finite UTF-8 JSON and independent declared Resource/Wire/Envelope/header/status snapshots replace arbitrary runtime objects and URI formatter overrides.
- Raw update attributes are not coerced on the wire; only the declared returned Resource view applies Source integer/default/null behavior, including bool remaining bool and non-digit strings becoming0. Unknown Body attributes are dropped from the view while actual Wire retains unknown response JSON. Descriptor projection errors are reported without claiming a successful wire operation or Python runtime object identity.
- New scalar-name update seeds id but never commits dirty id; duplicate attrs id is rejected, and no recognized changes produces no HTTP/Discovery or fabricated receipt. Existing mutable Resource equality/recommit, optional response cache and generic adapter/session/runtime SDK-R1/C1/S1 remain whole-SDK goals. Header/version options are explicit Go conveniences and caller custom transport cache bypass is not claimed.
- Nova explicitly selected client/per-call versions are not class-maximum capped. Only an empty selected version triggers owned endpoint discovery/minimum-maximum selection with ceiling2.56 and independent Discovery receipts; this bounded JSON profile does not emulate arbitrary Python adapters/session caches.

### `python:compute/v2/create_share_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/shareattachments.API.CreateRecord`

- Caller IDs fix both URI segments and never derive routes from returned uuid/share_id; Capture and owned config precede callbacks/discovery, request Body/header/microversion remains fixed through callbacks and clean authentication renewal. Invalid inputs and observed source changes terminate before discovery or later options.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareGetFixesParentMemberAndProxySeed`, `compute/v2/shareattachments/contracts_test.go:TestShareInvalidOptionsAndIdentityPrecedeDiscovery`, `compute/v2/shareattachments/contracts_test.go:TestShareCleanAuthenticationRenewalRetainsRequest`, `compute/v2/shareattachments/contracts_test.go:TestShareOwnedConfigRepairsNilMapsAndCopiesPolicyAroundCallbacks`.
- Selected microversion is literal and uncapped; missing selected version discovers the advertised range with Source ceiling2.97. Explicit empty override owns required version header absence without mutating shared client. Discovery receipt is separate from member response evidence.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareVersionDiscoveryCeilingAndExplicitSelection`, `compute/v2/shareattachments/contracts_test.go:TestShareExplicitEmptyOverrideOwnsVersionAbsence`.
- Actual accepted/rejected status, body, header and all physical Read/Close causes survive. Four operations by eight statuses by three fault modes prove one HTTP/Close and zero retry, backoff or reauthentication callbacks after physical faults; native401 pointer ErrOriginal/ErrReauth preserved, while clean401 renewal keeps exact route/body/version.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestSharePhysicalFaultsKeepEvidenceAndStopAllHooks`, `compute/v2/shareattachments/contracts_test.go:TestShareCleanAuthenticationRenewalRetainsRequest`.
- POST uses share envelope containing explicit share_id and only supplied declared dirty attributes; omitted default export_location/tag stays absent in request, explicitnull and arbitrary untyped values survive. Source view merges accepted response fields over seed and retains defaults; actual Wire preserves unknown/self/exact numbers independently.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareCreateDirtyBodyAndSeededDescriptors`, `compute/v2/shareattachments/contracts_test.go:TestShareMemberNonJSONRetainsSeedWithoutFakeWire`.

Python/Go differences:

- Go concrete IDs use server.ID/share.ID instead of Python arbitrary Resource inputs. Literal UTF8 identity, fixed route/selected header guards and physical handling-error termination improve safety; this is not arbitrary session/base_path or Resource lifecycle parity.
- Source has six untyped declared Body descriptors plus inherited id/name. Go returns independently owned Resource/Wire/Envelope/header/status, preserves seed/default/null and alternate ID presence, and supplies explicit per-call microversion options. The shared SDK-R1/C1 Connection location and optional HTTP cache contracts are separate; direct client location is null.
- Official Nova31.0.0 stock policy is project reader on index/show and project member on create/delete, checked against instance project. Endpoints need2.97, request accepts share_id and optional tag, tag default belongs to Nova; asynchronous mapping receipt does not prove mount acceptance or complete Manila access workflow.

### `python:compute/v2/get_share_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/shareattachments.API.GetRecord`

- Caller IDs fix both URI segments and never derive routes from returned uuid/share_id; Capture and owned config precede callbacks/discovery, request Body/header/microversion remains fixed through callbacks and clean authentication renewal. Invalid inputs and observed source changes terminate before discovery or later options.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareGetFixesParentMemberAndProxySeed`, `compute/v2/shareattachments/contracts_test.go:TestShareInvalidOptionsAndIdentityPrecedeDiscovery`, `compute/v2/shareattachments/contracts_test.go:TestShareCleanAuthenticationRenewalRetainsRequest`, `compute/v2/shareattachments/contracts_test.go:TestShareOwnedConfigRepairsNilMapsAndCopiesPolicyAroundCallbacks`.
- Selected microversion is literal and uncapped; missing selected version discovers the advertised range with Source ceiling2.97. Explicit empty override owns required version header absence without mutating shared client. Discovery receipt is separate from member response evidence.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareVersionDiscoveryCeilingAndExplicitSelection`, `compute/v2/shareattachments/contracts_test.go:TestShareExplicitEmptyOverrideOwnsVersionAbsence`.
- Actual accepted/rejected status, body, header and all physical Read/Close causes survive. Four operations by eight statuses by three fault modes prove one HTTP/Close and zero retry, backoff or reauthentication callbacks after physical faults; native401 pointer ErrOriginal/ErrReauth preserved, while clean401 renewal keeps exact route/body/version.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestSharePhysicalFaultsKeepEvidenceAndStopAllHooks`, `compute/v2/shareattachments/contracts_test.go:TestShareCleanAuthenticationRenewalRetainsRequest`.
- GET seeds Proxy member id, independently retains caller server URI and actual returned share_id/uuid. Untyped descriptors preserve null/value; missing export_location default empty and inherited fields null; nonJSON accepted body keeps seed with no fabricated Wire while parsed wrongshape returns actual receipt.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareGetFixesParentMemberAndProxySeed`, `compute/v2/shareattachments/contracts_test.go:TestShareMemberNonJSONRetainsSeedWithoutFakeWire`, `compute/v2/shareattachments/contracts_test.go:TestShareMemberParsedInvalidShapeKeepsActualReceipt`.

Python/Go differences:

- Go concrete IDs use server.ID/share.ID instead of Python arbitrary Resource inputs. Literal UTF8 identity, fixed route/selected header guards and physical handling-error termination improve safety; this is not arbitrary session/base_path or Resource lifecycle parity.
- Source has six untyped declared Body descriptors plus inherited id/name. Go returns independently owned Resource/Wire/Envelope/header/status, preserves seed/default/null and alternate ID presence, and supplies explicit per-call microversion options. The shared SDK-R1/C1 Connection location and optional HTTP cache contracts are separate; direct client location is null.
- Official Nova31.0.0 stock policy is project reader on index/show and project member on create/delete, checked against instance project. Endpoints need2.97, request accepts share_id and optional tag, tag default belongs to Nova; asynchronous mapping receipt does not prove mount acceptance or complete Manila access workflow.

### `python:compute/v2/share_attachments`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/shareattachments.API.ListRecords`

- Caller IDs fix both URI segments and never derive routes from returned uuid/share_id; Capture and owned config precede callbacks/discovery, request Body/header/microversion remains fixed through callbacks and clean authentication renewal. Invalid inputs and observed source changes terminate before discovery or later options.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareGetFixesParentMemberAndProxySeed`, `compute/v2/shareattachments/contracts_test.go:TestShareInvalidOptionsAndIdentityPrecedeDiscovery`, `compute/v2/shareattachments/contracts_test.go:TestShareCleanAuthenticationRenewalRetainsRequest`, `compute/v2/shareattachments/contracts_test.go:TestShareOwnedConfigRepairsNilMapsAndCopiesPolicyAroundCallbacks`.
- Selected microversion is literal and uncapped; missing selected version discovers the advertised range with Source ceiling2.97. Explicit empty override owns required version header absence without mutating shared client. Discovery receipt is separate from member response evidence.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareVersionDiscoveryCeilingAndExplicitSelection`, `compute/v2/shareattachments/contracts_test.go:TestShareExplicitEmptyOverrideOwnsVersionAbsence`.
- Actual accepted/rejected status, body, header and all physical Read/Close causes survive. Four operations by eight statuses by three fault modes prove one HTTP/Close and zero retry, backoff or reauthentication callbacks after physical faults; native401 pointer ErrOriginal/ErrReauth preserved, while clean401 renewal keeps exact route/body/version.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestSharePhysicalFaultsKeepEvidenceAndStopAllHooks`, `compute/v2/shareattachments/contracts_test.go:TestShareCleanAuthenticationRenewalRetainsRequest`.
- List is lazy; canonical Body filters apply locally after Source raw row cap and do not become wire query. MaxItems implies Source limit hint, cap prevents nextlink lookahead, marker fallback selects id by presence before alternate share_id; nullprimary marker is terminal with page receipt. Ordinary unknown kwargs are discarded before serialization and reserved Source controls explicitly rejected before caller marshaller. Local equality uses CPython-loaded float and arbitraryprecision integer/Boolean semantics; dictionary conditions use Source actualtruthiness and recursive subset, including missing nested None and truthy nonmapping empty-filter behavior.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareListProjectionLocalCapAndLazyNoLookahead`, `compute/v2/shareattachments/contracts_test.go:TestShareListAlternateMarkerPresenceAndContinuation`, `compute/v2/shareattachments/contracts_test.go:TestShareListNullPrimaryMarkerIsTerminalWithPageReceipt`, `compute/v2/shareattachments/contracts_test.go:TestShareListUnknownDiscardAndControlRejectBeforeMarshaler`, `compute/v2/shareattachments/contracts_test.go:TestShareListActualLinksUseFixedRouteAndSinglePageStopsBeforeLink`, `compute/v2/shareattachments/contracts_test.go:TestShareListForeignContinuationFailsWithActualPage`, `compute/v2/shareattachments/contracts_test.go:TestShareSourceLoadedLocalEqualityAndDictionaryTruthiness`.

Python/Go differences:

- Go concrete IDs use server.ID/share.ID instead of Python arbitrary Resource inputs. Literal UTF8 identity, fixed route/selected header guards and physical handling-error termination improve safety; this is not arbitrary session/base_path or Resource lifecycle parity.
- Source has six untyped declared Body descriptors plus inherited id/name. Go returns independently owned Resource/Wire/Envelope/header/status, preserves seed/default/null and alternate ID presence, and supplies explicit per-call microversion options. The shared SDK-R1/C1 Connection location and optional HTTP cache contracts are separate; direct client location is null.
- Official Nova31.0.0 stock policy is project reader on index/show and project member on create/delete, checked against instance project. Endpoints need2.97, request accepts share_id and optional tag, tag default belongs to Nova; asynchronous mapping receipt does not prove mount acceptance or complete Manila access workflow.
- Source inherits limit/marker and maxItems limit hint but stock Nova31.0.0 index schema accepts no query properties; these options may be server-rejected, default no-query listing works. Go primitive limit/marker types and same-origin/path/cursor guards are stricter than arbitrary Source kwargs.
- Arbitrary JMESPath and session/routing/constructor controls explicitly fail before HTTP; shared SDK-S1 stays separate. Existing Source REST link, marker, raw cap and local filter policies are reused rather than labeling transport wrappers full SDK parity.
- Source marker fallback can place null/false/empty primary id into query params. Go preserves primary presence precedence but requires nonblank string cursor and returns the actual page receipt on failure; it never silently substitutes alternate share_id. TestShareListNullPrimaryMarkerIsTerminalWithPageReceipt proves this deliberate strict continuation difference.

### `python:compute/v2/delete_share_attachment`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/shareattachments.API.DeleteRecord`

- Caller IDs fix both URI segments and never derive routes from returned uuid/share_id; Capture and owned config precede callbacks/discovery, request Body/header/microversion remains fixed through callbacks and clean authentication renewal. Invalid inputs and observed source changes terminate before discovery or later options.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareGetFixesParentMemberAndProxySeed`, `compute/v2/shareattachments/contracts_test.go:TestShareInvalidOptionsAndIdentityPrecedeDiscovery`, `compute/v2/shareattachments/contracts_test.go:TestShareCleanAuthenticationRenewalRetainsRequest`, `compute/v2/shareattachments/contracts_test.go:TestShareOwnedConfigRepairsNilMapsAndCopiesPolicyAroundCallbacks`.
- Selected microversion is literal and uncapped; missing selected version discovers the advertised range with Source ceiling2.97. Explicit empty override owns required version header absence without mutating shared client. Discovery receipt is separate from member response evidence.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareVersionDiscoveryCeilingAndExplicitSelection`, `compute/v2/shareattachments/contracts_test.go:TestShareExplicitEmptyOverrideOwnsVersionAbsence`.
- Actual accepted/rejected status, body, header and all physical Read/Close causes survive. Four operations by eight statuses by three fault modes prove one HTTP/Close and zero retry, backoff or reauthentication callbacks after physical faults; native401 pointer ErrOriginal/ErrReauth preserved, while clean401 renewal keeps exact route/body/version.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestSharePhysicalFaultsKeepEvidenceAndStopAllHooks`, `compute/v2/shareattachments/contracts_test.go:TestShareCleanAuthenticationRenewalRetainsRequest`.
- DELETE has no Body, uses server/share member route and preserves literal accepted receipt without Resource fabrication. IgnoreMissing defaults true and suppresses only clean404; strict404/conflict/physical faults return original causal errors.
  Tests: `compute/v2/shareattachments/contracts_test.go:TestShareDeleteDefaultOnlyCleanMissingAndBodylessSuccess`, `compute/v2/shareattachments/contracts_test.go:TestSharePhysicalFaultsKeepEvidenceAndStopAllHooks`.

Python/Go differences:

- Go concrete IDs use server.ID/share.ID instead of Python arbitrary Resource inputs. Literal UTF8 identity, fixed route/selected header guards and physical handling-error termination improve safety; this is not arbitrary session/base_path or Resource lifecycle parity.
- Source has six untyped declared Body descriptors plus inherited id/name. Go returns independently owned Resource/Wire/Envelope/header/status, preserves seed/default/null and alternate ID presence, and supplies explicit per-call microversion options. The shared SDK-R1/C1 Connection location and optional HTTP cache contracts are separate; direct client location is null.
- Official Nova31.0.0 stock policy is project reader on index/show and project member on create/delete, checked against instance project. Endpoints need2.97, request accepts share_id and optional tag, tag default belongs to Nova; asynchronous mapping receipt does not prove mount acceptance or complete Manila access workflow.

## References

- [compute v2 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)
- [Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py)
- [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [Aggregate](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/aggregate.py)
- [Flavor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/flavor.py)
- [Extension](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/extension.py)
- [Hypervisor](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/hypervisor.py)
- [Service](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/service.py)
- [Migration](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/migration.py)
- [ServerMigration](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_migration.py)
- [Server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server.py)
- [ServerDiagnostics](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/server_diagnostics.py)
- [Usage](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/usage.py)
- [Python Parity](python-parity.md)
- [Python Precache Images](aggregates/python-precache-images.md)
- [Serverexternalevents](serverexternalevents/README.md)
- [Python Service Actions](services/python-service-actions.md)
- [Python Record Find](extensions/python-record-find.md)
- [Python Records](hypervisors/python-records.md)
- [Python Find Record](services/python-find-record.md)
- [Python Quota Class](quotasets/python-quota-class.md)
- [Share Attachments Python](share-attachments-python.md)
- [owned precache API](aggregates/python-precache-images.md)
- [Python Server Migrations](servers/python-server-migrations.md)
- [External events](serverexternalevents/README.md)
