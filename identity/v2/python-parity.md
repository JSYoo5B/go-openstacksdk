# openstacksdk identity v2 Proxy 대응

이 문서는 고정한 openstacksdk 커밋 `ef55d7d`의 [identity v2 Proxy](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py)가 보내는 요청을 이 SDK의 `identity/v2` 호출과 비교합니다. Keystone v2.0 요청 자체의 status와 decode 규칙은 [native 호출 문서](native-calls.md)에 있고, 여기서는 Python 코드를 옮길 때 달라지는 기본값과 반환값을 정리합니다. 판정이 `go_mapping`이면 Python이 보내는 요청을 공개 Go API로 재현할 수 있다는 뜻이고, `unresolved`이면 Go에 없는 동작이 남아 있다는 뜻입니다. 판정 근거는 `extensions`, `roles`, `tenants`, `users` 패키지의 `python_parity_test.go`가 고정합니다.

| Python 메서드 | Go 호출 | 판정 |
|---|---|---|
| [`extensions`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L29) | `extensions.API.List` | go_mapping |
| [`get_extension`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L36) | `extensions.API.Get` | go_mapping |
| [`create_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L53) | 없음 | unresolved |
| [`delete_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L64) | 없음 | unresolved |
| [`find_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L95) | 없음 | unresolved |
| [`get_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L114) | 없음 | unresolved |
| [`roles`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L126) | `roles.API.List` (인자 없는 호출만) | unresolved |
| [`update_role`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L136) | 없음 | unresolved |
| [`create_tenant`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L150) | `tenants.API.Create` | go_mapping |
| [`delete_tenant`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L161) | `tenants.API.Remove` | go_mapping |
| [`find_tenant`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L192) | `tenants.API.Find`를 ID, 이름 순서로 호출 | go_mapping |
| [`get_tenant`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L211) | `tenants.API.Get` | go_mapping |
| [`tenants`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L223) | `tenants.API.List` | unresolved |
| [`update_tenant`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L233) | `tenants.API.Update`와 `tenants.WithUpdateField("id", id)` | go_mapping |
| [`create_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L249) | `users.API.Create` | go_mapping |
| [`delete_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L260) | `users.API.Remove` | go_mapping |
| [`find_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L291) | `users.API.Find`를 ID, 이름 순서로 호출 | go_mapping |
| [`get_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L310) | `users.API.Get` | go_mapping |
| [`users`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L322) | `users.API.List` | unresolved |
| [`update_user`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L332) | `users.API.Update`와 `users.WithUpdateField("id", id)` | go_mapping |
| [`wait_for_status`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L346) | tenant와 user의 `WaitFor`만 있음 | unresolved |
| [`wait_for_delete`](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/_proxy.py#L385) | tenant와 user의 `WaitForDeletion`만 있음 | unresolved |

22개 중 12개가 `go_mapping`, 10개가 `unresolved`입니다.

## extension

Python `extensions()`는 [Extension.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v2/extension.py)를 직접 구현해서 `GET /extensions`를 한 번 보내고 `extensions.values` 배열을 읽습니다. `extensions.API.List`도 같은 요청 하나를 보내고 같은 위치에서 행을 읽으며, 결과는 Gophercloud 공통 `Extension` 모델의 `Alias`, `Name`, `Namespace`, `Updated`, `Description`, `Links`입니다. Python은 응답 status를 검사하지 않고 JSON을 바로 읽지만, Go pager는 200, 204, 300 외의 status를 오류로 돌려줍니다. Python의 `updated_at`은 Go에서 `Updated` 문자열이고 시간으로 해석하지 않습니다.

`get_extension(alias)`는 `GET /extensions/{alias}`를 보내고 `extension` envelope를 풉니다. `extensions.API.Get(ctx, alias)`가 같은 요청이며, 404는 Python의 `NotFoundException` 대신 `gophercloud.ResponseCodeIs(err, 404)`로 확인합니다. Python은 Extension 객체도 받지만 Go는 alias 문자열만 받습니다.

## tenant와 user

`create_tenant`, `get_tenant`, `update_tenant`와 user의 같은 메서드는 Resource의 create, fetch, commit을 거쳐 `tenant`, `user` resource key로 감싼 본문을 보냅니다. Go의 `Create`, `Get`, `Update`가 같은 경로와 envelope를 사용합니다.

```go
enabled := true
tenant, err := tenants.New(client).Create(ctx, tenants.CreateOpts{Name: "demo", Description: "d", Enabled: &enabled})
// Python update_tenant("t-1", name="renamed")와 같은 본문
tenant, err = tenants.New(client).Update(ctx, "t-1", tenants.UpdateOpts{Name: "renamed"}, tenants.WithUpdateField("id", "t-1"))
```

Python `_update`는 ID 문자열로 새 Resource를 만들기 때문에 `id`도 변경된 속성으로 남아 PUT 본문에 `{"tenant": {"id": "t-1", ...}}`처럼 들어갑니다. Go `Update`는 경로에만 ID를 넣으므로 같은 본문이 필요하면 `WithUpdateField("id", id)`를 덧붙입니다. Keystone은 이 값을 무시하므로 생략해도 결과는 같습니다.

옮길 때 달라지는 점은 다음과 같습니다.

- Python은 400 미만 status를 모두 성공으로 받습니다. Go는 생성에 200과 201, 조회와 수정에 200만 받습니다.
- Python은 이름 없이도 `{"tenant": {}}`를 보내지만 `tenants.CreateOpts.Name`은 필수라서 Go는 HTTP 요청 전에 거부합니다. user 생성도 Go는 `Name`이나 `Username` 중 하나를 요구합니다.
- Go `UpdateOpts`는 빈 이름과 nil 포인터를 생략하고, `name`이나 `description` 같은 typed 필드는 `WithUpdateField`로 덮어쓸 수 없습니다. 그래서 Python처럼 `name=""`이나 `description=None`을 보내 null 또는 빈 값으로 지우는 요청은 만들 수 없습니다.
- Python Resource에 없는 속성은 본문에서 빠지므로 Python은 `tenantId`나 `username`을 보낼 수 없습니다. Go는 `CreateOpts.TenantID`, `Username`으로 보낼 수 있습니다.
- 반환값은 Python Resource 대신 Gophercloud `Tenant`, `User` 모델입니다. native `User`는 응답의 `tenant_id`만 읽고 다른 필드는 JSON tag 없이 대소문자 구분 없이 맞춥니다.

## 삭제

`delete_tenant`와 `delete_user`는 `ignore_missing=True`가 기본이어서 404를 조용히 넘깁니다. `tenants.API.Remove(ctx, resource.ID(id))`와 `users.API.Remove`도 기본으로 없는 대상을 성공으로 처리하고, `ignore_missing=False`에 해당하는 동작은 `resource.WithMissingError()`로 켭니다. 이때 오류는 `errors.Is(err, resource.ErrNotFound)`로 확인합니다. generated `Delete`를 직접 부르면 404가 그대로 오류가 되므로 Python 기본값과 다릅니다. 성공 status는 Python이 400 미만 전체, Go가 202와 204입니다.

## find

Python `find_tenant`와 `find_user`는 [Resource.find](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2489)를 그대로 씁니다. 먼저 `GET /tenants/{name_or_id}`를 보내고, 400, 403, 404가 나면 이름 query 없이 전체 목록을 읽어 ID나 이름이 같은 행을 찾습니다. 기본값 `ignore_missing=True`이면 못 찾았을 때 None을 돌려줍니다. Go에는 이 두 단계를 한 번에 하는 호출이 없어서 `Find`를 두 번 부릅니다.

```go
api := tenants.New(client)
tenant, err := api.Find(ctx, resource.ID(nameOrID), resource.WithIgnoreMissing())
if err == nil && tenant == nil {
	tenant, err = api.Find(ctx, resource.Name(nameOrID), resource.WithIgnoreMissing())
}
```

두 번째 호출은 이름 query 없이 `GET /tenants`를 보내고 `tenants_links`를 따라가며 이름이 같은 행을 찾으므로 Python이 보내는 요청 순서와 같습니다. 이름 단계에서 `resource.WithIgnoreMissing()`을 빼면 `ignore_missing=False`처럼 `resource.ErrNotFound`가 나고, 같은 이름이 둘이면 Python의 `DuplicateResource` 대신 `resource.ErrAmbiguous`가 납니다. Python은 GET 400과 403에서도 목록 단계로 넘어가지만 Go `Find`는 404만 nil로 바꾸므로, 그런 경우 호출자가 오류를 보고 이름 단계를 직접 이어야 합니다. 목록 단계의 일치 비교도 Go는 이름만 보고 Python은 ID와 이름을 함께 봅니다. user 목록은 Go pager가 `users_links`를 따라가지 않는 한 페이지 요청입니다.

## 목록

`tenants(**query)`는 기본 호출에서 Go `tenants.API.List`와 같은 `GET /tenants`를 보내고 `tenants_links`의 `rel=next`를 따라갑니다. `limit`과 `marker`도 `tenants.ListOpts`로 보낼 수 있습니다. 다만 Python은 `limit`이 있는데 next link가 없으면 마지막 ID를 marker로 삼아 빈 페이지가 나올 때까지 다시 요청하고, 본문에 `links` 키가 있으면 그쪽을 먼저 봅니다. Go pager는 `tenants_links`만 따라갑니다. 또 Python은 query 중 `description`, `is_enabled` 같은 Body 속성을 내려받은 행에 지역 필터로 적용하는데, Go `tenants.API.Resources`에는 이름 외의 지역 필터가 없습니다.

`users(**query)`에 대응하는 Go `users.API.List`는 query 없이 `GET /users`를 한 번만 보내고 `users_links`를 따라가지 않습니다. `users.API.All`에 `resource.WithQuery`를 주면 HTTP 요청 전에 `resource.ErrUnsupported`가 납니다. 그래서 Python의 `limit`, `marker`, 다음 페이지 요청과 Body 지역 필터를 재현할 수 없습니다.

## 대기

`wait_for_status`와 `wait_for_delete`는 Proxy에 넘긴 어떤 Resource든 받습니다. Tenant, User, Role, Extension에는 `status` 속성이 없어서 Python `wait_for_status`는 `attribute`를 바꾸지 않으면 HTTP 요청 전에 `AttributeError`가 나고, Go `tenants.API.WaitFor`와 `users.API.WaitFor`도 `resource.WithStatusAttribute` 없이는 `resource.ErrUnsupported`로 끝납니다. `resource.WithStatusAttribute("name")`처럼 문자열 필드를 고르면 Go도 대소문자를 무시하고 비교합니다.

`tenants.API.WaitForDeletion`과 `users.API.WaitForDeletion`은 같은 ID로 GET을 반복하다 404에서 성공합니다. Python 기본값에 맞추려면 `resource.WithTimeout(120 * time.Second)`를 주고, 간격은 Go 기본값 2초가 Python과 같습니다. 진행률 callback은 `resource.WithProgressCallback`이며 진행률 필드가 없어 Python처럼 0을 받습니다.

## role과 남은 메서드

Python Role은 `OS-KSADM/roles`에 대한 CRUD를 모두 허용하지만 Gophercloud v2 `roles`에는 `List`, `AddUser`, `DeleteUser`만 있습니다. `roles.API.List`는 인자 없는 `roles()`와 같은 `GET OS-KSADM/roles`를 보내지만 query를 받지 않고 `roles_links`도 따라가지 않습니다.

아래 메서드는 Go에 없는 동작이 남아 있어 `unresolved`입니다.

- `create_role`, `delete_role`, `get_role`, `update_role`: `POST`, `DELETE`, `GET`, `PUT OS-KSADM/roles[/{id}]`를 보내는 Go API가 없습니다.
- `find_role`: role 단건 GET이 없어서 Python의 첫 단계인 `GET OS-KSADM/roles/{name_or_id}`를 보낼 수 없습니다.
- `roles`: `limit`, `marker` query, `roles_links` 다음 페이지, Body 지역 필터를 지원하지 않습니다.
- `tenants`: 다음 link 없이 `limit`으로 이어지는 marker 요청과 이름 외 Body 지역 필터가 없습니다.
- `users`: query, `users_links` 다음 페이지, Body 지역 필터를 지원하지 않습니다.
- `wait_for_status`: Role과 Extension용 대기 호출이 없고, Python처럼 이미 받은 Resource의 상태가 목표와 같으면 HTTP 없이 돌려주는 단축 경로가 없습니다. Go는 항상 첫 GET을 보내고 갱신된 Resource 대신 모델 pointer를 돌려줍니다.
- `wait_for_delete`: Role과 Extension용 삭제 대기 호출이 없고, Python이 돌려주는 원래 Resource 대신 Go는 오류만 돌려줍니다.
