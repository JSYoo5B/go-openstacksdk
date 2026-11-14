# Server Find Python

Python openstacksdk mappings, contracts and differences for these operations are recorded in the review ledger, `api/sdk_reviews.json`.

## Usage examples

Examples use the surrounding guide's service object and context unless they declare a complete program. Python examples describe the pinned comparison API.

```python
import argparse
import json
import openstack

parser = argparse.ArgumentParser()
parser.add_argument("--cloud", default="dev")
parser.add_argument("--server", required=True)
parser.add_argument("--basic", action="store_true")
parser.add_argument("--all-projects", action="store_true")
parser.add_argument("--strict", action="store_true")
args = parser.parse_args()

conn = openstack.connect(cloud=args.cloud)
server = conn.compute.find_server(
    args.server,
    ignore_missing=not args.strict,
    details=not args.basic,
    all_projects=args.all_projects,
)
print(json.dumps(None if server is None else server.to_dict(), indent=2))
```

```go
package main

import (
    "context"
    "encoding/json"
    "errors"
    "flag"
    "fmt"
    "log"
    "time"

    "github.com/JSYoo5B/go-openstacksdk/request"
    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    identity := flag.String("server", "", "exact server ID or name")
    basic := flag.Bool("basic", false, "use basic fallback list")
    allProjects := flag.Bool("all-projects", false, "request cross-project lookup")
    strict := flag.Bool("strict", false, "return an error for missing server")
    version := flag.String("microversion", "", "optional selected service microversion")
    flag.Parse()
    if *identity == "" { log.Fatal("-server is required") }
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    connectionOptions := []sdk.ConnectionOption{sdk.WithCloud(*cloud)}
    if *version != "" {
        connectionOptions = append(connectionOptions,
            sdk.WithMicroversion(sdk.Compute, *version))
    }
    conn, err := sdk.Connect(ctx, connectionOptions...)
    if err != nil { log.Fatal(err) }
    service, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    record, err := service.Servers.FindRecord(ctx, *identity,
        request.IgnoreMissing(!*strict),
        request.Custom(func(c *servers.RecordFindOpts) error { v := !*basic; c.Details = &v; return nil }),
        request.Custom(func(c *servers.RecordFindOpts) error { v := *allProjects; c.AllProjects = &v; return nil }))
    if errors.Is(err, resource.ErrNotFound) { log.Fatal("server not found") }
    if errors.Is(err, resource.ErrAmbiguous) { log.Fatal("server name is ambiguous") }
    if err != nil { log.Fatal(err) }
    if record == nil { fmt.Println("null"); return }
    body, err := json.MarshalIndent(record.Resource.Body, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(body))
    fmt.Printf("HTTP %d microversion=%s request-id=%s\n",
        record.StatusCode, record.Microversion,
        record.Header.Get("X-Openstack-Request-Id"))
}
```

## References

- [Install](../../docs/install.md)
