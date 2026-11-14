# Console Auth Token

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import sys
import openstack

conn = openstack.connect()
console = conn.compute.validate_console_auth_token(sys.argv[1])
print(console.to_dict())
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
)

func main() {
    token := flag.String("console-token", "", "console token to look up")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *token); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, token string) error {
    if token == "" {
        return fmt.Errorf("-console-token is required")
    }
    conn, err := sdk.Connect(ctx)
    if err != nil {
        return err
    }
    service, err := conn.Compute(ctx)
    if err != nil {
        return err
    }
    record, err := service.ValidateConsoleAuthToken(ctx, token)
    if record != nil {
        data, marshalErr := json.MarshalIndent(map[string]any{
            "resource": record.Resource,
            "wire": record.Wire,
            "status_code": record.StatusCode,
            "header": record.Header,
        }, "", "  ")
        if marshalErr != nil {
            return marshalErr
        }
        fmt.Println(string(data))
    }
    return err
}
```

## References

- [Policy](https://docs.openstack.org/nova/latest/configuration/policy.html)
- [Console Selection](console-selection.md)
- [SDK Support Ledger](../docs/sdk-support-ledger.md)
- [Implementation Plan](../docs/implementation-plan.md)
