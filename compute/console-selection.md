# Console Selection

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import openstack

conn = openstack.connect()
console = conn.compute.create_console("server-id", "novnc")
print(console)
```

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
    "github.com/JSYoo5B/go-openstacksdk/request"
)

func main() {
    serverID := flag.String("server-id", "", "server ID")
    kind := flag.String("console-type", "novnc", "console type")
    protocol := flag.String("protocol", "", "optional console protocol")
    flag.Parse()
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := run(ctx, *serverID, *kind, *protocol); err != nil {
        log.Fatal(err)
    }
}

func run(ctx context.Context, serverID, kind, protocol string) error {
    if serverID == "" {
        return fmt.Errorf("-server-id is required")
    }
    conn, err := sdk.Connect(ctx)
    if err != nil {
        return err
    }
    service, err := conn.Compute(ctx)
    if err != nil {
        return err
    }
    var options []compute.ConsoleOption
    flag.Visit(func(f *flag.Flag) {
        if f.Name == "protocol" {
            options = append(options, compute.ConsoleOption(func(c *request.Config[compute.ConsoleOpts]) error { c.Options.Protocol = request.Present(protocol); return nil }))
        }
    })
    console, err := service.CreateConsole(ctx, serverID, kind, options...)
    if err != nil {
        return err
    }
    fmt.Println(string(console))
    return nil
}
```

## References

- [User Actions](user-actions.md)
- [Keypairs Console](keypairs-console.md)
