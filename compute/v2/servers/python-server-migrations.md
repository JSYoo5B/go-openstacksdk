# Python Server Migrations

This guide describes the public Go APIs and their contracts against the pinned Gophercloud and Python openstacksdk sources. Support decisions apply to each operation below.

See the [support ledger](../../../docs/sdk-support-ledger.md) for status definitions and shared limitations.

## Operation contracts

### `python:compute/v2/server_migrations`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ListServerMigrationRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.AllServerMigrationRecords`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationIgnoreMissing`

- Lazy server-scoped migration iteration uses the bound parent collection, untyped nullable descriptor/default projection, absent-id uuid alternate identity and actual physical page evidence; unknown Body/server_id values are not allowed to alter routing or Resource attributes. Complete advertised pages are consumed while early consumer break avoids further requests.
  Tests: `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationListIsLazyAndKeepsUntypedProjection`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationListFollowsCompletePagesAndHonorsEarlyBreak`.
- Malformed accepted list payloads preserve actual response proof and selected source/version/header snapshots and preflight remain owned.
  Tests: `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationListMalformedAcceptedResponseKeepsActualProof`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationOptionsAreOwnedAcrossReusableCalls`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationPreflightAndOptionGuardsStopRequests`.

Python/Go differences:

- Go takes explicit parent server/migration string IDs in safe escaped segments instead of Python Resource objects with inferred URI parents. Owned immutable record snapshots and actual response channels replace mutable cached Resource/session state. Runtime object/formatter injection, caller custom transport cache bypass and full SDK-R1/C1/S1 or live-cloud/release parity are not claimed.
- For migration operations, explicit selected client/per-call versions take precedence without class-maximum capping. Only an empty selection triggers owned Nova discovery and endpoint minimum/maximum selection with ceiling2.80, retaining actual Discovery receipts. This bounded JSON discovery profile is distinct from Python arbitrary adapter/session cache state; class maximum is not an action-local feature minimum.

### `python:compute/v2/get_server_migration`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.GetServerMigrationRecord`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationIgnoreMissing`

- Member GET escapes parent and migration IDs independently, overlays passive actual response id on constructor seed and returns declared view plus physical response evidence. Source ignore_missing is a constructor-only discarded attr, so GET404 remains strict rather than missing suppression.
  Tests: `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationGetOwnsParentURIAndPassiveResponseID`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationGet404IsStrictForBothMissingFlags`.
- Copied options and selected source/header/version guards surround callbacks; terminal physical faults preserve causes instead of GET fallback or replay.
  Tests: `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationOptionsAreOwnedAcrossReusableCalls`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationPreflightAndOptionGuardsStopRequests`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationPhysicalFaultsAreTerminalAndNeverReplay`.

Python/Go differences:

- Go takes explicit parent server/migration string IDs in safe escaped segments instead of Python Resource objects with inferred URI parents. Owned immutable record snapshots and actual response channels replace mutable cached Resource/session state. Runtime object/formatter injection, caller custom transport cache bypass and full SDK-R1/C1/S1 or live-cloud/release parity are not claimed.
- For migration operations, explicit selected client/per-call versions take precedence without class-maximum capping. Only an empty selection triggers owned Nova discovery and endpoint minimum/maximum selection with ceiling2.80, retaining actual Discovery receipts. This bounded JSON discovery profile is distinct from Python arbitrary adapter/session cache state; class maximum is not an action-local feature minimum.

### `python:compute/v2/abort_server_migration`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.AbortServerMigration`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationIgnoreMissing`

- Abort sends bodyless DELETE to the selected migration member, defaults to ignoring clean404 and permits strict missing. Completed/IgnoredMissing and actual Response receipt distinguish accepted completion from ignored absence without fabricating a Resource.
  Tests: `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationAbortMissingDefaultAndStrictPolicies`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationActionsOwnLiteralForceCompleteAndReceipts`.
- Physical read/Close/cancel failures remain terminal with actual errors and are not ignored as clean absence or replayed.
  Tests: `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationPhysicalFaultsAreTerminalAndNeverReplay`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationPreflightAndOptionGuardsStopRequests`.

Python/Go differences:

- Go takes explicit parent server/migration string IDs in safe escaped segments instead of Python Resource objects with inferred URI parents. Owned immutable record snapshots and actual response channels replace mutable cached Resource/session state. Runtime object/formatter injection, caller custom transport cache bypass and full SDK-R1/C1/S1 or live-cloud/release parity are not claimed.
- For migration operations, explicit selected client/per-call versions take precedence without class-maximum capping. Only an empty selection triggers owned Nova discovery and endpoint minimum/maximum selection with ceiling2.80, retaining actual Discovery receipts. This bounded JSON discovery profile is distinct from Python arbitrary adapter/session cache state; class maximum is not an action-local feature minimum.
- Python abort returns None; Go returns an action carrier with actual receipt. Completed=false with an error does not prove the server did not execute the request.

### `python:compute/v2/force_complete_server_migration`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.ForceCompleteServerMigration`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationOpts`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationHeaders`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationMicroversion`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithServerMigrationIgnoreMissing`

- Force-complete posts exactly {"force_complete":null} to the selected migration member action without resource-body decoding; non-JSON successful response remains actual receipt and the API does not wait for migration completion.
  Tests: `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationActionsOwnLiteralForceCompleteAndReceipts`.
- Physical action failures do not replay; copied options/source guards reject invalid route/context/callback state before requests.
  Tests: `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationPhysicalFaultsAreTerminalAndNeverReplay`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationOptionsAreOwnedAcrossReusableCalls`, `compute/v2/servers/python_server_migrations_test.go:TestPythonServerMigrationPreflightAndOptionGuardsStopRequests`.

Python/Go differences:

- Go takes explicit parent server/migration string IDs in safe escaped segments instead of Python Resource objects with inferred URI parents. Owned immutable record snapshots and actual response channels replace mutable cached Resource/session state. Runtime object/formatter injection, caller custom transport cache bypass and full SDK-R1/C1/S1 or live-cloud/release parity are not claimed.
- For migration operations, explicit selected client/per-call versions take precedence without class-maximum capping. Only an empty selection triggers owned Nova discovery and endpoint minimum/maximum selection with ceiling2.80, retaining actual Discovery receipts. This bounded JSON discovery profile is distinct from Python arbitrary adapter/session cache state; class maximum is not an action-local feature minimum.
- Python force_complete returns None; Go exposes a Completed/Response carrier for accepted request evidence. This is not migration lifecycle completion or automatic cleanup.

### `python:compute/v2/trigger_server_crash_dump`

Status: `go_mapping`.

Go API:

- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.API.TriggerServerCrashDump`
- `github.com/JSYoo5B/go-openstacksdk/compute/v2/servers.WithUserActionHeader`

- TriggerServerCrashDump posts exactly {"trigger_crash_dump":null} to the escaped server action with owned empty Accept and selected version, leaving the shared client unchanged. Like the pinned Source method it adds no action-local2.17 rejection gate.
  Tests: `compute/v2/servers/python_crash_dump_test.go:TestPythonTriggerServerCrashDumpOwnsPayloadRouteAndSelectedVersion`.
- Options/source/cancellation guards precede HTTP, actual rejected statuses preserve native operation proof and physical read/Close/cancel failures never replay. Success is request acceptance without guest dump/reboot waiting.
  Tests: `compute/v2/servers/python_crash_dump_test.go:TestPythonTriggerServerCrashDumpOptionsAndSourceGuardsPrecedeHTTP`, `compute/v2/servers/python_crash_dump_test.go:TestPythonTriggerServerCrashDumpStatusesKeepNativeOperationProof`, `compute/v2/servers/python_crash_dump_test.go:TestPythonTriggerServerCrashDumpPhysicalFaultNeverReplays`.

Python/Go differences:

- Go takes explicit parent server/migration string IDs in safe escaped segments instead of Python Resource objects with inferred URI parents. Owned immutable record snapshots and actual response channels replace mutable cached Resource/session state. Runtime object/formatter injection, caller custom transport cache bypass and full SDK-R1/C1/S1 or live-cloud/release parity are not claimed.
- The Go API uses the selected client/per-call Nova version rather than Source default-first/endpoint-discovered class maximum negotiation. ServerMigration class maximum2.80 and Server maximum2.100 are not action-local API minimum gates. A selected Source session default can exceed the class maximum and is not capped; Go does not claim automatic negotiation.
- Nova API minimum2.17 is distinct from pinned trigger_crash_dump local code, which only delegates _action. The existing Go version validator checks selected Nova-version syntax/range, leaves action availability to the server and exposes error-only success rather than a fabricated receipt.

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
    options := []servers.ServerMigrationOption{servers.WithServerMigrationMicroversion(*version)}
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
        options = append(options, servers.WithServerMigrationIgnoreMissing(!*strict))
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
