# Python Parity Vpn Bgp

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:network/v2/create_vpn_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.WithCreateField`

- Create sends POST vpn/vpnservices with {"vpnservice": {...}} containing the given attributes and decodes the returned resource.
  Tests: `network/v2/extensions/vpnaas/services/python_parity_test.go:TestPythonVPNServiceProxyRequests`.

Python/Go differences:

- CreateOpts.AdminStateUp has no omitempty, so Go sends admin_state_up null when it is nil; Python omits the key and Neutron applies true, so Go callers set AdminStateUp explicitly.
- Python accepts any 2xx create response; the Go facade accepts only 201 or 202.
- Python sends unknown keyword attributes in the body only for resources with _allow_unknown_attrs_in_body; Go sends extension keys through With*Field, which rejects keys that collide with an Opts JSON tag.

### `python:network/v2/delete_vpn_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE vpn/vpnservices/{id}, returns nil for 404 and 204, reports resource.ErrNotFound with WithMissingError and keeps 409; plain Delete returns the 404.
  Tests: `network/v2/extensions/vpnaas/services/python_parity_test.go:TestPythonVPNServiceProxyRequests`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- Remove rejects IDs containing /, \, ?, #, %, or whitespace before HTTP, while Python sends any ID string.
- Python accepts any 2xx delete response; Go accepts 202 or 204.

### `python:network/v2/get_vpn_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.GetWithFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.Get`

- GetWithFields sends GET vpn/vpnservices/{id} with each Python VpnService Body attribute name mapped to its wire name as repeated fields query keys, keeping caller order, repeats, empty and unknown names, and decodes the selected attributes into the typed model.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersSendAliasedRepeatedFields`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- GetWithFields with nil or empty fields sends the plain member GET without a query, which is Python's fields=None request, and a sparse response decodes with zero values for omitted attributes.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersWithoutFieldsSendPlainGet`.
- GetWithFields accepts HTTP 200 through 399 like the Python read, and returns 404 and other failures as the native gophercloud status error wrapped in resource.OperationError with the operation name.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersAcceptPythonStatusesAndWrapFailures`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.
- Get sends GET vpn/vpnservices/{id} without any query and decodes the resource, which is Python's fields=None request.
  Tests: `network/v2/extensions/vpnaas/services/python_parity_test.go:TestPythonVPNServiceProxyRequests`, `network/v2/extensions/vpnaas/services/python_parity_test.go:TestPythonVPNServiceFindChainAndGetWithoutFields`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Go GetWithFields returns the Gophercloud v2.15.0 typed model, so attributes Neutron returns outside that model are dropped and attributes the server omits are zero values instead of None; Python returns a sparse Resource.
- The SDK field getter accepts HTTP 200..399 and keeps unknown field names in the query like Python; plain API.Get keeps the native status policy.

### `python:network/v2/vpn_services`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.WithRecordListOpts`

- List sends GET vpn/vpnservices with WithListOptions and WithListQuery values as query and follows the vpnservices_links next href.
  Tests: `network/v2/extensions/vpnaas/services/python_parity_test.go:TestPythonVPNServiceListQueryAndLinks`.
- SDK-owned record listing owns all-page iteration and classifies declared QueryParameters/aliases as server query versus declared Body-only local descriptor filters. Missing/null, project alias, raw unknown fields and class-specific boolean/list projection are separate from original page evidence.
  Tests: `network/v2/extensions/vpnaas/services/python_record_contracts_test.go:TestVPNServiceRecordsAllPagesServerQueryAndLocalFilter`, `network/v2/extensions/vpnaas/services/python_record_contracts_test.go:TestVPNServiceRecordsProjectionDefaultsAndUnknownFields`.
- ProxyListRecords retains actual opaque Resource.list generator semantics: projection/type may return null without HTTP and @ returns a generator. Owned options discard unknown keys before serialization, guard cancellation/invalid controls before HTTP, and propagate failed page reads without fabricated collected results.
  Tests: `network/v2/extensions/vpnaas/services/python_record_contracts_test.go:TestVPNServiceProxyJMESReturnsOpaqueGeneratorWithoutHTTP`, `network/v2/extensions/vpnaas/services/python_record_contracts_test.go:TestVPNServiceRecordsOptionsCancellationAndFailures`.

Python/Go differences:

- Legacy native List follows {resources}_links but lacks Source marker fallback. The new record pager also retries with the last physical ID when a positive limit is set and no next link exists.
- The SDK owns Source-shaped ListRecords/AllRecords local Body filters and an explicit ProxyListRecords opaque-generator result. Existing native typed List remains a separate ABI; immutable raw views are not Python mutable Resource instances.
- Explicit Go profile boundaries include configured guarded HTTP/route/constructor decoding, full Requests charset/nonfinite JSON and arbitrary Python session/cache/dynamic object behavior. These do not retain the now-closed absence of SDK local filtering.

### `python:network/v2/update_vpn_service`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/services.WithUpdateField`

- Update sends PUT vpn/vpnservices/{id} with {"vpnservice": {...}} including explicit empty name or description pointers and decodes the result.
  Tests: `network/v2/extensions/vpnaas/services/python_parity_test.go:TestPythonVPNServiceProxyRequests`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Python sends no request when no attributes are given; Go Update always sends PUT with the envelope.

### `python:network/v2/create_vpn_endpoint_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.WithCreateField`

- Create sends POST vpn/endpoint-groups with {"endpoint_group": {...}} containing the given attributes and decodes the returned resource.
  Tests: `network/v2/extensions/vpnaas/endpointgroups/python_parity_test.go:TestPythonEndpointGroupProxyRequests`.

Python/Go differences:

- CreateOpts.Endpoints has no omitempty and is sent as null when nil, while Python omits endpoints it was not given.
- Python accepts any 2xx create response; the Go facade accepts only 201 or 202.
- Python sends unknown keyword attributes in the body only for resources with _allow_unknown_attrs_in_body; Go sends extension keys through With*Field, which rejects keys that collide with an Opts JSON tag.

### `python:network/v2/delete_vpn_endpoint_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE vpn/endpoint-groups/{id}, returns nil for 404 and 204, reports resource.ErrNotFound with WithMissingError and keeps 409; plain Delete returns the 404.
  Tests: `network/v2/extensions/vpnaas/endpointgroups/python_parity_test.go:TestPythonEndpointGroupProxyRequests`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- Remove rejects IDs containing /, \, ?, #, %, or whitespace before HTTP, while Python sends any ID string.
- Python accepts any 2xx delete response; Go accepts 202 or 204.

### `python:network/v2/get_vpn_endpoint_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.API.GetWithFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.API.Get`

- GetWithFields sends GET vpn/endpoint-groups/{id} with each Python VpnEndpointGroup Body attribute name mapped to its wire name as repeated fields query keys, keeping caller order, repeats, empty and unknown names, and decodes the selected attributes into the typed model.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersSendAliasedRepeatedFields`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- GetWithFields with nil or empty fields sends the plain member GET without a query, which is Python's fields=None request, and a sparse response decodes with zero values for omitted attributes.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersWithoutFieldsSendPlainGet`.
- GetWithFields accepts HTTP 200 through 399 like the Python read, and returns 404 and other failures as the native gophercloud status error wrapped in resource.OperationError with the operation name.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersAcceptPythonStatusesAndWrapFailures`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.
- Get sends GET vpn/endpoint-groups/{id} without any query and decodes the resource, which is Python's fields=None request.
  Tests: `network/v2/extensions/vpnaas/endpointgroups/python_parity_test.go:TestPythonEndpointGroupProxyRequests`, `network/v2/extensions/vpnaas/endpointgroups/python_parity_test.go:TestPythonEndpointGroupFindChainAndGetWithoutFields`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Go GetWithFields returns the Gophercloud v2.15.0 typed model, so attributes Neutron returns outside that model are dropped and attributes the server omits are zero values instead of None; Python returns a sparse Resource.
- The SDK field getter accepts HTTP 200..399 and keeps unknown field names in the query like Python; plain API.Get keeps the native status policy.

### `python:network/v2/vpn_endpoint_groups`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.WithRecordListOpts`

- List sends GET vpn/endpoint-groups with WithListOptions and WithListQuery values as query and follows the endpoint_groups_links next href.
  Tests: `network/v2/extensions/vpnaas/endpointgroups/python_parity_test.go:TestPythonEndpointGroupListQueryAndLinks`.
- SDK-owned record listing owns all-page iteration and classifies declared QueryParameters/aliases as server query versus declared Body-only local descriptor filters. Missing/null, project alias, raw unknown fields and class-specific boolean/list projection are separate from original page evidence.
  Tests: `network/v2/extensions/vpnaas/endpointgroups/python_record_contracts_test.go:TestVPNEndpointGroupRecordsAllPagesServerQueryAndLocalFilter`, `network/v2/extensions/vpnaas/endpointgroups/python_record_contracts_test.go:TestVPNEndpointGroupRecordsProjectionDefaultsAndUnknownFields`.
- ProxyListRecords retains actual opaque Resource.list generator semantics: projection/type may return null without HTTP and @ returns a generator. Owned options discard unknown keys before serialization, guard cancellation/invalid controls before HTTP, and propagate failed page reads without fabricated collected results.
  Tests: `network/v2/extensions/vpnaas/endpointgroups/python_record_contracts_test.go:TestVPNEndpointGroupProxyJMESReturnsOpaqueGeneratorWithoutHTTP`, `network/v2/extensions/vpnaas/endpointgroups/python_record_contracts_test.go:TestVPNEndpointGroupRecordsOptionsCancellationAndFailures`.

Python/Go differences:

- Legacy native List follows {resources}_links but lacks Source marker fallback. The new record pager also retries with the last physical ID when a positive limit is set and no next link exists.
- The SDK owns Source-shaped ListRecords/AllRecords local Body filters and an explicit ProxyListRecords opaque-generator result. Existing native typed List remains a separate ABI; immutable raw views are not Python mutable Resource instances.
- Explicit Go profile boundaries include configured guarded HTTP/route/constructor decoding, full Requests charset/nonfinite JSON and arbitrary Python session/cache/dynamic object behavior. These do not retain the now-closed absence of SDK local filtering.

### `python:network/v2/update_vpn_endpoint_group`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/endpointgroups.WithUpdateField`

- Update sends PUT vpn/endpoint-groups/{id} with {"endpoint_group": {...}} including explicit empty name or description pointers and decodes the result.
  Tests: `network/v2/extensions/vpnaas/endpointgroups/python_parity_test.go:TestPythonEndpointGroupProxyRequests`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Python sends no request when no attributes are given; Go Update always sends PUT with the envelope.

### `python:network/v2/create_vpn_ike_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithCreateField`

- Create sends POST vpn/ikepolicies with {"ikepolicy": {...}} containing the given attributes and decodes the returned resource.
  Tests: `network/v2/extensions/vpnaas/ikepolicies/python_parity_test.go:TestPythonIKEPolicyProxyRequests`.

Python/Go differences:

- Empty algorithm strings and a zero lifetime value are omitted; Neutron rejects those values, so no valid Python request is lost.
- Python accepts any 2xx create response; the Go facade accepts only 201 or 202.
- Python sends unknown keyword attributes in the body only for resources with _allow_unknown_attrs_in_body; Go sends extension keys through With*Field, which rejects keys that collide with an Opts JSON tag.

### `python:network/v2/delete_vpn_ike_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE vpn/ikepolicies/{id}, returns nil for 404 and 204, reports resource.ErrNotFound with WithMissingError and keeps 409; plain Delete returns the 404.
  Tests: `network/v2/extensions/vpnaas/ikepolicies/python_parity_test.go:TestPythonIKEPolicyProxyRequests`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- Remove rejects IDs containing /, \, ?, #, %, or whitespace before HTTP, while Python sends any ID string.
- Python accepts any 2xx delete response; Go accepts 202 or 204.

### `python:network/v2/get_vpn_ike_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.API.GetWithFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.API.Get`

- GetWithFields sends GET vpn/ikepolicies/{id} with each Python VpnIkePolicy Body attribute name mapped to its wire name as repeated fields query keys, keeping caller order, repeats, empty and unknown names, and decodes the selected attributes into the typed model.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersSendAliasedRepeatedFields`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- GetWithFields with nil or empty fields sends the plain member GET without a query, which is Python's fields=None request, and a sparse response decodes with zero values for omitted attributes.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersWithoutFieldsSendPlainGet`.
- GetWithFields accepts HTTP 200 through 399 like the Python read, and returns 404 and other failures as the native gophercloud status error wrapped in resource.OperationError with the operation name.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersAcceptPythonStatusesAndWrapFailures`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.
- Get sends GET vpn/ikepolicies/{id} without any query and decodes the resource, which is Python's fields=None request.
  Tests: `network/v2/extensions/vpnaas/ikepolicies/python_parity_test.go:TestPythonIKEPolicyProxyRequests`, `network/v2/extensions/vpnaas/ikepolicies/python_parity_test.go:TestPythonIKEPolicyFindChainAndGetWithoutFields`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Go GetWithFields returns the Gophercloud v2.15.0 typed model, so attributes Neutron returns outside that model are dropped and attributes the server omits are zero values instead of None; Python returns a sparse Resource.
- The SDK field getter accepts HTTP 200..399 and keeps unknown field names in the query like Python; plain API.Get keeps the native status policy.

### `python:network/v2/vpn_ike_policies`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithRecordListConflictingAttrs`

- List sends GET vpn/ikepolicies with WithListOptions and WithListQuery values as query and follows the ikepolicies_links next href.
  Tests: `network/v2/extensions/vpnaas/ikepolicies/python_parity_test.go:TestPythonIKEPolicyListQueryAndLinks`.
- Owned nullable Source policy descriptors keep lifetime nested values raw, units separate and value integer; unknown/tenant/tag/revision fields do not leak into Resource while actual Wire/Envelope remain unmodified.
  Tests: `network/v2/extensions/vpnaas/ikepolicies/python_record_contracts_test.go:TestVPNPolicyRecordsDescriptorDefaultsAndActualProof`.
- Exact server query including phase1_negotiation_mode is separated from nested lifetime/top-level units/value local filters; allpages follow next and unknown values are discarded before encoding.
  Tests: `network/v2/extensions/vpnaas/ikepolicies/python_record_contracts_test.go:TestVPNPolicyRecordsServerQueryAndLocalBodyFiltersAllPages`.
- No-link limit marker continuation and paginated=false are SDK-owned; max_items counts raw rows before filtering and stops before invalid tail/foreign next.
  Tests: `network/v2/extensions/vpnaas/ikepolicies/python_record_contracts_test.go:TestVPNPolicyRecordsMarkerContinuationRawCapAndPaginationFlag`.
- Header/version/basepath/rawquery options are snapshotted and apply only to private operation copies.
  Tests: `network/v2/extensions/vpnaas/ikepolicies/python_record_contracts_test.go:TestVPNPolicyRecordsSnapshotControlsAndVersion`.
- Descriptor/constructor/shape errors retain accepted actual proof, physical body faults suppress retry callbacks, unsafe options/source changes fail before HTTP; Proxy generator JMESPath has correct noHTTP behavior.
  Tests: `network/v2/extensions/vpnaas/ikepolicies/python_record_contracts_test.go:TestVPNPolicyRecordsResponseErrorsAndPhysicalFaults`, `network/v2/extensions/vpnaas/ikepolicies/python_record_contracts_test.go:TestVPNPolicyRecordsPreflightAndProxyGenerator`.

Python/Go differences:

- Existing native typed List retains its native query/model behavior; new record APIs use pinned Source QueryParameters and apply non-query Body kwargs locally across all pages.
- Source policy lifetime is a nullable dict; comment defaults seconds/3600 are not descriptor defaults. Top-level units/value remain separate. Go JSONprofile preserves received bytes and owned descriptor projection, with literal path/origin safety and full Requests/NaN/Infinity/runtime-type differences.
- Source declaration/dict-conversion/nested-filter audit used22 AST-extracted assertions with stubs; this is not full Proxy/HTTP/decoder runtime or live Neutron evidence.
- Native ListOpts.Phase1NegotiationMode retains phase_1_negotiation_mode spelling; new WithRecordListQuery("phase1_negotiation_mode",...) sends the exact Source spelling.

### `python:network/v2/update_vpn_ike_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ikepolicies.WithUpdateField`

- Update sends PUT vpn/ikepolicies/{id} with {"ikepolicy": {...}} including explicit empty name or description pointers and decodes the result.
  Tests: `network/v2/extensions/vpnaas/ikepolicies/python_parity_test.go:TestPythonIKEPolicyProxyRequests`.

Python/Go differences:

- UpdateOpts.Phase1NegotiationMode is sent as phase_1_negotiation_mode; Python's phase1_negotiation_mode is sent with WithUpdateField("phase1_negotiation_mode", value), which is not a reserved Opts key.
- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Python sends no request when no attributes are given; Go Update always sends PUT with the envelope.
- Empty algorithm strings and a zero lifetime value are omitted; Neutron rejects those values.

### `python:network/v2/create_vpn_ipsec_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithCreateField`

- Create sends POST vpn/ipsecpolicies with {"ipsecpolicy": {...}} containing the given attributes and decodes the returned resource.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/python_parity_test.go:TestPythonIPsecPolicyProxyRequests`.

Python/Go differences:

- Empty algorithm strings and a zero lifetime value are omitted; Neutron rejects those values, so no valid Python request is lost.
- Python accepts any 2xx create response; the Go facade accepts only 201 or 202.
- Python sends unknown keyword attributes in the body only for resources with _allow_unknown_attrs_in_body; Go sends extension keys through With*Field, which rejects keys that collide with an Opts JSON tag.

### `python:network/v2/delete_vpn_ipsec_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE vpn/ipsecpolicies/{id}, returns nil for 404 and 204, reports resource.ErrNotFound with WithMissingError and keeps 409; plain Delete returns the 404.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/python_parity_test.go:TestPythonIPsecPolicyProxyRequests`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- Remove rejects IDs containing /, \, ?, #, %, or whitespace before HTTP, while Python sends any ID string.
- Python accepts any 2xx delete response; Go accepts 202 or 204.

### `python:network/v2/get_vpn_ipsec_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.GetWithFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.Get`

- GetWithFields sends GET vpn/ipsecpolicies/{id} with each Python VpnIpsecPolicy Body attribute name mapped to its wire name as repeated fields query keys, keeping caller order, repeats, empty and unknown names, and decodes the selected attributes into the typed model.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersSendAliasedRepeatedFields`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- GetWithFields with nil or empty fields sends the plain member GET without a query, which is Python's fields=None request, and a sparse response decodes with zero values for omitted attributes.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersWithoutFieldsSendPlainGet`.
- GetWithFields accepts HTTP 200 through 399 like the Python read, and returns 404 and other failures as the native gophercloud status error wrapped in resource.OperationError with the operation name.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersAcceptPythonStatusesAndWrapFailures`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.
- Get sends GET vpn/ipsecpolicies/{id} without any query and decodes the resource, which is Python's fields=None request.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/python_parity_test.go:TestPythonIPsecPolicyProxyRequests`, `network/v2/extensions/vpnaas/ipsecpolicies/python_parity_test.go:TestPythonIPsecPolicyFindChainAndGetWithoutFields`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Go GetWithFields returns the Gophercloud v2.15.0 typed model, so attributes Neutron returns outside that model are dropped and attributes the server omits are zero values instead of None; Python returns a sparse Resource.
- The SDK field getter accepts HTTP 200..399 and keeps unknown field names in the query like Python; plain API.Get keeps the native status policy.

### `python:network/v2/vpn_ipsec_policies`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.New`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListHeaders`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListBasePath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListMarker`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListJMESPath`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithRecordListConflictingAttrs`

- List sends GET vpn/ipsecpolicies with WithListOptions and WithListQuery values as query and follows the ipsecpolicies_links next href.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/python_parity_test.go:TestPythonIPsecPolicyListQueryAndLinks`.
- Owned nullable Source policy descriptors keep lifetime nested values raw, units separate and value integer; unknown/tenant/tag/revision fields do not leak into Resource while actual Wire/Envelope remain unmodified.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/python_record_contracts_test.go:TestVPNPolicyRecordsDescriptorDefaultsAndActualProof`.
- Exact server query including phase1_negotiation_mode is separated from nested lifetime/top-level units/value local filters; allpages follow next and unknown values are discarded before encoding.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/python_record_contracts_test.go:TestVPNPolicyRecordsServerQueryAndLocalBodyFiltersAllPages`.
- No-link limit marker continuation and paginated=false are SDK-owned; max_items counts raw rows before filtering and stops before invalid tail/foreign next.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/python_record_contracts_test.go:TestVPNPolicyRecordsMarkerContinuationRawCapAndPaginationFlag`.
- Header/version/basepath/rawquery options are snapshotted and apply only to private operation copies.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/python_record_contracts_test.go:TestVPNPolicyRecordsSnapshotControlsAndVersion`.
- Descriptor/constructor/shape errors retain accepted actual proof, physical body faults suppress retry callbacks, unsafe options/source changes fail before HTTP; Proxy generator JMESPath has correct noHTTP behavior.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/python_record_contracts_test.go:TestVPNPolicyRecordsResponseErrorsAndPhysicalFaults`, `network/v2/extensions/vpnaas/ipsecpolicies/python_record_contracts_test.go:TestVPNPolicyRecordsPreflightAndProxyGenerator`.

Python/Go differences:

- Existing native typed List retains its native query/model behavior; new record APIs use pinned Source QueryParameters and apply non-query Body kwargs locally across all pages.
- Source policy lifetime is a nullable dict; comment defaults seconds/3600 are not descriptor defaults. Top-level units/value remain separate. Go JSONprofile preserves received bytes and owned descriptor projection, with literal path/origin safety and full Requests/NaN/Infinity/runtime-type differences.
- Source declaration/dict-conversion/nested-filter audit used22 AST-extracted assertions with stubs; this is not full Proxy/HTTP/decoder runtime or live Neutron evidence.

### `python:network/v2/update_vpn_ipsec_policy`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/ipsecpolicies.WithUpdateField`

- Update sends PUT vpn/ipsecpolicies/{id} with {"ipsecpolicy": {...}} including explicit empty name or description pointers and decodes the result.
  Tests: `network/v2/extensions/vpnaas/ipsecpolicies/python_parity_test.go:TestPythonIPsecPolicyProxyRequests`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Python sends no request when no attributes are given; Go Update always sends PUT with the envelope.
- Empty algorithm strings and a zero lifetime value are omitted; Neutron rejects those values.

### `python:network/v2/create_vpn_ipsec_site_connection`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.API.Create`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithCreateField`

- Create sends POST vpn/ipsec-site-connections with {"ipsec_site_connection": {...}} containing the given attributes and decodes the returned resource.
  Tests: `network/v2/extensions/vpnaas/siteconnections/python_parity_test.go:TestPythonSiteConnectionProxyRequests`.

Python/Go differences:

- Go always sends ikepolicy_id, ipsecpolicy_id, vpnservice_id, peer_address, peer_id and psk, which Neutron requires anyway; zero mtu and DPD values are omitted.
- Python accepts any 2xx create response; the Go facade accepts only 201 or 202.
- Python sends unknown keyword attributes in the body only for resources with _allow_unknown_attrs_in_body; Go sends extension keys through With*Field, which rejects keys that collide with an Opts JSON tag.

### `python:network/v2/delete_vpn_ipsec_site_connection`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE vpn/ipsec-site-connections/{id}, returns nil for 404 and 204, reports resource.ErrNotFound with WithMissingError and keeps 409; plain Delete returns the 404.
  Tests: `network/v2/extensions/vpnaas/siteconnections/python_parity_test.go:TestPythonSiteConnectionProxyRequests`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- Remove rejects IDs containing /, \, ?, #, %, or whitespace before HTTP, while Python sends any ID string.
- Python accepts any 2xx delete response; Go accepts 202 or 204.

### `python:network/v2/get_vpn_ipsec_site_connection`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.API.GetWithFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.API.Get`

- GetWithFields sends GET vpn/ipsec-site-connections/{id} with each Python VpnIPSecSiteConnection Body attribute name mapped to its wire name as repeated fields query keys, keeping caller order, repeats, empty and unknown names, and decodes the selected attributes into the typed model.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersSendAliasedRepeatedFields`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- GetWithFields with nil or empty fields sends the plain member GET without a query, which is Python's fields=None request, and a sparse response decodes with zero values for omitted attributes.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersWithoutFieldsSendPlainGet`.
- GetWithFields accepts HTTP 200 through 399 like the Python read, and returns 404 and other failures as the native gophercloud status error wrapped in resource.OperationError with the operation name.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersAcceptPythonStatusesAndWrapFailures`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.
- Get sends GET vpn/ipsec-site-connections/{id} without any query and decodes the resource, which is Python's fields=None request.
  Tests: `network/v2/extensions/vpnaas/siteconnections/python_parity_test.go:TestPythonSiteConnectionProxyRequests`, `network/v2/extensions/vpnaas/siteconnections/python_parity_test.go:TestPythonSiteConnectionFindChainAndGetWithoutFields`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Go GetWithFields returns the Gophercloud v2.15.0 typed model, so attributes Neutron returns outside that model are dropped and attributes the server omits are zero values instead of None; Python returns a sparse Resource.
- The SDK field getter accepts HTTP 200..399 and keeps unknown field names in the query like Python; plain API.Get keeps the native status policy.

### `python:network/v2/vpn_ipsec_site_connections`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithListOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithRecordListOpts`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithRecordListQuery`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithRecordListJMESPath`

- List sends GET vpn/ipsec-site-connections with WithListOptions and WithListQuery values as query and follows the ipsec_site_connections_links next href.
  Tests: `network/v2/extensions/vpnaas/siteconnections/python_parity_test.go:TestPythonSiteConnectionListQueryAndLinks`.
- Owned Source profile classifies only declared query names/wires as server filters, applies non-query Body filters locally after descriptor conversion, discards unknown kwargs before marshaling, leaves missing fields null and separates descriptor view from physical raw response.
  Tests: `network/v2/extensions/vpnaas/siteconnections/python_record_list_test.go:TestVPNSiteRecordsDescriptorsDefaultsAndOwnedProof`, `network/v2/extensions/vpnaas/siteconnections/python_record_list_test.go:TestVPNSiteRecordsServerQueryAndLocalFiltersAcrossPages`.
- Lazy all-page records follow links and explicit-limit marker fallback including nonempty short pages; marker/count accounting precedes local filtering. AllRecords drops partial collection after failure while iterator consumers retain prior rows and actual native HTTP evidence.
  Tests: `network/v2/extensions/vpnaas/siteconnections/python_record_list_test.go:TestVPNSiteRecordsMarkerCountsPhysicalRowsBeforeFiltering`, `network/v2/extensions/vpnaas/siteconnections/python_record_list_test.go:TestVPNSiteRecordsFailureDropsAllPreservesYieldAndReceipt`.
- Owned options preserve snapshots, reject bound Source controls before HTTP and detect route/context changes. Proxy JMESPath preserves opaque generator identity without eagerly collecting HTTP rows, returning null/error for generator projections and invalid length exactly as the shared Source evaluator.
  Tests: `network/v2/extensions/vpnaas/siteconnections/python_record_list_test.go:TestVPNSiteRecordsControlsUnknownAndOwnedOptions`, `network/v2/extensions/vpnaas/siteconnections/python_record_list_test.go:TestVPNSiteRecordsProxyOpaqueGeneratorExpressions`, `network/v2/extensions/vpnaas/siteconnections/python_record_list_test.go:TestVPNSiteRecordsCancellationAndRouteGuard`.

Python/Go differences:

- The native generated List and its existing typed options remain available; the new ListRecords/AllRecords/ProxyListRecords own Source query/local Body classification, status query support, descriptor projection and paging rather than requiring caller filtering.
- Go uses copied finite JSON attributes and concrete typed options. Mutable Python Resource subclasses, arbitrary object values and dynamic constructor binding are outside this profile; safe literal base paths/headers/continuations retain shared SDK validation.
- ProxyListRecords preserves the upstream opaque-generator JMESPath semantics. ListRecords/AllRecords are typed conveniences and reject nonempty expressions rather than fabricating record associations for arbitrary results. This API mapping does not claim live Neutron VPN or full SDK lifecycle parity.

### `python:network/v2/update_vpn_ipsec_site_connection`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithUpdateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithUpdateRecordField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/vpnaas/siteconnections.WithUpdateRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/request.Null`
- `github.com/JSYoo5B/go-openstacksdk/request.Present`

- UpdateRecord sends PUT vpn/ipsec-site-connections/{id} with only the supplied attributes in the ipsec_site_connection envelope, including empty local_id, peer_address, peer_id, psk, initiator and names, an empty peer_cidrs list, mtu 0, an empty dpd object, admin_state_up false, a null endpoint group ID and other declared attributes through WithUpdateRecordField, accepts any status below 400, decodes enveloped and flat responses, and sends nothing when no attribute is supplied.
  Tests: `network/v2/extensions/vpnaas/siteconnections/python_update_record_test.go:TestPythonSiteConnectionUpdateRecordSendsEmptyValues`.
- Extension fields that collide with typed keys or the id identity, nil options and blank or dot IDs fail before HTTP, and a 404 response is returned as the Gophercloud status error wrapped in resource.OperationError.
  Tests: `network/v2/extensions/vpnaas/siteconnections/python_update_record_test.go:TestPythonSiteConnectionUpdateRecordRejectsInputsAndErrorStatuses`.

Python/Go differences:

- Python accepts a VpnIPSecSiteConnection Resource or an ID string; Go takes the ID string and rejects blank, '.' and '..' IDs before HTTP.
- Python takes is_admin_state_up as the keyword name; Go uses the AdminStateUp field that serializes as admin_state_up.
- Python drops undeclared keyword attributes because VpnIPSecSiteConnection does not allow unknown body attributes; WithUpdateRecordField sends any extension key except the typed keys and id.
- Python converts values of typed attributes such as mtu before sending; Go serializes the Go value as given.
- Python returns the Resource merged from request attributes and the response; Go returns ConnectionRecord with the native typed decode, the raw response object fields, headers and status, and Sent false when no request was needed.

### `python:network/v2/create_bgp_peer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.Create`

- Create sends POST bgp-peers with name, auth_type, peer_ip and remote_as in {"bgp_peer": {...}} and decodes the peer.
  Tests: `network/v2/extensions/bgp/peers/python_parity_test.go:TestPythonBGPPeerProxyRequests`.

Python/Go differences:

- Go always sends name, auth_type, peer_ip and remote_as; Python omits unset keys, so Neutron's auth_type default none must be passed explicitly in Go.
- remote_as is an int in Go; Python sends whatever JSON type the caller passed.
- Python accepts any 2xx create response; the Go facade accepts only 201 or 202.
- Go has no extension field option for create_bgp_peer; BgpPeer accepts unknown Python attributes in the body.

### `python:network/v2/delete_bgp_peer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE bgp-peers/{id}, returns nil for 404 and 204, reports resource.ErrNotFound with WithMissingError and keeps 409; plain Delete returns the 404.
  Tests: `network/v2/extensions/bgp/peers/python_parity_test.go:TestPythonBGPPeerProxyRequests`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- Remove rejects IDs containing /, \, ?, #, %, or whitespace before HTTP, while Python sends any ID string.
- Python accepts any 2xx delete response; Go accepts 202 or 204.

### `python:network/v2/get_bgp_peer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.Get`

- Get sends GET bgp-peers/{id} and decodes name, auth_type, peer_ip, remote_as and project_id.
  Tests: `network/v2/extensions/bgp/peers/python_parity_test.go:TestPythonBGPPeerProxyRequests`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Go decodes remote_as as an integer and fails on a string value.

### `python:network/v2/update_bgp_peer`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.WithUpdateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.WithUpdateRecordName`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.WithUpdateRecordPassword`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.WithUpdateRecordField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.WithUpdateRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/request.Null`
- `github.com/JSYoo5B/go-openstacksdk/request.Present`

- UpdateRecord sends PUT bgp-peers/{id} with only the supplied attributes in the bgp_peer envelope, including an empty name, a null name, an empty password and unknown body attributes through WithUpdateRecordField, accepts any status below 400, decodes enveloped, flat and empty responses, and sends nothing when no attribute is supplied.
  Tests: `network/v2/extensions/bgp/peers/python_update_record_test.go:TestPythonBGPPeerUpdateRecordSendsEmptyName`.
- Extension fields that collide with typed keys or the id identity, empty extension keys, nil options and blank or dot IDs fail before HTTP, and a 409 response is returned as the Gophercloud status error wrapped in resource.OperationError.
  Tests: `network/v2/extensions/bgp/peers/python_update_record_test.go:TestPythonBGPPeerUpdateRecordRejectsInputsAndErrorStatuses`.

Python/Go differences:

- Python accepts a BgpPeer Resource or an ID string; Go takes the ID string and rejects blank, '.' and '..' IDs before HTTP.
- Python silently discards an id attribute from the commit; WithUpdateRecordField rejects the id key before HTTP.
- Python returns the Resource merged from request attributes and the response; Go returns BGPPeerRecord with the native typed decode, the raw response object fields, headers and status, and Sent false when no request was needed.

### `python:network/v2/bgp_peers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.All`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithName`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/peers.WithRecordListQuery`

- Retained native API: List sends one GET bgp-peers without query and ignores bgp_peers_links; All with resource.WithName filters locally, and any wire query fails with resource.ErrUnsupported before HTTP.
  Tests: `network/v2/extensions/bgp/peers/python_parity_test.go:TestPythonBGPPeerListHasNoQueryOrPaging`.
- Lazy Source resource profile preserves tenant/project aliases and untyped fields with actual response evidence; only limit/marker are server query parameters while declared filters run locally across complete pages.
  Tests: `network/v2/extensions/bgp/peers/python_records_test.go:TestBGPPeerListOwnsAliasesUntypedFieldsAndPhysicalEvidence`, `network/v2/extensions/bgp/peers/python_records_test.go:TestBGPPeerListUsesLocalBodyFiltersAndCompleteMarkerPages`.
- Proxy JMESPath uses shared opaque generator identity without pre-consuming HTTP.
  Tests: `network/v2/extensions/bgp/peers/python_records_test.go:TestBGPPeerProxyJMESOwnsUnconsumedGenerator`.

Python/Go differences:

- Go uses owned resource views, explicit string identities and finite JSON options rather than mutable Python Resource instances or arbitrary object inputs. Typed headers and service-relative safe routes are library-owned adaptations. Shared session/Resource and live-cloud/release contracts remain open.

### `python:network/v2/create_bgp_speaker`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.Create`

- Create sends POST bgp-speakers with name, ip_version, local_as and both advertise flags in {"bgp_speaker": {...}} and decodes local_as as an integer.
  Tests: `network/v2/extensions/bgp/speakers/python_parity_test.go:TestPythonBGPSpeakerProxyRequests`.

Python/Go differences:

- Go always sends advertise_floating_ip_host_routes and advertise_tenant_networks (false when unset) while Python omits them and Neutron defaults both to true, so Go callers set them explicitly.
- LocalAS is a string in CreateOpts and is sent as "65000"; Python sends the caller's value, usually an integer, and Neutron converts both.
- Python accepts any 2xx create response; the Go facade accepts only 201 or 202.
- Go has no extension field option for create_bgp_speaker; BgpSpeaker accepts unknown Python attributes in the body.

### `python:network/v2/delete_bgp_speaker`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE bgp-speakers/{id}, returns nil for 404 and 204, reports resource.ErrNotFound with WithMissingError and keeps 409; plain Delete returns the 404.
  Tests: `network/v2/extensions/bgp/speakers/python_parity_test.go:TestPythonBGPSpeakerProxyRequests`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- Remove rejects IDs containing /, \, ?, #, %, or whitespace before HTTP, while Python sends any ID string.
- Python accepts any 2xx delete response; Go accepts 202 or 204.

### `python:network/v2/get_bgp_speaker`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.Get`

- Get sends GET bgp-speakers/{id} and decodes ip_version, local_as, the advertise flags, networks and peers.
  Tests: `network/v2/extensions/bgp/speakers/python_parity_test.go:TestPythonBGPSpeakerProxyRequests`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Go decodes local_as as an integer and fails on a string value; Go also exposes peers, which the Python resource does not declare.

### `python:network/v2/update_bgp_speaker`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.UpdateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithUpdateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithUpdateRecordName`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithUpdateRecordAdvertiseFloatingIPHostRoutes`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithUpdateRecordAdvertiseTenantNetworks`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithUpdateRecordField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithUpdateRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/request.Null`
- `github.com/JSYoo5B/go-openstacksdk/request.Present`

- UpdateRecord sends PUT bgp-speakers/{id} with only the supplied attributes in the bgp_speaker envelope, so a name-only update omits both advertise flags, while explicit false, null, an empty name and unknown body attributes through WithUpdateRecordField are sent; it accepts any status below 400, decodes enveloped and flat responses, and sends nothing when no attribute is supplied.
  Tests: `network/v2/extensions/bgp/speakers/python_update_record_test.go:TestPythonBGPSpeakerUpdateRecordSendsOnlySuppliedFlags`.
- Extension fields that collide with typed keys or the id identity, nil options and a blank ID fail before HTTP, and a 400 response is returned as the Gophercloud status error wrapped in resource.OperationError.
  Tests: `network/v2/extensions/bgp/speakers/python_update_record_test.go:TestPythonBGPSpeakerUpdateRecordRejectsInputsAndErrorStatuses`.

Python/Go differences:

- Python accepts a BgpSpeaker Resource or an ID string; Go takes the ID string and rejects blank, '.' and '..' IDs before HTTP.
- Python silently discards an id attribute from the commit; WithUpdateRecordField rejects the id key before HTTP.
- Python returns the Resource merged from request attributes and the response; Go returns BGPSpeakerRecord with the native typed decode, the raw response object fields, headers and status, and Sent false when no request was needed.

### `python:network/v2/bgp_speakers`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.List`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.All`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.ListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.AllRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.ProxyListRecords`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.WithRecordListQuery`

- Retained native API: List sends one GET bgp-speakers without query and ignores bgp_speakers_links, and any wire query fails with resource.ErrUnsupported before HTTP.
  Tests: `network/v2/extensions/bgp/speakers/python_parity_test.go:TestPythonBGPSpeakerListAndFindHaveNoQuery`.
- Lazy Source resource profile preserves tenant/project aliases and untyped fields with actual response evidence; only limit/marker are server query parameters while declared filters run locally across complete pages.
  Tests: `network/v2/extensions/bgp/speakers/python_records_test.go:TestBGPSpeakerListOwnsAliasesUntypedFieldsAndPhysicalEvidence`, `network/v2/extensions/bgp/speakers/python_records_test.go:TestBGPSpeakerListUsesLocalBodyFiltersAndCompleteMarkerPages`.
- Proxy JMESPath uses shared opaque generator identity without pre-consuming HTTP.
  Tests: `network/v2/extensions/bgp/speakers/python_records_test.go:TestBGPSpeakerProxyJMESOwnsUnconsumedGenerator`.

Python/Go differences:

- Go uses owned resource views, explicit string identities and finite JSON options rather than mutable Python Resource instances or arbitrary object inputs. Typed headers and service-relative safe routes are library-owned adaptations. Shared session/Resource and live-cloud/release contracts remain open.

### `python:network/v2/add_bgp_peer_to_speaker`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.AddBGPPeer`

- AddBGPPeer sends PUT bgp-speakers/{id}/add_bgp_peer with {"bgp_peer_id": ...} and decodes bgp_peer_id from the response.
  Tests: `network/v2/extensions/bgp/speakers/python_parity_test.go:TestPythonBGPSpeakerPeerActions`.

Python/Go differences:

- Python returns the whole response dict; Go decodes only bgp_peer_id into AddBGPPeerOpts.
- Python accepts a peer Resource or ID; Go takes the peer ID inside AddBGPPeerOpts.
- Python accepts any 2xx; Go accepts only 200.

### `python:network/v2/remove_bgp_peer_from_speaker`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.RemoveBGPPeer`

- RemoveBGPPeer sends PUT bgp-speakers/{id}/remove_bgp_peer with {"bgp_peer_id": ...}, ignores the body and returns the HTTP error like Python's raise_from_response.
  Tests: `network/v2/extensions/bgp/speakers/python_parity_test.go:TestPythonBGPSpeakerPeerActions`.

Python/Go differences:

- Python accepts any 2xx; Go accepts only 200.

### `python:network/v2/get_bgp_dragents_hosting_speaker`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.ListDRAgentHostingBGPSpeakers`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.List`

- ListDRAgentHostingBGPSpeakers sends GET bgp-speakers/{id}/bgp-dragents and decodes the agents; repeating Python's GET agents and keeping the hosting IDs gives the same rows.
  Tests: `network/v2/extensions/bgp/speakers/python_parity_test.go:TestPythonBGPSpeakerDynamicRoutingAgentCalls`.

Python/Go differences:

- Python always issues a second GET agents and returns those rows filtered by the hosting IDs; Go returns the hosting rows directly and the second list is optional.
- ListDRAgentHostingBGPSpeakers follows agents_links like the Python agent list.

### `python:network/v2/add_bgp_speaker_to_dragent`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.ScheduleBGPSpeaker`

- ScheduleBGPSpeaker sends POST agents/{agent}/bgp-drinstances with {"bgp_speaker_id": ...} and reports a non-201 status such as 409.
  Tests: `network/v2/extensions/bgp/speakers/python_parity_test.go:TestPythonBGPSpeakerDynamicRoutingAgentCalls`.

Python/Go differences:

- Python never checks the POST response and returns None even on HTTP errors; Go returns the error for any status other than 201.
- Python accepts agent and speaker Resources or IDs; Go takes the two ID strings.

### `python:network/v2/get_bgp_speakers_hosted_by_dragent`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.ListBGPSpeakers`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgp/speakers.API.List`

- ListBGPSpeakers sends GET agents/{agent}/bgp-drinstances and decodes the speakers; repeating Python's GET bgp-speakers and keeping the hosted IDs gives the same rows.
  Tests: `network/v2/extensions/bgp/speakers/python_parity_test.go:TestPythonBGPSpeakerDynamicRoutingAgentCalls`.

Python/Go differences:

- Python always issues a second GET bgp-speakers and returns those rows filtered by the hosted IDs; Go returns the hosted rows directly and the second list is optional.
- The Go speaker list reads one page while Python follows bgp_speakers_links.

### `python:network/v2/remove_bgp_speaker_from_dragent`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/agents.API.RemoveBGPSpeaker`

- RemoveBGPSpeaker sends DELETE agents/{agent}/bgp-drinstances/{speaker} and reports an HTTP error such as 404.
  Tests: `network/v2/extensions/bgp/speakers/python_parity_test.go:TestPythonBGPSpeakerDynamicRoutingAgentCalls`.

Python/Go differences:

- Python never checks the DELETE response and returns None even on HTTP errors; Go returns the error.
- Python accepts agent and speaker Resources or IDs; Go takes the two ID strings.

### `python:network/v2/create_bgpvpn`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.CreateRecord`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreateRecordOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreateRecordLocalPref`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreateRecordField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreateRecordHeader`
- `github.com/JSYoo5B/go-openstacksdk/request.Null`
- `github.com/JSYoo5B/go-openstacksdk/request.Present`

- CreateRecord sends POST bgpvpn/bgpvpns with only the supplied attributes in the bgpvpn envelope, including local_pref 0, vni 0, an empty name, an empty route_targets list, null import_targets and unknown body attributes through WithCreateRecordField, sends an empty envelope when no attribute is supplied, accepts any status below 400 and decodes enveloped, flat and empty responses.
  Tests: `network/v2/extensions/bgpvpns/python_create_record_test.go:TestPythonBGPVPNCreateRecordSendsZeroLocalPref`.
- Extension fields that collide with typed keys, blank extension keys and nil options fail before HTTP, and a 400 response is returned as the Gophercloud status error wrapped in resource.OperationError.
  Tests: `network/v2/extensions/bgpvpns/python_create_record_test.go:TestPythonBGPVPNCreateRecordRejectsInputsAndErrorStatuses`.

Python/Go differences:

- Python converts values of list-typed attributes before sending; Go serializes the Go value as given.
- Python returns the Resource merged from request attributes and the response; Go returns BGPVPNRecord with the native typed decode, the raw response object fields, headers and status.

### `python:network/v2/delete_bgpvpn`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.Remove`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/resource.ID`
- `github.com/JSYoo5B/go-openstacksdk/resource.WithMissingError`

- Remove sends DELETE bgpvpn/bgpvpns/{id}, returns nil for 404 and 204, reports resource.ErrNotFound with WithMissingError and keeps 409; plain Delete returns the 404.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNProxyRequests`.

Python/Go differences:

- Python ignore_missing=True maps to Remove(ctx, resource.ID(id)) and ignore_missing=False to resource.WithMissingError(); the plain generated Delete returns the 404 error.
- Remove rejects IDs containing /, \, ?, #, %, or whitespace before HTTP, while Python sends any ID string.
- Python accepts any 2xx delete response; Go accepts 202 or 204.

### `python:network/v2/get_bgpvpn`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.GetWithFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.Get`

- GetWithFields sends GET bgpvpn/bgpvpns/{id} with each Python BgpVpn Body attribute name mapped to its wire name as repeated fields query keys, keeping caller order, repeats, empty and unknown names, and decodes the selected attributes into the typed model.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersSendAliasedRepeatedFields`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- GetWithFields with nil or empty fields sends the plain member GET without a query, which is Python's fields=None request, and a sparse response decodes with zero values for omitted attributes.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersWithoutFieldsSendPlainGet`.
- GetWithFields accepts HTTP 200 through 399 like the Python read, and returns 404 and other failures as the native gophercloud status error wrapped in resource.OperationError with the operation name.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersAcceptPythonStatusesAndWrapFailures`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationRejectsNativeResultAndInputDrift`.
- Get sends GET bgpvpn/bgpvpns/{id} without any query and decodes the BGP VPN.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNProxyRequests`, `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNFindListsWithoutNameAndGetHasNoFields`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Go GetWithFields returns the Gophercloud v2.15.0 typed model, so attributes Neutron returns outside that model are dropped and attributes the server omits are zero values instead of None; Python returns a sparse Resource.
- The SDK field getter accepts HTTP 200..399 and keeps unknown field names in the query like Python; plain API.Get keeps the native status policy.

### `python:network/v2/update_bgpvpn`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithUpdateField`

- Update sends PUT bgpvpn/bgpvpns/{id} with pointer fields, so an empty name, an empty target list and local_pref 0 are sent explicitly.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNProxyRequests`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Python sends no request when no attributes are given; Go Update always sends PUT with the envelope.

### `python:network/v2/create_bgpvpn_network_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.CreateNetworkAssociation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreateNetworkAssociationField`

- CreateNetworkAssociation sends POST bgpvpn/bgpvpns/{bgpvpn}/network_associations with {"network_association": {...}} and decodes the association.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNAssociationRequests`.

Python/Go differences:

- Python accepts a BgpVpn Resource or ID; Go takes the BGP VPN ID string.
- Go rejects an empty network_id before HTTP.
- Python accepts any 2xx create response; the Go facade accepts only 201 or 202.

### `python:network/v2/delete_bgpvpn_network_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.DeleteNetworkAssociation`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- resource.IgnoreMissing(DeleteNetworkAssociation(...)) sends DELETE bgpvpn/bgpvpns/{bgpvpn}/network_associations/{id}, returns nil for 204 and 404 and keeps 409.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNAssociationRequests`.

Python/Go differences:

- Python ignore_missing=True maps to resource.IgnoreMissing around the generated delete; ignore_missing=False is the plain call, which returns the 404 error.
- Python accepts any 2xx delete response; Go accepts 202 or 204.

### `python:network/v2/get_bgpvpn_network_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.GetNetworkAssociationWithFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.GetNetworkAssociation`

- GetNetworkAssociationWithFields sends GET bgpvpn/bgpvpns/{bgpvpn_id}/network_associations/{id} with each Python BgpVpnNetworkAssociation Body attribute name mapped to its wire name as repeated fields query keys, keeping caller order, repeats, empty and unknown names, and decodes the selected attributes into the typed model.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersSendAliasedRepeatedFields`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- GetNetworkAssociationWithFields with nil or empty fields sends the plain member GET without a query, which is Python's fields=None request, and a sparse response decodes with zero values for omitted attributes.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersWithoutFieldsSendPlainGet`.
- GetNetworkAssociationWithFields accepts HTTP 200 through 399 like the Python read, and returns 404 and other failures as the native gophercloud status error wrapped in resource.OperationError with the operation name.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersAcceptPythonStatusesAndWrapFailures`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronChildFieldGenerationValidatesNativeParentGetter`.
- The generated bgpvpns package keeps the parent BGP VPN getter and the three association field getters in one file, each with its own Python Body alias map and a parent ID route segment.
  Tests: `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldRenderingGroupsParentAndChildGetters`.
- GetNetworkAssociation sends GET bgpvpn/bgpvpns/{bgpvpn}/network_associations/{id} without any query and decodes the association.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNAssociationRequests`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Go GetWithFields returns the Gophercloud v2.15.0 typed model, so attributes Neutron returns outside that model are dropped and attributes the server omits are zero values instead of None; Python returns a sparse Resource.
- The SDK field getter accepts HTTP 200..399 and keeps unknown field names in the query like Python; plain API.Get keeps the native status policy.
- Python accepts a BgpVpn Resource or ID for the parent and an association Resource or ID; Go takes the parent BGP VPN ID string and the association ID string.

### `python:network/v2/bgpvpn_network_associations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.ListNetworkAssociations`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithListNetworkAssociationsOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithListNetworkAssociationsQuery`

- ListNetworkAssociations sends GET bgpvpn/bgpvpns/{bgpvpn}/network_associations with fields, limit and extension query values and continues with the last ID as marker while limit is set.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNAssociationRequests`.

Python/Go differences:

- Python's association QueryParameters only hold limit and marker, so other keywords such as network_id are local filters in Python; Go sends WithListQuery keys to the server.
- Go builds the next page from the current URL and marker and ignores *_links; Python follows the links and falls back to the same marker request.

### `python:network/v2/create_bgpvpn_port_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.CreatePortAssociation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreatePortAssociationField`

- CreatePortAssociation sends POST bgpvpn/bgpvpns/{bgpvpn}/port_associations with {"port_association": {...}} and decodes the association.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNAssociationRequests`.

Python/Go differences:

- Python accepts a BgpVpn Resource or ID; Go takes the BGP VPN ID string.
- Go rejects an empty port_id or a route without type before HTTP.
- Python accepts any 2xx create response; the Go facade accepts only 201 or 202.

### `python:network/v2/delete_bgpvpn_port_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.DeletePortAssociation`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- resource.IgnoreMissing(DeletePortAssociation(...)) sends DELETE bgpvpn/bgpvpns/{bgpvpn}/port_associations/{id}, returns nil for 204 and 404 and keeps 409.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNAssociationRequests`.

Python/Go differences:

- Python ignore_missing=True maps to resource.IgnoreMissing around the generated delete; ignore_missing=False is the plain call, which returns the 404 error.
- Python accepts any 2xx delete response; Go accepts 202 or 204.

### `python:network/v2/get_bgpvpn_port_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.GetPortAssociationWithFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.GetPortAssociation`

- GetPortAssociationWithFields sends GET bgpvpn/bgpvpns/{bgpvpn_id}/port_associations/{id} with each Python BgpVpnPortAssociation Body attribute name mapped to its wire name as repeated fields query keys, keeping caller order, repeats, empty and unknown names, and decodes the selected attributes into the typed model.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersSendAliasedRepeatedFields`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- GetPortAssociationWithFields with nil or empty fields sends the plain member GET without a query, which is Python's fields=None request, and a sparse response decodes with zero values for omitted attributes.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersWithoutFieldsSendPlainGet`.
- GetPortAssociationWithFields accepts HTTP 200 through 399 like the Python read, and returns 404 and other failures as the native gophercloud status error wrapped in resource.OperationError with the operation name.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersAcceptPythonStatusesAndWrapFailures`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronChildFieldGenerationValidatesNativeParentGetter`.
- The generated bgpvpns package keeps the parent BGP VPN getter and the three association field getters in one file, each with its own Python Body alias map and a parent ID route segment.
  Tests: `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldRenderingGroupsParentAndChildGetters`.
- GetPortAssociation sends GET bgpvpn/bgpvpns/{bgpvpn}/port_associations/{id} without any query and decodes the association.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNAssociationRequests`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Go GetWithFields returns the Gophercloud v2.15.0 typed model, so attributes Neutron returns outside that model are dropped and attributes the server omits are zero values instead of None; Python returns a sparse Resource.
- The SDK field getter accepts HTTP 200..399 and keeps unknown field names in the query like Python; plain API.Get keeps the native status policy.
- Python accepts a BgpVpn Resource or ID for the parent and an association Resource or ID; Go takes the parent BGP VPN ID string and the association ID string.

### `python:network/v2/bgpvpn_port_associations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.ListPortAssociations`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithListPortAssociationsOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithListPortAssociationsQuery`

- ListPortAssociations sends GET bgpvpn/bgpvpns/{bgpvpn}/port_associations with fields, limit and extension query values and continues with the last ID as marker while limit is set.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNAssociationRequests`.

Python/Go differences:

- Python's association QueryParameters only hold limit and marker, so other keywords such as port_id are local filters in Python; Go sends WithListQuery keys to the server.
- Go builds the next page from the current URL and marker and ignores *_links; Python follows the links and falls back to the same marker request.

### `python:network/v2/update_bgpvpn_port_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.UpdatePortAssociation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithUpdatePortAssociationField`

- UpdatePortAssociation sends PUT bgpvpn/bgpvpns/{bgpvpn}/port_associations/{id} with pointer fields, including explicit false flags and an empty routes list.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNAssociationRequests`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Python sends no request when no attributes are given; Go Update always sends PUT with the envelope.

### `python:network/v2/create_bgpvpn_router_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.CreateRouterAssociation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithCreateRouterAssociationField`

- CreateRouterAssociation sends POST bgpvpn/bgpvpns/{bgpvpn}/router_associations with {"router_association": {...}} and decodes the association.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNAssociationRequests`.

Python/Go differences:

- Python accepts a BgpVpn Resource or ID; Go takes the BGP VPN ID string.
- Go rejects an empty router_id before HTTP.
- Python accepts any 2xx create response; the Go facade accepts only 201 or 202.

### `python:network/v2/delete_bgpvpn_router_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.DeleteRouterAssociation`
- `github.com/JSYoo5B/go-openstacksdk/resource.IgnoreMissing`

- resource.IgnoreMissing(DeleteRouterAssociation(...)) sends DELETE bgpvpn/bgpvpns/{bgpvpn}/router_associations/{id}, returns nil for 204 and 404 and keeps 409.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNAssociationRequests`.

Python/Go differences:

- Python ignore_missing=True maps to resource.IgnoreMissing around the generated delete; ignore_missing=False is the plain call, which returns the 404 error.
- Python accepts any 2xx delete response; Go accepts 202 or 204.

### `python:network/v2/get_bgpvpn_router_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.GetRouterAssociationWithFields`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.GetRouterAssociation`

- GetRouterAssociationWithFields sends GET bgpvpn/bgpvpns/{bgpvpn_id}/router_associations/{id} with each Python BgpVpnRouterAssociation Body attribute name mapped to its wire name as repeated fields query keys, keeping caller order, repeats, empty and unknown names, and decodes the selected attributes into the typed model.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersSendAliasedRepeatedFields`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldExtractionUsesBodyMROWithoutExecutingSource`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldGenerationMatchesFreshPinnedSource`.
- GetRouterAssociationWithFields with nil or empty fields sends the plain member GET without a query, which is Python's fields=None request, and a sparse response decodes with zero values for omitted attributes.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersWithoutFieldsSendPlainGet`.
- GetRouterAssociationWithFields accepts HTTP 200 through 399 like the Python read, and returns 404 and other failures as the native gophercloud status error wrapped in resource.OperationError with the operation name.
  Tests: `api/neutron_python_field_getters_test.go:TestNeutronPythonFieldGettersAcceptPythonStatusesAndWrapFailures`, `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronChildFieldGenerationValidatesNativeParentGetter`.
- The generated bgpvpns package keeps the parent BGP VPN getter and the three association field getters in one file, each with its own Python Body alias map and a parent ID route segment.
  Tests: `internal/cmd/sdkgen/neutron_fields_test.go:TestNeutronFieldRenderingGroupsParentAndChildGetters`.
- GetRouterAssociation sends GET bgpvpn/bgpvpns/{bgpvpn}/router_associations/{id} without any query and decodes the association.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNAssociationRequests`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Go GetWithFields returns the Gophercloud v2.15.0 typed model, so attributes Neutron returns outside that model are dropped and attributes the server omits are zero values instead of None; Python returns a sparse Resource.
- The SDK field getter accepts HTTP 200..399 and keeps unknown field names in the query like Python; plain API.Get keeps the native status policy.
- Python accepts a BgpVpn Resource or ID for the parent and an association Resource or ID; Go takes the parent BGP VPN ID string and the association ID string.

### `python:network/v2/bgpvpn_router_associations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.ListRouterAssociations`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithListRouterAssociationsOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithListRouterAssociationsQuery`

- ListRouterAssociations sends GET bgpvpn/bgpvpns/{bgpvpn}/router_associations with fields, limit and extension query values and continues with the last ID as marker while limit is set.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNAssociationRequests`.

Python/Go differences:

- Python's association QueryParameters only hold limit and marker, so other keywords such as router_id are local filters in Python; Go sends WithListQuery keys to the server.
- Go builds the next page from the current URL and marker and ignores *_links; Python follows the links and falls back to the same marker request.

### `python:network/v2/update_bgpvpn_router_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.UpdateRouterAssociation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.WithUpdateRouterAssociationField`

- UpdateRouterAssociation sends PUT bgpvpn/bgpvpns/{bgpvpn}/router_associations/{id} with pointer fields, including explicit false flags.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNAssociationRequests`.

Python/Go differences:

- Python accepts a Resource or an ID string and returns a mutable Resource; Go takes ID strings and returns the native Gophercloud model pointer with zero values for null or omitted fields.
- Python sends no request when no attributes are given; Go Update always sends PUT with the envelope.

### `python:network/v2/find_bgpvpn_port_association`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.GetPortAssociation`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.ListPortAssociations`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/bgpvpns.API.FindPortAssociationRecord`

- GetPortAssociation returns the 404 error for a missing ID and ListPortAssociations sends the unfiltered association list, the two requests Python's find sends.
  Tests: `network/v2/extensions/bgpvpns/python_parity_test.go:TestPythonBGPVPNPortAssociationFindHasNoHelper`.
- Owned parent-scoped GET-first find reuses Source fetch controls, declared Body defaults/casts, local fallback filters, complete pagination and late duplicate detection; raw HTTP proof and terminal faults remain distinct.
  Tests: `network/v2/extensions/bgpvpns/python_port_association_find_test.go:TestPythonPortAssociationFindMemberProjectionAndOwnership`, `network/v2/extensions/bgpvpns/python_port_association_find_test.go:TestPythonPortAssociationFindFallbackLocalFiltersAndLateDuplicates`, `network/v2/extensions/bgpvpns/python_port_association_find_test.go:TestPythonPortAssociationFindMissingAndStrictJSONIdentity`, `network/v2/extensions/bgpvpns/python_port_association_find_test.go:TestPythonPortAssociationFindRawFetchControlsAndHeadersPhase`, `network/v2/extensions/bgpvpns/python_port_association_find_test.go:TestPythonPortAssociationFindOptionsGuardAndPreflight`, `network/v2/extensions/bgpvpns/python_port_association_find_test.go:TestPythonPortAssociationFindPhysicalFaultNeverFallback`.

Python/Go differences:

- Go takes fixed parent and lookup string IDs plus finite UTF-8 JSON options; arbitrary Python Resource/class/session/connection/location runtime and shared cache are not modeled. Raw URI formatter supports only common literal/%(id)s/%% paths, not dynamic %(bgpvpn_id)s; the parent argument supplies that URI value.
- SDK-R1/C1/S1, real-cloud compatibility and release validation remain; local tests and source review do not establish full SDK parity.

## References

- [`network/v2/_proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [`vpn_service.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/vpn_service.py)
- [`vpn_endpoint_group.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/vpn_endpoint_group.py)
- [`vpn_ike_policy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/vpn_ike_policy.py)
- [`vpn_ipsec_policy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/vpn_ipsec_policy.py)
- [`vpn_ipsec_site_connection.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/vpn_ipsec_site_connection.py)
- [`bgp_peer.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgp_peer.py)
- [`bgp_speaker.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgp_speaker.py)
- [`agent.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/agent.py)
- [`bgpvpn.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgpvpn.py)
- [network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgpvpn_network_association.py)
- [port](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgpvpn_port_association.py)
- [router](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/bgpvpn_router_association.py)
- [`proxy.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py)
- [`resource.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [native-bgp.md](native-bgp.md)
- [SDK Reviews](../../api/sdk_reviews.json)
- [Python Records](extensions/vpnaas/services/python-records.md)
- [Python Records](extensions/vpnaas/endpointgroups/python-records.md)
- [Python Records](extensions/vpnaas/ikepolicies/python-records.md)
- [Python Records](extensions/vpnaas/ipsecpolicies/python-records.md)
- [Python Record List](extensions/vpnaas/siteconnections/python-record-list.md)
- [Python Records](extensions/bgp/peers/python-records.md)
- [Python Records](extensions/bgp/speakers/python-records.md)
- [Python List Records](extensions/bgpvpns/python-list-records.md)
- [Python Port Association Find](extensions/bgpvpns/python-port-association-find.md)
- [BGP peer](extensions/bgp/peers/python-records.md)
- [speaker](extensions/bgp/speakers/python-records.md)
