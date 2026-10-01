# Designate 프로젝트 quota

`dns/v2/quotas`는 Designate의 프로젝트 quota singleton에 `InProject`와 `CurrentProject` scope를 제공합니다. 기존 generated `API.Get`, `API.Update`, `API.URL`과 native `Quota`, `UpdateOpts` alias는 그대로 유지합니다. Native에 없는 reset은 `ProjectQuotaScope.Reset`으로 제공합니다.

## 프로젝트 고정

```go
api := quotas.New(dnsClient)
scope, err := api.InProject(ctx, resource.ID("project-id"),
    quotas.WithAllProjects(true))
if err != nil { return err }

limits, err := scope.Get(ctx)
if err != nil { return err }
fmt.Println(limits.ProjectID, limits.Zones, limits.StatusCode)

zero := 0
updated, err := scope.Update(ctx, quotas.UpdateOpts{Zones: &zero})
if err != nil { return err }
fmt.Println(updated.Header.Get("X-Openstack-Request-Id"))

reset, err := scope.Reset(ctx)
if err != nil { return err }
fmt.Println(reset.ProjectID, reset.StatusCode)
```

`resource.ID`는 사전 HTTP 조회 없이 project ID를 고정합니다. `resource.Name`은 별도의 Keystone v3 client를 사용하는 `WithIdentityClient(identityClient)`가 필요합니다. 전체 project collection에서 정확한 이름을 한 번 해석하며, 같은 이름이 여러 domain에 있으면 `resource.ErrAmbiguous`를 반환합니다. UUID 모양의 이름도 이름으로 처리합니다. DNS client를 Keystone 요청에 재사용하거나 quota 응답의 `id`·`project_id`로 다음 요청 대상을 바꾸지 않습니다.

`CurrentProject(ctx, options...)`는 ProviderClient에 기록된 Keystone v3 project 또는 v2 token tenant를 읽고 고정합니다. 인증 갱신이나 HTTP 조회를 하지 않습니다. 수동 token·system/domain/unscoped 인증·기록된 인증 결과가 없는 경우에는 `resource.ErrUnsupported`이며, token 문자열과 endpoint 경로에서 프로젝트를 추측하지 않습니다. 인증 결과가 바뀌어도 기존 scope의 project ID는 유지됩니다.

## HTTP와 헤더

[Designate quota API](https://docs.openstack.org/api-ref/dns/dns-api-v2-index.html#quotas)의 경로와 성공 코드를 사용합니다. Endpoint 또는 명시적인 `ResourceBase` 뒤에 경로를 붙입니다.

| scope 메서드 | HTTP 경로 | 성공 | 결과 |
|---|---|---|---|
| Get | `GET /quotas/{project}` | 200 | root quota 객체 |
| Update | `PATCH /quotas/{project}` | 200 | root quota 객체 |
| Reset | `DELETE /quotas/{project}` | 204 | 본문 없는 acknowledgement |

Pinned Python `Quota._prepare_request`와 같이 scope 요청은 `X-Auth-Sudo-Project-ID`에 고정한 project ID를 넣습니다. 원래 client의 동일 헤더가 다른 project를 가리켜도 이 요청에서는 고정한 ID를 사용합니다. 요청마다 ServiceClient와 `MoreHeaders`를 독립적으로 복사하므로 원본 client/provider/token lock을 변경하지 않습니다.

`WithAllProjects(true)` 또는 `WithAllProjects(false)`는 scope 생성 시 `X-Auth-All-Projects`를 명시합니다. 옵션을 생략하면 원래 client에 설정된 값을 보존합니다. 명시한 false는 대소문자가 다른 동일 헤더도 교체하며, 다른 설정 헤더에는 영향을 주지 않습니다. 마지막 옵션이 우선합니다. Cross-project 요청의 권한은 Designate가 검증합니다.

공유 fixed-request helper는 redirect의 method·origin·경로·query 변경을 거부하고 같은 대상의 redirect는 원래 redirect policy를 거쳐 허용합니다. 재인증·backoff·retry는 원래 provider의 기능을 사용하며, context와 HTTPClient 설정을 보존합니다. Reset 실패 시 다른 프로젝트나 current-project 경로로 대체하지 않습니다.

## 입력과 결과 보존

`UpdateOpts`는 pinned Gophercloud의 구체적인 native alias입니다. `*int`가 nil이면 필드를 생략하고 0이면 0을 보냅니다. Scope Update는 typed limit을 직접 JSON으로 인코딩하므로 플랫폼 int 범위에서 2^53보다 큰 정수도 정확하게 전송합니다. 재시도 전에 request body를 확정하며, caller가 원래 pointer 값을 바꿔도 재시도 본문은 바뀌지 않습니다. 숫자 범위와 값의 의미는 Designate의 schema가 검증합니다. SDK는 음수나 null을 임의로 defaults 상속으로 해석하지 않습니다.

```go
zones := 7
snapshot := quotas.WithQuotaOptions(quotas.UpdateOpts{Zones: &zones})
zones = 99
_, err := scope.Update(ctx, quotas.UpdateOpts{}, snapshot) // zones=7
```

`WithQuotaOptions`는 생성 시 모든 typed pointer를 복사하며 재사용하는 호출마다 독립된 입력을 제공합니다. 기존 `WithUpdateOptions`는 전체 typed 입력을 교체하고 pointer 값은 Update 호출 시 복사합니다. 확장 quota 필드는 `WithUpdateField("vendor_quota", value)`로 넣습니다. 확장 값은 옵션 생성 시 JSON으로 복사하며, nil 확장 값은 명시적인 null로 전송합니다. Core limit과 `id`·`project`·`project_id`는 확장 필드로 덮어쓸 수 없습니다. 임의 query·header·argument와 custom builder는 scope Update에서 받지 않습니다.

`QuotaResource`에는 embedded native `Quota`, 고정한 `ProjectID`, 전체 root 객체의 `Body`, 독립적으로 복사한 `Header`, 실제 `StatusCode`가 있습니다. 알려지지 않은 필드·nested 객체·large integer도 `Body`에 남습니다. Native int의 null은 0으로 읽지만 `Body`의 키 존재 여부와 raw 값으로 omitted·null·0을 구분할 수 있습니다. 응답 identity 필드는 raw body로 보존하며 요청 대상을 바꾸지 않습니다.

배열·scalar·null root, malformed JSON, 잘못된 known field 타입은 decode 오류입니다. 성공 응답을 decode하지 못하면 `QuotaResponseError`에 원본 본문·헤더·status를 보존하며 `errors.As`로 원래 JSON/read/context 오류를 읽을 수 있습니다. HTTP 실패는 native 오류의 status·본문·헤더를 유지합니다. 404는 `resource.ErrNotFound`로도 판별할 수 있습니다.

Reset은 204의 헤더와 status를 `ResetResponse`로 반환하며 뒤따르는 GET을 하지 않습니다. 기본 404는 오류입니다. `WithResetIgnoreMissing(true)`는 404만 `nil, nil`로 바꾸고, 뒤의 false 옵션으로 엄격하게 되돌릴 수 있습니다.

## Python 비교와 지원 범위

비교 기준은 pinned [DNS Quota resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/dns/v2/quota.py), [DNS proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/dns/v2/_proxy.py), [Gophercloud native API](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/dns/v2/quotas/requests.go)입니다.

| Python | Go scope | 차이 |
|---|---|---|
| get_quota(project) | Get | native typed 값과 raw/status/header를 함께 반환 |
| update_quota(project, **attrs) | Update | concrete native opts와 충돌을 거부하는 확장 필드; PATCH/root 객체 |
| delete_quota(project, ignore_missing=True) | Reset | Go 기본값은 strict이며 ignoreMissing은 명시적인 옵션 |
| Quota._prepare_request sudo-project header | 고정 project header | 원본 client 헤더를 변경하지 않음 |
| inherited Resource 변경 추적·commit·cache/session 옵션 | 명시적인 Update 호출 | inherited Python 상태·cache 동작은 partial |

Python proxy의 `quotas()`와 `allow_list=True`는 있지만, 공식 `GET /quotas/`는 current project의 quota 객체를 반환합니다. 이 route를 여러 project의 quota 목록으로 가정하지 않습니다. Scope는 기록된 인증에서 고정한 ID의 명시적인 경로를 사용하며 List·Find·Wait·사용자 scope를 제공하지 않습니다.

[Designate 관리 문서](https://docs.openstack.org/designate/latest/admin/quotas.html#default-quotas)에서 deployment defaults는 `designate.conf`로 설정합니다. 이 단위는 별도의 Defaults endpoint를 만들어내지 않으며, reset으로 project override를 제거합니다.

HTTP 계약은 [scope·입력·응답 테스트](../../../api/designate_project_quotas_contracts_test.go), [project binding 테스트](../../../api/designate_project_quotas_projects_test.go), [redirect·retry·reauth·context 테스트](../../../api/designate_project_quotas_transport_test.go)로 검증합니다.
