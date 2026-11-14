# Application Credentials Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
credentials = conn.identity.application_credentials(user_id, name="ci", max_items=100)
created = conn.identity.create_application_credential(
    user_id, "ci", description="", unrestricted=False, roles=[{"name": "member"}])
found = conn.identity.find_application_credential(user_id, "ci")
rule = conn.identity.get_access_rule(user_id, rule_id)
# conn.identity.delete_application_credential(user_id, created.id)
# conn.identity.delete_access_rule(user_id, rule.id, ignore_missing=False)
```

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/applicationcredentials"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func printRecord(v *applicationcredentials.ApplicationCredentialRecord) {
    if v == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(v.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    user := flag.String("user", "", "literal Keystone user ID")
    name := flag.String("name", "ci", "exact credential name or ID")
    credential := flag.String("credential", "", "credential ID to GET")
    rule := flag.String("rule", "", "access rule ID to GET")
    create := flag.Bool("create", false, "create an application credential")
    deleteCredential := flag.String("delete-credential", "", "credential ID to delete")
    deleteRule := flag.String("delete-rule", "", "access rule ID to delete")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    api := service.ApplicationCredentials
    scope, err := api.InUser(ctx, resource.ID(*user))
    if err != nil { log.Fatal(err) }
    rules, err := api.AccessRules(ctx, resource.ID(*user))
    if err != nil { log.Fatal(err) }
    for v, err := range scope.ListRecords(ctx,
        applicationcredentials.WithRecordListFilter("name", *name),
        request.MaxItems(100)) {
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    found, err := scope.FindRecord(ctx, *name)
    if err != nil { log.Fatal(err) }
    printRecord(found)
    if *credential != "" {
        v, err := scope.GetRecord(ctx, *credential)
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    for v, err := range rules.ListRecords(ctx) {
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    if *rule != "" {
        v, err := rules.GetRecord(ctx, *rule)
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    if *create {
        v, err := scope.CreateRecord(ctx, *name,
            request.Field("description", ""),
            request.Field("unrestricted", false),
            request.Field("roles", []map[string]string{{"name": "member"}}))
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    if *deleteCredential != "" {
        if err := scope.DeleteRecord(ctx, *deleteCredential); err != nil { log.Fatal(err) }
    }
    if *deleteRule != "" {
        if err := rules.DeleteRecord(ctx, *deleteRule,
            request.IgnoreMissing(false)); err != nil { log.Fatal(err) }
    }
}
```

## References

- [Install](../../docs/install.md)
- [Python Records Test](applicationcredentials/python_records_test.go)
