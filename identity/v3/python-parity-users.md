# openstacksdk identity v3 Proxy 대응: domain·group·project·user·role과 role 할당

이 문서는 고정한 openstacksdk 커밋 `ef55d7d`의 [identity v3 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py) 가운데 domain, group, project, user, role의 생성·조회·수정·삭제·목록과 group membership, 암호 변경, role 할당 메서드가 보내는 요청을 이 SDK의 `identity/v3` 호출과 비교합니다. Keystone 요청 자체의 status, 입력 검사, decode 규칙은 [native 관리 호출 문서](native-admin-a.md)와 [native trust·사용자 본인 호출 문서](native-trusts.md)에 있고, 여기서는 Python 코드를 옮길 때 달라지는 기본값과 반환값을 정리합니다. `find_*`, `user_projects`, `user_groups`는 이미 판정되어 있어 [자동 조회 비교](../../docs/finding-identities.md)와 [사용자 membership 문서](users/memberships.md)를 참고합니다.

판정이 `go_mapping`이면 Python이 보내는 요청을 공개 Go API로 재현할 수 있다는 뜻이고, `unresolved`이면 Go에 없는 동작이 남아 있다는 뜻입니다. 판정 근거는 `domains`, `groups`, `projects`, `users`, `roles`, `osinherit` 패키지의 `python_parity_test.go`가 고정합니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| [`create_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L217) | `domains.API.Create` | unresolved |
| [`delete_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L228) | `domains.API.Remove` | go_mapping |
| [`get_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L278) | `domains.API.Get` | unresolved |
| [`domains`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L290) | `domains.API.List` | unresolved |
| [`update_domain`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L301) | `domains.API.Update`와 `domains.WithUpdateField("id", id)` | unresolved |
| [`create_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L738) | `groups.API.Create` | go_mapping |
| [`delete_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L749) | `groups.API.Remove` | go_mapping |
| [`get_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L805) | `groups.API.Get` | go_mapping |
| [`groups`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L818) | `groups.API.List` | unresolved |
| [`update_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L829) | `groups.API.Update`와 `groups.WithUpdateField("id", id)` | go_mapping |
| [`add_user_to_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L843) | `users.API.AddToGroup` | go_mapping |
| [`remove_user_from_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L860) | `users.API.RemoveFromGroup` | go_mapping |
| [`check_user_in_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L877) | `users.API.IsMemberOfGroup` | go_mapping |
| [`group_users`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L894) | `users.API.ListInGroup` | unresolved |
| [`create_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1019) | `projects.API.Create` | go_mapping |
| [`delete_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1030) | `projects.API.Remove` | go_mapping |
| [`get_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1086) | `projects.API.Get` | go_mapping |
| [`projects`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1098) | `projects.API.List` | unresolved |
| [`update_project`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1150) | `projects.API.Update`와 `projects.WithUpdateField("id", id)` | go_mapping |
| [`create_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1335) | `users.API.Create` | go_mapping |
| [`delete_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1346) | `users.API.Remove` | go_mapping |
| [`get_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1402) | `users.API.Get` | go_mapping |
| [`users`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1429) | `users.API.List` | unresolved |
| [`update_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1440) | `users.API.Update`와 `users.WithUpdateField("id", id)` | go_mapping |
| [`update_password`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1452) | `users.API.ChangePassword` | go_mapping |
| [`create_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1699) | `roles.API.Create` | go_mapping |
| [`delete_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1710) | `roles.API.Remove` | go_mapping |
| [`get_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1766) | `roles.API.Get` | go_mapping |
| [`roles`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1778) | `roles.API.List` | unresolved |
| [`update_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1788) | `roles.API.Update`와 `roles.WithUpdateField("id", id)` | go_mapping |
| [`role_assignments_filter`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1877) | `roles.API.ListAssignmentsOnResource` | go_mapping |
| [`role_assignments`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1983) | `roles.API.ListAssignments` | unresolved |
| [`assign_domain_role_to_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1999) | `roles.API.Assign`, inherited=True는 `osinherit.API.Assign` | go_mapping |
| [`unassign_domain_role_from_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2023) | `roles.API.Unassign`, inherited=True는 `osinherit.API.Unassign` | go_mapping |
| [`validate_user_has_domain_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2047) | `roles.API.Validate`, inherited=True는 `osinherit.API.Validate` | go_mapping |
| [`assign_domain_role_to_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2070) | `roles.API.Assign`, inherited=True는 `osinherit.API.Assign` | go_mapping |
| [`unassign_domain_role_from_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2094) | `roles.API.Unassign`, inherited=True는 `osinherit.API.Unassign` | go_mapping |
| [`validate_group_has_domain_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2118) | `roles.API.Validate`, inherited=True는 `osinherit.API.Validate` | go_mapping |
| [`assign_project_role_to_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2141) | `roles.API.Assign`, inherited=True는 `osinherit.API.Assign` | go_mapping |
| [`unassign_project_role_from_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2166) | `roles.API.Unassign`, inherited=True는 `osinherit.API.Unassign` | go_mapping |
| [`validate_user_has_project_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2191) | `roles.API.Validate`, inherited=True는 `osinherit.API.Validate` | go_mapping |
| [`assign_project_role_to_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2215) | `roles.API.Assign`, inherited=True는 `osinherit.API.Assign` | go_mapping |
| [`unassign_project_role_from_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2240) | `roles.API.Unassign`, inherited=True는 `osinherit.API.Unassign` | go_mapping |
| [`validate_group_has_project_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2265) | `roles.API.Validate`, inherited=True는 `osinherit.API.Validate` | go_mapping |
| [`assign_system_role_to_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2289) | `roles.API.Assign` (`System: true`) | go_mapping |
| [`unassign_system_role_from_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2309) | `roles.API.Unassign` (`System: true`) | go_mapping |
| [`validate_user_has_system_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2329) | `roles.API.Validate` (`System: true`) | go_mapping |
| [`assign_system_role_to_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2349) | `roles.API.Assign` (`System: true`) | go_mapping |
| [`unassign_system_role_from_group`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2369) | `roles.API.Unassign` (`System: true`) | go_mapping |
| [`validate_group_has_system_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L2389) | `roles.API.Validate` (`System: true`) | go_mapping |

50개 중 40개가 `go_mapping`, 10개가 `unresolved`입니다.

## 공통 차이

Python 메서드는 ID 문자열과 Resource 객체를 모두 받고 변경 가능한 Resource를 돌려줍니다. Go 호출은 ID 문자열만 받고 Gophercloud 모델 포인터나 `iter.Seq2`를 돌려줍니다. identity v3 Resource에는 `_max_microversion`이 없어서 Python도 microversion header를 보내지 않으므로 이 부분은 두 쪽이 같습니다.

Python은 [Resource](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py)의 `raise_from_response`를 거쳐 400 미만 status를 모두 성공으로 받습니다. Go native 호출은 정해진 status만 받습니다. 생성은 201(project만 201, 202), 조회와 수정은 200, 삭제는 202, 204, membership·암호 변경·role 할당은 204입니다. 그래서 Python에서 성공하던 200 응답이 Go에서는 `gophercloud.ErrUnexpectedResponseCode`가 될 수 있습니다.

## domain·group·project·user·role 생성, 조회, 수정

`create_*`, `get_*`, `update_*`는 resource key로 감싼 본문을 `POST {collection}`, `GET {collection}/{id}`, `PATCH {collection}/{id}`로 보냅니다. Go의 `Create`, `Get`, `Update`가 같은 경로와 envelope를 씁니다.

```go
enabled := false
project, err := projects.New(client).Create(ctx, projects.CreateOpts{Name: "demo", DomainID: "default", Enabled: &enabled})
// Python update_project("p-1", description=None, tags=[])와 같은 본문
empty := []string{}
project, err = projects.New(client).Update(ctx, "p-1",
    projects.UpdateOpts{Tags: &empty, Extra: map[string]any{"description": nil}},
    projects.WithUpdateField("id", "p-1"))
```

Python `Proxy._update`는 ID 문자열로 새 Resource를 만들기 때문에 `id`가 변경된 속성으로 남아 PATCH 본문에 `{"project": {"id": "p-1", ...}}`처럼 들어갑니다. Go `Update`는 경로에만 ID를 넣으므로 같은 본문이 필요하면 `WithUpdateField("id", id)`를 덧붙입니다. Keystone은 이 값을 무시하므로 생략해도 결과는 같습니다.

옮길 때 달라지는 점은 다음과 같습니다.

- 모든 `CreateOpts.Name`이 필수라서 Python처럼 이름 없이 생성 요청을 보내려 하면 Go는 HTTP 전에 거부합니다.
- typed `UpdateOpts`는 빈 문자열과 nil 포인터를 생략합니다. Python에서 `description=None`처럼 null을 보내려면 group·project·user·role은 `Extra` map에 nil 값을 넣습니다. typed 필드와 같은 key를 `WithUpdateField`로 주면 HTTP 전에 오류가 납니다.
- Python Resource에 typed 필드가 없는 속성(user의 `email`, project의 알 수 없는 속성 등)은 `CreateOpts.Extra`, `UpdateOpts.Extra`나 `With...Field`로 envelope 안에 넣습니다. 응답에서도 이런 key는 `Extra`에 모입니다. project의 `links`도 `Extra`에 들어갑니다.
- user의 `password_expires_at`은 Go에서 시간대 없는 Keystone 형식으로 해석한 `time.Time`입니다. Python은 문자열 그대로 두므로 끝에 `Z`가 붙은 값은 Go에서만 decode 오류가 됩니다.

domain은 이 중 생성, 조회, 수정이 모두 `unresolved`입니다. native `domains.Domain`에는 `Options`도 `Extra`도 없어서 Keystone이 돌려주는 `options` 객체가 버려지고, Python `Domain.options`에 해당하는 값을 어떤 공개 Go 호출로도 읽을 수 없습니다. 요청 쪽에서는 `domains.WithCreateField("options", ...)`로 options를 보낼 수 있지만, `domains.UpdateOpts`에 `Extra`가 없어서 `description`이나 `enabled`를 null로 보내는 수정도 만들 수 없습니다.

## 삭제

`delete_*`는 Python 기본값이 `ignore_missing=True`라서 404를 무시합니다. Go에서는 생성된 `Delete` 대신 공통 collection의 `Remove`를 씁니다.

```go
err := users.New(client).Remove(ctx, resource.ID("u-1"))                            // 404면 nil
err = users.New(client).Remove(ctx, resource.ID("u-1"), resource.WithMissingError()) // ignore_missing=False
```

`Remove`는 404만 성공으로 바꾸고 409 같은 다른 오류는 그대로 돌려줍니다. 생성된 `Delete`를 직접 부르면 404도 오류입니다.

## 목록

`domains()`, `groups()`, `projects()`, `users()`, `roles()`, `group_users()`, `role_assignments()`는 모두 `unresolved`입니다. 인자 없이 부르면 Python과 Go 모두 `GET {collection}` 하나를 보내고, Python query 이름은 각 `ListOpts` 필드나 `WithListQuery`로 같은 wire key가 됩니다. 판정을 막는 동작은 다음 두 가지입니다.

- Python `Resource.list`는 `limit`이 있으면 다음 링크가 없어도 마지막 ID를 `marker`로 붙여 빈 페이지가 나올 때까지 다시 요청합니다. Go native pager는 첫 요청만 보냅니다.
- Python은 query 매핑에 없는 Body 속성(예: `users(email=...)`, `domains(description=...)`)을 받은 응답에 지역 filter로 적용합니다. 이 collection들에는 Body filter binding이 없어서 `resource.WithBodyFilter`는 HTTP 전에 `resource.ErrUnsupported`가 됩니다.

나머지 차이는 판정에 영향을 주지 않지만 알아 둘 만합니다. Python은 bool 값을 `enabled=True`처럼 대문자 텍스트로 보내고 typed `ListOpts`는 `enabled=true`를 보냅니다. Python과 같은 텍스트가 필요하면 `WithListQuery("enabled", "True")`를 씁니다. Python은 Keystone `links` dict에 `rel` key가 없어서 `links.next`를 따라가지 않지만 Go pager는 따라갑니다. Python은 모르는 query 이름을 버리고 Go는 `WithListQuery` key를 모두 서버에 보냅니다.

group·project·user·role의 `ListOpts.Filters`는 `name__contains` 같은 비교 filter를 보내지만, Gophercloud가 `q:"-"` 태그를 건너뛰지 않아 `-=...` 모양의 query가 하나 더 붙습니다. Python은 비교 filter를 아예 보낼 수 없으므로 대응 관계에는 쓰지 않고, 필요하면 `WithListQuery("name__contains", "x")`를 씁니다.

`group_users(group, **attrs)`는 `users.API.ListInGroup(ctx, groupID, ...)`이 같은 `GET groups/{id}/users`와 User query 매핑을 보내지만 위의 marker 반복과 지역 filter가 없습니다.

`role_assignments(**query)`는 `role_id`, `user_id`, `group_id`, `scope_project_id`, `scope_domain_id`, `scope_system`, `inherited_to`를 `role.id`, `user.id`, `scope.OS-INHERIT:inherited_to` 같은 점 표기 key로 바꿉니다. `roles.ListAssignmentsOpts`가 앞의 여섯 개를 typed 필드로 받고 `inherited_to`와 Python 텍스트의 bool은 `WithListAssignmentsQuery`로 보냅니다. Keystone은 이 flag가 있으면 값이 `0`일 때만 꺼진 것으로 보므로 Python의 `effective=False`도 Go의 `false`처럼 켜진 것으로 해석됩니다. 다만 native `RoleAssignment`가 `links`와 scope의 `OS-INHERIT:inherited_to`를 버려서 상속 할당과 직접 할당을 결과로 구분할 수 없고, 그래서 `unresolved`입니다.

`role_assignments_filter(domain|project|system, group|user)`는 `go_mapping`입니다. `roles.API.ListAssignmentsOnResource`가 같은 `GET {domains|projects}/{id}/{users|groups}/{id}/roles`나 `GET system/{users|groups}/{id}/roles` 하나를 보냅니다. Python은 domain, project, system을 셋 다 줄 때만 거부하고 둘을 주면 domain, project, system 순서로 하나를 고르는 반면, Go는 정확히 하나를 요구하고 나머지는 HTTP 전에 거부합니다. Python은 system 인자 값을 쓰지 않고 항상 `/system`을 쓰므로 Go에서는 `System: true`만 둡니다.

## group membership과 암호 변경

`add_user_to_group`, `remove_user_from_group`, `check_user_in_group`은 `users.API.AddToGroup`, `RemoveFromGroup`, `IsMemberOfGroup`과 같은 `PUT`, `DELETE`, `HEAD groups/{group}/users/{user}`를 보냅니다. Go 인자 순서는 group ID가 먼저이고 Python은 user가 먼저입니다. `check_user_in_group`과 `IsMemberOfGroup`은 모두 204를 true, 404를 오류 없는 false로 돌려주고 403은 오류입니다. Python은 그 밖의 400 미만 status를 false로 보지만 Go는 오류로 돌려줍니다.

`update_password(user, current_password, password)`는 `users.API.ChangePassword(ctx, userID, users.ChangePasswordOpts{OriginalPassword: ..., Password: ...})`가 같은 `POST users/{id}/password`와 `{"user": {"original_password": ..., "password": ...}}` 본문을 보냅니다. Go는 204만 성공으로 받습니다.

## role 할당

`assign_*`, `unassign_*`, `validate_*` 18개는 [Domain](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/domain.py), [Project](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/project.py), [System](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/system.py)의 helper를 거쳐 `PUT`, `DELETE`, `HEAD {domains|projects}/{id}/{users|groups}/{id}/roles/{role}`나 `system/{users|groups}/{id}/roles/{role}`를 보냅니다. Go에서는 `roles.API.Assign`, `Unassign`, `Validate`에 target과 actor를 하나씩 채운 옵션을 넘깁니다.

```go
api := roles.New(client)
// Python assign_project_role_to_user("p-1", "u-1", "r-1")
err := api.Assign(ctx, "r-1", roles.AssignOpts{ProjectID: "p-1", UserID: "u-1"})
// Python validate_group_has_system_role("g-1", "r-1", "all")
hasRole := api.Validate(ctx, "r-1", roles.ValidateOpts{System: true, GroupID: "g-1"}) == nil
// Python assign_domain_role_to_group("d-1", "g-1", "r-1", inherited=True)
err = osinherit.New(client).Assign(ctx, "r-1", osinherit.AssignOpts{DomainID: "d-1", GroupID: "g-1"})
```

domain·project 할당에 `inherited=True`를 주면 Python은 경로 앞뒤에 `OS-INHERIT`와 `inherited_to_projects`를 붙입니다. Go에서는 같은 경로를 보내는 `osinherit.API.Assign`, `Unassign`, `Validate`를 부릅니다. system 할당에는 `inherited` 인자가 없습니다.

status 처리는 Python과 다릅니다. Python helper는 `Proxy.request`의 `raise_exc=False` 기본값으로 요청하고 결과를 버리기 때문에 `assign_*`와 `unassign_*`는 403이나 404에도 예외 없이 None을 돌려줍니다. `validate_*`는 204일 때만 True이고 404를 포함한 다른 status는 모두 False입니다. Go 호출은 204만 nil로 받고 나머지는 status가 담긴 오류로 돌려줍니다. 그래서 Python과 같은 bool이 필요하면 `Validate`의 오류가 nil인지만 보면 되고, 할당 실패를 Python처럼 무시하려면 오류를 버리면 됩니다. Go 쪽이 실패 원인을 더 많이 알려 주므로 이 차이는 `go_mapping` 판정 안의 차이로 기록합니다.

## unresolved 목록

- `create_domain`, `get_domain`, `update_domain`: native `domains.Domain`이 `options` 응답 필드를 버립니다. `update_domain`은 typed 필드를 null로 보내는 방법도 없습니다.
- `domains`, `groups`, `projects`, `users`, `roles`: `limit`이 있을 때의 marker 반복 요청과 Body 속성 지역 filter가 없습니다. `domains`는 options 필드도 잃습니다.
- `group_users`: `ListInGroup`에 marker 반복 요청과 User Body 속성 지역 filter가 없습니다.
- `role_assignments`: native `RoleAssignment`가 `links`와 `OS-INHERIT:inherited_to`를 버리고, marker 반복 요청과 `role`·`scope`·`user`·`group` dict 지역 filter가 없습니다.
