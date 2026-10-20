# Quotas

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:dns/v2/quotas.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas.API.CurrentProject`
- `github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas.ProjectQuotaScope.Get`

- The quota project is fixed once: explicit IDs require no HTTP, exact project names use a separate Keystone client, recorded v2/v3 authentication supplies CurrentProject, and invalid/manual/unscoped references never guess a target.
  Tests: `api/designate_project_quotas_projects_test.go:TestDesignateQuotaProjectReferencesResolveOnceWithSeparateKeystone`, `api/designate_project_quotas_projects_test.go:TestDesignateQuotaProjectResolutionFailuresDoNotCallDNS`, `api/designate_project_quotas_projects_test.go:TestDesignateQuotaCurrentProjectFreezesRecordedV2V3Identity`, `api/designate_project_quotas_projects_test.go:TestDesignateQuotaCurrentProjectRejectsMissingUnscopedAndInvalidAuth`, `connection_project_quotas_test.go:TestConnectionServiceQuotaNamesUseSeparateIdentityClient`.
- GET and PATCH accept only HTTP200 and decode the unwrapped quota object; native integer fields and raw omitted/null/extension values retain independent HTTP headers/status. Invalid accepted JSON retains raw bytes, headers/status and original decode cause.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateProjectQuotasUseRootObjectAndOfficialMethods`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaDecodeErrorsRetainSuccessfulHTTPEvidence`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaSuccessfulCodeContractDoesNotBroaden`.
- Every scoped request sets the fixed sudo-project header, explicit false can override all-projects while omission preserves source configuration, and retries/reauth cannot retarget the method, URL or project. Context owns transport and accepted body reads.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaAllProjectsHeaderIsExplicitAndLastOptionWins`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaRetriesAndReauthKeepFixedProjectAndSnapshottedBody`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaRedirectCannotResetAnotherProjectOrCurrentProject`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaCancellationDoesNotWaitForSharedReauth`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaContextOwnsTransportAndBodyRead`.
- 404 gains ErrNotFound while original status/body/header remains available; unexpected success codes remain errors, and generated native Get/Update methods retain their original contracts.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaErrorsPreserveNativeResponseAndMissingPolicy`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaGeneratedNativeAPIRemainsUnchanged`.

Python/Go differences:

- The Go singleton returns QuotaResource with fixed ProjectID, raw fields, Header and StatusCode rather than a native Result/Python mutable Resource; context and error are explicit.
- The fixed scope preserves the configured service client and sends the target through Designate sudo-project headers; WithAllProjects is explicit and does not enumerate other projects.

### `gophercloud:dns/v2/quotas.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas.ProjectQuotaScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas.WithQuotaOptions`
- `github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas.WithUpdateField`

- PATCH sends an unwrapped partial quota object: nil typed pointers omit fields, zero/large native integer values are exact, WithQuotaOptions snapshots pointers, extensions snapshot JSON, and core identity/capability collisions fail before HTTP.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateProjectQuotasUseRootObjectAndOfficialMethods`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaInvalidOptionsAndZeroScopeFailBeforeHTTP`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaRetriesAndReauthKeepFixedProjectAndSnapshottedBody`.
- GET and PATCH accept only HTTP200 and decode the unwrapped quota object; native integer fields and raw omitted/null/extension values retain independent HTTP headers/status. Invalid accepted JSON retains raw bytes, headers/status and original decode cause.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateProjectQuotasUseRootObjectAndOfficialMethods`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaDecodeErrorsRetainSuccessfulHTTPEvidence`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaSuccessfulCodeContractDoesNotBroaden`.
- Every scoped request sets the fixed sudo-project header, explicit false can override all-projects while omission preserves source configuration, and retries/reauth cannot retarget the method, URL or project. Context owns transport and accepted body reads.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaAllProjectsHeaderIsExplicitAndLastOptionWins`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaRetriesAndReauthKeepFixedProjectAndSnapshottedBody`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaRedirectCannotResetAnotherProjectOrCurrentProject`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaCancellationDoesNotWaitForSharedReauth`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaContextOwnsTransportAndBodyRead`.
- 404 gains ErrNotFound while original status/body/header remains available; unexpected success codes remain errors, and generated native Get/Update methods retain their original contracts.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaErrorsPreserveNativeResponseAndMissingPolicy`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaGeneratedNativeAPIRemainsUnchanged`.

Python/Go differences:

- The Go singleton returns QuotaResource with fixed ProjectID, raw fields, Header and StatusCode rather than a native Result/Python mutable Resource; context and error are explicit.
- The fixed scope preserves the configured service client and sends the target through Designate sudo-project headers; WithAllProjects is explicit and does not enumerate other projects.
- Concrete native UpdateOpts and library-owned functional options replace caller-implemented builders; known limit validation remains the service schema contract.

### `python:dns/v2/get_quota`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas.ProjectQuotaScope.Get`

- The quota project is fixed once: explicit IDs require no HTTP, exact project names use a separate Keystone client, recorded v2/v3 authentication supplies CurrentProject, and invalid/manual/unscoped references never guess a target.
  Tests: `api/designate_project_quotas_projects_test.go:TestDesignateQuotaProjectReferencesResolveOnceWithSeparateKeystone`, `api/designate_project_quotas_projects_test.go:TestDesignateQuotaProjectResolutionFailuresDoNotCallDNS`, `api/designate_project_quotas_projects_test.go:TestDesignateQuotaCurrentProjectFreezesRecordedV2V3Identity`, `api/designate_project_quotas_projects_test.go:TestDesignateQuotaCurrentProjectRejectsMissingUnscopedAndInvalidAuth`, `connection_project_quotas_test.go:TestConnectionServiceQuotaNamesUseSeparateIdentityClient`.
- GET and PATCH accept only HTTP200 and decode the unwrapped quota object; native integer fields and raw omitted/null/extension values retain independent HTTP headers/status. Invalid accepted JSON retains raw bytes, headers/status and original decode cause.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateProjectQuotasUseRootObjectAndOfficialMethods`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaDecodeErrorsRetainSuccessfulHTTPEvidence`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaSuccessfulCodeContractDoesNotBroaden`.
- Every scoped request sets the fixed sudo-project header, explicit false can override all-projects while omission preserves source configuration, and retries/reauth cannot retarget the method, URL or project. Context owns transport and accepted body reads.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaAllProjectsHeaderIsExplicitAndLastOptionWins`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaRetriesAndReauthKeepFixedProjectAndSnapshottedBody`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaRedirectCannotResetAnotherProjectOrCurrentProject`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaCancellationDoesNotWaitForSharedReauth`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaContextOwnsTransportAndBodyRead`.
- 404 gains ErrNotFound while original status/body/header remains available; unexpected success codes remain errors, and generated native Get/Update methods retain their original contracts.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaErrorsPreserveNativeResponseAndMissingPolicy`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaGeneratedNativeAPIRemainsUnchanged`.

Python/Go differences:

- The Go singleton returns QuotaResource with fixed ProjectID, raw fields, Header and StatusCode rather than a native Result/Python mutable Resource; context and error are explicit.
- The fixed scope preserves the configured service client and sends the target through Designate sudo-project headers; WithAllProjects is explicit and does not enumerate other projects.

Unresolved boundaries:

- Compare inherited DNS Resource/proxy inputs, cached Resource mutation/dirty-state behavior and adapter/microversion selection beyond the verified fixed-project HTTP mapping.

### `python:dns/v2/update_quota`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas.ProjectQuotaScope.Update`

- The quota project is fixed once: explicit IDs require no HTTP, exact project names use a separate Keystone client, recorded v2/v3 authentication supplies CurrentProject, and invalid/manual/unscoped references never guess a target.
  Tests: `api/designate_project_quotas_projects_test.go:TestDesignateQuotaProjectReferencesResolveOnceWithSeparateKeystone`, `api/designate_project_quotas_projects_test.go:TestDesignateQuotaProjectResolutionFailuresDoNotCallDNS`, `api/designate_project_quotas_projects_test.go:TestDesignateQuotaCurrentProjectFreezesRecordedV2V3Identity`, `api/designate_project_quotas_projects_test.go:TestDesignateQuotaCurrentProjectRejectsMissingUnscopedAndInvalidAuth`, `connection_project_quotas_test.go:TestConnectionServiceQuotaNamesUseSeparateIdentityClient`.
- PATCH sends an unwrapped partial quota object: nil typed pointers omit fields, zero/large native integer values are exact, WithQuotaOptions snapshots pointers, extensions snapshot JSON, and core identity/capability collisions fail before HTTP.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateProjectQuotasUseRootObjectAndOfficialMethods`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaInvalidOptionsAndZeroScopeFailBeforeHTTP`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaRetriesAndReauthKeepFixedProjectAndSnapshottedBody`.
- GET and PATCH accept only HTTP200 and decode the unwrapped quota object; native integer fields and raw omitted/null/extension values retain independent HTTP headers/status. Invalid accepted JSON retains raw bytes, headers/status and original decode cause.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateProjectQuotasUseRootObjectAndOfficialMethods`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaDecodeErrorsRetainSuccessfulHTTPEvidence`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaSuccessfulCodeContractDoesNotBroaden`.
- Every scoped request sets the fixed sudo-project header, explicit false can override all-projects while omission preserves source configuration, and retries/reauth cannot retarget the method, URL or project. Context owns transport and accepted body reads.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaAllProjectsHeaderIsExplicitAndLastOptionWins`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaRetriesAndReauthKeepFixedProjectAndSnapshottedBody`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaRedirectCannotResetAnotherProjectOrCurrentProject`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaCancellationDoesNotWaitForSharedReauth`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaContextOwnsTransportAndBodyRead`.
- 404 gains ErrNotFound while original status/body/header remains available; unexpected success codes remain errors, and generated native Get/Update methods retain their original contracts.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaErrorsPreserveNativeResponseAndMissingPolicy`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaGeneratedNativeAPIRemainsUnchanged`.

Python/Go differences:

- The Go singleton returns QuotaResource with fixed ProjectID, raw fields, Header and StatusCode rather than a native Result/Python mutable Resource; context and error are explicit.
- The fixed scope preserves the configured service client and sends the target through Designate sudo-project headers; WithAllProjects is explicit and does not enumerate other projects.

Unresolved boundaries:

- Compare inherited DNS Resource/proxy inputs, cached Resource mutation/dirty-state behavior and adapter/microversion selection beyond the verified fixed-project HTTP mapping.

### `python:dns/v2/delete_quota`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas.ProjectQuotaScope.Reset`

- The quota project is fixed once: explicit IDs require no HTTP, exact project names use a separate Keystone client, recorded v2/v3 authentication supplies CurrentProject, and invalid/manual/unscoped references never guess a target.
  Tests: `api/designate_project_quotas_projects_test.go:TestDesignateQuotaProjectReferencesResolveOnceWithSeparateKeystone`, `api/designate_project_quotas_projects_test.go:TestDesignateQuotaProjectResolutionFailuresDoNotCallDNS`, `api/designate_project_quotas_projects_test.go:TestDesignateQuotaCurrentProjectFreezesRecordedV2V3Identity`, `api/designate_project_quotas_projects_test.go:TestDesignateQuotaCurrentProjectRejectsMissingUnscopedAndInvalidAuth`, `connection_project_quotas_test.go:TestConnectionServiceQuotaNamesUseSeparateIdentityClient`.
- Reset uses DELETE204 without a follow-up read; strict missing behavior is the Go default and WithResetIgnoreMissing changes only404. Reset responses carry the fixed project and independent HTTP metadata.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateProjectQuotasUseRootObjectAndOfficialMethods`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaErrorsPreserveNativeResponseAndMissingPolicy`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaRedirectCannotResetAnotherProjectOrCurrentProject`.
- Every scoped request sets the fixed sudo-project header, explicit false can override all-projects while omission preserves source configuration, and retries/reauth cannot retarget the method, URL or project. Context owns transport and accepted body reads.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaAllProjectsHeaderIsExplicitAndLastOptionWins`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaRetriesAndReauthKeepFixedProjectAndSnapshottedBody`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaRedirectCannotResetAnotherProjectOrCurrentProject`, `api/designate_project_quotas_transport_test.go:TestDesignateQuotaCancellationDoesNotWaitForSharedReauth`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaContextOwnsTransportAndBodyRead`.
- 404 gains ErrNotFound while original status/body/header remains available; unexpected success codes remain errors, and generated native Get/Update methods retain their original contracts.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaErrorsPreserveNativeResponseAndMissingPolicy`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaGeneratedNativeAPIRemainsUnchanged`.

Python/Go differences:

- The Go singleton returns QuotaResource with fixed ProjectID, raw fields, Header and StatusCode rather than a native Result/Python mutable Resource; context and error are explicit.
- The fixed scope preserves the configured service client and sends the target through Designate sudo-project headers; WithAllProjects is explicit and does not enumerate other projects.
- Python delete_quota defaults ignore_missing=True; Go Reset is strict unless WithResetIgnoreMissing(true) is supplied.

Unresolved boundaries:

- Compare inherited DNS Resource/proxy inputs, cached Resource mutation/dirty-state behavior and adapter/microversion selection beyond the verified fixed-project HTTP mapping.

### `python:dns/v2/quotas`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas.API.CurrentProject`
- `github.com/JSYoo5B/go-openstacksdk/dns/v2/quotas.ProjectQuotaScope.Get`

- The quota project is fixed once: explicit IDs require no HTTP, exact project names use a separate Keystone client, recorded v2/v3 authentication supplies CurrentProject, and invalid/manual/unscoped references never guess a target.
  Tests: `api/designate_project_quotas_projects_test.go:TestDesignateQuotaProjectReferencesResolveOnceWithSeparateKeystone`, `api/designate_project_quotas_projects_test.go:TestDesignateQuotaProjectResolutionFailuresDoNotCallDNS`, `api/designate_project_quotas_projects_test.go:TestDesignateQuotaCurrentProjectFreezesRecordedV2V3Identity`, `api/designate_project_quotas_projects_test.go:TestDesignateQuotaCurrentProjectRejectsMissingUnscopedAndInvalidAuth`, `connection_project_quotas_test.go:TestConnectionServiceQuotaNamesUseSeparateIdentityClient`.
- GET and PATCH accept only HTTP200 and decode the unwrapped quota object; native integer fields and raw omitted/null/extension values retain independent HTTP headers/status. Invalid accepted JSON retains raw bytes, headers/status and original decode cause.
  Tests: `api/designate_project_quotas_contracts_test.go:TestDesignateProjectQuotasUseRootObjectAndOfficialMethods`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaDecodeErrorsRetainSuccessfulHTTPEvidence`, `api/designate_project_quotas_contracts_test.go:TestDesignateQuotaSuccessfulCodeContractDoesNotBroaden`.

Python/Go differences:

- The documented GET /quotas/ is a current-project quota object; the Go scope does not fabricate a multi-project Collection from Python allow_list.

Unresolved boundaries:

- Reconcile the pinned Python generator/list Resource contract with the documented singleton collection-path response before implementing any list mapping; compare inherited query/pagination and adapter behavior.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := quotas.New(dnsClient)
scope, err := api.InProject(ctx, resource.ID("project-id"),
    quotas.WithAllProjects(true))
if err != nil { return err }

limits, err := scope.Get(ctx)
if err != nil { return err }
fmt.Println(limits.ProjectID, limits.Zones, limits.StatusCode)

zero := 0
updated, err := scope.Update(ctx, quotas.UpdateOpts{Zones: &zero})
if err != nil { return err }
fmt.Println(updated.Header.Get("X-Openstack-Request-Id"))

reset, err := scope.Reset(ctx)
if err != nil { return err }
fmt.Println(reset.ProjectID, reset.StatusCode)
```

```go
zones := 7
snapshot := quotas.WithQuotaOptions(quotas.UpdateOpts{Zones: &zones})
zones = 99
_, err := scope.Update(ctx, quotas.UpdateOpts{}, snapshot) // zones=7
```

## References

- [Designate quota API](https://docs.openstack.org/api-ref/dns/dns-api-v2-index.html#quotas)
- [DNS Quota resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/dns/v2/quota.py)
- [DNS proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/dns/v2/_proxy.py)
- [Gophercloud native API](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/dns/v2/quotas/requests.go)
- [Quotas](https://docs.openstack.org/designate/latest/admin/quotas.html#default-quotas)
- [Designate Project Quotas Contracts Test](../../../api/designate_project_quotas_contracts_test.go)
- [Designate Project Quotas Projects Test](../../../api/designate_project_quotas_projects_test.go)
- [Designate Project Quotas Transport Test](../../../api/designate_project_quotas_transport_test.go)
