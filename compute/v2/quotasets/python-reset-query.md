# Python Reset Query

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/revert_quota_set`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.API.InProject`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.Reset`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.ProjectQuotaScope.InUser`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.UserQuotaScope.Reset`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithResetQuery`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/quotasets.WithResetIgnoreMissing`

- WithResetQuery adds revert_quota_set(**query) parameters as literal values to the project DELETE os-quota-sets/{project} and next to the fixed user_id on the user scope, a later value for a key wins, a 404 stays an error that is also resource.ErrNotFound unless WithResetIgnoreMissing(true) is given, and a 500 stays a Gophercloud status error inside resource.OperationError.
  Tests: `compute/v2/quotasets/python_reset_query_test.go:TestPythonQuotaResetForwardsQueryAndMissingPolicy`.
- WithResetQuery rejects user_id, a blank or control-character key, an invalid UTF-8 value and a nil option on both scopes before HTTP.
  Tests: `compute/v2/quotasets/python_reset_query_test.go:TestPythonQuotaResetQueryRejectsScopeOwnedAndInvalidKeys`.
- The no-query project reset uses bodyless DELETE and strict missing handling without an additional GET.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaLimitsDetailUpdateAndResetPreserveTargetAndResponse`, `api/project_quotas_contracts_test.go:TestProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`.
- The original HTTP status, body and response header remain in the error chain; 404 is additionally ErrNotFound and context cancellation/deadlines remain observable.
  Tests: `api/project_quotas_contracts_test.go:TestProjectQuotaHTTPFailuresPreserveStatusBodyHeaderAndNotFound`, `api/project_quotas_contracts_test.go:TestProjectQuotaCancellationAndTimeoutPreserveCause`.
- User Reset sends bodyless DELETE with exactly one fixed user_id, accepts 202/204 without a follow-up request, rejects unexpected success codes, preserves both identities and HTTP causes, and ignores only an explicitly permitted 404. Invalid or zero scopes cannot fall back to project-wide Reset.
  Tests: `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserOperationsKeepQueryTargetAndRawResponse`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserResetAccepts202And204WithoutFollowUp`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserErrorsKeepBothIdentitiesAndMissingPolicy`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserInvalidInputsAndZeroScopeCannotResetProject`.
- Reset redirect fixtures reject dropped, changed or duplicate user queries, changed project/origin/method, opaque URLs and virtual-host mutation before a second unsafe request. Same-pair redirects retain the caller policy; its rejection remains in the error chain.
  Tests: `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserRedirectCannotChangeProjectUserOriginOrMethod`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserRedirectKeepsSamePairAndCallerPolicy`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserPreservesCallerRedirectRejectionAndExplicitAuthHeaders`.
- User Reset retains its pair and single query through reauthentication/backoff/retry, observes canceled and expired contexts, and preserves caller and HTTP timeout cancellation even when redirect policy replaces the request context.
  Tests: `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserRetryAndReauthKeepOneQueryAndExactBody`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserContextAndCurrentProjectKeepExplicitUser`, `api/nova_quota_defaults_users_test.go:TestNovaQuotaUserRedirectRetainsCallerAndHTTPTimeoutContexts`.

Python/Go differences:

- Python urlencodes str(value) for each query value; WithResetQuery takes the literal text, so the caller passes "True" or a Python list spelling to reproduce those values.
- Python accepts every status below 400; Go accepts Nova's 202 and 204.
- Python sends min(QuotaSet _max_microversion 2.56, server maximum); Go sends the client microversion.
- Python passes user_id as a query keyword; Go maps it to UserQuotaScope and rejects user_id in WithResetQuery. Go encodes query keys in sorted order.
- Python returns None; Go returns ResetResponse with the fixed project/user identities and copied headers and can explicitly ignore a 404.
- UserQuotaScope is separate from ProjectQuotaScope, so a user Reset cannot promote a project-wide operation; guarded redirects reject changes to the exact request target rather than forwarding arbitrary query mutations.

## References

- [compute proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py)
- [`Resource._prepare_request`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)
