# openstacksdk identity cloud helper 대응

이 문서는 고정한 openstacksdk 커밋 `ef55d7d`의 [IdentityCloudMixin](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py) 메서드(shade 시절부터 이어진 `conn.create_user`, `conn.get_project` 같은 cloud 계층 helper)를 이 SDK의 공개 API와 비교합니다. cloud helper는 identity Proxy 위에 이름 또는 ID 조회, 지역 필터, domain 처리, 참과 거짓 반환을 더한 것이라서 Proxy 메서드 하나와 일대일로 맞지 않습니다. 그래서 기본 인자로 호출했을 때 Python이 보내는 요청 순서와 결과를 공개 Go API만으로 그대로 만들 수 있고, 문서화된 인자의 효과도 SDK 호출로 낼 수 있을 때만 `go_mapping`으로 판정했습니다. 호출자가 직접 행을 거르거나 404를 성공으로 바꿔야 하는 경우는 `unresolved`입니다.

Go 쪽 호출은 모두 `conn.Identity(ctx)`가 돌려주는 identity v3 `Service`의 필드(`Services`, `Endpoints`, `Domains`, `Groups`, `Roles`, `Projects`, `Users`)를 씁니다. 판정 근거는 [`identity/v3/python_cloud_parity_test.go`](v3/python_cloud_parity_test.go)가 고정하며, 각 Proxy 메서드 자체의 판정은 [identity v3 catalog 문서](v3/python-parity-catalog.md)와 [이름 또는 ID 조회 문서](../docs/finding-identities.md)를 따릅니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| [`list_projects`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L82) | `Projects.All` (filters 제한) | unresolved |
| [`search_projects`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L123) | `Projects.All` (지역 필터 없음) | unresolved |
| [`get_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L155) | `Projects.FindIdentity` | unresolved |
| [`update_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L207) | `Projects.FindIdentity`, `Projects.Update` | unresolved |
| [`create_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L235) | `Projects.Create` (`Extra`로 description null) | go_mapping |
| [`delete_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L263) | `Projects.FindIdentity`, `Projects.Remove` | unresolved |
| [`list_users`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L291) | `Users.List` + `users.WithListOptions` | go_mapping |
| [`search_users`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L305) | `Users.List` (지역 필터 없음) | unresolved |
| [`get_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L344) | `Users.FindIdentity` | unresolved |
| [`get_user_by_id`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L393) | `Users.Get` | go_mapping |
| [`update_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L421) | `Users.FindIdentity`, `Users.Update` | unresolved |
| [`create_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L442) | `Users.Create` (`default_project`는 `Projects.FindIdentity`) | unresolved |
| [`delete_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L469) | `Users.FindIdentity`, `Users.Remove` | unresolved |
| [`add_user_to_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L500) | `Users.AddToGroup` | unresolved |
| [`is_user_in_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L517) | `Users.IsMemberOfGroup` | unresolved |
| [`remove_user_from_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L533) | `Users.RemoveFromGroup` | unresolved |
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
| [`delete_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1073) | `Domains.Update`, `Domains.Delete` | unresolved |
| [`list_domains`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1113) | `Domains.All` (filters 제한) | unresolved |
| [`search_domains`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1125) | `Domains.All` (지역 필터 없음) | unresolved |
| [`get_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1161) | `Domains.Get`, `Domains.FindIdentity` | unresolved |
| [`list_groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1209) | `Groups.List` + `groups.WithListOptions` | go_mapping |
| [`search_groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1222) | `Groups.List` (지역 필터 없음) | unresolved |
| [`get_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1253) | `Groups.FindIdentity` | unresolved |
| [`create_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1291) | `Domains.Get`, `Groups.Create` | go_mapping |
| [`update_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1324) | `Groups.FindIdentity`, `Groups.Update` | unresolved |
| [`delete_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1355) | `Groups.FindIdentity`, `Groups.Remove` | unresolved |
| [`list_roles`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1380) | `Roles.All` (filters 제한) | unresolved |
| [`search_roles`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1391) | `Roles.List` (지역 필터 없음) | unresolved |
| [`get_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1422) | `Roles.FindIdentity` | unresolved |
| [`list_role_assignments`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1503) | `Roles.ListAssignments` | go_mapping |
| [`create_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1577) | `Roles.Create` | go_mapping |
| [`update_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1592) | `Roles.FindIdentity`, `Roles.Update` | unresolved |
| [`delete_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1614) | `Roles.FindIdentity`, `Roles.Remove` | unresolved |
| [`grant_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1687) | `*.FindIdentity`, `Roles.Validate`, `Roles.Assign` | unresolved |
| [`revoke_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/cloud/_identity.py#L1812) | `*.FindIdentity`, `Roles.Validate`, `Roles.Unassign` | unresolved |

55개 중 16개가 `go_mapping`, 39개가 `unresolved`입니다.

## 공통 차이

모든 helper는 먼저 `utils.ensure_service_version`으로 identity 어댑터가 v3인지 확인합니다. Go는 `conn.Identity(ctx)`가 주는 v3 client를 그대로 쓰고 이 확인을 하지 않습니다. 반환값은 Python Resource 대신 Gophercloud 모델 pointer이며, Python의 `is_enabled`는 Go에서 `Enabled`이고 모델에 없는 Keystone 필드는 `Extra`가 있는 모델에서만 남습니다. Keystone identity 리소스는 microversion을 쓰지 않으므로 협상 차이는 없습니다.

목록 helper는 limit 없이 GET 한 번을 보냅니다. Python `Resource.list`는 Keystone의 `links` dict를 따라가지 않지만 Go `All`과 `List`는 `links.next`가 null이 아니면 다음 URL을 이어서 요청합니다. Keystone은 보통 `next`를 null로 주기 때문에 실제 요청은 같지만, 응답이 다음 링크를 주면 Go가 요청을 더 보냅니다.

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

`update_domain`과 `delete_domain`은 domain_id를 주면 조회 없이 PATCH와 DELETE만 보내고, 이 부분은 `Domains.Update`, `Domains.Delete`로 재현됩니다. `delete_domain`은 먼저 `enabled: false`로 PATCH한 뒤 404를 무시하지 않는 DELETE를 보냅니다. 두 메서드 모두 name_or_id 경로에서 `find_domain`을 쓰기 때문에 판정은 아래 미해결 사유를 따릅니다.

## project와 user

`create_project`는 description을 주지 않아도 `description: null`을 보내고, 나머지 keyword 인자도 Project가 모르는 속성까지 본문에 넣습니다. Go는 typed 필드와 `Extra`로 같은 본문을 만듭니다. `list_users`는 `domain_id`와 `name`만 받으므로 `users.WithListOptions(users.ListOpts{DomainID: ..., Name: ...})`와 같고, `get_user_by_id`는 `Users.Get`과 같은 `GET /v3/users/<id>`입니다. deprecated `normalize` 인자는 경고만 냅니다.

`create_user`는 domain_id가 없으면 HTTP 전에 `SDKException`을 내지만 Go는 domain_id 없이 요청을 보냅니다. domain_id를 주고 `default_project`를 주지 않으면 `email: null`이 들어간 본문이 나가며 Go에서는 `Extra: map[string]any{"email": nil}`로 같게 만들 수 있습니다. `default_project`를 주면 `find_project`를 거쳐 `default_project_id`를 채우므로 판정은 `unresolved`입니다.

그룹 소속 helper는 ID를 알면 `Users.AddToGroup`, `Users.IsMemberOfGroup`, `Users.RemoveFromGroup`과 같은 PUT, HEAD, DELETE를 보냅니다. `IsMemberOfGroup`은 Python `check_user`처럼 HEAD 404를 거짓으로 돌려줍니다. 하지만 user와 group을 이름 또는 ID로 찾는 단계가 `find_user`, `find_group`이라서 판정은 `unresolved`입니다.

## 미해결 메서드

다음 메서드는 Proxy `find_project`, `find_user`, `find_group`, `find_role`, `find_domain` 가운데 하나 이상을 거칩니다. Go `FindIdentity`는 `resource.WithIdentityFindQuery("domain_id", ...)`와 함께 같은 GET과 이름 목록 요청을 보내지만, 그 Proxy 메서드들 자체가 Resource.list query 분류, 선택적 응답 cache, Resource 응답 변환, 상속된 continuation 경로 때문에 `unresolved`입니다. 그래서 그 위에 선 cloud helper도 같은 사유로 남겼습니다.

- `get_project`, `update_project`, `delete_project`: `find_project`를 거칩니다. `get_project`의 deprecated `filters`는 지역 필터도 필요합니다.
- `get_user`, `update_user`, `delete_user`: `find_user`를 거칩니다.
- `create_user`: `default_project` 경로가 `find_project`를 거칩니다.
- `add_user_to_group`, `is_user_in_group`, `remove_user_from_group`: `find_user`와 `find_group`을 거칩니다.
- `get_group`, `update_group`, `delete_group`: `find_group`을 거칩니다.
- `get_role`, `update_role`, `delete_role`: `find_role`을 거칩니다.
- `get_domain`, `update_domain`, `delete_domain`: name_or_id 경로가 `find_domain`을 거칩니다.
- `grant_role`, `revoke_role`: domain, role, user 또는 group, project를 모두 `find_*`로 찾습니다. 또 Python `validate_*_has_*_role`은 204가 아닌 HEAD를 거짓으로 보지만 Go `roles.API.Validate`와 `osinherit.API.Validate`는 404를 오류로 돌려주므로, 할당이 없다고 판단하는 일이 호출자 몫으로 남습니다. Python은 `wait`, `timeout` 인자를 받기만 하고 쓰지 않습니다.

다음 메서드는 결과를 `_utils._filter_list`로 지역에서 거릅니다. 이 함수는 ID나 이름의 정확한 일치와 fnmatch glob 일치, 중첩 dict 필터, deprecated jmespath 문자열을 처리합니다. 이 SDK에는 identity 행에 이 필터를 적용하는 공개 helper가 없고 identity collection에는 `WithFilter`나 `WithBodyFilter` 바인딩도 없습니다.

- `search_projects`, `search_users`, `search_groups`, `search_roles`, `search_domains`, `search_services`, `search_endpoints`
- `get_service`, `get_endpoint`: deprecated `filters`를 줄 때만 해당합니다.

`list_projects`, `list_domains`, `list_roles`는 임의의 filter keyword를 `Resource.list`에 넘깁니다. Python은 query가 아닌 Body 속성(예를 들어 description)을 각 행에서 지역으로 거르고, `limit`을 주면 마지막 행 ID를 marker로 삼아 빈 페이지가 나올 때까지 다시 요청합니다. Go collection에는 이 바인딩의 Body 필터가 없고 limit 기반 marker 요청도 하지 않습니다.

endpoint group helper(`create_endpoint_group`, `update_endpoint_group`, `list_endpoint_groups`, `search_endpoint_groups`, `get_endpoint_group`, `delete_endpoint_group`)는 Gophercloud v2.15.0에 Keystone `/OS-EP-FILTER/endpoint_groups` 바인딩이 없어서 SDK 고유 호출이 필요합니다.
