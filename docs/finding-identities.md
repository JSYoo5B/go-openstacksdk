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
| Nova flavor | `conn.compute.find_flavor("small", get_extra_specs=True)` | `compute.Flavors.FindIdentity(ctx, "small", resource.WithIdentityFindExtraSpecs(true))` |
| Cinder v3 볼륨 | `conn.block_storage.find_volume("data-01")` | `storage.Volumes.FindIdentity(ctx, "data-01")` |
| Glance v2 이미지 | `conn.image.find_image("ubuntu")` | `image.Images.FindIdentity(ctx, "ubuntu")` |
| Neutron 포트 | `conn.network.find_port("web-port")` | `network.Ports.FindIdentity(ctx, "web-port")` |
| Neutron 네트워크 | `conn.network.find_network("web-net")` | `network.Networks.FindIdentity(ctx, "web-net")` |
| Neutron subnet | `conn.network.find_subnet("web-subnet")` | `network.Subnets.FindIdentity(ctx, "web-subnet")` |
| Neutron router | `conn.network.find_router("app-router", project_id=project_id)` | `network.Routers.FindIdentity(ctx, "app-router", projectQuery)` |
| Neutron security group | `conn.network.find_security_group("app-sg", project_id=project_id)` | `network.SecurityGroups.FindIdentity(ctx, "app-sg", projectQuery)` |
| Keystone 프로젝트 | `conn.identity.find_project("app", domain_id=domain_id)` | `identity.Projects.FindIdentity(ctx, "app", domainQuery)` |
| Keystone 사용자 | `conn.identity.find_user("app-user", domain_id=domain_id)` | `identity.Users.FindIdentity(ctx, "app-user", domainQuery)` |
| Keystone 그룹 | `conn.identity.find_group("app-group", domain_id=domain_id)` | `identity.Groups.FindIdentity(ctx, "app-group", domainQuery)` |
| Keystone domain | `conn.identity.find_domain("Default")` | `identity.Domains.FindIdentity(ctx, "Default")` |
| Keystone role | `conn.identity.find_role("reader", domain_id=domain_id)` | `identity.Roles.FindIdentity(ctx, "reader", domainQuery)` |
| Designate recordset | `conn.dns.find_recordset(zone, "www.example.org.")` | `records.FindIdentity(ctx, "www.example.org.")`, `records`는 `RecordSets.InZone`의 반환값 |
| Octavia member | `conn.load_balancer.find_member("backend-01", pool)` | `members.FindIdentity(ctx, "backend-01")`, `members`는 `Pools.Members`의 반환값 |

위 16개 리소스가 공통 자동 조회를 지원합니다. 기존 상위 `conn.Compute(ctx).Servers/Flavors`,
`conn.BlockStorage(ctx).Volumes`, `conn.Image(ctx).Images`, `conn.Network(ctx).Networks/Ports`도 같은 조회 정책을 제공합니다.
Router·Security Group은 `conn.Network(ctx).API.Routers/SecurityGroups`로 접근합니다.
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
| `WithIdentityFindDetails(false)` | Nova·Cinder fallback에서 summary 목록 사용, 기본값은 true |
| `WithIdentityFindAllProjects(true)` | Nova·Cinder fallback 목록에만 `all_tenants=true` 추가, 기본값은 false |
| `WithIdentityFindExtraSpecs(true)` | Nova flavor의 단일 결과에 extra specs가 없을 때 후속 GET, 기본값은 false |
| `WithIdentityFindQuery(key, value)` | 직접 GET과 fallback 목록의 서버 query, 같은 key는 뒤의 옵션 우선 |
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

caller query는 첫 GET과 fallback 목록 모두에 동일하게 전달합니다. 이름 hint는 목록으로
전환할 때만 추가하므로 GET에 자동 이름 필터를 넣지 않습니다. caller가 binding의
이름 query key를 지정하면 자동 이름 hint로 덮어쓰지 않습니다. 기본 hint는 Nova에서
`^`·`$`와 `regexp.QuoteMeta`로 만든 정확한 정규식, 다른 14개 binding에서는 literal
문자열입니다. Flavor는 자동 이름 hint를 서버에 보내지 않습니다. 서버가 hint를 무시해도 SDK의 ID/이름 비교는 그대로 수행합니다.

명시 wire `status`는 모델에 Status 필드가 없어도 GET과 목록에 전달합니다.
이 query만으로 응답 status를 로컬에서 필터링하지 않습니다. 자동 조회 binding의 일반
`Resources.List/All`에서도 raw `WithQuery("status", ...)` 단독은 같은 wire 정책입니다.
Octavia Member의 typed `WithStatus`는 기존 로컬 필터를 유지하고 wire status를 보내지
않습니다. `WithStatus`와 raw status를 함께 쓰면 typed 필터가 활성화된 채 마지막
값으로 로컬 비교하므로, raw query 단독과 구분합니다. 로컬 cap은 이 필터보다 먼저
적용됩니다. 상태 필드가 없는 리소스의 typed `WithStatus`는 계속 `ErrUnsupported`입니다.

query가 없으면 기존 native Get을 사용합니다. query를 지정하면 SDK가 감사한 member
경로·성공 코드를 사용하고 같은 native `GetResult.Extract`로 응답을 해석합니다. 서비스
client와 provider를 복제하거나 변경하지 않으며 최신 인증 token·HTTP client·기본 헤더와
microversion을 유지합니다. 내부 binding에 GET-query hook이 없으면 안전한 ID의 query
조회는 첫 HTTP 전에 `ErrUnsupported`입니다. GET을 사용하지 않는 unsafe 이름의
목록 검색은 이 hook을 요구하지 않습니다.

GET 결과와 소비한 목록의 각 행은 nil이 아닌 리소스와 안전한 ID를 요구합니다.
빈 ID나 경로에 사용할 수 없는 ID가 있는 행은 일치 여부와 관계없이 오류입니다.
typed native 응답에 없는 HTTP 본문·헤더를 이 검증 오류에 만들어 붙이지 않습니다.

`max_items`, `paginated`, `base_path`, `list_base_path`, `jmespath_filters`, `headers`,
`microversion`, `allow_unknown_params`, `ignore_missing`, `fallback`, `get_extra_specs`는 이 조회의 query가
아닙니다. 대소문자를 바꾼 키도 HTTP 전에 거부합니다. 중복과 후속 오류를 확인하기 위해 자동 조회는 전체
목록을 소비하며 로컬 cap이나 첫 페이지 중단 옵션을 받지 않습니다. 생성한 옵션은
bool pointer, query map과 slice를 복사하고 호출마다 독립적으로 적용합니다. 옵션
생성 뒤 원본 설정을 바꾸어도 이미 생성한 옵션의 동작은 바뀌지 않습니다.

## Nova·Cinder 목록 모드

Python의 `find_server(..., details=False, all_projects=True)`와
`find_volume(..., details=False, all_projects=True)`는 두 concrete 옵션으로 표현합니다.
이 옵션은 첫 GET의 경로나 query를 변경하지 않습니다. GET에서 찾았다면 그 native
응답을 즉시 반환하므로 `details=False`에서도 상세 GET 결과를 받을 수 있습니다.

GET이 fallback으로 전환되면 `Details`의 nil/true는 `/servers/detail` 또는
`/volumes/detail`, false는 `/servers` 또는 `/volumes`를 선택합니다. summary에서도
같은 native 모델을 해석하며 생략된 필드는 zero value입니다. 이름을 찾은 뒤 상세
GET을 자동으로 추가하지 않습니다. summary 목록에도 전체 페이지의 정확한 ID/이름·
중복·후속 실패 검사가 적용됩니다.

`AllProjects`의 nil/false는 자동 wire key를 넣지 않고 true만 목록 query에
`all_tenants=true`를 추가합니다. cloud 권한은 서버가 판단합니다. 직접 지정한
`all_tenants`와 명시적 `AllProjects`를 함께 쓰면 값이나 옵션 순서와 관계없이 요청
전에 `ErrInvalidOption`입니다. nil/빈 slice도 직접 지정한 key로 취급합니다.
`AllProjects`를 지정하지 않은 raw `all_tenants` query는 일반 query 확장으로서
GET과 목록 모두에 전달됩니다. `details`·`all_projects`를 raw query로 쓰면 오류입니다.

두 옵션은 감사된 Nova 서버·Cinder v3 볼륨에만 제공합니다. 다른 `FindIdentity`
binding에서 명시적으로 true나 false를 지정하면 첫 HTTP 전에 `ErrUnsupported`입니다.

```go
package example

import (
    "context"

    computev2 "gophercloudsdk/compute/v2"
    blockstoragev3 "gophercloudsdk/blockstorage/v3"
    "gophercloudsdk/resource"
)

func FindAcrossProjects(ctx context.Context, compute *computev2.Service,
    storage *blockstoragev3.Service) error {
    summary := resource.WithIdentityFindDetails(false)
    allProjects := resource.WithIdentityFindAllProjects(true)
    strict := resource.WithIdentityFindIgnoreMissing(false)
    server, err := compute.Servers.FindIdentity(ctx, "web-01", summary, allProjects, strict)
    if err != nil { return err }
    volume, err := storage.Volumes.FindIdentity(ctx, "data-01", summary, allProjects, strict)
    if err != nil { return err }
    _, _ = server, volume
    return nil
}
```

## Nova flavor와 extra specs

Flavor 자동 조회는 GET부터 시도하고 fallback은 `/flavors/detail` 전체를 검사합니다.
이름은 서버 query mapping에 없으므로 SDK가 자동 `name` hint를 추가하지 않습니다.
목록에서 `is_public`을 지정하지 않았다면 문자열 `None`을 보내 공개·비공개 flavor를
함께 검색합니다. 이 기본값은 GET에 전달하지 않으며 서버 권한은 그대로 적용됩니다.
caller가 지정한 raw wire `is_public`과 반복 query는 GET·목록에서 보존합니다.
Go의 명시 nil/빈 slice는 key를 지정한 것으로 취급해 기본값을 막고 wire에서는 생략합니다.
Python은 이 인자의 None도 문자열 `None`으로 바꾸므로, Go에서 그 요청이 필요하면
`WithIdentityFindQuery("is_public", "None")`으로 명시합니다.

Python `find_flavor(..., get_extra_specs=True)`는
`resource.WithIdentityFindExtraSpecs(true)`로 표현합니다. 생략과 false는 추가 호출을
하지 않습니다. true일 때 GET 결과 또는 전체 목록을 검증한 단일 결과의 `ExtraSpecs`가
nil/빈 map이면, 반환된 canonical ID의 `/flavors/{id}/os-extra_specs`를 한 번 조회합니다.
이미 한 개 이상의 spec이 있으면 추가 GET을 하지 않으며, 이름으로 찾은 뒤 상세 member
GET을 추가하지도 않습니다. 이 조회에는 caller query를 전달하지 않습니다.

후속 GET 결과는 복사한 Flavor 모델의 ExtraSpecs를 교체합니다. HTTP403·404,
잘못된 typed 응답·전송·취소 오류는 조회 실패이며 fallback이나 IgnoreMissing으로
숨기지 않습니다. 목록의 중복·후속 실패·미존재도 추가 specs 요청을 시작하지 않습니다.
inline extra specs가 도입된 2.61을 별도 extra-specs GET의 최소 microversion으로
강제하지 않고 원래 client의 설정을 유지합니다. native Extract의 missing/null
`extra_specs`는 nil map이며 Python의 missing 기본 빈 dict와 구별합니다.

```go
package example

import (
    "context"

    computev2 "gophercloudsdk/compute/v2"
    "gophercloudsdk/resource"
)

func FindFlavor(ctx context.Context, compute *computev2.Service) error {
    flavor, err := compute.Flavors.FindIdentity(ctx, "small",
        resource.WithIdentityFindExtraSpecs(true),
        resource.WithIdentityFindIgnoreMissing(false),
        resource.WithIdentityFindQuery("minRam", "256"))
    if err != nil { return err }
    _, _ = flavor.ID, flavor.ExtraSpecs
    return nil
}
```

Go query는 `minRam`·`minDisk` 같은 wire key를 사용하며 Python의 `min_ram`·`min_disk`
별칭을 자동 변환하지 않습니다. ExtraSpecs 옵션의 명시 true/false는 Flavor만 지원합니다.
Flavor에 typed `Details`나 `AllProjects`를 지정하면 HTTP 전에 `ErrUnsupported`입니다.
일반 Get/List와 explicit ID/Name Find는 이 자동 조회의 목록 기본값·후속 specs 정책을
적용하지 않습니다.
생성된 Flavor `Resources.List/All`의 query 전달도 보정했습니다. 전체 query map을
복사해 보존하며, 이전 adapter가 버리던 raw `status`도 서버에 전달합니다.

## Glance 숨김 이미지 검색

Python `conn.image.find_image("ubuntu", ignore_missing=False)`는 일반 조회에서 찾지
못하면 숨김 이미지 목록도 검색합니다. Go의 `Images.FindIdentity`는 첫 GET과 일반
목록이 정상적으로 끝나도 일치하는 이미지가 없을 때만 이 두 번째 검색을 실행합니다.
GET에서 찾으면 반환하며 일반 목록은 끝까지 검사해 단일 일치가 확인되면 숨김 검색을 생략합니다. 일반 목록의
오류·중복·취소는 숨김 검색으로 전환하지 않습니다.

숨김 검색은 원래 caller query에 wire key `os_hidden=true`를 덮어쓴 LIST입니다.
자동 이름 hint는 이 단계에 추가하지 않으므로 ID로만 일치하는 숨김 이미지도 검사합니다.
caller가 지정한 `name` 필터와 반복 query 값은 보존합니다. 앞서 `os_hidden=false`를
지정했어도 두 번째 검색에서는 true이며, 옵션과 원본 query는 변경하지 않습니다.
두 검색 모두 모든 페이지의 정확한 ID/이름·중복·후속 오류를 검사하며 추가 GET은 없습니다.
미존재 기본값·strict 옵션은 두 검색이 모두 정상적으로 끝난 뒤 적용합니다.
`FindFallbackNever`는 기존 GET만 사용하는 정책을 유지합니다.

Python의 직접 `find_image` 선언은 문자열과 `ignore_missing`만 받습니다. Go의 raw
wire query는 추가 기능이며 Python 속성 별칭 `is_hidden`은 자동 변환하지 않습니다.
`details`와 `all_projects`도 이 binding의 옵션이 아닙니다. 응답은 native Image 모델이며
추가 속성은 `Properties`, 숨김 여부는 `Hidden`에서 확인합니다.

```go
package example

import (
    "context"

    imagev2 "gophercloudsdk/image/v2"
    "gophercloudsdk/resource"
)

func FindImage(ctx context.Context, images *imagev2.Service) error {
    image, err := images.Images.FindIdentity(ctx, "ubuntu",
        resource.WithIdentityFindIgnoreMissing(false))
    if err != nil { return err }
    _, _ = image.ID, image.Hidden
    return nil
}
```

## Keystone domain과 네트워크 필터

이름만으로 프로젝트·사용자·그룹을 검색하면 서로 다른 domain의 같은 이름도 중복으로
판정합니다. SDK는 인증한 사용자의 domain을 검색 필터로 추정하지 않습니다.
`domain_id`를 명시하면 첫 GET과 모든 fallback 페이지에 전달합니다. 서버가 필터를
무시하면 정확한 이름·중복 검사도 유지합니다. Keystone role의 domain 필터도 서버의
권한·API 계약에 따릅니다. Domains의 query 확장은 Go 기능이며 Python `find_domain`은
추가 query 인자를 받지 않습니다.

```go
package example

import (
    "context"

    identityv3 "gophercloudsdk/identity/v3"
    networkv2 "gophercloudsdk/network/v2"
    "gophercloudsdk/resource"
)

func FindTenantResources(ctx context.Context, identity *identityv3.Service,
    network *networkv2.Service) error {
    strict := resource.WithIdentityFindIgnoreMissing(false)
    domain, err := identity.Domains.FindIdentity(ctx, "Default", strict)
    if err != nil { return err }
    domainQuery := resource.WithIdentityFindQuery("domain_id", domain.ID)
    project, err := identity.Projects.FindIdentity(ctx, "app", strict, domainQuery)
    if err != nil { return err }
    user, err := identity.Users.FindIdentity(ctx, "app-user", strict, domainQuery)
    if err != nil { return err }
    group, err := identity.Groups.FindIdentity(ctx, "app-group", strict, domainQuery)
    if err != nil { return err }
    role, err := identity.Roles.FindIdentity(ctx, "reader", strict, domainQuery)
    if err != nil { return err }
    net, err := network.Networks.FindIdentity(ctx, "web-net", strict)
    if err != nil { return err }
    subnet, err := network.Subnets.FindIdentity(ctx, "web-subnet", strict,
        resource.WithIdentityFindQuery("network_id", net.ID))
    if err != nil { return err }
    _, _, _, _, _ = project, user, group, role, subnet
    return nil
}
```

## Python과 추가로 비교할 범위

Nova와 Cinder의 기본 `details=True`와 `all_projects=False` 및 명시적 변경은 위
옵션에 대응합니다. 임의 `list_base_path` 교체, 호출별 header·microversion 선택은
이 공통 API에 없습니다. `WithIdentityFindQuery`로 지정한 wire query는 GET과 목록에
모두 전달하며, Python의 descriptor query 별칭·로컬 Body 필터 변환까지 자동 적용하지 않습니다.

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

Neutron의 [find_network](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L3214),
[find_subnet](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L7487)과 Keystone의
[find_project](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1063),
[find_user](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1379),
[find_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L782),
[find_domain](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L259),
[find_role](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/identity/v3/_proxy.py#L1743)도 같은 pin으로 비교했습니다.

## Neutron Router·Security Group와 project query

Python의 `conn.network.find_router("app-router", project_id=project_id)`와
`conn.network.find_security_group("app-sg", project_id=project_id)`는 다음처럼 사용합니다.
프로젝트 scope는 호출자가 지정하며, SDK가 현재 프로젝트 필터를 자동으로 넣지 않습니다.
반복 wire query는 concrete 옵션으로 보존합니다.

```go
package example

import (
    "context"
    "net/url"

    networkv2 "gophercloudsdk/network/v2"
    "gophercloudsdk/resource"
)

func FindNetworkExtensions(ctx context.Context, network *networkv2.Service,
    projectID string) error {
    filter := resource.WithIdentityFindOptions(resource.IdentityFindOpts{
        Query: url.Values{"project_id": {projectID}, "fields": {"id", "name"}},
    })
    strict := resource.WithIdentityFindIgnoreMissing(false)
    router, err := network.Routers.FindIdentity(ctx, "app-router", filter, strict)
    if err != nil { return err }
    group, err := network.SecurityGroups.FindIdentity(ctx, "app-sg", filter, strict)
    if err != nil { return err }
    _, _ = router, group
    return nil
}
```

두 리소스는 native GET200과 각각 `routers`·`security_groups` envelope/continuation을
사용합니다. fallback 목록에만 literal name hint를 추가하며 caller의 name key가
nil/빈 slice여도 덮어쓰지 않습니다. 같은 이름의 두 행은 프로젝트가 달라도 중복입니다.
요청한 `fields`에 ID가 없어 응답 ID를 검증할 수 없다면 조회는 오류로 끝납니다.

Security Group의 native `List`는 concrete `ListOpts`를 받습니다. 공통
`Resources.List/All`은 SDK 소유 pager로 반복·빈 값·확장 wire query를 보존하고 native
모델과 추출·페이지 링크를 사용합니다. `WithMaxItems`·`WithPaginated` 소비 옵션도
같이 적용하며 직접 typed `List`의 API는 유지합니다. Router의 공통 목록도 raw query를
전달합니다. 이 보정은 일반 목록의 raw status와 확장 key 전달에도 적용됩니다.

Python `is_admin_state_up`·`is_distributed`·`is_ha`·`is_shared`는 각각
wire `admin_state_up`·`distributed`·`ha`·`shared`로 직접 지정해야 합니다.
Python의 query alias/로컬 Body 분류·descriptor 변환·conditional GET cache·상속
continuation/session 정책까지 동일하게 구현한 것은 아닙니다. Details·AllProjects·ExtraSpecs
옵션은 명시적 false도 첫 HTTP 전에 `ErrUnsupported`이며 자동 hidden 검색이나
microversion 변경은 없습니다.

고정 Python [find_router](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L5570)·[find_security_group](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/_proxy.py#L6598)는 문자열·ignore_missing·query를 선언합니다.
[Router query mapping](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/router.py#L36)·[SecurityGroup query mapping](https://github.com/openstack/openstacksdk/blob/ef55d7d1666099f50bf1e1c40b59d7e7b72a51fe/openstack/network/v2/security_group.py#L29)의 별칭은 Go의 wire query와 구분합니다.
