# Glance v2 native task 호출

`service.Tasks`(`image/v2/tasks`)의 generated `Create`·`Get`·`List`는 Gophercloud `v2.15.0`의 [tasks 요청](https://github.com/gophercloud/gophercloud/blob/v2.15.0/openstack/image/v2/tasks/requests.go)을 바꾸지 않고 호출합니다. SDK는 오류에 `resource.OperationError{Resource: "tasks"}` 문맥을 더하고 `WithCreateField` 확장 필드를 합칠 뿐입니다. 목록 stream의 page 오류에는 operation 문맥이 붙지 않습니다. 경로는 client `ResourceBase`(예: `.../v2/`) 아래에 붙습니다.

Glance 기본 정책(`tasks_api_access`)에서 task API는 관리자 호출입니다. 일반 사용자의 이미지 가져오기는 [image import 호출](imageimport/README.md)을 사용합니다.

| 메서드 | 요청 | 성공 status |
|---|---|---|
| `Create(ctx, opts, options...)` | `POST tasks`, `{"type": ..., "input": {...}}` | 201 |
| `Get(ctx, taskID)` | `GET tasks/{taskID}` | 200 |
| `List(ctx, options...)` | `GET tasks?limit=...&marker=...&sort_key=...&sort_dir=...&status=...` | native pager 200, 204, 300 |

## 입력 규칙

`CreateOpts.Type`은 필수라 비어 있으면 HTTP 전에 오류입니다. 본문에는 envelope가 없고 `Input`에 omitempty가 없어 nil이면 `"input": null`을 보냅니다. `WithCreateField` 확장은 최상위에 붙으며 `type`·`input`과 겹치면 HTTP 전에 거부합니다. nil 옵션도 HTTP 전에 오류입니다.

`ListOpts`의 `ID`와 `Type`은 query tag 없이 json tag만 있어서 native query에서 빠집니다. type으로 거르려면 `WithListQuery("type", ...)`를 더합니다. `Limit`·`Marker`·`SortKey`·`SortDir`·`Status`는 값이 있을 때만 보냅니다.

## decode와 paging

`Create`·`Get` 응답은 envelope 없는 task 본문입니다. `{}`이면 빈 task이고, 본문이 JSON null이면 오류 없이 nil을 돌려줍니다. `created_at`·`updated_at`·`expires_at`은 Go 기본 RFC 3339 해석이라 시간대가 없으면 decode 오류이고 null은 0 시각입니다. `input`·`result`는 `map[string]any`로 읽습니다.

`List`는 `tasks` 배열을 읽고 본문의 `next` 문자열로 다음 페이지를 찾습니다. `next`의 host는 버리고 path와 query만 client 주소의 version 앞부분에 붙이므로, `/v2/tasks?marker=...`는 `.../v2/tasks?marker=...`가 됩니다. `next`가 비어 있으면 목록이 끝나고, `first`·`schema` 같은 다른 링크는 쓰지 않습니다. 빈 `tasks` 배열은 아무 값도 내지 않고, 본문 없는 204는 `io.EOF` 오류입니다.

이 문서는 native 호출만 다룹니다. task 완료 대기는 같은 package의 SDK 소유 `WaitForTask`·`WaitForTaskState`가 맡습니다.
