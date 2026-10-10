# Compute extension native 호출

`extensions.New(client)`(또는 `service.Extensions`)의 generated 호출은 Gophercloud `v2.15.0`의 [compute extensions](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/compute/v2/extensions/delegate.go)를 바꾸지 않고 호출합니다. SDK는 `Get` 오류에 `resource.OperationError{Resource: "extensions"}` 문맥만 더합니다.

| 메서드 | 요청 | 성공 status | 반환 |
|---|---|---|---|
| `List(ctx)` | `GET extensions` | native pager 200, 204, 300 | `extensions` 행 |
| `Get(ctx, alias)` | `GET extensions/{alias}` | 200 | 응답의 `extension` |
| `ActionURL(ctx, id)` | 요청 없음 | 해당 없음 | `servers/{id}/action` URL 문자열 |

`List`는 단일 페이지로 다루므로 `extensions_links`를 따르지 않습니다. 빈 목록은 아무 값도 내보내지 않고, 본문 없는 204는 `io.EOF` 오류 하나로 끝나며, 목록 오류는 operation 문맥 없이 전달됩니다. `Updated`는 서버가 보낸 문자열 그대로이며 시각으로 해석하지 않습니다. `ActionURL`은 client의 ResourceBase로 URL만 만들고 HTTP 요청이나 context 확인을 하지 않습니다. extension API는 Nova 2.1 이후 고정된 목록만 돌려주므로 기능 감지는 microversion으로 하는 편이 정확합니다.
