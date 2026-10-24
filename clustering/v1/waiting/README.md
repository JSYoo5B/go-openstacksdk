# Waiting

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:clustering/v1/wait_for_status`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/events.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes.API.WaitForStatus`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.Scope.WaitForStatus`

- Required nonempty status uses unlimited SDK time, ERROR-only failures and2-second intervals by default, with case-insensitive target precedence and caller last-wins failure/attribute/timeout policy. Four status models and five explicit string-attribute models are verified; default unsupported attributes fail before HTTP.
  Tests: `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNineAPIsPreserveHTTPAndEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsValidateTargetsAndAttributesBeforeHTTP`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsFailureDefaultsReplacementAndTargetPrecedence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeletionStatusAndExplicitAttribute`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeadlineDefaultsAndCallerOverrides`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameOnceStableIDAndFreshAuth`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsSourceRecheckedBeforeEachPoll`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsStatelessFirstGETAndCallbackCancellation`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsHTTPFailuresAndAcceptedMalformedEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsTransportAndInFlightContextFailures`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameLookupFailuresNeverPollDetail`.
- Typed polling uses the selected Senlin source, exact reverse-proxy path and current provider token for each GET; explicit IDs remain fixed even when response IDs differ. Name lookup is exact, happens once and preserves missing/duplicate/invalid-ID/later-page failures before detail polling. Neither waiter submits DELETE or follows action/channel/Location URLs.
  Tests: `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNineAPIsPreserveHTTPAndEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsValidateTargetsAndAttributesBeforeHTTP`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsFailureDefaultsReplacementAndTargetPrecedence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeletionStatusAndExplicitAttribute`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeadlineDefaultsAndCallerOverrides`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameOnceStableIDAndFreshAuth`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsSourceRecheckedBeforeEachPoll`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsStatelessFirstGETAndCallbackCancellation`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsHTTPFailuresAndAcceptedMalformedEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsTransportAndInFlightContextFailures`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameLookupFailuresNeverPollDetail`, `connection_senlin_wait_test.go:TestConnectionSenlinWaitersShareLiveTokenVersionAndFixedRoutes`.
- Default and overridden deadlines are observed on actual outgoing HTTP contexts, without sleeping for120seconds. Parent cancellation/deadlines are respected; callbacks report zero for nonterminal Senlin models, stop at completion/failure/HTTP errors, and callback cancellation prevents another GET. Existing Resources.Wait retains its5-minute common policy.
  Tests: `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNineAPIsPreserveHTTPAndEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsValidateTargetsAndAttributesBeforeHTTP`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsFailureDefaultsReplacementAndTargetPrecedence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeletionStatusAndExplicitAttribute`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeadlineDefaultsAndCallerOverrides`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameOnceStableIDAndFreshAuth`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsSourceRecheckedBeforeEachPoll`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsStatelessFirstGETAndCallbackCancellation`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsHTTPFailuresAndAcceptedMalformedEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsTransportAndInFlightContextFailures`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameLookupFailuresNeverPollDetail`.
- ClusterPolicy polling fixes one canonical parent and direct policy identity, uses PolicyID rather than binding ID for Name lookup, rejects default status before HTTP, and supports GET-only deletion polling without detaching a policy. Exact lower-case raw identities/names own routes and each request rechecks the service source.
  Tests: `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesPreflightAndScopedWaits`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesExactBodyIdentityOwnsRoutes`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesExactNameIgnoresCaseFoldedExtensions`, `connection_senlin_scopes_test.go:TestConnectionSenlinParentScopesShareLiveAuthVersionAndFixedRoutes`.

Python/Go differences:

- Go uses explicit Ref and new typed results rather than cached mutable Python Resource input. IDs begin with a fresh GET; Names begin with the server list lookup result. WaitForDelete returns error instead of retaining the original Resource pointer.
- Go requires a nonempty string target, positive duration intervals and exported typed string attributes. Python nullable target/attribute and interval=None/zero normalization are distinct; unknown raw Body attributes or nested attribute paths are not inferred.
- Go passes deadlines into HTTP requests and gives cancellation priority instead of checking timeout only between fetches. Published status/envelope rules preserve exact Body/Header/StatusCode and reject null/malformed success instead of permissive falsy completion.
- BuildInfo has no new singleton polling route: pinned proxy get uses requires_id=False, while inherited wait fetch uses its normal requires_id=True and a normal ID-less BuildInfo cannot be fetched that way. Services remain list-only rather than inventing GET/delete/status capabilities.
- Python returns the passed Resource immediately, with no request, when its cached attribute already equals the target (wait_for_status), and otherwise fetches with skip_cache=True; Go always begins with a fresh request.
- Python wait_for_delete returns the original Resource when a fetch is missing or its status equals deleted; Go WaitForDelete returns only an error.
- Python accepts any Resource subclass and an arbitrary attribute name; Go provides waiters for the typed Senlin models and exported string attributes only, so arbitrary or custom Resource inputs have no equivalent.
- Python callback receives the progress attribute or 0; Senlin models have no progress field and Go reports zero.

### `python:clustering/v1/wait_for_delete`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/actions.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusters.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/nodes.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/events.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiles.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policies.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/receivers.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/profiletypes.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/policytypes.API.WaitForDelete`
- `github.com/JSYoo5B/go-openstacksdk/clustering/v1/clusterpolicies.Scope.WaitForDelete`

- Deletion defaults to120-second SDK timeout. Four declared status models complete on deleted or404, while five statusless models complete on404 and ignore incidental raw status. No delete capability is fabricated and403/409/native/transport/malformed200 remain errors.
  Tests: `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNineAPIsPreserveHTTPAndEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsValidateTargetsAndAttributesBeforeHTTP`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsFailureDefaultsReplacementAndTargetPrecedence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeletionStatusAndExplicitAttribute`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeadlineDefaultsAndCallerOverrides`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameOnceStableIDAndFreshAuth`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsSourceRecheckedBeforeEachPoll`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsStatelessFirstGETAndCallbackCancellation`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsHTTPFailuresAndAcceptedMalformedEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsTransportAndInFlightContextFailures`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameLookupFailuresNeverPollDetail`.
- Typed polling uses the selected Senlin source, exact reverse-proxy path and current provider token for each GET; explicit IDs remain fixed even when response IDs differ. Name lookup is exact, happens once and preserves missing/duplicate/invalid-ID/later-page failures before detail polling. Neither waiter submits DELETE or follows action/channel/Location URLs.
  Tests: `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNineAPIsPreserveHTTPAndEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsValidateTargetsAndAttributesBeforeHTTP`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsFailureDefaultsReplacementAndTargetPrecedence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeletionStatusAndExplicitAttribute`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeadlineDefaultsAndCallerOverrides`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameOnceStableIDAndFreshAuth`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsSourceRecheckedBeforeEachPoll`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsStatelessFirstGETAndCallbackCancellation`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsHTTPFailuresAndAcceptedMalformedEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsTransportAndInFlightContextFailures`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameLookupFailuresNeverPollDetail`, `connection_senlin_wait_test.go:TestConnectionSenlinWaitersShareLiveTokenVersionAndFixedRoutes`.
- Default and overridden deadlines are observed on actual outgoing HTTP contexts, without sleeping for120seconds. Parent cancellation/deadlines are respected; callbacks report zero for nonterminal Senlin models, stop at completion/failure/HTTP errors, and callback cancellation prevents another GET. Existing Resources.Wait retains its5-minute common policy.
  Tests: `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNineAPIsPreserveHTTPAndEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsValidateTargetsAndAttributesBeforeHTTP`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsFailureDefaultsReplacementAndTargetPrecedence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeletionStatusAndExplicitAttribute`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsDeadlineDefaultsAndCallerOverrides`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameOnceStableIDAndFreshAuth`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsSourceRecheckedBeforeEachPoll`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsStatelessFirstGETAndCallbackCancellation`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsHTTPFailuresAndAcceptedMalformedEvidence`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsTransportAndInFlightContextFailures`, `api/clustering_wait_contracts_test.go:TestClusteringWaitContractsNameLookupFailuresNeverPollDetail`.
- ClusterPolicy polling fixes one canonical parent and direct policy identity, uses PolicyID rather than binding ID for Name lookup, rejects default status before HTTP, and supports GET-only deletion polling without detaching a policy. Exact lower-case raw identities/names own routes and each request rechecks the service source.
  Tests: `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesPreflightAndScopedWaits`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesExactBodyIdentityOwnsRoutes`, `api/clustering_cluster_policies_test.go:TestClusteringClusterPoliciesExactNameIgnoresCaseFoldedExtensions`, `connection_senlin_scopes_test.go:TestConnectionSenlinParentScopesShareLiveAuthVersionAndFixedRoutes`.

Python/Go differences:

- Go uses explicit Ref and new typed results rather than cached mutable Python Resource input. IDs begin with a fresh GET; Names begin with the server list lookup result. WaitForDelete returns error instead of retaining the original Resource pointer.
- Go requires a nonempty string target, positive duration intervals and exported typed string attributes. Python nullable target/attribute and interval=None/zero normalization are distinct; unknown raw Body attributes or nested attribute paths are not inferred.
- Go passes deadlines into HTTP requests and gives cancellation priority instead of checking timeout only between fetches. Published status/envelope rules preserve exact Body/Header/StatusCode and reject null/malformed success instead of permissive falsy completion.
- BuildInfo has no new singleton polling route: pinned proxy get uses requires_id=False, while inherited wait fetch uses its normal requires_id=True and a normal ID-less BuildInfo cannot be fetched that way. Services remain list-only rather than inventing GET/delete/status capabilities.
- Python returns the passed Resource immediately, with no request, when its cached attribute already equals the target (wait_for_status), and otherwise fetches with skip_cache=True; Go always begins with a fresh request.
- Python wait_for_delete returns the original Resource when a fetch is missing or its status equals deleted; Go WaitForDelete returns only an error.
- Python accepts any Resource subclass and an arbitrary attribute name; Go provides waiters for the typed Senlin models and exported string attributes only, so arbitrary or custom Resource inputs have no equivalent.
- Python callback receives the progress attribute or 0; Senlin models have no progress field and Go reports zero.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
cluster = conn.clustering.get_cluster("CLUSTER_ID")
cluster = conn.clustering.wait_for_status(
    cluster, "ACTIVE", failures=["ERROR"], interval=2, wait=180,
)
```

```go
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
cluster, err := service.Clusters.WaitForStatus(ctx,
    resource.ID("CLUSTER_ID"), "ACTIVE",
    resource.WithTimeout(3*time.Minute))
if err != nil { return err }
fmt.Println(cluster.ID, cluster.Status, cluster.Header, cluster.Body)
```

```go
profile, err := service.Profiles.WaitForStatus(ctx,
    resource.ID("PROFILE_ID"), "renamed-profile",
    resource.WithStatusAttribute("name"),
    resource.WithFailureStates(),
    resource.WithTimeout(30*time.Second))
if err != nil { return err }
fmt.Println(profile.Name)
```

```python
conn.clustering.delete_node(node)
node = conn.clustering.wait_for_delete(node)
```

```go
submission, err := service.Nodes.Delete(ctx, resource.ID("NODE_ID"))
if err != nil { return err }
fmt.Println(submission.ActionID)
if err := service.Nodes.WaitForDelete(ctx, resource.ID("NODE_ID")); err != nil {
    return err
}
```

```go
err = service.Nodes.WaitForDelete(ctx, resource.ID("NODE_ID"),
    resource.WithTimeout(10*time.Minute),
    resource.WithUnlimitedWait(),
    resource.WithPollInterval(time.Second),
    resource.WithProgressCallback(func(progress int) { fmt.Println(progress) }))
if err != nil { return err }
```

## References

- [ClusterPolicy scope](../clusterpolicies/README.md)
- [ClusterAttributes](../clusterattributes/README.md)
- [Senlin proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L1350)
- [Resource wait helpers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591)
- [timeout iterator](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53)
- [Clustering Wait Contracts Test](../../../api/clustering_wait_contracts_test.go)
- [Clustering Cluster Policies Test](../../../api/clustering_cluster_policies_test.go)
