# Credential Update Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect(cloud="dev")
updated = conn.identity.update_credential(
    credential_id,
    blob='{"access":"example","secret":"replace-me"}',
)
print(updated.id, updated.blob)
```

```go
package main

import (
	"github.com/JSYoo5B/go-openstacksdk/request"
    "context"
    "encoding/json"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/identity/v3/credentials"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    id := flag.String("id", "", "generic Credential ID")
    blob := flag.String("blob", "", "replacement credential blob string")
    clearProject := flag.Bool("clear-project", false, "send project_id=null")
    flag.Parse()
    if *id == "" { log.Fatal("-id is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud))
    if err != nil { log.Fatal(err) }
    service, err := conn.IdentityV3(ctx)
    if err != nil { log.Fatal(err) }
    options := []credentials.RecordUpdateOption{
        request.Field("blob", *blob),
    }
    if *clearProject {
        options = append(options, request.Field("project_id", nil))
    }
    updated, err := service.Credentials.UpdateRecord(ctx, *id, options...)
    if err != nil { log.Fatal(err) }
    data, err := json.MarshalIndent(updated.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}
```

## References

- [Credentials Python](credentials-python.md)
- [Install](../../docs/install.md)
- [Python Update Test](credentials/python_update_test.go)
