# Trusts Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
created = conn.identity.create_trust(
    trustor_user_id=trustor_id,
    trustee_user_id=trustee_id,
    project_id=project_id,
    is_impersonation=False,
    roles=[{"name": "member"}],
)
for trust in conn.identity.trusts(trustor_user_id=trustor_id):
    print(trust.id, trust.is_impersonation)
found = conn.identity.find_trust(created.id, ignore_missing=False)
conn.identity.delete_trust(created.id, ignore_missing=False)
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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/trusts"
)

func printTrust(value *trusts.TrustRecord) {
    if value == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(value.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    trustor := flag.String("trustor", "", "trustor user ID")
    trustee := flag.String("trustee", "", "trustee user ID for creation")
    project := flag.String("project", "", "project ID for creation")
    role := flag.String("role", "member", "role name for creation")
    id := flag.String("id", "", "Trust ID to get and find")
    create := flag.Bool("create", false, "create a Trust")
    remove := flag.String("delete", "", "Trust ID to delete")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Trusts
    options := []trusts.RecordListOption{request.MaxItems(100)}
    if *trustor != "" {
        options = append(options, trusts.WithRecordListTrustorUserID(*trustor))
    }
    for value, err := range api.ListRecords(ctx, options...) {
        if err != nil { log.Fatal(err) }
        printTrust(value)
    }
    if *id != "" {
        value, err := api.GetRecord(ctx, *id)
        if err != nil { log.Fatal(err) }
        printTrust(value)
        found, err := api.FindRecord(ctx, *id, request.IgnoreMissing(false))
        if err != nil { log.Fatal(err) }
        printTrust(found)
    }
    if *create {
        value, err := api.CreateRecord(ctx,
            request.Field("trustor_user_id", *trustor),
            request.Field("trustee_user_id", *trustee),
            request.Field("project_id", *project),
            request.Field("is_impersonation", false),
            request.Field("roles", []map[string]string{{"name": *role}}))
        if err != nil { log.Fatal(err) }
        printTrust(value)
    }
    if *remove != "" {
        if err := api.DeleteRecord(ctx, *remove,
            request.IgnoreMissing(false)); err != nil { log.Fatal(err) }
    }
}
```

## References

- [Install](../../docs/install.md)
- [Python Records Test](trusts/python_records_test.go)
