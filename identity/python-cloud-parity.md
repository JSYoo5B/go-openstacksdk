# openstacksdk identity cloud helper 대응

이 문서는 고정한 openstacksdk 커밋 `ef55d7d`의 [IdentityCloudMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py) 메서드(shade 시절부터 이어진 `conn.create_user`, `conn.get_project` 같은 cloud 계층 helper)를 이 SDK의 공개 API와 비교합니다. cloud helper는 identity Proxy 위에 이름 또는 ID 조회, 지역 필터, domain 처리, 참과 거짓 반환을 더한 것이라서 Proxy 메서드 하나와 일대일로 맞지 않습니다. 그래서 기본 인자로 호출했을 때 Python이 보내는 요청 순서와 결과를 공개 Go API만으로 그대로 만들 수 있고, 문서화된 인자의 효과도 SDK 호출로 낼 수 있을 때만 `go_mapping`으로 판정했습니다. 호출자가 직접 행을 거르거나 404를 성공으로 바꿔야 하는 경우는 `unresolved`입니다.

기존 native 호출은 `conn.Identity(ctx)`가 돌려주는 identity v3 `Service`의 필드(`Services`, `Endpoints`, `Domains`, `Groups`, `Roles`, `Projects`, `Users`)를 씁니다. 판정 근거는 [`identity/v3/python_cloud_parity_test.go`](v3/python_cloud_parity_test.go)와 [`identity/v3/python_rejudge_test.go`](v3/python_rejudge_test.go)가 고정하며, 새 Project·Group 읽기는 [Project 가이드](v3/projects/python-record-reads.md)·[Group 가이드](v3/groups/python-records.md)의 owned facade와 record tests를 사용합니다. 각 Proxy의 현행 경계는 [사용자 Proxy 문서](v3/python-parity-users.md)와 [catalog 문서](v3/python-parity-catalog.md), 기존 native 자동 조회는 [이름 또는 ID 조회 문서](../docs/finding-identities.md)를 따릅니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| [`list_projects`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L82) | `conn.ListProjects` 또는 `identity.NewProjectReadCloud(svc.Projects).ListProjects` | go_mapping |
| [`search_projects`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L123) | `conn.SearchProjects` 또는 `identity.ProjectReadCloud.SearchProjects` | go_mapping |
| [`get_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L155) | `Projects.FindIdentity` | unresolved |
| [`update_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L207) | `Projects.FindIdentity` (`WithIdentityFindIgnoreMissing(false)`), `Projects.Update` | go_mapping |
| [`create_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L235) | `Projects.Create` (`Extra`로 description null) | go_mapping |
| [`delete_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L263) | `Projects.FindIdentity`, `Projects.Remove` | go_mapping |
| [`list_users`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L291) | `Users.List` + `users.WithListOptions` | go_mapping |
| [`search_users`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L305) | `Users.List` (지역 필터 없음) | unresolved |
| [`get_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L344) | `Users.FindIdentity` | unresolved |
| [`get_user_by_id`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L393) | `Users.Get` | go_mapping |
| [`update_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L421) | `Users.FindIdentity`, `Users.Update` | go_mapping |
| [`create_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L442) | `Users.Create` (`default_project`는 `Projects.FindIdentity`) | go_mapping |
| [`delete_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L469) | `Users.FindIdentity`, `Users.Remove` | go_mapping |
| [`add_user_to_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L500) | `Users.FindIdentity`, `Groups.FindIdentity`, `Users.AddToGroup` | go_mapping |
| [`is_user_in_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L517) | `Users.FindIdentity`, `Groups.FindIdentity`, `Users.IsMemberOfGroup` | go_mapping |
| [`remove_user_from_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L533) | `Users.FindIdentity`, `Groups.FindIdentity`, `Users.RemoveFromGroup` | go_mapping |
| [`create_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L551) | `Services.Create` | go_mapping |
| [`update_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L582) | `Services.Find` 두 번, `Services.Update` | go_mapping |
| [`list_services`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L600) | `Services.All` | go_mapping |
| [`search_services`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L611) | `Services.All` (지역 필터 없음) | unresolved |
| [`get_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L640) | `Services.Find`를 ID, 이름 순서로 호출 | unresolved |
| [`delete_service`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L675) | `Services.Find` 두 번, `Services.Remove` | go_mapping |
| [`create_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L699) | `Services.Find` 두 번, URL마다 `Endpoints.Create` | go_mapping |
| [`update_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L784) | `Endpoints.Update` + `WithUpdateField("region_id", ...)` | go_mapping |
| [`list_endpoints`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L797) | `Endpoints.All` | go_mapping |
| [`search_endpoints`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L808) | `Endpoints.All` (지역 필터 없음) | unresolved |
| [`get_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L841) | `Endpoints.Find`를 ID, 이름 순서로 호출 | unresolved |
| [`delete_endpoint`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L872) | `Endpoints.Find` 두 번, `Endpoints.Remove` | go_mapping |
| [`create_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L895) | 없음 | unresolved |
| [`update_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L923) | 없음 | unresolved |
| [`list_endpoint_groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L931) | 없음 | unresolved |
| [`search_endpoint_groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L944) | 없음 | unresolved |
| [`get_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L972) | 없음 | unresolved |
| [`delete_endpoint_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L984) | 없음 | unresolved |
| [`create_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1007) | `Domains.Create` | go_mapping |
| [`update_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1031) | `Domains.Update` (이름이면 `Domains.FindIdentity`) | unresolved |
| [`delete_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1073) | `Domains.FindIdentity`, `Domains.Update`, `Domains.Delete` | go_mapping |
| [`list_domains`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1113) | `Domains.All` (filters 제한) | unresolved |
| [`search_domains`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1125) | `Domains.All` (지역 필터 없음) | unresolved |
| [`get_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1161) | `Domains.Get`, `Domains.FindIdentity` | unresolved |
| [`list_groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1209) | `Groups.List` + `groups.WithListOptions` | go_mapping |
| [`search_groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1222) | `identity.NewGroupReadCloud(svc.Groups).SearchGroups` | go_mapping |
| [`get_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1253) | `Groups.FindIdentity` | unresolved |
| [`create_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1291) | `Domains.Get`, `Groups.Create` | go_mapping |
| [`update_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1324) | `Groups.FindIdentity`, `Groups.Update` | unresolved |
| [`delete_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1355) | `Groups.FindIdentity`, `Groups.Remove` | go_mapping |
| [`list_roles`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1380) | `Roles.All` (filters 제한) | unresolved |
| [`search_roles`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1391) | `Roles.List` (지역 필터 없음) | unresolved |
| [`get_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1422) | `Roles.FindIdentity` | unresolved |
| [`list_role_assignments`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1503) | `Roles.ListAssignments` | go_mapping |
| [`create_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1577) | `Roles.Create` | go_mapping |
| [`update_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1592) | `Roles.FindIdentity`, `Roles.Update` | go_mapping |
| [`delete_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1614) | `Roles.FindIdentity`, `Roles.Remove` | go_mapping |
| [`grant_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1687) | `*.FindIdentity`, `Roles.Validate`, `Roles.Assign` | unresolved |
| [`revoke_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1812) | `*.FindIdentity`, `Roles.Validate`, `Roles.Unassign` | unresolved |

이 표의 55개 선언 중 31개가 `go_mapping`, 24개가 `unresolved`입니다. 이번 정합성 갱신은 Project·Group·Limit 조회에 한정하며, 다른 연산의 기존 기록은 유지합니다. 전체 SDK 진행량은 [중앙 지원 기록](../docs/sdk-support-ledger.md)에서 집계합니다.

## 공통 차이

기존 native helper는 `conn.Identity(ctx)`의 v3 client와 Gophercloud 모델을 사용합니다. 새 ProjectReadCloud·GroupReadCloud도 이미 선택된 Keystone v3 API를 요구하며 자동 v2 전환을 추가하지 않습니다. 새 결과의 Resource.Body는 Python 속성 이름·Source descriptor를, Wire/Envelope/header/status는 실제 HTTP 증거를 보존합니다. Source 클래스가 maximum microversion discovery를 요구하지 않는 점과, 선택 서비스 기본값·명시적 Go version 옵션을 소유하는 점을 구분합니다. Connection 전체 cache·Resource lifecycle·session은 별도 SDK-R1/C1/S1 범위입니다.

기존 native `All/List`는 Keystone links.next pager를 유지합니다. 새 Project·Group facade가 사용하는 record 목록은 dictionary next를 무시하는 Source 정책, root link 우선순위, explicit limit의 short-page marker fallback과 raw cap을 적용합니다. Project cloud helper의 filters에는 limit/marker 같은 Proxy query도 포함될 수 있으므로 한 번의 GET으로 고정하지 않습니다.

Python의 생성 helper는 `None`으로 넘긴 인자도 Resource 속성으로 넣기 때문에 본문에 JSON null이 들어갑니다. Go typed 필드는 빈 값을 생략하므로 같은 본문을 만들려면 `CreateOpts.Extra`나 `WithCreateField`로 null을 넣어야 합니다. 다만 typed 필드와 같은 이름의 `WithCreateField`는 HTTP 전에 거부되므로 `Extra`가 있는 모델은 `Extra`를 씁니다.

```go
// Python: conn.create_project("demo", domain_id="default")
p, err := svc.Projects.Create(ctx, projects.CreateOpts{
	Name: "demo", DomainID: "default", Enabled: &enabled,
	Extra: map[string]any{"description": nil},
})
```

수정 helper는 찾은 Resource에서 실제로 바뀐 속성만 PATCH에 담고, 바뀐 것이 없으면 요청을 보내지 않습니다. Go `Update`는 호출자가 채운 필드를 모두 보내므로 현재 값과 같은 필드도 본문에 들어갑니다. 삭제 helper는 참과 거짓을 돌려주고 삭제 중 `SDKException`을 기록만 하고 삼키는 경우가 많습니다. Go는 조회 결과가 nil이면 삭제를 건너뛰고, `Remove`가 404를 nil로 바꾸며, 409 같은 다른 오류는 그대로 돌려줍니다.

## service와 endpoint

`get_service`, `update_service`, `delete_service`, `create_endpoint`는 Proxy `find_service`를 거칩니다. Go에서는 `Find(ctx, resource.ID(x), resource.WithIgnoreMissing())`가 nil을 돌려줄 때만 `Find(ctx, resource.Name(x), resource.WithIgnoreMissing())`로 이어 가면 같은 두 요청(`GET /v3/services/<x>`, `GET /v3/services?name=<x>`)이 나갑니다. Python은 GET 400과 403에서도 목록으로 넘어가고 ID 또는 이름이 같은 행을 찾지만, Go 첫 호출은 404에서만 넘어가고 이름 조회는 이름만 비교합니다. 중복은 Python `DuplicateResource`, Go `resource.ErrAmbiguous`입니다. Endpoint는 이름 query를 선언하지 않았으므로 두 번째 요청은 query 없는 `GET /v3/endpoints`입니다.

```go
// Python: conn.delete_service("nova")
found, err := svc.Services.Find(ctx, resource.ID("nova"), resource.WithIgnoreMissing())
if err == nil && found == nil {
	found, err = svc.Services.Find(ctx, resource.Name("nova"), resource.WithIgnoreMissing())
}
if err == nil && found != nil {
	err = svc.Services.Remove(ctx, resource.ID(found.ID))
}
```

`create_service`는 `type`과 `service_type` 중 하나를 받아 `type`으로 보내고, 둘 다 없으면 `type: null`을 보냅니다. Go `CreateOpts.Type`은 항상 문자열로 나가므로 null이 필요하면 `Extra: map[string]any{"type": nil}`을 씁니다.

`create_endpoint`는 `url`과 `interface`를 주면 endpoint 하나를, `public_url`, `internal_url`, `admin_url`을 주면 public, internal, admin 순서로 하나씩 만듭니다. 본문에는 `region_id`가 항상 들어가며 region을 주지 않으면 null입니다. Go `CreateOpts.Region`은 옛 `region` key로 나가므로 `endpoints.WithCreateField("region_id", region)`이나 `WithCreateField("region_id", nil)`을 붙입니다. Python은 `url`과 `*_url`을 함께 주면 HTTP 전에 거부하고 `interface` 없이 `url`만 주면 `interface: null`을 보내지만, Go `CreateOpts`는 interface, url, service_id가 비면 HTTP 전에 거부합니다. `update_endpoint`는 조회 없이 바로 PATCH하며 `service_name_or_id`를 그대로 `service_id`로, `region`을 `region_id`로 보냅니다. Go에서는 `UpdateOpts.ServiceID`와 `endpoints.WithUpdateField("region_id", ...)`가 같은 본문을 만듭니다.

`get_service`와 `get_endpoint`는 기본 인자에서 위 두 단계 조회와 같습니다. 그러나 deprecated `filters` 인자를 주면 `search_*`의 지역 필터를 거치는데 이것을 대신할 공개 Go helper가 없어서 `unresolved`로 남겼습니다.

## domain, group, role

`create_domain`, `create_role`은 넘긴 값만 본문에 담으므로 typed `CreateOpts`가 그대로 같습니다. `create_group`은 domain을 주면 `get_domain(domain)`을 domain_id로 불러 `GET /v3/domains/<domain>`을 보내고, 404이면 문서와 달리 `NotFoundException`을 그대로 올립니다. Go도 `Domains.Get`의 404 오류에서 멈추면 같은 순서가 됩니다. `list_groups`는 `domain_id`만 받으므로 `groups.WithListOptions(groups.ListOpts{DomainID: ...})`와 같습니다.

`list_role_assignments`는 `user`, `group`, `role`을 `user.id`, `group.id`, `role.id`로, `project`와 `domain`을 `scope.project.id`, `scope.domain.id`로, `system`을 `scope.system`으로, `inherited_to`를 `scope.OS-INHERIT:inherited_to`로 바꾸어 `GET /v3/role_assignments`에 붙입니다. Go `roles.ListAssignmentsOpts`가 ID와 system을 담고, typed 필드가 없는 `inherited_to`와 Python이 `True`로 적는 `effective`는 `roles.WithListAssignmentsQuery`로 같은 철자를 보낼 수 있습니다. typed `Effective`를 쓰면 소문자 `true`가 나갑니다.

`delete_domain`은 domain_id를 주면 조회 없이, name_or_id를 주면 `find_domain(name_or_id, ignore_missing=True)`로 찾은 뒤 먼저 `enabled: false`로 PATCH하고 404를 무시하지 않는 DELETE를 보냅니다. Go `Domains.FindIdentity`는 같은 `GET /v3/domains/<x>`와 `GET /v3/domains?name=<x>`를 보내고 찾지 못하면 nil을 돌려줍니다. 그래서 결과가 nil이 아닐 때 `Domains.Update(ctx, id, domains.UpdateOpts{Enabled: &disabled})`와 `Domains.Delete`를 이어 부르면 같은 순서가 됩니다. Python은 찾지 못하면 False를 돌려주고, DELETE의 404는 Python과 Go 모두 오류입니다. `update_domain`도 domain_id 경로에서는 `Domains.Update`와 같은 PATCH를 보내지만 판정은 아래 미해결 목록을 따릅니다.

## project와 user

`create_project`는 description을 주지 않아도 `description: null`을 보내고, 나머지 keyword 인자도 Project가 모르는 속성까지 본문에 넣습니다. Go는 typed 필드와 `Extra`로 같은 본문을 만듭니다. `list_users`는 `domain_id`와 `name`만 받으므로 `users.WithListOptions(users.ListOpts{DomainID: ..., Name: ...})`와 같고, `get_user_by_id`는 `Users.Get`과 같은 `GET /v3/users/<id>`입니다. deprecated `normalize` 인자는 경고만 냅니다.

`create_user`는 domain_id가 없으면 HTTP 전에 `SDKException`을 내지만 Go는 domain_id 없이 요청을 보냅니다. domain_id를 주면 `email: null`이 들어간 본문이 나가며 Go에서는 `Extra: map[string]any{"email": nil}`로 같게 만들 수 있습니다. `default_project`를 주면 POST 전에 `get_project(default_project)`가 아래와 같은 두 조회 요청을 보내고, 찾으면 `default_project_id`를 채우며 없으면 그 key 없이 생성합니다. Go에서는 `Projects.FindIdentity` 결과가 nil이 아닐 때 `CreateOpts.DefaultProjectID`를 채우면 같은 본문이 됩니다.

## Project 목록·검색과 Group 검색

`list_projects/search_projects/search_groups`는 새 owned read API로 `go_mapping`입니다. `ProjectReadCloud.ListProjects`는 falsey filters를 `{}`로 바꾸고 object를 kwargs로 펼친 뒤 truthy name/domain 인자로 덮어씁니다. `SearchProjects`는 name을 server query에 넣지 않고 그 목록 뒤 exact/glob ID/name와 ordered nested loaded-JSON filter를 적용합니다. **truthy JMESPath 문자열은 list_projects의 dict(**filters) 단계에서 HTTP 전에 실패**하므로 Source 문서에 적힌 문자열 지원을 실제 성공 경로로 주장하지 않습니다. wire `enabled`는 server query로 valid하지만 local cloud Resource membership은 `is_enabled`를 사용합니다.

Group은 다릅니다. `identity.NewGroupReadCloud(svc.Groups).SearchGroups`는 domain query만 서버에 보내 모든 페이지를 수집한 뒤 이름과 nested mapping/JMESPath를 평가합니다. 따라서 Group의 truthy string expression은 actual row 배열에서 JSON scalar/object/list를 반환할 수 있습니다. expression 결과의 Groups는 nil이고 Inventory에는 실제 원래 rows와 늦은 HTTP/local 오류 증거가 남습니다. Project의 이와 다른 순서를 한 공통 filter 전략으로 축약하지 않습니다.

Connection의 Project helper는 cached IdentityV3와 CurrentLocation을 local 선택 전에 공급하고 immutable per-call location override도 보존합니다. 직접 facade·Group은 native client만으로 cloud/project 이름을 추정하지 않고 명시적 location 옵션을 사용합니다. Proxy literal·opaque 결과의 Project list는 Value와 선택적 JSON을 구분하고 실제 Project/HTTP association을 만들어내지 않습니다. Go의 object key 정렬·iterator 재순회·안전한 continuation 검증과 concrete 오류 타입은 각 가이드의 문서화된 차이입니다.

Project 목록과 Group get/list는 stock policy의 system/domain reader 경로이며 일반 project reader 전체 목록 권한을 뜻하지 않습니다. Project GET은 own-project user도 허용하지만 Find fallback과 cloud 검색은 해당 목록 권한을 필요로 합니다. 정책 근거는 [Keystone28.0.0 Project](https://github.com/openstack/keystone/blob/28.0.0/keystone/common/policies/project.py#L157-L174), [Group](https://opendev.org/openstack/keystone/src/tag/28.0.0/keystone/common/policies/group.py)과 [domain 목록 controller](https://opendev.org/openstack/keystone/src/tag/28.0.0/keystone/api/groups.py)입니다.

Proxy `find_project/find_group`는 concrete FindRecord 옵션·GET 우선·clean fallback·중복/missing이 있어도 **named unresolved**를 유지합니다. raw Resource.fetch control/header kwargs와 generic synchronized Resource/cache remaining을 [사용자 Proxy 지원 경계](v3/python-parity-users.md#projectgroup-record-조회와-지원-경계)에 기록합니다. 이 목록·검색 판정으로 cloud `get_project/get_group`, 수정·삭제·role helper의 후속 workflow까지 승격하지 않습니다.

## 이름 또는 ID로 찾는 helper

`update_project`, `delete_project`, `update_user`, `delete_user`, `delete_group`, `update_role`, `delete_role`과 그룹 소속 helper는 먼저 Proxy `find_*`로 대상을 찾습니다. 이때 넘기는 값은 name_or_id, ignore_missing, domain_id뿐이라서 Proxy `find_*`의 임의 query 분류 문제는 생기지 않습니다. Go `FindIdentity`는 ID 경로 GET을 먼저 보내고 400, 403, 404이면 이름 목록으로 넘어갑니다. `resource.WithIdentityFindQuery("domain_id", id)`를 주면 두 요청에 같은 query를 붙입니다. 찾지 못하면 기본으로 nil을 돌려주는데 이것이 Python의 `ignore_missing=True`입니다. `update_project`처럼 `ignore_missing=False`로 찾는 경우는 `resource.WithIdentityFindIgnoreMissing(false)`를 주면 `resource.ErrNotFound`를 받습니다.

```go
// Python: conn.delete_user("alice", domain_id="default")
user, err := svc.Users.FindIdentity(ctx, "alice",
	resource.WithIdentityFindQuery("domain_id", "default"))
if err == nil && user != nil {
	err = svc.Users.Remove(ctx, resource.ID(user.ID))
}
```

삭제 helper는 Proxy `delete_*`의 기본값 `ignore_missing=True`로 지우므로 404를 nil로 바꾸는 `Remove`와 같습니다. 수정 helper는 찾은 ID로 `Update`를 부르고 호출자가 넘긴 속성만 typed 필드나 `Extra`에 담으면 됩니다. `update_user`의 `email`은 `Extra`로 보내고, User 속성이 아닌 `default_project`는 Python도 본문에서 버립니다. 그룹 소속 helper는 `get_user`, `get_group` 순서로 찾고 하나라도 없으면 `SDKException`을 냅니다. 둘 다 찾으면 `Users.AddToGroup`, `Users.IsMemberOfGroup`, `Users.RemoveFromGroup`과 같은 PUT, HEAD, DELETE를 보내며, `IsMemberOfGroup`은 Python `check_user`처럼 HEAD 404를 거짓으로 돌려줍니다.

기존 native FindIdentity는 Python Connection cache·mutable Resource 변환을 복제하지 않고 native 모델·Keystone links.next pager를 유지합니다. 새 record FindRecord의 tested concrete 부분과 named unresolved control/cache 범위는 위 단락에서 구분합니다. ID 경로에 그대로 넣기 안전하지 않은 이름(예를 들어 `/`가 들어간 이름)은 Go가 GET 없이 목록만 조회합니다. 이름이 겹치면 Python은 `DuplicateResource`, Go는 `resource.ErrAmbiguous`입니다.

## 미해결 메서드

- `get_project/get_group`: 기본 native FindIdentity와 새 record/검색 경로는 있지만 cloud get의 filters·single-match·missing/ambiguity 전체 workflow 판정은 이번 범위에서 갱신하지 않습니다. 두 named Proxy find의 raw control/cache remaining도 unresolved입니다.
- `get_user/get_role`: deprecated filters를 사용하면 search/local 선택·single-match workflow가 필요하고, 이 문서의 기존 native 대응은 unresolved입니다.
- `update_group`: 새 concrete FindRecord가 있어도 named find_group의 raw fetch control/header와 generic cache/lifecycle remaining이 남습니다. 기존 cloud update 판정도 유지합니다. `get_domain/update_domain`은 기존 find_domain와 Resource 대응 범위에서 unresolved입니다.
- `grant_role`, `revoke_role`: domain, role, user 또는 group, project를 모두 `find_*`로 찾습니다. 또 Python `validate_*_has_*_role`은 204가 아닌 HEAD를 거짓으로 보지만 Go `roles.API.Validate`와 `osinherit.API.Validate`는 404를 오류로 돌려주므로, 할당이 없다고 판단하는 일이 호출자 몫으로 남습니다. Python은 `wait`, `timeout` 인자를 받기만 하고 쓰지 않습니다.

다음 메서드는 결과를 `_utils._filter_list`로 지역에서 거릅니다. 이 함수는 ID나 이름의 정확한 일치와 fnmatch glob 일치, 중첩 dict 필터, deprecated jmespath 문자열을 처리합니다. 이번 Project·Group에는 위 owned helper가 있지만 아래 다른 native identity helper의 지역 filter 문제는 기존 미해결 범위로 남습니다.

- `search_users`, `search_roles`, `search_domains`, `search_services`, `search_endpoints`
- `get_service`, `get_endpoint`: deprecated `filters`를 줄 때만 해당합니다.

`list_domains/list_roles`는 임의의 filter keyword를 `Resource.list`에 넘깁니다. `list_projects`는 위 concrete facade의 현행 구현을 사용합니다. Python은 query가 아닌 Body 속성(예를 들어 description)을 각 행에서 지역으로 거르고, `limit`을 주면 마지막 행 ID를 marker로 삼아 빈 페이지가 나올 때까지 다시 요청합니다. 이 두 남은 native 바인딩에는 Body 필터와 limit 기반 marker 요청이 없습니다.

endpoint group helper(`create_endpoint_group`, `update_endpoint_group`, `list_endpoint_groups`, `search_endpoint_groups`, `get_endpoint_group`, `delete_endpoint_group`)는 Gophercloud v2.15.0에 Keystone `/OS-EP-FILTER/endpoint_groups` 바인딩이 없어서 SDK 고유 호출이 필요합니다.
