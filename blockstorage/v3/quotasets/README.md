# 프로젝트별 Cinder v3 quota

`InProject(ctx, projectRef)`가 quota를 조회·수정할 프로젝트를 한 번 고정합니다. 결과 `ProjectQuotaScope`는 Get·Defaults·Usage·Update·Reset을 제공합니다. Cinder의 quota는 프로젝트 singleton이며 일반 리소스 collection의 List·Find·Wait를 가정하지 않습니다.

| openstacksdk | Go scope |
|---|---|
| `conn.block_storage.get_quota_set(project)` | `scope.Get(ctx)` |
| `conn.block_storage.get_quota_set_defaults(project)` | `scope.Defaults(ctx)` |
| `conn.block_storage.get_quota_set(project, usage=True)` | `scope.Usage(ctx)` |
| `conn.block_storage.update_quota_set(project, volumes=0)` | `scope.Update(ctx, quotasets.UpdateOpts{Volumes: &zero})` |
| `conn.block_storage.revert_quota_set(project)` | `scope.Reset(ctx)` |
| `conn.get_volume_quotas("tenant")`의 Keystone 이름 조회 | `conn.BlockStorageProjectQuotas(ctx, resource.Name("tenant"))` |
| `conn.current_project_id`를 quota 조회에 전달 | `conn.CurrentBlockStorageProjectQuotas(ctx)` |

```go
// context.Context ctx와 Cinder v3 *gophercloud.ServiceClient volumeClient를
// 사용하는 함수 안에서. resource/quotasets는 gophercloudsdk 패키지입니다.
api := quotasets.New(volumeClient)
scope, err := api.InProject(ctx, resource.ID("project-id"))
if err != nil { return err }

limits, err := scope.Get(ctx)
if err != nil { return err }
fmt.Println(scope.ProjectID(), limits.Volumes, limits.Header.Get("X-Openstack-Request-Id"))

defaults, err := scope.Defaults(ctx)
if err != nil { return err }
fmt.Println(defaults.Volumes)

usage, err := scope.Usage(ctx)
if err != nil { return err }
fmt.Println(usage.Volumes.Limit, usage.Volumes.InUse, usage.Volumes.Reserved)

zero, unlimited := 0, -1
updated, err := scope.Update(ctx, quotasets.UpdateOpts{
    Volumes: &unlimited,
    Snapshots: &zero,
}, quotasets.WithVolumeTypeQuota(quotasets.VolumeTypeVolumes, "SSD", 12))
if err != nil { return err }
fmt.Println(updated.Volumes, string(updated.Body["volumes_SSD"]))

reset, err := scope.Reset(ctx)
if err != nil { return err }
fmt.Println(reset.Header.Get("X-Openstack-Request-Id"))
```

`resource.ID`는 Keystone 조회 없이 명시한 project ID를 사용합니다. `resource.Name`은 별도 Keystone v3 client의 project collection에서 모든 페이지의 정확한 이름을 확인합니다. 중복·미존재·권한 오류를 숨기지 않으며, Cinder client로 Keystone 요청을 보내거나 이후 quota 작업에서 이름을 다시 찾지 않습니다. 응답의 `id`가 요청한 project ID와 달라도 scope의 대상은 바뀌지 않습니다.

`CurrentProject(ctx)`는 ProviderClient가 기록한 Keystone v3 project 또는 v2 token tenant ID를 읽고 고정합니다. 수동 token, system/domain/unscoped 인증, 프로젝트가 없는 인증 결과에는 `resource.ErrUnsupported`를 반환합니다. endpoint나 token 문자열에서 ID를 추측하지 않습니다. 인증 결과가 나중에 바뀌어도 이미 만든 scope의 프로젝트는 유지합니다. 이는 Go 편의 진입점이며 Python `current_project_id`의 token 갱신 정책까지 구현한 것은 아닙니다.

## 입력과 옵션 복사

`UpdateOpts`는 pinned Gophercloud의 concrete alias입니다. `*int`가 nil이면 limit을 생략하고, 0이면 0을 전송하며, -1은 무제한입니다. -1보다 작은 typed limit은 HTTP 전에 `resource.ErrInvalidOption`으로 거부합니다. Scope는 typed limit을 raw JSON으로 구성하므로 플랫폼 int 범위에서 2^53보다 큰 값도 float64 반올림 없이 보냅니다. 직접 전달한 opts와 기존 `WithUpdateOptions`의 결과는 Update 호출 중에 복사됩니다. 옵션을 만드는 시점의 limit·중첩 map·slice까지 고정하려면 scope의 `WithQuotaOptions`를 사용합니다. 두 옵션 모두 typed 입력 전체를 대체합니다.

```go
limit := 5
nested := map[string]any{"tier": "original"}
snapshot := quotasets.WithQuotaOptions(quotasets.UpdateOpts{
    Volumes: &limit,
    Extra: map[string]any{"vendor_quota": nested},
})
limit = 99
nested["tier"] = "changed"
_, err := scope.Update(ctx, quotasets.UpdateOpts{}, snapshot) // volumes=5, original
if err != nil { return err }
```

Native `Extra`와 `WithUpdateField`는 JSON 확장을 허용하되 `volumes`, `snapshots`, `gigabytes`, `per_volume_gigabytes`, `backups`, `backup_gigabytes`, `groups`, `force` 등 core 필드를 덮어쓰면 생략된 core 필드라도 거부합니다. Extra와 `WithUpdateField`의 동일 키 충돌도 거부합니다. `WithUpdateField`의 값은 옵션 생성 시 JSON으로 복사됩니다. 확장 필드의 의미는 서버가 검증합니다. 임의 query·request header·사용자 정의 builder는 scope Update에서 받지 않습니다.

`WithVolumeTypeQuota`는 volumes·snapshots·gigabytes에 대해 `quotaName_volumeTypeName` 형식의 키를 구성합니다. 이름은 정확한 볼륨 타입 **이름**이며 ID로 변환하거나 볼륨 타입 조회를 하지 않습니다. 빈 이름, 지원하지 않는 quota 종류, -1보다 작은 limit은 HTTP 전에 거부합니다. 예를 들어 `WithVolumeTypeQuota(VolumeTypeGigabytes, "SSD", 0)`은 `gigabytes_SSD: 0`을 보냅니다.

Gophercloud가 선언한 `Force bool`은 false를 생략합니다. Scope 전용 `WithUpdateForce(false)`는 false를 명시적으로 전송하며 마지막 옵션이 우선합니다. 이는 native 필드의 전송 기능을 보존하는 옵션입니다. 현재 [Cinder quota API](https://docs.openstack.org/api-ref/block-storage/v3/#quota-sets-extension-os-quota-sets)는 force의 사용량 우회를 보장하지 않으므로 해당 동작을 가정하지 않습니다. 서버가 거부하면 원래 HTTP 오류를 반환합니다. 기본 Update는 force를 보내지 않습니다.

## 조회와 응답

Get은 `/os-quota-sets/{project}`를, Defaults는 `/os-quota-sets/{project}/defaults`를 호출합니다. Usage는 같은 기본 endpoint에 `?usage=true`를 붙입니다. Nova의 `/detail` 경로로 보내지 않습니다. Defaults 조회는 현재 override를 변경하지 않습니다.

Get·Defaults·Update의 `QuotaResource`는 native `QuotaSet`과 고정된 `ProjectID`, 전체 quota 객체의 `Body map[string]json.RawMessage`, 독립적으로 복사한 응답 `Header`를 제공합니다. Usage의 `QuotaUsageResource`는 native `QuotaUsageSet`을 보존하므로 `Volumes.Limit/InUse/Reserved/Allocated`처럼 읽습니다. allocated가 생략된 경우와 0인 경우는 Body의 키로 구분합니다. Python common QuotaSet처럼 usage·reservation을 별도 dictionary로 펼치지 않습니다.

Native limits 모델은 볼륨 타입별 필드를 `Extra map[string]any`로 읽으며, native Usage 모델에는 Extra가 없습니다. 두 경우 모두 Body가 알려지지 않은 per-type quota·배열·null·중첩 값·큰 정수의 정확한 JSON 값을 보존합니다. Native Extra의 숫자 변환으로 정밀도가 줄어드는 값도 Body에서 읽을 수 있습니다. quota_set envelope가 없거나 null·배열·scalar인 응답과 잘못된 known field 타입은 decode 오류입니다.

Reset은 quota override를 지워 기본값으로 되돌리는 DELETE입니다. Cinder native 계약대로 **200만 성공**으로 처리하며 Nova의 202/204 성공 정책을 적용하지 않습니다. ResetResponse는 project ID와 헤더를 반환하고 자동 후속 GET을 하지 않습니다. 기본 404는 `resource.ErrNotFound`이며 `WithResetIgnoreMissing(true)`는 404만 `nil, nil`로 바꿉니다. 후속 false 옵션으로 다시 엄격하게 설정할 수 있습니다. HTTP status·본문·헤더, JSON decode와 context 취소·timeout 원인은 보존합니다.

## 지원 범위와 근거

이 단위는 Cinder v3 프로젝트 quota의 native Get·GetDefaults·GetUsage·Update·Delete에 대응합니다. Python Proxy가 받는 Get·Reset의 추가 `**query`는 제공하지 않습니다. 현재 Cinder API가 정의하는 Get usage는 Usage에 매핑했고, user quota·user_id query·Reset query endpoint 동작을 추정하지 않았습니다. Python Resource의 변경 추적·cache·자동 commit 및 legacy QuotaSet 인자 fallback도 이 scope의 계약에 포함하지 않습니다. Quota class는 별도 API입니다. 이 프로젝트 singleton은 quota List·Find·Wait를 제공하지 않습니다. 기존 하위 `API.Get/GetDefaults/GetUsage/Update/Delete`는 native 반환 타입을 유지합니다. Scope는 client의 microversion을 변경하지 않습니다.

비교 근거는 pinned [block storage proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py), [quota model](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/quota_set.py), [common QuotaSet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/common/quota_set.py), [cloud quota helpers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_block_storage.py), [Gophercloud v2.15.0 requests](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/blockstorage/v3/quotasets/requests.go)입니다. [HTTP 계약 테스트](../../../api/cinder_project_quotas_contracts_test.go)와 [응답·snapshot 테스트](../../../api/cinder_project_quotas_responses_test.go)가 이 단위를 검증합니다.
