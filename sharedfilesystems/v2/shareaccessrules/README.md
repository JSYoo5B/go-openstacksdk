# Shareaccessrules

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:shared_file_system/v2/access_rules`

Status: `unresolved`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules.API.InShare`
- `github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules.AccessRuleScope.List`
- `github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules.AccessRuleScope.All`
- `github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules.WithListMaxItems`
- `github.com/JSYoo5B/go-openstacksdk/sharedfilesystems/v2/shareaccessrules.WithListPaginated`

- A fixed share scope lists GET200 access_list once with owned typed local caps and first-response controls. Repeated traversals have independent counters, options use last-wins and zero unlimited, explicit vendor/limit/marker/header inputs remain unchanged and local controls never invent a wire-limit hint or follow advertised links.
  Tests: `api/manila_access_list_controls_test.go:TestManilaAccessListControlsSnapshotAndSingleCollection`, `api/manila_access_list_controls_test.go:TestManilaAccessListControlsCapBeforeLaterDecode`.
- Numeric Manila>=2.45 validation checks actual service type and case-insensitive version headers, rejects mismatches/latest/invalid selections, and rechecks after caller options or parent-name resolution. A matching legacy header is required for a manual empty-Type client.
  Tests: `api/manila_access_list_controls_test.go:TestManilaAccessListControlsStrictSourceVersion`, `api/manila_access_list_controls_test.go:TestManilaAccessListControlsRecheckSourceAfterOptionsAndLookup`.
- Negative caps, fixed share/local-control query collisions, unsupported fields/arguments/headers and canceled context fail before child HTTP. Consumed-row/envelope failures retain whole accepted-page body/header/status; HTTP failures keep native evidence, and terminal cancellation remains observable while consumer break stops consumption.
  Tests: `api/manila_access_list_controls_test.go:TestManilaAccessListControlsRejectBeforeHTTP`, `api/manila_access_list_controls_test.go:TestManilaAccessListControlsAcceptedAndNativeErrors`, `api/manila_access_list_controls_test.go:TestManilaAccessListControlsTerminalCancellation`.
- Only exact lowercase response id/share_id establish access and parent identities. Case-folded aliases remain raw extensions, cannot shadow a foreign parent, and do not fabricate an omitted parent; invalid canonical identities fail before routing while a known valid created ID survives later decoding errors.
  Tests: `api/manila_access_identity_test.go:TestManilaAccessIdentityCanonicalFieldsAndRawAliases`, `api/manila_access_identity_test.go:TestManilaAccessIdentityShadowCannotAuthorizeDeny`, `api/manila_access_identity_test.go:TestManilaAccessIdentityOmittedParentAndFixedMutationRoute`, `api/manila_access_identity_test.go:TestManilaAccessIdentityRejectsMissingInvalidCanonicalIDs`, `api/manila_access_identity_test.go:TestManilaAccessIdentityKnownCreatedIDSurvivesDecodeFailure`.

Python/Go differences:

- Go requires an explicit ID/Name share reference and returns stateless native-backed rule snapshots with raw Body/Header and separate ParentShareID, rather than reusing Python Resource input/cache or overlaying a response share_id.
- The reviewed modern Manila endpoint returns one collection with no server paging contract. Go does not infer Resource.list marker fallback or follow next links; both Paginated settings consume one GET and MaxItems is only a raw-row bound.
- Known options protect fixed parent and local control keys, but explicitly supplied vendor query values reach the deployment rather than Python unknown-query omission. Only exact lowercase wire identities establish scope ownership; strict typed/parent validation and accepted-response evidence are deliberate Go policies.

Unresolved boundaries:

- Compare automatic known Body/query classification, descriptor aliases/defaults/coercion and unknown-query omission; the current scope has no arbitrary local Body filter API.
- Map inherited per-call base_path/microversion/header controls and deprecated proxy JMESPath behavior beyond the tested local cap/one-response policy.
- Compare arbitrary Resource/cache input ownership, URI overlay and full lifecycle semantics. Strict modern2.45+ scope behavior does not implement the earlier legacy access_list action contract.

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

## References

- [Shared File System](https://docs.openstack.org/api-ref/shared-file-system/#share-access-rules)
- [Microversions](../../../docs/microversions.md)
- [Shared File System](https://docs.openstack.org/api-ref/shared-file-system/#allow-access)
- [Python Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/shared_file_system/v2/_proxy.py)
- [ShareAccessRule](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/shared_file_system/v2/share_access_rule.py)
- [Gophercloud v2.15.0 requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/sharedfilesystems/v2/shareaccessrules/requests.go)
- [share actions](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/sharedfilesystems/v2/shares/requests.go)
