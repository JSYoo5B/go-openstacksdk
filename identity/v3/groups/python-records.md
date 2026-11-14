# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import openstack

def main():
    conn = openstack.connect()
    found = conn.identity.find_group(
        "ops", domain_id="domain-id", skip_cache=True,
        error_message="Group lookup failed",
    )
    if found is not None:
        print(json.dumps(found.to_dict(), indent=2, ensure_ascii=False))

    for row in conn.identity.groups(
        domain_id="domain-id", description="operations", limit=100,
        max_items=100,
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))

    selected = conn.search_groups(
        name_or_id="ops-*", domain_id="domain-id",
        filters={"description": "operations"},
    )
    print(json.dumps([row.to_dict() for row in selected], indent=2))

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

    openstack "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/groups"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSON(value any) {
    encoded, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Printf("%s\n", encoded)
}

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }

    found, err := service.Groups.FindRecord(ctx, "ops",
        request.Query("domain_id", "domain-id"),
        request.Query("skip_cache", true),
        request.Query("error_message", "Group lookup failed"),
    )
    if err != nil { log.Fatal(err) }
    if found != nil { printJSON(found.Resource.Body) }

    for row, err := range service.Groups.ListRecords(ctx,
        groups.WithRecordListQuery("domain_id", "domain-id"),
        groups.WithRecordListQuery("description", "operations"),
        request.Limit(100),
        request.MaxItems(100),
    ) {
        if err != nil { log.Fatal(err) }
        printJSON(row.Resource.Body)
    }

    cloud := identity.NewGroupReadCloud(service.Groups)
    selected, err := cloud.SearchGroups(ctx, "ops-*",
        request.Custom(func(c *identity.GroupSearchOpts) error { v := "domain-id"; c.DomainID = &v; return nil }),
        identity.WithGroupSearchFilters(json.RawMessage(`{"description":"operations"}`)),
    )
    if err != nil { log.Fatal(err) }
    printJSON(selected.Value)
}
```

```go
proxy, err := service.Groups.ProxyListRecords(ctx,
    groups.WithRecordListJMESPath("[*].id"),
)
if err != nil { log.Fatal(err) }
fmt.Println(string(proxy.JSON))
```

## References

- [Policy](https://docs.openstack.org/keystone/2025.2/configuration/policy.html)
- [Group](https://opendev.org/openstack/keystone/src/tag/28.0.0/keystone/common/policies/group.py)
- [Groups](https://opendev.org/openstack/keystone/src/tag/28.0.0/keystone/api/groups.py)
- [User Find](../users/python-record-find.md)
- [Project Find](../projects/python-record-reads.md)
