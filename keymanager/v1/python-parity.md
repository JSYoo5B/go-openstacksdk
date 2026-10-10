# openstacksdk key manager v1 Proxy의 수정과 대기 메서드

고정한 openstacksdk 커밋 `ef55d7d`의 [key manager v1 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py)에서 조회, 생성, 삭제, ACL, consumer, quota, secret store 메서드는 이미 [Metadata 조회 비교](metadata-fetch.md), [생성 비교](metadata-create.md), [삭제 비교](metadata-delete.md)와 각 패키지 README에서 다룹니다. 이 문서는 남은 다섯 메서드인 container·order·secret 수정과 공통 대기 메서드를 비교합니다. 다섯 메서드 모두 Go에 없는 동작이 남아 있어 `unresolved`이고, Go에서 확인한 부분은 `orders`와 `secrets` 패키지의 `python_parity_test.go`가 고정합니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| [`update_container`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L142) | 없음 | unresolved |
| [`update_order`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L243) | 없음 | unresolved |
| [`update_secret`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L345) | `secrets.API.Update`는 payload 업로드라서 다른 요청 | unresolved |
| [`wait_for_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L615) | container, order, secret의 `WaitFor`만 있음 | unresolved |
| [`wait_for_delete`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L654) | container, order, secret의 `WaitForDeletion`만 있음 | unresolved |

## 수정

세 수정 메서드는 모두 `Proxy._update`를 거쳐 [Resource.commit](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1881)을 부릅니다. Container, Order, Secret은 `commit_method`를 바꾸지 않았고 resource key도 없어서, Python은 `PUT /containers/{id}`, `PUT /orders/{id}`, `PUT /secrets/{id}`에 변경된 속성을 감싸지 않은 JSON 객체로 보냅니다. ID 문자열로 새 Resource를 만들기 때문에 이 객체에는 `id`도 들어가고, 400 미만 status는 모두 성공으로 받아 응답 본문이 있으면 Resource에 합칩니다.

이 SDK의 `containers`와 `orders` 패키지에는 PUT을 보내는 호출이 없습니다. `secrets.API.Update`는 Barbican의 payload 추가 API를 그대로 감싸서 `UpdateOpts.Payload`를 원문 본문으로 보내고 `ContentType`, `ContentEncoding`을 header로 보냅니다. 성공 status도 204 하나뿐이고 오류만 반환합니다.

```go
// Python update_secret("s1", payload="new", payload_content_type="text/plain")은
// JSON 객체를 PUT하지만, 이 호출은 "new"를 text/plain 본문으로 PUT합니다.
err := secrets.New(client).Update(ctx, "s1", secrets.UpdateOpts{ContentType: "text/plain", Payload: "new"})
```

payload를 올리려는 목적이라면 실제 Barbican API에 맞는 호출은 `secrets.API.Update`입니다. 다만 이 호출은 Python이 보내는 요청과 반환 Resource를 재현하지 않습니다.

## 대기

`wait_for_status`와 `wait_for_delete`는 [resource.wait_for_status](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591)와 [resource.wait_for_delete](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2670)를 그대로 부릅니다. container, order, secret이라면 Go의 `WaitFor`, `WaitForDeletion`이 같은 ID로 GET을 반복하므로 Python 기본값을 옵션으로 맞출 수 있습니다.

```go
order, err := orders.New(client).WaitFor(ctx, resource.ID("o1"), "ACTIVE",
	resource.WithUnlimitedWait(),          // Python wait=None
	resource.WithFailureStates("ERROR"),  // Python failures=None
)
err = orders.New(client).WaitForDeletion(ctx, resource.ID("o1"),
	resource.WithTimeout(120*time.Second), // key manager Proxy의 wait=120
)
```

Python과 Go의 기본값 차이는 다음과 같습니다.

- Go 대기는 기본 제한 시간이 5분이고, Python `wait_for_status`는 제한이 없으며 `wait_for_delete`는 120초입니다. `resource.WithUnlimitedWait()`나 `resource.WithTimeout`으로 맞춥니다.
- Go 기본 실패 판정은 `error`로 시작하거나 `fail`, `failed`로 끝나거나 `killed`인 상태입니다. Python은 `failures=['ERROR']`만 대소문자 구분 없이 비교하므로 `resource.WithFailureStates("ERROR")`를 주면 같아집니다. 목표 상태 비교가 실패 비교보다 먼저인 순서는 같습니다.
- 간격은 둘 다 2초입니다. Python은 `interval=0`을 0.1초로 바꾸지만 Go `resource.WithPollInterval`은 양수만 받습니다.
- `attribute`는 `resource.WithStatusAttribute`, `callback`은 `resource.WithProgressCallback`입니다. Barbican 모델에는 진행률 필드가 없어 둘 다 0을 전달합니다.
- 제한 시간이 지나면 Python은 `ResourceTimeout`, Go는 `context.DeadlineExceeded`를 감싼 오류를 돌려주고, 실패 상태는 Python `ResourceFailure` 대신 `resource.FailedStateError`입니다.

## 남은 부분

- `update_container`, `update_order`: 변경된 속성 JSON을 `PUT /containers/{id}`, `PUT /orders/{id}`로 보내는 Go API가 없습니다.
- `update_secret`: Go `secrets.API.Update`는 원문 payload와 Content-Type header를 보내고 204만 받습니다. Python처럼 속성 JSON 객체(`id` 포함)를 보내고 400 미만 응답을 갱신된 Secret으로 돌려주는 호출이 없습니다.
- `wait_for_status`: Python은 key manager의 어떤 Resource든 받지만 Go 대기는 container, order, secret에만 있고 secret store, ACL, consumer, quota용 대기가 없습니다. Python Secret은 직접 구현한 `Secret.fetch`를 쓰기 때문에 대기 중에도 payload GET을 함께 보낼 수 있는데 Go `secrets.API.WaitFor`는 metadata GET만 보냅니다. 이미 받은 Resource의 상태가 목표와 같으면 Python은 HTTP 없이 돌려주지만 Go는 항상 첫 GET을 보내고 모델 pointer를 돌려줍니다.
- `wait_for_delete`: 같은 이유로 container, order, secret 외의 Resource를 기다릴 수 없고, Python이 돌려주는 원래 Resource 대신 Go는 오류만 돌려줍니다.
