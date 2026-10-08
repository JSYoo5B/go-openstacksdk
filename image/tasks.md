# Glance Task 생성·조회·목록

`image.Service`의 `CreateTask`, `GetTask`, `Tasks`, `AllTasks`는 `/v2/tasks`의 생성·조회·목록을 다룹니다. 결과 `TaskInfo`는 응답 필드의 raw JSON bytes·header·status를 소유합니다. 기존 [generated/native Task API와 WaitForTask](v2/tasks/README.md)는 계속 사용할 수 있습니다. 여기의 생성은 task 접수이며 완료 대기나 이미지 import 성공 확인을 포함하지 않습니다.

[공식 Tasks API](https://docs.openstack.org/api-ref/image/v2/index.html#tasks)는 v2.2부터 제공되며 기본 정책상 일반 사용자에게 제한될 수 있습니다. [고정 controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/tasks.py#L70-L155)의 실제 계약은 POST 201, 개별 GET 200, 목록 GET 200입니다. SDK가 policy·version·Schema를 먼저 조회하지는 않습니다.

| 호출 | 요청과 기본값 |
|---|---|
| `CreateTask(ctx, taskType, ...)` | type/input을 담은 collection POST. type은 비어 있지 않은 UTF8 literal이고 local enum 검사를 하지 않습니다. nil Input은 `{}`입니다. |
| `GetTask(ctx, taskID, ...)` | 지정한 literal task ID의 GET. 404는 오류입니다. Ref·Name·Find·missing suppression을 제공하지 않습니다. |
| `Tasks(ctx, ...)` | 실행 시 요청하는 lazy iterator. 기본은 서버 limit·sort 설정을 사용하고 광고된 canonical body next만 따라갑니다. |
| `AllTasks(ctx, ...)` | 같은 iterator를 모아 nonnil 빈 slice 또는 전체 결과를 반환합니다. 오류 시 부분 slice 대신 nil/error입니다. |

아래 함수들은 각각 명시적으로 요청합니다. `createTask` 후에 조회하려면 actual ID를 확인하여 `getTask`에 전달합니다. 목록의 `MaxItems`는 local cap이며 wire limit을 자동으로 만들지 않습니다.

```go
package examples

import (
    "context"
    "encoding/json"

    "github.com/JSYoo5B/go-openstacksdk/image"
)

func createTask(ctx context.Context, svc *image.Service) (*image.TaskInfo, error) {
    return svc.CreateTask(ctx, "import",
        image.WithCreateTaskOpts(image.CreateTaskOpts{
            Input: map[string]json.RawMessage{"import_from_format": json.RawMessage(`"qcow2"`)},
        }),
        image.WithCreateTaskHeader("X-Request-Id", "task-create-example"),
        image.WithCreateTaskHeaders(map[string]string{"X-Example": "create"}),
        image.WithCreateTaskInput(map[string]any{
            "import_from": "https://example.invalid/image.qcow2",
            "import_from_format": "qcow2",
            "image_properties": map[string]any{
                "container_format": "bare", "disk_format": "qcow2",
            },
        }))
}

func getTask(ctx context.Context, svc *image.Service, taskID string) (*image.TaskInfo, error) {
    return svc.GetTask(ctx, taskID,
        image.WithGetTaskOpts(image.GetTaskOpts{}),
        image.WithGetTaskHeader("X-Request-Id", "task-get-example"),
        image.WithGetTaskHeaders(map[string]string{"X-Example": "get"}))
}

func listTasks(ctx context.Context, svc *image.Service, marker string) ([]*image.TaskInfo, error) {
    options := []image.ListTasksOption{
        image.WithListTasksOpts(image.ListTasksOpts{}),
        image.WithListTasksHeader("X-Request-Id", "task-list-example"),
        image.WithListTasksHeaders(map[string]string{"X-Example": "list"}),
        image.WithListTasksLimit(20),
        image.WithListTasksMarker(marker),
        image.WithListTasksType("import"),
        image.WithListTasksStatus("pending"),
        image.WithListTasksSortKey("created_at"),
        image.WithListTasksSortDir("desc"),
        image.WithListTasksMaxItems(40),
        image.WithListTasksSinglePage(false),
    }
    rows := make([]*image.TaskInfo, 0)
    for task, err := range svc.Tasks(ctx, options...) {
        if err != nil {
            return nil, err
        }
        rows = append(rows, task)
    }
    return rows, nil
}

func allTasks(ctx context.Context, svc *image.Service) ([]*image.TaskInfo, error) {
    return svc.AllTasks(ctx, image.WithListTasksMaxItems(40))
}
```

`WithCreateTaskInput(map[string]any)`는 factory에서 JSON snapshot을 만들며 원래 top-level key의 UTF8을 marshal 전에 검사합니다. marshal 오류는 option 실행 때 반환됩니다. FullOpts의 raw Input은 원래 JSON 숫자 token과 null을 유지할 수 있습니다. 모든 raw 값은 valid UTF8 JSON이어야 하고 예약된 input key는 없습니다. 임의 any의 nested 값은 `encoding/json` 규칙을 따르므로 손실 없는 JSON 변환을 약속하지 않습니다. nil helper 입력은 기본 `{}`로 재설정합니다.

`With...Opts`는 해당 설정 전체를 교체하고 마지막 helper가 우선합니다. Headers·Input map/raw buffer·Limit pointer·callback config는 실행 전후에 복사합니다. `Tasks`는 생성 시 option slice를 복사하고 iteration마다 준비합니다. 재사용과 병렬 iteration은 caller가 공유 입력을 동시에 변경하지 않는 범위에서 지원합니다.

[고정 task Schema](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/tasks.py#L354-L430)는 input에 null/object를 허용하지만 controller는 `task['input'].get('image_id')`를 사용합니다. Go는 이 차이를 고려하여 항상 object를 보냅니다. type은 server의 `import`, `api_image_import`, `location_import` 검증을 그대로 둡니다. 생성 Schema가 허용하는 일부 다른 root 속성도 controller는 type/input 이외에는 전달하지 않습니다. SDK의 concrete CreateOpts는 input과 headers만 받습니다.

Python의 실제 세 proxy 호출은 다음과 같습니다. [Task Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/task.py#L16-L53)는 create/fetch/list를 제공하며 update/delete는 제공하지 않습니다.

```python
def task_examples(conn):
    created = conn.image.create_task(type="import", input={
        "import_from": "https://example.invalid/image.qcow2",
        "import_from_format": "qcow2",
        "image_properties": {"container_format": "bare", "disk_format": "qcow2"},
    })
    fetched = conn.image.get_task(created.id)
    rows = list(conn.image.tasks(type="import", status="pending", limit=20))
    return created, fetched, rows
```

Python `get_task`는 전달한 mutable Resource를 재사용할 수 있지만 [fetch](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L1778-L1842)는 항상 Adapter GET을 호출합니다. 실제 HTTP cache는 설정한 Adapter 정책에 달려 있습니다. Go는 dirty Resource/cache/동일 객체 갱신·반환을 구현하지 않고 요청마다 소유한 새 모델을 반환합니다.

`TaskInfo`의 ID·Type·Status·Owner·Message·Self·Schema·ImageID·RequestID·UserID·ExpiresAt과 CreatedAt/UpdatedAt은 canonical optional `*string`입니다. missing/null은 nil, 명시적 빈 문자열은 present이며 날짜를 time으로 parse하지 않습니다. Input/Result는 missing/null이면 nil, `{}`면 nonnil raw map이고 다른 nonnull 타입이면 decode 오류입니다. 각 raw map과 `Metadata.Body`는 독립된 bytes를 소유하며 Body는 explicit null·unknown JSON·큰 숫자 precision을 보존합니다. canonical 필드의 다른 타입은 원자적으로 실패합니다. requested ID·type·status로 응답을 채우거나 일치하도록 강제하지 않습니다. Self·Schema·Location·links는 route가 아닙니다.

[목록 serializer](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/tasks.py#L308-L351)는 input/result/message를 생략한 sparse task를 반환합니다. SDK는 각 row를 추가 GET하거나 생략 필드를 생성하지 않습니다. native `Task`의 `time.Time`과 `map[string]any` 변환은 이 facade의 literal 날짜·raw JSON과 다릅니다.

List query는 limit·marker·type·status·sort_key·sort_dir의 여섯 가지입니다. nil Limit은 생략하고 0은 전송하며 음수는 사전 오류입니다. 다른 빈 문자열은 생략하고 nonempty UTF8/control-free literal을 전송합니다. SortDir만 asc/desc를 검사하며 UUID·type·status·sort-key enum gate는 추가하지 않습니다. 서버는 생성 type 세 가지와 달리 목록 type으로 `import`만 허용합니다. [native v2.15.0 ListOpts](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/tasks/requests.go#L36-L72)의 Type에는 q tag 대신 json tag가 있어 기본 native builder는 이를 query에 넣지 않습니다. 이 facade는 명시적 `type` query를 보내며 기존 native 동작을 변경하지 않습니다.

목록은 canonical nonnull tasks array를 요구하고 whole-body UTF8/syntax를 먼저 검사한 뒤 소비한 row만 decode합니다. 광고된 `/v2/tasks` next는 캡처한 reverse proxy prefix로 정규화하고 같은 origin·collection·초기 filter query를 유지하며 단일 advancing marker만 허용합니다. malformed·foreign·fragment·userinfo·encoded dot·duplicate/누락/추가 query·cycle은 다음 HTTP 전에 오류입니다. HTTP Link·first·schema·row self를 따라가거나 last-ID continuation을 추측하지 않습니다. 빈 page는 종료합니다. cap·early break·SinglePage는 남은 row와 next를 검사하지 않고 끝내며, iterator가 이미 전달한 row는 이후 오류에도 caller가 갖습니다. `AllTasks`는 오류 시 nil/error입니다.

요청은 callback 전에 service/client/provider·Type·Endpoint·base·microversion을 캡처하고 요청 전·응답 후·소비 row 전후에 source/context를 확인합니다. 원래 provider의 auth는 live이며, 다음 page의 ordinary source headers는 새로 읽고 고정 option headers가 덮어씁니다. source/option의 auth·transport·version 보호 규칙은 기존 image 정책과 같습니다. native RetryFunc의 명시적 ordinary header 정책은 유지됩니다.

actual POST 201/GET 200만 성공으로 decode합니다. accepted read·Close·context/custom cause·source·JSON/model 실패는 typed nil과 실제 whole response bytes·header·status를 가진 ResponseError를 반환하며 body를 한 번 닫고 accepted 요청을 replay하지 않습니다. native unexpected status·prebody retry/reauth/backoff와 기존 same-target redirect 정책은 유지되며 method/origin/path/query와 owned body/status guard가 적용됩니다. 권한·owner·expiry에 따라 GET 404가 숨겨진 task를 뜻할 수도 있어 없음의 증거로 해석하지 않습니다. 이 facade는 import 완료·자동 wait·cleanup·policy 통과·실제 cloud 실행을 보장하지 않습니다.

실행 가능한 계약 검증은 [core 테스트](tasks_core_test.go), [options 테스트](tasks_options_test.go), [외부 HTTP 계약 테스트](tasks_contracts_test.go), [Connection 통합 테스트](../connection_image_tasks_test.go), [native·registry 보존 테스트](../internal/cmd/sdkgen/glance_tasks_test.go)에 있습니다. 이 검증은 고정 source와 로컬 fixture를 대상으로 하며 실제 cloud 실행 결과는 포함하지 않습니다.
