# Octavia 프로젝트 quota

`InProject(ctx, projectRef)`는 프로젝트를 한 번 확정하고 quota singleton을 다루는 `ProjectQuotaScope`를 반환합니다. `Reset`은 quota override를 지우고 deployment 기본값으로 되돌리는 연산입니다.

| Python openstacksdk | Go |
|---|---|
| `conn.load_balancer.get_quota(project_id)` | `scope.Get(ctx)` |
| `conn.load_balancer.get_quota_default()` | `api.Defaults(ctx)` 또는 `scope.Defaults(ctx)` |
| `conn.load_balancer.update_quota(project_id, listeners=0)` | `scope.Update(ctx, quotas.UpdateOpts{Listener: &zero})` |
| `conn.load_balancer.delete_quota(project_id, ignore_missing=False)` | `scope.Reset(ctx)` |
| `ignore_missing=True` | `scope.Reset(ctx, quotas.WithResetIgnoreMissing(true))` |
| `conn.load_balancer.quotas(**query)` | `api.ListProjects(ctx, quotas.WithProjectListOptions(opts))` |
| Keystone 프로젝트 이름 해석 | `api.InProject(ctx, resource.Name(name), quotas.WithIdentityClient(identityClient))` |
| 인증된 프로젝트의 quota | `api.CurrentProject(ctx)` |

```go
func manageQuotas(ctx context.Context, client *gophercloud.ServiceClient) error {
    api := quotas.New(client)
    scope, err := api.InProject(ctx, resource.ID("project-id"))
    if err != nil { return err }

    limits, err := scope.Get(ctx)
    if err != nil { return err }
    fmt.Println(limits.ProjectID, limits.Loadbalancer, limits.Listener)

    defaults, err := api.Defaults(ctx)
    if err != nil { return err }
    fmt.Println(defaults.Loadbalancer, defaults.Listener)

    zero, unlimited := 0, -1
    updated, err := scope.Update(ctx, quotas.UpdateOpts{
        Listener: &zero, Loadbalancer: &unlimited,
    }, quotas.WithDefaultLimit(quotas.LimitMembers))
    if err != nil { return err }
    fmt.Println(updated.Header.Get("X-Openstack-Request-Id"))

    for quota, err := range api.ListProjects(ctx, quotas.WithProjectListOptions(
        quotas.ProjectListOpts{Limit: 100},
    )) {
        if err != nil { return err }
        fmt.Println(quota.ProjectID, quota.Pool)
    }

    _, err = scope.Reset(ctx)
    return err
}
```

예제 import는 `context`, `fmt`, `github.com/gophercloud/gophercloud/v2`, `github.com/JSYoo5B/gophercloudsdk/loadbalancer/v2/quotas`, `github.com/JSYoo5B/gophercloudsdk/resource`입니다. 인증과 endpoint가 설정된 LoadBalancer `ServiceClient`를 전달하며, SDK가 builder와 응답 처리를 관리합니다.

## 프로젝트와 기본값

`resource.ID`는 Keystone 요청 없이 scope를 만듭니다. `resource.Name`은 별도의 Identity v3 client로 모든 목록 페이지에서 정확한 이름을 찾고, 없는 이름·중복 이름·Keystone 권한 오류를 구분합니다. 프로젝트 ID는 이후 GET·PUT·DELETE에서 고정되며 응답의 다른 `project_id`나 재인증 결과로 바뀌지 않습니다.

`CurrentProject`는 ProviderClient에 기록된 Keystone v3 project 또는 v2 token tenant만 사용합니다. 수동 token·system/domain/unscoped 인증·프로젝트 없는 결과는 `resource.ErrUnsupported`입니다. endpoint에서 ID를 추측하거나 인증을 갱신하지 않습니다. `defaults`는 global endpoint와 충돌하므로 명시적 프로젝트 ID로 사용하면 HTTP 전에 거절합니다.

`Defaults`는 `/lbaas/quotas/defaults`의 **전역 기본값**을 반환합니다. `DefaultQuotaResource`에 프로젝트 ID를 만들지 않습니다. `scope.Defaults`도 같은 global 조회이며 해당 프로젝트의 override를 조회하는 `scope.Get`과 구분됩니다.

## 값과 응답

`UpdateOpts`는 Gophercloud v2.15.0 concrete alias입니다. limit pointer가 nil이면 생략하고, 0은 그대로 보내며, -1은 무제한입니다. -1보다 작은 typed limit은 HTTP 전에 거절합니다. `WithDefaultLimit`은 선택한 limit을 JSON null로 보내 기본값을 상속합니다. 같은 limit에 typed 값과 null 옵션을 동시에 주면 오류이며 같은 null 옵션의 반복은 허용합니다.

`WithQuotaOptions(opts)`는 옵션 생성 시 모든 limit pointer 값을 복사하고 매번 새 pointer로 적용합니다. 원본 값이나 이전 호출에서 적용된 pointer를 바꿔도 재사용한 옵션의 값은 유지됩니다. 기존 `WithUpdateOptions`는 호출 시점 pointer 값을 사용합니다. Scope는 PUT 전 최종 JSON을 한 번 직렬화하므로 재인증에서도 같은 snapshot을 보내며 큰 정수를 반올림하지 않습니다.

`WithUpdateField("vendor_quota", value)`는 생성 시 JSON snapshot을 보관합니다. native core 필드·legacy alias·프로젝트 identity는 확장 필드로 덮어쓸 수 없습니다. 임의 query/header/알 수 없는 argument도 HTTP 전에 거절합니다. 확장 지원 여부는 서버가 판단합니다.

`QuotaResource`는 native `Quota`, scope의 고정 요청 대상 `ProjectID`, quota 객체의 원문 `Body`, 복사된 `Header`, `StatusCode`를 제공합니다. `DefaultQuotaResource`는 같은 정보에서 프로젝트 ID만 제외합니다. native decode는 `load_balancer`/`health_monitor`와 canonical `loadbalancer`/`healthmonitor`를 읽으며 non-null canonical 값이 우선합니다. canonical이 null이면 non-null legacy 값으로 fallback하고, 양쪽이 없거나 null이면 native 정수 값은 0입니다. 상속·0·누락·alias 충돌·알 수 없는 필드의 구분에는 `Body`를 사용합니다.

quota 객체가 없거나 null·배열·scalar이거나 알려진 필드 타입이 잘못되면 오류입니다. 성공 상태의 잘못된 JSON도 `QuotaResponseError`에 status·header·원문 byte·decode 원인을 보존하며 이미 수락된 PUT을 decode 실패 때문에 재시도하지 않습니다. HTTP 원인과 취소·timeout은 `errors.As`/`errors.Is`로 확인할 수 있습니다.

## 목록

`ListProjects`는 lazy iterator이며 `AllProjects`는 전체 목록을 수집합니다. `ProjectListOpts`는 `ProjectID`, 반복되는 `Fields`, `Limit`, `Marker`, `PageReverse`를 제공합니다. PageReverse는 [Octavia의 대소문자 구분](https://docs.openstack.org/octavia/latest/_modules/octavia/api/common/pagination.html)에 맞춰 `True`/`False`로 전송하고 nil이면 생략합니다. `WithProjectListOptions`는 fields와 bool pointer를 생성 시 복사하고 재사용할 때마다 다시 복사합니다. iterator는 생성 시 옵션 배열도 복사합니다. limit 0은 query 생략, 음수는 오류입니다.

페이지의 `quotas_links` 중 `next`만 따라가며 상대 URL·빈 페이지를 지원합니다. 서버 링크에서 빠진 최초 query 옵션은 다음 페이지에도 유지하고, 링크에 명시된 paging 값은 우선합니다. iterator break는 다음 항목 decode와 다음 GET을 중단합니다. 취소·pagination cycle·다른 origin 링크·서로 다른 복수 next 링크를 거절합니다. 응답 item의 `project_id`를 읽고 fields projection으로 빠진 ID는 빈 값으로 둡니다. 필터 값이나 현재 인증 프로젝트를 응답 ID로 만들지 않습니다.

## HTTP와 지원 범위

SDK scope와 collection은 pinned Python 및 [공식 Octavia quota API](https://docs.openstack.org/api-ref/load-balancer/v2/#quotas)의 `/lbaas/quotas` 경로를 사용합니다. 선택한 ServiceClient의 ResourceBase가 이미 `/lbaas/`로 끝나면 prefix를 중복하지 않습니다. Gophercloud v2.15.0 quota URL에는 `lbaas/`가 빠져 있어 **생성된 `API.Get/Update/Delete`의 기존 `/quotas` 계약과 경로가 다릅니다**. 이 보정은 새 scope/collection 연산에만 적용하며 client 설정을 변경하지 않습니다.

GET·Defaults·ListProjects 성공은 200만 허용합니다. PUT은 native 호환 정책의 200·202, Reset은 202·204를 유지합니다. 공식 API의 PUT·DELETE 성공은 각각 202·204입니다. Reset은 metadata만 반환하고 후속 GET을 보내지 않습니다.

Reset 기본 404 정책은 엄격합니다. Python의 `ignore_missing=True` 기본값에 대응하려면 `WithResetIgnoreMissing(true)`를 지정하고, 뒤의 false 옵션으로 다시 엄격하게 설정할 수 있습니다. 403 등 다른 오류는 숨기지 않습니다.

이 구현은 quota 조회·전역 defaults·갱신·override reset·collection 목록을 제공합니다. Python Resource 입력·dirty-state·자동 commit·cache·임의 query 계약까지 동일하다는 의미는 아닙니다. quota singleton에 생성·이름 Find·상태 Wait를 추가하지 않습니다.

근거: [native requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/loadbalancer/v2/quotas/requests.go), [native URL](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/loadbalancer/v2/quotas/urls.go), pinned [Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/load_balancer/v2/_proxy.py), [Python quota resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/load_balancer/v2/quota.py). 테스트는 [HTTP·옵션·오류](../../../api/octavia_project_quotas_contracts_test.go), [프로젝트·인증](../../../api/octavia_project_quotas_projects_test.go), [목록](../../../api/octavia_project_quotas_list_test.go)에 있습니다.
