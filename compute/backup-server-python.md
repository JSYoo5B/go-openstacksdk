# Server backup: Nova action, Glance lookup, optional wait

`conn.Compute(ctx).BackupServer` implements the pinned
`conn.compute.backup_server(server, name, backup_type, rotation, wait=False,
timeout=120)` workflow. The SDK owns the createBackup request, version choice,
Glance lookup, and wait defaults. Connection supplies the Glance dependency
lazily; applications do not implement a resolver or builder.

| Pinned openstacksdk | Go |
|---|---|
| `backup_server(server_id, "backup", "daily", 3)` | `service.BackupServer(ctx, serverID, compute.BackupServerRequest{Name: "backup", BackupType: "daily", Rotation: 3})` |
| `wait=True` | `compute.WithBackupServerWait(true)` |
| `timeout=120` | library default; override with `compute.WithBackupServerTimeout(120*time.Second)` |
| `timeout=None` runtime behavior | `compute.WithBackupServerUnlimitedWait()` |
| direct `Server.backup(...)` action only | `service.API.Servers.CreateBackupAction(ctx, serverID, request)` |

The full workflow always looks up the resulting Glance image, even with the
default `wait=false`. Waiting performs fresh cloud `get_image` lookups rather
than treating a cached active image as completion. Only exact lowercase
`"active"` succeeds and exact `"error"` fails; missing images during polling
continue until timeout. If the initial lookup returns no image, wait=false
returns nil Image and wait=true reports an invalid wait seed, corresponding to
the Source's attempt to pass None to `wait_for_image`.

```python
image = conn.compute.backup_server(
    server_id, "backup", "daily", 3, wait=True, timeout=120)
print(image.id, image.status)
```

## Runnable Go main

Prepare a module using the [installation guide](../docs/install.md), save this
as `main.go`, then run `go run . -cloud dev -id SERVER_ID -wait`. This submits
one server backup and looks up its image. `-wait` also waits for active status.
The example selects Compute 2.96, while the operation uses 2.45 when the actual
Nova advertisement supports it. No manually built Image service is required.

```go
package main

import (
    "context"
    "encoding/json"
    "flag"
    "log"
    "time"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute"
)

func main() {
    cloud := flag.String("cloud", "dev", "clouds.yaml cloud")
    id := flag.String("id", "", "literal server ID")
    name := flag.String("name", "backup", "backup image name")
    kind := flag.String("type", "daily", "backup type")
    rotation := flag.Int("rotation", 3, "backups retained by Nova")
    wait := flag.Bool("wait", false, "wait for Glance active status")
    timeout := flag.Duration("timeout", 120*time.Second, "image wait timeout")
    flag.Parse()
    if *id == "" { log.Fatal("-id is required") }
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    conn, err := sdk.Connect(ctx, sdk.WithCloud(*cloud), sdk.WithMicroversion(sdk.Compute, "2.96"))
    if err != nil { log.Fatal(err) }
    service, err := conn.Compute(ctx)
    if err != nil { log.Fatal(err) }
    result, err := service.BackupServer(ctx, *id,
        compute.BackupServerRequest{Name: *name, BackupType: *kind, Rotation: *rotation},
        compute.WithBackupServerWait(*wait), compute.WithBackupServerTimeout(*timeout))
    if result != nil && result.Action != nil {
        log.Printf("Nova action status=%d image_id=%q operation_version=%q",
            result.Action.StatusCode, result.Action.ImageID, result.Action.Microversion)
    }
    if err != nil { log.Fatal(err) }
    if result.Image == nil { log.Print("Glance image not found"); return }
    data, err := json.MarshalIndent(result.Image.Resource, "", "  ")
    if err != nil { log.Fatal(err) }
    log.Print(string(data))
}
```

## Version and response contract

The pinned Source `Server.backup` first asks whether Nova supports 2.45,
including the advertised min/max and selected default. This Go API reuses the
SDK's finite Nova version discovery: version endpoint then root when a clean
404/405 or eligible-row absence permits fallback. Discovery requests have no
microversion or body. Advertised links do not change the selected origin or
routing. The workflow retains discovery responses and the last clean unavailable
rejection separately from the action.

If 2.45 is supported, the action uses an operation-private 2.45 client. Otherwise
it uses the existing nonempty selected version; without a selected version it
chooses the Source minimum of the advertised maximum and Server's 2.100 ceiling,
subject to the advertised minimum. Missing bounds can leave the action
versionless. The shared client remains unchanged. Python may reuse cached
endpoint discovery; Go performs this finite discovery for each owned backup
operation.

POST `servers/{id}/action` sends:

```json
{"createBackup":{"name":"backup","backup_type":"daily","rotation":3}}
```

All three fields are always included, including empty strings and zero or
negative rotation. Nova validates their domain constraints. The action owns
`Accept: ""` and its selected microversion headers even when the original
client has different default Accept or version header aliases.

Like Source, image ID extraction first tries a nonempty parsed JSON object
with an `image_id` key at every version. Missing keys, empty bodies, and JSON
parse failures fall back to `Location.rsplit('/', 1)[1]`. The Location is never
followed; its final slash component becomes a literal ID for the separately
selected Glance service. An unrelated Location host does not receive a token.
Non-string, empty, dot, invalid UTF-8, or control-character IDs produce an
accepted-response error with the actual body/header/status. Malformed Python
runtime types are surfaced as Go validation errors at this boundary.

Owned action requests accept final 200–399. Rejected statuses preserve native
Gophercloud status/body/headers and physical body read/Close failures. Accepted
processing failures retain `resource.ResponseError`. Source changes, cancellation
and custom causes, retry failures, or attempts to replace the owned body/version
are terminal and do not start image lookup or repeat the accepted action.

## Options, partial results, and direct clients

`WithBackupServerWait` defaults false. The wait timeout defaults **120 seconds**
and the Cloud polling interval defaults two seconds. A zero or negative timeout
expires before the first wait lookup, after the unconditional initial lookup;
its value is unused when wait=false. `WithBackupServerPollInterval` is a positive
Go extension. Waiting reuses `image.Service.WaitForCloudImageRecord`, including
its missing-as-nil Find fallback, exact status comparison, full sleep interval,
and timeout checks between requests rather than cancelling an in-flight lookup.
Caller context cancellation still interrupts requests and waits.

`WithBackupServerActionOptions` forwards concrete action options exactly once.
`servers.WithBackupActionField` snapshots unknown JSON fields inside
createBackup; name, backup_type, and rotation cannot be replaced through JSON
extensions. `servers.WithBackupActionOptions` supplies an owned typed replacement.
`servers.WithBackupActionHeader` validates ordinary extra headers; authentication,
protocol headers, selected version, and the operation's Accept remain SDK owned.
Query and secondary argument extensions are unsupported.

`BackupServerResult.Action` contains the original Nova action Body/Header/StatusCode,
ImageID, operation Microversion, and discovery evidence. `Lookup` retains the
initial cloud lookup, `Wait` retains polling progress and last observation,
and `Image` is the final or latest known image. Action acceptance and actual
Glance observations remain separate. An image/dependency/timeout error returns
known progress without cleanup, backup deletion, or another createBackup POST.

Direct clients can use `compute.WithBackupImageService(imageService)` with an
existing concrete Glance service. Connection users need no override. The
selected Compute source is captured before options and Glance bindings stay
fixed across lookup/wait; callbacks or physical response handling cannot switch
services, clients, origins, or microversions midway through the workflow.

This named operation does not establish complete Python Resource/session
parity, every Nova action, live-cloud policy compatibility, or authenticated
acceptance validation.
