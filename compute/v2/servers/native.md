# Nova native 서버 호출

`servers.New(client)`(또는 `service.Servers`)의 generated 메서드는 Gophercloud `v2.15.0`의 [servers 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/servers/requests.go)을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError{Resource: "servers"}` 문맥만 더하며, 다른 status는 native `gophercloud.ErrUnexpectedResponseCode`로 남습니다. ID는 escape 없이 경로 segment로 이어 붙입니다.

## 조회·목록·생성·수정·삭제

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Get(ctx, id)` | `GET servers/{id}` | 200, 203 |
| `List(ctx, options...)` | `GET servers/detail` | native pager |
| `ListSimple(ctx, options...)` | `GET servers` | native pager |
| `Create(ctx, opts, options...)` | `POST servers` | 200, 202 |
| `Update(ctx, id, opts, options...)` | `PUT servers/{id}` | 200 |
| `Delete(ctx, id)` | `DELETE servers/{id}` | 202, 204 |

`Server`의 `image`가 빈 문자열이면(볼륨 부팅) `Image`는 nil이 됩니다. 두 목록은 `ListOpts`(이름·상태·태그·`all_tenants` 등)와 `WithListQuery`/`WithListSimpleQuery` 확장 query를 보내고, `servers_links`의 `rel=next` href를 받은 그대로 따라갑니다. 호출자가 순회를 멈추면 다음 페이지를 요청하지 않습니다.

`Create`는 `CreateOpts`를 `{"server": {...}}`로 감쌉니다. `Name`은 필수이며, `UserData`는 base64가 아니면 인코딩하고, `SecurityGroups`는 `{"name": ...}` 목록으로, `Networks`는 `[]servers.Network` 또는 `"auto"`/`"none"` 문자열로 보냅니다. 다른 문자열은 HTTP 전에 오류입니다. `WithCreateField`의 확장 필드는 단일 `server` envelope 안에 들어가고 기존 key와 겹치면 HTTP 전에 거부됩니다. `WithCreateHintOpts`의 scheduler hints는 최상위 `os:scheduler_hints`로 보내며, UUID가 아닌 `Group` 같은 잘못된 hint는 HTTP 전에 오류입니다.

`Update`는 이름·access IP·`Hostname`을 `{"server": {...}}`로 보내고 비어 있는 값은 생략합니다. 생성·수정 응답은 서버가 돌려준 `Server`이며 SDK가 요청 값을 합성하지 않습니다. Python의 mutable Server Resource와 대기·주소 helper는 [서버 생성 사용법](../../create-with-floating-ip.md)과 [서버 대기](../../../docs/service-waits.md)를 참고합니다.
