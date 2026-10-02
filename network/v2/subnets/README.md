# Subnet 목록과 로컬 필터

Python `conn.network.subnets(**query)`의 서버 query와 로컬 Body 조건을 Go에서는 명시적인
옵션으로 선택합니다. SDK가 필드 선택·값 복사·페이지 순회·로컬 비교를 처리합니다.

```python
subnets = conn.network.subnets(
    network_id="network-a",
    prefix_length=24,
    dns_nameservers=["192.0.2.53"],
    max_items=100,
)
for subnet in subnets:
    print(subnet.id, subnet.cidr)
```

```go
package examples

import (
    "context"
    "fmt"

    networkv2 "gophercloudsdk/network/v2"
    "gophercloudsdk/resource"
)

func ListSubnets(ctx context.Context, service *networkv2.Service) error {
    for subnet, err := range service.Subnets.Resources.List(ctx,
        resource.WithQuery("network_id", "network-a"),
        resource.WithBodyFilters(map[string]any{
            "prefix_length": 24,
            "dns_nameservers": []string{"192.0.2.53"},
        }),
        resource.WithMaxItems(100),
    ) {
        if err != nil { return err }
        fmt.Println(subnet.ID, subnet.CIDR)
    }
    return nil
}
```

Connection에서는 `conn.Network(ctx)`의 `service.API.Subnets.Resources`를 사용합니다.
로컬 필드는 `allocation_pools`, `dns_nameservers`, `host_routes`, `service_types`,
`created_at`, `updated_at`, `prefixlen`, `tenant_id`, `revision_number`의 9개입니다.
Python 이름 `prefix_length`도 `prefixlen`의 별칭으로 받습니다. 개별 옵션은 마지막 조건이
이기고, 같은 bulk map에 두 이름을 넣으면 HTTP 전에 `ErrInvalidOption`입니다.

배열은 순서·길이·내부 객체 전체가 같아야 합니다. 원본 응답의 추가 key·null 원소·정확한 정수·
timestamp 표기를 비교하며, 반환값은 native typed Subnet입니다. `prefix_length`를 CIDR에서
계산하지 않고 서비스가 실제 반환한 `prefixlen`을 사용합니다. 누락/null은 null 조건과 일치하지만
빈 값과는 다릅니다. `tenant_id`와 `project_id`는 독립 필드입니다.

타입이 선언되지 않은 `prefixlen`은 원래 JSON 값 그대로 비교하고 문자열을 숫자로 강제
변환하지 않습니다. number는 정확한 decimal 값으로 비교합니다. native
decoder가 현재 페이지 전체를 먼저 검사하므로 알려진 필드의 잘못된 응답은 cap 뒤에도
오류입니다. `WithMaxItems`는 로컬 조건 적용 전의 행을 세며, consumer break·첫 페이지 옵션은
후속 요청을 멈춥니다. Subnet은 typed `WithStatus`를 지원하지 않습니다.

native `API.List`와 `FindIdentity`는 각각 기존 concrete List 옵션과 identity 옵션을 사용합니다.
공통 Body 조건은 ordinary `Resources.List/All`에서 선택하며 raw `WithQuery`는 서버로 전달합니다.
Python 속성을 자동 분류하는 옵션·Resource/cache·상속 continuation 전체 계약은 남아 있습니다.
[전체 목록 정책과 타입 경계](../../../docs/listing.md#subnet의-원본-응답-필터),
[자동 조회 비교](../../../docs/finding-identities.md), [실제 HTTP 검증](../../../api/network_subnet_raw_body_filters_test.go)을 참고하세요.
