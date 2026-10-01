# Accelerator requests (ARQs)

`accel.AcceleratorRequests`는 Cyborg 요청 UUID와 `state`를 사용하는 Collection입니다. device profile 이름은 여러 ARQ에 반복될 수 있으므로 `resource.Name`으로 조회하지 않습니다. `Get`, `List`, `All`, `Wait`, `Delete`, `WaitDeleted`는 공통 오류·취소·pagination 정책을 따릅니다.

```go
accel, err := conn.Accelerator(ctx)
if err != nil { return err }
created, err := accel.AcceleratorRequests.Create(ctx,
    acceleratorrequests.CreateOpts{DeviceProfileName: "fpga-profile"},
    acceleratorrequests.WithCreateHeader("X-Custom", "example"))
if err != nil { return err }
for _, arq := range created.Requests {
    fmt.Println(arq.UUID, arq.State, arq.DeviceProfileGroupID)
}
```

| Python openstacksdk | Go SDK |
|---|---|
| `accelerator_requests(**query)` | `AcceleratorRequests.List(ctx, resource.WithQuery(...))` |
| `get_accelerator_request(uuid)` | `AcceleratorRequests.Get(ctx, uuid)` |
| `create_accelerator_request(device_profile_name=...)` | `Create(ctx, CreateOpts{DeviceProfileName: ...})` → 모든 ARQ |
| `patch_accelerator_request(uuid, patch)` | `Patch(ctx, resource.ID(uuid), []BindingOperation{...})` |
| 바인딩 필드 patch 작성 | `Bind(ctx, resource.ID(uuid), BindOpts{...})` / `Unbind(...)` |
| `delete_accelerator_request(uuid, ignore_missing=True)` | `Delete(ctx, resource.ID(uuid))` |
| 여러 UUID / instance selector 삭제 | `DeleteMany(ctx, []string{...})` / `DeleteByInstance(ctx, instanceUUID)` |
| `wait_for_status(arq, "Bound", attribute="state")` | `Wait(ctx, resource.ID(uuid), "Bound", ...)` |

생성은 flat JSON `{"device_profile_name": ...}`을 POST하고 201의 `arqs` 배열 전체를 `CreateResponse.Requests`로 반환합니다. 빈 배열도 보존합니다. 고정한 Python `AcceleratorRequest._consume_attrs`는 첫 항목만 소비하지만 Go는 나머지 생성 결과를 버리지 않습니다. 응답 전체는 `RawBody`와 `Body`, 각 리소스는 자기 `Body`에 보존하며 HTTP 헤더는 서로 독립된 복사본입니다. 성공한 HTTP 응답의 JSON 또는 항목 해석이 실패하면 생성 응답·정상 해석한 항목·원문을 오류와 함께 반환합니다. 이때 자동으로 다시 생성하거나 정리하지 않습니다. provider에 설정된 인증·HTTP 재시도 정책은 그대로 적용됩니다.

바인딩 PATCH의 실제 경로는 `/accelerator_requests`입니다. UUID는 URL에 추가하지 않고 JSON 객체의 키로 전달합니다. 고정한 Python Resource의 일반 단건 request 준비와 이 controller 계약의 차이를 Go에서 바로잡습니다. `PatchMany`로 같은 방식의 add/remove batch를 보내며, `Bind`는 hostname·device RP UUID·instance UUID 필드를 구성합니다. `/project_id`를 지정하려면 microversion 2.1 이상이 필요합니다. `remove`는 unbind 연산입니다. 이 API는 임의 리소스 필드 수정이나 일반 RFC6902 patch를 제공하지 않습니다. 202 응답은 빈 본문이므로 상태를 추측하지 않고 HTTP metadata를 반환합니다. 갱신된 상태는 명시적으로 `Wait`합니다.

서버가 service token 또는 관리자 권한을 요구하는 요청은 `WithBindingHeader("X-Service-Token", token)` / `WithDeleteHeader(...)`를 사용할 수 있습니다. SDK가 권한을 추정하거나 token을 만들어 넣지 않습니다. HTTP 403 등은 원본 code·header·body·cause를 유지합니다. extension header로 인증·content·microversion을 덮어쓸 수 없습니다. 생성 `WithCreateOptions`, `WithCreateField`, `WithCreateHeader`는 concrete 입력과 추가 필드를 지원하며 core 필드 충돌은 HTTP 전에 오류입니다. binding과 delete는 header 확장만 허용합니다.

삭제는 `/accelerator_requests?arqs=uuid` 또는 `?instance=uuid`에서 204를 받습니다. 단건 `Delete`는 기본적으로 404를 무시하며 `resource.WithMissingError()`로 바꿀 수 있습니다. `DeleteMany`는 누락된 한 항목 때문에 나머지 삭제가 생략되는 것을 숨기지 않도록 404를 그대로 반환합니다. `DeleteByInstance`의 반복 삭제는 controller의 idempotent 계약을 사용합니다. 단건 ID에 쉼표를 허용하지 않으며, batch는 명시적인 UUID slice로만 구성합니다. bound ARQ 삭제에는 서버 정책에 따른 service token이 필요할 수 있습니다.

`Wait`는 `state`를 비교하고 `ERROR`·`BindFailed`를 실패로 처리합니다. 기본 제한 시간 5분·간격 2초는 `WithTimeout`·`WithPollInterval`로 설정합니다. Python의 무한 대기·callback·임의 status attribute는 아직 제공하지 않습니다. `WithStatus`는 서버가 지원하지 않는 `status` query를 전송하지 않고 로컬에서 비교합니다. native controller의 list filter는 `bind_state=resolved`, `instance=...`이며 `resource.WithQuery`로 명시합니다. 호출자가 설정하는 `WithPageSize`·`marker`는 controller가 지원하지 않아 `ErrUnsupported`입니다. 서버가 명시한 continuation link는 그대로 따릅니다.

근거: [Cyborg API reference](https://docs.openstack.org/api-ref/accelerator/), [고정 controller](https://github.com/openstack/cyborg/blob/dfa0b80ca7060a799db191f73c32b5d9bd577103/cyborg/api/controllers/v2/arqs.py), [고정 Python Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/accelerator/v2/accelerator_request.py), [전체 서비스](../README.md).
