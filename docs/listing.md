# 목록의 페이지와 전체 읽기 범위

`Resources.List`와 부모 scope의 `List`는 같은 `resource.With...` 옵션으로 목록을 읽습니다. `List`는 Go iterator를 반환하고, `All`은 같은 정책으로 읽은 결과를 slice로 모읍니다. 옵션이나 builder interface를 애플리케이션에서 구현할 필요가 없습니다.

현재 native pager를 사용하는 generated Collection·scope 121개와 별도 구현한 Nova instance action, Swift container·object, Trove database·user 5개는 `IterateControlled`에 연결되어 있습니다. 이 연결은 공통 `Resources`·scope 경로에 적용합니다. 기존 native API의 공개 `List` 메서드와 서비스별 typed options는 그대로이며, 그 메서드에 새로운 `WithListMaxItems` 옵션을 추가한 것은 아닙니다. Senlin의 직접 구현된 REST 목록은 자체 typed controls도 제공하지만 페이지 해석 정책은 아래 native 목록과 구분합니다.

별도로 [Manila access rule scope](../sharedfilesystems/v2/shareaccessrules/README.md)는 typed `WithListMaxItems`와 `WithListPaginated`를 제공합니다. 현대 endpoint는 하나의 collection을 반환하므로 두 pagination 값 모두 한 번만 GET하며 cap을 wire limit으로 보내거나 next link를 따라가지 않습니다. 내부 scope adapter도 같은 제어 iterator를 사용합니다. Neutron floating IP 이름 해석의 private 목록은 별도입니다. 페이지 경계를 노출하지 않는 `Iterate` 전용 adapter도 공통 `WithMaxItems`로 반환 행을 제한할 수 있지만 `WithPaginated(false)`는 `resource.ErrUnsupported`입니다. 모든 내부 목록이나 native 공개 typed List의 per-call 제어까지 완료했다는 의미는 아닙니다.

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

## 명시적인 native Body 필터

Python의 `conn.network.qos_policies(rules=[])`와
`conn.network.address_groups(addresses=["192.0.2.0/24"])`는 선언된 Body 속성을 로컬에서
비교합니다. `conn.network.subnet_pools(prefixes=["192.0.2.0/24"])`와
`conn.network.networks(subnet_ids=["subnet-a", "subnet-b"])`도 같은 방식입니다.
Go의 ordinary `Resources.List/All`은 다음처럼 SDK 소유 옵션을 사용합니다.
`conn.Network(ctx).API`에서 얻는 versioned 서비스와 직접 만든 `network/v2.Service` 모두
같은 동작입니다. caller가 builder나 predicate를 구현하지 않습니다.

```go
package examples

import (
    "context"

    networkv2 "gophercloudsdk/network/v2"
    "gophercloudsdk/resource"
)

func ListLocalMatches(ctx context.Context, service *networkv2.Service) error {
    policies, err := service.QoSPolicies.Resources.All(ctx,
        resource.WithBodyFilter("rules", []map[string]any{}),
        resource.WithMaxItems(100))
    if err != nil { return err }
    groups, err := service.SecurityAddressGroups.Resources.All(ctx,
        resource.WithBodyFilters(map[string]any{
            "addresses": []string{"192.0.2.0/24"},
        }), resource.WithPaginated(false))
    if err != nil { return err }
    _, _ = policies, groups
    return nil
}

func ListSubnetMatches(ctx context.Context, service *networkv2.Service) error {
    for value, err := range service.Networks.Resources.List(ctx,
        resource.WithBodyFilter("subnet_ids", []string{"subnet-a", "subnet-b"}),
        resource.WithStatus("ACTIVE")) {
        if err != nil { return err }
        _ = value
    }
    _, err := service.SubnetPools.Resources.All(ctx,
        resource.WithBodyFilter("prefixes", []string{"192.0.2.0/24"}),
        resource.WithMaxItems(100))
    return err
}
```

| binding | Go canonical 필드 | SDK 별칭 | Python 로컬 필터 이름 |
|---|---|---|---|
| QoS Policy | `rules` | 없음 | `rules` |
| Address Group | `addresses` | 없음 | `addresses` |
| Subnet Pool | `prefixes` | 없음 | `prefixes` |
| Network | `subnets` | `subnet_ids` | `subnet_ids` |
| Subnet | 아래 9개 필드 | `prefix_length` → `prefixlen` | 아래 표 참고 |

처음 네 binding은 위의 한 응답 필드를 지원합니다. 상위 `network.Service.Networks`도 같은 Network
필터를 제공합니다. 지원이 없는 binding의 명시
옵션은 `ErrUnsupported`, 알 수 없는 필드·JSON으로 표현할 수 없는 값은 `ErrInvalidOption`을
첫 HTTP 전에 반환합니다. 오류는 iterator를 소비할 때 나타나며 malformed JSON·NaN 등
원래 인코딩 오류도 error chain에 보존합니다. 잘 구성된 JSON scalar를 임의 변환하지 않으므로
배열과 scalar는 일치하지 않습니다.

`WithBodyFilter`는 기존 조건에 추가하고 같은 canonical 필드는 마지막 옵션이 이깁니다.
`WithBodyFilters`는 전체 조건을 교체하며 nil/빈 map은 조건을 지웁니다. 명시적인 clear도
지원 binding을 요구합니다. `WithBodyFilter("rules", nil)`은 null을 비교하는 조건입니다.
SDK는 옵션 생성 시 값을 JSON으로 복사하고 순회별로 독립 복사하므로 caller map·slice·raw JSON
변경과 옵션 재사용이 진행 중인 조회를 바꾸지 않습니다. SDK가 등록한 alias가 있는 경우 bulk의
alias/canonical 중복은 결정적으로 거부합니다. Network의 `subnet_ids`와 `subnets`는 같은 필드이므로
개별 옵션을 순서대로 지정하면 마지막 조건만 적용하고, 같은 bulk map에 넣으면 HTTP 전에 실패합니다.

배열은 순서·길이·원소 전체가 같아야 합니다. `rules` 배열 안의 dict도 모든 key/value가 같아야
하므로 일부 rule 속성이나 일부 주소가 포함되는지를 검색하지 않습니다. null과 빈 배열은 다릅니다.
JSON bool과 number를 구별하며 Python의 `False == 0`까지 적용하지 않습니다. 일반 비교 엔진의
object 필터는 recursive subset이지만 배열 내부 object는 전체 equality이고, object 필터와
non-object 응답은 불일치로 처리합니다.
문자열 배열도 전체 값이 같아야 하며 subnet ID 순서를 바꾸거나 CIDR을 병합·정규화하지 않습니다.
Python Network의 `subnets`는 wire Body 이름이며 로컬 필터용 속성 이름은 `subnet_ids`입니다.
Go는 SDK가 명시 등록한 두 이름을 받아 같은 native `Subnets`를 비교합니다.

`WithMaxItems`는 로컬 필터 이전의 raw 행을 세므로 필터에서 제외된 행을 추가 페이지로 보충하지
않습니다. `WithName`·`WithStatus`와 AND로 조합하며 Network는 Status가 있지만 나머지 모델은
Status가 없어 typed `WithStatus`를 지원하지 않습니다. 첫 페이지·consumer break·후속 HTTP/decode/cycle/취소·native 전체 페이지
decode 정책을 유지합니다. Body 필터를 서버 query로 넣지 않으며, 명시 `WithQuery("rules", ...)`나
`WithQuery("addresses", ...)`·`WithQuery("prefixes", ...)`·`WithQuery("subnet_ids", ...)`·
`WithQuery("subnets", ...)`는 기존 wire 확장으로 독립 전달하며 별칭을 자동 변환하지 않습니다. native typed List와
`FindIdentity`는 이 공통 ListOption을 받지 않습니다.

처음 네 binding의 비교 값은 native typed 모델의 field projection입니다. 누락/null `Rules`·`Addresses`는 모두
nil이 되지만 빈 배열은 구별됩니다. `Prefixes`·`Subnets`도 같은 presence 한계를 갖습니다.
Subnet Pool의 native prefix length와 timestamp, Network의 native timestamp decoder 등 다른
필드의 decode 오류도 로컬 필터 전에 발생하며, 현재 응답 페이지 안에서 cap 뒤에 위치한
행의 decode 오류도 숨기지 않습니다. cap으로 방문하지 않은 다음 페이지는 검사하지 않습니다.
QoS `Rules`는 native `map[string]any`의 float64 decoding을
이미 거쳤으므로 비교 엔진이 원래 wire 정수의 정밀도를 복구하지 않습니다. unknown outer Body·
Python descriptor/default/alias/coercion과 자동 query/Body 분류 전체를 제공한 것으로 확대하지
않습니다. 다른 리소스의 필드도 별도 source 감사를 거쳐 연결합니다.

### Subnet의 원본 응답 필터

Python `conn.network.subnets(prefix_length=24, dns_nameservers=["192.0.2.53"])`에 대응하는
Go 호출은 [Subnet README](../network/v2/subnets/README.md)에 있습니다. 다음 9개는 Python의
query mapping에 포함되지 않은 Body 속성이며, Go는 명시적인 `WithBodyFilter(s)`로 선택합니다.

| Python 로컬 속성 | Go canonical 필드 | 비교하는 응답 값 |
|---|---|---|
| `allocation_pools` | `allocation_pools` | 배열 전체, 내부 추가 필드·null 원소 포함 |
| `dns_nameservers` | `dns_nameservers` | 배열 전체, 순서·null 원소 포함 |
| `host_routes` | `host_routes` | 배열 전체, 내부 추가 필드·null 원소 포함 |
| `service_types` | `service_types` | 배열 전체 |
| `created_at` | `created_at` | 서비스가 반환한 timestamp 문자열 |
| `updated_at` | `updated_at` | 서비스가 반환한 timestamp 문자열 |
| `prefix_length` | `prefixlen` | 타입이 선언되지 않은 원본 JSON, `prefix_length` 별칭 지원 |
| `tenant_id` | `tenant_id` | `project_id`와 독립적인 원본 값 |
| `revision_number` | `revision_number` | 정확한 정수 |

Subnet은 native typed 값과 같은 페이지의 원본 행을 함께 사용합니다. 추가 HTTP 요청 없이
native 모델이 생략한 `prefixlen`·내부 추가 key와 정수 정밀도를 보존합니다. 반환값은 기존
typed Subnet이며, 로컬 비교에만 원본 필드를 사용합니다. 원본 timestamp의 표기를 정규화하지
않고, 누락/null은 null 필터와 일치하며 빈 문자열·배열·객체는 구별합니다. 중첩 배열의 객체는
전체 equality이므로 부분 allocation pool·route를 지정하면 추가 key가 있는 원본과 일치하지 않습니다.

`prefixlen`의 Python Body descriptor에는 타입이 없습니다. Go도 응답 문자열·number·bool·
객체·배열을 원형대로 비교합니다. 숫자 `24`·`24.0`·`2.4e1`은 같은 정확한 decimal 값이며
문자열 `"24"`는 숫자 `24`와 다릅니다. 큰 숫자도 float64로 변환하지 않습니다. 소수와 JSON
bool도 정상 값으로 비교하지만 Python의 bool/int 동등성은 적용하지 않습니다.
`revision_number`는 native int decoder가 먼저 처리하므로 문자열·소수 응답은 그 단계에서 실패합니다.
다른 필드에도 Python scalar→list 변환을 추가하지 않습니다.

native extractor가 현재 페이지 전체를 먼저 검사합니다. DNS 배열·route/pool의 알려진 필드·
timestamp·revision의 decode 오류는 cap 뒤의 행이어도 유지됩니다. native 모델에 없는
`prefixlen`에 임의 타입 검증을 추가하지 않습니다. cap 뒤의 미소비 값과
필터 없는 ordinary List에 새 검증을 추가하지 않습니다. clear 옵션도 원래 typed 목록 경로로
돌아갑니다. Subnet의 typed `WithStatus`는 지원하지 않고, `WithQuery("tenant_id", ...)`와
`WithQuery("revision_number", ...)`도 명시 wire query로 독립 전달합니다.

고정 Python [qos_policies](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5177)·[address_groups](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L392)의 추가 query는
[Resource.list의 Body 분류와 비교](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2217)에 연결됩니다.
Network [subnet_ids](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/network.py#L119)·Subnet Pool [prefixes](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/subnet_pool.py#L91)도 non-query Body 속성입니다.
Trunk `sub_ports`는 query mapping에 있어 Python도 wire query로 전달하며, 이 SDK에서 로컬 필터로 등록하지 않습니다.
실제 동작은 [native HTTP 계약](../api/network_body_filters_test.go), [Network·Subnet Pool HTTP 계약](../api/network_subnet_body_filters_test.go), [공통 옵션·소비 계약](../resource/body_filters_test.go),
[JSON 비교 엔진](../internal/jsonfilter/filter_test.go)에서 검증합니다.
Subnet 원본 행·native decoder·cap/continuation 계약은 [Subnet HTTP 테스트](../api/network_subnet_raw_body_filters_test.go),
공통 행 소유권과 정수 변환은 [BodyRecord 테스트](../resource/body_record_test.go)와
[정확한 정수 테스트](../internal/jsonfilter/integer_test.go)에서 검증합니다.
