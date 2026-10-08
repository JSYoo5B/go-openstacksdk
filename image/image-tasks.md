# 이미지별 Glance Task 조회

`image.Service.ImageTasks`와 `AllImageTasks`는 특정 이미지에 연결된 task를 `/v2/images/{image_id}/tasks`에서 조회합니다. [일반 Task API](tasks.md)의 `/v2/tasks`와 별도 endpoint이며, task 생성이나 완료 대기를 포함하지 않습니다. `ImageTaskInfo`는 실제 응답 필드의 raw JSON bytes·header·status를 소유합니다.

[고정 공식 API-ref](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/api-ref/source/v2/images-images-v2.inc#L311-L350)는 이 API를 **v2.12부터** 제공하며 정상 응답을 200으로 명시합니다. Python [ImageTasks Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/image_tasks.py#L20-L61)의 `_max_microversion = '2.17'`은 협상 시 최대값이며 최소 지원 버전이 아닙니다. Go는 2.17 gate나 version·Schema discovery를 추가하지 않습니다.

| 항목 | 동작과 기본값 |
|---|---|
| Parent | `resource.Ref`가 필수입니다. `resource.ID`는 parent metadata GET 없이 지정한 ID를 사용합니다. `resource.Name`은 기존 정확한 image name resolver를 사용합니다. |
| `ImageTasks` | option slice를 소유한 lazy iterator입니다. iteration마다 준비하고 body/query 없는 task GET 200을 한 번 수행합니다. Name 조회에 필요한 native 요청은 별도입니다. |
| `AllImageTasks` | 같은 iterator를 모읍니다. 성공한 빈 결과는 nonnil 빈 slice이고, 오류 시 부분 slice 대신 nil/error를 반환합니다. |
| Options | Headers와 local `MaxItems`만 받습니다. 0은 무제한, 양수는 local cap, 음수는 사전 오류입니다. wire limit·marker·sort·filter를 생성하지 않습니다. |

아래 두 함수는 각각 요청을 수행합니다. 첫 함수의 iterator와 두 번째 함수의 전체 수집을 필요에 따라 사용합니다.

```go
package examples

import (
    "context"

    "github.com/JSYoo5B/go-openstacksdk/image"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func imageTaskExamples(ctx context.Context, svc *image.Service, imageID string) ([]*image.ImageTaskInfo, error) {
    options := []image.ListImageTasksOption{
        image.WithListImageTasksOpts(image.ListImageTasksOpts{MaxItems: 0}),
        image.WithListImageTasksHeader("X-Request-Id", "image-tasks-example"),
        image.WithListImageTasksHeaders(map[string]string{"X-Example": "tasks"}),
        image.WithListImageTasksMaxItems(20),
    }
    rows := make([]*image.ImageTaskInfo, 0)
    for task, err := range svc.ImageTasks(ctx, resource.ID(imageID), options...) {
        if err != nil {
            return nil, err
        }
        rows = append(rows, task)
    }
    return rows, nil
}

func allImageTaskExamples(ctx context.Context, svc *image.Service, imageID string) ([]*image.ImageTaskInfo, error) {
    return svc.AllImageTasks(ctx, resource.ID(imageID), image.WithListImageTasksMaxItems(20))
}
```

`WithListImageTasksOpts`는 Headers·MaxItems 전체를 교체하고 마지막 helper가 우선합니다. Header factory는 map을 snapshot으로 소유하고, callback 전후에도 Headers를 복사합니다. iterator는 생성 시 option slice를 복사하며 각 실행에서 source와 options를 준비합니다. caller가 공유 입력을 동시에 변경하지 않는 범위에서 재사용·병렬 iteration을 지원합니다.

Parent ID는 비어 있지 않은 UTF8 URI literal이어야 합니다. dot/dotdot·slash·backslash·control 문자는 사전 오류이며, UUID나 local 최대 길이를 강제하지 않습니다. 안전한 literal은 한 번 escape합니다. Name은 SDK의 명시적 확장입니다. 캡처한 client의 `Images.ResolveID`가 기존 native image 목록·정확한 이름·중복/없음 정책을 사용하고, 선택한 ID와 source를 확인한 뒤 task endpoint를 요청합니다. Name resolver의 native 모델·pagination·body 소유권은 기존 동작을 유지합니다.

Python의 실제 proxy 호출은 하나입니다.

```python
def image_tasks_example(conn, image_id):
    return list(conn.image.image_tasks(image_id))
```

[고정 proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/image/v2/_proxy.py#L1181-L1194)의 docstring은 name을 언급하지만 실제 `Resource._get_id(image)`는 string을 그대로 사용하거나 Resource의 ID를 가져옵니다. Name lookup을 호출하지 않습니다. Python은 mutable Resource와 Adapter/session·microversion·일반 Resource pager를 사용합니다. Go의 Name 확장과 concrete 옵션은 이 동작과 차이가 있습니다.

[Controller](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/api/v2/images.py#L599-L614)는 server에서 parent image를 읽고 `get_image` policy를 확인합니다. NotFound·Forbidden은 404로 숨겨질 수 있으므로 404가 없음의 증거는 아닙니다. 이후 [SQL 조회](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/sqlalchemy/api.py#L1834-L1873)는 유한한 `query.all()` 결과를 반환합니다. owner·expiry·deleted visibility는 서버 정책이며 요청 query·페이지·순서 계약은 없습니다. source의 반환값을 대입하지 않은 `updated_at` filter를 실제 cutoff 보장으로 해석하지 않습니다.

응답은 canonical nonnull object와 정확한 `tasks` nonnull array를 요구합니다. whole-body UTF8·JSON syntax를 먼저 검사하고 소비한 row만 decode합니다. cap이나 early break는 사용하지 않은 row를 검사하지 않습니다. `next`·HTTP Link·first·schema·row self는 타입에 관계없이 passive data이며 검증하거나 따라가지 않습니다. 숨은 row filtering·다음 요청·last-ID fallback·추가 task GET을 수행하지 않습니다.

`ImageTaskInfo`는 기존 `TaskInfo`를 포함하고 `Deleted *bool`, `DeletedAt *string`을 별도의 원자적 decoder에서 검사합니다. deleted missing/null은 nil, false/true는 present이며 다른 nonnull 타입은 오류입니다. deleted_at과 나머지 날짜는 literal optional string이고 time으로 parse하지 않습니다. canonical `request_id`·`user_id`는 실제 [SQL formatter](https://github.com/openstack/glance/blob/57f7dd9e76ef24e1e9013eceaa703bd442469a24/glance/db/sqlalchemy/api.py#L2018-L2043)의 필드입니다. 공식 sample의 `request-id`·`user` 및 uppercase alias는 raw-only이며 canonical 필드를 채우지 않습니다. 요청한 parent ID나 task ID·deleted 기본값으로 응답을 생성하거나 일치를 강제하지 않습니다.

Input/Result는 missing/null이면 nil, `{}`면 nonnil raw object map입니다. SQL의 `JSONEncodedDict`는 arbitrary JSON을 읽을 수 있고 Python Body 필드는 타입 제한이 없지만, SDK는 기존 `TaskInfo`의 nullable object 정책을 유지합니다. 따라서 nonobject Input/Result는 decode 오류이며 이 제한은 명시적인 Go 차이입니다. `Metadata.Body`는 explicit null·unknown JSON·큰 숫자의 raw field bytes를 보존하고 typed map과 독립적으로 소유합니다. 성공한 전체 document의 byte-for-byte 복사나 Python Resource 전체 parity를 뜻하지 않습니다. 원래 `TaskInfo`, generated/native 일반 Task API와 `WaitForTask`는 유지됩니다.

요청은 callback 전에 client/provider·Type·Endpoint·base·microversion과 ordinary source headers를 캡처합니다. source headers는 Name resolver와 단일 task GET 동안 고정되며 option headers가 덮어씁니다. 원래 provider의 auth는 live입니다. 요청 전·resolver 후·accepted 응답 후·소비 row 전후의 source/context 확인은 retargeting을 막습니다. native RetryFunc의 명시적 ordinary header 정책과 설정된 same-target redirect 동작은 유지됩니다.

actual GET 200만 decode합니다. accepted read·Close·context/custom cause·source·envelope/model 오류는 nil row와 실제 whole body·header·status를 가진 `ResponseError`를 반환합니다. body는 한 번 닫고 accepted 요청을 replay하지 않습니다. native unexpected status·prebody retry/reauth/backoff와 method/URL/body/status ownership guard는 기존 정책을 유지합니다. direct 404는 오류이고 missing suppression은 없습니다. 이 가이드는 고정 source의 동작을 설명하며 실제 cloud 실행·task 완료·권한 통과를 검증했다는 의미가 아닙니다.

이 동작은 [core 테스트](image_tasks_core_test.go), [options 테스트](image_tasks_options_test.go), [외부 HTTP 계약 테스트](image_tasks_contracts_test.go), [Connection 통합 테스트](../connection_image_tasks_associated_test.go), [native·registry 보존 테스트](../internal/cmd/sdkgen/glance_image_tasks_test.go)에서 고정 source와 로컬 fixture로 검증합니다.
