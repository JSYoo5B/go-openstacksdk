# Keystone v2.0 native 호출

`identity/v2`의 `Extensions`, `Roles`, `Tenants`, `Tokens`, `Users` generated 메서드는 Gophercloud `v2.15.0`의 [extensions](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v2/extensions/requests.go), [roles](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v2/roles/requests.go), [tenants](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v2/tenants/requests.go), [tokens](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v2/tokens/requests.go), [users](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v2/users/requests.go) 요청을 바꾸지 않고 호출합니다. SDK는 단건 호출 오류에 `resource.OperationError` 문맥을 더하고, typed 옵션에 없는 본문 필드를 `With...Field`로 덧붙이는 확장만 제공합니다. 목록 stream 오류에는 문맥을 더하지 않습니다.

Keystone v2.0 API는 Queens 릴리스에서 upstream Keystone에서 제거되었습니다. 그래서 이 호출들은 v2.0 endpoint를 아직 제공하는 오래된 배포에서만 동작하고, 현재 Keystone에는 `identity/v3`을 사용해야 합니다. 아래 표의 경로는 v2.0 endpoint(예: `.../v2.0/`) 기준입니다.

## 관리자 호출

기본 v2 policy에서 `OS-KSADM` 확장에 속하는 role·user 관리와 tenant 생성·조회·수정·삭제, token 검증(`Tokens.Get`)은 관리자 token과 admin endpoint가 필요합니다. `Extensions`와 `Tokens.Create`는 일반 사용자도 호출할 수 있습니다. `Tenants.List`는 public endpoint에서는 현재 token이 속한 tenant만, admin endpoint에서는 전체 tenant를 돌려줍니다.

## extension

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Get(ctx, alias)` | `GET extensions/{alias}` | 200 |
| `List(ctx)` | `GET extensions` | native pager |

`Get`은 `{"extension": {...}}`를 [공통 extension 모델](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/common/extensions/results.go)로 읽고, envelope가 없거나 null이면 오류 없이 nil을 돌려줍니다. v2 목록은 `{"extensions": {"values": [...]}}`처럼 중간 `values` 객체가 있는 형태만 받습니다. v3이나 다른 서비스처럼 `{"extensions": [...]}` 배열을 받으면 JSON 타입 오류가 납니다. 목록은 한 페이지로 끝나므로 응답에 `links`가 있어도 따라가지 않고, v2 page의 `IsEmpty`는 204를 따로 검사하지 않아 본문 없는 204는 `io.EOF`로 끝납니다.

## role

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `List(ctx)` | `GET OS-KSADM/roles` | native pager |
| `AddUser(ctx, tenantID, userID, roleID)` | `PUT tenants/{tenant}/users/{user}/roles/OS-KSADM/{role}`, 본문 없음 | 200, 201 |
| `DeleteUser(ctx, tenantID, userID, roleID)` | `DELETE tenants/{tenant}/users/{user}/roles/OS-KSADM/{role}` | 202, 204 |

`Role` 모델에는 JSON tag가 없어서 `id`, `name`, `description`, `serviceId`를 대소문자 구분 없이 필드 이름과 맞춥니다. 밑줄이 들어간 `service_id`는 읽지 않습니다. 목록은 한 페이지이며 `roles_links`를 따라가지 않습니다.

## tenant

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST tenants`, `{"tenant": {...}}` | 200, 201 |
| `Get(ctx, id)` | `GET tenants/{id}` | 200 |
| `Update(ctx, id, opts, options...)` | `PUT tenants/{id}`, `{"tenant": {...}}` | 200 |
| `Delete(ctx, id)` | `DELETE tenants/{id}` | 202, 204 |
| `List(ctx, options...)` | `GET tenants?marker=&limit=` | native pager |

`CreateOpts.Name`은 필수이고 `Description`, `Enabled`는 비어 있으면 생략합니다. `UpdateOpts`는 필수 값이 없어 빈 옵션이면 `{"tenant": {}}`를 보내고, `Description`은 포인터라 빈 문자열도 보낼 수 있습니다. 목록은 `tenants_links`의 `rel=next` 항목만 따라가고 `links.next`는 무시합니다. `WithListQuery`로 추가한 값은 native query 뒤에 합쳐집니다.

## token

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, auth, options...)` | `POST tokens`, `{"auth": {...}}` | 200, 203 |
| `Get(ctx, token)` | `GET tokens/{token}` | 200, 203 |
| `CreateURL(ctx)` | 요청 없음, `tokens` URL | 없음 |
| `GetURL(ctx, token)` | 요청 없음, `tokens/{token}` URL | 없음 |

`AuthOptions.Password`가 있으면 `{"auth": {"passwordCredentials": {"username": ..., "password": ...}}}`를 보내고, 없으면 `TokenID`로 `{"auth": {"token": {"id": ...}}}`를 보냅니다. 두 값을 모두 주면 암호가 우선하고 `TokenID`는 버려집니다. 암호 인증에 `Username`이 없거나 두 값이 모두 없으면 HTTP 요청 전에 오류가 납니다. `TenantID`와 `TenantName`은 비어 있지 않을 때 `auth` 안의 `tenantId`, `tenantName`으로 들어갑니다. `With...Field` 확장은 `auth` 객체 안에 합쳐지는데, `username`·`password`·`tenantId`·`tenantName`은 핵심 필드라 거부하고 이미 만든 `passwordCredentials`도 거부합니다. 반면 `token`은 선언된 입력 필드가 아니어서 암호 인증 본문 옆에 함께 들어갈 수 있습니다. native 요청은 `OmitHeaders`로 `X-Auth-Token`을 빼려 하지만 provider token을 그 다음에 붙이기 때문에, 이미 인증된 client로 호출하면 기존 token이 header에 그대로 실립니다.

응답은 [tokens README](tokens/README.md)의 `Authentication`으로 읽습니다. `access.token`의 `id`, `expires`, `tenant`가 `Token`이 되고 `access.serviceCatalog`는 `Catalog.Entries`, `access.user`는 `User`가 됩니다. `expires`는 `2006-01-02T15:04:05.999999Z` 형식이고 끝의 `Z`가 문자 그대로여서 `+09:00` 같은 offset이나 빈 값은 decode 오류가 됩니다. tenant·catalog·user가 없는 unscoped 응답은 해당 필드가 비어 있는 채로 성공합니다. `GetURL`은 token을 escape하지 않고 경로 뒤에 그대로 붙입니다.

## user

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST users`, `{"user": {...}}` | 200, 201 |
| `Get(ctx, id)` | `GET users/{id}` | 200 |
| `Update(ctx, id, opts, options...)` | `PUT users/{id}`, `{"user": {...}}` | 200 |
| `Delete(ctx, id)` | `DELETE users/{id}` | 202, 204 |
| `List(ctx)` | `GET users` | native pager |
| `ListRoles(ctx, tenantID, userID)` | `GET tenants/{tenant}/users/{user}/roles` | native pager |
| `ResourceURL(ctx, id)` | 요청 없음, `users/{id}` URL | 없음 |

`CreateOpts`에는 `Name`이나 `Username` 중 하나가 있어야 하고, 둘 다 없으면 HTTP 요청 전에 `gophercloud.ErrMissingInput`이 납니다. 나머지 필드와 `UpdateOpts` 전체는 빈 값이면 생략하므로 빈 `UpdateOpts`는 `{"user": {}}`를 보냅니다. 요청은 `tenantId`로 보내지만 native `User` 모델은 응답의 `tenant_id`만 읽습니다. Keystone v2는 사용자 응답에 `tenantId`를 쓰기 때문에 실제 서버 응답에서는 `TenantID`가 비어 있을 수 있습니다. 다른 필드는 JSON tag가 없어 대소문자 구분 없이 맞춥니다. `Delete`는 `*User`를 반환하지만 native 호출이 응답 본문을 읽지 않아 성공해도 항상 nil입니다. `List`와 `ListRoles`는 한 페이지로 끝나며 `users_links`, `roles_links`를 따라가지 않습니다.

## 공통 규칙

허용하지 않는 status는 `resource.OperationError`로 감싼 `gophercloud.ErrUnexpectedResponseCode`이고, `Expected`에 위 표의 성공 status가 들어 있습니다. nil 옵션, 빈 query key, 핵심 필드와 겹치는 확장 필드는 HTTP 요청 전에 거부합니다. 목록 pager는 200, 204, 300만 받으므로 404는 `Expected`가 `[200 204 300]`인 오류가 되고, 빈 목록은 오류 없이 끝나며, 본문 없는 204는 JSON 해석 단계에서 `io.EOF`가 됩니다.

openstacksdk의 identity v2 resource와 cloud helper는 별도로 추적합니다. 이 문서는 고정한 Gophercloud native 호출의 wire 계약만 다룹니다.
