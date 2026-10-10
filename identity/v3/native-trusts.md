# Keystone v3 native trust·사용자 본인 호출

`service.Trusts`(`identity/v3/trusts`)와 사용자 본인이 쓰는 `Projects.ListAvailable`·`Users.ChangePassword`의 generated 메서드는 Gophercloud `v2.15.0`의 [trusts](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/trusts/requests.go), [projects](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/projects/requests.go), [users](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/identity/v3/users/requests.go) 요청을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError` 문맥만 더하고, 목록은 Keystone `links.next` 문자열을 따라갑니다.

## trust

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST OS-TRUST/trusts`, `{"trust": {...}}` | 201 |
| `Get(ctx, id)` | `GET OS-TRUST/trusts/{id}` | 200 |
| `List(ctx, options...)` | `GET OS-TRUST/trusts?trustor_user_id=&trustee_user_id=` | native pager |
| `Delete(ctx, id)` | `DELETE OS-TRUST/trusts/{id}` | 202, 204 |
| `ListRoles(ctx, id)` | `GET OS-TRUST/trusts/{id}/roles` | native pager |
| `GetRole(ctx, id, roleID)` | `GET OS-TRUST/trusts/{id}/roles/{role}` | 200 |
| `CheckRole(ctx, id, roleID)` | `HEAD OS-TRUST/trusts/{id}/roles/{role}` | 200 |

`CreateOpts`의 `TrusteeUserID`·`TrustorUserID`는 필수입니다. `Impersonation`은 omitempty가 없어 false도 항상 보내고, 나머지 빈 값은 생략합니다. `ExpiresAt`은 `2006-01-02T15:04:05.999999Z` 형식으로 보내는데 끝의 `Z`는 문자 그대로라 시각을 UTC로 바꾸지 않습니다. 다른 시간대의 시각은 그 시계 값에 `Z`가 붙어 서버가 다른 시각으로 해석하므로 UTC 시각을 넘겨야 합니다. 응답 시각은 RFC3339이고 null `deleted_at`은 zero time입니다. `CheckRole`은 native HEAD 기본값이라 200만 받습니다.

## 사용자 본인 호출

`Projects.ListAvailable(ctx)`는 `GET auth/projects`로 현재 token이 접근할 수 있는 project를 돌려줍니다. 옵션은 없습니다. `Users.ChangePassword(ctx, userID, opts, options...)`는 `POST users/{user}/password`에 `{"user": {"original_password": ..., "password": ...}}`를 보내고 204만 받습니다. 두 암호 모두 필수 검사와 omitempty가 없어 빈 문자열도 그대로 보냅니다. 사용자의 project·group 목록은 기존 판정의 [사용자 membership 문서](users/memberships.md)를 참고합니다.
