# Quotas

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:load_balancer/v2/get_quota`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.API.CurrentProject`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.ProjectQuotaScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.WithGetFields`
- `github.com/JSYoo5B/go-openstacksdk.Connection.LoadBalancerProjectQuotas`
- `github.com/JSYoo5B/go-openstacksdk.Connection.CurrentLoadBalancerProjectQuotas`

- Scope GET uses the official /lbaas/quotas/{project} route, respects ResourceBase already ending in lbaas and preserves the generated native API.Get /quotas route without mutating the client.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaScopesCorrectNativeURLWithoutChangingGeneratedAPI`, `api/octavia_project_quotas_contracts_test.go:TestOctaviaProjectQuotasUseOfficialPathsAndGlobalDefaults`.
- Repeated fields inputs are snapshotted; the scope project remains separate from response project_id and native/raw limits retain null/large extensions and independent response metadata. Non-null canonical aliases win, null canonical aliases fall back to non-null legacy values, and raw fields retain all aliases.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaProjectQuotasUseOfficialPathsAndGlobalDefaults`, `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaAliasesKeepNativeNullFallbackAndRawFields`.
- Exact project names resolve once with a separate Keystone client across pages; recorded v2/v3 authentication freezes the current project, explicit IDs avoid Identity/catalog lookup and canceled Connection scope construction preserves cancellation.
  Tests: `api/octavia_project_quotas_projects_test.go:TestOctaviaQuotaProjectNameUsesSeparateIdentityClientOnce`, `api/octavia_project_quotas_projects_test.go:TestOctaviaQuotaCurrentProjectRecordsAndFreezesV2V3Authentication`, `api/octavia_project_quotas_projects_test.go:TestOctaviaQuotaScopePreflightAndLookupFailures`, `connectiontest/connection_project_quotas_test.go:TestConnectionServiceQuotaNamesUseSeparateIdentityClient`, `connectiontest/connection_project_quotas_test.go:TestConnectionServiceQuotasIDAuthAndCancellation`.
- GET accepts only 200 and retains rejected HTTP code/header and 404 classification; nil/invalid fields options fail before HTTP, pre-canceled requests stop and an in-flight GET deadline preserves context.DeadlineExceeded.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaSuccessCodesAndStrictMissingPolicy`, `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaInvalidOptionsFailBeforeHTTP`, `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaCancellationAndDeadlinePreserveCause`.

Python/Go differences:

- The high-level Go route corrects pinned Gophercloud quota URLs that omit lbaas; generated low-level operations retain their existing route.
- Go references distinguish exact Keystone names from IDs and can derive a recorded-auth scope, while the direct Python proxy accepts a project ID or Quota Resource.
- Go returns native integer limits plus raw fields/copied header/status rather than Python plural descriptor properties; null and alias conflicts remain distinguishable through Body.
- Python Quota.project_id is the alternate ID, so a project ID string or Quota selects /lbaas/quotas/{project}; Go uses an explicit project reference (ID or exact Keystone name) or the recorded authentication scope.
- Python exposes plural descriptors (load_balancers from load_balancer, health_monitors from health_monitor, listeners, pools, members) and coerces them to int; Go returns the native integer limits plus raw Body, where null and alias conflicts stay distinguishable.
- Python Resource.fetch/list normalization (dict/location merge, response cache, skip_cache) has no Go counterpart; Go performs fresh stateless requests and keeps the native model separate from HTTP evidence.

### `python:load_balancer/v2/update_quota`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.ProjectQuotaScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.WithQuotaOptions`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.WithDefaultLimit`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.WithUpdateField`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.WithLegacyQuotaKeys`

- PUT binds the official /lbaas/quotas/{project} route and sends omitted/nil, zero and -1 native limits distinctly; WithDefaultLimit sends explicit null inheritance and extension values preserve creation-time JSON snapshots.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaProjectQuotasUseOfficialPathsAndGlobalDefaults`, `api/octavia_project_quotas_list_test.go:TestOctaviaQuotaDefaultInheritanceDuplicateAndForgedArgument`.
- By default the body uses the documented loadbalancer/healthmonitor keys; WithLegacyQuotaKeys(true) renames only those two keys (including explicit null inheritance) to Python's load_balancer/health_monitor, other limits such as listener and l7policy are unchanged, and the last option wins.
  Tests: `api/py_lb_quota_spelling_test.go:TestOctaviaQuotaLegacyKeysOptInSpelling`.
- Quota options snapshot pointer values at creation and clone them when reused; exact integers and a stable final JSON body survive reauthentication and mutation of a previously applied private pointer.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaSnapshotOmissionExactIntegersAndRetry`.
- Invalid negative limits, nil options, core/legacy alias extensions (also with WithLegacyQuotaKeys), identity collisions, unserializable extensions, unsupported query/header/arguments, forged spelling arguments and conflicting value/default inputs fail before HTTP.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaInvalidOptionsFailBeforeHTTP`, `api/octavia_project_quotas_list_test.go:TestOctaviaQuotaDefaultInheritanceDuplicateAndForgedArgument`, `api/py_lb_quota_spelling_test.go:TestOctaviaQuotaLegacyKeysRejectAliasesAndForgedArgumentBeforeHTTP`.
- PUT accepts 200/202 and preserves rejected HTTP code/header/404 classification; malformed accepted quota bodies retain exact bytes, status, header and decode cause without invoking the retry hook or replaying the accepted mutation. Pre-canceled Update makes no request and cancellation during Update preserves context.Canceled.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaSuccessCodesAndStrictMissingPolicy`, `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaMalformedAcceptedResponsesRetainEvidenceWithoutRetry`, `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaCancellationAndDeadlinePreserveCause`.

Python/Go differences:

- Go sends the Octavia-documented body keys loadbalancer and healthmonitor by default; WithLegacyQuotaKeys(true) sends Python's load_balancer and health_monitor. Octavia's QuotaBase request type (octavia/api/v2/types/quotas.py, reviewed at upstream octavia 12.0.0.0rc1-74-g12d8e0de) declares both spellings, marks the underscore names deprecated since Rocky and stores either in the same quota column (the canonical key wins when both are sent). Acceptance is source-reviewed and fixture-pinned, not verified against a deployment.
- Pinned Python Quota declares no l7policy/l7rule attributes and does not allow unknown body attributes, so update_quota silently drops L7 limits and the canonical loadbalancer/healthmonitor names before sending. Go sends L7 limits as typed fields and rejects alias extensions before HTTP instead of dropping them.
- Python commit sends only dirty attributes and makes no request when nothing changed; Go always issues exactly one PUT with the supplied limits (an empty quota object when none are set) and never replays it. Python Quota._prepare_request strips id from the body; Go takes the project only from the scope and rejects id/project_id extensions.
- Go performs an explicit fixed-project update and returns a typed/raw snapshot with copied header/status, rather than changing and committing a mutable Python Resource; Resource input/descriptor coercion, cache and adapter/microversion controls are the shared SDK-R1/C1/S1 boundary (docs/sdk-support-ledger.md).
- Go preserves native-compatible 200/202 success codes; the official Octavia PUT contract documents 202.

### `python:load_balancer/v2/delete_quota`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.ProjectQuotaScope.Reset`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.WithResetIgnoreMissing`

- Reset DELETE targets /lbaas/quotas/{project} with no body, retains the fixed project and 204 metadata, and issues no follow-up GET to infer defaults.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaProjectQuotasUseOfficialPathsAndGlobalDefaults`.
- Native-compatible 202/204 are accepted; default missing policy is strict, explicit true suppresses only 404, later false restores strict behavior, and 403/500 and unexpected success codes retain response code/header causes.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaSuccessCodesAndStrictMissingPolicy`.
- Nil Reset options fail before HTTP and pre-canceled Reset contexts make no request.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaInvalidOptionsFailBeforeHTTP`, `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaCancellationAndDeadlinePreserveCause`.

Python/Go differences:

- Python delete_quota defaults ignore_missing=True; Go Reset defaults strict and requires WithResetIgnoreMissing(true) for suppression.
- Go returns ResetResponse metadata and keeps native-compatible 202/204 success; Python returns None and official Octavia DELETE documents 204.
- The scope resets quota overrides through the corrected official route, while generated API.Delete retains its native /quotas path.
- Python Quota.project_id is the alternate ID, so a project ID string or Quota selects /lbaas/quotas/{project}; Go uses an explicit project reference (ID or exact Keystone name) or the recorded authentication scope.
- Python exposes plural descriptors (load_balancers from load_balancer, health_monitors from health_monitor, listeners, pools, members) and coerces them to int; Go returns the native integer limits plus raw Body, where null and alias conflicts stay distinguishable.
- Python Resource.fetch/list normalization (dict/location merge, response cache, skip_cache) has no Go counterpart; Go performs fresh stateless requests and keeps the native model separate from HTTP evidence.

### `python:load_balancer/v2/get_quota_default`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.API.Defaults`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.ProjectQuotaScope.Defaults`

- Defaults reads global /lbaas/quotas/defaults with no project/query, returns raw large/null defaults and response headers independently of project override GET, and does not invent a project identity.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaProjectQuotasUseOfficialPathsAndGlobalDefaults`.
- Defaults accepts only HTTP 200, preserves rejected response code/header and strict 404 classification, and stops pre-canceled calls before HTTP.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaSuccessCodesAndStrictMissingPolicy`, `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaCancellationAndDeadlinePreserveCause`.

Python/Go differences:

- Both API.Defaults and scope.Defaults return deployment-wide DefaultQuotaResource; a scoped call does not make the defaults project-specific.
- Go returns native typed/raw limits and copied metadata rather than mutating a Python QuotaDefault Resource; native null integer values require Body to distinguish inheritance from zero.
- Python Quota.project_id is the alternate ID, so a project ID string or Quota selects /lbaas/quotas/{project}; Go uses an explicit project reference (ID or exact Keystone name) or the recorded authentication scope.
- Python exposes plural descriptors (load_balancers from load_balancer, health_monitors from health_monitor, listeners, pools, members) and coerces them to int; Go returns the native integer limits plus raw Body, where null and alias conflicts stay distinguishable.
- Python Resource.fetch/list normalization (dict/location merge, response cache, skip_cache) has no Go counterpart; Go performs fresh stateless requests and keeps the native model separate from HTTP evidence.

### `python:load_balancer/v2/quotas`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.API.ListProjects`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.API.AllProjects`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.WithProjectListOptions`

- ListProjects is lazy and copies its option slice; reusable typed options independently snapshot fields and the reverse pointer. The collection serializes project_id, repeated fields, positive limit, marker and explicit server-form True/False page_reverse, and preserves missing initial filters/projections while explicit next-link paging values override them.
  Tests: `api/octavia_project_quotas_list_test.go:TestOctaviaQuotaListProjectsQuerySnapshotPaginationAndRawValues`, `api/octavia_project_quotas_list_test.go:TestOctaviaQuotaListReverseUsesServerBooleanAndCarriesQuery`.
- Relative quotas_links next URLs and empty intermediate pages are followed; rows retain native/raw large/null fields and independent metadata, and a projected-away project_id stays empty rather than being copied from a filter or auth record.
  Tests: `api/octavia_project_quotas_list_test.go:TestOctaviaQuotaListProjectsQuerySnapshotPaginationAndRawValues`.
- Breaking consumption skips later row/link decoding and extra HTTP; cancellation before and between rows stops consumption/continuation, and cycles, foreign origins, URL credentials and conflicting next links are rejected without refetching.
  Tests: `api/octavia_project_quotas_list_test.go:TestOctaviaQuotaListBreakSkipsLaterDecodeAndNextHTTP`, `api/octavia_project_quotas_list_test.go:TestOctaviaQuotaListCancellationStopsAdditionalHTTP`, `api/octavia_project_quotas_list_test.go:TestOctaviaQuotaListCycleOriginAndConflictingLinks`.
- Malformed quota envelopes/items/native fields/project identities/link shapes retain exact accepted response bytes/status/header without decode-triggered retry. Empty arrays are valid, only HTTP 200 succeeds, and rejected statuses retain HTTP code/header evidence.
  Tests: `api/octavia_project_quotas_list_test.go:TestOctaviaQuotaListMalformedResponsesRetainEvidence`, `api/octavia_project_quotas_list_test.go:TestOctaviaQuotaListStatusPolicyAndEmptyCollection`.
- Nil options, negative limit, unsafe marker/project IDs, blank fields and generic query/header/body/argument extensions fail before HTTP.
  Tests: `api/octavia_project_quotas_list_test.go:TestOctaviaQuotaListInvalidOptionsFailBeforeHTTP`.

Python/Go differences:

- Go exposes an explicit server-backed ProjectListOpts whitelist although pinned Proxy.quotas documents no supported resource-specific query; generic arbitrary query inputs are rejected.
- Go follows recognized same-origin quotas_links continuations with cycle checks and retained initial query, instead of implementing all inherited Resource.list pagination fallbacks.
- Go returns typed/raw project-quota snapshots; projected-away identities remain empty and are not constructed from request filters.
- Python Quota.project_id is the alternate ID, so a project ID string or Quota selects /lbaas/quotas/{project}; Go uses an explicit project reference (ID or exact Keystone name) or the recorded authentication scope.
- Python exposes plural descriptors (load_balancers from load_balancer, health_monitors from health_monitor, listeners, pools, members) and coerces them to int; Go returns the native integer limits plus raw Body, where null and alias conflicts stay distinguishable.
- Python Resource.fetch/list normalization (dict/location merge, response cache, skip_cache) has no Go counterpart; Go performs fresh stateless requests and keeps the native model separate from HTTP evidence.

### `gophercloud:loadbalancer/v2/quotas.Get`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.API.Get`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.API.CurrentProject`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.ProjectQuotaScope.Get`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.WithGetFields`

- The project scope uses the official /lbaas/quotas/{project} route and avoids duplicating an existing lbaas ResourceBase; generated API.Get keeps the pinned native /quotas route and the selected client configuration remains unchanged.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaScopesCorrectNativeURLWithoutChangingGeneratedAPI`, `api/octavia_project_quotas_contracts_test.go:TestOctaviaProjectQuotasUseOfficialPathsAndGlobalDefaults`.
- Exact project names resolve once through separate Keystone pages; explicit IDs require no catalog/Identity request, recorded v2/v3 auth fixes the target, invalid/defaults-route IDs and lookup authorization failures stop quota HTTP, and Connection cancellation stays observable.
  Tests: `api/octavia_project_quotas_projects_test.go:TestOctaviaQuotaProjectNameUsesSeparateIdentityClientOnce`, `api/octavia_project_quotas_projects_test.go:TestOctaviaQuotaCurrentProjectRecordsAndFreezesV2V3Authentication`, `api/octavia_project_quotas_projects_test.go:TestOctaviaQuotaScopePreflightAndLookupFailures`, `connectiontest/connection_project_quotas_test.go:TestConnectionServiceQuotaNamesUseSeparateIdentityClient`, `connectiontest/connection_project_quotas_test.go:TestConnectionServiceQuotasIDAuthAndCancellation`.
- The native Quota model is retained: non-null canonical loadbalancer/healthmonitor values override legacy aliases, null canonical values fall back to non-null legacy values, and absent/both-null aliases become native zero; raw response fields retain the exact alias/null distinctions.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaAliasesKeepNativeNullFallbackAndRawFields`.
- Scope GET returns fixed ProjectID plus raw large-integer/null/extension fields and copied header/status; response project_id does not change the scope, repeated fields inputs are snapshotted and response metadata is independent across operations.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaProjectQuotasUseOfficialPathsAndGlobalDefaults`.
- GET keeps native HTTP 200-only success, retains rejected response code/header and adds ErrNotFound only for 404; nil/blank fields options fail before HTTP, pre-canceled GET makes no request and an in-flight deadline remains context.DeadlineExceeded.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaSuccessCodesAndStrictMissingPolicy`, `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaInvalidOptionsFailBeforeHTTP`, `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaCancellationAndDeadlinePreserveCause`.

Python/Go differences:

- The SDK-owned project scope deliberately corrects the pinned native /quotas URL to official /lbaas/quotas; the generated low-level API.Get remains available with its original native URL contract.
- Result extraction becomes a QuotaResource containing the unchanged native model, fixed ProjectID, exact raw Body and independent Header/StatusCode; explicit IDs, exact-name resolution and recorded-auth scope binding replace repeated project arguments.
- The scope adds a concrete snapshotted fields-projection option, validates project/option inputs and rejects malformed quota objects instead of accepting the native missing/null quota extractor result. Accepted response decode errors retain raw HTTP evidence outside native retry handling.

### `gophercloud:loadbalancer/v2/quotas.Update`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.API.Update`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.ProjectQuotaScope.Update`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.WithQuotaOptions`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.WithDefaultLimit`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.WithUpdateField`

- The fixed-project PUT uses official /lbaas/quotas/{project} with a quota envelope; native nil/zero/-1 limits stay distinct, explicit null inheritance and creation-time extension snapshots are serialized, and the updated native/raw result retains ProjectID and response metadata.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaProjectQuotasUseOfficialPathsAndGlobalDefaults`, `api/octavia_project_quotas_list_test.go:TestOctaviaQuotaDefaultInheritanceDuplicateAndForgedArgument`.
- WithQuotaOptions snapshots pointer inputs at creation and returns independent application copies; exact machine-sized integers, omitted inputs and stable serialized bodies survive reauthentication and reuse after prior private pointer mutation.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaSnapshotOmissionExactIntegersAndRetry`.
- Typed limits below -1, nil options, core/legacy aliases, identity collisions, unserializable extensions, unsupported query/header/argument inputs and value/default conflicts fail before HTTP; duplicate default inheritance is safe and forged null-limit names fail.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaInvalidOptionsFailBeforeHTTP`, `api/octavia_project_quotas_list_test.go:TestOctaviaQuotaDefaultInheritanceDuplicateAndForgedArgument`.
- PUT preserves native 200/202 success codes, original rejected response code/header and 404 classification; malformed accepted quota JSON/objects/native fields retain exact byte/header/status evidence without retrying the accepted mutation.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaSuccessCodesAndStrictMissingPolicy`, `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaMalformedAcceptedResponsesRetainEvidenceWithoutRetry`.
- Pre-canceled Update prevents HTTP and cancellation during PUT remains context.Canceled.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaCancellationAndDeadlinePreserveCause`.

Python/Go differences:

- The scope uses official /lbaas/quotas while generated API.Update continues delegating to the pinned native /quotas route.
- Caller-owned UpdateOptsBuilder implementations are replaced by concrete native UpdateOpts and SDK-owned extension/default-inheritance options. Core/alias/identity protection and limits-below--1 preflight intentionally strengthen the native builder contract.
- WithQuotaOptions and raw final JSON provide construction-time pointer snapshots, independent reuse and exact integer/retry bodies rather than native BuildRequestBody any/float64 conversion; explicit null limit inheritance is a scope-only input.
- The scope returns native typed values plus fixed ProjectID/raw fields/copied metadata instead of an UpdateResult extractor. Accepted-body decoding preserves HTTP evidence and does not replay an already accepted mutation through the native retry hook.

### `gophercloud:loadbalancer/v2/quotas.Delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.API.Delete`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.ProjectQuotaScope.Reset`
- `github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/quotas.WithResetIgnoreMissing`

- Reset sends a bodyless DELETE to the official fixed-project quota route, returns ProjectID with actual 204 response metadata and performs no follow-up GET to infer limits.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaProjectQuotasUseOfficialPathsAndGlobalDefaults`.
- Reset retains native 202/204 success; native-style strict 404 is the default, explicit ignore-missing suppresses only 404, later false restores strictness and other HTTP statuses retain original code/header causes.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaSuccessCodesAndStrictMissingPolicy`.
- Nil Reset options fail before HTTP and pre-canceled Reset retains context.Canceled without a request.
  Tests: `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaInvalidOptionsFailBeforeHTTP`, `api/octavia_project_quotas_contracts_test.go:TestOctaviaQuotaCancellationAndDeadlinePreserveCause`.

Python/Go differences:

- The SDK singleton exposes Reset to remove overrides and returns ResetResponse(ProjectID, Header, StatusCode), rather than returning a DeleteResult for caller extraction or fabricating updated limits.
- The scope deliberately uses official /lbaas/quotas while generated API.Delete retains the pinned native /quotas route.
- Default strict missing handling preserves the native contract and additionally exposes ErrNotFound; optional WithResetIgnoreMissing changes only 404, with deterministic last-option precedence.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
func manageQuotas(ctx context.Context, client *gophercloud.ServiceClient) error {
    api := quotas.New(client)
    scope, err := api.InProject(ctx, resource.ID("project-id"))
    if err != nil { return err }

    limits, err := scope.Get(ctx)
    if err != nil { return err }
    fmt.Println(limits.ProjectID, limits.Loadbalancer, limits.Listener)

    defaults, err := api.Defaults(ctx)
    if err != nil { return err }
    fmt.Println(defaults.Loadbalancer, defaults.Listener)

    zero, unlimited := 0, -1
    updated, err := scope.Update(ctx, quotas.UpdateOpts{
        Listener: &zero, Loadbalancer: &unlimited,
    }, quotas.WithDefaultLimit(quotas.LimitMembers))
    if err != nil { return err }
    fmt.Println(updated.Header.Get("X-Openstack-Request-Id"))

    for quota, err := range api.ListProjects(ctx, quotas.WithProjectListOptions(
        quotas.ProjectListOpts{Limit: 100},
    )) {
        if err != nil { return err }
        fmt.Println(quota.ProjectID, quota.Pool)
    }

    _, err = scope.Reset(ctx)
    return err
}
```

## References

- [Pagination](https://docs.openstack.org/octavia/latest/_modules/octavia/api/common/pagination.html)
- [V2](https://docs.openstack.org/api-ref/load-balancer/v2/#quotas)
- [native requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/loadbalancer/v2/quotas/requests.go)
- [native URL](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/loadbalancer/v2/quotas/urls.go)
- [Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/load_balancer/v2/_proxy.py)
- [Python quota resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/load_balancer/v2/quota.py)
- [Octavia Project Quotas Contracts Test](../../../api/octavia_project_quotas_contracts_test.go)
- [Octavia Project Quotas Projects Test](../../../api/octavia_project_quotas_projects_test.go)
- [Octavia Project Quotas List Test](../../../api/octavia_project_quotas_list_test.go)
