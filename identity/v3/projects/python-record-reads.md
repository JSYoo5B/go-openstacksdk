# Python Record Reads

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import json
import os
import openstack

conn = openstack.connect(cloud=os.environ["OS_CLOUD"])
domain = os.environ["OS_PROJECT_DOMAIN_ID"]

for project in conn.identity.projects(domain_id=domain, limit=2, max_items=2):
    print(json.dumps(project.to_dict(), indent=2))

project = conn.identity.find_project(
    "demo", domain_id=domain, skip_cache=True,
    error_message="Project lookup failed",
)
if project is not None:
    print(json.dumps(project.to_dict(), indent=2))

listed = conn.list_projects(domain_id=domain, filters={"is_enabled": True})
print(json.dumps([project.to_dict() for project in listed], indent=2))
searched = conn.search_projects("demo*", domain_id=domain,
                                filters={"is_enabled": True})
print(json.dumps([project.to_dict() for project in searched], indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(os.Getenv("OS_CLOUD")))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    domain := os.Getenv("OS_PROJECT_DOMAIN_ID")

    for project, err := range service.Projects.ListRecords(ctx,
        request.Query("domain_id", domain),
        request.Limit(2), request.MaxItems(2)) {
        if err != nil { log.Fatal(err) }
        printJSON(project.Resource)
    }
    project, err := service.Projects.FindRecord(ctx, "demo",
        request.Query("domain_id", domain),
        request.Query("skip_cache", true),
        request.Query("error_message", "Project lookup failed"))
    if err != nil { log.Fatal(err) }
    if project != nil { printJSON(project.Resource) }

    filters := identity.WithProjectReadFilters(json.RawMessage(`{"is_enabled":true}`))
    listed, err := conn.ListProjects(ctx,
        request.Custom(func(c *identity.ProjectReadOpts) error { c.DomainID = domain; return nil }), filters)
    if err != nil { log.Fatal(err) }
    printJSON(listed.JSON)
    searched, err := conn.SearchProjects(ctx, "demo*",
        request.Custom(func(c *identity.ProjectReadOpts) error { c.DomainID = domain; return nil }), filters)
    if err != nil { log.Fatal(err) }
    printJSON(searched.Value)
}
```

## References

- [openstacksdk ef55d7d](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/project.py)
- [Python Record Find](../users/python-record-find.md)
- [Source list_projects](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L82)
- [Project](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/project.py#L157-L174)
- [Base](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/base.py)
