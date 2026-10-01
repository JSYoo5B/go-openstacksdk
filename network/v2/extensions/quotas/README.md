# 프로젝트별 Neutron quota

`InProject(ctx, projectRef)`는 프로젝트를 한 번 확정하고 `ProjectQuotaScope`를 반환합니다. Scope의 `Get`, `Defaults`, `Detail`, `Update`, `Delete`는 프로젝트 quota singleton을 다룹니다. `Delete`는 quota override를 지워 기본값으로 되돌리는 작업이며 프로젝트를 삭제하지 않습니다.

| Python openstacksdk | Go scope |
|---|---|
| `conn.network.get_quota(project_id)` | `scope.Get(ctx)` |
| `conn.network.get_quota_default(project_id)` | `scope.Defaults(ctx)` |
| `conn.network.get_quota(project_id, details=True)` | `scope.Detail(ctx)` |
| `conn.network.update_quota(project_id, ports=0, check_limit=False)` | `scope.Update(ctx, quotas.UpdateOpts{Port: &zero}, quotas.WithUpdateCheckLimit(false))` |
| `conn.network.delete_quota(project_id, ignore_missing=False)` | `scope.Delete(ctx)` |
| `ignore_missing=True` | `scope.Delete(ctx, quotas.WithDeleteIgnoreMissing(true))` |
| Keystone 프로젝트 이름 해석 | `api.InProject(ctx, resource.Name(name), quotas.WithIdentityClient(identityClient))` |
| 인증 프로젝트 ID 전달 | `api.CurrentProject(ctx)` |
| `conn.network.quotas()` | `api.ListProjects(ctx)` / `api.AllProjects(ctx)` |

```go
func manageQuotas(ctx context.Context, networkClient *gophercloud.ServiceClient) error {
    api := quotas.New(networkClient)
    scope, err := api.InProject(ctx, resource.ID("project-id"))
    if err != nil { return err }

    limits, err := scope.Get(ctx)
    if err != nil { return err }
    fmt.Println(scope.ProjectID(), limits.Network, limits.Port)

    defaults, err := scope.Defaults(ctx)
    if err != nil { return err }
    fmt.Println(defaults.Network, defaults.Port)

    usage, err := scope.Detail(ctx)
    if err != nil { return err }
    fmt.Println(usage.Network.Used, usage.Network.Reserved, usage.Network.Limit)

    zero, unlimited := 0, -1
    updated, err := scope.Update(ctx, quotas.UpdateOpts{
        Port: &zero, Network: &unlimited,
    }, quotas.WithUpdateCheckLimit(false))
    if err != nil { return err }
    fmt.Println(updated.Port, updated.Header.Get("X-Openstack-Request-Id"))

    reset, err := scope.Delete(ctx)
    if err != nil { return err }
    fmt.Println(reset.StatusCode, reset.Header.Get("X-Openstack-Request-Id"))
    return nil
}
```

예제의 import는 `context`, `fmt`, `github.com/gophercloud/gophercloud/v2`, `gophercloudsdk/network/v2/extensions/quotas`, `gophercloudsdk/resource`입니다. `New`에는 인증과 endpoint가 설정된 Network `ServiceClient`를 전달합니다. SDK가 builder와 결과 처리를 관리합니다.

Connection을 사용하면 `conn.NetworkProjectQuotas(ctx, resource.ID("project-id"))` 또는 `conn.CurrentNetworkProjectQuotas(ctx)`로 같은 scope를 얻습니다. 이름 reference는 Connection의 Identity client로 해석합니다.

## 프로젝트와 인증

`resource.ID`는 Keystone 요청 없이 고정합니다. `resource.Name`은 별도의 Keystone v3 client로 모든 목록 페이지에서 정확히 찾습니다. UUID처럼 생긴 이름도 명시적인 이름으로 처리하며 중복 이름은 `resource.ErrAmbiguous`, 없는 이름은 `resource.ErrNotFound`입니다. 잘못된 반환 ID와 Keystone 권한 오류를 숨기지 않고, Neutron client나 endpoint로 Identity 요청을 보내지 않습니다. 이후 quota 작업에서 이름을 다시 해석하지 않습니다.

`CurrentProject`는 ProviderClient에 기록된 Keystone v3 인증의 project 또는 v2 token의 tenant ID만 사용합니다. 수동 token·system/domain/unscoped 인증·프로젝트 없는 결과에는 `resource.ErrUnsupported`, 잘못된 ID에는 `resource.ErrInvalidOption`을 반환합니다. endpoint나 token 문자열에서 ID를 추측하거나 인증을 갱신하지 않습니다. 이후 재인증이 다른 프로젝트를 기록해도 기존 scope의 프로젝트는 바뀌지 않습니다. Python network proxy의 quota 인자는 필수이며 이 진입점은 Go의 편의 기능입니다.

## 값과 응답

`UpdateOpts`는 Gophercloud v2.15.0의 concrete alias입니다. `*int`가 nil이면 생략하고, 0은 그대로 보내며, -1은 무제한입니다. -1보다 작은 typed limit은 HTTP 전에 거절합니다. `WithUpdateOptions`가 typed 옵션을 대체하고 뒤 옵션이 우선합니다. Scope는 호출 시 pointer 값을 원문 JSON으로 직렬화하므로 큰 정수를 반올림하지 않으며 재인증 이후에도 같은 요청 snapshot을 사용합니다.

`WithQuotaOptions(opts)`는 옵션 **생성 시점**에 typed limit pointer 값을 복사하고, 적용할 때마다 새 pointer를 제공합니다. 생성 뒤 원본을 바꾸거나 한 호출의 typed 옵션을 변경해도 같은 옵션을 재사용하는 다음 요청은 처음 값을 사용합니다. 기존 `WithUpdateOptions`는 생성 시 deep copy를 하지 않으므로 옵션을 적용하는 호출 시점의 pointer 값을 사용합니다.

`WithUpdateCheckLimit(false)`는 false를 명시적으로 보내며, 생략하면 `check_limit`을 추가하지 않습니다. 마지막 옵션이 우선합니다. 이 옵션은 scope의 `Update` 전용이고 하위 `API.Update`에서는 지원하지 않습니다. 서버가 해당 quota usage-check 정책과 확장 지원을 판단합니다. 현재 [Neutron API reference](https://docs.openstack.org/api-ref/network/v2/#quotas-extension-quotas)는 `check_limit`을 deprecated로 설명하며 `force`도 문서화합니다. 별도 quota·flag 확장은 `WithUpdateField("vendor_quota", value)`처럼 전달하고 서버가 검증합니다.

`WithUpdateField`는 옵션 생성 시 값을 JSON으로 복사합니다. Native core limit 필드와 `check_limit`을 확장 필드로 덮어쓰면 HTTP 전에 `resource.ErrInvalidOption`입니다. Query·header·알 수 없는 argument 확장은 받지 않습니다. 선택한 ServiceClient의 endpoint·ResourceBase·헤더·버전·ProviderClient 설정은 변경하지 않습니다.

Get·Defaults·Update의 `QuotaResource`는 native `Quota`, 고정 요청 대상 `ProjectID`, 전체 quota 객체의 `Body`, 복사된 `Header`, `StatusCode`를 제공합니다. 응답에 다른 `project_id`가 있어도 대상은 바뀌지 않습니다. Body에서 알 수 없는 필드·큰 정수·null·빈 값·누락을 구분할 수 있습니다. Detail의 `QuotaDetailResource`는 native `QuotaDetailSet`의 `Used/Reserved/Limit`과 nested 원문을 보존합니다. Native Neutron 호환 처리에 따라 문자열 `reserved` 정수도 읽고, 잘못된 문자열은 원래 decode 오류를 반환합니다. quota 객체가 없거나 null·배열·scalar이거나 알려진 필드의 타입이 잘못되면 오류입니다.

`Defaults`는 `/quotas/{project}/default`에서 기본 limit을 조회합니다. 현재 override를 반환하는 `Get`이나 사용량을 반환하는 `Detail`과 별도의 HTTP 작업입니다. 기본값 응답에 프로젝트 ID가 없어도 scope의 요청 대상 `ProjectID`를 유지하며, response body에 없는 ID를 추가하지 않습니다.

## HTTP와 지원 범위

GET·PUT은 native와 같은 200 성공 정책입니다. Detail은 Gophercloud의 `/quotas/{project}/details.json`을 사용하며 pinned Python은 `/details` 경로를 사용합니다. 서버의 `quota_details` 확장 지원을 가정하지 않습니다. DELETE는 native의 202·204를 모두 보존하고 metadata만 반환하며 후속 GET을 보내지 않습니다. 공식 API가 문서화하는 정상 DELETE 응답은 204입니다.

Scope의 DELETE 기본 404 정책은 엄격하며 `resource.ErrNotFound`와 원본 HTTP 원인을 보존합니다. Python `delete_quota`의 기본 `ignore_missing=True`에 대응하려면 `WithDeleteIgnoreMissing(true)`를 지정하세요. 뒤의 false 옵션으로 다시 엄격하게 설정할 수 있고, 403 등 다른 오류는 무시하지 않습니다. HTTP 오류의 code·header·body와 decode·부모 context 취소·timeout 원인은 `errors.As`와 `errors.Is`로 확인합니다.

이 단위는 native Get·GetDetail·Update·Delete, Python의 defaults endpoint, 별도 quota collection 조회를 제공합니다. Server 조회 query, Resource 입력·dirty-state·자동 commit은 별도 계약입니다. 존재하지 않는 생성·이름 Find·상태 Wait를 singleton에 추가하지 않습니다. 하위 `API.Get/GetDetail/Update/Delete`의 native 반환 계약은 유지합니다.

근거: pinned [Gophercloud requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/networking/v2/extensions/quotas/requests.go), [Python proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py), [Python quota resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/quota.py), [Neutron quota details API](https://docs.openstack.org/api-ref/network/v2/#quotas-details-extension-quota-details). 계약 테스트는 [HTTP·입력·오류](../../../../api/neutron_project_quotas_contracts_test.go), [프로젝트 해석·인증](../../../../api/neutron_project_quotas_projects_test.go), [기본 quota 조회](../../../../api/neutron_quota_defaults_test.go)에 있습니다.

## Quota override 목록

`api.ListProjects(ctx)`는 `/quotas`의 단일 배열을 lazy iterator로 반환합니다. `api.AllProjects(ctx)`는 iterator를 소비해 slice로 수집하며, 오류가 있으면 부분 slice 대신 오류를 반환합니다. 목록은 quota override가 저장된 프로젝트만 포함하며 각 행의 limit에는 기본값이 합쳐집니다. 모든 Keystone 프로젝트를 열거하려면 Identity project API를 사용하세요.

```go
func listQuotaOverrides(ctx context.Context, networkClient *gophercloud.ServiceClient) error {
    api := quotas.New(networkClient)
    for quota, err := range api.ListProjects(ctx, quotas.WithListProjectID("project-id"), quotas.WithListMaxItems(1)) {
        if err != nil { return err }
        fmt.Println(quota.ProjectID, quota.Network, quota.Port)
        fmt.Println(quota.Header.Get("X-Openstack-Request-Id"), quota.StatusCode)
    }
    return nil
}
```

이 fragment의 import는 `context`, `fmt`, `github.com/gophercloud/gophercloud/v2`, `gophercloudsdk/network/v2/extensions/quotas`입니다. 목록을 모두 수집하려면 `values, err := api.AllProjects(ctx)`를 사용합니다.

`WithListProjectID(id)`는 정확한 프로젝트 ID의 행만 소비하는 **로컬 필터**이며 Keystone 조회나 server query를 보내지 않습니다. `WithListMaxItems(n)`은 양의 최대 소비 수이며 HTTP 응답 크기를 제한하지 않습니다. 같은 옵션이 반복되면 마지막 값이 우선합니다. iterator 생성은 HTTP나 옵션 검증을 수행하지 않고 첫 소비 시 검증·GET을 수행합니다. 호출자의 옵션 slice는 생성 시 복사합니다. `break`나 최대 소비 수에 도달하면 이후 행을 decode하지 않으며 context 취소는 방문하는 행 사이에서도 확인합니다. 로컬 필터가 제외하는 방문 행도 먼저 검증하므로 잘못된 응답을 숨기지 않습니다.

각 행은 `QuotaResource`이며 native limits와 전체 행 `Body`, 복사된 `Header`, 실제 `StatusCode`를 보존합니다. 목록의 `ProjectID`는 검증한 `project_id`이고, 이것이 없을 때만 deprecated `tenant_id`를 사용합니다. 두 필드가 모두 있으면 유효한 같은 ID여야 하며 빈 값·null·잘못된 타입·안전하지 않은 경로 문자·서로 다른 ID·식별자 누락은 오류입니다. `Body`에는 응답이 제공하지 않은 identity를 추가하지 않으며 unknown limit·큰 정수·null·누락을 구분할 수 있습니다. 행과 호출 간 원문 byte slice와 header를 공유하지 않습니다. `quotas`가 없거나 null·object·scalar이거나 방문한 행이 object가 아니거나 native limit 타입이 잘못되면 오류입니다. 빈 `quotas: []`는 정상입니다.

Pinned Python `quotas(**query)`는 quota별 query를 지원하지 않는다고 설명합니다. 공식 API reference에는 일반 pagination/filter 문장이 있지만 확인한 Neutron `QuotaSetsController.index`와 DB driver는 query를 읽지 않고 quota override 전체 배열을 한 번 반환합니다. 따라서 SDK는 server fields/filter/limit/marker/sort 옵션이나 자동 다음 페이지 추측을 제공하지 않습니다. Python inherited `Resource.list`의 일반 pagination 재시도와 adapter 동작을 구현한 것으로 간주하지 않습니다.

응답에 인식 가능한 nonempty `quotas_links`의 `rel=next`/`href` 또는 `links.next` 문자열이 있으면 `resource.ErrUnsupported`를 반환합니다. 지원 근거가 없는 continuation을 따라가거나 부분 배열을 전체 결과로 반환하지 않습니다. 빈 next·다른 link relation·인식되지 않는 metadata 구조는 허용합니다. GET 성공 코드는 200이며 HTTP 오류의 status/header/body와 context/decode 원인은 보존합니다.

서버 근거: [QuotaSetsController](https://github.com/openstack/neutron/blob/cadc448a995a30d9b18514089acd7b1770142032/neutron/extensions/quotasv2.py), [DbQuotaDriver](https://github.com/openstack/neutron/blob/cadc448a995a30d9b18514089acd7b1770142032/neutron/db/quota/driver.py), [Neutron quota API](https://docs.openstack.org/api-ref/network/v2/#quotas-extension-quotas). 계약 테스트는 [단일 배열·로컬 옵션·응답·오류·취소](../../../../api/neutron_quotas_list_test.go)에 있습니다.
