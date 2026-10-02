# Subnet 목록과 로컬 필터

Python `conn.network.subnets(**query)`처럼 속성 이름과 값을 전달하면 SDK가 서버 query와
로컬 Body 조건을 나눕니다. Go는 `resource.WithFilter`/`WithFilters`를 사용하며
builder나 predicate를 구현할 필요가 없습니다. SDK가 이름 변환·값 복사·페이지 순회·로컬 비교를 처리합니다.

```python
subnets = conn.network.subnets(
    network_id="network-a",
    is_dhcp_enabled=False,
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
        resource.WithFilters(map[string]any{
            "network_id": "network-a",
            "is_dhcp_enabled": false,
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
`network_id`와 `is_dhcp_enabled`는 서버 query로, `prefix_length`와 `dns_nameservers`는
로컬 조건으로 처리합니다. `tenant_id`와 `revision_number`도 로컬 조건이며 native
ListOpts에 같은 wire 이름이 있다는 이유로 서버 query로 보내지 않습니다.

서버 query의 canonical 이름은 24개입니다. 아래 별칭과 canonical 이름을 합쳐 30개 이름을
받으며, `name`·`id`·`tags`는 서버 query로만 처리합니다. 서버가 결과를 어떻게 거르는지는
Neutron의 query 의미를 따릅니다.

| canonical 이름 | HTTP 이름 |
|---|---|
| `any_tags` | `tags-any` |
| `not_any_tags` | `not-tags-any` |
| `not_tags` | `not-tags` |
| `is_dhcp_enabled` | `enable_dhcp` |
| `subnet_pool_id` | `subnetpool_id` |
| `use_default_subnet_pool` | `use_default_subnetpool` |
| `cidr`, `description`, `dns_publish_fixed_ip`, `fields`, `gateway_ip`, `id`, `ip_version`, `ipv6_address_mode`, `ipv6_ra_mode`, `limit`, `marker`, `name`, `network_id`, `project_id`, `segment_id`, `sort_dir`, `sort_key`, `tags` | 같은 이름 |

개별 `WithFilter`는 같은 target의 마지막 값이 이깁니다. 한 `WithFilters` map에 canonical
이름과 wire 별칭을 함께 넣으면 canonical 값이 이기며 nil·false·빈 배열도 지정한 값으로
셉니다. `WithFilters`는 semantic 조건 전체를 교체하고 nil/빈 map은 그 조건만 지웁니다.
옵션 생성 시 map·slice·pointer 값을 복사하므로 나중에 caller가 바꾸거나 같은 옵션을
동시에 재사용해도 조건은 바뀌지 않습니다. 최종 선택된 값만 검증하므로 덮어쓰거나 지운
이전 값의 JSON 오류도 제거됩니다.

query 값은 문자열·bool·JSON number·그 값들의 배열을 받습니다. bool은 `true`/`false`,
배열은 반복 query key이며 CSV로 합치지 않습니다. JSON number의 정밀도와 표기를 보존하고
null 원소는 생략합니다. nil·빈 배열은 URL 값을 만들지 않지만 key를 지정했다는 사실은
별칭 우선순위와 충돌 검사에 유지됩니다. 객체·중첩 배열 query는 HTTP 전에 `ErrInvalidOption`입니다.
이 HTTP 인코딩은 Go SDK의 정책이며 Python HTTP 의존성의 모든 변환을 재현하지 않습니다.

선언되지 않은 semantic 이름은 버립니다. `prefixlen`도 semantic 이름으로 받지 않으므로
`prefix_length`를 사용합니다. `max_items`와 `paginated`는 전용 `WithMaxItems`·`WithPaginated`를
사용해야 하며 semantic map에 남아 있으면 `ErrInvalidOption`입니다. `allow_unknown_params`,
`base_path`, `headers`, `jmespath_filters`, `microversion`, `resource_type`, `session`은
field 값으로 받을 수 없어 `ErrUnsupported`입니다. 다른 binding에서 `WithFilter(s)`를
사용하면 clear만 지정해도 `ErrUnsupported`이며, 현재 이 분류는 Subnet에 연결되어 있습니다.

raw `WithQuery`와 명시 `WithBodyFilter(s)`도 사용할 수 있습니다. semantic query와 raw query/
`WithPageSize`가 같은 HTTP key를 지정하거나, semantic `name`과 `WithName`의 query hint가
겹치면 값이 같아도 HTTP 전에 `ErrInvalidOption`입니다. semantic Body와 명시 Body가 같은
canonical 필드를 지정해도 같은 오류입니다. 각 namespace는 독립적이므로 semantic clear는
raw query와 명시 Body 조건을 지우지 않습니다.

## 명시적인 Body 조건과 응답 비교

로컬 필드는 `allocation_pools`, `dns_nameservers`, `host_routes`, `service_types`,
`created_at`, `updated_at`, `prefixlen`, `tenant_id`, `revision_number`의 9개입니다.
명시 `WithBodyFilter(s)`는 Python 이름 `prefix_length`도 `prefixlen`의 별칭으로 받습니다.
이 명시 Body 옵션의 개별 조건은 마지막 값이 이기고, 같은 bulk map에 두 이름을 넣으면
HTTP 전에 `ErrInvalidOption`입니다.

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
Python의 전체 Resource descriptor/default/coercion·conditional cache·상속 continuation/session·
Proxy의 `__conflicting_attrs` 복구·deprecated JMESPath 조건은 남아 있습니다.
고정 소스의 분류는 [AST manifest](../../../api/openstacksdk/resources/network/v2/subnet.json)와
[생성기 소스 검증](../../../internal/cmd/sdkgen/README.md)에 기록합니다.
[전체 목록 정책과 타입 경계](../../../docs/listing.md#subnet의-원본-응답-필터),
[자동 조회 비교](../../../docs/finding-identities.md), [semantic HTTP 검증](../../../api/network_subnet_semantic_filters_test.go),
[원본 행 HTTP 검증](../../../api/network_subnet_raw_body_filters_test.go)을 참고하세요.
