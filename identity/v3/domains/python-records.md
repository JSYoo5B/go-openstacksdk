# Python Records

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import openstack

def main():
    conn = openstack.connect()
    for row in conn.identity.domains(
        is_enabled=True, description="operations", limit=100, max_items=100
    ):
        print(json.dumps(row.to_dict(), indent=2, ensure_ascii=False))

    listed = conn.list_domains(is_enabled=True, description="operations")
    print(json.dumps([row.to_dict() for row in listed], indent=2))

    # Source distinguishes None from an explicitly empty identifier.
    server_filtered = conn.search_domains(filters={"name": "team"})
    locally_filtered = conn.search_domains(
        filters={"description": "operations"}, name_or_id="team-*"
    )
    print(json.dumps([row.to_dict() for row in server_filtered], indent=2))
    print(json.dumps([row.to_dict() for row in locally_filtered], indent=2))

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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/domains"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Printf("%s\n", data)
}

func main() {
    ctx := context.Background()
    conn, err := openstack.Connect(ctx)
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    for row, err := range service.Domains.ListRecords(ctx,
        domains.WithRecordListQuery("is_enabled", true),
        domains.WithRecordListQuery("description", "operations"),
        request.Limit(100),
        request.MaxItems(100),
    ) {
        if err != nil { log.Fatal(err) }
        printJSON(row.Resource.Body)
    }

    cloud := identity.NewDomainReadCloud(service.Domains)
    listed, err := cloud.ListDomains(ctx,
        identity.WithDomainReadFilters(json.RawMessage(`{"is_enabled":true,"description":"operations"}`)),
    )
    if err != nil { log.Fatal(err) }
    printJSON(listed.JSON)

    serverFiltered, err := cloud.SearchDomains(ctx,
        identity.WithDomainReadFilters(json.RawMessage(`{"name":"team"}`)),
    )
    if err != nil { log.Fatal(err) }
    printJSON(serverFiltered.Value)

    locallyFiltered, err := cloud.SearchDomains(ctx,
        request.Custom(func(c *identity.DomainReadOpts) error { v := "team-*"; c.NameOrID = &v; return nil }),
        identity.WithDomainReadFilters(json.RawMessage(`{"description":"operations"}`)),
    )
    if err != nil { log.Fatal(err) }
    printJSON(locallyFiltered.Value)
}
```

## References

- [Policy](https://docs.openstack.org/keystone/2025.2/configuration/policy.html)
- [Domain](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/domain.py)
- [Domains](https://github.com/openstack/keystone/blob/28.0.0/keystone/api/domains.py)
- [Python Record Mutations](python-record-mutations.md)
- [Python Domain Config](python-domain-config.md)
