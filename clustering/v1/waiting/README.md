# Senlin 상태·삭제 대기

9개 리소스 API와 고정 ClusterPolicy scope의 `WaitForStatus`와 `WaitForDelete`는 Senlin proxy의 대기 기본값을
제공합니다. SDK가 조회와 polling을 담당하므로 호출자가 builder나 상태 adapter를
구현하지 않아도 됩니다. 대기 자체에 추가 microversion gate는 없으며 각 GET의
service type, 선택한 numeric microversion, 인증과 endpoint 검증을 재사용합니다.

| 항목 | pinned openstacksdk | Go 리소스 API |
|---|---|---|
| 상태 대기 | `conn.clustering.wait_for_status(res, status, ...)` | `API.WaitForStatus(ctx, ref, status, options...)` |
| 목표 상태 | 필수 인자 | 필수 nonempty 문자열 |
| 기본 실패 상태 | `['ERROR']` | `ERROR`; 대소문자 무시 |
| 상태 대기 제한 | `wait=None`: 무제한 | SDK 제한 없음; parent context 적용 |
| 확인 간격 | `interval=2`초 | 2초; `WithPollInterval`로 변경 |
| 삭제 대기 | `conn.clustering.wait_for_delete(res, ...)` | `API.WaitForDelete(ctx, ref, options...)` |
| 삭제 대기 제한 | proxy 기본 `wait=120`초 | 120초; context와 뒤의 옵션 적용 |
| 삭제 완료 | fetch의 404, falsy resource 또는 `status='deleted'` | GET의 404 또는 해당 status; null 응답은 오류 |
| 결과 | 상태·삭제 모두 Resource | 상태는 typed 모델, 삭제는 `error` |
| 진행 callback | 비종결 fetch의 progress, 없으면 0 | 비종결 조회의 progress, Senlin 모델에는 없어 0 |

이 표의 Go 기본값은 리소스 API의 `WaitForStatus`/`WaitForDelete`에 적용합니다.
공통 `API.Resources.Wait`와 `WaitDeleted`는 기존 5분 제한을 유지합니다.
`Resources.Wait`의 기본 실패 판정은 service binding을 따르며 Senlin API의
`ERROR` 기본값을 자동으로 적용하지 않습니다.

## 리소스별 범위

| 서비스 필드 | `WaitForStatus` 기본 속성 | `WaitForDelete` 기본 종결 조건 |
|---|---|---|
| `Actions`, `Clusters`, `Nodes`, `Events` | `status` | 404 또는 `status='deleted'` |
| `Profiles`, `Policies`, `Receivers` | 상태 속성 없음; 문자열 속성 명시 필요 | 404 |
| `ProfileTypes`, `PolicyTypes` | 상태 속성 없음; 문자열 속성 명시 필요 | 404 |
| `ClusterPolicies.InCluster(ctx, ref)` | 상태 속성 없음; 문자열 속성 명시 필요 | 고정 cluster의 policy GET에서 404 |

기본 status가 없는 모델의 상태 대기는 HTTP 전에 `resource.ErrUnsupported`를
반환합니다. `WithStatusAttribute("name")`처럼 exported 문자열 필드의 JSON 이름이나
Go 필드 이름을 선택하면 상태 대기를 사용할 수 있습니다. Type 모델의 `version`도
문자열 포인터 필드이지만 nil 값은 지원하지 않습니다. `support_status`는 raw JSON이며
문자열 속성으로 선택할 수 없습니다. unknown Body 필드와 중첩 경로도 선택하지 않습니다.
이름 lookup이 없는 `Events`에는 `resource.ID`를 사용합니다. Type 모델의 ID는 정확한
타입 이름이며 version과 합친 가상 ID를 만들지 않습니다.

삭제 대기는 서버의 DELETE capability와 관계없이 GET으로 존재 여부를 확인합니다.
따라서 `Actions`, `Events`, 타입 catalog도 삭제 polling을 제공하지만 새로운 DELETE
요청을 만들지는 않습니다. 상태가 없는 모델에 incidental raw `status`가 반환되어도
기본 삭제 조건은 바뀌지 않습니다. 삭제 대기의 `WithStatusAttribute`는 Python helper에
없는 Go 확장이며 해당 문자열 값이 `deleted`이면 완료로 처리합니다.

`Services`는 Python의 `allow_fetch=False`와 Go의 목록 전용 API 때문에 대기 GET route를
제공하지 않습니다. Python은 이미 목표 상태인 cached Service를 즉시 반환할 수 있지만
새 Go 대기 API는 Ref 조회를 사용합니다. `BuildInfo`도 새 polling route를 제공하지
않습니다. pinned `get_build_info`는 `requires_id=False`를 넘기지만 Resource wait의 fetch는
다시 기본 `requires_id=True`를 사용하여 ID 없는 정상 BuildInfo에서 InvalidRequest를 냅니다.
[ClusterPolicy scope](../clusterpolicies/README.md)는 부모 cluster를 한 번 해석하고 같은 policy
route에서 대기합니다. binding의 별도 `ID`로 요청 경로를 바꾸지 않으며 Name lookup은
`PolicyID`를 사용합니다. 연결 해제는 `Clusters.DetachPolicy`로 별도 요청합니다.
[ClusterAttributes](../clusterattributes/README.md)는 list-only여서 대기 GET을 제공하지 않습니다.

## 상태 대기 사용

```python
cluster = conn.clustering.get_cluster("CLUSTER_ID")
cluster = conn.clustering.wait_for_status(
    cluster, "ACTIVE", failures=["ERROR"], interval=2, wait=180,
)
```

```go
// context.Context ctx, *gophercloudsdk.Connection conn을 사용하는 함수 안에서
service, err := conn.ClusteringV1(ctx)
if err != nil { return err }
cluster, err := service.Clusters.WaitForStatus(ctx,
    resource.ID("CLUSTER_ID"), "ACTIVE",
    resource.WithTimeout(3*time.Minute))
if err != nil { return err }
fmt.Println(cluster.ID, cluster.Status, cluster.Header, cluster.Body)
```

예제의 `resource`는 `gophercloudsdk/resource`, `time`과 `fmt`는 표준 라이브러리입니다.
Go는 `resource.ID`와 `resource.Name`을 명시하므로 UUID 모양 이름도 이름 그대로 검색할 수
있습니다. ID는 fresh GET으로 시작합니다. Name은 정확한 서버 목록 lookup의 결과로 시작하며
그 결과가 이미 목표 상태이면 단건 GET을 추가하지 않습니다. lookup 이후의 polling은 같은
ID를 사용하고 응답에 다른 ID가 와도 route를 변경하지 않습니다. 이름 중복은 오류입니다.

실패 상태 목록은 기본 목록을 교체합니다. `WithFailureStates()`는 실패 상태 판정을
끄고, `WithFailureStates("ERROR", "FAILED")`는 두 상태를 지정합니다. 기본값이 `ERROR`라고
해서 `FAILED`나 `CANCELLED`를 추가로 추측하지 않습니다. 목표 상태가 실패 목록에도 있으면
목표 도달을 먼저 판정합니다. `WithStatusAttribute`는 목표와 실패 상태 모두에 적용합니다.

```go
profile, err := service.Profiles.WaitForStatus(ctx,
    resource.ID("PROFILE_ID"), "renamed-profile",
    resource.WithStatusAttribute("name"),
    resource.WithFailureStates(),
    resource.WithTimeout(30*time.Second))
if err != nil { return err }
fmt.Println(profile.Name)
```

## 삭제 대기와 옵션

```python
conn.clustering.delete_node(node)
node = conn.clustering.wait_for_delete(node)  # 기본 최대 120초
```

```go
submission, err := service.Nodes.Delete(ctx, resource.ID("NODE_ID"))
if err != nil { return err }
fmt.Println(submission.ActionID)
if err := service.Nodes.WaitForDelete(ctx, resource.ID("NODE_ID")); err != nil {
    return err
}
```

Delete와 wait는 별도 호출입니다. waiter는 action Location, receiver channel 등을 해석하거나
follow하지 않고 리소스 GET만 polling합니다. `WaitForStatus`에서 404는 오류이며
`WaitForDelete`에서는 성공입니다. 403, 409, transport 오류와 잘못된 성공 응답은 모두 대기를
끝내고 원래 원인을 보존합니다. 성공 응답의 잘못된 JSON/envelope/type은 Body, Header,
StatusCode를 가진 `resource.ResponseError`로 반환하며 해당 GET을 자동 재전송하지 않습니다.

SDK 기본 옵션 뒤에 caller 옵션을 적용하므로 같은 정책은 마지막 옵션이 이깁니다.
`WithTimeout(d)`는 양수만 받고 `WithUnlimitedWait()`는 SDK timeout을 제거합니다.
parent context의 deadline이나 취소는 항상 적용됩니다.

```go
err = service.Nodes.WaitForDelete(ctx, resource.ID("NODE_ID"),
    resource.WithTimeout(10*time.Minute),
    resource.WithUnlimitedWait(), // 이 호출의 SDK timeout을 제거
    resource.WithPollInterval(time.Second),
    resource.WithProgressCallback(func(progress int) { fmt.Println(progress) }))
if err != nil { return err }
```

callback은 동기 호출이며 비종결 조회에만 실행됩니다. 목표·실패 상태, 삭제 완료, HTTP 오류에는
호출하지 않습니다. Name 상태 대기는 초기 목록 결과도 callback 대상이고, Name 삭제 대기는
이름 해석 후의 GET 결과부터 callback을 호출합니다. callback의 context 취소는 추가 GET을 막습니다.

## Python Resource와의 차이

Python helper는 문자열 ID나 이름을 받지 않고 Resource를 받습니다. 현재 Resource의 속성이
목표와 같으면 HTTP 없이 같은 객체를 반환하며, 이후 fetch는 보통 그 객체를 갱신합니다.
삭제의 404도 원래 Resource를 반환합니다. Go API는 explicit Ref로 새 조회를 수행하고 이전에
받은 모델을 변경하지 않으며 삭제 대기는 `error`만 반환합니다.

Python Resource helper는 nullable 목표 상태와 nullable 속성, `interval=None` 및 0의 특별한
간격 처리도 허용합니다. Go 목표는 nonempty 문자열이고 간격은 양수이며 nullable 문자열
포인터가 nil인 선택 속성은 `ErrUnsupported`입니다. Python은 timeout을 fetch 사이에 검사하여
느린 HTTP 요청이 제한을 넘긴 뒤 성공할 수 있습니다. Go는 context deadline을 HTTP에도 전달하고
취소를 우선합니다. Python Resource의 permissive response/cache/dirty-state 동작 전체를
이 대기 API에 옮기지는 않습니다.

비교 기준은 pinned
[Senlin proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/clustering/v1/_proxy.py#L1350),
[Resource wait helpers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2591),
[timeout iterator](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/utils.py#L53)입니다.
[HTTP 회귀 테스트](../../../api/clustering_wait_contracts_test.go)는 9개 API의 기본값과 override,
실패·삭제 종결, 이름·ID, deadline/cancel과 응답 증거를 검증합니다.

[고정 ClusterPolicy HTTP 테스트](../../../api/clustering_cluster_policies_test.go)는 parent·policy ID 구분, statusless 대기, 삭제 GET과 동일 scope의 요청 재검사를 검증합니다.
