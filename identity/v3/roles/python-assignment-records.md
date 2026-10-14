# Keystone role assignment record API

이 문서는 `roles.API`에 SDK가 직접 추가한 role assignment record 목록을 설명합니다. Gophercloud v2.15.0 `RoleAssignment` 모델은 `links`와 scope의 `OS-INHERIT:inherited_to` 키를 버려서 상속된 할당과 직접 할당을 구분할 수 없으므로, record 목록은 각 행의 JSON을 그대로 보관합니다. 기준 소스는 [`_proxy.py` role_assignments](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1983-L1997)와 [`role_assignment.py`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/role_assignment.py)입니다.

| Python `conn.identity` | Go `roles.API` | 요청 |
|---|---|---|
| `role_assignments(**query)` | `ListAssignmentRecords(ctx, options...)`, `AllAssignmentRecords(ctx, options...)` | `GET role_assignments` |

```go
api := roles.New(identityClient)
values, err := api.AllAssignmentRecords(ctx,
    roles.WithAssignmentRecordListQuery("user_id", userID),
    roles.WithAssignmentRecordListQuery("scope", map[string]any{"OS-INHERIT:inherited_to": "projects"}))
```

## 요청

Python `QueryParameters`가 선언한 이름은 server query가 됩니다. `role_id`, `user_id`, `group_id`, `scope_project_id`, `scope_domain_id`, `scope_system`, `inherited_to`는 각각 `role.id`, `user.id`, `group.id`, `scope.project.id`, `scope.domain.id`, `scope.system`, `scope.OS-INHERIT:inherited_to`로 나가고, `effective`, `include_names`, `include_subtree`는 이름 그대로 나갑니다. 값은 Requests 방식으로 인코딩하므로 `true`는 `True`가 됩니다. Python처럼 점이 들어간 server 이름도 받으며, 같은 query를 두 이름으로 주면 client 이름의 값이 이깁니다.

`id`, `name`, `links`, `role`, `scope`, `user`, `group`은 Python `Resource.list`처럼 응답 행을 로컬에서 거릅니다. 사전 값은 `_dict_filter`와 같이 중첩된 부분 집합으로 비교하므로 `{"OS-INHERIT:inherited_to": "projects"}` 같은 scope 조건도 쓸 수 있습니다. 그 밖의 이름은 무시하고, 요청에는 `Accept: application/json`이 붙습니다.

`WithAssignmentRecordListLimit`, `WithAssignmentRecordListMarker`, `WithAssignmentRecordListMaxItems`, `WithAssignmentRecordListPaginated(false)`는 Python의 `limit`, `marker`, `max_items`, `paginated`와 같습니다. Keystone 할당 행에는 `id`가 없어서 Python은 limit을 주면 첫 페이지 뒤에 marker가 None인 요청을 만들려다 무한 페이지 오류를 냅니다. Go도 첫 페이지 행을 돌려준 뒤 marker가 없다는 오류로 멈추고 두 번째 요청을 보내지 않습니다.

## 응답

Python은 400 미만의 모든 status를 성공으로 받으므로 Go도 200부터 399까지를 받습니다. 400 이상은 Gophercloud `ErrUnexpectedResponseCode`이고 `resource.OperationError`(Resource `role_assignments`, Operation `List`)로 감쌉니다.

`AssignmentRecord`의 `Role`, `Scope`, `User`, `Group`, `Links`는 Keystone이 돌려준 객체의 raw JSON이고 값이 없으면 nil입니다. 그래서 `Scope`에서 `OS-INHERIT:inherited_to`를 읽어 상속 여부를 알 수 있고, `include_names`가 붙인 이름도 남습니다. `Body`는 선언된 Body 키 7개를 raw JSON으로 보관하고 값이 없으면 null이며, `Wire`는 실제 행 전체, `Envelope`, `Header`, `StatusCode`는 그 페이지의 실제 응답입니다.

## Python과 다른 점

Python은 `role`, `scope`, `user`, `group`을 dict로 변환한 Resource를 돌려주지만 Go는 raw JSON을 돌려줍니다. 로컬 필터는 JSON 값을 비교하므로 bool과 숫자를 구분합니다. 목록은 Keystone `links.next`가 문자열이면 따라가는데, pinned Python 페이지네이터는 이 dict 형태를 무시합니다. limit을 준 목록이 실패하는 시점은 같지만 오류 타입은 Python `SDKException`이 아니라 `resource.ErrInvalidOption`을 감싼 `resource.OperationError`입니다. 헤더 옵션은 Go 확장입니다.
