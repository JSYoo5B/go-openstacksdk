# Introspection 조회와 완료 대기

Introspection의 식별자는 bare metal node UUID와 같은 `Introspection.UUID`입니다. SDK는 Gophercloud v2.15.0의 `GetIntrospectionStatus`와 `ListIntrospections`를 공통 Collection에 연결합니다. 별도의 이름, 문자열 `Status`, Delete endpoint는 가정하지 않습니다.

| openstacksdk | Go |
|---|---|
| `conn.baremetal_introspection.get_introspection(uuid)` | `service.Introspection.Resources.Get(ctx, uuid)` |
| `conn.baremetal_introspection.introspections()` | `service.Introspection.Resources.List(ctx)` |
| `conn.baremetal_introspection.wait_for_introspection(record)` | `service.Introspection.WaitUntilFinished(ctx, resource.ID(uuid))` |

```go
// context.Context ctx, *gophercloudsdk.Connection conn을 사용하는 함수 안에서
service, err := conn.BareMetalIntrospectionV1(ctx)
if err != nil { return err }
value, err := service.Introspection.WaitUntilFinished(ctx, resource.ID("node-uuid"),
    resource.WithTimeout(10*time.Minute),
    resource.WithPollInterval(3*time.Second))
if err != nil { return err }
fmt.Println(value.UUID, value.Finished, value.State)
```

`resource`는 `gophercloudsdk/resource`, `time`과 `fmt`는 표준 라이브러리입니다. 실제 node UUID를 입력해야 합니다. UUID처럼 보이는 문자열로 이름 조회를 추측하지 않습니다. Node 이름을 사용하려면 먼저 Bare Metal 서비스의 `Nodes.Resources.ResolveID(ctx, resource.Name(name))`로 UUID를 구한 뒤 Introspection 서비스에 전달합니다. `resource.Name`을 Introspection Collection과 waiter에 직접 전달하면 요청 전에 `resource.ErrUnsupported`를 반환합니다.

공통 `Get/Find/ResolveID/List/All`을 사용할 수 있습니다. Find는 ID 미존재를 `resource.ErrNotFound`로 반환하며 `resource.WithIgnoreMissing()`으로 `nil, nil`을 선택합니다. 목록은 UUID와 `Finished`, `State`, `Error`, 시간을 보존하고 lazy page 순회와 `break`, context 취소를 적용합니다. 생성기에서 조회/목록 모델, UUID 타입, Finished bool과 Error string의 일치를 검사합니다.

## WaitUntilFinished의 정책

Waiter는 공통 `resource.WaitOption`을 그대로 사용합니다. 기본 timeout은 5분, poll 간격은 2초입니다. 옵션은 요청 전에 검사하며 뒤에 지정한 값이 우선합니다. 부모 context의 취소와 더 짧은 deadline을 보존합니다.

입력 UUID를 한 번 해석한 뒤 같은 UUID를 계속 조회합니다. 후속 응답에 UUID가 없거나 다른 UUID가 있더라도 다른 URL로 조회를 바꾸지 않습니다. 반환 모델의 UUID는 서버 응답 그대로 보존합니다. 이미 `Finished=true`이면 첫 응답에서 즉시 반환합니다.

- `Error`가 비어 있지 않으면 `Finished`와 관계없이 즉시 실패합니다.
- `State=error`인데 메시지가 비어 있어도 실패합니다.
- 실패는 `resource.ErrFailedState`와 `errors.Is`로 확인하고, `*introspection.IntrospectionFailureError`와 `errors.As`로 입력 ID, 실제 메시지, state, 마지막 typed 응답을 얻습니다.
- HTTP 404는 미존재, 403과 통신·decode 오류는 원래 오류를 보존합니다. 오류를 완료 상태로 바꾸거나 조회를 무한 재시도하지 않습니다.

`introspection`은 이 패키지입니다. 오류의 `Details.Error`는 원래 응답 문자열입니다. SDK가 error state에 보충한 설명은 `Message`에 있으므로 원문과 구분할 수 있습니다.

Python의 pinned [Introspection.wait](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/baremetal_introspection/v1/introspection.py)은 기본 timeout을 지정하지 않고 `ignore_error`를 선택할 수 있습니다. Go는 유한 timeout을 기본값으로 제공하며 실패 무시 옵션은 현재 제공하지 않습니다. 또한 Python은 error state를 검사하지만 Go는 state와 무관하게 비어 있지 않은 Error도 실패로 처리합니다.

`Resources.Wait`는 없는 native Status를 가정하지 않으므로 `resource.ErrUnsupported`입니다. 완료 대기는 `WaitUntilFinished`를 사용합니다. `Resources.Delete`도 미지원이며 진행 중 작업의 취소는 별도 `AbortIntrospection` 연산입니다.

## 확인한 요청 오류

고정 Gophercloud의 `StartIntrospection`은 `ToStartIntrospectionQuery()`를 검사하지만 반환 query를 POST URL에 추가하지 않습니다. 따라서 현재 생성 API에서도 `StartOpts.ManageBoot` 및 `WithStartIntrospectionQuery`가 서버에 전달되지 않습니다. 서버 기본값과 같은 동작이며, 이 옵션을 지원한다고 판단해서는 안 됩니다. SDK의 query-preserving start helper를 별도 수정 단위로 추가할 대상입니다. 이 waiter는 조회 연산만 사용하므로 이 오류의 영향을 받지 않습니다.

구현은 [공통 binding](resources_generated.go)과 [waiter](wait.go), 검증은 [HTTP 계약 테스트](../../../api/introspection_contracts_test.go)에 있습니다. [전체 Introspection 서비스](../README.md)와 [SDK 지원 판정 기준](../../../docs/sdk-support-ledger.md)도 참고하세요.
