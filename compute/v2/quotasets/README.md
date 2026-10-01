# 프로젝트별 Nova quota

Quota는 프로젝트당 하나의 limit 집합입니다. `InProject(ctx, projectRef)`가 프로젝트를 한 번 고정하며, 결과는 Get·Detail·Update·Reset을 제공합니다. List·Find·Wait나 일반 리소스 Delete는 제공하지 않습니다. Reset은 프로젝트의 quota override를 지워 Nova 기본값으로 되돌리는 작업입니다.

| openstacksdk | Go scope |
|---|---|
| `conn.compute.get_quota_set(project)` | `scope.Get(ctx)` |
| `conn.compute.get_quota_set(project, usage=True)` | `scope.Detail(ctx)` |
| `conn.compute.update_quota_set(project, cores=0, force=False)` | `scope.Update(ctx, quotasets.UpdateOpts{Cores: &zero}, quotasets.WithUpdateForce(false))` |
| `conn.compute.revert_quota_set(project)` | `scope.Reset(ctx)` |
| `conn.get_compute_quotas("tenant")`의 Keystone 이름 조회 | `InProject(ctx, resource.Name("tenant"), quotasets.WithIdentityClient(identityClient))` |
| `conn.current_project_id`를 quota 조회에 전달 | `api.CurrentProject(ctx)` |

```go
// context.Context ctx, Nova *gophercloud.ServiceClient computeClient을
// 사용하는 함수 안에서. resource와 quotasets는 gophercloudsdk 패키지입니다.
api := quotasets.New(computeClient)
scope, err := api.InProject(ctx, resource.ID("project-id"))
if err != nil { return err }
limits, err := scope.Get(ctx)
if err != nil { return err }
fmt.Println(scope.ProjectID(), limits.Cores, limits.Header.Get("X-Openstack-Request-Id"))

usage, err := scope.Detail(ctx)
if err != nil { return err }
fmt.Println(usage.Cores.Limit, usage.Cores.InUse, usage.Cores.Reserved)

zero, unlimited := 0, -1
updated, err := scope.Update(ctx, quotasets.UpdateOpts{
    Cores: &unlimited,
    Instances: &zero,
}, quotasets.WithUpdateForce(false))
if err != nil { return err }
fmt.Println(updated.Cores, updated.Instances)

reset, err := scope.Reset(ctx)
if err != nil { return err }
fmt.Println(reset.Header.Get("X-Openstack-Request-Id"))
```

`resource.ID`는 Keystone 사전 조회 없이 URL의 project ID로 사용합니다. `resource.Name`은 `WithIdentityClient`로 전달한 **Keystone v3** client의 project collection에서 전체 페이지를 읽고 정확히 해석합니다. Compute client로 Identity endpoint를 호출하지 않습니다. 중복 이름·미존재·권한 오류를 숨기지 않으며, 작업 중 이름을 다시 찾거나 응답의 `id`로 대상 프로젝트를 바꾸지 않습니다.

`CurrentProject(ctx)`는 ProviderClient가 보관한 인증 결과의 Keystone v3 project 또는 v2 token tenant ID를 읽고 고정합니다. 수동 token·system/domain/unscoped 인증·프로젝트가 없는 인증 결과에는 `resource.ErrUnsupported`를 반환합니다. token 문자열이나 Compute endpoint에서 ID를 추측하지 않습니다. 인증 결과가 바뀌어도 이미 생성한 scope의 프로젝트는 바뀌지 않습니다. Python compute proxy와 cloud quota 메서드는 project 인자가 필수이며, current-project 진입점은 Go의 편의 기능입니다. Python의 `current_project_id`가 수행할 수 있는 token 갱신까지 구현한 것은 아닙니다.

## 값과 응답 보존

`UpdateOpts`는 Gophercloud v2.15.0의 native alias입니다. `*int`가 nil이면 해당 limit을 보내지 않고, 0이면 0을 보내며, -1은 무제한입니다. -1보다 작은 typed limit은 HTTP 전송 전에 `resource.ErrInvalidOption`으로 거부합니다. `WithUpdateOptions`는 typed 옵션 전체를 대체합니다.

Native `Force bool`의 false는 `omitempty`로 생략됩니다. Scope의 `WithUpdateForce(false)`는 false를 명시적으로 보내고, `WithUpdateForce(true)`는 사용량·예약량보다 작은 새 limit의 적용을 요청합니다. 마지막 force 옵션이 우선합니다. 생략 시 [Nova API의 force 기본값](https://docs.openstack.org/api-ref/compute/#update-quotas)은 false입니다. Python cloud의 `set_compute_quotas`는 force를 True로 강제하지만 이 scope는 compute proxy/API 계약을 따르므로 같은 cloud 동작은 `WithUpdateForce(true)`로 요청합니다. 이 force 옵션은 scope 전용이며 하위 `API.Update`에서는 사용할 수 없습니다.

추가 quota 필드는 `WithUpdateField("vendor_quota", value)`로 전달합니다. 값은 옵션 생성 시 JSON으로 복사하며 native input의 core 필드와 충돌하면 거부합니다. 서버가 확장 필드의 의미를 검증합니다. Query·header 확장이나 사용자 정의 builder는 받지 않습니다.

Get·Update의 `QuotaResource`는 embedded native `QuotaSet`, 고정된 `ProjectID`, 전체 quota 객체의 `Body`, 복사된 응답 `Header`를 제공합니다. Detail의 `QuotaDetailResource`는 native `QuotaDetailSet`을 보존하므로 `Cores.Limit/InUse/Reserved`처럼 읽습니다. Python common QuotaSet처럼 usage·reservation을 별도 dictionary로 펼치지 않습니다. 알려지지 않은 quota와 nested 확장 값도 `Body`에서 읽을 수 있으며 키 존재 여부로 생략·null을 구분합니다. null·배열·scalar quota 객체, 잘못된 known field 타입은 decode 오류입니다.

Reset은 본문 없는 DELETE 성공을 처리하고 `ResetResponse`의 project ID·헤더만 반환합니다. 결과 quota를 자동으로 GET하지 않습니다. 기본 404는 `resource.ErrNotFound`이며 `WithResetIgnoreMissing(true)`는 404만 `nil, nil`로 바꿉니다. 후속 `WithResetIgnoreMissing(false)`로 다시 엄격하게 설정할 수 있습니다. 다른 HTTP 오류의 status·본문·헤더와 decode·context 취소 원인은 모두 보존합니다.

## 지원 범위

이 단위는 native Get·GetDetail·Update·Delete에 대응하는 프로젝트 quota singleton입니다. Python의 `get_quota_set_defaults`, user/user_id quota, Get·Reset 추가 query, Resource 변경 추적·자동 commit은 아직 구현하지 않았습니다. 일반 quota 목록 endpoint나 리소스 상태 대기를 가정하지 않습니다. 기존 하위 `API.Get/GetDetail/Update/Delete`는 native 모델 반환 계약을 유지합니다.

Scope는 선택한 Compute microversion을 변경하지 않습니다. [Nova version history](https://docs.openstack.org/nova/latest/reference/api-microversion-history.html)에 따라 2.36부터 network quota, 2.57부터 injected-file quota가 제거됩니다. Native alias에 필드가 있어도 모든 버전에서 전송할 수 있다는 뜻은 아니며 서버가 허용 여부를 검증합니다. Pinned Python QuotaSet의 최대 microversion 2.56을 전체 quota API의 최소 요구로 해석하지 않습니다.

비교 근거는 pinned [compute proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py), [cloud quota helpers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_compute.py), [common QuotaSet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/quota_set.py)와 [native quota API](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/quotasets/requests.go)입니다. 의미 있는 HTTP 검증은 [quota 계약 테스트](../../../api/project_quotas_contracts_test.go)에 있습니다.
