# 이름·ID 문자열 자동 조회

`FindIdentity(ctx, identity, options...)`는 애플리케이션이 문자열을 ID와 이름으로 나누지
않아도 SDK가 조회를 처리하는 API입니다. 안전한 ID 경로로 GET을 먼저 시도하고,
정해진 HTTP 오류에만 목록 검색으로 전환합니다. 목록에서는 모든 페이지의 ID와 이름을
원래 문자열과 정확히 비교합니다. 두 행이 일치하면 같은 ID를 반복한 경우에도
`resource.ErrAmbiguous`입니다.

## 서비스별 대응

| 서비스 | openstacksdk | Go |
|---|---|---|
| Nova 서버 | `conn.compute.find_server("web-01")` | `compute.Servers.FindIdentity(ctx, "web-01")` |
| Cinder v3 볼륨 | `conn.block_storage.find_volume("data-01")` | `storage.Volumes.FindIdentity(ctx, "data-01")` |
| Neutron 포트 | `conn.network.find_port("web-port")` | `network.Ports.FindIdentity(ctx, "web-port")` |
| Designate recordset | `conn.dns.find_recordset("www.example.org.", zone)` | `records.FindIdentity(ctx, "www.example.org.")`, `records`는 `RecordSets.InZone`의 반환값 |
| Octavia member | `conn.load_balancer.find_member("backend-01", pool)` | `members.FindIdentity(ctx, "backend-01")`, `members`는 `Pools.Members`의 반환값 |

위 다섯 리소스가 공통 자동 조회를 지원합니다. 기존 상위 `conn.Compute(ctx).Servers`,
`conn.BlockStorage(ctx).Volumes`, `conn.Network(ctx).Ports`도 같은 조회 정책을 제공합니다.
다른 native collection은 아직
`FindIdentity`에 `ErrUnsupported`를 반환합니다. 숫자 ID, Swift 객체 키, URL에서 추출한
ID와 이름이 없는 리소스에 이 정책을 일괄 적용하지 않습니다. Senlin의 다섯
`FindIdentity` facade는 [전용 조회 옵션과 응답 검증](../clustering/v1/finding/README.md)을
사용합니다.

상위 서비스 객체를 받은 뒤 동일한 옵션을 여러 서비스에 사용할 수 있습니다.

```go
package example

import (
    "context"

    computev2 "gophercloudsdk/compute/v2"
    storagev3 "gophercloudsdk/blockstorage/v3"
    networkv2 "gophercloudsdk/network/v2"
    "gophercloudsdk/resource"
)

func FindApplicationResources(ctx context.Context, compute *computev2.Service,
    storage *storagev3.Service, network *networkv2.Service) error {
    strict := resource.WithIdentityFindIgnoreMissing(false)
    server, err := compute.Servers.FindIdentity(ctx, "web-01", strict)
    if err != nil { return err }
    volume, err := storage.Volumes.FindIdentity(ctx, "data-01", strict)
    if err != nil { return err }
    port, err := network.Ports.FindIdentity(ctx, "web-port", strict)
    if err != nil { return err }
    _, _, _ = server, volume, port
    return nil
}
```

## 고정 부모 범위

부모는 `resource.ID` 또는 `resource.Name`으로 한 번 해석합니다. 자식의 자동 조회가
GET에서 목록으로 전환해도 같은 zone 또는 pool을 유지합니다. 부모 이름이 중복되거나
없으면 scope를 만들 때 오류를 반환하며 자식 요청을 보내지 않습니다.

```go
package example

import (
    "context"

    dnsv2 "gophercloudsdk/dns/v2"
    lbv2 "gophercloudsdk/loadbalancer/v2"
    "gophercloudsdk/resource"
)

func FindScopedResources(ctx context.Context, dns *dnsv2.Service,
    loadbalancer *lbv2.Service) error {
    records, err := dns.RecordSets.InZone(ctx, resource.Name("example.org."))
    if err != nil { return err }
    record, err := records.FindIdentity(ctx, "www.example.org.",
        resource.WithIdentityFindIgnoreMissing(false))
    if err != nil { return err }
    members, err := loadbalancer.Pools.Members(ctx, resource.ID("pool-id"))
    if err != nil { return err }
    member, err := members.FindIdentity(ctx, "backend-01",
        resource.WithIdentityFindIgnoreMissing(false))
    if err != nil { return err }
    _, _ = record, member
    return nil
}
```

## 기본값과 오류

| 설정 | 동작 |
|---|---|
| 옵션 생략 | GET400·403·404 뒤 목록 검색, 성공한 빈 검색은 `nil, nil` |
| `WithIdentityFindIgnoreMissing(false)` | 성공한 빈 검색은 `ErrNotFound` |
| `WithIdentityFindFallback(resource.FindFallbackNotFoundOnly)` | 시도한 GET이 404일 때만 목록 검색 |
| `WithIdentityFindFallback(resource.FindFallbackNever)` | GET만 사용, 404에만 미존재 옵션 적용 |
| `WithIdentityFindQuery(key, value)` | fallback 목록의 서버 query, 같은 key는 뒤의 옵션 우선 |
| `WithIdentityFindOptions(resource.IdentityFindOpts{...})` | bool pointer와 query를 포함한 concrete 설정 |

기본 미존재 무시는 고정한 Python 소스의 `ignore_missing=True`에 대응합니다. 이 Python
소스는 앞으로 기본값을 변경한다는 deprecation 경고도 포함합니다. 기존 Go
`Find(ctx, resource.ID/Name(...))`는 명시적인 조회 방식과 기본 strict 정책을 유지합니다.

401·409·5xx, 전송·본문 읽기·decode 오류, context 취소는 목록 검색으로 바꾸지 않습니다.
받아들인 응답을 해석하지 못한 `ResponseError`도 원인이 400·403·404라고 해서 fallback하지
않습니다. 목록의 HTTP 오류·decode 오류·페이지 순환은 일치하는 행을 이미 찾았어도
반환합니다. 목록 403·404는 미존재 무시 옵션으로 숨기지 않습니다. 반면 GET403 뒤
정상적인 빈 목록을 받았다면 strict 오류는 새 검색의 `ErrNotFound`이며 원래 403을
cause로 붙이지 않습니다.

공백이나 `/`, `%`, `?`, `#` 등 ID 경로에 넣을 수 없는 문자가 있는 이름은 원문 그대로
query에 인코딩해 목록에서 찾습니다. 이 경우 GET을 생략하며 `NotFoundOnly`도 목록을
사용합니다. `Never`는 HTTP 전에 `ErrInvalidOption`입니다. 예를 들어 `%2F`라는 이름은
slash로 decode하지 않고 query에서 `%252F`가 됩니다. 빈 문자열, 공백뿐인 문자열,
잘못된 UTF-8과 제어 문자는 요청 전에 거부합니다. Python의 모든 문자열 GET-first와
다른 Go 경로 정책입니다.

query는 fallback에만 전달하며 GET의 필터로 적용하지 않습니다. caller가 binding의
이름 query key를 지정하면 자동 이름 hint로 덮어쓰지 않습니다. 기본 hint는 Nova에서
`^`·`$`와 `regexp.QuoteMeta`로 만든 정확한 정규식, 다른 네 binding에서는 literal
문자열입니다. 서버가 hint를 무시해도 SDK의 ID/이름 비교는 그대로 수행합니다.

GET 결과와 소비한 목록의 각 행은 nil이 아닌 리소스와 안전한 ID를 요구합니다.
빈 ID나 경로에 사용할 수 없는 ID가 있는 행은 일치 여부와 관계없이 오류입니다.
typed native 응답에 없는 HTTP 본문·헤더를 이 검증 오류에 만들어 붙이지 않습니다.

`max_items`, `paginated`, `base_path`, `list_base_path`, `jmespath_filters`, `headers`,
`microversion`, `allow_unknown_params`, `ignore_missing`, `fallback`은 이 조회의 query가
아닙니다. 대소문자를 바꾼 키도 HTTP 전에 거부합니다. 중복과 후속 오류를 확인하기 위해 자동 조회는 전체
목록을 소비하며 로컬 cap이나 첫 페이지 중단 옵션을 받지 않습니다. 생성한 옵션은
bool pointer, query map과 slice를 복사하고 호출마다 독립적으로 적용합니다. 옵션
생성 뒤 원본 설정을 바꾸어도 이미 생성한 옵션의 동작은 바뀌지 않습니다.

## Python과 추가로 비교할 범위

Nova와 Cinder의 fallback은 native detail pager를 사용하여 Python의 기본
`details=True`에 대응합니다. `details=False`로 summary 경로를 선택하는 옵션,
`list_base_path` 교체, 호출별 header·microversion 선택은 이 공통 API에 없습니다.
`all_projects` 같은 서버 필터는 `WithIdentityFindQuery`로 명시합니다. query 자체의
권한과 의미는 해당 OpenStack 서버가 결정합니다.

이 API는 typed Gophercloud 응답을 반환합니다. Python의 Resource 입력·cache·descriptor
변환·dirty state 전체와 native 응답의 추가 raw HTTP metadata를 제공한다는 뜻은
아닙니다. 공통 알고리즘을 공유해도 개별 proxy 연산의 전체 지원 판정은
[SDK 지원 판정대장](sdk-support-ledger.md)에서 별도로 추적합니다.

비교한 소스는 [Resource.find와 정확한 ID/이름 비교](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/resource.py#L2439),
[Nova find_server](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/compute/v2/_proxy.py#L1023),
[Cinder find_volume](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/block_storage/v3/_proxy.py#L855),
[Neutron find_port](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3950),
[Designate find_recordset](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/dns/v2/_proxy.py#L337),
[Octavia find_member](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/load_balancer/v2/_proxy.py#L542)입니다.
