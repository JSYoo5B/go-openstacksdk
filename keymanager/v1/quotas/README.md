# Barbican quotas

SDK가 effective quota 조회와 프로젝트별 override의 조회·교체·삭제를 제공합니다. builder interface를 구현할 필요 없이 concrete `UpdateOpts`와 `With...` 함수를 사용합니다.

| openstacksdk | Go | HTTP 계약 |
|---|---|---|
| `conn.key_manager.get_quota()` | `api.Get(ctx)` | `GET /quotas`, `quotas` 객체, 200 |
| `get_project_quota(project)` | `scope.Get(ctx)` | `GET /project-quotas/{id}`, `project_quotas` 객체, 200 |
| `update_project_quota(project, **attrs)` | `scope.Update(ctx, opts, options...)` | `PUT /project-quotas/{id}`, `project_quotas` envelope, 204 |
| `delete_project_quota(project, ignore_missing=True)` | `scope.Delete(ctx, options...)` | `DELETE /project-quotas/{id}`, 204; 기본적으로 실제 HTTP 404 무시 |

`api.Get`은 현재 Provider token의 프로젝트에 적용되는 quota를 읽습니다. `InProject(ctx, resource.ID(id))`는 조회 없이 명시적인 프로젝트 ID를 고정합니다. 해당 ID는 token의 프로젝트나 응답의 추가 필드가 바뀌어도 유지됩니다. Python의 `_get_id`에 대응하며 `resource.Name`은 지원하지 않습니다. Keystone 이름 검색이나 프로젝트 목록 요청을 추가하지 않습니다.

```go
package quotaexample

import (
    "context"
    "fmt"

    "github.com/gophercloud/gophercloud/v2"
    "gophercloudsdk/keymanager/v1/quotas"
    "gophercloudsdk/request"
    "gophercloudsdk/resource"
)

func Configure(ctx context.Context, client *gophercloud.ServiceClient, projectID string) error {
    api := quotas.New(client)
    effective, err := api.Get(ctx)
    if err != nil { return err }
    fmt.Println(string(effective.Secrets))

    project, err := api.InProject(ctx, resource.ID(projectID))
    if err != nil { return err }
    configured, err := project.Get(ctx)
    if err != nil { return err }
    fmt.Println(string(configured.Orders))

    accepted, err := project.Update(ctx, quotas.UpdateOpts{
        Secrets: request.Present[int64](100),
    }, quotas.WithUpdateOrders(0), quotas.WithUpdateContainers(-1),
        quotas.WithUpdateHeader("X-Request-ID", "quota-change"))
    if err != nil { return err }
    fmt.Println(accepted.StatusCode, accepted.Header.Get("X-Request-ID"))

    return project.Delete(ctx, quotas.WithDeleteIgnoreMissing(false))
}
```

`Update`는 다섯 override를 교체합니다. `secrets`, `orders`, `containers`, `consumers`, `cas` 중 생략한 값은 서버 기본값으로 돌아갑니다. 현재 값을 읽거나 생략한 값을 보충하는 GET은 없습니다. 빈 `UpdateOpts{}`는 `{"project_quotas":{}}`를 전송하며, 기존 override 전체를 기본값으로 돌리는 요청입니다. `0`과 `-1`은 명시적으로 전송됩니다. 공개 integer schema에 minimum이 없으므로 SDK는 `-2` 같은 값을 임의로 거부하지 않으며, 정책 해석은 서버가 결정합니다.

고정 Python Resource는 변경된 속성이 없는 empty update에서 HTTP 요청을 생략할 수 있습니다.
Go의 빈 `UpdateOpts{}`는 명시적인 전체 override 초기화 PUT을 제출하므로, 이 차이를 고려해 사용합니다.

Python quota 속성은 untyped입니다. Go의 변경 입력은 stock Barbican validator에 맞춰 `request.Optional[int64]`로 제한합니다. 명시적 null, bool, 소수, 숫자 문자열과 알려지지 않은 body 필드는 요청 전에 거부합니다. 전송 가능한 정수 범위는 `int64`이며, 응답 조회에는 이 제한을 적용하지 않습니다. `WithUpdateOptions`는 입력을 소유한 snapshot으로 교체하고, 이후의 옵션이 같은 필드의 값을 바꿉니다. 헤더 옵션은 대소문자를 구분하지 않고 마지막 값이 적용됩니다.

조회 결과의 다섯 필드는 `json.RawMessage`입니다. 생략은 nil, JSON null은 `null` 바이트로 구분하며, 큰 숫자와 알려지지 않은 응답 필드를 보존합니다. `Body`에는 실제 envelope 내부 객체 전체가, `Data`에는 추가 필드가 들어갑니다. `Header`와 `StatusCode`는 실제 GET 응답의 증거입니다. 잘못된 envelope나 객체 타입, accepted 응답의 decode/read 오류는 `resource.ResponseError`에 상태·헤더·읽힌 body를 보존합니다.

PUT의 204는 `UpdateResult`의 실제 body·header·status로 반환합니다. 변경 입력으로 quota 객체를 만들거나 응답 후 자동 GET을 하지 않습니다. Python Resource의 cached/dirty 수명주기와는 다른 명시적 Go 결과입니다. accepted 응답을 읽는 중 오류가 나도 자동 재전송하지 않으므로, 재시도 여부는 호출자가 응답 증거를 확인해 결정합니다. Delete도 실제 HTTP 404만 기본적으로 무시하며, 전송 오류나 accepted 응답의 read 오류에 404 원인이 포함돼 있어도 그 오류를 반환합니다.

모든 요청은 원래 ServiceClient와 공유 Provider를 사용합니다. ResourceBase, 최신 token, HTTP transport, 설정된 microversion과 context를 유지하며 버전을 자동 변경하지 않습니다. singleton/scope 요청은 query와 body 확장을 제공하지 않습니다. 인증·버전·전송 소유 헤더와 잘못된 헤더는 사전에 거부합니다. scope와 option을 재사용해도 SDK 내부 snapshot이 요청 중 입력 변경을 막습니다.

계약 근거는 [고정 openstacksdk proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/key_manager/v1/_proxy.py#L397), [Barbican quota API](https://docs.openstack.org/api-ref/key-manager/v1/#quotas), stock controller의 [ProjectQuotaValidator](https://github.com/openstack/barbican/blob/master/barbican/common/validators.py)입니다. 실제 HTTP 회귀는 [quota 테스트](../../../api/keymanager_quotas_test.go)에서 확인합니다.
