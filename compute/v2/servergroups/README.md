# 서버 그룹 native 호출

`servergroups.New(client)`(또는 `service.ServerGroups`)의 generated `Create/Get/List/Delete`는 Gophercloud `v2.15.0`의 [servergroups 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/servergroups/requests.go)을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError{Resource: "servergroups"}` 문맥만 더하며 다른 status는 native `gophercloud.ErrUnexpectedResponseCode`로 남습니다. ID는 escape 없이 경로에 이어 붙입니다.

| 메서드 | 요청 | 성공 status | 반환 |
|---|---|---|---|
| `Create(ctx, opts, options...)` | `POST os-server-groups`, `{"server_group": {...}}` | 200 | 응답의 `server_group` |
| `Get(ctx, id)` | `GET os-server-groups/{id}` | 200 | 응답의 `server_group` |
| `List(ctx, options...)` | `GET os-server-groups` | native pager 200, 204, 300 | `server_groups` 행 |
| `Delete(ctx, id)` | `DELETE os-server-groups/{id}` | 202, 204 | error |

`CreateOpts.Name`은 필수이며 비어 있으면 HTTP 전에 오류입니다. `Policies`(2.64 이전 목록), `Policy`(2.64 이후 단일 값), `Rules.MaxServerPerHost`는 비어 있으면 생략하고, 어느 형식을 쓸지는 호출자가 선택한 microversion에 맞춥니다. `WithCreateField` 확장 필드는 `server_group` envelope 안에 들어가고 `name` 같은 기존 key와 겹치거나 nil 옵션이면 HTTP 전에 거부됩니다. 응답의 `policy`가 null이면 `Policy`는 nil입니다.

`List`는 `ListOpts`의 `all_projects`, `limit`, `offset`과 `WithListQuery` 확장 query를 보냅니다. 응답은 단일 페이지로 다루므로 `server_groups_links`가 있어도 다음 페이지를 요청하지 않습니다. 더 읽으려면 `offset`을 바꿔 다시 호출합니다. 빈 목록은 아무 값도 내보내지 않고, 본문 없는 204는 native pager가 JSON을 먼저 읽기 때문에 `io.EOF` 오류 하나로 끝납니다. 목록 오류는 operation 문맥 없이 native 오류 그대로 전달됩니다.
