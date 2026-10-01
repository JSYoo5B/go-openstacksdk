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
| exact deployable name lookup | `accel.Deployables.Find(ctx, resource.Name(name))` |
| `wait_for_status(device, status)` | `accel.Devices.Wait(ctx, resource.ID(uuid), status, ...)` |

Device identity is its UUID, never a database numeric ID or hostname. Deployable name lookup checks all pages and rejects duplicates. Devices and deployables have no DELETE API. Deployables have no status waiter. Those unsupported Collection methods return `resource.ErrUnsupported`.

Responses retain full object JSON in `Body`, cloned HTTP headers in `Header`, and HTTP `StatusCode`. Omitted, null, unknown fields and non-RFC3339 timestamp strings remain available. List follows explicit `links`, `<collection>_links` or `next` with loop detection and rejects a next link that changes the service origin; it does not invent marker pagination for controllers that return no continuation link. Breaking an iterator stops further requests.

Python's deprecated `fields` argument on getters is a no-op; Go omits it. Common waits use a finite five-minute default and two-second interval, configured by `resource.WithTimeout` and `WithPollInterval`; Python's infinite wait, callback and arbitrary status attribute are not implemented yet. Device actions, deployable programming, device profiles, ARQs and attributes are the next implementation units; this README does not claim complete Accelerator parity.

Sources: [Cyborg API reference](https://docs.openstack.org/api-ref/accelerator/), [Cyborg controller revision dfa0b80](https://github.com/openstack/cyborg/tree/dfa0b80ca7060a799db191f73c32b5d9bd577103/cyborg/api/controllers/v2), and the [pinned Python inventory](../../api/openstacksdk/accelerator/v2.json).
