# Accelerator v2 (Cyborg)

이 서비스는 Gophercloud에 없는 Python openstacksdk 서비스이므로 SDK가 모델과 transport를 직접 소유합니다. `Connection.Accelerator(ctx)`와 `AcceleratorV2(ctx)`는 인증·HTTP 설정을 공유하고 lazy cache를 사용합니다. endpoint는 `/v2` 유무에 관계없이 한 번만 정규화합니다. Cyborg microversion은 `WithMicroversion`, `WithMicroversionRange`, `WithLatestMicroversion`으로 선택합니다.

```go
accel, err := conn.Accelerator(ctx)
if err != nil { return err }
for device, err := range accel.Devices.List(ctx, resource.WithQuery("hostname", "compute-1")) {
    if err != nil { return err }
    fmt.Println(device.UUID, device.Type, device.Status)
}
deployable, err := accel.Deployables.Find(ctx, resource.Name("compute-1_FPGA"))
if err != nil { return err }
fmt.Println(deployable.UUID)
```

| Python openstacksdk | Go SDK |
|---|---|
| `conn.accelerator.devices(**query)` | `accel.Devices.List(ctx, resource.WithQuery(...))` |
| `get_device(uuid)` | `accel.Devices.Get(ctx, uuid)` |
| `deployables(**query)` | `accel.Deployables.List(ctx, ...)` |
| `get_deployable(uuid)` | `accel.Deployables.Get(ctx, uuid)` |
| `enable_device(uuid)` / `disable_device(uuid)` | `accel.Devices.Enable(ctx, resource.ID(uuid))` / `Disable(...)` |
| `patch_deployable(uuid, patch)` | `accel.Deployables.Patch(ctx, resource.ID(uuid), []deployables.ProgramOperation{...})` |
| programming a bitstream image | `accel.Deployables.Program(ctx, resource.ID(uuid), imageUUID)` |
| exact deployable name lookup | `accel.Deployables.Find(ctx, resource.Name(name))` |
| `device_profiles()` / `get_device_profile(uuid)` | `accel.DeviceProfiles.List(ctx)` / `Get(ctx, uuid)` |
| `create_device_profile(**attrs)` | `accel.DeviceProfiles.Create(ctx, deviceprofiles.CreateOpts{...}, ...)` |
| `delete_device_profile(uuid)` | `accel.DeviceProfiles.Delete(ctx, resource.ID(uuid))` |
| `accelerator_requests()` / `get_accelerator_request(uuid)` | `accel.AcceleratorRequests.List(ctx)` / `Get(ctx, uuid)` |
| `create_accelerator_request(device_profile_name=...)` | `accel.AcceleratorRequests.Create(ctx, acceleratorrequests.CreateOpts{...})` → 전체 batch |
| `patch_accelerator_request(uuid, patch)` | `accel.AcceleratorRequests.Patch(ctx, resource.ID(uuid), patch)`; `Bind`/`Unbind`가 문서 구성 |
| `delete_accelerator_request(uuid)` | `accel.AcceleratorRequests.Delete(ctx, resource.ID(uuid))` |
| `attributes()` / `get_attribute(uuid)` / `create_attribute(**attrs)` | `accel.Attributes.List(ctx)` / `Get(ctx, uuid)` / `Create(ctx, attributes.CreateOpts{...})` |
| `wait_for_status(device, status)` | `accel.Devices.Wait(ctx, resource.ID(uuid), status, ...)` |

Device identity is its UUID, never a database numeric ID or hostname. Deployable name lookup checks all pages and rejects duplicates. Devices and deployables have no DELETE API. Deployables have no status waiter. Those unsupported Collection methods return `resource.ErrUnsupported`.

Responses retain full object JSON in `Body`, cloned HTTP headers in `Header`, and HTTP `StatusCode`. Omitted, null, unknown fields and non-RFC3339 timestamp strings remain available. List follows explicit `links`, `<collection>_links` or `next` with loop detection and continues through an empty page with a next link. It rejects both next links and HTTP redirects that change the service origin; it does not invent marker pagination for controllers that return no continuation link. Breaking an iterator stops further requests. The redirect boundary keeps the caller's HTTP policy, uses the latest locked shared token at each actual HTTP attempt (including later pages and retries), waits for ongoing shared reauthentication with caller cancellation, delegates 401 reauthentication to the original provider, and preserves configured backoff/retry hooks. The original HTTP client remains unchanged. Treat service client microversion/header configuration as immutable after construction.

`WithStatus` is a local filter for Device status and ARQ state; it does not send an unsupported `status` query. Explicit `WithQuery` values still reach the server for validation. Caller-supplied `limit`/`marker` and `WithPageSize` return `ErrUnsupported` for the pinned controllers; server-provided continuation links still work.

[Device profiles](deviceprofiles/README.md) provide exact name lookup, single-element-array creation and UUID-only deletion. [Attributes](attributes/README.md) use UUIDs; their key is not a unique name and their creation uses a flat object. [ARQs](acceleratorrequests/README.md) retain the whole creation batch, build collection-level binding patches, and distinguish ARQ/instance deletion selectors. They do not infer refreshed state from an empty 202 response. Accepted creation with malformed data preserves metadata, raw response and every valid batch member alongside the error.

Enable/disable send an empty POST body and return response metadata rather than an unchanged Device instance. Their controller has no microversion gate. Device status fetch/list and status wait require a selected microversion of at least 2.3; at older/default 2.0, status filtering and wait return `resource.ErrUnsupported` before HTTP. [Version history](https://docs.openstack.org/cyborg/latest/contributor/rest_api_version_history.html) distinguishes this response-field change from the actions. Errors preserve the HTTP code, headers and body. Programming sends one `replace /program` operation with one image to `/deployables/{uuid}/program`; `Program` constructs this patch. `Patch` accepts the controller's concrete program document, not arbitrary RFC6902 edits. Headers use `WithActionHeader`/`WithProgramHeader`, while authentication, content headers and microversion remain protected. Inputs are serialized before name lookup so caller slices are not mutated or read after that HTTP boundary.

The pinned Python deployable patch override appends the resource ID to an already expanded `/deployables/{id}/program` path, and its deprecated `update_deployable` recursively calls itself. Go uses the documented program URL and does not reproduce those defects. An empty or malformed patch is rejected before HTTP. Actions and programming do not add a re-fetch or a retry loop. Configured native provider reauthentication/backoff/retry policies still apply, including to mutations; use an explicit Device waiter for refreshed status.

Python's deprecated `fields` argument on getters is a no-op; Go omits it. Common waits use a finite five-minute default and two-second interval, configured by `resource.WithTimeout` and `WithPollInterval`; Python's infinite wait, callback, configurable failure set and arbitrary status attribute remain separate work. The five resources have service-specific capabilities; this README does not claim complete Accelerator parity or Python Resource dirty-state behavior.

Sources: [Cyborg API reference](https://docs.openstack.org/api-ref/accelerator/), [Cyborg controller revision dfa0b80](https://github.com/openstack/cyborg/tree/dfa0b80ca7060a799db191f73c32b5d9bd577103/cyborg/api/controllers/v2), and the [pinned Python inventory](../../api/openstacksdk/accelerator/v2.json).
