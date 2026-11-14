# Credentials Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
created = conn.identity.create_credential(
    user_id=user_id, type="ec2", project_id=project_id,
    blob='{"access":"ACCESS","secret":"SECRET"}')
credentials = conn.identity.credentials(user_id=user_id, type="ec2")
found = conn.identity.find_credential(credential_id)
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
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials"
)

func printRecord(v *credentials.CredentialRecord) {
    if v == nil { fmt.Println("null"); return }
    data, err := json.MarshalIndent(v.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    user := flag.String("user", "", "credential owner user ID")
    project := flag.String("project", "", "credential project ID")
    kind := flag.String("type", "ec2", "credential blob type")
    id := flag.String("id", "", "credential ID to GET and find")
    blob := flag.String("blob", "{}", "credential blob string")
    create := flag.Bool("create", false, "create a generic Credential")
    remove := flag.String("delete", "", "credential ID to delete")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Credentials
    for v, err := range api.ListRecords(ctx,
        credentials.WithRecordListType(*kind),
        credentials.WithRecordListUserID(*user),
        request.MaxItems(100)) {
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    if *id != "" {
        v, err := api.GetRecord(ctx, *id)
        if err != nil { log.Fatal(err) }
        printRecord(v)
        found, err := api.FindRecord(ctx, *id)
        if err != nil { log.Fatal(err) }
        printRecord(found)
    }
    if *create {
        v, err := api.CreateRecord(ctx,
            request.Field("user_id", *user),
            request.Field("project_id", *project),
            request.Field("type", *kind),
            request.Field("blob", *blob))
        if err != nil { log.Fatal(err) }
        printRecord(v)
    }
    if *remove != "" {
        if err := api.DeleteRecord(ctx, *remove,
            request.IgnoreMissing(false)); err != nil { log.Fatal(err) }
    }
}
```

## References

- [Install](../../docs/install.md)
- [Python Records Test](credentials/python_records_test.go)
- [Native Contracts Test](credentials/native_contracts_test.go)
