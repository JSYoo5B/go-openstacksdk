# Limits

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `gophercloud:compute/v2/limits.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/limits.API.Fetch`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/limits.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/limits.API.CurrentProject`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/limits.ProjectLimitsScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/limits.WithGetReserved`

- Implicit Fetch sends no tenant query and reports no guessed project; exact names bind once through a separate Keystone client, IDs need no lookup, and CurrentProject fixes recorded v2/v3 authentication independently of later auth changes.
  Tests: `api/nova_project_limits_projects_test.go:TestNovaLimitsProjectIDsAreLazyAndExactNamesUseSeparateKeystoneOnce`, `api/nova_project_limits_projects_test.go:TestNovaLimitsProjectLookupFailuresNeverCallCompute`, `api/nova_project_limits_projects_test.go:TestNovaLimitsCurrentProjectFreezesRecordedKeystoneV2AndV3Scope`, `api/nova_project_limits_projects_test.go:TestNovaLimitsCurrentProjectRejectsMissingUnscopedAndMalformedAuth`, `connectiontest/connection_limits_test.go:TestConnectionProjectLimitsSeparateIdentityAndFixQueries`.
- GET200 preserves native absolute integers plus the missing legacy rate groups/rules, exact raw next-available timestamps, nested extension/null/omission and independent body/header/status; strict invalid limits/rate JSON retains accepted HTTP bytes and original cause.
  Tests: `api/nova_project_limits_contracts_test.go:TestNovaLimitsFetchAndProjectGetPreserveLegacyRateAndRawResponse`, `api/nova_project_limits_contracts_test.go:TestNovaLimitsFetchExplicitTenantAndLargeTypedIntegers`, `api/nova_project_limits_contracts_test.go:TestNovaLimitsNullEmptyAndOmittedFieldsRemainDistinct`, `api/nova_project_limits_contracts_test.go:TestNovaLimitsDecodeFailuresRetainAcceptedHTTPBodyHeaderAndStatus`.
- WithGetReserved maps bool to integer0/1 for both native and SDK Get; raw query values retain Nova integer/noninteger server semantics. Fixed scopes reject tenant_id/project_id overrides and duplicates before HTTP.
  Tests: `api/nova_project_limits_contracts_test.go:TestNovaLimitsReservedHelperUsesIntegersAndRawQueryKeepsNativeValues`, `api/nova_project_limits_contracts_test.go:TestNovaLimitsFixedScopeRejectsIdentityOverridesBeforeHTTP`.
- Tenant query/method/path/origin remain fixed across redirects/retry/reauth with source transport/auth preserved; native status/body/header, caller retry/transport errors and context during header/body/shared auth are retained without fallback.
  Tests: `api/nova_project_limits_transport_test.go:TestNovaLimitsRetriesAndReauthKeepSingleTenantQueryAndSourceTransport`, `api/nova_project_limits_transport_test.go:TestNovaLimitsRedirectCannotDropProjectChangeOriginOrMethod`, `api/nova_project_limits_transport_test.go:TestNovaLimitsRetryAndTransportFailuresPreserveSourceErrorsWithoutFallback`, `api/nova_project_limits_transport_test.go:TestNovaLimitsCanceledCallerDoesNotWaitForSharedReauth`, `api/nova_project_limits_contracts_test.go:TestNovaLimitsHTTPFailuresRetainOriginalEvidenceAndOperation`, `api/nova_project_limits_contracts_test.go:TestNovaLimitsContextOwnsHeaderWaitAndPartialBodyRead`.

Python/Go differences:

- Native generated Get remains unchanged. SDK Fetch/scope return LimitsResource with native absolute data, SDK-owned legacy Rate, raw fields and HTTP metadata; Go context/error are explicit.
- Fetch without tenant query retains server current-project behavior and leaves ProjectID empty; a scope explicitly fixes tenant_id even for recorded current authentication and does not provide list/change/wait methods.
- Current Nova v2.1 reports empty rate data. SDK rate models only preserve historical/returned data; they do not enforce limits or restore the removed legacy Nova v2 API.

### `python:compute/v2/get_limits`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/limits.API.Fetch`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/limits.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/limits.API.CurrentProject`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/limits.ProjectLimitsScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/limits.WithGetReserved`

- Implicit Fetch sends no tenant query and reports no guessed project; exact names bind once through a separate Keystone client, IDs need no lookup, and CurrentProject fixes recorded v2/v3 authentication independently of later auth changes.
  Tests: `api/nova_project_limits_projects_test.go:TestNovaLimitsProjectIDsAreLazyAndExactNamesUseSeparateKeystoneOnce`, `api/nova_project_limits_projects_test.go:TestNovaLimitsProjectLookupFailuresNeverCallCompute`, `api/nova_project_limits_projects_test.go:TestNovaLimitsCurrentProjectFreezesRecordedKeystoneV2AndV3Scope`, `api/nova_project_limits_projects_test.go:TestNovaLimitsCurrentProjectRejectsMissingUnscopedAndMalformedAuth`, `connectiontest/connection_limits_test.go:TestConnectionProjectLimitsSeparateIdentityAndFixQueries`.
- GET200 preserves native absolute integers plus the missing legacy rate groups/rules, exact raw next-available timestamps, nested extension/null/omission and independent body/header/status; strict invalid limits/rate JSON retains accepted HTTP bytes and original cause.
  Tests: `api/nova_project_limits_contracts_test.go:TestNovaLimitsFetchAndProjectGetPreserveLegacyRateAndRawResponse`, `api/nova_project_limits_contracts_test.go:TestNovaLimitsFetchExplicitTenantAndLargeTypedIntegers`, `api/nova_project_limits_contracts_test.go:TestNovaLimitsNullEmptyAndOmittedFieldsRemainDistinct`, `api/nova_project_limits_contracts_test.go:TestNovaLimitsDecodeFailuresRetainAcceptedHTTPBodyHeaderAndStatus`.
- WithGetReserved maps bool to integer0/1 for both native and SDK Get; raw query values retain Nova integer/noninteger server semantics. Fixed scopes reject tenant_id/project_id overrides and duplicates before HTTP.
  Tests: `api/nova_project_limits_contracts_test.go:TestNovaLimitsReservedHelperUsesIntegersAndRawQueryKeepsNativeValues`, `api/nova_project_limits_contracts_test.go:TestNovaLimitsFixedScopeRejectsIdentityOverridesBeforeHTTP`.
- Tenant query/method/path/origin remain fixed across redirects/retry/reauth with source transport/auth preserved; native status/body/header, caller retry/transport errors and context during header/body/shared auth are retained without fallback.
  Tests: `api/nova_project_limits_transport_test.go:TestNovaLimitsRetriesAndReauthKeepSingleTenantQueryAndSourceTransport`, `api/nova_project_limits_transport_test.go:TestNovaLimitsRedirectCannotDropProjectChangeOriginOrMethod`, `api/nova_project_limits_transport_test.go:TestNovaLimitsRetryAndTransportFailuresPreserveSourceErrorsWithoutFallback`, `api/nova_project_limits_transport_test.go:TestNovaLimitsCanceledCallerDoesNotWaitForSharedReauth`, `api/nova_project_limits_contracts_test.go:TestNovaLimitsHTTPFailuresRetainOriginalEvidenceAndOperation`, `api/nova_project_limits_contracts_test.go:TestNovaLimitsContextOwnsHeaderWaitAndPartialBodyRead`.

Python/Go differences:

- Native generated Get remains unchanged. SDK Fetch/scope return LimitsResource with native absolute data, SDK-owned legacy Rate, raw fields and HTTP metadata; Go context/error are explicit. Native typed integers use platform int and represent null/missing as zero while raw fields preserve their presence.
- Fetch without tenant query retains server current-project behavior and leaves ProjectID empty; a scope explicitly fixes tenant_id even for recorded current authentication and does not provide list/change/wait methods. Named Resource/project convenience inputs map to explicit resource.Ref/ID and library-owned optional name resolution.
- Current Nova v2.1 reports empty rate data. SDK rate models only preserve historical/returned data; they do not enforce limits or restore the removed legacy Nova v2 API.
- Pinned Limits declares a project_id-to-tenant_id QueryParameters alias, but get_limits calls Limits.fetch and Resource.fetch forwards **params directly to session.get without transposing that mapping. A Python project_id keyword therefore remains project_id on the wire. Go deliberately rejects this unsupported wire key and uses InProject, GetOpts.TenantID or WithGetQuery("tenant_id", value). Other Go query extensions use explicit string values rather than Python dynamic kwargs coercion.
- Python returns a fresh mutable Limits Resource and supplies Resource/session version discovery and fetch controls such as skip_cache, base_path, resource_response_key or explicit microversion. Go reads a fixed limits singleton using the selected source client, owned raw/typed output and caller context; it does not emulate those Resource/session controls or mutable cache state. AbsoluteLimits ceiling2.57 is not the outer Limits minimum version.
- Python Resource translation accepts statuses below400 and may retain Resource defaults after non-parseable JSON. Go requires actual200 and strict known limits/absolute/rate JSON, retaining accepted raw body/header/status and the original decode/read/context cause on failure.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
api := limits.New(computeClient)
current, err := api.Fetch(ctx, limits.WithGetReserved(true))
if err != nil { return err }
fmt.Println(current.Absolute.MaxTotalCores, current.StatusCode)

scope, err := api.InProject(ctx, resource.ID("project-id"))
if err != nil { return err }
value, err := scope.Get(ctx, limits.WithGetReserved(false))
if err != nil { return err }
fmt.Println(value.ProjectID, value.Absolute.TotalCoresUsed)
```

```go
for _, group := range value.Rate {
    for _, rule := range group.Limits {
        fmt.Println(group.URI, rule.Verb, rule.Remaining,
            string(rule.NextAvailable))
    }
}
```

## References

- [Nova limits API](https://docs.openstack.org/api-ref/compute/#limits-limits)
- [Python limits resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/limits.py)
- [Gophercloud v2.15.0](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/limits/results.go)
- [get_limits proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)
- [Resource.fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
- [Stable API](https://docs.openstack.org/nova/15.0.4/stable_api.html#background)
- [Compute](https://docs.openstack.org/api-ref/compute/#show-rate-and-absolute-limits)
- [Nova Project Limits Contracts Test](../../../api/nova_project_limits_contracts_test.go)
- [project binding·recorded auth](../../../api/nova_project_limits_projects_test.go)
- [redirect·retry·reauth·context](../../../api/nova_project_limits_transport_test.go)
