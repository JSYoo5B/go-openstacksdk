# Python Server Migrations

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
parser.add_argument("--migration", default="")
parser.add_argument("--action", choices=("list", "get", "abort", "force", "crash"), required=True)
parser.add_argument("--microversion", default="2.80")
parser.add_argument("--strict-abort", action="store_true")
args = parser.parse_args()
if args.action in ("get", "abort", "force") and not args.migration:
    parser.error("--migration is required")
with openstack.connect(cloud=args.cloud, compute_api_version=args.microversion) as conn:
    if args.action == "list":
        for row in conn.compute.server_migrations(args.server):
            print(json.dumps(row.to_dict(), indent=2))
    elif args.action == "get":
        row = conn.compute.get_server_migration(args.migration, args.server)
        print(json.dumps(row.to_dict(), indent=2))
    elif args.action == "abort":
        conn.compute.abort_server_migration(args.migration, args.server,
                                            ignore_missing=not args.strict_abort)
    elif args.action == "force":
        conn.compute.force_complete_server_migration(args.migration, args.server)
    else:
        conn.compute.trigger_server_crash_dump(args.server)
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
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/servers"
)

func printJSON(value any) {
    data, err := json.MarshalIndent(value, "", "  ")
    if err != nil { log.Fatal(err) }
    fmt.Println(string(data))
}

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    server := flag.String("server", "", "parent server ID")
    migration := flag.String("migration", "", "migration ID")
    action := flag.String("action", "", "list, get, abort, force or crash")
    version := flag.String("microversion", "2.80", "explicit Nova version")
    strict := flag.Bool("strict-abort", false, "abort missing is an error")
    flag.Parse()
    if *server == "" { log.Fatal("server is required") }
    if (*action == "get" || *action == "abort" || *action == "force") && *migration == "" {
        log.Fatal("migration is required")
    }
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud), sdk.WithMicroversion(sdk.Compute, *version))
    if err != nil { log.Fatal(err) }
    service, err := conn.ComputeV2(ctx)
    if err != nil { log.Fatal(err) }
    api := service.Servers
    options := []servers.ServerMigrationOption{request.Microversion(*version)}
    switch *action {
    case "list":
        for row, err := range api.ListServerMigrationRecords(ctx, *server, options...) {
            if err != nil { log.Fatal(err) }
            printJSON(row.Resource)
        }
    case "get":
        row, err := api.GetServerMigrationRecord(ctx, *server, *migration, options...)
        if err != nil { log.Fatal(err) }
        printJSON(row.Resource)
    case "abort":
        options = append(options, request.IgnoreMissing(!*strict))
        result, err := api.AbortServerMigration(ctx, *server, *migration, options...)
        if err != nil { log.Fatal(err) }
        printJSON(result.Completed)
    case "force":
        result, err := api.ForceCompleteServerMigration(ctx, *server, *migration, options...)
        if err != nil { log.Fatal(err) }
        printJSON(result.Completed)
    case "crash":
        if err := api.TriggerServerCrashDump(ctx, *server); err != nil { log.Fatal(err) }
        fmt.Println("crash-dump request accepted")
    default:
        log.Fatal("action must be list, get, abort, force or crash")
    }
}
```

## References

- [Install](../../../docs/install.md)
