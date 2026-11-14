# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import openstack

def main():
    conn = openstack.connect()
    for role in conn.identity.roles(description="operations", limit=2,
                                    max_items=2):
        print(json.dumps(role.to_dict(), indent=2, ensure_ascii=False))

    listed = conn.list_roles(description="operations")
    print(json.dumps([role.to_dict() for role in listed], indent=2))
    selected = conn.search_roles("read*", filters={"options": {"immutable": False}})
    print(json.dumps([role.to_dict() for role in selected], indent=2))
    count = conn.search_roles(filters="length(@)")
    print(json.dumps(count, indent=2))

if __name__ == "__main__":
    main()
```

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSON(value any) {
    encoded, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(encoded))
}

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }

    for role, err := range service.Roles.ListRecords(ctx,
        request.Query("description", "operations"),
        request.Limit(2), request.MaxItems(2)) {
        if err != nil { log.Fatal(err) }
        printJSON(role.Resource)
    }
    cloud := identity.NewRoleReadCloud(service.Roles)
    listed, err := cloud.ListRoles(ctx,
        request.Query("description", "operations"))
    if err != nil { log.Fatal(err) }
    printJSON(listed.JSON)
    selected, err := cloud.SearchRoles(ctx, "read*",
        identity.WithRoleSearchFilters(json.RawMessage(`{"options":{"immutable":false}}`)))
    if err != nil { log.Fatal(err) }
    printJSON(selected.Value)
    count, err := cloud.SearchRoles(ctx, "",
        identity.WithRoleSearchFilters(json.RawMessage(`"length(@)"`)))
    if err != nil { log.Fatal(err) }
    printJSON(count.Value)
}
```

## References

- [openstacksdk ef55d7d](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/role.py)
- [Source Proxy._list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/proxy.py#L866)
- [Keystone28.0.0 Role policy](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/role.py#L99-L119)
- [controller](https://github.com/openstack/keystone/blob/28.0.0/keystone/api/roles.py#L45-L72)
- [Base](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/base.py)
- [Policy](https://docs.openstack.org/keystone/2025.2/configuration/policy.html)
- [Python Record Find](python-record-find.md)
