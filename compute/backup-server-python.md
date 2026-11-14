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
| `timeout=None` runtime behavior | `compute.WithBackupServerWaitOptions(resource.WithUnlimitedWait())` |
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

See `Example_backupServer` in [example_backup_server_test.go](example_backup_server_test.go), compiled by `go test`.

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
its value is unused when wait=false. `WithBackupServerWaitOptions` (or
`BackupServerOpts.WaitOptions`) takes the shared resource wait options:
`resource.WithTimeout` or `resource.WithUnlimitedWait` replaces the timeout, and
`resource.WithPollInterval` or `resource.WithWaitStrategy` replaces the fixed
interval as a Go extension; repeated options append and the later one wins per
control. With wait=true, an invalid option, failure states, a status attribute or
progress fail with `resource.ErrInvalidOption` before the backup action is sent. Waiting reuses `image.Service.WaitForCloudImageRecord`, including
its missing-as-nil Find fallback, exact status comparison, full sleep interval,
and timeout checks between requests rather than cancelling an in-flight lookup.
Caller context cancellation still interrupts requests and waits.

`WithBackupServerActionOptions` forwards concrete action options exactly once.
`request.Field` snapshots unknown JSON fields inside
createBackup; name, backup_type, and rotation cannot be replaced through JSON
extensions. `servers.WithBackupActionOptions` supplies an owned typed replacement.
`request.Header` validates ordinary extra headers; authentication,
protocol headers, selected version, and the operation's Accept remain SDK owned.
Query and secondary argument extensions are unsupported.

`BackupServerResult.Action` contains the original Nova action Body/Header/StatusCode,
ImageID, operation Microversion, and discovery evidence. `Lookup` retains the
initial cloud lookup, `Wait` retains polling progress and last observation,
and `Image` is the final or latest known image. Action acceptance and actual
Glance observations remain separate. An image/dependency/timeout error returns
known progress without cleanup, backup deletion, or another createBackup POST.

Direct clients can set `compute.BackupServerOpts.ImageService` to an
existing concrete Glance service. Connection users need no override. The
selected Compute source is captured before options and Glance bindings stay
fixed across lookup/wait; callbacks or physical response handling cannot switch
services, clients, origins, or microversions midway through the workflow.

This named operation does not establish complete Python Resource/session
parity, every Nova action, live-cloud policy compatibility, or authenticated
acceptance validation.
