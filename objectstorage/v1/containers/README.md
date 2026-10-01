# Swift container 목록

`conn.ObjectStorage(ctx)`가 반환한 서비스에서 `service.Containers.Resources`를 사용합니다. container는 이름이 식별자이며 `resource.ID(name)`은 원래 이름을 받습니다. SDK가 URL을 escape하므로 호출자가 미리 인코딩하지 않습니다. `resource.Name(name)` 조회는 prefix 목록에서 정확한 이름을 비교합니다.

| 작업 | openstacksdk | Go |
|---|---|---|
| 전체 목록 | `conn.object_store.containers()` | `service.Containers.Resources.List(ctx)` |
| 읽을 행 수 제한 | `containers(max_items=20)` | 목록에 `resource.WithMaxItems(20)` |
| 첫 페이지만 읽기 | pinned proxy는 `paginated=True`를 고정 | 목록에 `resource.WithPaginated(false)` |
| metadata 조회 | `get_container_metadata(name)` | `service.Containers.Resources.Get(ctx, name)` |

`WithMaxItems(0)`은 무제한이며 음수는 iterator 순회 시 HTTP 전에 오류를 반환합니다. 같은 옵션은 마지막 값이 적용됩니다. 제한은 서버 응답 행을 세므로 `WithName`의 정확한 로컬 비교보다 먼저 적용합니다. prefix query 결과 중 이름이 다른 항목도 cap을 소비합니다. native 경로는 cap에서 `limit`을 추정하지 않으며, 아래 `WithPageSize(100)`만 서버 페이지 크기를 요청합니다.

```go
package examples

import (
	"context"
	"fmt"

	sdk "gophercloudsdk"
	"gophercloudsdk/resource"
)

func ListContainers(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ObjectStorage(ctx)
	if err != nil {
		return err
	}
	for container, err := range service.Containers.Resources.List(ctx,
		resource.WithPageSize(100),
		resource.WithMaxItems(20),
		resource.WithPaginated(false),
	) {
		if err != nil {
			return err
		}
		fmt.Println(container.Name, container.Count, container.Bytes)
	}
	return nil
}
```

목록의 `ContainerResource`는 container 이름·object 수·bytes를 보존합니다. `Details`, 사용자 `Metadata`, 전체 `Header`는 nil이며, 목록에서 자동 HEAD를 실행하지 않습니다. `Resources.Get`은 HEAD로 이 값을 채웁니다. 상태 필드가 없으므로 `WithStatus`와 상태 `Wait`는 `resource.ErrUnsupported`입니다.

`break`와 context 취소는 추가 페이지 요청을 중단합니다. native Swift marker pagination과 반복 링크 검사를 유지하며, 현재 페이지의 전체 decode 오류는 cap 이후 행에서도 관찰할 수 있습니다. 첫 페이지 옵션은 다음 marker를 처리하기 전에 멈춥니다. 기존 공개 `service.Containers.List(ctx, containers.WithListOptions(...))`와 typed query 입력은 유지됩니다. 이 문서의 두 로컬 제어는 `Resources.List/All` 옵션입니다.

Python proxy의 첫 페이지 차이와 전체 공통 정책은 [목록 가이드](../../../docs/listing.md), object scope는 [Swift object 사용법](../objects/README.md), HTTP 증거는 [native 범위 목록 계약](../../../api/native_scope_list_controls_test.go)와 [Swift 목록 계약](../../../api/swift_listing_contracts_test.go)을 참고합니다.
