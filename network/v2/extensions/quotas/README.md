# Quotas

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:networking/v2/extensions/quotas.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.API.CurrentProject`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.ProjectQuotaScope.Get`

- Explicit project IDs perform no lookup; exact names are resolved once with a separate Keystone v3 client across pages, retaining missing/duplicate/invalid-ID/403 failures and selected Network-client headers.
  Tests: `api/neutron_project_quotas_projects_test.go:TestNeutronProjectQuotaResolvesNameOnceUsingSeparateIdentityClient`, `api/neutron_project_quotas_projects_test.go:TestNeutronProjectQuotaLookupFailuresPreventNetworkRequests`, `connection_project_quotas_test.go:TestConnectionServiceQuotaNamesUseSeparateIdentityClient`.
- CurrentProject freezes the recorded Keystone v2/v3 project ID; later auth changes do not move the target, and missing/unsupported/typed-nil/system/domain/malformed auth results fail before HTTP.
  Tests: `api/neutron_project_quotas_projects_test.go:TestNeutronProjectQuotaCurrentProjectUsesRecordedV3AndV2Auth`, `api/neutron_project_quotas_projects_test.go:TestNeutronProjectQuotaCurrentProjectRejectsMissingOrMalformedAuth`, `connection_project_quotas_test.go:TestConnectionServiceQuotasIDAuthAndCancellation`.
- GET uses /quotas/{fixedProject}, keeps native zero/unlimited limits, exact raw extension/null/large-integer fields, selected ResourceBase/configured headers and independent response metadata.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaOperationsKeepTargetRawJSONAndMetadata`.
- Missing/null/array/scalar quota envelopes and wrong known field types fail; Detail accepts native integer-string reserved values but rejects malformed integer strings.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaRejectsMalformedObjects`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaOperationsKeepTargetRawJSONAndMetadata`.
- GET/PUT retain native HTTP 200 only, while DELETE accepts native 202 and 204 and returns the actual accepted code with independent headers.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaSuccessCodesMatchNativePolicy`.
- Get, Detail, Update and Delete preserve 404/403/500 code/body/header causes, pre-canceled contexts and ErrNotFound only for 404; in-flight Update cancellation and Detail deadlines remain observable.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaHTTPFailuresAndStrictMissingPolicy`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaCancellationPreservesCause`.

Python/Go differences:

- The singleton scope replaces per-call project IDs and result extraction with native typed values plus fixed ProjectID, raw Body, copied Header and HTTP StatusCode.

### `gophercloud:networking/v2/extensions/quotas.GetDetail`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.API.CurrentProject`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.ProjectQuotaScope.Detail`

- Explicit project IDs perform no lookup; exact names are resolved once with a separate Keystone v3 client across pages, retaining missing/duplicate/invalid-ID/403 failures and selected Network-client headers.
  Tests: `api/neutron_project_quotas_projects_test.go:TestNeutronProjectQuotaResolvesNameOnceUsingSeparateIdentityClient`, `api/neutron_project_quotas_projects_test.go:TestNeutronProjectQuotaLookupFailuresPreventNetworkRequests`, `connection_project_quotas_test.go:TestConnectionServiceQuotaNamesUseSeparateIdentityClient`.
- CurrentProject freezes the recorded Keystone v2/v3 project ID; later auth changes do not move the target, and missing/unsupported/typed-nil/system/domain/malformed auth results fail before HTTP.
  Tests: `api/neutron_project_quotas_projects_test.go:TestNeutronProjectQuotaCurrentProjectUsesRecordedV3AndV2Auth`, `api/neutron_project_quotas_projects_test.go:TestNeutronProjectQuotaCurrentProjectRejectsMissingOrMalformedAuth`, `connection_project_quotas_test.go:TestConnectionServiceQuotasIDAuthAndCancellation`.
- GET uses native /quotas/{fixedProject}/details.json, retains Used/Reserved/Limit values and exact nested raw fields including integer-string reserved compatibility.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaOperationsKeepTargetRawJSONAndMetadata`.
- Missing/null/array/scalar quota envelopes and wrong known field types fail; Detail accepts native integer-string reserved values but rejects malformed integer strings.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaRejectsMalformedObjects`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaOperationsKeepTargetRawJSONAndMetadata`.
- GET/PUT retain native HTTP 200 only, while DELETE accepts native 202 and 204 and returns the actual accepted code with independent headers.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaSuccessCodesMatchNativePolicy`.
- Get, Detail, Update and Delete preserve 404/403/500 code/body/header causes, pre-canceled contexts and ErrNotFound only for 404; in-flight Update cancellation and Detail deadlines remain observable.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaHTTPFailuresAndStrictMissingPolicy`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaCancellationPreservesCause`.

Python/Go differences:

- The singleton scope replaces per-call project IDs and result extraction with native typed values plus fixed ProjectID, raw Body, copied Header and HTTP StatusCode.

### `gophercloud:networking/v2/extensions/quotas.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.ProjectQuotaScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.WithQuotaOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.WithUpdateCheckLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.WithUpdateField`

- Nil limits omit, zero/-1 remain explicit and check_limit omission/true/false are distinct with last-option precedence; integer values stay exact and the serialized body is unchanged across 401 reauthentication.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaUpdateOmittedFalseZeroAndSnapshotAcrossReauth`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaOperationsKeepTargetRawJSONAndMetadata`.
- WithQuotaOptions freezes typed pointers at option construction and provides new pointers on each application, preserving reuse after callers mutate either the original input or a previous application.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaOptionsSnapshotAtCreationAndRemainReusable`.
- Nil options, invalid typed limits, core/check_limit field collisions, nonserializable extensions and unsupported query/header/argument options fail before HTTP.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaOptionsFailBeforeHTTP`.
- Missing/null/array/scalar quota envelopes and wrong known field types fail; Detail accepts native integer-string reserved values but rejects malformed integer strings.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaRejectsMalformedObjects`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaOperationsKeepTargetRawJSONAndMetadata`.
- GET/PUT retain native HTTP 200 only, while DELETE accepts native 202 and 204 and returns the actual accepted code with independent headers.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaSuccessCodesMatchNativePolicy`.
- Get, Detail, Update and Delete preserve 404/403/500 code/body/header causes, pre-canceled contexts and ErrNotFound only for 404; in-flight Update cancellation and Detail deadlines remain observable.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaHTTPFailuresAndStrictMissingPolicy`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaCancellationPreservesCause`.

Python/Go differences:

- Raw JSON request construction preserves exact integers and retry snapshots instead of native any/float64 conversion; callers use concrete options and SDK-owned field/flag helpers rather than custom builders.
- WithUpdateCheckLimit supplies an explicit optional boolean absent from pinned native UpdateOpts; extension/usage-check support remains a server decision.

### `gophercloud:networking/v2/extensions/quotas.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.ProjectQuotaScope.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.WithDeleteIgnoreMissing`

- DELETE resets fixed-project overrides with no request body or follow-up GET and exposes actual native 202/204 status plus independent response headers.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaOperationsKeepTargetRawJSONAndMetadata`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaSuccessCodesMatchNativePolicy`.
- Delete is strict by default; explicit ignore-missing suppresses only 404, a later false restores strict errors and nil delete options fail before HTTP.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaHTTPFailuresAndStrictMissingPolicy`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaOptionsFailBeforeHTTP`.
- Get, Detail, Update and Delete preserve 404/403/500 code/body/header causes, pre-canceled contexts and ErrNotFound only for 404; in-flight Update cancellation and Detail deadlines remain observable.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaHTTPFailuresAndStrictMissingPolicy`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaCancellationPreservesCause`.

Python/Go differences:

- Delete returns metadata rather than requiring extraction from an empty response; strict missing behavior preserves the native default and can be explicitly changed.

### `python:network/v2/get_quota`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.ProjectQuotaScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.ProjectQuotaScope.Detail`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.API.InProject`

- Ordinary and detailed quota variants map to separate typed methods with fixed project identity and raw/metadata evidence.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaOperationsKeepTargetRawJSONAndMetadata`.
- Missing/null/array/scalar quota envelopes and wrong known field types fail; Detail accepts native integer-string reserved values but rejects malformed integer strings.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaRejectsMalformedObjects`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaOperationsKeepTargetRawJSONAndMetadata`.
- Get, Detail, Update and Delete preserve 404/403/500 code/body/header causes, pre-canceled contexts and ErrNotFound only for 404; in-flight Update cancellation and Detail deadlines remain observable.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaHTTPFailuresAndStrictMissingPolicy`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaCancellationPreservesCause`.
- get_quota(project) maps to InProject(resource.ID(project)) (no HTTP) and ProjectQuotaScope.Get, which sends GET /quotas/<project> and decodes the limits; details=True maps to Detail, which sends GET /quotas/<project>/details.json and decodes used/reserved/limit; a 404 is resource.ErrNotFound like NotFoundException.
  Tests: `network/v2/extensions/quotas/python_rejudge_test.go:TestPythonRejudgeQuotaGet`.

Python/Go differences:

- Go Detail retains native /details.json routing; pinned Python QuotaDetails uses /details.
- Native detail triples remain typed in Go; Python quota properties use plural attribute names and dictionary usage values.
- The Detail path carries the .json format suffix that the Neutron API reference documents; Neutron serves it from the same quota details resource as the Python /details path.
- Python accepts a Quota Resource or a project ID and uses plural attribute names (networks, floating_ips, health_monitors, ...); Go fixes the project once with API.InProject (resource.ID needs no request) and uses the Gophercloud field names (Network, FloatingIP, ...), with other quota keys such as healthmonitor only in the raw Body.
- Python translates the response into a Quota Resource (descriptor type coercion, project_id alternate ID with tenant_id alias, unknown attributes kept through _allow_unknown_attrs_in_body); Go decodes the native typed limits, keeps every quota field as raw JSON in Body and fixes ProjectID to the requested project.
- Go has no client-side GET response cache; every call sends the request.
- Neutron has no microversion, so neither side sends a version header.

### `python:network/v2/update_quota`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.ProjectQuotaScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.WithQuotaOptions`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.WithUpdateCheckLimit`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.API.InProject`

- Optional limits and check_limit have concrete Go equivalents with exact/retry-stable request bodies and construction-time typed snapshots.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaUpdateOmittedFalseZeroAndSnapshotAcrossReauth`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaOptionsSnapshotAtCreationAndRemainReusable`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaOperationsKeepTargetRawJSONAndMetadata`.
- update_quota(project, networks=10, ports=0, health_monitors=7, check_limit=False) maps to ProjectQuotaScope.Update with typed Network/Port, WithUpdateField("healthmonitor", 7) and WithUpdateCheckLimit(false), sending PUT /quotas/<project> {"quota":{...}} with only those keys and no id.
  Tests: `network/v2/extensions/quotas/python_rejudge_test.go:TestPythonRejudgeQuotaUpdate`.

Python/Go differences:

- Go uses fixed scopes, protected native fields and optional explicit check_limit booleans rather than mutating a dirty Python Quota object.
- Python accepts plural attribute names or wire keys; Go uses typed UpdateOpts fields for the native limits and WithUpdateField with the wire key (healthmonitor, listener, loadbalancer, l7policy, pool, router_route) for the others.
- Python skips the PUT when no attribute is dirty; Go Update with empty options still sends PUT {"quota":{}}.
- Go cannot send JSON null for a typed limit; Neutron rejects null limits.
- Python translates the response into a Quota Resource (descriptor type coercion, project_id alternate ID with tenant_id alias, unknown attributes kept through _allow_unknown_attrs_in_body); Go decodes the native typed limits, keeps every quota field as raw JSON in Body and fixes ProjectID to the requested project.
- Neutron has no microversion, so neither side sends a version header.

### `python:network/v2/delete_quota`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.ProjectQuotaScope.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.WithDeleteIgnoreMissing`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.API.InProject`

- The scope supports explicit missing suppression, fixed-project deletion and native response status/header evidence.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaOperationsKeepTargetRawJSONAndMetadata`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaHTTPFailuresAndStrictMissingPolicy`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaSuccessCodesMatchNativePolicy`.
- delete_quota(project) maps to ProjectQuotaScope.Delete with WithDeleteIgnoreMissing(true): DELETE /quotas/<project>, 404 returns nil, 409 is returned, and the default strict Delete raises resource.ErrNotFound like ignore_missing=False.
  Tests: `network/v2/extensions/quotas/python_rejudge_test.go:TestPythonRejudgeQuotaDelete`.

Python/Go differences:

- Go defaults to strict 404; Python defaults ignore_missing=True, represented by WithDeleteIgnoreMissing(true).
- Go returns deletion metadata while Python returns None.
- Python accepts a Quota Resource or a project ID; Go fixes the project once with API.InProject (resource.ID needs no request).
- Python accepts any non-error status; Go accepts the native 202 and 204.
- Neutron has no microversion, so neither side sends a version header.

### `python:network/v2/get_quota_default`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.ProjectQuotaScope.Defaults`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.API.InProject`

- Defaults uses the distinct /quotas/{project}/default endpoint with ResourceBase/auth/configured headers, fixes ProjectID even when no identity is returned, and retains -1/zero limits, raw large extension integers/nulls and independent header/body snapshots.
  Tests: `api/neutron_quota_defaults_test.go:TestNeutronQuotaDefaultsUsesDistinctEndpointAndKeepsRawResponse`.
- Strict 404 and 403/unexpected-201 failures retain HTTP code/body/header evidence; missing/null/array/scalar quota envelopes and wrong native limit types fail rather than returning a zero-valued resource.
  Tests: `api/neutron_quota_defaults_test.go:TestNeutronQuotaDefaultsPreservesHTTPAndDecodeErrors`.
- Pre-canceled contexts prevent HTTP and in-flight cancellation/deadline causes remain observable.
  Tests: `api/neutron_quota_defaults_test.go:TestNeutronQuotaDefaultsCancellationAndPreflight`.
- get_quota_default(project) maps to ProjectQuotaScope.Defaults, which sends GET /quotas/<project>/default and decodes the default limits with ProjectID fixed to the requested project.
  Tests: `network/v2/extensions/quotas/python_rejudge_test.go:TestPythonRejudgeQuotaDefault`.

Python/Go differences:

- Go uses the already resolved project scope and returns a native typed/raw response with copied metadata instead of constructing and mutating a Python QuotaDefault URI Resource.
- A missing response project identity is not inserted into raw Body; the request target is available separately as ProjectID.
- Python accepts a Quota Resource or a project ID and uses plural attribute names (networks, floating_ips, health_monitors, ...); Go fixes the project once with API.InProject (resource.ID needs no request) and uses the Gophercloud field names (Network, FloatingIP, ...), with other quota keys such as healthmonitor only in the raw Body.
- Python translates the response into a Quota Resource (descriptor type coercion, project_id alternate ID with tenant_id alias, unknown attributes kept through _allow_unknown_attrs_in_body); Go decodes the native typed limits, keeps every quota field as raw JSON in Body and fixes ProjectID to the requested project.
- Go has no client-side GET response cache; every call sends the request.
- Neutron has no microversion, so neither side sends a version header.

### `python:network/v2/quotas`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.API.ListProjects`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.API.AllProjects`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.WithListProjectID`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.WithListMaxItems`

- ListProjects performs no HTTP until consumed, fetches one /quotas array without query, and returns native limits plus validated project_id or legacy tenant_id, raw large/null/unknown fields and independent per-row/call headers and body snapshots.
  Tests: `api/neutron_quotas_list_test.go:TestNeutronQuotasListIsLazySinglePageAndKeepsProjectRawMetadata`.
- Exact project filtering and maximum matching-item consumption are local; last options win, reused options and the iterator option-slice snapshot are independent, break/max stop before decoding later malformed rows, and malformed visited rows are not hidden by a filter.
  Tests: `api/neutron_quotas_list_test.go:TestNeutronQuotasListBreakAndLocalOptionsDoNotSendServerPaging`.
- Missing/null/non-array quota collections, non-object rows, missing/empty/null/non-string/unsafe/conflicting project identities and invalid native limit types fail with retained decoder causes.
  Tests: `api/neutron_quotas_list_test.go:TestNeutronQuotasListRejectsMalformedEnvelopeRowsAndIdentity`.
- Recognized nonempty quotas_links rel=next/href or links.next strings are rejected as unsupported continuation without another request; empty/previous/unknown metadata shapes remain allowed, including unknown neighbors beside a real next link.
  Tests: `api/neutron_quotas_list_test.go:TestNeutronQuotasListSinglePageContinuationPolicyPreservesOtherMetadata`.
- Nil options/clients, invalid local IDs and nonpositive local maxima fail before HTTP; 401/403/404/204/500 retain original response code/header/body evidence.
  Tests: `api/neutron_quotas_list_test.go:TestNeutronQuotasListPreflightOptionsAndHTTPFailures`.
- Cancellation before or between consumption and during HTTP preserves its cause, deadlines remain observable, and AllProjects discards a partial slice after a later row error.
  Tests: `api/neutron_quotas_list_test.go:TestNeutronQuotasListCancellationAndPartialIteration`.
- quotas() maps to AllProjects or ListProjects, which send one GET /quotas without query and return each row with its limits and project_id.
  Tests: `network/v2/extensions/quotas/python_rejudge_test.go:TestPythonRejudgeQuotaList`.

Python/Go differences:

- The actual Neutron quota controller and DB driver return a single array of projects with quota overrides, with default limits merged into each row; this is not a list of every Keystone project.
- Pinned Proxy.quotas documents no supported quota query. Go provides local filter/consumption options and no server paging/filter/projection helpers or generic Resource.list limit/marker fallback.
- Go requires safe unambiguous row identity and independent native/raw metadata; it does not return Python descriptor-backed Quota Resource objects. Recognized continuation is rejected rather than guessed or followed.
- Python Resource.list would follow quotas_links, a Link header or limit/marker continuation if a server returned it; Neutron returns the quota list as one unpaginated array, and Go rejects recognized continuation metadata instead of following it.
- Python quotas(**query) sends limit/marker to the server and filters other Quota Body attributes locally; Go offers only the local WithListProjectID and WithListMaxItems options, and the pinned proxy documents no supported query.
- Go requires a valid project_id or tenant_id in every row; Neutron always returns them.
- Python translates the response into a Quota Resource (descriptor type coercion, project_id alternate ID with tenant_id alias, unknown attributes kept through _allow_unknown_attrs_in_body); Go decodes the native typed limits, keeps every quota field as raw JSON in Body and fixes ProjectID to the requested project.
- Neutron has no microversion, so neither side sends a version header.

### `python:cloud/_network/NetworkCloudMixin/get_network_quotas`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.NetworkProjectQuotas`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.ProjectQuotaScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.ProjectQuotaScope.Detail`

- A project name resolves once through a separate Identity client and is then reused; Get reads GET /quotas/{project} and Detail reads the usage endpoint for the resolved ID.
  Tests: `connection_project_quotas_test.go:TestConnectionServiceQuotaNamesUseSeparateIdentityClient`, `connection_py_cloud_net_quotas_test.go:TestPyCloudNetNetworkQuotasGetDetailAndNamedProjectLookup`, `connection_py_cloud_net_quotas_test.go:TestPyCloudNetNetworkQuotasMissingNamedProjectFailsBeforeNeutron`.
- Limit and detail endpoints preserve fixed project identity and native/raw result fields.
  Tests: `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaOperationsKeepTargetRawJSONAndMetadata`.

Python/Go differences:

- Go callers pass an explicit resource.ID or resource.Name and reuse a fixed scope; Python takes an ambiguous string and repeats the Keystone lookup on every call.
- An explicit project ID is not verified through Keystone (Python always runs find_project and raises when it finds nothing); a project name that does not resolve fails before any Neutron request.
- Results are typed Go quota models with raw Body evidence instead of Quota/QuotaDetails Resource objects; Python **kwargs become concrete UpdateOpts fields.
- Details are read from /quotas/{project}/details.json; Python uses /quotas/{project}/details.

### `python:cloud/_network/NetworkCloudMixin/set_network_quotas`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.NetworkProjectQuotas`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.ProjectQuotaScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.WithUpdateCheckLimit`

- Named project lookup happens once through a separate Identity client; a missing named project fails before any Neutron request.
  Tests: `connection_project_quotas_test.go:TestConnectionServiceQuotaNamesUseSeparateIdentityClient`, `connection_py_cloud_net_quotas_test.go:TestPyCloudNetNetworkQuotasGetDetailAndNamedProjectLookup`, `connection_py_cloud_net_quotas_test.go:TestPyCloudNetNetworkQuotasMissingNamedProjectFailsBeforeNeutron`.
- Update sends PUT /quotas/{project} with a quota envelope that keeps zero, -1 and an explicit check_limit=false and omits unset limits.
  Tests: `connection_py_cloud_net_quotas_test.go:TestPyCloudNetNetworkQuotasUpdateBodyKeepsZeroUnlimitedAndCheckLimit`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaUpdateOmittedFalseZeroAndSnapshotAcrossReauth`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaOptionsSnapshotAtCreationAndRemainReusable`.

Python/Go differences:

- Go callers pass an explicit resource.ID or resource.Name and reuse a fixed scope; Python takes an ambiguous string and repeats the Keystone lookup on every call.
- An explicit project ID is not verified through Keystone (Python always runs find_project and raises when it finds nothing); a project name that does not resolve fails before any Neutron request.
- Results are typed Go quota models with raw Body evidence instead of Quota/QuotaDetails Resource objects; Python **kwargs become concrete UpdateOpts fields.
- An update with no limits sends an empty quota envelope; Python commits only dirty fields and sends no request.

### `python:cloud/_network/NetworkCloudMixin/delete_network_quotas`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk.Connection.NetworkProjectQuotas`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.ProjectQuotaScope.Delete`
- `github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/quotas.WithDeleteIgnoreMissing`

- Named project lookup happens once through a separate Identity client; a missing named project fails before any Neutron request.
  Tests: `connection_project_quotas_test.go:TestConnectionServiceQuotaNamesUseSeparateIdentityClient`, `connection_py_cloud_net_quotas_test.go:TestPyCloudNetNetworkQuotasGetDetailAndNamedProjectLookup`, `connection_py_cloud_net_quotas_test.go:TestPyCloudNetNetworkQuotasMissingNamedProjectFailsBeforeNeutron`.
- Delete sends DELETE /quotas/{project} without a body; strict versus ignore-missing handling is explicit.
  Tests: `connection_py_cloud_net_quotas_test.go:TestPyCloudNetNetworkQuotasDeleteSendsNoBody`, `api/neutron_project_quotas_contracts_test.go:TestNeutronProjectQuotaHTTPFailuresAndStrictMissingPolicy`.

Python/Go differences:

- Go callers pass an explicit resource.ID or resource.Name and reuse a fixed scope; Python takes an ambiguous string and repeats the Keystone lookup on every call.
- An explicit project ID is not verified through Keystone (Python always runs find_project and raises when it finds nothing); a project name that does not resolve fails before any Neutron request.
- Results are typed Go quota models with raw Body evidence instead of Quota/QuotaDetails Resource objects; Python **kwargs become concrete UpdateOpts fields.
- Python reaches the proxy delete_quota with ignore_missing=True and returns None; Go exposes the ignore-missing choice and returns the delete response evidence.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
func manageQuotas(ctx context.Context, networkClient *gophercloud.ServiceClient) error {
    api := quotas.New(networkClient)
    scope, err := api.InProject(ctx, resource.ID("project-id"))
    if err != nil { return err }

    limits, err := scope.Get(ctx)
    if err != nil { return err }
    fmt.Println(scope.ProjectID(), limits.Network, limits.Port)

    defaults, err := scope.Defaults(ctx)
    if err != nil { return err }
    fmt.Println(defaults.Network, defaults.Port)

    usage, err := scope.Detail(ctx)
    if err != nil { return err }
    fmt.Println(usage.Network.Used, usage.Network.Reserved, usage.Network.Limit)

    zero, unlimited := 0, -1
    updated, err := scope.Update(ctx, quotas.UpdateOpts{
        Port: &zero, Network: &unlimited,
    }, quotas.WithUpdateCheckLimit(false))
    if err != nil { return err }
    fmt.Println(updated.Port, updated.Header.Get("X-Openstack-Request-Id"))

    reset, err := scope.Delete(ctx)
    if err != nil { return err }
    fmt.Println(reset.StatusCode, reset.Header.Get("X-Openstack-Request-Id"))
    return nil
}
```

```go
func listQuotaOverrides(ctx context.Context, networkClient *gophercloud.ServiceClient) error {
    api := quotas.New(networkClient)
    for quota, err := range api.ListProjects(ctx, quotas.WithListProjectID("project-id"), quotas.WithListMaxItems(1)) {
        if err != nil { return err }
        fmt.Println(quota.ProjectID, quota.Network, quota.Port)
        fmt.Println(quota.Header.Get("X-Openstack-Request-Id"), quota.StatusCode)
    }
    return nil
}
```

## References

- [Neutron API reference](https://docs.openstack.org/api-ref/network/v2/#quotas-extension-quotas)
- [Gophercloud requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/quotas/requests.go)
- [Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py)
- [Python quota resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/quota.py)
- [Neutron quota details API](https://docs.openstack.org/api-ref/network/v2/#quotas-details-extension-quota-details)
- [Neutron Project Quotas Contracts Test](../../../../api/neutron_project_quotas_contracts_test.go)
- [Neutron Project Quotas Projects Test](../../../../api/neutron_project_quotas_projects_test.go)
- [Neutron Quota Defaults Test](../../../../api/neutron_quota_defaults_test.go)
- [QuotaSetsController](https://github.com/openstack/neutron/blob/cadc448a995a30d9b18514089acd7b1770142032/neutron/extensions/quotasv2.py)
- [DbQuotaDriver](https://github.com/openstack/neutron/blob/cadc448a995a30d9b18514089acd7b1770142032/neutron/db/quota/driver.py)
- [Neutron quota API](https://docs.openstack.org/api-ref/network/v2/#quotas-extension-quotas)
- [Neutron Quotas List Test](../../../../api/neutron_quotas_list_test.go)
