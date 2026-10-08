# Masakari 상태·삭제 대기

`Notifications`, `Segments`, 고정 segment의 `Hosts`, 고정 notification의 `VMoves`는
`WaitForStatus`와 `WaitForDelete`를 제공합니다. SDK가 UUID 식별자, 부모 범위와 polling을
관리하며 호출자가 상태 adapter나 builder를 구현할 필요가 없습니다.

| 항목 | pinned openstacksdk | Go |
|---|---|---|
| 상태 대기 | `conn.instance_ha.wait_for_status(res, status, ...)` | `API.WaitForStatus(ctx, ref, status, options...)` |
| 실패 기본값 | `['ERROR']` | `ERROR`, 대소문자 무시 |
| 상태 timeout | `wait=None` | SDK 제한 없음, caller context 적용 |
| 간격 | 2초 | 2초, `WithPollInterval`로 변경 |
| 삭제 대기 | `conn.instance_ha.wait_for_delete(res, ...)` | `API.WaitForDelete(ctx, ref, options...)` |
| 삭제 timeout | 120초 | 120초, context와 caller 옵션 적용 |
| 반환값 | 원래 Resource | 상태는 typed 모델, 삭제는 `error` |

`Resources.Wait`와 `Resources.WaitDeleted`는 기존 공통 5분 정책을 유지합니다.
`WaitForStatus`의 실패 목록은 service binding의 기존 실패 상태를 교체합니다.
`FAILED`를 기본 실패로 추측하지 않으므로 필요하면 `WithFailureStates("ERROR", "FAILED")`를
지정합니다. `WithFailureStates()`는 실패 상태 판정을 끕니다. 목표 상태가 실패 목록에도
있으면 목표 도달을 먼저 판정합니다.

| API | 기본 상태 속성 | 기본 삭제 종결 |
|---|---|---|
| `Notifications` | `status` | 404 또는 `status='deleted'` |
| `VMoves.InNotification(...)` | `status`, numeric microversion >=1.3 | 404 또는 `status='deleted'` |
| `Segments` | 없음, 기본 상태 대기는 HTTP 전에 `ErrUnsupported` | 404 |
| `Hosts.InSegment(...)` | 없음, 기본 상태 대기는 HTTP 전에 `ErrUnsupported` | 404 |

상태가 없는 모델은 `WithStatusAttribute("name")`처럼 exported 문자열 필드의 JSON 이름이나
Go 이름을 선택할 수 있습니다. boolean, raw JSON, unknown Body 필드와 중첩 속성은 지원하지
않습니다. 기본 삭제 대기는 상태가 없는 모델의 incidental raw `status`를 사용하지 않습니다.
삭제 waiter 자체는 GET만 수행하므로 DELETE API가 없는 Notification과 VMove에도 사용할 수
있습니다. 별도의 삭제 요청을 만들어내거나 Location을 follow하지 않습니다.

```python
notification = conn.instance_ha.get_notification("NOTIFICATION_UUID")
notification = conn.instance_ha.wait_for_status(
    notification, "finished", failures=["ERROR", "FAILED"], wait=180,
)
```

```go
// context.Context ctx, *openstack.Connection conn을 사용하는 함수 안에서
service, err := conn.InstanceHAV1(ctx)
if err != nil { return err }
notification, err := service.Notifications.WaitForStatus(ctx,
    resource.ID("22222222-2222-4222-8222-222222222222"), "finished",
    resource.WithFailureStates("ERROR", "FAILED"),
    resource.WithTimeout(3*time.Minute))
if err != nil { return err }
fmt.Println(notification.UUID, notification.Status, notification.Header, notification.Body)
```

Go의 route identity는 database `id`와 별개인 `notification_uuid` 또는 `uuid`입니다.
응답의 database ID나 다른 UUID가 polling route를 바꾸지 않습니다. VMove의 부모는 scope
생성 시 고정하고 각 GET 직전에 microversion 1.3을 재검사합니다.

```go
service, err := conn.InstanceHAV1(ctx)
if err != nil { return err }
scope, err := service.VMoves.InNotification(ctx,
    resource.ID("22222222-2222-4222-8222-222222222222"))
if err != nil { return err }
move, err := scope.WaitForStatus(ctx,
    resource.ID("55555555-5555-4555-8555-555555555555"), "succeeded",
    resource.WithTimeout(5*time.Minute))
if err != nil { return err }
fmt.Println(move.NotificationID, move.UUID, move.Status)
```

```go
service, err := conn.InstanceHAV1(ctx)
if err != nil { return err }
ref := resource.ID("11111111-1111-4111-8111-111111111111")
if err := service.Segments.Delete(ctx, ref); err != nil { return err }
if err := service.Segments.WaitForDelete(ctx, ref,
    resource.WithTimeout(10*time.Minute),
    resource.WithPollInterval(time.Second)); err != nil { return err }
```

예제에서 `resource`는 `github.com/JSYoo5B/go-openstacksdk/resource`, `fmt`와 `time`은 표준 라이브러리입니다.
뒤의 옵션이 SDK 기본값을 덮어씁니다. `WithUnlimitedWait()`는 SDK timeout을 없애며 parent
context의 deadline과 취소는 항상 적용합니다. `WithProgressCallback`은 비종결 조회에서
동기 호출하고 context 취소가 발생하면 다음 GET을 수행하지 않습니다. 현재 네 모델에는
top-level 정수 `progress`가 없어 callback 값은 0입니다. notification의 nested workflow
progress를 하나의 percentage로 합성하지 않습니다.

Python은 cached Resource가 이미 목표 상태이면 HTTP 없이 같은 객체를 반환하고 후속 fetch로
그 객체를 갱신합니다. Go는 explicit Ref로 fresh 조회하며 기존 모델을 변경하지 않습니다.
Name은 Segment와 Host에서 exact lookup을 한 번 수행한 뒤 UUID를 고정합니다. Notification과
VMove의 이름 검색은 지원하지 않습니다. Python의 nullable 목표·속성과 interval None/0,
임의 Resource 입력 및 삭제 시 원래 객체 반환은 별도 차이입니다.

Go는 HTTP에도 context deadline을 전달합니다. 403·409·transport 오류는 원래 오류를 유지하고,
잘못된 성공 envelope/JSON은 Body·Header·StatusCode를 가진 `resource.ResponseError`를 반환합니다.
404는 상태 대기에서 오류, 삭제 대기에서는 성공입니다. 응답 해석 오류가 자동 재전송을
유발하지 않습니다. 이 비교는 cached Resource lifecycle 전체 구현 완료를 뜻하지 않습니다.

비교 기준은 pinned
[Masakari proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/instance_ha/v1/_proxy.py#L335)와
[Resource wait helpers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591)입니다.
[HTTP 계약 테스트](../../../api/instanceha_wait_contracts_test.go)는 네 facade의 실제 경로,
상태·삭제 종결, 기본 deadline과 override, 버전 재검사 및 오류 증거를 확인합니다.
