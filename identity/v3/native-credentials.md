# Keystone v3 native application credential·credential·EC2 credential 호출

`service.ApplicationCredentials`, `service.Credentials`, `service.EC2Credentials`의 generated 메서드는 Gophercloud `v2.15.0`의 [applicationcredentials](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/applicationcredentials/requests.go), [credentials](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/credentials/requests.go), [ec2credentials](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/ec2credentials/requests.go) 요청을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError` 문맥만 더합니다. 세 목록 모두 Keystone 형식의 `links.next` 문자열을 따라가고 null이면 멈춥니다. 응답 envelope가 없으면 오류 없이 nil을 돌려줍니다.

## application credential

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, userID, opts, options...)` | `POST users/{user}/application_credentials` | 201 |
| `Get(ctx, userID, id)` | `GET users/{user}/application_credentials/{id}` | 200 |
| `List(ctx, userID, options...)` | `GET users/{user}/application_credentials` | native pager |
| `Delete(ctx, userID, id)` | `DELETE users/{user}/application_credentials/{id}` | 202, 204 |
| `ListAccessRules(ctx, userID)` | `GET users/{user}/access_rules` | native pager |
| `GetAccessRule(ctx, userID, id)` | `GET users/{user}/access_rules/{id}` | 200 |
| `DeleteAccessRule(ctx, userID, id)` | `DELETE users/{user}/access_rules/{id}` | 202, 204 |

`CreateOpts.Name`은 필수입니다. `Unrestricted`는 omitempty가 없어 false도 항상 보냅니다. `ExpiresAt`은 `2006-01-02T15:04:05.999999` 형식으로 보내는데 시간대를 바꾸지 않고 그 시각의 location 값 그대로 적으므로, UTC가 아닌 시각을 주면 서버는 시간대가 빠진 값을 다른 시각으로 해석합니다. UTC 시각을 넘겨야 합니다. 응답의 `secret`은 생성 직후에만 받을 수 있는 비밀 값이고, `expires_at`은 시간대 없는 형식만 받아 `Z`가 붙으면 decode 오류입니다.

## credential

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST credentials`, `{"credential": {...}}` | 201 |
| `Get(ctx, id)` | `GET credentials/{id}` | 200 |
| `List(ctx, options...)` | `GET credentials?user_id=&type=` | native pager |
| `Update(ctx, id, opts, options...)` | `PATCH credentials/{id}` | 200 |
| `Delete(ctx, id)` | `DELETE credentials/{id}` | 202, 204 |

`CreateOpts`의 `Blob`·`Type`·`UserID`는 필수입니다. `Blob`은 JSON 문자열 그대로 보내고 받습니다. `Update`는 PATCH이며 빈 필드는 생략합니다.

## EC2 credential

`EC2Credentials`는 `users/{user}/credentials/OS-EC2` 아래에서 `Create`(201)·`Get`(200)·`List`·`Delete`(202·204)를 제공합니다. `Create`의 `TenantID`는 필수이고 본문은 envelope 없는 `{"tenant_id": ...}`라 확장 필드도 최상위에 붙습니다. 응답은 `credential` key의 access·secret 쌍이며, `Get`·`Delete`의 ID는 access key입니다.
