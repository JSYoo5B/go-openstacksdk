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
| Subnet Pool | 아래 10개 필드 | prefix length의 Python 이름 3개 | 아래 설명 참고 |
| Network | 아래 14개 필드 | `subnet_ids`·`is_vlan_qinq`·`is_vlan_transparent` | 아래 설명 참고 |
| Router | 아래 10개 필드 | `revision_number` → `revision` | 아래 설명 참고 |
| Security Group | `created_at`·`updated_at`·`security_group_rules` | 없음 | canonical 이름과 같음 |
| Trunk | `id`·`tenant_id` | 없음 | canonical 이름과 같음 |
| Subnet | 아래 9개 필드 | `prefix_length` → `prefixlen` | 아래 표 참고 |

QoS Policy·Address Group·Subnet Pool·Network·Router·Security Group·Trunk·Subnet은 아래에 설명한 추가 응답 필드도 지원합니다.
상위 `network.Service.Networks`도 같은 Network
필터를 제공합니다. 지원이 없는 binding의 명시
옵션은 `ErrUnsupported`, 알 수 없는 필드·JSON으로 표현할 수 없는 값은 `ErrInvalidOption`을
첫 HTTP 전에 반환합니다. 오류는 iterator를 소비할 때 나타나며 malformed JSON·NaN 등
원래 인코딩 오류도 error chain에 보존합니다. 일반 JSON 필드는 scalar를 임의 변환하지 않으므로
배열과 scalar는 일치하지 않습니다. 감사한 boolean·정수 응답 descriptor의 변환은 아래에 별도로 설명합니다.

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
Go는 명시 Body 옵션에서 두 이름을 받아 같은 원문 `subnets`를 비교합니다.

`WithMaxItems`는 로컬 필터 이전의 raw 행을 세므로 필터에서 제외된 행을 추가 페이지로 보충하지
않습니다. `WithName`·`WithStatus`와 AND로 조합하며 Network는 Status가 있지만 나머지 모델은
Status가 없어 typed `WithStatus`를 지원하지 않습니다. 첫 페이지·consumer break·후속 HTTP/decode/cycle/취소·native 전체 페이지
decode 정책을 유지합니다. Body 필터를 서버 query로 넣지 않으며, 명시 `WithQuery("rules", ...)`나
`WithQuery("addresses", ...)`·`WithQuery("prefixes", ...)`·`WithQuery("subnet_ids", ...)`·
`WithQuery("subnets", ...)`는 기존 wire 확장으로 독립 전달하며 별칭을 자동 변환하지 않습니다. native typed List와
`FindIdentity`는 이 공통 ListOption을 받지 않습니다.

Network도 QoS·Address Group·Subnet Pool처럼 원문 행의 필드를 비교합니다.
`subnets`의 누락/null은 null로 비교하고 빈 배열은 구별합니다. 배열의 null 요소도 원문으로
비교하지만 반환 native `Subnets`의 해당 요소는 빈 문자열입니다. 기존 typed projection을
사용하던 명시 `subnets/subnet_ids` 조건도 이 원문 선택 정책을 따릅니다.
Subnet Pool의 native prefix length와 timestamp, Network의 native timestamp decoder 등 다른
필드의 decode 오류도 로컬 필터 전에 발생하며, 현재 응답 페이지 안에서 cap 뒤에 위치한
행의 decode 오류도 숨기지 않습니다. cap으로 방문하지 않은 다음 페이지는 검사하지 않습니다.
QoS `Rules`는 원문 JSON의 숫자로 비교하지만 반환 `map[string]any`는 native float64 decoding을
유지합니다. rule 배열 안의 null map·null 필드는 기존 native map도 보존하던 값이며 추가 개선으로
주장하지 않습니다. 선택한 원문 필드의 비교가 전체 unknown Body·Python descriptor/default/alias/coercion을
제공하지는 않습니다. 다른 리소스의 필드도 별도 source 감사를 거쳐 연결합니다.

### Network의 속성 이름 분류

`Networks.Resources.List/All`과 상위 `Network(ctx).Networks`는 Python
`conn.network.networks(**query)`의 query 23개와 로컬 Body 14개를 분류합니다. query는
wire 별칭을 포함한 35개 이름을 받고 `name`·`status`·`id`는 서버 조건입니다. 기존
`WithName`은 이름 hint와 정확한 로컬 이름 비교를, `WithStatus`는 status hint와 대소문자를
무시한 로컬 비교를 유지합니다. raw `WithQuery("status", ...)`만 지정하면 로컬 상태 비교는
추가하지 않습니다. 두 목록 경로 모두 wire status를 보존합니다.
`WithStatus`로 로컬 비교를 활성화한 뒤 raw status를 덮어쓰면 마지막 wire 값이 로컬 비교의
기준도 됩니다. 기존 옵션 순서 정책을 유지합니다.

| 로컬 속성 | 응답 선택 정책 |
|---|---|
| `is_default`·`is_vlan_qinq`·`is_vlan_transparent`·`pvlan` | missing/null 보존, boolean truthiness |
| `mtu`·`revision_number` | 정확한 signed 정수 |
| `availability_zone_hints`·`availability_zones`·`created_at`·`updated_at`·`dns_domain`·`qos_policy_id`·`segments`·`subnet_ids` | 원문 JSON |

semantic 이름 `subnet_ids`·`is_vlan_qinq`·`is_vlan_transparent`는 각각 원문
`subnets`·`vlan_qinq`·`vlan_transparent`에 연결합니다. 명시 Body 옵션은 세 쌍의 이름을 모두
받지만 raw 이름은 semantic 옵션에서 알 수 없는 이름입니다. `tenant_id`도 이 Python 선언의
semantic 조건이 아니며 `project_id`로 변환하지 않습니다. raw query는 독립적으로 전달합니다.
alias 우선·bulk 교체·snapshot·reserved controls·최종값 검증·target 충돌은 공통 옵션 정책을 따릅니다.

네 boolean 응답의 null은 false로 채우지 않습니다. 숫자 0·빈 문자열·빈 배열·빈 object는 false,
그 외 nonzero/nonempty 값은 true이고 문자열 `"false"`도 true입니다. 숫자는 float64 변환 없이
정확한 zero/nonzero로 비교하므로 극단적으로 작은 지수가 Python JSON float에서 0으로
underflow되는 경우와 다릅니다. caller 조건은 변환하지 않으며 bool/number도 구별합니다.
두 정수는 정확한 integral number와 signed decimal string 응답을 변환하고 fractional·bool·
배열·object는 거부합니다. native revision number의 int 디코드는 먼저 적용합니다. Python의
bool-as-int·float 절삭·digit-only string 및 default 0 처리를 모두 재현하지 않습니다.
나머지 필드는 원문 값이며 Python list descriptor의 scalar wrapping도 적용하지 않습니다.

native 모델의 알려진 필드는 cap·로컬 비교보다 먼저 페이지 전체를 디코드합니다. query-only
null 행은 native zero model일 수 있고, 선택한 Body 조건이 소비하는 null 행은 오류입니다.
cap·break로 소비하지 않는 행의 로컬 변환이나 다음 페이지를 검사하지 않습니다. native
`networks_links` continuation·client/provider·timestamp decoding과 반환 모델은 유지합니다.
native typed List·FindIdentity·Get/Delete는 semantic 옵션과 독립적입니다. SDK가 제공하는 새
`ResourceAdapter()`를 상위 facade에 조립하며 metadata map은 각각 복사합니다. 상위 facade의
singular `network` 오류 이름과 exact `ERROR` waiter는 기존 정책입니다.

[Network Python/Go 예제](../network/v2/networks/listing/README.md),
[HTTP 7개 그룹](../api/network_list_filters_test.go),
[Connection·adapter·waiter 3개 그룹](../connection_network_filters_test.go),
[boolean 응답 변환](../internal/jsonfilter/boolean_test.go)에서 이 계약을 검증합니다.
[AST manifest](../api/openstacksdk/resources/network/v2/network.json)는 source SHA 7개·AST 43개를,
[생성기 검증](../internal/cmd/sdkgen/network_filters_test.go)은 native 함수 선언 28개와
NoZ timestamp 상수 1개를 확인합니다. Resource/cache·revision if-match·Proxy lifecycle 전체는
별도 구현 대상입니다.

### Router의 속성 이름 분류

`Routers.Resources.List/All`은 Python `conn.network.routers(**query)`의 query 18개와
로컬 Body 10개를 분류합니다. `is_admin_state_up`→`admin_state_up`, `is_distributed`→`distributed`,
`is_ha`→`ha`와 태그 별칭을 포함한 24개 query 이름을 받습니다. `name`·`status`·`id`는
서버 조건입니다. `WithName`은 정확한 로컬 이름 비교를, `WithStatus`는 대소문자를 무시한
로컬 상태 비교를 별도로 활성화합니다. raw status만 지정하면 로컬 조건을 추가하지 않습니다.
두 목록 경로 모두 wire status를 보존하며 `WithStatus`가 활성화되어 있으면 뒤의 raw status
옵션이 바꾼 최종 wire 값이 로컬 비교 기준이 됩니다.

| 로컬 속성 | 응답 선택 정책 |
|---|---|
| `enable_ndp_proxy` | missing/null 보존, boolean truthiness |
| `evpn_vni`·`revision_number` | 정확한 signed 정수; `revision_number`는 raw `revision` 선택 |
| `availability_zone_hints`·`availability_zones`·`created_at`·`updated_at`·`external_gateway_info`·`routes`·`tenant_id` | 원문 JSON |

Python Router는 상속한 revision descriptor를 덮어씁니다. 따라서 semantic `revision_number`와
명시 Body의 `revision`·`revision_number`는 원문 `revision`을 비교하며 native Router의
`RevisionNumber`가 읽는 응답 `revision_number`로 fallback하지 않습니다. raw 이름 `revision`은
semantic 옵션에서 알 수 없는 이름으로 버리지만 명시 Body와 raw query는 각각 받습니다.
`tenant_id`는 로컬 조건이고 `project_id`는 query입니다. project descriptor의 response alias로
tenant query 별칭을 추가하지 않습니다. canonical 우선·bulk 교체·clear·snapshot·최종값 검증·
reserved controls·query/Body target 충돌은 공통 옵션 정책을 따릅니다.

응답 boolean은 missing/null을 false로 채우지 않습니다. 빈 값·0은 false, nonempty/nonzero는
true이며 문자열 `"false"`도 true입니다. caller 값은 변환하지 않습니다. 정확한 숫자 truthiness는
극단적인 지수의 Python float underflow와 다를 수 있습니다. 두 정수는 integral number와
signed decimal string을 변환하고 fractional·bool·배열·object는 거부합니다. Python의
bool-as-int·float 절삭·digit-only string 정책을 모두 재현하지 않습니다. 나머지 필드는 원문
JSON이며 Python list wrapping·dict coercion을 적용하지 않습니다.

gateway·routes의 unknown nested 필드와 큰 숫자·null 요소는 원문 비교에 남지만 반환값은
native Router입니다. native `[]Route`·`[]ExternalFixedIP`의 null 요소는 zero struct이며
`AvailabilityZoneHints`의 null 요소는 빈 문자열입니다. 알려진 nested 필드·`revision_number`와
timestamp 디코드는 cap·로컬 비교보다 먼저 전체 페이지에 적용됩니다. 두 timestamp의
NoZ→RFC3339 재시도는 native 정책이며 혼합 형식은 실패할 수 있습니다. query-only null 행은
native zero model일 수 있고 Body 조건이 소비하는 null 행은 오류입니다. cap·break 이후의
로컬 변환과 다음 페이지는 검사하지 않습니다. native `routers_links` continuation·client/provider와
typed List·FindIdentity·Get/Delete·interface 변경 API는 유지합니다.

`Network(ctx).API.Routers.Resources`와 `NetworkV2(ctx).Routers.Resources`는 캐시된 같은 client를
사용합니다. [Router Python/Go 예제](../network/v2/extensions/layer3/routers/listing/README.md),
[HTTP 7개 그룹](../api/router_list_filters_test.go),
[Connection 2개 그룹](../connection_router_filters_test.go)에서 이 계약을 검증합니다.
[AST manifest](../api/openstacksdk/resources/network/v2/router.json)는 source SHA 7개·AST 41개를,
[생성기 검증](../internal/cmd/sdkgen/router_filters_test.go)은 native 함수 23개와 resource path·
NoZ timestamp 상수 2개를 확인합니다. 전체 Python Resource/cache·revision if-match·Proxy lifecycle은
계속 구현할 대상입니다.

### Security Group의 속성 이름 분류

`SecurityGroups.Resources.List/All`은 Python `conn.network.security_groups(**query)`의
query 17개와 로컬 Body 3개를 분류합니다. `is_shared`→`shared`와 태그 별칭을 포함한 21개 이름을
받습니다. `revision_number`·`tenant_id`·`project_id`·`stateful`·이름·ID는 모두 서버 조건이며
상속한 int/bool descriptor가 있어도 로컬 필터로 변환하지 않습니다. project의 response alias가
별도로 선언된 tenant query를 대체하지 않습니다. 로컬 조건은 `created_at`·`updated_at`·
`security_group_rules`의 원문 JSON입니다. semantic `status`는 알 수 없는 이름으로 버리고 raw
status는 query로 보존합니다. 기존 `WithName`은 정확한 로컬 비교와 서버 hint를 유지하며
`WithStatus`는 native 모델에 Status가 없어 `ErrUnsupported`입니다.

rule 배열은 순서·길이·완전한 요소를 비교하므로 중첩 object는 subset으로 비교하지 않습니다.
unknown rule 필드·큰 숫자·null 요소는 원문으로 보존합니다. 반환 `SecGroup.Rules`는 native
`[]SecGroupRule`이며 null 요소는 zero struct입니다. missing/null rules는 nil slice, 빈 배열은
빈 slice로 반환합니다. caller 조건·timestamp 문자열은 변환하지 않고 Python list wrapping도
적용하지 않습니다. Group과 Rule의 알려진 문자열·port/revision 정수·timestamp 디코드는 cap과
로컬 비교보다 먼저 전체 페이지에 적용합니다. Group과 각 Rule은 각각 NoZ→RFC3339 재시도하며
같은 객체 안의 두 timestamp가 혼합 형식이면 실패할 수 있습니다.

native List는 concrete ListOpts를 받습니다. SDK가 소유하는 두 pager 경로가 반복/nil/확장
query와 client/provider를 보존하므로 caller builder는 필요하지 않습니다. Body 경로도 기존
query-only 경로와 같은 service URL 검사·`security-groups` 경로·`security_groups` envelope·
native 200/204/300·plural links·foreign continuation을 사용합니다. 소비한 Body null 행은
오류이며 query-only null 행은 native zero model입니다. raw cap·first-page·break는 미소비
로컬 비교와 continuation을 생략하며 wire limit hint를 추가하지 않습니다. 전체 결과 수집의
terminal 오류는 부분 slice를 반환하지 않습니다. bulk canonical 우선·입력 snapshot·전체 교체·
clear·최종값 검증·namespace 충돌은 공통 옵션 정책을 따릅니다.

`Network(ctx).API.SecurityGroups.Resources`와 `NetworkV2(ctx).SecurityGroups.Resources`는
같은 캐시된 client를 사용하며 typed native List·Get/Delete·FindIdentity·rule 변경 API는 유지합니다.
[Security Group Python/Go 예제](../network/v2/extensions/security/groups/listing/README.md),
[HTTP 7개 그룹](../api/security_group_list_filters_test.go),
[Connection 2개 그룹](../connection_security_group_filters_test.go),
[SDK 소유 raw pager](../internal/nativefind/security_group_bodies_test.go)에서 이 계약을 검증합니다.
[AST manifest](../api/openstacksdk/resources/network/v2/security_group.json)는 source SHA 7개·AST 39개를,
[생성기 검증](../internal/cmd/sdkgen/security_group_filters_test.go)은 native 함수 21개와 상수 2개를
확인합니다. 전체 Resource/cache·descriptor coercion·session·상속 continuation과 Proxy/JMESPath는
계속 구현할 대상입니다.

### Trunk의 속성 이름 분류

`Trunks.Resources.List/All`은 Python `conn.network.trunks(**query)`의 canonical query 14개와
wire 별칭을 포함한 18개 이름, non-query Body 2개를 분류합니다. query는 description·fields·
is_admin_state_up→admin_state_up·limit·marker·name·port_id·project_id·status·sub_ports·tags와
세 태그 별칭입니다. 원문 `id`·`tenant_id`만 JSON으로 로컬 비교합니다. `project_id`의 response
alias는 로컬 tenant를 query 별칭으로 바꾸지 않습니다. semantic tenant와 raw
`WithQuery("tenant_id", ...)`, semantic id와 raw `WithQuery("id", ...)`는 각각 독립적으로
지정할 수 있습니다. query-only sub_ports와 bool에는 로컬 배열 비교나 truthiness를 추가하지 않습니다.

Trunk는 Resource와 TagMixin을 상속하며 NetworkResource를 상속하지 않습니다. native에 있는
created_at·updated_at·revision_number는 이 Python 클래스의 semantic 이름이 아니므로 버립니다.
기존 `WithName`은 서버 hint와 정확한 로컬 이름 조건을 추가하고 `WithStatus`는 최종 status
query 값과 대소문자를 무시해 비교합니다. semantic name/status와 해당 기존 옵션을 동시에
지정하면 충돌 오류입니다. raw status 단독과 semantic status 단독은 서버 조건만 지정합니다.

native Trunk·Subport·time.Time의 알려진 필드를 페이지 전체에서 먼저 디코드합니다.
timestamp는 RFC3339를 요구하며 NoZ 형식을 추가하지 않습니다. 뒤 행의 잘못된 nested
segmentation_id·bool·timestamp는 앞 행에서 cap에 도달해도 오류입니다. null Subport 요소는
native zero struct를 유지합니다. missing/null id·tenant는 원문 null 조건으로 비교하며 빈
문자열과 구분합니다. query-only null 행은 native zero Trunk지만 소비한 Body-filter null 행은
`ErrInvalidOption`입니다. 빈 object는 유효하며 cap/break 이후 미소비 로컬 조건은 검사하지 않습니다.

목록은 기존 native 200/204/300과 inherited `LinkedPageBase.NextPageURL`을 유지합니다.
top-level `links.next`만 다음 페이지로 처리하며 trunks_links·top-level next·HTTP Link를
새로 해석하지 않습니다. foreign next도 native 정책대로 따르고 잘못된 next 타입·순환·후속
HTTP/decode/취소는 terminal 오류입니다. marker fallback·limit hint·origin 제한을 추가하지 않습니다.
같은 cached client의 `Network(ctx).API.Trunks.Resources`와 `NetworkV2(ctx).Trunks.Resources`에
옵션을 재사용할 수 있으며 typed List·Get/Delete·FindIdentity·subport 변경 API는 유지합니다.

[Trunk Python/Go 예제](../network/v2/extensions/trunks/listing/README.md),
[HTTP 7개 그룹](../api/trunk_list_filters_test.go),
[Connection 2개 그룹](../connection_trunk_filters_test.go)가 옵션과 native 응답 경계를 검증합니다.
[AST manifest](../api/openstacksdk/resources/network/v2/trunk.json)는 source SHA 6개·AST 35개를,
[생성기 검증](../internal/cmd/sdkgen/trunk_filters_test.go)은 native 함수 20개와 resourcePath 상수를
확인합니다. 전체 Resource/cache·descriptor coercion·session·상속 continuation과 Proxy/JMESPath는
계속 구현할 대상입니다.

### Subnet의 속성 이름 분류

`Subnets.Resources.List/All`의 `resource.WithFilter`/`WithFilters`는 Python
`conn.network.subnets(**query)`처럼 속성 이름을 받아 서버 query와 로컬 Body로 나눕니다.
query 24개의 canonical/wire 이름 30개와 non-query Body 속성 9개 전체를 고정 Python
소스에서 추출합니다. `is_dhcp_enabled`→`enable_dhcp`, `any_tags`→`tags-any` 같은 query
별칭은 변환하며 `tenant_id`·`revision_number`는 로컬 비교입니다. `name`·`id`·`tags`는
서버 query만 지정합니다. Secret에도 아래의 별도 속성 분류가 연결되어 있습니다.

개별 조건은 마지막 target 값이 이깁니다. bulk map에서는 canonical query 이름이 wire
별칭보다 우선하며 nil·false·빈 배열도 지정한 값입니다. bulk는 semantic 전체 교체이고
nil/빈 map은 semantic만 clear합니다. 최종 선택값만 검증하므로 제거된 조건의 encoding
오류도 제거됩니다. 옵션은 생성 시 JSON snapshot을 만들며 알 수 없는 이름은 버립니다.
전용 로컬 제어 이름과 지원하지 않는 Python list controls는 field 값으로 사용하지 않습니다.

Go query encoding은 scalar와 scalar 배열을 받으며 배열은 반복 key, bool은 소문자,
JSON number는 원래 정밀도/표기, null은 URL 생략으로 처리합니다. nil/빈 배열의 key 존재는
충돌 검사에 유지됩니다. 객체/중첩 배열 query는 HTTP 전에 오류입니다. 같은 key의 raw query,
`WithPageSize` 또는 `WithName` query hint와 겹치거나 같은 Body 필드의 명시 조건과 겹치면
`ErrInvalidOption`입니다. raw query와 명시 Body 조건은 semantic clear로 지워지지 않습니다.

[Subnet 사용법](../network/v2/subnets/README.md)에 전체 이름·reserved controls·Python/Go
예제가 있으며 [HTTP 7개 그룹](../api/network_subnet_semantic_filters_test.go)과
[공통 6개 그룹](../resource/filters_test.go)에서 분류·최종값·snapshot·충돌·페이지를 검증합니다.
[AST manifest](../api/openstacksdk/resources/network/v2/subnet.json)는
[생성기](../internal/cmd/sdkgen/README.md)가 현재 소스 SHA와 재추출 결과를 확인합니다.
전체 Resource descriptor/coercion/cache·상속 continuation/session·Proxy `__conflicting_attrs`
복구·deprecated JMESPath 조건은 별도 비교 범위로 남습니다.

### Subnet Pool의 속성 이름 분류

`SubnetPools.Resources.List/All`은 Python `conn.network.subnet_pools(**query)`의 선언된
query 16개와 non-query Body 속성 10개를 분류합니다. `is_shared`→`shared`와 태그 별칭을
포함한 query 이름 20개를 받습니다. `name`·`project_id`·`ip_version`·`is_default` 등 query
속성은 응답에서 다시 비교하지 않습니다. `tenant_id`는 `project_id`의 query 별칭이 아닙니다.
semantic `id`는 원문 로컬 조건이며 native ListOpts의 ID query와는 다릅니다.
`resource.WithQuery("id", ...)`로 서버 ID query를 별도로 지정할 수 있습니다.

| Python 로컬 속성 | 원문 필드 | 응답 비교 정책 |
|---|---|---|
| `id`, `tenant_id` | 같은 이름 | 원문 JSON, 누락/null을 빈 문자열과 구별 |
| `created_at`, `updated_at` | 같은 이름 | 원문 timestamp 문자열 |
| `prefixes` | 같은 이름 | 배열 순서·길이·원소 전체, null 요소 보존 |
| `default_prefix_length` | `default_prefixlen` | 정확한 정수 |
| `minimum_prefix_length` | `min_prefixlen` | 정확한 정수 |
| `maximum_prefix_length` | `max_prefixlen` | 정확한 정수 |
| `default_quota`, `revision_number` | 같은 이름 | 정확한 정수 |

semantic 옵션은 위 Python 속성 이름만 분류합니다. raw prefix length 이름은 알 수 없는
semantic 이름이므로 버립니다. 명시 `WithBodyFilter`는 세 raw 이름과 Python 이름을 모두
받으며 같은 필드의 bulk 중복은 거부합니다. query alias의 canonical 우선·snapshot·전체
교체·clear·충돌 검사는 Subnet과 같은 공통 옵션 정책입니다.

정수 응답의 비교는 Go의 정확한 정수 정책을 따릅니다. 정수인 JSON decimal/exponent와
부호·공백이 있는 십진 정수 문자열을 받아들이고 fractional/bool/object를 거부합니다.
Python은 bool을 int로 인정하고 float을 절삭하며 일부 문자열을 0으로 변환하므로 완전한
descriptor 변환 호환을 주장하지 않습니다. 필터 operand에는 응답 정수 변환을 적용하지 않습니다.

native 전체 페이지 디코드가 원문 비교와 raw cap보다 먼저 실행됩니다. 모든 행에 세 prefix
length가 필요하며 null 행·빈 object도 native 오류입니다. native decoder는 prefix 숫자를
절삭할 수 있으나 선택한 정수 필드의 로컬 비교는 fractional 값을 거부합니다. default quota와
revision은 native int decoder가 먼저 검증하며, timestamp 두 개는 같은 native decoding
branch에서 처리됩니다. prefix 배열의 null 요소를 원문 null로 비교해도 반환 `Prefixes`의
해당 값은 native 빈 문자열입니다. 누락/null 배열과 빈 배열은 구별합니다.
`subnetpools_links`의 next link와 native 성공 코드 200/204/300을 유지하며 방문하지 않은
다음 페이지를 미리 검사하지 않습니다. 기존 `WithName` hint·로컬 이름 조건, typed List,
Get/Delete·FindIdentity·AddPrefixes/RemovePrefixes는 semantic 옵션과 독립적입니다.

[Subnet Pool Python/Go 사용법](../network/v2/extensions/subnetpools/listing/README.md),
[HTTP 7개 그룹](../api/subnet_pool_list_filters_test.go),
[공유 Connection 2개 그룹](../connection_subnet_pool_filters_test.go)에서 실제 사용과 경계를 확인합니다.
[AST manifest](../api/openstacksdk/resources/network/v2/subnet_pool.json)는 source SHA 7개와
AST 33개를 기록하고 [생성기](../internal/cmd/sdkgen/README.md)가 현재 소스를 다시 검증합니다.
전체 Resource/cache·coercion·상속 continuation/session·Proxy conflicting attrs·JMESPath는
추가 비교 범위입니다.

### AddressGroup의 속성 이름 분류

`SecurityAddressGroups.Resources.List/All`은 Python `conn.network.address_groups(**query)`의
query 8개와 non-query Body 3개를 `resource.WithFilter`/`WithFilters`로 분류합니다.
`name`·`description`·`project_id`·`fields`·정렬·페이지 인자는 서버에 전달하며,
`id`·`tenant_id`·`addresses`는 native 디코드 뒤 원본 행에서 비교합니다. `tenant_id`는
`project_id` query의 별칭으로 바꾸지 않으며 native 모델에 없는 원문 값도 비교할 수 있습니다.

`WithName`의 기존 서버 hint와 로컬 이름 조건은 유지됩니다. semantic `name`은 서버
query만 지정하므로 두 옵션을 동시에 쓰면 같은 query 목적지의 충돌 오류입니다. raw
`tenant_id`·`id`·`addresses` query는 같은 이름의 로컬 Body 조건과 독립적입니다.
raw `status`도 기존대로 서버에 전달하지만 모델에 Status가 없어 `WithStatus`는 지원하지 않습니다.

주소 배열의 원문 null 요소는 native `[]string`의 빈 문자열 변환과 구분합니다.
전체 페이지의 native 디코드는 로컬 조건과 cap보다 먼저 적용되며 결과 모델은 그대로 반환합니다.
[AddressGroup 사용법](../network/v2/extensions/security/addressgroups/listing/README.md)에
Python/Go 예제, 이름과 프로젝트 필터, 원문·native 결과 및 plural-links 페이지 경계를 설명합니다.
[AST manifest](../api/openstacksdk/resources/network/v2/address_group.json)는 고정 소스를 검증합니다.

### QoS Policy의 속성 이름 분류

`QoSPolicies.Resources.List/All`은 `resource.WithFilter`/`WithFilters`로 canonical query15개와
로컬 Body2개를 분류합니다. `is_shared`→`shared`, `any_tags`→`tags-any`,
`not_tags`→`not-tags`, `not_any_tags`→`not-tags-any`의 별칭을 포함한19개 이름을 받습니다.
같은 bulk의 canonical 값이 우선하며 `false`·null·빈 배열도 선택값입니다. `id`·`name`·
`description`·`project_id`·`is_default`·태그는 서버 query입니다. 응답의 native 필드가 다른 값이어도
해당 query를 로컬 predicate로 다시 적용하지 않습니다.

`rules`와 deprecated `tenant_id`는 native 전체 페이지 디코드 뒤 원본 JSON에서 비교합니다.
`tenant_id`는 `project_id`의 query 별칭이나 로컬 fallback이 아닙니다. missing/null과 빈 문자열을
구분하며 알려진 tenant ID의 non-null non-string 응답은 native 오류입니다. rule 배열은 전체 순서·길이·
내부 object의 모든 필드를 비교하고 큰 숫자를 정확하게 구분합니다. 반환 Policy의 `Rules`는 기존
float64 decoding을 유지합니다. typed 모델의 timestamp·revision number는 Python QoSPolicy에 선언된
필터가 아니므로 알 수 없는 semantic 이름으로 버립니다. `WithName`의 hint+로컬 이름 비교와
raw status query·typed List·FindIdentity는 각각 유지합니다.

[QoS Policy Python/Go 사용법](../network/v2/extensions/qos/policies/listing/README.md)에 전체 필터·
충돌·원문 비교·plural-links continuation·로컬 cap을 설명합니다. [AST manifest](../api/openstacksdk/resources/network/v2/qos_policy.json)는
Resource+TagMixin의 선언과 SHA6·AST18을 검증합니다. Python의 list coercion·Resource/default/alias/cache/session·
상속 continuation·deprecated JMESPath와 Proxy conflicting attrs 처리는 추가 비교 범위입니다.

### Secret의 속성 이름 분류

`Secrets.Resources.List/All`은 Python `conn.key_manager.secrets(**query)`의
query 12개와 로컬 Body 속성 12개를 `WithFilter`/`WithFilters`로 분류합니다.
`algorithm`은 `alg`로 보내고 wire 이름도 받습니다. `bits`·`created`·`updated`·
`expiration`은 서버 조건이며 `bit_length`·`created_at`·`updated_at`·`expires_at`은
원본 행을 비교합니다. `id`는 literal id의 존재를 우선하고 없을 때 전체 secret_ref를
사용합니다. `secret_id`는 별도의 HREF accessor로 마지막 path 부분을 비교하므로 두
조건을 하나로 합치지 않습니다. 반복 query 값·bulk canonical 우선·snapshot·충돌과
clear 규칙은 위의 공통 semantic 옵션 정책을 따릅니다.

[Secret 목록 사용법](../keymanager/v1/secrets/listing/README.md)에 전체 이름,
native 전체 페이지 디코드와 원문 비교의 순서, JSON 타입 및 pagination의 Python/Go
차이를 설명합니다. [Secret AST manifest](../api/openstacksdk/resources/key_manager/v1/secret.json)는
생성 시 현재 고정 소스의 checksum과 독립 AST 재추출 결과를 확인합니다.

### Container의 속성 이름 분류

`Containers.Resources.List/All`은 Python `conn.key_manager.containers(**query)`의
상속 query `limit`·`marker`와 로컬 Body 속성 10개를 분류합니다. `name`은 원본 응답을
비교하며, 기존 `WithName`은 서버 hint와 로컬 비교를 함께 지정합니다. `WithQuery("name", ...)`과
`WithQuery("offset", ...)`은 별도의 wire 확장이고 semantic `offset`은 버립니다.

`id`는 literal id의 존재를 우선하고 없을 때 전체 `container_ref`를 사용합니다.
`container_id`는 passive HREF accessor의 마지막 path 부분을 비교합니다. `created_at`·
`updated_at`은 timestamp 원문이며 `secret_refs`·`consumers`는 순서·추가 필드를 포함한
원문 배열 전체를 비교합니다. native 전체 페이지 디코드가 먼저 적용되므로 잘못된 중첩
배열이나 timestamp를 로컬 조건 또는 행 수 제한으로 숨기지 않습니다.

[Container 목록 사용법](../keymanager/v1/containers/listing/README.md)에 전체 속성 이름과
Python scalar-to-list coercion·응답 타입·pagination의 경계를 설명합니다.
[Container AST manifest](../api/openstacksdk/resources/key_manager/v1/container.json)는
상속 query defaults와 ID 접근을 포함해 생성 시 현재 소스를 검증합니다.

### Order의 속성 이름 분류

`Orders.Resources.List/All`은 Python `conn.key_manager.orders(**query)`의 상속 query
`limit`·`marker`와 로컬 Body 속성 14개를 분류합니다. 최상위 원문 `name`은 중첩
`meta.name`과 별개이며, `meta`는 원문 객체의 추가 필드·정확한 숫자·중첩 조건을 비교합니다.
native Order에 최상위 Name이 없으므로 공통 `WithName`과 이름 Ref 조회는 지원하지 않습니다.

`id`는 literal id의 존재를 우선하고 없을 때 전체 `order_ref`를 사용합니다. `order_id`는
order 참조, `secret_id`는 secret 참조의 마지막 path 부분이며 서로 다른 accessor입니다.
두 참조는 조건 비교에만 쓰고 요청으로 따라가지 않습니다. ID 기반 조회·삭제·대기는
호출자의 ID로 고정된 order 경로를 사용하며 응답 참조가 바뀌어도 같은 경로를 유지합니다.

원문 timestamp와 metadata 비교에는 native Order·Meta의 전체 페이지 디코드가 먼저
적용됩니다. 잘못된 Meta 정수나 timestamp도 로컬 cap 뒤에서 오류가 될 수 있습니다.
[Order 목록 사용법](../keymanager/v1/orders/listing/README.md)에 Python/Go 예제와 native
응답 모델·dict coercion·페이지 경계를 설명합니다.
[Order AST manifest](../api/openstacksdk/resources/key_manager/v1/order.json)는 두 formatter와
metadata 속성을 포함해 생성 시 현재 고정 소스를 검증합니다.

### Subnet의 원본 응답 필터

Python `conn.network.subnets(prefix_length=24, dns_nameservers=["192.0.2.53"])`에 대응하는
Go 호출은 [Subnet README](../network/v2/subnets/README.md)에 있습니다. 다음 9개는 Python의
query mapping에 포함되지 않은 Body 속성입니다. Go는 `WithFilter(s)`로 자동 분류하거나
명시적인 `WithBodyFilter(s)`로 선택합니다.

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
