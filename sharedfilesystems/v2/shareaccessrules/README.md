# Shareaccessrules

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:shared_file_system/v2/access_rules`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules.API.InShare`
- `github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules.AccessRuleScope.List`
- `github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules.AccessRuleScope.All`
- `github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules.WithListPaginated`
- `github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules.WithListBodyFilter`
- `github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules.API.ListAccessAction`

- A fixed share scope lists GET200 access_list once with owned typed local caps and first-response controls. Repeated traversals have independent counters, options use last-wins and zero unlimited, explicit vendor/limit/marker/header inputs remain unchanged and local controls never invent a wire-limit hint or follow advertised links.
  Tests: `api/manila_access_list_controls_test.go:TestManilaAccessListControlsSnapshotAndSingleCollection`, `api/manila_access_list_controls_test.go:TestManilaAccessListControlsCapBeforeLaterDecode`.
- Numeric Manila>=2.45 validation checks actual service type and case-insensitive version headers, rejects mismatches/latest/invalid selections, and rechecks after caller options or parent-name resolution. A matching legacy header is required for a manual empty-Type client.
  Tests: `api/manila_access_list_controls_test.go:TestManilaAccessListControlsStrictSourceVersion`, `api/manila_access_list_controls_test.go:TestManilaAccessListControlsRecheckSourceAfterOptionsAndLookup`.
- Negative caps, fixed share/local-control query collisions, unsupported fields/arguments/headers and canceled context fail before child HTTP. Consumed-row/envelope failures retain whole accepted-page body/header/status; HTTP failures keep native evidence, and terminal cancellation remains observable while consumer break stops consumption.
  Tests: `api/manila_access_list_controls_test.go:TestManilaAccessListControlsRejectBeforeHTTP`, `api/manila_access_list_controls_test.go:TestManilaAccessListControlsAcceptedAndNativeErrors`, `api/manila_access_list_controls_test.go:TestManilaAccessListControlsTerminalCancellation`.
- Only exact lowercase response id/share_id establish access and parent identities. Case-folded aliases remain raw extensions, cannot shadow a foreign parent, and do not fabricate an omitted parent; invalid canonical identities fail before routing while a known valid created ID survives later decoding errors.
  Tests: `api/manila_access_identity_test.go:TestManilaAccessIdentityCanonicalFieldsAndRawAliases`, `api/manila_access_identity_test.go:TestManilaAccessIdentityShadowCannotAuthorizeDeny`, `api/manila_access_identity_test.go:TestManilaAccessIdentityOmittedParentAndFixedMutationRoute`, `api/manila_access_identity_test.go:TestManilaAccessIdentityRejectsMissingInvalidCanonicalIDs`, `api/manila_access_identity_test.go:TestManilaAccessIdentityKnownCreatedIDSurvivesDecodeFailure`.
- WithListBodyFilter applies Python Resource.list client-side Body filters locally to the fixed share listing: named ShareAccessRule attributes only (share_id stays scope-owned), creation-time JSON snapshots, last-wins per name, object subset/exact/null-as-missing matching through the shared jsonfilter policy, no wire query, MaxItems counting raw rows before filtering, and unknown/unserializable/forged filters failing before HTTP.
  Tests: `api/py_manila_access_body_filters_test.go:TestManilaAccessListBodyFiltersStayLocalAndCountRawRows`.
- ListAccessAction exposes the legacy share action listing: POST shares/{id}/action with a null value, choosing os-access_list for an empty/2.0-2.6 selection and access_list for 2.7-2.44 from the client microversion without mutating it. 2.45+ fails with ErrUnsupported and latest/invalid versions, wrong service types, empty IDs and canceled contexts fail before HTTP. Only 200 succeeds; rows keep raw fields, ParentShareID and an omitted share_id, a foreign share_id fails with ErrParentMismatch, and envelope/row decode failures keep the accepted response.
  Tests: `api/py_manila_access_list_action_test.go:TestManilaLegacyAccessListActionSelectsActionByMicroversion`, `api/py_manila_access_list_action_test.go:TestManilaLegacyAccessListActionRejectsBeforeHTTP`, `api/py_manila_access_list_action_test.go:TestManilaLegacyAccessListActionAcceptedAndHTTPErrors`.

Python/Go differences:

- Go requires an explicit ID/Name share reference and returns stateless native-backed rule snapshots with raw Body/Header and separate ParentShareID, rather than reusing Python Resource input/cache or overlaying a response share_id.
- The reviewed modern Manila endpoint returns one collection with no server paging contract. Go does not infer Resource.list marker fallback or follow next links; both Paginated settings consume one GET and MaxItems is only a raw-row bound.
- Known options protect fixed parent and local control keys, but explicitly supplied vendor query values reach the deployment rather than Python unknown-query omission. Only exact lowercase wire identities establish scope ownership; strict typed/parent validation and accepted-response evidence are deliberate Go policies.
- Python classifies unknown keyword arguments automatically: Body attributes become client-side filters and other unknown keys are discarded. Go uses explicit WithListBodyFilter for local filters and rejects unknown filter names; filters compare the raw response JSON (shared jsonfilter policy) rather than Python descriptor-coerced values such as lock_reason's declared bool type.
- Python forwards per-call base_path, microversion, headers and the deprecated jmespath_filters through Proxy._list. Go fixes the 2.45+ share-access-rules route, uses the client's configured microversion (select another client to change it), takes per-call headers through WithListHeader and provides no JMESPath; remaining Adapter/session controls are the shared SDK-R1/C1/S1 boundary (docs/sdk-support-ledger.md).
- Pinned Python access_rules has no pre-2.45 path. Go additionally exposes the legacy os-access_list/access_list share action through API.ListAccessAction (reviewed against Manila's v2 shares controller), gated by the selected microversion; it is a separate method and never a silent fallback for the scope List.

## Generated `API.Get` and `API.List`

`API.Get` (`GET /share-access-rules/{id}`) and `API.List` (`GET /share-access-rules?share_id={id}`) are thin wrappers over the native calls. Only HTTP 200 succeeds; other statuses stay native errors and nothing is retried. `Get` does not check the rule's `share_id` against a parent, and `List` has no paging, caps or extra query options. `List` reads one response, never follows links, and returns an empty slice for an empty, null or missing `access_list`. Neither call validates the microversion locally; the client's selected microversion is sent as the header when set. Use `InShare` and `AccessRuleScope` for parent-checked, 2.45-validated, capped access.

The pinned native `List` appends the raw share ID after `?share_id=`, so an ID such as `x&injected=yes` would add a query parameter. The generator therefore replaces that call with an audited helper that sends the ID through `url.QueryEscape` as a single `share_id` value; the URL, 200-only handling and result type are otherwise native. The override is pinned to the inspected native declaration and generation fails if it drifts. `Get` places the ID in the URL path unchanged.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```go
package main

import (
    "context"
    "fmt"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx,
        sdk.WithMicroversionRange(sdk.SharedFileSystem, "2.45", "2.82"),
    )
    if err != nil { panic(err) }
    service, err := conn.SharedFileSystemV2(ctx)
    if err != nil { panic(err) }
    rules, err := service.ShareAccessRules.InShare(ctx, resource.Name("team-data"))
    if err != nil { panic(err) }

    created, err := rules.Allow(ctx, shareaccessrules.AllowOpts{
        AccessType: "ip",
        AccessTo: "192.0.2.0/24",
    }, shareaccessrules.WithAccessLevel("ro"),
       shareaccessrules.WithMetadata(map[string]string{"owner": "ops"}))
    if err != nil {
        // The action may have succeeded before response decoding failed.
        // A non-nil created value retains any known ID and raw response.
        fmt.Println(created)
        panic(err)
    }
    ready, err := rules.Wait(ctx, resource.ID(created.ID), "active",
        resource.WithTimeout(2*time.Minute),
        resource.WithPollInterval(time.Second))
    if err != nil { panic(err) }
    fmt.Println(ready.ID, ready.State)

    for rule, err := range rules.List(ctx) {
        if err != nil { panic(err) }
        fmt.Println(rule.ID, rule.AccessTo, rule.State)
    }
    if err := rules.Delete(ctx, resource.ID(created.ID)); err != nil { panic(err) }
    if err := rules.WaitDeleted(ctx, resource.ID(created.ID)); err != nil { panic(err) }
}
```

```go
package example

import (
    "context"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/resource"
    "github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules"
)

func FirstAccessRules(ctx context.Context, conn *sdk.Connection,
    share resource.Ref) ([]*shareaccessrules.AccessRule, error) {
    service, err := conn.SharedFileSystemV2(ctx)
    if err != nil { return nil, err }
    scope, err := service.ShareAccessRules.InShare(ctx, share)
    if err != nil { return nil, err }
    return scope.All(ctx,
        shareaccessrules.WithListMaxItems(5),
        shareaccessrules.WithListPaginated(false))
}
```

Python Body filters such as `access_rules(share, access_level="rw")` become an
explicit local option. The filter never reaches the wire and `WithListMaxItems`
still counts raw rows before filtering, as Python's `max_items` does:

```python
rules = conn.shared_file_system.access_rules("share-id", access_level="rw", state="active")
```

```go
func ActiveWritableRules(ctx context.Context, scope *shareaccessrules.AccessRuleScope) ([]*shareaccessrules.AccessRule, error) {
    return scope.All(ctx,
        shareaccessrules.WithListBodyFilter("access_level", "rw"),
        shareaccessrules.WithListBodyFilter("state", "active"))
}
```

Deployments that only serve Manila 2.44 or earlier can list rules through the
legacy share action. The client's numeric microversion selects
`os-access_list` (unset through 2.6) or `access_list` (2.7 to 2.44); 2.45 or
newer fails with `resource.ErrUnsupported` before HTTP, so use `InShare` there.
Legacy rows carry no `share_id`; each rule keeps `ParentShareID`.

```go
func LegacyRules(ctx context.Context, client *gophercloud.ServiceClient, shareID string) error {
    client.Microversion = "2.40" // configure before sharing the client
    result, err := shareaccessrules.New(client).ListAccessAction(ctx, shareID)
    if err != nil { return err }
    for _, rule := range result.Rules {
        fmt.Println(result.Action, rule.ID, rule.AccessTo, rule.State)
    }
    return nil
}
```

## References

- [Shared File System](https://docs.openstack.org/api-ref/shared-file-system/#share-access-rules)
- [Microversions](../../../docs/microversions.md)
- [Shared File System](https://docs.openstack.org/api-ref/shared-file-system/#allow-access)
- [Python Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/shared_file_system/v2/_proxy.py)
- [ShareAccessRule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/shared_file_system/v2/share_access_rule.py)
- [Gophercloud v2.15.0 requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/sharedfilesystems/v2/shareaccessrules/requests.go)
- [share actions](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/sharedfilesystems/v2/shares/requests.go)
