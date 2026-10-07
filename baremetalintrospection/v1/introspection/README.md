# Introspection 시작, 조회와 완료 대기

Introspection의 식별자는 bare metal node UUID와 같은 `Introspection.UUID`입니다. SDK는 Gophercloud v2.15.0의 `GetIntrospectionStatus`와 `ListIntrospections`를 공통 Collection에 연결합니다. 별도의 이름, 문자열 `Status`, Delete endpoint는 가정하지 않습니다.

| openstacksdk | Go |
|---|---|
| `conn.baremetal_introspection.get_introspection(uuid)` | `service.Introspection.Resources.Get(ctx, uuid)` |
| `conn.baremetal_introspection.introspections()` | `service.Introspection.Resources.List(ctx)` |
| `conn.baremetal_introspection.start_introspection(uuid, manage_boot=False)` | `service.Introspection.StartIntrospection(ctx, uuid, introspection.StartOpts{ManageBoot: &manageBoot})` (`manageBoot := false`) |
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

`resource`는 `github.com/JSYoo5B/gophercloudsdk/resource`, `time`과 `fmt`는 표준 라이브러리입니다. 실제 node UUID를 입력해야 합니다. UUID처럼 보이는 문자열로 이름 조회를 추측하지 않습니다. Node 이름을 사용하려면 먼저 Bare Metal 서비스의 `Nodes.Resources.ResolveID(ctx, resource.Name(name))`로 UUID를 구한 뒤 Introspection 서비스에 전달합니다. `resource.Name`을 Introspection Collection과 waiter에 직접 전달하면 요청 전에 `resource.ErrUnsupported`를 반환합니다.

공통 `Get/Find/ResolveID/List/All`을 사용할 수 있습니다. Find는 ID 미존재를 `resource.ErrNotFound`로 반환하며 `resource.WithIgnoreMissing()`으로 `nil, nil`을 선택합니다. 목록은 UUID와 `Finished`, `State`, `Error`, 시간을 보존하고 lazy page 순회와 `break`, context 취소를 적용합니다. 생성기에서 조회/목록 모델, UUID 타입, Finished bool과 Error string의 일치를 검사합니다.

## WaitUntilFinished의 정책

Waiter는 공통 `resource.WaitOption`을 사용합니다. 기본 timeout은 5분, poll 간격은 2초입니다. `WithUnlimitedWait()`도 부모 context의 취소와 deadline을 보존합니다. `WithProgressCallback`은 모델에 progress가 없어 비종료 응답마다 0을 받습니다. `WithStatusAttribute`는 `Finished`라는 완료 조건을 바꾸므로 이 전용 waiter에서는 거부합니다. `WithFailureStates()`로도 조회 중 확인한 서비스 Error를 무시할 수 없습니다. 옵션은 요청 전에 검사하며 뒤에 지정한 값이 우선합니다.

입력 UUID를 한 번 해석한 뒤 같은 UUID를 계속 조회합니다. 후속 응답에 UUID가 없거나 다른 UUID가 있더라도 다른 URL로 조회를 바꾸지 않습니다. 반환 모델의 UUID는 서버 응답 그대로 보존합니다. 이미 `Finished=true`이면 첫 응답에서 즉시 반환합니다.

- `Error`가 비어 있지 않으면 `Finished`와 관계없이 즉시 실패합니다.
- `State=error`인데 메시지가 비어 있어도 실패합니다.
- 실패는 `resource.ErrFailedState`와 `errors.Is`로 확인하고, `*introspection.IntrospectionFailureError`와 `errors.As`로 입력 ID, 실제 메시지, state, 마지막 typed 응답을 얻습니다.
- HTTP 404는 미존재, 403과 통신·decode 오류는 원래 오류를 보존합니다. 오류를 완료 상태로 바꾸거나 조회를 무한 재시도하지 않습니다.

`introspection`은 이 패키지입니다. 오류의 `Details.Error`는 원래 응답 문자열입니다. SDK가 error state에 보충한 설명은 `Message`에 있으므로 원문과 구분할 수 있습니다.

Python의 pinned [Introspection.wait](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/baremetal_introspection/v1/introspection.py)은 기본 timeout을 지정하지 않고 `ignore_error`를 선택할 수 있습니다. Go는 유한 timeout을 기본값으로 제공하며 실패 무시 옵션은 현재 제공하지 않습니다. 또한 Python은 error state를 검사하지만 Go는 state와 무관하게 비어 있지 않은 Error도 실패로 처리합니다.

`Resources.Wait`는 없는 native Status를 가정하지 않으므로 `resource.ErrUnsupported`입니다. 완료 대기는 `WaitUntilFinished`를 사용합니다. `Resources.Delete`도 미지원이며 진행 중 작업의 취소는 별도 `AbortIntrospection` 연산입니다.

## Start 요청의 선택 인자

`StartOpts.ManageBoot`의 nil은 query를 생략해 서버 기본값을 사용합니다. `&false`와 `&true`는 각각 `manage_boot=false`와 `manage_boot=true`를 전달합니다. Python의 `manage_boot=None/False/True`와 같은 구분입니다.

```go
// 위와 같이 service를 얻은 뒤
manageBoot := false
err = service.Introspection.StartIntrospection(ctx, "node-uuid",
    introspection.StartOpts{ManageBoot: &manageBoot},
    introspection.WithStartIntrospectionQuery("vendor", "a&b"))
if err != nil { return err }
```

SDK는 typed 옵션과 확장 query를 함께 URL에 넣고 값을 인코딩합니다. `WithStartIntrospectionOptions`는 기본 typed 옵션을 교체합니다. 다른 query 확장과 같이 `WithStartIntrospectionQuery`는 같은 query key의 앞선 값을 덮어씁니다. 따라서 `manage_boot`를 확장 query로 지정하면 typed 값보다 우선합니다. JSON 본문·헤더 확장과 잘못된 옵션은 요청 전에 거부합니다.

Python의 시작 연산은 Node 또는 문자열을 받고 Introspection Resource를 반환합니다. Go의 `StartIntrospection`은 node UUID를 받고 error만 반환합니다. 서버 상태는 `GetIntrospectionStatus`나 `WaitUntilFinished`로 조회합니다.

고정 [Gophercloud StartIntrospection](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/baremetalintrospection/v1/introspection/requests.go)은 serializer가 반환한 query를 URL에 넣지 않습니다. SDK의 생성 API는 이 연산만 [private helper](start.go)에 연결해 누락을 보정합니다. POST 본문을 추가하지 않으며 202만 성공으로 처리합니다. HTTP 오류의 status·본문·header·URL과 context 취소 원인은 보존합니다. 시작 성공은 작업 완료를 뜻하지 않으므로 완료 확인에는 `WaitUntilFinished`를 사용합니다.

이 예외는 [감사된 요청 규칙](../../../internal/cmd/sdkgen/audited_requests.go)에 등록합니다. 생성기는 pinned 함수 선언의 SHA-256, context/client/nodeID/builder signature, `ManageBoot *bool`과 query tag, `StartResult` 타입을 검사하며 drift가 있으면 재검토를 요구하는 오류로 중단합니다. [연산 목록](../../../api/gophercloud_inventory.json)의 이 연산에는 `request_policy: sdk_query_preserving_start`가 기록됩니다. SDK를 거치지 않고 native 함수를 직접 호출하면 이 보정은 적용되지 않습니다.

구현은 [공통 binding](resources_generated.go), [waiter](wait.go), [start helper](start.go)에 있습니다. 검증은 [조회·완료 대기 HTTP 테스트](../../../api/introspection_contracts_test.go), [Start HTTP 테스트](../../../api/introspection_start_contracts_test.go), [serializer·header 테스트](start_test.go), [생성 규칙 테스트](../../../internal/cmd/sdkgen/audited_requests_test.go)에 있습니다. [전체 Introspection 서비스](../README.md)와 [SDK 지원 판정 기준](../../../docs/sdk-support-ledger.md)도 참고하세요.
