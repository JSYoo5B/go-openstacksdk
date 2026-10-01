# 목록의 페이지와 전체 읽기 범위

`Resources.List`와 부모 scope의 `List`는 같은 `resource.With...` 옵션으로 목록을 읽습니다. `List`는 Go iterator를 반환하고, `All`은 같은 정책으로 읽은 결과를 slice로 모읍니다. 옵션이나 builder interface를 애플리케이션에서 구현할 필요가 없습니다.

현재 native pager를 사용하는 generated Collection·scope 121개와 별도 구현한 Nova instance action, Swift container·object, Trove database·user 5개는 `IterateControlled`에 연결되어 있습니다. 이 연결은 공통 `Resources`·scope 경로에 적용합니다. 기존 native API의 공개 `List` 메서드와 서비스별 typed options는 그대로이며, 그 메서드에 새로운 `WithListMaxItems` 옵션을 추가한 것은 아닙니다. Senlin의 직접 구현된 REST 목록은 자체 typed controls도 제공하지만 페이지 해석 정책은 아래 native 목록과 구분합니다.

Neutron floating IP 이름 해석과 Manila access rule scope의 내부 목록 adapter는 이번 native 연결 범위에 포함하지 않습니다. 페이지 경계를 노출하지 않는 `Iterate` 전용 adapter도 공통 `WithMaxItems`로 반환 행을 제한할 수 있지만 `WithPaginated(false)`는 `resource.ErrUnsupported`입니다. 모든 내부 목록이나 native 공개 typed List의 per-call 제어까지 완료했다는 의미는 아닙니다.

## openstacksdk와 옵션 대응

Python의 기준은 [pinned Resource.list](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2155-L2355)입니다. 각 proxy가 실제로 허용하는 query와 부모 입력은 서비스마다 다릅니다.

| 의도 | openstacksdk Resource.list | Go 공통 목록 |
|---|---|---|
| 전체 읽기 범위 제한 | `max_items=20` | `resource.WithMaxItems(20)` |
| 첫 페이지만 읽기 | `paginated=False` | `resource.WithPaginated(false)` |
| 페이지를 계속 따라가기 | `paginated=True` 또는 기본값 | `resource.WithPaginated(true)` 또는 기본값 |
| 서버에 요청할 페이지 크기 | 지원하는 query의 `limit=100` | 지원하는 binding의 `resource.WithPageSize(100)` |
| 정확한 이름으로 거르기 | Resource와 query mapping에 따라 다름 | 이름 capability가 있는 binding의 `resource.WithName(name)` |
| 상태로 거르기 | Resource와 query mapping에 따라 다름 | 상태 capability가 있는 binding의 `resource.WithStatus(status)` |

`WithMaxItems`는 서버 응답에서 읽을 행 수이고 `WithPageSize`는 서버에 전달하는 페이지 크기입니다. native 경로는 `WithMaxItems`만으로 wire `limit`을 추가하지 않습니다. 이는 `max_items`에서 `limit` hint를 만드는 pinned Python Resource.list와 다른 Go 정책입니다. 명시한 `WithPageSize`는 기존 query builder를 거쳐 전달되며 서비스의 버전·query 지원 조건을 그대로 따릅니다. Trove database·user scope는 초기 query와 `WithPageSize`를 지원하지 않아 HTTP 전에 `resource.ErrUnsupported`를 반환하지만, 두 로컬 제어 옵션은 사용할 수 있습니다.

Pinned Swift [containers](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L143-L152)와 [objects](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/object_store/v1/_proxy.py#L249-L270) proxy는 `paginated=True`를 명시합니다. 이 proxy에 `paginated=False`를 추가하면 중복 인자가 되므로, Swift 공통 Go 목록의 첫 페이지 옵션은 해당 Python proxy에 대한 명시적 확장입니다.

## 행 수, 필터와 실행 시점

기본 `WithMaxItems(0)`은 무제한입니다. 양수는 응답 행 수를 제한하고, 음수는 iterator를 순회할 때 첫 HTTP 요청 전에 오류를 반환합니다. 옵션은 순서대로 적용하므로 같은 제어를 여러 번 전달하면 마지막 값이 적용됩니다. 예를 들어 `WithMaxItems(-1), WithMaxItems(20)`은 최종값 20으로 실행합니다. `List` 호출은 옵션 slice를 복사하며, 같은 iterator를 다시 순회하면 행 수를 0부터 세고 새 요청을 시작합니다. `All`은 바로 순회하므로 해당 오류를 반환값으로 받습니다.

행 수 제한은 응답에서 나온 행을 기준으로 공통의 정확한 이름·상태 필터와 Trove user scope의 host 필터보다 먼저 적용됩니다. 서버 query로 이미 제외된 행은 세지 않습니다. 첫 두 응답 행 중 하나만 이름이 일치하면 `WithMaxItems(2)` 결과는 한 개입니다. 일치하는 행이 세 번째에만 있으면 결과가 비어 있을 수 있습니다. Swift의 `subdir` 항목도 응답 행 하나로 셉니다. 일치하는 결과를 N개 얻는 제한으로 해석하지 않습니다.

이름은 대소문자까지 정확하게 비교하며 Nova의 regex query에는 literal 이름을 escape하여 전달합니다. 상태의 공통 로컬 비교는 대소문자를 구분하지 않습니다. 예를 들어 Octavia member의 상태는 `ProvisioningStatus`를 비교하며 generic `status` query를 보내지 않습니다. 이름이나 상태 capability가 없는 Nova instance action에 해당 필터를 전달하면 `resource.ErrUnsupported`입니다.

`break`는 다음 행과 다음 페이지 요청을 중단합니다. context 취소는 진행 중인 HTTP 요청과 순회에 전달되며 원래 오류 원인을 보존합니다. 목록은 상세 GET·HEAD나 다른 연산을 자동으로 실행하지 않습니다. 부모가 필요한 scope의 부모 해석은 scope를 만드는 시점에 수행하므로, 자식 목록의 lazy 실행과는 구분합니다. `resource.ID(...)` 부모는 사전 조회 없이 고정하고 `resource.Name(...)` 부모는 한 번 정확하게 찾습니다.

## 페이지 decode와 다음 링크

native pager의 `IsEmpty`와 `Extract`는 한 페이지 전체를 decode할 수 있습니다. 따라서 제한 이후에 위치한 잘못된 행도 현재 페이지의 decode 오류로 관찰할 수 있습니다. 행 수 제한은 페이지 내부의 malformed JSON이나 모델 오류를 숨기는 기능이 아닙니다. 전체 페이지를 성공적으로 해석한 다음 행을 세고, 제한에 도달하거나 첫 페이지 처리가 끝나면 다음 링크를 읽거나 요청하기 전에 멈춥니다.

다음 페이지가 필요한 경우 기존 native marker·linked-page 프로토콜과 공통 반복 링크 검사를 사용합니다. 이 경로에 origin·path 제한을 새로 추가하지 않았습니다. origin·경로 검증을 수행하는 Senlin REST 목록의 정책을 native 목록에도 적용한다고 해석하지 않습니다. `WithPaginated(false)`나 이미 충족된 cap은 사용하지 않을 다음 링크의 반복 여부를 검사하거나 그 URL을 요청하지 않습니다. 기본 무제한·다중 페이지 목록은 원래 continuation과 반복 링크 오류를 유지합니다.

## Nova 서버 목록

Python의 `conn.compute.servers(max_items=100, limit=50)`에 대응하는 읽기 범위와 페이지 크기를 지정하고, 이름·상태를 함께 비교합니다. 아래 이름 `worker.*`는 정규식이 아니라 literal 이름입니다.

```go
package examples

import (
	"context"
	"fmt"

	sdk "gophercloudsdk"
	"gophercloudsdk/resource"
)

func ListServers(ctx context.Context, conn *sdk.Connection) error {
	service, err := conn.ComputeV2(ctx)
	if err != nil {
		return err
	}
	for server, err := range service.Servers.Resources.List(ctx,
		resource.WithName("worker.*"),
		resource.WithStatus("ACTIVE"),
		resource.WithPageSize(50),
		resource.WithMaxItems(100),
	) {
		if err != nil {
			return err
		}
		fmt.Println(server.ID, server.Name, server.Status)
	}
	return nil
}
```

## Designate zone의 record set

Python의 `conn.dns.recordsets(zone, max_items=20, paginated=False)`와 같은 부모·첫 페이지 제어입니다. Go는 부모 zone ID를 scope에 고정하고 `All`로 결과를 모읍니다.

```go
package examples

import (
	"context"
	"fmt"

	sdk "gophercloudsdk"
	"gophercloudsdk/resource"
)

func ListRecordSets(ctx context.Context, conn *sdk.Connection, zoneID string) error {
	service, err := conn.DNS(ctx)
	if err != nil {
		return err
	}
	scope, err := service.RecordSets.InZone(ctx, resource.ID(zoneID))
	if err != nil {
		return err
	}
	records, err := scope.All(ctx,
		resource.WithMaxItems(20), resource.WithPaginated(false))
	if err != nil {
		return err
	}
	for _, record := range records {
		fmt.Println(record.ID, record.Name, record.Type, record.Records)
	}
	return nil
}
```

## Octavia pool의 member

Python의 `conn.load_balancer.members(pool, max_items=10, paginated=False)`에 대응합니다. Go의 공개 부모 메서드는 `Pools.Members`입니다. 첫 페이지에서 최대 10개 응답 행을 읽고 `ProvisioningStatus`가 ACTIVE인 행만 반환합니다.

```go
package examples

import (
	"context"
	"fmt"

	sdk "gophercloudsdk"
	"gophercloudsdk/resource"
)

func ListMembers(ctx context.Context, conn *sdk.Connection, poolID string) error {
	service, err := conn.LoadBalancer(ctx)
	if err != nil {
		return err
	}
	scope, err := service.Pools.Members(ctx, resource.ID(poolID))
	if err != nil {
		return err
	}
	for member, err := range scope.List(ctx,
		resource.WithMaxItems(10),
		resource.WithPaginated(false),
		resource.WithStatus("ACTIVE"),
	) {
		if err != nil {
			return err
		}
		fmt.Println(member.ID, member.Name, member.ProvisioningStatus)
	}
	return nil
}
```

서비스별 native 범위 예제는 [Nova action](../compute/v2/instanceactions/README.md), [Swift container](../objectstorage/v1/containers/README.md), [Swift object](../objectstorage/v1/objects/README.md), [Trove database](../db/v1/databases/README.md), [Trove user](../db/v1/users/README.md)에 있습니다. 공개 Collection·scope의 HTTP 연결은 [generated 목록 계약](../api/generated_list_controls_test.go)과 [5개 native 범위 계약](../api/native_scope_list_controls_test.go)에서 검증합니다. 공통 옵션·pager 동작은 [목록 제어 테스트](../resource/list_control_test.go)와 [stream 테스트](../resource/stream_test.go)를 참고합니다.
