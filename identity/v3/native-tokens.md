# Keystone v3 native token·catalog 호출

`service.Tokens`(`identity/v3/tokens`)와 `service.Catalog`(`identity/v3/catalog`)의 generated 메서드는 Gophercloud `v2.15.0`의 [tokens](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/tokens/requests.go)와 [catalog](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/catalog/requests.go) 요청을 호출합니다. SDK는 오류에 `resource.OperationError` 문맥을 더합니다. Connection이 수행하는 인증은 이 문서의 호출과 별개이며 [인증과 설치](../../docs/install.md)를 참고합니다.

| 메서드 | 요청 | 결과 | 성공 status |
|---|---|---|---|
| `Tokens.Create(ctx, opts, options...)` | `POST auth/tokens` | `*Token` | 201, 202 |
| `Tokens.Get(ctx, token)` | `GET auth/tokens`, `X-Subject-Token` | `*Token` | 200, 203 |
| `Tokens.Validate(ctx, token)` | `HEAD auth/tokens`, `X-Subject-Token` | `bool` | 200, 204, 404 |
| `Tokens.Revoke(ctx, token)` | `DELETE auth/tokens`, `X-Subject-Token` | `*Token` | 202, 204 |
| `Catalog.List(ctx)` | `GET auth/catalog` | catalog entry stream | native pager |

`Create`의 본문은 `AuthOptions`에서 password(사용자 이름+domain 또는 사용자 ID), token, application credential 중 하나의 방식을 고르고 `Scope`를 project·domain·system 범위로 바꿉니다. 인증 방식이 하나도 없거나 `opts`가 nil이면 HTTP 전에 오류입니다. `WithCreateField`의 확장 필드는 `auth` 객체에만 들어가며 scope 객체에는 넣지 않습니다. native `Create`가 header builder를 호출하지 않으므로 header 확장 옵션은 제공하지 않고, 설정에 남은 raw header 확장은 HTTP 전에 거부합니다. 결과 `Token`의 ID는 본문이 아니라 응답 `X-Subject-Token` header에서 읽고, 만료 시각은 RFC3339 `expires_at`입니다.

native 구현은 `X-Auth-Token`을 빼도록 요청하지만 Gophercloud가 그 뒤에 provider token을 다시 붙이므로, 이미 인증한 client로 `Create`를 호출하면 현재 token도 함께 전송됩니다.

`Validate`는 200·204를 true, 404를 오류 없는 false로 돌려줍니다. 이 메서드는 native 오류를 감싸지 않아서 다른 status의 오류에는 `OperationError` 문맥이 없습니다. `Revoke`는 본문 없는 응답을 decode하므로 빈 `Token`을 돌려줍니다. `Catalog.List`는 `catalog` 배열을 service별 endpoint 목록으로 돌려줍니다.
