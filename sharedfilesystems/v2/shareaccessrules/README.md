# Shareaccessrules

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

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

    "github.com/JSYoo5B/go-openstacksdk/request"
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
    }, request.Custom(func(c *request.Config[shareaccessrules.AllowOpts]) error { c.Options.AccessLevel = "ro"; return nil }),
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
    "github.com/JSYoo5B/go-openstacksdk/request"
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
        request.MaxItems(5),
        request.Paginated(false))
}
```

Python Body filters such as `access_rules(share, access_level="rw")` become an
explicit local option. The filter never reaches the wire and `request.MaxItems`
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
