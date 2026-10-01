# Magnum 프로젝트·리소스 quota

Magnum quota의 요청 대상은 프로젝트 ID와 리소스 이름의 쌍입니다. `InProject` 또는 `CurrentProject`로 프로젝트를 한 번 확정하고, `ForResource(quotas.Cluster)`로 quota를 선택합니다. scope 구성은 quota 조회를 보내지 않습니다.

| 작업 | Go |
|---|---|
| 프로젝트 ID 고정 | `api.InProject(ctx, resource.ID(projectID))` |
| Keystone 프로젝트 이름 해석 | `api.InProject(ctx, resource.Name(name), quotas.WithIdentityClient(identityClient))` |
| 기록된 인증 프로젝트 선택 | `api.CurrentProject(ctx)` |
| Cluster quota 선택 | `project.ForResource(quotas.Cluster)` |
| hard limit 0으로 생성 | `scope.Create(ctx, quotas.WithHardLimit(0))` |
| concrete 옵션 snapshot | `quotas.WithQuotaCreateOptions(quotas.QuotaCreateOpts{HardLimit: &limit})` |
| 추가 서버 필드 | `quotas.WithQuotaCreateField("vendor", value)` |

pinned openstacksdk의 `container_infrastructure_management.v1.Proxy`에는 quota model과 quota proxy method가 없습니다. 따라서 이 표에는 존재하지 않는 Python quota 호출을 대응시키지 않습니다. Python에서 이 서비스의 quota를 사용하려면 REST 호출 또는 별도 Magnum client가 필요합니다. Go의 새 scope는 Gophercloud의 native quota Create 계약을 SDK가 직접 관리합니다.

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
    return nil
}
```

예제 import는 `context`, `fmt`, `github.com/gophercloud/gophercloud/v2`, `gophercloudsdk/containerinfra/v1/quotas`, `gophercloudsdk/resource`입니다. 인증과 endpoint가 설정된 Magnum ServiceClient를 전달합니다. 서버가 해당 확장을 제공할 때만 `vendor` 필드를 보낼 수 있습니다.

## 프로젝트와 resource

`resource.ID`는 Keystone 요청 없이 scope를 만듭니다. `resource.Name`은 별도의 Identity v3 client를 사용해 정확한 이름을 해석하며, 이름 없음·중복·Identity HTTP 오류를 구분합니다. 프로젝트 ID는 응답의 다른 `project_id`나 나중에 바뀐 인증 결과로 교체하지 않습니다.

`CurrentProject`는 ProviderClient에 기록된 Keystone v3 project 또는 v2 token tenant를 읽습니다. 인증 결과 없는 수동 token·system/domain/unscoped 인증은 `resource.ErrUnsupported`입니다. endpoint에서 프로젝트를 추측하거나 인증을 갱신하지 않습니다.

`ResourceName`은 대소문자를 보존합니다. `Cluster`가 문서화된 Magnum resource입니다. 다른 안전한 단일 이름도 extension resource로 지정할 수 있으며 서버가 지원 여부를 판단합니다. quota 응답의 row `id`는 project ID 또는 resource 이름을 대신하지 않습니다.

## 값과 응답

Create는 hard limit을 명시적으로 요구합니다. `WithHardLimit(0)`은 0을 생략하지 않으며, `QuotaCreateOpts.HardLimit`이 nil인 상태는 HTTP 전에 `resource.ErrInvalidOption`입니다. SDK는 관리자 limit을 임의의 기본값으로 선택하지 않습니다. limit 값의 서버 유효성 규칙과 권한은 Magnum이 검증하며, 다른 서비스의 `-1=무제한` 규칙을 적용하지 않습니다.

`WithQuotaCreateOptions`는 생성 시 pointer 값을 복사하고 매번 새 pointer로 적용합니다. `WithQuotaCreateField`는 생성 시 JSON snapshot을 보관합니다. 옵션은 나중 옵션이 우선하며 `hard_limit`, `project_id`, `resource`, `id`를 extension으로 덮어쓸 수 없습니다. query·header·알 수 없는 argument 옵션도 HTTP 전에 거절합니다.

POST body는 최종 JSON을 한 번 직렬화하여 재인증·재시도에서도 같은 snapshot을 보냅니다. 플랫폼 int 범위에서 2^53보다 큰 hard limit을 float64 반올림 없이 보존합니다. scoped 요청은 다른 method·path·query·origin으로의 HTTP redirect를 거절하고 선택한 ServiceClient 설정을 변경하지 않습니다.

`QuotaResource`는 native `Quotas`, 고정 요청 대상 `RequestProjectID`·`RequestResource`, 원문 필드 `Body`, 복사된 `Header`, 실제 `StatusCode`를 제공합니다. `ProjectID`와 `Resource`는 서버 응답 값이며 요청 대상을 대체하지 않습니다. native decoder가 float64로 변환하는 숫자 row ID는 원문 JSON에서 다시 읽어 정확한 문자열로 보존합니다. `Body`로 unknown field·null·누락·큰 JSON 정수를 구분할 수 있습니다.

성공 응답의 object·integer hard_limit이 잘못되면 오류입니다. `QuotaResponseError`에 status·header·원문 byte·decode/read 원인을 보존하며, 성공한 POST를 decode 실패 때문에 다시 보내지 않습니다. HTTP 404는 `resource.ErrNotFound`로도 확인할 수 있고 HTTP 원문과 403·409·500·취소·timeout 원인은 `errors.As`/`errors.Is`에 보존됩니다.

## HTTP와 남은 범위

Create는 선택한 ResourceBase의 `/quotas`에 flat JSON `{project_id, resource, hard_limit, ...}`를 보내고 native 정책대로 201만 허용합니다. 생성된 `API.Create(ctx, CreateOpts, ...CreateOption)`는 기존 native 계약을 그대로 유지합니다. native convenience와 달리 scope는 고정 대상·생성 시 옵션 snapshot·정확한 정수·응답 metadata를 제공합니다.

현재 scope 구현은 quota 생성까지입니다. 공식 Magnum API의 quota Get·PATCH Update·DELETE·페이지 목록은 별도 구현이 남아 있습니다. quota 이름 Find·상태 Wait는 제공하지 않으며 Python Resource의 dirty-state·자동 commit·cache와 같다고 주장하지 않습니다. pinned Python에 quota 선언이 없다는 사실도 Magnum REST 전체 지원을 의미하지 않습니다.

근거: [pinned native Create](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/containerinfra/v1/quotas/requests.go), [native quota model](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/containerinfra/v1/quotas/results.go), [pinned Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/container_infrastructure_management/v1/_proxy.py), [공식 Magnum quota API](https://docs.openstack.org/api-ref/container-infrastructure-management/#magnum-quota-api). [HTTP·옵션·인증·오류 테스트](../../../api/magnum_quota_create_test.go)가 동작을 검증합니다.
