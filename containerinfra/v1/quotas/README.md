# Magnum 프로젝트·리소스 quota

Magnum quota의 요청 대상은 프로젝트 ID와 리소스 이름의 쌍입니다. `InProject` 또는 `CurrentProject`로 프로젝트를 한 번 확정하고, `ForResource(quotas.Cluster)`로 quota를 선택합니다. scope 구성은 quota 조회를 보내지 않습니다.

| 작업 | Go |
|---|---|
| 프로젝트 ID 고정 | `api.InProject(ctx, resource.ID(projectID))` |
| Keystone 프로젝트 이름 해석 | `api.InProject(ctx, resource.Name(name), quotas.WithIdentityClient(identityClient))` |
| 기록된 인증 프로젝트 선택 | `api.CurrentProject(ctx)` |
| Cluster quota 선택 | `project.ForResource(quotas.Cluster)` |
| hard limit 0으로 생성 | `scope.Create(ctx, quotas.WithHardLimit(0))` |
| 고정 프로젝트·리소스 조회 | `scope.Get(ctx)` |
| hard limit 2로 갱신 | `scope.Update(ctx, quotas.WithHardLimit(2))` |
| explicit quota 삭제 | `scope.Delete(ctx)` |
| 없는 quota의 삭제 허용 | `scope.Delete(ctx, quotas.WithDeleteIgnoreMissing(true))` |
| quota 페이지 순회 | `api.List(ctx, quotas.WithListOptions(...))` |
| 모든 프로젝트의 quota | `api.All(ctx, quotas.WithListAllTenants(true))` |
| concrete 옵션 snapshot | `quotas.WithQuotaCreateOptions(quotas.QuotaCreateOpts{HardLimit: &limit})` |
| concrete 갱신 옵션 snapshot | `quotas.WithQuotaUpdateOptions(quotas.QuotaUpdateOpts{HardLimit: &limit})` |
| 추가 서버 필드 | `quotas.WithQuotaCreateField("vendor", value)` |

pinned openstacksdk의 `container_infrastructure_management.v1.Proxy`에는 quota model과 quota proxy method가 없습니다. 따라서 이 표에는 존재하지 않는 Python quota 호출을 대응시키지 않습니다. Python에서 이 서비스의 quota를 사용하려면 REST 호출 또는 별도 Magnum client가 필요합니다. Go SDK는 Gophercloud의 native quota Create와 추가 REST 조회·갱신·삭제·목록 계약을 관리합니다.

```go
func createClusterQuota(ctx context.Context, client *gophercloud.ServiceClient) error {
    api := quotas.New(client)
    project, err := api.InProject(ctx, resource.ID("project-id"))
    if err != nil { return err }
    scope, err := project.ForResource(quotas.Cluster)
    if err != nil { return err }

    quota, err := scope.Create(ctx, quotas.WithHardLimit(0),
        quotas.WithQuotaCreateField("vendor", map[string]string{"tier": "small"}))
    if err != nil { return err }
    fmt.Println(quota.RequestProjectID, quota.RequestResource,
        quota.ID, quota.HardLimit, quota.Header.Get("X-Openstack-Request-Id"))
    current, err := scope.Get(ctx)
    if err != nil { return err }
    fmt.Println(current.HardLimit)

    updated, err := scope.Update(ctx, quotas.WithHardLimit(2))
    if err != nil { return err }
    fmt.Println(updated.StatusCode)

    _, err = scope.Delete(ctx)
    return err
}
```

예제 import는 `context`, `fmt`, `github.com/gophercloud/gophercloud/v2`, `github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/quotas`, `github.com/JSYoo5B/go-openstacksdk/resource`입니다. 인증과 endpoint가 설정된 Magnum ServiceClient를 전달합니다. 서버가 해당 확장을 제공할 때만 `vendor` 필드를 보낼 수 있습니다.

## 프로젝트와 resource

`resource.ID`는 Keystone 요청 없이 scope를 만듭니다. `resource.Name`은 별도의 Identity v3 client를 사용해 정확한 이름을 해석하며, 이름 없음·중복·Identity HTTP 오류를 구분합니다. 프로젝트 ID는 응답의 다른 `project_id`나 나중에 바뀐 인증 결과로 교체하지 않습니다.

`CurrentProject`는 ProviderClient에 기록된 Keystone v3 project 또는 v2 token tenant를 읽습니다. 인증 결과 없는 수동 token·system/domain/unscoped 인증은 `resource.ErrUnsupported`입니다. endpoint에서 프로젝트를 추측하거나 인증을 갱신하지 않습니다.

`ResourceName`은 대소문자를 보존합니다. `Cluster`가 문서화된 Magnum resource입니다. 다른 안전한 단일 이름도 extension resource로 지정할 수 있으며 서버가 지원 여부를 판단합니다. quota 응답의 row `id`는 project ID 또는 resource 이름을 대신하지 않습니다.

## 값과 응답

Create와 Update는 hard limit을 명시적으로 요구합니다. `WithHardLimit(0)`은 0을 생략하지 않으며, `QuotaCreateOpts.HardLimit` 또는 `QuotaUpdateOpts.HardLimit`이 nil인 상태는 HTTP 전에 `resource.ErrInvalidOption`입니다. SDK는 관리자 limit을 임의의 기본값으로 선택하지 않습니다. limit 값의 서버 유효성 규칙과 권한은 Magnum이 검증하며, 다른 서비스의 `-1=무제한` 규칙을 적용하지 않습니다.

`WithQuotaCreateOptions`와 `WithQuotaUpdateOptions`는 생성 시 pointer 값을 복사하고 매번 새 pointer로 적용합니다. `WithQuotaCreateField`와 `WithQuotaUpdateField`는 생성 시 JSON snapshot을 보관합니다. 옵션은 나중 옵션이 우선하며 `hard_limit`, `project_id`, `resource`, `id`를 extension으로 덮어쓸 수 없습니다. query·header·알 수 없는 argument 옵션도 HTTP 전에 거절합니다.

POST·PATCH body는 최종 JSON을 한 번 직렬화하여 재인증·재시도에서도 같은 snapshot을 보냅니다. 플랫폼 int 범위에서 2^53보다 큰 hard limit을 float64 반올림 없이 보존합니다. scoped 요청은 다른 method·path·query·origin으로의 HTTP redirect를 거절하고 선택한 ServiceClient 설정을 변경하지 않습니다.

`QuotaResource`는 native `Quotas`, 고정 요청 대상 `RequestProjectID`·`RequestResource`, 원문 필드 `Body`, 복사된 `Header`, 실제 `StatusCode`를 제공합니다. `ProjectID`와 `Resource`는 서버 응답 값이며 요청 대상을 대체하지 않습니다. native decoder가 float64로 변환하는 숫자 row ID는 원문 JSON에서 다시 읽어 정확한 문자열로 보존합니다. `Body`로 unknown field·null·누락·큰 JSON 정수를 구분할 수 있습니다.

성공 응답의 object·integer hard_limit이 잘못되면 오류입니다. `QuotaResponseError`에 status·header·원문 byte·decode/read 원인을 보존하며, 성공한 POST를 decode 실패 때문에 다시 보내지 않습니다. HTTP 404는 `resource.ErrNotFound`로도 확인할 수 있고 HTTP 원문과 403·409·500·취소·timeout 원인은 `errors.As`/`errors.Is`에 보존됩니다.

## HTTP와 남은 범위

Create는 선택한 ResourceBase의 `/quotas`에 flat JSON `{project_id, resource, hard_limit, ...}`를 보내고 native 정책대로 201만 허용합니다. 생성된 `API.Create(ctx, CreateOpts, ...CreateOption)`는 기존 native 계약을 그대로 유지합니다. native convenience와 달리 scope는 고정 대상·생성 시 옵션 snapshot·정확한 정수·응답 metadata를 제공합니다.

Get·Update·Delete는 `/quotas/{project_id}/{resource}`를 사용합니다. Get은 200, flat JSON object를 보내는 PATCH Update는 202, request body가 없는 Delete는 204만 허용합니다. JSON Patch operation 배열을 보내지 않습니다. Delete는 삭제 metadata를 반환하고 후속 GET을 하지 않습니다.

Magnum은 explicit quota가 없는 프로젝트의 Get에 deployment hard limit을 반환할 수 있습니다. 이 응답에는 row ID 또는 resource가 없을 수 있으며 `Body`와 native 응답 필드에 없는 값을 만들지 않습니다. 조회 요청 대상은 `RequestProjectID`·`RequestResource`로 별도 제공됩니다. Delete는 explicit override를 삭제하며 다음 조회의 기본값 처리는 서버가 결정합니다.

Delete의 기본 404 정책은 엄격합니다. `WithDeleteIgnoreMissing(true)`는 404만 숨기며 뒤의 false 옵션으로 엄격하게 되돌릴 수 있습니다. 403·500 등의 오류는 숨기지 않습니다. 이 정책은 pinned Python의 quota 기본값에 대응한다는 뜻이 아닙니다. 해당 Python quota API가 존재하지 않습니다.

## 페이지 목록

```go
import (
    "context"
    "fmt"
    "github.com/JSYoo5B/go-openstacksdk/containerinfra/v1/quotas"
)

func listQuotas(ctx context.Context, api *quotas.API) error {
    for quota, err := range api.List(ctx,
        quotas.WithListOptions(quotas.ListOpts{Limit: 100}),
        quotas.WithListAllTenants(true),
    ) {
        if err != nil { return err }
        fmt.Println(quota.ID, quota.ProjectID, quota.Resource, quota.HardLimit)
    }
    return nil
}
```

List는 `/quotas`의 `quotas` 배열과 top-level `next` 링크를 사용하며, All은 같은 iterator를 모읍니다. collection 행의 `RequestProjectID`·`RequestResource`는 빈 값입니다. 실제 서버 identity는 `ID`·`ProjectID`·`Resource`에서 읽고 unknown/null 필드·큰 숫자·페이지별 HTTP header/status를 보존합니다.

기본 sort는 `id`/`asc`이며 Limit 0은 페이지 크기를 생략해 서버 기본값을 사용합니다. 서버가 설정한 최대 크기로 limit을 줄이면 첫 continuation에서 그 크기를 채택하고 이후 고정합니다. Marker는 project ID가 아닌 양의 정수 quota row ID이며 문자열로 정확하게 전달합니다. `AllTenants == nil`은 서버 기본 false를 사용하고, `WithListAllTenants(false/true)`는 값을 명시합니다. 권한은 서버가 검증합니다. `WithListOptions`의 pointer는 생성 시 snapshot을 보관하고 반복 적용할 때 복사합니다. core query는 concrete 옵션으로만 지정하며 `WithListQuery`는 추가 필드를 전달합니다.

Magnum의 next 링크는 `all_tenants`를 누락할 수 있습니다. SDK는 원래 flag와 확장 query를 유지하며 marker만 전진시킵니다. 링크가 origin·collection path·sort·tenant flag·확장 query를 바꾸거나 page limit을 늘리면 거절합니다. 반복 marker는 cycle 오류입니다. `break`는 남은 행 decode와 다음 페이지 요청을 중단하며, 취소와 페이지 오류는 원래 원인·수락된 응답 증거를 보존합니다.

quota 이름 Find·상태 Wait는 제공하지 않습니다. Python Resource의 dirty-state·자동 commit·cache는 별도 계약이며, 추가 REST 구현은 pinned Python에 없는 quota API에 대한 지원입니다. server SHA별 capability inventory는 아직 별도로 구축하지 않았습니다.

근거: [pinned native Create](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/containerinfra/v1/quotas/requests.go), [native quota model](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/containerinfra/v1/quotas/results.go), [pinned Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/container_infrastructure_management/v1/_proxy.py), [공식 Magnum quota API](https://docs.openstack.org/api-ref/container-infrastructure-management/#magnum-quota-api), [server quota controller](https://github.com/openstack/magnum/blob/master/magnum/api/controllers/v1/quota.py), [collection next](https://github.com/openstack/magnum/blob/master/magnum/api/controllers/v1/collection.py), [server limit validation](https://github.com/openstack/magnum/blob/master/magnum/api/utils.py). 이 추가 REST 계약은 별도 server SHA에 고정한 inventory가 아니며 native·Python 선언의 완전 지원 수치에 포함하지 않습니다. [생성·인증](../../../api/magnum_quota_create_test.go), [조회·갱신·삭제·오류](../../../api/magnum_quota_operations_test.go), [페이지 목록](../../../api/magnum_quota_list_test.go) 테스트가 동작을 검증합니다.
