# 서버 태그

`Tags.InServer`로 서버를 고정하면 각 요청에 서버 ID를 반복하거나 직접 builder를 구현할 필요가 없습니다. `resource.Name`은 공통 서버 이름 검색으로 정확히 한 번 해석하며, 같은 이름이 여러 개면 `resource.ErrAmbiguous`를 반환합니다. `resource.ID`는 서버 조회 요청 없이 범위를 만듭니다. 이후 서버가 삭제된 경우 실제 태그 요청의 404로 확인합니다.

```go
package main

import (
    "context"
    "fmt"

    sdk "github.com/JSYoo5B/go-openstacksdk"
    "github.com/JSYoo5B/go-openstacksdk/compute/v2/tags"
    "github.com/JSYoo5B/go-openstacksdk/resource"
)

func main() {
    ctx := context.Background()
    conn, err := sdk.Connect(ctx,
        sdk.WithMicroversionRange(sdk.Compute, "2.26", "2.100"),
    )
    if err != nil { panic(err) }
    service, err := conn.ComputeV2(ctx)
    if err != nil { panic(err) }
    serverTags, err := service.Tags.InServer(ctx, resource.Name("web.prod"))
    if err != nil { panic(err) }

    if err := serverTags.Add(ctx, "managed"); err != nil { panic(err) }
    present, err := serverTags.Check(ctx, "managed")
    if err != nil { panic(err) }
    fmt.Println(serverTags.ServerID(), present)
    values, err := serverTags.List(ctx)
    if err != nil { panic(err) }
    fmt.Println(values)
    values, err = serverTags.Replace(ctx, []string{"managed", "role=db"})
    if err != nil { panic(err) }
    fmt.Println(values)
    if err := serverTags.Remove(ctx, "managed", tags.WithMissingError()); err != nil {
        panic(err)
    }
    if err := serverTags.RemoveAll(ctx); err != nil { panic(err) }
}
```

태그 API는 [Nova microversion 2.26부터](https://docs.openstack.org/api-ref/compute/#server-tags-servers-tags) 지원됩니다. 범위를 만들거나 요청할 때 선택된 Compute microversion을 확인하며, 버전 미설정이나 2.25 이하에서는 HTTP 요청 전에 `resource.ErrUnsupported`를 반환합니다. 클라이언트의 버전을 자동으로 올리지 않습니다. 위 예제처럼 명시적으로 [범위 협상](../../../docs/microversions.md)을 사용하거나 `sdk.WithMicroversion(sdk.Compute, "2.26")`으로 버전을 고정합니다. 범위 상한은 애플리케이션이 검증한 버전으로 지정합니다.

| openstacksdk | Go 범위 객체 |
|---|---|
| `conn.compute.add_tag_to_server(server, tag)` / `server.add_tag(conn.compute, tag)` | `serverTags.Add(ctx, tag)` |
| `server.check_tag(conn.compute, tag)` | `serverTags.Check(ctx, tag, tags.WithMissingError())` |
| `server.fetch_tags(conn.compute)` 후 `server.tags` | `serverTags.List(ctx)`의 `[]string` |
| `server.set_tags(conn.compute, values)` | `serverTags.Replace(ctx, values)`의 `[]string` |
| `conn.compute.remove_tag_from_server(server, tag)` / `server.remove_tag(conn.compute, tag)` | `serverTags.Remove(ctx, tag, tags.WithMissingError())` |
| `conn.compute.remove_tags_from_server(server)` / `server.remove_all_tags(conn.compute)` | `serverTags.RemoveAll(ctx, tags.WithMissingError())` |

`Add`는 새 태그의 201과 이미 존재하는 태그의 204를 모두 성공으로 처리합니다. `Replace`는 전달한 전체 목록으로 교체하며 `nil`과 빈 slice 모두 `{"tags":[]}`를 전송해 모든 태그를 지웁니다. `List`와 `Replace`의 빈 결과는 비nil 빈 slice입니다. 반환 목록은 Nova 응답 순서를 유지하며 입력 순서에 맞춰 다시 정렬하지 않습니다.

`Check`는 기본적으로 404를 `false, nil`로, `Remove`와 `RemoveAll`은 404를 성공으로 처리합니다. Python TagMixin처럼 미존재를 오류로 처리하려면 `tags.WithMissingError()` 또는 `tags.WithIgnoreMissing(false)`를 사용합니다. 여러 옵션은 마지막 값이 적용됩니다. 엄격한 404는 `errors.Is(err, resource.ErrNotFound)`와 `gophercloud.ResponseCodeIs(err, 404)` 모두로 확인할 수 있으며, 403 등 다른 HTTP 실패는 항상 보존합니다. Nova는 태그 미존재와 서버 미존재를 모두 404로 표현하므로 `Check`의 기본 정책도 두 경우를 구분하지 않습니다.

태그 문자열은 1~60 Unicode 문자이며 `/`와 쉼표를 포함할 수 없습니다. 전체 교체 목록은 최대 50개입니다. 공백, Unicode, `?`, `#`, `%` 등 허용되는 값은 SDK가 URL 경로에 맞게 escape합니다. 단일 태그 추가 후 서버의 전체 개수 제한, 권한과 상태 제한은 Nova가 검증합니다. `Replace`의 선택 인자는 기존 `ReplaceAllOption`을 사용합니다. concrete `WithReplaceAllOptions`가 목록을 바꾼 경우 최종 목록을 검증하며, `WithReplaceAllField("tags", ...)` 덮어쓰기와 query/header/argument 확장은 요청 전에 거부합니다. 다른 body 확장은 SDK가 그대로 전송하지만 표준 Nova는 정의되지 않은 필드를 거부할 수 있습니다.

태그 목록 자체에는 필터 query가 없습니다. 서버 목록의 `tags`, `tags-any`, `not-tags`, `not-tags-any` 필터는 `service.Servers.All(ctx, resource.WithQuery("tags", "managed,role=db"))`처럼 서버 API에 전달합니다. Python의 `any_tags` 등 keyword 별칭은 위 HTTP query 이름으로 지정합니다. 서버 목록도 microversion 2.26 이상이 필요합니다.

범위 객체는 태그를 캐시하거나 기존 `Server` 모델의 `Tags` 필드를 수정하지 않습니다. Python Resource의 변경 추적·dirty state·`commit` 동작은 제공하지 않습니다. 변경 후 현재 태그를 다시 확인하려면 `List`를 호출합니다. 범위 객체는 호출마다 context 취소를 전달하며 공유 클라이언트 설정은 동시 요청 중 변경하지 않습니다. 서버 ID를 직접 전달하는 기존 `Tags.Add/Check/List/ReplaceAll/Delete/DeleteAll` API도 계속 사용할 수 있으며, 위 사전 검증과 범위 정책은 `InServer`로 얻은 객체에 적용됩니다.
