# Glance Task 대기와 396 재생성

`API.WaitForTask(ctx, resource.ID(id), options...)`는 fresh GET으로 시작해 `success`를 기다립니다. 기본 실패 상태는 `failure`, 전체 시간 제한은 120초, 조회 간격은 2초입니다. 다른 목표는 `WaitForTaskState(ctx, ref, target, options...)`로 지정합니다. Task는 Name 참조를 지원하지 않습니다.

```go
package example

import (
    "context"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/gophercloudsdk/image/v2/tasks"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func waitForTask(ctx context.Context, client *gophercloud.ServiceClient, id string) (*tasks.TaskWaitResult, error) {
    return tasks.New(client).WaitForTask(ctx, resource.ID(id))
}
```

Python의 대응 호출은 이미 보유한 Task Resource를 받습니다.

```python
task = conn.image.wait_for_task(task)  # success, failure, 120초, 2초
```

## 정확한 396 흐름

조회한 상태가 선택된 실패 상태이고 `message`가 정확히 `Image cannot be imported. Error code: '396'`이면 SDK가 조회한 `type`과 원본 `input`만으로 새 Task를 만듭니다. 다른 실패 메시지나 HTTP 오류를 이 흐름으로 재시도하지 않습니다. 성공 목표를 먼저 비교하므로 목표를 `failure`로 지정한 경우 그 상태를 성공으로 반환합니다.

새 Task의 ID가 다음 조회 대상이 됩니다. 생성 응답이 `success`라고 해도 다음 GET으로 확인하며, 최초 GET·재생성 POST·새 ID의 GET·대기 간격에 하나의 시간 예산을 적용합니다. 재생성 횟수마다 120초를 다시 시작하지 않습니다. `Self`, `Schema`, `Location`에서 ID를 추론하거나 그 URL을 따라가지 않습니다.

원본 `input`의 큰 JSON 정수·중첩 값은 typed 모델에서 다시 만들어 보내지 않습니다. 조회 응답의 canonical lowercase `type`과 `input`을 보존한 값으로 POST합니다. 이 workflow의 반환 모델도 `input`/`result` 숫자를 `json.Number`로 읽습니다. 기존 native `Get`/`Create`의 map 숫자 해석은 변경하지 않습니다. 재생성이 필요한 응답에는 nonempty `type`과 canonical `input`이 있어야 합니다. 명시적인 `input:null`과 `{}`는 구분해 보존하며, 누락된 input을 cached 값으로 채우지 않습니다.

## 호출별 정책과 재사용

```go
package example

import (
    "context"
    "time"

    "github.com/gophercloud/gophercloud/v2"
    "github.com/JSYoo5B/gophercloudsdk/image/v2/tasks"
    "github.com/JSYoo5B/gophercloudsdk/resource"
)

func waitForProcessing(ctx context.Context, client *gophercloud.ServiceClient, id string) (*tasks.TaskWaitResult, error) {
    timeout := time.Minute
    interval := 500 * time.Millisecond
    policy := tasks.WithTaskWaitOpts(tasks.TaskWaitOpts{
        Timeout: &timeout,
        PollInterval: &interval,
        FailureStates: []string{},
    })
    return tasks.New(client).WaitForTaskState(ctx, resource.ID(id), "processing", policy)
}
```

이 예제는 실패 목록을 비워 상태에 따른 실패·396 재생성을 끕니다. HTTP·응답 해석·context 오류는 여전히 반환합니다. 대응 Python 호출은 다음과 같습니다.

```python
task = conn.image.wait_for_task(
    task, status="processing", failures=[], interval=0.5, wait=60)
```

`TaskWaitOpts.Timeout == nil`은 기본 120초이고, 0을 가리키면 SDK 시간 제한이 없습니다. `WithUnlimitedTaskWait()`도 제한을 제거합니다. parent context의 취소·deadline은 항상 적용됩니다. `PollInterval == nil`은 2초이고 명시적인 간격은 양수여야 합니다. `FailureStates == nil`은 기본 `failure`, 비어 있는 non-nil slice는 실패 판정을 끕니다.

`WithTaskWaitTimeout`, `WithTaskWaitPollInterval`, `WithTaskWaitFailureStates`, `WithTaskWaitHeader` / `WithTaskWaitHeaders`로 설정할 수 있습니다. `WithTaskWaitTimeout`은 양수만 허용하고 무제한은 `WithUnlimitedTaskWait`로 지정합니다. 옵션은 순서대로 적용하며 마지막 값이 유효합니다. `WithTaskWaitOpts`는 pointer·slice·header를 snapshot하고 전체 정책을 교체하므로 동일한 옵션을 독립 호출에서 재사용할 수 있습니다. 헤더는 모든 단계에서 snapshot한 값으로 사용하며 auth·transport·version 헤더는 SDK가 소유합니다. 기존 Provider를 공유해 live token을 사용하고 선택한 Task collection 경로를 고정합니다. 공유 ServiceClient를 수정하지 않습니다. 이 namespace는 `resource.WaitOption`과 별개이며 progress callback·임의 상태 속성·query·base path 제어를 제공하지 않습니다.

## 실제 응답과 부분 실패

`TaskWaitResult`의 `Task`, `Body`, `Header`, `StatusCode`는 마지막으로 성공적으로 읽고 해석한 실제 GET 또는 POST 응답입니다. `OriginalID`는 최초 요청 ID, `CurrentID`는 현재 고정 조회 대상, `Recreated`는 성공적으로 이어진 재생성 횟수입니다. 응답의 우연한 GET ID 변경은 요청 대상을 바꾸지 않습니다.

`Created *TaskWaitResponse`는 마지막으로 받아들인 POST 201의 실제 `Task`·본문·헤더·상태를 별도로 보존합니다. POST의 읽기·decode·새 ID 검증에 실패한 경우에도 생성 응답 증거와 이전 성공 응답을 구분합니다. `result, err`를 함께 확인하면 실패 직전의 실제 Task와 생성 증거를 볼 수 있습니다. 응답이 없던 단계의 모델이나 HTTP 상태를 합성하지 않습니다.

생성 실패, 일반 실패 상태, 시간 초과·취소, 전송 오류와 받아들인 응답의 읽기·해석 오류는 terminal입니다. workflow 자체는 오류에 대한 추가 재생성·재전송을 하지 않고 404를 완료로 처리하지 않습니다. 기존 Provider의 configured retry·reauth는 같은 시간 예산과 고정 method/URL 아래 유지됩니다. 받아들인 응답의 읽기·decode 실패는 이 workflow의 재전송 대상이 아닙니다. 생성 응답은 다음 GET 경로로 사용할 수 있는 canonical ID가 필요합니다. 서로 다른 대소문자 키나 `Location`은 이 ID의 대체 값으로 쓰지 않습니다.

## 원본 SDK와 서버 계약

기준은 openstacksdk `ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe`의 [`wait_for_task`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L2198-L2269)입니다. Python은 supplied Task가 이미 목표 상태이면 HTTP 없이 같은 객체를 반환합니다. Go는 명시적인 ID의 fresh GET으로 시작하며 mutable Resource cache·seed merge·session injection을 받지 않습니다. Python fetch의 선택적 cache와 negotiated microversion도 전체 재현하지 않습니다.

Python Task의 input/result·시간·상태는 일반 Resource descriptor이고, Go의 Task는 native map·string·`time.Time` 모델입니다. 잘못된 typed 필드·timestamp·UTF-8 본문은 상태 판정 전에 오류가 될 수 있으며 받아들인 원본 응답 증거를 보존합니다. 원본 본문은 별도로 보존하지만 Python의 nullable/default/coercion 전체를 재현하지 않습니다. Python의 0 간격 내부 보정과 0/음수 시간 제한 정책도 같다고 가정하지 않습니다.

[공식 Task API](https://docs.openstack.org/api-ref/image/v2/index.html#tasks)는 flat GET 200과 flat POST 201을 정의합니다. input 내용과 사용 가능한 Task 유형은 cloud의 schema·정책에 따릅니다. schema 예제는 실제 `/v2/schemas/task` 응답의 대체가 아닙니다. Task API 접근 권한을 SDK가 확대하지 않습니다.

기존 typed `Get`, `Create`, `List`, `Task` alias와 generic `Resources.Wait` / `WaitFor`는 그대로 유지합니다. 일반 [서비스 상태·삭제 대기](../../../docs/service-waits.md)와 이 changing-ID workflow는 각각 별도 정책을 사용합니다. Task의 Find·Delete·progress·Resource 전체 지원을 추가하지 않습니다.

실제 계약은 [HTTP workflow 테스트](../../../api/glance_task_wait_contracts_test.go), [옵션 snapshot 테스트](wait_options_test.go), [응답 증거·숫자 테스트](wait_results_test.go), [생성기 binding 경계 테스트](../../../internal/cmd/sdkgen/glance_task_wait_test.go)로 확인합니다.
